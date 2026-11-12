package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/itemnet"
	"github.com/presentkim/skygear/content/skygear/conduit"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// hourglassClick sends one real click of p on the block at pos with n fairy
// hourglasses in hand (Player.UseItemOnBlock: the handler's subscribers, then
// the block's Activate unless p sneaks, then the held item's UseOnBlock) and
// returns how many are left.
func hourglassClick(p *player.Player, pos cube.Pos, face cube.Face, click mgl64.Vec3, sneak bool, n int) int {
	p.SetHeldItems(item.NewStack(sgitem.FairyHourglass{}, n), item.Stack{})
	if sneak {
		p.StartSneaking()
	} else {
		p.StopSneaking()
	}
	p.UseItemOnBlock(pos, face, click)
	held, _ := p.HeldItems()
	return held.Count()
}

// TestHourglassThroughTheMachine: a fairy hourglass winds the machine it is
// used on, standing (the machine's Activate takes it) or sneaking (the
// hourglass's own UseOnBlock). A connector plate on the clicked face takes a
// standing player's click before the machine is asked, and leaves a sneaking
// player's click alone. No subscriber of the click takes part in the winding,
// so the result is the same in every order of the subscribers of one tier.
func TestHourglassThroughTheMachine(t *testing.T) {
	s := testSession(t)
	island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
	loadAround(s)
	h := spawnPlayer(s)
	pos := cube.Pos{4, island.IslandY + 2, 4}
	gen := cube.Pos{4, island.IslandY + 2, 8}
	onPlate, offPlate, top := mgl64.Vec3{1, 0.5, 0.5}, mgl64.Vec3{1, 0.95, 0.95}, mgl64.Vec3{0.5, 1, 0.5}
	st := placeTestMachine(s, pos, machine.KindCrusher)
	genState := placeTestMachine(s, gen, machine.KindGenerator)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		b := conduit.Bundle{Item: 1}
		b.Modes[cube.FaceWest] = itemnet.Input
		at := pos.Side(cube.FaceEast)
		tx.SetBlock(at, b, nil)
		b.Refresh(at, tx)
		if _, _, _, ok := conduit.PlateHit(tx, pos, cube.FaceEast, onPlate); !ok {
			t.Fatal("the click is not on a connector plate")
		}
		if _, _, _, ok := conduit.PlateHit(tx, pos, cube.FaceEast, offPlate); ok {
			t.Fatal("the click beside the plate is on it")
		}
		p.Handle(s.Handler(p))
		p.Teleport(pos.Vec3Centre().Add(mgl64.Vec3{2, 1, 0}))

		for _, c := range []struct {
			name  string
			sneak bool
			face  cube.Face
			click mgl64.Vec3
			wound bool
		}{
			{"the plate, standing", false, cube.FaceEast, onPlate, false},
			{"the plate, sneaking", true, cube.FaceEast, onPlate, true},
			{"beside the plate, standing", false, cube.FaceEast, offPlate, true},
			{"beside the plate, sneaking", true, cube.FaceEast, offPlate, true},
			{"the top, standing", false, cube.FaceUp, top, true},
			{"the top, sneaking", true, cube.FaceUp, top, true},
		} {
			st.Boost = 0
			left := hourglassClick(p, pos, c.face, c.click, c.sneak, 5)
			wantBoost, wantLeft := 0, 5
			if c.wound {
				wantBoost, wantLeft = machine.HourglassTicks, 4
			}
			if st.Boost != wantBoost || left != wantLeft {
				t.Errorf("%s: wound by %d ticks with %d hourglasses left, want %d and %d", c.name, st.Boost, left, wantBoost, wantLeft)
			}
		}

		// A machine that takes no more keeps its click: nothing is spent.
		for _, sneak := range []bool{false, true} {
			st.Boost = machine.HourglassMaxTicks
			if left := hourglassClick(p, pos, cube.FaceUp, top, sneak, 5); left != 5 || st.Boost != machine.HourglassMaxTicks {
				t.Errorf("a wound-up machine, sneaking=%v: %d ticks, %d hourglasses left, want %d and 5", sneak, st.Boost, left, machine.HourglassMaxTicks)
			}
		}
		// A generator is not wound.
		p.Teleport(gen.Vec3Centre().Add(mgl64.Vec3{2, 1, 0}))
		for _, sneak := range []bool{false, true} {
			if left := hourglassClick(p, gen, cube.FaceUp, top, sneak, 5); left != 5 || genState.Boost != 0 {
				t.Errorf("a generator, sneaking=%v: %d ticks, %d hourglasses left, want 0 and 5", sneak, genState.Boost, left)
			}
		}
		// In creative the hourglass winds and is not spent.
		p.Teleport(pos.Vec3Centre().Add(mgl64.Vec3{2, 1, 0}))
		p.SetGameMode(world.GameModeCreative)
		for _, sneak := range []bool{false, true} {
			st.Boost = 0
			if left := hourglassClick(p, pos, cube.FaceUp, top, sneak, 5); left != 5 || st.Boost != machine.HourglassTicks {
				t.Errorf("creative, sneaking=%v: %d ticks, %d hourglasses left, want %d and 5", sneak, st.Boost, left, machine.HourglassTicks)
			}
		}
	})
}

