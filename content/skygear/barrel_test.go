package skygear

import (
	"testing"
	"time"

	"github.com/presentkim/skygear/content/skygear/farm"

	"github.com/presentkim/skygear/content/skygear/machine"

	"github.com/presentkim/skygear/content/skygear/island"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/fluidnet"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/itemio"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/sievetable"
)

// TestBarrelStages: six fill stages at fixed counts, water on top.
func TestBarrelStages(t *testing.T) {
	want := map[int]int{0: 0, 5: 0, 6: 1, 10: 1, 11: 2, 16: 3, 21: 3, 22: 4, 27: 5, 32: 6}
	for n, stage := range want {
		if got := farm.BarrelStageOf("oak", &farm.BarrelRecord{Count: n}); got != stage {
			t.Errorf("%d items: stage %d, want %d", n, got, stage)
		}
	}
	if farm.BarrelStageOf("oak", &farm.BarrelRecord{Count: 32, Water: true}) != farm.BarrelWater || farm.BarrelStageOf("oak", nil) != 0 {
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
		pos := cube.Pos{4, island.IslandY, 4}
		barrel := farm.Barrel{Wood: "oak"}
		tx.SetBlock(pos, barrel, nil)
		leaves := item.NewStack(block.Leaves{Type: block.OakLeaves()}, 1)
		var ctx item.UseContext
		if farm.UseBarrel(s.Session, p, tx, pos, barrel, item.NewStack(item.Stick{}, 1), &ctx) {
			t.Fatal("a stick was put in a barrel")
		}
		for i := 1; i <= farm.MatterCapacity; i++ {
			cur := tx.Block(pos).(farm.Barrel)
			if !farm.UseBarrel(s.Session, p, tx, pos, cur, leaves, &ctx) {
				t.Fatalf("item %d was not taken", i)
			}
			if got := tx.Block(pos).(farm.Barrel).Stage; got != i*farm.BarrelFillStages/farm.MatterCapacity {
				t.Fatalf("after %d items the stage is %d", i, got)
			}
		}
		if farm.UseBarrel(s.Session, p, tx, pos, tx.Block(pos).(farm.Barrel), leaves, &ctx) != true || farm.BarrelAt(s.Session, pos).Count != farm.MatterCapacity {
			t.Fatal("a full barrel took another item")
		}
		bucket := item.NewStack(item.Bucket{}, 1)
		farm.UseBarrel(s.Session, p, tx, pos, tx.Block(pos).(farm.Barrel), bucket, &ctx)
		if farm.BarrelAt(s.Session, pos).Water {
			t.Fatal("the water was ready at once")
		}
		// Time passes: the clock runs on, the second tick makes water.
		farm.BarrelAt(s.Session, pos).ReadyAt = 0
		farm.TickBarrels(s.Session, tx)
		if got := tx.Block(pos).(farm.Barrel).Stage; got != farm.BarrelWater {
			t.Fatalf("stage %d after the delay, want water", got)
		}
		// A non-bucket does not take the water; an empty bucket does.
		if !farm.UseBarrel(s.Session, p, tx, pos, tx.Block(pos).(farm.Barrel), leaves, &ctx) || farm.BarrelAt(s.Session, pos) == nil {
			t.Fatal("leaves took the water")
		}
		if !farm.UseBarrel(s.Session, p, tx, pos, tx.Block(pos).(farm.Barrel), bucket, &ctx) {
			t.Fatal("the bucket did not answer")
		}
		if farm.BarrelAt(s.Session, pos) != nil || tx.Block(pos).(farm.Barrel).Stage != 0 {
			t.Error("the barrel is not empty after the water was taken")
		}
		// A single held bucket turns into the full one in the hand.
		farm.BarrelAt(s.Session, pos)
		farm.StateOf(s.Session).Barrels[farm.CropKey(pos)] = &farm.BarrelRecord{Count: farm.MatterCapacity, Water: true}
		p.SetHeldItems(bucket, item.Stack{})
		if !farm.UseBarrel(s.Session, p, tx, pos, tx.Block(pos).(farm.Barrel), bucket, &item.UseContext{}) {
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
	s.World.Do(func(tx *world.Tx) {
		pos := cube.Pos{5, island.IslandY, 5}
		tx.SetBlock(pos, farm.Barrel{Wood: "oak"}, nil)
		farm.StateOf(s.Session).Barrels = map[string]*farm.BarrelRecord{farm.CropKey(pos): {Count: 3}}
		tx.SetBlock(pos, block.Air{}, nil)
		farm.TickBarrels(s.Session, tx)
		if len(farm.StateOf(s.Session).Barrels) != 0 {
			t.Error("the record of a missing barrel stayed")
		}
	})
}

// TestStoneBarrelMeltsLikeACrucible: a stone barrel takes blocks to melt up
// to 1000 mB of solids (250 mB each) and melts every 0.5 seconds as many mB
// as the heat value of the block under it (Ex Deorum's crucible heat
// sources), up to 4000 mB of lava; without heat nothing melts; a bucket takes
// 1000 mB of lava, and only when there is that much.
func TestStoneBarrelMeltsLikeACrucible(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	cobble := item.NewStack(block.Cobblestone{}, 1)
	for _, c := range []struct {
		name  string
		below world.Block
		heat  int
	}{
		{"torch", block.Torch{Facing: cube.FaceDown, Type: block.NormalFire()}, 1},
		{"soul torch", block.Torch{Facing: cube.FaceDown, Type: block.SoulFire()}, 2},
		{"lantern", block.Lantern{Type: block.NormalFire()}, 1},
		{"soul lantern", block.Lantern{Type: block.SoulFire()}, 2},
		{"campfire", block.Campfire{Type: block.NormalFire()}, 2},
		{"put out campfire", block.Campfire{Type: block.NormalFire(), Extinguished: true}, 0},
		{"lava", block.Lava{Still: true, Depth: 8}, 3},
		{"fire", block.Fire{Type: block.NormalFire()}, 5},
		{"magma block", block.Magma{}, 0},
		{"nothing", block.Air{}, 0},
	} {
		withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
			pos := cube.Pos{4, island.IslandY + 3, 4}
			barrel := farm.Barrel{Wood: farm.StoneWood}
			delete(farm.StateOf(s.Session).Barrels, farm.CropKey(pos))
			tx.SetBlock(pos.Side(cube.FaceDown), c.below, nil)
			tx.SetBlock(pos, barrel, nil)
			var ctx item.UseContext
			if farm.UseBarrel(s.Session, p, tx, pos, barrel, item.NewStack(item.Stick{}, 1), &ctx) {
				t.Fatal("a stick was put in a stone barrel")
			}
			for i := 1; i <= farm.StoneSolidsMax/farm.StoneMeltMB; i++ {
				ctx = item.UseContext{}
				if !farm.UseBarrel(s.Session, p, tx, pos, tx.Block(pos).(farm.Barrel), cobble, &ctx) || ctx.CountSub != 1 {
					t.Fatalf("%s: cobblestone %d was not taken", c.name, i)
				}
			}
			ctx = item.UseContext{}
			if !farm.UseBarrel(s.Session, p, tx, pos, tx.Block(pos).(farm.Barrel), cobble, &ctx) || ctx.CountSub != 0 {
				t.Fatalf("%s: a fifth cobblestone was taken (or the click placed it)", c.name)
			}
			rec := farm.BarrelAt(s.Session, pos)
			if rec.Solids != farm.StoneSolidsMax || tx.Block(pos).(farm.Barrel).Stage != farm.BarrelFillStages {
				t.Fatalf("%s: %d mB of solids, stage %d", c.name, rec.Solids, tx.Block(pos).(farm.Barrel).Stage)
			}
			// Ten seconds are 20 steps.
			rec.MeltAt -= 10 * time.Second
			farm.TickBarrels(s.Session, tx)
			if want := 20 * c.heat; rec.LavaMB != want || rec.Solids != farm.StoneSolidsMax-want {
				t.Fatalf("%s: %d mB lava and %d solids after 10 s, want %d lava", c.name, rec.LavaMB, rec.Solids, want)
			}
		})
	}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		pos := cube.Pos{8, island.IslandY + 3, 8}
		barrel := farm.Barrel{Wood: farm.StoneWood}
		tx.SetBlock(pos.Side(cube.FaceDown), block.Torch{Facing: cube.FaceDown, Type: block.NormalFire()}, nil)
		tx.SetBlock(pos, barrel, nil)
		for range farm.StoneSolidsMax / farm.StoneMeltMB {
			farm.UseBarrel(s.Session, p, tx, pos, tx.Block(pos).(farm.Barrel), cobble, &item.UseContext{})
		}
		rec := farm.BarrelAt(s.Session, pos)
		// A torch melts the 1000 mB in 500 seconds; a bucket wants 1000 mB.
		rec.MeltAt -= 499 * time.Second
		farm.TickBarrels(s.Session, tx)
		if rec.LavaMB != 998 || tx.Block(pos).(farm.Barrel).Stage != farm.BarrelWater {
			t.Fatalf("%d mB of lava after 499 s, stage %d", rec.LavaMB, tx.Block(pos).(farm.Barrel).Stage)
		}
		var ctx item.UseContext
		p.SetHeldItems(item.NewStack(item.Bucket{}, 1), item.Stack{})
		if !farm.UseBarrel(s.Session, p, tx, pos, tx.Block(pos).(farm.Barrel), item.NewStack(item.Bucket{}, 1), &ctx) {
			t.Fatal("the bucket did not answer")
		}
		if held, _ := p.HeldItems(); !held.Comparable(item.NewStack(item.Bucket{}, 1)) || rec.LavaMB != 998 {
			t.Fatal("a bucket took less than 1000 mB of lava")
		}
		rec.MeltAt -= time.Second
		farm.TickBarrels(s.Session, tx)
		if rec.LavaMB != machine.BucketMB || rec.Solids != 0 {
			t.Fatalf("%d mB of lava and %d solids after 500 s", rec.LavaMB, rec.Solids)
		}
		if !farm.UseBarrel(s.Session, p, tx, pos, tx.Block(pos).(farm.Barrel), item.NewStack(item.Bucket{}, 1), &ctx) {
			t.Fatal("the bucket did not answer")
		}
		if held, _ := p.HeldItems(); machine.FluidOfBucket(held.Item().(item.Bucket)) != fluidnet.Lava || farm.BarrelAt(s.Session, pos) != nil {
			t.Fatalf("held %v, barrel %+v: want a lava bucket and an empty barrel", held, farm.BarrelAt(s.Session, pos))
		}
		if tx.Block(pos).(farm.Barrel).Stage != 0 {
			t.Fatal("the empty barrel still shows lava")
		}
		// The lava bucket pours back in, up to 4000 mB.
		for i := range farm.StoneLavaMax / machine.BucketMB {
			held, _ := p.HeldItems()
			if !farm.UseBarrel(s.Session, p, tx, pos, tx.Block(pos).(farm.Barrel), held, &item.UseContext{}) || farm.BarrelAt(s.Session, pos).LavaMB != (i+1)*machine.BucketMB {
				t.Fatalf("lava bucket %d was not poured in", i+1)
			}
			p.SetHeldItems(item.NewStack(farm.LavaBucket, 1), item.Stack{})
		}
		if !farm.UseBarrel(s.Session, p, tx, pos, tx.Block(pos).(farm.Barrel), item.NewStack(farm.LavaBucket, 1), &item.UseContext{}) || farm.BarrelAt(s.Session, pos).LavaMB != farm.StoneLavaMax {
			t.Fatal("lava went over 4000 mB")
		}
		if held, _ := p.HeldItems(); machine.FluidOfBucket(held.Item().(item.Bucket)) != fluidnet.Lava {
			t.Fatal("the bucket of a full barrel was emptied")
		}
		// Full of lava, the solids wait.
		rec = farm.BarrelAt(s.Session, pos)
		rec.Solids, rec.MeltAt = farm.StoneSolidsMax, rec.MeltAt-time.Hour
		farm.TickBarrels(s.Session, tx)
		if rec.LavaMB != farm.StoneLavaMax || rec.Solids != farm.StoneSolidsMax {
			t.Fatalf("%d lava, %d solids: melted over the top", rec.LavaMB, rec.Solids)
		}
		lines := farm.StoneBarrelLines(s.Session, i18n.Default, tx, pos)
		if len(lines) != 2 || lines[1] != i18n.T(i18n.Default, "barrel.stone.lines.lavaFull") {
			t.Fatalf("panel %q", lines)
		}
		// A conduit takes any part of the lava.
		if k, n := (farm.Barrel{Wood: farm.StoneWood}).DrainFluid(tx, pos, 300, false); k != fluidnet.Lava || n != 300 || rec.LavaMB != farm.StoneLavaMax-300 {
			t.Fatalf("drained %v %d, %d left", k, n, rec.LavaMB)
		}
	})
}

