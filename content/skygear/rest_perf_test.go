package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/machine"
)

// TestOwnsAfterBuyChunk: the owned-chunk set cache sees a chunk added after
// it was built.
func TestOwnsAfterBuyChunk(t *testing.T) {
	s := testSession(t)
	c := island.ChunkPos{5, 5}
	if island.Owns(s.Session, c) || island.CanBuild(s.Session, cube.Pos{5 * 16, 60, 5 * 16}) {
		t.Fatal("chunk owned before it was bought")
	}
	st := island.StateOf(s.Session)
	st.Chunks = append(st.Chunks, c)
	if !island.Owns(s.Session, c) || !island.CanBuild(s.Session, cube.Pos{5*16 + 3, 60, 5*16 + 3}) {
		t.Fatal("bought chunk not owned")
	}
	if !island.Owns(s.Session, st.Chunks[0]) {
		t.Fatal("starting chunk lost")
	}
}

// TestBatchedCyclesRun: cycles are passed on in the monitor's second.
func TestBatchedCyclesRun(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, machine.KindFurnace)
	_ = st.Inv().SetItem(11, item.NewStack(block.Cobblestone{}, 3))
	tick(s, 3*(machine.Kinds[machine.KindFurnace].Interval+2))
	runSeconds(s, 2)
	if machine.OutputCount(st, block.Stone{}) == 0 {
		t.Fatal("nothing smelted")
	}
}
