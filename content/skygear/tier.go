package skygear

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// Machine tiers: a machine is made faster
// and larger with kits that are made from alloys and control circuits. The
// tier of a machine is saved with it.

const maxTier = 3

// tierSpec are the numbers of a tier. A machine of speed n runs its work n
// times a tick and spends use10/10 times the energy of a basic one in all, so
// a better machine spends less energy on the same work. A generator makes gen
// times the energy a tick and burns its fuel burn times as fast; an energy
// cell holds cell times as much; the buffer of the others grows by buffer.
type tierSpec struct {
	speed, use10, buffer, gen, burn, cell int
}

var tierSpecs = [maxTier + 1]tierSpec{
	{1, 10, 1, 1, 1, 1},
	{2, 16, 2, 2, 1, 4},
	{4, 30, 4, 4, 2, 16},
	{8, 55, 8, 8, 3, 64},
}

// tierKinds are the machines that take kits.
var tierKinds = map[string]bool{
	KindCrusher: true, KindFurnace: true, KindWasher: true, KindCrucible: true, KindAlloy: true,
	KindSimulator: true, KindLab: true, KindCrafter: true,
	KindAutoSieve: true, KindMagicSieve: true, KindMiner: true,
	KindHarvester: true, KindVacuum: true, KindFeller: true,
	KindGenerator: true, KindLavaGenerator: true, KindEnergyCell: true,
}

// tierRanges are the reaches of the machines that gain range with their tier
// (blocks each way sideways).
var tierRanges = map[string][maxTier + 1]int{
	KindHarvester: {harvestRadius, 6, 8, 12},
	KindVacuum:    {vacuumReach, 9, 12, 16},
	KindFeller:    {fellerReach, 7, 9, 12},
}

// tierable reports whether the machines of a kind take kits.
func tierable(kind string) bool { return tierKinds[kind] }

func (st *machineState) spec() tierSpec { return tierSpecs[st.tier] }

// reps is how many times a tick the machine runs its work; generators and
// cells do not run faster, they make more.
func (st *machineState) reps() int {
	if st.kind.generates > 0 || st.kind.cell {
		return 1
	}
	return st.spec().speed
}

// energyCap is the RE the machine's buffer holds.
func (st *machineState) energyCap() int {
	k := st.kind
	switch {
	case k.cell:
		return k.energyCap * st.spec().cell
	case k.generates > 0:
		return k.energyCap * st.spec().gen
	}
	return k.energyCap * st.spec().buffer
}

// useCost is the RE one run of the work takes (rounded up).
func (st *machineState) useCost() int {
	if st.kind.use == 0 {
		return 0
	}
	per := 10 * st.reps()
	return (st.kind.use*st.spec().use10 + per - 1) / per
}

// useTotal is the RE a tick of work takes.
func (st *machineState) useTotal() int { return st.useCost() * st.reps() }

// genRate is the RE a generator makes a tick, burnRate how fast it burns.
func (st *machineState) genRate() int  { return st.kind.generates * st.spec().gen }
func (st *machineState) burnRate() int { return st.spec().burn }

// reach is the reach of a machine with a range: sideways, in blocks.
func (st *machineState) reach(kind string) int {
	if r, ok := tierRanges[kind]; ok {
		return r[st.tier]
	}
	return 0
}

func tierNameKey(tier int) string { return fmt.Sprintf("tier.name.%d", tier) }

func tierName(lang string, tier int) string { return i18n.T(lang, tierNameKey(tier)) }

// tierTitle is a machine's name with its tier in lang when it has one.
func tierTitle(lang, name string, tier int) string {
	if tier == 0 {
		return name
	}
	return name + " [" + tierName(lang, tier) + "]"
}

func init() {
	for _, a := range sgitem.TierAlloys {
		world.RegisterItem(sgitem.TierAlloy{Kind: a})
	}
	for l := 1; l <= 4; l++ {
		world.RegisterItem(sgitem.Circuit{Level: l})
	}
	for l := 1; l <= maxTier; l++ {
		world.RegisterItem(sgitem.TierKit{Level: l})
	}
}

// useKit handles a player using a kit on a machine: it goes up one tier when
// the kit is the next one.
func (s *session) useKit(p *player.Player, tx *world.Tx, pos cube.Pos, m Machine, kit sgitem.TierKit, held, off item.Stack) {
	lang, st := i18n.Of(p), m.st
	switch {
	case !tierable(m.Kind):
		p.SendTip(i18n.T(lang, "tier.cant"))
	case st.tier >= maxTier:
		p.SendTip(i18n.T(lang, "tier.max"))
	case kit.Level != st.tier+1:
		p.SendTip(fmt.Sprintf(i18n.T(lang, "tier.need"), i18n.T(lang, fmt.Sprintf("tier.kit.%d", st.tier+1))))
	default:
		st.tier++
		st.shownEnergy = -1
		st.showEnergy()
		showTier(tx, pos, st)
		if !p.GameMode().CreativeInventory() {
			p.SetHeldItems(held.Grow(-1), off)
		}
		tx.PlaySound(pos.Vec3Centre(), sound.Experience{})
		tx.AddParticle(pos.Vec3Centre(), particle.HugeExplosion{})
		count(p, "tier_up", 1)
		s.advance(tx, advancement.MachineTier)
		count(p, fmt.Sprintf("tier_up_%d", st.tier), 1)
	}
}
