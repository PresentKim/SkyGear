// Package dfhook reaches into Dragonfly sessions without patching
// Dragonfly, the way bedrock-gophers/inv does: the server's listeners are
// wrapped so every packet a client sends or receives passes hooks first,
// and unexported session methods and fields are reached with go:linkname
// and reflection.
//
// It depends on Dragonfly internals by name (see TestInternals). After a
// Dragonfly upgrade the test points at what moved.
package dfhook

import (
	"reflect"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// Conn is a client connection whose packets pass the hooks.
type Conn struct {
	session.Conn

	sess      atomic.Pointer[session.Session]
	closeOnce sync.Once
	moveInput atomic.Pointer[mgl64.Vec2]
}

// Session returns the session of the connection, once Of found it.
func (c *Conn) Session() *session.Session { return c.sess.Load() }

// WritePacket queues pk on the session like Dragonfly's own packets. It
// must not be called from a server packet hook (the writer itself).
func (c *Conn) WritePacket(pk packet.Packet) error {
	if s := c.sess.Load(); s != nil {
		session_writePacket(s, pk)
		return nil
	}
	return c.write(pk)
}

func (c *Conn) write(pk packet.Packet) error {
	for _, h := range serverHooks {
		if h(c, pk) {
			return nil
		}
	}
	return c.Conn.WritePacket(pk)
}

// ReadPacket reads the next client packet that no hook dropped.
func (c *Conn) ReadPacket() (packet.Packet, error) {
read:
	for {
		pk, err := c.Conn.ReadPacket()
		if err != nil {
			return pk, err
		}
		for _, h := range clientHooks {
			if h(c, pk) {
				continue read
			}
		}
		return pk, nil
	}
}

// Close runs the close hooks once and closes the connection.
func (c *Conn) Close() error {
	c.closeOnce.Do(func() {
		for _, h := range closeHooks {
			h(c)
		}
	})
	return c.Conn.Close()
}

// conn is what the session holds: Dragonfly writes through WritePacket,
// which must reach the client, not the session queue again.
type conn struct{ *Conn }

func (c conn) WritePacket(pk packet.Packet) error { return c.Conn.write(pk) }

var (
	clientHooks []func(c *Conn, pk packet.Packet) (drop bool)
	serverHooks []func(c *Conn, pk packet.Packet) (drop bool)
	closeHooks  []func(c *Conn)
)

// OnClientPacket adds a hook for packets from clients, run on the
// connection's read goroutine before Dragonfly handles them. It returns
// true to drop the packet. Hooks are added at startup only.
func OnClientPacket(f func(c *Conn, pk packet.Packet) (drop bool)) {
	clientHooks = append(clientHooks, f)
}

// OnServerPacket adds a hook for packets to clients, run on the session's
// writer goroutine. It may change pk, or return true to drop it.
func OnServerPacket(f func(c *Conn, pk packet.Packet) (drop bool)) {
	serverHooks = append(serverHooks, f)
}

// OnClose adds a hook run once when a connection closes (disconnect).
func OnClose(f func(c *Conn)) { closeHooks = append(closeHooks, f) }

// Wrap makes the connections of conf's listeners pass the hooks. Call it
// before conf.New.
func Wrap(conf *server.Config) {
	for i, l := range conf.Listeners {
		conf.Listeners[i] = func(c server.Config) (server.Listener, error) {
			orig, err := l(c)
			if err != nil {
				return nil, err
			}
			return listener{orig}, nil
		}
	}
}

type listener struct{ server.Listener }

func (l listener) Accept() (session.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return conn{&Conn{Conn: c}}, nil
}

func (l listener) Disconnect(c session.Conn, reason string) error {
	if w, ok := c.(conn); ok {
		c = w.Conn.Conn
	}
	return l.Listener.Disconnect(c, reason)
}

// Session returns the session of p (session.Nop for players without a
// connection).
func Session(p *player.Player) *session.Session { return player_session(p) }

// Of returns the hooked connection of p, or nil if p has none (test
// players, unwrapped listeners).
func Of(p *player.Player) *Conn {
	s := player_session(p)
	if s == session.Nop || s == nil {
		return nil
	}
	w, ok := (*field[session.Conn](s, "conn")).(conn)
	if !ok {
		return nil
	}
	w.sess.Store(s)
	return w.Conn
}

// Attach makes the hooks of p's connection know its session. Call it for
// every player that joins; hooks that need the session (forms) skip
// connections without one.
func Attach(p *player.Player) { Of(p) }

// WritePacket queues pk for the client of s.
func WritePacket(s *session.Session, pk packet.Packet) {
	if s != session.Nop {
		session_writePacket(s, pk)
	}
}

// SendInventory sends the whole contents of inv as window windowID.
func SendInventory(s *session.Session, inv *inventory.Inventory, windowID byte) {
	session_sendInv(s, inv, uint32(windowID))
}

// ContainerOpen reports whether s has a container window open.
func ContainerOpen(s *session.Session) bool {
	return field[atomic.Bool](s, "containerOpened").Load()
}

// OpenWindow takes the next window ID of s and makes inv the open
// container window there, so Dragonfly moves items in and out of it
// (ContainerLevelEntity). The caller sends ContainerOpen. pos is where
// Dragonfly looks for the container block when it closes the window.
func OpenWindow(s *session.Session, inv *inventory.Inventory, pos cube.Pos, containerType byte) (windowID byte) {
	windowID = session_nextWindowID(s)
	field[atomic.Pointer[cube.Pos]](s, "openedPos").Store(&pos)
	field[atomic.Pointer[inventory.Inventory]](s, "openedWindow").Store(inv)
	field[atomic.Uint32](s, "openedContainerID").Store(uint32(containerType))
	field[atomic.Bool](s, "containerOpened").Store(true)
	return windowID
}

// CloseWindow closes the container window s has open, if any, as when the
// server closes it (the client is told). It must run in the player's
// transaction.
func CloseWindow(s *session.Session, tx *world.Tx) {
	if s != session.Nop {
		session_closeCurrentContainer(s, tx, false)
	}
}

// EditCustomItems replaces each entry of srv's custom items with what edit
// makes of it. Call it once, after the server is made and before players
// join: the client gets these entries in two item registries, and the first,
// sent by gophertunnel during StartGame, does not pass OnServerPacket hooks.
func EditCustomItems(srv *server.Server, edit func(protocol.ItemEntry) protocol.ItemEntry) {
	items := field[[]protocol.ItemEntry](srv, "customItems")
	for i, e := range *items {
		(*items)[i] = edit(e)
	}
}

// field returns a pointer to the unexported field name of *v.
func field[T any](v any, name string) *T {
	f := reflect.ValueOf(v).Elem().FieldByName(name)
	if !f.IsValid() || f.Type() != reflect.TypeFor[T]() {
		panic("dfhook: no field " + name + " of type " + reflect.TypeFor[T]().String())
	}
	return (*T)(unsafe.Pointer(f.UnsafeAddr()))
}

//go:linkname player_session github.com/df-mc/dragonfly/server/player.(*Player).session
func player_session(*player.Player) *session.Session

//go:linkname session_writePacket github.com/df-mc/dragonfly/server/session.(*Session).writePacket
func session_writePacket(*session.Session, packet.Packet)

//go:linkname session_sendInv github.com/df-mc/dragonfly/server/session.(*Session).sendInv
func session_sendInv(*session.Session, *inventory.Inventory, uint32)

//go:linkname session_nextWindowID github.com/df-mc/dragonfly/server/session.(*Session).nextWindowID
func session_nextWindowID(*session.Session) byte

//go:linkname session_closeCurrentContainer github.com/df-mc/dragonfly/server/session.(*Session).closeCurrentContainer
func session_closeCurrentContainer(*session.Session, *world.Tx, bool)
