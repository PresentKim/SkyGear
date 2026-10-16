package dragonfactory

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

// TestActiveLook: an animated machine shows its working look while it works,
// keeps it a moment after, and rests afterwards; the look is its own block
// state.
func TestActiveLook(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindFurnace, "")
	look := func() bool {
		on := false
		s.world.Do(func(tx *world.Tx) { on = tx.Block(machinePos).(Machine).active })
		return on
	}
	tick(s, 5)
	if look() {
		t.Fatal("an idle furnace shows as working")
	}
	_ = st.inv().SetItem(11, item.NewStack(block.Cobblestone{}, 1))
	tick(s, 10)
	if !look() {
		t.Fatal("a working furnace does not show it")
	}
	tick(s, machineKinds[KindFurnace].interval+activeHold+10)
	if look() {
		t.Fatal("a furnace that finished keeps showing as working")
	}
	if outputCount(st, block.Stone{}) != 1 {
		t.Fatal("the state change lost the machine's contents")
	}
	a, b := (Machine{Kind: KindFurnace}), (Machine{Kind: KindFurnace, active: true})
	if _, ha := a.Hash(); true {
		if _, hb := b.Hash(); ha == hb {
			t.Fatal("both looks have the same block hash")
		}
	}
	if _, props := b.EncodeBlock(); props[tierState] != "1" {
		t.Fatalf("encoded state %v", props)
	}
}
