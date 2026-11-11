package farm_test

import (
	"math/rand/v2"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core/coretest"
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

// TestCrookIsQuickOnLeaves: the crook is the leaves' tool and wears a point a
// leaf; its client speeds list every kind of leaves and the infested ones.
func TestCrookIsQuickOnLeaves(t *testing.T) {
	for _, tier := range sgitem.CrookTiers {
		c := sgitem.Crook{Tier: tier.Name}
		if !(block.Leaves{Type: block.OakLeaves()}).BreakInfo().Effective(c) {
			t.Errorf("%s crook is not effective on leaves", tier.Name)
		}
		if !(farm.SilkwormLeaves{Cocoon: farm.CocoonMax}).BreakInfo().Effective(c) {
			t.Errorf("%s crook is not effective on infested leaves", tier.Name)
		}
		if c.BaseMiningEfficiency(farm.SilkwormLeaves{Cocoon: farm.CocoonMax}) <= 1 || c.BaseMiningEfficiency(block.Stone{}) != 1 {
			t.Errorf("%s crook: mining speeds", tier.Name)
		}
		speeds := c.DestroySpeeds()
		for _, name := range []string{"minecraft:oak_leaves", "minecraft:jungle_leaves", "skygear:infesting_leaves"} {
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
	s := coretest.Session(t)
	coretest.LoadAround(s)
	pos := cube.Pos{0, island.IslandY + 3, 1}
	h := coretest.SpawnPlayer(s)
	coretest.WithPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		mine := func(held item.Stack) []item.Stack {
			l := block.Leaves{Type: block.OakLeaves()}
			tx.SetBlock(pos, l, nil)
			p.SetHeldItems(held, item.Stack{})
			drops := l.BreakInfo().Drops(item.ToolNone{}, nil)
			s.Handler(p).HandleBlockBreak(player.NewEventContext(tx, p), pos, &drops, new(int))
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
	s := coretest.Session(t)
	coretest.LoadAround(s)
	coretest.WithPlayer(t, s, coretest.SpawnPlayer(s), func(p *player.Player, tx *world.Tx) {
		p.SetFood(10)
		farm.CookedSilkworm{}.Consume(tx, p)
		if got := p.Food(); got != 10+farm.CookedSilkwormFood {
			t.Errorf("food %d after a cooked silkworm, want %d", got, 10+farm.CookedSilkwormFood)
		}
	})
}
