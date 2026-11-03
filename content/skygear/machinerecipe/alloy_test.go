package machinerecipe

import (
	"testing"

	"github.com/df-mc/dragonfly/server/item"
)

func TestMatchAlloyAllocs(t *testing.T) {
	a := item.NewStack(item.CopperIngot{}, 3)
	tin := Alloys[0].B
	bb := item.NewStack(tin, 1)
	if r, ok := MatchAlloy(bb, a); !ok || r.OutCount != 4 {
		t.Fatalf("match: %v %v", r, ok)
	}
	if _, ok := MatchAlloy(a, a); ok {
		t.Fatal("copper+copper matched")
	}
	// Stack.Item encodes the item name, which allocates for metal ingots, so
	// the zero is checked on vanilla items (the steel recipe)
	fe, co := item.NewStack(item.IronIngot{}, 1), item.NewStack(item.Coal{}, 4)
	if r, ok := MatchAlloy(co, fe); !ok || r.OutCount != 1 {
		t.Fatalf("steel: %v %v", r, ok)
	}
	if n := testing.AllocsPerRun(100, func() { MatchAlloy(fe, co) }); n != 0 {
		t.Fatalf("allocs=%v", n)
	}
	if n := testing.AllocsPerRun(100, func() { MatchAlloy(a, bb) }); n > 2 {
		t.Fatalf("allocs=%v", n)
	}
}
