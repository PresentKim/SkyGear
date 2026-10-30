package skygear

import (
	"slices"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/progress"
)

// TestAdvancementTreeIsValid: ids are unique, parents come before their
// children, items exist and researches are in the research tree.
func TestAdvancementTreeIsValid(t *testing.T) {
	seen := map[string]bool{}
	for _, a := range advancement.Tree {
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
			if _, ok := world.ItemByName(naming.UnlockName(n), 0); n != "" && !ok {
				t.Errorf("%s: no item %s", a.ID, n)
			}
		}
		if _, ok := progress.ByID(a.Research); a.Research != "" && !ok {
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
		if a, _ := progress.NextAdvancement(s.Session); a.ID != "wood" {
			t.Fatalf("first advancement %s", a.ID)
		}
		_, _ = p.Inventory().AddItem(item.NewStack(block.CraftingTable{}, 1))
		progress.CheckHeld(s.Session, tx, advancement.InventoryNames(tx))
		if !progress.Advanced(s.Session, "crafting_table") {
			t.Error("crafting_table not done")
		}
		for _, id := range []string{"wood", "planks"} {
			if progress.Advanced(s.Session, id) {
				t.Errorf("%s done by the crafting table", id)
			}
		}
		if a, _ := progress.NextAdvancement(s.Session); a.ID != "wood" {
			t.Errorf("next advancement %s", a.ID)
		}
		progress.Advance(s.Session, tx, advancement.Sift)
		if !progress.Advanced(s.Session, advancement.Sift) || progress.Advanced(s.Session, "sieve") {
			t.Errorf("sift must complete itself alone: %v", progress.StateOf(s.Session).Advancements)
		}
		if progress.Advanced(s.Session, advancement.GrowTree) {
			t.Error("grow tree done without growing")
		}
	})
}

// TestAdvancementTreeShowsEveryNode: a new save shows every node of a
// chapter; a child sits one column right of its parent with its line from
// it; every advancement is in one chapter.
func TestAdvancementTreeShowsEveryNode(t *testing.T) {
	s := testSession(t)
	progress.StateOf(s.Session).Advancements = map[string]bool{}
	l := advancement.LayoutOf(advancement.ChapterStart)
	for _, a := range advancement.In(advancement.ChapterStart) {
		if _, ok := l.Pos[a.ID]; !ok {
			t.Errorf("%s not shown", a.ID)
		}
	}
	wood, planks := l.Pos["wood"], l.Pos["planks"]
	if planks.X != wood.X+1 {
		t.Errorf("planks at column %d beside wood at %d", planks.X, wood.X)
	}
	if l.Line["planks"] != "wood" {
		t.Errorf("the line of planks comes from %q, want wood", l.Line["planks"])
	}
	seen := map[advancement.Pos]string{}
	for id, ps := range l.Pos {
		if o, dup := seen[ps]; dup {
			t.Errorf("%s and %s share %v", id, o, ps)
		}
		seen[ps] = id
	}
	total := 0
	for tab := advancement.Chapter(0); tab < advancement.ChapterCount; tab++ {
		n := len(advancement.In(tab))
		if n == 0 {
			t.Errorf("tab %d is empty", tab)
		}
		total += n
	}
	if total != len(advancement.Tree) {
		t.Errorf("tabs list %d of %d", total, len(advancement.Tree))
	}
}

// TestAdvancementLayout: in every chapter a node sits right of each of its
// parents, a parent in another chapter is an entrance left of its children,
// no two places share a cell and every node with parents has the line of
// one.
func TestAdvancementLayout(t *testing.T) {
	for ch := range advancement.ChapterCount {
		l := advancement.LayoutOf(ch)
		seen := map[advancement.Pos]string{}
		for id, ps := range l.Pos {
			if o, dup := seen[ps]; dup {
				t.Errorf("%s: %s and %s share %v", advancement.ChapterKeys[ch], id, o, ps)
			}
			seen[ps] = id
		}
		for _, g := range l.Gates {
			if _, ok := l.Pos[advancement.GateID(g.ID)]; g.Chapter == ch || !ok {
				t.Errorf("%s: entrance of %s at %v", advancement.ChapterKeys[ch], g.ID, l.Pos[advancement.GateID(g.ID)])
			}
		}
		for _, a := range l.Order {
			c := l.Pos[a.ID]
			for _, id := range a.Parents {
				par, _ := advancement.ByID(id)
				key := id
				if par.Chapter != ch {
					key = advancement.GateID(id)
				}
				p, ok := l.Pos[key]
				if !ok || p.X >= c.X {
					t.Errorf("%s: %s at %v, its parent %s at %v", advancement.ChapterKeys[ch], a.ID, c, key, p)
				}
			}
			if len(a.Parents) > 0 && l.Line[a.ID] == "" {
				t.Errorf("%s: %s has no line", advancement.ChapterKeys[ch], a.ID)
			}
		}
	}
}

