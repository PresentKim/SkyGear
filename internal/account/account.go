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

// Achievement is something a player can earn once per account. Its texts are
// catalog keys: the name, what to do, and the title it unlocks (shown before
// the name when equipped; none when TitleKey is empty).
type Achievement struct {
	ID                         string
	NameKey, DescKey, TitleKey string
	// GroupKey is the catalog key of the category the screens list it under.
	GroupKey string
}

// Keyed is the achievement id with the texts under "achievement.<id>.name",
// ".desc" and ".title", listed in the group of groupKey.
func Keyed(id, groupKey string) Achievement {
	prefix := "achievement." + id + "."
	return Achievement{ID: id, NameKey: prefix + "name", DescKey: prefix + "desc", TitleKey: prefix + "title", GroupKey: groupKey}
}

// Name is the achievement's name in lang.
func (a Achievement) Name(lang string) string { return i18n.T(lang, a.NameKey) }

// Desc says in lang what earns the achievement.
func (a Achievement) Desc(lang string) string { return i18n.T(lang, a.DescKey) }

// Title is the title the achievement unlocks in lang, "" for none.
func (a Achievement) Title(lang string) string {
	if a.TitleKey == "" {
		return ""
	}
	return i18n.T(lang, a.TitleKey)
}

// Color is a name colour. Need is the number of achievements that unlock it.
type Color struct {
	ID, NameKey, Code string
	Need              int
}

// Name is the colour's name in lang.
func (c Color) Name(lang string) string { return i18n.T(lang, c.NameKey) }

// Colors are the name colours in display order; the first is the default.
// Unlock counts are balancing drafts.
var Colors = []Color{
	{"", "account.account.init.1", "§f", 0}, {"yellow", "account.account.init.2", "§e", 0}, {"aqua", "account.account.init.3", "§b", 0}, {"green", "account.account.init.4", "§a", 0},
	{"gold", "account.account.init.5", "§6", 3}, {"pink", "account.account.init.6", "§d", 3}, {"red", "account.account.init.7", "§c", 6}, {"blue", "account.account.init.8", "§9", 6},
	{"darkgreen", "account.color.darkgreen", "§2", 12}, {"darkaqua", "account.color.darkaqua", "§3", 20}, {"purple", "account.color.purple", "§5", 30},
	{"copper", "account.color.copper", "§n", 45}, {"emerald", "account.color.emerald", "§q", 65}, {"diamond", "account.color.diamond", "§s", 90},
	{"amethyst", "account.color.amethyst", "§u", 120}, {"netherite", "account.color.netherite", "§j", 160},
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

// SetGroupOrder sets the order the achievement groups (their keys) are
// listed in; groups it does not name come after, in the order they were
// registered.
func SetGroupOrder(keys ...string) {
	mu.Lock()
	defer mu.Unlock()
	groupOrder = keys
}

// Groups returns the group keys of the registered achievements in listing
// order. Achievements without a group are in the group "".
func Groups() []string {
	var seen []string
	for _, a := range All() {
		if !slices.Contains(seen, a.GroupKey) {
			seen = append(seen, a.GroupKey)
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
	mu.Lock()
	defer mu.Unlock()
	for _, a := range achievements {
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
	if store == nil {
		return
	}
	acc := instance.AccountID(p)
	if hasGranted(acc, id) {
		return
	}
	a, ok := Get(id)
	if !ok {
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
	ui.Tell(p, i18n.T(lang, "account.account.Grant.3"), fmt.Sprintf(i18n.T(lang, "account.account.Grant.1"), a.Name(lang), a.Desc(lang)))
	if title := a.Title(lang); title != "" {
		ui.Tell(p, i18n.T(lang, "account.account.Grant.3"), fmt.Sprintf(i18n.T(lang, "account.account.Grant.2"), title))
	}
	p.PlaySound(sound.LevelUp{})
}

// hasGranted reports whether the account is known to have the achievement,
// reading its achievements the first time.
func hasGranted(acc, id string) bool {
	mu.Lock()
	if set, ok := granted[acc]; ok {
		has := set[id]
		mu.Unlock()
		return has
	}
	mu.Unlock()
	ids, err := store.Achievements(acc)
	if err != nil {
		log.Error("load achievements", "account", acc, "err", err)
		return false
	}
	loaded := make(map[string]bool, len(ids))
	for _, x := range ids {
		loaded[x] = true
	}
	mu.Lock()
	defer mu.Unlock()
	if set, ok := granted[acc]; ok {
		return set[id]
	}
	granted[acc] = loaded
	return loaded[id]
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
	// Everyone sees the same name tag and chat name: the default language.
	if a, ok := Get(acc.Title); ok && a.Title(i18n.Default) != "" {
		name = "§7[" + a.Title(i18n.Default) + "]§r " + name
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

// Forget writes the counts of a player who left and drops its cached name.
func Forget(p *player.Player) {
	acc := instance.AccountID(p)
	flushMu.Lock()
	defer flushMu.Unlock()
	mu.Lock()
	delta := dirty[acc]
	delete(dirty, acc)
	d := store
	mu.Unlock()
	if d != nil && len(delta) > 0 {
		if err := d.AddCounters(acc, delta); err != nil {
			log.Error("store counters", "account", acc, "err", err)
		}
	}
	mu.Lock()
	delete(display, acc)
	delete(counts, acc)
	delete(granted, acc)
	mu.Unlock()
}
