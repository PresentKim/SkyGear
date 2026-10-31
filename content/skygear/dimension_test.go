package skygear

import (
	"fmt"
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
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/progress"
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

// TestBladesHit: the blades of a hunter carrying the spinning blade hurt a
// monster beside it within a turn.
func TestBladesHit(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		_, _ = p.Inventory().AddItem(sgitem.RelicStack(sgitem.RelicBlade, 1))
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

// TestCollectors: a cleared monster's pool fills by its best time and hands
// its essences to the collectors of that monster in turn; more collectors
// split them, and with none the pool keeps at most a stack.
func TestCollectors(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	a := placeTestMachine(s, machinePos, machine.KindCollector, "")
	b := placeTestMachine(s, machinePos.Add(cube.Pos{2, 0, 0}), machine.KindCollector, "")
	tick(s, 2) // the collectors tick, so the island knows them
	s.World.Do(func(tx *world.Tx) {
		slime := sgitem.MobSlime
		if dimension.PerMinute(s.Session, slime) != 0 {
			t.Fatal("essences before the first clear")
		}
		// 60 slimes in 6 s: 600 a minute, 30 essences a minute
		dimension.RecordOf(s.Session, slime).Best = 6
		if got := dimension.PerMinute(s.Session, slime); math.Abs(got-30) > 1e-9 {
			t.Fatalf("%v essences a minute, want 30", got)
		}
		a.SetMob(slime)
		b.SetMob(slime)
		for range 60 {
			dimension.TickCollectors(s.Session, tx)
		}
		count := func(st *machine.State) int {
			n := 0
			for _, it := range st.Inv().Items() {
				if e, ok := it.Item().(sgitem.Essence); ok && e.Mob == slime {
					n += it.Count()
				}
			}
			return n
		}
		na, nb := count(a), count(b)
		if na+nb < 29 || na+nb > 30 {
			t.Errorf("%d essences in a minute, want 30", na+nb)
		}
		if math.Abs(float64(na-nb)) > 1 {
			t.Errorf("the collectors got %d and %d, not in turn", na, nb)
		}
		a.SetMob("")
		b.SetMob("")
		for range 600 {
			dimension.TickCollectors(s.Session, tx)
		}
		if p := dimension.Pool(s.Session, slime); p > 64 {
			t.Errorf("the pool holds %v with no collector, want at most 64", p)
		}
	})
}

// TestRelicTable: after the research the relic table makes a relic for its
// materials and raises it a grade for more of them and essences of any
// monster.
func TestRelicTable(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		give := func(it world.Item, n int) { _, _ = p.Inventory().AddItem(item.NewStack(it, n)) }
		give(item.IronIngot{}, 12)
		give(item.RedstoneWire{}, 4)
		dimension.CraftRelic(s.Session, p, tx, sgitem.RelicMagnet)
		if _, ok := dimension.Carried(p, sgitem.RelicMagnet); ok {
			t.Fatal("a relic was made before the research")
		}
		if progress.StateOf(s.Session).Researched == nil {
			progress.StateOf(s.Session).Researched = map[string]bool{}
		}
		progress.StateOf(s.Session).Researched[dimension.RelicsResearch] = true
		dimension.CraftRelic(s.Session, p, tx, sgitem.RelicMagnet)
		if g, ok := dimension.Carried(p, sgitem.RelicMagnet); !ok || g != 1 {
			t.Fatalf("made grade %d (%v), want 1", g, ok)
		}
		if n := core.CountPlain(p.Inventory(), item.IronIngot{}); n != 4 {
			t.Errorf("%d iron left, want 4", n)
		}
		// grade 1 → 2: the materials once more and 4 essences
		give(item.IronIngot{}, 4)
		give(item.RedstoneWire{}, 4)
		give(sgitem.Essence{Mob: sgitem.MobSlime}, 3)
		dimension.CraftRelic(s.Session, p, tx, sgitem.RelicMagnet)
		if g, _ := dimension.Carried(p, sgitem.RelicMagnet); g != 1 {
			t.Fatal("raised without enough essences")
		}
		give(sgitem.Essence{Mob: sgitem.MobZombie}, 1)
		dimension.CraftRelic(s.Session, p, tx, sgitem.RelicMagnet)
		if g, _ := dimension.Carried(p, sgitem.RelicMagnet); g != 2 {
			t.Fatalf("grade %d after raising, want 2", g)
		}
		if n := core.CountPlain(p.Inventory(), sgitem.Essence{Mob: sgitem.MobSlime}) + core.CountPlain(p.Inventory(), sgitem.Essence{Mob: sgitem.MobZombie}); n != 0 {
			t.Errorf("%d essences left, want 0", n)
		}
	})
}

// TestLoadoutAndLevels: a hunter brings its best relics (two weapons, three
// charms) unless it picked others; a life charm raises its health for the run
// only; gems level it up, which stops the hunt until it picks an upgrade.
func TestLoadoutAndLevels(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		for _, k := range []string{sgitem.RelicBlade, sgitem.RelicCrossbow, sgitem.RelicFlameRing, sgitem.RelicLife} {
			_, _ = p.Inventory().AddItem(sgitem.RelicStack(k, 1))
		}
		_, _ = p.Inventory().AddItem(sgitem.RelicStack(sgitem.RelicLife, 3))
		got := dimension.Loadout(s.Session, p)
		want := []string{sgitem.RelicBlade, sgitem.RelicCrossbow, sgitem.RelicLife}
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("loadout %v, want %v", got, want)
		}
		dimension.ToggleLoadout(s.Session, p, sgitem.RelicCrossbow)
		dimension.ToggleLoadout(s.Session, p, sgitem.RelicFlameRing)
		want = []string{sgitem.RelicBlade, sgitem.RelicLife, sgitem.RelicFlameRing}
		if got := dimension.Loadout(s.Session, p); fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("loadout %v after the changes, want %v", got, want)
		}

		dimension.Begin(s.Session, tx, p, sgitem.MobSlime)
		// a grade 3 life charm: 4 + 2 + 2
		if p.MaxHealth() != 28 {
			t.Errorf("max health %v in the run, want 28", p.MaxHealth())
		}
		for range 101 {
			runTick(t, s, tx)
		}
		r := dimension.RunOf(s.Session)
		dimension.GiveGems(r, tx, p, 4) // 8 experience: level 1
		if !r.Paused() {
			t.Fatal("a level did not stop the hunt")
		}
		ticks := r.Ticks()
		runTick(t, s, tx)
		if r.Ticks() != ticks {
			t.Error("the hunt went on while paused")
		}
		dimension.ChooseFirst(r, tx, p)
		if r.Paused() {
			t.Error("still paused after the choice")
		}
		dimension.GiveGems(r, tx, p, 10) // two more levels: the second waits for the first choice
		dimension.ChooseFirst(r, tx, p)
		if !r.Paused() {
			t.Error("the second level of a batch of gems did not stop the hunt")
		}
		dimension.ChooseFirst(r, tx, p)
		dimension.Died(s.Session, tx, p)
		if p.MaxHealth() != 20 || p.ExperienceLevel() != 0 {
			t.Errorf("after the run max health %v, level %d, want 20 and 0", p.MaxHealth(), p.ExperienceLevel())
		}
	})
}
