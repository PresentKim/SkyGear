package skygear

import (
	"math/rand/v2"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/fairy"
	"github.com/presentkim/skygear/content/skygear/farm"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// always returns a roll that always (0) or never (just under 1) comes up.
func always(v float64) func() float64 { return func() float64 { return v } }

// count counts the items of it's kind in drops.
func count[T world.Item](drops []item.Stack) int {
	n := 0
	for _, d := range drops {
		if _, ok := d.Item().(T); ok {
			n += d.Count()
		}
	}
	return n
}

// TestCrookLeafDrops: a crook gives a sapling and a silkworm when the rolls
// come up and neither when they do not; over many leaves the wooden crook
// gives saplings about 30% of the time (more than the hand's 20%), jungle
// leaves half that.
func TestCrookLeafDrops(t *testing.T) {
	oak := block.Leaves{Type: block.OakLeaves()}
	wood := sgitem.Crook{Tier: sgitem.CrookWood}
	got := farm.CrookLeafDrops(oak, wood, always(0))
	if count[block.Sapling](got) != 1 || count[farm.Silkworm](got) != 1 {
		t.Errorf("lucky rolls: %v", got)
	}
	got = farm.CrookLeafDrops(oak, wood, always(0.999))
	if count[block.Sapling](got) != 0 || count[farm.Silkworm](got) != 0 {
		t.Errorf("unlucky rolls: %v", got)
	}
	r := rand.New(rand.NewPCG(1, 2))
	const n = 20000
	rate := func(l block.Leaves, c sgitem.Crook) (sapling, worm float64) {
		s, w := 0, 0
		for range n {
			d := farm.CrookLeafDrops(l, c, r.Float64)
			s += count[block.Sapling](d)
			w += count[farm.Silkworm](d)
			if count[item.Stick](d) > 0 {
				t.Fatal("a crook pulled a stick out of leaves")
			}
		}
		return float64(s) / n, float64(w) / n
	}
	for _, tier := range sgitem.CrookTiers {
		s, w := rate(oak, sgitem.Crook{Tier: tier.Name})
		if s < tier.Sapling-0.02 || s > tier.Sapling+0.02 {
			t.Errorf("%s crook: sapling rate %.3f, want %.2f", tier.Name, s, tier.Sapling)
		}
		if w < tier.Silkworm-0.01 || w > tier.Silkworm+0.01 {
			t.Errorf("%s crook: silkworm rate %.3f, want %.2f", tier.Name, w, tier.Silkworm)
		}
		if s <= block.LeavesSaplingChances[0] {
			t.Errorf("%s crook: sapling rate %.3f no better than the hand's %.2f", tier.Name, s, block.LeavesSaplingChances[0])
		}
	}
	if s, _ := rate(block.Leaves{Type: block.JungleLeaves()}, wood); s > 0.17 || s < 0.13 {
		t.Errorf("jungle leaves: sapling rate %.3f, want half of the oak's", s)
	}
}

// TestInfestedLeavesDropString: infested leaves drop a string for every roll
// that comes up, more rolls with a crook; infesting leaves drop nothing.
func TestInfestedLeavesDropString(t *testing.T) {
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
		if got := count[block.String](farm.InfestedDrops(c.tool, always(c.roll))); got != c.want {
			t.Errorf("%T roll %.3f: %d string, want %d", c.tool, c.roll, got, c.want)
		}
	}
	// The block itself drops through its break info.
	byCrook, byHand := 0, 0
	for range 2000 {
		byCrook += count[block.String](farm.InfestedLeaves{}.BreakInfo().Drops(crook, nil))
		byHand += count[block.String](farm.InfestedLeaves{}.BreakInfo().Drops(item.ToolNone{}, nil))
	}
	if byCrook <= 2*byHand {
		t.Errorf("crook %d string, hand %d: the crook should give far more", byCrook, byHand)
	}
	for stage := range farm.InfestStages {
		if d := (farm.InfestingLeaves{Stage: stage}).BreakInfo().Drops(crook, nil); len(d) != 0 {
			t.Errorf("infesting leaves at stage %d dropped %v", stage, d)
		}
	}
}

