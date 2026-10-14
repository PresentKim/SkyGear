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
)

// Sieve meshes (docs/design/02-dragonfactory.md §6.1). A sieve is made
// without its mesh: the mesh is made apart and fitted by using it on the
// sieve, and the tier of the fitted mesh picks the drop table (sievetable.go).
// Only the string mesh exists so far; better tiers (iron, gold, diamond,
// netherite) come later as more tiers here and more tables there.

// meshTiers are the mesh tiers in order, with the name of their material.
var meshTiers = []struct{ tier, name string }{{"string", "실"}}

// meshName is the name of the mesh of a tier.
func meshName(tier string) string {
	for _, t := range meshTiers {
		if t.tier == tier {
			return t.name + " 메쉬"
		}
	}
	return "메쉬"
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
		p.SendTip("§7이 체에는 이미 " + meshName(m.Tier) + "가 끼워져 있습니다.")
		return true
	}
	if old := st.mesh; old != "" {
		s.giveItem(p, tx, p.Position(), item.NewStack(Mesh{Tier: old}, 1))
	}
	st.mesh = m.Tier
	if !p.GameMode().CreativeInventory() {
		p.SetHeldItems(held.Grow(-1), off)
	}
	tx.PlaySound(pos.Vec3Centre(), sound.ItemAdd{})
	p.SendTip("§a" + meshName(m.Tier) + "를 끼웠습니다. 이제 자갈·모래·흙·먼지를 걸러 보세요.")
	return true
}

// meshRecipes registers the recipes of the meshes: four string in a square
// (open with the sieve research).
func meshRecipes() {
	str := item.NewStack(block.String{}, 1)
	recipe.Register(recipe.NewShaped([]recipe.Item{str, str, str, str}, item.NewStack(Mesh{Tier: "string"}, 1), recipe.NewShape(2, 2), "crafting_table").WithoutMirror())
}

func init() {
	for _, t := range meshTiers {
		world.RegisterItem(Mesh{Tier: t.tier})
	}
	meshRecipes()
}
