package machinerecipe

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
)

// MatchGrid finds the crafting-table recipe a 3×3 crafting grid (cells row
// by row) makes, one item per cell like the vanilla grid: a shaped recipe
// in its shape anywhere in the grid, mirrored left to right when it allows
// it (the rule of Dragonfly's patched shapedArrangementOK), or a shapeless
// one whose ingredients the cells hold one each.
func MatchGrid(cells []item.Stack) (*Crafter, bool) {
	for _, cr := range Crafters() {
		switch r := cr.Recipe.(type) {
		case recipe.Shaped:
			if shapedFits(cells, 3, r) {
				return cr, true
			}
		case recipe.Shapeless:
			if shapelessFits(cells, r.Input()) {
				return cr, true
			}
		}
	}
	return nil, false
}

func matchCell(has item.Stack, want recipe.Item) bool {
	return MatchReq(has, CrafterReq{Need: want})
}

func shapedFits(cells []item.Stack, cols int, r recipe.Shaped) bool {
	rows := len(cells) / cols
	minR, maxR, minC, maxC := rows, -1, cols, -1
	for i, c := range cells {
		if c.Empty() {
			continue
		}
		minR, maxR = min(minR, i/cols), max(maxR, i/cols)
		minC, maxC = min(minC, i%cols), max(maxC, i%cols)
	}
	if maxR < 0 {
		return false
	}
	w, h := maxC-minC+1, maxR-minR+1
	if w != r.Shape().Width() || h != r.Shape().Height() {
		return false
	}
	input := r.Input()
	match := func(mirror bool) bool {
		for y := range h {
			for x := range w {
				cx := x
				if mirror {
					cx = w - 1 - x
				}
				has, want := cells[(minR+y)*cols+minC+cx], input[y*w+x]
				empty := want == nil || want.Empty()
				if has.Empty() != empty || (!empty && !matchCell(has, want)) {
					return false
				}
			}
		}
		return true
	}
	return match(false) || (!r.NoMirror() && match(true))
}

// shapelessFits reports whether the non-empty cells hold the ingredients one
// each, in any order.
func shapelessFits(cells []item.Stack, input []recipe.Item) bool {
	var want []recipe.Item
	for _, in := range input {
		if in == nil || in.Empty() {
			continue
		}
		for range max(1, in.Count()) {
			want = append(want, in)
		}
	}
	var has []item.Stack
	for _, c := range cells {
		if !c.Empty() {
			has = append(has, c)
		}
	}
	if len(has) != len(want) {
		return false
	}
	used := make([]bool, len(want))
	var assign func(i int) bool
	assign = func(i int) bool {
		if i == len(has) {
			return true
		}
		for j, w := range want {
			if !used[j] && matchCell(has[i], w) {
				used[j] = true
				if assign(i + 1) {
					return true
				}
				used[j] = false
			}
		}
		return false
	}
	return assign(0)
}
