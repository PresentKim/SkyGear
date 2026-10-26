package dfhook

import (
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// readyTimeout is how long AfterClientReady waits for a client that never
// reports the end of its loading screen.
const readyTimeout = 10 * time.Second

// readiness is whether a client's loading screen is closed, and what waits
// for it to close.
type readiness struct {
	mu      sync.Mutex
	ready   bool
	waiting map[int]func()
	next    int
}

var (
	readyMu sync.Mutex
	readies = map[*Conn]*readiness{}
)

func readinessOf(c *Conn) *readiness {
	readyMu.Lock()
	defer readyMu.Unlock()
	r := readies[c]
	if r == nil {
		r = &readiness{waiting: map[int]func(){}}
		readies[c] = r
	}
	return r
}

func init() {
	OnClientPacket(func(c *Conn, pk packet.Packet) bool {
		if l, ok := pk.(*packet.ServerBoundLoadingScreen); ok {
			r := readinessOf(c)
			r.mu.Lock()
			r.ready = l.Type == packet.LoadingScreenTypeEnd
			var waiting []func()
			if r.ready {
				for id, f := range r.waiting {
					waiting = append(waiting, f)
					delete(r.waiting, id)
				}
			}
			r.mu.Unlock()
			for _, f := range waiting {
				f()
			}
		}
		return false
	})
	OnClose(func(c *Conn) {
		readyMu.Lock()
		delete(readies, c)
		readyMu.Unlock()
	})
}

// AfterClientReady runs f once p's client has closed its loading screen
// (ServerBoundLoadingScreen end), since a client still loading drops forms.
// f runs at once if the screen is already closed or p is not hooked, else
// later in p's transaction, at most readyTimeout later. Call it in p's
// transaction.
func AfterClientReady(p *player.Player, f func(p *player.Player, tx *world.Tx)) {
	c := Of(p)
	if c == nil {
		f(p, p.Tx())
		return
	}
	r := readinessOf(c)
	r.mu.Lock()
	if r.ready {
		r.mu.Unlock()
		f(p, p.Tx())
		return
	}
	h := p.H()
	id := r.next
	r.next++
	r.waiting[id] = func() {
		h.Do(func(tx *world.Tx, e world.Entity) { f(e.(*player.Player), tx) })
	}
	r.mu.Unlock()
	time.AfterFunc(readyTimeout, func() {
		r.mu.Lock()
		run, ok := r.waiting[id]
		delete(r.waiting, id)
		r.mu.Unlock()
		if ok {
			run()
		}
	})
}
