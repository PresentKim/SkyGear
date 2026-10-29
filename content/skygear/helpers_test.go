package skygear

import (
	"log/slog"
	"math/rand/v2"
	"path/filepath"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/debug"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/platform/db"
	"github.com/presentkim/skygear/platform/instance"
	"golang.org/x/text/language"
)

// testSession returns a session over a synchronous world, without an
// instance manager.
func testSession(t *testing.T) *session {
	t.Helper()
	d, err := db.Open(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	if _, err := d.TouchAccount("host", "", "host"); err != nil {
		t.Fatal(err)
	}
	save, err := d.CreateSave("host", 1, "test")
	if err != nil {
		t.Fatal(err)
	}
	w := world.Config{Synchronous: true, Entities: entity.DefaultRegistry.Config().New(EntityTypes())}.New()
	t.Cleanup(func() { _ = w.Close() })
	g := &Game{log: slog.Default(), db: d}
	c := core.New(core.Config{
		World: w, Clock: instance.NewClock(0, 1, 0, 20*time.Minute),
		SaveID: save.ID, SaveName: save.Name, HostID: "host", Log: slog.Default(),
	})
	c.Rand = rand.New(rand.NewPCG(1, 2))
	s := &session{
		Session: c, g: g,
	}
	s.state = savedState.Of(c)
	t.Cleanup(c.Close)
	g.sessions.Store(w, s)
	core.Register(c)
	t.Cleanup(func() { core.Unregister(c) })
	activeGame = g
	registerItemIO()
	w.Handle(&worldHandler{s: s})
	return s
}

func init() { world.DefaultBlockRegistry.Finalize() }

func loadAround(s *session) {
	s.World.Do(func(tx *world.Tx) {
		l := world.NewLoader(2, s.World, world.NopViewer{})
		l.Move(tx, mgl64.Vec3{})
		l.Load(tx, 50)
	})
}

func spawnPlayer(s *session) *world.EntityHandle {
	id := uuid.New()
	h := world.EntitySpawnOpts{ID: id}.New(player.Type, player.Config{Name: "p", UUID: id, Locale: language.Korean, Position: mgl64.Vec3{0.5, 64, 2.5}})
	s.World.Do(func(tx *world.Tx) { tx.AddEntity(h) })
	return h
}

// withPlayer runs f with the player in a transaction. A panic in f fails
// the test (Dragonfly would only log it).
func withPlayer(t *testing.T, s *session, h *world.EntityHandle, f func(p *player.Player, tx *world.Tx)) {
	t.Helper()
	finished := false
	s.World.Do(func(tx *world.Tx) {
		e, _ := h.Entity(tx)
		f(e.(*player.Player), tx)
		finished = true
	})
	if !finished {
		t.Fatal("transaction did not finish (panicked)")
	}
}

func tick(s *session, n int) {
	for range n {
		s.World.AdvanceTick()
	}
}

// shapeAs is the shape s keeps under key (nil if none).
func shapeAs(s *session, key any) debug.Shape {
	sh, _ := s.Shape(key)
	return sh
}
