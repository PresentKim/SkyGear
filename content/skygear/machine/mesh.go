package machine

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// Sieve meshes are made apart from the sieve and fitted by using them on it
// (an empty hand takes one out); the tier fitted picks the drop table. Better
// tiers are made from the string mesh and their material.

func init() {
	for _, t := range sgitem.MeshTiers {
		world.RegisterItem(sgitem.Mesh{Tier: t.Tier})
	}
	sgitem.MeshRecipes()
}

// equipMesh fits the mesh the player holds into the sieve st (the mesh it had
// goes back to the player) and reports whether the held item was a mesh.
func equipMesh(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, st *MachineState, held item.Stack, off item.Stack) bool {
	m, ok := held.Item().(sgitem.Mesh)
	if !ok {
		return false
	}
	if st.Mesh == m.Tier {
		return true
	}
	if old := st.Mesh; old != "" {
		s.GiveItem(p, tx, p.Position(), item.NewStack(sgitem.Mesh{Tier: old}, 1))
	}
	st.Mesh = m.Tier
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
func takeMesh(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, st *MachineState) bool {
	if st.Mesh == "" {
		return false
	}
	if _, _, filled := st.FirstInput(); filled {
		return false
	}
	s.GiveItem(p, tx, p.Position(), item.NewStack(sgitem.Mesh{Tier: st.Mesh}, 1))
	st.Mesh = ""
	showSieve(tx, pos, st)
	tx.PlaySound(pos.Vec3Centre(), sound.ItemAdd{})
	return true
}
