package netstore_test

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
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/machine/machinetest"
	"github.com/presentkim/skygear/content/skygear/netstore"
)

func TestSneakingClickOpensTheTerminalOverEmptySlots(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	ctl := machinetest.Pos
	machinetest.Place(s, ctl, machine.KindNetController)
	machinetest.Place(s, ctl.Add(cube.Pos{1, 0, 0}), machine.KindNetTerminal)
	h := coretest.SpawnPlayer(s)
	coretest.WithPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		for _, held := range []item.Stack{{}, item.NewStack(block.Stone{}, 1)} {
			s.ForgetOpen(p)
			useBlock(s, tx, p, ctl.Add(cube.Pos{1, 0, 0}), held, true)
			if !netstore.IsTerminal(tx.Block(ctl)) {
				t.Fatal("the controller is no terminal block")
			}
		}
	})
}

func useBlock(s core.Lifecycle, tx *world.Tx, p *player.Player, pos cube.Pos, held item.Stack, sneak bool) bool {
	p.SetHeldItems(held, item.Stack{})
	if sneak {
		p.StartSneaking()
	} else {
		p.StopSneaking()
	}
	ctx := player.NewEventContext(tx, p)
	s.Handler(p).HandleItemUseOnBlock(ctx, pos, cube.FaceUp, mgl64.Vec3{0.5, 1, 0.5})
	if ctx.Cancelled() {
		return true
	}
	held, _ = p.HeldItems()
	act, ok := tx.Block(pos).(block.Activatable)
	return ok && (!sneak || held.Empty()) && act.Activate(pos, cube.FaceUp, tx, p, &item.UseContext{})
}
