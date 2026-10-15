package dragonfactory

import (
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/i18n"
)

// The laboratory (docs/design/11-research-essences.md): it brews a research
// essence from a glass bottle and the materials of an era, using energy and
// time. The recipe is whatever the nine input slots hold (as the alloy
// smelter's is): the first recipe of labRecipes whose ingredients are all in
// there. Its work is paced by the recipe, not the machine, so the machine
// keeps the length of the cycle it is on in machineState.period.

// LabIngredient is an item with the count a recipe takes.
type LabIngredient struct {
	Item  world.Item
	Count int
}

// LabRecipe is one recipe of the laboratory.
type LabRecipe struct {
	// Out is the essence made, OutCount how many at once.
	Out      ResearchEssence
	OutCount int
	// In are the ingredients, the glass bottle among them.
	In []LabIngredient
	// Ticks is how long a cycle takes.
	Ticks int
}

// labCheckEvery is how many ticks of a cycle pass between two looks at the
// input slots.
const labCheckEvery = 10

// labBottle is the glass bottle every recipe uses one of.
var labBottle = LabIngredient{item.GlassBottle{}, 1}

// labRecipes are the recipes, drafts (open-questions.md).
var labRecipes = []LabRecipe{
	{Out: ResearchEssence{EssenceEngineering}, OutCount: 2, Ticks: 160, In: []LabIngredient{
		labBottle, {item.IronIngot{}, 1}, {item.RedstoneWire{}, 2}}},
	{Out: ResearchEssence{EssenceAlchemy}, OutCount: 1, Ticks: 240, In: []LabIngredient{
		labBottle, {Ingot{Metal: MetalBronze}, 1}, {Ingot{Metal: MetalInvar}, 1}}},
	{Out: ResearchEssence{EssenceFarming}, OutCount: 1, Ticks: 240, In: []LabIngredient{
		labBottle, {item.Wheat{}, 1}, {block.Carrot{}, 1}, {block.Potato{}, 1}, {item.Beetroot{}, 1}, {block.Pumpkin{}, 1}, {item.MelonSlice{}, 1}}},
	{Out: ResearchEssence{EssenceHunting}, OutCount: 1, Ticks: 320, In: []LabIngredient{
		labBottle, {item.Gunpowder{}, 2}, {item.Bone{}, 2}, {item.Slimeball{}, 1}, {item.SpiderEye{}, 1}}},
	{Out: ResearchEssence{EssenceMagic}, OutCount: 1, Ticks: 320, In: []LabIngredient{
		labBottle, {Petal{Colour: "red"}, 1}, {Petal{Colour: "yellow"}, 1}, {Petal{Colour: "blue"}, 1}, {item.LapisLazuli{}, 1}, {item.AmethystShard{}, 1}}},
}

// labAccepts reports whether an item is an ingredient of some recipe.
func labAccepts(it item.Stack) bool {
	k := itemKey(it.Item())
	for _, r := range labRecipes {
		for _, in := range r.In {
			if itemKey(in.Item) == k {
				return true
			}
		}
	}
	return false
}

// labFeedCycles is how many cycles of an ingredient an automated feed
// (hoppers, conduits) leaves in the laboratory.
const labFeedCycles = 8

// labFeedCap is how many of an ingredient the feed puts in the laboratory:
// the most any recipe takes of it for labFeedCycles cycles. A feed of one
// ingredient cannot fill the nine slots and crowd the others out; what a
// player puts in by hand is not limited.
func labFeedCap(it item.Stack) int {
	k := itemKey(it.Item())
	most := 1
	for _, r := range labRecipes {
		for _, in := range r.In {
			if itemKey(in.Item) == k {
				most = max(most, in.Count)
			}
		}
	}
	return most * labFeedCycles
}

