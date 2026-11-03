package netstore

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/storagenet"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/decor"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/storage"
)

// The terminal (spec §5): the list of everything the network holds, in one
// scrolling grid under a search box, like the storage controller's. The
// controller opens the same list.

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
)

// ShowTerminal handles a click on a terminal or controller: a sneaking click
// opens the list, any other deposits the held stack, and a second click within
// 500ms also deposits the inventory kinds the network holds.
func ShowTerminal(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) {
	lang := i18n.Of(p)
	n := At(tx, pos)
	if n == nil {
		return
	}
	if !n.Online() {
		p.Message(i18n.T(lang, "netstore.terminal.offline"))
		return
	}
	if p.Sneaking() {
		if !s.Debounced(p) {
			openList(s, p, pos, "")
		}
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

func openList(s *core.Session, p *player.Player, pos cube.Pos, query string) {
	lang := i18n.Of(p)
	again := func(p *player.Player, tx *world.Tx) { openList(s, p, pos, query) }
	find := func(p *player.Player, q string) ui.SearchResult {
		q = strings.TrimSpace(q)
		n := At(p.Tx(), pos)
		if n == nil || !n.Online() {
			return ui.SearchResult{Info: i18n.T(lang, "netstore.terminal.offline")}
		}
		list, keys, found := terminalList(n, lang, q, orders.Of(s)[p.H()])
		var items int64
		n.Root.Each(func(_ storagenet.Key, c int64) { items += c })
		kinds := len(entriesOf(n))
		re := func(p *player.Player, tx *world.Tx) { openList(s, p, pos, q) }
		res := ui.SearchResult{
			Info:     terminalInfo(lang, kinds, items, q, found),
			Sections: []ui.GridSection{{Slots: terminalItemSlots(s, n, lang, list, keys, re)}},
		}
		if CraftableHook != nil && RequestHook != nil {
			if cs := craftSlots(s, n, lang, re); len(cs) > 0 {
				res.Sections = append(res.Sections, ui.GridSection{Heading: i18n.T(lang, "netstore.terminal.craftable"), Slots: cs})
			}
		}
		return res
	}
	opts := []ui.Option{
		ui.BarButton(i18n.T(lang, "storage.depositAll"), "textures/ui/sg/arrow_down", func(p *player.Player, tx *world.Tx) {
			if n := At(tx, pos); n != nil && depositKinds(n, p.Inventory()) > 0 {
				tx.PlaySound(pos.Vec3Centre(), sound.ItemAdd{})
			}
			again(p, tx)
		}),
		ui.BarButton(i18n.T(lang, orderKeys[orders.Of(s)[p.H()]]), "textures/ui/store_filter_icon", func(p *player.Player, tx *world.Tx) {
			orders.Of(s)[p.H()] = (orders.Of(s)[p.H()] + 1) % len(orderKeys)
			again(p, tx)
		}),
	}
	if MonitorHook != nil {
		opts = append(opts, ui.BarButton(i18n.T(lang, "netstore.terminal.monitor"), "textures/ui/sg/arrow_down", func(p *player.Player, tx *world.Tx) {
			if n := At(tx, pos); n != nil {
				MonitorHook(s, p, n)
			}
		}))
	}
	ui.SearchGrid(p, i18n.T(lang, "netstore.terminal.title"), i18n.T(lang, "storage.search.label"), query, find, opts...)
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

func terminalItemSlots(s *core.Session, n *Network, lang string, list []storage.Entry, keys []storagenet.Key, again func(p *player.Player, tx *world.Tx)) []ui.ItemSlot {
	var slots []ui.ItemSlot
	for i, e := range list {
		k := keys[i]
		slots = append(slots, ui.ItemSlot{
			Icon:  ui.ItemIcon(e.Item.Item()),
			Count: "§f" + ui.CountText(int(e.Count)),
			Tip:   decor.TipName(lang, e.Item) + "\n§7" + fmt.Sprintf(i18n.T(lang, "storage.tip"), e.Count),
			Run: func(p *player.Player, tx *world.Tx) {
				if got := netExtract(n, k, e.Item.MaxCount()); !got.Empty() {
					s.GiveItem(p, tx, p.Position(), got)
				}
				again(p, tx)
			},
		})
	}
	return slots
}

func craftSlots(s *core.Session, n *Network, lang string, again func(p *player.Player, tx *world.Tx)) []ui.ItemSlot {
	var slots []ui.ItemSlot
	for _, k := range CraftableHook(n) {
		st := Proto(n.Sess, k)
		if st.Empty() {
			continue
		}
		slots = append(slots, ui.ItemSlot{
			Icon: ui.ItemIcon(st.Item()),
			Tip:  decor.TipName(lang, st) + "\n§7" + i18n.T(lang, "netstore.terminal.craftTip"),
			Run: func(p *player.Player, tx *world.Tx) {
				RequestHook(s, p, n, k)
			},
		})
	}
	return slots
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
