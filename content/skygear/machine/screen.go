package machine

import (
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/invclick"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/decor"
	"github.com/presentkim/skygear/content/skygear/naming"
)

// The machine window as an Ore UI inventory screen, built from the machine's
// definition alone: its energy and fluid gauges on the left, its input and
// charge slots, its progress and its output slots, then the player's
// inventory with the transfer bars. Every machine that opens a window draws
// it here, so a change to the layout reaches them all. The screen redraws
// itself while it is open (watchMachine).

// watchers are the machine each player has open, by player.
var watchers = core.NewPlayers[cube.Pos]()

func machineKey(pos cube.Pos) string { return fmt.Sprintf("machine@%d,%d,%d", pos[0], pos[1], pos[2]) }

// cols is how many columns a block of n machine slots takes: a 3×3 store,
// a 2×2 output, a row otherwise.
func cols(n int) int {
	switch n {
	case 9:
		return 3
	case 4:
		return 2
	}
	return max(1, n)
}

// In GUI pixels: a slot, a gauge with its border, and the window around the
// machine row (title, inventory heading, inventory, hotbar, padding).
const slotH, gaugeH, windowRest = 18, 56, 112

// windowHeight is the height of a machine window whose machine row holds
// parts, so the window ends under the hotbar.
func windowHeight(parts []ui.InvPart) int {
	row := slotH
	for _, p := range parts {
		switch {
		case p.Gauge != nil:
			row = max(row, gaugeH)
		case len(p.Slots) > 0:
			row = max(row, slotH*((len(p.Slots)+p.Cols-1)/p.Cols))
		}
	}
	return windowRest + row
}

// showsEnergy reports whether the machine's window has an energy gauge.
func (st *State) showsEnergy() bool {
	return st.Def.EnergyCap > 0 && (st.Def.Use > 0 || st.Def.Generates > 0)
}

// showsProgress reports whether the machine's window shows its progress.
func (st *State) showsProgress() bool { return st.Def.Marker != "" }

// screenLook is what the open window shows of the machine, to tell whether
// it needs drawing again.
type screenLook struct {
	pct, energy, fluid int
	ver                uint64
	stuck              stuckKind
	target             string
}

func (st *State) screenLook() screenLook {
	l := screenLook{ver: st.invVer, stuck: st.Stuck, target: st.target}
	if st.showsProgress() {
		l.pct = st.progressPct() / 10
	}
	if st.showsEnergy() {
		l.energy = st.Energy * 20 / max(1, st.EnergyCap())
	}
	if st.Def.FluidCap > 0 {
		l.fluid = st.FluidAmt * 20 / st.Def.FluidCap
	}
	return l
}

// minerTarget is the block the auto miner st at pos mines next, as an item,
// and false when the block it faces cannot be mined.
func (st *State) minerTarget(tx *world.Tx, pos cube.Pos) (item.Stack, bool) {
	b := tx.Block(pos.Side(st.Facing))
	it, isItem := b.(world.Item)
	if _, ok := Mineable(b); !ok || !isItem {
		return item.Stack{}, false
	}
	return item.NewStack(it, 1), true
}

// noteTarget keeps what the auto miner at pos would mine in st.target, so
// the open window redraws when it changes.
func (st *State) noteTarget(tx *world.Tx, pos cube.Pos) {
	if st.Def.ID != idMiner {
		return
	}
	st.target = ""
	if it, ok := st.minerTarget(tx, pos); ok {
		st.target = naming.ItemKey(it.Item())
	}
}

// targetPart is the auto miner's slot showing the block it mines, a barrier
// when there is none.
func (st *State) targetPart(lang string, tx *world.Tx, pos cube.Pos) ui.InvPart {
	it, ok := st.minerTarget(tx, pos)
	tip := i18n.T(lang, "machine.miner.noTarget")
	if ok {
		tip = fmt.Sprintf(i18n.T(lang, "machine.miner.target"), decor.TipName(lang, it))
	} else {
		it = item.NewStack(block.Barrier{}, 1)
	}
	return ui.InvPart{Area: "target", Cols: 1, Slots: []ui.InvSlot{ui.StackSlot(it, tip)}}
}

