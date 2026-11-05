package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/netstore"
)

func TestSneakingClickOpensTheTerminalOverEmptySlots(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	ctl := machinePos
	placeTestMachine(s, ctl, machine.KindNetController, "")
	placeTestMachine(s, ctl.Add(cube.Pos{1, 0, 0}), machine.KindNetTerminal, "")
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		for _, held := range []item.Stack{{}, item.NewStack(block.Stone{}, 1)} {
			s.ForgetOpen(p)
			useBlock(s, tx, p, ctl.Add(cube.Pos{1, 0, 0}), held, true)
			if !netstore.IsTerminal(tx.Block(ctl)) {
				t.Fatal("the controller is no terminal block")
			}
		}
	})
}
