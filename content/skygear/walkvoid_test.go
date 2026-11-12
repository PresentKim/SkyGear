package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/platform/account"
	"github.com/presentkim/skygear/platform/instance"
)

// TestMoveIntoTheVoidIsNotWalked: the move that takes a player below the
// island is cancelled (the island brings the player back), and a cancelled
// move is not a walk: the walk counter is not called for it. The next move
// counts from where the island put the player.
func TestMoveIntoTheVoidIsNotWalked(t *testing.T) {
	d := bridgeAccount(t)
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		acc := instance.AccountID(p)
		if _, err := d.TouchAccount(acc, "", p.Name()); err != nil {
			t.Fatal(err)
		}
		// The only block around: the player comes back above it, two blocks
		// from where the move ends.
		tx.SetBlock(cube.Pos{3, 60, 2}, block.Stone{}, nil)
		from := p.Position()
		to := mgl64.Vec3{from.X() + 1, island.VoidY - 10, from.Z()}
		ctx := player.NewEventContext(tx, p)
		s.Handler(p).HandleMove(ctx, to, cube.Rotation{})
		if !ctx.Cancelled() {
			t.Fatal("the move into the void was not cancelled")
		}
		back := p.Position()
		if back.X() != 3.5 || back.Y() != island.VoidReturnY {
			t.Fatalf("the player was put at %v", back)
		}
		if got := account.Counter(acc, "walk"); got != 0 {
			t.Errorf("walk counted %d for the cancelled move into the void, want 0", got)
		}
		if got := account.Counter(acc, "void_return"); got != 1 {
			t.Errorf("void_return counted %d, want 1", got)
		}
		s.Handler(p).HandleMove(player.NewEventContext(tx, p), back.Add(mgl64.Vec3{1, 0, 0}), cube.Rotation{})
		if got := account.Counter(acc, "walk"); got != 1 {
			t.Errorf("walk counted %d after a move of one block from where the island put the player, want 1", got)
		}
	})
}
