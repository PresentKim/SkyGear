package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/internal/games/skygear/sgitem"
)

// TestEquipToolFor: starting to break a block with a tool that does not fit
// swaps the first fitting tool in the inventory into the hand; a fitting
// tool, a hammer on what it crushes and a block no tool fits are left alone.
func TestEquipToolFor(t *testing.T) {
	s := testSession(t)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		inv := p.Inventory()
		dirt := item.NewStack(block.Dirt{}, 5)
		wood, stone := item.NewStack(item.Pickaxe{Tier: item.ToolTierWood}, 1), item.NewStack(item.Pickaxe{Tier: item.ToolTierStone}, 1)
		axe := item.NewStack(item.Axe{Tier: item.ToolTierWood}, 1)
		hammer := item.NewStack(sgitem.Hammer{Stone: true}, 1)
		_ = inv.SetItem(3, hammer)
		_ = inv.SetItem(4, axe)
		_ = inv.SetItem(12, wood)
		_ = inv.SetItem(20, stone)
		p.SetHeldItems(dirt, item.Stack{})

		equipToolFor(p, block.Planks{})
		if held, _ := p.HeldItems(); !held.Equal(axe) {
			t.Fatalf("holding %v on planks, want the axe", held)
		}
		if got, _ := inv.Item(4); !got.Equal(dirt) {
			t.Fatalf("slot 4 has %v, want the dirt that was held", got)
		}

		// Iron ore needs a stone pickaxe: the wooden one does not fit, and
		// the hammer is never picked.
		equipToolFor(p, block.IronOre{})
		if held, _ := p.HeldItems(); !held.Equal(stone) {
			t.Fatalf("holding %v on iron ore, want the stone pickaxe", held)
		}
		equipToolFor(p, block.Stone{})
		if held, _ := p.HeldItems(); !held.Equal(stone) {
			t.Fatalf("a fitting pickaxe was swapped for %v", held)
		}

		p.SetHeldItems(hammer, item.Stack{})
		equipToolFor(p, block.Gravel{})
		if held, _ := p.HeldItems(); !sgitem.IsHammer(held) {
			t.Fatalf("the hammer on gravel was swapped for %v", held)
		}
		p.SetHeldItems(dirt, item.Stack{})
		equipToolFor(p, block.Air{})
		if held, _ := p.HeldItems(); !held.Equal(dirt) {
			t.Fatalf("swapped to %v for air", held)
		}
	})
}
