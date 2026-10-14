// Package account has the account-wide extras of players
// (docs/design/01-platform.md §5): achievements, the title they unlock and
// the name colour. Games register their achievements and grant them with
// Grant; the platform shows titles and colours in chat and name tags.
package account

import (
	"github.com/presentkim/dragonfactory/internal/i18n"
	"log/slog"
	"sync"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/presentkim/dragonfactory/internal/db"
	"github.com/presentkim/dragonfactory/internal/instance"
)

// Achievement is something a player can earn once per account. Title is the
// title it unlocks (shown before the name when equipped).
type Achievement struct {
	ID, Name, Desc, Title string
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
}

var (
	mu           sync.Mutex
	achievements []Achievement
	store        *db.DB
	log          = slog.Default()
	// display caches the chat/name tag name of online accounts.
	display = map[string]string{}
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
	isNew, err := store.GrantAchievement(instance.AccountID(p), id)
	if err != nil {
		log.Error("grant achievement", "player", p.Name(), "id", id, "err", err)
		return
	}
	if !isNew {
		return
	}
	p.Messagef(i18n.T(i18n.Of(p), "account.account.Grant.1"), a.Name, a.Desc)
	if a.Title != "" {
		p.Messagef(i18n.T(i18n.Of(p), "account.account.Grant.2"), a.Title)
	}
	p.PlaySound(sound.LevelUp{})
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
	mu.Lock()
	delete(display, instance.AccountID(p))
	mu.Unlock()
}
