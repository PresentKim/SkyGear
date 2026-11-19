package netstore

import (
	"fmt"
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
	"github.com/presentkim/skygear/content/skygear/storage"
)

// The terminal window (after Refined Storage's grid): a chest of six rows,
// the network's items a page of five rows and the buttons in the last, over
// the vanilla inventory. An item of the list never leaves the window: taking
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

// The slots of the terminal window: the page of the list, a slot left empty
// after it for the items put into the network (a shift click moves a stack
// to the first empty slot), then the buttons of the last row. The marker
// routes the window to the pack's terminal panel (ui/sg_machine.json),
// which draws the counts of the list.
const (
	termPage      = 44
	termListSlots = 45
	termPrev      = 45
	termSearch    = 46
	termSort      = 47
	termInfo      = 48
	termGrid      = 49
	termTasks     = 50
	termTake      = 51
	termPut       = 52
	termNext      = 53
	termSize      = 54
	termMarker    = "§d§m§t"
	// termShown marks the stacks of the list, which are only pictures.
	termShown = "sg_terminal_shown"
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
	w.storage = chestui.NewStorage(termSize, termMarker+"§r"+i18n.T(i18n.Of(p), "netstore.terminal.title"))
	w.fill(p, tx)
	w.storage.Inventory().Handle(termHandler{w})
	w.storage.OnChange(w.changed)
	w.storage.Open(p, nil)
}

// shown is the picture of a kind in the list: one of it, named with its
// count text and the item's name, the count and the clicks in its lore.
func shown(lang string, proto item.Stack, count string, lore ...string) item.Stack {
	st := proto.Grow(1 - proto.Count())
	st = st.WithCustomName(invui.CountMark(count) + "§r" + decor.TipName(lang, proto))
	return st.WithLore(lore...).WithValue(termShown, true)
}

// termButton is a button of the last row: it named with text and lore.
func termButton(it world.Item, text string, lore ...string) item.Stack {
	return item.NewStack(it, 1).WithCustomName(invui.CountMark("") + "§r" + text).WithLore(lore...).WithValue(termShown, true)
}

