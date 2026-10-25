package dragonfactory

import (
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/common/ui"
)

// Witch water and the barrel mixing of Ex Deorum, as ATM10: To the Sky uses
// them (docs/design/16-ore-balance.md §2.1): mycelium spores sifted from dirt
// turn dirt into mycelium; an oak barrel of water standing on mycelium turns
// into witch water after witchWaterDelay; and barrels of witch water and lava
// make the blocks of the Nether and the End.
//   - witch water + lava bucket (or a lava source on top) → netherrack
//   - witch water + sand → soul sand
//   - lava + witch water bucket → blackstone
//   - lava + glowstone dust → end stone
//
// The result drops on top of the barrel, which is empty again.

// witchWaterDelay is how long water on mycelium takes to turn (Ex Deorum:
// 1,700 ticks).
const witchWaterDelay = 1700 * time.Second / 20

// MyceliumSpores turn the dirt they are used on into mycelium.
type MyceliumSpores struct{}

func (MyceliumSpores) EncodeItem() (string, int16) { return "dragonfactory:mycelium_spores", 0 }
func (MyceliumSpores) Name() string                { return i18n.T(i18n.Default, "name.MyceliumSpores") }
func (MyceliumSpores) Category() category.Category { return category.Items() }

// UseOnBlock implements item.UsableOnBlock.
func (MyceliumSpores) UseOnBlock(pos cube.Pos, _ cube.Face, _ mgl64.Vec3, tx *world.Tx, _ item.User, ctx *item.UseContext) bool {
	if d, ok := tx.Block(pos).(block.Dirt); !ok || d.Coarse {
		return false
	}
	tx.SetBlock(pos, block.Mycelium{}, nil)
	tx.PlaySound(pos.Vec3Centre(), sound.ItemUseOn{Block: block.Mycelium{}})
	ctx.SubtractFromCount(1)
	return true
}

// WitchWaterBucket is a bucket of witch water, taken from a barrel.
type WitchWaterBucket struct{}

func (WitchWaterBucket) EncodeItem() (string, int16) { return "dragonfactory:witch_water_bucket", 0 }
func (WitchWaterBucket) Name() string                { return i18n.T(i18n.Default, "name.WitchWaterBucket") }
func (WitchWaterBucket) Category() category.Category { return category.Items() }
func (WitchWaterBucket) MaxCount() int               { return 1 }

// LightDiffusionLevel lets light through a barrel, as through a vanilla
// composter: the mycelium or grass under it needs light or it turns to dirt.
func (Barrel) LightDiffusionLevel() uint8 { return 0 }

// tubModel is the model of a barrel: a full block that is not model.Solid, so
// the mycelium or grass under it stays (vanilla keeps them under a barrel).
type tubModel struct{}

func (tubModel) BBox(cube.Pos, world.BlockSource) []cube.BBox {
	return []cube.BBox{cube.Box(0, 0, 0, 1, 1, 1)}
}
func (tubModel) FaceSolid(cube.Pos, cube.Face, world.BlockSource) bool { return true }

// barrelMixRecipe is a mixing: a barrel of witch water (lava) and in make out;
// a bucket in comes back empty.
type barrelMixRecipe struct {
	lava    bool
	in, out world.Item
}

var lavaBucket = item.Bucket{Content: item.LiquidBucketContent(block.Lava{Depth: 8, Still: true})}

// barrelMixes are the mixings of Ex Deorum's barrels.
func barrelMixes() []barrelMixRecipe {
	return []barrelMixRecipe{
		{false, lavaBucket, block.Netherrack{}},
		{false, block.Sand{}, block.SoulSand{}},
		{true, WitchWaterBucket{}, block.Blackstone{}},
		{true, item.GlowstoneDust{}, block.EndStone{}},
	}
}

