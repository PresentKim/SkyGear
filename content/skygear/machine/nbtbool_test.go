package machine

import (
	"reflect"
	"testing"

	"github.com/df-mc/dragonfly/server/item"

	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

// TestUnlimitedSurvivesNBTBytes: Unlimited must survive a real NBT byte round trip.
func TestUnlimitedSurvivesNBTBytes(t *testing.T) {
	m := New(KindCrusher, "", "")
	m.State.Unlimited = true
	raw, err := nbt.MarshalEncoding(m.EncodeNBT(), nbt.LittleEndian)
	if err != nil {
		t.Fatal(err)
	}
	var data map[string]any
	if err := nbt.UnmarshalEncoding(raw, &data, nbt.LittleEndian); err != nil {
		t.Fatal(err)
	}
	got := m.DecodeNBT(data).(Machine)
	if !got.State.Unlimited {
		t.Fatal("Unlimited lost after NBT byte round trip")
	}
}

// TestMachineItemsStack: for every kind, a new machine item, one dropped by
// breaking a placed machine and one read back from a save stack together;
// a machine that holds something keeps all of it.
func TestMachineItemsStack(t *testing.T) {
	for kind := range Kinds {
		fresh := Item(kind)
		placed := New(kind, "", "")
		dropped := placed.BreakInfo().Drops(item.ToolNone{}, nil)[0]
		saved := item.NewStack(placed.DecodeNBT(fresh.Item().(Machine).EncodeNBT()).(Machine), 1)
		again := item.NewStack(saved.Item().(Machine).DecodeNBT(saved.Item().(Machine).EncodeNBT()).(Machine), 1)
		for name, st := range map[string]item.Stack{"dropped": dropped, "saved": saved, "saved twice": again} {
			if !fresh.Comparable(st) {
				t.Errorf("%s: a new item and a %s one do not stack: %v", kind, name, st.Item().(Machine).EncodeNBT())
			}
		}
		if got := placed.DecodeNBT(map[string]any{"id": machineNBTID}).(Machine).encodeState(); !reflect.DeepEqual(got, blankNBT(kind)) {
			t.Errorf("%s: a machine loaded from its id alone is not blank: %v", kind, got)
		}
	}
	m := New(KindSolarGenerator, "", "")
	m.State.Energy = 5
	back := m.DecodeNBT(m.EncodeNBT()).(Machine)
	if back.State.Energy != 5 {
		t.Errorf("a machine holding energy saved %v", m.EncodeNBT())
	}
}
