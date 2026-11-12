package machine

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// Machine tiers: a machine is made faster
// and larger with kits that are made from alloys and control circuits. The
// tier of a machine is saved with it.

func init() {
	for _, a := range sgitem.TierAlloys {
		world.RegisterItem(sgitem.TierAlloy{Kind: a})
	}
	for l := 1; l <= 4; l++ {
		world.RegisterItem(sgitem.Circuit{Level: l})
	}
	for l := 1; l <= MaxTier; l++ {
		world.RegisterItem(sgitem.TierKit{Level: l})
	}
}

const MaxTier = 3

// tierSpec are the numbers of a tier. A machine of speed n runs its work n
// times a tick and spends use10/10 times the energy of a basic one in all, so
// a better machine spends less energy on the same work. A generator makes gen
// times the energy a tick and burns its fuel burn times as fast; an energy
// cell holds cell times as much; the buffer of the others grows by buffer.
type tierSpec struct {
	speed, use10, Buffer, gen, Burn, Cell int
}

var TierSpecs = [MaxTier + 1]tierSpec{
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
	KindGenerator: true, KindLavaGenerator: true, KindSolarGenerator: true, KindHydroGenerator: true, KindEnergyCell: true,
}

// tierRanges are the reaches of the machines that gain range with their tier
// (blocks each way sideways).
var tierRanges = map[string][MaxTier + 1]int{
	KindHarvester: {harvestRadius, 6, 8, 12},
	KindVacuum:    {vacuumReach, 9, 12, 16},
	KindFeller:    {fellerReach, 7, 9, 12},
}

// tierable reports whether the machines of a kind take kits.
func tierable(kind string) bool { return tierKinds[kind] }

func (st *State) spec() tierSpec { return TierSpecs[st.Tier] }

// reps is how many times a tick the machine runs its work; generators and
// cells do not run faster, they make more.
func (st *State) Reps() int {
	if st.Def.Generates > 0 || st.Def.Cell {
		return 1
	}
	return st.spec().speed
}

// energyCap is the RE the machine's buffer holds.
func (st *State) EnergyCap() int {
	k := st.Def
	switch {
	case k.Cell:
		return k.EnergyCap * st.spec().Cell
	case k.Generates > 0:
		return k.EnergyCap * st.spec().gen
	}
	return k.EnergyCap * st.spec().Buffer
}

// useCost is the RE one run of the work takes (rounded up). A machine that
// works on an area spends by the size of the area it is set to instead
// (rangeUse).
func (st *State) useCost() int {
	if st.Def.Use == 0 {
		return 0
	}
	reps := st.Reps()
	if kind, ok := st.rangeKind(); ok {
		return (st.rangeUse(kind) + reps - 1) / reps
	}
	per := 10 * reps
	return (st.Def.Use*st.spec().use10 + per - 1) / per
}

// rangeUse is the RE a tick a machine of kind that works on an area spends:
// its Use for the whole area of a basic machine, and as much more or less as
// the area it is set to has more or fewer blocks (at least 1). A bigger tier
// reaches further and spends more; a smaller range set on the screen spends
// less.
func (st *State) rangeUse(kind string) int {
	base := baseArea(kind).blocks()
	if base == 0 {
		return st.Def.Use
	}
	return max(1, (st.Def.Use*st.Area(kind).blocks()+base-1)/base)
}

// baseArea is the area of a basic (tier 0) machine of kind.
func baseArea(kind string) Area {
	down, up := rangeHeight(kind)
	r := tierRanges[kind][0]
	return Area{X: r, Z: r, Down: down, Up: up}
}

// blocks is how many blocks the area holds.
func (a Area) blocks() int { return (2*a.X + 1) * (2*a.Z + 1) * (a.Down + a.Up + 1) }

// useTotal is the RE a tick of work takes.
func (st *State) UseTotal() int { return st.useCost() * st.Reps() }

// genRate is the RE a generator makes a tick, burnRate how fast it burns.
func (st *State) GenRate() int { return st.Def.Generates * st.spec().gen }

func (st *State) BurnRate() int { return st.spec().Burn }

// Area is the box a machine with a range works on around itself, in blocks:
// X and Z each way sideways, Down below it and Up above it.
type Area struct{ X, Z, Down, Up int }

