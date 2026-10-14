package dragonfactory

import (
	"testing"

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
		adminDense{adminBase: adminBase{g: s.g}, Count: 8}.Run(p, &o, tx)
		if len(o.Errors()) > 0 {
			t.Fatal(o.Errors())
		}
		feet := cube.PosFromVec3(p.Position())
		cols, rows := 2, 4
		at := feet.Add(cube.Pos{-cols, -1, -rows / 2})
		if m, ok := tx.Block(at.Add(cube.Pos{0, 4, 0})).(Machine); !ok || m.Kind != KindMiner {
			t.Errorf("no miner in the first cell")
		}
		if _, ok := tx.Block(at.Add(cube.Pos{1, 2, 0})).(Drawer); !ok {
			t.Errorf("no drawer in the first cell")
		}
	})
}
