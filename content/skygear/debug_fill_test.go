package skygear

import (
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// fillEntities counts the conduit display entities of tx.
func fillEntities(tx *world.Tx) (n int) {
	for e := range tx.Entities() {
		if _, ok := e.(interface{ MarkVariant() int32 }); ok {
			n++
		}
	}
	return n
}

// TestDebugFill: each mode fills the air of the cube (and only air), the fill
// is taken away on stop and after its time, and one fill runs at a time.
func TestDebugFill(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s) // feet at 0.5 64 2.5
	centre := cube.Pos{0, 64, 2}
	stone := centre.Add(cube.Pos{1, 0, 0})
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		tx.SetBlock(stone, block.Stone{}, nil)
	})
	const cube1 = 27 // radius 1; the stone is not filled
	run := func(p *player.Player, tx *world.Tx, radius, seconds int, what string) *cmd.Output {
		t.Helper()
		var o cmd.Output
		debugFill{adminBase: adminBase{g: s.g}, Radius: radius, Seconds: seconds, What: fillEnum(what)}.Run(p, &o, tx)
		return &o
	}
	stop := func(p *player.Player, tx *world.Tx) *cmd.Output {
		t.Helper()
		var o cmd.Output
		debugFillStop{adminBase: adminBase{g: s.g}}.Run(p, &o, tx)
		return &o
	}
	// layers reports what the cube holds: blocks on layer 0 other than air and
	// stone, and on layer 1.
	layers := func(tx *world.Tx) (l0, l1 int) {
		for x := -1; x <= 1; x++ {
			for y := -1; y <= 1; y++ {
				for z := -1; z <= 1; z++ {
					pos := centre.Add(cube.Pos{x, y, z})
					if pos == stone {
						continue
					}
					if _, air := tx.BlockLayer(pos, 0).(block.Air); !air {
						l0++
					}
					if _, air := tx.BlockLayer(pos, 1).(block.Air); !air {
						l1++
					}
				}
			}
		}
		return l0, l1
	}
	old := fillUnit
	fillUnit = time.Millisecond
	t.Cleanup(func() { fillUnit = old })

	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		if o := run(p, tx, 0, 10, "layers"); len(o.Errors()) != 1 {
			t.Errorf("radius 0: %v", o.Errors())
		}
		if o := run(p, tx, 1, 4, "layers"); len(o.Errors()) != 1 {
			t.Errorf("4 seconds: %v", o.Errors())
		}
		if o := stop(p, tx); len(o.Errors()) != 1 {
			t.Errorf("stop with no fill: %v", o.Errors())
		}

		// layers: probes on layers 0 and 1 in every air position.
		if o := run(p, tx, 1, 600, "layers"); len(o.Errors()) != 0 || len(o.Messages()) != 1 {
			t.Fatalf("layers: %v %v", o.Errors(), o.Messages())
		}
		if l0, l1 := layers(tx); l0 != cube1-1 || l1 != cube1-1 {
			t.Errorf("layers: %d on layer 0, %d on layer 1, want %d each", l0, l1, cube1-1)
		}
		if tx.Block(stone) != (block.Stone{}) {
			t.Errorf("layers: the stone is %#v", tx.Block(stone))
		}
		if o := run(p, tx, 1, 10, "entity"); len(o.Errors()) != 1 {
			t.Errorf("second fill: %v", o.Errors())
		}
		if o := stop(p, tx); len(o.Errors()) != 0 {
			t.Errorf("stop: %v", o.Errors())
		}
		if l0, l1 := layers(tx); l0 != 0 || l1 != 0 {
			t.Errorf("stop: %d on layer 0, %d on layer 1 left", l0, l1)
		}
		if tx.Block(stone) != (block.Stone{}) {
			t.Errorf("stop: the stone is %#v", tx.Block(stone))
		}

		// block: the state probe may be off, then the layer probe is used.
		run(p, tx, 1, 600, "block")
		if l0, _ := layers(tx); l0 != cube1-1 {
			t.Errorf("block: %d on layer 0, want %d", l0, cube1-1)
		}
		stop(p, tx)
		if l0, l1 := layers(tx); l0 != 0 || l1 != 0 {
			t.Errorf("block stop: %d on layer 0, %d on layer 1 left", l0, l1)
		}

		// entity: one display entity per position, no block.
		run(p, tx, 1, 600, "entity")
		if n := fillEntities(tx); n != cube1-1 {
			t.Errorf("entity: %d entities, want %d", n, cube1-1)
		}
		if l0, l1 := layers(tx); l0 != 0 || l1 != 0 {
			t.Errorf("entity: %d on layer 0, %d on layer 1", l0, l1)
		}
	})
	// The visuals check their bundle every 20 ticks: they stay.
	tick(s, 60)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		if n := fillEntities(tx); n != cube1-1 {
			t.Errorf("after 60 ticks: %d entities, want %d", n, cube1-1)
		}
		stop(p, tx)
		if n := fillEntities(tx); n != 0 {
			t.Errorf("entity stop: %d entities left", n)
		}
	})

	// After its time a fill takes itself away, and what changed meanwhile stays.
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		p.Teleport(mgl64.Vec3{0.5, 64, 2.5}) // it fell during the ticks
		if o := run(p, tx, 1, fillMinSeconds, "layers"); len(o.Errors()) > 0 {
			t.Fatalf("timed fill: %v", o.Errors())
		}
		tx.SetBlock(centre.Add(cube.Pos{0, 1, 0}), block.Dirt{}, nil) // replaced
	})
	deadline := time.Now().Add(5 * time.Second)
	for {
		done := false
		withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
			l0, l1 := layers(tx)
			done = l0 == 1 && l1 == 0
		})
		if done {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the fill was not taken away in time")
		}
		time.Sleep(5 * time.Millisecond)
	}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		if tx.Block(centre.Add(cube.Pos{0, 1, 0})) != (block.Dirt{}) {
			t.Errorf("the dirt placed in between was removed")
		}
		if o := stop(p, tx); len(o.Errors()) != 1 {
			t.Errorf("stop after the end: %v", o.Errors())
		}
	})
}

// TestDebugFillCap: a cube of more than the cap in air fills the cap only.
func TestDebugFillCap(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	s.World.Do(func(tx *world.Tx) {
		pos, capped := fillPositions(tx, cube.Pos{0, 80, 0}, fillMaxRadius)
		if len(pos) != fillMaxPositions || !capped {
			t.Errorf("%d positions (capped %v), want %d capped", len(pos), capped, fillMaxPositions)
		}
		pos, capped = fillPositions(tx, cube.Pos{0, 80, 0}, 1)
		if len(pos) != 27 || capped {
			t.Errorf("%d positions (capped %v), want 27", len(pos), capped)
		}
	})
}
