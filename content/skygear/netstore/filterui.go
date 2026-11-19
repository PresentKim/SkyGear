package netstore

import (
	"slices"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/chestui"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/storagenet"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/wintext"
)

// The window of an importer, exporter or external storage, laid out like
// Refined Storage's: a chest of two rows, the filter slots on top and under
// them the upgrade slots, the info and the mode (and priority) buttons. An
// item put on a filter slot (a shift click in the inventory too) adds its
// kind there and goes back; taking a filter slot empties it.

// clickFilter is the click on an importer, exporter or external storage: it
// opens the filter window.
func clickFilter(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool {
	if !s.Debounced(p) {
		showFilter(s, p, pos, nodeAt(tx, pos))
	}
	return true
}

const (
	// priorityStep is how much one press changes the priority of an
	// external storage.
	priorityStep = 1
	// filterSlots is the filter row, as in Refined Storage.
	filterSlots = 9
)

// The slots of the second row: the upgrades from its start, then the info
// and the buttons at its end.
const (
	upgradeSlot0 = filterSlots
	infoSlot     = 14
	modeSlot     = 15
	prioUpSlot   = 16
	prioDownSlot = 17
	filterSize   = 18
)

// filterWindow is a filter window open for one player.
type filterWindow struct {
	s       *core.Session
	p       *player.Player
	pos     cube.Pos
	kind    string
	storage *chestui.Storage
	// ups is how many upgrade slots the node has.
	ups int
	// filling is set while the window's slots are written from the node.
	filling bool
}

func showFilter(s *core.Session, p *player.Player, pos cube.Pos, kind string) {
	w := &filterWindow{s: s, p: p, pos: pos, kind: kind}
	if st := stateAt(p.Tx(), pos); st != nil {
		w.ups = len(st.Def.Input)
	}
	w.storage = chestui.NewStorage(filterSize, machine.Kinds[kind].Title(i18n.Of(p)))
	w.fill(p.Tx())
	w.storage.Inventory().Handle(filterHandler{w})
	w.storage.OnChange(w.changed)
	w.storage.Open(p, nil)
}

// fill writes the window's slots from the node's filter, upgrades and modes.
func (w *filterWindow) fill(tx *world.Tx) {
	w.filling = true
	defer func() { w.filling = false }()
	cfg := Config(w.s, w.pos)
	inv := w.storage.Inventory()
	for i := range filterSlots {
		it := item.Stack{}
		if i < len(cfg.Filter) {
			it = Proto(w.s, storagenet.Key(cfg.Filter[i]))
		}
		_ = inv.SetItem(i, it)
	}
	info := []wintext.Text{{wintext.Raw("§7"), wintext.Key("netstore.io.body." + w.kind)}}
	if st := stateAt(tx, w.pos); st != nil {
		for i, slot := range st.Def.Input {
			it, _ := st.Inv().Item(slot)
			_ = inv.SetItem(upgradeSlot0+i, it)
		}
		if len(st.Def.Input) > 0 {
			info = append(info, wintext.Text{wintext.Raw("§7"), wintext.Key("netstore.io.speed", ioRateOf(st), ioAmountOf(st))})
		}
	}
	if w.kind == machine.KindNetExternal {
		info = append(info, wintext.Text{wintext.Raw("§7"), wintext.Key("netstore.io.priority", cfg.Priority)})
		_ = inv.SetItem(prioUpSlot, button(item.Paper{}, "netstore.io.prioUp"))
		_ = inv.SetItem(prioDownSlot, button(item.Paper{}, "netstore.io.prioDown"))
	}
	_ = inv.SetItem(infoSlot, wintext.With(item.NewStack(item.Book{}, 1), wintext.Text{wintext.Raw("§r"), wintext.Key("netstore.io.title")}, info...))
	if w.kind != machine.KindNetExporter {
		mode := button(item.Dye{Colour: item.ColourRed()}, "netstore.io.deny")
		if cfg.Allow {
			mode = button(item.Dye{Colour: item.ColourLime()}, "netstore.io.allow")
		}
		_ = inv.SetItem(modeSlot, mode)
	}
}

// button is a button of the window: it named with the text of key.
func button(it world.Item, key string) item.Stack {
	return wintext.With(item.NewStack(it, 1), wintext.Text{wintext.Raw("§r"), wintext.Key(key)})
}

// press runs what pressing slot does, in the player's world after the move
// is answered, and writes the window again.
func (w *filterWindow) press(slot int, held item.Stack) {
	w.p.H().Do(func(tx *world.Tx, _ world.Entity) {
		cfg := Config(w.s, w.pos)
		switch {
		case slot < filterSlots && !held.Empty():
			k := string(KeyOf(held))
			switch {
			case slices.Contains(cfg.Filter, k):
			case slot < len(cfg.Filter):
				cfg.Filter[slot] = k
			case len(cfg.Filter) < filterSlots:
				cfg.Filter = append(cfg.Filter, k)
			}
		case slot < filterSlots && slot < len(cfg.Filter):
			cfg.Filter = slices.Delete(cfg.Filter, slot, slot+1)
		case slot == modeSlot && w.kind != machine.KindNetExporter:
			cfg.Allow = !cfg.Allow
		case slot == prioUpSlot && w.kind == machine.KindNetExternal:
			cfg.Priority += priorityStep
		case slot == prioDownSlot && w.kind == machine.KindNetExternal:
			cfg.Priority -= priorityStep
		}
		w.s.MarkDirty()
		Invalidate(tx, w.pos)
		w.fill(tx)
	})
}

// changed keeps the node's upgrade slots to the window's: one upgrade a
// slot, the rest back to the player.
func (w *filterWindow) changed(slot int, _, after item.Stack) {
	if w.filling || slot < upgradeSlot0 || slot >= infoSlot {
		return
	}
	w.p.H().Do(func(tx *world.Tx, e world.Entity) {
		st := stateAt(tx, w.pos)
		if st == nil || slot-upgradeSlot0 >= len(st.Def.Input) {
			return
		}
		if after.Count() > 1 {
			if _, err := e.(*player.Player).Inventory().AddItem(after.Grow(-1)); err != nil {
				e.(*player.Player).Drop(after.Grow(-1))
			}
			after = after.Grow(1 - after.Count())
		}
		_ = st.Inv().SetItem(st.Def.Input[slot-upgradeSlot0], after)
		Invalidate(tx, w.pos)
		w.fill(tx)
	})
}

// filterHandler keeps the moves in the filter window to its upgrade slots;
// the filter slots and buttons take a press instead.
type filterHandler struct{ w *filterWindow }

func (h filterHandler) upgrade(slot int) bool {
	return slot >= upgradeSlot0 && slot < upgradeSlot0+h.w.ups
}

func (h filterHandler) HandleTake(ctx *inventory.Context, slot int, _ item.Stack) {
	if h.upgrade(slot) {
		return
	}
	ctx.Cancel()
	h.w.press(slot, item.Stack{})
}

func (h filterHandler) HandlePlace(ctx *inventory.Context, slot int, it item.Stack) {
	if h.upgrade(slot) {
		if p, ok := ctx.Val().(*player.Player); ok {
			if st := stateAt(p.Tx(), h.w.pos); st != nil {
				if cur, _ := h.w.storage.Inventory().Item(slot); cur.Empty() && upgradeFits(st, it) {
					return
				}
			}
		}
		ctx.Cancel()
		return
	}
	ctx.Cancel()
	if slot < filterSlots {
		h.w.press(slot, it)
	}
}

func (h filterHandler) HandleDrop(ctx *inventory.Context, slot int, _ item.Stack) {
	if !h.upgrade(slot) {
		ctx.Cancel()
	}
}
