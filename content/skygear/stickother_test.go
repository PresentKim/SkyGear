package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/handwork"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/storage"
	"github.com/presentkim/skygear/platform/account"
	"github.com/presentkim/skygear/platform/instance"
)

// TestStickOtherCountsTheTapNothingTook: "stick_other" records a stick used
// on a block that is neither grass nor dirt, when the click went through:
// no subscriber cancelled it. A click another feature took (the menu of a
// drawer, a crop) is not such a tap, wherever the record's subscriber stands
// among the others.
func TestStickOtherCountsTheTapNothingTook(t *testing.T) {
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
		p.SetHeldItems(item.NewStack(item.Stick{}, 1), item.Stack{})
		tap := func() *player.Context {
			delete(handwork.LastStick.Of(s.Session), p.H()) // not the same click sent twice
			ctx := player.NewEventContext(tx, p)
			s.Handler(p).HandleItemUseOnBlock(ctx, pos, cube.FaceUp, mgl64.Vec3{0.5, 0.5, 0.5})
			return ctx
		}
		counted := func() int64 { return account.Counter(acc, "stick_other") }

		tx.SetBlock(pos, block.Stone{}, nil)
		if tap().Cancelled() {
			t.Fatal("a stick on stone was cancelled")
		}
		if got := counted(); got != 1 {
			t.Errorf("stick_other counted %d after a stick on stone, want 1", got)
		}

		tx.SetBlock(pos, storage.Drawer{Facing: cube.South}, nil)
		p.StartSneaking()
		if !tap().Cancelled() {
			t.Fatal("a sneaking stick on a drawer did not take the click")
		}
		p.StopSneaking()
		if got := counted(); got != 1 {
			t.Errorf("stick_other counted %d after the stick opened a drawer's menu, want 1", got)
		}

		tx.SetBlock(pos.Side(cube.FaceDown), block.Farmland{Hydration: 7}, nil)
		tx.SetBlock(pos, block.WheatSeeds{}, nil)
		if !tap().Cancelled() {
			t.Fatal("a crop did not take the click")
		}
		if got := counted(); got != 1 {
			t.Errorf("stick_other counted %d after the crop took the stick's click, want 1", got)
		}
	})
}
