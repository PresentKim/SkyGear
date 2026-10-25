package dragonfactory

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

func withOreInv(t *testing.T, f func(p *player.Player, st *oreInv)) {
	t.Helper()
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, _ *world.Tx) { f(p, &oreInv{g: s.g}) })
}

func slotsOf(ids ...string) map[string]any {
	list := make([]any, len(ids))
	for i, id := range ids {
		list[i] = id
	}
	return map[string]any{"slots": list}
}

func countAt(p *player.Player, id string) int {
	s, _ := parseOreSlot(id)
	return s.get(p).Count()
}

func setAt(p *player.Player, id string, st item.Stack) {
	s, _ := parseOreSlot(id)
	s.set(p, st)
}

func TestOreInvLeftPickAndPlace(t *testing.T) {
	withOreInv(t, func(p *player.Player, st *oreInv) {
		setAt(p, "h0", item.NewStack(block.Dirt{}, 10))
		st.press(p, oreSlot{'h', 0}, 0, false, false)
		st.release(p, slotsOf("h0"), 0)
		st.press(p, oreSlot{'i', 0}, 0, false, false)
		st.release(p, slotsOf("i0"), 0)
		if countAt(p, "h0") != 0 || countAt(p, "i0") != 10 || !st.cursor.Empty() {
			t.Fatalf("h0=%d i0=%d cursor=%v", countAt(p, "h0"), countAt(p, "i0"), st.cursor)
		}
	})
}

func TestOreInvRightTakesHalfAndPlacesOne(t *testing.T) {
	withOreInv(t, func(p *player.Player, st *oreInv) {
		setAt(p, "h0", item.NewStack(block.Dirt{}, 9))
		st.press(p, oreSlot{'h', 0}, 2, false, false)
		if st.cursor.Count() != 5 || countAt(p, "h0") != 4 {
			t.Fatalf("cursor=%d h0=%d", st.cursor.Count(), countAt(p, "h0"))
		}
		st.press(p, oreSlot{'h', 1}, 2, false, false)
		st.release(p, slotsOf("h1"), 2)
		if st.cursor.Count() != 4 || countAt(p, "h1") != 1 {
			t.Fatalf("cursor=%d h1=%d", st.cursor.Count(), countAt(p, "h1"))
		}
	})
}

func TestOreInvDragSplitsEvenly(t *testing.T) {
	withOreInv(t, func(p *player.Player, st *oreInv) {
		st.cursor = item.NewStack(block.Dirt{}, 10)
		st.press(p, oreSlot{'i', 0}, 0, false, false)
		st.release(p, slotsOf("i0", "i1", "i2"), 0)
		if countAt(p, "i0") != 3 || countAt(p, "i1") != 3 || countAt(p, "i2") != 3 || st.cursor.Count() != 1 {
			t.Fatalf("%d %d %d cursor=%d", countAt(p, "i0"), countAt(p, "i1"), countAt(p, "i2"), st.cursor.Count())
		}
	})
}

func TestOreInvRightDragPlacesOneEach(t *testing.T) {
	withOreInv(t, func(p *player.Player, st *oreInv) {
		st.cursor = item.NewStack(block.Dirt{}, 10)
		st.press(p, oreSlot{'i', 0}, 2, false, false)
		st.release(p, slotsOf("i0", "i1"), 2)
		if countAt(p, "i0") != 1 || countAt(p, "i1") != 1 || st.cursor.Count() != 8 {
			t.Fatalf("%d %d cursor=%d", countAt(p, "i0"), countAt(p, "i1"), st.cursor.Count())
		}
	})
}

func TestOreInvShiftMergesIntoInventoryFirst(t *testing.T) {
	withOreInv(t, func(p *player.Player, st *oreInv) {
		setAt(p, "h0", item.NewStack(block.Dirt{}, 10))
		setAt(p, "i5", item.NewStack(block.Dirt{}, 60))
		st.press(p, oreSlot{'h', 0}, 0, true, false)
		if countAt(p, "h0") != 0 || countAt(p, "i5") != 64 || countAt(p, "i0") != 6 {
			t.Fatalf("h0=%d i5=%d i0=%d", countAt(p, "h0"), countAt(p, "i5"), countAt(p, "i0"))
		}
	})
}

func TestOreInvDoubleClickCollects(t *testing.T) {
	withOreInv(t, func(p *player.Player, st *oreInv) {
		setAt(p, "h3", item.NewStack(block.Dirt{}, 7))
		setAt(p, "i8", item.NewStack(block.Dirt{}, 5))
		st.cursor = item.NewStack(block.Dirt{}, 2)
		st.press(p, oreSlot{'h', 0}, 0, false, true)
		st.release(p, slotsOf("h0"), 0)
		if st.cursor.Count() != 14 || countAt(p, "h3") != 0 || countAt(p, "i8") != 0 || countAt(p, "h0") != 0 {
			t.Fatalf("cursor=%d", st.cursor.Count())
		}
	})
}

func TestOreInvArmourSlotRefusesOtherItems(t *testing.T) {
	withOreInv(t, func(p *player.Player, st *oreInv) {
		st.cursor = item.NewStack(block.Dirt{}, 1)
		st.press(p, oreSlot{'a', 0}, 0, false, false)
		st.release(p, slotsOf("a0"), 0)
		if countAt(p, "a0") != 0 || st.cursor.Count() != 1 {
			t.Fatalf("a0=%d", countAt(p, "a0"))
		}
		st.cursor = item.NewStack(item.Helmet{Tier: item.ArmourTierIron{}}, 1)
		st.release(p, slotsOf("a0"), 0)
		st.press(p, oreSlot{'a', 0}, 0, false, false)
		st.release(p, slotsOf("a0"), 0)
		if countAt(p, "a0") != 1 || !st.cursor.Empty() {
			t.Fatalf("a0=%d cursor=%v", countAt(p, "a0"), st.cursor)
		}
	})
}

func TestOreInvNumberKeySwapsWithHotbar(t *testing.T) {
	withOreInv(t, func(p *player.Player, st *oreInv) {
		setAt(p, "i0", item.NewStack(block.Dirt{}, 4))
		setAt(p, "h2", item.NewStack(block.Stone{}, 1))
		st.key(p, oreSlot{'i', 0}, oreKey1+2, false)
		a, _ := parseOreSlot("i0")
		b, _ := parseOreSlot("h2")
		if _, ok := a.get(p).Item().(block.Stone); !ok || b.get(p).Count() != 4 {
			t.Fatalf("i0=%v h2=%v", a.get(p), b.get(p))
		}
	})
}
