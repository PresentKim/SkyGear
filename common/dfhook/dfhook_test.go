package dfhook

import (
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player/form"
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
	if !reflect.ValueOf(s).Elem().FieldByName("handlers").IsValid() {
		t.Fatal("no field handlers")
	}
	fh := &session.ModalFormResponseHandler{}
	field[sync.Mutex](fh, "mu")
	field[map[uint32]form.Form](fh, "forms")
	_ = []any{player_session, session_writePacket, session_sendInv, session_nextWindowID, session_closeCurrentContainer}
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
	if len(f.out) != 2 || f.out[0].ID() != packet.IDCameraPresets || f.out[1].ID() != packet.IDAvailableActorIdentifiers {
		t.Fatalf("written %T", f.out)
	}
}

func TestMoveInput(t *testing.T) {
	f := &fakeConn{in: []packet.Packet{&packet.PlayerAuthInput{RawMoveVector: [2]float32{1, -1}}}}
	c := &Conn{Conn: f}
	_, _ = c.ReadPacket()
	if v := c.moveInput.Load(); v == nil || v[0] != 1 || v[1] != -1 {
		t.Fatalf("move input %v", v)
	}
}

func TestVoxelShapesOnSpawn(t *testing.T) {
	defer func(v bool) { SendBasicVoxelShapes = v }(SendBasicVoxelShapes)
	SendBasicVoxelShapes = true
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

func TestFormResponse(t *testing.T) {
	r := &packet.ModalFormResponse{FormID: 1, ResponseData: protocol.Option([]byte(" null\n"))}
	f := &fakeConn{in: []packet.Packet{r}}
	if pk, _ := (&Conn{Conn: f}).ReadPacket(); pk != r {
		t.Fatal("dropped without a session")
	}
	if _, ok := r.ResponseData.Value(); ok {
		t.Fatal("null response not turned into a cancel")
	}
}
