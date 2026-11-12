package elevator_test

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/core/coretest"
	"github.com/presentkim/skygear/content/skygear/elevator"
	"github.com/presentkim/skygear/content/skygear/island"
)

// held are the players the test's takeover holds.
var held = map[*world.EntityHandle]bool{}

func init() {
	core.RegisterTakeover("elevatortest", func(_ *core.Session, p *player.Player) bool { return held[p.H()] })
}

// TestHeldPlayerDoesNotRideDown: a sneak on an elevator rides it down, but
// not for a player a takeover holds (island expansion mode, a hunt): the mode
// has the player's body, and its own use for the sneak.
func TestHeldPlayerDoesNotRideDown(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	h := coretest.SpawnPlayer(s)
	x, z := island.Centre+2, island.Centre
	base := island.IslandY + 2
	red := elevator.Elevator{Colour: item.ColourRed()}
	coretest.WithPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		tx.SetBlock(cube.Pos{x, base, z}, red, nil)
		tx.SetBlock(cube.Pos{x, base + 12, z}, red, nil)
		top := mgl64.Vec3{float64(x) + 0.3, float64(base + 13), float64(z) + 0.7}
		p.Teleport(top)
		hd := s.Handler(p).(*core.PlayerHandler)
		sneak := func() { hd.HandleToggleSneak(player.NewEventContext(tx, p), true) }

		held[p.H()] = true
		sneak()
		if got := p.Position(); got != top {
			t.Fatalf("a held player rode the elevator to %v", got)
		}
		delete(held, p.H())
		sneak()
		if got := p.Position().Y(); got != float64(base+1) {
			t.Fatalf("a free player is at y %v after the sneak, want %d", got, base+1)
		}
	})
}
