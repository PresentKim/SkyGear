package machine

import (
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/decor"
	"github.com/presentkim/skygear/content/skygear/formicon"
	"github.com/presentkim/skygear/content/skygear/machinerecipe"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/progress"
)

// The laboratory's screen is a split screen like the research screen: the
// essences research has reached on the left, the chosen one on the right.
// The essence being made shows its progress, ingredients and products with
// fill/take/stop buttons; another one shows its recipe and a start button.

// showLab shows the essence extractor's screen on the essence it makes, or
// the first one.
func showLab(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, st *State) {
	showLabOn(s, p, pos, st, st.Lab)
}

// showLabOn shows the essence extractor's screen with the recipe r chosen
// (nil: the first open one).
func showLabOn(s *core.Session, p *player.Player, pos cube.Pos, st *State, r *machinerecipe.Lab) {
	lang := i18n.Of(p)
	open := LabOpenRecipes(s)
	if r == nil && len(open) > 0 {
		r = open[0]
	}
	// again runs f on the extractor, if it still stands, and shows the screen
	// again on the recipe on.
	again := func(on *machinerecipe.Lab, f func(p *player.Player, tx *world.Tx)) func(p *player.Player, tx *world.Tx) {
		return func(p *player.Player, tx *world.Tx) {
			if !labStill(s, tx, pos, st) {
				return
			}
			f(p, tx)
			showLabOn(s, p, pos, st, on)
		}
	}
	var sp ui.Split
	for _, o := range open {
		out := item.NewStack(o.Out, 1)
		tip := decor.TipName(lang, out) + "\n" + fmt.Sprintf(i18n.T(lang, "lab.makes"), o.OutCount, o.Ticks/20)
		sl := ui.ItemSlot{Icon: formicon.Of(out), Tip: tip, Selected: o == r, Run: again(o, func(*player.Player, *world.Tx) {})}
		if o == st.Lab {
			sl.Count = "§a▶"
		}
		if o == r {
			sl.Run = nil
		}
		sp.Slots = append(sp.Slots, sl)
	}
	sp.Pages = 1
	if r != nil {
		sp.Detail, sp.Rows, sp.Actions = LabDetail(s, lang, st, r, again)
	}
	ui.SplitMenu(p, i18n.T(lang, "machine.lab.name"), sp)
}

// LabOpenRecipes are the recipes whose essence the island's research has
// come to, in table order.
func LabOpenRecipes(s *core.Session) []*machinerecipe.Lab {
	var out []*machinerecipe.Lab
	for i := range machinerecipe.Labs {
		if r := &machinerecipe.Labs[i]; labOpen(s, r) {
			out = append(out, r)
		}
	}
	return out
}

// labStill reports whether the extractor a screen was shown for still
// stands where it did, in the session's world.
func labStill(s *core.Session, tx *world.Tx, pos cube.Pos, st *State) bool {
	m, ok := tx.Block(pos).(Machine)
	return tx.World() == s.World && ok && m.State == st
}

// labOpen reports whether the island's research has come to the essence a
// recipe makes: a research that costs it is done or can be done (or every
// recipe is open).
func labOpen(s *core.Session, r *machinerecipe.Lab) bool {
	if progress.LocalOf(s).RecipesFree {
		return true
	}
	for i := range progress.Tree {
		if rr := &progress.Tree[i]; progress.EssenceCost(rr, r.Out.Kind) > 0 && (progress.Researched(s, rr.ID) || progress.Open(s, rr)) {
			return true
		}
	}
	return false
}

