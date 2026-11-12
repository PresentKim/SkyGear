package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/itemnet"
	"github.com/presentkim/skygear/content/skygear/conduit"
	"github.com/presentkim/skygear/content/skygear/island"
)

// What a block leaves behind when it goes is cleaned up by the block's own
// callbacks, which Dragonfly calls after the handler's subscribers. The pin
// scenarios call the handler only, so these tests remove the block through
// Dragonfly: a player's finished break (Player.BreakBlock) and an explosion.

// brokenSession is a session with a player who stands next to pos and may
// break it.
func brokenSession(t *testing.T, pos cube.Pos) (*session, *world.EntityHandle) {
	t.Helper()
	s := testSession(t)
	island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		p.Handle(s.Handler(p))
		p.Teleport(pos.Vec3Centre().Add(mgl64.Vec3{0, 1, 2}))
	})
	return s, h
}

// blowUp removes the blocks around pos as TNT that goes off beside it does.
func blowUp(tx *world.Tx, pos cube.Pos) {
	block.ExplosionConfig{}.Explode(tx, world.BlockExplosionSource{Pos: pos.Side(cube.FaceUp), ExplosionSize: 4})
}

// TestBrokenBundleLosesItsDisplay: a conduit bundle a player breaks loses its
// display and its connector plates, and so does one an explosion removes.
// No subscriber of the break does it: the bundle's BlockChanged does.
func TestBrokenBundleLosesItsDisplay(t *testing.T) {
	chest := cube.Pos{4, island.IslandY + 2, 4}
	for _, c := range []struct {
		name   string
		remove func(p *player.Player, tx *world.Tx, at cube.Pos)
	}{
		{"broken by hand", func(p *player.Player, _ *world.Tx, at cube.Pos) { p.BreakBlock(at) }},
		{"blown up", func(_ *player.Player, tx *world.Tx, at cube.Pos) { blowUp(tx, at) }},
	} {
		s, h := brokenSession(t, chest.Side(cube.FaceEast))
		withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
			at := chestWithBundle(tx, chest, itemnet.Input, 5, 9)
			l := conduit.LocalOf(s.Session)
			if _, ok := l.Visuals[at]; !ok {
				t.Fatalf("%s: the bundle has no display", c.name)
			}
			if len(l.Plates) == 0 {
				t.Fatalf("%s: the bundle has no connector plate", c.name)
			}
			c.remove(p, tx, at)
			if _, still := tx.Block(at).(conduit.Bundle); still {
				t.Fatalf("%s: the bundle is still there", c.name)
			}
			if _, ok := l.Visuals[at]; ok {
				t.Errorf("%s: the bundle's display is still there", c.name)
			}
			if len(l.Plates) != 0 {
				t.Errorf("%s: %d connector plates are still there", c.name, len(l.Plates))
			}
		})
	}
}
