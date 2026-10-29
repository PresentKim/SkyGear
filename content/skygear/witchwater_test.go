package skygear

import (
	"testing"

	"github.com/presentkim/skygear/content/skygear/farm"

	"github.com/presentkim/skygear/content/skygear/machine"

	"github.com/presentkim/skygear/content/skygear/island"

	"github.com/presentkim/skygear/common/i18n"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// droppedKeys are the item keys of the item entities lying in tx.
func droppedKeys(tx *world.Tx) map[string]int {
	out := map[string]int{}
	for e := range tx.Entities() {
		if ent, ok := e.(*entity.Ent); ok {
			if ib, ok := ent.Behaviour().(*entity.ItemBehaviour); ok {
				out[naming.ItemKey(ib.Item().Item())] += ib.Item().Count()
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
		pos := cube.Pos{4, island.IslandY + 2, 4}
		under := pos.Side(cube.FaceDown)
		tx.SetBlock(under, block.Dirt{}, nil)
		ctx := &item.UseContext{}
		if !(sgitem.MyceliumSpores{}).UseOnBlock(under, cube.FaceUp, mgl64.Vec3{}, tx, p, ctx) || ctx.CountSub != 1 {
			t.Fatal("spores were not used on dirt")
		}
		if _, ok := tx.Block(under).(block.Mycelium); !ok {
			t.Fatalf("spores made %T", tx.Block(under))
		}
		barrel := farm.Barrel{Wood: "oak"}
		tx.SetBlock(pos, barrel, nil)
		if farm.StateOf(s.Session).Barrels == nil {
			farm.StateOf(s.Session).Barrels = map[string]*farm.BarrelRecord{}
		}
		rec := &farm.BarrelRecord{Count: farm.MatterCapacity, Water: true}
		farm.StateOf(s.Session).Barrels[farm.CropKey(pos)] = rec
		farm.TickBarrels(s.Session, tx)
		if rec.WitchAt == 0 || rec.Witch {
			t.Fatalf("water on mycelium did not start turning: %+v", rec)
		}
		rec.WitchAt = -1 // the time is up
		farm.TickBarrels(s.Session, tx)
		if !rec.Witch || rec.Water || tx.Block(pos).(farm.Barrel).Stage != farm.BarrelWitch {
			t.Fatalf("no witch water after the delay: %+v", rec)
		}
		// witch water + sand → soul sand
		ctx = &item.UseContext{}
		if !farm.UseBarrel(s.Session, p, tx, pos, tx.Block(pos).(farm.Barrel), item.NewStack(block.Sand{}, 4), ctx) || ctx.CountSub != 1 {
			t.Fatal("sand was not mixed")
		}
		if farm.BarrelAt(s.Session, pos) != nil || droppedKeys(tx)[naming.ItemKey(block.SoulSand{})] != 1 {
			t.Fatalf("no soul sand, barrel %+v", farm.BarrelAt(s.Session, pos))
		}
		// witch water + lava bucket → netherrack, the bucket comes back empty
		farm.StateOf(s.Session).Barrels[farm.CropKey(pos)] = &farm.BarrelRecord{Count: farm.MatterCapacity, Witch: true}
		p.SetHeldItems(item.NewStack(farm.LavaBucket, 1), item.Stack{})
		if !farm.UseBarrel(s.Session, p, tx, pos, tx.Block(pos).(farm.Barrel), item.NewStack(farm.LavaBucket, 1), &item.UseContext{}) {
			t.Fatal("lava was not mixed")
		}
		if held, _ := p.HeldItems(); naming.ItemKey(held.Item()) != naming.ItemKey(item.Bucket{}) || droppedKeys(tx)[naming.ItemKey(block.Netherrack{})] != 1 {
			t.Fatalf("lava mixing left %v in the hand", held)
		}
		// an empty bucket takes the witch water
		farm.StateOf(s.Session).Barrels[farm.CropKey(pos)] = &farm.BarrelRecord{Count: farm.MatterCapacity, Witch: true}
		farm.UseBarrel(s.Session, p, tx, pos, tx.Block(pos).(farm.Barrel), item.NewStack(item.Bucket{}, 1), &item.UseContext{})
		if held, _ := p.HeldItems(); naming.ItemKey(held.Item()) != naming.ItemKey(sgitem.WitchWaterBucket{}) || farm.BarrelAt(s.Session, pos) != nil {
			t.Fatalf("the bucket took %v", held)
		}
		// lava + witch water bucket → blackstone, lava + glowstone → end stone
		stone := farm.Barrel{Wood: farm.StoneWood}
		tx.SetBlock(pos, stone, nil)
		farm.StateOf(s.Session).Barrels[farm.CropKey(pos)] = &farm.BarrelRecord{LavaMB: machine.BucketMB}
		farm.UseBarrel(s.Session, p, tx, pos, stone, item.NewStack(sgitem.WitchWaterBucket{}, 1), &item.UseContext{})
		farm.StateOf(s.Session).Barrels[farm.CropKey(pos)] = &farm.BarrelRecord{LavaMB: machine.BucketMB}
		farm.UseBarrel(s.Session, p, tx, pos, stone, item.NewStack(item.GlowstoneDust{}, 1), &item.UseContext{})
		got := droppedKeys(tx)
		if got[naming.ItemKey(block.Blackstone{})] != 1 || got[naming.ItemKey(block.EndStone{})] != 1 {
			t.Fatalf("lava mixings dropped %v", got)
		}
		// water off mycelium stays water
		tx.SetBlock(pos, barrel, nil)
		tx.SetBlock(under, block.Dirt{}, nil)
		rec = &farm.BarrelRecord{Count: farm.MatterCapacity, Water: true}
		farm.StateOf(s.Session).Barrels[farm.CropKey(pos)] = rec
		farm.TickBarrels(s.Session, tx)
		if rec.WitchAt != 0 || rec.Witch {
			t.Fatal("water off mycelium turns into witch water")
		}
	})
}

// TestMyceliumStaysUnderBarrel: a barrel lets light through and is not a
// solid block, so the mycelium or grass under it does not turn to dirt
// (spreadDirt reverts them without light or under model.Solid).
func TestMyceliumStaysUnderBarrel(t *testing.T) {
	var b world.Block = farm.Barrel{Wood: "oak"}
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
	farm.StateOf(s.Session).Barrels = map[string]*farm.BarrelRecord{}
	if farm.BarrelLiquid(s.Session, i18n.Default, pos) != nil {
		t.Fatal("an empty barrel shows a liquid")
	}
	for kind, rec := range map[string]*farm.BarrelRecord{
		sgitem.LiquidWater:      {Count: farm.MatterCapacity, Water: true},
		sgitem.LiquidWitchWater: {Count: farm.MatterCapacity, Witch: true},
		sgitem.LiquidLava:       {LavaMB: machine.BucketMB},
	} {
		farm.StateOf(s.Session).Barrels[farm.CropKey(pos)] = rec
		line := farm.BarrelLiquid(s.Session, i18n.Default, pos)
		if line == nil || naming.ItemKey(line.Item.Item()) != naming.ItemKey(sgitem.LiquidIcon{Kind: kind}) {
			t.Errorf("%s barrel shows %v", kind, line)
		}
	}
	farm.StateOf(s.Session).Barrels[farm.CropKey(pos)] = &farm.BarrelRecord{Count: 10}
	if farm.BarrelLiquid(s.Session, i18n.Default, pos) != nil {
		t.Fatal("a filling barrel shows a liquid")
	}
}
