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
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/invclick"
	"github.com/presentkim/skygear/common/storagenet"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/decor"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/storage"
)

// The terminal window (after AE2's terminals): the network's items in a
// scrolling grid under a search box, the terminal's 3×3 crafting grid when
// the player shows it, and the player's inventory, all worked like the
// vanilla inventory (ui.ShowInventory). The grid's items are the terminal's
// inventory slots 0-8, so they stay with the block.

// gridSlots are the terminal's slots that hold its crafting grid.
const gridSlots = 9

// minNetSlots is how many slots the network grid shows at least, so there
// is room to drop a stack into an empty network.
const minNetSlots = 27

// craftShown are the players who show the crafting grid.
var craftShown = core.NewPlayers[bool]()

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

// terminal is the state of a terminal window as drawn.
type terminal struct {
	s     *core.Session
	pos   cube.Pos
	query string
	n     *Network
	keys  []storagenet.Key
	list  []storage.Entry
	craft []storagenet.Key
	found int
	grid  invclick.Area
	book  []bookEntry
}

func openTerminal(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, query string) terminal {
	t := terminal{s: s, pos: pos, query: strings.TrimSpace(query)}
	if n := At(tx, pos); n != nil && n.Online() {
		t.n = n
		lang := i18n.Of(p)
		t.list, t.keys, t.found = terminalList(n, lang, t.query, orders.Of(s)[p.H()])
		if kept, ok := keptOrders.Of(s)[p.H()]; ok {
			if kept != nil {
				t.list, t.keys = keepOrder(n, kept, t.list)
			}
			keptOrders.Of(s)[p.H()] = t.keys
		}
		if CraftableHook != nil && RequestHook != nil {
			for _, k := range CraftableHook(n) {
				pr := Proto(n.Sess, k)
				if pr.Empty() || len(storage.Filter(lang, []storage.Entry{{Item: pr}}, t.query)) == 0 {
					continue
				}
				t.craft = append(t.craft, k)
			}
		}
	}
	if g, ok := gridOf(tx, pos); ok && craftShown.Of(s)[p.H()] {
		t.grid = g
		t.book = bookOf(s, p, t.n, g)
	}
	return t
}

