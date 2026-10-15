package dragonfactory

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/i18n"
	"github.com/presentkim/dragonfactory/internal/ui"
)

// The auto crafter (docs/design/09-automation-equipment.md §2): it makes one
// crafting-table recipe over and over from the items in its 3×3 store.
// The recipe is chosen with a sample item put on its recipe slot, or from a
// list of the recipes researched; only researched recipes can be chosen.

// crafterRecipeSlot is the window slot of the recipe (ui/df_machine.json).
const crafterRecipeSlot = 4

// crafterReq is one ingredient of a recipe: an item (or a tag of items) and
// how many.
type crafterReq struct {
	need  recipe.Item
	key   string
	count int
}

// crafterRecipe is a crafting-table recipe the auto crafter can make.
type crafterRecipe struct {
	id   string
	r    recipe.Recipe
	out  item.Stack
	reqs []crafterReq
}

// reqKey identifies an ingredient; equal ingredients of a recipe are added.
func reqKey(in recipe.Item) string {
	switch in := in.(type) {
	case item.Stack:
		name, meta := in.Item().EncodeItem()
		if _, variants := in.Value("variants"); variants {
			return "v:" + name
		}
		return fmt.Sprintf("s:%s:%d", name, meta)
	case recipe.ItemTag:
		return "t:" + in.Tag()
	}
	return ""
}

var (
	crafterMu    sync.Mutex
	crafterList  []*crafterRecipe
	crafterBuilt int
)

// crafterRecipes lists the recipes the auto crafter can make: the shaped and
// shapeless ones of the crafting table (their shape does not matter), in
// registration order. The list is built again when recipes were registered.
func crafterRecipes() []*crafterRecipe {
	crafterMu.Lock()
	defer crafterMu.Unlock()
	all := recipe.Recipes()
	if crafterBuilt == len(all) && crafterList != nil {
		return crafterList
	}
	seen := map[string]int{}
	var list []*crafterRecipe
	for _, r := range all {
		if r.Block() != "crafting_table" {
			continue
		}
		switch r.(type) {
		case recipe.Shaped, recipe.Shapeless:
		default:
			continue
		}
		outs := r.Output()
		if len(outs) != 1 || outs[0].Empty() {
			continue
		}
		cr := &crafterRecipe{r: r, out: outs[0]}
		index := map[string]int{}
		for _, in := range r.Input() {
			if in == nil || in.Empty() {
				continue
			}
			k := reqKey(in)
			if i, ok := index[k]; ok {
				cr.reqs[i].count += in.Count()
				continue
			}
			index[k] = len(cr.reqs)
			cr.reqs = append(cr.reqs, crafterReq{need: in, key: k, count: in.Count()})
		}
		if len(cr.reqs) == 0 {
			continue
		}
		name, meta := outs[0].Item().EncodeItem()
		base := fmt.Sprintf("%s:%d", name, meta)
		cr.id = fmt.Sprintf("%s#%d", base, seen[base])
		seen[base]++
		list = append(list, cr)
	}
	crafterList, crafterBuilt = list, len(all)
	return list
}

// crafterByID returns the recipe with an id.
func crafterByID(id string) *crafterRecipe {
	for _, cr := range crafterRecipes() {
		if cr.id == id {
			return cr
		}
	}
	return nil
}

// matchReq reports whether a stack is the ingredient (the matching of the
// crafting grid).
func matchReq(has item.Stack, req crafterReq) bool {
	if has.Empty() {
		return false
	}
	switch need := req.need.(type) {
	case item.Stack:
		if _, variants := need.Value("variants"); variants {
			a, _ := has.Item().EncodeItem()
			b, _ := need.Item().EncodeItem()
			return a == b
		}
		return has.Comparable(need)
	case recipe.ItemTag:
		name, _ := has.Item().EncodeItem()
		return need.Contains(name)
	}
	return false
}

// crafterWants reports whether a recipe uses an item as an ingredient.
func (cr *crafterRecipe) wants(it item.Stack) bool {
	for _, req := range cr.reqs {
		if matchReq(it, req) {
			return true
		}
	}
	return false
}

// reqName is the name of an ingredient for a tooltip.
func reqName(lang string, req crafterReq) string {
	switch need := req.need.(type) {
	case item.Stack:
		return itemName(need)
	case recipe.ItemTag:
		return i18n.T(lang, "crafter.anyOf") + strings.TrimPrefix(need.Tag(), "minecraft:")
	}
	return ""
}

// tip is the tooltip of a recipe: the result and what it takes.
func (cr *crafterRecipe) tip(lang string) []string {
	lines := []string{"§7" + i18n.T(lang, "crafter.needs")}
	for _, req := range cr.reqs {
		lines = append(lines, fmt.Sprintf("§f%d× %s", req.count, reqName(lang, req)))
	}
	return lines
}

