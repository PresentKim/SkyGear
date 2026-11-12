package sgitem

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/i18n"
)

// FairyDust is what the fairy leaves: the stuff of the fairy magic.
type FairyDust struct{}

func (FairyDust) EncodeItem() (string, int16) { return "skygear:fairy_dust", 0 }

func (FairyDust) Name() string { return i18n.T(i18n.Default, "name.FairyDust") }

func (FairyDust) Category() category.Category { return category.Items() }

// FairyEmerald is spent by every spell of the fairy board.
type FairyEmerald struct{}

func (FairyEmerald) EncodeItem() (string, int16) { return "skygear:fairy_emerald", 0 }

func (FairyEmerald) Name() string { return i18n.T(i18n.Default, "name.FairyEmerald") }

func (FairyEmerald) Category() category.Category { return category.Items() }

// FairyBoneMeal grows the timed crops and magic plants ahead, which plain
// bone meal does not (fairytime.go).
type FairyBoneMeal struct{}

func (FairyBoneMeal) EncodeItem() (string, int16) { return "skygear:fairy_bone_meal", 0 }

func (FairyBoneMeal) Name() string { return i18n.T(i18n.Default, "name.FairyBoneMeal") }

func (FairyBoneMeal) Category() category.Category { return category.Items() }

// FairyHourglass runs an energy machine faster for a while. It is used on the
// machine: the machine's Activate takes it from a standing player, UseOnBlock
// below from a sneaking one.
type FairyHourglass struct{}

func (FairyHourglass) EncodeItem() (string, int16) { return "skygear:fairy_hourglass", 0 }

func (FairyHourglass) Name() string { return i18n.T(i18n.Default, "name.FairyHourglass") }

func (FairyHourglass) Category() category.Category { return category.Items() }

// HourglassWound is a block a fairy hourglass winds.
type HourglassWound interface {
	// WindHourglass adds the time of one hourglass to the block at pos and
	// reports whether the block took it.
	WindHourglass(pos cube.Pos, tx *world.Tx) bool
}

// UseOnBlock implements item.UsableOnBlock: the hourglass winds the block it
// is used on. Dragonfly calls it when the block's Activate did not take the
// click, which is while the player sneaks.
func (FairyHourglass) UseOnBlock(pos cube.Pos, _ cube.Face, _ mgl64.Vec3, tx *world.Tx, _ item.User, ctx *item.UseContext) bool {
	w, ok := tx.Block(pos).(HourglassWound)
	if !ok || !w.WindHourglass(pos, tx) {
		return false
	}
	ctx.SubtractFromCount(1)
	tx.PlaySound(pos.Vec3Centre(), sound.ItemAdd{})
	return true
}