// TestHourglassOnTheOutwardFaceIsRefused: on the face of a machine that looks
// out of the island (a machine in the island's last row) the side guard
// refuses the click, as it refuses opening that machine there: the hourglass
// is used by the machine and by the item, which Dragonfly asks only when no
// subscriber refused the click. The inward faces of the same machine wind it.
func TestHourglassOnTheOutwardFaceIsRefused(t *testing.T) {
	s := testSession(t)
	island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
	loadAround(s)
	h := spawnPlayer(s)
	edge := cube.Pos{15, island.IslandY + 2, 4}
	st := placeTestMachine(s, edge, machine.KindCrusher)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		if !island.CanBuild(s.Session, edge) || island.CanBuild(s.Session, edge.Side(cube.FaceEast)) {
			t.Fatal("the machine is not in the island's last row")
		}
		p.Handle(s.Handler(p))
		p.Teleport(edge.Vec3Centre().Add(mgl64.Vec3{-2, 1, 0}))
		for _, sneak := range []bool{false, true} {
			st.Boost = 0
			if left := hourglassClick(p, edge, cube.FaceEast, mgl64.Vec3{1, 0.5, 0.5}, sneak, 5); left != 5 || st.Boost != 0 {
				t.Errorf("the outward face, sneaking=%v: wound by %d ticks with %d hourglasses left, want 0 and 5", sneak, st.Boost, left)
			}
			st.Boost = 0
			if left := hourglassClick(p, edge, cube.FaceWest, mgl64.Vec3{0, 0.5, 0.5}, sneak, 5); left != 4 || st.Boost != machine.HourglassTicks {
				t.Errorf("an inward face, sneaking=%v: wound by %d ticks with %d hourglasses left, want %d and 4", sneak, st.Boost, left, machine.HourglassTicks)
			}
		}
	})
}

// armSwings counts the arm swings a viewer of the world is shown.
type armSwings struct {
	world.NopViewer
	n int
}

func (v *armSwings) ViewEntityAction(_ world.Entity, a world.EntityAction) {
	if _, ok := a.(entity.SwingArmAction); ok {
		v.n++
	}
}

// TestHourglassWindShowsTheArmSwing: winding a machine is a click the machine
// (or the hourglass) took, and Dragonfly shows the arm swing of such a click
// to the players around, as it does for a tier kit. A click a subscriber
// cancels is not shown: the plate's.
func TestHourglassWindShowsTheArmSwing(t *testing.T) {
	s := testSession(t)
	island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
	loadAround(s)
	h := spawnPlayer(s)
	pos := cube.Pos{4, island.IslandY + 2, 4}
	top := mgl64.Vec3{0.5, 1, 0.5}
	st := placeTestMachine(s, pos, machine.KindCrusher)
	seen := &armSwings{}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		l := world.NewLoader(2, s.World, seen)
		l.Move(tx, pos.Vec3Centre())
		l.Load(tx, 50)
		p.Handle(s.Handler(p))
		p.Teleport(pos.Vec3Centre().Add(mgl64.Vec3{2, 1, 0}))
		for _, sneak := range []bool{false, true} {
			st.Boost, seen.n = 0, 0
			if left := hourglassClick(p, pos, cube.FaceUp, top, sneak, 5); left != 4 {
				t.Fatalf("sneaking=%v: the hourglass was not used", sneak)
			}
			if seen.n == 0 {
				t.Errorf("sneaking=%v: the players around saw no arm swing when the hourglass wound the machine", sneak)
			}
		}
	})
}
