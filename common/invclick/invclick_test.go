package invclick

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
)

func dirt(n int) item.Stack  { return item.NewStack(block.Dirt{}, n) }
func stone(n int) item.Stack { return item.NewStack(block.Stone{}, n) }

func counts(s Slots) []int {
	out := make([]int, len(s))
	for i, st := range s {
		out[i] = st.Count()
	}
	return out
}

func same(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestClick(t *testing.T) {
	for _, c := range []struct {
		name         string
		slot, cursor item.Stack
		right        bool
		wantSlot     int
		wantCursor   int
	}{
		{"left picks the stack", dirt(10), item.Stack{}, false, 0, 10},
		{"right picks half rounded up", dirt(9), item.Stack{}, true, 4, 5},
		{"left places all", item.Stack{}, dirt(7), false, 7, 0},
		{"right places one", item.Stack{}, dirt(7), true, 1, 6},
		{"left merges up to the max", dirt(60), dirt(10), false, 64, 6},
		{"right adds one to the same item", dirt(3), dirt(5), true, 4, 4},
		{"full slot takes nothing", dirt(64), dirt(5), false, 64, 5},
	} {
		s := Slots{c.slot}
		got := Click(s, 0, c.cursor, c.right)
		if s[0].Count() != c.wantSlot || got.Count() != c.wantCursor {
			t.Errorf("%s: slot %d cursor %d, want %d %d", c.name, s[0].Count(), got.Count(), c.wantSlot, c.wantCursor)
		}
	}
}

func TestClickSwapsOtherItems(t *testing.T) {
	s := Slots{stone(3)}
	got := Click(s, 0, dirt(5), false)
	if _, ok := s[0].Item().(block.Dirt); !ok || s[0].Count() != 5 {
		t.Errorf("slot %v", s[0])
	}
	if _, ok := got.Item().(block.Stone); !ok || got.Count() != 3 {
		t.Errorf("cursor %v", got)
	}
}

type refuse struct{ Slots }

func (refuse) Accepts(int, item.Stack) bool { return false }

func TestClickHonoursAccepts(t *testing.T) {
	r := refuse{Slots{item.Stack{}}}
	if got := Click(r, 0, dirt(5), false); got.Count() != 5 || !r.Slots[0].Empty() {
		t.Errorf("a refusing slot took the cursor: %v", r.Slots[0])
	}
	r.Slots[0] = dirt(4)
	if got := Click(r, 0, item.Stack{}, false); got.Count() != 4 {
		t.Errorf("taking out of a refusing slot gave %v", got)
	}
}

func TestCollectTakesSmallStacksFirst(t *testing.T) {
	a := Slots{dirt(64), dirt(3), stone(5), dirt(10)}
	got := Collect(dirt(20), a)
	if got.Count() != 64 || !same(counts(a), []int{33, 0, 5, 0}) {
		t.Errorf("cursor %d, slots %v", got.Count(), counts(a))
	}
}

func TestDoubleClickAfterAPlacingClick(t *testing.T) {
	a := Slots{dirt(28), dirt(29), dirt(1)}
	got := DoubleClick(a, 0, item.Stack{}, a)
	if got.Count() != 58 || !same(counts(a), []int{0, 0, 0}) {
		t.Errorf("cursor %d, slots %v", got.Count(), counts(a))
	}
	b := Slots{dirt(5), dirt(3)}
	if got := DoubleClick(b, 0, dirt(2), b); got.Count() != 10 || b[0].Count() != 0 {
		t.Errorf("with a held stack: cursor %d, slots %v", got.Count(), counts(b))
	}
}

func TestDrag(t *testing.T) {
	a := Slots{item.Stack{}, dirt(2), stone(1), item.Stack{}}
	rest := Drag([]Ref{{a, 0}, {a, 1}, {a, 2}, {a, 3}}, dirt(10), false)
	if rest.Count() != 1 || !same(counts(a), []int{3, 5, 1, 3}) {
		t.Errorf("left drag: rest %d, slots %v", rest.Count(), counts(a))
	}
	b := Slots{item.Stack{}, item.Stack{}}
	rest = Drag([]Ref{{b, 0}, {b, 1}}, dirt(5), true)
	if rest.Count() != 3 || !same(counts(b), []int{1, 1}) {
		t.Errorf("right drag: rest %d, slots %v", rest.Count(), counts(b))
	}
}

func TestMoveFillsStacksBeforeEmptySlots(t *testing.T) {
	a := Slots{item.Stack{}, dirt(60), stone(1)}
	b := Slots{dirt(63)}
	rest := Move(dirt(10), a, b)
	if !rest.Empty() || !same(counts(a), []int{5, 64, 1}) || b[0].Count() != 64 {
		t.Errorf("rest %v, a %v, b %v", rest, counts(a), counts(b))
	}
	if rest := Move(dirt(200), Slots{item.Stack{}}); rest.Count() != 136 {
		t.Errorf("leftover %d, want 136", rest.Count())
	}
}
