package dragonfactory

import (
	"strings"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

// Form button icons: texture paths the client already has. Custom items get
// their texture from Dragonfly's generated pack (textures/items/<name>);
// vanilla items use the vanilla texture paths below.

const (
	iconMission = "textures/items/book_normal"
	iconShop    = "textures/items/emerald"
	iconSell    = "textures/items/gold_ingot"
	iconCheque  = "textures/items/paper"
	iconCores   = "textures/items/diamond"
	iconIsland  = "textures/blocks/grass_side_carried"
	iconLost    = "textures/blocks/chest_front"
	iconMenu    = "textures/items/nether_star"
	iconLobby   = "textures/items/compass_item"
	iconBack    = "textures/items/arrow"
	iconLocked  = "textures/blocks/barrier"
	iconAll     = "textures/blocks/hopper_top"
	iconTitle   = "textures/items/name_tag"
)

// vanillaIcons maps item keys (itemKey) to vanilla texture paths.
var vanillaIcons = map[string]string{
	"minecraft:cobblestone:0":      "textures/blocks/cobblestone",
	"minecraft:dirt:0":             "textures/blocks/dirt",
	"minecraft:sand:0":             "textures/blocks/sand",
	"minecraft:gravel:0":           "textures/blocks/gravel",
	"minecraft:stone:0":            "textures/blocks/stone",
	"minecraft:oak_log:0":          "textures/blocks/log_oak",
	"minecraft:glass:0":            "textures/blocks/glass",
	"minecraft:charcoal:0":         "textures/items/charcoal",
	"minecraft:coal:0":             "textures/items/coal",
	"minecraft:flint:0":            "textures/items/flint",
	"minecraft:clay_ball:0":        "textures/items/clay_ball",
	"minecraft:iron_nugget:0":      "textures/items/iron_nugget",
	"minecraft:gold_nugget:0":      "textures/items/gold_nugget",
	"minecraft:clay:0":             "textures/blocks/clay",
	"minecraft:raw_iron:0":         "textures/items/raw_iron",
	"minecraft:raw_copper:0":       "textures/items/raw_copper",
	"minecraft:raw_gold:0":         "textures/items/raw_gold",
	"minecraft:iron_ingot:0":       "textures/items/iron_ingot",
	"minecraft:copper_ingot:0":     "textures/items/copper_ingot",
	"minecraft:gold_ingot:0":       "textures/items/gold_ingot",
	"minecraft:torch:0":            "textures/blocks/torch_on",
	"minecraft:hopper:0":           "textures/items/hopper",
	"minecraft:water_bucket:0":     "textures/items/bucket_water",
	"minecraft:iron_pickaxe:0":     "textures/items/iron_pickaxe",
	"minecraft:stone_hoe:0":        "textures/items/stone_hoe",
	"minecraft:carrot:0":           "textures/items/carrot",
	"minecraft:potato:0":           "textures/items/potato",
	"minecraft:beetroot:0":         "textures/items/beetroot",
	"minecraft:melon_slice:0":      "textures/items/melon",
	"minecraft:wheat_seeds:0":      "textures/items/seeds_wheat",
	"minecraft:diamond:0":          "textures/items/diamond",
	"minecraft:apple:0":            "textures/items/apple",
	"minecraft:pumpkin:0":          "textures/blocks/pumpkin_face_off",
	"minecraft:melon_block:0":      "textures/blocks/melon_side",
	"minecraft:hay_block:0":        "textures/blocks/hay_block_side",
	"minecraft:dried_kelp_block:0": "textures/blocks/dried_kelp_side_a",
	"minecraft:mud:0":              "textures/blocks/mud",
	"minecraft:wheat:0":            "textures/items/wheat",
	"minecraft:grass_block:0":      "textures/blocks/grass_side_carried",
	"minecraft:birch_log:0":        "textures/blocks/log_birch",
	"minecraft:spruce_log:0":       "textures/blocks/log_spruce",
	"minecraft:oak_planks:0":       "textures/blocks/planks_oak",
	"minecraft:coal_ore:0":         "textures/blocks/coal_ore",
	"minecraft:iron_ore:0":         "textures/blocks/iron_ore",
	"minecraft:copper_ore:0":       "textures/blocks/copper_ore",
	"minecraft:gold_ore:0":         "textures/blocks/gold_ore",
	"minecraft:coal_block:0":       "textures/blocks/coal_block",
	"minecraft:iron_block:0":       "textures/blocks/iron_block",
	"minecraft:copper_block:0":     "textures/blocks/copper_block",
	"minecraft:gold_block:0":       "textures/blocks/gold_block",
	"minecraft:beetroot_seeds:0":   "textures/items/seeds_beetroot",
	"minecraft:pumpkin_seeds:0":    "textures/items/seeds_pumpkin",
	"minecraft:melon_seeds:0":      "textures/items/seeds_melon",
	"minecraft:farmland:0":         "textures/blocks/farmland_dry",
	"minecraft:oak_sapling:0":      "textures/blocks/sapling_oak",
	"minecraft:chest:0":            "textures/blocks/chest_front",
	"minecraft:furnace:0":          "textures/blocks/furnace_front_off",
	"minecraft:barrel:0":           "textures/blocks/barrel_side",
	"minecraft:bucket:0":           "textures/items/bucket_empty",
	"minecraft:paper:0":            "textures/items/paper",
	"minecraft:wooden_pickaxe:0":   "textures/items/wood_pickaxe",
	"minecraft:stone_pickaxe:0":    "textures/items/stone_pickaxe",
	"minecraft:wooden_axe:0":       "textures/items/wood_axe",
	"minecraft:stone_axe:0":        "textures/items/stone_axe",
	"minecraft:wooden_hoe:0":       "textures/items/wood_hoe",
	"minecraft:wooden_shovel:0":    "textures/items/wood_shovel",
	"minecraft:stone_shovel:0":     "textures/items/stone_shovel",
	"minecraft:nether_star:0":      "textures/items/nether_star",
}

// iconOf returns the texture path of an item, or "" if it has none known.
func iconOf(it item.Stack) string {
	if it.Empty() {
		return ""
	}
	if c, ok := it.Item().(world.CustomItem); ok {
		id, _ := c.EncodeItem()
		return "textures/items/" + id[strings.Index(id, ":")+1:]
	}
	return vanillaIcons[itemKey(it.Item())]
}
