package skygear

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/sievetable"
)

// The magic sieve: an auto sieve with an
// essence slot. A mob's essence charges it for sievetable.MagicCharges sifts, and while
// it is charged, what it sifts gets the mob's theme (magicsieve_tables.go) on
// top of, or instead of, the table of the mesh.

// chargeSieve takes an essence from the essence slot when the sieve has no
// charge left.
func (s *session) chargeSieve(st *machineState) {
	if len(st.kind.charge) == 0 || st.charges > 0 {
		return
	}
	st.mob = ""
	for _, slot := range st.kind.charge {
		in, _ := st.inv().Item(slot)
		e, ok := in.Item().(sgitem.Essence)
		if !ok {
			continue
		}
		_ = st.inv().SetItem(slot, in.Grow(-1))
		st.mob, st.charges = e.Mob, sievetable.MagicCharges
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
	theme := sievetable.MagicThemes[mob]
	key := naming.ItemKey(in.Item())
	if !theme.Replace[key] {
		out = s.sieveDrops(in, mesh)
	}
	for _, d := range theme.Adds[key] {
		if s.rand.Float64()*1000 < sievetable.ThemeChance(d, mesh, mob) {
			out = append(out, d.Item)
			if d.Rare {
				rare++
			}
		}
	}
	return out, rare
}