// MaxArea is the area the machine's tier gives a machine of kind.
func (st *State) MaxArea(kind string) Area {
	r, ok := tierRanges[kind]
	if !ok {
		return Area{}
	}
	down, up := rangeHeight(kind)
	return Area{X: r[st.Tier], Z: r[st.Tier], Down: down, Up: up}
}

// Area is the area a machine of kind works on: the range set on its screen
// (RangeX, RangeY, RangeZ; 0 is all), at most what its tier gives. RangeY
// limits the reach below and above alike.
func (st *State) Area(kind string) Area {
	a := st.MaxArea(kind)
	if st.RangeX > 0 {
		a.X = min(a.X, st.RangeX)
	}
	if st.RangeZ > 0 {
		a.Z = min(a.Z, st.RangeZ)
	}
	if st.RangeY > 0 {
		a.Down, a.Up = min(a.Down, st.RangeY), min(a.Up, st.RangeY)
	}
	return a
}

// Offset is how far the centre of a machine's area is moved from the
// machine, set on its range screen. Each axis is clamped to MaxOffset, so a
// value saved under another limit stays in bounds.
func (st *State) Offset(kind string) cube.Pos {
	most := st.MaxOffset(kind)
	clamp := func(v, top int) int { return max(-top, min(top, v)) }
	return cube.Pos{clamp(st.OffsetX, most[0]), clamp(st.OffsetY, most[1]), clamp(st.OffsetZ, most[2])}
}

// harvesterLift is how far up or down the harvester's row of crops may be
// moved: its area is one block high, so it has no reach of its own to go by.
const harvesterLift = 2

// rangeLifts are the kinds whose area moves up and down by a fixed number of
// blocks instead of by its own height.
var rangeLifts = map[string]int{KindHarvester: harvesterLift}

// MaxOffset is how far the area of a machine of kind may be moved each way
// on each axis: as far as its tier reaches sideways, and as far as it reaches
// below or above.
func (st *State) MaxOffset(kind string) cube.Pos {
	most := st.MaxArea(kind)
	y := max(most.Down, most.Up)
	if lift, ok := rangeLifts[kind]; ok {
		y = lift
	}
	return cube.Pos{most.X, y, most.Z}
}

// Centre is the block a machine of kind at pos works around: the machine
// itself moved by its offset.
func (st *State) Centre(kind string, pos cube.Pos) cube.Pos {
	return pos.Add(st.Offset(kind))
}

func tierNameKey(tier int) string { return fmt.Sprintf("tier.name.%d", tier) }

func TierName(lang string, tier int) string { return i18n.T(lang, tierNameKey(tier)) }

// tierTitle is a machine's name with its tier in lang when it has one.
func tierTitle(lang, name string, tier int) string {
	if tier == 0 {
		return name
	}
	return name + " [" + TierName(lang, tier) + "]"
}

// UseKit handles a player using a kit on a machine: it goes up one tier when
// the kit is the next one.
func UseKit(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, m Machine, kit sgitem.TierKit, held, off item.Stack) {
	lang, st := i18n.Of(p), m.State
	switch {
	case !tierable(m.Kind):
		p.SendTip(i18n.T(lang, "tier.cant"))
	case st.Tier >= MaxTier:
		p.SendTip(i18n.T(lang, "tier.max"))
	case kit.Level != st.Tier+1:
		p.SendTip(fmt.Sprintf(i18n.T(lang, "tier.need"), i18n.T(lang, fmt.Sprintf("tier.kit.%d", st.Tier+1))))
	default:
		st.Tier++
		st.shownEnergy = -1
		st.ShowEnergy()
		showTier(tx, pos, st)
		if !p.GameMode().CreativeInventory() {
			p.SetHeldItems(held.Grow(-1), off)
		}
		tx.PlaySound(pos.Vec3Centre(), sound.Experience{})
		tx.AddParticle(pos.Vec3Centre(), particle.HugeExplosion{})
		achievement.Count(p, "tier_up", 1)
		progress.Advance(s, tx, advancement.MachineTier)
		achievement.Count(p, fmt.Sprintf("tier_up_%d", st.Tier), 1)
	}
}
