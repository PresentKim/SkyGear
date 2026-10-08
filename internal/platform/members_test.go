package platform

import (
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/presentkim/dragonfactory/internal/db"
	"github.com/presentkim/dragonfactory/internal/instance"
	"github.com/presentkim/dragonfactory/internal/playerstate"
)

func TestStoredInventoryEdit(t *testing.T) {
	d, err := db.Open(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	_, _ = d.TouchAccount("h", "", "h")
	_, _ = d.TouchAccount("g", "", "g")
	s, _ := d.CreateSave("h", "test", 1, "s")
	_ = d.AddMember(s.ID, "g", db.RoleGuest)
	m := instance.NewManager(slog.Default(), d, instance.Config{}, instance.Lobby{})
	pf := &Platform{log: slog.Default(), db: d, m: m}

	diamond := item.WriteNBT(item.NewStack(item.Diamond{}, 5), true)
	diamond["Slot"] = byte(3)
	st := playerstate.State{Version: playerstate.Version, Pos: []float64{0, 0, 0}, Health: 20, Inventory: []map[string]any{diamond}}
	b, _ := playerstate.Marshal(st)
	_ = d.SetPlayerState(s.ID, "g", b)

	tg := storedTarget{pf: pf, saveID: s.ID, acc: "g"}
	items := tg.items()
	if len(items) != 1 || items[0].slot != 3 || items[0].stack.Count() != 5 {
		t.Fatalf("items = %+v", items)
	}
	got, err := tg.take(slotRef{secInventory, 3})
	if err != nil || got.Count() != 5 {
		t.Fatalf("take = %v %v", got, err)
	}
	if len(tg.items()) != 0 {
		t.Fatal("taken item is still stored")
	}
	if err := tg.put(item.NewStack(item.Apple{}, 2)); err != nil {
		t.Fatal(err)
	}
	if items := tg.items(); len(items) != 1 || items[0].stack.Count() != 2 {
		t.Fatalf("after put = %+v", items)
	}
	unlock, _ := m.LockState(s.ID, "g")
	if _, err := tg.take(slotRef{secInventory, 0}); err == nil {
		t.Fatal("edited a state that is locked")
	}
	unlock()
}
