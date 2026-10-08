package dragonfactory

import (
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

func TestUrgentMissionLifecycle(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		s.Second(tx)
		if s.state.NextUrgent != 0 || s.state.Urgent != nil {
			t.Fatal("urgent mission scheduled at Lv.1")
		}
		s.state.Level = 2
		s.Second(tx)
		if s.state.NextUrgent != urgentFirstDelay {
			t.Fatalf("first urgent mission at %v", s.state.NextUrgent)
		}
		s.clock.Advance(urgentFirstDelay)
		s.Second(tx)
		u := s.state.Urgent
		if u == nil {
			t.Fatal("no urgent mission after the first delay")
		}
		it, ok := urgentItem(u.Item)
		if !ok {
			t.Fatalf("unknown urgent item %s", u.Item)
		}
		// Only items of the current level are asked for.
		for _, c := range urgentItems {
			if itemKey(c.Item) == u.Item && c.Level > 2 {
				t.Fatalf("Lv.2 asked for %s (Lv.%d)", u.Item, c.Level)
			}
		}
		// Selling to it pays outside the buyback budget.
		budget := s.buyback().Budget
		_ = p.Inventory().SetItem(0, item.NewStack(it, 10))
		before := s.Money(p)
		n, earned := s.deliverUrgentInventory(p, tx)
		if n != 10 || earned != 10*urgentPrice(u) || s.Money(p)-before != earned {
			t.Fatalf("sold %d for %d", n, earned)
		}
		if s.buyback().Budget != budget {
			t.Fatal("urgent sale used the buyback budget")
		}
		// Selling the rest ends it and schedules the next.
		_ = p.Inventory().SetItem(0, item.NewStack(it, 64))
		_ = p.Inventory().SetItem(1, item.NewStack(it, 64))
		s.deliverUrgentInventory(p, tx)
		if u.Sold != u.Max {
			t.Fatalf("sold %d of %d", u.Sold, u.Max)
		}
		s.Second(tx)
		now, _, _ := s.clock.Now()
		if s.state.Urgent != nil || s.state.NextUrgent <= now {
			t.Fatalf("not ended: urgent %v next %v now %v", s.state.Urgent, s.state.NextUrgent, now)
		}
	})
}

func TestUrgentMissionTimesOut(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		s.state.Level = 2
		now, _, _ := s.clock.Now()
		s.startUrgent(tx, now)
		u := s.state.Urgent
		if d := u.Ends - now; d < 10*time.Minute || d > 30*time.Minute {
			t.Fatalf("duration %v", d)
		}
		s.clock.Advance(u.Ends - now)
		s.Second(tx)
		if s.state.Urgent != nil {
			t.Fatal("urgent mission did not end in time")
		}
	})
}

func TestDeliveryBoxHopperFeedsUrgent(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		s.state.Urgent = &UrgentMission{Item: itemKey(block.Gravel{}), Max: 5, Percent: 200, Ends: time.Hour}
		// Lv.1 needs cobblestone, not gravel: the hopper item goes to the
		// urgent mission and pays the host.
		if s.needed(block.Gravel{}) != 0 {
			t.Fatal("test assumes the mission does not need gravel")
		}
		if n := s.deliverUrgent(tx, nil, block.Gravel{}, 3); n != 3 {
			t.Fatalf("took %d", n)
		}
		if got, _ := s.g.db.Money(s.saveID, "host"); got != 3*urgentPrice(s.state.Urgent) {
			t.Fatalf("host got %d", got)
		}
	})
}
