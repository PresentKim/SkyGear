package chestui

// Windows backed by an invisible, client-only entity instead of a block,
// like PocketMine's InvMenu actor menus, reached through dfhook. Options
// enables InvMenu's extra open steps (Ride, WaitLatency, Retry until the
// client acks with a PacketViolationWarning). A client close of window 0xff
// (sent when a window opens while a form closes) never closes an entity window.

import (
	"bytes"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/presentkim/dragonfactory/common/dfhook"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// EntityIdentifier is the actor identifier of the invisible entity behind
// a window. It is announced in AvailableActorIdentifiers so clients accept
// it; no client entity definition is needed.
const EntityIdentifier = "dragonfly:entity_container"

const (
	// transitionWindow is how soon after a window closed a new one is
	// treated as a switch of screens.
	transitionWindow = time.Second
	// openRetries limits the ContainerOpen resends (one per tick).
	openRetries = 20
	// selfRuntimeID is the runtime ID Dragonfly gives a player's own entity.
	selfRuntimeID = 1
	// eyeHeight is the height of PlayerAuthInput positions above the feet.
	eyeHeight = 1.62
)

// Options switches the optional steps of opening windows. Set it once at
// startup.
// Ride makes the entity ride the player; WaitLatency opens the window after
// a NetworkStackLatency round trip following AddActor; Retry resends
// ContainerClose+ContainerOpen every tick (at most openRetries) until the
// client acks, ignoring client closes meanwhile.
var Options struct {
	Ride, WaitLatency, Retry bool
	// Debug logs the opening and closing packets at INFO level.
	Debug bool
}

func debug(msg string, args ...any) {
	if Options.Debug {
		slog.Info("chestui: "+msg, args...)
	}
}

// nextRuntimeID numbers the window entities far above the runtime IDs
// Dragonfly gives real entities.
var nextRuntimeID atomic.Uint64

type windowState int

const (
	waitingActor windowState = iota
	opening
	open
)

type window struct {
	cl        *client
	h         *world.EntityHandle
	runtimeID uint64
	inv       *inventory.Inventory
	pos       cube.Pos
	onClose   func(p *player.Player, tx *world.Tx)

	mu       sync.Mutex
	state    windowState
	windowID byte
	stop     chan struct{}
}

// client is the window state of one connection.
type client struct {
	c         *dfhook.Conn
	s         *session.Session
	lang      string
	win       atomic.Pointer[window]
	lastClose atomic.Int64
	token     atomic.Uint64
	lastPos   atomic.Pointer[mgl32.Vec3]
}

var clients sync.Map // *dfhook.Conn → *client

func clientOf(p *player.Player) *client {
	c := dfhook.Of(p)
	if c == nil {
		return nil
	}
	if cl := loadClient(c); cl != nil {
		return cl
	}
	cl, _ := clients.LoadOrStore(c, &client{c: c, s: c.Session(), lang: connLang(c)})
	return cl.(*client)
}

func loadClient(c *dfhook.Conn) *client {
	cl, ok := clients.Load(c)
	if !ok {
		return nil
	}
	return cl.(*client)
}

func init() {
	nextRuntimeID.Store(1 << 62)
	dfhook.OnClientPacket(clientPacket)
	dfhook.OnServerPacket(serverPacket)
	dfhook.OnClose(func(c *dfhook.Conn) {
		if cl := loadClient(c); cl != nil {
			clients.Delete(c)
			if w := cl.win.Load(); w != nil {
				// Disconnected: the player is gone.
				cl.finish(w, nil, nil)
			}
		}
	})
}

// openWindow opens inv for p in a window titled title. onClose, if not nil,
// is called once when the window closes for any reason; p and tx are nil
// if the player disconnected. Opening inv while it is already open or
// opening does nothing. It must run in p's transaction.
func openWindow(p *player.Player, inv *inventory.Inventory, title string, onClose func(p *player.Player, tx *world.Tx)) {
	cl := clientOf(p)
	if cl == nil {
		return
	}
	if w := cl.win.Load(); w != nil && w.inv == inv {
		return
	}
	Close(p)
	token := cl.token.Add(1)
	spawn := func(p *player.Player, _ *world.Tx) {
		if cl.token.Load() == token {
			cl.spawn(p, inv, title, onClose)
		}
	}
	// A window opened right after another one closed (switching screens)
	// is ignored by the client while it is still closing the old one. Wait
	// for a NetworkStackLatency round trip first, like InvMenu does.
	if since := time.Since(time.Unix(0, cl.lastClose.Load())); since < transitionWindow {
		debug("switching windows: waiting for the client", "since", since)
		dfhook.AfterClientSync(p, spawn)
		return
	}
	spawn(p, p.Tx())
}

// Close closes the container window p has open: a chest window, or a
// block container of Dragonfly. It must run in p's transaction.
func Close(p *player.Player) {
	s := dfhook.Session(p)
	if s == session.Nop {
		return
	}
	tx := p.Tx()
	cl := clientOf(p)
	if cl != nil {
		cl.token.Add(1) // cancel a window waiting to open
		if dfhook.ContainerOpen(s) {
			cl.lastClose.Store(time.Now().UnixNano())
		}
	}
	dfhook.CloseWindow(s, tx)
	if cl != nil {
		if w := cl.win.Load(); w != nil {
			cl.finish(w, p, tx)
		}
	}
}

// spawn spawns the entity of a window and opens it.
func (cl *client) spawn(p *player.Player, inv *inventory.Inventory, title string, onClose func(*player.Player, *world.Tx)) {
	id := nextRuntimeID.Add(1)
	m := protocol.NewEntityMetadata()
	m.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagInvisible)
	m.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagNoAI)
	m[protocol.EntityDataKeyName] = title
	m[protocol.EntityDataKeyContainerType] = byte(0xff) // ContainerTypeInventory (-1)
	m[protocol.EntityDataKeyContainerSize] = int32(inv.Size())
	var links []protocol.EntityLink
	if Options.Ride {
		links = []protocol.EntityLink{{RiddenEntityUniqueID: selfRuntimeID, RiderEntityUniqueID: int64(id), Type: protocol.EntityLinkPassenger, Immediate: true}}
	}
	pos := p.Position()
	pos32 := mgl32.Vec3{float32(pos[0]), float32(pos[1]), float32(pos[2])}
	cl.lastPos.Store(&pos32)
	dfhook.WritePacket(cl.s, &packet.AddActor{
		EntityUniqueID:  int64(id),
		EntityRuntimeID: id,
		EntityType:      EntityIdentifier,
		EntityMetadata:  m,
		Position:        pos32,
		EntityLinks:     links,
	})

	w := &window{cl: cl, h: p.H(), runtimeID: id, inv: inv, pos: cube.PosFromVec3(pos), onClose: onClose, stop: make(chan struct{})}
	cl.win.Store(w)
	debug("spawned", "actor", id, "size", inv.Size(), "ride", Options.Ride)
	if !Options.WaitLatency {
		w.open()
		return
	}
	dfhook.AfterClientSync(p, func(*player.Player, *world.Tx) { w.open() })
}

