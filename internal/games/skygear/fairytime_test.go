package skygear

import (
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

// TestFairyBoneMealGrowsCropsAhead: each use grows a timed crop
// fairyBoneMealTime ahead, the queue keeps one live update per crop, and a
// grown crop takes none.
func TestFairyBoneMealGrowsCropsAhead(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	plantAt(s, cropPos, block.Carrot{})
	used := func() (ok bool) {
		s.world.Do(func(tx *world.Tx) { ok = s.fairyBoneMeal(tx, cropPos) })
		return ok
	}
	if !used() {
		t.Fatal("fairy bone meal did nothing on a young carrot")
	}
	now := s.growNow()
	if want := s.stageAt("carrot", now-fairyBoneMealTime, now); stageOf(t, s, cropPos) != want {
		t.Fatalf("stage %d after one use, want %d", stageOf(t, s, cropPos), want)
	}
	for used() {
	}
	if g := stageOf(t, s, cropPos); g != maxCropStage {
		t.Fatalf("stage %d when fairy bone meal stopped working", g)
	}
	// The updates queued before the uses are stale and fall away.
	advance(s, s.growTime("carrot"))
	if s.cropQueue.Len() != 0 {
		t.Fatalf("queue left: %v", s.cropQueue)
	}
}

// TestFairyBoneMealBringsFruitSooner: on a grown stem it moves the next fruit.
func TestFairyBoneMealBringsFruitSooner(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	plantAt(s, cropPos, block.PumpkinSeeds{})
	advance(s, s.growTime("pumpkin"))
	r := s.state.Crops[cropKey(cropPos)]
	before := r.NextFruit
	s.world.Do(func(tx *world.Tx) {
		if !s.fairyBoneMeal(tx, cropPos) {
			t.Error("fairy bone meal did nothing on a grown stem")
		}
	})
	if r.NextFruit != before-fairyBoneMealTime {
		t.Fatalf("next fruit %v, want %v", r.NextFruit, before-fairyBoneMealTime)
	}
}

// TestFairyBoneMealOnPlantsAndSaplings: a magic plant blooms sooner and a
// sapling grows into its tree.
func TestFairyBoneMealOnPlantsAndSaplings(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	pos := cropPos
	s.world.Do(func(tx *world.Tx) {
		tx.SetBlock(pos.Side(cube.FaceDown), block.Dirt{}, nil)
		tx.SetBlock(pos, MagicPlant{Kind: "flower_red"}, nil)
		s.state.Plants = map[string]*PlantRecord{cropKey(pos): {Kind: "flower_red", ReadyAt: s.growNow() + time.Minute}}
		if !s.fairyBoneMeal(tx, pos) {
			t.Error("fairy bone meal did nothing on a magic plant")
		}
		if b := tx.Block(pos).(MagicPlant); !b.Bloom {
			t.Error("the plant did not bloom a minute short of its time")
		}
		if s.fairyBoneMeal(tx, pos) {
			t.Error("fairy bone meal was spent on a bloom")
		}

		tx.SetBlock(pos.Side(cube.FaceDown), block.Grass{}, nil)
		tx.SetBlock(pos, block.Sapling{Type: block.OakSapling()}, nil)
		if !s.fairyBoneMeal(tx, pos) {
			t.Error("fairy bone meal did nothing on a sapling")
		}
		if _, still := tx.Block(pos).(block.Sapling); still {
			t.Error("the sapling did not grow")
		}
	})
}

// TestFairyHourglass: it winds an energy machine up to the cap, doubles its
// runs while it lasts, and stays on the machine's NBT; a generator takes none.
func TestFairyHourglass(t *testing.T) {
	m := newMachine(KindCrusher, "", "")
	st := m.st
	if st.hourglassReps(2) != 0 {
		t.Fatal("extra runs without an hourglass")
	}
	n := 0
	for st.windHourglass() {
		n++
	}
	if n != hourglassMaxTicks/hourglassTicks || st.boost != hourglassMaxTicks {
		t.Fatalf("%d hourglasses wound %d ticks", n, st.boost)
	}
	if st.hourglassReps(2) != 2 {
		t.Fatal("an hourglass does not double the runs")
	}
	back := Machine{Kind: KindCrusher}.DecodeNBT(m.EncodeNBT()).(Machine)
	if back.st.boost != st.boost {
		t.Fatalf("boost %d after NBT, want %d", back.st.boost, st.boost)
	}
	if hourglassLine("ko_KR", st) == "" {
		t.Fatal("no WAILA line while the hourglass runs")
	}
	if newMachine(KindGenerator, "", "").st.windHourglass() {
		t.Fatal("a generator took an hourglass")
	}
}
