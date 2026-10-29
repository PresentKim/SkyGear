package skygear

import (
	"testing"

	"github.com/presentkim/skygear/content/skygear/machine"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// TestAdminDense: the dense cells hold a core with a miner over its output
// and a crusher, hopper and drawer beside it.
func TestAdminDense(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		var o cmd.Output
		debugDense{adminBase: adminBase{g: s.g}, Count: 8}.Run(p, &o, tx)
		if len(o.Errors()) > 0 {
			t.Fatal(o.Errors())
		}
		feet := cube.PosFromVec3(p.Position())
		cols, rows := 2, 4
		at := feet.Add(cube.Pos{-cols, -1, -rows / 2})
		if m, ok := tx.Block(at.Add(cube.Pos{0, 4, 0})).(machine.Machine); !ok || m.Kind != machine.KindMiner {
			t.Errorf("no miner in the first cell")
		}
		if _, ok := tx.Block(at.Add(cube.Pos{1, 2, 0})).(Drawer); !ok {
			t.Errorf("no drawer in the first cell")
		}
	})
}

// TestDebugPlace: /sgdebug place puts a vanilla block and a machine where
// the offsets say, through the player's own placing.
func TestDebugPlace(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s) // feet at (0, 64, 2)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		tx.SetBlock(cube.Pos{2, 63, 2}, block.Stone{}, nil)
		tx.SetBlock(cube.Pos{2, 63, 3}, block.Stone{}, nil)
		base := debugPlace{adminBase: adminBase{g: s.g}}
		var o cmd.Output
		c := base
		c.Item, c.DX = "cobblestone", 2
		c.Run(p, &o, tx)
		if _, ok := tx.Block(cube.Pos{2, 64, 2}).(block.Cobblestone); !ok {
			t.Fatalf("cobblestone not placed: %T, %v", tx.Block(cube.Pos{2, 64, 2}), o.Errors())
		}
		c = base
		c.Item, c.DX, c.DZ = machine.KindCrusher, 2, 1
		c.Run(p, &o, tx)
		if m, ok := tx.Block(cube.Pos{2, 64, 3}).(machine.Machine); !ok || m.Kind != machine.KindCrusher {
			t.Fatalf("crusher not placed: %T, %v", tx.Block(cube.Pos{2, 64, 3}), o.Errors())
		}
		if held, _ := p.HeldItems(); !held.Empty() {
			t.Fatalf("hand kept %v", held)
		}
	})
}

// TestDebugFloor: /sgdebug floor fills the air around the feet's level and
// keeps what is there.
func TestDebugFloor(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s) // feet at (0, 64, 2)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		tx.SetBlock(cube.Pos{1, 63, 2}, block.Dirt{}, nil)
		var o cmd.Output
		debugFloor{adminBase: adminBase{g: s.g}, Radius: 2}.Run(p, &o, tx)
		if _, ok := tx.Block(cube.Pos{2, 63, 4}).(block.Stone); !ok {
			t.Fatalf("no stone at the corner: %T", tx.Block(cube.Pos{2, 63, 4}))
		}
		if _, ok := tx.Block(cube.Pos{1, 63, 2}).(block.Dirt); !ok {
			t.Fatal("the dirt was replaced")
		}
	})
}
