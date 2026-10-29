package skygear

import (
	"fmt"
	"sort"
	"strings"

	"github.com/presentkim/skygear/content/skygear/decor"

	"github.com/presentkim/skygear/content/skygear/progress"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/formicon"
	"github.com/presentkim/skygear/content/skygear/machinerecipe"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/wintext"
)

// The auto crafter: it makes one
// crafting-table recipe over and over from the items in its 3×3 store.
// The recipe is chosen with a sample item put on its recipe slot, or from a
// list of the recipes researched; only researched recipes can be chosen.

// crafterRecipeSlot is the window slot of the recipe (ui/sg_machine.json).
const crafterRecipeSlot = 4

// crafterStore is how many slots the 3×3 store has.
const crafterStore = 9

// crafterPlan finds the stacks of the store that make one run of the recipe:
// the slot and how many of it to take. ok is false when something is
// missing.
func (st *machineState) crafterPlan(cr *machinerecipe.Crafter) (plan [][2]int, ok bool) {
	var left [crafterStore]int
	var stacks [crafterStore]item.Stack
	for i, slot := range st.kind.input {
		it, _ := st.inv().Item(slot)
		if !it.Empty() {
			left[i], stacks[i] = it.Count(), it
		}
	}
	for _, req := range cr.Reqs {
		need := req.Count
		for i, slot := range st.kind.input {
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
func (st *machineState) setRecipe(cr *machinerecipe.Crafter) {
	st.recipe, st.recipeID = cr, cr.ID
	st.progress = 0
	st.shownRecipe = "\x00"
}

// clearRecipe takes the recipe off.
func (st *machineState) clearRecipe() {
	st.recipe, st.recipeID = nil, ""
	st.progress = 0
	st.shownRecipe = "\x00"
}

// showRecipe puts the recipe (or the hint) on the recipe slot when it changed.
func (st *machineState) showRecipe() {
	if st.shownRecipe == st.recipeID && st.shownRecipe != "\x00" {
		return
	}
	st.shownRecipe = st.recipeID
	it := wintext.With(item.NewStack(item.Paper{}, 1), wintext.Text{wintext.Raw("§r§7"), wintext.Key("crafter.none")}, wintext.Text{wintext.Raw("§f"), wintext.Key("crafter.none.hint")})
	if st.recipe != nil {
		it = st.recipe.Preview()
	}
	_ = st.inv().SetItem(crafterRecipeSlot, it)
}

// crafterCandidates are the researched recipes that make an item, the
// preferred first.
func (s *session) crafterCandidates(sample item.Stack) []*machinerecipe.Crafter {
	var out []*machinerecipe.Crafter
	k := naming.ItemKey(sample.Item())
	open := s.crafterOpener()
	for _, cr := range machinerecipe.Crafters() {
		if cr.OutKey == k && open(cr) {
			out = append(out, cr)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Recipe.Priority() < out[j].Recipe.Priority() })
	return out
}

// crafterOpen reports whether a recipe is researched (or all are open).
func (s *session) crafterOpen(cr *machinerecipe.Crafter) bool {
	return s.crafterOpener()(cr)
}

// crafterOpener returns crafterOpen for the researches done now.
func (s *session) crafterOpener() func(*machinerecipe.Crafter) bool {
	if progress.LocalOf(s.Session).RecipesFree {
		return func(*machinerecipe.Crafter) bool { return true }
	}
	filter := progress.RecipeFilter(s.Session)
	return func(cr *machinerecipe.Crafter) bool { return filter(cr.Recipe) }
}

// runCrafter makes the recipe: the ingredients are used when the interval
// has passed and the result fits the output.
func (s *session) runCrafter(st *machineState) {
	if st.recipe == nil && st.recipeID != "" {
		st.recipe = machinerecipe.CrafterByID(st.recipeID)
	}
	if st.dropRecipe {
		st.dropRecipe = false
		st.clearRecipe()
	}
	if sample := st.sample; !sample.Empty() {
		st.sample = item.Stack{}
		if c := s.crafterCandidates(sample); len(c) > 0 {
			st.setRecipe(c[0])
		}
	}
	st.showRecipe()
	cr := st.recipe
	if cr == nil {
		st.progress = 0
		return
	}
	plan, ok := st.crafterPlan(cr)
	if !ok {
		st.progress = 0
		return
	}
	if !st.roomFor([]item.Stack{cr.Out}) {
		return
	}
	if st.progress < st.kind.interval {
		st.tickWork()
		return
	}
	for _, p := range plan {
		it, _ := st.inv().Item(p[0])
		_ = st.inv().SetItem(p[0], it.Grow(-p[1]))
	}
	st.emit(cr.Out)
	st.progress = 0
}

// crafterPage is how many recipes the list shows at once.
const crafterPage = 36

// showCrafterRecipes lists the researched recipes (a page of them) to pick
// the one the crafter at pos makes.
func (s *session) showCrafterRecipes(p *player.Player, tx *world.Tx, pos cube.Pos, st *machineState, page int) {
	lang := i18n.Of(p)
	var all []*machinerecipe.Crafter
	open := s.crafterOpener()
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
				st.setRecipe(cr)
			},
		})
	}
	actions := ui.Pager(p, page, pages, func(p *player.Player, tx *world.Tx, to int) { s.showCrafterRecipes(p, tx, pos, st, to) })
	actions = append(actions, ui.Option{Label: i18n.T(lang, "crafter.clear"), Run: func(p *player.Player, tx *world.Tx) {
		if tx.World() == s.World {
			st.clearRecipe()
		}
	}})
	body := fmt.Sprintf(i18n.T(lang, "crafter.body"), page+1, pages, len(all))
	ui.ItemGrid(p, i18n.T(lang, "crafter.title"), body, slots, actions...)
}
