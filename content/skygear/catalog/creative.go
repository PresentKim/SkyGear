package catalog

import (
	"sync"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/creative"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/conduit"
	"github.com/presentkim/skygear/content/skygear/fairy"
	"github.com/presentkim/skygear/content/skygear/farm"
	"github.com/presentkim/skygear/content/skygear/hopper"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/storage"
)

// Creative inventory entries of the game's own items and blocks. The client
// builds its recipe book from the creative inventory: a recipe whose result
// is not in it is never listed, so every item and block the game makes is
// registered here, in anonymous groups of the fitting tab.
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
		add := func(c creative.Category, it world.Item) {
			creative.RegisterItem(creative.Item{Stack: item.NewStack(it, 1), Group: groupName(c)})
		}
		add(construction, progress.ResearchTable{})
		add(construction, storage.Drawer{})
		for _, w := range farm.BarrelWoods {
			add(construction, farm.Barrel{Wood: w})
		}
		for t := 1; t <= conduit.Tiers; t++ {
			add(construction, conduit.EnergyConduit{Tier: t})
			add(construction, conduit.ItemConduit{Tier: t})
			add(construction, conduit.FluidConduit{Tier: t})
		}
		for _, k := range machine.Order {
			if k != machine.KindSellBox {
				add(construction, machine.Machine{Kind: k})
			}
		}
		add(construction, sgitem.Dust{})
		for _, k := range sgitem.CrushedKinds {
			add(construction, sgitem.CrushedBlock{Kind: k})
		}
		for _, id := range hopper.HopperOrder {
			add(construction, hopper.NewFlowHopper(id, cube.FaceDown))
		}
		add(equipment, sgitem.Hammer{Stone: true})
		add(equipment, sgitem.Hammer{})
		add(items, sgitem.WoodenBucket{})
		add(items, sgitem.WoodenBucket{Water: true})
		add(equipment, sgitem.DrawerKey{})
		add(items, sgitem.DrawerUpgrade{Kind: sgitem.UpgradeCapacity})
		add(items, sgitem.DrawerUpgrade{Kind: sgitem.UpgradeVoid})
		add(items, sgitem.BarkFiber{})
		add(items, sgitem.FairyDust{})
		add(items, sgitem.FairyEmerald{})
		add(equipment, fairy.Board{})
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
		for _, sd := range farm.PlantSeeds {
			add(items, farm.PlantSeed{Seed: sd})
		}
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
	})
}
