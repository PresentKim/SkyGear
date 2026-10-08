package dragonfactory

import (
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
	w := world.Config{Synchronous: true, Entities: entity.DefaultRegistry}.New()
	t.Cleanup(func() { _ = w.Close() })
	g := &Game{log: slog.Default(), db: d, sessions: map[*world.World]*session{}}
	s := &session{
		g: g, world: w, clock: instance.NewClock(0, 1, 0, 20*time.Minute), radius: 4,
		saveID: save.ID, saveName: save.Name, state: newState(), money: map[string]int64{},
		rand: rand.New(rand.NewPCG(1, 2)),
	}
	s.ensureCores()
	g.sessions[w] = s
	activeGame = g
	w.Handle(&worldHandler{s: s})
	return s
}