// open sends the ContainerOpen of a waiting window and starts resending it
// until the client acknowledges it. It runs in the player's transaction.
func (w *window) open() {
	w.mu.Lock()
	if w.state != waitingActor || w.cl.win.Load() != w {
		w.mu.Unlock()
		return
	}
	w.state = opening
	w.windowID = dfhook.OpenWindow(w.cl.s, w.inv, w.pos, protocol.ContainerTypeContainer)
	w.mu.Unlock()

	dfhook.WritePacket(w.cl.s, w.openPacket())
	dfhook.SendInventory(w.cl.s, w.inv, w.windowID)
	debug("opened", "actor", w.runtimeID, "window", w.windowID)
	if !Options.Retry {
		w.ack()
		return
	}
	go w.retry()
}

func (w *window) openPacket() *packet.ContainerOpen {
	return &packet.ContainerOpen{
		WindowID:                w.windowID,
		ContainerType:           protocol.ContainerTypeContainer,
		ContainerPosition:       protocol.BlockPos{int32(w.pos[0]), int32(w.pos[1]), int32(w.pos[2])},
		ContainerEntityUniqueID: int64(w.runtimeID),
	}
}

// retry resends the window every tick until it is acknowledged.
func (w *window) retry() {
	t := time.NewTicker(time.Second / 20)
	defer t.Stop()
	for range openRetries {
		select {
		case <-w.stop:
			return
		case <-t.C:
		}
		if !w.opening() {
			return
		}
		dfhook.WritePacket(w.cl.s, &packet.ContainerClose{WindowID: w.windowID, ContainerType: protocol.ContainerTypeContainer})
		dfhook.WritePacket(w.cl.s, w.openPacket())
	}
	// Never acknowledged: keep the window as it is.
	debug("never acknowledged", "actor", w.runtimeID)
	w.ack()
}

// ack marks the window open and, after retries, sends its contents again.
func (w *window) ack() {
	w.mu.Lock()
	if w.state != opening {
		w.mu.Unlock()
		return
	}
	w.state = open
	w.mu.Unlock()
	if w.cl.win.Load() == w && Options.Retry {
		dfhook.SendInventory(w.cl.s, w.inv, w.windowID)
	}
}

// current returns the state of the window.
func (w *window) current() windowState {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.state
}

// opening reports whether the window is still being opened.
func (w *window) opening() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.state == opening && w.cl.win.Load() == w
}