// TestStoneBarrelOldSave: a stone barrel of an old save (a bucket of lava,
// or cobblestone heating) holds the same as lava and solids.
func TestStoneBarrelOldSave(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		lava, cobble := cube.Pos{4, island.IslandY + 3, 4}, cube.Pos{6, island.IslandY + 3, 4}
		tx.SetBlock(lava, farm.Barrel{Wood: farm.StoneWood, Stage: farm.BarrelWater}, nil)
		tx.SetBlock(cobble, farm.Barrel{Wood: farm.StoneWood, Stage: 3}, nil)
		farm.StateOf(s.Session).Barrels = map[string]*farm.BarrelRecord{
			farm.CropKey(lava):   {Count: farm.LegacyStoneCount, Lava: true, Heat: time.Minute},
			farm.CropKey(cobble): {Count: 4, Heat: 10 * time.Second},
		}
		farm.TickBarrels(s.Session, tx)
		if r := farm.BarrelAt(s.Session, lava); r.LavaMB != machine.BucketMB || r.Solids != 0 || r.Lava || r.Count != 0 || r.Heat != 0 {
			t.Errorf("old lava barrel: %+v", r)
		}
		if r := farm.BarrelAt(s.Session, cobble); r.Solids != 500 || r.LavaMB != 0 || r.Count != 0 {
			t.Errorf("old cobblestone barrel: %+v", r)
		}
	})
}

