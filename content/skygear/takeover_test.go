package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/dimension"
	"github.com/presentkim/skygear/content/skygear/expand"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// TestTakeoversDoNotNest: island expansion mode, hunting and watching a hunt
// each hold a player's game mode; a player in one is refused the others and
// stays as it was.
func TestTakeoversDoNotNest(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h1, h2 := spawnPlayer(s), spawnPlayer(s)
	withPlayer(t, s, h1, func(hunter *player.Player, tx *world.Tx) {
		e, _ := h2.Entity(tx)
		p := e.(*player.Player)
		p.SetGameMode(world.GameModeSurvival)
		if !dimension.Begin(s.Session, tx, hunter, sgitem.MobSlime) {
			t.Fatal("the hunt did not begin")
		}

		// A watcher asks for expansion mode.
		dimension.Watch(s.Session, tx, p)
		if !dimension.Watching(s.Session, p) {
			t.Fatal("not watching")
		}
		expand.Show(s.Session, p)
		if _, ok := expand.Of(s.Session, p); ok || !dimension.Watching(s.Session, p) || p.GameMode() != world.GameModeSpectator {
			t.Fatalf("watcher after Show: expanding %v, watching %v, mode %T", ok, dimension.Watching(s.Session, p), p.GameMode())
		}
		dimension.StopWatching(s.Session, p)

		// An expanding player asks to watch.
		expand.Show(s.Session, p)
		if _, ok := expand.Of(s.Session, p); !ok {
			t.Fatal("not expanding")
		}
		dimension.Watch(s.Session, tx, p)
		if _, ok := expand.Of(s.Session, p); !ok || dimension.Watching(s.Session, p) || core.InArena(p.Position()) {
			t.Fatalf("expanding player after Watch: expanding %v, watching %v, at %v", ok, dimension.Watching(s.Session, p), p.Position())
		}
		if _, ok := p.GameMode().(expand.ViewOnlyMode); !ok {
			t.Fatalf("expanding player's mode %T after Watch", p.GameMode())
		}
		expand.Stop(s.Session, p, tx)
		if p.GameMode() != world.GameModeSurvival {
			t.Fatalf("mode %T after expansion mode, want survival", p.GameMode())
		}

		// The hunter asks for expansion mode.
		expand.Show(s.Session, hunter)
		if _, ok := expand.Of(s.Session, hunter); ok || !dimension.Hunting(s.Session, hunter) {
			t.Fatalf("hunter after Show: expanding %v, hunting %v", ok, dimension.Hunting(s.Session, hunter))
		}
		dimension.Died(s.Session, tx, hunter)

		// An expanding player asks to hunt.
		expand.Show(s.Session, p)
		if dimension.Begin(s.Session, tx, p, sgitem.MobSlime) || dimension.Hunting(s.Session, p) {
			t.Fatal("an expanding player began a hunt")
		}
		if _, ok := expand.Of(s.Session, p); !ok {
			t.Fatal("not expanding after the refused hunt")
		}
		expand.Stop(s.Session, p, tx)
	})
}
