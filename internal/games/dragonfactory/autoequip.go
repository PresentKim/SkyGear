package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
)

// The automation equipment (docs/design/09-automation-equipment.md): the
// electric furnace, the item vacuum and the tree feller. The auto crafter is
// in autocraft.go.

// smeltOf is what a furnace makes of an item.
func smeltOf(in item.Stack) (item.Stack, bool) {
	sm, ok := in.Item().(item.Smeltable)
	if !ok {
		return item.Stack{}, false
	}
	p := sm.SmeltInfo().Product
	return p, !p.Empty()
}

// runFurnace smelts the input into the output slot without fuel.
func (st *machineState) runFurnace() {
	st.process(func(in item.Stack) []item.Stack {
		out, _ := smeltOf(in)
		return []item.Stack{out}
	}, nil)
}

// The reach of the item vacuum around it: sideways, below and above.
const (
	vacuumReach = 6
	vacuumDown  = 3
	vacuumUp    = 5
)

// runVacuum collects one stack of the dropped items around it every
// interval. A stack that does not fit the output is left lying.
func (s *session) runVacuum(st *machineState, pos cube.Pos, tx *world.Tx) {
	if st.progress < st.kind.interval {
		st.tickWork()
		return
	}
	st.progress = 0
	x, y, z := float64(pos.X()), float64(pos.Y()), float64(pos.Z())
	reach := float64(st.reach(KindVacuum))
	box := cube.Box(x-reach, y-vacuumDown, z-reach, x+reach+1, y+vacuumUp+1, z+reach+1)
	for e := range tx.EntitiesWithin(box) {
		ent, ok := e.(*entity.Ent)
		if !ok {
			continue
		}
		ib, ok := ent.Behaviour().(*entity.ItemBehaviour)
		if !ok {
			continue
		}
		stack := ib.Item()
		if stack.Empty() || !fits(st.inv(), st.kind.output, []item.Stack{stack}) {
			continue
		}
		put(st.inv(), st.kind.output, stack)
		st.done++
		tx.AddParticle(ent.Position(), particle.EndermanTeleport{})
		_ = ent.Close()
		return
	}
}

// The reach of the tree feller: sideways, below and above.
const (
	fellerReach = 5
	fellerDown  = 1
	fellerUp    = 14
	// plantStep is the distance between the saplings it plants.
	plantStep = 3
)

// runFeller fells the highest log in reach every interval; with no log left
// it plants a sapling from its input.
func (s *session) runFeller(st *machineState, pos cube.Pos, tx *world.Tx) {
	if st.progress < st.kind.interval {
		st.tickWork()
		return
	}
	var logAt cube.Pos
	found := false
	reach := st.reach(KindFeller)
	for dy := fellerUp; dy >= -fellerDown && !found; dy-- {
		for dx := -reach; dx <= reach && !found; dx++ {
			for dz := -reach; dz <= reach && !found; dz++ {
				at := pos.Add(cube.Pos{dx, dy, dz})
				if _, ok := tx.Block(at).(block.Log); ok {
					logAt, found = at, true
				}
			}
		}
	}
	if found {
		b := tx.Block(logAt)
		var drops []item.Stack
		if br, ok := b.(block.Breakable); ok && br.BreakInfo().Drops != nil {
			drops = br.BreakInfo().Drops(item.ToolNone{}, nil)
		}
		if !st.roomFor(drops) {
			return
		}
		tx.SetBlock(logAt, block.Air{}, nil)
		tx.AddParticle(logAt.Vec3Centre(), particle.BlockBreak{Block: b})
		for _, it := range drops {
			if !it.Empty() {
				st.emit(it)
			}
		}
		st.progress = 0
		st.done++
		return
	}
	st.progress = 0
	slot, in, ok := st.firstInput()
	if !ok {
		return
	}
	sap, ok := in.Item().(block.Sapling)
	if !ok {
		return
	}
	for dx := -reach / plantStep * plantStep; dx <= reach; dx += plantStep {
		for dz := -reach / plantStep * plantStep; dz <= reach; dz += plantStep {
			if dx == 0 && dz == 0 {
				continue
			}
			at := pos.Add(cube.Pos{dx, 0, dz})
			if _, air := tx.Block(at).(block.Air); !air {
				continue
			}
			switch tx.Block(at.Side(cube.FaceDown)).(type) {
			case block.Dirt, block.Grass:
			default:
				continue
			}
			tx.SetBlock(at, sap, nil)
			_ = st.inv().SetItem(slot, in.Grow(-1))
			return
		}
	}
}