// TestDustMakesClayInWater: a dust used on a barrel with water becomes a clay
// ball and the water stays; a dry barrel does nothing with it.
func TestDustMakesClayInWater(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	pos := machinePos
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		tx.SetBlock(pos, farm.Barrel{Wood: "oak"}, nil)
		dust := item.NewStack(sgitem.Dust{}, 2)
		ctx := &item.UseContext{}
		if farm.UseBarrel(s.Session, p, tx, pos, tx.Block(pos).(farm.Barrel), dust, ctx) && ctx.CountSub != 0 {
			t.Fatal("a dry barrel took the dust")
		}
		if farm.StateOf(s.Session).Barrels == nil {
			farm.StateOf(s.Session).Barrels = map[string]*farm.BarrelRecord{}
		}
		farm.StateOf(s.Session).Barrels[farm.CropKey(pos)] = &farm.BarrelRecord{Count: farm.MatterCapacity, Water: true}
		ctx = &item.UseContext{}
		if !farm.UseBarrel(s.Session, p, tx, pos, tx.Block(pos).(farm.Barrel), dust, ctx) || ctx.CountSub != 1 {
			t.Fatalf("the dust was not used (count sub %d)", ctx.CountSub)
		}
		if !p.Inventory().ContainsItem(item.NewStack(block.Clay{}, 1)) {
			t.Error("no clay block")
		}
		if rec := farm.BarrelAt(s.Session, pos); rec == nil || !rec.Water {
			t.Error("the water went")
		}
	})
	for _, drops := range sievetable.Tables["string"] {
		for _, d := range drops {
			if _, clay := d.Item.Item().(item.ClayBall); clay {
				t.Error("a sieve still drops clay")
			}
		}
	}
}

