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
	"github.com/presentkim/skygear/content/skygear/conduit"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/storage"
)

// What a block leaves behind when it goes is cleaned up by the block's own
// callbacks, which Dragonfly calls after the handler's subscribers. The pin
// scenarios call the handler only, so these tests remove the block through
// Dragonfly: a player's finished break (Player.BreakBlock) and an explosion.

// brokenSession is a session with a player who stands next to pos and may
// break it.
func brokenSession(t *testing.T, pos cube.Pos) (*session, *world.EntityHandle) {
	t.Helper()
	s := testSession(t)
	island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		p.Handle(s.Handler(p))
		p.Teleport(pos.Vec3Centre().Add(mgl64.Vec3{0, 1, 2}))
	})
	return s, h
}

// blowUp removes the blocks around pos as TNT that goes off beside it does.
func blowUp(tx *world.Tx, pos cube.Pos) {
	block.ExplosionConfig{}.Explode(tx, world.BlockExplosionSource{Pos: pos.Side(cube.FaceUp), ExplosionSize: 4})
}

// TestBrokenBundleLosesItsDisplay: a conduit bundle a player breaks loses its
// display and its connector plates, and so does one an explosion removes.
// No subscriber of the break does it: the bundle's BlockChanged does.
func TestBrokenBundleLosesItsDisplay(t *testing.T) {
	chest := cube.Pos{4, island.IslandY + 2, 4}
	for _, c := range []struct {
		name   string
		remove func(p *player.Player, tx *world.Tx, at cube.Pos)
	}{
		{"broken by hand", func(p *player.Player, _ *world.Tx, at cube.Pos) { p.BreakBlock(at) }},
		{"blown up", func(_ *player.Player, tx *world.Tx, at cube.Pos) { blowUp(tx, at) }},
	} {
		s, h := brokenSession(t, chest.Side(cube.FaceEast))
		withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
			at := chestWithBundle(tx, chest, itemnet.Input, 5, 9)
			l := conduit.LocalOf(s.Session)
			if _, ok := l.Visuals[at]; !ok {
				t.Fatalf("%s: the bundle has no display", c.name)
			}
			if len(l.Plates) == 0 {
				t.Fatalf("%s: the bundle has no connector plate", c.name)
			}
			c.remove(p, tx, at)
			if _, still := tx.Block(at).(conduit.Bundle); still {
				t.Fatalf("%s: the bundle is still there", c.name)
			}
			if _, ok := l.Visuals[at]; ok {
				t.Errorf("%s: the bundle's display is still there", c.name)
			}
			if len(l.Plates) != 0 {
				t.Errorf("%s: %d connector plates are still there", c.name, len(l.Plates))
			}
		})
	}
}

// TestBrokenMachineIsTakenDown: a machine a player breaks is forgotten by the
// monitor (and its windows close: the same call, machine.Break), and so is
// one an explosion removes. The machine's BreakInfo does it, which Dragonfly
// calls for every break; no subscriber of the player's break does.
func TestBrokenMachineIsTakenDown(t *testing.T) {
	pos := cube.Pos{4, island.IslandY + 2, 4}
	for _, c := range []struct {
		name   string
		remove func(p *player.Player, tx *world.Tx)
	}{
		{"broken by hand", func(p *player.Player, _ *world.Tx) { p.BreakBlock(pos) }},
		{"blown up", func(_ *player.Player, tx *world.Tx) { blowUp(tx, pos) }},
	} {
		s, h := brokenSession(t, pos)
		placeTestMachine(s, pos, machine.KindCrusher)
		tick(s, 2)
		if _, ok := machine.Ticking(s.Session, machine.KindCrusher)[pos]; !ok {
			t.Fatalf("%s: the monitor does not know the machine", c.name)
		}
		withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
			c.remove(p, tx)
			if _, still := tx.Block(pos).(machine.Machine); still {
				t.Fatalf("%s: the machine is still there", c.name)
			}
		})
		if _, ok := machine.Ticking(s.Session, machine.KindCrusher)[pos]; ok {
			t.Errorf("%s: the monitor still lists the machine", c.name)
		}
	}
}

