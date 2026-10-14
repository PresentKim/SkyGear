package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
	"log/slog"
	"math/rand/v2"
	"path/filepath"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/db"
	"github.com/presentkim/dragonfactory/internal/instance"
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
	save, err := d.CreateSave("host", "dragonfactory", 1, "test")
	if err != nil {
		t.Fatal(err)
	}
	w := world.Config{Synchronous: true, Entities: entity.DefaultRegistry.Config().New(EntityTypes())}.New()
	t.Cleanup(func() { _ = w.Close() })
	g := &Game{log: slog.Default(), db: d, sessions: map[*world.World]*session{}}
	s := &session{
		g: g, world: w, clock: instance.NewClock(0, 1, 0, 20*time.Minute),
		saveID: save.ID, saveName: save.Name, hostID: "host", state: newState(), displays: map[cube.Pos]*drawerDisplay{},
		lastUse: map[*world.EntityHandle]drawerUse{}, lastPunch: map[*world.EntityHandle]drawerPunch{},
		rand: rand.New(rand.NewPCG(1, 2)),
	}
	g.sessions[w] = s
	activeGame = g
	w.Handle(&worldHandler{s: s})
	return s
}

func init() { world.DefaultBlockRegistry.Finalize() }

func loadAround(s *session) {
	s.world.Do(func(tx *world.Tx) {
		l := world.NewLoader(2, s.world, world.NopViewer{})
		l.Move(tx, mgl64.Vec3{})
		l.Load(tx, 50)
	})
}

func spawnPlayer(s *session) *world.EntityHandle {
	id := uuid.New()
	h := world.EntitySpawnOpts{ID: id}.New(player.Type, player.Config{Name: "p", UUID: id, Position: mgl64.Vec3{0.5, 64, 2.5}})
	s.world.Do(func(tx *world.Tx) { tx.AddEntity(h) })
	return h
}

// withPlayer runs f with the player in a transaction. A panic in f fails
// the test (Dragonfly would only log it).
func withPlayer(t *testing.T, s *session, h *world.EntityHandle, f func(p *player.Player, tx *world.Tx)) {
	t.Helper()
	finished := false
	s.world.Do(func(tx *world.Tx) {
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
		s.world.AdvanceTick()
	}
}
