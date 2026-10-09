package timed

import (
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
)

func at(t *testing.T, now time.Time) {
	old := Now
	Now = func() time.Time { return now }
	t.Cleanup(func() { Now = old })
}

func TestExpiryAndLore(t *testing.T) {
	start := time.Date(2026, 10, 9, 12, 0, 0, 0, Zone)
	at(t, start)
	st := For(item.NewStack(item.Apple{}, 3).WithLore("§7맛있는 사과"), time.Hour)
	if exp, ok := Expiry(st); !ok || !exp.Equal(start.Add(time.Hour)) {
		t.Fatalf("expiry %v %v", exp, ok)
	}
	if lore := st.Lore(); len(lore) != 2 || lore[0] != "§7맛있는 사과" || lore[1] != lorePrefix+"2026-10-09 13:00까지" {
		t.Fatalf("lore %q", lore)
	}
	// A new period replaces the old line instead of adding one.
	st = For(st, 2*time.Hour)
	if lore := st.Lore(); len(lore) != 2 || lore[1] != lorePrefix+"2026-10-09 14:00까지" {
		t.Fatalf("lore after extending %q", lore)
	}
	if Expired(st) {
		t.Fatal("expired too early")
	}
	if d, _ := Remaining(st); d != 2*time.Hour {
		t.Fatalf("remaining %v", d)
	}
	at(t, start.Add(2*time.Hour))
	if !Expired(st) {
		t.Fatal("not expired at its time")
	}
	if c := Clear(st); Is(c) || len(c.Lore()) != 1 {
		t.Fatalf("cleared %v", c)
	}
	if Is(item.NewStack(item.Apple{}, 1)) || Expired(item.NewStack(item.Apple{}, 1)) {
		t.Fatal("a plain item is timed")
	}
}

// TestExpirySurvivesNBT: the expiry is saved with the item, so it holds in
// player states, containers and the lost-and-found box.
func TestExpirySurvivesNBT(t *testing.T) {
	at(t, time.Unix(1_800_000_000, 0))
	st := For(item.NewStack(item.Apple{}, 1), time.Minute)
	back := item.ReadNBT(item.WriteNBT(st, true), nil)
	if exp, ok := Expiry(back); !ok || exp.Unix() != 1_800_000_060 {
		t.Fatalf("after NBT: %v %v", exp, ok)
	}
	if !back.Comparable(st) {
		t.Fatal("round trip changed the stack")
	}
}

func TestSweepRemovesOnlyExpired(t *testing.T) {
	start := time.Unix(1_800_000_000, 0)
	at(t, start)
	inv := inventory.New(4, nil)
	_ = inv.SetItem(0, For(item.NewStack(item.Apple{}, 2), time.Minute))
	_ = inv.SetItem(1, For(item.NewStack(item.Apple{}, 1), time.Hour))
	_ = inv.SetItem(2, item.NewStack(item.Apple{}, 5))
	if got := Sweep(inv); len(got) != 0 {
		t.Fatalf("swept %v before expiry", got)
	}
	at(t, start.Add(time.Minute))
	got := Sweep(inv)
	if len(got) != 1 || got[0].Count() != 2 {
		t.Fatalf("swept %v", got)
	}
	if it, _ := inv.Item(0); !it.Empty() {
		t.Fatal("expired stack left")
	}
	if it, _ := inv.Item(1); it.Empty() {
		t.Fatal("unexpired stack removed")
	}
	if it, _ := inv.Item(2); it.Count() != 5 {
		t.Fatal("plain stack touched")
	}
}

func TestDifferentExpiriesDoNotStack(t *testing.T) {
	at(t, time.Unix(1_800_000_000, 0))
	a := For(item.NewStack(item.Apple{}, 1), time.Minute)
	b := For(item.NewStack(item.Apple{}, 1), time.Hour)
	if a.Comparable(b) || a.Comparable(item.NewStack(item.Apple{}, 1)) {
		t.Fatal("stacks with different periods would merge")
	}
}

func TestDescribe(t *testing.T) {
	for d, want := range map[time.Duration]string{
		0: "만료", 30 * time.Second: "30초", 5 * time.Minute: "5분",
		2*time.Hour + 5*time.Minute: "2시간 5분", 50 * time.Hour: "2일 2시간",
	} {
		if got := Describe(d); got != want {
			t.Errorf("Describe(%v) = %q, want %q", d, got, want)
		}
	}
}
