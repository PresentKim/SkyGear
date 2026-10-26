package dragonfactory

import (
	"log/slog"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/internal/account"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/dfitem"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/naming"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/sievetable"
	"github.com/presentkim/dragonfactory/internal/instance"
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
		hammer := item.NewStack(dfitem.Hammer{}, 1)
		if d := mine(block.Cobblestone{}, hammer); len(d) != 1 || naming.ItemKey(d[0].Item()) != naming.ItemKey(block.Gravel{}) {
			t.Fatalf("cobblestone with a hammer drops %v", d)
		}
		if d := mine(block.Gravel{}, hammer); len(d) != 1 || naming.ItemKey(d[0].Item()) != naming.ItemKey(block.Sand{}) {
			t.Fatalf("gravel with a hammer drops %v", d)
		}
		if d := mine(block.Cobblestone{}, item.NewStack(item.Pickaxe{Tier: item.ToolTierStone}, 1)); naming.ItemKey(d[0].Item()) != naming.ItemKey(block.Cobblestone{}) {
			t.Fatalf("cobblestone with a pickaxe drops %v", d)
		}
	})
	// Clients get the hammer's mining speeds.
	if (dfitem.Hammer{}).DestroySpeeds()["minecraft:cobblestone"] == 0 {
		t.Fatal("no client speed for cobblestone")
	}
	if block.BreakDuration(block.Cobblestone{}, item.NewStack(dfitem.Hammer{}, 1), block.BreakContext{}) >= block.BreakDuration(block.Cobblestone{}, item.Stack{}, block.BreakContext{}) {
		t.Fatal("the hammer does not mine cobblestone faster than a hand")
	}
}

