package machine

import (
	"fmt"
	"sort"
	"strings"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/invclick"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/decor"
	"github.com/presentkim/skygear/content/skygear/formicon"
	"github.com/presentkim/skygear/content/skygear/machinerecipe"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/wintext"
)

// The auto crafter: it makes one
// crafting-table recipe over and over from the items in its 3×3 store.
// The recipe is chosen with a sample item put on its recipe slot, or from a
// list of the recipes researched; only researched recipes can be chosen.

// CrafterRecipeSlot is the window slot of the recipe (ui/sg_machine.json).
const CrafterRecipeSlot = 4

// crafterStore is how many slots the 3×3 store has.
const crafterStore = 9

// crafterPlan finds the stacks of the store that make one run of the recipe:
// the slot and how many of it to take. ok is false when something is
// missing.
func (st *State) CrafterPlan(cr *machinerecipe.Crafter) (plan [][2]int, ok bool) {
	var left [crafterStore]int
	var stacks [crafterStore]item.Stack
	for i, slot := range st.Def.Input {
		it, _ := st.Inv().Item(slot)
		if !it.Empty() {
			left[i], stacks[i] = it.Count(), it
		}
	}
	for _, req := range cr.Reqs {
		need := req.Count
		for i, slot := range st.Def.Input {
			if need == 0 {
				break
			}
			if left[i] == 0 || !machinerecipe.MatchReq(stacks[i], req) {
				continue
			}
			n := min(need, left[i])
			left[i] -= n
			need -= n
			plan = append(plan, [2]int{slot, n})
		}
		if need > 0 {
			return nil, false
		}
	}
	return plan, true
}

// setRecipe makes the crafter work on a recipe.
func (st *State) SetRecipe(cr *machinerecipe.Crafter) {
	st.Recipe, st.RecipeID = cr, cr.ID
	st.Progress = 0
	st.shownRecipe = "\x00"
}

// clearRecipe takes the recipe off.
func (st *State) clearRecipe() {
	st.Recipe, st.RecipeID = nil, ""
	st.Progress = 0
	st.shownRecipe = "\x00"
}

// showRecipe puts the recipe (or the hint) on the recipe slot when it changed.
func (st *State) showRecipe() {
	if st.shownRecipe == st.RecipeID && st.shownRecipe != "\x00" {
		return
	}
	st.shownRecipe = st.RecipeID
	it := wintext.With(item.NewStack(item.Paper{}, 1), wintext.Text{wintext.Raw("§r§7"), wintext.Key("crafter.none")}, wintext.Text{wintext.Raw("§f"), wintext.Key("crafter.none.hint")})
	if st.Recipe != nil {
		it = st.Recipe.Preview()
	}
	_ = st.Inv().SetItem(CrafterRecipeSlot, it)
}

// CrafterCandidates are the researched recipes that make an item, the
// preferred first.
func CrafterCandidates(s *core.Session, sample item.Stack) []*machinerecipe.Crafter {
	var out []*machinerecipe.Crafter
	k := naming.ItemKey(sample.Item())
	open := crafterOpener(s)
	for _, cr := range machinerecipe.Crafters() {
		if cr.OutKey == k && open(cr) {
			out = append(out, cr)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Recipe.Priority() < out[j].Recipe.Priority() })
	return out
}

// CrafterOpen reports whether a recipe is researched (or all are open).
func CrafterOpen(s *core.Session, cr *machinerecipe.Crafter) bool {
	return crafterOpener(s)(cr)
}

// crafterOpener returns crafterOpen for the researches done now.
func crafterOpener(s *core.Session) func(*machinerecipe.Crafter) bool {
	if progress.LocalOf(s).RecipesFree {
		return func(*machinerecipe.Crafter) bool { return true }
	}
	filter := progress.RecipeFilter(s)
	return func(cr *machinerecipe.Crafter) bool { return filter(cr.Recipe) }
}

// runCrafter makes the recipe: the ingredients are used when the interval
// has passed and the result fits the output.
func runCrafter(s *core.Session, st *State) {
	if st.Recipe == nil && st.RecipeID != "" {
		st.Recipe = machinerecipe.CrafterByID(st.RecipeID)
	}
	if st.dropRecipe {
		st.dropRecipe = false
		st.clearRecipe()
	}
	if sample := st.Sample; !sample.Empty() {
		st.Sample = item.Stack{}
		if c := CrafterCandidates(s, sample); len(c) > 0 {
			st.SetRecipe(c[0])
		}
	}
	st.showRecipe()
	cr := st.Recipe
	if cr == nil {
		st.Progress = 0
		return
	}
	plan, ok := st.CrafterPlan(cr)
	if !ok {
		st.Progress = 0
		return
	}
	if !st.RoomFor([]item.Stack{cr.Out}) {
		return
	}
	if st.Progress < st.Def.Interval {
		st.TickWork()
		return
	}
	for _, p := range plan {
		it, _ := st.Inv().Item(p[0])
		_ = st.Inv().SetItem(p[0], it.Grow(-p[1]))
	}
	st.Emit(cr.Out)
	st.Progress = 0
}

// stateSlots is some of a machine's window slots, as an invclick area; put
// tells which take an item put in (an input, or nothing for the outputs).
type stateSlots struct {
	st    *State
	slots []int
	put   bool
}

