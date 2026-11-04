package netstore

import (
	"sync"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/invclick"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/machinerecipe"
)

func TestNetClick(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		n := termNet(t, tx, 0)
		cobble := item.NewStack(block.Cobblestone{}, 100)
		netInsert(n, cobble)
		k := KeyOf(cobble)
		if got := netClick(n, k, item.Stack{}, false, false); got.Count() != 64 {
			t.Errorf("left took %d, want a stack", got.Count())
		}
		if got := netClick(n, k, item.Stack{}, true, false); got.Count() != 32 {
			t.Errorf("right took %d, want half a stack", got.Count())
		}
		if rest := netClick(n, "", item.NewStack(block.Dirt{}, 5), true, false); rest.Count() != 4 {
			t.Errorf("right put in %d, want one", 5-rest.Count())
		}
		inv := invclick.Slots(make([]item.Stack, 2))
		netClick(n, k, item.Stack{}, false, true, inv)
		if inv[0].Count() != 4 {
			t.Errorf("shift moved %d into the inventory, want the 4 left", inv[0].Count())
		}
		if c := n.Root.Count(k); c != 0 {
			t.Errorf("%d cobblestone left in the network", c)
		}
	})
}

var registerTerminalRecipe = sync.OnceFunc(func() {
	recipe.Register(recipe.NewShaped([]recipe.Item{item.NewStack(block.Obsidian{}, 1)}, item.NewStack(block.Glowstone{}, 2), recipe.NewShape(1, 1), "crafting_table"))
	recipe.Register(recipe.NewShaped([]recipe.Item{item.NewStack(block.Netherrack{}, 1), nil, nil, item.NewStack(block.SoulSand{}, 1)}, item.NewStack(block.Magma{}, 1), recipe.NewShape(2, 2), "crafting_table"))
	machinerecipe.ResetCrafters()
})

func crafterOf(t *testing.T, out world.Item) *machinerecipe.Crafter {
	t.Helper()
	for _, cr := range machinerecipe.Crafters() {
		if cr.Out.Item() == out {
			return cr
		}
	}
	t.Fatalf("no recipe for %T", out)
	return nil
}

func TestLayRecipeTakesFromTheNetworkInItsShape(t *testing.T) {
	registerTerminalRecipe()
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		n := termNet(t, tx, 0)
		netInsert(n, item.NewStack(block.Netherrack{}, 3))
		netInsert(n, item.NewStack(block.SoulSand{}, 1))
		grid := invclick.Slots(make([]item.Stack, 9))
		cells, _ := gridPattern(crafterOf(t, block.Magma{}))
		for i, want := range cells {
			if want == nil {
				continue
			}
			got := takeOne(n, want)
			if got.Empty() {
				t.Fatalf("cell %d: nothing to take", i)
			}
			grid[i] = got
		}
		if _, ok := grid[0].Item().(block.Netherrack); !ok || grid[4].Empty() || !grid[1].Empty() {
			t.Errorf("grid %v", grid)
		}
		if cr, ok := gridRecipe(nil, grid); !ok || cr.Out.Item() != (block.Magma{}) {
			t.Error("the laid grid makes nothing")
		}
		if n.Root.Count(KeyOf(item.NewStack(block.Netherrack{}, 1))) != 2 {
			t.Error("the network still holds every netherrack")
		}
	})
}

func TestGridRefusesALockedRecipe(t *testing.T) {
	registerTerminalRecipe()
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		grid := invclick.Slots(make([]item.Stack, 9))
		grid[4] = item.NewStack(block.Obsidian{}, 1)
		if _, ok := gridRecipe(s, grid); ok {
			t.Error("the grid makes a recipe no research opened")
		}
		if out, ok := craftOnce(s, nil, grid); ok || grid[4].Empty() {
			t.Errorf("crafted %v from a locked recipe", out)
		}
	})
}

func TestStockMakes(t *testing.T) {
	registerTerminalRecipe()
	cr := crafterOf(t, block.Magma{})
	have := newStock(nil, invclick.Slots{item.NewStack(block.Netherrack{}, 1)})
	if have.makes(cr) {
		t.Error("one ingredient is enough")
	}
	have = newStock(nil, invclick.Slots{item.NewStack(block.Netherrack{}, 1), item.NewStack(block.SoulSand{}, 4)})
	if !have.makes(cr) {
		t.Error("both ingredients are not enough")
	}
}

func TestCraftOnceRefillsFromTheNetwork(t *testing.T) {
	registerTerminalRecipe()
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		n := termNet(t, tx, 0)
		obsidian := item.NewStack(block.Obsidian{}, 1)
		netInsert(n, obsidian.Grow(2))
		grid := invclick.Slots(make([]item.Stack, 9))
		grid[4] = obsidian
		cur := item.Stack{}
		for range 4 {
			cur = craftClick(nil, n, grid, cur, false, nil)
		}
		if _, ok := cur.Item().(block.Glowstone); !ok || cur.Count() != 8 {
			t.Errorf("cursor %v, want 8 glowstone", cur)
		}
		if !grid[4].Empty() || n.Root.Count(KeyOf(obsidian)) != 0 {
			t.Errorf("grid %v, network %d", grid[4], n.Root.Count(KeyOf(obsidian)))
		}
		if got := craftClick(nil, n, grid, cur, false, nil); got.Count() != 8 {
			t.Errorf("an empty grid made something: %v", got)
		}
	})
}
