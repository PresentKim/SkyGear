package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/fluidnet"
	"github.com/presentkim/skygear/content/skygear/conduit"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/network"
)

// TestRestartKeepsBuffer: the buffer of a network survives an autosave, a
// world close and a reopen.
func TestRestartKeepsBuffer(t *testing.T) {
	dir := t.TempDir()
	a := cube.Pos{3, island.IslandY, 3}
	b := a.Add(cube.Pos{1, 0, 0})

	s, closeAll := openPersisted(t, dir)
	s.World.Do(func(tx *world.Tx) {
		tx.SetBlock(a, conduit.Bundle{Energy: 1, Fluid: 1, Buf: &conduit.BufCell{Energy: 150, FluidKind: int(fluidnet.Water), Fluid: 400}}, nil)
		tx.SetBlock(b, conduit.Bundle{Energy: 1, Fluid: 1}, nil)
		if e, _, _ := network.Energy(s.Session).Buffer(a); e != 150 {
			t.Fatalf("filled: %d", e)
		}
		s.Autosave(tx)
	})
	closeAll()

	s, _ = openPersisted(t, dir)
	s.World.Do(func(tx *world.Tx) {
		e, _, ok := network.Energy(s.Session).Buffer(a)
		if !ok || e != 150 {
			t.Errorf("energy after reopen = %d (in grid %v), want 150", e, ok)
		}
		k, f, _, ok := network.Fluids(s.Session).Buffer(a)
		if !ok || k != fluidnet.Water || f != 400 {
			t.Errorf("fluid after reopen = %v %d (in grid %v), want water 400", k, f, ok)
		}
	})
}

// TestVisualsAfterReload: a bundle of a column that was loaded again has a
// display entity.
func TestVisualsAfterReload(t *testing.T) {
	dir := t.TempDir()
	a := cube.Pos{3, island.IslandY, 3}

	s, closeAll := openPersisted(t, dir)
	s.World.Do(func(tx *world.Tx) { tx.SetBlock(a, conduit.Bundle{Energy: 1}, nil) })
	closeAll()

	s, _ = openPersisted(t, dir)
	s.World.Do(func(tx *world.Tx) {
		h, ok := conduit.LocalOf(s.Session).Visuals[a]
		if !ok {
			t.Fatal("no display entity after reload")
		}
		if _, ok := h.Entity(tx); !ok {
			t.Error("display entity not in the world")
		}
	})
}
