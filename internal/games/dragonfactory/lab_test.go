package dragonfactory

import (
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

// TestLabRecipes: the table makes every research essence once, every recipe
// takes a glass bottle and distinct ingredients that fit the input slots.
func TestLabRecipes(t *testing.T) {
	made := map[string]bool{}
	for _, r := range labRecipes {
		if made[r.Out.Kind] {
			t.Errorf("%s is made by two recipes", r.Out.Kind)
		}
		made[r.Out.Kind] = true
		if r.OutCount < 1 || r.Ticks < 1 {
			t.Errorf("%s: count %d, ticks %d", r.Out.Kind, r.OutCount, r.Ticks)
		}
		if len(r.In) > len(machineKinds[KindLab].input) {
			t.Errorf("%s: %d ingredients for %d slots", r.Out.Kind, len(r.In), len(machineKinds[KindLab].input))
		}
		keys := map[string]bool{}
		for _, in := range r.In {
			k := itemKey(in.Item)
			if keys[k] || in.Count < 1 {
				t.Errorf("%s: ingredient %s twice or empty", r.Out.Kind, k)
			}
			keys[k] = true
		}
		if !keys[itemKey(item.GlassBottle{})] {
			t.Errorf("%s: no glass bottle", r.Out.Kind)
		}
	}
	for _, k := range researchEssenceKinds {
		if !made[k] {
			t.Errorf("no recipe makes the essence of %s", k)
		}
	}
}

// fillLab puts the ingredients of a recipe, times sets of them, in the input slots.
func fillLab(st *machineState, r LabRecipe, sets int) {
	for i, in := range r.In {
		_ = st.inv().SetItem(st.kind.input[i], item.NewStack(in.Item, in.Count*sets))
	}
}

// TestLabBrews: every recipe brews its essence in its own time and uses its
// ingredients up.
func TestLabBrews(t *testing.T) {
	for _, r := range labRecipes {
		t.Run(r.Out.Kind, func(t *testing.T) {
			s := testSession(t)
			loadAround(s)
			st := placeTestMachine(s, machinePos, KindLab, "")
			fillLab(st, r, 2)
			tick(s, r.Ticks/2)
			if got := outputCount(st, r.Out); got != 0 {
				t.Fatalf("%d essences half way", got)
			}
			tick(s, r.Ticks/2+5)
			if got := outputCount(st, r.Out); got != r.OutCount {
				t.Fatalf("%d essences after a cycle, want %d", got, r.OutCount)
			}
			for i, in := range r.In {
				if got, _ := st.inv().Item(st.kind.input[i]); got.Count() != in.Count {
					t.Errorf("%s: %d left, want %d", itemKey(in.Item), got.Count(), in.Count)
				}
			}
			tick(s, r.Ticks+5)
			if got := outputCount(st, r.Out); got != 2*r.OutCount {
				t.Fatalf("%d essences after two cycles, want %d", got, 2*r.OutCount)
			}
			tick(s, r.Ticks+5)
			if got := outputCount(st, r.Out); got != 2*r.OutCount {
				t.Fatalf("%d essences without ingredients, want %d", got, 2*r.OutCount)
			}
		})
	}
}

// TestLabWaits: without every ingredient nothing brews and the panel says
// what is missing; without room for the essences it stops.
func TestLabWaits(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindLab, "")
	hunting := labRecipes[3]
	_ = st.inv().SetItem(st.kind.input[0], item.NewStack(item.GlassBottle{}, 4))
	_ = st.inv().SetItem(st.kind.input[1], item.NewStack(item.Bone{}, 1))
	tick(s, hunting.Ticks+10)
	if st.progress != 0 || outputCount(st, hunting.Out) != 0 {
		t.Fatalf("brewed without the ingredients: progress %d", st.progress)
	}
	lines := strings.Join(labLines(st), "\n")
	if !strings.Contains(lines, "재료 부족") || !strings.Contains(lines, "화약 ×2") || !strings.Contains(lines, "뼈다귀 ×1") {
		t.Fatalf("lines %q", lines)
	}
}

