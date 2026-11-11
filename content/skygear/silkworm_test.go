package skygear

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/fairy"
	"github.com/presentkim/skygear/content/skygear/farm"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// fillCube fills the 5×5×5 around centre with b.
func fillCube(tx *world.Tx, centre cube.Pos, r int, b world.Block) {
	for dx := -r; dx <= r; dx++ {
		for dy := -r; dy <= r; dy++ {
			for dz := -r; dz <= r; dz++ {
				tx.SetBlock(centre.Add(cube.Pos{dx, dy, dz}), b, nil)
			}
		}
	}
}

// silkwormLeavesAround counts the silkworm leaves within r of centre, not
// centre itself.
func silkwormLeavesAround(tx *world.Tx, centre cube.Pos, r int) int {
	n := 0
	for dx := -r; dx <= r; dx++ {
		for dy := -r; dy <= r; dy++ {
			for dz := -r; dz <= r; dz++ {
				at := centre.Add(cube.Pos{dx, dy, dz})
				if _, ok := tx.Block(at).(farm.SilkwormLeaves); ok && at != centre {
					n++
				}
			}
		}
	}
	return n
}

// ticksFor is how many world ticks d takes.
func ticksFor(d time.Duration) int { return int(d / (time.Second / 20)) }

// TestCocoonDropsString: full silkworm leaves drop a string for every roll
// that comes up, more rolls with a crook; leaves still spinning drop nothing.
func TestCocoonDropsString(t *testing.T) {
	crook := sgitem.Crook{Tier: sgitem.CrookWood}
	cases := []struct {
		tool item.Tool
		roll float64
		want int
	}{
		{crook, 0, len(farm.StringByCrook)},
		{crook, 0.999, 2}, // the two sure rolls
		{item.ToolNone{}, 0, len(farm.StringByHand)},
		{item.ToolNone{}, 0.999, 0},
	}
	for _, c := range cases {
		if got := count[block.String](farm.CocoonDrops(c.tool, always(c.roll))); got != c.want {
			t.Errorf("%T roll %.3f: %d string, want %d", c.tool, c.roll, got, c.want)
		}
	}
	full := farm.SilkwormLeaves{Cocoon: farm.CocoonMax}
	byCrook, byHand := 0, 0
	for range 2000 {
		byCrook += count[block.String](full.BreakInfo().Drops(crook, nil))
		byHand += count[block.String](full.BreakInfo().Drops(item.ToolNone{}, nil))
	}
	if byCrook <= 2*byHand {
		t.Errorf("crook %d string, hand %d: the crook should give far more", byCrook, byHand)
	}
	for p := 0; p < farm.CocoonMax; p++ {
		if d := (farm.SilkwormLeaves{Cocoon: p}).BreakInfo().Drops(crook, nil); len(d) != 0 {
			t.Errorf("silkworm leaves at %d%% dropped %v", p*farm.CocoonStep, d)
		}
	}
}

// TestCocoonGrowsOnTime: in a running world a silkworm's leaves reach 100%
// in CocoonTime (not sooner, not by random ticks), a step at a time, and
// keep whether the leaves were placed.
func TestCocoonGrowsOnTime(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	defer func(c float64) { farm.SpreadChance = c }(farm.SpreadChance)
	farm.SpreadChance = 0
	pos := cube.Pos{6, island.IslandY + 6, 6}
	placed := pos.Add(cube.Pos{0, 3, 0})
	s.World.Do(func(tx *world.Tx) {
		tx.SetBlock(pos.Side(cube.FaceDown), block.Dirt{}, nil)
		if farm.Infest(tx, pos.Side(cube.FaceDown)) {
			t.Error("dirt was infested")
		}
		tx.SetBlock(pos.Add(cube.Pos{1, 0, 0}), fairy.OakLog, nil)
		tx.SetBlock(pos, block.Leaves{Type: block.OakLeaves()}, nil)
		tx.SetBlock(placed, block.Leaves{Type: block.OakLeaves(), Persistent: true}, nil)
		if !farm.Infest(tx, pos) || !farm.Infest(tx, placed) {
			t.Error("leaves were not infested")
		}
	})
	cocoon := func(at cube.Pos) (l farm.SilkwormLeaves) {
		s.World.Do(func(tx *world.Tx) { l, _ = tx.Block(at).(farm.SilkwormLeaves) })
		return l
	}
	tick(s, ticksFor(farm.CocoonTime()/2)+2)
	if p := cocoon(pos).CocoonPercent(); p != 50 {
		t.Errorf("half way: cocoon %d%%, want 50%%", p)
	}
	tick(s, ticksFor(farm.CocoonTime()/2))
	if l := cocoon(pos); l.Cocoon != farm.CocoonMax || l.Persistent {
		t.Errorf("after the cocoon time: %+v", l)
	}
	if l := cocoon(placed); l.Cocoon != farm.CocoonMax || !l.Persistent {
		t.Errorf("placed leaves: %+v", l)
	}
	// A random tick never moves the cocoon on, it only schedules it again.
	s.World.Do(func(tx *world.Tx) {
		l := farm.SilkwormLeaves{Cocoon: 3}
		tx.SetBlock(pos, l, nil)
		l.RandomTick(pos, tx, rand.New(rand.NewPCG(1, 1)))
		if got := tx.Block(pos).(farm.SilkwormLeaves); got.Cocoon != 3 {
			t.Errorf("random tick moved the cocoon to %d", got.Cocoon)
		}
	})
	tick(s, ticksFor(farm.CocoonStepTime)+2)
	if p := cocoon(pos).Cocoon; p != 4 {
		t.Errorf("a lost tick scheduled again: cocoon step %d, want 4", p)
	}
}

