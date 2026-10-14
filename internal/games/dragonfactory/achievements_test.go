package dragonfactory

import (
	"log/slog"
	"slices"
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/account"
)

// TestFirstResearchGrantsAchievement: completing a research earns the
// research achievement for the player who paid it.
func TestFirstResearchGrantsAchievement(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	account.Init(s.g.db, slog.Default())
	t.Cleanup(func() { account.Init(nil, slog.Default()) })
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		if _, err := s.g.db.TouchAccount(p.UUID().String(), "", p.Name()); err != nil {
			t.Fatal(err)
		}
		sticks, _ := researchByID("stick")
		_, _ = p.Inventory().AddItem(item.NewStack(oakPlanks, 16))
		s.doResearch(p, tx, sticks)
		if got := account.Owned(p.UUID().String()); !slices.Contains(got, achResearch) {
			t.Fatalf("achievements %v", got)
		}
	})
}