// showTerminal draws the terminal window at pos for p.
func showTerminal(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, query string) {
	t := openTerminal(s, p, tx, pos, query)
	lang := i18n.Of(p)
	inv := p.Inventory()
	tipOf := func(st item.Stack) string {
		if st.Empty() {
			return ""
		}
		return decor.TipName(lang, st)
	}

	netRow := ui.InvRow{Grow: true}
	if t.n == nil {
		netRow.Parts = []ui.InvPart{{Area: "net", Cols: 9, Slots: make([]ui.InvSlot, minNetSlots), Off: true}}
	} else {
		var slots []ui.InvSlot
		for _, e := range t.list {
			slots = append(slots, ui.InvSlot{
				Icon:  ui.ItemIcon(e.Item.Item()),
				Count: "§f" + ui.CountText(int(e.Count)),
				Tip:   tipOf(e.Item) + "\n§7" + fmt.Sprintf(i18n.T(lang, "storage.tip"), e.Count),
			})
		}
		for _, k := range t.craft {
			pr := Proto(t.n.Sess, k)
			slots = append(slots, ui.InvSlot{Icon: ui.ItemIcon(pr.Item()), Count: "§e" + i18n.T(lang, "netstore.terminal.craftTag"), Tip: tipOf(pr) + "\n" + i18n.T(lang, "netstore.terminal.craftTip")})
		}
		for len(slots) < minNetSlots || len(slots)%9 != 0 {
			slots = append(slots, ui.InvSlot{Off: true})
		}
		netRow.Parts = []ui.InvPart{{Area: "net", Cols: 9, Slots: slots}}
	}
	rows := []ui.InvRow{netRow}
	if t.grid != nil {
		cells := gridCells(t.grid)
		grid := make([]ui.InvSlot, len(cells))
		for i, c := range cells {
			grid[i] = ui.StackSlot(c, tipOf(c))
		}
		result := ui.InvSlot{}
		if cr, ok := gridRecipe(s, t.grid); ok {
			result = ui.StackSlot(cr.Out, tipOf(cr.Out))
		}
		clear := ui.Option{Label: i18n.T(lang, "netstore.terminal.clear"), Icon: "textures/ui/close_button_default_light", Run: func(p *player.Player, tx *world.Tx) {
			if g, ok := gridOf(tx, pos); ok {
				clearGrid(At(tx, pos), g, p)
			}
			showTerminal(s, p, tx, pos, query)
		}}
		rows = append(rows, ui.InvRow{Parts: []ui.InvPart{
			{Area: "craft", Cols: 3, Slots: grid},
			{Button: &clear},
			{Arrow: true},
			{Area: "result", Cols: 1, Big: true, Slots: []ui.InvSlot{result}},
		}})
	}
	main, hot := invclick.Main(inv), invclick.Hotbar(inv)
	again := func(p *player.Player, tx *world.Tx) { showTerminal(s, p, tx, pos, query) }
	if t.n != nil {
		rows[0].Bar = netTakeBar(s, pos, lang, query, again).Options()
	}
	invRow := ui.InvRow{Heading: i18n.T(lang, "netstore.terminal.inventory"), Parts: []ui.InvPart{{Area: "inv", Cols: 9, Slots: areaSlots(main, tipOf)}}, BarTop: true}
	invRow.Bar = netPutBar(pos, lang, t.n != nil, again).Options()
	rows = append(rows, invRow, ui.InvRow{Parts: []ui.InvPart{{Area: "hotbar", Cols: 9, Slots: areaSlots(hot, tipOf)}}})

	info := ""
	if t.n != nil {
		var items int64
		t.n.Root.Each(func(_ storagenet.Key, c int64) { items += c })
		info = terminalInfo(lang, len(entriesOf(t.n)), items, t.query, t.found)
	}
	var book *ui.InvBook
	if t.grid != nil {
		book = bookPanel(s, p, t.book, again)
	}
	ui.ShowInventory(p, ui.InventoryScreen{
		Book:   book,
		Title:  i18n.T(lang, "netstore.terminal.title"),
		Search: &ui.InvSearch{Query: query, Placeholder: i18n.T(lang, "storage.search.label"), Change: func(p *player.Player, tx *world.Tx, q string) { showTerminal(s, p, tx, pos, q) }},
		Info:   info,
		Rows:   rows,
		Tools:  terminalTools(s, p, tx, pos, lang, again),
		Side:   terminalSide(s, pos, lang),
		Quick:  i18n.T(lang, "netstore.terminal.quick"),
		Slot: func(p *player.Player, tx *world.Tx, a ui.SlotAct) {
			if openTerminal(s, p, tx, pos, query).slot(p, a) {
				again(p, tx)
			}
		},
		Drag: func(p *player.Player, tx *world.Tx, d ui.DragAct) {
			openTerminal(s, p, tx, pos, query).drag(p, d)
			again(p, tx)
		},
		Hold: func(p *player.Player, tx *world.Tx, on bool) {
			_, was := keptOrders.Of(s)[p.H()]
			switch {
			case on && !was:
				keptOrders.Of(s)[p.H()] = nil
			case !on && was:
				delete(keptOrders.Of(s), p.H())
				again(p, tx)
			}
		},
		Again: again,
		Size:  "terminal",
	})
}

func areaSlots(a invclick.Area, tipOf func(item.Stack) string) []ui.InvSlot {
	out := make([]ui.InvSlot, a.Len())
	for i := range out {
		st := a.Item(i)
		out[i] = ui.StackSlot(st, tipOf(st))
	}
	return out
}

