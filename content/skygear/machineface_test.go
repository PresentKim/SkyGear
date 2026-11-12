package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/itemnet"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/machine"
)

// machineAt returns the machine at pos, set up (with a state).
func machineAt(t *testing.T, tx *world.Tx, pos cube.Pos) machine.Machine {
	t.Helper()
	m, ok := tx.Block(pos).(machine.Machine)
	if !ok || m.State == nil {
		t.Fatalf("no machine was set up at %v: %T", pos, tx.Block(pos))
	}
	return m
}

// TestMachineItemPlacesAndFaces: a machine item is placed by real clicks
// (Player.UseItemOnBlock: the handler's subscribers, the clicked block's
// Activate, then the held item's UseOnBlock, which places the machine). The
// item hands the placement the face it was used on: a miner and a network
// attachment face the clicked block, another machine does not turn. A click
// places like Dragonfly places a block item: beside the clicked face, one
// item spent, none in creative, nothing where the place is taken or the
// island's guard refuses.
func TestMachineItemPlacesAndFaces(t *testing.T) {
	s := testSession(t)
	island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
	loadAround(s)
	h := spawnPlayer(s)
	base := cube.Pos{4, island.IslandY + 2, 4}
	side, top := mgl64.Vec3{1, 0.5, 0.5}, mgl64.Vec3{0.5, 1, 0.5}
	place := func(p *player.Player, kind string, n int, pos cube.Pos, face cube.Face, click mgl64.Vec3) int {
		p.SetHeldItems(item.NewStack(machine.Machine{Kind: kind}, n), item.Stack{})
		p.UseItemOnBlock(pos, face, click)
		held, _ := p.HeldItems()
		return held.Count()
	}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		p.Handle(s.Handler(p))
		p.Teleport(base.Vec3Centre().Add(mgl64.Vec3{0, 3, 3}))
		for i := range 4 {
			tx.SetBlock(base.Add(cube.Pos{0, 0, i * 2}), block.Stone{}, nil)
		}
		if left := place(p, machine.KindMiner, 3, base, cube.FaceEast, side); left != 2 {
			t.Errorf("%d miners left after placing one, want 2", left)
		}
		if left := place(p, machine.KindNetImporter, 3, base.Add(cube.Pos{0, 0, 2}), cube.FaceUp, top); left != 2 {
			t.Errorf("%d importers left after placing one, want 2", left)
		}
		if left := place(p, machine.KindCrusher, 3, base.Add(cube.Pos{0, 0, 4}), cube.FaceEast, side); left != 2 {
			t.Errorf("%d crushers left after placing one, want 2", left)
		}
		// the place beside the clicked face is taken: nothing is placed
		if left := place(p, machine.KindCrusher, 3, base.Add(cube.Pos{0, 0, 4}), cube.FaceEast, side); left != 3 {
			t.Errorf("%d crushers left after a click on a taken place, want 3", left)
		}
		p.SetGameMode(world.GameModeCreative)
		if left := place(p, machine.KindMiner, 3, base.Add(cube.Pos{0, 0, 6}), cube.FaceNorth, mgl64.Vec3{0.5, 0.5, 0}); left != 3 {
			t.Errorf("%d miners left after placing one in creative, want 3", left)
		}
		p.SetGameMode(world.GameModeSurvival)
	})
	// The placed machines were set up when the placing transaction ended.
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		if m := machineAt(t, tx, base.Side(cube.FaceEast)); m.Kind != machine.KindMiner || m.State.Facing != cube.FaceWest {
			t.Errorf("the miner placed on an east face is a %s facing %v, want a miner facing west", m.Kind, m.State.Facing)
		}
		imp := machineAt(t, tx, base.Add(cube.Pos{0, 1, 2}))
		if imp.Kind != machine.KindNetImporter || imp.State.Facing != cube.FaceDown {
			t.Errorf("the importer placed on a top face is a %s facing %v, want an importer facing down", imp.Kind, imp.State.Facing)
		}
		if m := machineAt(t, tx, base.Add(cube.Pos{1, 0, 4})); m.Kind != machine.KindCrusher || m.State.Facing != cube.FaceSouth {
			t.Errorf("the crusher is a %s facing %v, want a crusher that did not turn (south)", m.Kind, m.State.Facing)
		}
		if m := machineAt(t, tx, base.Add(cube.Pos{0, 0, 5})); m.State.Facing != cube.FaceSouth {
			t.Errorf("the miner placed on a north face in creative faces %v, want south", m.State.Facing)
		}
	})
	// On the last row a click on the outward face is refused: nothing placed.
	edge := cube.Pos{15, island.IslandY + 2, 4}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		tx.SetBlock(edge, block.Stone{}, nil)
		p.Teleport(edge.Vec3Centre().Add(mgl64.Vec3{-2, 1, 0}))
		if left := place(p, machine.KindMiner, 3, edge, cube.FaceEast, side); left != 3 {
			t.Errorf("%d miners left after a click the side guard refuses, want 3", left)
		}
		if _, ok := tx.Block(edge.Side(cube.FaceEast)).(machine.Machine); ok {
			t.Error("a miner was placed outside the island")
		}
	})
}

// TestMachinePlacedWithoutAClickFacesItsDefault: the face a miner is placed
// by belongs to the click that places it. A click with a miner in hand that
// places nothing (a connector plate takes it; a chest opens) leaves nothing
// behind: a miner placed afterwards without a click of its own, as the
// construction wand places (the handler's HandleBlockPlace), faces down.
func TestMachinePlacedWithoutAClickFacesItsDefault(t *testing.T) {
	s := testSession(t)
	island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
	loadAround(s)
	h := spawnPlayer(s)
	chest := cube.Pos{4, island.IslandY + 2, 4}
	at := cube.Pos{8, island.IslandY + 2, 8}
	n := 0
	placed := func() cube.Face {
		n++
		pos := at.Add(cube.Pos{0, 0, n})
		withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
			s.Handler(p).HandleBlockPlace(player.NewEventContext(tx, p), pos, machine.New(machine.KindMiner))
		})
		var facing cube.Face
		withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) { facing = machineAt(t, tx, pos).State.Facing })
		return facing
	}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		p.Handle(s.Handler(p))
		p.Teleport(chest.Vec3Centre().Add(mgl64.Vec3{2, 1, 0}))
		chestWithBundle(tx, chest, itemnet.Input, 5, 9)
		p.SetHeldItems(item.NewStack(machine.Machine{Kind: machine.KindMiner}, 3), item.Stack{})
		// the connector plate on the chest's east face takes the click
		p.UseItemOnBlock(chest, cube.FaceEast, mgl64.Vec3{1, 0.5, 0.5})
		if held, _ := p.HeldItems(); held.Count() != 3 {
			t.Fatalf("%d miners left after a click on a connector plate, want 3", held.Count())
		}
	})
	if got := placed(); got != cube.FaceDown {
		t.Errorf("a miner placed after a connector plate took a click faces %v, want down", got)
	}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		// the chest opens on its west face: the click passes every
		// subscriber and places nothing
		p.SetHeldItems(item.NewStack(machine.Machine{Kind: machine.KindMiner}, 3), item.Stack{})
		p.UseItemOnBlock(chest, cube.FaceWest, mgl64.Vec3{0, 0.5, 0.5})
		if held, _ := p.HeldItems(); held.Count() != 3 {
			t.Fatalf("%d miners left after a click that opened a chest, want 3", held.Count())
		}
	})
	if got := placed(); got != cube.FaceDown {
		t.Errorf("a miner placed after a click that opened a chest faces %v, want down", got)
	}
}
