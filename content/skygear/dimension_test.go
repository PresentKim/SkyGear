package skygear

import (
	"math"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
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

// TestHuntClear: a hunt lays the ground, takes the hunter there, waits out
// the countdown, brings stronger monsters stage by stage, and killing them
// all clears it and takes the hunter back where it came from.
func TestHuntClear(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		from := p.Position()
		if !dimension.Begin(s.Session, tx, p, sgitem.MobSlime) {
			t.Fatal("the hunt did not begin")
		}
		if dimension.Begin(s.Session, tx, p, sgitem.MobSlime) {
			t.Error("two hunts at once")
		}
		c := core.ArenaCentre
		if _, ok := tx.Block(c.Sub(cube.Pos{0, 1, 0})).(block.Grass); !ok {
			t.Error("no grass in the middle of the ground")
		}
		if _, ok := tx.Block(c.Add(cube.Pos{core.ArenaRadius, 1, 0})).(block.Barrier); !ok {
			t.Error("no wall at the rim")
		}
		if !core.InArena(p.Position()) || !dimension.Hunting(s.Session, p) {
			t.Fatalf("the hunter is at %v, not hunting", p.Position())
		}
		for range 100 {
			runTick(t, s, tx)
		}
		if n := len(dimension.RunOf(s.Session).Alive); n != 0 {
			t.Fatalf("%d monsters during the countdown", n)
		}
		r := dimension.RunOf(s.Session)
		count := r.Hunt.Count
		base := dimension.MonsterTable[sgitem.MobSlime].Health
		var strongest float64
		for i := 0; dimension.RunOf(s.Session) != nil && i < 20*r.Hunt.Seconds; i++ {
			runTick(t, s, tx)
			if dimension.RunOf(s.Session) == nil {
				break
			}
			for _, m := range runMonsters(s, tx) {
				strongest = max(strongest, m.Health())
				m.Hurt(tx, 1000, p)
			}
		}
		if dimension.RunOf(s.Session) != nil {
			t.Fatal("the hunt goes on with every monster dead")
		}
		if r.Kills != count {
			t.Errorf("%d kills, want %d", r.Kills, count)
		}
		if math.Abs(strongest-base*1.75) > 1e-9 {
			t.Errorf("the strongest monster had %v health, want %v (last stage)", strongest, base*1.75)
		}
		if p.Position().Sub(from).Len() > 1e-6 || core.InArena(p.Position()) {
			t.Errorf("the hunter is at %v, want back at %v", p.Position(), from)
		}
	})
}

// TestHuntFails: the hunt fails when the time runs out (its monsters go) or
// the hunter dies (it comes back on the island when it respawns).
func TestHuntFails(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		dimension.Begin(s.Session, tx, p, sgitem.MobSlime)
		r := dimension.RunOf(s.Session)
		var monsters []*world.EntityHandle
		for range 100 + 20*r.Hunt.Seconds + 1 {
			if dimension.RunOf(s.Session) == nil {
				break
			}
			// keep the hunter out of the monsters' reach
			p.Heal(20, entity.FoodHealingSource{})
			runTick(t, s, tx)
			for m := range r.Alive {
				monsters = append(monsters, m)
			}
		}
		if dimension.RunOf(s.Session) != nil {
			t.Fatal("the hunt goes on after its time")
		}
		for _, m := range monsters {
			if _, ok := m.Entity(tx); ok {
				t.Fatal("a monster stayed after the hunt failed")
			}
		}
		if core.InArena(p.Position()) {
			t.Error("the hunter stayed in the ground")
		}

		dimension.Begin(s.Session, tx, p, sgitem.MobSlime)
		dimension.Died(s.Session, tx, p)
		if dimension.RunOf(s.Session) != nil {
			t.Fatal("the hunt goes on after the hunter died")
		}
		pos := p.Position()
		dimension.Respawned(s.Session, p, &pos)
		if pos != island.Spawn(s.Session) {
			t.Errorf("respawn at %v, want the island spawn %v", pos, island.Spawn(s.Session))
		}
	})
}

// TestBladesHit: the blades turning around the hunter hurt a monster beside
// it within a turn.
func TestBladesHit(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		dimension.Begin(s.Session, tx, p, sgitem.MobSlime)
		for range 100 {
			runTick(t, s, tx)
		}
		runTick(t, s, tx) // the first monsters come
		mons := runMonsters(s, tx)
		if len(mons) == 0 {
			t.Fatal("no monster came")
		}
		m := mons[0]
		full := m.Health()
		spot := p.Position().Add(mgl64.Vec3{2, 0, 0})
		for range 30 {
			m.Data.Pos = spot
			runTick(t, s, tx)
			if m.Health() < full {
				return
			}
		}
		t.Errorf("a monster beside the hunter kept %v health through a turn", m.Health())
	})
}
