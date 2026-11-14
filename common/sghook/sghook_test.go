package sghook

import (
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// TestInternals checks the Dragonfly internals the package reaches by
// name. The linknamed functions fail at link time if they are gone.
func TestInternals(t *testing.T) {
	s := &session.Session{}
	field[session.Conn](s, "conn")
	field[atomic.Bool](s, "containerOpened")
	field[atomic.Pointer[cube.Pos]](s, "openedPos")
	field[atomic.Pointer[inventory.Inventory]](s, "openedWindow")
	field[atomic.Uint32](s, "openedContainerID")
	field[sync.RWMutex](s, "entityMutex")
	field[map[uuid.UUID]struct{}](s, "hiddenEntities")
	field[world.BlockRegistry](s, "br")
	field[int32](s, "chunkRadius")
	field[[]protocol.ItemEntry](&server.Server{}, "customItems")
	if sf, ok := reflect.TypeFor[server.Server]().FieldByName("customBlocks"); !ok || sf.Type != reflect.TypeFor[[]protocol.BlockEntry]() {
		t.Fatal("server.Server has no field customBlocks []protocol.BlockEntry: Dragonfly moved the custom block entries, update EditCustomBlocks")
	}
	field[[]protocol.BlockEntry](&server.Server{}, "customBlocks")
	_ = []any{session_craftingData, session_handleRuntimeID, player_session, session_writePacket, session_sendInv, session_nextWindowID, session_closeCurrentContainer, tx_chunk}
}

func TestFieldCached(t *testing.T) {
	a, b := &session.Session{}, &session.Session{}
	for range 2 {
		if p := field[int32](a, "chunkRadius"); p != (*int32)(reflect.ValueOf(a).Elem().FieldByName("chunkRadius").Addr().UnsafePointer()) {
			t.Fatal("field of a points elsewhere")
		}
		if field[int32](b, "chunkRadius") == field[int32](a, "chunkRadius") {
			t.Fatal("field of b points into a")
		}
	}
	for range 2 {
		func() {
			defer func() {
				if recover() == nil {
					t.Fatal("no panic for a wrong type")
				}
			}()
			field[int64](a, "chunkRadius")
		}()
		func() {
			defer func() {
				if recover() == nil {
					t.Fatal("no panic for a missing field")
				}
			}()
			field[int32](a, "noSuchField")
		}()
	}
}

// fakeConn reads packets from a list and records the written ones.
type fakeConn struct {
	session.Conn
	in     []packet.Packet
	out    []packet.Packet
	closed int
}

func (f *fakeConn) ReadPacket() (packet.Packet, error) {
	pk := f.in[0]
	f.in = f.in[1:]
	return pk, nil
}

func (f *fakeConn) WritePacket(pk packet.Packet) error {
	f.out = append(f.out, pk)
	return nil
}

func (f *fakeConn) Close() error {
	f.closed++
	return nil
}

func TestHooks(t *testing.T) {
	defer func(c, s []func(*Conn, packet.Packet) bool, cl []func(*Conn)) {
		clientHooks, serverHooks, closeHooks = c, s, cl
	}(clientHooks, serverHooks, closeHooks)
	clientHooks, serverHooks, closeHooks = nil, nil, nil

	OnClientPacket(func(_ *Conn, pk packet.Packet) bool { _, ok := pk.(*packet.Animate); return ok })
	OnServerPacket(func(_ *Conn, pk packet.Packet) bool {
		if t, ok := pk.(*packet.Text); ok {
			t.Message = "changed"
		}
		_, ok := pk.(*packet.Animate)
		return ok
	})
	closes := 0
	OnClose(func(*Conn) { closes++ })

	f := &fakeConn{in: []packet.Packet{&packet.Animate{}, &packet.Text{}}}
	c := conn{&Conn{Conn: f}}
	if pk, _ := c.ReadPacket(); pk.ID() != packet.IDText {
		t.Fatalf("read %T, want the dropped Animate skipped", pk)
	}
	_ = c.WritePacket(&packet.Animate{})
	_ = c.WritePacket(&packet.Text{Message: "x"})
	if len(f.out) != 1 || f.out[0].(*packet.Text).Message != "changed" {
		t.Fatalf("written %v", f.out)
	}
	_ = c.Close()
	_ = c.Close()
	if closes != 1 || f.closed != 2 {
		t.Fatalf("close hooks %d, closes %d", closes, f.closed)
	}
}

func TestCameraPresetsOnSpawn(t *testing.T) {
	f := &fakeConn{}
	c := conn{&Conn{Conn: f}}
	_ = c.WritePacket(&packet.AvailableActorIdentifiers{})
	ids := map[uint32]bool{}
	for _, pk := range f.out {
		ids[pk.ID()] = true
	}
	if !ids[packet.IDCameraPresets] || !ids[packet.IDAvailableActorIdentifiers] || f.out[len(f.out)-1].ID() != packet.IDAvailableActorIdentifiers {
		t.Fatalf("written %T", f.out)
	}
}

func TestVoxelShapesOnSpawn(t *testing.T) {
	f := &fakeConn{}
	_ = conn{&Conn{Conn: f}}.WritePacket(&packet.AvailableActorIdentifiers{})
	ids := map[uint32]bool{}
	for _, pk := range f.out {
		ids[pk.ID()] = true
	}
	if !ids[packet.IDVoxelShapes] || !ids[packet.IDAvailableActorIdentifiers] {
		t.Fatalf("written %T", f.out)
	}
}

// TestEditCustomItems: the edit waits for the first accepted connection (the
// server builds its entries when it starts listening) and is made once.
func TestEditCustomItems(t *testing.T) {
	srv := &server.Server{}
	edits := 0
	EditCustomItems(srv, func(e protocol.ItemEntry) protocol.ItemEntry {
		edits++
		e.Name += "!"
		return e
	})
	t.Cleanup(func() { EditCustomItems(nil, nil) })
	items := field[[]protocol.ItemEntry](srv, "customItems")
	*items = []protocol.ItemEntry{{Name: "a"}} // as Listen builds them
	if (*items)[0].Name != "a" {
		t.Fatal("edited before a connection")
	}
	ApplyEntryEdits()
	ApplyEntryEdits()
	if (*items)[0].Name != "a!" || edits != 1 {
		t.Fatalf("entry %q after %d edits", (*items)[0].Name, edits)
	}
}

func TestEditCustomBlocks(t *testing.T) {
	srv := &server.Server{}
	edits := 0
	EditCustomBlocks(srv, func(e protocol.BlockEntry) protocol.BlockEntry {
		edits++
		e.Properties["x"] = int32(1)
		return e
	})
	t.Cleanup(func() { EditCustomBlocks(nil, nil) })
	blocks := field[[]protocol.BlockEntry](srv, "customBlocks")
	shared := map[string]any{"a": map[string]any{}}
	*blocks = []protocol.BlockEntry{{Name: "a", Properties: shared}, {Name: "b", Properties: map[string]any{}}}
	if edits != 0 || (*blocks)[0].Properties["x"] != nil {
		t.Fatal("edited before a connection")
	}
	ApplyEntryEdits()
	ApplyEntryEdits()
	if edits != 2 || (*blocks)[0].Properties["x"] != int32(1) || (*blocks)[1].Name != "b" || len(*blocks) != 2 {
		t.Fatalf("entries %v after %d edits", *blocks, edits)
	}
	if _, ok := shared["x"]; ok {
		t.Fatal("edit changed the maps Dragonfly shares")
	}
}

func TestEditCustomBlocksGuards(t *testing.T) {
	t.Cleanup(func() { EditCustomBlocks(nil, nil) })
	for name, edit := range map[string]func(protocol.BlockEntry) protocol.BlockEntry{
		"name": func(e protocol.BlockEntry) protocol.BlockEntry { e.Name += "!"; return e },
		"block_id": func(e protocol.BlockEntry) protocol.BlockEntry {
			e.Properties["vanilla_block_data"].(map[string]any)["block_id"] = int32(2)
			return e
		},
		"vanilla_block_data": func(e protocol.BlockEntry) protocol.BlockEntry { delete(e.Properties, "vanilla_block_data"); return e },
	} {
		srv := &server.Server{}
		EditCustomBlocks(srv, edit)
		blocks := field[[]protocol.BlockEntry](srv, "customBlocks")
		*blocks = []protocol.BlockEntry{{Name: "a", Properties: map[string]any{"vanilla_block_data": map[string]any{"block_id": int32(1)}}}}
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s change not refused", name)
				}
			}()
			ApplyEntryEdits()
		}()
		if (*blocks)[0].Name != "a" || vanillaBlockID((*blocks)[0].Properties) != int32(1) {
			t.Errorf("%s: entries changed despite the panic", name)
		}
	}
}

