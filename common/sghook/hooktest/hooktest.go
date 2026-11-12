package hooktest

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/google/uuid"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const timeout = 5 * time.Second

// Client is the fake client of a server started by Join.
type Client struct {
	t      testing.TB
	srv    *server.Server
	handle *world.EntityHandle
	conn   *conn
}

// Join starts a server from conf with one fake listener and joins one fake
// client, "tap" (language ko_KR, chunk radius 4, no client cache). conf gets
// a discard Log, DisableResourceBuilding and a NopGenerator unless set, and
// its Listeners are replaced. setup runs between New and Listen. Join goes
// through sghook.Wrap and sghook.Attach and returns once the player spawned
// and the item registry, crafting data and creative content are captured.
// The server is closed when the test ends.
func Join(t testing.TB, conf server.Config, setup func(*server.Server)) *Client {
	t.Helper()
	if conf.Log == nil {
		conf.Log = slog.New(slog.DiscardHandler)
	}
	conf.DisableResourceBuilding = true
	if conf.Generator == nil {
		conf.Generator = func(world.Dimension) world.Generator { return world.NopGenerator{} }
	}
	c := newConn()
	conf.Listeners = []func(server.Config) (server.Listener, error){
		func(server.Config) (server.Listener, error) { return &listener{c: c, done: make(chan struct{})}, nil },
	}
	sghook.Wrap(&conf)
	srv := conf.New()
	if setup != nil {
		setup(srv)
	}
	srv.Listen()
	t.Cleanup(func() { _ = srv.Close() })

	spawned := make(chan *world.EntityHandle, 1)
	go func() {
		for p := range srv.Accept() {
			sghook.Attach(p)
			select {
			case spawned <- p.H():
			default:
			}
		}
	}()
	cl := &Client{t: t, srv: srv, conn: c}
	select {
	case cl.handle = <-spawned:
	case <-time.After(timeout):
		t.Fatalf("hooktest: client did not spawn within %v", timeout)
	}
	for _, m := range []struct {
		name  string
		match func(packet.Packet) bool
	}{
		{"ItemRegistry", func(p packet.Packet) bool { _, ok := p.(*packet.ItemRegistry); return ok }},
		{"CraftingData", func(p packet.Packet) bool { _, ok := p.(*packet.CraftingData); return ok }},
		{"CreativeContent", func(p packet.Packet) bool { _, ok := p.(*packet.CreativeContent); return ok }},
	} {
		if cl.find(m.match, false) == nil {
			t.Fatalf("hooktest: no %s packet within %v", m.name, timeout)
		}
	}
	return cl
}

// Server returns the running server.
func (c *Client) Server() *server.Server { return c.srv }

// Handle returns the entity handle of the joined player.
func (c *Client) Handle() *world.EntityHandle { return c.handle }

// GameData returns the start game data the server sent.
func (c *Client) GameData() minecraft.GameData {
	c.conn.mu.Lock()
	defer c.conn.mu.Unlock()
	return c.conn.data
}

// Packets returns a copy of every packet written to the client, in write
// order (after the server hooks).
func (c *Client) Packets() []packet.Packet {
	c.conn.mu.Lock()
	defer c.conn.mu.Unlock()
	return append([]packet.Packet(nil), c.conn.pks...)
}

// Wait returns the first packet matching match that no earlier Wait
// returned. It fails the test if there is none within 5 s.
func (c *Client) Wait(match func(packet.Packet) bool) packet.Packet {
	c.t.Helper()
	pk := c.find(match, true)
	if pk == nil {
		c.t.Fatalf("hooktest: no matching packet within %v", timeout)
	}
	return pk
}

// find polls for a matching packet; take marks it as returned to Wait.
func (c *Client) find(match func(packet.Packet) bool, take bool) packet.Packet {
	deadline := time.Now().Add(timeout)
	for {
		c.conn.mu.Lock()
		for i, pk := range c.conn.pks {
			if !c.conn.returned[i] && match(pk) {
				if take {
					c.conn.returned[i] = true
				}
				c.conn.mu.Unlock()
				return pk
			}
		}
		c.conn.mu.Unlock()
		if time.Now().After(deadline) {
			return nil
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// conn is the fake session.Conn.
type conn struct {
	id   string
	mu   sync.Mutex
	data minecraft.GameData
	pks  []packet.Packet

	returned map[int]bool // packets Wait handed out

	closeOnce sync.Once
	closed    chan struct{}
}

func newConn() *conn {
	return &conn{id: uuid.NewString(), closed: make(chan struct{}), returned: map[int]bool{}}
}

func (c *conn) IdentityData() login.IdentityData {
	return login.IdentityData{Identity: c.id, DisplayName: "tap", XUID: "2535400000000001"}
}
func (c *conn) ClientData() login.ClientData { return login.ClientData{LanguageCode: "ko_KR"} }
func (c *conn) ClientCacheEnabled() bool     { return false }
func (c *conn) ChunkRadius() int             { return 4 }
func (c *conn) Latency() time.Duration       { return 0 }
func (c *conn) Flush() error                 { return nil }
func (c *conn) RemoteAddr() net.Addr         { return &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 19132} }

func (c *conn) ReadPacket() (packet.Packet, error) {
	<-c.closed
	return nil, io.EOF
}

func (c *conn) WritePacket(pk packet.Packet) error {
	select {
	case <-c.closed:
		return net.ErrClosed
	default:
	}
	c.mu.Lock()
	c.pks = append(c.pks, pk)
	c.mu.Unlock()
	return nil
}

func (c *conn) StartGameContext(_ context.Context, data minecraft.GameData) error {
	c.mu.Lock()
	c.data = data
	c.mu.Unlock()
	return nil
}

func (c *conn) Close() error {
	c.closeOnce.Do(func() { close(c.closed) })
	return nil
}

// listener hands out the one fake conn, then blocks until closed.
type listener struct {
	c    *conn
	once sync.Once
	done chan struct{}
	cl   sync.Once
}

func (l *listener) Accept() (session.Conn, error) {
	first := false
	l.once.Do(func() { first = true })
	if first {
		return l.c, nil
	}
	<-l.done
	return nil, errors.New("listener closed")
}

func (l *listener) Disconnect(c session.Conn, reason string) error { return c.Close() }

func (l *listener) Close() error {
	l.cl.Do(func() { close(l.done) })
	return nil
}
