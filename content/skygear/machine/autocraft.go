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
func (st *MachineState) CrafterPlan(cr *machinerecipe.Crafter) (plan [][2]int, ok bool) {
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
func (st *MachineState) SetRecipe(cr *machinerecipe.Crafter) {
	st.Recipe, st.RecipeID = cr, cr.ID
	st.Progress = 0
	st.shownRecipe = "\x00"
}

// clearRecipe takes the recipe off.
func (st *MachineState) clearRecipe() {
	st.Recipe, st.RecipeID = nil, ""
	st.Progress = 0
	st.shownRecipe = "\x00"
}

// showRecipe puts the recipe (or the hint) on the recipe slot when it changed.
func (st *MachineState) showRecipe() {
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
func runCrafter(s *core.Session, st *MachineState) {
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

// crafterPage is how many recipes the list shows at once.
const crafterPage = 36

// showCrafterRecipes lists the researched recipes (a page of them) to pick
// the one the crafter at pos makes.
func showCrafterRecipes(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, st *MachineState, page int) {
	lang := i18n.Of(p)
	var all []*machinerecipe.Crafter
	open := crafterOpener(s)
	for _, cr := range machinerecipe.Crafters() {
		if open(cr) {
			all = append(all, cr)
		}
	}
	sort.SliceStable(all, func(i, j int) bool { return naming.ItemName(lang, all[i].Out) < naming.ItemName(lang, all[j].Out) })
	pages := max(1, (len(all)+crafterPage-1)/crafterPage)
	page = min(max(page, 0), pages-1)
	var slots []ui.ItemSlot
	for _, cr := range all[min(page*crafterPage, len(all)):min((page+1)*crafterPage, len(all))] {
		count := ""
		if cr.Out.Count() > 1 {
			count = fmt.Sprint(cr.Out.Count())
		}
		slots = append(slots, ui.ItemSlot{
			Icon: formicon.Of(cr.Out), Count: count, Tip: decor.TipName(lang, cr.Out) + "\n" + strings.Join(cr.Tip(lang), "\n"),
			Run: func(p *player.Player, tx *world.Tx) {
				if tx.World() != s.World {
					return
				}
				st.SetRecipe(cr)
			},
		})
	}
	actions := ui.Pager(p, page, pages, func(p *player.Player, tx *world.Tx, to int) { showCrafterRecipes(s, p, tx, pos, st, to) })
	actions = append(actions, ui.Option{Label: i18n.T(lang, "crafter.clear"), Run: func(p *player.Player, tx *world.Tx) {
		if tx.World() == s.World {
			st.clearRecipe()
		}
	}})
	body := fmt.Sprintf(i18n.T(lang, "crafter.body"), page+1, pages, len(all))
	ui.ItemGrid(p, i18n.T(lang, "crafter.title"), body, slots, actions...)
}
