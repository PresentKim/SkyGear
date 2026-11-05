package netstore

import (
	"fmt"
	"sort"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/storagenet"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/storage"
)

// The terminal (spec §5): putting items in with a click, and the terminal
// window (terminalui.go). The controller opens the same window.

// terminalSlots is how many kinds a terminal list shows.
const terminalSlots = 240

// Hooks set by the crafting part; nil means that part is absent.
var (
	// CraftableHook lists the keys the network can craft; the terminal shows
	// them under their own heading.
	CraftableHook func(n *Network) []storagenet.Key
	// RequestHook runs when a craftable key is pressed.
	RequestHook func(s *core.Session, p *player.Player, n *Network, k storagenet.Key)
	// MonitorHook runs when the "crafting tasks" bar button is pressed.
	MonitorHook func(s *core.Session, p *player.Player, n *Network)
	// InsertHook may take part of an inserted amount before it reaches Root
	// (outputs a crafting task waits for); it returns how much it took.
	InsertHook func(n *Network, k storagenet.Key, cnt int64) int64
)

// netInsert puts st into the network of n and returns how many items went in;
// an offline network takes none. InsertHook goes first.
func netInsert(n *Network, st item.Stack) int64 {
	if n == nil || !n.Online() || st.Empty() {
		return 0
	}
	k := KeyOf(st)
	cnt := int64(st.Count())
	var taken int64
	if InsertHook != nil {
		taken = max(0, min(cnt, InsertHook(n, k, cnt)))
	}
	if taken < cnt {
		taken += n.Root.Insert(k, cnt-taken, false)
	}
	return taken
}

// netExtract takes up to cnt (at most one stack) of the kind k out of n.
func netExtract(n *Network, k storagenet.Key, cnt int) item.Stack {
	if n == nil || !n.Online() {
		return item.Stack{}
	}
	proto := Proto(n.Sess, k)
	if proto.Empty() {
		return item.Stack{}
	}
	got := n.Root.Extract(k, int64(min(cnt, proto.MaxCount())), false)
	if got <= 0 {
		return item.Stack{}
	}
	return proto.Grow(int(got) - 1)
}

// terminalList is the entries of n in the order o, filtered by query and cut
// to terminalSlots; found is the number of matching kinds before the cut.
func terminalList(n *Network, lang, query string, o int) (list []storage.Entry, keys []storagenet.Key, found int) {
	found0 := storage.Filter(lang, entriesOf(n), query)
	found = len(found0)
	ord := storage.OrderCountDesc
	switch o {
	case 1:
		ord = storage.OrderCountAsc
	case 2:
		ord = storage.OrderNameAsc
	case 3:
		ord = storage.OrderNameDesc
	}
	list = storage.Sort(lang, found0, ord)
	if len(list) > terminalSlots {
		list = list[:terminalSlots]
	}
	for _, e := range list {
		keys = append(keys, KeyOf(e.Item))
	}
	return list, keys, found
}

// keepOrder is the list in the order of kept, each kind with what n holds
// now (0 when it ran out), then the kinds of list not in kept, cut to the
// terminal's slots.
func keepOrder(n *Network, kept []storagenet.Key, list []storage.Entry) ([]storage.Entry, []storagenet.Key) {
	var out []storage.Entry
	var keys []storagenet.Key
	in := map[storagenet.Key]bool{}
	for _, k := range kept {
		if p := Proto(n.Sess, k); !p.Empty() {
			out = append(out, storage.Entry{Item: p, Count: n.Root.Count(k)})
			keys = append(keys, k)
			in[k] = true
		}
	}
	for _, e := range list {
		if k := KeyOf(e.Item); !in[k] {
			out = append(out, e)
			keys = append(keys, k)
		}
	}
	if len(out) > terminalSlots {
		out, keys = out[:terminalSlots], keys[:terminalSlots]
	}
	return out, keys
}

// terminalInfo is the line over the list: kind and item totals, the search
// and the hint when the list is cut.
func terminalInfo(lang string, kinds int, items int64, query string, found int) string {
	info := fmt.Sprintf(i18n.T(lang, "netstore.terminal.lines"), kinds, items)
	if query != "" {
		info += "\n" + fmt.Sprintf(i18n.T(lang, "storage.search.found"), query, found)
	}
	if found > terminalSlots {
		info += "\n" + fmt.Sprintf(i18n.T(lang, "netstore.terminal.cap"), terminalSlots)
	}
	return info
}

