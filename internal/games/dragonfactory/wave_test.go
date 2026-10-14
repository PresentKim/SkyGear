package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/entity"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// monsterAt returns the live monsters of the wave.
func waveMonsters(s *session, tx *world.Tx) []*Monster {
	var out []*Monster
	if s.wave == nil {
		return out
	}
	for h := range s.wave.left {
		if e, ok := h.Entity(tx); ok {
			out = append(out, e.(*Monster))
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
		placeTestMachine(s, gate, KindGate, "")
		p.Teleport(gate.Vec3Centre().Add(mgl64.Vec3{3, 0, 0}))
		if s.startWave(tx, gate, 2) {
			t.Fatal("level 2 started before level 1 was cleared")
		}
		if !s.startWave(tx, gate, 1) {
			t.Fatal("level 1 did not start")
		}
		if s.startWave(tx, gate, 1) {
			t.Error("two waves at once")
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
		if mons[0].Health() != monsterTable[MobZombie].health-5 {
			t.Errorf("health %v after a hit of 5", mons[0].Health())
		}
		for _, m := range mons {
			m.Hurt(tx, 1000, p)
		}
		if s.wave != nil {
			t.Fatal("the wave goes on with every monster dead")
		}
		if s.gateTier() != 1 || !s.canStartWave(2) {
			t.Errorf("tier %d, level 2 open %v", s.gateTier(), s.canStartWave(2))
		}
		if !s.advanced(advWave1) {
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
		placeTestMachine(s, gate, KindGate, "")
		s.startWave(tx, gate, 1)
		s.wave.deadline = s.wave.deadline.Add(-1000 * 1e9)
		s.tickWave(tx)
		if s.wave != nil {
			t.Error("the wave outlived its time")
		}
		if s.gateTier() != 0 {
			t.Error("a failed wave opened a level")
		}
	})
}

func TestWaveLevelsAreValid(t *testing.T) {
	for i, def := range waveLevels {
		if len(def.Groups) == 0 || def.Seconds <= 0 || len(def.Reward) == 0 {
			t.Errorf("level %d is incomplete", i+1)
		}
		for _, g := range def.Groups {
			if _, ok := monsterTable[g.Mob]; !ok || g.N <= 0 {
				t.Errorf("level %d: %+v", i+1, g)
			}
		}
	}
	for _, mob := range mobKinds {
		if _, ok := monsterTypes[mob]; !ok {
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
		placeTestMachine(s, gate, KindGate, "")
		s.wave = &waveState{level: 1, gate: gate, left: map[*world.EntityHandle]string{}, deadline: time.Now().Add(time.Hour)}
		at := gate.Vec3Centre().Add(mgl64.Vec3{0, 0.5, 0})
		p.Teleport(at.Add(mgl64.Vec3{8, 0, 0}))
		hd := world.EntitySpawnOpts{Position: at}.New(monsterTypes[MobSkeleton], monsterConfig{mob: MobSkeleton})
		s.wave.left[hd] = MobSkeleton
		e := tx.AddEntity(hd)
		m := e.(*Monster)
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
		if dist := m.Position().Sub(p.Position()).Len(); dist < monsterTable[MobSkeleton].holdRange-1 {
			t.Errorf("the skeleton walked up to %.1f blocks", dist)
		}
	})
}
