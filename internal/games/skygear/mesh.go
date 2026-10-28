package skygear

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/presentkim/skygear/internal/games/skygear/sgitem"
)

// Sieve meshes are made apart from the sieve and fitted by using them on it
// (an empty hand takes one out); the tier fitted picks the drop table. Better
// tiers are made from the string mesh and their material.

// equipMesh fits the mesh the player holds into the sieve st (the mesh it had
// goes back to the player) and reports whether the held item was a mesh.
func (s *session) equipMesh(p *player.Player, tx *world.Tx, pos cube.Pos, st *machineState, held item.Stack, off item.Stack) bool {
	m, ok := held.Item().(sgitem.Mesh)
	if !ok {
		return false
	}
	if st.mesh == m.Tier {
		return true
	}
	if old := st.mesh; old != "" {
		s.giveItem(p, tx, p.Position(), item.NewStack(sgitem.Mesh{Tier: old}, 1))
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
	s.giveItem(p, tx, p.Position(), item.NewStack(sgitem.Mesh{Tier: st.mesh}, 1))
	st.mesh = ""
	showSieve(tx, pos, st)
	tx.PlaySound(pos.Vec3Centre(), sound.ItemAdd{})
	return true
}

func init() {
	for _, t := range sgitem.MeshTiers {
		world.RegisterItem(sgitem.Mesh{Tier: t.Tier})
	}
	sgitem.MeshRecipes()
}