// ShowMachine shows p the window of the machine at pos.
func ShowMachine(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, st *State) {
	lang := i18n.Of(p)
	st.pos = pos
	watchers.Of(s)[p.H()] = pos
	st.noteTarget(tx, pos)
	st.shownLook = st.screenLook()
	tipOf := func(it item.Stack) string {
		if it.Empty() {
			return ""
		}
		return decor.TipName(lang, it)
	}
	slotsOf := func(a invclick.Area) []ui.InvSlot {
		out := make([]ui.InvSlot, a.Len())
		for i := range out {
			out[i] = ui.StackSlot(a.Item(i), tipOf(a.Item(i)))
		}
		return out
	}
	in, charge, out := stateSlots{st, st.Def.Input, true}, stateSlots{st, st.Def.Charge, true}, stateSlots{st, st.Def.Output, false}
	main, hot := invclick.Main(p.Inventory()), invclick.Hotbar(p.Inventory())
	areas := map[string]invclick.Area{"in": in, "charge": charge, "out": out, "inv": main, "hotbar": hot}

	var parts []ui.InvPart
	if st.showsEnergy() {
		have := st.Energy
		if st.Unlimited {
			have = st.EnergyCap()
		}
		parts = append(parts, ui.InvPart{Gauge: &ui.InvGauge{Kind: "energy", Value: have, Max: st.EnergyCap(), Tip: fmt.Sprintf(i18n.T(lang, "machine.energy.gauge"), have, st.EnergyCap())}})
	}
	if st.Def.FluidCap > 0 {
		parts = append(parts, ui.InvPart{Gauge: &ui.InvGauge{Kind: "fluid", Value: st.FluidAmt, Max: st.Def.FluidCap, Tip: fmt.Sprintf(i18n.T(lang, "machine.fluid.gauge"), st.FluidAmt, st.Def.FluidCap)}})
	}
	for _, a := range []struct {
		name string
		area stateSlots
	}{{"in", in}, {"charge", charge}} {
		if a.area.Len() > 0 {
			parts = append(parts, ui.InvPart{Area: a.name, Cols: cols(a.area.Len()), Slots: slotsOf(a.area)})
		}
	}
	if st.Def.ID == idMiner {
		parts = append(parts, st.targetPart(lang, tx, pos))
	}
	if st.showsProgress() {
		ring, key, withPct := st.progressLook()
		tip := i18n.T(lang, key)
		if withPct {
			tip = fmt.Sprintf(tip, st.progressPct())
		}
		parts = append(parts, ui.InvPart{Progress: &ui.InvProgress{Pct: st.progressPct(), Ring: ring, Tip: tip}})
	}
	if out.Len() > 0 {
		parts = append(parts, ui.InvPart{Area: "out", Cols: cols(out.Len()), Slots: slotsOf(out)})
	}

	again := func(p *player.Player, tx *world.Tx) {
		if m, ok := tx.Block(pos).(Machine); ok && m.State == st {
			ShowMachine(s, p, tx, pos, st)
		}
	}
	take := ui.TransferBar{All: ui.Option{Label: i18n.T(lang, "machine.screen.takeAll"), Icon: "textures/ui/sg/arrow_down", Run: func(p *player.Player, tx *world.Tx) {
		for i := range out.Len() {
			out.Set(i, invclick.Move(out.Item(i), hot, main))
		}
		again(p, tx)
	}}}
	fill := func(only bool) func(p *player.Player, tx *world.Tx) {
		return func(p *player.Player, tx *world.Tx) {
			for _, a := range []invclick.Area{main, hot} {
				for i := range a.Len() {
					it := a.Item(i)
					if !it.Empty() && (!only || st.holds(it)) {
						a.Set(i, it.Grow(-st.Fill(it)))
					}
				}
			}
			again(p, tx)
		}
	}
	put := ui.TransferBar{
		All:      ui.Option{Label: i18n.T(lang, "machine.screen.putAll"), Icon: "textures/ui/sg/arrow_up", Run: fill(false)},
		Matching: ui.Option{Label: i18n.T(lang, "machine.screen.putMatching"), Icon: "textures/ui/store_filter_icon", Run: fill(true)},
	}
	machineRow := ui.InvRow{Parts: parts, Center: true}
	if out.Len() > 0 {
		machineRow.Bar = take.Options()
	}
	invRow := ui.InvRow{Heading: i18n.T(lang, "machine.screen.inventory"), Parts: []ui.InvPart{{Area: "inv", Cols: 9, Slots: slotsOf(main)}}, BarTop: true}
	if in.Len()+charge.Len() > 0 {
		invRow.Bar = put.Options()
	}

	help := []string{}
	for _, k := range st.Def.helpKeys {
		help = append(help, i18n.T(lang, k))
	}
	info := append(BlockedLines(lang, st), ConsumerLines(lang, st)...)
	height := windowHeight(parts)
	ui.ShowInventory(p, ui.InventoryScreen{
		Title:   tierTitle(lang, st.Def.Title(lang), st.Tier),
		Info:    strings.Join(info, "\n"),
		Rows:    []ui.InvRow{machineRow, invRow, {Parts: []ui.InvPart{{Area: "hotbar", Cols: 9, Slots: slotsOf(hot)}}}},
		Options: []ui.Option{ui.Help(strings.Join(help, "\n"), nil)},
		Tools:   st.rangeTools(lang, again),
		Slot: func(p *player.Player, tx *world.Tx, a ui.SlotAct) {
			if m, ok := tx.Block(pos).(Machine); tx.World() != s.World || !ok || m.State != st {
				return
			}
			switch area, ok := areas[a.Area]; {
			case !ok || a.Index >= area.Len():
			case a.Shift && (a.Area == "in" || a.Area == "charge" || a.Area == "out"):
				area.Set(a.Index, invclick.Move(area.Item(a.Index), hot, main))
			case a.Shift:
				it := area.Item(a.Index)
				area.Set(a.Index, it.Grow(-st.Fill(it)))
			default:
				ui.SetCarried(p, invclick.Click(area, a.Index, a.Carried, a.Right))
			}
			again(p, tx)
		},
		Again:  again,
		Close:  func(p *player.Player, _ *world.Tx) { delete(watchers.Of(s), p.H()) },
		Size:   fmt.Sprintf("machine:%d", height),
		Height: height,
		Key:    machineKey(pos),
	})
}

