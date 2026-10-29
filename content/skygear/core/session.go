package core

import (
	"log/slog"
	"math"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/platform/instance"
)

// Session is one running save. It and every part in it are owned by the
// save's world: read and change them only in that world's transactions.
type Session struct {
	World *world.World
	// Clock is the save's instance clock (nil in tests that need none).
	Clock *instance.Clock
	Rand  *rand.Rand
	// SaveID, SaveName and HostID are the save's id, name and owner account.
	SaveID   int64
	SaveName string
	HostID   string
	Log      *slog.Logger
	// IsOperator reports whether a player may use the operator tools.
	IsOperator func(*player.Player) bool

	parts map[int]any
	// dirty is whether the saved parts changed since they were last handed
	// to writer, which stores them (MarkDirty).
	dirty  bool
	writer *stateWriter
	store  func([]byte) error
	// lastOpen is when each player last opened a screen (Debounced).
	lastOpen map[*world.EntityHandle]time.Time
}

// Config is what New needs of a save.
type Config struct {
	World    *world.World
	Clock    *instance.Clock
	SaveID   int64
	SaveName string
	HostID   string
	Log      *slog.Logger
	// IsOperator may be nil: no player is an operator.
	IsOperator func(*player.Player) bool
	// State is the save's state document (nil: a new save), Store writes a
	// new one.
	State []byte
	Store func([]byte) error
}

// New makes the session of a save. A state document that cannot be read
// is logged and the save starts from new parts.
func New(c Config) *Session {
	s := &Session{
		World: c.World, Clock: c.Clock, SaveID: c.SaveID, SaveName: c.SaveName, HostID: c.HostID,
		Log: c.Log, IsOperator: c.IsOperator, store: c.Store, parts: map[int]any{},
		Rand: rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), uint64(c.SaveID))),
	}
	if s.Log == nil {
		s.Log = slog.Default()
	}
	if s.IsOperator == nil {
		s.IsOperator = func(*player.Player) bool { return false }
	}
	if err := s.loadSaved(c.State); err != nil {
		s.Log.Error("load skygear state; starting from a new state", "save", s.SaveID, "err", err)
		_ = s.loadSaved(nil)
	}
	return s
}

var sessions sync.Map // *world.World -> *Session

// Register makes s the session of its world for Of, until Unregister.
func Register(s *Session) { sessions.Store(s.World, s) }

// Unregister forgets the session of s's world.
func Unregister(s *Session) { sessions.Delete(s.World) }

// Of is the session running in the world of tx, if any. Blocks and items,
// whose methods get only a transaction, find their session this way.
func Of(tx *world.Tx) (*Session, bool) { return OfWorld(tx.World()) }

// OfWorld is the session running in w, if any.
func OfWorld(w *world.World) (*Session, bool) {
	v, ok := sessions.Load(w)
	if !ok {
		return nil, false
	}
	return v.(*Session), true
}

// Leave drops p's entries of every Players part.
func (s *Session) Leave(p *player.Player) {
	h := p.H()
	delete(s.lastOpen, h)
	partsMu.Lock()
	drop := leaving
	partsMu.Unlock()
	for _, f := range drop {
		f(s, h)
	}
}

// Debounced reports whether p opened a screen less than 500ms ago, and
// otherwise records this opening: the client repeats the use packet while
// the button is held.
func (s *Session) Debounced(p *player.Player) bool {
	if s.lastOpen == nil {
		s.lastOpen = map[*world.EntityHandle]time.Time{}
	}
	if t, ok := s.lastOpen[p.H()]; ok && time.Since(t) < 500*time.Millisecond {
		return true
	}
	s.lastOpen[p.H()] = time.Now()
	return false
}

// DropAt drops an item stack at pos with a small random push, so drops
// never just fall dead still. Every item the game drops goes through here
// unless it is thrown on purpose (the fairy's gift).
func (s *Session) DropAt(tx *world.Tx, pos mgl64.Vec3, it item.Stack) {
	a := s.Rand.Float64() * 2 * math.Pi
	speed := 0.06 + s.Rand.Float64()*0.08
	vel := mgl64.Vec3{speed * math.Cos(a), 0.22 + s.Rand.Float64()*0.1, speed * math.Sin(a)}
	tx.AddEntity(entity.NewItem(world.EntitySpawnOpts{Position: pos, Velocity: vel}, it))
}

// Broadcast messages every player in the save the text of key.
func (s *Session) Broadcast(tx *world.Tx, key string, args ...any) {
	for e := range tx.Players() {
		p := e.(*player.Player)
		p.Message(i18n.T(i18n.Of(p), key, args...))
	}
}

// MarkDirty marks the saved parts to be stored on the next Flush.
func (s *Session) MarkDirty() { s.dirty = true }

// Dirty reports whether the saved parts changed since the last Flush.
func (s *Session) Dirty() bool { return s.dirty }

// Flush hands the saved parts to the writer, which stores them outside the
// world.
func (s *Session) Flush() {
	b, err := s.marshalSaved()
	if err != nil {
		s.Log.Error("store skygear state", "save", s.SaveID, "err", err)
		return
	}
	s.dirty = false
	if s.writer == nil {
		s.writer = newStateWriter(s.write)
	}
	s.writer.put(b)
}

// Close stops the writer and stores the saved parts before returning.
func (s *Session) Close() {
	if s.writer != nil {
		s.writer.stop()
		s.writer = nil
	}
	b, err := s.marshalSaved()
	if err != nil {
		s.Log.Error("store skygear state", "save", s.SaveID, "err", err)
		return
	}
	s.dirty = false
	s.write(b)
}

func (s *Session) write(b []byte) {
	if s.store == nil {
		return
	}
	if err := s.store(b); err != nil {
		s.Log.Error("store skygear state", "save", s.SaveID, "err", err)
	}
}

// Opened reports whether p opened a screen not long ago (Debounced).
func (s *Session) Opened(p *player.Player) bool {
	_, ok := s.lastOpen[p.H()]
	return ok
}

// ForgetOpen forgets when p last opened a screen, so the next opening is not
// debounced.
func (s *Session) ForgetOpen(p *player.Player) { delete(s.lastOpen, p.H()) }

// ForgetOpens forgets every player's last opening.
func (s *Session) ForgetOpens() { s.lastOpen = nil }
