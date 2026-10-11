package dragonfactory

import (
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// TestHUD: the quest panel lists each item of the mission, and the level
// and mission progress show on the experience bar.
func TestHUD(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		s.state.Level = 2 // 돌과 흙: cobblestone 64, dirt 32
		m, _ := s.mission()
		if s.state.Delivered == nil {
			s.state.Delivered = map[string]int{}
		}
		s.state.Delivered[itemKey(m.Requirements[0].Item)] = 64
		s.state.Delivered[itemKey(m.Requirements[1].Item)] = 8
		title, rows := s.questRows()
		if !strings.Contains(title, m.Name) || len(rows) != 2 || !strings.HasPrefix(rows[0], "§a") || !strings.Contains(rows[1], "8§7/32") {
			t.Fatalf("quest %q %q", title, rows)
		}
		s.updateBoard(p)
		if p.ExperienceLevel() != 2 || p.ExperienceProgress() < 0.7 || p.ExperienceProgress() > 0.8 {
			t.Fatalf("experience level %d progress %.2f, want 2 and about 72/96 (kept as whole experience points)", p.ExperienceLevel(), p.ExperienceProgress())
		}
		if !strings.Contains(itemName(item.NewStack(m.Requirements[1].Item, 1)), "흙") {
			t.Fatal("item name")
		}
	})
}