func TestEmptyBarrelTakesItsLiquidBack(t *testing.T) {
	water := item.NewStack(item.Bucket{Content: item.LiquidBucketContent(block.Water{Depth: 8, Still: true})}, 1)
	cases := []struct {
		wood       string
		held       item.Stack
		want       farm.BarrelRecord
		emptyAfter world.Item
	}{
		{"oak", water, farm.BarrelRecord{Water: true}, item.Bucket{}},
		{"oak", item.NewStack(sgitem.WoodenBucket{Water: true}, 1), farm.BarrelRecord{Water: true}, sgitem.WoodenBucket{}},
		{"oak", item.NewStack(sgitem.WitchWaterBucket{}, 1), farm.BarrelRecord{Witch: true}, item.Bucket{}},
		{farm.StoneWood, item.NewStack(farm.LavaBucket, 1), farm.BarrelRecord{LavaMB: machine.BucketMB}, item.Bucket{}},
		{farm.StoneWood, water, farm.BarrelRecord{}, nil},
		{"oak", item.NewStack(farm.LavaBucket, 1), farm.BarrelRecord{}, nil},
	}
	for _, c := range cases {
		s := testSession(t)
		loadAround(s)
		h := spawnPlayer(s)
		withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
			tx.SetBlock(machinePos, farm.Barrel{Wood: c.wood}, nil)
			p.SetHeldItems(c.held, item.Stack{})
			used := farm.UseBarrel(s.Session, p, tx, machinePos, tx.Block(machinePos).(farm.Barrel), c.held, &item.UseContext{})
			rec := farm.BarrelAt(s.Session, machinePos)
			if c.emptyAfter == nil {
				if used || rec != nil {
					t.Errorf("%s barrel took %s", c.wood, naming.ItemKey(c.held.Item()))
				}
				return
			}
			if !used || rec == nil || *rec != c.want {
				t.Errorf("%s barrel with %s: used %v, record %+v", c.wood, naming.ItemKey(c.held.Item()), used, rec)
			}
			if held, _ := p.HeldItems(); naming.ItemKey(held.Item()) != naming.ItemKey(c.emptyAfter) {
				t.Errorf("%s left %s in the hand", naming.ItemKey(c.held.Item()), naming.ItemKey(held.Item()))
			}
			if got := tx.Block(machinePos).(farm.Barrel).Stage; got != farm.BarrelStageOf(c.wood, rec) {
				t.Errorf("stage %d", got)
			}
		})
	}
}

