package session

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
)

// TestShapedArrangement: a shaped recipe crafts only in its arrangement,
// anywhere in the grid, mirrored only when allowed (skygear patch).
func TestShapedArrangement(t *testing.T) {
	c, s := item.NewStack(block.Cobblestone{}, 1), item.NewStack(item.Stick{}, 1)
	e := item.Stack{}
	in := []recipe.Item{e, c, e, e, s, c, s, e, e}
	rec := recipe.NewShaped(in, item.NewStack(item.Stick{}, 1), recipe.NewShape(3, 3), "crafting_table").WithoutMirror()
	grid := []item.Stack{e, c, e, e, s, c, s, e, e}
	if !shapedArrangementOK(grid, 3, rec) {
		t.Error("the arrangement itself does not craft")
	}
	mirrored := []item.Stack{e, c, e, c, s, e, e, e, s}
	if shapedArrangementOK(mirrored, 3, rec) {
		t.Error("the mirrored arrangement crafts a recipe without mirror")
	}
	if !shapedArrangementOK(mirrored, 3, recipe.NewShaped(in, item.NewStack(item.Stick{}, 1), recipe.NewShape(3, 3), "crafting_table")) {
		t.Error("the mirrored arrangement does not craft a recipe with mirror")
	}
	shapeless := []item.Stack{c, c, s, s, e, e, e, e, e}
	if shapedArrangementOK(shapeless, 3, rec) {
		t.Error("the right items in another shape craft the recipe")
	}
	row := recipe.NewShaped([]recipe.Item{c, c}, item.NewStack(item.Stick{}, 1), recipe.NewShape(2, 1), "crafting_table")
	if !shapedArrangementOK([]item.Stack{e, e, e, e, c, c, e, e, e}, 3, row) || !shapedArrangementOK([]item.Stack{c, c, e, e}, 2, row) {
		t.Error("a row does not craft anywhere in the grid")
	}
}
