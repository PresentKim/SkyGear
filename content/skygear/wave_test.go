package skygear

import (
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/content/skygear/dimension"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// waveMonsters returns the live monsters of the wave.
func waveMonsters(s *session, tx *world.Tx) []*dimension.Monster {
	var out []*dimension.Monster
	if dimension.LocalOf(s.Session).Wave == nil {
		return out
	}
	for h := range dimension.LocalOf(s.Session).Wave.Left {
		if e, ok := h.Entity(tx); ok {
			out = append(out, e.(*dimension.Monster))
		}
	}
	return out
}

// TestWaveFlow: using the gate spawns the level's monsters, they walk to the
// player and hit, a player's attack hurts them, and killing them all clears the
// level, opens the next and gives the first-clear reward.
func TestWaveFlow(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	gate := machinePos
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		// solid ground around the gate
		for x := -8; x <= 8; x++ {
			for z := -8; z <= 8; z++ {
				tx.SetBlock(gate.Add(cube.Pos{x, -1, z}), block.Stone{}, nil)
			}
		}
		placeTestMachine(s, gate, machine.KindGate, "")
		p.Teleport(gate.Vec3Centre().Add(mgl64.Vec3{3, 0, 0}))
		if dimension.Start(s.Session, tx, gate, p.Position(), 2) {
			t.Fatal("level 2 started before level 1 was cleared")
		}
		if !dimension.Start(s.Session, tx, gate, p.Position(), 1) {
			t.Fatal("level 1 did not start")
		}
		if dimension.Start(s.Session, tx, gate, p.Position(), 1) {
			t.Error("two waves at once")
		}
		if dimension.LocalOf(s.Session).Wave.Portal == nil || len(dimension.LocalOf(s.Session).Wave.Left) != 0 || len(dimension.LocalOf(s.Session).Wave.Pending) != 4 {
			t.Fatalf("the gate opened with %d monsters out and %d to come", len(dimension.LocalOf(s.Session).Wave.Left), len(dimension.LocalOf(s.Session).Wave.Pending))
		}
		if dimension.LocalOf(s.Session).Wave.Out != (mgl64.Vec3{1, 0, 0}) {
			t.Errorf("the gate faces %v, want east to the player", dimension.LocalOf(s.Session).Wave.Out)
		}
		dimension.Tick(s.Session, tx)
		if dimension.LocalOf(s.Session).Wave == nil {
			t.Fatal("the wave ended before its monsters came out")
		}
		for range 4 * dimension.GatePortalEvery {
			dimension.EmergeMonsters(s.Session, tx)
		}
		mons := waveMonsters(s, tx)
		if len(mons) != 4 {
			t.Fatalf("%d monsters, want 4", len(mons))
		}
		// they walk towards the player
		before := mons[0].Position().Sub(p.Position()).Len()
		for range 40 {
			for _, m := range mons {
				m.Tick(tx, 0)
			}
		}
		if after := mons[0].Position().Sub(p.Position()).Len(); after >= before {
			t.Errorf("a monster did not come closer: %.2f then %.2f", before, after)
		}
		// hurt, then die
		p.Teleport(mons[0].Position())
		mons[0].Hurt(tx, 5, p)
		if mons[0].Health() != dimension.MonsterTable[sgitem.MobZombie].Health-5 {
			t.Errorf("health %v after a hit of 5", mons[0].Health())
		}
		for _, m := range mons {
			m.Hurt(tx, 1000, p)
		}
		if dimension.LocalOf(s.Session).Wave != nil {
			t.Fatal("the wave goes on with every monster dead")
		}
		if dimension.StateOf(s.Session).GateTier != 1 || !dimension.CanStart(s.Session, 2) {
			t.Errorf("tier %d, level 2 open %v", dimension.StateOf(s.Session).GateTier, dimension.CanStart(s.Session, 2))
		}
		if !progress.Advanced(s.Session, advancement.Wave1) {
			t.Error("clearing level 1 did not do its advancement")
		}
	})
}

// TestWaveFailsOnTimeout: the monsters vanish and the wave ends when the time is out.
func TestWaveFailsOnTimeout(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	gate := machinePos
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		for x := -8; x <= 8; x++ {
			for z := -8; z <= 8; z++ {
				tx.SetBlock(gate.Add(cube.Pos{x, -1, z}), block.Stone{}, nil)
			}
		}
		placeTestMachine(s, gate, machine.KindGate, "")
		dimension.Start(s.Session, tx, gate, p.Position(), 1)
		dimension.LocalOf(s.Session).Wave.Deadline = dimension.LocalOf(s.Session).Wave.Deadline.Add(-1000 * 1e9)
		dimension.Tick(s.Session, tx)
		if dimension.LocalOf(s.Session).Wave != nil {
			t.Error("the wave outlived its time")
		}
		if dimension.StateOf(s.Session).GateTier != 0 {
			t.Error("a failed wave opened a level")
		}
	})
}

