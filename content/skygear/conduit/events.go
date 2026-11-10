package conduit

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/fluidnet"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/network"
)

// The grids hear of conduits by events: BlockChanged (any SetBlock, whoever
// made it), NeighbourUpdateTick (a block beside a bundle changed) and the
// column calls below (LoadColumn, UnloadColumn, WriteShares). The bundle is not
// ticked.

var _ world.BlockChangeHook = Bundle{}

// NetChanged, if set, is told when a network conduit came to pos or left it,
// so the storage networks there are built again (netstore sets it).
var NetChanged func(tx *world.Tx, pos cube.Pos)

func netChanged(tx *world.Tx, pos cube.Pos) {
	if NetChanged != nil {
		NetChanged(tx, pos)
	}
}

// claimCell makes sure b has a buffer cell that belongs to pos alone: a bundle
// value copied to another position gets a cell of its own (empty, so the
// buffer is not duplicated). The block is set again when the cell changed.
func claimCell(s *core.Session, tx *world.Tx, pos cube.Pos, b Bundle) Bundle {
	owners := LocalOf(s).owners
	fresh := b.Buf == nil
	if !fresh {
		if at, taken := owners[b.Buf]; taken && at != pos {
			if o, ok := tx.Block(at).(Bundle); ok && o.Buf == b.Buf {
				fresh = true
			}
		}
	}
	if fresh {
		b.Buf = &BufCell{}
		owners[b.Buf] = pos
		tx.SetBlock(pos, b, &world.SetOpts{DisableBlockUpdates: true})
		return b
	}
	owners[b.Buf] = pos
	return b
}

// BlockChanged keeps the grids and the display entity in step with the bundle
// at pos: conduit kinds that came, went or changed tier, and item settings.
// A change of connection bits only touches nothing.
func (Bundle) BlockChanged(tx *world.Tx, pos cube.Pos, before, after world.Block) {
	s, ok := core.Of(tx)
	if !ok {
		return
	}
	old, wasBundle := before.(Bundle)
	nb, isBundle := after.(Bundle)
	if wasBundle && old.Buf != nil && (!isBundle || old.Buf != nb.Buf) {
		delete(LocalOf(s).owners, old.Buf)
	}
	if !isBundle {
		energyG, itemG, fluidG := network.Energy(s), network.Items(s), network.Fluids(s)
		if old.Energy > 0 {
			energyG.Remove(tx, pos)
		}
		if old.Item > 0 {
			itemG.Remove(tx, pos)
		}
		if old.Fluid > 0 {
			fluidG.Remove(tx, pos)
		}
		if old.Net > 0 {
			netChanged(tx, pos)
		}
		RemoveVisual(s, tx, pos)
		return
	}
	if !wasBundle {
		old = Bundle{}
	}
	nb = claimCell(s, tx, pos, nb)
	cell := nb.Buf
	if old.Energy != nb.Energy {
		switch {
		case old.Energy == 0:
			network.Energy(s).Place(tx, pos, cell.Energy)
		case nb.Energy == 0:
			network.Energy(s).Remove(tx, pos)
			cell.Energy = 0
		default:
			network.Energy(s).Changed(tx, pos)
		}
	}
	if old.Fluid != nb.Fluid {
		switch {
		case old.Fluid == 0:
			network.Fluids(s).Place(tx, pos, fluidnet.Kind(cell.FluidKind), cell.Fluid)
		case nb.Fluid == 0:
			network.Fluids(s).Remove(tx, pos)
			cell.FluidKind, cell.Fluid = 0, 0
		default:
			network.Fluids(s).Changed(tx, pos)
		}
	}
	sig := old.settingsSig() != nb.settingsSig()
	if sig {
		switch {
		case old.Item == 0:
			network.Items(s).Place(tx, pos)
		case nb.Item == 0:
			network.Items(s).Remove(tx, pos)
		default:
			network.Items(s).Changed(tx, pos)
		}
	}
	if old.Net != nb.Net {
		netChanged(tx, pos)
	}
	if old.Energy != nb.Energy || old.Fluid != nb.Fluid || old.Net != nb.Net || sig {
		// A new or changed bundle finds its sides and shows itself.
		nb.refresh(pos, tx, true)
	}
}

