package dragonfactory

import (
	"strings"
	"testing"

	"github.com/presentkim/dragonfactory/internal/ui"
)

// TestResearchCopiesFillTheTabs: test copies of the researches make the lists
// long, are never saved, can be done without cost and keep their original's
// texts; the grid of the split screen holds one page of them.
func TestResearchCopiesFillTheTabs(t *testing.T) {
	s := testSession(t)
	todoBefore, _, _ := s.researchEntries()
	for i := range 130 {
		r := researchTree[i%len(researchTree)]
		r.ID += "#" + string(rune('A'+i%26)) + string(rune('a'+i/26))
		r.Needs = nil
		s.copies = append(s.copies, &r)
	}
	s.copiesDone = map[string]bool{}
	todo, done, openN := s.researchEntries()
	if len(todo) != len(todoBefore)+130 || len(done) != 0 || openN < 130 {
		t.Fatalf("%d to do, %d done, %d open with 130 copies (%d before)", len(todo), len(done), openN, len(todoBefore))
	}
	c := s.copies[0]
	if !c.isCopy() || !strings.Contains(c.Title("ko"), " #") {
		t.Errorf("a copy is named %q", c.Title("ko"))
	}
	if researchText("ko", c.ID).About != researchText("ko", researchTree[0].ID).About {
		t.Error("a copy does not read its original's text")
	}
	s.copiesDone[c.ID] = true
	_, done, _ = s.researchEntries()
	if len(done) != 1 || len(s.state.Researched) != 0 {
		t.Errorf("a done copy: %d done, %d saved", len(done), len(s.state.Researched))
	}
	if pages := (len(todo) + ui.SplitCells - 1) / ui.SplitCells; pages < 4 {
		t.Errorf("%d pages for %d researches", pages, len(todo))
	}
}
