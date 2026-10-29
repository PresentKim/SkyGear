package skygear

import (
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/farm"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/machinerecipe"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// TestLabRecipes: the table makes every research essence once, every recipe
// takes a glass bottle and distinct ingredients that fit the input slots.
func TestLabRecipes(t *testing.T) {
	made := map[string]bool{}
	for _, r := range machinerecipe.Labs {
		if made[r.Out.Kind] {
			t.Errorf("%s is made by two recipes", r.Out.Kind)
		}
		made[r.Out.Kind] = true
		if r.OutCount < 1 || r.Ticks < 1 {
			t.Errorf("%s: count %d, ticks %d", r.Out.Kind, r.OutCount, r.Ticks)
		}
		if len(r.In) > len(machine.Kinds[machine.KindLab].Input) {
			t.Errorf("%s: %d ingredients for %d slots", r.Out.Kind, len(r.In), len(machine.Kinds[machine.KindLab].Input))
		}
		keys := map[string]bool{}
		for _, in := range r.In {
			k := naming.ItemKey(in.Item)
			if keys[k] || in.Count < 1 {
				t.Errorf("%s: ingredient %s twice or empty", r.Out.Kind, k)
			}
			keys[k] = true
		}
		if !keys[naming.ItemKey(item.GlassBottle{})] {
			t.Errorf("%s: no glass bottle", r.Out.Kind)
		}
	}
	for _, k := range sgitem.ResearchEssenceKinds {
		if !made[k] {
			t.Errorf("no recipe makes the essence of %s", k)
		}
	}
}

// fillLab puts the ingredients of a recipe, times sets of them, in the input slots.
func fillLab(st *machine.State, r machinerecipe.Lab, sets int) {
	for i, in := range r.In {
		_ = st.Inv().SetItem(st.Def.Input[i], item.NewStack(in.Item, in.Count*sets))
	}
}

// TestLabBrews: every recipe brews its essence in its own time and uses its
// ingredients up.
func TestLabBrews(t *testing.T) {
	for _, r := range machinerecipe.Labs {
		t.Run(r.Out.Kind, func(t *testing.T) {
			s := testSession(t)
			loadAround(s)
			st := placeTestMachine(s, machinePos, machine.KindLab, "")
			st.SetLab(machinerecipe.LabByKind(r.Out.Kind))
			fillLab(st, r, 2)
			tick(s, r.Ticks/2)
			if got := machine.OutputCount(st, r.Out); got != 0 {
				t.Fatalf("%d essences half way", got)
			}
			tick(s, r.Ticks/2+5)
			if got := machine.OutputCount(st, r.Out); got != r.OutCount {
				t.Fatalf("%d essences after a cycle, want %d", got, r.OutCount)
			}
			for i, in := range r.In {
				if got, _ := st.Inv().Item(st.Def.Input[i]); got.Count() != in.Count {
					t.Errorf("%s: %d left, want %d", naming.ItemKey(in.Item), got.Count(), in.Count)
				}
			}
			tick(s, r.Ticks+5)
			if got := machine.OutputCount(st, r.Out); got != 2*r.OutCount {
				t.Fatalf("%d essences after two cycles, want %d", got, 2*r.OutCount)
			}
			tick(s, r.Ticks+5)
			if got := machine.OutputCount(st, r.Out); got != 2*r.OutCount {
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
	st := placeTestMachine(s, machinePos, machine.KindLab, "")
	hunting := machinerecipe.Labs[3]
	st.SetLab(&machinerecipe.Labs[3])
	_ = st.Inv().SetItem(st.Def.Input[0], item.NewStack(item.GlassBottle{}, 4))
	_ = st.Inv().SetItem(st.Def.Input[1], item.NewStack(item.Bone{}, 1))
	tick(s, hunting.Ticks+10)
	if st.Progress != 0 || machine.OutputCount(st, hunting.Out) != 0 {
		t.Fatalf("brewed without the ingredients: progress %d", st.Progress)
	}
	lines := strings.Join(machine.LabLines(i18n.Default, st), "\n")
	if !strings.Contains(lines, "재료 부족") || !strings.Contains(lines, "화약 ×2") || !strings.Contains(lines, "뼈다귀 ×1") {
		t.Fatalf("lines %q", lines)
	}
}

// TestLabStopsWithoutRoom: a full output keeps the finished cycle waiting,
// and it goes on when room is made.
func TestLabStopsWithoutRoom(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, machine.KindLab, "")
	r := machinerecipe.Labs[0]
	st.SetLab(&machinerecipe.Labs[0])
	for _, slot := range st.Def.Output {
		_ = st.Inv().SetItem(slot, item.NewStack(block.Cobblestone{}, 64))
	}
	fillLab(st, r, 1)
	tick(s, r.Ticks+20)
	if machine.OutputCount(st, r.Out) != 0 || st.Progress < r.Ticks {
		t.Fatalf("progress %d with a full output", st.Progress)
	}
	if lines := strings.Join(machine.LabLines(i18n.Default, st), "\n"); !strings.Contains(lines, "출력 슬롯 부족") {
		t.Fatalf("lines %q", lines)
	}
	_ = st.Inv().SetItem(st.Def.Output[0], item.Stack{})
	tick(s, 5)
	if machine.OutputCount(st, r.Out) != r.OutCount {
		t.Fatal("did not go on once there was room")
	}
}

// TestLabAccepts: the lab takes nothing until an essence is chosen, then the
// ingredients of that recipe only.
func TestLabAccepts(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, machine.KindLab, "")
	if st.Accepts(item.NewStack(item.GlassBottle{}, 1)) || st.Insert(item.NewStack(item.GlassBottle{}, 1)) {
		t.Fatal("takes a glass bottle with no essence chosen")
	}
	st.SetLab(machinerecipe.LabByKind(sgitem.EssenceHunting))
	for _, in := range []world.Item{item.GlassBottle{}, item.Gunpowder{}, item.SpiderEye{}} {
		if !st.Accepts(item.NewStack(in, 1)) {
			t.Errorf("does not take %s", naming.ItemKey(in))
		}
	}
	for _, in := range []world.Item{block.Dirt{}, item.Diamond{}, item.IronIngot{}, sgitem.Petal{Colour: "blue"}} {
		if st.Accepts(item.NewStack(in, 1)) {
			t.Errorf("takes %s", naming.ItemKey(in))
		}
	}
}

// TestFarmingIngredientsGrow: the crops the essence of farming takes are
// what the six crops of the island give.
func TestFarmingIngredientsGrow(t *testing.T) {
	grown := map[string]bool{}
	for _, b := range []world.Block{farm.WithGrowth(block.WheatSeeds{}, 7), farm.WithGrowth(block.Carrot{}, 7), farm.WithGrowth(block.Potato{}, 7), farm.WithGrowth(block.BeetrootSeeds{}, 7), block.Melon{}, block.Pumpkin{}} {
		for _, st := range b.(block.Breakable).BreakInfo().Drops(item.ToolNone{}, nil) {
			grown[naming.ItemKey(st.Item())] = true
		}
	}
	for _, in := range machinerecipe.Labs[2].In[1:] {
		if !grown[naming.ItemKey(in.Item)] {
			t.Errorf("%s does not grow on the island's crops", naming.ItemKey(in.Item))
		}
	}
}

// TestLabNoticesAnIngredientTakenAway: an ingredient taken out during a cycle
// stops the brewing, even though the slots are only looked at now and then.
func TestLabNoticesAnIngredientTakenAway(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, machine.KindLab, "")
	r := machinerecipe.Labs[1]
	st.SetLab(&machinerecipe.Labs[1])
	fillLab(st, r, 1)
	tick(s, r.Ticks/2)
	if st.Progress == 0 {
		t.Fatal("did not start")
	}
	_ = st.Inv().SetItem(st.Def.Input[1], item.Stack{})
	tick(s, r.Ticks)
	if machine.OutputCount(st, r.Out) != 0 || st.Progress != 0 {
		t.Fatalf("brewed without an ingredient: %d essences, progress %d", machine.OutputCount(st, r.Out), st.Progress)
	}
}

// TestLabFeedLeavesRoom: a feed of one ingredient stops at a few cycles'
// worth, so the other ingredients still find a slot.
func TestLabFeedLeavesRoom(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, machine.KindLab, "")
	st.SetLab(machinerecipe.LabByKind(sgitem.EssenceEngineering))
	iron := item.NewStack(item.IronIngot{}, 1)
	fed := 0
	for st.Insert(iron) {
		fed++
		if fed > 1000 {
			t.Fatal("the feed never stopped")
		}
	}
	if fed != st.LabFeedCap(iron) {
		t.Fatalf("fed %d iron, want the cap %d", fed, st.LabFeedCap(iron))
	}
	if !st.Insert(item.NewStack(item.RedstoneWire{}, 1)) || !st.Insert(item.NewStack(item.GlassBottle{}, 1)) {
		t.Fatal("the other ingredients found no room")
	}
}

// TestLabScreen: the screen lists the essences the research has come to;
// starting one puts its ingredients in from the inventory (and nothing else),
// the essences made are handed out, and stopping gives everything back and
// lets the extractor take nothing.
func TestLabScreen(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, machine.KindLab, "")
	conduit, _ := progress.ByID("energy_conduit")
	progress.StateOf(s.Session).Researched = map[string]bool{}
	for _, id := range conduit.Needs {
		progress.StateOf(s.Session).Researched[id] = true
	}
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		open := machine.LabOpenRecipes(s.Session)
		if len(open) != 1 || open[0].Out.Kind != sgitem.EssenceEngineering {
			t.Fatalf("%d essences open, want the essence of engineering", len(open))
		}
		// the screen's buttons run at once in the test
		run := func(_ *machinerecipe.Lab, f func(*player.Player, *world.Tx)) func(*player.Player, *world.Tx) {
			return f
		}
		_, _, actions := machine.LabDetail(s.Session, i18n.Default, st, open[0], run)
		_, _ = p.Inventory().AddItem(item.NewStack(item.GlassBottle{}, 5))
		_, _ = p.Inventory().AddItem(item.NewStack(item.IronIngot{}, 200))
		_, _ = p.Inventory().AddItem(item.NewStack(item.RedstoneWire{}, 7))
		_, _ = p.Inventory().AddItem(item.NewStack(item.Diamond{}, 3))
		press(t, p, tx, actions, "§a만들기 시작")
		have := st.LabHave()
		if st.Lab == nil || have[naming.ItemKey(item.GlassBottle{})] != 5 || have[naming.ItemKey(item.RedstoneWire{})] != 7 || have[naming.ItemKey(item.Diamond{})] != 0 {
			t.Fatalf("chosen %v, holds %v", st.Lab != nil, have)
		}
		if n := have[naming.ItemKey(item.IronIngot{})]; n != st.LabHandCap(item.NewStack(item.IronIngot{}, 1)) {
			t.Fatalf("holds %d iron, want the hand cap", n)
		}
		_ = st.Inv().SetItem(st.Def.Output[0], item.NewStack(sgitem.ResearchEssence{Kind: sgitem.EssenceEngineering}, 4))
		_, rows, actions := machine.LabDetail(s.Session, i18n.Default, st, st.Lab, run)
		if len(rows) != 2 || rows[1].Slots[0].Count != "4" {
			t.Fatalf("rows %+v", rows)
		}
		press(t, p, tx, actions, "정수 꺼내기")
		if machine.OutputCount(st, sgitem.ResearchEssence{Kind: sgitem.EssenceEngineering}) != 0 || countIn(p, sgitem.ResearchEssence{Kind: sgitem.EssenceEngineering}) != 4 {
			t.Fatal("the essences were not handed out")
		}
		press(t, p, tx, actions, "§c생산 중지")
		if st.Lab != nil || len(st.LabHave()) != 0 || countIn(p, item.GlassBottle{}) != 5 || countIn(p, item.IronIngot{}) != 200 {
			t.Fatalf("stopped: chosen %v, holds %v, bottles %d, iron %d", st.Lab != nil, st.LabHave(), countIn(p, item.GlassBottle{}), countIn(p, item.IronIngot{}))
		}
		if st.Insert(item.NewStack(item.GlassBottle{}, 1)) {
			t.Fatal("a stopped lab takes a glass bottle")
		}
	})
}

// countIn counts the items of a kind in p's inventory.
func countIn(p *player.Player, it world.Item) int {
	n := 0
	for _, st := range p.Inventory().Items() {
		if naming.ItemKey(st.Item()) == naming.ItemKey(it) {
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
	st := placeTestMachine(s, machinePos, machine.KindLab, "")
	st.SetLab(machinerecipe.LabByKind(sgitem.EssenceMagic))
	back := machine.Machine{Kind: machine.KindLab}.DecodeNBT(machine.Machine{Kind: machine.KindLab, State: st}.EncodeNBT()).(machine.Machine)
	if back.State.Lab == nil || back.State.Lab.Out.Kind != sgitem.EssenceMagic {
		t.Fatal("the chosen essence was not saved")
	}
	st.SetLab(nil)
	_ = st.Inv().SetItem(st.Def.Input[0], item.NewStack(item.Bone{}, 2))
	back = machine.Machine{Kind: machine.KindLab}.DecodeNBT(machine.Machine{Kind: machine.KindLab, State: st}.EncodeNBT()).(machine.Machine)
	if back.State.Lab == nil || back.State.Lab.Out.Kind != sgitem.EssenceHunting {
		t.Fatal("an old lab with bones does not brew the essence of hunting")
	}
}
