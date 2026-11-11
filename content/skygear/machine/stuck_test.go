package machine

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
)

// stuckCrusher is a crusher holding a finished cycle whose output is full.
func stuckCrusher(t *testing.T) *State {
	t.Helper()
	st := New(KindCrusher).State
	for i, slot := range st.Def.Output {
		_ = st.Inv().SetItem(slot, item.NewStack(block.Dirt{}, 64).WithValue("fill", i))
	}
	if !st.Insert(item.NewStack(block.Cobblestone{}, 8)) {
		t.Fatal("cobblestone not accepted")
	}
	st.Progress = st.Def.Interval
	return st
}

func runOnce(st *State) {
	st.Stuck = stuckNone
	runMachine(nil, st, cube.Pos{}, nil)
}

// TestStuckResumesWhenOutputTaken: an output-blocked crusher resumes the tick
// after its output slot empties.
func TestStuckResumesWhenOutputTaken(t *testing.T) {
	st := stuckCrusher(t)
	for range 5 {
		runOnce(st)
		if st.Stuck != StuckOutput {
			t.Fatalf("Stuck=%v, want output", st.Stuck)
		}
	}
	_ = st.Inv().SetItem(st.Def.Output[0], item.Stack{})
	runOnce(st)
	if st.Stuck != stuckNone || st.Progress != 0 {
		t.Fatalf("did not resume: stuck=%v progress=%d", st.Stuck, st.Progress)
	}
	if out, _ := st.Inv().Item(st.Def.Output[0]); out.Empty() {
		t.Fatal("no product made")
	}
}

// TestStuckKeepsStatus: skipping the work of a stuck machine keeps it
// reporting StuckOutput, also past the safety-net recompute.
func TestStuckKeepsStatus(t *testing.T) {
	st := stuckCrusher(t)
	for tick := range 450 {
		st.curTick = int64(tick)
		runOnce(st)
		if st.Stuck != StuckOutput {
			t.Fatalf("tick %d: Stuck=%v", tick, st.Stuck)
		}
	}
	// a stale cache is not trusted forever: if the output opened without a
	// slot change being seen, the safety net finds it
	st.stuckVer = st.invVer
	_ = st.Inv().SetItem(st.Def.Output[0], item.Stack{})
	st.invVer = st.stuckVer
	st.curTick += stuckRecheck
	runOnce(st)
	if st.Stuck != stuckNone {
		t.Fatal("safety net did not recompute")
	}
}

func TestFitsNoAllocs(t *testing.T) {
	st := New(KindCrusher).State
	stacks := []item.Stack{item.NewStack(block.Dirt{}, 3)}
	if !Fits(st.Inv(), st.Def.Output, stacks) {
		t.Fatal("does not fit")
	}
	if n := testing.AllocsPerRun(100, func() { Fits(st.Inv(), st.Def.Output, stacks) }); n != 0 {
		t.Fatalf("allocs=%v", n)
	}
}