// TestDustFromCrushedSand: sand crushes into dust (sky progression); dirt
// and sand sift out the farming and kelp drops.
func TestDustFromCrushedSand(t *testing.T) {
	if out, ok := dfitem.Crushed(block.Sand{}); !ok || naming.ItemKey(out) != naming.ItemKey(dfitem.Dust{}) {
		t.Fatalf("crushed sand = %v", out)
	}
	has := func(in world.Item, want world.Item) bool {
		for _, d := range sievetable.Tables["string"][naming.ItemKey(in)] {
			if naming.ItemKey(d.Item.Item()) == naming.ItemKey(want) {
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
		block.Deepslate{}: dfitem.CrushedBlock{Kind: dfitem.CrushedDeepslate},
		block.Deepslate{Type: block.CobbledDeepslate()}: dfitem.CrushedBlock{Kind: dfitem.CrushedDeepslate},
		block.Netherrack{}: dfitem.CrushedBlock{Kind: dfitem.CrushedNetherrack},
		dfitem.CrushedBlock{Kind: dfitem.CrushedNetherrack}: block.Sand{Red: true},
		block.Blackstone{}:    dfitem.CrushedBlock{Kind: dfitem.CrushedBlackstone},
		block.EndStone{}:      dfitem.CrushedBlock{Kind: dfitem.CrushedEndStone},
		block.Sand{Red: true}: dfitem.Dust{},
	} {
		if out, ok := dfitem.Crushed(in.(world.Block)); !ok || naming.ItemKey(out) != naming.ItemKey(want) {
			t.Errorf("crushed %s = %v, want %s", naming.ItemKey(in), out, naming.ItemKey(want))
		}
	}
}

// TestHammerOre: the hammer breaks an ore block into two dusts of its metal.
func TestHammerOre(t *testing.T) {
	for _, b := range []world.Block{block.IronOre{}, dfitem.OreBlock{Metal: dfitem.MetalNickel}} {
		it := b.(world.Item)
		metal, _ := dfitem.OreBlockMetal(it)
		got, ok := dfitem.HammerDrop(b)
		if !ok || got.Count() != dfitem.HammerDust || naming.ItemKey(got.Item()) != naming.ItemKey(dfitem.MetalDust{Metal: metal}) {
			t.Errorf("hammer on %s drops %v", naming.ItemKey(it), got)
		}
	}
}

// TestHammerTiers: the stone hammer lasts twice and the iron hammer four
// times as long as tools of their material, and only the iron hammer vein
// mines.
func TestHammerTiers(t *testing.T) {
	stone, iron := dfitem.Hammer{Stone: true}, dfitem.Hammer{}
	if stone.DurabilityInfo().MaxDurability != 2*item.ToolTierStone.Durability || iron.DurabilityInfo().MaxDurability != 4*item.ToolTierIron.Durability {
		t.Fatalf("durability stone %d iron %d", stone.DurabilityInfo().MaxDurability, iron.DurabilityInfo().MaxDurability)
	}
	if dfitem.IsIronHammer(item.NewStack(stone, 1)) || !dfitem.IsIronHammer(item.NewStack(iron, 1)) || !dfitem.IsHammer(item.NewStack(stone, 1)) {
		t.Fatal("hammer kinds mixed up")
	}
	if (dfitem.Hammer{}).BaseMiningEfficiency(block.Gravel{}) != 3 || (dfitem.Hammer{}).BaseMiningEfficiency(dfitem.Dust{}) != 3 {
		t.Fatal("gravel and dust speeds")
	}
}

func TestVeinDiagonalSetting(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	base := cube.Pos{4, IslandY + 3, 4}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		tx.SetBlock(base, block.Gravel{}, nil)
		tx.SetBlock(base.Add(cube.Pos{1, 1, 0}), block.Gravel{}, nil)
		tx.SetBlock(base.Add(cube.Pos{2, 2, 1}), block.Gravel{}, nil)
		if got := veinBlocks(tx, base, veinLimit, false, s.canBuild); len(got) != 1 {
			t.Fatalf("face vein of %d blocks, want 1", len(got))
		}
		if got := veinBlocks(tx, base, veinLimit, true, s.canBuild); len(got) != 3 {
			t.Fatalf("diagonal vein of %d blocks, want 3", len(got))
		}
		if _, err := s.g.db.TouchAccount(instance.AccountID(p), "", p.Name()); err != nil {
			t.Fatal(err)
		}
		account.Init(s.g.db, slog.Default())
		t.Cleanup(func() { account.Init(nil, slog.Default()) })
		if account.On(p, veinDiagonalSetting) {
			t.Fatal("diagonal on before it was switched")
		}
		s.toggleVeinDiagonal(p)
		account.Forget(p) // read it again from the database
		if !account.On(p, veinDiagonalSetting) {
			t.Fatal("the switched setting was not stored")
		}
	})
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
		if got := veinBlocks(tx, base, veinLimit, false, s.canBuild); len(got) != 5 {
			t.Fatalf("vein of %d blocks, want 5", len(got))
		}
		if got := veinBlocks(tx, base, 3, false, s.canBuild); len(got) != 3 {
			t.Fatalf("limited vein of %d blocks, want 3", len(got))
		}
		p.StartSneaking()
		p.SetHeldItems(item.NewStack(dfitem.Hammer{Stone: true}, 1), item.Stack{})
		s.veinMine(p, tx, base)
		if _, ok := tx.Block(base.Add(cube.Pos{1, 0, 0})).(block.Gravel); !ok {
			t.Fatal("the stone hammer vein mined")
		}
		p.SetHeldItems(item.NewStack(dfitem.Hammer{}, 1), item.Stack{})
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
		if held, _ := p.HeldItems(); held.Durability() != held.MaxDurability() {
			t.Fatalf("the iron hammer wore down %d", held.MaxDurability()-held.Durability())
		}
		if got := droppedKeys(tx)[naming.ItemKey(block.Sand{})]; got != 4 {
			t.Fatalf("%d sand dropped, want 4", got)
		}
		var at []mgl64.Vec3
		for e := range tx.Entities() {
			if it, ok := e.(*entity.Ent); ok {
				if _, ok := it.Behaviour().(*entity.ItemBehaviour); ok {
					at = append(at, it.Position())
				}
			}
		}
		if len(at) != 1 || at[0] != base.Vec3Centre() {
			t.Fatalf("drops at %v, want one stack at %v", at, base.Vec3Centre())
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
		held := item.NewStack(dfitem.Hammer{}, 1)
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
		if broken := droppedKeys(tx)[naming.ItemKey(block.Gravel{})]; broken != oneFood {
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
