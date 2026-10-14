package dragonfactory

import (
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// TestHUDQuestIsNextResearch: the quest panel shows the first research
// that can be done and its cost, paid parts counted.
func TestHUDQuestIsNextResearch(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		title, rows := s.questRows()
		if !strings.Contains(title, "막대기") || len(rows) != 1 || !strings.Contains(rows[0], "0§7/16") {
			t.Fatalf("quest %q %q", title, rows)
		}
		sticks, _ := researchByID("stick")
		_, _ = p.Inventory().AddItem(item.NewStack(oakPlanks, 16))
		s.payResearch(p, tx, sticks)
		if title, _ := s.questRows(); !strings.Contains(title, "나무 도구") {
			t.Fatalf("quest after sticks %q", title)
		}
	})
}
