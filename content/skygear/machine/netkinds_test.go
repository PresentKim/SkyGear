package machine_test

import (
	"slices"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/netstore"
)

// The network kinds get their behaviour from the netstore feature's Setup,
// which no feature list runs in this package's tests.
func init() { netstore.Feature.Setup() }

var netKinds = []string{"net_controller", "net_cable", "disk_drive", "net_terminal", "net_importer", "net_exporter", "net_external", "pattern_encoder", "net_autocrafter"}

func TestNetKindsRegistered(t *testing.T) {
	oldEnd := slices.Index(machine.Order, machine.KindRelicTable)
	for i, k := range netKinds {
		def, ok := machine.Kinds[k]
		if !ok {
			t.Fatalf("%s: no kind", k)
		}
		if def.Hash == 0 {
			t.Errorf("%s: not registered as a block", k)
		}
		if at := slices.Index(machine.Order, k); at != oldEnd+1+i {
			t.Errorf("%s: at %d in Order, want %d", k, at, oldEnd+1+i)
		}
	}
	if machine.Kinds[machine.KindNetController].Use == 0 {
		t.Error("the controller needs energy")
	}
}

func TestDriveAcceptsOnlyDisks(t *testing.T) {
	m := machine.New(machine.KindDiskDrive)
	st := m.State
	disk := netstore.WithDiskID(item.NewStack(netstore.Disk{Tier: 1}, 1), "d1")
	if !st.SlotAccepts(3, disk) {
		t.Error("the drive refuses a disk")
	}
	if st.SlotAccepts(3, item.NewStack(block.Dirt{}, 1)) {
		t.Error("the drive takes dirt")
	}
	if st.SlotAccepts(8, disk) {
		t.Error("slot 8 is no disk slot")
	}
	for slot := range 8 {
		_ = st.Inv().SetItem(slot, netstore.WithDiskID(item.NewStack(netstore.Disk{Tier: slot % 4}, 1), "id"+string(rune('a'+slot))))
	}
	back := m.DecodeNBT(m.EncodeNBT()).(machine.Machine)
	for slot := range 8 {
		got, _ := back.State.Inv().Item(slot)
		if want := "id" + string(rune('a'+slot)); netstore.DiskID(got) != want {
			t.Errorf("slot %d: disk id %q, want %q", slot, netstore.DiskID(got), want)
		}
		if d, ok := got.Item().(netstore.Disk); !ok || d.Tier != slot%4 {
			t.Errorf("slot %d: item %v", slot, got.Item())
		}
	}
}

func TestDiskID(t *testing.T) {
	if got := netstore.DiskID(netstore.WithDiskID(item.NewStack(netstore.Disk{Tier: 1}, 1), "d1")); got != "d1" {
		t.Errorf("got %q", got)
	}
	if got := netstore.DiskID(item.NewStack(netstore.Disk{Tier: 1}, 1)); got != "" {
		t.Errorf("got %q", got)
	}
}
