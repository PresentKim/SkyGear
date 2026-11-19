package netstore

import (
	"fmt"
	"slices"
	"strings"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/presentkim/skygear/common/chestui"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/invclick"
	"github.com/presentkim/skygear/common/invui"
	"github.com/presentkim/skygear/common/storagenet"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/decor"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/netstore/termlayout"
	"github.com/presentkim/skygear/content/skygear/storage"
)

// The terminal window (after Refined Storage's grid): the network's items a
// page of twelve rows the pack scrolls apart from the vanilla inventory, and
// the buttons in a column left of it. An item of the list never leaves the window: taking
// it puts a stack of its kind into the inventory. An item put on an empty
// slot of the list, or moved there with a shift click, goes into the
// network. The terminal's 3×3 crafting grid has a window of its own; its
// items are the terminal's inventory slots 0-8, so they stay with the block.

// gridSlots are the terminal's slots that hold its crafting grid.
const gridSlots = 9

// gridArea is a terminal's crafting grid.
type gridArea struct{ inv *inventory.Inventory }

func (g gridArea) Len() int { return gridSlots }

func (g gridArea) Item(i int) item.Stack {
	st, _ := g.inv.Item(i)
	return st
}

func (g gridArea) Set(i int, st item.Stack) { _ = g.inv.SetItem(i, st) }

func (gridArea) Accepts(int, item.Stack) bool { return true }

func gridOf(tx *world.Tx, pos cube.Pos) (gridArea, bool) {
	st := stateAt(tx, pos)
	if st == nil || st.Def != machine.Kinds[machine.KindNetTerminal] {
		return gridArea{}, false
	}
	return gridArea{st.Inv()}, true
}

func gridCells(g invclick.Area) []item.Stack {
	cells := make([]item.Stack, g.Len())
	for i := range cells {
		cells[i] = g.Item(i)
	}
	return cells
}

// The slots of the terminal window are termlayout's. The marker routes the
// window to the pack's terminal panel (ui/sg_terminal.json).
const (
	termPage   = termlayout.Kinds
	termMarker = "§d§m§t"
	// termShown marks the stacks of the window that are only pictures or
	// buttons; termCraft a take slot's stack of a kind the network crafts.
	termShown = "sg_terminal_shown"
	termCraft = "sg_terminal_craft"
)

// terminal is the network's list for one player.
type terminal struct {
	n     *Network
	list  []storage.Entry
	keys  []storagenet.Key
	craft []storagenet.Key
	found int
}

func openTerminal(p *player.Player, tx *world.Tx, s *core.Session, pos cube.Pos, query string) terminal {
	var t terminal
	n := At(tx, pos)
	if n == nil || !n.Online() {
		return t
	}
	t.n = n
	lang := i18n.Of(p)
	t.list, t.keys, t.found = terminalList(n, lang, query, orders.Of(s)[p.H()])
	if CraftableHook != nil && RequestHook != nil {
		for _, k := range CraftableHook(n) {
			pr := Proto(n.Sess, k)
			if pr.Empty() || len(storage.Filter(lang, []storage.Entry{{Item: pr}}, query)) == 0 {
				continue
			}
			t.craft = append(t.craft, k)
		}
	}
	return t
}

// size is how many kinds the list shows: the stored ones, then those the
// network can craft.
func (t terminal) size() int { return len(t.keys) + len(t.craft) }

// termWindow is a terminal window open for one player.
type termWindow struct {
	s       *core.Session
	h       *world.EntityHandle
	pos     cube.Pos
	query   string
	page    int
	storage *chestui.Storage
	// kept is the order of the list when the window opened or was sorted,
	// kept while the player takes and puts so the page does not shuffle.
	kept []storagenet.Key
	// filling is set while the window's slots are written.
	filling bool
}

// list is the network's list for the window, in its kept order without the
// kinds that ran out.
func (w *termWindow) list(p *player.Player, tx *world.Tx) terminal {
	t := openTerminal(p, tx, w.s, w.pos, w.query)
	if t.n == nil {
		return t
	}
	if w.kept == nil {
		w.kept = t.keys
		return t
	}
	list, keys := keepOrder(t.n, w.kept, t.list)
	t.list, t.keys = t.list[:0], t.keys[:0]
	for i, e := range list {
		if e.Count > 0 {
			t.list, t.keys = append(t.list, e), append(t.keys, keys[i])
		}
	}
	return t
}

