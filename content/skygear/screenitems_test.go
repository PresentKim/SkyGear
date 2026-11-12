package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/dimension"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// TestHunterStoneOpensNothing: a hunter's use is the hunting ground's to
// refuse, on a block and in the air, also with the dimension stone in hand:
// the use is cancelled and no window opens, wherever dimension.stone stands
// among the subscribers.
func TestHunterStoneOpensNothing(t *testing.T) {
	s := testSession(t)
	island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
	loadAround(s)
	h := spawnPlayer(s)
	t.Cleanup(func() {
		// the hunt schedules the player's landing: out before the world closes
		s.World.Do(func(tx *world.Tx) {
			if e, ok := h.Entity(tx); ok {
				tx.RemoveEntity(e)
			}
		})
	})
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		if !dimension.Begin(s.Session, tx, p, sgitem.MobSlime) {
			t.Fatal("the hunt did not begin")
		}
		p.SetHeldItems(item.NewStack(sgitem.DimensionStone{}, 1), item.Stack{})
		hd := s.Handler(p)
		for name, use := range map[string]func(ctx *player.Context){
			"on a block": func(ctx *player.Context) {
				hd.HandleItemUseOnBlock(ctx, cube.PosFromVec3(p.Position()).Side(cube.FaceDown), cube.FaceUp, mgl64.Vec3{0.5, 1, 0.5})
			},
			"in the air": func(ctx *player.Context) { hd.HandleItemUse(ctx) },
		} {
			ctx := player.NewEventContext(tx, p)
			if shown := pinScreens(func() { use(ctx) }); shown != "screens=-" {
				t.Errorf("%s: the hunter's stone opened %s", name, shown)
			}
			if !ctx.Cancelled() {
				t.Errorf("%s: the hunter's use was not cancelled", name)
			}
		}
	})
}
