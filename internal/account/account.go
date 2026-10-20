// Package account has the account-wide extras of players
// (docs/design/01-platform.md §5): achievements, the title they unlock and
// the name colour. Games register their achievements and grant them with
// Grant; the platform shows titles and colours in chat and name tags.
package account

import (
	"fmt"
	"github.com/presentkim/dragonfactory/common/i18n"
	_ "github.com/presentkim/dragonfactory/internal/locales" // this server's texts
	"log/slog"
	"slices"
	"sync"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/presentkim/dragonfactory/common/ui"
	"github.com/presentkim/dragonfactory/internal/db"
	"github.com/presentkim/dragonfactory/internal/instance"
)

// Achievement is something a player can earn once per account. Title is the
// title it unlocks (shown before the name when equipped).
type Achievement struct {
	ID, Name, Desc, Title string
	// Group is the category the screens list it under.
	Group string
}

// Color is a name colour. Need is the number of achievements that unlock it.
type Color struct {
	ID, Name, Code string
	Need           int
}

// Colors are the name colours in display order; the first is the default.
// Unlock counts are balancing drafts.
var Colors = []Color{
	{"", i18n.T(i18n.Default, "account.account.init.1"), "§f", 0}, {"yellow", i18n.T(i18n.Default, "account.account.init.2"), "§e", 0}, {"aqua", i18n.T(i18n.Default, "account.account.init.3"), "§b", 0}, {"green", i18n.T(i18n.Default, "account.account.init.4"), "§a", 0},
	{"gold", i18n.T(i18n.Default, "account.account.init.5"), "§6", 3}, {"pink", i18n.T(i18n.Default, "account.account.init.6"), "§d", 3}, {"red", i18n.T(i18n.Default, "account.account.init.7"), "§c", 6}, {"blue", i18n.T(i18n.Default, "account.account.init.8"), "§9", 6},
	{"darkgreen", i18n.T(i18n.Default, "account.color.darkgreen"), "§2", 12}, {"darkaqua", i18n.T(i18n.Default, "account.color.darkaqua"), "§3", 20}, {"purple", i18n.T(i18n.Default, "account.color.purple"), "§5", 30},
	{"copper", i18n.T(i18n.Default, "account.color.copper"), "§n", 45}, {"emerald", i18n.T(i18n.Default, "account.color.emerald"), "§q", 65}, {"diamond", i18n.T(i18n.Default, "account.color.diamond"), "§s", 90},
	{"amethyst", i18n.T(i18n.Default, "account.color.amethyst"), "§u", 120}, {"netherite", i18n.T(i18n.Default, "account.color.netherite"), "§j", 160},
}

var (
	mu           sync.Mutex
	achievements []Achievement
	store        *db.DB
	log          = slog.Default()
	// display caches the chat/name tag name of online accounts.
	display = map[string]string{}
	// granted caches the achievements of the accounts that were granted
	// something, so granting again costs no database call.
	granted    = map[string]map[string]bool{}
	groupOrder []string
)

// Init sets the database achievements are stored in. It is called once at
// startup.
func Init(d *db.DB, l *slog.Logger) {
	mu.Lock()
	defer mu.Unlock()
	store, log = d, l
}

// Register adds achievements. IDs must be unique across games.
func Register(a ...Achievement) {
	mu.Lock()
	defer mu.Unlock()
	achievements = append(achievements, a...)
}

// All returns every registered achievement in registration order.
func All() []Achievement {
	mu.Lock()
	defer mu.Unlock()
	return append([]Achievement(nil), achievements...)
}

// SetGroupOrder sets the order the achievement groups are listed in; groups
// it does not name come after, in the order they were registered.
func SetGroupOrder(names ...string) {
	mu.Lock()
	defer mu.Unlock()
	groupOrder = names
}

