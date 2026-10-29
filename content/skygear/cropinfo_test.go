package skygear

import (
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/farm"
	"github.com/presentkim/skygear/content/skygear/look"
)

// TestUsingCrops: using a growing crop places nothing (the WAILA panel shows
// its time); using a grown crop harvests and replants it.
func TestUsingCrops(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	young, grown := cube.Pos{1, 64, 1}, cube.Pos{2, 64, 1}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		for _, pos := range []cube.Pos{young, grown} {
			tx.SetBlock(pos.Side(cube.FaceDown), block.Farmland{Hydration: 7}, nil)
		}
		tx.SetBlock(young, block.WheatSeeds{}, nil)
		tx.SetBlock(grown, farm.WithGrowth(block.WheatSeeds{}, 7), nil)

		ctx := player.NewEventContext(tx, p)
		(&handler{s: s}).HandleItemUseOnBlock(ctx, young, cube.FaceUp, mgl64.Vec3{})
		if !ctx.Cancelled() {
			t.Fatal("using a crop was not cancelled (a block could be placed on it)")
		}
		// The WAILA panel shows the time left on a growing crop.
		if info := look.Info(s.Session, p, tx, young, tx.Block(young)); info.Name != "밀" || len(info.Lines) != 1 || !strings.Contains(info.Lines[0], "남은 시간") {
			t.Fatalf("crop panel %+v", info)
		}

		if !farm.UseCrop(s.Session, p, tx, grown) {
			t.Fatal("grown crop not harvested")
		}
		if _, g, ok := farm.CropOf(tx.Block(grown)); !ok || g != 0 {
			t.Fatalf("after harvest: %#v", tx.Block(grown))
		}
		if farm.StateOf(s.Session).Crops[farm.CropKey(grown)] == nil {
			t.Fatal("replanted crop not scheduled")
		}
	})
}
