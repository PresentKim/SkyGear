package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/dimension"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/platform/account"
	"github.com/presentkim/skygear/platform/instance"
)

// A hunter that takes a hit it does not survive dies for real: the death ends
// the run and the respawn brings the hunter back. These tests kill through
// Dragonfly (Player.Hurt). The respawn itself needs a connected client
// (Player.Respawn does nothing without one), so the tests call the respawn
// handler as Dragonfly calls it.

// keepsInventory is the handler of a player in an instance as far as a death
// goes: the platform keeps the inventory for every death
// (platform/instance.Handler.HandleDeath) before the game's handler runs.
type keepsInventory struct{ player.Handler }

func (h keepsInventory) HandleDeath(p *player.Player, src world.DamageSource, keepInv *bool) {
	*keepInv = true
	h.Handler.HandleDeath(p, src, keepInv)
}

// fallenHunter is a session whose player carries three stone and seven
// levels, is in the second dimension and has the handler of an instance.
func fallenHunter(t *testing.T) (*session, *world.EntityHandle, string, mgl64.Vec3) {
	t.Helper()
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
	var (
		acc  string
		from mgl64.Vec3
	)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		acc = instance.AccountID(p)
		if _, err := d.TouchAccount(acc, "", p.Name()); err != nil {
			t.Fatal(err)
		}
		p.Handle(keepsInventory{s.Handler(p)})
		_ = p.Inventory().SetItem(0, item.NewStack(block.Stone{}, 3))
		p.SetExperienceLevel(7)
		from = p.Position()
		if !dimension.Begin(s.Session, tx, p, sgitem.MobSlime) {
			t.Fatal("the run did not begin")
		}
	})
	return s, h, acc, from
}

// TestHunterDiesAndComesBack: a hit the hunter survives is a hurt; the hit
// that kills it is a hurt and a death. The death ends the run, once: the
// hunter has what it carried and its levels again, nothing lies on the
// ground, the lost run is counted once. The respawn puts the hunter where it
// went in from.
func TestHunterDiesAndComesBack(t *testing.T) {
	s, h, acc, from := fallenHunter(t)
	count := func(key string) int64 { return account.Counter(acc, key) }
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		if _, hurt := p.Hurt(1, entity.AttackDamageSource{}); !hurt || p.Dead() || !dimension.Hunting(s.Session, p) {
			t.Fatal("a light hit did not land, or ended the run")
		}
		if got := count("hurt"); got != 1 {
			t.Errorf("hurt counted %d after a hit the hunter took, want 1", got)
		}
		if _, hurt := p.Hurt(1000, entity.AttackDamageSource{}); !hurt {
			t.Fatal("the hit that kills the hunter did not land")
		}
		if !p.Dead() {
			t.Fatal("the hunter did not die")
		}
		if dimension.Hunting(s.Session, p) || dimension.RunOf(s.Session) != nil {
			t.Fatal("the run did not end with the hunter's death")
		}
		if got, want := [3]int64{count("hurt"), count("death"), count("wave_fail")}, [3]int64{2, 1, 1}; got != want {
			t.Errorf("hurt, death, wave_fail counted %v, want %v", got, want)
		}
		if st, _ := p.Inventory().Item(0); !st.Equal(item.NewStack(block.Stone{}, 3)) {
			t.Errorf("slot 0 holds %v after the death, want the three stone the hunter carried", st)
		}
		for _, st := range p.Armour().Slots() {
			if !st.Empty() {
				t.Errorf("the dead hunter still wears %v of the kit", st)
			}
		}
		if n := len(droppedItems(tx)); n != 0 {
			t.Errorf("%d kinds of items lie on the ground after the death: %v", n, droppedItems(tx))
		}
		if p.ExperienceLevel() != 7 {
			t.Errorf("the hunter has %d levels after the death, want 7", p.ExperienceLevel())
		}
		if n := len(p.Effects()); n != 0 {
			t.Errorf("the dead hunter has %d effects", n)
		}
		if !core.InArena(p.Position()) {
			t.Error("the dead hunter left the second dimension before it respawned")
		}

		// The respawn: Dragonfly asks the handler where the player goes.
		pos, w := mgl64.Vec3{}, tx.World()
		p.Handler().HandleRespawn(p, &pos, &w)
		if pos.Sub(from).Len() > 1e-6 {
			t.Errorf("the hunter respawns at %v, want where it went in from, %v", pos, from)
		}
		// A second respawn is an island's: the handler leaves the place alone.
		island := mgl64.Vec3{1, 2, 3}
		pos = island
		p.Handler().HandleRespawn(p, &pos, &w)
		if pos != island {
			t.Errorf("a second respawn was sent to %v", pos)
		}
		// Nothing ends the run a second time.
		dimension.Died(s.Session, tx, p)
		s.Leave(p)
		if got := count("wave_fail"); got != 1 {
			t.Errorf("wave_fail counted %d for one lost run, want 1", got)
		}
	})
}

