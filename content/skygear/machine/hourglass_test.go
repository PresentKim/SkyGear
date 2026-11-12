package machine_test

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/core/coretest"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/machine/machinetest"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// TestHourglassIsUsedByTheMachine: the machine's Activate winds it with the
// hourglass in hand and asks for one to be spent, and the hourglass's
// UseOnBlock does the same for the click Activate is not handed (a sneaking
// player's). One the machine does not take is not spent, and the hourglass
// used on another block does nothing.
func TestHourglassIsUsedByTheMachine(t *testing.T) {
	s := coretest.Session(t)
	island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
	coretest.LoadAround(s)
	st := machinetest.Place(s, machinetest.Pos, machine.KindCrusher)
	h := coretest.SpawnPlayer(s)
	coretest.WithPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		pos := machinetest.Pos
		m := tx.Block(pos).(machine.Machine)
		p.SetHeldItems(item.NewStack(sgitem.FairyHourglass{}, 2), item.Stack{})

		ctx := &item.UseContext{}
		if !m.Activate(pos, cube.FaceUp, tx, p, ctx) || st.Boost != machine.HourglassTicks || ctx.CountSub != 1 {
			t.Errorf("Activate with an hourglass: wound by %d ticks, %d spent, want %d and 1", st.Boost, ctx.CountSub, machine.HourglassTicks)
		}

		use, ok := world.Item(sgitem.FairyHourglass{}).(item.UsableOnBlock)
		if !ok {
			t.Fatal("the fairy hourglass is not used on a block")
		}
		st.Boost = 0
		ctx = &item.UseContext{}
		if !use.UseOnBlock(pos, cube.FaceUp, mgl64.Vec3{}, tx, p, ctx) || st.Boost != machine.HourglassTicks || ctx.CountSub != 1 {
			t.Errorf("UseOnBlock on a machine: wound by %d ticks, %d spent, want %d and 1", st.Boost, ctx.CountSub, machine.HourglassTicks)
		}

		st.Boost = machine.HourglassMaxTicks
		ctx = &item.UseContext{}
		if use.UseOnBlock(pos, cube.FaceUp, mgl64.Vec3{}, tx, p, ctx) || ctx.CountSub != 0 || st.Boost != machine.HourglassMaxTicks {
			t.Errorf("UseOnBlock on a wound-up machine: %d ticks, %d spent", st.Boost, ctx.CountSub)
		}
		ctx = &item.UseContext{}
		if use.UseOnBlock(pos.Side(cube.FaceDown), cube.FaceUp, mgl64.Vec3{}, tx, p, ctx) || ctx.CountSub != 0 {
			t.Error("UseOnBlock on a block that is no machine took the hourglass")
		}
	})
}
