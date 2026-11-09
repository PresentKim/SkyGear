package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

type levels = [sgitem.CompactMaxLevel + 1]int

func worth(n levels) int {
	t := 0
	for l, c := range n {
		t += c * sgitem.CompactWorth(l)
	}
	return t
}

func TestPlanCompactPayment(t *testing.T) {
	for _, c := range []struct {
		name         string
		have         levels
		cost         int
		take, change levels
		ok           bool
	}{
		{"loose only", levels{100}, 64, levels{64}, levels{}, true},
		{"loose first, then level 1", levels{10, 8}, 64, levels{10, 6}, levels{}, true},
		{"breaks one level 1 block", levels{10, 8}, 66, levels{10, 7}, levels{7}, true},
		{"only a level 2 block", levels{0, 0, 1}, 64, levels{0, 0, 1}, levels{8, 1}, true},
		{"lowest level broken first", levels{0, 0, 1, 1}, 82, levels{0, 0, 1, 1}, levels{8, 8, 8}, true},
		{"level 4 for a small cost", levels{0, 0, 0, 0, 1}, 1, levels{0, 0, 0, 0, 1}, levels{8, 8, 8, 8}, true},
		{"loose used, change back as loose", levels{5, 1, 1}, 90, levels{5, 1, 1}, levels{5}, true},
		{"too little", levels{8, 8}, 81, levels{}, levels{}, false},
		{"exactly enough", levels{8, 8}, 80, levels{8, 8}, levels{}, true},
		{"nothing to pay", levels{}, 0, levels{}, levels{}, true},
	} {
		p, ok := sgitem.PlanCompactPayment(c.have, c.cost)
		if ok != c.ok || p.Take != c.take || p.Change != c.change {
			t.Errorf("%s: plan(%v, %d) = take %v change %v ok %v, want take %v change %v ok %v",
				c.name, c.have, c.cost, p.Take, p.Change, ok, c.take, c.change, c.ok)
			continue
		}
		if ok && worth(p.Take)-worth(p.Change) != c.cost {
			t.Errorf("%s: paid %d, want %d", c.name, worth(p.Take)-worth(p.Change), c.cost)
		}
	}
}

// TestPlanCompactPaymentAlwaysExact checks every small holding against every
// cost: what is paid is exactly the cost, nothing taken is more than held,
// change is less than one of the block broken.
func TestPlanCompactPaymentAlwaysExact(t *testing.T) {
	for a := 0; a <= 20; a += 4 {
		for b := 0; b <= 3; b++ {
			for c := 0; c <= 2; c++ {
				for d := 0; d <= 1; d++ {
					have := levels{a, b, c, d}
					for cost := 0; cost <= worth(have)+1; cost++ {
						p, ok := sgitem.PlanCompactPayment(have, cost)
						if ok != (cost <= worth(have)) {
							t.Fatalf("plan(%v, %d) ok = %v", have, cost, ok)
						}
						if !ok {
							continue
						}
						if worth(p.Take)-worth(p.Change) != cost {
							t.Fatalf("plan(%v, %d) pays %d", have, cost, worth(p.Take)-worth(p.Change))
						}
						for l := range have {
							if p.Take[l] > have[l] || p.Take[l] < 0 || p.Change[l] < 0 || p.Change[l] > 8 {
								t.Fatalf("plan(%v, %d) = %+v", have, cost, p)
							}
						}
						// Loose cobblestone goes first.
						if p.Take[0] < min(a, cost) {
							t.Fatalf("plan(%v, %d) takes %d loose", have, cost, p.Take[0])
						}
					}
				}
			}
		}
	}
}

func compact(level int) world.Item {
	if level == 0 {
		return block.Cobblestone{}
	}
	return sgitem.CompactBlock{Kind: "cobblestone", Level: level}
}