// TestHitThatKillsLeavesNoEffect: a monster whose hit slows the hunter leaves
// no slowness on a hunter its hit killed, which a respawn would bring home.
func TestHitThatKillsLeavesNoEffect(t *testing.T) {
	s, h, _, _ := fallenHunter(t)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		r := dimension.RunOf(s.Session)
		p.Armour().Clear()
		p.SetMaxHealth(1)
		m := world.EntitySpawnOpts{Position: p.Position().Add(mgl64.Vec3{1, 0, 0})}.New(dimension.VariantType(sgitem.MobSlime, dimension.VariantElite),
			dimension.MonsterConfig{Mob: sgitem.MobSlime, Variant: dimension.VariantElite})
		r.Alive[m] = true
		r.Spawned++
		tx.AddEntity(m)
		e, _ := m.Entity(tx)
		for range 200 {
			if p.Dead() {
				break
			}
			e.(*dimension.Monster).Tick(tx, 0)
		}
		if !p.Dead() {
			t.Fatal("the monster did not kill the hunter")
		}
		if n := len(p.Effects()); n != 0 {
			t.Errorf("the dead hunter has %d effects: %v", n, p.Effects())
		}
	})
}

// TestDeadHunterWhoLeavesIsBackOnTheIsland: a hunter that disconnects between
// its death and its respawn is respawned by Dragonfly first and then leaves.
// It is sent where it went in from, the lost run is counted once, and it
// holds what it carried: nothing is left for the next join to give back.
func TestDeadHunterWhoLeavesIsBackOnTheIsland(t *testing.T) {
	s, h, acc, from := fallenHunter(t)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		p.Hurt(1000, entity.AttackDamageSource{})
		if !p.Dead() {
			t.Fatal("the hunter did not die")
		}
		pos, w := mgl64.Vec3{}, tx.World()
		p.Handler().HandleRespawn(p, &pos, &w)
		s.Leave(p)
		if pos.Sub(from).Len() > 1e-6 {
			t.Errorf("the leaving hunter is sent to %v, want %v", pos, from)
		}
		if got := account.Counter(acc, "wave_fail"); got != 1 {
			t.Errorf("wave_fail counted %d, want 1", got)
		}
		// The next join finds nothing put away: the inventory stays as it is.
		before := p.Inventory().Slots()
		s.Join(p, false)
		for slot, st := range p.Inventory().Slots() {
			if !st.Equal(before[slot]) {
				t.Errorf("slot %d holds %v after the next join, was %v", slot, st, before[slot])
			}
		}
		if st, _ := p.Inventory().Item(0); !st.Equal(item.NewStack(block.Stone{}, 3)) {
			t.Errorf("slot 0 holds %v, want the three stone the hunter carried", st)
		}
	})
}

// TestDeathOnTheIslandIsUntouched: a player who is not in the second
// dimension dies as before: the death is counted, the respawn handler leaves
// the place to the platform, and no run is involved.
func TestDeathOnTheIslandIsUntouched(t *testing.T) {
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
		p.Handle(keepsInventory{s.Handler(p)})
		p.Hurt(1000, entity.AttackDamageSource{})
		if !p.Dead() {
			t.Fatal("the player did not die")
		}
		if got, want := [3]int64{account.Counter(acc, "hurt"), account.Counter(acc, "death"), account.Counter(acc, "wave_fail")}, [3]int64{1, 1, 0}; got != want {
			t.Errorf("hurt, death, wave_fail counted %v, want %v", got, want)
		}
		spawn := mgl64.Vec3{1, 2, 3}
		pos, w := spawn, tx.World()
		p.Handler().HandleRespawn(p, &pos, &w)
		if pos != spawn {
			t.Errorf("the island death respawns at %v, want the platform's place %v", pos, spawn)
		}
	})
}

// TestBlastThatKillsDoesNotThrowTheHunter: the blast that kills the hunter
// does not throw it: a dead player takes no knock-back, so none follows the
// hunter to where it respawns.
func TestBlastThatKillsDoesNotThrowTheHunter(t *testing.T) {
	s, h, _, _ := fallenHunter(t)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		r := dimension.RunOf(s.Session)
		p.Armour().Clear()
		p.SetMaxHealth(1)
		c := world.EntitySpawnOpts{Position: p.Position().Add(mgl64.Vec3{1, 0, 0})}.New(dimension.VariantType(sgitem.MobCreeper, dimension.VariantBase),
			dimension.MonsterConfig{Mob: sgitem.MobCreeper})
		r.Alive[c] = true
		r.Spawned++
		tx.AddEntity(c)
		e, _ := c.Entity(tx)
		for range 40 {
			if dimension.RunOf(s.Session) == nil {
				break
			}
			e.(*dimension.Monster).Tick(tx, 0)
		}
		if dimension.RunOf(s.Session) != nil {
			t.Fatal("the blast did not end the run")
		}
		if v := p.Velocity(); v.Len() != 0 {
			t.Errorf("the blast that felled the hunter threw it: velocity %v", v)
		}
		if !p.Dead() {
			t.Error("the hunter did not die of the blast")
		}
	})
}
