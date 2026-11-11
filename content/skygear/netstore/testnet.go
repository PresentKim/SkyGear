package netstore

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/machine"
)

// BuildTestNet builds an always powered network on stone at at, at+x and
// at+2x (controller, terminal, drive with the largest disk) and fills it
// with up to kinds stacks of registered items. It returns the terminal, or
// false and builds nothing if any of those blocks or the ones below them
// is not air.
func BuildTestNet(tx *world.Tx, at cube.Pos, kinds int) (cube.Pos, bool) {
	for dx := range 3 {
		for dy := -1; dy <= 0; dy++ {
			if _, air := tx.Block(at.Add(cube.Pos{dx, dy, 0})).(block.Air); !air {
				return cube.Pos{}, false
			}
		}
	}
	place := func(dx int, kind string) machine.Machine {
		pos := at.Add(cube.Pos{dx, 0, 0})
		tx.SetBlock(pos.Side(cube.FaceDown), block.Stone{}, nil)
		m := machine.New(kind)
		m.State.Unlimited = kind == machine.KindNetController
		tx.SetBlock(pos, m, nil)
		Invalidate(tx, pos)
		return tx.Block(pos).(machine.Machine)
	}
	place(0, machine.KindNetController)
	term := at.Add(cube.Pos{1, 0, 0})
	place(1, machine.KindNetTerminal)
	d := place(2, machine.KindDiskDrive)
	_ = d.State.Inv().SetItem(0, item.NewStack(Disk{Tier: len(diskNames) - 1}, 1))
	n := At(tx, term)
	if n == nil {
		return term, true
	}
	n.syncDrives(tx)
	filled := 0
	for _, it := range world.Items() {
		if filled >= kinds {
			break
		}
		st := item.NewStack(it, max(1, item.NewStack(it, 1).MaxCount()))
		if netInsert(n, st) > 0 {
			filled++
		}
	}
	return term, true
}
