package dragonfactory

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

var cropPos = cube.Pos{4, IslandY, 4}

// plantAt puts a crop on farmland and records it as planted now.
func plantAt(s *session, pos cube.Pos, b world.Block) {
	s.world.Do(func(tx *world.Tx) {
		tx.SetBlock(pos.Side(cube.FaceDown), block.Farmland{Hydration: 7}, nil)
		tx.SetBlock(pos, b, nil)
		kind, _, _ := cropOf(b)
		s.plantCrop(pos, kind)
	})
}

func stageOf(t *testing.T, s *session, pos cube.Pos) int {
	t.Helper()
	stage := -1
	s.world.Do(func(tx *world.Tx) {
		if _, g, ok := cropOf(tx.Block(pos)); ok {
			stage = g
		}
	})
	return stage
}

// advance moves instance time on and runs the per-second update.
func advance(s *session, d time.Duration) {
	s.clock.Advance(d)
	s.world.Do(func(tx *world.Tx) { s.growCrops(tx) })
}

func TestCropsGrowOnInstanceTime(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	plantAt(s, cropPos, block.WheatSeeds{})
	// One stage, rounded up to the nanosecond as the queue schedules it.
	step := (s.growTime("wheat") + maxCropStage - 1) / maxCropStage
	advance(s, step-time.Second)
	if g := stageOf(t, s, cropPos); g != 0 {
		t.Fatalf("stage %d before the first stage time", g)
	}
	advance(s, time.Second)
	if g := stageOf(t, s, cropPos); g != 1 {
		t.Fatalf("stage %d after one stage time", g)
	}
	// Nothing is due until the next stage: the queue head is in the future.
	now, _, _ := s.clock.Now()
	if s.cropQueue.Len() != 1 || s.cropQueue[0].at <= now {
		t.Fatalf("queue %v at %v", s.cropQueue, now)
	}
	// A long pause (inactive instance) does not grow it; instance time does.
	advance(s, s.growTime("wheat"))
	if g := stageOf(t, s, cropPos); g != maxCropStage {
		t.Fatalf("stage %d after the full growth time", g)
	}
	if s.cropQueue.Len() != 0 {
		t.Fatal("a grown crop is still queued")
	}
}

func TestRandomTicksDoNotGrowCrops(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	plantAt(s, cropPos, block.WheatSeeds{})
	s.world.Do(func(tx *world.Tx) {
		r := rand.New(rand.NewPCG(1, 1))
		for range 200 {
			tx.Block(cropPos).(block.WheatSeeds).RandomTick(cropPos, tx, r)
		}
	})
	if g := stageOf(t, s, cropPos); g != 0 {
		t.Fatalf("random ticks grew wheat to %d", g)
	}
}

func TestBoneMealAndRemovedCrops(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	plantAt(s, cropPos, block.Carrot{})
	// Bone meal took it to stage 5: it continues from there.
	s.world.Do(func(tx *world.Tx) { tx.SetBlock(cropPos, withGrowth(block.Carrot{}, 5), nil) })
	step := (s.growTime("carrot") + maxCropStage - 1) / maxCropStage
	advance(s, step)
	if g := stageOf(t, s, cropPos); g != 5 {
		t.Fatalf("stage %d after bone meal", g)
	}
	advance(s, step+time.Second)
	if g := stageOf(t, s, cropPos); g != 6 {
		t.Fatalf("stage %d one stage after bone meal", g)
	}
	// A crop that was broken is forgotten.
	s.world.Do(func(tx *world.Tx) { tx.SetBlock(cropPos, nil, nil) })
	advance(s, s.growTime("carrot"))
	if len(s.state.Crops) != 0 {
		t.Fatalf("records left: %v", s.state.Crops)
	}
}

func TestStemGrowsFruit(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	s.world.Do(func(tx *world.Tx) {
		for _, d := range cube.Directions() {
			tx.SetBlock(cropPos.Side(d.Face()).Side(cube.FaceDown), block.Dirt{}, nil)
		}
	})
	plantAt(s, cropPos, block.MelonSeeds{})
	advance(s, s.growTime("melon"))
	advance(s, s.stemFruitTime())
	found := false
	s.world.Do(func(tx *world.Tx) {
		for _, d := range cube.Directions() {
			if _, ok := tx.Block(cropPos.Side(d.Face())).(block.Melon); ok {
				found = true
			}
		}
	})
	if !found {
		t.Fatal("no melon after the stem grew and the fruit time passed")
	}
}

func TestCropUpgradeKeepsProgress(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	plantAt(s, cropPos, block.WheatSeeds{})
	advance(s, s.growTime("wheat")/2)
	before := stageOf(t, s, cropPos)
	old := s.growTime("wheat")
	s.state.CropLevel = 3
	s.rescheduleCrops(old, s.growTime("wheat"))
	advance(s, 0)
	if g := stageOf(t, s, cropPos); g != before {
		t.Fatalf("stage %d after the upgrade, was %d", g, before)
	}
	advance(s, s.growTime("wheat"))
	if g := stageOf(t, s, cropPos); g != maxCropStage {
		t.Fatalf("stage %d after the faster growth time", g)
	}
}

func TestCropsLoadFromState(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	plantAt(s, cropPos, block.Potato{})
	s.cropQueue = nil
	advance(s, s.growTime("potato"))
	if g := stageOf(t, s, cropPos); g != 0 {
		t.Fatal("grew without being queued")
	}
	s.loadCrops()
	advance(s, 0)
	if g := stageOf(t, s, cropPos); g != maxCropStage {
		t.Fatalf("stage %d after loading", g)
	}
}

func TestPlantingRecordsCrop(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		ctx := player.NewEventContext(tx, p)
		(&handler{s: s}).HandleBlockPlace(ctx, cropPos, block.BeetrootSeeds{})
		if r := s.state.Crops[cropKey(cropPos)]; ctx.Cancelled() || r == nil || r.Kind != "beetroot" {
			t.Fatalf("record %+v cancelled %v", r, ctx.Cancelled())
		}
	})
}
