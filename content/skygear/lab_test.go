package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/machinerecipe"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// fillLab puts the ingredients of a recipe, times sets of them, in the input slots.
func fillLab(st *machine.State, r machinerecipe.Lab, sets int) {
	for i, in := range r.In {
		_ = st.Inv().SetItem(st.Def.Input[i], item.NewStack(in.Item, in.Count*sets))
	}
}

// TestLabScreen: the screen lists the essences the research has come to;
// starting one puts its ingredients in from the inventory (and nothing else),
// the essences made are handed out, and stopping gives everything back and
// lets the extractor take nothing.
func TestLabScreen(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, machine.KindLab)
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
