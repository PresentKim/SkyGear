package dfitem

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/i18n"
)

// Metals: iron, gold and copper
// use the ingots of the game; tin, lead, silver and nickel are new ores found
// by sieving (orechunk.go); bronze, electrum, invar and steel are alloys made
// in the alloy smelter (alloy.go). Every ore metal has a dust (ore.go), which
// a furnace smelts into an ingot.
const (
	MetalTin      = "tin"
	MetalLead     = "lead"
	MetalSilver   = "silver"
	MetalNickel   = "nickel"
	MetalBronze   = "bronze"
	MetalElectrum = "electrum"
	MetalInvar    = "invar"
	MetalSteel    = "steel"
)

// DustMetals have a dust: the ores of the game and the new ones.
var DustMetals = []string{OreIron, OreGold, OreCopper, MetalTin, MetalLead, MetalSilver, MetalNickel}

// IngotMetals have an ingot item of their own.
var IngotMetals = []string{MetalTin, MetalLead, MetalSilver, MetalNickel, MetalBronze, MetalElectrum, MetalInvar, MetalSteel}

func metalName(metal string) string { return i18n.T(i18n.Default, "metal."+metal) }

// IngotOf returns the ingot of a metal.
func IngotOf(metal string) world.Item {
	switch metal {
	case OreIron:
		return item.IronIngot{}
	case OreGold:
		return item.GoldIngot{}
	case OreCopper:
		return item.CopperIngot{}
	}
	return Ingot{Metal: metal}
}

// MetalDust is a metal ground to dust: a furnace smelts it into an ingot.
type MetalDust struct{ Metal string }

func (d MetalDust) EncodeItem() (string, int16) { return "dragonfactory:metal_dust_" + d.Metal, 0 }

func (d MetalDust) Name() string {
	return metalName(d.Metal) + i18n.T(i18n.Default, "name.MetalDust.suffix")
}

func (MetalDust) Category() category.Category { return category.Items() }

// SmeltInfo implements item.Smeltable.
func (d MetalDust) SmeltInfo() item.SmeltInfo {
	return item.SmeltInfo{Product: item.NewStack(IngotOf(d.Metal), 1), Experience: 0.7}
}

// Ingot is the ingot of a metal that the game adds: the new ores and the alloys.
type Ingot struct{ Metal string }

func (i Ingot) EncodeItem() (string, int16) { return "dragonfactory:ingot_" + i.Metal, 0 }

func (i Ingot) Name() string {
	return metalName(i.Metal) + i18n.T(i18n.Default, "name.Ingot.suffix")
}

func (Ingot) Category() category.Category { return category.Items() }
