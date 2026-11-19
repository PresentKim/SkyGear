package netstore

import (
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
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/decor"
	"github.com/presentkim/skygear/content/skygear/formicon"
	"github.com/presentkim/skygear/content/skygear/machine"
)

// The crafting window of a terminal: a chest of three rows with the
// terminal's 3×3 grid on the left, the result in the middle and the buttons
// on the right. The grid holds real items, kept in the terminal's slots
// 0-8; taking the result crafts once into the inventory, and a cell left
// empty by a craft takes the same item from the network again.

// gridWindowSlots are the window slots of the grid cells, row by row.
var gridWindowSlots = [gridSlots]int{0, 1, 2, 9, 10, 11, 18, 19, 20}

// The other slots of the crafting window.
const (
	gridResult = 13
	gridStack  = 15
	gridBook   = 16
	gridClear  = 17
	gridBack   = 26
	gridSize   = 27
	// gridMarker routes the window to the pack's crafting panel
	// (ui/sg_machine.json).
	gridMarker = "§d§m§g"
)

// gridWindow is a crafting window open for one player.
type gridWindow struct {
	s       *core.Session
	h       *world.EntityHandle
	pos     cube.Pos
	query   string
	storage *chestui.Storage
	filling bool
}

func showGrid(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, query string) {
	w := &gridWindow{s: s, h: p.H(), pos: pos, query: query}
	w.storage = chestui.NewStorage(gridSize, gridMarker+"§r"+i18n.T(i18n.Of(p), "netstore.terminal.grid"))
	w.fill(p, tx)
	w.storage.Inventory().Handle(gridHandler{w})
	w.storage.OnChange(w.changed)
	w.storage.Open(p, nil)
}

func cellOf(slot int) (int, bool) {
	for i, s := range gridWindowSlots {
		if s == slot {
			return i, true
		}
	}
	return 0, false
}

// fill writes the window from the terminal's grid.
func (w *gridWindow) fill(p *player.Player, tx *world.Tx) {
	w.filling = true
	defer func() { w.filling = false }()
	lang := i18n.Of(p)
	inv := w.storage.Inventory()
	for slot := range gridSize {
		_ = inv.SetItem(slot, item.Stack{})
	}
	g, ok := gridOf(tx, w.pos)
	if !ok {
		return
	}
	for i, slot := range gridWindowSlots {
		_ = inv.SetItem(slot, g.Item(i))
	}
	if cr, ok := gridRecipe(w.s, g); ok {
		_ = inv.SetItem(gridResult, cr.Out.WithLore(i18n.T(lang, "netstore.terminal.craftOnce")).WithValue(termShown, true))
		_ = inv.SetItem(gridStack, termButton(cr.Out.Item(), i18n.T(lang, "netstore.terminal.craftStack")))
	}
	_ = inv.SetItem(gridBook, termButton(item.Book{}, i18n.T(lang, "netstore.terminal.book")))
	_ = inv.SetItem(gridClear, termButton(block.Hopper{}, i18n.T(lang, "netstore.terminal.clear")))
	_ = inv.SetItem(gridBack, termButton(machine.Machine{Kind: machine.KindNetTerminal}, i18n.T(lang, "netstore.terminal.back")))
}

// press runs what pressing slot does and writes the window again.
func (w *gridWindow) press(slot int) {
	w.h.Do(func(tx *world.Tx, e world.Entity) {
		p := e.(*player.Player)
		g, ok := gridOf(tx, w.pos)
		if !ok {
			return
		}
		n := At(tx, w.pos)
		switch slot {
		case gridResult:
			if craftInto(w.s, n, g, p, 1) > 0 {
				tx.PlaySound(w.pos.Vec3Centre(), sound.ItemAdd{})
			}
		case gridStack:
			if cr, ok := gridRecipe(w.s, g); ok && craftInto(w.s, n, g, p, max(1, cr.Out.MaxCount())) > 0 {
				tx.PlaySound(w.pos.Vec3Centre(), sound.ItemAdd{})
			}
		case gridBook:
			showBook(w.s, p, w.pos, w.query)
			return
		case gridClear:
			clearGrid(n, g, p)
		case gridBack:
			showTerminal(w.s, p, tx, w.pos, w.query)
			return
		}
		w.fill(p, tx)
	})
}

// changed keeps the terminal's grid to the window's cells.
func (w *gridWindow) changed(slot int, _, after item.Stack) {
	i, ok := cellOf(slot)
	if w.filling || !ok {
		return
	}
	w.h.Do(func(tx *world.Tx, e world.Entity) {
		if g, ok := gridOf(tx, w.pos); ok {
			g.Set(i, after)
			w.fill(e.(*player.Player), tx)
		}
	})
}

