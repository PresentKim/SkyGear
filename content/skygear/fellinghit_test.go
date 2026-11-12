package skygear

import (
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/dimension"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/platform/account"
	"github.com/presentkim/skygear/platform/instance"
)

// TestFellingHitIsNotAHurt: the hit that would kill a hunter is cancelled
// (the run ends instead and the hunter goes back unharmed), and a cancelled
// hit is not a hurt: the hurt counter is not called for it. A hit the hunter
// takes counts.
func TestFellingHitIsNotAHurt(t *testing.T) {
	d := bridgeAccount(t)
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	t.Cleanup(func() {
		s.World.Do(func(tx *world.Tx) {
			if e, ok := h.Entity(tx); ok {
				tx.RemoveEntity(e)
			}
		})
	})
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		acc := instance.AccountID(p)
		if _, err := d.TouchAccount(acc, "", p.Name()); err != nil {
			t.Fatal(err)
		}
		if !dimension.Begin(s.Session, tx, p, sgitem.MobSlime) {
			t.Fatal("the hunt did not begin")
		}
		hurt := func(dmg float64) *player.Context {
			ctx := player.NewEventContext(tx, p)
			var immunity time.Duration
			s.Handler(p).HandleHurt(ctx, &dmg, false, &immunity, nil)
			return ctx
		}
		if hurt(1).Cancelled() {
			t.Fatal("a hit the hunter survives was cancelled")
		}
		if got := account.Counter(acc, "hurt"); got != 1 {
			t.Errorf("hurt counted %d for a hit the hunter took, want 1", got)
		}
		if !hurt(p.Health() + 100).Cancelled() {
			t.Fatal("the hit that would kill the hunter was not cancelled")
		}
		if dimension.Hunting(s.Session, p) {
			t.Fatal("the run did not end")
		}
		if got := account.Counter(acc, "hurt"); got != 1 {
			t.Errorf("hurt counted %d after the cancelled felling hit, want 1", got)
		}
	})
}
