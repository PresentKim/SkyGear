package dragonfactory

import (
	"github.com/presentkim/dragonfactory/internal/i18n"
	"slices"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// Blocks of removed systems (13th playtest: the dimension core, the main
// mission's delivery box). They stay registered so old worlds and
// inventories load; they do nothing and say so when used. They can be
// mined back as items.

// coreOrder is the registration order of the old core types: the index is
// part of the block hash.
var coreOrder = []string{"earth", "life", "wood", "stone", "dirt", "sand", "gravel", "coal", "clay", "pumpkin", "melon", "hay", "iron", "copper", "kelp", "mud", "gold", "t1", "t2", "t3", "t4"}

// Core is an old dimension core.
type Core struct {
	Type string
}

var coreHash = block.NextHash()

func (c Core) EncodeItem() (string, int16)           { return "dragonfactory:core_" + c.Type, 0 }
func (c Core) EncodeBlock() (string, map[string]any) { return "dragonfactory:core_" + c.Type, nil }
func (c Core) Hash() (uint64, uint64) {
	return coreHash, uint64(slices.Index(coreOrder, c.Type))
}
func (Core) Model() world.BlockModel            { return model.Solid{} }
func (Core) Name() string                       { return i18n.T(i18n.Default, "name.Core") }
func (Core) Category() category.Category        { return category.Items() }
func (Core) Properties() customblock.Properties { return cubeProperties("core_t1") }
func (Core) LightEmissionLevel() uint8          { return 10 }
func (Core) EncodeNBT() map[string]any          { return map[string]any{"id": "DragonFactoryCore"} }
func (c Core) DecodeNBT(map[string]any) any     { return c }
func (c Core) BreakInfo() block.BreakInfo       { return legacyBreakInfo(c) }
func (Core) Activate(_ cube.Pos, _ cube.Face, _ *world.Tx, u item.User, _ *item.UseContext) bool {
	return legacyMessage(u, i18n.T(i18n.Default, "dragonfactory.legacy.Activate.1"))
}

// DeliveryBox is the old main mission's delivery box.
type DeliveryBox struct{}

var deliveryHash = block.NextHash()

func (DeliveryBox) EncodeItem() (string, int16)           { return "dragonfactory:delivery_box", 0 }
func (DeliveryBox) EncodeBlock() (string, map[string]any) { return "dragonfactory:delivery_box", nil }
func (DeliveryBox) Hash() (uint64, uint64)                { return deliveryHash, 0 }
func (DeliveryBox) Model() world.BlockModel               { return model.Solid{} }
func (DeliveryBox) Name() string                          { return i18n.T(i18n.Default, "name.DeliveryBox") }
func (DeliveryBox) Category() category.Category           { return category.Items() }
func (DeliveryBox) Properties() customblock.Properties    { return frontCubeProperties("delivery_box") }
func (d DeliveryBox) BreakInfo() block.BreakInfo          { return legacyBreakInfo(d) }
func (DeliveryBox) Activate(_ cube.Pos, _ cube.Face, _ *world.Tx, u item.User, _ *item.UseContext) bool {
	return legacyMessage(u, i18n.T(i18n.Default, "dragonfactory.legacy.Activate.2"))
}

func legacyBreakInfo(it world.Item) block.BreakInfo {
	return block.BreakInfo{
		Hardness: 1, BlastResistance: 2,
		Harvestable: func(item.Tool) bool { return true },
		Effective:   customBlockEffective,
		Drops:       func(item.Tool, []item.Enchantment) []item.Stack { return []item.Stack{item.NewStack(it, 1)} },
	}
}

func legacyMessage(u item.User, msg string) bool {
	if p, ok := u.(*player.Player); ok {
		p.Message(msg)
	}
	return true
}

func init() {
	for _, id := range coreOrder {
		world.RegisterBlock(Core{Type: id})
		world.RegisterItem(Core{Type: id})
	}
	world.RegisterBlock(DeliveryBox{})
	world.RegisterItem(DeliveryBox{})
}