// labHave counts what the input slots hold by item key.
func (st *machineState) labHave() map[string]int {
	have := map[string]int{}
	for _, slot := range st.kind.input {
		if it, _ := st.inv().Item(slot); !it.Empty() {
			have[itemKey(it.Item())] += it.Count()
		}
	}
	return have
}

// labMissing lists what a recipe still lacks of the ingredients.
func labMissing(r *LabRecipe, have map[string]int) []LabIngredient {
	var missing []LabIngredient
	for _, in := range r.In {
		if short := in.Count - have[itemKey(in.Item)]; short > 0 {
			missing = append(missing, LabIngredient{in.Item, short})
		}
	}
	return missing
}

// labPlan is the recipe to work on: the one on now if its ingredients are
// still there, else the first one whose ingredients are.
func (st *machineState) labPlan() *LabRecipe {
	have := st.labHave()
	if r := st.lab; r != nil && len(labMissing(r, have)) == 0 {
		return r
	}
	for i := range labRecipes {
		if len(labMissing(&labRecipes[i], have)) == 0 {
			return &labRecipes[i]
		}
	}
	return nil
}

// labClosest is the recipe the input slots are closest to: the one with the
// most of its ingredients started, none if no ingredient is in them.
func (st *machineState) labClosest() (*LabRecipe, []LabIngredient) {
	have := st.labHave()
	var best *LabRecipe
	var bestMissing []LabIngredient
	bestHave := 0
	for i := range labRecipes {
		r := &labRecipes[i]
		got := 0
		for _, in := range r.In {
			if have[itemKey(in.Item)] > 0 {
				got++
			}
		}
		if got > bestHave {
			best, bestMissing, bestHave = r, labMissing(r, have), got
		}
	}
	return best, bestMissing
}

// runLab brews: it works the cycle of the recipe it has the ingredients
// for, and when the cycle is out and the essences fit the output, takes the
// ingredients and makes them. With no recipe the cycle starts over.
func (st *machineState) runLab() {
	// Mid-cycle the ingredients are looked at once every labCheckEvery ticks,
	// not every tick: finding the recipe counts the input slots.
	if r := st.lab; r != nil && st.progress%labCheckEvery != 0 && st.progress < r.Ticks {
		st.tickWork()
		return
	}
	r := st.labPlan()
	if r != st.lab {
		st.progress = 0
	}
	st.lab = r
	if r == nil {
		st.period = 0
		return
	}
	st.period = r.Ticks
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
			if cur.Empty() || itemKey(cur.Item()) != itemKey(in.Item) {
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

// labLines say what the laboratory brews, or what it lacks for the recipe
// it is closest to.
func labLines(st *machineState) []string {
	var lines []string
	if r := st.lab; r != nil {
		lines = append(lines, "§f"+itemName(item.NewStack(r.Out, r.OutCount)))
	} else if r, missing := st.labClosest(); r != nil {
		lines = append(lines, "§f"+itemName(item.NewStack(r.Out, 1)))
		var parts []string
		for _, m := range missing {
			parts = append(parts, fmt.Sprintf("%s ×%d", itemName(item.NewStack(m.Item, 1)), m.Count))
		}
		lines = append(lines, fmt.Sprintf(i18n.T(i18n.Default, "crafter.status.missing"), strings.Join(parts, ", ")))
	} else {
		lines = append(lines, i18n.T(i18n.Default, "dragonfactory.look.machineLines.4"))
	}
	lines = append(lines, consumerLines(st)...)
	return append(lines, blockedLines(st)...)
}

// labIngredientsText lists the ingredients of a recipe in one line.
func labIngredientsText(r *LabRecipe) string {
	var parts []string
	for _, in := range r.In {
		if in.Count > 1 {
			parts = append(parts, fmt.Sprintf("%s ×%d", itemName(item.NewStack(in.Item, 1)), in.Count))
		} else {
			parts = append(parts, itemName(item.NewStack(in.Item, 1)))
		}
	}
	return strings.Join(parts, ", ")
}
