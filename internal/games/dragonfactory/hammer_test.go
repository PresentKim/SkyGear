package dragonfactory

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

func TestHammerDropsCrushedBlock(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	pos := cube.Pos{0, IslandY, 1}
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		mine := func(b world.Block, held item.Stack) []item.Stack {
			tx.SetBlock(pos, b, nil)
			p.SetHeldItems(held, item.Stack{})
			drops := []item.Stack{item.NewStack(b.(world.Item), 1)}
			(&handler{s: s}).HandleBlockBreak(player.NewEventContext(tx, p), pos, &drops, new(int))
			return drops
		}
		hammer := item.NewStack(Hammer{}, 1)
		if d := mine(block.Cobblestone{}, hammer); len(d) != 1 || itemKey(d[0].Item()) != itemKey(block.Gravel{}) {
			t.Fatalf("cobblestone with a hammer drops %v", d)
		}
		if d := mine(block.Gravel{}, hammer); len(d) != 1 || itemKey(d[0].Item()) != itemKey(block.Sand{}) {
			t.Fatalf("gravel with a hammer drops %v", d)
		}
		if d := mine(block.Cobblestone{}, item.NewStack(item.Pickaxe{Tier: item.ToolTierStone}, 1)); itemKey(d[0].Item()) != itemKey(block.Cobblestone{}) {
			t.Fatalf("cobblestone with a pickaxe drops %v", d)
		}
	})
	// Clients get the hammer's mining speeds.
	if (Hammer{}).DestroySpeeds()["minecraft:cobblestone"] == 0 {
		t.Fatal("no client speed for cobblestone")
	}
	if block.BreakDuration(block.Cobblestone{}, item.NewStack(Hammer{}, 1), block.BreakContext{}) >= block.BreakDuration(block.Cobblestone{}, item.Stack{}, block.BreakContext{}) {
		t.Fatal("the hammer does not mine cobblestone faster than a hand")
	}
}

// TestDustFromCrushedSand: sand crushes into dust (sky progression); dirt
// and sand sift out the farming and kelp drops.
func TestDustFromCrushedSand(t *testing.T) {
	if out, ok := crushed(block.Sand{}); !ok || itemKey(out) != itemKey(Dust{}) {
		t.Fatalf("crushed sand = %v", out)
	}
	has := func(in world.Item, want world.Item) bool {
		for _, d := range sieveTables["string"][itemKey(in)] {
			if itemKey(d.Item.Item()) == itemKey(want) {
				return true
			}
		}
		return false
	}
	if !has(block.Dirt{}, block.PumpkinSeeds{}) || !has(block.Dirt{}, block.MelonSeeds{}) || !has(block.Sand{}, block.Kelp{}) {
		t.Fatal("sieve drops for farming and kelp missing")
	}
}

// TestCrushedStones: the hammer breaks the stones of the deep, the Nether and
// the End into their crushed blocks, and crushed netherrack into red sand
// (Ex Deorum, docs/design/16-ore-balance.md §2.1).
func TestCrushedStones(t *testing.T) {
	for in, want := range map[world.Item]world.Item{
		block.Deepslate{}: CrushedBlock{Kind: CrushedDeepslate},
		block.Deepslate{Type: block.CobbledDeepslate()}: CrushedBlock{Kind: CrushedDeepslate},
		block.Netherrack{}:                    CrushedBlock{Kind: CrushedNetherrack},
		CrushedBlock{Kind: CrushedNetherrack}: block.Sand{Red: true},
		block.Blackstone{}:                    CrushedBlock{Kind: CrushedBlackstone},
		block.EndStone{}:                      CrushedBlock{Kind: CrushedEndStone},
		block.Sand{Red: true}:                 Dust{},
	} {
		if out, ok := crushed(in.(world.Block)); !ok || itemKey(out) != itemKey(want) {
			t.Errorf("crushed %s = %v, want %s", itemKey(in), out, itemKey(want))
		}
	}
}

// TestHammerOre: the hammer breaks an ore block into two dusts of its metal.
func TestHammerOre(t *testing.T) {
	for _, b := range []world.Block{block.IronOre{}, OreBlock{Metal: MetalNickel}} {
		it := b.(world.Item)
		metal, _ := oreBlockMetal(it)
		got, ok := hammerDrop(b)
		if !ok || got.Count() != hammerDust || itemKey(got.Item()) != itemKey(MetalDust{Metal: metal}) {
			t.Errorf("hammer on %s drops %v", itemKey(it), got)
		}
	}
}

// TestHammerTiers: the stone and iron hammers last twice as long as tools of
// their material, and only the iron hammer vein mines.
func TestHammerTiers(t *testing.T) {
	stone, iron := Hammer{Stone: true}, Hammer{}
	if stone.DurabilityInfo().MaxDurability != 2*item.ToolTierStone.Durability || iron.DurabilityInfo().MaxDurability != 2*item.ToolTierIron.Durability {
		t.Fatalf("durability stone %d iron %d", stone.DurabilityInfo().MaxDurability, iron.DurabilityInfo().MaxDurability)
	}
	if isIronHammer(item.NewStack(stone, 1)) || !isIronHammer(item.NewStack(iron, 1)) || !isHammer(item.NewStack(stone, 1)) {
		t.Fatal("hammer kinds mixed up")
	}
	if (Hammer{}).BaseMiningEfficiency(block.Gravel{}) != 3 || (Hammer{}).BaseMiningEfficiency(Dust{}) != 3 {
		t.Fatal("gravel and dust speeds")
	}
}