// TestLabStopsWithoutRoom: a full output keeps the finished cycle waiting,
// and it goes on when room is made.
func TestLabStopsWithoutRoom(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindLab, "")
	r := labRecipes[0]
	for _, slot := range st.kind.output {
		_ = st.inv().SetItem(slot, item.NewStack(block.Cobblestone{}, 64))
	}
	fillLab(st, r, 1)
	tick(s, r.Ticks+20)
	if outputCount(st, r.Out) != 0 || st.progress < r.Ticks {
		t.Fatalf("progress %d with a full output", st.progress)
	}
	if lines := strings.Join(labLines(st), "\n"); !strings.Contains(lines, "출력 슬롯 부족") {
		t.Fatalf("lines %q", lines)
	}
	_ = st.inv().SetItem(st.kind.output[0], item.Stack{})
	tick(s, 5)
	if outputCount(st, r.Out) != r.OutCount {
		t.Fatal("did not go on once there was room")
	}
}

// TestLabAccepts: the lab takes the ingredients of its recipes, and nothing
// else.
func TestLabAccepts(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindLab, "")
	for _, in := range []world.Item{item.GlassBottle{}, item.IronIngot{}, item.SpiderEye{}, Petal{Colour: "blue"}, Ingot{Metal: MetalInvar}} {
		if !st.accepts(item.NewStack(in, 1)) {
			t.Errorf("does not take %s", itemKey(in))
		}
	}
	for _, in := range []world.Item{block.Dirt{}, item.Diamond{}, Petal{Colour: "pink"}} {
		if st.accepts(item.NewStack(in, 1)) {
			t.Errorf("takes %s", itemKey(in))
		}
	}
}

// TestFarmingIngredientsGrow: the crops the essence of farming takes are
// what the six crops of the island give.
func TestFarmingIngredientsGrow(t *testing.T) {
	grown := map[string]bool{}
	for _, b := range []world.Block{withGrowth(block.WheatSeeds{}, 7), withGrowth(block.Carrot{}, 7), withGrowth(block.Potato{}, 7), withGrowth(block.BeetrootSeeds{}, 7), block.Melon{}, block.Pumpkin{}} {
		for _, st := range b.(block.Breakable).BreakInfo().Drops(item.ToolNone{}, nil) {
			grown[itemKey(st.Item())] = true
		}
	}
	for _, in := range labRecipes[2].In[1:] {
		if !grown[itemKey(in.Item)] {
			t.Errorf("%s does not grow on the island's crops", itemKey(in.Item))
		}
	}
}

// TestLabNoticesAnIngredientTakenAway: an ingredient taken out during a cycle
// stops the brewing, even though the slots are only looked at now and then.
func TestLabNoticesAnIngredientTakenAway(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindLab, "")
	r := labRecipes[1]
	fillLab(st, r, 1)
	tick(s, r.Ticks/2)
	if st.progress == 0 {
		t.Fatal("did not start")
	}
	_ = st.inv().SetItem(st.kind.input[1], item.Stack{})
	tick(s, r.Ticks)
	if outputCount(st, r.Out) != 0 || st.progress != 0 {
		t.Fatalf("brewed without an ingredient: %d essences, progress %d", outputCount(st, r.Out), st.progress)
	}
}

// TestLabFeedLeavesRoom: a feed of one ingredient stops at a few cycles'
// worth, so the other ingredients still find a slot; by hand any amount goes.
func TestLabFeedLeavesRoom(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindLab, "")
	fed := 0
	for st.insert(item.NewStack(item.IronIngot{}, 1)) {
		fed++
		if fed > 1000 {
			t.Fatal("the feed never stopped")
		}
	}
	if fed != labFeedCap(item.NewStack(item.IronIngot{}, 1)) {
		t.Fatalf("fed %d iron, want the cap %d", fed, labFeedCap(item.NewStack(item.IronIngot{}, 1)))
	}
	if !st.insert(item.NewStack(item.RedstoneWire{}, 1)) || !st.insert(item.NewStack(item.GlassBottle{}, 1)) {
		t.Fatal("the other ingredients found no room")
	}
	_ = st.inv().SetItem(st.kind.input[8], item.NewStack(item.IronIngot{}, 64))
	if st.labHave()[itemKey(item.IronIngot{})] <= fed {
		t.Fatal("a stack put in by hand was refused")
	}
}
