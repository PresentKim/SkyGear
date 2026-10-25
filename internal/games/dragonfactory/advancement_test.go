package dragonfactory

import (
	"testing"

	"github.com/presentkim/dragonfactory/common/i18n"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/ui"
)

// TestAdvancementTreeIsValid: ids are unique, parents come before their
// children, items exist and researches are in the research tree.
func TestAdvancementTreeIsValid(t *testing.T) {
	seen := map[string]bool{}
	for _, a := range advancementTree {
		if seen[a.ID] {
			t.Errorf("%s twice", a.ID)
		}
		if a.Parent != "" && !seen[a.Parent] {
			t.Errorf("%s: parent %s not before it", a.ID, a.Parent)
		}
		seen[a.ID] = true
		if !i18n.Has(a.NameKey) || !i18n.Has(a.HintKey) {
			t.Errorf("%s: no name or hint in the catalog", a.ID)
		}
		for _, n := range []string{a.Item, a.Icon} {
			if _, ok := world.ItemByName(unlockName(n), 0); n != "" && !ok {
				t.Errorf("%s: no item %s", a.ID, n)
			}
		}
		if _, ok := researchByID(a.Research); a.Research != "" && !ok {
			t.Errorf("%s: no research %s", a.ID, a.Research)
		}
		if a.Item == "" && a.Research == "" && !a.Event {
			t.Errorf("%s: nothing completes it", a.ID)
		}
	}
}

// TestAdvancementsFromItems: having an item completes its advancement and
// the ones before it; events complete theirs.
func TestAdvancementsFromItems(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		if a, _ := s.nextAdvancement(); a.ID != "wood" {
			t.Fatalf("first advancement %s", a.ID)
		}
		_, _ = p.Inventory().AddItem(item.NewStack(block.CraftingTable{}, 1))
		s.checkAdvancements(tx)
		for _, id := range []string{"wood", "planks", "crafting_table"} {
			if !s.advanced(id) {
				t.Errorf("%s not done", id)
			}
		}
		if a, _ := s.nextAdvancement(); a.ID != "research_table" {
			t.Errorf("next advancement %s", a.ID)
		}
		s.advance(tx, advSift)
		if !s.advanced(advSift) || !s.advanced("sieve") || !s.advanced("string") || !s.advanced("first_research") {
			t.Errorf("sift did not complete its ancestors: %v", s.state.Advancements)
		}
		if s.advanced(advGrowTree) {
			t.Error("grow tree done without growing")
		}
	})
}

// TestAdvancementTreeShowsNearNodes: the tree shows done and open nodes and
// the locked one right behind an open node; a child sits one column right of
// its parent with connector lines between them; every advancement is in one
// tab.
func TestAdvancementTreeShowsNearNodes(t *testing.T) {
	s := testSession(t)
	s.state.Advancements = map[string]bool{"wood": true}
	l := s.advancementLayout(advTabBasic)
	for id, shown := range map[string]bool{"wood": true, "planks": true, "grow_tree": true, "crafting_table": true, "research_table": false} {
		if _, ok := l.pos[id]; ok != shown {
			t.Errorf("%s shown=%v, want %v", id, ok, shown)
		}
	}
	wood, planks := l.pos["wood"], l.pos["planks"]
	if planks.x != wood.x+1 {
		t.Errorf("planks at column %d beside wood at %d", planks.x, wood.x)
	}
	if l.strips[planks]&ui.StripRight == 0 || l.strips[advPos{planks.x, wood.y}]&ui.StripLeft == 0 {
		t.Errorf("no connector between wood and planks: %v", l.strips)
	}
	seen := map[advPos]string{}
	for id, ps := range l.pos {
		if o, dup := seen[ps]; dup {
			t.Errorf("%s and %s share %v", id, o, ps)
		}
		seen[ps] = id
	}
	total := 0
	for tab := advTab(0); tab < advTabCount; tab++ {
		n := len(advancementsIn(tab))
		if n == 0 {
			t.Errorf("tab %d is empty", tab)
		}
		total += n
	}
	if total != len(advancementTree) {
		t.Errorf("tabs list %d of %d", total, len(advancementTree))
	}
}

