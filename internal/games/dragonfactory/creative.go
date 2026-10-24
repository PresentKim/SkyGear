package dragonfactory

import (
	"sync"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/creative"
	"github.com/df-mc/dragonfly/server/world"
)

// Creative inventory entries of the game's own items and blocks. The client
// builds its recipe book from the creative inventory the server sends: a
// recipe whose result is not in it is sent but never listed (the hammer, the
// sieve and the other recipes of the game's items were missing from the
// crafting list, 14th playtest). So every item and block the game makes is
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
		add(construction, Dust{})
		for _, k := range crushedKinds {
			add(construction, CrushedBlock{Kind: k})
		}
		for _, id := range hopperOrder {
			add(construction, newFlowHopper(id, cube.FaceDown))
		}
		add(equipment, Hammer{Stone: true})
		add(equipment, Hammer{})
		add(items, WoodenBucket{})
		add(items, WoodenBucket{Water: true})
		add(equipment, DrawerKey{})
		add(items, DrawerUpgrade{Kind: UpgradeCapacity})
		add(items, DrawerUpgrade{Kind: UpgradeVoid})
		add(items, BarkFiber{})
		add(items, GrassSeeds{})
		add(items, MyceliumSpores{})
		add(items, WitchWaterBucket{})
		add(items, ConduitBinderMix{})
		add(items, ConduitBinder{})
		for _, t := range meshTiers {
			add(items, Mesh{Tier: t.tier})
		}
		for _, k := range oreKinds {
			add(items, OreChunk{Ore: k.ore})
		}
		for _, m := range newOreMetals {
			add(construction, OreBlock{Metal: m})
			add(items, RawOre{Metal: m})
		}
		for _, k := range pebbleKinds {
			add(items, Pebble{Kind: k.kind})
		}
		for _, c := range petalColours {
			add(items, Petal{Colour: c})
		}
		for _, sd := range plantSeeds {
			add(items, PlantSeed{Seed: sd})
		}
		for _, m := range mobKinds {
			add(items, Essence{Mob: m})
		}
		for _, m := range dustMetals {
			add(items, MetalDust{Metal: m})
		}
		for _, m := range ingotMetals {
			add(items, Ingot{Metal: m})
		}
		for _, a := range tierAlloys {
			add(items, TierAlloy{Kind: a})
		}
		for l := 1; l <= 4; l++ {
			add(items, Circuit{Level: l})
		}
		for l := 1; l <= maxTier; l++ {
			add(items, TierKit{Level: l})
		}
		add(equipment, ConfigCopier{})
	})
}
