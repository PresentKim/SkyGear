package farm

import (
	"fmt"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/progress"
)

// Tree safety nets of the start: /묘목
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

// GiveSapling gives p an oak sapling if the cooldown has passed.
func GiveSapling(s *core.Session, p *player.Player, tx *world.Tx, now time.Time) error {
	if wait := StateOf(s).LastSapling.Add(saplingCooldown).Sub(now); wait > 0 {
		return fmt.Errorf(i18n.T(i18n.Of(p), "sapling.wait"), int(wait.Minutes()), int(wait.Seconds())%60)
	}
	StateOf(s).LastSapling = now
	achievement.Count(p, "sapling_cmd", 1)
	s.MarkDirty()
	st := item.NewStack(block.Sapling{Type: block.OakSapling()}, 1)
	if added, _ := p.Inventory().AddItem(st); added < 1 {
		s.DropAt(tx, p.Position(), st)
	}
	return nil
}

// CrouchGrow bone meals saplings around p by chance (crouching again and
// again grows trees).
func CrouchGrow(s *core.Session, p *player.Player, tx *world.Tx) {
	feet := cube.PosFromVec3(p.Position())
	for x := -crouchGrowReach; x <= crouchGrowReach; x++ {
		for z := -crouchGrowReach; z <= crouchGrowReach; z++ {
			for y := -1; y <= 2; y++ {
				pos := feet.Add(cube.Pos{x, y, z})
				sp, ok := tx.Block(pos).(block.Sapling)
				if !ok || s.Rand.IntN(100) >= crouchGrowChance {
					continue
				}
				sp.BoneMeal(pos, tx)
				tx.AddParticle(pos.Vec3Centre(), particle.BoneMeal{})
				achievement.Count(p, "sapling_grow", 1)
				progress.Advance(s, tx, advancement.GrowTree)
			}
		}
	}
}
