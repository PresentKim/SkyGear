package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/elevator"
	"github.com/presentkim/skygear/content/skygear/expand"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/platform/account"
	"github.com/presentkim/skygear/platform/instance"
)

// sneakElevators puts two red elevators 12 blocks apart at the island's
// centre and returns where a player stands on the upper and on the lower one.
func sneakElevators(tx *world.Tx) (top, bottom mgl64.Vec3) {
	x, z := island.Centre+2, island.Centre
	base := island.IslandY + 2
	red := elevator.Elevator{Colour: item.ColourRed()}
	tx.SetBlock(cube.Pos{x, base, z}, red, nil)
	tx.SetBlock(cube.Pos{x, base + 12, z}, red, nil)
	return mgl64.Vec3{float64(x) + 0.3, float64(base + 13), float64(z) + 0.7}, mgl64.Vec3{float64(x) + 0.3, float64(base + 1), float64(z) + 0.7}
}

// TestEverySneakCounts: no subscriber of Sneak keeps another from running.
// A sneak counts whatever else it does: in island expansion mode, where it
// shows the expansion screen again and the body stays where the mode put it,
// and on an elevator, which it rides down.
func TestEverySneakCounts(t *testing.T) {
	d := bridgeAccount(t)
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		acc := instance.AccountID(p)
		if _, err := d.TouchAccount(acc, "", p.Name()); err != nil {
			t.Fatal(err)
		}
		top, bottom := sneakElevators(tx)
		sneak := func() { s.Handler(p).HandleToggleSneak(player.NewEventContext(tx, p), true) }

		expand.Show(s.Session, p)
		body := p.Position()
		sneak()
		if got := account.Counter(acc, "sneak"); got != 1 {
			t.Errorf("a sneak in island expansion mode counted %d, want 1", got)
		}
		if got := p.Position(); got != body {
			t.Errorf("a sneak in island expansion mode moved the player from %v to %v", body, got)
		}
		if _, ok := expand.Of(s.Session, p); !ok {
			t.Error("the sneak ended island expansion mode")
		}
		expand.Stop(s.Session, p, tx)

		p.Teleport(top)
		sneak()
		if got := account.Counter(acc, "sneak"); got != 2 {
			t.Errorf("a sneak that rides the elevator: %d sneaks counted, want 2", got)
		}
		if got := p.Position(); got != bottom {
			t.Errorf("the sneak on the elevator left the player at %v, want %v", got, bottom)
		}
	})
}

// TestSneakRideGrowsSaplingsWhereItArrives: a sneak that rides an elevator
// down is still a sneak for the saplings: those around the place the player
// arrives at get their chance to grow.
func TestSneakRideGrowsSaplingsWhereItArrives(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		top, bottom := sneakElevators(tx)
		at := cube.PosFromVec3(bottom).Add(cube.Pos{1, 0, 0})
		tx.SetBlock(at.Side(cube.FaceDown), block.Dirt{}, nil)
		tx.SetBlock(at, block.Sapling{Type: block.OakSapling()}, nil)
		for range 200 {
			p.Teleport(top)
			elevator.ResetCooldown(s.Session, p)
			s.Handler(p).HandleToggleSneak(player.NewEventContext(tx, p), true)
			if got := p.Position(); got != bottom {
				t.Fatalf("the sneak left the player at %v, want %v", got, bottom)
			}
			if _, ok := tx.Block(at).(block.Sapling); !ok {
				return
			}
		}
		t.Fatal("200 sneaks that rode the elevator down grew no sapling where they arrived")
	})
}