// showTerminal opens the terminal window at pos for p with the list
// filtered by query.
func showTerminal(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, query string) {
	w := &termWindow{s: s, h: p.H(), pos: pos, query: strings.TrimSpace(query)}
	w.storage = chestui.NewStorage(termlayout.Size, termMarker+"§r"+i18n.T(i18n.Of(p), "netstore.terminal.title"))
	w.fill(p, tx)
	w.storage.Inventory().Handle(termHandler{w})
	w.storage.OnChange(w.changed)
	w.storage.Open(p, nil)
}

// shown is the picture of a kind in its show slot: one of it, named with its
// count text and the item's name, the count and the clicks in its lore.
func shown(lang string, proto item.Stack, count string, lore ...string) item.Stack {
	st := proto.Grow(1 - proto.Count())
	st = st.WithCustomName(invui.CountMark(count) + "§r" + i18n.Plain(decor.TipName(lang, proto)))
	return st.WithLore(lore...).WithValue(termShown, true)
}

// termButton is a button of the last row: it named with text and lore.
func termButton(it world.Item, text string, lore ...string) item.Stack {
	return item.NewStack(it, 1).WithCustomName(invui.CountMark("") + "§r" + text).WithLore(lore...).WithValue(termShown, true)
}

// fill writes the window's slots: the insert slots empty, the page of the
// list as take and show slots, and the buttons. Only the slots that change
// are written: a slot written again is drawn empty for a moment.
func (w *termWindow) fill(p *player.Player, tx *world.Tx) {
	w.filling = true
	defer func() { w.filling = false }()
	inv := w.storage.Inventory()
	want := make([]item.Stack, termlayout.Size)
	set := func(slot int, st item.Stack) { want[slot] = st }
	defer func() {
		for slot, st := range want {
			if old, _ := inv.Item(slot); !sameStack(old, st) {
				_ = inv.SetItem(slot, st)
			}
		}
	}()
	lang := i18n.Of(p)
	t := w.list(p, tx)
	pages := max(1, (t.size()+termPage-1)/termPage)
	w.page = min(max(w.page, 0), pages-1)
	for slot := range termlayout.Take0 {
		set(slot, item.Stack{})
	}
	for k := range termlayout.Kinds {
		i := w.page*termPage + k
		take, show := item.Stack{}, item.Stack{}
		switch {
		case i < len(t.list):
			e := t.list[i]
			take = e.Item.Grow(int(min(e.Count, int64(e.Item.MaxCount()))) - e.Item.Count())
			show = shown(lang, e.Item, ui.CountText(int(e.Count)), fmt.Sprintf(i18n.T(lang, "storage.tip"), e.Count))
		case i < t.size():
			proto := Proto(t.n.Sess, t.craft[i-len(t.list)])
			take = proto.Grow(1 - proto.Count()).WithValue(termCraft, true)
			show = shown(lang, proto, "+", i18n.T(lang, "netstore.terminal.craftTip"))
		}
		set(termlayout.Take0+k, take)
		set(termlayout.Show0+k, show)
	}
	for slot := termlayout.Button0; slot < termlayout.Size; slot++ {
		set(slot, item.Stack{})
	}
	page := fmt.Sprintf(i18n.T(lang, "netstore.terminal.page"), w.page+1, pages)
	if w.page > 0 {
		set(termlayout.Prev, termButton(item.Arrow{}, i18n.T(lang, "netstore.terminal.prev"), page))
	}
	if w.page < pages-1 {
		set(termlayout.Next, termButton(item.Arrow{}, i18n.T(lang, "netstore.terminal.next"), page))
	}
	search := i18n.T(lang, "netstore.terminal.search")
	if w.query != "" {
		search += "§r  §e" + w.query
	}
	set(termlayout.Search, termButton(item.Spyglass{}, search, i18n.T(lang, "storage.search.label")))
	set(termlayout.Sort, termButton(item.Clock{}, i18n.T(lang, sortKeys[orders.Of(w.s)[w.h]]), i18n.T(lang, "netstore.terminal.sortNext")))
	if t.n == nil {
		set(termlayout.Info, termButton(machine.Info{}, i18n.T(lang, "netstore.terminal.title"), i18n.T(lang, "netstore.terminal.offline")))
		return
	}
	var items int64
	t.n.Root.Each(func(_ storagenet.Key, c int64) { items += c })
	info := strings.Split(terminalInfo(lang, len(entriesOf(t.n)), items, w.query, t.found), "\n")
	set(termlayout.Info, termButton(machine.Info{}, i18n.T(lang, "netstore.terminal.title"), append(info, page)...))
	if _, ok := gridOf(tx, w.pos); ok {
		set(termlayout.Grid, termButton(block.CraftingTable{}, i18n.T(lang, "netstore.terminal.grid")))
	}
	if MonitorHook != nil {
		set(termlayout.Tasks, termButton(item.Book{}, i18n.T(lang, "netstore.terminal.monitor")))
	}
	set(termlayout.TakeMatching, termButton(block.Chest{}, i18n.T(lang, "netstore.terminal.takeMatching")))
	set(termlayout.PutAll, termButton(block.Hopper{}, i18n.T(lang, "netstore.terminal.putAll")))
}

