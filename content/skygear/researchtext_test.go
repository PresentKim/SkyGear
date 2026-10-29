package skygear

import (
	"github.com/presentkim/skygear/content/skygear/screens"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// TestEveryResearchIsExplained: every research has its texts.
func TestEveryResearchIsExplained(t *testing.T) {
	for _, r := range progress.Tree {
		for _, f := range append([]string{"name"}, progress.TextFields...) {
			if !i18n.Has("research." + r.ID + "." + f) {
				t.Errorf("research %s lacks its %s text", r.ID, f)
			}
		}
	}
	// No catalog text of an unknown research.
	for _, l := range i18n.Languages() {
		for _, key := range i18n.Keys(l) {
			if id, ok := strings.CutPrefix(key, "research."); ok {
				id, _, _ = strings.Cut(id, ".")
				if _, ok := progress.ByID(id); !ok {
					t.Errorf("%s: text %s for an unknown research", l, key)
				}
			}
		}
	}
}

// TestResearchListHidesLockedResearches: a research shows up to do only when
// everything before it is done; one that is a step from open shows in grey
// after them, and nothing further is shown.
func TestResearchListHidesLockedResearches(t *testing.T) {
	s := testSession(t)
	split := func() (open, grey int) {
		slots, _, _, _ := screens.ResearchSlots(s.Session, nil)
		for _, sl := range slots {
			if strings.HasPrefix(sl.Tip, "§8") {
				grey++
			} else {
				open++
			}
		}
		return open, grey
	}
	open, grey := split()
	if open != 2 || grey != 4 { // the stick and fairy magic researches; pebbles, wood tools, chest, fairy time a step away
		t.Fatalf("at the start: %d open, %d grey", open, grey)
	}
	progress.StateOf(s.Session).Researched = map[string]bool{"stick": true}
	open, grey = split()
	if open != 4 { // pebbles, wood tools, chest, fairy magic
		t.Fatalf("%d researches after the stick research", open)
	}
	if grey == 0 {
		t.Fatal("nothing is shown a step ahead")
	}
	slots, _, _, _ := screens.ResearchSlots(s.Session, nil)
	for i, sl := range slots {
		if i < open && strings.HasPrefix(sl.Tip, "§8") || i >= open && !strings.HasPrefix(sl.Tip, "§8") {
			t.Fatalf("grey ones do not come last: slot %d %q", i, sl.Tip)
		}
	}
	if _, done, _, _ := screens.ResearchSlots(s.Session, nil); len(done) != 1 {
		t.Fatalf("%d done researches", len(done))
	}
}

// TestResearchPreviewButton: the screen of a research that is not open has no
// research button but one that goes to the research it needs first.
func TestResearchPreviewButton(t *testing.T) {
	s := testSession(t)
	r, _ := progress.ByID("wood_tools")
	need, ok := progress.Blocker(s.Session, r)
	if !ok || need.ID != "stick" {
		t.Fatalf("blocker %v %v", need, ok)
	}
	_, _, opts := screens.ResearchNode(s.Session, nil, r)
	var found bool
	for _, o := range opts {
		if strings.Contains(o.Label, need.Title(i18n.Default)) && strings.Contains(o.Label, "선행 연구") {
			found = true
		}
		if strings.Contains(o.Label, "연구 진행") {
			t.Fatal("a research that is not open can be done")
		}
	}
	if !found {
		t.Fatalf("no button to the research needed first: %+v", opts)
	}
	// a research two steps away is not a preview
	far, _ := progress.ByID("pebbles")
	_ = far
	if _, ok := progress.Blocker(s.Session, &progress.Research{ID: "x", Needs: []string{"wood_tools"}}); ok {
		t.Fatal("a research behind one that is not open is a preview")
	}
}

// TestResearchListSections: the researches to do and the done ones are in
// two sections of one list, the tooltip title is yellow and green when p has
// all the materials.
func TestResearchListSections(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	progress.StateOf(s.Session).Researched = map[string]bool{"stick": true}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		todo, done, todoN, doneN := screens.ResearchSlots(s.Session, p)
		if todoN != 4 || doneN != 1 || len(done) != 1 || len(todo) < todoN {
			t.Fatalf("%d to do (%d slots), %d done (%d slots)", todoN, len(todo), doneN, len(done))
		}
		if !strings.HasPrefix(todo[0].Tip, "§e돌 다지기\n§7") || strings.Contains(todo[0].Tip, "모자랍니다") {
			t.Errorf("tooltip %q is not the yellow name and the short text", todo[0].Tip)
		}
		_, _ = p.Inventory().AddItem(item.NewStack(sgitem.Pebble{Kind: sgitem.PebbleStone}, 4))
		todo, _, _, _ = screens.ResearchSlots(s.Session, p)
		if !strings.HasPrefix(todo[0].Tip, "§a돌 다지기") || strings.HasPrefix(todo[1].Tip, "§a") {
			t.Errorf("tooltips %q %q: only the one with all materials is green", todo[0].Tip, todo[1].Tip)
		}
	})
}

// TestResearchScreen: an open research has the 연구 진행 button and shows
// its materials and recipes as slots; no word of what it needs first; a
// done one shows 연구 완료 and no button.
func TestResearchScreen(t *testing.T) {
	s := testSession(t)
	r, _ := progress.ByID("wood_tools")
	progress.StateOf(s.Session).Researched = map[string]bool{"stick": true}
	d, rows, opts := screens.ResearchNode(s.Session, nil, r)
	if len(opts) != 2 || !strings.Contains(opts[0].Label, "연구 진행") {
		t.Fatalf("options %+v", opts)
	}
	if strings.Contains(d.Body, "먼저 할 연구") || strings.Contains(d.Body, "납품") {
		t.Errorf("body mentions prerequisites or delivery: %q", d.Body)
	}
	if len(rows) != 2 || len(rows[0].Slots) != 2 || len(rows[1].Slots) != 5 {
		t.Fatalf("rows %+v", rows)
	}
	if rows[0].Slots[0].Count != "§f8" || rows[0].Slots[1].Count != "§f4" {
		t.Errorf("material counts %q %q", rows[0].Slots[0].Count, rows[0].Slots[1].Count)
	}
	progress.StateOf(s.Session).Researched["wood_tools"] = true
	d, _, opts = screens.ResearchNode(s.Session, nil, r)
	if len(opts) != 1 || !strings.Contains(d.Status, "연구 완료") {
		t.Errorf("done research: status %q options %+v", d.Status, opts)
	}
}
