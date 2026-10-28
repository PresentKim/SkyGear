package skygear

import (
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/internal/skygear/machinerecipe"
	"github.com/presentkim/skygear/internal/skygear/naming"
)

// The laboratory brews the research essence chosen on its screen from a
// glass bottle and an era's materials. Its work is paced by the recipe, not
// the machine, so the cycle length is kept in machineState.period.

// labCheckEvery is how many ticks of a cycle pass between two looks at the
// input slots.
const labCheckEvery = 10

// setLab makes the laboratory brew r (nil: nothing), from the start of a
// cycle.
func (st *machineState) setLab(r *machinerecipe.Lab) {
	st.lab, st.recipeID, st.progress, st.period = r, "", 0, 0
	if r != nil {
		st.recipeID, st.period = r.Out.Kind, r.Ticks
	}
}

// labAccepts reports whether the laboratory takes an item: an ingredient of
// the recipe it brews.
func (st *machineState) labAccepts(it item.Stack) bool {
	if st.lab == nil {
		return false
	}
	_, ok := st.lab.Ingredient(it)
	return ok
}

// labFeedCycles is how many cycles of an ingredient an automated feed
// (hoppers, conduits) leaves in the laboratory.
const labFeedCycles = 8

// labFeedCap is how many of an ingredient the feed puts in the laboratory:
// what the recipe takes of it for labFeedCycles cycles. A feed of one
// ingredient cannot fill the nine slots and crowd the others out.
func (st *machineState) labFeedCap(it item.Stack) int {
	in, _ := st.lab.Ingredient(it)
	return max(1, in.Count) * labFeedCycles
}

// labHandCap is how many of an ingredient a player puts in by hand: the
// stacks of the input slots shared out among the ingredients.
func (st *machineState) labHandCap(it item.Stack) int {
	return it.MaxCount() * max(1, len(st.kind.input)/len(st.lab.In))
}

// labHave counts what the input slots hold by item key.
func (st *machineState) labHave() map[string]int {
	have := map[string]int{}
	for _, slot := range st.kind.input {
		if it, _ := st.inv().Item(slot); !it.Empty() {
			have[naming.ItemKey(it.Item())] += it.Count()
		}
	}
	return have
}

// labClosest is the recipe the input slots are closest to: the one with the
// most of its ingredients started, none if no ingredient is in them. A
// laboratory saved before it was chosen on its screen brews this one.
func (st *machineState) labClosest() (*machinerecipe.Lab, []machinerecipe.Ingredient) {
	have := st.labHave()
	var best *machinerecipe.Lab
	var bestMissing []machinerecipe.Ingredient
	bestHave := 0
	for i := range machinerecipe.Labs {
		r := &machinerecipe.Labs[i]
		got := 0
		for _, in := range r.In {
			if have[naming.ItemKey(in.Item)] > 0 {
				got++
			}
		}
		if got > bestHave {
			best, bestMissing, bestHave = r, machinerecipe.LabMissing(r, have), got
		}
	}
	return best, bestMissing
}

// runLab brews: it works the cycle of the chosen recipe while its
// ingredients are there, and when the cycle is out and the essences fit the
// output, takes the ingredients and makes them. Without an ingredient the
// cycle starts over.
func (st *machineState) runLab() {
	r := st.lab
	if r == nil {
		st.progress, st.period = 0, 0
		return
	}
	st.period = r.Ticks
	// Mid-cycle the ingredients are looked at once every labCheckEvery ticks,
	// not every tick: it counts the input slots.
	if st.progress%labCheckEvery != 0 && st.progress < r.Ticks {
		st.tickWork()
		return
	}
	if len(machinerecipe.LabMissing(r, st.labHave())) > 0 {
		st.progress = 0
		return
	}
	if st.progress < r.Ticks {
		st.tickWork()
		return
	}
	out := item.NewStack(r.Out, r.OutCount)
	if !st.roomFor([]item.Stack{out}) {
		return
	}
	for _, in := range r.In {
		left := in.Count
		for _, slot := range st.kind.input {
			cur, _ := st.inv().Item(slot)
			if left == 0 {
				break
			}
			if cur.Empty() || naming.ItemKey(cur.Item()) != naming.ItemKey(in.Item) {
				continue
			}
			take := min(left, cur.Count())
			_ = st.inv().SetItem(slot, cur.Grow(-take))
			left -= take
		}
	}
	st.emit(out)
	st.progress = 0
	st.done++
	st.tally("lab_"+r.Out.Kind, int64(r.OutCount))
}

// labLines say in lang what the laboratory brews and what it lacks for it.
func labLines(lang string, st *machineState) []string {
	r := st.lab
	if r == nil {
		return []string{i18n.T(lang, "lab.none")}
	}
	lines := []string{"§f" + naming.ItemName(lang, item.NewStack(r.Out, r.OutCount))}
	if missing := machinerecipe.LabMissing(r, st.labHave()); len(missing) > 0 {
		var parts []string
		for _, m := range missing {
			parts = append(parts, fmt.Sprintf("%s ×%d", naming.ItemName(lang, item.NewStack(m.Item, 1)), m.Count))
		}
		lines = append(lines, fmt.Sprintf(i18n.T(lang, "crafter.status.missing"), strings.Join(parts, ", ")))
	}
	lines = append(lines, consumerLines(lang, st)...)
	return append(lines, blockedLines(lang, st)...)
}
