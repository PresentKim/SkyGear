package farm

import "github.com/df-mc/dragonfly/server/world"

// Prices are the sell prices per item, keyed by item identifier.
var Prices = map[string]int64{
	"minecraft:wheat":                  4,
	"minecraft:wheat_seeds":            1,
	"minecraft:carrot":                 4,
	"minecraft:potato":                 4,
	"minecraft:baked_potato":           6,
	"minecraft:beetroot":               5,
	"minecraft:beetroot_seeds":         1,
	"minecraft:melon_slice":            2,
	"minecraft:melon_block":            16,
	"minecraft:pumpkin":                14,
	"minecraft:sugar_cane":             3,
	"minecraft:cocoa_beans":            5,
	"minecraft:nether_wart":            8,
	"minecraft:apple":                  6,
	"minecraft:hay_block":              40,
	"minecraft:bread":                  14,
	"minecraft:golden_carrot":          60,
	"minecraft:glistering_melon_slice": 50,
	"minecraft:pumpkin_pie":            30,
}

// PriceOf returns the unit sell price of it, and false if it can't be sold.
func PriceOf(it world.Item) (int64, bool) {
	name, _ := it.EncodeItem()
	p, ok := Prices[name]
	return p, ok
}
