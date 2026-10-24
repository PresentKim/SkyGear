package dragonfactory

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// droppedKeys are the item keys of the item entities lying in tx.
func droppedKeys(tx *world.Tx) map[string]int {
	out := map[string]int{}
	for e := range tx.Entities() {
		if ent, ok := e.(*entity.Ent); ok {
			if ib, ok := ent.Behaviour().(*entity.ItemBehaviour); ok {
				out[itemKey(ib.Item().Item())] += ib.Item().Count()
			}
		}
	}
	return out
}

// TestWitchWater: spores turn dirt into mycelium, an oak barrel of water on
// it turns into witch water after the delay (not elsewhere), the mixings make
// their blocks and empty the barrel, and a bucket takes the witch water.
func TestWitchWater(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		pos := cube.Pos{4, IslandY + 2, 4}
		under := pos.Side(cube.FaceDown)
		tx.SetBlock(under, block.Dirt{}, nil)
		ctx := &item.UseContext{}
		if !(MyceliumSpores{}).UseOnBlock(under, cube.FaceUp, mgl64.Vec3{}, tx, p, ctx) || ctx.CountSub != 1 {
			t.Fatal("spores were not used on dirt")
		}
		if _, ok := tx.Block(under).(block.Mycelium); !ok {
			t.Fatalf("spores made %T", tx.Block(under))
		}
		barrel := Barrel{Wood: "oak"}
		tx.SetBlock(pos, barrel, nil)
		if s.state.Barrels == nil {
			s.state.Barrels = map[string]*BarrelRecord{}
		}
		rec := &BarrelRecord{Count: matterCapacity, Water: true}
		s.state.Barrels[cropKey(pos)] = rec
		s.tickBarrels(tx)
		if rec.WitchAt == 0 || rec.Witch {
			t.Fatalf("water on mycelium did not start turning: %+v", rec)
		}
		rec.WitchAt = -1 // the time is up
		s.tickBarrels(tx)
		if !rec.Witch || rec.Water || tx.Block(pos).(Barrel).Stage != barrelWitch {
			t.Fatalf("no witch water after the delay: %+v", rec)
		}
		// witch water + sand → soul sand
		ctx = &item.UseContext{}
		if !s.useBarrel(p, tx, pos, tx.Block(pos).(Barrel), item.NewStack(block.Sand{}, 4), ctx) || ctx.CountSub != 1 {
			t.Fatal("sand was not mixed")
		}
		if s.barrelAt(pos) != nil || droppedKeys(tx)[itemKey(block.SoulSand{})] != 1 {
			t.Fatalf("no soul sand, barrel %+v", s.barrelAt(pos))
		}
		// witch water + lava bucket → netherrack, the bucket comes back empty
		s.state.Barrels[cropKey(pos)] = &BarrelRecord{Count: matterCapacity, Witch: true}
		p.SetHeldItems(item.NewStack(lavaBucket, 1), item.Stack{})
		if !s.useBarrel(p, tx, pos, tx.Block(pos).(Barrel), item.NewStack(lavaBucket, 1), &item.UseContext{}) {
			t.Fatal("lava was not mixed")
		}
		if held, _ := p.HeldItems(); itemKey(held.Item()) != itemKey(item.Bucket{}) || droppedKeys(tx)[itemKey(block.Netherrack{})] != 1 {
			t.Fatalf("lava mixing left %v in the hand", held)
		}
		// an empty bucket takes the witch water
		s.state.Barrels[cropKey(pos)] = &BarrelRecord{Count: matterCapacity, Witch: true}
		s.useBarrel(p, tx, pos, tx.Block(pos).(Barrel), item.NewStack(item.Bucket{}, 1), &item.UseContext{})
		if held, _ := p.HeldItems(); itemKey(held.Item()) != itemKey(WitchWaterBucket{}) || s.barrelAt(pos) != nil {
			t.Fatalf("the bucket took %v", held)
		}
		// lava + witch water bucket → blackstone, lava + glowstone → end stone
		stone := Barrel{Wood: stoneWood}
		tx.SetBlock(pos, stone, nil)
		s.state.Barrels[cropKey(pos)] = &BarrelRecord{Count: stoneBarrelCapacity, Lava: true}
		s.useBarrel(p, tx, pos, stone, item.NewStack(WitchWaterBucket{}, 1), &item.UseContext{})
		s.state.Barrels[cropKey(pos)] = &BarrelRecord{Count: stoneBarrelCapacity, Lava: true}
		s.useBarrel(p, tx, pos, stone, item.NewStack(item.GlowstoneDust{}, 1), &item.UseContext{})
		got := droppedKeys(tx)
		if got[itemKey(block.Blackstone{})] != 1 || got[itemKey(block.EndStone{})] != 1 {
			t.Fatalf("lava mixings dropped %v", got)
		}
		// water off mycelium stays water
		tx.SetBlock(pos, barrel, nil)
		tx.SetBlock(under, block.Dirt{}, nil)
		rec = &BarrelRecord{Count: matterCapacity, Water: true}
		s.state.Barrels[cropKey(pos)] = rec
		s.tickBarrels(tx)
		if rec.WitchAt != 0 || rec.Witch {
			t.Fatal("water off mycelium turns into witch water")
		}
	})
}

// TestMyceliumStaysUnderBarrel: a barrel lets light through and is not a
// solid block, so the mycelium or grass under it does not turn to dirt
// (spreadDirt reverts them without light or under model.Solid).
func TestMyceliumStaysUnderBarrel(t *testing.T) {
	var b world.Block = Barrel{Wood: "oak"}
	if d, ok := b.(block.LightDiffuser); !ok || d.LightDiffusionLevel() > 2 {
		t.Fatal("a barrel blocks light")
	}
	if _, solid := b.Model().(model.Solid); solid {
		t.Fatal("a barrel is a solid block")
	}
}

// TestBarrelLiquidIcon: the WAILA panel shows the liquid of a barrel as its
// icon: water, witch water or lava, none while it holds no liquid.
func TestBarrelLiquidIcon(t *testing.T) {
	s := testSession(t)
	pos := cube.Pos{1, 2, 3}
	s.state.Barrels = map[string]*BarrelRecord{}
	if s.barrelLiquid(pos) != nil {
		t.Fatal("an empty barrel shows a liquid")
	}
	for kind, rec := range map[string]*BarrelRecord{
		liquidWater:      {Count: matterCapacity, Water: true},
		liquidWitchWater: {Count: matterCapacity, Witch: true},
		liquidLava:       {Count: stoneBarrelCapacity, Lava: true},
	} {
		s.state.Barrels[cropKey(pos)] = rec
		line := s.barrelLiquid(pos)
		if line == nil || itemKey(line.Item.Item()) != itemKey(LiquidIcon{Kind: kind}) {
			t.Errorf("%s barrel shows %v", kind, line)
		}
	}
	s.state.Barrels[cropKey(pos)] = &BarrelRecord{Count: 10}
	if s.barrelLiquid(pos) != nil {
		t.Fatal("a filling barrel shows a liquid")
	}
}
