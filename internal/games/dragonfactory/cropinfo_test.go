package dragonfactory

import (
	"strings"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// TestUsingCrops: using a growing crop shows its time (one tag per player)
// and places nothing; using a grown crop harvests and replants it.
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
		tx.SetBlock(grown, withGrowth(block.WheatSeeds{}, 7), nil)

		ctx := player.NewEventContext(tx, p)
		(&handler{s: s}).HandleItemUseOnBlock(ctx, young, cube.FaceUp, mgl64.Vec3{})
		if !ctx.Cancelled() {
			t.Fatal("using a crop was not cancelled (a block could be placed on it)")
		}
		tag := s.cropTags[p.H()]
		if tag == nil || tag.text == nil || !strings.Contains(tag.text.Text, "다 자라기까지") {
			t.Fatalf("crop tag %v", tag)
		}
		// The tag counts down while shown and goes when its time is up.
		before := tag.text
		s.clock.Advance(3 * time.Second)
		s.refreshCropTags(tx)
		if tag.text == before || tag.text.Text == before.Text {
			t.Fatalf("tag did not count down: %q", tag.text.Text)
		}
		tag.until = time.Now().Add(-time.Second)
		s.refreshCropTags(tx)
		if s.cropTags[p.H()] != nil {
			t.Fatal("expired tag still shown")
		}
		s.useCrop(p, tx, young)
		tag = s.cropTags[p.H()]
		s.useCrop(p, tx, young)
		if s.cropTags[p.H()] == tag {
			t.Fatal("a second use did not replace the tag")
		}

		if !s.useCrop(p, tx, grown) {
			t.Fatal("grown crop not harvested")
		}
		if _, g, ok := cropOf(tx.Block(grown)); !ok || g != 0 {
			t.Fatalf("after harvest: %#v", tx.Block(grown))
		}
		if s.state.Crops[cropKey(grown)] == nil {
			t.Fatal("replanted crop not scheduled")
		}
	})
}
