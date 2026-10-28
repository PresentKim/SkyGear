package sgitem

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
)

// MeshTiers are the mesh tiers in order, with the catalog key of the name of
// their material and what a tier is made with (the first one is made from
// string).
var MeshTiers = []struct {
	Tier, nameKey string
	Material      world.Item
}{
	{"string", "mesh.string", nil},
	{"flint", "mesh.flint", item.Flint{}},
	{"iron", "mesh.iron", item.IronIngot{}},
	{"gold", "mesh.gold", item.GoldIngot{}},
	{"diamond", "mesh.diamond", item.Diamond{}},
	{"netherite", "mesh.netherite", item.NetheriteIngot{}},
}

// MeshName is the name of the mesh of a tier in lang.
func MeshName(lang, tier string) string {
	for _, t := range MeshTiers {
		if t.Tier == tier {
			return i18n.T(lang, t.nameKey) + i18n.T(lang, "mesh.suffix")
		}
	}
	return i18n.T(lang, "mesh.name")
}

// Mesh is the mesh item of a tier.
type Mesh struct{ Tier string }

func (m Mesh) EncodeItem() (string, int16) { return "skygear:mesh_" + m.Tier, 0 }

func (m Mesh) Name() string { return MeshName(i18n.Default, m.Tier) }

func (Mesh) Category() category.Category { return category.Items() }

// MeshRecipes registers the recipes of the meshes: nine string in a 3×3 for
// the string mesh; every better tier has two recipes, the string mesh with
// one of its material (shapeless) and its material in the middle with string
// all round (opened by the researches, research.go).
func MeshRecipes() {
	str := item.NewStack(block.String{}, 1)
	ring := func(centre recipe.Item) []recipe.Item {
		in := make([]recipe.Item, 9)
		for i := range in {
			in[i] = str
		}
		in[4] = centre
		return in
	}
	recipe.Register(recipe.NewShaped(ring(str), item.NewStack(Mesh{Tier: "string"}, 1), recipe.NewShape(3, 3), "crafting_table").WithoutMirror())
	for _, t := range MeshTiers[1:] {
		out := item.NewStack(Mesh{Tier: t.Tier}, 1)
		mat := item.NewStack(t.Material, 1)
		recipe.Register(recipe.NewShapeless([]recipe.Item{item.NewStack(Mesh{Tier: "string"}, 1), mat}, out, "crafting_table"))
		recipe.Register(recipe.NewShaped(ring(mat), out, recipe.NewShape(3, 3), "crafting_table").WithoutMirror())
	}
}