func TestPayTakesCompressedCobblestone(t *testing.T) {
	inv := inventory.New(36, nil)
	_, _ = inv.AddItem(item.NewStack(block.Cobblestone{}, 10))
	_, _ = inv.AddItem(item.NewStack(compact(2), 1))
	if total, compressed := island.Worth(inv, block.Cobblestone{}); total != 91 || compressed != 81 {
		t.Fatalf("worth = %d (%d compressed), want 91 (81)", total, compressed)
	}
	left, ok := island.Pay(inv, block.Cobblestone{}, 64)
	if !ok || len(left) != 0 {
		t.Fatalf("pay = %v, %v", left, ok)
	}
	// 10 loose + the level 2 block (81) for 64: 27 back, as 3 level 1 blocks.
	if n := core.CountPlain(inv, compact(2)); n != 0 {
		t.Errorf("%d level 2 left", n)
	}
	if n := core.CountPlain(inv, compact(1)); n != 3 {
		t.Errorf("%d level 1, want 3", n)
	}
	if n := core.CountPlain(inv, block.Cobblestone{}); n != 0 {
		t.Errorf("%d loose, want 0", n)
	}
	if _, ok := island.Pay(inv, block.Cobblestone{}, 28); ok {
		t.Error("paid 28 with 27")
	}
	if n := core.CountPlain(inv, compact(1)); n != 3 {
		t.Errorf("a failed payment took something: %d level 1", n)
	}
}

func TestPayChangeThatDoesNotFit(t *testing.T) {
	inv := inventory.New(2, nil)
	_ = inv.SetItem(0, item.NewStack(compact(2), 1))
	_ = inv.SetItem(1, item.NewStack(item.Apple{}, 1))
	// 81 for 1: 8 level 1 blocks and 8 cobblestone back, one free slot.
	left, ok := island.Pay(inv, block.Cobblestone{}, 1)
	if !ok {
		t.Fatal("not paid")
	}
	if n := core.CountPlain(inv, compact(1)); n != 8 {
		t.Errorf("%d level 1 in the inventory, want 8 (the higher level first)", n)
	}
	if len(left) != 1 || left[0].Count() != 8 || !left[0].Comparable(item.NewStack(block.Cobblestone{}, 1)) {
		t.Errorf("left = %v, want 8 cobblestone", left)
	}
}

func TestBuyChunkWithCompressedCobblestone(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		// Fill every slot but one, so the change cannot all fit.
		_ = p.Inventory().SetItem(0, item.NewStack(compact(3), 1))
		for i := 1; i < p.Inventory().Size(); i++ {
			_ = p.Inventory().SetItem(i, item.NewStack(item.Apple{}, 64))
		}
		cost := island.ChunkCost(len(island.StateOf(s.Session).Chunks))
		island.Buy(s.Session, p, tx, island.ChunkPos{1, 0})
		if !island.Owns(s.Session, island.ChunkPos{1, 0}) {
			t.Fatal("chunk not bought")
		}
		have, _ := island.Worth(p.Inventory(), block.Cobblestone{})
		lost := 0
		for _, l := range island.StateOf(s.Session).Lost {
			st := l.Decoded()
			name, _ := st.Item().EncodeItem()
			lost += l.Count * map[string]int{"minecraft:cobblestone": 1, "skygear:compressed_cobblestone_1": 9, "skygear:compressed_cobblestone_2": 81}[name]
		}
		if have+lost != 729-cost.Count {
			t.Fatalf("kept %d + lost-and-found %d, want %d", have, lost, 729-cost.Count)
		}
		if lost == 0 {
			t.Fatal("nothing went to the lost-and-found box")
		}
	})
}

// TestCompactRecipesAndResearch: each level crafts from nine of the one
// below and back, and the researches open both ways.
func TestCompactRecipesAndResearch(t *testing.T) {
	r, ok := progress.ByID("compact_cobblestone")
	if !ok {
		t.Fatal("no compact_cobblestone research")
	}
	for l := 1; l <= 4; l++ {
		name, _ := compact(l).EncodeItem()
		found := false
		for _, u := range r.Unlocks {
			found = found || u == name
		}
		if !found {
			t.Errorf("%s not opened by the research", name)
		}
	}
	soil, ok := progress.ByID("compact_soil")
	if !ok {
		t.Fatal("no compact_soil research")
	}
	for _, n := range []string{"gravel", "sand", "dirt", "skygear:compressed_dirt_2"} {
		found := false
		for _, u := range soil.Unlocks {
			found = found || u == n
		}
		if !found {
			t.Errorf("%s not opened by compact_soil", n)
		}
	}
	if b := (sgitem.CompactBlock{Kind: "sand", Level: 2}).Below(); b != (sgitem.CompactBlock{Kind: "sand", Level: 1}) {
		t.Errorf("below sand 2 = %v", b)
	}
	if b := (sgitem.CompactBlock{Kind: "gravel", Level: 1}).Below(); b != (block.Gravel{}) {
		t.Errorf("below gravel 1 = %v", b)
	}
}
