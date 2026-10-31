package skygear

import (
	"math"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
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
				if m.Md().Variant == dimension.VariantBase && !m.Md().Elite {
					strongest = max(strongest, m.Health())
				}
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
			t.Errorf("the strongest base monster had %v health, want %v (last stage)", strongest, base*1.75)
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

// TestVariants: a big slime falls apart into small ones that count among the
// hunt's monsters, and a creeper beside the hunter blows up, hurting it.
func TestVariants(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		dimension.Begin(s.Session, tx, p, sgitem.MobSlime)
		r := dimension.RunOf(s.Session)
		big := world.EntitySpawnOpts{Position: p.Position().Add(mgl64.Vec3{5, 0, 0})}.New(dimension.VariantType(sgitem.MobSlime, dimension.VariantSecond),
			dimension.MonsterConfig{Mob: sgitem.MobSlime, Variant: dimension.VariantSecond})
		r.Alive[big] = true
		r.Spawned++
		tx.AddEntity(big)
		e, _ := big.Entity(tx)
		if sc := e.(*dimension.Monster).Scale(); sc <= 1 {
			t.Errorf("a big slime drawn at scale %v", sc)
		}
		spawned := r.Spawned
		e.(*dimension.Monster).Hurt(tx, 1e6, p)
		if r.Spawned != spawned+2 || len(r.Alive) != 2 {
			t.Errorf("a big slime left %d monsters (spawned %d → %d), want 2", len(r.Alive), spawned, r.Spawned)
		}
		dimension.Died(s.Session, tx, p)

		dimension.Begin(s.Session, tx, p, sgitem.MobCreeper)
		r = dimension.RunOf(s.Session)
		c := world.EntitySpawnOpts{Position: p.Position().Add(mgl64.Vec3{1, 0, 0})}.New(dimension.VariantType(sgitem.MobCreeper, dimension.VariantBase),
			dimension.MonsterConfig{Mob: sgitem.MobCreeper})
		r.Alive[c] = true
		r.Spawned++
		tx.AddEntity(c)
		ce, _ := c.Entity(tx)
		health := p.Health()
		for range 40 {
			if !r.Owns(c) {
				break
			}
			ce.(*dimension.Monster).Tick(tx, 0)
		}
		if r.Owns(c) {
			t.Fatal("a creeper beside the hunter did not blow up")
		}
		if p.Health() >= health {
			t.Error("the blast did not hurt the hunter")
		}
		if r.Kills != 1 {
			t.Errorf("%d kills after the blast, want 1", r.Kills)
		}
	})
}

// TestWatch: another player watches a run as a spectator over the ground and
// comes back as it was when the run ends.
func TestWatch(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h1, h2 := spawnPlayer(s), spawnPlayer(s)
	withPlayer(t, s, h1, func(hunter *player.Player, tx *world.Tx) {
		e, _ := h2.Entity(tx)
		w := e.(*player.Player)
		from := w.Position()
		dimension.Begin(s.Session, tx, hunter, sgitem.MobSlime)
		if dimension.Begin(s.Session, tx, w, sgitem.MobSlime) {
			t.Fatal("a second hunter got in")
		}
		dimension.Watch(s.Session, tx, w)
		if !dimension.Watching(s.Session, w) || w.GameMode() != world.GameModeSpectator || !core.InArena(w.Position()) {
			t.Fatalf("watcher: watching %v, mode %v, at %v", dimension.Watching(s.Session, w), w.GameMode(), w.Position())
		}
		dimension.Died(s.Session, tx, hunter)
		if dimension.Watching(s.Session, w) || w.GameMode() == world.GameModeSpectator || w.Position() != from {
			t.Errorf("after the run the watcher is at %v in mode %v", w.Position(), w.GameMode())
		}
	})
}
