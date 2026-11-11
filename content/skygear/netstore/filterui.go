package netstore

import (
	"fmt"
	"slices"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/invclick"
	"github.com/presentkim/skygear/common/storagenet"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/decor"
	"github.com/presentkim/skygear/content/skygear/machine"
)

// The window of an importer, exporter or external storage, laid out like
// Refined Storage's: a row of filter slots, the mode (and priority) buttons
// beside the window and the player's inventory below. A filter slot pressed
// with an item held takes that kind (nothing is used up), pressed with an
// empty hand it empties; a shift click in the inventory adds the kind.

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

func showFilter(s *core.Session, p *player.Player, pos cube.Pos, kind string) {
	lang := i18n.Of(p)
	cfg := Config(s, pos)
	again := func(p *player.Player, tx *world.Tx) {
		s.MarkDirty()
		Invalidate(tx, pos)
		showFilter(s, p, pos, kind)
	}
	tipOf := func(st item.Stack) string {
		if st.Empty() {
			return ""
		}
		return decor.TipName(lang, st)
	}
	n := max(filterSlots, (len(cfg.Filter)+filterSlots-1)/filterSlots*filterSlots)
	filter := make([]ui.InvSlot, n)
	for i, k := range cfg.Filter {
		filter[i] = ui.StackSlot(Proto(s, storagenet.Key(k)), tipOf(Proto(s, storagenet.Key(k))))
	}
	inv := p.Inventory()
	main, hot := invclick.Main(inv), invclick.Hotbar(inv)

	mode, icon := "netstore.io.deny", "textures/ui/cancel"
	if cfg.Allow {
		mode, icon = "netstore.io.allow", "textures/ui/confirm"
	}
	var side []ui.Option
	if kind != machine.KindNetExporter {
		side = append(side, ui.Option{Label: i18n.T(lang, mode), Icon: icon, Run: func(p *player.Player, tx *world.Tx) {
			cfg.Allow = !cfg.Allow
			again(p, tx)
		}})
	}
	info := i18n.T(lang, "netstore.io.body."+kind)
	if kind == machine.KindNetExternal {
		side = append(side,
			ui.Option{Label: i18n.T(lang, "netstore.io.prioUp"), Icon: "textures/ui/sg/arrow_up", Run: func(p *player.Player, tx *world.Tx) {
				cfg.Priority += priorityStep
				again(p, tx)
			}},
			ui.Option{Label: i18n.T(lang, "netstore.io.prioDown"), Icon: "textures/ui/sg/arrow_down", Run: func(p *player.Player, tx *world.Tx) {
				cfg.Priority -= priorityStep
				again(p, tx)
			}})
		info += "\n" + fmt.Sprintf(i18n.T(lang, "netstore.io.priority"), cfg.Priority)
	}

	add := func(st item.Stack) {
		if k := string(KeyOf(st)); !st.Empty() && !slices.Contains(cfg.Filter, k) {
			cfg.Filter = append(cfg.Filter, k)
		}
	}
	rows := []ui.InvRow{{Heading: i18n.T(lang, "netstore.io.title"), Parts: []ui.InvPart{{Area: "filter", Cols: filterSlots, Slots: filter}}}}
	st := stateAt(p.Tx(), pos)
	var ups invclick.Slots
	if st != nil && len(st.Def.Input) > 0 {
		ups = make(invclick.Slots, len(st.Def.Input))
		for i, s := range st.Def.Input {
			ups[i], _ = st.Inv().Item(s)
		}
		rows = append(rows, ui.InvRow{Heading: i18n.T(lang, "netstore.io.upgrades"), Parts: []ui.InvPart{{Area: "upgrade", Cols: len(ups), Slots: areaSlots(ups, tipOf)}}})
		info += "\n" + fmt.Sprintf(i18n.T(lang, "netstore.io.speed"), ioRateOf(st), ioAmountOf(st))
	}
	rows = append(rows,
		ui.InvRow{Heading: i18n.T(lang, "netstore.terminal.inventory"), Parts: []ui.InvPart{{Area: "inv", Cols: 9, Slots: areaSlots(main, tipOf)}}},
		ui.InvRow{Parts: []ui.InvPart{{Area: "hotbar", Cols: 9, Slots: areaSlots(hot, tipOf)}}},
	)
	// putUpgrade moves one upgrade of it into a free upgrade slot and reports
	// whether it went in.
	putUpgrade := func(tx *world.Tx, slot int, it item.Stack) bool {
		st := stateAt(tx, pos)
		if st == nil || slot < 0 || slot >= len(st.Def.Input) || !upgradeFits(st, it) {
			return false
		}
		if cur, _ := st.Inv().Item(st.Def.Input[slot]); !cur.Empty() {
			return false
		}
		_ = st.Inv().SetItem(st.Def.Input[slot], it.Grow(1-it.Count()))
		return true
	}
	freeUpgradeSlot := func(tx *world.Tx) int {
		if st := stateAt(tx, pos); st != nil {
			for i, s := range st.Def.Input {
				if cur, _ := st.Inv().Item(s); cur.Empty() {
					return i
				}
			}
		}
		return -1
	}
	ui.ShowInventory(p, ui.InventoryScreen{
		Title: machine.Kinds[kind].Title(lang),
		Info:  info,
		Rows:  rows,
		Side:  side,
		Slot: func(p *player.Player, tx *world.Tx, a ui.SlotAct) {
			cur := a.Carried
			switch a.Area {
			case "upgrade":
				st := stateAt(tx, pos)
				if st == nil || a.Index >= len(st.Def.Input) {
					break
				}
				in, _ := st.Inv().Item(st.Def.Input[a.Index])
				switch {
				case in.Empty() && !cur.Empty():
					if putUpgrade(tx, a.Index, cur) {
						ui.SetCarried(p, cur.Grow(-1))
					}
				case in.Empty():
				case a.Shift:
					if left := invclick.Move(in, hot, main); left.Empty() {
						_ = st.Inv().SetItem(st.Def.Input[a.Index], item.Stack{})
					}
				case cur.Empty():
					_ = st.Inv().SetItem(st.Def.Input[a.Index], item.Stack{})
					ui.SetCarried(p, in)
				}
				Invalidate(tx, pos)
			case "filter":
				switch {
				case !cur.Empty() && a.Index < len(cfg.Filter):
					if k := string(KeyOf(cur)); !slices.Contains(cfg.Filter, k) {
						cfg.Filter[a.Index] = k
					}
				case !cur.Empty():
					add(cur)
				case a.Index < len(cfg.Filter):
					cfg.Filter = slices.Delete(cfg.Filter, a.Index, a.Index+1)
				}
			case "inv", "hotbar":
				area := invclick.Area(main)
				if a.Area == "hotbar" {
					area = hot
				}
				if a.Index >= area.Len() {
					break
				}
				if it := area.Item(a.Index); a.Shift {
					if _, up := it.Item().(Upgrade); up {
						if putUpgrade(tx, freeUpgradeSlot(tx), it) {
							area.Set(a.Index, it.Grow(-1))
						}
						Invalidate(tx, pos)
						break
					}
					add(it)
					break
				}
				ui.SetCarried(p, invclick.Click(area, a.Index, cur, a.Right))
			}
			again(p, tx)
		},
		Again: again,
		Size:  "filter",
	})
}