// barrelMix is what the barrel of rec makes with held in a hand (ok false for
// nothing) and what the hand holds afterwards when the item is a bucket.
func barrelMix(rec *BarrelRecord, held item.Stack) (out world.Item, bucket item.Stack, ok bool) {
	if rec == nil || held.Empty() || !plain(held) {
		return nil, item.Stack{}, false
	}
	for _, m := range barrelMixes() {
		if (m.lava && rec.LavaMB < bucketMB) || (!m.lava && !rec.Witch) || itemKey(held.Item()) != itemKey(m.in) {
			continue
		}
		switch m.in.(type) {
		case item.Bucket, WitchWaterBucket:
			bucket = item.NewStack(item.Bucket{}, 1)
		}
		return m.out, bucket, true
	}
	return nil, item.Stack{}, false
}

// barrelMixCards are the codex cards of the mixings of a barrel of witch
// water (lava).
func (s *session) barrelMixCards(lang string, barrel item.Stack, lava bool) []codexCard {
	liquid := item.NewStack(WitchWaterBucket{}, 1)
	if lava {
		liquid = item.NewStack(lavaBucket, 1)
	}
	var cards []codexCard
	for _, m := range barrelMixes() {
		if m.lava == lava {
			cards = append(cards, machineCard(lang, barrel,
				[]ui.BoardCell{s.codexCell(lang, liquid, ""), s.codexCell(lang, item.NewStack(m.in, 1), "")},
				[]ui.BoardCell{s.codexCell(lang, item.NewStack(m.out, 1), "")}))
		}
	}
	return cards
}

// tickWitchWater turns an oak barrel of water on mycelium into witch water
// once its time is up, and makes netherrack of a barrel of witch water with a
// lava source on top (which the mixing uses up). It reports whether the
// barrel changed.
func (s *session) tickWitchWater(tx *world.Tx, pos cube.Pos, b Barrel, rec *BarrelRecord, now time.Duration) bool {
	if rec.Witch {
		if l, ok := tx.Block(pos.Side(cube.FaceUp)).(block.Lava); ok && l.Depth == 8 && !l.Falling {
			tx.SetBlock(pos.Side(cube.FaceUp), nil, nil)
			s.mixBarrel(tx, pos, b, block.Netherrack{})
			return true
		}
		return false
	}
	if !rec.Water {
		return false
	}
	if _, ok := tx.Block(pos.Side(cube.FaceDown)).(block.Mycelium); !ok {
		changed := rec.WitchAt != 0
		rec.WitchAt = 0
		return changed
	}
	if rec.WitchAt == 0 {
		rec.WitchAt = now + witchWaterDelay
		return true
	}
	if now < rec.WitchAt {
		return false
	}
	rec.Water, rec.Witch, rec.WitchAt = false, true, 0
	s.showBarrel(tx, pos, b)
	tx.PlaySound(pos.Vec3Centre(), sound.BucketFill{Liquid: block.Water{}})
	return true
}

// mixBarrel empties the barrel at pos (uses up a bucket's worth of the lava
// of a stone barrel) and drops out on top.
func (s *session) mixBarrel(tx *world.Tx, pos cube.Pos, b Barrel, out world.Item) {
	rec := s.barrelAt(pos)
	switch {
	case b.Wood == stoneWood && rec != nil:
		// A stone barrel uses up a bucket's worth of its lava.
		rec.LavaMB -= bucketMB
		s.forgetEmptyStone(pos, rec)
	default:
		if rec != nil && rec.Clay > 0 {
			s.dropAt(tx, pos.Side(cube.FaceUp).Vec3Middle(), item.NewStack(block.Clay{}, rec.Clay))
		}
		delete(s.state.Barrels, cropKey(pos))
	}
	s.showBarrel(tx, pos, b)
	s.dropAt(tx, pos.Side(cube.FaceUp).Vec3Middle(), item.NewStack(out, 1))
	tx.PlaySound(pos.Vec3Centre(), sound.Fizz{})
}

func init() {
	world.RegisterItem(MyceliumSpores{})
	world.RegisterItem(WitchWaterBucket{})
}
