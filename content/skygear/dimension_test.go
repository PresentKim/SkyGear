package skygear

import (
	"fmt"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/dimension"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// runTick drives the run one tick, as its gate does.
func runTick(t *testing.T, s *session, tx *world.Tx) {
	t.Helper()
	r := dimension.RunOf(s.Session)
	e, ok := r.Portal.Entity(tx)
	if !ok {
		t.Fatal("the run has no gate")
	}
	e.(*dimension.GatePortal).Tick(tx, 0)
}

// runMonsters are the run's monsters out now.
func runMonsters(s *session, tx *world.Tx) []*dimension.Monster {
	var out []*dimension.Monster
	for h := range dimension.RunOf(s.Session).Alive {
		if e, ok := h.Entity(tx); ok {
			out = append(out, e.(*dimension.Monster))
		}
	}
	return out
}

// clearHunt runs p's hunt of mob to a clear, killing every monster as it
// comes.
func clearHunt(t *testing.T, s *session, tx *world.Tx, p *player.Player, mob string) {
	t.Helper()
	if !dimension.Begin(s.Session, tx, p, mob) {
		t.Fatalf("the %s hunt did not begin", mob)
	}
	r := dimension.RunOf(s.Session)
	for range 100 + 20*r.Hunt.Seconds {
		if dimension.RunOf(s.Session) == nil {
			return
		}
		runTick(t, s, tx)
		if dimension.RunOf(s.Session) == nil {
			return
		}
		for _, m := range runMonsters(s, tx) {
			m.Hurt(tx, 1e6, p)
		}
	}
	t.Fatalf("the %s hunt did not clear", mob)
}

// TestHuntRecords: a clear sets the island's records and stars and gives the
// first-clear essences and the stars' dust; the next monster opens with the
// three stars of the one before and a fairy emerald.
func TestHuntRecords(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		slime, zombie := sgitem.MobSlime, sgitem.MobZombie
		if !dimension.IsOpen(s.Session, slime) || dimension.IsOpen(s.Session, zombie) || dimension.CanOpen(s.Session, zombie) {
			t.Fatal("only the slime should be open at first")
		}
		clearHunt(t, s, tx, p, slime)
		rec := dimension.RecordOf(s.Session, slime)
		if rec.Clears != 1 || rec.Best == 0 || rec.Kills != dimension.Hunts[0].Count {
			t.Errorf("record %+v after a clear", *rec)
		}
		stars := dimension.Stars(s.Session, slime)
		if !stars[0] || !stars[1] || stars[2] {
			t.Errorf("stars %v after a quick clear of %d kills", stars, rec.Kills)
		}
		essences := core.CountPlain(p.Inventory(), sgitem.Essence{Mob: slime})
		if essences < 8 {
			t.Errorf("%d slime essences after the first clear, want at least 8", essences)
		}
		if dust := core.CountPlain(p.Inventory(), sgitem.FairyDust{}); dust != 4 {
			t.Errorf("%d fairy dust for two new stars, want 4", dust)
		}
		// a second clear gives no more first-clear essences or star dust
		clearHunt(t, s, tx, p, slime)
		if dust := core.CountPlain(p.Inventory(), sgitem.FairyDust{}); dust != 4 {
			t.Errorf("%d fairy dust after the second clear, want still 4", dust)
		}

		rec.Kills = dimension.Hunts[0].Star3
		if !dimension.CanOpen(s.Session, zombie) {
			t.Fatal("the zombie cannot be opened with the slime's three stars")
		}
		if dimension.Open(s.Session, p, zombie) {
			t.Error("the zombie opened without a fairy emerald")
		}
		_, _ = p.Inventory().AddItem(item.NewStack(sgitem.FairyEmerald{}, 1))
		if !dimension.Open(s.Session, p, zombie) || !dimension.IsOpen(s.Session, zombie) {
			t.Fatal("the zombie did not open for an emerald")
		}
		if core.CountPlain(p.Inventory(), sgitem.FairyEmerald{}) != 0 {
			t.Error("opening kept the emerald")
		}
	})
}

// TestWatchdog: a hunter moved out of the second dimension by something else
// than the run (a member's "bring") ends its run and gets its health back,
// a player moved into it without hunting is sent back, and a player stored
// there enters the save at the island instead.
func TestWatchdog(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h1, h2 := spawnPlayer(s), spawnPlayer(s)
	withPlayer(t, s, h1, func(p *player.Player, tx *world.Tx) {
		e, _ := h2.Entity(tx)
		other := e.(*player.Player)
		dimension.SetKit(s.Session, p, dimension.KitRelic, sgitem.RelicStack(sgitem.RelicLife, 1))
		dimension.Begin(s.Session, tx, p, sgitem.MobSlime)
		p.Teleport(island.Spawn(s.Session))
		dimension.Watchdog(s.Session, tx)
		if dimension.RunOf(s.Session) != nil || p.MaxHealth() != 20 {
			t.Errorf("run %v, max health %v after the hunter left the ground", dimension.RunOf(s.Session) != nil, p.MaxHealth())
		}
		other.Teleport(core.ArenaCentre.Vec3Middle())
		dimension.Watchdog(s.Session, tx)
		if core.InArena(other.Position()) {
			t.Error("a player who does not hunt stays in the ground")
		}
	})
	if !s.Relocate(core.ArenaCentre.Vec3Middle()) || s.Relocate(island.Spawn(s.Session)) {
		t.Error("Relocate should move players stored in the ground only")
	}
}

// TestGiveRelics: /sgadmin give hands out every relic at every grade, and the
// dimension's own items.
func TestGiveRelics(t *testing.T) {
	_ = testSession(t)
	items := giveItems()
	for _, k := range append(append([]string{}, sgitem.RelicWeapons...), sgitem.RelicCharms...) {
		for g := 1; g <= sgitem.RelicMaxGrade; g++ {
			f, ok := items[fmt.Sprintf("relic_%s_%d", k, g)]
			if !ok || sgitem.RelicGrade(f()) != g {
				t.Errorf("no relic_%s_%d of that grade", k, g)
			}
		}
	}
	for _, id := range []string{"dimension_stone", "fairy_emerald", "fairy_dust", "relic_table", "essence_collector"} {
		if _, ok := items[id]; !ok {
			t.Errorf("no %s to give", id)
		}
	}
	for id := range items {
		if strings.HasPrefix(id, "progress_") {
			t.Errorf("%s is given", id)
		}
	}
}
