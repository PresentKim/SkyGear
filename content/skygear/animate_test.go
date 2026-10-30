package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/machine"
)

// TestActiveLook: an animated machine shows its working look while it works,
// keeps it a moment after, and rests afterwards; the look is its own block
// state.
func TestActiveLook(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, machine.KindFurnace, "")
	look := func() bool {
		on := false
		s.World.Do(func(tx *world.Tx) { on = tx.Block(machinePos).(machine.Machine).Active })
		return on
	}
	tick(s, 5)
	if look() {
		t.Fatal("an idle furnace shows as working")
	}
	_ = st.Inv().SetItem(11, item.NewStack(block.Cobblestone{}, 1))
	tick(s, 10)
	if !look() {
		t.Fatal("a working furnace does not show it")
	}
	tick(s, machine.Kinds[machine.KindFurnace].Interval+machine.ActiveHold+10)
	if look() {
		t.Fatal("a furnace that finished keeps showing as working")
	}
	if machine.OutputCount(st, block.Stone{}) != 1 {
		t.Fatal("the state change lost the machine's contents")
	}
	a, b := (machine.Machine{Kind: machine.KindFurnace}), (machine.Machine{Kind: machine.KindFurnace, Active: true})
	if _, ha := a.Hash(); true {
		if _, hb := b.Hash(); ha == hb {
			t.Fatal("both looks have the same block hash")
		}
	}
	if _, props := b.EncodeBlock(); props[machine.TierState] != "1" {
		t.Fatalf("encoded state %v", props)
	}
}
