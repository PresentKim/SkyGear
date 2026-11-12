package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/presentkim/skygear/content/skygear/elevator"
)

// TestElevatorRecipes: eight wool of a colour around an ender pearl make
// the elevator of that colour.
func TestElevatorRecipes(t *testing.T) {
	var elevators int
	for _, r := range recipe.Recipes() {
		out := r.Output()
		if len(out) != 1 {
			continue
		}
		o, ok := out[0].Item().(elevator.Elevator)
		if !ok {
			continue
		}
		in := r.Input()
		if len(in) != 9 {
			t.Fatalf("elevator recipe has %d cells", len(in))
		}
		if st := in[4].(item.Stack); st.Item() != (item.EnderPearl{}) {
			t.Errorf("%v: centre is %v, want an ender pearl", o.Colour, st.Item())
		}
		if st := in[0].(item.Stack); st.Item() != (block.Wool{Colour: o.Colour}) {
			t.Errorf("%v: made of %v, want its wool", o.Colour, st.Item())
		}
		elevators++
	}
	if elevators != len(item.Colours()) {
		t.Fatalf("%d elevator recipes, want one per colour", elevators)
	}
}