// TestSilkwormsSpreadInPlay: in a running world, one silkworm in a crown of
// leaves has reached every leaf around it when its cocoon is full (the steps
// spread at a chance, the full cocoon to all that are left), and through
// them every leaf of the crown.
func TestSilkwormsSpreadInPlay(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	centre := cube.Pos{6, island.IslandY + 8, 6}
	s.World.Do(func(tx *world.Tx) {
		fillCube(tx, centre, 2, block.Leaves{Type: block.OakLeaves(), Persistent: true})
		farm.Infest(tx, centre)
	})
	tick(s, ticksFor(farm.CocoonTime())+2)
	s.World.Do(func(tx *world.Tx) {
		if n := silkwormLeavesAround(tx, centre, 1); n != 26 {
			t.Errorf("a full cocoon reached %d of the 26 leaves around", n)
		}
	})
	tick(s, 2*ticksFor(farm.CocoonTime())+4)
	s.World.Do(func(tx *world.Tx) {
		if n := silkwormLeavesAround(tx, centre, 2); n != 124 {
			t.Errorf("after three cocoon times %d of the 124 leaves of the crown", n)
		}
	})
}

// TestSpreadStaysOnTheIsland: the worms reach only leaves, only on the
// island's chunks, and never in a world without an island session.
func TestSpreadStaysOnTheIsland(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	r := rand.New(rand.NewPCG(5, 6))
	far := cube.Pos{40, island.IslandY + 6, 6}
	if island.CanBuild(s.Session, far) {
		t.Fatal("the test position is on the island")
	}
	s.World.Do(func(tx *world.Tx) {
		stone := cube.Pos{6, island.IslandY + 12, 6}
		fillCube(tx, stone, 1, block.Stone{})
		fillCube(tx, far, 1, block.Leaves{Type: block.OakLeaves(), Persistent: true})
		for range 50 {
			if farm.TrySpread(tx, stone, r) || farm.TrySpread(tx, far, r) {
				t.Error("the worms spread into stone or off the island")
				return
			}
		}
	})
	w := world.Config{Synchronous: true}.New()
	defer func() { _ = w.Close() }()
	w.Do(func(tx *world.Tx) {
		c := cube.Pos{0, 70, 0}
		fillCube(tx, c, 1, block.Leaves{Type: block.OakLeaves(), Persistent: true})
		if farm.TrySpread(tx, c, r) {
			t.Error("spread in a world with no island")
		}
	})
}

// TestSilkwormLeavesDecayWithTheTree: with the log felled, silkworm leaves of
// a tree decay with its leaves, through to the leaves behind them; those of
// placed leaves stay, still passing the news on. Leaves keep their log
// through silkworm leaves while it stands.
func TestSilkwormLeavesDecayWithTheTree(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	type row struct{ log, inner, outer cube.Pos }
	at := func(x int) row {
		log := cube.Pos{x, island.IslandY + 5, 4}
		return row{log, log.Add(cube.Pos{1, 0, 0}), log.Add(cube.Pos{2, 0, 0})}
	}
	grown, placed := at(4), at(10)
	s.World.Do(func(tx *world.Tx) {
		for _, r := range []row{grown, placed} {
			tx.SetBlock(r.log, fairy.OakLog, nil)
			tx.SetBlock(r.outer, block.Leaves{Type: block.OakLeaves()}, nil)
		}
		tx.SetBlock(grown.inner, farm.SilkwormLeaves{Cocoon: 5}, nil)
		tx.SetBlock(placed.inner, farm.SilkwormLeaves{Cocoon: 5, Persistent: true}, nil)
	})
	tick(s, 40)
	s.World.Do(func(tx *world.Tx) {
		for _, r := range []row{grown, placed} {
			if _, ok := tx.Block(r.outer).(block.Leaves); !ok {
				t.Errorf("leaves behind silkworm leaves decayed with the log standing")
			}
			if _, ok := tx.Block(r.inner).(farm.SilkwormLeaves); !ok {
				t.Errorf("silkworm leaves decayed with the log standing")
			}
			tx.SetBlock(r.log, nil, nil)
		}
	})
	tick(s, 60)
	s.World.Do(func(tx *world.Tx) {
		if _, ok := tx.Block(grown.inner).(farm.SilkwormLeaves); ok {
			t.Error("silkworm leaves of a felled tree stayed")
		}
		if _, ok := tx.Block(grown.outer).(block.Leaves); ok {
			t.Error("leaves behind them stayed")
		}
		if _, ok := tx.Block(placed.inner).(farm.SilkwormLeaves); !ok {
			t.Error("silkworm leaves of placed leaves decayed")
		}
		if _, ok := tx.Block(placed.outer).(block.Leaves); ok {
			t.Error("leaves behind placed silkworm leaves stayed")
		}
	})
}

// TestOldInfestedLeavesUpgrade: the infesting stages of 26.11.7 ("0" to "3")
// still read (the infested leaves block of 26.11.7 is the 26.11.9 patch's,
// TestRetiredBloomsAndLeaves).
func TestOldInfestedLeavesUpgrade(t *testing.T) {
	for stage := range 4 {
		b, ok := world.BlockByName("skygear:infesting_leaves", map[string]any{"skygear:infest_stage": string(rune('0' + stage))})
		if l, isLeaves := b.(farm.SilkwormLeaves); !ok || !isLeaves || l.Cocoon != stage || l.Persistent {
			t.Errorf("26.11.7 stage %d reads as %#v", stage, b)
		}
	}
}
