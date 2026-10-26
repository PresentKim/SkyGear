package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/dfitem"
)

// Sieve meshes (docs/design/02-dragonfactory.md §6). A sieve is made
// without its mesh: the mesh is made apart and fitted by using it on the
// sieve, and the tier of the fitted mesh picks the drop table (sievetable.go).
// The string mesh is made from nine string; each better tier (flint, iron,
// gold, diamond, netherite) is made from the string mesh and one of its
// material, or from its material surrounded by string, and finds more
// (sievetable.go). A mesh is fitted by using it on a sieve and taken out
// again by using the sieve with an empty hand.

// equipMesh fits the mesh the player holds into the sieve st (the mesh it had
// goes back to the player) and reports whether the held item was a mesh.
func (s *session) equipMesh(p *player.Player, tx *world.Tx, pos cube.Pos, st *machineState, held item.Stack, off item.Stack) bool {
	m, ok := held.Item().(dfitem.Mesh)
	if !ok {
		return false
	}
	if st.mesh == m.Tier {
		return true
	}
	if old := st.mesh; old != "" {
		s.giveItem(p, tx, p.Position(), item.NewStack(dfitem.Mesh{Tier: old}, 1))
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
	s.giveItem(p, tx, p.Position(), item.NewStack(dfitem.Mesh{Tier: st.mesh}, 1))
	st.mesh = ""
	showSieve(tx, pos, st)
	tx.PlaySound(pos.Vec3Centre(), sound.ItemAdd{})
	return true
}

func init() {
	for _, t := range dfitem.MeshTiers {
		world.RegisterItem(dfitem.Mesh{Tier: t.Tier})
	}
	dfitem.MeshRecipes()
}
