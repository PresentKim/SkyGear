package bridge

import (
	"sync"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// The preview shows the block in its cell to the player only, so the
// client takes it for a real block: a click on it arrives as a use on a
// block that is air on the server, which Dragonfly drops before any
// handler. The packet hook turns such a click into a use on the air, which
// the assist handles (Place), and keeps the trigger of each use (a press of
// the button or a repeat of a held one) for Place to tell them apart.

// conn is what the packet hook knows of one connection. The hook runs on
// the connection's read goroutine, the game on the world's, hence the lock.
type conn struct {
	mu sync.Mutex
	// cell is the preview's cell, if has.
	cell cube.Pos
	has  bool
	// trigger is the TriggerType of the last use the client sent.
	trigger uint32
	// ghost is where the client put the block it predicted for a click on
	// the preview, if hasGhost: it is not on the server.
	ghost    cube.Pos
	hasGhost bool
}

var conns sync.Map // *session.Session -> *conn

func connOf(s *session.Session) *conn {
	c, _ := conns.LoadOrStore(s, &conn{})
	return c.(*conn)
}

func (c *conn) setPreview(cell cube.Pos, has bool) {
	c.mu.Lock()
	c.cell, c.has = cell, has
	c.mu.Unlock()
}

// lastTrigger is the TriggerType of the use being handled.
func (c *conn) lastTrigger() uint32 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.trigger
}

// takeGhost returns the position of the client's predicted block, if any,
// and forgets it.
func (c *conn) takeGhost() (cube.Pos, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	g, ok := c.ghost, c.hasGhost
	c.hasGhost = false
	return g, ok
}

// see notes the use d the client sent: its trigger, and a click on the
// preview's cell becomes a use on the air.
func (c *conn) see(d *protocol.UseItemTransactionData) {
	if d.ActionType != protocol.UseItemActionClickBlock && d.ActionType != protocol.UseItemActionClickAir {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.trigger = d.TriggerType
	pos := cube.Pos{int(d.BlockPosition[0]), int(d.BlockPosition[1]), int(d.BlockPosition[2])}
	if d.ActionType == protocol.UseItemActionClickBlock && c.has && pos == c.cell {
		d.ActionType = protocol.UseItemActionClickAir
		c.ghost, c.hasGhost = pos.Side(cube.Face(d.BlockFace)), true
	}
}

func init() {
	sghook.OnClientPacket(func(sc *sghook.Conn, pk packet.Packet) bool {
		it, ok := pk.(*packet.InventoryTransaction)
		if !ok {
			return false
		}
		d, ok := it.TransactionData.(*protocol.UseItemTransactionData)
		if !ok {
			return false
		}
		if s := sc.Session(); s != nil {
			connOf(s).see(d)
		}
		return false
	})
	sghook.OnClose(func(sc *sghook.Conn) {
		if s := sc.Session(); s != nil {
			conns.Delete(s)
		}
	})
}
