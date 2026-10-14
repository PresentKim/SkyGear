package dragonfactory

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

func TestHammerDropsCrushedBlock(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	pos := cube.Pos{0, IslandY, 1}
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		mine := func(b world.Block, held item.Stack) []item.Stack {
			tx.SetBlock(pos, b, nil)
			p.SetHeldItems(held, item.Stack{})
			drops := []item.Stack{item.NewStack(b.(world.Item), 1)}
			(&handler{s: s}).HandleBlockBreak(player.NewEventContext(tx, p), pos, &drops, new(int))
			return drops
		}
		hammer := item.NewStack(Hammer{}, 1)
		if d := mine(block.Cobblestone{}, hammer); len(d) != 1 || itemKey(d[0].Item()) != itemKey(block.Gravel{}) {
			t.Fatalf("cobblestone with a hammer drops %v", d)
		}
		if d := mine(block.Gravel{}, hammer); len(d) != 1 || itemKey(d[0].Item()) != itemKey(block.Sand{}) {
			t.Fatalf("gravel with a hammer drops %v", d)
		}
		if d := mine(block.Cobblestone{}, item.NewStack(item.Pickaxe{Tier: item.ToolTierStone}, 1)); itemKey(d[0].Item()) != itemKey(block.Cobblestone{}) {
			t.Fatalf("cobblestone with a pickaxe drops %v", d)
		}
	})
	// Clients get the hammer's mining speeds.
	if (Hammer{}).DestroySpeeds()["minecraft:cobblestone"] == 0 {
		t.Fatal("no client speed for cobblestone")
	}
	if block.BreakDuration(block.Cobblestone{}, item.NewStack(Hammer{}, 1), block.BreakContext{}) >= block.BreakDuration(block.Cobblestone{}, item.Stack{}, block.BreakContext{}) {
		t.Fatal("the hammer does not mine cobblestone faster than a hand")
	}
}

// TestDustFromCrushedSand: sand crushes into dust (sky progression); dirt
// and sand sift out the farming and kelp drops.
func TestDustFromCrushedSand(t *testing.T) {
	if out, ok := crushed(block.Sand{}); !ok || itemKey(out) != itemKey(Dust{}) {
		t.Fatalf("crushed sand = %v", out)
	}
	has := func(in world.Item, want world.Item) bool {
		for _, d := range sieveTables["string"][itemKey(in)] {
			if itemKey(d.Item.Item()) == itemKey(want) {
				return true
			}
		}
		return false
	}
	if !has(block.Dirt{}, block.PumpkinSeeds{}) || !has(block.Dirt{}, block.MelonSeeds{}) || !has(block.Sand{}, block.Kelp{}) {
		t.Fatal("sieve drops for farming and kelp missing")
	}
}
