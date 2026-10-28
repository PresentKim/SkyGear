package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

// TestLeavesDropSaplingsOften: oak leaves give a sapling about one time in
// five (vanilla one in twenty).
func TestLeavesDropSaplingsOften(t *testing.T) {
	leaves := block.Leaves{Type: block.OakLeaves()}
	const n = 4000
	got := 0
	for range n {
		for _, d := range leaves.BreakInfo().Drops(item.ToolNone{}, nil) {
			if _, ok := d.Item().(block.Sapling); ok {
				got++
			}
		}
	}
	if got < n*15/100 || got > n*25/100 {
		t.Errorf("%d saplings from %d leaves", got, n)
	}
}

// TestLeavesDropNoSticks: a stick is only for after its research, so
// leaves never drop one.
func TestLeavesDropNoSticks(t *testing.T) {
	leaves := block.Leaves{Type: block.OakLeaves()}
	for range 5000 {
		for _, d := range leaves.BreakInfo().Drops(item.ToolNone{}, nil) {
			if _, ok := d.Item().(item.Stick); ok {
				t.Fatal("leaves dropped a stick")
			}
		}
	}
}

// TestLeavesDecayOnScheduledTick: leaves with no log near are gone at their
// scheduled tick and drop what leaves drop; with a log beside them they
// stay.
func TestLeavesDecayOnScheduledTick(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	s.world.Do(func(tx *world.Tx) {
		alone, near := cube.Pos{6, IslandY + 5, 6}, cube.Pos{9, IslandY + 5, 9}
		leaves := block.Leaves{Type: block.OakLeaves(), ShouldUpdate: true}
		tx.SetBlock(alone, leaves, nil)
		tx.SetBlock(near, leaves, nil)
		tx.SetBlock(near.Add(cube.Pos{1, 0, 0}), oakLog, nil)
		leaves.ScheduledTick(alone, tx, nil)
		leaves.ScheduledTick(near, tx, nil)
		if _, ok := tx.Block(alone).(block.Leaves); ok {
			t.Error("leaves without a log did not decay")
		}
		if l, ok := tx.Block(near).(block.Leaves); !ok || l.ShouldUpdate {
			t.Errorf("leaves beside a log: %v", tx.Block(near))
		}
	})
}
