package dragonfactory

import (
	"log/slog"
	"slices"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/account"
)

func TestLevelUpGrantsAchievements(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	account.Init(s.g.db, slog.Default())
	t.Cleanup(func() { account.Init(nil, slog.Default()) })
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		if _, err := s.g.db.TouchAccount(p.UUID().String(), "", p.Name()); err != nil {
			t.Fatal(err)
		}
		s.state.Level = 2
		_, _ = p.Inventory().AddItem(item.NewStack(block.Cobblestone{}, 64))
		_, _ = p.Inventory().AddItem(item.NewStack(block.Dirt{}, 32))
		s.deliverInventory(p, tx)
		if s.state.Level != 3 {
			t.Fatalf("level %d", s.state.Level)
		}
		if got := account.Owned(p.UUID().String()); !slices.Contains(got, achLevel3) || slices.Contains(got, achLevel5) {
			t.Fatalf("achievements %v", got)
		}
	})
}

func TestJoinGrantsReachedLevels(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	account.Init(s.g.db, slog.Default())
	t.Cleanup(func() { account.Init(nil, slog.Default()) })
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		_, _ = s.g.db.TouchAccount(p.UUID().String(), "", p.Name())
		s.state.Level = len(Levels) + 1
		s.levelAchievementsFor(p)
		got := account.Owned(p.UUID().String())
		for _, id := range []string{achLevel3, achLevel5, achLevel8, achAllMissions} {
			if !slices.Contains(got, id) {
				t.Fatalf("missing %s in %v", id, got)
			}
		}
	})
}
