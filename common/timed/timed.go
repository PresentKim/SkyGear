// Package timed implements time-limited items (기간제 아이템). The period is
// real time only: an item stores the wall-clock time it expires at and is
// removed after that, whether or not the server, the save or the player was
// online in between. Limits by use count are separate (durability or uses)
// and do not belong here (docs/tech/timed-items.md).
package timed

import (
	"fmt"
	"github.com/presentkim/dragonfactory/common/i18n"
	"strings"
	"time"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
)

// key is the item stack value holding the expiry as Unix seconds. Stack
// values are saved with the item (Dragonfly's dragonflyData), so the expiry
// survives inventories, containers and the database.
const key = "df-server:expires"

// lorePrefix starts the lore line that shows the expiry.
const lorePrefix = "§r§e기간제 §7· "

// Now is the clock used for expiry. Tests replace it.
var Now = time.Now

// Zone is the time zone the expiry is shown in.
var Zone = func() *time.Location {
	if l, err := time.LoadLocation("Asia/Seoul"); err == nil {
		return l
	}
	return time.FixedZone("KST", 9*60*60)
}()

// For returns st expiring d from now.
func For(st item.Stack, d time.Duration) item.Stack { return Until(st, Now().Add(d)) }

// Until returns st expiring at t. A stack that already had an expiry gets
// the new one.
func Until(st item.Stack, t time.Time) item.Stack {
	if st.Empty() {
		return st
	}
	lore := withoutExpiryLine(st.Lore())
	lore = append(lore, lorePrefix+t.In(Zone).Format("2006-01-02 15:04")+i18n.T(i18n.Default, "timed.until"))
	return st.WithValue(key, t.Unix()).WithLore(lore...)
}

// Clear returns st without an expiry.
func Clear(st item.Stack) item.Stack {
	if _, ok := Expiry(st); !ok {
		return st
	}
	return st.WithValue(key, nil).WithLore(withoutExpiryLine(st.Lore())...)
}

func withoutExpiryLine(lore []string) []string {
	out := make([]string, 0, len(lore)+1)
	for _, l := range lore {
		if !strings.HasPrefix(l, lorePrefix) {
			out = append(out, l)
		}
	}
	return out
}

// Expiry returns when st expires, or false if it is not a timed item.
func Expiry(st item.Stack) (time.Time, bool) {
	v, ok := st.Value(key)
	if !ok {
		return time.Time{}, false
	}
	sec, ok := v.(int64)
	if !ok {
		return time.Time{}, false
	}
	return time.Unix(sec, 0), true
}

// Is reports whether st is a timed item, expired or not.
func Is(st item.Stack) bool {
	_, ok := Expiry(st)
	return ok
}

// Expired reports whether st is a timed item whose time is up.
func Expired(st item.Stack) bool {
	t, ok := Expiry(st)
	return ok && !Now().Before(t)
}

// Remaining returns the time left on a timed item (0 when expired), or false
// if st is not timed.
func Remaining(st item.Stack) (time.Duration, bool) {
	t, ok := Expiry(st)
	if !ok {
		return 0, false
	}
	return max(0, t.Sub(Now())), true
}

// Sweep removes the expired stacks of inv and returns them.
func Sweep(inv *inventory.Inventory) []item.Stack {
	var out []item.Stack
	for slot, st := range inv.Slots() {
		if Expired(st) {
			_ = inv.SetItem(slot, item.Stack{})
			out = append(out, st)
		}
	}
	return out
}

// SweepPlayer removes the expired items a player carries (inventory, armour,
// off hand and ender chest) and tells the player what was removed. It must
// be called in the player's world transaction.
func SweepPlayer(p *player.Player) []item.Stack {
	removed := Sweep(p.Inventory())
	removed = append(removed, Sweep(p.Armour().Inventory())...)
	removed = append(removed, Sweep(p.EnderChestInventory())...)
	if main, off := p.HeldItems(); Expired(off) {
		p.SetHeldItems(main, item.Stack{})
		removed = append(removed, off)
	}
	Notify(p, removed)
	return removed
}

// Notify tells p that the stacks were removed because their time was up.
func Notify(p *player.Player, removed []item.Stack) {
	for _, st := range removed {
		lang := i18n.Of(p)
		p.Messagef(i18n.T(lang, "timed.removed"), Name(lang, st), st.Count())
	}
}

// Name is the name shown for a stack in messages in lang: its custom name or
// its item identifier. Games with their own item names can replace it.
var Name = func(_ string, st item.Stack) string {
	if n := st.CustomName(); n != "" {
		return n
	}
	name, _ := st.Item().EncodeItem()
	return name
}

// Describe returns the time left on a timed item for texts, e.g. "2일 3시간".
func Describe(d time.Duration) string {
	switch {
	case d <= 0:
		return i18n.T(i18n.Default, "timed.expired")
	case d >= 24*time.Hour:
		return fmt.Sprintf(i18n.T(i18n.Default, "timed.left.dayHour"), int(d/(24*time.Hour)), int(d%(24*time.Hour)/time.Hour))
	case d >= time.Hour:
		return fmt.Sprintf(i18n.T(i18n.Default, "timed.left.hourMin"), int(d/time.Hour), int(d%time.Hour/time.Minute))
	case d >= time.Minute:
		return fmt.Sprintf(i18n.T(i18n.Default, "timed.left.min"), int(d/time.Minute))
	}
	return fmt.Sprintf(i18n.T(i18n.Default, "timed.left.sec"), int(d/time.Second))
}
