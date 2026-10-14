package dragonfactory

import (
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// TestStartIsland: a new save is a 3×3×3 island, bedrock at the bottom,
// with an oak tree in the middle.
func TestStartIsland(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	s.world.Do(func(tx *world.Tx) {
		(&Game{}).Init(tx, nil)
		for x := -1; x <= 1; x++ {
			for z := -1; z <= 1; z++ {
				if _, ok := tx.Block(cube.Pos{x, IslandY - 3, z}).(block.Bedrock); !ok {
					t.Fatalf("no bedrock at %d,%d", x, z)
				}
				if _, ok := tx.Block(cube.Pos{x, IslandY - 1, z}).(block.Grass); !ok && (x != 0 || z != 0) {
					t.Fatalf("no grass at %d,%d", x, z)
				}
			}
		}
		if _, ok := tx.Block(cube.Pos{0, IslandY, 0}).(block.Log); !ok {
			t.Fatalf("no tree in the middle: %T", tx.Block(cube.Pos{0, IslandY, 0}))
		}
		if _, ok := tx.Block(cube.Pos{2, IslandY - 1, 0}).(block.Air); !ok {
			t.Fatal("the island is wider than 3")
		}
	})
}

// TestSaplingCommandCooldown: /묘목 gives a sapling once per 10 minutes.
func TestSaplingCommandCooldown(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		now := time.Now()
		sapling := item.NewStack(block.Sapling{Type: block.OakSapling()}, 1)
		if err := s.giveSapling(p, tx, now); err != nil || !p.Inventory().ContainsItem(sapling) {
			t.Fatalf("first sapling: %v", err)
		}
		if err := s.giveSapling(p, tx, now.Add(9*time.Minute)); err == nil {
			t.Fatal("a second sapling within 10 minutes")
		}
		if err := s.giveSapling(p, tx, now.Add(10*time.Minute)); err != nil || !p.Inventory().ContainsItem(sapling.Grow(1)) {
			t.Fatalf("sapling after 10 minutes: %v", err)
		}
	})
}

// TestCrouchGrowsSaplings: crouching again and again next to a sapling
// grows it into a tree.
func TestCrouchGrowsSaplings(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		pos := cube.PosFromVec3(p.Position()).Add(cube.Pos{1, 0, 0})
		tx.SetBlock(pos.Side(cube.FaceDown), block.Dirt{}, nil)
		tx.SetBlock(pos, block.Sapling{Type: block.OakSapling()}, nil)
		for range 200 {
			s.crouchGrow(p, tx)
			if _, ok := tx.Block(pos).(block.Sapling); !ok {
				break
			}
		}
		if _, ok := tx.Block(pos).(block.Log); !ok {
			t.Fatalf("sapling became %T after crouching", tx.Block(pos))
		}
	})
}