// TestBarrelClayByHopper: dust put into a barrel of water by an item
// transfer makes clay blocks the barrel keeps until they are taken out; a dry
// barrel takes no dust, the buffer is limited and breaking the barrel drops
// the clay.
func TestBarrelClayByHopper(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		pos := cube.Pos{4, island.IslandY + 3, 4}
		tx.SetBlock(pos, farm.Barrel{Wood: "oak"}, nil)
		dust := item.NewStack(sgitem.Dust{}, 40)
		if n := itemio.Insert(tx, pos, cube.FaceUp, dust); n != 0 {
			t.Fatalf("a dry barrel took %d dust", n)
		}
		if farm.StateOf(s.Session).Barrels == nil {
			farm.StateOf(s.Session).Barrels = map[string]*farm.BarrelRecord{}
		}
		farm.StateOf(s.Session).Barrels[farm.CropKey(pos)] = &farm.BarrelRecord{Count: farm.MatterCapacity, Water: true}
		if n := itemio.Insert(tx, pos, cube.FaceUp, dust); n != farm.BarrelClayBuffer {
			t.Fatalf("took %d dust, want %d (the clay buffer)", n, farm.BarrelClayBuffer)
		}
		if n := itemio.Insert(tx, pos, cube.FaceUp, dust); n != 0 {
			t.Fatalf("a full clay buffer took %d more dust", n)
		}
		_, ex := itemio.IO(tx, pos)
		peek := ex.PeekItems(tx, pos, cube.FaceDown, nil)
		if _, ok := peek.Item().(block.Clay); !ok || peek.Count() != farm.BarrelClayBuffer {
			t.Fatalf("peek = %v", peek)
		}
		got := ex.ExtractItems(tx, pos, cube.FaceDown, peek, 4)
		if got.Count() != 4 || farm.BarrelAt(s.Session, pos).Clay != farm.BarrelClayBuffer-4 {
			t.Fatalf("extracted %v, clay left %d", got, farm.BarrelAt(s.Session, pos).Clay)
		}
		if !farm.BarrelAt(s.Session, pos).Water {
			t.Fatal("the water went")
		}
		var drops []item.Stack
		farm.BreakBarrel(s.Session, pos, &drops)
		if len(drops) != 1 || drops[0].Count() != farm.BarrelClayBuffer-4 {
			t.Fatalf("broken barrel drops %v", drops)
		}
	})
}

