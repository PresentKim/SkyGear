package dragonfactory

import (
	"strings"
	"testing"
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
	body, _ := s.researchNode(r)
	if !strings.Contains(body, researchText("stick").Hint) {
		t.Errorf("open research without its hint: %q", body)
	}
	s.state.Researched = map[string]bool{"stick": true}
	body, _ = s.researchNode(r)
	if !strings.Contains(body, researchText("stick").After) {
		t.Errorf("done research without its next step: %q", body)
	}
}

// TestResearchListHidesLockedResearches: a research shows up only when
// everything before it is done.
func TestResearchListHidesLockedResearches(t *testing.T) {
	s := testSession(t)
	_, opts := s.researchMenu()
	if len(opts) != 2 { // stick and the back button
		t.Fatalf("%d entries at the start", len(opts))
	}
	r, _ := researchByID("stick")
	s.state.Researched = map[string]bool{r.ID: true}
	_, opts = s.researchMenu()
	// stick (done), pebbles, wood tools, chest, back
	if len(opts) != 5 {
		t.Fatalf("%d entries after the stick research", len(opts))
	}
}