// TestInfestationProgresses: a silkworm's leaves go through every stage on
// their scheduled ticks and end as infested leaves; anything but leaves is
// not infested.
func TestInfestationProgresses(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	defer func(c float64) { farm.InfestSpreadChance = c }(farm.InfestSpreadChance)
	farm.InfestSpreadChance = 0 // no spreading here
	r := rand.New(rand.NewPCG(3, 4))
	s.World.Do(func(tx *world.Tx) {
		pos := cube.Pos{6, island.IslandY + 5, 6}
		tx.SetBlock(pos.Side(cube.FaceDown), block.Dirt{}, nil)
		if farm.Infest(tx, pos.Side(cube.FaceDown), r) {
			t.Error("dirt was infested")
		}
		tx.SetBlock(pos, block.Leaves{Type: block.OakLeaves()}, nil)
		if !farm.Infest(tx, pos, r) {
			t.Error("leaves were not infested")
			return
		}
		for want := 0; want < farm.InfestStages; want++ {
			l, ok := tx.Block(pos).(farm.InfestingLeaves)
			if !ok || l.Stage != want {
				t.Errorf("step %d: %#v", want, tx.Block(pos))
				return
			}
			l.ScheduledTick(pos, tx, r)
		}
		if _, ok := tx.Block(pos).(farm.InfestedLeaves); !ok {
			t.Errorf("after the last stage: %#v", tx.Block(pos))
			return
		}
		// A random tick moves a stage on as well.
		tx.SetBlock(pos, farm.InfestingLeaves{Stage: 1}, nil)
		farm.InfestingLeaves{Stage: 1}.RandomTick(pos, tx, r)
		if l, ok := tx.Block(pos).(farm.InfestingLeaves); !ok || l.Stage != 2 {
			t.Errorf("random tick: %#v", tx.Block(pos))
		}
	})
}

// TestInfestationSpreads: spreading reaches leaves in the 3×3×3 around, not
// other blocks, and not leaves off the island's chunks; the first stage does
// not spread.
func TestInfestationSpreads(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	defer func(c float64) { farm.InfestSpreadChance = c }(farm.InfestSpreadChance)
	r := rand.New(rand.NewPCG(5, 6))
	fill := func(tx *world.Tx, centre cube.Pos, b world.Block) {
		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				for dz := -1; dz <= 1; dz++ {
					tx.SetBlock(centre.Add(cube.Pos{dx, dy, dz}), b, nil)
				}
			}
		}
	}
	infested := func(tx *world.Tx, centre cube.Pos) int {
		n := 0
		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				for dz := -1; dz <= 1; dz++ {
					at := centre.Add(cube.Pos{dx, dy, dz})
					if _, ok := tx.Block(at).(farm.InfestingLeaves); ok && at != centre {
						n++
					}
				}
			}
		}
		return n
	}
	leaves := block.Leaves{Type: block.OakLeaves(), Persistent: true}
	far := cube.Pos{40, island.IslandY + 6, 6}
	s.World.Do(func(tx *world.Tx) {
		centre := cube.Pos{6, island.IslandY + 6, 6}
		fill(tx, centre, leaves)
		tx.SetBlock(centre, farm.InfestingLeaves{}, nil)
		// Stage 0 never spreads.
		farm.InfestSpreadChance = 1
		farm.AdvanceInfesting(tx, centre, farm.InfestingLeaves{}, r)
		if n := infested(tx, centre); n != 0 {
			t.Errorf("stage 0 spread to %d leaves", n)
		}
		for range 20 {
			farm.TrySpread(tx, centre, r)
		}
		if n := infested(tx, centre); n < 5 {
			t.Errorf("20 spreads into leaves all around reached only %d", n)
		}
		// Infested leaves spread on their random ticks.
		other := cube.Pos{12, island.IslandY + 6, 12}
		fill(tx, other, leaves)
		tx.SetBlock(other, farm.InfestedLeaves{}, nil)
		defer func(c float64) { farm.InfestedSpreadChance = c }(farm.InfestedSpreadChance)
		farm.InfestedSpreadChance = 1
		for range 20 {
			farm.InfestedLeaves{}.RandomTick(other, tx, r)
		}
		if infested(tx, other) == 0 {
			t.Error("infested leaves did not spread")
		}
		// Only leaves catch it.
		stone := cube.Pos{6, island.IslandY + 12, 6}
		fill(tx, stone, block.Stone{})
		for range 50 {
			if farm.TrySpread(tx, stone, r) {
				t.Error("the worms spread into stone")
				return
			}
		}
		// Not off the island's chunks.
		if island.CanBuild(s.Session, far) {
			t.Error("the test position is on the island")
			return
		}
		fill(tx, far, leaves)
		for range 50 {
			farm.TrySpread(tx, far, r)
		}
		if n := infested(tx, far); n != 0 {
			t.Errorf("the worms spread to %d leaves off the island", n)
		}
	})
}

