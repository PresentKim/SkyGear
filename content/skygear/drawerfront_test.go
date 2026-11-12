package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/storage"
	"github.com/presentkim/skygear/content/skygear/tinker"
	"github.com/presentkim/skygear/platform/account"
	"github.com/presentkim/skygear/platform/instance"
)

// TestDrawerFrontHitIsNoBreak: a break that ends a hit on a drawer's front is
// cancelled (the hit took items out), and a cancelled break is no break for
// the subscribers after it: it is not counted and the tool in hand gains no
// experience. A break after a hit on another face is a break.
func TestDrawerFrontHitIsNoBreak(t *testing.T) {
	d := bridgeAccount(t)
	s := testSession(t)
	island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
	loadAround(s)
	h := spawnPlayer(s)
	pos := cube.Pos{4, island.IslandY + 2, 4}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		acc := instance.AccountID(p)
		if _, err := d.TouchAccount(acc, "", p.Name()); err != nil {
			t.Fatal(err)
		}
		tx.SetBlock(pos, storage.Drawer{Facing: cube.South}, nil)
		p.SetHeldItems(item.NewStack(tinker.Tool{Kind: tinker.ToolPickaxe, Head: "iron", Handle: "wood"}, 1), item.Stack{})
		hit := func(face cube.Face) *player.Context {
			tx.Block(pos).(storage.Drawer).Punch(pos, face, tx, p)
			drops, xp := []item.Stack{item.NewStack(block.Stone{}, 1)}, 0
			ctx := player.NewEventContext(tx, p)
			s.Handler(p).HandleBlockBreak(ctx, pos, &drops, &xp)
			return ctx
		}
		toolXP := func() int {
			held, _ := p.HeldItems()
			return tinker.ToolXP(held)
		}

		if !hit(cube.South.Face()).Cancelled() {
			t.Fatal("the break after a hit on the drawer's front was not cancelled")
		}
		if got := account.Counter(acc, "block_break"); got != 0 {
			t.Errorf("block_break counted %d for the cancelled break, want 0", got)
		}
		if got := toolXP(); got != 0 {
			t.Errorf("the tool gained %d experience from the cancelled break, want 0", got)
		}

		if hit(cube.North.Face()).Cancelled() {
			t.Fatal("the break after a hit on the drawer's back was cancelled")
		}
		if got := account.Counter(acc, "block_break"); got != 1 {
			t.Errorf("block_break counted %d after the break, want 1", got)
		}
		if got := toolXP(); got != 1 {
			t.Errorf("the tool has %d experience after the break, want 1", got)
		}
	})
}