// terminalTools are the buttons left of the terminal window, as Refined
// Storage's grid: the sort direction, the sort kind and the crafting grid.
// The orders (orderKeys) are count high to low, count low to high, name
// A to Z and name Z to A.
func terminalTools(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, lang string, again func(p *player.Player, tx *world.Tx)) []ui.Option {
	o := orders.Of(s)[p.H()]
	desc := o == 0 || o == 3
	dirKey, dirIcon := "netstore.terminal.sortAsc", "textures/ui/sg/arrow_up"
	if desc {
		dirKey, dirIcon = "netstore.terminal.sortDesc", "textures/ui/sg/arrow_down"
	}
	kindKey, kindIcon := "netstore.terminal.sortCount", "textures/ui/sg/menu_stats"
	if o >= 2 {
		kindKey, kindIcon = "netstore.terminal.sortName", "textures/ui/sg/menu_codex"
	}
	tools := []ui.Option{
		{Label: i18n.T(lang, dirKey), Icon: dirIcon, Run: func(p *player.Player, tx *world.Tx) {
			orders.Of(s)[p.H()] = [4]int{1, 0, 3, 2}[orders.Of(s)[p.H()]]
			again(p, tx)
		}},
		{Label: i18n.T(lang, kindKey), Icon: kindIcon, Run: func(p *player.Player, tx *world.Tx) {
			orders.Of(s)[p.H()] = [4]int{3, 2, 1, 0}[orders.Of(s)[p.H()]]
			again(p, tx)
		}},
	}
	if _, ok := gridOf(tx, pos); ok {
		key := "netstore.terminal.craftShow"
		if craftShown.Of(s)[p.H()] {
			key = "netstore.terminal.craftHide"
		}
		tools = append(tools, ui.Option{Label: i18n.T(lang, key), Icon: ui.ItemIcon(block.CraftingTable{}), Run: func(p *player.Player, tx *world.Tx) {
			craftShown.Of(s)[p.H()] = !craftShown.Of(s)[p.H()]
			again(p, tx)
		}})
	}
	return tools
}

// terminalSide is the column right of the window: the crafting jobs.
func terminalSide(s *core.Session, pos cube.Pos, lang string) []ui.Option {
	if MonitorHook == nil {
		return nil
	}
	return []ui.Option{{Label: i18n.T(lang, "netstore.terminal.monitor"), Icon: "textures/ui/sg/menu_stats", Run: func(p *player.Player, tx *world.Tx) {
		if n := At(tx, pos); n != nil {
			MonitorHook(s, p, n)
		}
	}}}
}

// netTakeBar is the network list's transfer bar: take stacks of the list,
// in its order, until the inventory is full; top up the kinds the
// inventory holds. The list sorts itself, so it has no sort button.
func netTakeBar(s *core.Session, pos cube.Pos, lang, query string, again func(p *player.Player, tx *world.Tx)) ui.TransferBar {
	run := func(take func(n *Network, inv *inventory.Inventory, list []storage.Entry) int) func(p *player.Player, tx *world.Tx) {
		return func(p *player.Player, tx *world.Tx) {
			if t := openTerminal(s, p, tx, pos, query); t.n != nil && take(t.n, p.Inventory(), t.list) > 0 {
				tx.PlaySound(pos.Vec3Centre(), sound.ItemAdd{})
			}
			again(p, tx)
		}
	}
	return ui.TransferBar{
		All:      ui.Option{Label: i18n.T(lang, "netstore.terminal.takeAll"), Icon: "textures/ui/sg/arrow_down", Run: run(takeAll)},
		Matching: ui.Option{Label: i18n.T(lang, "netstore.terminal.takeMatching"), Icon: "textures/ui/store_filter_icon", Run: run(takeMatching)},
	}
}