// moved is what left a slot and what came into it when its stack went from
// before to after.
func moved(before, after item.Stack) (taken, put item.Stack) {
	switch {
	case before.Empty():
		return item.Stack{}, after
	case after.Empty():
		return before, item.Stack{}
	case !after.Comparable(before):
		return before, after
	case after.Count() < before.Count():
		return before.Grow(-after.Count()), item.Stack{}
	default:
		return item.Stack{}, after.Grow(-before.Count())
	}
}

func sameStack(a, b item.Stack) bool {
	if a.Empty() || b.Empty() {
		return a.Empty() == b.Empty()
	}
	return a.Comparable(b) && a.Count() == b.Count() && a.CustomName() == b.CustomName() && slices.Equal(a.Lore(), b.Lore())
}

// sortKeys name the orders of the list by their index (terminalList).
var sortKeys = [4]string{"netstore.terminal.sortCountDesc", "netstore.terminal.sortCountAsc", "netstore.terminal.sortNameAsc", "netstore.terminal.sortNameDesc"}

// press runs what pressing slot does, in the player's world after the
// window's answer, and writes the window again.
func (w *termWindow) press(slot int) {
	w.h.Do(func(tx *world.Tx, e world.Entity) {
		p := e.(*player.Player)
		t := w.list(p, tx)
		lang := i18n.Of(p)
		switch {
		case slot >= termlayout.Take0 && slot < termlayout.Show0:
			if i := w.page*termPage + slot - termlayout.Take0; i >= len(t.keys) && i < t.size() && RequestHook != nil {
				RequestHook(w.s, p, t.n, t.craft[i-len(t.keys)])
				return
			}
		case slot == termlayout.Prev:
			w.page--
		case slot == termlayout.Next:
			w.page++
		case slot == termlayout.Search:
			reopen := func(p *player.Player, q string) {
				p.H().Do(func(tx *world.Tx, e world.Entity) { showTerminal(w.s, e.(*player.Player), tx, w.pos, q) })
			}
			chestui.Close(p)
			ui.Input(p, i18n.T(lang, "netstore.terminal.title"), i18n.T(lang, "storage.search.label"), w.query,
				func(p *player.Player, _ *world.Tx, q string) { reopen(p, q) },
				func(p *player.Player, _ *world.Tx) { reopen(p, w.query) })
			return
		case slot == termlayout.Sort:
			orders.Of(w.s)[w.h] = (orders.Of(w.s)[w.h] + 1) % len(sortKeys)
			w.kept = nil
		case slot == termlayout.Grid:
			if _, ok := gridOf(tx, w.pos); ok {
				showGrid(w.s, p, tx, w.pos, w.query)
				return
			}
		case slot == termlayout.Tasks && MonitorHook != nil && t.n != nil:
			MonitorHook(w.s, p, t.n)
			return
		case slot == termlayout.TakeMatching && t.n != nil:
			if takeMatching(t.n, p.Inventory(), t.list) > 0 {
				tx.PlaySound(w.pos.Vec3Centre(), sound.ItemAdd{})
			}
		case slot == termlayout.PutAll && t.n != nil:
			if putMain(t.n, p.Inventory()) > 0 {
				tx.PlaySound(w.pos.Vec3Centre(), sound.ItemAdd{})
			}
		}
		w.fill(p, tx)
	})
}

