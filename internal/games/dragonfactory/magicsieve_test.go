package dragonfactory

import (
	"slices"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

// TestMagicThemesAreComplete: every mob has a theme with drops that all
// exist, the replaced inputs have drops, and the difficulties and flower
// kinds fit the mob list.
func TestMagicThemesAreComplete(t *testing.T) {
	for _, mob := range mobKinds {
		th, ok := magicThemes[mob]
		if !ok {
			t.Errorf("%s has no theme", mob)
			continue
		}
		if mobDifficulty[mob] < 1 {
			t.Errorf("%s has no difficulty", mob)
		}
		for in, drops := range th.Adds {
			if len(drops) == 0 {
				t.Errorf("%s: empty drop list for %s", mob, in)
			}
			for _, d := range drops {
				if d.Item.Empty() || d.Permille <= 0 || d.Permille > 1000 {
					t.Errorf("%s: bad drop %v", mob, d)
				}
				name, meta := d.Item.Item().EncodeItem()
				if _, ok := world.ItemByName(name, meta); !ok {
					t.Errorf("%s: %s is not a registered item", mob, name)
				}
			}
		}
		for in := range th.Replace {
			if len(th.Adds[in]) == 0 {
				t.Errorf("%s replaces %s but gives nothing for it", mob, in)
			}
		}
	}
	for mob := range magicThemes {
		if !slices.Contains(mobKinds, mob) {
			t.Errorf("theme of unknown mob %s", mob)
		}
	}
}

// TestThemeChance: the mesh multiplies a theme and the mob's difficulty the
// rare drops, up to the cap.
func TestThemeChance(t *testing.T) {
	plainDrop := ThemeDrop{Permille: 100}
	rareDrop := ThemeDrop{Permille: 10, Rare: true}
	if got := themeChance(plainDrop, "string", MobWither); got != 100 {
		t.Fatalf("plain drop with the string mesh: %v, want 100", got)
	}
	if got := themeChance(plainDrop, "netherite", MobZombie); got != 300 {
		t.Fatalf("plain drop with the netherite mesh: %v, want 300", got)
	}
	easy, hard := themeChance(rareDrop, "string", MobZombie), themeChance(rareDrop, "string", MobWither)
	if easy != 10 || hard <= easy*2 {
		t.Fatalf("rare drop: %v for the easiest mob, %v for the hardest", easy, hard)
	}
	if got := themeChance(ThemeDrop{Permille: 900, Rare: true}, "netherite", MobWither); got != themeCap {
		t.Fatalf("capped chance %v, want %d", got, themeCap)
	}
}

// TestMagicSieveCharging: an essence goes to the essence slot only, charges
// the sieve for magicCharges sifts, and the sieve falls back to its mesh
// table when the charges run out.
func TestMagicSieveCharging(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindMagicSieve, "")
	ess := item.NewStack(Essence{Mob: MobPiglin}, 2)
	if st.slotAccepts(10, ess) || !st.slotAccepts(11, ess) {
		t.Fatal("the essence belongs in the essence slot only")
	}
	if st.slotAccepts(11, item.NewStack(block.Sand{}, 1)) || !st.slotAccepts(10, item.NewStack(block.Sand{}, 1)) {
		t.Fatal("the essence slot takes only essences")
	}
	if !st.insert(ess) || !st.insert(ess) {
		t.Fatal("essences were refused")
	}
	if got, _ := st.inv().Item(11); got.Count() != 2 {
		t.Fatalf("%d essences in the essence slot, want 2", got.Count())
	}
	in := item.NewStack(block.Sand{}, 1)
	s.chargeSieve(st)
	if st.charges != magicCharges || st.mob != MobPiglin {
		t.Fatalf("charged with %d of %q", st.charges, st.mob)
	}
	for range magicCharges {
		s.siftDrops(st, in)
	}
	if st.charges != 0 || st.mob != "" {
		t.Fatalf("after %d sifts: %d charges of %q", magicCharges, st.charges, st.mob)
	}
	s.chargeSieve(st)
	if st.charges != magicCharges {
		t.Fatal("the second essence did not charge")
	}
	if got, _ := st.inv().Item(11); !got.Empty() {
		t.Fatal("both essences should be used")
	}
	// no charge left: the mesh table
	st.charges, st.mob = 0, ""
	for range 200 {
		for _, d := range s.siftDrops(st, in) {
			if _, ok := d.Item().(block.SoulSand); ok {
				t.Fatal("a theme drop came out of an uncharged sieve")
			}
		}
	}
}

// TestMagicThemeReplaces: the piglin replaces what sand gives, the zombie only
// adds to it.
func TestMagicThemeReplaces(t *testing.T) {
	s := testSession(t)
	sand := item.NewStack(block.Sand{}, 1)
	base := map[string]bool{}
	for _, d := range sieveTables["string"][itemKey(block.Sand{})] {
		base[itemKey(d.Item.Item())] = true
	}
	var sawSoul, sawBase bool
	for range 3000 {
		drops, _ := s.magicDrops(sand, "string", MobPiglin)
		for _, d := range drops {
			if _, ok := d.Item().(block.SoulSand); ok {
				sawSoul = true
			}
			if base[itemKey(d.Item())] {
				t.Fatalf("the piglin theme left the base drop %v", d)
			}
		}
	}
	if !sawSoul {
		t.Fatal("the piglin theme gave no soul sand from sand")
	}
	for range 3000 {
		drops, _ := s.magicDrops(sand, "string", MobZombie)
		for _, d := range drops {
			if base[itemKey(d.Item())] {
				sawBase = true
			}
		}
	}
	if !sawBase {
		t.Fatal("the zombie theme dropped the base table")
	}
}

// TestMagicSieveWorks: a charged sieve sifts on its own and counts the sifts.
func TestMagicSieveWorks(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindMagicSieve, "")
	_ = st.insert(item.NewStack(Essence{Mob: MobEnder}, 1))
	for range 3 {
		_ = st.insert(item.NewStack(block.Gravel{}, 1))
	}
	tick(s, 3*(machineKinds[KindMagicSieve].interval+3))
	if st.charges >= magicCharges {
		t.Fatalf("no charge was used (%d left)", st.charges)
	}
	if st.mob != MobEnder {
		t.Fatalf("charged with %q", st.mob)
	}
}
