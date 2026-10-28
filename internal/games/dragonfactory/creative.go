package dragonfactory

import (
	"sync"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/creative"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/dfitem"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/hopper"
)

// Creative inventory entries of the game's own items and blocks. The client
// builds its recipe book from the creative inventory: a recipe whose result
// is not in it is never listed, so every item and block the game makes is
// registered here, in anonymous groups of the fitting tab.
var registerCreativeOnce sync.Once

// registerCreativeItems adds the game's items to the creative inventory. It
// runs when the game starts, after every item is registered.
func registerCreativeItems() {
	registerCreativeOnce.Do(func() {
		construction, equipment, items := creative.ConstructionCategory(), creative.EquipmentCategory(), creative.ItemsCategory()
		groupName := func(c creative.Category) string {
			switch c {
			case construction:
				return "anon_df_construction"
			case equipment:
				return "anon_df_equipment"
			}
			return "anon_df_items"
		}
		for _, c := range []creative.Category{construction, equipment, items} {
			// No icon makes the group anonymous: its items are not grouped.
			creative.RegisterGroup(creative.Group{Category: c, Name: groupName(c)})
		}
		add := func(c creative.Category, it world.Item) {
			creative.RegisterItem(creative.Item{Stack: item.NewStack(it, 1), Group: groupName(c)})
		}
		add(construction, ResearchTable{})
		add(construction, Drawer{})
		for _, w := range barrelWoods {
			add(construction, Barrel{Wood: w})
		}
		for t := 1; t <= conduitTiers; t++ {
			add(construction, EnergyConduit{Tier: t})
			add(construction, ItemConduit{Tier: t})
			add(construction, FluidConduit{Tier: t})
		}
		for _, k := range machineOrder {
			if k != KindSellBox {
				add(construction, Machine{Kind: k})
			}
		}
		add(construction, dfitem.Dust{})
		for _, k := range dfitem.CrushedKinds {
			add(construction, dfitem.CrushedBlock{Kind: k})
		}
		for _, id := range hopper.HopperOrder {
			add(construction, hopper.NewFlowHopper(id, cube.FaceDown))
		}
		add(equipment, dfitem.Hammer{Stone: true})
		add(equipment, dfitem.Hammer{})
		add(items, dfitem.WoodenBucket{})
		add(items, dfitem.WoodenBucket{Water: true})
		add(equipment, dfitem.DrawerKey{})
		add(items, dfitem.DrawerUpgrade{Kind: dfitem.UpgradeCapacity})
		add(items, dfitem.DrawerUpgrade{Kind: dfitem.UpgradeVoid})
		add(items, dfitem.BarkFiber{})
		add(items, dfitem.FairyDust{})
		add(items, dfitem.FairyEmerald{})
		add(equipment, FairyBoard{})
		add(items, dfitem.FairyBoneMeal{})
		add(items, dfitem.GrassSeeds{})
		add(items, dfitem.MyceliumSpores{})
		add(items, dfitem.WitchWaterBucket{})
		add(items, dfitem.ConduitBinderMix{})
		add(items, dfitem.ConduitBinder{})
		for _, t := range dfitem.MeshTiers {
			add(items, dfitem.Mesh{Tier: t.Tier})
		}
		for _, k := range dfitem.OreKinds {
			add(items, dfitem.OreChunk{Ore: k.Ore})
		}
		for _, m := range dfitem.NewOreMetals {
			add(construction, dfitem.OreBlock{Metal: m})
			add(items, dfitem.RawOre{Metal: m})
		}
		for _, k := range dfitem.PebbleKinds {
			add(items, dfitem.Pebble{Kind: k.Kind})
		}
		for _, c := range dfitem.PetalColours {
			add(items, dfitem.Petal{Colour: c})
		}
		for _, sd := range plantSeeds {
			add(items, PlantSeed{Seed: sd})
		}
		for _, m := range dfitem.MobKinds {
			add(items, dfitem.Essence{Mob: m})
		}
		for _, m := range dfitem.DustMetals {
			add(items, dfitem.MetalDust{Metal: m})
		}
		for _, m := range dfitem.IngotMetals {
			add(items, dfitem.Ingot{Metal: m})
		}
		for _, a := range dfitem.TierAlloys {
			add(items, dfitem.TierAlloy{Kind: a})
		}
		for l := 1; l <= 4; l++ {
			add(items, dfitem.Circuit{Level: l})
		}
		for l := 1; l <= maxTier; l++ {
			add(items, dfitem.TierKit{Level: l})
		}
		add(equipment, dfitem.ConfigCopier{})
	})
}