// ShowPreview shows p the screen of a machine of kind with state st that is
// not in the world, as if it stood at pos (/sgdebug ui): the screen draws,
// its slots do nothing.
func ShowPreview(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, kind string, st *State) {
	switch {
	case kind == KindLab:
		showLab(s, p, tx, pos, st)
	case st.Def.Crafter:
		showCrafter(s, p, pos, st)
	default:
		ShowMachine(s, p, tx, pos, st)
	}
}

// holds reports whether one of the machine's input or charge slots holds
// the kind of it.
func (st *State) holds(it item.Stack) bool {
	for _, s := range append(append([]int{}, st.Def.Input...), st.Def.Charge...) {
		if cur, _ := st.Inv().Item(s); !cur.Empty() && cur.Comparable(it) {
			return true
		}
	}
	return false
}

// watchMachine draws the windows open on the machine at pos again when what
// they show has changed; it runs from the machine's tick every few ticks.
func watchMachine(s *core.Session, tx *world.Tx, pos cube.Pos, st *State) {
	st.noteTarget(tx, pos)
	if st.screenLook() == st.shownLook {
		return
	}
	key := machineKey(pos)
	for h, at := range watchers.Of(s) {
		if at != pos {
			continue
		}
		e, ok := h.Entity(tx)
		p, isPlayer := e.(*player.Player)
		if !ok || !isPlayer || ui.ShowingKey(p) != key {
			delete(watchers.Of(s), h)
			continue
		}
		ShowMachine(s, p, tx, pos, st)
	}
	st.shownLook = st.screenLook()
}
