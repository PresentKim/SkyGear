package dragonfactory

import (
	"slices"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/common/ui"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/machinerecipe"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/naming"
	"github.com/presentkim/dragonfactory/internal/instance"
)

// The raw materials of an item: the codex works out what it takes down the
// recipe chain. Favourite items: each player's
// items starred in the codex, shown above the recently viewed ones.

// favouriteMax is how many favourites a player keeps; a new one pushes the
// oldest out.
const favouriteMax = 33

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
	k := naming.ItemKey(it.Item())
	best, bestRate := recipeStep{}, 0.0
	for _, cr := range machinerecipe.Crafters() {
		if cr.OutKey != k {
			continue
		}
		step := recipeStep{out: max(1, cr.Out.Count())}
		ins := 0
		for _, req := range cr.Reqs {
			var st item.Stack
			switch need := req.Need.(type) {
			case item.Stack:
				st = need
			case recipe.ItemTag:
				st = tagStack(need)
			}
			if st.Empty() {
				step.ins = nil
				break
			}
			step.ins = append(step.ins, codexLink{stack: item.NewStack(st.Item(), 1), count: req.Count})
			ins += req.Count
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
	for _, r := range machinerecipe.Alloys {
		if naming.ItemKey(r.Out) == k {
			return recipeStep{out: max(1, r.OutCount), ins: []codexLink{
				{stack: item.NewStack(r.A, 1), count: r.CountA}, {stack: item.NewStack(r.B, 1), count: r.CountB}}}, true
		}
	}
	for i := range machinerecipe.Labs {
		if naming.ItemKey(machinerecipe.Labs[i].Out) == k {
			step := recipeStep{out: max(1, machinerecipe.Labs[i].OutCount)}
			for _, in := range machinerecipe.Labs[i].In {
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
		k := naming.ItemKey(it.Item())
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

// codexItemByKey finds the indexed item with the key.
func codexItemByKey(key string) (item.Stack, bool) {
	codexIndex(i18n.Default)
	st, ok := codexIndexByKey[key]
	return st, ok
}

// favourites are p's favourite items, oldest first.
func (s *session) favourites(p *player.Player) []item.Stack {
	var out []item.Stack
	for _, k := range s.state.Favourites[instance.AccountID(p)] {
		if st, ok := codexItemByKey(k); ok {
			out = append(out, st)
		}
	}
	return out
}

func (s *session) isFavourite(p *player.Player, it item.Stack) bool {
	return slices.Contains(s.state.Favourites[instance.AccountID(p)], naming.ItemKey(it.Item()))
}

// toggleFavourite stars it for p, or takes the star off.
func (s *session) toggleFavourite(p *player.Player, it item.Stack) {
	acc, k := instance.AccountID(p), naming.ItemKey(it.Item())
	if s.state.Favourites == nil {
		s.state.Favourites = map[string][]string{}
	}
	list := s.state.Favourites[acc]
	if i := slices.Index(list, k); i >= 0 {
		list = slices.Delete(list, i, i+1)
	} else {
		list = append(list, k)
		if len(list) > favouriteMax {
			list = list[len(list)-favouriteMax:]
		}
	}
	if len(list) == 0 {
		delete(s.state.Favourites, acc)
		return
	}
	s.state.Favourites[acc] = list
}

// favouriteSlots are p's favourites as slots that open their pages.
func (s *session) favouriteSlots(p *player.Player, lang string) []ui.ItemSlot {
	var links []codexLink
	for _, it := range s.favourites(p) {
		links = append(links, codexLink{stack: it})
	}
	return s.codexLinkSlots(lang, links)
}

// codexReverses reports whether a recipe takes an ingredient that a crafting
// recipe makes from the item key alone (the recipe undoes another: diamonds
// from a block of diamonds, which is made of diamonds).
func codexReverses(step recipeStep, key string) bool {
	for _, in := range step.ins {
		ik := naming.ItemKey(in.stack.Item())
		for _, cr := range machinerecipe.Crafters() {
			if cr.OutKey != ik || len(cr.Reqs) != 1 {
				continue
			}
			if st, ok := cr.Reqs[0].Need.(item.Stack); ok && naming.ItemKey(st.Item()) == key {
				return true
			}
		}
	}
	return false
}