func (a stateSlots) Len() int { return len(a.slots) }
func (a stateSlots) Item(i int) item.Stack {
	it, _ := a.st.Inv().Item(a.slots[i])
	return it
}
func (a stateSlots) Set(i int, it item.Stack) { _ = a.st.Inv().SetItem(a.slots[i], it) }
func (a stateSlots) Accepts(i int, it item.Stack) bool {
	return a.put && a.st.SlotAccepts(a.slots[i], it)
}

// crafterStatus says what the crafter does: its recipe and what it lacks,
// and its energy.
func crafterStatus(lang string, st *State) string {
	line := i18n.T(lang, "crafter.status.none")
	if st.Recipe != nil {
		line = "§f" + naming.ItemName(lang, st.Recipe.Out) + " · "
		if _, ok := st.CrafterPlan(st.Recipe); ok {
			line += i18n.T(lang, "crafter.status.working")
		} else {
			line += fmt.Sprintf(i18n.T(lang, "crafter.status.missing"), CrafterMissing(lang, st))
		}
	}
	return strings.Join(append([]string{line}, ConsumerLines(lang, st)...), "\n")
}

// showCrafter is the crafter's window: the researched recipes to pick the
// one it makes, its 3×3 store and outputs, and the player's inventory.
func showCrafter(s *core.Session, p *player.Player, pos cube.Pos, st *State) {
	lang := i18n.Of(p)
	var all []*machinerecipe.Crafter
	open := crafterOpener(s)
	for _, cr := range machinerecipe.Crafters() {
		if open(cr) {
			all = append(all, cr)
		}
	}
	sort.SliceStable(all, func(i, j int) bool { return naming.ItemName(lang, all[i].Out) < naming.ItemName(lang, all[j].Out) })
	recipes := make([]ui.InvSlot, len(all))
	for i, cr := range all {
		sl := ui.InvSlot{Icon: formicon.Of(cr.Out), Tip: decor.TipName(lang, cr.Out) + "\n" + strings.Join(cr.Tip(lang), "\n")}
		if cr.Out.Count() > 1 {
			sl.Count = fmt.Sprint(cr.Out.Count())
		}
		if st.Recipe == cr {
			sl.Count = "§a✔"
		}
		recipes[i] = sl
	}
	for len(recipes) < 9 || len(recipes)%9 != 0 {
		recipes = append(recipes, ui.InvSlot{})
	}
	tipOf := func(it item.Stack) string {
		if it.Empty() {
			return ""
		}
		return decor.TipName(lang, it)
	}
	in, out := stateSlots{st, st.Def.Input, true}, stateSlots{st, st.Def.Output, false}
	main, hot := invclick.Main(p.Inventory()), invclick.Hotbar(p.Inventory())
	slotsOf := func(a invclick.Area) []ui.InvSlot {
		sl := make([]ui.InvSlot, a.Len())
		for i := range sl {
			sl[i] = ui.StackSlot(a.Item(i), tipOf(a.Item(i)))
		}
		return sl
	}
	result := ui.InvSlot{}
	if st.Recipe != nil {
		result = ui.StackSlot(st.Recipe.Out, tipOf(st.Recipe.Out))
	}
	clear := ui.Option{Label: i18n.T(lang, "crafter.clear"), Icon: "textures/ui/cancel", Run: func(p *player.Player, tx *world.Tx) {
		st.clearRecipe()
		showCrafter(s, p, pos, st)
	}}
	areas := map[string]invclick.Area{"in": in, "out": out, "inv": main, "hotbar": hot}
	ui.ShowInventory(p, ui.InventoryScreen{
		Title: i18n.T(lang, "machine.crafter.name"),
		Info:  crafterStatus(lang, st),
		Rows: []ui.InvRow{
			{Grow: true, Parts: []ui.InvPart{{Area: "recipe", Cols: 9, Slots: recipes}}},
			{Heading: i18n.T(lang, "crafter.window.store"), Parts: []ui.InvPart{
				{Area: "in", Cols: 3, Slots: slotsOf(in)},
				{Arrow: true},
				{Area: "result", Cols: 1, Big: true, Slots: []ui.InvSlot{result}},
				{Area: "out", Cols: 2, Slots: slotsOf(out)},
				{Button: &clear},
			}},
			{Heading: i18n.T(lang, "crafter.window.inventory"), Parts: []ui.InvPart{{Area: "inv", Cols: 9, Slots: slotsOf(main)}}},
			{Parts: []ui.InvPart{{Area: "hotbar", Cols: 9, Slots: slotsOf(hot)}}},
		},
		Slot: func(p *player.Player, tx *world.Tx, a ui.SlotAct) {
			if m, ok := tx.Block(pos).(Machine); tx.World() != s.World || !ok || m.State != st {
				return
			}
			switch area, ok := areas[a.Area]; {
			case a.Area == "recipe" && a.Index < len(all):
				st.SetRecipe(all[a.Index])
			case !ok || a.Index >= area.Len():
			case a.Shift && (a.Area == "in" || a.Area == "out"):
				area.Set(a.Index, invclick.Move(area.Item(a.Index), hot, main))
			case a.Shift:
				it := area.Item(a.Index)
				area.Set(a.Index, it.Grow(-st.Fill(it)))
			default:
				ui.SetCarried(p, invclick.Click(area, a.Index, a.Carried, a.Right))
			}
			showCrafter(s, p, pos, st)
		},
		Again: func(p *player.Player, tx *world.Tx) { showCrafter(s, p, pos, st) },
		Size:  "crafter",
	})
}
