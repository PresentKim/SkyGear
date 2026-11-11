package sghook_test

import (
	"testing"

	"github.com/df-mc/dragonfly/server"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/presentkim/skygear/common/sghook/hooktest"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

// TestEditCustomBlocksJoin: a client that joins after the edit, the first
// one included, gets the edited entries in its start game data; the entries
// the edit leaves alone are byte-identical to a join without it.
func TestEditCustomBlocksJoin(t *testing.T) {
	base := hooktest.Join(t, server.Config{}, nil).GameData().CustomBlocks
	if len(base) < 2 {
		t.Fatalf("%d block entries to test with", len(base))
	}
	want := base[0].Name
	var edits int
	edited := hooktest.Join(t, server.Config{}, func(srv *server.Server) {
		sghook.EditCustomBlocks(srv, func(e protocol.BlockEntry) protocol.BlockEntry {
			edits++
			if e.Name == want {
				e.Properties["sg_edit"] = "yes"
			}
			return e
		})
	})
	t.Cleanup(func() { sghook.EditCustomBlocks(nil, nil) })
	got := edited.GameData().CustomBlocks
	if len(got) != len(base) || edits != len(base) {
		t.Fatalf("%d entries (%d edits), want %d", len(got), edits, len(base))
	}
	for i := range got {
		if got[i].Name != base[i].Name {
			t.Fatalf("entry %d is %q, want %q", i, got[i].Name, base[i].Name)
		}
		if i == 0 {
			if got[i].Properties["sg_edit"] != "yes" {
				t.Fatal("edit missing in the first client's entry")
			}
			delete(got[i].Properties, "sg_edit")
		}
		if hooktest.Canon(got[i]) != hooktest.Canon(base[i]) {
			t.Fatalf("entry %q changed beyond the edit", got[i].Name)
		}
	}
	// The edit did not leak into Dragonfly's shared data.
	again := hooktest.Join(t, server.Config{}, nil).GameData().CustomBlocks
	if _, ok := again[0].Properties["sg_edit"]; ok {
		t.Fatal("edit leaked into a later server")
	}
}