// LabDetail is the right side of the screen for recipe r: the card, the
// rows of the ingredients and the essences made, and the buttons.
func LabDetail(s *core.Session, lang string, st *State, r *machinerecipe.Lab, again func(*machinerecipe.Lab, func(*player.Player, *world.Tx)) func(*player.Player, *world.Tx)) (ui.Detail, []ui.SlotRow, []ui.Option) {
	out := item.NewStack(r.Out, 1)
	d := ui.Detail{Icon: formicon.Of(out), Name: naming.UIItemName(lang, out)}
	running := st.Lab == r
	have := st.LabHave()
	var ins []ui.ItemSlot
	for _, in := range r.In {
		it := item.NewStack(in.Item, 1)
		sl := ui.ItemSlot{Icon: formicon.Of(it), Count: fmt.Sprint(in.Count), Tip: decor.TipName(lang, it) + "\n" + fmt.Sprintf(i18n.T(lang, "lab.need"), in.Count)}
		if running {
			n := have[naming.ItemKey(in.Item)]
			sl.Count = ui.CountText(n)
			if n < in.Count {
				sl.Count = "§c" + sl.Count
			}
			sl.Tip += "\n" + fmt.Sprintf(i18n.T(lang, "lab.held"), n)
			sl.Run = again(r, func(p *player.Player, _ *world.Tx) { labDeposit(s, p, st, in.Item) })
		}
		ins = append(ins, sl)
	}
	made := ui.ItemSlot{Icon: formicon.Of(out), Count: fmt.Sprint(r.OutCount), Tip: decor.TipName(lang, out) + "\n" + fmt.Sprintf(i18n.T(lang, "lab.makes"), r.OutCount, r.Ticks/20)}
	rows := []ui.SlotRow{{Heading: i18n.T(lang, "lab.ingredients"), Slots: ins}}
	if !running {
		d.Status = fmt.Sprintf(i18n.T(lang, "lab.makes"), r.OutCount, r.Ticks/20)
		rows = append(rows, ui.SlotRow{Heading: i18n.T(lang, "lab.output"), Slots: []ui.ItemSlot{made}})
		start := func(p *player.Player, tx *world.Tx) {
			if st.Lab != nil {
				labStop(s, p, tx, st)
			}
			st.SetLab(r)
			labDeposit(s, p, st, nil)
		}
		key := "lab.start"
		if st.Lab != nil {
			key = "lab.switch"
		}
		return d, rows, []ui.Option{{Label: i18n.T(lang, key), Icon: formicon.Of(out), Run: again(r, start)}}
	}
	n := OutputCount(st, r.Out)
	made.Count = ui.CountText(n)
	if n > 0 {
		made.Run = again(r, func(p *player.Player, tx *world.Tx) { labTake(s, p, tx, st) })
	}
	rows = append(rows, ui.SlotRow{Heading: i18n.T(lang, "lab.output"), Slots: []ui.ItemSlot{made}})
	d.Status = fmt.Sprintf(i18n.T(lang, "lab.progress"), min(100, st.Progress*100/r.Ticks))
	d.Body = strings.Join(LabLines(lang, st)[1:], "\n")
	actions := []ui.Option{
		{Label: i18n.T(lang, "lab.deposit"), Icon: formicon.Of(item.NewStack(r.In[0].Item, 1)), Run: again(r, func(p *player.Player, _ *world.Tx) { labDeposit(s, p, st, nil) })},
		{Label: i18n.T(lang, "lab.take"), Icon: formicon.Of(out), Run: again(r, func(p *player.Player, tx *world.Tx) { labTake(s, p, tx, st) })},
		{Label: i18n.T(lang, "lab.stop"), Icon: formicon.Locked, Run: again(r, func(p *player.Player, tx *world.Tx) { labStop(s, p, tx, st) })},
	}
	return d, rows, actions
}

// labDeposit puts the ingredients of the recipe (only, when it is set, the
// one that is that item) from p's inventory in the laboratory, up to
// labHandCap of each.
func labDeposit(s *core.Session, p *player.Player, st *State, only world.Item) {
	inv := p.Inventory()
	for slot, it := range inv.Slots() {
		if it.Empty() || !st.Accepts(it) || (only != nil && naming.ItemKey(it.Item()) != naming.ItemKey(only)) {
			continue
		}
		n := min(it.Count(), st.LabHandCap(it)-st.LabHave()[naming.ItemKey(it.Item())], st.inputRoom(it))
		if n <= 0 {
			continue
		}
		put(st.Inv(), st.Def.Input, it.Grow(n-it.Count()))
		_ = inv.SetItem(slot, it.Grow(-n))
	}
}

// inputRoom is how many of it the input slots still take.
func (st *State) inputRoom(it item.Stack) int {
	room := 0
	for _, slot := range st.Def.Input {
		cur, _ := st.Inv().Item(slot)
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
func labTake(s *core.Session, p *player.Player, tx *world.Tx, st *State) {
	labGive(s, p, tx, st, st.Def.Output)
}

// labStop stops the laboratory: p gets back what it holds, and it takes
// nothing until an essence is chosen again.
func labStop(s *core.Session, p *player.Player, tx *world.Tx, st *State) {
	labGive(s, p, tx, st, st.Def.Input)
	labGive(s, p, tx, st, st.Def.Output)
	st.SetLab(nil)
}

// labGive empties slots of the laboratory into p's inventory (what does not
// fit drops at p).
func labGive(s *core.Session, p *player.Player, tx *world.Tx, st *State, slots []int) {
	for _, slot := range slots {
		if it, _ := st.Inv().Item(slot); !it.Empty() {
			_ = st.Inv().SetItem(slot, item.Stack{})
			s.GiveItem(p, tx, p.Position(), it)
		}
	}
}
