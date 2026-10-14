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
		// A single held bucket turns into the full one in the hand.
		s.barrelAt(pos)
		s.state.Barrels[cropKey(pos)] = &BarrelRecord{Count: matterCapacity, Water: true}
		p.SetHeldItems(bucket, item.Stack{})
		if !s.useBarrel(p, tx, pos, tx.Block(pos).(Barrel), bucket, &item.UseContext{}) {
			t.Fatal("the held bucket did not answer")
		}
		if h, _ := p.HeldItems(); h.Empty() || h.Item().(item.Bucket).Empty() {
			t.Errorf("the hand holds %v, want the water bucket", h)
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

// TestStoneBarrelMakesLava: eight cobblestone fill a stone barrel; heat
// under it turns them into lava after 2 minutes with a candle (1×), 30
// seconds over lava (4×) and 15 over a magma block (8×); no heat, no lava; a
// bucket takes the lava.
func TestStoneBarrelMakesLava(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	cobble := item.NewStack(block.Cobblestone{}, 1)
	for _, c := range []struct {
		name  string
		below world.Block
		after time.Duration
	}{
		{"candle", block.Candle{}, 2 * time.Minute},
		{"lava", block.Lava{Still: true, Depth: 8}, 30 * time.Second},
		{"magma", block.Magma{}, 15 * time.Second},
	} {
		withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
			pos := cube.Pos{4, IslandY + 3, 4}
			barrel := Barrel{Wood: stoneWood}
			tx.SetBlock(pos.Side(cube.FaceDown), c.below, nil)
			tx.SetBlock(pos, barrel, nil)
			var ctx item.UseContext
			if s.useBarrel(p, tx, pos, barrel, item.NewStack(item.Stick{}, 1), &ctx) {
				t.Fatal("a stick was put in a stone barrel")
			}
			for i := 1; i <= stoneBarrelCapacity; i++ {
				if !s.useBarrel(p, tx, pos, tx.Block(pos).(Barrel), cobble, &ctx) {
					t.Fatalf("%s: cobblestone %d was not taken", c.name, i)
				}
			}
			rec := s.barrelAt(pos)
			if rec.Count != stoneBarrelCapacity {
				t.Fatalf("%s: %d cobblestone", c.name, rec.Count)
			}
			// Just short of the time: still cobblestone.
			rec.HeatAt -= c.after - time.Second
			s.tickBarrels(tx)
			if s.barrelAt(pos).Lava {
				t.Fatalf("%s: lava a second too early", c.name)
			}
			rec.HeatAt -= time.Second
			s.tickBarrels(tx)
			if !s.barrelAt(pos).Lava || tx.Block(pos).(Barrel).Stage != barrelWater {
				t.Fatalf("%s: no lava after %s", c.name, c.after)
			}
			var bucketCtx item.UseContext
			if !s.useBarrel(p, tx, pos, tx.Block(pos).(Barrel), item.NewStack(item.Bucket{}, 1), &bucketCtx) {
				t.Fatalf("%s: the bucket did not answer", c.name)
			}
			lava := item.NewStack(item.Bucket{Content: item.LiquidBucketContent(block.Lava{Depth: 8, Still: true})}, 1)
			if !p.Inventory().ContainsItem(lava) || s.barrelAt(pos) != nil {
				t.Errorf("%s: no lava bucket or the barrel is not empty", c.name)
			}
			_ = p.Inventory().RemoveItem(lava)
		})
	}
	// Without a heat source nothing happens, however long it waits.
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		pos := cube.Pos{8, IslandY + 3, 8}
		barrel := Barrel{Wood: stoneWood}
		tx.SetBlock(pos, barrel, nil)
		var ctx item.UseContext
		for range stoneBarrelCapacity {
			s.useBarrel(p, tx, pos, tx.Block(pos).(Barrel), cobble, &ctx)
		}
		s.barrelAt(pos).HeatAt -= time.Hour
		s.tickBarrels(tx)
		if s.barrelAt(pos).Lava {
			t.Error("lava without a heat source")
		}
	})
}
