package machine

import (
	"slices"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/presentkim/skygear/content/skygear/core"
)

// The automation equipment: the
// electric furnace, the item vacuum and the tree feller. The auto crafter is
// in autocraft.go.

// SmeltOf is what a furnace makes of an item.
func SmeltOf(in item.Stack) (item.Stack, bool) {
	sm, ok := in.Item().(item.Smeltable)
	if !ok {
		return item.Stack{}, false
	}
	p := sm.SmeltInfo().Product
	return p, !p.Empty()
}

// runFurnace smelts the input into the output slot without fuel.
func (st *MachineState) runFurnace() {
	st.process(func(in item.Stack) []item.Stack {
		out, _ := SmeltOf(in)
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
func runVacuum(s *core.Session, st *MachineState, pos cube.Pos, tx *world.Tx) {
	if st.Progress < st.Def.Interval {
		st.TickWork()
		return
	}
	st.Progress = 0
	x, y, z := float64(pos.X()), float64(pos.Y()), float64(pos.Z())
	reach := float64(st.Reach(KindVacuum))
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
		if stack.Empty() || !Fits(st.Inv(), st.Def.Output, []item.Stack{stack}) {
			continue
		}
		put(st.Inv(), st.Def.Output, stack)
		st.Done++
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
	// plantRise is how far above or below the machine the ground it plants on
	// may be.
	plantRise = 1
)

// runFeller fells the highest log in reach every interval; with no log left
// it plants a sapling from its input.
func runFeller(s *core.Session, st *MachineState, pos cube.Pos, tx *world.Tx) {
	if st.Progress < st.Def.Interval {
		st.TickWork()
		return
	}
	var logAt cube.Pos
	found := false
	reach := st.Reach(KindFeller)
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
		if !st.RoomFor(drops) {
			return
		}
		tx.SetBlock(logAt, block.Air{}, nil)
		tx.AddParticle(logAt.Vec3Centre(), particle.BlockBreak{Block: b})
		for _, it := range drops {
			if !it.Empty() {
				st.Emit(it)
			}
		}
		st.Progress = 0
		st.Done++
		return
	}
	st.Progress = 0
	if !st.plantSapling(tx, pos, reach) {
		st.boneMealSapling(tx, pos, reach)
	}
}

// plantSapling plants a sapling from the input of the feller at pos,
// reporting whether it did.
func (st *MachineState) plantSapling(tx *world.Tx, pos cube.Pos, reach int) bool {
	slot, in, ok := st.FirstInput()
	if !ok {
		return false
	}
	sap, ok := in.Item().(block.Sapling)
	if !ok {
		return false
	}
	at, ok := plantSpot(tx, pos, reach, sap)
	if !ok {
		return false
	}
	tx.SetBlock(at, sap, nil)
	_ = st.Inv().SetItem(slot, in.Grow(-1))
	return true
}

// boneMealSapling uses one bone meal of the feller at pos on the nearest
// sapling in reach that it works on, as a player using it would.
func (st *MachineState) boneMealSapling(tx *world.Tx, pos cube.Pos, reach int) {
	slot := st.Def.Charge[0]
	meal, _ := st.Inv().Item(slot)
	if _, ok := meal.Item().(item.BoneMeal); !ok {
		return
	}
	for _, at := range saplingsNear(tx, pos, reach) {
		if tx.Block(at).(block.Sapling).BoneMeal(at, tx) != item.BoneMealResultNone {
			tx.AddParticle(at.Vec3(), particle.BoneMeal{})
			_ = st.Inv().SetItem(slot, meal.Grow(-1))
			return
		}
	}
}

// saplingsNear are the saplings in reach of the feller at pos, nearest first.
func saplingsNear(tx *world.Tx, pos cube.Pos, reach int) []cube.Pos {
	var out []cube.Pos
	for dx := -reach; dx <= reach; dx++ {
		for dz := -reach; dz <= reach; dz++ {
			for dy := -plantRise; dy <= plantRise; dy++ {
				at := pos.Add(cube.Pos{dx, dy, dz})
				if _, ok := tx.Block(at).(block.Sapling); ok {
					out = append(out, at)
				}
			}
		}
	}
	horizontal := func(p cube.Pos) int { return (p[0]-pos[0])*(p[0]-pos[0]) + (p[2]-pos[2])*(p[2]-pos[2]) }
	slices.SortStableFunc(out, func(a, b cube.Pos) int { return horizontal(a) - horizontal(b) })
	return out
}

// plantSpot is the nearest place in reach of the feller at pos where sap
// grows: on soil, free or covered only by grass and the like, and with no
// sapling or log next to it so the trees have room.
func plantSpot(tx *world.Tx, pos cube.Pos, reach int, sap block.Sapling) (cube.Pos, bool) {
	var best cube.Pos
	bestDist := -1
	for dx := -reach; dx <= reach; dx++ {
		for dz := -reach; dz <= reach; dz++ {
			if dx == 0 && dz == 0 {
				continue
			}
			for dy := -plantRise; dy <= plantRise; dy++ {
				at := pos.Add(cube.Pos{dx, dy, dz})
				if dist := dx*dx + dz*dz; (bestDist < 0 || dist < bestDist) && plantable(tx, at, sap) {
					best, bestDist = at, dist
				}
			}
		}
	}
	return best, bestDist >= 0
}

// plantable reports whether sap can be planted at pos.
func plantable(tx *world.Tx, pos cube.Pos, sap block.Sapling) bool {
	switch b := tx.Block(pos).(type) {
	case block.Air:
	case world.Liquid:
		return false
	case block.Replaceable:
		if !b.ReplaceableBy(sap) {
			return false
		}
	default:
		return false
	}
	if soil, ok := tx.Block(pos.Side(cube.FaceDown)).(block.Soil); !ok || !soil.SoilFor(sap) {
		return false
	}
	for dx := -1; dx <= 1; dx++ {
		for dz := -1; dz <= 1; dz++ {
			switch tx.Block(pos.Add(cube.Pos{dx, 0, dz})).(type) {
			case block.Sapling, block.Log:
				return false
			}
		}
	}
	return true
}
