package sghook

import (
	"context"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
)

// How long a player's join takes, per connection, logged as one INFO line
// ("join timing") when the client closes its loading screen. It compares the
// client load with more or fewer custom block states (SKYGEAR_STATE_PROBE).
//
// The marks and where each is seen, all without patching Dragonfly:
//
//   - login: the Allower runs when the Login packet is handled (after the
//     identity is verified), keyed by the identity because the connection is
//     not handed out yet.
//   - packs: Listener.Accept returns a connection the moment the client's
//     ResourcePackClientResponse "completed" arrives.
//   - start: StartGameContext is called, which sends StartGame.
//   - loaded: StartGameContext returns, which it does when the client's
//     SetLocalPlayerAsInitialised arrives. gophertunnel handles that packet
//     while spawning, so it never reaches ReadPacket; this is its end.
//
// Not seen: the client's pack download steps (they are inside gophertunnel's
// handshake), so there is no packs_ms of its own; connect_to_packs_ms covers
// login to packs completed.

// JoinProbe is the state probe mode named in the join timing line ("off",
// "pair2"); the program sets it at startup.
var JoinProbe = "off"

// JoinLog is where the join timing lines go; nil is slog.Default().
var JoinLog *slog.Logger

// pendingLoginTTL is how long a login mark waits for its connection to be
// accepted; the marks of connections that never got that far are dropped.
const pendingLoginTTL = 2 * time.Minute

// joinMarks are the times of one join; a zero time is a mark not seen.
type joinMarks struct {
	mu                          sync.Mutex
	login, packs, start, loaded time.Time
}

func (m *joinMarks) set(t *time.Time, at time.Time) {
	m.mu.Lock()
	*t = at
	m.mu.Unlock()
}

// ms is b - a in milliseconds (0.1 precision), false if either is missing.
func ms(a, b time.Time) (float64, bool) {
	if a.IsZero() || b.IsZero() {
		return 0, false
	}
	return float64((b.Sub(a)).Microseconds()/100) / 10, true
}

// attrs are the log attributes of a finished join, nil if the client has not
// finished loading (no line is logged then). A duration whose marks are
// missing is left out.
func (m *joinMarks) attrs(player, probe string) []any {
	m.mu.Lock()
	login, packs, start, loaded := m.login, m.packs, m.start, m.loaded
	m.mu.Unlock()
	if loaded.IsZero() {
		return nil
	}
	out := []any{"player", player, "probe", probe}
	for _, d := range []struct {
		name string
		a, b time.Time
	}{
		{"total_ms", login, loaded},
		{"connect_to_packs_ms", login, packs},
		{"to_start_game_ms", packs, start},
		{"client_load_ms", start, loaded},
	} {
		if v, ok := ms(d.a, d.b); ok {
			out = append(out, d.name, v)
		}
	}
	return out
}

// logins are the login marks of connections not accepted yet, by identity.
type logins struct {
	mu sync.Mutex
	m  map[string]time.Time
}

var pendingLogins logins

// add records the login of identity at t and drops marks older than the TTL.
func (l *logins) add(identity string, t time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.m == nil {
		l.m = map[string]time.Time{}
	}
	for id, at := range l.m {
		if t.Sub(at) > pendingLoginTTL {
			delete(l.m, id)
		}
	}
	l.m[identity] = t
}

// take removes and returns the login mark of identity (zero if none).
func (l *logins) take(identity string) time.Time {
	l.mu.Lock()
	defer l.mu.Unlock()
	t := l.m[identity]
	delete(l.m, identity)
	return t
}

// timingAllower marks the login of every connection and lets the Allower it
// wraps decide.
type timingAllower struct{ server.Allower }

func (a timingAllower) Allow(addr net.Addr, d login.IdentityData, c login.ClientData) (string, bool) {
	if a.Allower != nil {
		if reason, ok := a.Allower.Allow(addr, d, c); !ok {
			return reason, false
		}
	}
	pendingLogins.add(d.Identity, time.Now())
	return "", true
}

// StartGameContext marks the start of the game and, once the client has
// finished loading, logs the join timing.
func (c *Conn) StartGameContext(ctx context.Context, data minecraft.GameData) error {
	c.join.set(&c.join.start, time.Now())
	err := c.Conn.StartGameContext(ctx, data)
	if err != nil {
		return err
	}
	c.join.set(&c.join.loaded, time.Now())
	log := JoinLog
	if log == nil {
		log = slog.Default()
	}
	if attrs := c.join.attrs(c.IdentityData().DisplayName, JoinProbe); attrs != nil {
		log.Info("join timing", attrs...)
	}
	return nil
}