// netPutBar is the inventory's transfer bar: put the inventory (not the
// hotbar) into the network, put the kinds the network holds, sort the
// inventory. The put buttons need a network that is on.
func netPutBar(pos cube.Pos, lang string, online bool, again func(p *player.Player, tx *world.Tx)) ui.TransferBar {
	put := func(f func(n *Network, inv *inventory.Inventory) int64) func(p *player.Player, tx *world.Tx) {
		return func(p *player.Player, tx *world.Tx) {
			if n := At(tx, pos); n != nil && n.Online() && f(n, p.Inventory()) > 0 {
				tx.PlaySound(pos.Vec3Centre(), sound.ItemAdd{})
			}
			again(p, tx)
		}
	}
	bar := ui.TransferBar{Sort: ui.Option{Label: i18n.T(lang, "netstore.terminal.sortInventory"), Icon: "textures/ui/sg/menu_codex", Run: func(p *player.Player, tx *world.Tx) {
		sortMain(p.Inventory())
		again(p, tx)
	}}}
	if online {
		bar.All = ui.Option{Label: i18n.T(lang, "netstore.terminal.putAll"), Icon: "textures/ui/sg/arrow_up", Run: put(putMain)}
		bar.Matching = ui.Option{Label: i18n.T(lang, "netstore.terminal.putMatching"), Icon: "textures/ui/store_filter_icon", Run: put(depositKinds)}
	}
	return bar
}

func (t terminal) area(p *player.Player, name string) (invclick.Area, bool) {
	inv := p.Inventory()
	switch name {
	case "inv":
		return invclick.Main(inv), true
	case "hotbar":
		return invclick.Hotbar(inv), true
	case "craft":
		return t.grid, t.grid != nil
	}
	return nil, false
}

// slot runs a click; it reports whether the window is to be drawn again (a
// craft request shows its own screen).
func (t terminal) slot(p *player.Player, a ui.SlotAct) bool {
	inv := p.Inventory()
	main, hot := invclick.Main(inv), invclick.Hotbar(inv)
	cur := a.Carried
	switch a.Area {
	case "net":
		if a.Index >= len(t.keys) && a.Index < len(t.keys)+len(t.craft) && cur.Empty() {
			RequestHook(t.s, p, t.n, t.craft[a.Index-len(t.keys)])
			return false
		}
		var key storagenet.Key
		if a.Index < len(t.keys) {
			key = t.keys[a.Index]
		}
		cur = netClick(t.n, key, cur, a.Right, a.Shift, hot, main)
	case "result":
		if t.grid == nil {
			return true
		}
		cur = craftClick(t.s, t.n, t.grid, cur, a.Shift, p)
	case "book":
		if t.grid == nil || a.Index >= len(t.book) {
			return true
		}
		cur = bookClick(t.s, t.n, t.grid, t.book[a.Index].cr, cur, a.Right, a.Shift, p)
	default:
		area, ok := t.area(p, a.Area)
		if !ok || a.Index >= area.Len() {
			return true
		}
		delete(splits.Of(t.s), p.H())
		switch {
		case a.Shift && a.Double && !cur.Empty() && a.Area != "craft":
			focus := area.Item(a.Index)
			if f, ok := shiftFocus.Of(t.s)[p.H()]; ok && f.ref == (ui.SlotRef{Area: a.Area, Index: a.Index}) {
				focus = f.item
			}
			if !focus.Empty() && t.n != nil {
				invclick.MoveAll(focus, func(st item.Stack) item.Stack { return st.Grow(-int(netInsert(t.n, st))) }, hot, main)
			}
		case a.Shift:
			st := area.Item(a.Index)
			if st.Empty() {
				break
			}
			shiftFocus.Of(t.s)[p.H()] = focusSlot{ui.SlotRef{Area: a.Area, Index: a.Index}, st}
			switch {
			case a.Area == "craft":
				st = invclick.Move(st, hot, main)
			case t.n != nil:
				st = st.Grow(-int(netInsert(t.n, st)))
			case a.Area == "hotbar":
				st = invclick.Move(st, main)
			default:
				st = invclick.Move(st, hot)
			}
			area.Set(a.Index, st)
		case a.Double && !a.Right && !cur.Empty():
			areas := []invclick.Area{hot, main}
			if t.grid != nil {
				areas = append(areas, t.grid)
			}
			cur = invclick.Collect(cur, areas...)
		default:
			var sp *invclick.Split
			if a.Split && !cur.Empty() {
				sp = invclick.StartSplit(invclick.Ref{Name: a.Area, A: area, I: a.Index}, cur, a.Right)
			}
			cur = invclick.Click(area, a.Index, cur, a.Right)
			if sp != nil && sp.Placed() {
				splits.Of(t.s)[p.H()] = sp
			}
		}
	}
	ui.SetCarried(p, cur)
	return true
}

