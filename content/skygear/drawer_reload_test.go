package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/storage"
)

func countHolders(s *session) (n int) {
	s.World.Do(func(tx *world.Tx) {
		for e := range tx.Entities() {
			if e.H().Type() == storage.HolderType {
				n++
			}
		}
	})
	return
}

// TestDrawerReloadKeepsContentsOneHolder: a drawer and its transient holder
// display over a save and reload keep the contents, get the holder re-created
// and never duplicate it.
func TestDrawerReloadKeepsContentsOneHolder(t *testing.T) {
	dir := t.TempDir()
	pos := cube.Pos{3, island.IslandY + 1, 3}

	s, closeAll := openPersisted(t, dir)
	s.World.Do(func(tx *world.Tx) {
		tx.SetBlock(pos, storage.Drawer{Facing: cube.North, Item: item.NewStack(item.Wheat{}, 1), Count: 77}, nil)
	})
	tick(s, 60)
	if n := countHolders(s); n != 1 {
		t.Fatalf("holders before reload = %d, want 1", n)
	}
	closeAll()

	s, _ = openPersisted(t, dir)
	tick(s, 60)
	s.World.Do(func(tx *world.Tx) {
		d, ok := tx.Block(pos).(storage.Drawer)
		if !ok || d.Count != 77 || d.Item.Empty() {
			t.Fatalf("drawer after reload = %+v", d)
		}
	})
	if n := countHolders(s); n != 1 {
		t.Fatalf("holders after reload = %d, want 1", n)
	}
}
