package skygear

import (
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/internal/games/skygear/formicon"
	"github.com/presentkim/skygear/internal/games/skygear/machinerecipe"
	"github.com/presentkim/skygear/internal/games/skygear/naming"
)

// The laboratory's screen is a split screen like the research screen: the
// essences research has reached on the left, the chosen one on the right.
// The essence being made shows its progress, ingredients and products with
// fill/take/stop buttons; another one shows its recipe and a start button.

// showLab shows the essence extractor's screen on the essence it makes, or
// the first one.
func (s *session) showLab(p *player.Player, tx *world.Tx, pos cube.Pos, st *machineState) {
	s.showLabOn(p, pos, st, st.lab)
}

// showLabOn shows the essence extractor's screen with the recipe r chosen
// (nil: the first open one).
func (s *session) showLabOn(p *player.Player, pos cube.Pos, st *machineState, r *machinerecipe.Lab) {
	lang := i18n.Of(p)
	open := s.labOpenRecipes()
	if r == nil && len(open) > 0 {
		r = open[0]
	}
	// again runs f on the extractor, if it still stands, and shows the screen
	// again on the recipe on.
	again := func(on *machinerecipe.Lab, f func(p *player.Player, tx *world.Tx)) func(p *player.Player, tx *world.Tx) {
		return func(p *player.Player, tx *world.Tx) {
			if !s.labStill(tx, pos, st) {
				return
			}
			f(p, tx)
			s.showLabOn(p, pos, st, on)
		}
	}
	var sp ui.Split
	for _, o := range open {
		out := item.NewStack(o.Out, 1)
		tip := tipName(lang, out) + "\n" + fmt.Sprintf(i18n.T(lang, "lab.makes"), o.OutCount, o.Ticks/20)
		sl := ui.ItemSlot{Icon: formicon.Of(out), Tip: tip, Selected: o == r, Run: again(o, func(*player.Player, *world.Tx) {})}
		if o == st.lab {
			sl.Count = "§a▶"
		}
		if o == r {
			sl.Run = nil
		}
		sp.Slots = append(sp.Slots, sl)
	}
	sp.Pages = 1
	if r != nil {
		sp.Detail, sp.Rows, sp.Actions = s.labDetail(lang, st, r, again)
	}
	ui.SplitMenu(p, i18n.T(lang, "machine.lab.name"), sp)
}

// labOpenRecipes are the recipes whose essence the island's research has
// come to, in table order.
func (s *session) labOpenRecipes() []*machinerecipe.Lab {
	var out []*machinerecipe.Lab
	for i := range machinerecipe.Labs {
		if r := &machinerecipe.Labs[i]; s.labOpen(r) {
			out = append(out, r)
		}
	}
	return out
}

// labStill reports whether the extractor a screen was shown for still
// stands where it did, in the session's world.
func (s *session) labStill(tx *world.Tx, pos cube.Pos, st *machineState) bool {
	m, ok := tx.Block(pos).(Machine)
	return tx.World() == s.world && ok && m.st == st
}

// labOpen reports whether the island's research has come to the essence a
// recipe makes: a research that costs it is done or can be done (or every
// recipe is open).
func (s *session) labOpen(r *machinerecipe.Lab) bool {
	if s.recipesFree {
		return true
	}
	for i := range researchTree {
		if rr := &researchTree[i]; essenceCostOf(rr, r.Out.Kind) > 0 && (s.researched(rr.ID) || s.researchOpen(rr)) {
			return true
		}
	}
	return false
}