// drag spreads the stack held when the drag's press came over the slots it
// reached (invclick.Split); a press that started no split leaves it.
func (t terminal) drag(p *player.Player, d ui.DragAct) {
	sp := splits.Of(t.s)[p.H()]
	if sp == nil {
		return
	}
	var refs []invclick.Ref
	for _, r := range d.Slots {
		if area, ok := t.area(p, r.Area); ok && r.Index < area.Len() {
			refs = append(refs, invclick.Ref{Name: r.Area, A: area, I: r.Index})
		}
	}
	ui.SetCarried(p, sp.Spread(refs))
}

// focusSlot is the stack a shift click took out of a slot: a shift double
// click on that slot moves every stack of the same item.
type focusSlot struct {
	ref  ui.SlotRef
	item item.Stack
}

var (
	splits     = core.NewPlayers[*invclick.Split]()
	shiftFocus = core.NewPlayers[focusSlot]()
)

// netClick is a click on a network slot holding key ("" an empty slot)
// with cur on the cursor: with nothing held, a stack (left), half a stack
// (right) or a stack into the areas (shift) comes out; a held stack goes in
// whole (left) or one item (right). It returns the cursor stack.
func netClick(n *Network, key storagenet.Key, cur item.Stack, right, shift bool, to ...invclick.Area) item.Stack {
	if n == nil {
		return cur
	}
	if !cur.Empty() {
		put := cur
		if right {
			put = cur.Grow(1 - cur.Count())
		}
		return cur.Grow(-int(netInsert(n, put)))
	}
	if key == "" {
		return cur
	}
	stack := Proto(n.Sess, key).MaxCount()
	switch {
	case shift:
		if rest := invclick.Move(netExtract(n, key, stack), to...); !rest.Empty() {
			netInsert(n, rest)
		}
		return cur
	case right:
		return netExtract(n, key, (stack+1)/2)
	}
	return netExtract(n, key, stack)
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

// craftClick is a click on the result slot: one craft onto the cursor when
// it can hold the result, or with shift as many as the inventory takes.
func craftClick(s *core.Session, n *Network, grid invclick.Area, cur item.Stack, shift bool, p *player.Player) item.Stack {
	if shift {
		inv := p.Inventory()
		for range 64 {
			out, ok := craftOnce(s, n, grid)
			if !ok {
				break
			}
			if rest := invclick.Move(out, invclick.Hotbar(inv), invclick.Main(inv)); !rest.Empty() {
				if n != nil {
					rest = rest.Grow(-int(netInsert(n, rest)))
				}
				if !rest.Empty() {
					p.Drop(rest)
				}
				break
			}
		}
		return cur
	}
	cr, ok := gridRecipe(s, grid)
	if !ok {
		return cur
	}
	if !cur.Empty() && (!cur.Comparable(cr.Out) || cur.Count()+cr.Out.Count() > cur.MaxCount()) {
		return cur
	}
	out, _ := craftOnce(s, n, grid)
	if cur.Empty() {
		return out
	}
	return cur.Grow(out.Count())
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
