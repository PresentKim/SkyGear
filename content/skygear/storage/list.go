package storage

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/itemio"
)

// ListItems implements itemio.Lister: a drawer lists the one kind it holds.
func (d Drawer) ListItems(*world.Tx, cube.Pos, cube.Face) []item.Stack {
	if d.Count <= 0 || d.Item.Empty() {
		return nil
	}
	return []item.Stack{d.Item.Grow(int(d.Count) - d.Item.Count())}
}

// controllerLister lists everything the drawers of a controller hold. The
// drawers are not listed on their own: a storage network mirroring a
// controller sees them through it only.
type controllerLister struct{}

func (controllerLister) ListItems(tx *world.Tx, pos cube.Pos, _ cube.Face) []item.Stack {
	list, _ := Contents(tx, pos)
	out := make([]item.Stack, 0, len(list))
	for _, e := range list {
		if e.Count > 0 {
			out = append(out, e.Item.Grow(int(e.Count)-e.Item.Count()))
		}
	}
	return out
}

func init() {
	itemio.RegisterLister(func(_ *world.Tx, _ cube.Pos, b world.Block) itemio.Lister {
		if IsStorageController(b) {
			return controllerLister{}
		}
		return nil
	})
}
