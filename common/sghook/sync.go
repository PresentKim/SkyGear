package sghook

import (
	"math/rand/v2"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// syncTimeout is how long AfterClientSync waits for the client's answer.
const syncTimeout = time.Second

var (
	syncMu sync.Mutex
	syncs  = map[*Conn]map[int64]func(){}
)

func init() {
	OnClientPacket(func(c *Conn, pk packet.Packet) bool {
		if l, ok := pk.(*packet.NetworkStackLatency); ok {
			// The client answers with the timestamp multiplied by 1, 1000
			// or 1000000 depending on the platform.
			for _, ts := range []int64{l.Timestamp, l.Timestamp / 1000, l.Timestamp / 1000000} {
				if f := takeSync(c, ts); f != nil {
					f()
					break
				}
			}
		}
		return false
	})
	OnClose(func(c *Conn) {
		syncMu.Lock()
		delete(syncs, c)
		syncMu.Unlock()
	})
}

// AfterClientSync runs f in p's transaction once the client has processed
// every packet sent to it before this call (a NetworkStackLatency round
// trip), or after a second if it does not answer. Players without a hooked
// connection never run f.
func AfterClientSync(p *player.Player, f func(p *player.Player, tx *world.Tx)) {
	c := Of(p)
	if c == nil {
		return
	}
	h := p.H()
	run := func() {
		h.Do(func(tx *world.Tx, e world.Entity) { f(e.(*player.Player), tx) })
	}
	ts := rand.Int64N(1<<20) + 1<<20
	syncMu.Lock()
	if syncs[c] == nil {
		syncs[c] = map[int64]func(){}
	}
	syncs[c][ts] = run
	syncMu.Unlock()
	c.WritePacket(&packet.NetworkStackLatency{Timestamp: ts, NeedsResponse: true})
	time.AfterFunc(syncTimeout, func() {
		if f := takeSync(c, ts); f != nil {
			f()
		}
	})
}

func takeSync(c *Conn, ts int64) func() {
	syncMu.Lock()
	defer syncMu.Unlock()
	f := syncs[c][ts]
	delete(syncs[c], ts)
	return f
}