// finish removes the entity of w and runs its close callback, once.
func (cl *client) finish(w *window, p *player.Player, tx *world.Tx) {
	if !cl.win.CompareAndSwap(w, nil) {
		return
	}
	close(w.stop)
	debug("closed", "actor", w.runtimeID)
	if p != nil {
		dfhook.WritePacket(cl.s, &packet.RemoveActor{EntityUniqueID: int64(w.runtimeID)})
	}
	if w.onClose != nil {
		w.onClose(p, tx)
	}
}

// clientPacket handles packets from the client that concern its window.
func clientPacket(c *dfhook.Conn, pk packet.Packet) bool {
	cl := loadClient(c)
	if cl == nil {
		return false
	}
	switch pk := pk.(type) {
	case *packet.ContainerClose:
		w := cl.win.Load()
		if w == nil {
			if pk.WindowID != 0 {
				cl.lastClose.Store(time.Now().UnixNano())
			}
			return false
		}
		debug("client close", "window", pk.WindowID, "type", pk.ContainerType, "opening", w.opening())
		if pk.WindowID == 0xff {
			return true
		}
		if pk.WindowID != 0 && w.opening() {
			c.WritePacket(&packet.ContainerClose{WindowID: pk.WindowID, ContainerType: pk.ContainerType})
			return true
		}
		if pk.WindowID != 0 {
			cl.lastClose.Store(time.Now().UnixNano())
		}
	case *packet.PacketViolationWarning:
		debug("violation warning", "type", pk.Type, "severity", pk.Severity, "packet", pk.PacketID, "context", pk.ViolationContext)
		if pk.Type == -1 && pk.Severity == packet.ViolationSeverityWarning {
			if w := cl.win.Load(); w != nil {
				w.ack()
			}
		}
	case *packet.PlayerAuthInput:
		cl.follow(pk.Position.Sub(mgl32.Vec3{0, eyeHeight, 0}))
	}
	return false
}

// follow keeps the entity of an open window at the player, so the client
// does not close the window when the player moves away from it (falling,
// pushed). Not needed when it rides.
func (cl *client) follow(pos mgl32.Vec3) {
	w := cl.win.Load()
	if w == nil || Options.Ride {
		return
	}
	if last := cl.lastPos.Load(); last != nil && last.ApproxEqual(pos) {
		return
	}
	cl.lastPos.Store(&pos)
	cl.c.WritePacket(&packet.MoveActorAbsolute{EntityRuntimeID: w.runtimeID, Position: pos, Flags: packet.MoveFlagTeleport})
}

// serverPacket announces the window entity, puts the client's texts into
// its window's contents (Localize) and notices windows that Dragonfly closed
// itself (the client closed one, or another container opened).
func serverPacket(c *dfhook.Conn, pk packet.Packet) bool {
	localizeWindow(c, pk)
	switch pk := pk.(type) {
	case *packet.AvailableActorIdentifiers:
		pk.SerialisedEntityIdentifiers = addIdentifier(pk.SerialisedEntityIdentifiers)
	case *packet.ContainerClose:
		cl := loadClient(c)
		if cl == nil || pk.WindowID == 0 {
			return false
		}
		cl.lastClose.Store(time.Now().UnixNano())
		w := cl.win.Load()
		if w == nil || w.current() != open || pk.WindowID != w.windowID {
			return false
		}
		// Not on this goroutine: it writes the packets the callback sends.
		w.h.Do(func(tx *world.Tx, e world.Entity) {
			cl.finish(w, e.(*player.Player), tx)
		})
	}
	return false
}

var identifiers struct {
	sync.Mutex
	in, out []byte
}

// addIdentifier adds EntityIdentifier to serialised actor identifiers.
func addIdentifier(data []byte) []byte {
	identifiers.Lock()
	defer identifiers.Unlock()
	if identifiers.out != nil && bytes.Equal(identifiers.in, data) {
		return identifiers.out
	}
	out, ok := encodeWithIdentifier(data)
	if !ok {
		return data
	}
	identifiers.in, identifiers.out = bytes.Clone(data), out
	return out
}

func encodeWithIdentifier(data []byte) ([]byte, bool) {
	var ids struct {
		IDList []map[string]any `nbt:"idlist"`
	}
	if err := nbt.NewDecoder(bytes.NewReader(data)).Decode(&ids); err != nil {
		return nil, false
	}
	ids.IDList = append(ids.IDList, map[string]any{"id": EntityIdentifier})
	out, err := nbt.Marshal(ids)
	if err != nil {
		return nil, false
	}
	return out, true
}

// viewer shows slot changes of an inventory to a client if that inventory
// is its open window. It may be used outside the player's transaction.
type viewer struct{ cl *client }

func (v viewer) viewSlotChange(inv *inventory.Inventory, slot int, it item.Stack) {
	if v.cl == nil {
		return
	}
	if w := v.cl.win.Load(); w != nil && w.inv == inv {
		if w.current() != waitingActor {
			v.cl.s.ViewSlotChange(slot, it)
		}
	}
}
