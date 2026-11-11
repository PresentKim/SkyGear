package expand_test

import (
	"testing"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/core/coretest"
	"github.com/presentkim/skygear/content/skygear/expand"
)

// heldByTest is the player the test takeover holds (nil: nobody). It stands
// for another feature's takeover, which this test binary does not set up.
var heldByTest *world.EntityHandle

func init() {
	core.RegisterTakeover("test", func(_ *core.Session, p *player.Player) bool { return p.H() == heldByTest })
}

// TestShowRefusedInTakeover: a player another takeover holds does not enter
// expansion mode; once it lets go, Show starts expansion mode.
func TestShowRefusedInTakeover(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	h := coretest.SpawnPlayer(s)
	t.Cleanup(func() { heldByTest = nil })
	coretest.WithPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		p.SetGameMode(world.GameModeSpectator)
		heldByTest = p.H()
		expand.Show(s.Session, p)
		if _, ok := expand.Of(s.Session, p); ok {
			t.Fatal("entered expansion mode while another takeover holds the player")
		}
		if m := p.GameMode(); m != world.GameModeSpectator {
			t.Fatalf("game mode %T after a refused Show, want spectator unchanged", m)
		}
		if p.Immobile() {
			t.Fatal("player frozen by a refused Show")
		}
		heldByTest = nil
		expand.Show(s.Session, p)
		if _, ok := expand.Of(s.Session, p); !ok {
			t.Fatal("not in expansion mode without a takeover")
		}
		if _, ok := p.GameMode().(expand.ViewOnlyMode); !ok {
			t.Fatalf("game mode %T in expansion mode", p.GameMode())
		}
		expand.Stop(s.Session, p, tx)
	})
}
