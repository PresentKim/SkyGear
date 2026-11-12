package skygear

import (
	"log/slog"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/tinker"
	"github.com/presentkim/skygear/platform/account"
	"github.com/presentkim/skygear/platform/instance"
)

func TestVeinDiagonalSetting(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	base := cube.Pos{4, island.IslandY + 3, 4}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		tx.SetBlock(base, block.Gravel{}, nil)
		tx.SetBlock(base.Add(cube.Pos{1, 1, 0}), block.Gravel{}, nil)
		tx.SetBlock(base.Add(cube.Pos{2, 2, 1}), block.Gravel{}, nil)
		if got := tinker.VeinBlocks(tx, base, tinker.VeinLimit, false, func(p cube.Pos) bool { return island.CanBuild(s.Session, p) }); len(got) != 1 {
			t.Fatalf("face vein of %d blocks, want 1", len(got))
		}
		if got := tinker.VeinBlocks(tx, base, tinker.VeinLimit, true, func(p cube.Pos) bool { return island.CanBuild(s.Session, p) }); len(got) != 3 {
			t.Fatalf("diagonal vein of %d blocks, want 3", len(got))
		}
		if _, err := s.g.db.TouchAccount(instance.AccountID(p), "", p.Name()); err != nil {
			t.Fatal(err)
		}
		account.Init(s.g.db, slog.Default())
		t.Cleanup(func() { account.Init(nil, slog.Default()) })
		if account.On(p, tinker.VeinDiagonalSetting) {
			t.Fatal("diagonal on before it was switched")
		}
		tinker.ToggleVeinDiagonal(s.Session, p)
		account.Forget(p) // read it again from the database
		if !account.On(p, tinker.VeinDiagonalSetting) {
			t.Fatal("the switched setting was not stored")
		}
	})
}
