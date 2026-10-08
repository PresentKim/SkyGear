package chestui

import (
	"testing"

	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/event"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

func TestTitle(t *testing.T) {
	const r9 = "§r§r§r§r§r§r§r§r§r"
	for _, c := range []struct {
		size int
		want string
	}{
		{1, "§1§0" + r9 + "x"},
		{9, "§1§0" + r9 + "x"},
		{10, "§2§0" + r9 + "x"},
		{27, "§3§0" + r9 + "x"},
		{54, "§6§0" + r9 + "x"},
		{55, "§6§1" + r9 + "x"},
		{200, "§6§1" + r9 + "x"},
	} {
		if got := Title(c.size, "x"); got != c.want {
			t.Errorf("Title(%d) = %q, want %q", c.size, got, c.want)
		}
	}
}

type countViewer struct {
	world.NopViewer
	slots []int
}

func (v *countViewer) ViewSlotChange(slot int, _ item.Stack) { v.slots = append(v.slots, slot) }

func TestStorageViewers(t *testing.T) {
	s := NewStorage(30, "t")
	v := &countViewer{}
	s.viewers[v] = nil
	var changed []int
	s.OnChange(func(slot int, _, _ item.Stack) { changed = append(changed, slot) })
	_ = s.Inventory().SetItem(29, item.NewStack(item.Apple{}, 3))
	if len(v.slots) != 1 || v.slots[0] != 29 || len(changed) != 1 {
		t.Fatalf("viewer saw %v, onChange %v", v.slots, changed)
	}
}

func TestMenuClick(t *testing.T) {
	w := world.Config{Synchronous: true, Entities: entity.DefaultRegistry}.New()
	defer w.Close()

	m := NewMenu(18, "menu")
	clicks := 0
	m.Set(4, Button{Item: item.NewStack(item.Apple{}, 1), Run: func(*player.Player, *world.Tx) { clicks++ }})
	if st, _ := m.Inventory().Item(4); st.Empty() {
		t.Fatal("button item not set")
	}

	<-w.Do(func(tx *world.Tx) {
		p := tx.AddEntity(world.EntitySpawnOpts{Position: mgl64.Vec3{0, 0, 0}}.New(player.Type, player.Config{Name: "p"})).(*player.Player)
		h := m.Inventory().Handler()
		for name, f := range map[string]func(*inventory.Context, int, item.Stack){"place": h.HandlePlace, "drop": h.HandleDrop, "take": h.HandleTake} {
			ctx := event.C(inventory.Holder(p))
			f(ctx, 4, item.NewStack(item.Apple{}, 1))
			if !ctx.Cancelled() {
				t.Errorf("%s was not cancelled", name)
			}
		}
		// An empty slot does nothing but is still cancelled.
		ctx := event.C(inventory.Holder(p))
		h.HandleTake(ctx, 5, item.Stack{})
		if !ctx.Cancelled() {
			t.Error("take of empty slot was not cancelled")
		}
	}).Done()
	// The button runs in a later transaction.
	<-w.Do(func(tx *world.Tx) {}).Done()
	if clicks != 1 {
		t.Fatalf("clicks = %d, want 1", clicks)
	}
}