// TestVeinMine: sneaking with the iron hammer breaks the connected blocks of
// the same kind into what the hammer makes of them, costing durability; the
// stone hammer and other kinds of blocks are left alone.
func TestVeinMine(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	base := cube.Pos{4, IslandY + 3, 4}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		// a line of five gravel and one sand touching it
		for i := 0; i < 5; i++ {
			tx.SetBlock(base.Add(cube.Pos{i, 0, 0}), block.Gravel{}, nil)
		}
		tx.SetBlock(base.Add(cube.Pos{0, 0, 1}), block.Sand{}, nil)
		if got := veinBlocks(tx, base, veinLimit, s.canBuild); len(got) != 5 {
			t.Fatalf("vein of %d blocks, want 5", len(got))
		}
		if got := veinBlocks(tx, base, 3, s.canBuild); len(got) != 3 {
			t.Fatalf("limited vein of %d blocks, want 3", len(got))
		}
		p.StartSneaking()
		p.SetHeldItems(item.NewStack(Hammer{Stone: true}, 1), item.Stack{})
		s.veinMine(p, tx, base)
		if _, ok := tx.Block(base.Add(cube.Pos{1, 0, 0})).(block.Gravel); !ok {
			t.Fatal("the stone hammer vein mined")
		}
		p.SetHeldItems(item.NewStack(Hammer{}, 1), item.Stack{})
		s.veinMine(p, tx, base)
		for i := 1; i < 5; i++ {
			if _, ok := tx.Block(base.Add(cube.Pos{i, 0, 0})).(block.Gravel); ok {
				t.Fatalf("gravel %d is still there", i)
			}
		}
		if _, ok := tx.Block(base).(block.Gravel); !ok {
			t.Fatal("the mined block is the game's to break")
		}
		if _, ok := tx.Block(base.Add(cube.Pos{0, 0, 1})).(block.Sand); !ok {
			t.Fatal("the sand was broken")
		}
		if held, _ := p.HeldItems(); held.MaxDurability()-held.Durability() != 4 {
			t.Fatalf("durability used %d, want 4", held.MaxDurability()-held.Durability())
		}
		if got := droppedKeys(tx)[itemKey(block.Sand{})]; got != 4 {
			t.Fatalf("%d sand dropped, want 4", got)
		}
	})
}

// TestVeinMineFood: vein mining costs food, a full food bar paying for
// veinLimit blocks; with little food it breaks fewer, and fractions carry
// over.
func TestVeinMineFood(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		held := item.NewStack(Hammer{}, 1)
		oneFood := 12 // 1 / veinFoodCost = 12.8
		p.SetFood(20)
		if got := s.veinBudget(p, held); got != veinLimit-1 {
			t.Fatalf("full food budget %d, want %d", got, veinLimit-1)
		}
		p.SetFood(1)
		if got := s.veinBudget(p, held); got != oneFood {
			t.Fatalf("one food budget %d, want %d", got, oneFood)
		}
		// a line of 15 cobblestone (in the owned chunk; gravel would fall) with one
		// food point: only 12 more break
		base := cube.Pos{0, IslandY + 3, 4}
		for i := 0; i < 15; i++ {
			tx.SetBlock(base.Add(cube.Pos{i, 0, 0}), block.Cobblestone{}, nil)
		}
		p.StartSneaking()
		p.SetHeldItems(held, item.Stack{})
		s.veinMine(p, tx, base)
		if broken := droppedKeys(tx)[itemKey(block.Gravel{})]; broken != oneFood {
			t.Fatalf("broke %d with one food point, want %d", broken, oneFood)
		}
		// 12 blocks cost 0.9375 food: the bar keeps its point, the fraction
		// is owed, and nothing is left for another block
		if p.Food() != 1 || s.veinBudget(p, held) != 0 {
			t.Fatalf("food %d budget %d after 12 blocks, want 1 and 0", p.Food(), s.veinBudget(p, held))
		}
		// fractions carry over
		p.SetFood(20)
		s.veinFood[p.H()] = 0
		s.payVeinFood(p, 6)
		s.payVeinFood(p, 7)
		if p.Food() != 19 {
			t.Fatalf("13 blocks cost %d food, want 1", 20-p.Food())
		}
	})
}

// TestDeathKeepsFood: respawning gives back the food level the player died
// with instead of a full bar.
func TestDeathKeepsFood(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		hd := &handler{s: s}
		p.SetFood(5)
		hd.HandleDeath(p, nil, new(bool))
		p.SetFood(20) // what Dragonfly's respawn does
		pos, w := p.Position(), tx.World()
		hd.HandleRespawn(p, &pos, &w)
		if p.Food() != 5 {
			t.Fatalf("food after respawn %d, want 5", p.Food())
		}
	})
}
