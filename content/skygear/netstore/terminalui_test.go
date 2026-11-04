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
	machinerecipe.ResetCrafters()
})

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
			cur = craftClick(n, grid, cur, false, nil)
		}
		if _, ok := cur.Item().(block.Glowstone); !ok || cur.Count() != 8 {
			t.Errorf("cursor %v, want 8 glowstone", cur)
		}
		if !grid[4].Empty() || n.Root.Count(KeyOf(obsidian)) != 0 {
			t.Errorf("grid %v, network %d", grid[4], n.Root.Count(KeyOf(obsidian)))
		}
		if got := craftClick(n, grid, cur, false, nil); got.Count() != 8 {
			t.Errorf("an empty grid made something: %v", got)
		}
	})
}
