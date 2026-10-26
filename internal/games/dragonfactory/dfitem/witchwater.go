package dfitem

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/common/i18n"
)

// MyceliumSpores turn the dirt they are used on into mycelium.
type MyceliumSpores struct{}

func (MyceliumSpores) EncodeItem() (string, int16) { return "dragonfactory:mycelium_spores", 0 }

func (MyceliumSpores) Name() string { return i18n.T(i18n.Default, "name.MyceliumSpores") }

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

func (WitchWaterBucket) Name() string { return i18n.T(i18n.Default, "name.WitchWaterBucket") }

func (WitchWaterBucket) Category() category.Category { return category.Items() }

func (WitchWaterBucket) MaxCount() int { return 1 }
