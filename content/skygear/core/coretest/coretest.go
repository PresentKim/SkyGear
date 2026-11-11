// Package coretest gives the tests of a game package what the root package's
// tests have: a running save on a synchronous world, a player in it, and the
// features the tests need, set up once in the test binary. A test binary of a
// package other than the root links no feature list, so nothing is installed
// in it (core.Install runs in the root package only): its tests name the
// features they need in Setup. Only test files import this package.
package coretest

import (
	"log/slog"
	"math/rand/v2"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/debug"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/platform/account"
	"github.com/presentkim/skygear/platform/instance"
	"golang.org/x/text/language"
)

var (
	mu sync.Mutex
	// setUp are the features Setup has set up in this test binary.
	setUp []*core.Feature
)

// Setup closes the block registry (every block is registered when a test
// file's init runs) and sets up each feature of fs that this test binary has
// not set up yet: it registers the feature's settings and runs its Setup, as
// core.Install does, but runs no Check, so a feature may be set up without
// the others. A feature core.Install installed (the root package's tests) or
// an earlier call set up is skipped, so naming a feature twice is safe.
//
// Call it from the init function of a test file, never from a test: a hook
// takes no subscriber after its first event.
func Setup(fs ...*core.Feature) {
	world.DefaultBlockRegistry.Finalize()
	mu.Lock()
	defer mu.Unlock()
	installed := core.Features()
	for _, f := range fs {
		if slices.Contains(setUp, f) || slices.Contains(installed, f) {
			continue
		}
		setUp = append(setUp, f)
		if len(f.Settings) > 0 {
			account.RegisterSetting(f.Settings...)
		}
		if f.Setup != nil {
			f.Setup()
		}
	}
}

// entityTypes are the default entity types, those of the installed features
// and those of the features Setup set up.
func entityTypes() []world.EntityType {
	types := append(entity.DefaultRegistry.Types(), core.EntityTypes()...)
	mu.Lock()
	defer mu.Unlock()
	for _, f := range setUp {
		types = append(types, f.Entities...)
	}
	return types
}

// Session returns the session of a new save (id 1, named "test", owned by the
// account "host") over a synchronous world, without an instance manager and
// without a database. core.Of finds the session, the world's handler is the
// kernel's, and the random source is seeded the same in every run. The world
// and the session are closed when the test ends.
//
// The result is the session as the instance sees it: its Session field is the
// core session, its methods are the calls the instance makes (Join, Leave,
// Second, Fast, NewDay, Autosave, Close) and Handler is a player's handler.
func Session(t testing.TB) core.Lifecycle {
	t.Helper()
	world.DefaultBlockRegistry.Finalize()
	w := world.Config{Synchronous: true, Entities: entity.DefaultRegistry.Config().New(entityTypes())}.New()
	t.Cleanup(func() { _ = w.Close() })
	c := core.New(core.Config{
		World: w, Clock: instance.NewClock(0, 1, 0, 20*time.Minute),
		SaveID: 1, SaveName: "test", HostID: "host", Log: slog.Default(),
	})
	c.Rand = rand.New(rand.NewPCG(1, 2))
	t.Cleanup(c.Close)
	core.Register(c)
	t.Cleanup(func() { core.Unregister(c) })
	w.Handle(&core.WorldHandler{S: c})
	return core.Lifecycle{Session: c}
}

// LoadAround loads the chunks around the origin of s's world.
func LoadAround(s core.Lifecycle) {
	s.World.Do(func(tx *world.Tx) {
		l := world.NewLoader(2, s.World, world.NopViewer{})
		l.Move(tx, mgl64.Vec3{})
		l.Load(tx, 50)
	})
}

// SpawnPlayer adds a player named "p" with the Korean locale to s's world, at
// (0.5, 64, 2.5), and returns its handle.
func SpawnPlayer(s core.Lifecycle) *world.EntityHandle {
	id := uuid.New()
	h := world.EntitySpawnOpts{ID: id}.New(player.Type, player.Config{Name: "p", UUID: id, Locale: language.Korean, Position: mgl64.Vec3{0.5, 64, 2.5}})
	s.World.Do(func(tx *world.Tx) { tx.AddEntity(h) })
	return h
}

// WithPlayer runs f with the player of h in a transaction. A panic in f fails
// the test (Dragonfly would only log it).
func WithPlayer(t testing.TB, s core.Lifecycle, h *world.EntityHandle, f func(p *player.Player, tx *world.Tx)) {
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

// Tick advances s's world by n ticks.
func Tick(s core.Lifecycle, n int) {
	for range n {
		s.World.AdvanceTick()
	}
}

// ShapeAs is the shape s keeps under key (nil if none).
func ShapeAs(s core.Lifecycle, key any) debug.Shape {
	sh, _ := s.Shape(key)
	return sh
}
