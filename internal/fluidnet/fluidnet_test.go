package fluidnet

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

type fakeWorld map[cube.Pos]world.Block

func (w fakeWorld) Block(p cube.Pos) world.Block {
	if b, ok := w[p]; ok {
		return b
	}
	return block.Air{}
}

type pipe struct{ block.Air }

func (pipe) FluidTier() int { return 1 }

type tank struct {
	block.Air
	kind         Kind
	amount, cap  int
	source, sink bool
}

func (t *tank) FluidRoles() (bool, bool) { return t.source, t.sink }
func (t *tank) FillFluid(_ World, _ cube.Pos, k Kind, mB int, simulate bool) int {
	if !t.sink || (t.amount > 0 && t.kind != k) {
		return 0
	}
	n := min(mB, t.cap-t.amount)
	if !simulate && n > 0 {
		t.kind, t.amount = k, t.amount+n
	}
	return n
}
func (t *tank) DrainFluid(_ World, _ cube.Pos, max int, simulate bool) (Kind, int) {
	if !t.source || t.amount == 0 {
		return None, 0
	}
	n := min(max, t.amount)
	k := t.kind
	if !simulate {
		t.amount -= n
	}
	return k, n
}

// TestStepMovesFluid: a tank full of lava empties into the tanks beside the
// pipe, at most a step's budget, and keeps the fluids apart.
func TestStepMovesFluid(t *testing.T) {
	src := &tank{kind: Lava, amount: 10000, cap: 10000, source: true}
	a := &tank{cap: 100000, sink: true}
	b := &tank{cap: 100000, sink: true}
	w := fakeWorld{{0, 0, 0}: src, {1, 0, 0}: pipe{}, {1, 0, 1}: a, {1, 0, -1}: b, {2, 0, 0}: &tank{kind: Water, amount: 5, cap: 100, sink: true}}
	g := NewGrid()
	g.AddConduit(cube.Pos{1, 0, 0})
	moved := g.Step(w)
	if budget := RateOf(1) * StepTicks; moved != budget {
		t.Fatalf("moved %d, want the budget %d", moved, budget)
	}
	if src.amount != 10000-moved || a.amount+b.amount != moved {
		t.Errorf("source %d, sinks %d and %d after %d", src.amount, a.amount, b.amount, moved)
	}
	if a.amount == 0 || b.amount == 0 {
		t.Errorf("the sinks did not share: %d and %d", a.amount, b.amount)
	}
	if w[cube.Pos{2, 0, 0}].(*tank).amount != 5 {
		t.Error("lava went into the tank of water")
	}
}

func TestRates(t *testing.T) {
	if RateOf(1) != 200 || RateOf(2) != 1000 || RateOf(3) != 8000 || RateOf(9) != 8000 {
		t.Errorf("rates %v", Rates)
	}
}