// TestLeavesFindLogThroughInfested: leaves keep their log through infested
// and infesting leaves between them, and neither decays.
func TestLeavesFindLogThroughInfested(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	s.World.Do(func(tx *world.Tx) {
		for i, between := range []world.Block{farm.InfestedLeaves{}, farm.InfestingLeaves{Stage: 2}} {
			log := cube.Pos{4 + 4*i, island.IslandY + 5, 4}
			tx.SetBlock(log, fairy.OakLog, nil)
			tx.SetBlock(log.Add(cube.Pos{1, 0, 0}), between, nil)
			outer := log.Add(cube.Pos{2, 0, 0})
			l := block.Leaves{Type: block.OakLeaves(), ShouldUpdate: true}
			tx.SetBlock(outer, l, nil)
			l.ScheduledTick(outer, tx, nil)
			if _, ok := tx.Block(outer).(block.Leaves); !ok {
				t.Errorf("leaves behind %T decayed", between)
			}
			// With the log gone, the infested block stays.
			tx.SetBlock(log, nil, nil)
			if _, ok := tx.Block(log.Add(cube.Pos{1, 0, 0})).(block.LeavesLike); !ok {
				t.Errorf("%T went with the log", between)
			}
		}
	})
}

// TestCrookIsQuickOnLeaves: the crook is the leaves' tool and wears a point a
// leaf; its client speeds list every kind of leaves and the infested ones.
func TestCrookIsQuickOnLeaves(t *testing.T) {
	for _, tier := range sgitem.CrookTiers {
		c := sgitem.Crook{Tier: tier.Name}
		if !(block.Leaves{Type: block.OakLeaves()}).BreakInfo().Effective(c) {
			t.Errorf("%s crook is not effective on leaves", tier.Name)
		}
		if !(farm.InfestedLeaves{}).BreakInfo().Effective(c) {
			t.Errorf("%s crook is not effective on infested leaves", tier.Name)
		}
		if c.BaseMiningEfficiency(farm.InfestedLeaves{}) <= 1 || c.BaseMiningEfficiency(block.Stone{}) != 1 {
			t.Errorf("%s crook: mining speeds", tier.Name)
		}
		speeds := c.DestroySpeeds()
		for _, name := range []string{"minecraft:oak_leaves", "minecraft:jungle_leaves", "skygear:infested_leaves", "skygear:infesting_leaves"} {
			if speeds[name] == 0 {
				t.Errorf("%s crook: no client speed for %s", tier.Name, name)
			}
		}
		if d := c.DurabilityInfo(); d.BreakDurability != 1 || d.MaxDurability != tier.Tier.Durability {
			t.Errorf("%s crook: durability %+v", tier.Name, d)
		}
	}
}