// depositKinds puts every stack of inv whose kind the network holds into it
// and returns how many items went in.
func depositKinds(n *Network, inv *inventory.Inventory) int64 {
	var total int64
	for slot, st := range inv.Slots() {
		if st.Empty() || n.Root.Count(KeyOf(st)) == 0 {
			continue
		}
		if got := netInsert(n, st); got > 0 {
			_ = inv.SetItem(slot, st.Grow(-int(got)))
			total += got
		}
	}
	return total
}

type terminalUse struct {
	Pos cube.Pos
	at  time.Time
}

var (
	lastUse = core.NewPlayers[terminalUse]()
	// orders are how each player sorts the list.
	orders = core.NewPlayers[int]()
	// keptOrders is the list order kept while the player holds a button, a
	// finger or shift on the terminal window (nil until the first draw after
	// the hold began).
	keptOrders = core.NewPlayers[[]storagenet.Key]()
)

// ShowTerminal handles a click on a terminal or controller: a sneaking click
// opens the terminal window (terminalui.go), also while the network is off,
// any other deposits the held stack, and a second click within 500ms also
// deposits the inventory kinds the network holds.
func ShowTerminal(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) {
	lang := i18n.Of(p)
	n := At(tx, pos)
	if n == nil {
		return
	}
	if p.Sneaking() {
		if !s.Debounced(p) {
			delete(keptOrders.Of(s), p.H())
			showTerminal(s, p, tx, pos, "")
		}
		return
	}
	if !n.Online() {
		p.Message(i18n.T(lang, "netstore.terminal.offline"))
		return
	}
	last := lastUse.Of(s)[p.H()]
	double := last.Pos == pos && time.Since(last.at) < 500*time.Millisecond
	lastUse.Of(s)[p.H()] = terminalUse{pos, time.Now()}
	held, off := p.HeldItems()
	var total int64
	if got := netInsert(n, held); got > 0 {
		p.SetHeldItems(held.Grow(-int(got)), off)
		total += got
	}
	if double {
		total += depositKinds(n, p.Inventory())
	}
	if total > 0 {
		tx.PlaySound(pos.Vec3Centre(), sound.ItemAdd{})
	}
}

// IsTerminal reports whether b opens the terminal window: a terminal or a
// network controller.
func IsTerminal(b world.Block) bool {
	m, ok := b.(machine.Machine)
	return ok && (m.Kind == machine.KindNetTerminal || m.Kind == machine.KindNetController)
}

var orderKeys = []string{"storage.order.countDesc", "storage.order.countAsc", "storage.order.nameAsc", "storage.order.nameDesc"}

// entriesOf is every entry of n, in key order.
func entriesOf(n *Network) []storage.Entry {
	var ks []storagenet.Key
	n.Root.Each(func(k storagenet.Key, c int64) {
		if c > 0 {
			ks = append(ks, k)
		}
	})
	sort.Slice(ks, func(i, j int) bool { return ks[i] < ks[j] })
	var out []storage.Entry
	for _, k := range ks {
		if p := Proto(n.Sess, k); !p.Empty() {
			out = append(out, storage.Entry{Item: p, Count: n.Root.Count(k)})
		}
	}
	return out
}

// The terminal and the controller open the list on a click, and take and give
// items through Root.
func init() {
	for _, kind := range []string{machine.KindNetTerminal, machine.KindNetController} {
		d := machine.Kinds[kind]
		d.Click = func(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool {
			ShowTerminal(s, p, tx, pos)
			return true
		}
		d.Insert = func(tx *world.Tx, pos cube.Pos, st item.Stack) int {
			return int(netInsert(At(tx, pos), st))
		}
		d.Peek = func(tx *world.Tx, pos cube.Pos, want func(item.Stack) bool) item.Stack {
			n := At(tx, pos)
			if n == nil || !n.Online() {
				return item.Stack{}
			}
			for _, e := range entriesOf(n) {
				if want == nil || want(e.Item) {
					return e.Item.Grow(int(min(e.Count, int64(e.Item.MaxCount()))) - 1)
				}
			}
			return item.Stack{}
		}
		d.Extract = func(tx *world.Tx, pos cube.Pos, st item.Stack, cnt int) item.Stack {
			n := At(tx, pos)
			return netExtract(n, KeyOf(st), cnt)
		}
	}
}
