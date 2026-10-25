package dragonfactory

import (
	"slices"
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
		for _, par := range a.Parents {
			if !seen[par] {
				t.Errorf("%s: parent %s not before it", a.ID, par)
			}
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

// TestAdvancementsFromItems: having an item completes its advancement but
// not the ones before it; events complete theirs.
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
		if !s.advanced("crafting_table") {
			t.Error("crafting_table not done")
		}
		for _, id := range []string{"wood", "planks"} {
			if s.advanced(id) {
				t.Errorf("%s done by the crafting table", id)
			}
		}
		if a, _ := s.nextAdvancement(); a.ID != "wood" {
			t.Errorf("next advancement %s", a.ID)
		}
		s.advance(tx, advSift)
		if !s.advanced(advSift) || s.advanced("sieve") {
			t.Errorf("sift must complete itself alone: %v", s.state.Advancements)
		}
		if s.advanced(advGrowTree) {
			t.Error("grow tree done without growing")
		}
	})
}

// TestAdvancementTreeShowsEveryNode: a new save shows every node of a
// chapter; a child sits one column right of its parent with connector lines
// between them; every advancement is in one chapter.
func TestAdvancementTreeShowsEveryNode(t *testing.T) {
	s := testSession(t)
	s.state.Advancements = map[string]bool{}
	l := advancementLayout(advChapterStart)
	for _, a := range advancementsIn(advChapterStart) {
		if _, ok := l.pos[a.ID]; !ok {
			t.Errorf("%s not shown", a.ID)
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
	for tab := advChapter(0); tab < advChapterCount; tab++ {
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

// TestAdvancementLayout: in every chapter a node sits right of each of its
// parents, a parent in another chapter is an entrance left of its children,
// no two places share a cell and every node with parents has the line of
// one in the tree form.
func TestAdvancementLayout(t *testing.T) {
	for ch := range advChapterCount {
		l := advancementLayout(ch)
		seen := map[advPos]string{}
		for id, ps := range l.pos {
			if o, dup := seen[ps]; dup {
				t.Errorf("%s: %s and %s share %v", advChapterKeys[ch], id, o, ps)
			}
			seen[ps] = id
		}
		for _, g := range l.gates {
			if _, ok := l.pos[advGateID(g.ID)]; g.Chapter == ch || !ok {
				t.Errorf("%s: entrance of %s at %v", advChapterKeys[ch], g.ID, l.pos[advGateID(g.ID)])
			}
		}
		for _, a := range l.order {
			c := l.pos[a.ID]
			for _, id := range a.Parents {
				par, _ := advancementByID(id)
				key := id
				if par.Chapter != ch {
					key = advGateID(id)
				}
				p, ok := l.pos[key]
				if !ok || p.x >= c.x {
					t.Errorf("%s: %s at %v, its parent %s at %v", advChapterKeys[ch], a.ID, c, key, p)
				}
			}
			if len(a.Parents) > 0 && l.line[a.ID] == "" {
				t.Errorf("%s: %s has no line", advChapterKeys[ch], a.ID)
			}
		}
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
	for tab := advChapter(0); tab < advChapterCount; tab++ {
		l := advancementLayout(tab)
		if l.w > ui.TreeCols || l.h > ui.TreeRows {
			t.Errorf("tab %s is %d×%d, the grid %d×%d", advChapterKeys[tab], l.w, l.h, ui.TreeCols, ui.TreeRows)
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
		advGrowTree: true, advChunk: true, advChunks8: true, advChunks16: true, advBloom: true,
		advStats: true, advFairyGift: true, advFavourite: true, advLost: true}
	for i := range advancementTree {
		a := &advancementTree[i]
		if _, ok := goalResearch(a); !ok && !free[a.ID] {
			t.Errorf("%s leads to no research", a.ID)
		}
	}
}

// TestAdvancementOrder: the quest panel follows the main path: the auto
// miner and the laboratory before the crusher, the stone barrel before the
// crucible, the barrel before the generator.
func TestAdvancementOrder(t *testing.T) {
	at := map[string]int{}
	for i, a := range advancementTree {
		at[a.ID] = i
	}
	for _, p := range [][2]string{
		{"auto_miner", "crusher"}, {"laboratory", "crusher"}, {"stone_barrel", "crucible"}, {"barrel", "generator"},
		{"energy_conduit", "fluid_conduit"}, {"item_conduit", "fluid_conduit"}, {"crusher", "alloy_smelter"},
	} {
		if at[p[0]] >= at[p[1]] {
			t.Errorf("%s comes after %s", p[0], p[1])
		}
	}
}

// TestAdvancementChapters: an advancement is in the chapter of how its thing
// is got, and the projects gather the branches before the dragon.
func TestAdvancementChapters(t *testing.T) {
	for id, ch := range map[string]advChapter{
		"wood": advChapterStart, advSift: advChapterSieve, "mesh_diamond": advChapterSieve, "seeds": advChapterFarming,
		"generator": advChapterAutomation, "alloy_smelter": advChapterAlloy, advWave1: advChapterHunting,
		"diamond_tools": advChapterGear, advBloom: advChapterMagic, "project_dragon": advChapterProject, advChunks16: advChapterExtra,
	} {
		if a, ok := advancementByID(id); !ok || a.Chapter != ch {
			t.Errorf("%s not in chapter %s", id, advChapterKeys[ch])
		}
	}
	for id, parents := range map[string][]string{
		"project_dragon": {"project_industry", "project_hunter", "project_botanist"},
		advWave5:         {advWave3},
		"bronze_ingot":   {"alloy_smelter", "tin_ingot"},
		"clay":           {"barrel", "dust"},
		"seeds":          {advSift},
		"mesh_diamond":   {"mesh_string"},
	} {
		if a, _ := advancementByID(id); !slices.Equal(a.Parents, parents) {
			t.Errorf("%s after %v, want %v", id, a.Parents, parents)
		}
	}
}
