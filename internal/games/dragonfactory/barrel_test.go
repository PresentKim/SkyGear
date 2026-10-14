package dragonfactory

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// TestBarrelStages: six fill stages at fixed counts, water on top.
func TestBarrelStages(t *testing.T) {
	want := map[int]int{0: 0, 5: 0, 6: 1, 10: 1, 11: 2, 16: 3, 21: 3, 22: 4, 27: 5, 32: 6}
	for n, stage := range want {
		if got := barrelStage(&BarrelRecord{Count: n}); got != stage {
			t.Errorf("%d items: stage %d, want %d", n, got, stage)
		}
	}
	if barrelStage(&BarrelRecord{Count: 32, Water: true}) != barrelWater || barrelStage(nil) != 0 {
		t.Error("water or no record shows the wrong stage")
	}
}

// TestBarrelMakesWater: 32 plant items fill the barrel, after the delay the
// contents are water, and a bucket takes it and empties the barrel.
func TestBarrelMakesWater(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		pos := cube.Pos{4, IslandY, 4}
		barrel := Barrel{Wood: "oak"}
		tx.SetBlock(pos, barrel, nil)
		leaves := item.NewStack(block.Leaves{Type: block.OakLeaves()}, 1)
		var ctx item.UseContext
		if s.useBarrel(p, tx, pos, barrel, item.NewStack(item.Stick{}, 1), &ctx) {
			t.Fatal("a stick was put in a barrel")
		}
		for i := 1; i <= matterCapacity; i++ {
			cur := tx.Block(pos).(Barrel)
			if !s.useBarrel(p, tx, pos, cur, leaves, &ctx) {
				t.Fatalf("item %d was not taken", i)
			}
			if got := tx.Block(pos).(Barrel).Stage; got != i*barrelFillStages/matterCapacity {
				t.Fatalf("after %d items the stage is %d", i, got)
			}
		}
		if s.useBarrel(p, tx, pos, tx.Block(pos).(Barrel), leaves, &ctx) != true || s.barrelAt(pos).Count != matterCapacity {
			t.Fatal("a full barrel took another item")
		}
		bucket := item.NewStack(item.Bucket{}, 1)
		s.useBarrel(p, tx, pos, tx.Block(pos).(Barrel), bucket, &ctx)
		if s.barrelAt(pos).Water {
			t.Fatal("the water was ready at once")
		}
		// Time passes: the clock runs on, the second tick makes water.
		s.barrelAt(pos).ReadyAt = 0
		s.tickBarrels(tx)
		if got := tx.Block(pos).(Barrel).Stage; got != barrelWater {
			t.Fatalf("stage %d after the delay, want water", got)
		}
		// A non-bucket does not take the water; an empty bucket does.
		if !s.useBarrel(p, tx, pos, tx.Block(pos).(Barrel), leaves, &ctx) || s.barrelAt(pos) == nil {
			t.Fatal("leaves took the water")
		}
		if !s.useBarrel(p, tx, pos, tx.Block(pos).(Barrel), bucket, &ctx) {
			t.Fatal("the bucket did not answer")
		}
		if s.barrelAt(pos) != nil || tx.Block(pos).(Barrel).Stage != 0 {
			t.Error("the barrel is not empty after the water was taken")
		}
		water := item.NewStack(item.Bucket{Content: item.LiquidBucketContent(block.Water{Depth: 8, Still: true})}, 1)
		if !p.Inventory().ContainsItem(water) {
			t.Error("no water bucket was given")
		}
	})
}

// TestBarrelRecordIsForgotten: a barrel that is gone loses its record.
func TestBarrelRecordIsForgotten(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	s.world.Do(func(tx *world.Tx) {
		pos := cube.Pos{5, IslandY, 5}
		tx.SetBlock(pos, Barrel{Wood: "oak"}, nil)
		s.state.Barrels = map[string]*BarrelRecord{cropKey(pos): {Count: 3}}
		tx.SetBlock(pos, block.Air{}, nil)
		s.tickBarrels(tx)
		if len(s.state.Barrels) != 0 {
			t.Error("the record of a missing barrel stayed")
		}
	})
}
