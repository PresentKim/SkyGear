package dragonfactory

import (
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/dfitem"
)

// Hand-made resources of the start (docs/design/02-dragonfactory.md §3,
// 05-sky-progression.md §3). Chances are drafts.
//   - A stick used on grass or dirt always drops a pebble; four pebbles
//     make the stone they come from (research "pebbles").
//   - Stripping a log with a stone or better axe sometimes drops one or two
//     bark fibers; four fibers make string (research "string").
//   - Dust is sand crushed once more, sifted for its own drops.

const (
	// stickCooldown is the shortest time between two stick uses on the
	// ground (the client sends one click twice).
	stickCooldown = 100 * time.Millisecond
	// fiberChance is the percent chance stripping a log drops fibers.
	fiberChance = 50
)

// grassSeedChance is the percent chance tilling dirt drops grass seeds.
const grassSeedChance = 10

// tillSeeds drops grass seeds grassSeedChance of the time when p tills dirt
// at pos with a hoe (the hoe itself turns it into farmland afterwards).
func (s *session) tillSeeds(p *player.Player, tx *world.Tx, pos cube.Pos, face cube.Face, held item.Stack) {
	if _, hoe := held.Item().(item.Hoe); !hoe || face == cube.FaceDown {
		return
	}
	if d, ok := tx.Block(pos).(block.Dirt); !ok || d.Coarse {
		return
	}
	if _, air := tx.Block(pos.Side(cube.FaceUp)).(block.Air); !air {
		return
	}
	if s.rand.IntN(100) < grassSeedChance {
		s.dropAt(tx, pos.Side(cube.FaceUp).Vec3Centre(), item.NewStack(dfitem.GrassSeeds{}, 1))
	}
}

// rollPebble picks a pebble kind by weight.
func (s *session) rollPebble() dfitem.Pebble {
	total := 0
	for _, k := range dfitem.PebbleKinds {
		total += k.Weight
	}
	n := s.rand.IntN(total)
	for _, k := range dfitem.PebbleKinds {
		if n < k.Weight {
			return dfitem.Pebble{Kind: k.Kind}
		}
		n -= k.Weight
	}
	return dfitem.Pebble{Kind: dfitem.PebbleStone}
}

// stickDebounced reports whether p used a stick less than stickCooldown
// ago, and otherwise records this use.
func (s *session) stickDebounced(p *player.Player) bool {
	if s.lastStick == nil {
		s.lastStick = map[*world.EntityHandle]time.Time{}
	}
	if t, ok := s.lastStick[p.H()]; ok && time.Since(t) < stickCooldown {
		return true
	}
	s.lastStick[p.H()] = time.Now()
	return false
}

// useStick knocks a pebble out of grass or dirt. It reports
// whether the use was a stick on grass or dirt.
func (s *session) useStick(p *player.Player, tx *world.Tx, pos cube.Pos, face cube.Face, held item.Stack) bool {
	if _, ok := held.Item().(item.Stick); !ok {
		return false
	}
	switch tx.Block(pos).(type) {
	case block.Grass, block.Dirt:
	default:
		if !s.stickDebounced(p) {
			count(p, "stick_other", 1)
		}
		return false
	}
	if s.stickDebounced(p) {
		return true
	}
	if _, dirt := tx.Block(pos).(block.Dirt); dirt {
		count(p, "stick_dirt", 1)
	} else {
		count(p, "stick_grass", 1)
	}
	// Every use knocks a pebble out, with dirt chips and the small
	// breaking sound of the block.
	b := tx.Block(pos)
	tx.AddParticle(pos.Side(face).Vec3Centre(), particle.PunchBlock{Block: b, Face: face})
	tx.AddParticle(pos.Vec3Centre(), particle.BlockBreak{Block: b})
	tx.PlaySound(pos.Vec3Centre(), sound.BlockBreaking{Block: b})
	pebble := s.rollPebble()
	count(p, "pebble_"+pebble.Kind, 1)
	s.dropAt(tx, pos.Side(face).Vec3Centre(), item.NewStack(pebble, 1))
	return true
}

// stripFiber drops bark fibers by chance when a stone or better axe strips
// a log (Dragonfly strips it after this).
func (s *session) stripFiber(p *player.Player, tx *world.Tx, pos cube.Pos, face cube.Face, held item.Stack) {
	axe, ok := held.Item().(item.Axe)
	if !ok {
		return
	}
	if l, ok := tx.Block(pos).(block.Log); !ok || l.Stripped {
		return
	}
	count(p, "strip_log", 1)
	if axe.Tier.HarvestLevel < item.ToolTierStone.HarvestLevel {
		return
	}
	if s.rand.IntN(100) < fiberChance {
		s.dropAt(tx, pos.Side(face).Vec3Centre(), item.NewStack(dfitem.BarkFiber{}, 1+s.rand.IntN(2)))
	}
}

func init() {
	for _, k := range dfitem.PebbleKinds {
		world.RegisterItem(dfitem.Pebble{Kind: k.Kind})
	}
	world.RegisterItem(dfitem.BarkFiber{})
	world.RegisterItem(dfitem.GrassSeeds{})
	world.RegisterBlock(dfitem.Dust{})
	world.RegisterItem(dfitem.Dust{})
	dfitem.HandworkRecipes()
}
