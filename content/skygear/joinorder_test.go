package skygear

import (
	"math/rand/v2"
	"testing"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/fairy"
	"github.com/presentkim/skygear/content/skygear/screens"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// TestFirstJoinKeepsBackfilledDust: a first-time player joins a save whose
// backfilled fairy dust fills more than eight stacks. The menu item takes
// slot 8 before the backfill hands out dust, so every dust the backfill
// rolled is in the inventory and the menu item is in slot 8.
func TestFirstJoinKeepsBackfilledDust(t *testing.T) {
	const nights = 401
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	day := s.Clock.DayLength()
	s.Clock.Advance((nights-1)*day + day*fairy.FairyTo/100)
	if n := fairy.PastNights(s.Session); n != nights {
		t.Fatalf("%d nights past, want %d", n, nights)
	}
	// The backfill rolls each night's dust from the session's random source
	// (fairy.BackfillDust); the same seed gives the expected total.
	const seed1, seed2 = 1, 2
	s.Rand = rand.New(rand.NewPCG(seed1, seed2))
	r := rand.New(rand.NewPCG(seed1, seed2))
	want := 0
	for range nights {
		want += fairy.FairyDustMin + r.IntN(fairy.FairyDustMax-fairy.FairyDustMin+1)
	}
	if want <= 8*64 {
		t.Fatalf("%d dust fills no more than eight stacks", want)
	}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		s.Join(p, true)
		if got := core.CountPlain(p.Inventory(), sgitem.FairyDust{}); got != want {
			t.Errorf("%d fairy dust in the inventory after the first join, want %d", got, want)
		}
		if it, _ := p.Inventory().Item(8); !screens.IsMenuItem(it) {
			t.Errorf("slot 8 holds %v, want the menu item", it)
		}
	})
}
