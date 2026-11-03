package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/conduit"
	"github.com/presentkim/skygear/content/skygear/dimension"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// TestCollectorFullNotDirty: a pool at its maximum does not mark the save
// changed every second.
func TestCollectorFullNotDirty(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	s.World.Do(func(tx *world.Tx) {
		slime := sgitem.MobSlime
		dimension.RecordOf(s.Session, slime).Best = 9
		for range 3000 {
			dimension.TickCollectors(s.Session, tx)
		}
		if dimension.Pool(s.Session, slime) < 64 {
			t.Fatal("the pool did not fill")
		}
		s.Session.Flush()
		dimension.TickCollectors(s.Session, tx)
		dimension.TickCollectors(s.Session, tx)
		if s.Session.Dirty() {
			t.Fatal("a full pool marked the save dirty")
		}
	})
}

// TestRemovePlatesLeavesNone: removing a bundle removes its plates and only
// those.
func TestRemovePlatesLeavesNone(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	pos, other := cube.Pos{3, 60, 3}, cube.Pos{5, 60, 3}
	pl := conduit.LocalOf(s.Session).Plates
	for _, side := range []int{0, 2, 5} {
		pl[conduit.PlateKey{Pos: pos, Side: side}] = nil
	}
	pl[conduit.PlateKey{Pos: other, Side: 1}] = nil
	s.World.Do(func(tx *world.Tx) { conduit.RemoveVisual(s.Session, tx, pos) })
	for k := range pl {
		if k.Pos == pos {
			t.Fatalf("plate %v is left", k)
		}
	}
	if len(pl) != 1 {
		t.Fatalf("%d plates left, want the 1 of the other bundle", len(pl))
	}
}

// TestStuckFellerKeepsItsLog: a feller with a full output keeps the log it
// found instead of scanning its area again, so a higher log placed meanwhile
// is not taken before it; it fells that one once the output has room.
func TestStuckFellerKeepsItsLog(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, machine.KindFeller, "")
	oak := block.Log{Wood: block.OakWood()}
	low, high := machinePos.Add(cube.Pos{2, 1, 0}), machinePos.Add(cube.Pos{2, 3, 0})
	s.World.Do(func(tx *world.Tx) { tx.SetBlock(low, oak, nil) })
	out := machine.Kinds[machine.KindFeller].Output
	for _, slot := range out {
		_ = st.Inv().SetItem(slot, item.NewStack(oak, 64))
	}
	tick(s, machine.Kinds[machine.KindFeller].Interval+2)
	s.World.Do(func(tx *world.Tx) {
		if _, ok := tx.Block(low).(block.Log); !ok {
			t.Error("the log was felled with a full output")
		}
		tx.SetBlock(high, oak, nil)
	})
	tick(s, 20)
	for _, slot := range out {
		_ = st.Inv().SetItem(slot, item.Stack{})
	}
	tick(s, 5)
	s.World.Do(func(tx *world.Tx) {
		if _, ok := tx.Block(low).(block.Log); ok {
			t.Error("the log it kept was not felled once there was room")
		}
		if _, ok := tx.Block(high).(block.Log); !ok {
			t.Error("the feller rescanned and took the higher log")
		}
	})
}