// TestEditEntriesConcurrentAccept: two listeners accepting at once make each
// edit once, and the loser of the race returns only after the winner is done
// (the server reads the entries right after Accept returns).
func TestEditEntriesConcurrentAccept(t *testing.T) {
	srv := &server.Server{}
	var itemEdits, blockEdits atomic.Int32
	EditCustomItems(srv, func(e protocol.ItemEntry) protocol.ItemEntry { itemEdits.Add(1); e.Name += "!"; return e })
	EditCustomBlocks(srv, func(e protocol.BlockEntry) protocol.BlockEntry {
		blockEdits.Add(1)
		e.Properties["x"] = int32(1)
		return e
	})
	t.Cleanup(func() { EditCustomItems(nil, nil); EditCustomBlocks(nil, nil) })
	items := field[[]protocol.ItemEntry](srv, "customItems")
	blocks := field[[]protocol.BlockEntry](srv, "customBlocks")
	*items = []protocol.ItemEntry{{Name: "a"}, {Name: "b"}}
	*blocks = []protocol.BlockEntry{{Name: "a", Properties: map[string]any{}}, {Name: "b", Properties: map[string]any{}}}
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ApplyEntryEdits()
			if (*items)[1].Name != "b!" || (*blocks)[1].Properties["x"] != int32(1) {
				t.Error("returned before the entries were edited")
			}
		}()
	}
	wg.Wait()
	if itemEdits.Load() != 2 || blockEdits.Load() != 2 {
		t.Fatalf("%d item and %d block edits, want 2 each", itemEdits.Load(), blockEdits.Load())
	}
}

// TestApplyEntryEditsOnce: the exported call makes both edits once, whoever
// calls it first.
func TestApplyEntryEditsOnce(t *testing.T) {
	srv := &server.Server{}
	items, blocks := 0, 0
	EditCustomItems(srv, func(e protocol.ItemEntry) protocol.ItemEntry { items++; return e })
	EditCustomBlocks(srv, func(e protocol.BlockEntry) protocol.BlockEntry { blocks++; return e })
	t.Cleanup(func() { EditCustomItems(nil, nil); EditCustomBlocks(nil, nil) })
	*field[[]protocol.ItemEntry](srv, "customItems") = []protocol.ItemEntry{{Name: "a"}, {Name: "b"}}
	*field[[]protocol.BlockEntry](srv, "customBlocks") = []protocol.BlockEntry{{Name: "c", Properties: map[string]any{}}}
	ApplyEntryEdits()
	ApplyEntryEdits()
	if items != 2 || blocks != 1 {
		t.Fatalf("%d item edits, %d block edits", items, blocks)
	}
}
