package machine

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/sievetable"
)

// The magic sieve: an auto sieve with an
// essence slot. A mob's essence charges it for sievetable.MagicCharges sifts, and while
// it is charged, what it sifts gets the mob's theme (magicsieve_tables.go) on
// top of, or instead of, the table of the mesh.

// ChargeSieve takes an essence from the essence slot when the sieve has no
// charge left.
func ChargeSieve(s *core.Session, st *MachineState) {
	if len(st.Def.Charge) == 0 || st.Charges > 0 {
		return
	}
	st.Mob = ""
	for _, slot := range st.Def.Charge {
		in, _ := st.Inv().Item(slot)
		e, ok := in.Item().(sgitem.Essence)
		if !ok {
			continue
		}
		_ = st.Inv().SetItem(slot, in.Grow(-1))
		st.Mob, st.Charges = e.Mob, sievetable.MagicCharges
		return
	}
}

// SiftDrops rolls the products of sifting in with the sieve: the mesh table,
// and for a charged magic sieve the theme of its essence, which uses up a
// charge.
func SiftDrops(s *core.Session, st *MachineState, in item.Stack) []item.Stack {
	if len(st.Def.Charge) == 0 || st.Charges == 0 {
		return sieveDrops(s, in, st.Mesh)
	}
	drops, rare := MagicDrops(s, in, st.Mesh, st.Mob)
	st.tally("magic_sift", 1)
	st.tally("magic_sift_"+st.Mob, 1)
	if rare > 0 {
		st.tally("rare_find", int64(rare))
	}
	if st.Charges--; st.Charges == 0 {
		st.Mob = ""
	}
	return drops
}

// MagicDrops rolls the products of sifting in with the theme of a mob: the
// mesh table unless the theme replaces it, then the theme's drops. It also
// returns how many of the drops were rare ones.
func MagicDrops(s *core.Session, in item.Stack, mesh, mob string) (out []item.Stack, rare int) {
	theme := sievetable.MagicThemes[mob]
	key := naming.ItemKey(in.Item())
	if !theme.Replace[key] {
		out = sieveDrops(s, in, mesh)
	}
	for _, d := range theme.Adds[key] {
		if s.Rand.Float64()*1000 < sievetable.ThemeChance(d, mesh, mob) {
			out = append(out, d.Item)
			if d.Rare {
				rare++
			}
		}
	}
	return out, rare
}
