package dragonfactory

import (
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/internal/fluidnet"
	"github.com/presentkim/dragonfactory/internal/itemio"
)

// TestBarrelStages: six fill stages at fixed counts, water on top.
func TestBarrelStages(t *testing.T) {
	want := map[int]int{0: 0, 5: 0, 6: 1, 10: 1, 11: 2, 16: 3, 21: 3, 22: 4, 27: 5, 32: 6}
	for n, stage := range want {
		if got := barrelStage(&BarrelRecord{Count: n}); got != stage {
			t.Errorf("%d items: stage %d, want %d", n, got, stage)
		}
	}
	if barrelStage(&BarrelRecord{Count: 32, Water: true}) != barrelWater || barrelStage(nil) != 0 {
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
		pos := cube.Pos{4, IslandY, 4}
		barrel := Barrel{Wood: "oak"}
		tx.SetBlock(pos, barrel, nil)
		leaves := item.NewStack(block.Leaves{Type: block.OakLeaves()}, 1)
		var ctx item.UseContext
		if s.useBarrel(p, tx, pos, barrel, item.NewStack(item.Stick{}, 1), &ctx) {
			t.Fatal("a stick was put in a barrel")
		}
		for i := 1; i <= matterCapacity; i++ {
			cur := tx.Block(pos).(Barrel)
			if !s.useBarrel(p, tx, pos, cur, leaves, &ctx) {
				t.Fatalf("item %d was not taken", i)
			}
			if got := tx.Block(pos).(Barrel).Stage; got != i*barrelFillStages/matterCapacity {
				t.Fatalf("after %d items the stage is %d", i, got)
			}
		}
		if s.useBarrel(p, tx, pos, tx.Block(pos).(Barrel), leaves, &ctx) != true || s.barrelAt(pos).Count != matterCapacity {
			t.Fatal("a full barrel took another item")
		}
		bucket := item.NewStack(item.Bucket{}, 1)
		s.useBarrel(p, tx, pos, tx.Block(pos).(Barrel), bucket, &ctx)
		if s.barrelAt(pos).Water {
			t.Fatal("the water was ready at once")
		}
		// Time passes: the clock runs on, the second tick makes water.
		s.barrelAt(pos).ReadyAt = 0
		s.tickBarrels(tx)
		if got := tx.Block(pos).(Barrel).Stage; got != barrelWater {
			t.Fatalf("stage %d after the delay, want water", got)
		}
		// A non-bucket does not take the water; an empty bucket does.
		if !s.useBarrel(p, tx, pos, tx.Block(pos).(Barrel), leaves, &ctx) || s.barrelAt(pos) == nil {
			t.Fatal("leaves took the water")
		}
		if !s.useBarrel(p, tx, pos, tx.Block(pos).(Barrel), bucket, &ctx) {
			t.Fatal("the bucket did not answer")
		}
		if s.barrelAt(pos) != nil || tx.Block(pos).(Barrel).Stage != 0 {
			t.Error("the barrel is not empty after the water was taken")
		}
		// A single held bucket turns into the full one in the hand.
		s.barrelAt(pos)
		s.state.Barrels[cropKey(pos)] = &BarrelRecord{Count: matterCapacity, Water: true}
		p.SetHeldItems(bucket, item.Stack{})
		if !s.useBarrel(p, tx, pos, tx.Block(pos).(Barrel), bucket, &item.UseContext{}) {
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
	s.world.Do(func(tx *world.Tx) {
		pos := cube.Pos{5, IslandY, 5}
		tx.SetBlock(pos, Barrel{Wood: "oak"}, nil)
		s.state.Barrels = map[string]*BarrelRecord{cropKey(pos): {Count: 3}}
		tx.SetBlock(pos, block.Air{}, nil)
		s.tickBarrels(tx)
		if len(s.state.Barrels) != 0 {
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
			pos := cube.Pos{4, IslandY + 3, 4}
			barrel := Barrel{Wood: stoneWood}
			delete(s.state.Barrels, cropKey(pos))
			tx.SetBlock(pos.Side(cube.FaceDown), c.below, nil)
			tx.SetBlock(pos, barrel, nil)
			var ctx item.UseContext
			if s.useBarrel(p, tx, pos, barrel, item.NewStack(item.Stick{}, 1), &ctx) {
				t.Fatal("a stick was put in a stone barrel")
			}
			for i := 1; i <= stoneSolidsMax/stoneMeltMB; i++ {
				ctx = item.UseContext{}
				if !s.useBarrel(p, tx, pos, tx.Block(pos).(Barrel), cobble, &ctx) || ctx.CountSub != 1 {
					t.Fatalf("%s: cobblestone %d was not taken", c.name, i)
				}
			}
			ctx = item.UseContext{}
			if !s.useBarrel(p, tx, pos, tx.Block(pos).(Barrel), cobble, &ctx) || ctx.CountSub != 0 {
				t.Fatalf("%s: a fifth cobblestone was taken (or the click placed it)", c.name)
			}
			rec := s.barrelAt(pos)
			if rec.Solids != stoneSolidsMax || tx.Block(pos).(Barrel).Stage != barrelFillStages {
				t.Fatalf("%s: %d mB of solids, stage %d", c.name, rec.Solids, tx.Block(pos).(Barrel).Stage)
			}
			// Ten seconds are 20 steps.
			rec.MeltAt -= 10 * time.Second
			s.tickBarrels(tx)
			if want := 20 * c.heat; rec.LavaMB != want || rec.Solids != stoneSolidsMax-want {
				t.Fatalf("%s: %d mB lava and %d solids after 10 s, want %d lava", c.name, rec.LavaMB, rec.Solids, want)
			}
		})
	}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		pos := cube.Pos{8, IslandY + 3, 8}
		barrel := Barrel{Wood: stoneWood}
		tx.SetBlock(pos.Side(cube.FaceDown), block.Torch{Facing: cube.FaceDown, Type: block.NormalFire()}, nil)
		tx.SetBlock(pos, barrel, nil)
		for range stoneSolidsMax / stoneMeltMB {
			s.useBarrel(p, tx, pos, tx.Block(pos).(Barrel), cobble, &item.UseContext{})
		}
		rec := s.barrelAt(pos)
		// A torch melts the 1000 mB in 500 seconds; a bucket wants 1000 mB.
		rec.MeltAt -= 499 * time.Second
		s.tickBarrels(tx)
		if rec.LavaMB != 998 || tx.Block(pos).(Barrel).Stage != barrelWater {
			t.Fatalf("%d mB of lava after 499 s, stage %d", rec.LavaMB, tx.Block(pos).(Barrel).Stage)
		}
		var ctx item.UseContext
		p.SetHeldItems(item.NewStack(item.Bucket{}, 1), item.Stack{})
		if !s.useBarrel(p, tx, pos, tx.Block(pos).(Barrel), item.NewStack(item.Bucket{}, 1), &ctx) {
			t.Fatal("the bucket did not answer")
		}
		if held, _ := p.HeldItems(); !held.Comparable(item.NewStack(item.Bucket{}, 1)) || rec.LavaMB != 998 {
			t.Fatal("a bucket took less than 1000 mB of lava")
		}
		rec.MeltAt -= time.Second
		s.tickBarrels(tx)
		if rec.LavaMB != bucketMB || rec.Solids != 0 {
			t.Fatalf("%d mB of lava and %d solids after 500 s", rec.LavaMB, rec.Solids)
		}
		if !s.useBarrel(p, tx, pos, tx.Block(pos).(Barrel), item.NewStack(item.Bucket{}, 1), &ctx) {
			t.Fatal("the bucket did not answer")
		}
		if held, _ := p.HeldItems(); fluidOfBucket(held.Item().(item.Bucket)) != fluidnet.Lava || s.barrelAt(pos) != nil {
			t.Fatalf("held %v, barrel %+v: want a lava bucket and an empty barrel", held, s.barrelAt(pos))
		}
		if tx.Block(pos).(Barrel).Stage != 0 {
			t.Fatal("the empty barrel still shows lava")
		}
		// The lava bucket pours back in, up to 4000 mB.
		for i := range stoneLavaMax / bucketMB {
			held, _ := p.HeldItems()
			if !s.useBarrel(p, tx, pos, tx.Block(pos).(Barrel), held, &item.UseContext{}) || s.barrelAt(pos).LavaMB != (i+1)*bucketMB {
				t.Fatalf("lava bucket %d was not poured in", i+1)
			}
			p.SetHeldItems(item.NewStack(lavaBucket, 1), item.Stack{})
		}
		if !s.useBarrel(p, tx, pos, tx.Block(pos).(Barrel), item.NewStack(lavaBucket, 1), &item.UseContext{}) || s.barrelAt(pos).LavaMB != stoneLavaMax {
			t.Fatal("lava went over 4000 mB")
		}
		if held, _ := p.HeldItems(); fluidOfBucket(held.Item().(item.Bucket)) != fluidnet.Lava {
			t.Fatal("the bucket of a full barrel was emptied")
		}
		// Full of lava, the solids wait.
		rec = s.barrelAt(pos)
		rec.Solids, rec.MeltAt = stoneSolidsMax, rec.MeltAt-time.Hour
		s.tickBarrels(tx)
		if rec.LavaMB != stoneLavaMax || rec.Solids != stoneSolidsMax {
			t.Fatalf("%d lava, %d solids: melted over the top", rec.LavaMB, rec.Solids)
		}
		lines := s.stoneBarrelLines(i18n.Default, tx, pos)
		if len(lines) != 2 || lines[1] != i18n.T(i18n.Default, "barrel.stone.lines.lavaFull") {
			t.Fatalf("panel %q", lines)
		}
		// A conduit takes any part of the lava.
		if k, n := (Barrel{Wood: stoneWood}).DrainFluid(tx, pos, 300, false); k != fluidnet.Lava || n != 300 || rec.LavaMB != stoneLavaMax-300 {
			t.Fatalf("drained %v %d, %d left", k, n, rec.LavaMB)
		}
	})
}

