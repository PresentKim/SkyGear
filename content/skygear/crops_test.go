package skygear

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/presentkim/skygear/content/skygear/farm"

	"github.com/presentkim/skygear/content/skygear/island"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

var cropPos = cube.Pos{4, island.IslandY, 4}

// plantAt puts a crop on farmland and records it as planted now.
func plantAt(s *session, pos cube.Pos, b world.Block) {
	s.World.Do(func(tx *world.Tx) {
		tx.SetBlock(pos.Side(cube.FaceDown), block.Farmland{Hydration: 7}, nil)
		tx.SetBlock(pos, b, nil)
		kind, _, _ := farm.CropOf(b)
		farm.PlantCrop(s.Session, pos, kind)
	})
}

func stageOf(t *testing.T, s *session, pos cube.Pos) int {
	t.Helper()
	stage := -1
	s.World.Do(func(tx *world.Tx) {
		if _, g, ok := farm.CropOf(tx.Block(pos)); ok {
			stage = g
		}
	})
	return stage
}

// advance moves instance time on and runs the per-second update.
func advance(s *session, d time.Duration) {
	s.Clock.Advance(d)
	s.World.Do(func(tx *world.Tx) { farm.GrowCrops(s.Session, tx) })
}

func TestCropsGrowOnInstanceTime(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	plantAt(s, cropPos, block.WheatSeeds{})
	// One stage, rounded up to the nanosecond as the queue schedules it.
	step := (farm.GrowTime(s.Session, "wheat") + farm.MaxCropStage - 1) / farm.MaxCropStage
	advance(s, step-time.Second)
	if g := stageOf(t, s, cropPos); g != 0 {
		t.Fatalf("stage %d before the first stage time", g)
	}
	advance(s, time.Second)
	if g := stageOf(t, s, cropPos); g != 1 {
		t.Fatalf("stage %d after one stage time", g)
	}
	// Nothing is due until the next stage: the queue head is in the future.
	now, _, _ := s.Clock.Now()
	if farm.LocalOf(s.Session).CropQueue.Len() != 1 || farm.LocalOf(s.Session).CropQueue[0].At <= now {
		t.Fatalf("queue %v at %v", farm.LocalOf(s.Session).CropQueue, now)
	}
	// A long pause (inactive instance) does not grow it; instance time does.
	advance(s, farm.GrowTime(s.Session, "wheat"))
	if g := stageOf(t, s, cropPos); g != farm.MaxCropStage {
		t.Fatalf("stage %d after the full growth time", g)
	}
	if farm.LocalOf(s.Session).CropQueue.Len() != 0 {
		t.Fatal("a grown crop is still queued")
	}
}

func TestRandomTicksDoNotGrowCrops(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	plantAt(s, cropPos, block.WheatSeeds{})
	s.World.Do(func(tx *world.Tx) {
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
	s.World.Do(func(tx *world.Tx) { tx.SetBlock(cropPos, farm.WithGrowth(block.Carrot{}, 5), nil) })
	step := (farm.GrowTime(s.Session, "carrot") + farm.MaxCropStage - 1) / farm.MaxCropStage
	advance(s, step)
	if g := stageOf(t, s, cropPos); g != 5 {
		t.Fatalf("stage %d after bone meal", g)
	}
	advance(s, step+time.Second)
	if g := stageOf(t, s, cropPos); g != 6 {
		t.Fatalf("stage %d one stage after bone meal", g)
	}
	// A crop that was broken is forgotten.
	s.World.Do(func(tx *world.Tx) { tx.SetBlock(cropPos, nil, nil) })
	advance(s, farm.GrowTime(s.Session, "carrot"))
	if len(farm.StateOf(s.Session).Crops) != 0 {
		t.Fatalf("records left: %v", farm.StateOf(s.Session).Crops)
	}
}

func TestStemGrowsFruit(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	s.World.Do(func(tx *world.Tx) {
		for _, d := range cube.Directions() {
			tx.SetBlock(cropPos.Side(d.Face()).Side(cube.FaceDown), block.Dirt{}, nil)
		}
	})
	plantAt(s, cropPos, block.MelonSeeds{})
	advance(s, farm.GrowTime(s.Session, "melon"))
	advance(s, farm.StemFruitTime(s.Session))
	found := false
	s.World.Do(func(tx *world.Tx) {
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

func TestCropsLoadFromState(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	plantAt(s, cropPos, block.Potato{})
	farm.LocalOf(s.Session).CropQueue = nil
	advance(s, farm.GrowTime(s.Session, "potato"))
	if g := stageOf(t, s, cropPos); g != 0 {
		t.Fatal("grew without being queued")
	}
	farm.LoadCrops(s.Session)
	advance(s, 0)
	if g := stageOf(t, s, cropPos); g != farm.MaxCropStage {
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
		if r := farm.StateOf(s.Session).Crops[farm.CropKey(cropPos)]; ctx.Cancelled() || r == nil || r.Kind != "beetroot" {
			t.Fatalf("record %+v cancelled %v", r, ctx.Cancelled())
		}
	})
}
