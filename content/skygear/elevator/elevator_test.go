package elevator_test

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
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

// TestElevatorRides: jumping on an elevator goes up to the next one of its
// colour, through the floors between; sneaking goes back down. Other
// colours, a blocked top and elevators out of range are passed over.
func TestElevatorRides(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	h := coretest.SpawnPlayer(s)
	x, z := island.Centre+2, island.Centre
	base := island.IslandY + 2
	red, blue := elevator.Elevator{Colour: item.ColourRed()}, elevator.Elevator{Colour: item.ColourBlue()}
	at := func(dy int) cube.Pos { return cube.Pos{x, base + dy, z} }
	coretest.WithPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		tx.SetBlock(at(0), red, nil)
		tx.SetBlock(at(3), block.Stone{}, nil) // a floor between
		tx.SetBlock(at(5), blue, nil)          // another colour
		tx.SetBlock(at(8), red, nil)           // no room on top
		tx.SetBlock(at(9), block.Stone{}, nil)
		tx.SetBlock(at(12), red, nil) // the next one up

		stand := func(dy int) { p.Teleport(mgl64.Vec3{float64(x) + 0.3, float64(base + dy + 1), float64(z) + 0.7}) }
		stand(0)
		if !elevator.Up(s.Session, p, tx) {
			t.Fatal("did not ride up")
		}
		if got := p.Position(); got.Y() != float64(base+13) || got.X() != float64(x)+0.3 || got.Z() != float64(z)+0.7 {
			t.Fatalf("rode up to %v, want y %d and the same x, z", got, base+13)
		}
		if elevator.Down(s.Session, p, tx) {
			t.Fatal("rode again within the cooldown")
		}
		lastRideReset(s, p)
		if !elevator.Down(s.Session, p, tx) || p.Position().Y() != float64(base+1) {
			t.Fatalf("rode down to %v, want y %d", p.Position(), base+1)
		}
		lastRideReset(s, p)
		if elevator.Down(s.Session, p, tx) {
			t.Fatal("rode down from the lowest elevator")
		}
		stand(5)
		if elevator.Up(s.Session, p, tx) {
			t.Fatal("a blue elevator led to a red one")
		}
		// Too far: only Range blocks are searched.
		tx.SetBlock(at(12), block.Air{}, nil)
		tx.SetBlock(at(8), block.Air{}, nil)
		tx.SetBlock(at(elevator.Range+1), red, nil)
		stand(0)
		if elevator.Up(s.Session, p, tx) {
			t.Fatal("rode to an elevator out of range")
		}
		// Not standing on one: in the air above it.
		tx.SetBlock(at(elevator.Range), red, nil)
		p.Teleport(mgl64.Vec3{float64(x) + 0.5, float64(base) + 1.6, float64(z) + 0.5})
		if elevator.Up(s.Session, p, tx) {
			t.Fatal("rode while not standing on the elevator")
		}
	})
}

// lastRideReset lets p ride again at once.
func lastRideReset(s core.Lifecycle, p *player.Player) { elevator.ResetCooldown(s.Session, p) }
