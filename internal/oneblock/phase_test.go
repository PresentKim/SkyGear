package oneblock

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
)

func TestPhasesOrdered(t *testing.T) {
	if Phases[0].Start != 0 {
		t.Fatal("first phase must start at 0")
	}
	for i := 1; i < len(Phases); i++ {
		if Phases[i].Start <= Phases[i-1].Start {
			t.Fatalf("phase %q starts before previous phase", Phases[i].Name)
		}
	}
}

func TestPhaseAt(t *testing.T) {
	for i, p := range Phases {
		if got, _ := PhaseAt(p.Start); got != i {
			t.Errorf("PhaseAt(%d) = %d, want %d", p.Start, got, i)
		}
		if i > 0 {
			if got, _ := PhaseAt(p.Start - 1); got != i-1 {
				t.Errorf("PhaseAt(%d) = %d, want %d", p.Start-1, got, i-1)
			}
		}
	}
}

func TestBlocksAreStable(t *testing.T) {
	for _, p := range Phases {
		for _, e := range p.Blocks {
			if e.Weight <= 0 {
				t.Errorf("phase %q has non-positive weight", p.Name)
			}
			switch e.Block.(type) {
			case block.Sand, block.Gravel, block.ConcretePowder, block.Anvil, block.DragonEgg:
				t.Errorf("phase %q contains gravity block %T", p.Name, e.Block)
			}
		}
	}
	if _, ok := Next(0).(block.Grass); !ok {
		t.Error("first OneBlock must be grass")
	}
}