// TestStoneBarrelOldSave: a stone barrel of an older save (a bucket of lava,
// or cobblestone heating) holds the same as lava and solids.
func TestStoneBarrelOldSave(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		lava, cobble := cube.Pos{4, IslandY + 3, 4}, cube.Pos{6, IslandY + 3, 4}
		tx.SetBlock(lava, Barrel{Wood: stoneWood, Stage: barrelWater}, nil)
		tx.SetBlock(cobble, Barrel{Wood: stoneWood, Stage: 3}, nil)
		s.state.Barrels = map[string]*BarrelRecord{
			cropKey(lava):   {Count: legacyStoneCount, Lava: true, Heat: time.Minute},
			cropKey(cobble): {Count: 4, Heat: 10 * time.Second},
		}
		s.tickBarrels(tx)
		if r := s.barrelAt(lava); r.LavaMB != bucketMB || r.Solids != 0 || r.Lava || r.Count != 0 || r.Heat != 0 {
			t.Errorf("old lava barrel: %+v", r)
		}
		if r := s.barrelAt(cobble); r.Solids != 500 || r.LavaMB != 0 || r.Count != 0 {
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
		tx.SetBlock(pos, Barrel{Wood: "oak"}, nil)
		dust := item.NewStack(Dust{}, 2)
		ctx := &item.UseContext{}
		if s.useBarrel(p, tx, pos, tx.Block(pos).(Barrel), dust, ctx) && ctx.CountSub != 0 {
			t.Fatal("a dry barrel took the dust")
		}
		if s.state.Barrels == nil {
			s.state.Barrels = map[string]*BarrelRecord{}
		}
		s.state.Barrels[cropKey(pos)] = &BarrelRecord{Count: matterCapacity, Water: true}
		ctx = &item.UseContext{}
		if !s.useBarrel(p, tx, pos, tx.Block(pos).(Barrel), dust, ctx) || ctx.CountSub != 1 {
			t.Fatalf("the dust was not used (count sub %d)", ctx.CountSub)
		}
		if !p.Inventory().ContainsItem(item.NewStack(block.Clay{}, 1)) {
			t.Error("no clay block")
		}
		if rec := s.barrelAt(pos); rec == nil || !rec.Water {
			t.Error("the water went")
		}
	})
	for _, drops := range sieveTables["string"] {
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
		want       BarrelRecord
		emptyAfter world.Item
	}{
		{"oak", water, BarrelRecord{Water: true}, item.Bucket{}},
		{"oak", item.NewStack(WoodenBucket{Water: true}, 1), BarrelRecord{Water: true}, WoodenBucket{}},
		{"oak", item.NewStack(WitchWaterBucket{}, 1), BarrelRecord{Witch: true}, item.Bucket{}},
		{stoneWood, item.NewStack(lavaBucket, 1), BarrelRecord{LavaMB: bucketMB}, item.Bucket{}},
		{stoneWood, water, BarrelRecord{}, nil},
		{"oak", item.NewStack(lavaBucket, 1), BarrelRecord{}, nil},
	}
	for _, c := range cases {
		s := testSession(t)
		loadAround(s)
		h := spawnPlayer(s)
		withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
			tx.SetBlock(machinePos, Barrel{Wood: c.wood}, nil)
			p.SetHeldItems(c.held, item.Stack{})
			used := s.useBarrel(p, tx, machinePos, tx.Block(machinePos).(Barrel), c.held, &item.UseContext{})
			rec := s.barrelAt(machinePos)
			if c.emptyAfter == nil {
				if used || rec != nil {
					t.Errorf("%s barrel took %s", c.wood, itemKey(c.held.Item()))
				}
				return
			}
			if !used || rec == nil || *rec != c.want {
				t.Errorf("%s barrel with %s: used %v, record %+v", c.wood, itemKey(c.held.Item()), used, rec)
			}
			if held, _ := p.HeldItems(); itemKey(held.Item()) != itemKey(c.emptyAfter) {
				t.Errorf("%s left %s in the hand", itemKey(c.held.Item()), itemKey(held.Item()))
			}
			if got := tx.Block(machinePos).(Barrel).Stage; got != barrelStageOf(c.wood, rec) {
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
		pos := cube.Pos{4, IslandY + 3, 4}
		tx.SetBlock(pos, Barrel{Wood: "oak"}, nil)
		dust := item.NewStack(Dust{}, 40)
		if n := itemio.Insert(tx, pos, cube.FaceUp, dust); n != 0 {
			t.Fatalf("a dry barrel took %d dust", n)
		}
		if s.state.Barrels == nil {
			s.state.Barrels = map[string]*BarrelRecord{}
		}
		s.state.Barrels[cropKey(pos)] = &BarrelRecord{Count: matterCapacity, Water: true}
		if n := itemio.Insert(tx, pos, cube.FaceUp, dust); n != barrelClayBuffer {
			t.Fatalf("took %d dust, want %d (the clay buffer)", n, barrelClayBuffer)
		}
		if n := itemio.Insert(tx, pos, cube.FaceUp, dust); n != 0 {
			t.Fatalf("a full clay buffer took %d more dust", n)
		}
		_, ex := itemio.IO(tx, pos)
		peek := ex.PeekItems(tx, pos, cube.FaceDown, nil)
		if _, ok := peek.Item().(block.Clay); !ok || peek.Count() != barrelClayBuffer {
			t.Fatalf("peek = %v", peek)
		}
		got := ex.ExtractItems(tx, pos, cube.FaceDown, peek, 4)
		if got.Count() != 4 || s.barrelAt(pos).Clay != barrelClayBuffer-4 {
			t.Fatalf("extracted %v, clay left %d", got, s.barrelAt(pos).Clay)
		}
		if !s.barrelAt(pos).Water {
			t.Fatal("the water went")
		}
		var drops []item.Stack
		s.breakBarrel(pos, &drops)
		if len(drops) != 1 || drops[0].Count() != barrelClayBuffer-4 {
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
		pos := cube.Pos{4, IslandY + 3, 4}
		tx.SetBlock(pos, Barrel{Wood: "oak"}, nil)
		if s.state.Barrels == nil {
			s.state.Barrels = map[string]*BarrelRecord{}
		}
		empty := item.NewStack(WoodenBucket{}, 1)
		s.state.Barrels[cropKey(pos)] = &BarrelRecord{Count: matterCapacity, Water: true}
		p.SetHeldItems(empty, item.Stack{})
		if !s.useBarrel(p, tx, pos, tx.Block(pos).(Barrel), empty, &item.UseContext{}) {
			t.Fatal("the water of the barrel was not used")
		}
		if held, _ := p.HeldItems(); !isWaterBucket(held) || held.Item().(WoodenBucket).Water != true {
			t.Fatalf("held %v, want a wooden bucket of water", held)
		}
		if s.barrelAt(pos) != nil {
			t.Fatal("the barrel kept its water")
		}
		// Lava stays in the barrel.
		stone := cube.Pos{6, IslandY + 3, 4}
		tx.SetBlock(stone, Barrel{Wood: stoneWood}, nil)
		s.state.Barrels[cropKey(stone)] = &BarrelRecord{LavaMB: bucketMB}
		p.SetHeldItems(empty, item.Stack{})
		if !s.useBarrel(p, tx, stone, tx.Block(stone).(Barrel), empty, &item.UseContext{}) {
			t.Fatal("the click on the lava barrel was not used")
		}
		if held, _ := p.HeldItems(); !held.Comparable(empty) {
			t.Fatalf("a wooden bucket took lava: %v", held)
		}
		if rec := s.barrelAt(stone); rec == nil || rec.LavaMB != bucketMB {
			t.Fatal("the lava went")
		}
		// A water source fills the bucket and the full bucket pours it out.
		src := cube.Pos{8, IslandY + 3, 8}
		tx.SetLiquid(src, block.Water{Depth: 8, Still: true})
		var ctx item.UseContext
		if !(WoodenBucket{}).UseOnBlock(src, cube.FaceUp, mgl64.Vec3{}, tx, p, &ctx) || ctx.NewItem.Empty() {
			t.Fatal("the bucket did not take the water source")
		}
		if _, ok := tx.Liquid(src); ok {
			t.Fatal("the source stayed")
		}
		ctx = item.UseContext{}
		if !(WoodenBucket{Water: true}).UseOnBlock(src, cube.FaceUp, mgl64.Vec3{}, tx, p, &ctx) {
			t.Fatal("the full bucket did not pour")
		}
		if _, ok := tx.Liquid(src); !ok {
			t.Fatal("no water poured")
		}
	})
}