// TestCrookBreaksLeavesInPlay: leaves broken by a player with a crook drop the
// crook's table (no leaves item, a silkworm now and then), the hand keeps
// the leaves' own drops, and the crook breaks leaves faster than the hand.
func TestCrookBreaksLeavesInPlay(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	pos := cube.Pos{0, island.IslandY + 3, 1}
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		mine := func(held item.Stack) []item.Stack {
			l := block.Leaves{Type: block.OakLeaves()}
			tx.SetBlock(pos, l, nil)
			p.SetHeldItems(held, item.Stack{})
			drops := l.BreakInfo().Drops(item.ToolNone{}, nil)
			(&handler{s: s}).HandleBlockBreak(player.NewEventContext(tx, p), pos, &drops, new(int))
			return drops
		}
		crook := item.NewStack(sgitem.Crook{Tier: sgitem.CrookDiamond}, 1)
		saplings, worms := 0, 0
		for range 2000 {
			d := mine(crook)
			saplings += count[block.Sapling](d)
			worms += count[farm.Silkworm](d)
		}
		if saplings < 800 || worms == 0 {
			t.Errorf("2000 leaves by diamond crook: %d saplings, %d silkworms", saplings, worms)
		}
		for range 500 {
			if count[farm.Silkworm](mine(item.Stack{})) > 0 {
				t.Fatal("leaves by hand dropped a silkworm")
			}
		}
	})
	l := block.Leaves{Type: block.OakLeaves()}
	if block.BreakDuration(l, item.NewStack(sgitem.Crook{}, 1), block.BreakContext{}) >= block.BreakDuration(l, item.Stack{}, block.BreakContext{}) {
		t.Error("the crook does not break leaves faster than a hand")
	}
}

// TestFelledTreeLeavesDecayBehindInfested: with the log gone, leaves behind a
// ring of infested leaves still hear of it and decay; the infested stay.
func TestFelledTreeLeavesDecayBehindInfested(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	s.World.Do(func(tx *world.Tx) {
		log := cube.Pos{6, island.IslandY + 5, 6}
		inner, outer := log.Add(cube.Pos{1, 0, 0}), log.Add(cube.Pos{2, 0, 0})
		tx.SetBlock(log, fairy.OakLog, nil)
		tx.SetBlock(inner, farm.InfestedLeaves{}, nil)
		tx.SetBlock(outer, block.Leaves{Type: block.OakLeaves()}, nil)
		tx.SetBlock(log, nil, nil)
		farm.InfestedLeaves{}.NeighbourUpdateTick(inner, log, tx)
		l, ok := tx.Block(outer).(block.Leaves)
		if !ok || !l.ShouldUpdate {
			t.Errorf("outer leaves not told of the felled log: %#v", tx.Block(outer))
			return
		}
		l.ScheduledTick(outer, tx, nil)
		if _, ok := tx.Block(outer).(block.Leaves); ok {
			t.Error("outer leaves did not decay")
		}
		if _, ok := tx.Block(inner).(farm.InfestedLeaves); !ok {
			t.Error("the infested leaves went")
		}
	})
}

// TestNoSpreadWithoutIsland: a world with no island session never catches it.
func TestNoSpreadWithoutIsland(t *testing.T) {
	w := world.Config{Synchronous: true}.New()
	defer func() { _ = w.Close() }()
	r := rand.New(rand.NewPCG(7, 8))
	w.Do(func(tx *world.Tx) {
		c := cube.Pos{0, 70, 0}
		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				for dz := -1; dz <= 1; dz++ {
					tx.SetBlock(c.Add(cube.Pos{dx, dy, dz}), block.Leaves{Type: block.OakLeaves(), Persistent: true}, nil)
				}
			}
		}
		for range 50 {
			if farm.TrySpread(tx, c, r) {
				t.Error("spread in a world with no island")
				return
			}
		}
	})
}

// TestCookedSilkworm: a furnace (or smoker, campfire) cooks a silkworm into a
// cooked silkworm, which fills the food bar; the raw one is not food.
func TestCookedSilkworm(t *testing.T) {
	info := farm.Silkworm{}.SmeltInfo()
	if _, ok := info.Product.Item().(farm.CookedSilkworm); !ok || !info.Food {
		t.Fatalf("silkworm smelts into %v (food %v)", info.Product, info.Food)
	}
	if _, ok := any(farm.Silkworm{}).(item.Consumable); ok {
		t.Error("the raw silkworm is food: eating it would take the leaves' right click")
	}
	s := testSession(t)
	loadAround(s)
	withPlayer(t, s, spawnPlayer(s), func(p *player.Player, tx *world.Tx) {
		p.SetFood(10)
		farm.CookedSilkworm{}.Consume(tx, p)
		if got := p.Food(); got != 10+farm.CookedSilkwormFood {
			t.Errorf("food %d after a cooked silkworm, want %d", got, 10+farm.CookedSilkwormFood)
		}
	})
}
