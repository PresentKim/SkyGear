package dragonfactory

import (
	"fmt"
	"github.com/presentkim/dragonfactory/internal/i18n"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
)

// Tree safety nets of the start (docs/design/02-dragonfactory.md §2): /묘목
// gives an oak sapling once every saplingCooldown (per save, real time), and
// crouching again and again makes nearby saplings grow. Drafts.
const (
	saplingCooldown = 10 * time.Minute
	// crouchGrowReach is the horizontal reach of crouch growth (blocks);
	// crouchGrowChance the percent chance per sapling and crouch to be
	// bone mealed.
	crouchGrowReach  = 2
	crouchGrowChance = 30
)

// giveSapling gives p an oak sapling if the cooldown has passed.
func (s *session) giveSapling(p *player.Player, tx *world.Tx, now time.Time) error {
	if wait := s.state.LastSapling.Add(saplingCooldown).Sub(now); wait > 0 {
		return fmt.Errorf(i18n.T(i18n.Of(p), "dragonfactory.sapling.giveSapling.1"), int(wait.Minutes()), int(wait.Seconds())%60)
	}
	s.state.LastSapling = now
	count(p, "sapling_cmd", 1)
	s.storeState()
	st := item.NewStack(block.Sapling{Type: block.OakSapling()}, 1)
	if added, _ := p.Inventory().AddItem(st); added < 1 {
		s.dropAt(tx, p.Position(), st)
	}
	return nil
}

// crouchGrow bone meals saplings around p by chance (crouching again and
// again grows trees).
func (s *session) crouchGrow(p *player.Player, tx *world.Tx) {
	feet := cube.PosFromVec3(p.Position())
	for x := -crouchGrowReach; x <= crouchGrowReach; x++ {
		for z := -crouchGrowReach; z <= crouchGrowReach; z++ {
			for y := -1; y <= 2; y++ {
				pos := feet.Add(cube.Pos{x, y, z})
				sp, ok := tx.Block(pos).(block.Sapling)
				if !ok || s.rand.IntN(100) >= crouchGrowChance {
					continue
				}
				sp.BoneMeal(pos, tx)
				tx.AddParticle(pos.Vec3Centre(), particle.BoneMeal{})
				count(p, "sapling_grow", 1)
				s.advance(tx, advGrowTree)
			}
		}
	}
}

// saplingCmd is /묘목 (and /sapling).
type saplingCmd struct{ g *Game }

func (c saplingCmd) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, ok := src.(*player.Player)
	if !ok {
		return
	}
	s, ok := c.g.sessionOf(tx.World())
	if !ok {
		o.Error(i18n.T(i18n.Of(p), "dragonfactory.sapling.Run.1"))
		return
	}
	if err := s.giveSapling(p, tx, time.Now()); err != nil {
		o.Errorf(i18n.T(i18n.Of(p), "dragonfactory.sapling.Run.2"), err.Error())
		return
	}
	o.Printf("%s", i18n.T(i18n.Of(p), "dragonfactory.sapling.Run.3"))
}
