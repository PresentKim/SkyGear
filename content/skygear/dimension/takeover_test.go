package dimension_test

import (
	"testing"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/core/coretest"
	"github.com/presentkim/skygear/content/skygear/dimension"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// heldByTest is the player the test takeover holds (nil: nobody). It stands
// for another feature's takeover, which this test binary does not set up.
var heldByTest *world.EntityHandle

func init() {
	core.RegisterTakeover("test", func(_ *core.Session, p *player.Player) bool { return p.H() == heldByTest })
}

// TestHuntRefusedInTakeover: a player another takeover holds cannot start a
// hunt; once it lets go, the hunt begins.
func TestHuntRefusedInTakeover(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	h := coretest.SpawnPlayer(s)
	t.Cleanup(func() { heldByTest = nil })
	coretest.WithPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		from := p.Position()
		heldByTest = p.H()
		if dimension.Begin(s.Session, tx, p, sgitem.MobSlime) {
			t.Fatal("a hunt began while another takeover holds the player")
		}
		if dimension.RunOf(s.Session) != nil || dimension.Hunting(s.Session, p) || p.Position() != from {
			t.Fatalf("refused hunt: run %v, hunting %v, at %v", dimension.RunOf(s.Session) != nil, dimension.Hunting(s.Session, p), p.Position())
		}
		heldByTest = nil
		if !dimension.Begin(s.Session, tx, p, sgitem.MobSlime) {
			t.Fatal("the hunt did not begin without a takeover")
		}
		if !dimension.Hunting(s.Session, p) || !core.InArena(p.Position()) {
			t.Fatalf("hunting %v at %v", dimension.Hunting(s.Session, p), p.Position())
		}
		dimension.Died(s.Session, tx, p)
	})
}

// TestWatchRefusedInTakeover: a player another takeover holds is not taken to
// watch a run; once it lets go, it watches.
func TestWatchRefusedInTakeover(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	h1, h2 := coretest.SpawnPlayer(s), coretest.SpawnPlayer(s)
	t.Cleanup(func() { heldByTest = nil })
	coretest.WithPlayer(t, s, h1, func(hunter *player.Player, tx *world.Tx) {
		e, _ := h2.Entity(tx)
		w := e.(*player.Player)
		w.SetGameMode(world.GameModeSurvival)
		from := w.Position()
		if !dimension.Begin(s.Session, tx, hunter, sgitem.MobSlime) {
			t.Fatal("the hunt did not begin")
		}
		heldByTest = w.H()
		dimension.Watch(s.Session, tx, w)
		if dimension.Watching(s.Session, w) || w.GameMode() != world.GameModeSurvival || w.Position() != from {
			t.Fatalf("refused watch: watching %v, mode %v, at %v", dimension.Watching(s.Session, w), w.GameMode(), w.Position())
		}
		heldByTest = nil
		dimension.Watch(s.Session, tx, w)
		if !dimension.Watching(s.Session, w) || w.GameMode() != world.GameModeSpectator || !core.InArena(w.Position()) {
			t.Fatalf("watcher: watching %v, mode %v, at %v", dimension.Watching(s.Session, w), w.GameMode(), w.Position())
		}
		dimension.Died(s.Session, tx, hunter)
	})
}
