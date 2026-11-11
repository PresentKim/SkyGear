package catalog

import (
	"sync"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/creative"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/elevator"
	"github.com/presentkim/skygear/content/skygear/fairy"
	"github.com/presentkim/skygear/content/skygear/hopper"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/netstore"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// Creative inventory entries of the game's own items and blocks. The client
// builds its recipe book from the creative inventory: a recipe whose result
// is not in it is never listed, so every item and block the game makes is
// registered here, in anonymous groups of the fitting tab. Slots order the
// list (core.CreativeItems): a run of items has one slot.
var registerCreativeOnce sync.Once

// RegisterCreativeItems adds the game's items to the creative inventory. It
// runs when the game starts, after every item is registered.
func RegisterCreativeItems() {
	registerCreativeOnce.Do(func() {
		construction, equipment, items := creative.ConstructionCategory(), creative.EquipmentCategory(), creative.ItemsCategory()
		groupName := func(c creative.Category) string {
			switch c {
			case construction:
				return "anon_sg_construction"
			case equipment:
				return "anon_sg_equipment"
			}
			return "anon_sg_items"
		}
		for _, c := range []creative.Category{construction, equipment, items} {
			// No icon makes the group anonymous: its items are not grouped.
			creative.RegisterGroup(creative.Group{Category: c, Name: groupName(c)})
		}
		for _, it := range core.CreativeItems(ownCreative()) {
			creative.RegisterItem(creative.Item{Stack: item.NewStack(it.Item, 1), Group: groupName(it.Category)})
		}
	})
}

// ownCreative are the creative items this package lists, each run with its
// slot.
func ownCreative() []core.CreativeItem {
	construction, equipment, items := creative.ConstructionCategory(), creative.EquipmentCategory(), creative.ItemsCategory()
	var out []core.CreativeItem
	slot := 0
	add := func(c creative.Category, it world.Item) {
		out = append(out, core.CreativeItem{Slot: slot, Category: c, Item: it})
	}

	slot = 50
	for _, k := range machine.Order {
		add(construction, machine.Machine{Kind: k})
	}

	slot = 60
	add(construction, sgitem.Dust{})

	slot = 70
	for _, c := range item.Colours() {
		add(construction, elevator.Elevator{Colour: c})
	}

	slot = 80
	for _, k := range sgitem.CrushedKinds {
		add(construction, sgitem.CrushedBlock{Kind: k})
	}
	for _, c := range sgitem.CompactBlocks() {
		add(construction, c)
	}
	for _, id := range hopper.HopperOrder {
		add(construction, hopper.NewFlowHopper(id, cube.FaceDown))
	}
	add(equipment, sgitem.Hammer{Stone: true})
	add(equipment, sgitem.Hammer{})
	for _, t := range sgitem.CrookTiers {
		add(equipment, sgitem.Crook{Tier: t.Name})
	}

	slot = 100
	for _, t := range sgitem.WandTiers {
		add(equipment, sgitem.NewWand(t))
	}
	add(items, sgitem.WoodenBucket{})
	add(items, sgitem.WoodenBucket{Water: true})
	add(equipment, sgitem.DrawerKey{})
	add(items, sgitem.DrawerUpgrade{Kind: sgitem.UpgradeCapacity})
	add(items, sgitem.DrawerUpgrade{Kind: sgitem.UpgradeVoid})
	add(items, sgitem.BarkFiber{})
	add(items, sgitem.FairyDust{})

	slot = 110
	for t := range netstore.DiskCaps {
		add(items, netstore.Disk{Tier: t})
	}
	add(items, netstore.Pattern{})
	add(items, netstore.Upgrade{})
	add(items, netstore.Upgrade{Stack: true})

	slot = 120
	add(items, sgitem.FairyEmerald{})
	add(items, sgitem.DimensionStone{})
	for _, kind := range append(append([]string(nil), sgitem.RelicWeapons...), sgitem.RelicCharms...) {
		add(equipment, sgitem.Relic{Kind: kind})
	}

	slot = 130
	add(equipment, fairy.Board{})

	slot = 140
	add(items, sgitem.FairyBoneMeal{})
	add(items, sgitem.FairyHourglass{})
	add(items, sgitem.GrassSeeds{})
	add(items, sgitem.MyceliumSpores{})
	add(items, sgitem.WitchWaterBucket{})
	add(items, sgitem.ConduitBinderMix{})
	add(items, sgitem.ConduitBinder{})
	for _, t := range sgitem.MeshTiers {
		add(items, sgitem.Mesh{Tier: t.Tier})
	}
	for _, k := range sgitem.OreKinds {
		add(items, sgitem.OreChunk{Ore: k.Ore})
	}
	for _, m := range sgitem.NewOreMetals {
		add(construction, sgitem.OreBlock{Metal: m})
		add(items, sgitem.RawOre{Metal: m})
	}
	for _, k := range sgitem.PebbleKinds {
		add(items, sgitem.Pebble{Kind: k.Kind})
	}
	for _, c := range sgitem.PetalColours {
		add(items, sgitem.Petal{Colour: c})
	}

	slot = 160
	for _, m := range sgitem.MobKinds {
		add(items, sgitem.Essence{Mob: m})
	}
	for _, m := range sgitem.DustMetals {
		add(items, sgitem.MetalDust{Metal: m})
	}
	for _, m := range sgitem.IngotMetals {
		add(items, sgitem.Ingot{Metal: m})
	}
	for _, a := range sgitem.TierAlloys {
		add(items, sgitem.TierAlloy{Kind: a})
	}
	for l := 1; l <= 4; l++ {
		add(items, sgitem.Circuit{Level: l})
	}
	for l := 1; l <= machine.MaxTier; l++ {
		add(items, sgitem.TierKit{Level: l})
	}
	add(equipment, sgitem.ConfigCopier{})
	return out
}
