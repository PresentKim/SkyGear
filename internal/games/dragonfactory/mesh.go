package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/presentkim/dragonfactory/internal/i18n"
)

// Sieve meshes (docs/design/02-dragonfactory.md §6.1). A sieve is made
// without its mesh: the mesh is made apart and fitted by using it on the
// sieve, and the tier of the fitted mesh picks the drop table (sievetable.go).
// The string mesh is made from nine string; each better tier (flint, iron,
// gold, diamond, netherite) is made from the string mesh and one of its
// material, or from its material surrounded by string, and finds more
// (sievetable.go). A mesh is fitted by using it on a sieve and taken out
// again by using the sieve with an empty hand.

// meshTiers are the mesh tiers in order, with the name of their material
// and what a tier is made with (the first one is made from string).
var meshTiers = []struct {
	tier, name string
	material   world.Item
}{
	{"string", i18n.T(i18n.Default, "dragonfactory.mesh.init.1"), nil},
	{"flint", i18n.T(i18n.Default, "dragonfactory.mesh.init.2"), item.Flint{}},
	{"iron", i18n.T(i18n.Default, "dragonfactory.mesh.init.3"), item.IronIngot{}},
	{"gold", i18n.T(i18n.Default, "dragonfactory.mesh.init.4"), item.GoldIngot{}},
	{"diamond", i18n.T(i18n.Default, "dragonfactory.mesh.init.5"), item.Diamond{}},
	{"netherite", i18n.T(i18n.Default, "dragonfactory.mesh.init.6"), item.NetheriteIngot{}},
}

// meshName is the name of the mesh of a tier.
func meshName(tier string) string {
	for _, t := range meshTiers {
		if t.tier == tier {
			return t.name + i18n.T(i18n.Default, "dragonfactory.mesh.meshName.1")
		}
	}
	return i18n.T(i18n.Default, "dragonfactory.mesh.meshName.2")
}

// Mesh is the mesh item of a tier.
type Mesh struct{ Tier string }

func (m Mesh) EncodeItem() (string, int16) { return "dragonfactory:mesh_" + m.Tier, 0 }
func (m Mesh) Name() string                { return meshName(m.Tier) }
func (Mesh) Category() category.Category   { return category.Items() }

// equipMesh fits the mesh the player holds into the sieve st (the mesh it had
// goes back to the player) and reports whether the held item was a mesh.
func (s *session) equipMesh(p *player.Player, tx *world.Tx, pos cube.Pos, st *machineState, held item.Stack, off item.Stack) bool {
	m, ok := held.Item().(Mesh)
	if !ok {
		return false
	}
	if st.mesh == m.Tier {
		return true
	}
	if old := st.mesh; old != "" {
		s.giveItem(p, tx, p.Position(), item.NewStack(Mesh{Tier: old}, 1))
	}
	st.mesh = m.Tier
	showSieve(tx, pos, st)
	if !p.GameMode().CreativeInventory() {
		p.SetHeldItems(held.Grow(-1), off)
	}
	tx.PlaySound(pos.Vec3Centre(), sound.ItemAdd{})
	return true
}

// takeMesh takes the mesh out of an empty sieve into the player's
// inventory and reports whether there was one. Fitting and taking out give
// no message.
func (s *session) takeMesh(p *player.Player, tx *world.Tx, pos cube.Pos, st *machineState) bool {
	if st.mesh == "" {
		return false
	}
	if _, _, filled := st.firstInput(); filled {
		return false
	}
	s.giveItem(p, tx, p.Position(), item.NewStack(Mesh{Tier: st.mesh}, 1))
	st.mesh = ""
	showSieve(tx, pos, st)
	tx.PlaySound(pos.Vec3Centre(), sound.ItemAdd{})
	return true
}

// meshRecipes registers the recipes of the meshes: nine string in a 3×3 for
// the string mesh; every better tier has two recipes, the string mesh with
// one of its material (shapeless) and its material in the middle with string
// all round (opened by the researches, research.go).
func meshRecipes() {
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
	for _, t := range meshTiers[1:] {
		out := item.NewStack(Mesh{Tier: t.tier}, 1)
		mat := item.NewStack(t.material, 1)
		recipe.Register(recipe.NewShapeless([]recipe.Item{item.NewStack(Mesh{Tier: "string"}, 1), mat}, out, "crafting_table"))
		recipe.Register(recipe.NewShaped(ring(mat), out, recipe.NewShape(3, 3), "crafting_table").WithoutMirror())
	}
}

func init() {
	for _, t := range meshTiers {
		world.RegisterItem(Mesh{Tier: t.tier})
	}
	meshRecipes()
}
