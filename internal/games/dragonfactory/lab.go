package dragonfactory

import (
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/i18n"
)

// The laboratory (docs/design/11-research-essences.md): it brews a research
// essence from a glass bottle and the materials of an era, using energy and
// time. The player chooses the essence on its screen (labui.go); until then
// it takes nothing, and then only the ingredients of that recipe. Its work is
// paced by the recipe, not the machine, so the machine keeps the length of
// the cycle it is on in machineState.period.

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

// labByKind is the recipe of the essence of a kind, nil if there is none.
func labByKind(kind string) *LabRecipe {
	for i := range labRecipes {
		if labRecipes[i].Out.Kind == kind {
			return &labRecipes[i]
		}
	}
	return nil
}

// ingredient is the ingredient of the recipe that the item is, if it is one.
func (r *LabRecipe) ingredient(it item.Stack) (LabIngredient, bool) {
	k := itemKey(it.Item())
	for _, in := range r.In {
		if itemKey(in.Item) == k {
			return in, true
		}
	}
	return LabIngredient{}, false
}

// setLab makes the laboratory brew r (nil: nothing), from the start of a
// cycle.
func (st *machineState) setLab(r *LabRecipe) {
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
	_, ok := st.lab.ingredient(it)
	return ok
}

// labFeedCycles is how many cycles of an ingredient an automated feed
// (hoppers, conduits) leaves in the laboratory.
const labFeedCycles = 8

// labFeedCap is how many of an ingredient the feed puts in the laboratory:
// what the recipe takes of it for labFeedCycles cycles. A feed of one
// ingredient cannot fill the nine slots and crowd the others out.
func (st *machineState) labFeedCap(it item.Stack) int {
	in, _ := st.lab.ingredient(it)
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

// labClosest is the recipe the input slots are closest to: the one with the
// most of its ingredients started, none if no ingredient is in them. A
// laboratory saved before it was chosen on its screen brews this one.
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
	if len(labMissing(r, st.labHave())) > 0 {
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

// labLines say in lang what the laboratory brews and what it lacks for it.
func labLines(lang string, st *machineState) []string {
	r := st.lab
	if r == nil {
		return []string{i18n.T(lang, "lab.none")}
	}
	lines := []string{"§f" + itemName(lang, item.NewStack(r.Out, r.OutCount))}
	if missing := labMissing(r, st.labHave()); len(missing) > 0 {
		var parts []string
		for _, m := range missing {
			parts = append(parts, fmt.Sprintf("%s ×%d", itemName(lang, item.NewStack(m.Item, 1)), m.Count))
		}
		lines = append(lines, fmt.Sprintf(i18n.T(lang, "crafter.status.missing"), strings.Join(parts, ", ")))
	}
	lines = append(lines, consumerLines(lang, st)...)
	return append(lines, blockedLines(lang, st)...)
}

// labIngredientsText lists the ingredients of a recipe in one line.
func labIngredientsText(lang string, r *LabRecipe) string {
	var parts []string
	for _, in := range r.In {
		if in.Count > 1 {
			parts = append(parts, fmt.Sprintf("%s ×%d", itemName(lang, item.NewStack(in.Item, 1)), in.Count))
		} else {
			parts = append(parts, itemName(lang, item.NewStack(in.Item, 1)))
		}
	}
	return strings.Join(parts, ", ")
}
