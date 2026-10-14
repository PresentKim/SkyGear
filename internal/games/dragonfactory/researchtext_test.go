package dragonfactory

import (
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/ui"
)

// TestEveryResearchIsExplained: a research with no text leaves the player
// without a next step (the stick research once gave no hint where pebbles
// come from).
func TestEveryResearchIsExplained(t *testing.T) {
	for _, r := range researchTree {
		txt := researchText(r.ID)
		if txt.About == "" || txt.Hint == "" || txt.After == "" {
			t.Errorf("research %s lacks text: %+v", r.ID, txt)
		}
	}
	for id := range researchTexts {
		if _, ok := researchByID(id); !ok {
			t.Errorf("text for unknown research %s", id)
		}
	}
}

// TestResearchScreenShowsHints: the open research shows its hint, the done
// research what to do next.
func TestResearchScreenShowsHints(t *testing.T) {
	s := testSession(t)
	r, _ := researchByID("stick")
	d, _, _ := s.researchNode(nil, r)
	body := d.Body
	if !strings.Contains(body, researchText("stick").Hint) {
		t.Errorf("open research without its hint: %q", body)
	}
	s.state.Researched = map[string]bool{"stick": true}
	d, _, _ = s.researchNode(nil, r)
	body = d.Body
	if !strings.Contains(body, researchText("stick").After) {
		t.Errorf("done research without its next step: %q", body)
	}
}

// TestResearchListHidesLockedResearches: a research shows up only when
// everything before it is done.
func TestResearchListHidesLockedResearches(t *testing.T) {
	s := testSession(t)
	_, slots, _ := s.researchList(nil, false)
	if len(slots) != 1 { // the stick research
		t.Fatalf("%d researches at the start", len(slots))
	}
	s.state.Researched = map[string]bool{"stick": true}
	_, slots, _ = s.researchList(nil, false)
	// pebbles, wood tools, chest
	if len(slots) != 3 {
		t.Fatalf("%d researches after the stick research", len(slots))
	}
	if _, done, _ := s.researchList(nil, true); len(done) != 1 {
		t.Fatalf("%d done researches", len(done))
	}
}

// TestResearchListTabs: the researches to do and the done ones are on
// separate tabs, the tooltip title is yellow and green when p has all the
// materials.
func TestResearchListTabs(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	s.state.Researched = map[string]bool{"stick": true}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		_, todo, tabs := s.researchList(p, false)
		if len(tabs) != 3 || !strings.HasPrefix(tabs[0].Label, ui.TabOnToken) || !strings.HasPrefix(tabs[1].Label, ui.TabToken) {
			t.Fatalf("tabs %+v", tabs)
		}
		if !strings.Contains(todo[0].Tip, "§e돌 다지기") {
			t.Errorf("tooltip %q is not yellow", todo[0].Tip)
		}
		_, _ = p.Inventory().AddItem(item.NewStack(Pebble{Kind: PebbleStone}, 4))
		_, todo, _ = s.researchList(p, false)
		if !strings.HasPrefix(todo[0].Tip, "§a돌 다지기") || strings.HasPrefix(todo[1].Tip, "§a") {
			t.Errorf("tooltips %q %q: only the one with all materials is green", todo[0].Tip, todo[1].Tip)
		}
		_, done, tabs := s.researchList(p, true)
		if len(done) != 1 || !strings.HasPrefix(tabs[1].Label, ui.TabOnToken) {
			t.Errorf("done tab: %d slots, tabs %+v", len(done), tabs)
		}
	})
}

// TestResearchScreen: an open research has the 연구 진행 button and shows
// its materials and recipes as slots; no word of what it needs first; a
// done one shows 연구 완료 and no button.
func TestResearchScreen(t *testing.T) {
	s := testSession(t)
	r, _ := researchByID("wood_tools")
	s.state.Researched = map[string]bool{"stick": true}
	d, rows, opts := s.researchNode(nil, r)
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
	s.state.Researched["wood_tools"] = true
	d, _, opts = s.researchNode(nil, r)
	if len(opts) != 1 || !strings.Contains(d.Status, "연구 완료") {
		t.Errorf("done research: status %q options %+v", d.Status, opts)
	}
}
