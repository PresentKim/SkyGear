package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/internal/fluidnet"
	"github.com/presentkim/dragonfactory/internal/i18n"
)

// WoodenBucket is the early bucket (docs/design/02-dragonfactory.md §5): it
// is made of planks as soon as the barrel is researched and holds water only
// (no lava). It takes the water of a barrel, tank or world water source and
// pours it like the iron bucket does.
type WoodenBucket struct{ Water bool }

func (b WoodenBucket) EncodeItem() (string, int16) {
	if b.Water {
		return "dragonfactory:wooden_water_bucket", 0
	}
	return "dragonfactory:wooden_bucket", 0
}

func (b WoodenBucket) Name() string {
	if b.Water {
		return i18n.T(i18n.Default, "name.WoodenWaterBucket")
	}
	return i18n.T(i18n.Default, "name.WoodenBucket")
}

func (WoodenBucket) Category() category.Category { return category.Items() }

// MaxCount is 16 for an empty bucket and 1 for a full one, like the iron one.
func (b WoodenBucket) MaxCount() int {
	if b.Water {
		return 1
	}
	return 16
}

// UseOnBlock fills an empty bucket from a water source and pours a full one.
func (b WoodenBucket) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, _ item.User, ctx *item.UseContext) bool {
	if !b.Water {
		liq, ok := tx.Liquid(pos)
		if !ok {
			return false
		}
		if _, water := liq.(block.Water); !water || liq.LiquidDepth() != 8 || liq.LiquidFalling() {
			return false
		}
		tx.SetLiquid(pos, nil)
		tx.PlaySound(pos.Vec3Centre(), sound.BucketFill{Liquid: liq})
		ctx.NewItem = item.NewStack(WoodenBucket{Water: true}, 1)
		ctx.NewItemSurvivalOnly = true
		ctx.SubtractFromCount(1)
		return true
	}
	water := block.Water{}.WithDepth(8, false)
	switch {
	case liquidFits(tx.Block(pos), water):
		tx.SetLiquid(pos, water)
	case liquidFits(tx.Block(pos.Side(face)), water):
		tx.SetLiquid(pos.Side(face), water)
	default:
		return false
	}
	tx.PlaySound(pos.Vec3Centre(), sound.BucketEmpty{Liquid: block.Water{}})
	ctx.NewItem = item.NewStack(WoodenBucket{}, 1)
	ctx.NewItemSurvivalOnly = true
	ctx.SubtractFromCount(1)
	return true
}

// liquidFits reports whether liquid can be placed in the block b: it
// displaces the liquid or b is replaced by it.
func liquidFits(b world.Block, liq world.Liquid) bool {
	if d, ok := b.(world.LiquidDisplacer); ok && d.CanDisplace(liq) {
		return true
	}
	if r, ok := b.(interface{ ReplaceableBy(world.Block) bool }); ok {
		return r.ReplaceableBy(liq)
	}
	return false
}

// isWaterBucket reports whether st is a bucket of water, of iron or of wood.
func isWaterBucket(st item.Stack) bool {
	switch b := st.Item().(type) {
	case WoodenBucket:
		return b.Water
	case item.Bucket:
		return fluidOfBucket(b) == fluidnet.Water
	}
	return false
}

func init() {
	world.RegisterItem(WoodenBucket{})
	world.RegisterItem(WoodenBucket{Water: true})
}