// fill writes the window's slots: the page of the list and the buttons.
func (w *termWindow) fill(p *player.Player, tx *world.Tx) {
	w.filling = true
	defer func() { w.filling = false }()
	lang := i18n.Of(p)
	t := w.list(p, tx)
	pages := max(1, (t.size()+termPage-1)/termPage)
	w.page = min(max(w.page, 0), pages-1)
	inv := w.storage.Inventory()
	for slot := range termListSlots {
		i := w.page*termPage + slot
		st := item.Stack{}
		switch {
		case slot >= termPage:
		case i < len(t.list):
			e := t.list[i]
			st = shown(lang, e.Item, ui.CountText(int(e.Count)), fmt.Sprintf(i18n.T(lang, "storage.tip"), e.Count))
		case i < t.size():
			st = shown(lang, Proto(t.n.Sess, t.craft[i-len(t.list)]), "+", i18n.T(lang, "netstore.terminal.craftTip"))
		}
		_ = inv.SetItem(slot, st)
	}
	filler := termButton(block.StainedGlassPane{Colour: item.ColourGrey()}, " ")
	for slot := termListSlots; slot < termSize; slot++ {
		_ = inv.SetItem(slot, filler)
	}
	page := fmt.Sprintf(i18n.T(lang, "netstore.terminal.page"), w.page+1, pages)
	if w.page > 0 {
		_ = inv.SetItem(termPrev, termButton(item.Arrow{}, i18n.T(lang, "netstore.terminal.prev"), page))
	}
	if w.page < pages-1 {
		_ = inv.SetItem(termNext, termButton(item.Arrow{}, i18n.T(lang, "netstore.terminal.next"), page))
	}
	search := i18n.T(lang, "netstore.terminal.search")
	if w.query != "" {
		search += "§r  §e" + w.query
	}
	_ = inv.SetItem(termSearch, termButton(item.Spyglass{}, search, i18n.T(lang, "storage.search.label")))
	_ = inv.SetItem(termSort, termButton(item.Clock{}, i18n.T(lang, sortKeys[orders.Of(w.s)[w.h]]), i18n.T(lang, "netstore.terminal.sortNext")))
	if t.n == nil {
		_ = inv.SetItem(termInfo, termButton(machine.Info{}, i18n.T(lang, "netstore.terminal.title"), i18n.T(lang, "netstore.terminal.offline")))
		return
	}
	var items int64
	t.n.Root.Each(func(_ storagenet.Key, c int64) { items += c })
	info := strings.Split(terminalInfo(lang, len(entriesOf(t.n)), items, w.query, t.found), "\n")
	_ = inv.SetItem(termInfo, termButton(machine.Info{}, i18n.T(lang, "netstore.terminal.title"), append(info, page)...))
	if _, ok := gridOf(tx, w.pos); ok {
		_ = inv.SetItem(termGrid, termButton(block.CraftingTable{}, i18n.T(lang, "netstore.terminal.grid")))
	}
	if MonitorHook != nil {
		_ = inv.SetItem(termTasks, termButton(item.Book{}, i18n.T(lang, "netstore.terminal.monitor")))
	}
	_ = inv.SetItem(termTake, termButton(block.Chest{}, i18n.T(lang, "netstore.terminal.takeMatching")))
	_ = inv.SetItem(termPut, termButton(block.Hopper{}, i18n.T(lang, "netstore.terminal.putAll")))
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
		case slot < termPage:
			i := w.page*termPage + slot
			switch {
			case i < len(t.keys):
				k := t.keys[i]
				if takeInto(t.n, p.Inventory(), k, Proto(t.n.Sess, k).MaxCount()) > 0 {
					tx.PlaySound(w.pos.Vec3Centre(), sound.ItemAdd{})
				}
			case i < t.size():
				RequestHook(w.s, p, t.n, t.craft[i-len(t.keys)])
				return
			}
		case slot == termPrev:
			w.page--
		case slot == termNext:
			w.page++
		case slot == termSearch:
			reopen := func(p *player.Player, q string) {
				p.H().Do(func(tx *world.Tx, e world.Entity) { showTerminal(w.s, e.(*player.Player), tx, w.pos, q) })
			}
			chestui.Close(p)
			ui.Input(p, i18n.T(lang, "netstore.terminal.title"), i18n.T(lang, "storage.search.label"), w.query,
				func(p *player.Player, _ *world.Tx, q string) { reopen(p, q) },
				func(p *player.Player, _ *world.Tx) { reopen(p, w.query) })
			return
		case slot == termSort:
			orders.Of(w.s)[w.h] = (orders.Of(w.s)[w.h] + 1) % len(sortKeys)
			w.kept = nil
		case slot == termGrid:
			if _, ok := gridOf(tx, w.pos); ok {
				showGrid(w.s, p, tx, w.pos, w.query)
				return
			}
		case slot == termTasks && MonitorHook != nil && t.n != nil:
			MonitorHook(w.s, p, t.n)
			return
		case slot == termTake && t.n != nil:
			if takeMatching(t.n, p.Inventory(), t.list) > 0 {
				tx.PlaySound(w.pos.Vec3Centre(), sound.ItemAdd{})
			}
		case slot == termPut && t.n != nil:
			if putMain(t.n, p.Inventory()) > 0 {
				tx.PlaySound(w.pos.Vec3Centre(), sound.ItemAdd{})
			}
		}
		w.fill(p, tx)
	})
}

// changed puts an item that came into an empty slot of the list into the
// network; what it does not take goes back to the player.
func (w *termWindow) changed(slot int, _, after item.Stack) {
	if w.filling || slot >= termListSlots || after.Empty() {
		return
	}
	if _, ok := after.Value(termShown); ok {
		return
	}
	w.h.Do(func(tx *world.Tx, e world.Entity) {
		p := e.(*player.Player)
		rest := after.Grow(-int(netInsert(At(tx, w.pos), after)))
		if !rest.Empty() {
			if n, _ := p.Inventory().AddItem(rest); n < rest.Count() {
				p.Drop(rest.Grow(-n))
			}
		} else {
			tx.PlaySound(w.pos.Vec3Centre(), sound.ItemAdd{})
		}
		w.fill(p, tx)
	})
}

// termHandler keeps the list's pictures and the buttons in the window: a
// take is a press, and only an empty slot of the list takes an item.
type termHandler struct{ w *termWindow }

func (h termHandler) HandleTake(ctx *inventory.Context, slot int, _ item.Stack) {
	ctx.Cancel()
	h.w.press(slot)
}

func (h termHandler) HandlePlace(ctx *inventory.Context, slot int, _ item.Stack) {
	if cur, _ := h.w.storage.Inventory().Item(slot); slot < termListSlots && cur.Empty() {
		return
	}
	ctx.Cancel()
}

func (termHandler) HandleDrop(ctx *inventory.Context, _ int, _ item.Stack) { ctx.Cancel() }

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