// TestAdvancementTreeFits: with everything done every chapter's tree stays
// within its bands' height.
func TestAdvancementTreeFits(t *testing.T) {
	s := testSession(t)
	progress.StateOf(s.Session).Advancements = map[string]bool{}
	for _, a := range advancement.Tree {
		progress.StateOf(s.Session).Advancements[a.ID] = true
	}
	for tab := advancement.Chapter(0); tab < advancement.ChapterCount; tab++ {
		l := advancement.LayoutOf(tab)
		if l.H > advancement.Rows {
			t.Errorf("tab %s is %d rows high, more than %d", advancement.ChapterKeys[tab], l.H, advancement.Rows)
		}
	}
}

// TestAdvancementStack: an advancement points at its item, else its icon.
func TestAdvancementStack(t *testing.T) {
	a, _ := advancement.ByID("hammer")
	if it, ok := advancement.Stack(a); !ok || naming.ItemName(i18n.Default, it) != "돌 망치" {
		t.Fatalf("hammer stack %v %v", it, ok)
	}
	b, _ := advancement.ByID("first_research")
	if _, ok := advancement.Stack(b); !ok {
		t.Fatal("first_research has an icon")
	}
	if _, ok := advancement.Stack(&advancement.Advancement{}); ok {
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
		progress.Advance(s.Session, tx, "research_table")
		chest, _ := advancement.ByID("chest")
		progress.StateOf(s.Session).Advancements["first_research"] = true
		if next, _ := progress.NextAdvancement(s.Session); next.ID == "chest" {
			t.Fatal("chest is not first in the tree order")
		}
		progress.Track(s.Session, tx, chest)
		if next, _ := progress.NextAdvancement(s.Session); next.ID != "chest" {
			t.Fatalf("tracked chest, panel follows %q", next.ID)
		}
		progress.Track(s.Session, tx, chest)
		if next, _ := progress.NextAdvancement(s.Session); next.ID == "chest" {
			t.Fatal("untracked chest must go back to the tree order")
		}
		progress.Track(s.Session, tx, chest)
		progress.StateOf(s.Session).Advancements["chest"] = true
		if next, _ := progress.NextAdvancement(s.Session); next.ID == "chest" {
			t.Fatal("a done advancement is not followed")
		}
	})
}

// TestAdvancementGuides: every advancement leads the quest panel to a
// research, but those the start of the game or an event alone does.
func TestAdvancementGuides(t *testing.T) {
	free := map[string]bool{"wood": true, "planks": true, "crafting_table": true, "research_table": true,
		advancement.GrowTree: true, advancement.Chunk: true, advancement.Chunks8: true, advancement.Chunks16: true, advancement.Bloom: true,
		advancement.Stats: true, advancement.FairyGift: true, advancement.Favourite: true, advancement.Lost: true}
	for i := range advancement.Tree {
		a := &advancement.Tree[i]
		if _, ok := progress.GoalResearch(a); !ok && !free[a.ID] {
			t.Errorf("%s leads to no research", a.ID)
		}
	}
}

// TestAdvancementChapters: an advancement is in the chapter of how its thing
// is got, and the projects gather the branches before the dragon.
func TestAdvancementChapters(t *testing.T) {
	for id, ch := range map[string]advancement.Chapter{
		"wood": advancement.ChapterStart, advancement.Sift: advancement.ChapterSieve, "mesh_diamond": advancement.ChapterSieve, "seeds": advancement.ChapterFarming,
		"generator": advancement.ChapterAutomation, "alloy_smelter": advancement.ChapterAlloy, advancement.Wave1: advancement.ChapterHunting,
		"diamond_tools": advancement.ChapterGear, advancement.Bloom: advancement.ChapterMagic, advancement.FairyGift: advancement.ChapterFairy, "fairy_board": advancement.ChapterFairy, "project_dragon": advancement.ChapterProject, advancement.Chunks16: advancement.ChapterExtra,
	} {
		if a, ok := advancement.ByID(id); !ok || a.Chapter != ch {
			t.Errorf("%s not in chapter %s", id, advancement.ChapterKeys[ch])
		}
	}
	for id, parents := range map[string][]string{
		"project_dragon":  {"project_industry", "project_hunter", "project_botanist"},
		advancement.Wave5: {advancement.Wave3},
		"bronze_ingot":    {"alloy_smelter", "tin_ingot"},
		"clay":            {"barrel", "dust"},
		"seeds":           {advancement.Sift},
		"mesh_diamond":    {"mesh_string"},
		"fairy_board":     {"fairy_dust"},
	} {
		if a, _ := advancement.ByID(id); !slices.Equal(a.Parents, parents) {
			t.Errorf("%s after %v, want %v", id, a.Parents, parents)
		}
	}
}
