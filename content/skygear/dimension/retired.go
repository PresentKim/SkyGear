package dimension

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/farm"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// The dimension gate and its waves gave way to the second dimension. Its
// block stays registered so old islands load: a gate on the island breaks by
// itself into what it was made of, and a gate item a player carries turns
// into those materials.

// gateMaterials are what a gate was made of.
func gateMaterials() []item.Stack {
	return []item.Stack{
		item.NewStack(sgitem.Ingot{Metal: sgitem.MetalSteel}, 4),
		item.NewStack(block.Obsidian{}, 3),
		item.NewStack(item.EnderPearl{}, 2),
	}
}

// monsterGone handles a monster of a run that died to p (nil when it blew
// itself up): the run counts it. Any other monster just goes.
func monsterGone(s *core.Session, tx *world.Tx, m *Monster, p *player.Player) {
	if r := RunOf(s); r != nil && r.Owns(m.Handle) {
		r.killed(s, tx, m, p)
		return
	}
	_ = m.Close()
}

// retireGate breaks the gate at pos into its materials.
func retireGate(s *core.Session, tx *world.Tx, pos cube.Pos) {
	tx.SetBlock(pos, nil, nil)
	for _, it := range gateMaterials() {
		s.DropAt(tx, pos.Vec3Centre(), it)
	}
}

// RetireItems turns the gates p carries into their materials and the seeds
// of essence blooms into redstone bloom seeds (session.Second and Join).
func RetireItems(s *core.Session, tx *world.Tx, p *player.Player) {
	farm.RetireSeeds(p)
	inv := p.Inventory()
	for slot, it := range inv.Slots() {
		if fixed := sgitem.FixRelic(it); !it.Empty() && !fixed.Equal(it) {
			_ = inv.SetItem(slot, fixed)
			continue
		}
		m, ok := it.Item().(machine.Machine)
		if !ok || m.Kind != machine.KindGate {
			continue
		}
		_ = inv.SetItem(slot, item.Stack{})
		for range it.Count() {
			for _, mat := range gateMaterials() {
				s.GiveItem(p, tx, p.Position().Add(mgl64.Vec3{0, 0.5, 0}), mat)
			}
		}
	}
}

func init() {
	machine.Kinds[machine.KindGate].Run = func(s *core.Session, _ *machine.State, pos cube.Pos, tx *world.Tx) {
		retireGate(s, tx, pos)
	}
	machine.Kinds[machine.KindGate].Click = func(s *core.Session, _ *player.Player, tx *world.Tx, pos cube.Pos) bool {
		retireGate(s, tx, pos)
		return true
	}
}
