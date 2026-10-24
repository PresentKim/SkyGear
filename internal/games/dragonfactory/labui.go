package dragonfactory

import (
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/common/ui"
)

// The laboratory's screen (docs/design/11-research-essences.md §3): using it
// shows the essences the island's research has come to as large cards, and a
// card chooses what it brews. A laboratory that brews something shows its
// recipe instead, with what it holds, and puts in the ingredients from the
// player's inventory, hands out the essences, or stops (giving everything
// back).

// showLab shows the laboratory's screen.
func (s *session) showLab(p *player.Player, _ *world.Tx, pos cube.Pos, st *machineState) {
	lang := i18n.Of(p)
	if st.lab == nil {
		ui.CardMenu(p, i18n.T(lang, "machine.lab.name"), "", s.labChoices(p, pos, st)...)
		return
	}
	body, rows, actions := s.labPanel(p, pos, st)
	ui.ItemBoard(p, ui.SubTitle(i18n.T(lang, "machine.lab.name"), itemName(item.NewStack(st.lab.Out, 1))), body, rows, actions...)
}

// labStill reports whether the laboratory a screen was shown for still
// stands where it did, in the session's world.
func (s *session) labStill(tx *world.Tx, pos cube.Pos, st *machineState) bool {
	m, ok := tx.Block(pos).(Machine)
	return tx.World() == s.world && ok && m.st == st
}

// labOpen reports whether the island's research has come to the essence a
// recipe makes: a research that costs it is done or can be done (or every
// recipe is open).
func (s *session) labOpen(r *LabRecipe) bool {
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

// labChoices are the cards of the essences that can be chosen.
func (s *session) labChoices(p *player.Player, pos cube.Pos, st *machineState) []ui.Option {
	lang := i18n.Of(p)
	var opts []ui.Option
	for i := range labRecipes {
		r := &labRecipes[i]
		if !s.labOpen(r) {
			continue
		}
		out := item.NewStack(r.Out, 1)
		opts = append(opts, ui.Option{
			Label: itemName(out) + "\n§8" + fmt.Sprintf(i18n.T(lang, "lab.card"), r.OutCount, r.Ticks/20),
			Icon:  iconOf(out),
			Run: func(p *player.Player, tx *world.Tx) {
				if !s.labStill(tx, pos, st) || st.lab != nil {
					return
				}
				st.setLab(r)
				s.labDeposit(p, st, nil)
				s.showLab(p, tx, pos, st)
			},
		})
	}
	return opts
}

// labPanel is the screen of a laboratory that brews: a row of the
// ingredients with how many of each it holds (pressing one puts that one in),
// an arrow and the essences made (pressing them takes them), the state, and
// the buttons.
func (s *session) labPanel(p *player.Player, pos cube.Pos, st *machineState) (string, [][]ui.BoardCell, []ui.Option) {
	lang := i18n.Of(p)
	r := st.lab
	// again runs f on the laboratory, if it is still there, and shows the
	// screen again.
	again := func(f func(p *player.Player, tx *world.Tx)) func(p *player.Player, tx *world.Tx) {
		return func(p *player.Player, tx *world.Tx) {
			if !s.labStill(tx, pos, st) {
				return
			}
			f(p, tx)
			s.showLab(p, tx, pos, st)
		}
	}
	have := st.labHave()
	var row []ui.BoardCell
	for _, in := range r.In {
		it := item.NewStack(in.Item, 1)
		count := ""
		if n := have[itemKey(in.Item)]; n > 0 {
			count = fmt.Sprint(n)
		}
		row = append(row, ui.SlotCell(ui.ItemSlot{
			Icon: iconOf(it), Count: count, Tip: tipName(it) + "\n" + fmt.Sprintf(i18n.T(lang, "lab.need"), in.Count),
			Run: again(func(p *player.Player, _ *world.Tx) { s.labDeposit(p, st, in.Item) }),
		}))
	}
	out := item.NewStack(r.Out, 1)
	made := ""
	if n := outputCount(st, r.Out); n > 0 {
		made = fmt.Sprint(n)
	}
	row = append(row, ui.Arrow(), ui.SlotCell(ui.ItemSlot{
		Icon: iconOf(out), Count: made, Tip: tipName(out) + "\n" + fmt.Sprintf(i18n.T(lang, "lab.makes"), r.OutCount, r.Ticks/20),
		Run: again(func(p *player.Player, tx *world.Tx) { s.labTake(p, tx, st) }),
	}))
	lines := []string{fmt.Sprintf(i18n.T(lang, "lab.progress"), min(100, st.progress*100/r.Ticks))}
	lines = append(lines, labLines(st)[1:]...)
	actions := []ui.Option{
		{Label: i18n.T(lang, "lab.deposit"), Icon: iconOf(item.NewStack(r.In[0].Item, 1)), Run: again(func(p *player.Player, _ *world.Tx) { s.labDeposit(p, st, nil) })},
		{Label: i18n.T(lang, "lab.take"), Icon: iconOf(out), Run: again(func(p *player.Player, tx *world.Tx) { s.labTake(p, tx, st) })},
		{Label: i18n.T(lang, "lab.stop"), Icon: iconLocked, Run: again(func(p *player.Player, tx *world.Tx) { s.labStop(p, tx, st) })},
	}
	return strings.Join(lines, "\n"), [][]ui.BoardCell{row}, actions
}

// labDeposit puts the ingredients of the recipe (only, when it is set, the
// one that is that item) from p's inventory in the laboratory, up to
// labHandCap of each.
func (s *session) labDeposit(p *player.Player, st *machineState, only world.Item) {
	inv := p.Inventory()
	for slot, it := range inv.Slots() {
		if it.Empty() || !st.accepts(it) || (only != nil && itemKey(it.Item()) != itemKey(only)) {
			continue
		}
		n := min(it.Count(), st.labHandCap(it)-st.labHave()[itemKey(it.Item())], st.inputRoom(it))
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
