package instance

import (
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
)

func TestSafeSpot(t *testing.T) {
	w := world.Config{Synchronous: true, Entities: entity.DefaultRegistry}.New()
	defer w.Close()
	<-w.Do(func(tx *world.Tx) {
		floor := func(x, z int) { tx.SetBlock(cube.Pos{x, 63, z}, block.Stone{}, nil) }
		for x := -3; x <= 3; x++ {
			for z := -3; z <= 3; z++ {
				floor(x, z)
			}
		}
		free := mgl64.Vec3{0.5, 64, 0.5}
		if got := SafeSpot(tx, free); got != free {
			t.Fatalf("free spot moved to %v", got)
		}
		// A block where the player stood: stand on top of it.
		tx.SetBlock(cube.Pos{0, 64, 0}, block.Planks{}, nil)
		if got := SafeSpot(tx, free); got != (mgl64.Vec3{0.5, 65, 0.5}) {
			t.Fatalf("on top = %v", got)
		}
		// A pillar too high to climb: step beside it.
		for y := 64; y <= 70; y++ {
			tx.SetBlock(cube.Pos{0, y, 0}, block.Planks{}, nil)
		}
		got := SafeSpot(tx, free)
		if c := cube.PosFromVec3(got); c.Y() != 64 || (c.X() == 0 && c.Z() == 0) {
			t.Fatalf("beside = %v", got)
		}
	}).Done()
}

// TestQuitRestoresPosition: disconnecting inside a save stores the
// position, and entering again puts the player there.
func TestQuitRestoresPosition(t *testing.T) {
	f := setup(t)
	id := uuid.New()
	if _, err := f.d.TouchAccount(id.String(), "", "host"); err != nil {
		t.Fatal(err)
	}
	join := func() *world.EntityHandle {
		h := world.EntitySpawnOpts{ID: id}.New(player.Type, player.Config{Name: "host", UUID: id, Position: mgl64.Vec3{0, 100, 0}})
		<-f.lobby.Do(func(tx *world.Tx) { f.m.Join(tx.AddEntity(h).(*player.Player)) }).Done()
		return h
	}
	save, _ := f.d.CreateSave(id.String(), "test", 1, "s")
	h := join()
	in(t, h, func(p *player.Player) {
		if err := f.m.Enter(p, save.ID); err != nil {
			t.Fatal(err)
		}
	})
	in(t, h, func(p *player.Player) {
		p.Teleport(mgl64.Vec3{0.5, 64, 0.5}) // on the stone Init placed
		_ = p.Close()                        // like a disconnect: HandleQuit stores the state
	})
	waitInactive(t, f.m, save.ID)
	time.Sleep(50 * time.Millisecond)

	h = join()
	in(t, h, func(p *player.Player) {
		if err := f.m.Enter(p, save.ID); err != nil {
			t.Fatal(err)
		}
	})
	in(t, h, func(p *player.Player) {
		if pos := p.Position(); pos != (mgl64.Vec3{0.5, 64, 0.5}) {
			t.Fatalf("position after rejoin = %v", pos)
		}
	})
	f.m.CloseAll()
}

// TestGrounded: a free position above a floor is a landing spot, one over
// air (the void) is not.
func TestGrounded(t *testing.T) {
	w := world.Config{Synchronous: true, Entities: entity.DefaultRegistry}.New()
	defer w.Close()
	w.Do(func(tx *world.Tx) {
		tx.SetBlock(cube.Pos{0, 63, 0}, block.Stone{}, nil)
		if !grounded(tx, cube.Pos{0, 64, 0}) {
			t.Error("above the floor")
		}
		if grounded(tx, cube.Pos{0, 70, 0}) && !solid(tx, cube.Pos{0, 66, 0}) {
			t.Error("the floor is 4 blocks down at most")
		}
		if grounded(tx, cube.Pos{5, 64, 5}) {
			t.Error("over air")
		}
	})
}

// TestLandingSpotFallsBack: a saved position over the void lands at the
// fallback when that has ground.
func TestLandingSpotFallsBack(t *testing.T) {
	w := world.Config{Synchronous: true, Entities: entity.DefaultRegistry}.New()
	defer w.Close()
	w.Do(func(tx *world.Tx) {
		tx.SetBlock(cube.Pos{0, 63, 0}, block.Stone{}, nil)
		got := LandingSpot(tx, mgl64.Vec3{20.5, 64, 20.5}, mgl64.Vec3{0.5, 64, 0.5})
		if got.Y() != 64 || got.X() != 0.5 {
			t.Errorf("landing spot %v, want the fallback", got)
		}
	})
}