// labDetail is the right side of the screen for recipe r: the card, the
// rows of the ingredients and the essences made, and the buttons.
func (s *session) labDetail(lang string, st *machineState, r *machinerecipe.Lab, again func(*machinerecipe.Lab, func(*player.Player, *world.Tx)) func(*player.Player, *world.Tx)) (ui.Detail, []ui.SlotRow, []ui.Option) {
	out := item.NewStack(r.Out, 1)
	d := ui.Detail{Icon: formicon.Of(out), Name: naming.UIItemName(lang, out)}
	running := st.lab == r
	have := st.labHave()
	var ins []ui.ItemSlot
	for _, in := range r.In {
		it := item.NewStack(in.Item, 1)
		sl := ui.ItemSlot{Icon: formicon.Of(it), Count: fmt.Sprint(in.Count), Tip: tipName(lang, it) + "\n" + fmt.Sprintf(i18n.T(lang, "lab.need"), in.Count)}
		if running {
			n := have[naming.ItemKey(in.Item)]
			sl.Count = ui.CountText(n)
			if n < in.Count {
				sl.Count = "§c" + sl.Count
			}
			sl.Tip += "\n" + fmt.Sprintf(i18n.T(lang, "lab.held"), n)
			sl.Run = again(r, func(p *player.Player, _ *world.Tx) { s.labDeposit(p, st, in.Item) })
		}
		ins = append(ins, sl)
	}
	made := ui.ItemSlot{Icon: formicon.Of(out), Count: fmt.Sprint(r.OutCount), Tip: tipName(lang, out) + "\n" + fmt.Sprintf(i18n.T(lang, "lab.makes"), r.OutCount, r.Ticks/20)}
	rows := []ui.SlotRow{{Heading: i18n.T(lang, "lab.ingredients"), Slots: ins}}
	if !running {
		d.Status = fmt.Sprintf(i18n.T(lang, "lab.makes"), r.OutCount, r.Ticks/20)
		rows = append(rows, ui.SlotRow{Heading: i18n.T(lang, "lab.output"), Slots: []ui.ItemSlot{made}})
		start := func(p *player.Player, tx *world.Tx) {
			if st.lab != nil {
				s.labStop(p, tx, st)
			}
			st.setLab(r)
			s.labDeposit(p, st, nil)
		}
		key := "lab.start"
		if st.lab != nil {
			key = "lab.switch"
		}
		return d, rows, []ui.Option{{Label: i18n.T(lang, key), Icon: formicon.Of(out), Run: again(r, start)}}
	}
	n := outputCount(st, r.Out)
	made.Count = ui.CountText(n)
	if n > 0 {
		made.Run = again(r, func(p *player.Player, tx *world.Tx) { s.labTake(p, tx, st) })
	}
	rows = append(rows, ui.SlotRow{Heading: i18n.T(lang, "lab.output"), Slots: []ui.ItemSlot{made}})
	d.Status = fmt.Sprintf(i18n.T(lang, "lab.progress"), min(100, st.progress*100/r.Ticks))
	d.Body = strings.Join(labLines(lang, st)[1:], "\n")
	actions := []ui.Option{
		{Label: i18n.T(lang, "lab.deposit"), Icon: formicon.Of(item.NewStack(r.In[0].Item, 1)), Run: again(r, func(p *player.Player, _ *world.Tx) { s.labDeposit(p, st, nil) })},
		{Label: i18n.T(lang, "lab.take"), Icon: formicon.Of(out), Run: again(r, func(p *player.Player, tx *world.Tx) { s.labTake(p, tx, st) })},
		{Label: i18n.T(lang, "lab.stop"), Icon: formicon.Locked, Run: again(r, func(p *player.Player, tx *world.Tx) { s.labStop(p, tx, st) })},
	}
	return d, rows, actions
}

// labDeposit puts the ingredients of the recipe (only, when it is set, the
// one that is that item) from p's inventory in the laboratory, up to
// labHandCap of each.
func (s *session) labDeposit(p *player.Player, st *machineState, only world.Item) {
	inv := p.Inventory()
	for slot, it := range inv.Slots() {
		if it.Empty() || !st.accepts(it) || (only != nil && naming.ItemKey(it.Item()) != naming.ItemKey(only)) {
			continue
		}
		n := min(it.Count(), st.labHandCap(it)-st.labHave()[naming.ItemKey(it.Item())], st.inputRoom(it))
		if n <= 0 {
			continue
		}
		put(st.inv(), st.kind.input, it.Grow(n-it.Count()))
		_ = inv.SetItem(slot, it.Grow(-n))
	}
}

// inputRoom is how many of it the input slots still take.
func (st *machineState) inputRoom(it item.Stack) int {
	room := 0
	for _, slot := range st.kind.input {
		cur, _ := st.inv().Item(slot)
		switch {
		case cur.Empty():
			room += it.MaxCount()
		case cur.Comparable(it):
			room += cur.MaxCount() - cur.Count()
		}
	}
	return room
}

// labTake gives p the essences made.
func (s *session) labTake(p *player.Player, tx *world.Tx, st *machineState) {
	s.labGive(p, tx, st, st.kind.output)
}

// labStop stops the laboratory: p gets back what it holds, and it takes
// nothing until an essence is chosen again.
func (s *session) labStop(p *player.Player, tx *world.Tx, st *machineState) {
	s.labGive(p, tx, st, st.kind.input)
	s.labGive(p, tx, st, st.kind.output)
	st.setLab(nil)
}

// labGive empties slots of the laboratory into p's inventory (what does not
// fit drops at p).
func (s *session) labGive(p *player.Player, tx *world.Tx, st *machineState, slots []int) {
	for _, slot := range slots {
		if it, _ := st.inv().Item(slot); !it.Empty() {
			_ = st.inv().SetItem(slot, item.Stack{})
			s.giveItem(p, tx, p.Position(), it)
		}
	}
}