// TestAdvancementTreeFits: with everything done every tab's tree fits the
// grid of the form.
func TestAdvancementTreeFits(t *testing.T) {
	s := testSession(t)
	s.state.Advancements = map[string]bool{}
	for _, a := range advancementTree {
		s.state.Advancements[a.ID] = true
	}
	for tab := advTab(0); tab < advTabCount; tab++ {
		l := s.advancementLayout(tab)
		if l.w > ui.TreeCols || l.h > ui.TreeRows {
			t.Errorf("tab %s is %d×%d, the grid %d×%d", advTabKeys[tab], l.w, l.h, ui.TreeCols, ui.TreeRows)
		}
	}
}

// TestAdvancementStack: an advancement points at its item, else its icon.
func TestAdvancementStack(t *testing.T) {
	a, _ := advancementByID("hammer")
	if it, ok := advancementStack(a); !ok || itemName(i18n.Default, it) != "돌 망치" {
		t.Fatalf("hammer stack %v %v", it, ok)
	}
	b, _ := advancementByID("first_research")
	if _, ok := advancementStack(b); !ok {
		t.Fatal("first_research has an icon")
	}
	if _, ok := advancementStack(&Advancement{}); ok {
		t.Fatal("empty advancement has no item")
	}
}

// TestTrackedAdvancement: the quest panel follows the tracked advancement
// while it is open and goes back to the tree order when it is done or
// tracked again.
func TestTrackedAdvancement(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		s.advance(tx, "research_table")
		chest, _ := advancementByID("chest")
		s.state.Advancements["first_research"] = true
		if next, _ := s.nextAdvancement(); next.ID == "chest" {
			t.Fatal("chest is not first in the tree order")
		}
		s.track(tx, chest)
		if next, _ := s.nextAdvancement(); next.ID != "chest" {
			t.Fatalf("tracked chest, panel follows %q", next.ID)
		}
		s.track(tx, chest)
		if next, _ := s.nextAdvancement(); next.ID == "chest" {
			t.Fatal("untracked chest must go back to the tree order")
		}
		s.track(tx, chest)
		s.state.Advancements["chest"] = true
		if next, _ := s.nextAdvancement(); next.ID == "chest" {
			t.Fatal("a done advancement is not followed")
		}
	})
}

// TestAdvancementGuides: every advancement leads the quest panel to a
// research, but those the start of the game or an event alone does.
func TestAdvancementGuides(t *testing.T) {
	free := map[string]bool{"wood": true, "planks": true, "crafting_table": true, "research_table": true,
		advGrowTree: true, advChunk: true, advChunks8: true, advChunks16: true, advBloom: true}
	for i := range advancementTree {
		a := &advancementTree[i]
		if _, ok := goalResearch(a); !ok && !free[a.ID] {
			t.Errorf("%s leads to no research", a.ID)
		}
	}
}

// TestAdvancementOrder: the quest panel follows the main path: the stone
// barrel and the auto miner before the crusher, the energy and item
// conduits before the fluid conduit.
func TestAdvancementOrder(t *testing.T) {
	at := map[string]int{}
	for i, a := range advancementTree {
		at[a.ID] = i
	}
	for _, p := range [][2]string{
		{"stone_barrel", "crusher"}, {"auto_miner", "crusher"}, {"energy_conduit", "fluid_conduit"},
		{"item_conduit", "fluid_conduit"}, {"crusher", "alloy_smelter"}, {"laboratory", "crusher"},
	} {
		if at[p[0]] >= at[p[1]] {
			t.Errorf("%s comes after %s", p[0], p[1])
		}
	}
}

// TestGardenTabStartsEarly: the garden tab starts with sifting seeds from
// dirt, then wheat and bread, so it shows a node once the first sifting is
// done.
func TestGardenTabStartsEarly(t *testing.T) {
	s := testSession(t)
	for id, parent := range map[string]string{"seeds": advSift, "wheat": "seeds", "bread": "wheat"} {
		a, ok := advancementByID(id)
		if !ok || a.Parent != parent || advTabOf(a) != advTabGarden {
			t.Fatalf("%s: parent %q, tab %d", id, a.Parent, advTabOf(a))
		}
	}
	s.state.Advancements = map[string]bool{}
	for cur, ok := advancementByID(advSift); ok; cur, ok = advancementByID(cur.Parent) {
		s.state.Advancements[cur.ID] = true
	}
	seeds, _ := advancementByID("seeds")
	if !s.advShown(seeds) {
		t.Fatal("the seeds advancement is not shown once sifting is done")
	}
}
