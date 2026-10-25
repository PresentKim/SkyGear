package dragonfactory

import (
	"fmt"
	"sync"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/hud"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/common/ui"
)

// Goal items: an item the save's players chose to make. The codex works out
// what it takes down to the raw materials, and the quest panel keeps those
// in view (docs/tech/codex-jei.md).

// goalMax is how many goals the save keeps; a new one pushes the oldest out.
const goalMax = 3

// goalChainDepth is how many recipes deep the raw materials are followed.
const goalChainDepth = 6

// recipeStep is one way to make an item: how many it makes and from what.
type recipeStep struct {
	out int
	ins []codexLink
}

// codexRecipeOf returns the recipe that makes an item: a crafting table
// recipe (an ingredient that takes any item of a tag counts as one of them),
// an alloy or a lab recipe. Of the crafting recipes, one whose ingredient is
// itself made from the item (a block of diamonds back into diamonds) is left
// out, and the one that makes the most per ingredient is taken (sticks from
// planks before sticks from bamboo).
func codexRecipeOf(it item.Stack) (recipeStep, bool) {
	k := itemKey(it.Item())
	best, bestRate := recipeStep{}, 0.0
	for _, cr := range crafterRecipes() {
		if cr.outKey != k {
			continue
		}
		step := recipeStep{out: max(1, cr.out.Count())}
		ins := 0
		for _, req := range cr.reqs {
			var st item.Stack
			switch need := req.need.(type) {
			case item.Stack:
				st = need
			case recipe.ItemTag:
				st = tagStack(need)
			}
			if st.Empty() {
				step.ins = nil
				break
			}
			step.ins = append(step.ins, codexLink{stack: item.NewStack(st.Item(), 1), count: req.count})
			ins += req.count
		}
		if len(step.ins) == 0 || codexReverses(step, k) {
			continue
		}
		if rate := float64(step.out) / float64(max(1, ins)); rate > bestRate {
			best, bestRate = step, rate
		}
	}
	if bestRate > 0 {
		return best, true
	}
	for _, r := range alloyRecipes {
		if itemKey(r.Out) == k {
			return recipeStep{out: max(1, r.OutCount), ins: []codexLink{
				{stack: item.NewStack(r.A, 1), count: r.CountA}, {stack: item.NewStack(r.B, 1), count: r.CountB}}}, true
		}
	}
	for i := range labRecipes {
		if itemKey(labRecipes[i].Out) == k {
			step := recipeStep{out: max(1, labRecipes[i].OutCount)}
			for _, in := range labRecipes[i].In {
				step.ins = append(step.ins, codexLink{stack: item.NewStack(in.Item, 1), count: in.Count})
			}
			return step, true
		}
	}
	return recipeStep{}, false
}

// codexRaw is what n of an item takes in raw materials: the items that no
// recipe makes, after following the recipes down. An item made by a recipe
// that loops back to it counts as raw.
func codexRaw(it item.Stack, n int) []codexLink {
	var order []string
	total := map[string]*codexLink{}
	var walk func(it item.Stack, n, depth int, path map[string]bool)
	walk = func(it item.Stack, n, depth int, path map[string]bool) {
		k := itemKey(it.Item())
		step, ok := codexRecipeOf(it)
		if !ok || depth >= goalChainDepth || path[k] {
			if total[k] == nil {
				total[k] = &codexLink{stack: it}
				order = append(order, k)
			}
			total[k].count += n
			return
		}
		path[k] = true
		crafts := (n + step.out - 1) / step.out
		for _, in := range step.ins {
			walk(in.stack, crafts*max(1, in.count), depth+1, path)
		}
		delete(path, k)
	}
	walk(it, n, 0, map[string]bool{})
	out := make([]codexLink, 0, len(order))
	for _, k := range order {
		out = append(out, *total[k])
	}
	return out
}

var (
	codexRawMu   sync.Mutex
	codexRawOnes = map[string][]codexLink{}
)

// codexRawOne is codexRaw(it, 1), worked out once for each item key. The
// result is shared: callers only read it.
func codexRawOne(it item.Stack) []codexLink {
	k := itemKey(it.Item())
	codexRawMu.Lock()
	defer codexRawMu.Unlock()
	if raw, ok := codexRawOnes[k]; ok {
		return raw
	}
	raw := codexRaw(it, 1)
	codexRawOnes[k] = raw
	return raw
}

// codexItemByKey finds the indexed item with the key.
func codexItemByKey(key string) (item.Stack, bool) {
	codexIndex(i18n.Default)
	st, ok := codexIndexByKey[key]
	return st, ok
}

// goalItems are the goals of the save as items.
func (s *session) goalItems() []item.Stack {
	var out []item.Stack
	for _, k := range s.state.Goals {
		if st, ok := codexItemByKey(k); ok {
			out = append(out, st)
		}
	}
	return out
}

func (s *session) isGoal(it item.Stack) bool {
	k := itemKey(it.Item())
	for _, g := range s.state.Goals {
		if g == k {
			return true
		}
	}
	return false
}

// toggleGoal makes the item a goal, or removes it from the goals.
func (s *session) toggleGoal(tx *world.Tx, it item.Stack) {
	k := itemKey(it.Item())
	if s.isGoal(it) {
		var keep []string
		for _, g := range s.state.Goals {
			if g != k {
				keep = append(keep, g)
			}
		}
		s.state.Goals = keep
	} else {
		s.state.Goals = append(s.state.Goals, k)
		if len(s.state.Goals) > goalMax {
			s.state.Goals = s.state.Goals[len(s.state.Goals)-goalMax:]
		}
	}
	s.updateBoards(tx)
}

// goalRows are the quest panel rows of the goals: the name and what is
// needed in raw materials.
func (s *session) goalRows(lang string) []string {
	var rows []string
	for _, it := range s.goalItems() {
		rows = append(rows, fmt.Sprintf(i18n.T(lang, "goal.row"), itemName(lang, it)))
		raw := codexRawOne(it)
		for i, l := range raw {
			if i == 4 {
				rows = append(rows, fmt.Sprintf(i18n.T(lang, "goal.more"), len(raw)-i))
				break
			}
			rows = append(rows, fmt.Sprintf("§f%s §e×%d", itemName(lang, l.stack), max(1, l.count)))
		}
	}
	return rows
}

// addGoalRows puts the goals under the quest panel.
func (s *session) addGoalRows(lang string, b *hud.Board) {
	for _, r := range s.goalRows(lang) {
		b.Add(hud.Quest, r)
	}
}

// goalSlots are the goals as slots that open their pages.
func (s *session) goalSlots(lang string) []ui.ItemSlot {
	var links []codexLink
	for _, it := range s.goalItems() {
		links = append(links, codexLink{stack: it})
	}
	return s.codexLinkSlots(lang, links)
}

// codexReverses reports whether a recipe takes an ingredient that a crafting
// recipe makes from the item key alone (the recipe undoes another: diamonds
// from a block of diamonds, which is made of diamonds).
func codexReverses(step recipeStep, key string) bool {
	for _, in := range step.ins {
		ik := itemKey(in.stack.Item())
		for _, cr := range crafterRecipes() {
			if cr.outKey != ik || len(cr.reqs) != 1 {
				continue
			}
			if st, ok := cr.reqs[0].need.(item.Stack); ok && itemKey(st.Item()) == key {
				return true
			}
		}
	}
	return false
}
