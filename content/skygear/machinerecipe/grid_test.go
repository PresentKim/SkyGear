package machinerecipe

import (
	"sync"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/world"
)

var registerGridRecipes = sync.OnceFunc(func() {
	recipe.Register(recipe.NewShaped([]recipe.Item{item.NewStack(block.Obsidian{}, 1)}, item.NewStack(block.Glowstone{}, 1), recipe.NewShape(1, 1), "crafting_table"))
	recipe.Register(recipe.NewShaped([]recipe.Item{item.NewStack(block.Netherrack{}, 1), item.NewStack(block.SoulSand{}, 1)}, item.NewStack(block.Magma{}, 1), recipe.NewShape(1, 2), "crafting_table"))
	recipe.Register(recipe.NewShapeless([]recipe.Item{item.NewStack(block.Clay{}, 1), item.NewStack(block.Gravel{}, 1)}, item.NewStack(block.Mud{}, 1), "crafting_table"))
	ResetCrafters()
})

func grid(cells ...world.Item) []item.Stack {
	out := make([]item.Stack, 9)
	for i, c := range cells {
		if c != nil {
			out[i] = item.NewStack(c, 1)
		}
	}
	return out
}

func makes(t *testing.T, cells []item.Stack) world.Item {
	t.Helper()
	cr, ok := MatchGrid(cells)
	if !ok {
		return nil
	}
	return cr.Out.Item()
}

func TestMatchGridShapedAnywhere(t *testing.T) {
	registerGridRecipes()
	for _, at := range []int{0, 4, 8} {
		cells := make([]item.Stack, 9)
		cells[at] = item.NewStack(block.Obsidian{}, 1)
		if _, ok := makes(t, cells).(block.Glowstone); !ok {
			t.Errorf("obsidian at %d does not make glowstone", at)
		}
	}
}

func TestMatchGridKeepsTheShape(t *testing.T) {
	registerGridRecipes()
	if _, ok := makes(t, grid(nil, block.Netherrack{}, nil, nil, block.SoulSand{})).(block.Magma); !ok {
		t.Error("the column does not make magma")
	}
	if got := makes(t, grid(block.SoulSand{}, nil, nil, block.Netherrack{})); got != nil {
		t.Errorf("the column upside down makes %v", got)
	}
	if got := makes(t, grid(block.Netherrack{}, block.SoulSand{})); got != nil {
		t.Errorf("the pair side by side makes %v", got)
	}
}

func TestMatchGridShapelessAnyOrder(t *testing.T) {
	registerGridRecipes()
	if _, ok := makes(t, grid(nil, block.Gravel{}, nil, nil, nil, nil, nil, block.Clay{})).(block.Mud); !ok {
		t.Error("clay and gravel apart do not make mud")
	}
	if got := makes(t, grid(block.Clay{}, block.Clay{})); got != nil {
		t.Errorf("clay twice makes %v", got)
	}
}