// preview is the item shown on the recipe slot.
func (cr *crafterRecipe) preview(lang string) item.Stack {
	return cr.out.WithCustomName("§r§f" + itemName(cr.out)).WithLore(cr.tip(lang)...)
}

// crafterPlan finds the stacks of the store that make one run of the recipe:
// the slot and how many of it to take. ok is false when something is
// missing.
func (st *machineState) crafterPlan(cr *crafterRecipe) (plan [][2]int, ok bool) {
	left := map[int]int{}
	stacks := map[int]item.Stack{}
	for _, slot := range st.kind.input {
		it, _ := st.inv().Item(slot)
		if !it.Empty() {
			left[slot], stacks[slot] = it.Count(), it
		}
	}
	for _, req := range cr.reqs {
		need := req.count
		for _, slot := range st.kind.input {
			if need == 0 {
				break
			}
			if left[slot] == 0 || !matchReq(stacks[slot], req) {
				continue
			}
			n := min(need, left[slot])
			left[slot] -= n
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
func (st *machineState) setRecipe(cr *crafterRecipe) {
	st.recipe, st.recipeID = cr, cr.id
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
	it := item.NewStack(item.Paper{}, 1).WithCustomName("§r§7" + i18n.T(i18n.Default, "crafter.none")).WithLore("§f" + i18n.T(i18n.Default, "crafter.none.hint"))
	if st.recipe != nil {
		it = st.recipe.preview(i18n.Default)
	}
	_ = st.inv().SetItem(crafterRecipeSlot, it)
}

// crafterCandidates are the researched recipes that make an item, the
// preferred first.
func (s *session) crafterCandidates(sample item.Stack) []*crafterRecipe {
	var out []*crafterRecipe
	for _, cr := range crafterRecipes() {
		if itemKey(cr.out.Item()) == itemKey(sample.Item()) && s.crafterOpen(cr) {
			out = append(out, cr)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].r.Priority() < out[j].r.Priority() })
	return out
}

// crafterOpen reports whether a recipe is researched (or all are open).
func (s *session) crafterOpen(cr *crafterRecipe) bool {
	return s.recipesFree || s.recipeFilter()(cr.r)
}

// runCrafter makes the recipe: the ingredients are used when the interval
// has passed and the result fits the output.
func (s *session) runCrafter(st *machineState) {
	if st.recipe == nil && st.recipeID != "" {
		st.recipe = crafterByID(st.recipeID)
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
	if !st.roomFor([]item.Stack{cr.out}) {
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
	st.emit(cr.out)
	st.progress = 0
}

// crafterPage is how many recipes the list shows at once.
const crafterPage = 36

// showCrafterRecipes lists the researched recipes (a page of them) to pick
// the one the crafter at pos makes.
func (s *session) showCrafterRecipes(p *player.Player, tx *world.Tx, pos cube.Pos, st *machineState, page int) {
	lang := i18n.Of(p)
	var all []*crafterRecipe
	for _, cr := range crafterRecipes() {
		if s.crafterOpen(cr) {
			all = append(all, cr)
		}
	}
	sort.SliceStable(all, func(i, j int) bool { return itemName(all[i].out) < itemName(all[j].out) })
	pages := max(1, (len(all)+crafterPage-1)/crafterPage)
	page = min(max(page, 0), pages-1)
	var slots []ui.ItemSlot
	for _, cr := range all[min(page*crafterPage, len(all)):min((page+1)*crafterPage, len(all))] {
		count := ""
		if cr.out.Count() > 1 {
			count = fmt.Sprint(cr.out.Count())
		}
		slots = append(slots, ui.ItemSlot{
			Icon: iconOf(cr.out), Count: count, Tip: "§f" + itemName(cr.out) + "\n" + strings.Join(cr.tip(lang), "\n"),
			Run: func(p *player.Player, tx *world.Tx) {
				if tx.World() != s.world {
					return
				}
				st.setRecipe(cr)
			},
		})
	}
	var actions []ui.Option
	if page > 0 {
		actions = append(actions, ui.Option{Label: i18n.T(lang, "crafter.prev"), Run: func(p *player.Player, tx *world.Tx) {
			s.showCrafterRecipes(p, tx, pos, st, page-1)
		}})
	}
	if page < pages-1 {
		actions = append(actions, ui.Option{Label: i18n.T(lang, "crafter.next"), Run: func(p *player.Player, tx *world.Tx) {
			s.showCrafterRecipes(p, tx, pos, st, page+1)
		}})
	}
	actions = append(actions, ui.Option{Label: i18n.T(lang, "crafter.clear"), Run: func(p *player.Player, tx *world.Tx) {
		if tx.World() == s.world {
			st.clearRecipe()
		}
	}})
	body := fmt.Sprintf(i18n.T(lang, "crafter.body"), page+1, pages, len(all))
	ui.ItemGrid(p, i18n.T(lang, "crafter.title"), body, slots, actions...)
}
