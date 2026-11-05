package skygear

import (
	"log/slog"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/netstore"
)

// TestSaveFlushesDisks: disk contents inserted since the last controller
// flush are in the save written by Autosave and by Close.
func TestSaveFlushesDisks(t *testing.T) {
	for _, closing := range []bool{false, true} {
		var doc []byte
		c := core.New(core.Config{Log: slog.Default(), Store: func(b []byte) error { doc = b; return nil }})
		s := &session{Session: c, g: &Game{log: slog.Default()}}
		id := netstore.Disks(c).NewID()
		k := netstore.KeyOf(item.NewStack(block.Stone{}, 1))
		if netstore.Disks(c).Open(id, 1024).Insert(k, 7, false) != 7 {
			t.Fatal("insert")
		}
		if closing {
			s.Close(nil)
		} else {
			s.Autosave(nil)
			c.Close() // waits for the writer
		}
		c2 := core.New(core.Config{Log: slog.Default(), State: doc})
		if got := netstore.Disks(c2).Open(id, 1024).Contents()[k]; got != 7 {
			t.Errorf("closing=%v: reloaded %d, want 7", closing, got)
		}
	}
}