// LoadColumn puts the bundles of a column that was loaded into the grids with
// their saved share, and makes their display entities. It reads only
// neighbours in loaded columns, so it never loads a column (which would fire
// the load hook again). Bundles of loaded columns beside the new ones are
// refreshed, since their NeighbourUpdateTick did not fire for the load.
func LoadColumn(s *core.Session, tx *world.Tx, blockEntities map[cube.Pos]world.Block) {
	for pos, blk := range blockEntities {
		b, ok := blk.(Bundle)
		if !ok {
			continue
		}
		cell := b.Buf
		if cell == nil {
			cell = &BufCell{}
		} else {
			LocalOf(s).owners[cell] = pos
		}
		if b.Energy > 0 {
			network.Energy(s).Place(tx, pos, cell.Energy)
		}
		if b.Item > 0 {
			network.Items(s).Place(tx, pos)
		}
		if b.Fluid > 0 {
			network.Fluids(s).Place(tx, pos, fluidnet.Kind(cell.FluidKind), cell.Fluid)
		}
		b.refreshOf(pos, tx, true, true)
	}
	// Bundles beside the column that were loaded before it find their new sides.
	done := map[cube.Pos]struct{}{}
	for pos, blk := range blockEntities {
		if _, ok := blk.(Bundle); !ok {
			continue
		}
		for _, f := range Faces {
			at := pos.Side(f)
			if _, own := blockEntities[at]; own {
				continue
			}
			if _, seen := done[at]; seen {
				continue
			}
			done[at] = struct{}{}
			if o, ok := tx.BlockLoaded(at); ok {
				if ob, isBundle := o.(Bundle); isBundle {
					ob.refreshOf(at, tx, true, true)
				}
			}
		}
	}
}

// UnloadColumn takes the bundles of a column that is going away out of the
// grids and writes the share each took with it into its cell.
func UnloadColumn(s *core.Session, tx *world.Tx, blockEntities map[cube.Pos]world.Block) {
	for pos, blk := range blockEntities {
		b, ok := blk.(Bundle)
		if !ok {
			continue
		}
		var e, fk, f int
		if b.Energy > 0 {
			e = network.Energy(s).Remove(tx, pos)
		}
		if b.Item > 0 {
			network.Items(s).Remove(tx, pos)
		}
		if b.Fluid > 0 {
			k, mB := network.Fluids(s).Remove(tx, pos)
			fk, f = int(k), mB
		}
		if b.Buf != nil {
			*b.Buf = BufCell{Energy: e, FluidKind: fk, Fluid: f}
			delete(LocalOf(s).owners, b.Buf)
		}
		tx.MarkModified(pos)
		RemoveVisual(s, tx, pos)
	}
}

// WriteShares writes the share of every conduit in the grids into its cell
// (before the world is saved); only cells whose value changed are marked.
func WriteShares(s *core.Session, tx *world.Tx) {
	set := func(pos cube.Pos, apply func(c *BufCell) bool) {
		b, ok := tx.Block(pos).(Bundle)
		if !ok || b.Buf == nil {
			return
		}
		if apply(b.Buf) {
			tx.MarkModified(pos)
		}
	}
	network.Energy(s).Shares(func(pos cube.Pos, share int) {
		set(pos, func(c *BufCell) bool {
			if c.Energy == share {
				return false
			}
			c.Energy = share
			return true
		})
	})
	network.Fluids(s).Shares(func(pos cube.Pos, k fluidnet.Kind, mB int) {
		set(pos, func(c *BufCell) bool {
			if c.FluidKind == int(k) && c.Fluid == mB {
				return false
			}
			c.FluidKind, c.Fluid = int(k), mB
			return true
		})
	})
}
