package dragonfactory

import (
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
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
			st.setLab(labByKind(r.Out.Kind))
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
	st.setLab(&labRecipes[3])
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
	st.setLab(&labRecipes[0])
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

// TestLabAccepts: the lab takes nothing until an essence is chosen, then the
// ingredients of that recipe only.
func TestLabAccepts(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindLab, "")
	if st.accepts(item.NewStack(item.GlassBottle{}, 1)) || st.insert(item.NewStack(item.GlassBottle{}, 1)) {
		t.Fatal("takes a glass bottle with no essence chosen")
	}
	st.setLab(labByKind(EssenceHunting))
	for _, in := range []world.Item{item.GlassBottle{}, item.Gunpowder{}, item.SpiderEye{}} {
		if !st.accepts(item.NewStack(in, 1)) {
			t.Errorf("does not take %s", itemKey(in))
		}
	}
	for _, in := range []world.Item{block.Dirt{}, item.Diamond{}, item.IronIngot{}, Petal{Colour: "blue"}} {
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
	st.setLab(&labRecipes[1])
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
// worth, so the other ingredients still find a slot.
func TestLabFeedLeavesRoom(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindLab, "")
	st.setLab(labByKind(EssenceEngineering))
	iron := item.NewStack(item.IronIngot{}, 1)
	fed := 0
	for st.insert(iron) {
		fed++
		if fed > 1000 {
			t.Fatal("the feed never stopped")
		}
	}
	if fed != st.labFeedCap(iron) {
		t.Fatalf("fed %d iron, want the cap %d", fed, st.labFeedCap(iron))
	}
	if !st.insert(item.NewStack(item.RedstoneWire{}, 1)) || !st.insert(item.NewStack(item.GlassBottle{}, 1)) {
		t.Fatal("the other ingredients found no room")
	}
}

// TestLabScreen: the cards are the essences the research has come to;
// choosing one puts its ingredients in from the inventory (and nothing
// else), the essences made are handed out, and stopping gives everything
// back and lets the lab take nothing.
func TestLabScreen(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindLab, "")
	conduit, _ := researchByID("energy_conduit")
	s.state.Researched = map[string]bool{}
	for _, id := range conduit.Needs {
		s.state.Researched[id] = true
	}
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		opts := s.labChoices(p, machinePos, st)
		if len(opts) != 1 || !strings.HasPrefix(opts[0].Label, "공학의 정수") {
			var labels []string
			for _, o := range opts {
				labels = append(labels, o.Label)
			}
			t.Fatalf("cards %q, want the essence of engineering", labels)
		}
		_, _ = p.Inventory().AddItem(item.NewStack(item.GlassBottle{}, 5))
		_, _ = p.Inventory().AddItem(item.NewStack(item.IronIngot{}, 200))
		_, _ = p.Inventory().AddItem(item.NewStack(item.RedstoneWire{}, 7))
		_, _ = p.Inventory().AddItem(item.NewStack(item.Diamond{}, 3))
		opts[0].Run(p, tx)
		have := st.labHave()
		if st.lab == nil || have[itemKey(item.GlassBottle{})] != 5 || have[itemKey(item.RedstoneWire{})] != 7 || have[itemKey(item.Diamond{})] != 0 {
			t.Fatalf("chosen %v, holds %v", st.lab != nil, have)
		}
		if n := have[itemKey(item.IronIngot{})]; n != st.labHandCap(item.NewStack(item.IronIngot{}, 1)) {
			t.Fatalf("holds %d iron, want the hand cap", n)
		}
		_ = st.inv().SetItem(st.kind.output[0], item.NewStack(ResearchEssence{EssenceEngineering}, 4))
		_, _, actions := s.labPanel(p, machinePos, st)
		press(t, p, tx, actions, "정수 꺼내기")
		if outputCount(st, ResearchEssence{EssenceEngineering}) != 0 || countIn(p, ResearchEssence{EssenceEngineering}) != 4 {
			t.Fatal("the essences were not handed out")
		}
		press(t, p, tx, actions, "§c생산 중지")
		if st.lab != nil || len(st.labHave()) != 0 || countIn(p, item.GlassBottle{}) != 5 || countIn(p, item.IronIngot{}) != 200 {
			t.Fatalf("stopped: chosen %v, holds %v, bottles %d, iron %d", st.lab != nil, st.labHave(), countIn(p, item.GlassBottle{}), countIn(p, item.IronIngot{}))
		}
		if st.insert(item.NewStack(item.GlassBottle{}, 1)) {
			t.Fatal("a stopped lab takes a glass bottle")
		}
	})
}

// countIn counts the items of a kind in p's inventory.
func countIn(p *player.Player, it world.Item) int {
	n := 0
	for _, st := range p.Inventory().Items() {
		if itemKey(st.Item()) == itemKey(it) {
			n += st.Count()
		}
	}
	return n
}

// TestLabSaved: the chosen essence is saved, and a lab saved before it was
// chosen brews what its ingredients are for.
func TestLabSaved(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindLab, "")
	st.setLab(labByKind(EssenceMagic))
	back := Machine{Kind: KindLab}.DecodeNBT(Machine{Kind: KindLab, st: st}.EncodeNBT()).(Machine)
	if back.st.lab == nil || back.st.lab.Out.Kind != EssenceMagic {
		t.Fatal("the chosen essence was not saved")
	}
	st.setLab(nil)
	_ = st.inv().SetItem(st.kind.input[0], item.NewStack(item.Bone{}, 2))
	back = Machine{Kind: KindLab}.DecodeNBT(Machine{Kind: KindLab, st: st}.EncodeNBT()).(Machine)
	if back.st.lab == nil || back.st.lab.Out.Kind != EssenceHunting {
		t.Fatal("an old lab with bones does not brew the essence of hunting")
	}
}
