package dragonfactory

import (
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// TestHUDQuestFollowsAdvancements: the quest panel shows the next
// advancement and its hint, with the research cost for a research
// advancement, then the next research once every advancement is done.
func TestHUDQuestFollowsAdvancements(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		if title, rows := s.questRows(); !strings.Contains(title, "나무 캐기") || len(rows) != 2 {
			t.Fatalf("quest %q %q", title, rows)
		}
		s.advance(tx, "research_table")
		title, rows := s.questRows()
		if !strings.Contains(title, "첫 연구") || len(rows) != 3 || !strings.Contains(rows[2], "0§7/4") {
			t.Fatalf("quest %q %q", title, rows)
		}
		sticks, _ := researchByID("stick")
		_, _ = p.Inventory().AddItem(item.NewStack(oakPlanks, 16))
		s.payResearch(p, tx, sticks)
		s.checkAdvancements(tx)
		if title, _ := s.questRows(); !strings.Contains(title, "돌멩이") {
			t.Fatalf("quest after sticks %q", title)
		}
		for _, a := range advancementTree {
			s.state.Advancements[a.ID] = true
		}
		if title, _ := s.questRows(); !strings.Contains(title, "돌 다지기") {
			t.Fatalf("quest after every advancement %q", title)
		}
	})
}
