package skygear

import (
	"strings"
	"testing"

	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/progress"
)

// TestResearchCopiesFillTheTabs: test copies of the researches make the lists
// long, are never saved, can be done without cost and keep their original's
// texts; the grid of the split screen holds one page of them.
func TestResearchCopiesFillTheTabs(t *testing.T) {
	s := testSession(t)
	todoBefore, _, _ := s.researchEntries()
	for i := range 130 {
		r := progress.Tree[i%len(progress.Tree)]
		r.ID += "#" + string(rune('A'+i%26)) + string(rune('a'+i/26))
		r.Needs = nil
		progress.LocalOf(s.Session).Copies = append(progress.LocalOf(s.Session).Copies, &r)
	}
	progress.LocalOf(s.Session).CopiesDone = map[string]bool{}
	todo, done, openN := s.researchEntries()
	if len(todo) != len(todoBefore)+130 || len(done) != 0 || openN < 130 {
		t.Fatalf("%d to do, %d done, %d open with 130 copies (%d before)", len(todo), len(done), openN, len(todoBefore))
	}
	c := progress.LocalOf(s.Session).Copies[0]
	if !c.IsCopy() || !strings.Contains(c.Title("ko"), " #") {
		t.Errorf("a copy is named %q", c.Title("ko"))
	}
	if progress.Text("ko", c.ID).About != progress.Text("ko", progress.Tree[0].ID).About {
		t.Error("a copy does not read its original's text")
	}
	progress.LocalOf(s.Session).CopiesDone[c.ID] = true
	_, done, _ = s.researchEntries()
	if len(done) != 1 || len(progress.StateOf(s.Session).Researched) != 0 {
		t.Errorf("a done copy: %d done, %d saved", len(done), len(progress.StateOf(s.Session).Researched))
	}
	if pages := (len(todo) + ui.SplitCells - 1) / ui.SplitCells; pages < 3 {
		t.Errorf("%d pages for %d researches", pages, len(todo))
	}
}
