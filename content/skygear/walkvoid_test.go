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

// TestWalkCountsTheMoveIntoTheVoid: the move that takes a player below the
// island counts the distance of that move, from where the player stood to
// where it moved, although the island brings the player back in the same
// event. It counts the same wherever achievement.walk stands relative to
// island.void: the event carries where the player stood (MoveEvent.From).
func TestWalkCountsTheMoveIntoTheVoid(t *testing.T) {
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
		if got := p.Position(); got.X() != 3.5 || got.Y() != island.VoidReturnY {
			t.Fatalf("the player was put at %v", got)
		}
		if got := account.Counter(acc, "walk"); got != 1 {
			t.Errorf("walk counted %d for a move of one block, want 1", got)
		}
	})
}
