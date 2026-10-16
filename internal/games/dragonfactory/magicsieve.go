package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/item"
)

// The magic sieve (docs/design/15-magic-sieve.md): an auto sieve with an
// essence slot. A mob's essence charges it for magicCharges sifts, and while
// it is charged, what it sifts gets the mob's theme (magicsieve_tables.go) on
// top of, or instead of, the table of the mesh.

// magicCharges is how many sifts one essence charges.
const magicCharges = 24

// themeCap is the most a theme drop is likely to come out, per mille.
const themeCap = 950

// rareBoost is how much a mob's difficulty raises its rare drops: ×1 for the
// easiest, +0.2 a step.
func rareBoost(mob string) float64 { return 1 + 0.2*float64(mobDifficulty[mob]-1) }

// meshBoost is how much a better mesh multiplies the chances of the theme
// drops of the magic sieve (percent).
var meshBoost = map[string]int{"flint": 130, "iron": 160, "gold": 200, "diamond": 250, "netherite": 300}

// themeChance is the chance, per mille, of a theme drop with a mesh fitted.
func themeChance(d ThemeDrop, mesh, mob string) float64 {
	boost := 1.0
	if percent, ok := meshBoost[mesh]; ok {
		boost = float64(percent) / 100
	}
	p := float64(d.Permille) * boost
	if d.Rare {
		p *= rareBoost(mob)
	}
	return min(p, themeCap)
}

// chargeSieve takes an essence from the essence slot when the sieve has no
// charge left.
func (s *session) chargeSieve(st *machineState) {
	if len(st.kind.charge) == 0 || st.charges > 0 {
		return
	}
	st.mob = ""
	for _, slot := range st.kind.charge {
		in, _ := st.inv().Item(slot)
		e, ok := in.Item().(Essence)
		if !ok {
			continue
		}
		_ = st.inv().SetItem(slot, in.Grow(-1))
		st.mob, st.charges = e.Mob, magicCharges
		return
	}
}

// siftDrops rolls the products of sifting in with the sieve: the mesh table,
// and for a charged magic sieve the theme of its essence, which uses up a
// charge.
func (s *session) siftDrops(st *machineState, in item.Stack) []item.Stack {
	if len(st.kind.charge) == 0 || st.charges == 0 {
		return s.sieveDrops(in, st.mesh)
	}
	drops, rare := s.magicDrops(in, st.mesh, st.mob)
	st.tally("magic_sift", 1)
	st.tally("magic_sift_"+st.mob, 1)
	if rare > 0 {
		st.tally("rare_find", int64(rare))
	}
	if st.charges--; st.charges == 0 {
		st.mob = ""
	}
	return drops
}

// magicDrops rolls the products of sifting in with the theme of a mob: the
// mesh table unless the theme replaces it, then the theme's drops. It also
// returns how many of the drops were rare ones.
func (s *session) magicDrops(in item.Stack, mesh, mob string) (out []item.Stack, rare int) {
	theme := magicThemes[mob]
	key := itemKey(in.Item())
	if !theme.Replace[key] {
		out = s.sieveDrops(in, mesh)
	}
	for _, d := range theme.Adds[key] {
		if s.rand.Float64()*1000 < themeChance(d, mesh, mob) {
			out = append(out, d.Item)
			if d.Rare {
				rare++
			}
		}
	}
	return out, rare
}