// craftInto crafts the grid's recipe until want items are made, into p's
// inventory (the network, then the ground, takes what does not fit), and
// returns how many were made.
func craftInto(s *core.Session, n *Network, grid invclick.Area, p *player.Player, want int) int {
	made := 0
	for made < want {
		out, ok := craftOnce(s, n, grid)
		if !ok {
			break
		}
		made += out.Count()
		inv := p.Inventory()
		if rest := invclick.Move(out, invclick.Hotbar(inv), invclick.Main(inv)); !rest.Empty() {
			if n != nil && n.Online() {
				rest = rest.Grow(-int(netInsert(n, rest)))
			}
			if !rest.Empty() {
				p.Drop(rest)
			}
			break
		}
	}
	return made
}

// gridHandler lets the grid cells hold items; the rest of the window is
// buttons.
type gridHandler struct{ w *gridWindow }

func (h gridHandler) HandleTake(ctx *inventory.Context, slot int, _ item.Stack) {
	if _, ok := cellOf(slot); ok {
		return
	}
	ctx.Cancel()
	h.w.press(slot)
}

func (h gridHandler) HandlePlace(ctx *inventory.Context, slot int, _ item.Stack) {
	if _, ok := cellOf(slot); !ok {
		ctx.Cancel()
	}
}

func (h gridHandler) HandleDrop(ctx *inventory.Context, slot int, _ item.Stack) {
	if _, ok := cellOf(slot); !ok {
		ctx.Cancel()
	}
}

// showBook shows the recipe book in the menu window: the tabs, the recipes
// of the open tab (bookOf), the filter and the search. Pressing a recipe
// lays it into the grid and goes back to the crafting window, opened after
// the press so the menu window closing does not close it.
func showBook(s *core.Session, p *player.Player, pos cube.Pos, query string) {
	tx := p.Tx()
	g, ok := gridOf(tx, pos)
	if !ok {
		return
	}
	lang := i18n.Of(p)
	again := func(p *player.Player, _ *world.Tx) { showBook(s, p, pos, query) }
	back := func(p *player.Player, _ *world.Tx) {
		p.H().Do(func(tx *world.Tx, e world.Entity) { showGrid(s, e.(*player.Player), tx, pos, query) })
	}
	open := openTab(s, p)
	var opts []ui.Option
	for i, t := range bookTabs {
		icon := "textures/ui/magnifyingGlass"
		if t.icon != nil {
			icon = ui.ItemIcon(t.icon)
		}
		opts = append(opts, ui.Tab(i18n.T(lang, t.key), icon, i == open, func(p *player.Player, tx *world.Tx) {
			bookTab.Of(s)[p.H()] = i
			again(p, tx)
		}))
	}
	opts = append(opts, ui.Toggle(i18n.T(lang, "netstore.terminal.bookCan"), bookCanOnly.Of(s)[p.H()], func(p *player.Player, tx *world.Tx) {
		bookCanOnly.Of(s)[p.H()] = !bookCanOnly.Of(s)[p.H()]
		again(p, tx)
	}))
	if open == searchTab {
		label := i18n.T(lang, "storage.search.label")
		if q := bookQuery.Of(s)[p.H()]; q != "" {
			label += "§r  §e" + q
		}
		opts = append(opts, ui.Option{Label: label, Icon: "textures/ui/magnifyingGlass", Run: func(p *player.Player, _ *world.Tx) {
			ui.Input(p, i18n.T(lang, "netstore.terminal.book"), i18n.T(lang, "storage.search.label"), bookQuery.Of(s)[p.H()],
				func(p *player.Player, tx *world.Tx, q string) {
					bookQuery.Of(s)[p.H()] = q
					again(p, tx)
				}, again)
		}})
	}
	opts = append(opts, ui.Back(i18n.T(lang, "netstore.terminal.grid"), back))
	var slots []ui.ItemSlot
	for _, e := range bookOf(s, p, At(tx, pos), g) {
		cr := e.cr
		tip := decor.TipName(lang, cr.Out) + "\n" + strings.Join(cr.Tip(lang), "\n")
		if !e.can {
			tip += "\n" + i18n.T(lang, "netstore.terminal.bookMissing")
		}
		slots = append(slots, ui.ItemSlot{Icon: formicon.Of(cr.Out), Tip: tip, Run: func(p *player.Player, tx *world.Tx) {
			if g, ok := gridOf(tx, pos); ok {
				layRecipe(At(tx, pos), g, cr, p)
			}
			back(p, tx)
		}})
	}
	body := i18n.T(lang, "netstore.terminal.bookTip")
	if len(slots) == 0 {
		body = i18n.T(lang, "netstore.terminal.bookNone")
	}
	ui.ItemGrid(p, i18n.T(lang, "netstore.terminal.book"), body, slots, opts...)
}