// TestMonsterReturnsFromVoid: a monster that falls into the void comes out of
// the gate again with the health it had: it stays in the wave, which goes on.
func TestMonsterReturnsFromVoid(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	gate := machinePos
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		for x := -8; x <= 8; x++ {
			for z := -8; z <= 8; z++ {
				tx.SetBlock(gate.Add(cube.Pos{x, -1, z}), block.Stone{}, nil)
			}
		}
		placeTestMachine(s, gate, machine.KindGate, "")
		if !dimension.Start(s.Session, tx, gate, p.Position(), 1) {
			t.Fatal("level 1 did not start")
		}
		dimension.EmergeMonsters(s.Session, tx)
		mons := waveMonsters(s, tx)
		if len(mons) != 1 || len(dimension.LocalOf(s.Session).Wave.Pending) == 0 {
			t.Fatalf("%d monsters out, %d to come", len(mons), len(dimension.LocalOf(s.Session).Wave.Pending))
		}
		m := mons[0]
		m.Hurt(tx, 3, p)
		health := m.Health()
		m.Data.Pos = mgl64.Vec3{m.Data.Pos[0], island.IslandY - dimension.MonsterVoid - 1, m.Data.Pos[2]}
		m.Tick(tx, 0)
		if dimension.LocalOf(s.Session).Wave == nil || !dimension.LocalOf(s.Session).Wave.Owns(m.Handle) || m.Handle.Closed() {
			t.Fatal("a monster in the void left the wave")
		}
		if m.Health() != health {
			t.Errorf("health %v after the void, want %v", m.Health(), health)
		}
		if d := m.Position().Sub(gate.Vec3Middle()); d.Len() > 2.5 {
			t.Errorf("the monster is %.1f blocks from the gate, want at its way out", d.Len())
		}
		if m.Md().ExitTicks != dimension.GateExitTicks {
			t.Error("the monster does not walk out of the gate again")
		}
	})
}

func TestWaveLevelsAreValid(t *testing.T) {
	for i, def := range dimension.Levels {
		if len(def.Groups) == 0 || def.Seconds <= 0 || len(def.Reward) == 0 {
			t.Errorf("level %d is incomplete", i+1)
		}
		for _, g := range def.Groups {
			if _, ok := dimension.MonsterTable[g.Mob]; !ok || g.N <= 0 {
				t.Errorf("level %d: %+v", i+1, g)
			}
		}
	}
	for _, mob := range sgitem.MobKinds {
		if _, ok := dimension.MobTypes[mob]; !ok {
			t.Errorf("no entity type for %s", mob)
		}
	}
}

// TestSkeletonShoots: a skeleton with a clear view of a player 8 blocks away stops
// to shoot an arrow instead of walking up to melee range.
func TestSkeletonShoots(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	gate := machinePos
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		for x := -12; x <= 12; x++ {
			for z := -12; z <= 12; z++ {
				tx.SetBlock(gate.Add(cube.Pos{x, -1, z}), block.Stone{}, nil)
			}
		}
		placeTestMachine(s, gate, machine.KindGate, "")
		dimension.LocalOf(s.Session).Wave = &dimension.State{Level: 1, Gate: gate, Left: map[*world.EntityHandle]string{}, Deadline: time.Now().Add(time.Hour)}
		at := gate.Vec3Centre().Add(mgl64.Vec3{0, 0.5, 0})
		p.Teleport(at.Add(mgl64.Vec3{8, 0, 0}))
		hd := world.EntitySpawnOpts{Position: at}.New(dimension.MobTypes[sgitem.MobSkeleton], dimension.MonsterConfig{Mob: sgitem.MobSkeleton})
		dimension.LocalOf(s.Session).Wave.Left[hd] = sgitem.MobSkeleton
		e := tx.AddEntity(hd)
		m := e.(*dimension.Monster)
		for range 60 {
			m.Tick(tx, 0)
		}
		arrows := 0
		for ent := range tx.EntitiesWithin(cube.Box(-40, -10, -40, 40, 100, 40).Translate(at)) {
			if ent.H().Type() == entity.ArrowType {
				arrows++
			}
		}
		if arrows == 0 {
			t.Error("the skeleton shot no arrow")
		}
		if dist := m.Position().Sub(p.Position()).Len(); dist < dimension.MonsterTable[sgitem.MobSkeleton].HoldRange-1 {
			t.Errorf("the skeleton walked up to %.1f blocks", dist)
		}
	})
}