// TestWoodenBucket: the wooden bucket takes the water of a barrel and of a
// water source and pours it again, but never takes lava.
func TestWoodenBucket(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		pos := cube.Pos{4, island.IslandY + 3, 4}
		tx.SetBlock(pos, farm.Barrel{Wood: "oak"}, nil)
		if farm.StateOf(s.Session).Barrels == nil {
			farm.StateOf(s.Session).Barrels = map[string]*farm.BarrelRecord{}
		}
		empty := item.NewStack(sgitem.WoodenBucket{}, 1)
		farm.StateOf(s.Session).Barrels[farm.CropKey(pos)] = &farm.BarrelRecord{Count: farm.MatterCapacity, Water: true}
		p.SetHeldItems(empty, item.Stack{})
		if !farm.UseBarrel(s.Session, p, tx, pos, tx.Block(pos).(farm.Barrel), empty, &item.UseContext{}) {
			t.Fatal("the water of the barrel was not used")
		}
		if held, _ := p.HeldItems(); held.Item() != (sgitem.WoodenBucket{Water: true}) {
			t.Fatalf("held %v, want a wooden bucket of water", held)
		}
		if farm.BarrelAt(s.Session, pos) != nil {
			t.Fatal("the barrel kept its water")
		}
		// Lava stays in the barrel.
		stone := cube.Pos{6, island.IslandY + 3, 4}
		tx.SetBlock(stone, farm.Barrel{Wood: farm.StoneWood}, nil)
		farm.StateOf(s.Session).Barrels[farm.CropKey(stone)] = &farm.BarrelRecord{LavaMB: machine.BucketMB}
		p.SetHeldItems(empty, item.Stack{})
		if !farm.UseBarrel(s.Session, p, tx, stone, tx.Block(stone).(farm.Barrel), empty, &item.UseContext{}) {
			t.Fatal("the click on the lava barrel was not used")
		}
		if held, _ := p.HeldItems(); !held.Comparable(empty) {
			t.Fatalf("a wooden bucket took lava: %v", held)
		}
		if rec := farm.BarrelAt(s.Session, stone); rec == nil || rec.LavaMB != machine.BucketMB {
			t.Fatal("the lava went")
		}
		// A water source fills the bucket and the full bucket pours it out.
		src := cube.Pos{8, island.IslandY + 3, 8}
		tx.SetLiquid(src, block.Water{Depth: 8, Still: true})
		var ctx item.UseContext
		if !(sgitem.WoodenBucket{}).UseOnBlock(src, cube.FaceUp, mgl64.Vec3{}, tx, p, &ctx) || ctx.NewItem.Empty() {
			t.Fatal("the bucket did not take the water source")
		}
		if _, ok := tx.Liquid(src); ok {
			t.Fatal("the source stayed")
		}
		ctx = item.UseContext{}
		if !(sgitem.WoodenBucket{Water: true}).UseOnBlock(src, cube.FaceUp, mgl64.Vec3{}, tx, p, &ctx) {
			t.Fatal("the full bucket did not pour")
		}
		if _, ok := tx.Liquid(src); !ok {
			t.Fatal("no water poured")
		}
	})
}
