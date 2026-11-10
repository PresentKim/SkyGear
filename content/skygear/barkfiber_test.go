package skygear

import (
	"slices"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/itemio"
	"github.com/presentkim/skygear/content/skygear/codex"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// TestBarkFiberFuel: a fiber burns as long as a stick (100 ticks), so the
// generator makes 100 x 20 RE of it, and the codex lists it as a fuel.
func TestBarkFiberFuel(t *testing.T) {
	fiber := item.NewStack(sgitem.BarkFiber{}, 1)
	if d := machine.FuelDuration(fiber); d != sgitem.FiberBurnTicks || d != machine.FuelDuration(item.NewStack(item.Stick{}, 1)) {
		t.Fatalf("a fiber burns %d ticks, want %d (a stick)", d, sgitem.FiberBurnTicks)
	}
	if machine.GeneratorOutput != 20 {
		t.Fatalf("the generator makes %d RE a tick, the draft counts on 20", machine.GeneratorOutput)
	}

	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, machine.KindGenerator, "")
	st.Unlimited = false
	if !st.Insert(item.NewStack(sgitem.BarkFiber{}, 1)) {
		t.Fatal("the generator does not take a fiber as fuel")
	}
	tick(s, 10)
	if st.Energy != 10*machine.GeneratorOutput || st.Burn != sgitem.FiberBurnTicks-10 {
		t.Fatalf("after 10 ticks: %d RE, burn %d", st.Energy, st.Burn)
	}
	tick(s, sgitem.FiberBurnTicks)
	if want := sgitem.FiberBurnTicks * machine.GeneratorOutput; st.Energy != want {
		t.Errorf("a fiber made %d RE, want %d", st.Energy, want)
	}

	if uses := strings.Join(codex.Uses(s.Session, i18n.Default, fiber), "\n"); !strings.Contains(uses, i18n.T(i18n.Default, "codex.use.fuel")) {
		t.Errorf("the codex does not list the fiber as a fuel: %q", uses)
	}
}

// TestBarkFiberIntoFurnace: a hopper or conduit on a furnace's side puts a
// fiber in its fuel slot.
func TestBarkFiberIntoFurnace(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	pos := machinePos.Side(cube.FaceUp)
	s.World.Do(func(tx *world.Tx) {
		tx.SetBlock(pos, block.NewFurnace(cube.North), nil)
		if n := itemio.Insert(tx, pos, cube.FaceEast, item.NewStack(sgitem.BarkFiber{}, 2)); n != 2 {
			t.Fatalf("a side insert put %d fibers in the furnace, want 2", n)
		}
		inv := tx.Block(pos).(block.Furnace).Inventory(tx, pos)
		if fuel, _ := inv.Item(1); fuel.Count() != 2 {
			t.Fatalf("fuel slot holds %v", fuel)
		}
	})
}

// TestFiberPaperRecipe: four fibers in a square make two paper, and no two
// fiber recipes share a shape.
func TestFiberPaperRecipe(t *testing.T) {
	var shapes []recipe.Shape
	found := false
	for _, r := range recipe.Recipes() {
		sh, ok := r.(recipe.Shaped)
		if !ok || !slices.ContainsFunc(sh.Input(), func(in recipe.Item) bool {
			st, ok := in.(item.Stack)
			_, fiber := st.Item().(sgitem.BarkFiber)
			return ok && fiber
		}) {
			continue
		}
		if slices.Contains(shapes, sh.Shape()) {
			t.Errorf("two fiber recipes share the shape %v", sh.Shape())
		}
		shapes = append(shapes, sh.Shape())
		out := sh.Output()[0]
		if _, ok := out.Item().(item.Paper); ok {
			found = true
			if sh.Shape() != recipe.NewShape(2, 2) || len(sh.Input()) != 4 || out.Count() != sgitem.FiberPaperCount || sgitem.FiberPaperCount != 2 {
				t.Errorf("paper recipe %#v", r)
			}
		}
	}
	if !found {
		t.Fatal("no fiber paper recipe")
	}
}

// TestPaperResearch: the "paper" research follows the stone tools (the
// stone axe strips fibers) and opens every paper recipe (the fiber's and the sugar cane's).
func TestPaperResearch(t *testing.T) {
	r, ok := progress.ByID("paper")
	if !ok {
		t.Fatal("no paper research")
	}
	if !slices.Contains(r.Needs, "stone_tools") || !slices.Contains(r.Unlocks, "paper") {
		t.Fatalf("paper research %+v", r)
	}
}
