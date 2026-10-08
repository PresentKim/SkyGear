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
)

// vanillaIcons maps item keys (itemKey) to vanilla texture paths.
var vanillaIcons = map[string]string{
	"minecraft:cobblestone:0":  "textures/blocks/cobblestone",
	"minecraft:dirt:0":         "textures/blocks/dirt",
	"minecraft:sand:0":         "textures/blocks/sand",
	"minecraft:gravel:0":       "textures/blocks/gravel",
	"minecraft:stone:0":        "textures/blocks/stone",
	"minecraft:oak_log:0":      "textures/blocks/log_oak",
	"minecraft:glass:0":        "textures/blocks/glass",
	"minecraft:charcoal:0":     "textures/items/charcoal",
	"minecraft:coal:0":         "textures/items/coal",
	"minecraft:flint:0":        "textures/items/flint",
	"minecraft:clay_ball:0":    "textures/items/clay_ball",
	"minecraft:iron_nugget:0":  "textures/items/iron_nugget",
	"minecraft:gold_nugget:0":  "textures/items/gold_nugget",
	"minecraft:torch:0":        "textures/blocks/torch_on",
	"minecraft:hopper:0":       "textures/items/hopper",
	"minecraft:water_bucket:0": "textures/items/bucket_water",
	"minecraft:iron_pickaxe:0": "textures/items/iron_pickaxe",
	"minecraft:wheat_seeds:0":  "textures/items/seeds_wheat",
	"minecraft:diamond:0":      "textures/items/diamond",
	"minecraft:apple:0":        "textures/items/apple",
	"minecraft:wheat:0":        "textures/items/wheat",
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