// Groups returns the groups of the registered achievements in listing order.
// Achievements without a group are in the group "".
func Groups() []string {
	var seen []string
	for _, a := range All() {
		if !slices.Contains(seen, a.Group) {
			seen = append(seen, a.Group)
		}
	}
	mu.Lock()
	order := groupOrder
	mu.Unlock()
	rank := func(g string) int {
		if i := slices.Index(order, g); i >= 0 {
			return i
		}
		return len(order)
	}
	slices.SortStableFunc(seen, func(a, b string) int { return rank(a) - rank(b) })
	return seen
}

// Get returns the achievement with id.
func Get(id string) (Achievement, bool) {
	for _, a := range All() {
		if a.ID == id {
			return a, true
		}
	}
	return Achievement{}, false
}

// ColorOf returns the colour with id, or the default.
func ColorOf(id string) Color {
	for _, c := range Colors {
		if c.ID == id {
			return c
		}
	}
	return Colors[0]
}

// Grant gives p the achievement id if it does not have it yet, telling p.
// It may be called from any world transaction.
func Grant(p *player.Player, id string) {
	a, ok := Get(id)
	if !ok || store == nil {
		return
	}
	acc := instance.AccountID(p)
	if hasGranted(acc, id) {
		return
	}
	isNew, err := store.GrantAchievement(acc, id)
	if err != nil {
		log.Error("grant achievement", "player", p.Name(), "id", id, "err", err)
		return
	}
	markGranted(acc, id)
	if !isNew {
		return
	}
	lang := i18n.Of(p)
	ui.Tell(p, i18n.T(lang, "account.account.Grant.3"), fmt.Sprintf(i18n.T(lang, "account.account.Grant.1"), a.Name, a.Desc))
	if a.Title != "" {
		ui.Tell(p, i18n.T(lang, "account.account.Grant.3"), fmt.Sprintf(i18n.T(lang, "account.account.Grant.2"), a.Title))
	}
	p.PlaySound(sound.LevelUp{})
}

// hasGranted reports whether the account is known to have the achievement,
// reading its achievements the first time.
func hasGranted(acc, id string) bool {
	mu.Lock()
	defer mu.Unlock()
	set, ok := granted[acc]
	if !ok {
		set = map[string]bool{}
		ids, err := store.Achievements(acc)
		if err != nil {
			log.Error("load achievements", "account", acc, "err", err)
			return false
		}
		for _, x := range ids {
			set[x] = true
		}
		granted[acc] = set
	}
	return set[id]
}

func markGranted(acc, id string) {
	mu.Lock()
	defer mu.Unlock()
	if set, ok := granted[acc]; ok {
		set[id] = true
	}
}

// Owned returns the achievement IDs of an account.
func Owned(accountID string) []string {
	if store == nil {
		return nil
	}
	ids, err := store.Achievements(accountID)
	if err != nil {
		log.Error("load achievements", "account", accountID, "err", err)
	}
	return ids
}

// Refresh reloads p's title and colour and applies them to its name tag.
func Refresh(p *player.Player) {
	if store == nil {
		return
	}
	acc, err := store.Account(instance.AccountID(p))
	if err != nil {
		return
	}
	name := ColorOf(acc.NameColor).Code + p.Name() + "§r"
	if a, ok := Get(acc.Title); ok && a.Title != "" {
		name = "§7[" + a.Title + "]§r " + name
	}
	mu.Lock()
	display[acc.ID] = name
	mu.Unlock()
	p.SetNameTag(name)
}

// DisplayName returns p's name with its title and colour for chat.
func DisplayName(p *player.Player) string {
	mu.Lock()
	defer mu.Unlock()
	if n, ok := display[instance.AccountID(p)]; ok {
		return n
	}
	return p.Name()
}

// Forget drops the cached name of a player who left.
func Forget(p *player.Player) {
	Flush()
	mu.Lock()
	delete(display, instance.AccountID(p))
	delete(counts, instance.AccountID(p))
	delete(granted, instance.AccountID(p))
	mu.Unlock()
}