// explode removes the one block at pos as an explosion removes a block: it
// clears the block and then runs the block's break handler without a player
// (block.ExplosionConfig.Explode). For where a blast would also remove what
// the test reads afterwards.
func explode(tx *world.Tx, pos cube.Pos) {
	b := tx.Block(pos)
	tx.SetBlock(pos, nil, nil)
	if h := b.(block.Breakable).BreakInfo().BreakHandler; h != nil {
		h(pos, tx, nil)
	}
}

// TestBrokenDrawerLosesItsDisplay: a drawer a player breaks loses its display
// and its label, and so does one an explosion removes. The drawer's BreakInfo
// does it; no subscriber of the player's break does.
func TestBrokenDrawerLosesItsDisplay(t *testing.T) {
	pos := cube.Pos{4, island.IslandY + 2, 4}
	for _, c := range []struct {
		name   string
		remove func(p *player.Player, tx *world.Tx)
	}{
		{"broken by hand", func(p *player.Player, _ *world.Tx) { p.BreakBlock(pos) }},
		{"blown up", func(_ *player.Player, tx *world.Tx) { blowUp(tx, pos) }},
	} {
		s, h := brokenSession(t, pos)
		withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
			tx.SetBlock(pos, storage.Drawer{Facing: cube.South, Item: item.NewStack(block.Dirt{}, 1), Count: 70}, nil)
		})
		tick(s, 40)
		withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
			if _, ok := storage.LocalOf(s.Session).Displays[pos]; !ok {
				t.Fatalf("%s: the drawer has no display", c.name)
			}
			p.Teleport(pos.Vec3Centre().Add(mgl64.Vec3{0, 1, 2})) // the ticks let the player fall
			c.remove(p, tx)
			if _, still := tx.Block(pos).(storage.Drawer); still {
				t.Fatalf("%s: the drawer is still there", c.name)
			}
			if _, ok := storage.LocalOf(s.Session).Displays[pos]; ok {
				t.Errorf("%s: the drawer's display is still there", c.name)
			}
		})
		if shapeAs(s, storage.DrawerLabelKey(pos)) != nil {
			t.Errorf("%s: the drawer's label is still shown", c.name)
		}
	}
}

// TestBrokenTrimCutsTheDrawerSet: a controller reaches a drawer through a
// drawer trim. When the trim is broken, by a player or by an explosion, the
// controller no longer reaches the drawer at once: the set kinds tell the
// storage that one of them was broken (machine.Behaviour.Broken), and the
// networks it remembers for the tick are dropped.
func TestBrokenTrimCutsTheDrawerSet(t *testing.T) {
	ctl := cube.Pos{4, island.IslandY + 2, 4}
	trim, drawer := ctl.Add(cube.Pos{0, 0, 1}), ctl.Add(cube.Pos{0, 0, 2})
	for _, c := range []struct {
		name   string
		remove func(p *player.Player, tx *world.Tx)
	}{
		{"broken by hand", func(p *player.Player, _ *world.Tx) { p.BreakBlock(trim) }},
		{"removed as an explosion removes it", func(_ *player.Player, tx *world.Tx) { explode(tx, trim) }},
	} {
		s, h := brokenSession(t, trim)
		placeTestMachine(s, ctl, machine.KindDrawerController)
		placeTestMachine(s, trim, machine.KindDrawerTrim)
		withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
			tx.SetBlock(drawer, storage.Drawer{Facing: cube.South}, nil)
			storage.InvalidateNetworks(s.Session)
			if n := len(storage.Network(tx, ctl)); n != 1 {
				t.Fatalf("%s: the controller reaches %d drawers through the trim, want 1", c.name, n)
			}
			c.remove(p, tx)
			if _, still := tx.Block(trim).(machine.Machine); still {
				t.Fatalf("%s: the trim is still there", c.name)
			}
			if n := len(storage.Network(tx, ctl)); n != 0 {
				t.Errorf("%s: the controller still reaches %d drawers", c.name, n)
			}
		})
	}
}
