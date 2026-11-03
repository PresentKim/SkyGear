package machine

import (
	"testing"

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
