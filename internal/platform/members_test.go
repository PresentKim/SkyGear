package platform

import (
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/event"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/chestui"
	"github.com/presentkim/skygear/common/playerstate"
	"github.com/presentkim/skygear/internal/db"
	"github.com/presentkim/skygear/internal/instance"
)

func TestInvSeeSlots(t *testing.T) {
	seen := map[[2]int]bool{}
	for slot := range invSeeSize {
		sec, i := invSeeSlot(slot)
		if sec == secNone {
			continue
		}
		k := [2]int{int(sec), i}
		if seen[k] {
			t.Fatalf("slot %d maps to %v again", slot, k)
		}
		seen[k] = true
	}
	// 36 inventory + 4 armour + 1 off hand + 27 ender chest.
	if len(seen) != 68 {
		t.Fatalf("mapped %d slots, want 68", len(seen))
	}
	if sec, i := invSeeSlot(invSeeHotbar); sec != secInventory || i != 0 {
		t.Fatalf("hotbar start = %v %d", sec, i)
	}
}

func TestStoredInvSee(t *testing.T) {
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

	w := world.Config{Synchronous: true, Entities: entity.DefaultRegistry}.New()
	defer w.Close()
	<-w.Do(func(tx *world.Tx) {
		if _, _, _, _, err := pf.targetSource(tx, s.ID, "g"); err != errNeverEntered {
			t.Errorf("never entered: err = %v", err)
		}
	}).Done()

	diamond := item.WriteNBT(item.NewStack(item.Diamond{}, 5), true)
	diamond["Slot"] = byte(3)
	st := playerstate.State{Version: playerstate.Version, Pos: []float64{0, 0, 0}, Health: 20, Inventory: []map[string]any{diamond}}
	b, _ := playerstate.Marshal(st)
	_ = d.SetPlayerState(s.ID, "g", b)

	<-w.Do(func(tx *world.Tx) {
		src, live, readOnly, unlock, err := pf.targetSource(tx, s.ID, "g")
		if err != nil || live || readOnly || unlock == nil {
			t.Fatalf("targetSource = %v %v %v", live, readOnly, err)
		}
		if _, ok := m.LockState(s.ID, "g"); ok {
			t.Error("state is not locked while viewed")
		}
		// A second viewer only reads.
		if _, _, ro, _, _ := pf.targetSource(tx, s.ID, "g"); !ro {
			t.Error("second viewer can edit")
		}
		viewer := tx.AddEntity(world.EntitySpawnOpts{}.New(player.Type, player.Config{Name: "h"})).(*player.Player)
		v := &invSee{st: chestui.NewStorage(invSeeSize, "t"), src: src}
		v.sync(tx)
		v.st.Inventory().Handle(v)
		v.st.OnChange(v.changed)

		if it, _ := v.st.Inventory().Item(invSeeHotbar + 3); it.Count() != 5 {
			t.Fatalf("hotbar slot 3 = %v", it)
		}
		// Take the diamonds, the way a session applies a move.
		slot := invSeeHotbar + 3
		ctx := event.C(inventory.Holder(viewer))
		v.HandleTake(ctx, slot, item.NewStack(item.Diamond{}, 5))
		if ctx.Cancelled() {
			t.Fatal("take was cancelled")
		}
		_ = v.st.Inventory().SetItem(slot, item.Stack{})
		// Put an apple in the off hand.
		ctx = event.C(inventory.Holder(viewer))
		v.HandlePlace(ctx, invSeeOffHand, item.NewStack(item.Apple{}, 2))
		_ = v.st.Inventory().SetItem(invSeeOffHand, item.NewStack(item.Apple{}, 2))
		// Fillers cannot be touched.
		ctx = event.C(inventory.Holder(viewer))
		v.HandleTake(ctx, invSeeOffHand+1, item.Stack{})
		if !ctx.Cancelled() {
			t.Error("filler was not cancelled")
		}
		unlock()
	}).Done()

	_, blob, _ := d.PlayerState(s.ID, "g")
	got, _ := playerstate.Unmarshal(blob)
	if len(got.Inventory) != 0 {
		t.Errorf("diamonds still stored: %v", got.Inventory)
	}
	if off := item.ReadNBT(got.OffHand, nil); off.Count() != 2 {
		t.Errorf("off hand = %v", off)
	}
}

func TestLiveInvSee(t *testing.T) {
	w := world.Config{Synchronous: true, Entities: entity.DefaultRegistry}.New()
	defer w.Close()
	<-w.Do(func(tx *world.Tx) {
		target := tx.AddEntity(world.EntitySpawnOpts{}.New(player.Type, player.Config{Name: "g"})).(*player.Player)
		viewer := tx.AddEntity(world.EntitySpawnOpts{}.New(player.Type, player.Config{Name: "h"})).(*player.Player)
		_ = target.Inventory().SetItem(10, item.NewStack(item.Diamond{}, 4))
		_ = target.EnderChestInventory().SetItem(0, item.NewStack(item.Apple{}, 1))

		v := &invSee{st: chestui.NewStorage(invSeeSize, "t"), src: liveSource{h: target.H()}}
		v.sync(tx)
		v.st.Inventory().Handle(v)
		v.st.OnChange(v.changed)
		inv := v.st.Inventory()
		if it, _ := inv.Item(1); it.Count() != 4 {
			t.Fatalf("slot 1 = %v", it)
		}
		if it, _ := inv.Item(invSeeEnder); it.Count() != 1 {
			t.Fatalf("ender 0 = %v", it)
		}

		// The target changes the slot; a move on the stale mirror is
		// refused and the mirror catches up.
		_ = target.Inventory().SetItem(10, item.NewStack(item.Diamond{}, 1))
		ctx := event.C(inventory.Holder(viewer))
		v.HandleTake(ctx, 1, item.NewStack(item.Diamond{}, 4))
		if !ctx.Cancelled() {
			t.Fatal("move on a stale slot was allowed")
		}
		if it, _ := inv.Item(1); it.Count() != 1 {
			t.Fatalf("mirror not synced: %v", it)
		}

		ctx = event.C(inventory.Holder(viewer))
		v.HandleTake(ctx, 1, item.NewStack(item.Diamond{}, 1))
		if ctx.Cancelled() {
			t.Fatal("take was cancelled")
		}
		_ = inv.SetItem(1, item.Stack{})
		if it, _ := target.Inventory().Item(10); !it.Empty() {
			t.Fatalf("target still has %v", it)
		}
	}).Done()
}
