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

func TestRightClickHoldingOverAnotherItemDoesNothing(t *testing.T) {
	s := Slots{stone(3)}
	if got := Click(s, 0, dirt(5), true); got.Count() != 5 || s[0].Count() != 3 {
		t.Errorf("cursor %v, slot %v", got, s[0])
	}
}

func split(a Slots, right bool, held item.Stack, cells ...int) (*Split, item.Stack) {
	s := StartSplit(Ref{"a", a, cells[0]}, held, right)
	cur := Click(a, cells[0], held, right)
	for n := 2; n <= len(cells); n++ {
		var refs []Ref
		for _, c := range cells[:n] {
			refs = append(refs, Ref{"a", a, c})
		}
		cur = s.Spread(refs)
	}
	return s, cur
}

func TestSplitSpreadsLiveAsSlotsAreReached(t *testing.T) {
	a := Slots{item.Stack{}, dirt(2), stone(1), item.Stack{}}
	s, cur := split(a, false, dirt(10), 0)
	if !s.Placed() || !cur.Empty() || a[0].Count() != 10 {
		t.Fatalf("the press placed %d, held %d", a[0].Count(), cur.Count())
	}
	c := Slots{item.Stack{}, dirt(2), stone(1), item.Stack{}}
	_, cur = split(c, false, dirt(10), 0, 1)
	if cur.Count() != 0 || !same(counts(c), []int{5, 7, 1, 0}) {
		t.Errorf("two slots: held %d, slots %v", cur.Count(), counts(c))
	}
	b := Slots{item.Stack{}, dirt(2), stone(1), item.Stack{}}
	_, cur = split(b, false, dirt(10), 0, 1, 2, 3)
	if cur.Count() != 1 || !same(counts(b), []int{3, 5, 1, 3}) {
		t.Errorf("four slots: held %d, slots %v", cur.Count(), counts(b))
	}
}

func TestRightSplitPlacesOneEach(t *testing.T) {
	a := Slots{item.Stack{}, item.Stack{}, item.Stack{}}
	_, cur := split(a, true, dirt(5), 0, 1, 2)
	if cur.Count() != 2 || !same(counts(a), []int{1, 1, 1}) {
		t.Errorf("held %d, slots %v", cur.Count(), counts(a))
	}
}

func TestSplitAfterASwapIsNoDrag(t *testing.T) {
	a := Slots{stone(3)}
	s := StartSplit(Ref{"a", a, 0}, dirt(5), false)
	Click(a, 0, dirt(5), false)
	if s.Placed() {
		t.Error("a swap counts as a placing press")
	}
}

func TestMoveAllTakesEveryStackOfTheItem(t *testing.T) {
	a := Slots{dirt(3), stone(2), dirt(64)}
	var got int
	MoveAll(dirt(1), func(st item.Stack) item.Stack { got += st.Count(); return item.Stack{} }, a)
	if got != 67 || !same(counts(a), []int{0, 2, 0}) {
		t.Errorf("moved %d, slots %v", got, counts(a))
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
