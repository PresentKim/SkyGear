package skygear

import (
	"slices"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// TestToolsLeaveTheClickOutsideTheIsland: with the configuration copier or
// the construction wand in hand, a click on a block outside the island is
// refused by the island's guard as any other click is. The guard is reached
// wherever the tools' steps stand among the subscribers: they do not take a
// click where the island does not allow building.
func TestToolsLeaveTheClickOutsideTheIsland(t *testing.T) {
	outside := cube.Pos{84, island.IslandY + 2, 4}
	for name, held := range map[string]item.Stack{
		"copier": item.NewStack(sgitem.ConfigCopier{}, 1),
		"wand":   item.NewStack(sgitem.NewWand(sgitem.WandStone), 1),
	} {
		s := testSession(t)
		island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
		loadAround(s)
		h := spawnPlayer(s)
		withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
			p.SetHeldItems(held, item.Stack{})
			ctx := player.NewEventContext(tx, p)
			var ran []string
			core.Trace = func(step string) { ran = append(ran, step) }
			func() {
				defer func() { core.Trace = nil }()
				s.Handler(p).HandleItemUseOnBlock(ctx, outside, cube.FaceUp, mgl64.Vec3{0.5, 1, 0.5})
			}()
			if !slices.Contains(ran, "island.guard") {
				t.Errorf("%s: the island's guard was not reached: %v", name, ran)
			}
			if last := ran[len(ran)-1]; last != "island.guard" {
				t.Errorf("%s: the click went on to %s after the guard", name, last)
			}
			if !ctx.Cancelled() {
				t.Errorf("%s: the click outside the island was not cancelled", name)
			}
		})
	}
}