// changed follows what the player moved in the window: items that left a
// take slot come out of the network, items that came into an insert or take
// slot go into it (what it does not take goes back to the player), and the
// window is written again.
func (w *termWindow) changed(slot int, before, after item.Stack) {
	if w.filling || slot >= termlayout.Show0 {
		return
	}
	taken, put := moved(before, after)
	if taken.Empty() && put.Empty() {
		return
	}
	if _, craft := taken.Value(termCraft); craft {
		taken = item.Stack{}
	}
	w.h.Do(func(tx *world.Tx, e world.Entity) {
		p := e.(*player.Player)
		n := At(tx, w.pos)
		if !taken.Empty() {
			if short := taken.Count() - netExtract(n, KeyOf(taken), taken.Count()).Count(); short > 0 {
				_ = p.Inventory().RemoveItem(taken.Grow(short - taken.Count()))
			}
		}
		if !put.Empty() {
			rest := put.Grow(-int(netInsert(n, put)))
			if !rest.Empty() {
				if got, _ := p.Inventory().AddItem(rest); got < rest.Count() {
					p.Drop(rest.Grow(-got))
				}
			} else {
				tx.PlaySound(w.pos.Vec3Centre(), sound.ItemAdd{})
			}
		}
		w.fill(p, tx)
	})
}

// termHandler lets the player take from the take slots and put into the
// insert and take slots; a take of a craftable kind's stack or of a button is
// a press.
type termHandler struct{ w *termWindow }

func (h termHandler) HandleTake(ctx *inventory.Context, slot int, it item.Stack) {
	_, craft := it.Value(termCraft)
	if slot >= termlayout.Take0 && slot < termlayout.Show0 && !craft {
		return
	}
	ctx.Cancel()
	if slot >= termlayout.Take0 {
		h.w.press(slot)
	}
}

// HandlePlace lets an item in only when the network takes all of it, so a
// put the network has no room for is refused, not given back.
func (h termHandler) HandlePlace(ctx *inventory.Context, slot int, it item.Stack) {
	if slot < termlayout.Show0 {
		if p, ok := ctx.Val().(*player.Player); ok && netFits(At(p.Tx(), h.w.pos), it) {
			return
		}
	}
	ctx.Cancel()
}

func (h termHandler) HandleDrop(ctx *inventory.Context, slot int, it item.Stack) {
	if _, craft := it.Value(termCraft); slot >= termlayout.Take0 && slot < termlayout.Show0 && !craft {
		return
	}
	ctx.Cancel()
}

// craftOnce makes the grid's recipe once if the island of s has it open:
// one item leaves each cell, and a cell left empty takes the same item from
// the network again (n may be nil). It returns what was made.
func craftOnce(s *core.Session, n *Network, grid invclick.Area) (item.Stack, bool) {
	cr, ok := gridRecipe(s, grid)
	if !ok {
		return item.Stack{}, false
	}
	for i := range grid.Len() {
		st := grid.Item(i)
		if st.Empty() {
			continue
		}
		grid.Set(i, st.Grow(-1))
		if st.Count() == 1 && n != nil && n.Online() {
			grid.Set(i, netExtract(n, KeyOf(st), 1))
		}
	}
	return cr.Out, true
}

// clearGrid puts the grid's items into the network, the rest into p's
// inventory, and drops what fits nowhere.
func clearGrid(n *Network, grid invclick.Area, p *player.Player) {
	inv := p.Inventory()
	for i := range grid.Len() {
		st := grid.Item(i)
		if st.Empty() {
			continue
		}
		if n != nil && n.Online() {
			st = st.Grow(-int(netInsert(n, st)))
		}
		if !st.Empty() {
			st = invclick.Move(st, invclick.Hotbar(inv), invclick.Main(inv))
		}
		if !st.Empty() {
			p.Drop(st)
		}
		grid.Set(i, item.Stack{})
	}
}
