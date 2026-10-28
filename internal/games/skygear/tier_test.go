package skygear

import (
	"testing"

	"github.com/presentkim/skygear/common/i18n"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/internal/games/skygear/machinerecipe"
	"github.com/presentkim/skygear/internal/games/skygear/sgitem"
)

// TestTierNumbers: a better tier spends less energy on the same work, the
// buffer grows with the tier, and a generator makes more.
func TestTierNumbers(t *testing.T) {
	crusher := &machineState{kind: machineKinds[KindCrusher]}
	use := crusher.kind.use
	var perWork []float64
	for tier := range maxTier + 1 {
		crusher.tier = tier
		total := crusher.useTotal()
		if tier == 0 && total != use {
			t.Fatalf("basic uses %d, want %d", total, use)
		}
		perWork = append(perWork, float64(total)/float64(crusher.reps()))
		if crusher.energyCap() != crusher.kind.energyCap*tierSpecs[tier].buffer {
			t.Fatalf("tier %d buffer %d", tier, crusher.energyCap())
		}
	}
	for i := 1; i < len(perWork); i++ {
		if perWork[i] >= perWork[i-1] {
			t.Fatalf("energy per work did not fall: %v", perWork)
		}
	}
	gen := &machineState{kind: machineKinds[KindGenerator], tier: 3}
	if gen.genRate() != 8*generatorOutput || gen.reps() != 1 || gen.burnRate() != 3 {
		t.Fatalf("tier 3 generator: %d RE/t, %d runs, burn %d", gen.genRate(), gen.reps(), gen.burnRate())
	}
	cell := &machineState{kind: machineKinds[KindEnergyCell], tier: 2}
	if cell.energyCap() != 16*cellCapacity {
		t.Fatalf("tier 2 cell holds %d", cell.energyCap())
	}
}

// TestTierSpeedsUpWork: a tier 3 crusher does eight times the work of a basic
// one in the same ticks.
func TestTierSpeedsUpWork(t *testing.T) {
	run := func(tier int) int {
		s := testSession(t)
		loadAround(s)
		st := placeTestMachine(s, machinePos, KindCrusher, "")
		st.tier = tier
		_ = st.inv().SetItem(11, item.NewStack(block.Stone{}, 64))
		tick(s, 800)
		return st.sess.stats.buckets[st.sess.stats.head].total
	}
	basic, fast := run(0), run(3)
	if basic < 8 || fast < basic*6 {
		t.Fatalf("basic made %d, tier 3 made %d", basic, fast)
	}
}

// TestTierEnergy: with energy the tier spends what its numbers say and stops
// when the buffer is empty.
func TestTierEnergy(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindCrusher, "")
	st.unlimited = false
	st.tier = 2
	st.energy = st.energyCap()
	_ = st.inv().SetItem(11, item.NewStack(block.Stone{}, 64))
	tick(s, 10)
	if got, want := st.energyCap()-st.energy, 10*st.useTotal(); got != want {
		t.Fatalf("spent %d RE in 10 ticks, want %d", got, want)
	}
	st.energy = 0
	before := st.progress
	tick(s, 5)
	if st.progress != before {
		t.Fatal("a machine without energy went on working")
	}
}

// TestUseKit: the next kit raises a machine, others change nothing, and
// breaking the machine gives the kits back.
func TestUseKit(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindCrusher, "")
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		m := Machine{Kind: KindCrusher, st: st}
		kit := func(l int) (item.Stack, item.Stack) { return item.NewStack(sgitem.TierKit{Level: l}, 1), item.Stack{} }
		held, off := kit(2)
		s.useKit(p, tx, machinePos, m, sgitem.TierKit{Level: 2}, held, off)
		if st.tier != 0 {
			t.Fatal("the second kit raised a basic machine")
		}
		for l := 1; l <= maxTier; l++ {
			held, off = kit(l)
			s.useKit(p, tx, machinePos, m, sgitem.TierKit{Level: l}, held, off)
			if st.tier != l {
				t.Fatalf("kit %d: tier %d", l, st.tier)
			}
			if got := tx.Block(machinePos).(Machine); got.tier != l {
				t.Fatalf("kit %d: the block shows tier %d", l, got.tier)
			}
		}
		held, off = kit(3)
		s.useKit(p, tx, machinePos, m, sgitem.TierKit{Level: 3}, held, off)
		if st.tier != maxTier {
			t.Fatal("the tier went past the top")
		}
		tank := placeTestMachine(s, machinePos.Add(cube.Pos{2, 0, 0}), KindTank, "")
		held, off = kit(1)
		s.useKit(p, tx, machinePos, Machine{Kind: KindTank, st: tank}, sgitem.TierKit{Level: 1}, held, off)
		if tank.tier != 0 {
			t.Fatal("a tank took a kit")
		}
	})
	drops := Machine{Kind: KindCrusher, st: st}.BreakInfo().Drops(nil, nil)
	kits := 0
	for _, d := range drops {
		if _, ok := d.Item().(sgitem.TierKit); ok {
			kits++
		}
	}
	if kits != maxTier {
		t.Fatalf("%d kits came back, want %d", kits, maxTier)
	}
}

// TestTierSaved: the tier is saved with the machine and the machine's name and
// info show it.
func TestTierSaved(t *testing.T) {
	m := newMachine(KindCrusher, "", "")
	m.st.tier = 2
	back := Machine{Kind: KindCrusher}.DecodeNBT(m.EncodeNBT()).(Machine)
	if back.st.tier != 2 {
		t.Fatalf("tier %d after a round trip", back.st.tier)
	}
	if back.Name() == machineKinds[KindCrusher].title(i18n.Default) {
		t.Fatal("the name does not show the tier")
	}
	if back.st.energyCap() != 4*consumerBuffer {
		t.Fatalf("buffer %d", back.st.energyCap())
	}
	// a machine that takes no kits keeps tier 0 whatever is saved
	tank := newMachine(KindTank, "", "")
	data := tank.EncodeNBT()
	data["Tier"] = int32(3)
	if got := (Machine{Kind: KindTank}).DecodeNBT(data).(Machine); got.st.tier != 0 {
		t.Fatal("a tank got a tier")
	}
}

// TestTierRanges: the harvester reaches further with its tier.
func TestTierRanges(t *testing.T) {
	st := &machineState{kind: machineKinds[KindHarvester]}
	var last int
	for tier := range maxTier + 1 {
		st.tier = tier
		if r := st.reach(KindHarvester); r <= last {
			t.Fatalf("tier %d reaches %d after %d", tier, r, last)
		} else {
			last = r
		}
	}
}

// TestTierAlloyRecipes: the alloy smelter makes each alloy of its ingredients.
func TestTierAlloyRecipes(t *testing.T) {
	cases := []struct {
		a, b item.Stack
		want string
	}{
		{item.NewStack(item.IronIngot{}, 1), item.NewStack(item.RedstoneWire{}, 4), "infused"},
		{item.NewStack(sgitem.TierAlloy{Kind: "infused"}, 2), item.NewStack(item.Diamond{}, 1), "reinforced"},
		{item.NewStack(block.Obsidian{}, 2), item.NewStack(sgitem.TierAlloy{Kind: "reinforced"}, 2), "atomic"},
	}
	for _, c := range cases {
		r, ok := machinerecipe.MatchAlloy(c.a, c.b)
		if !ok || r.Out.(sgitem.TierAlloy).Kind != c.want || r.OutCount != 2 {
			t.Errorf("%s: %v %v", c.want, r, ok)
		}
	}
}

// TestTierLooks: the machines that show a tier have a block state for every
// tier (and working look), with a hash of its own, and the other machines have
// none.
func TestTierLooks(t *testing.T) {
	for _, kind := range machineOrder {
		if !tierLook(kind) {
			if _, ok := (Machine{Kind: kind}).States()[tierState]; ok {
				t.Errorf("%s has tier states", kind)
			}
			continue
		}
		want := maxTier + 1
		if animatedMachines[kind] {
			want *= 2
		}
		vals := Machine{Kind: kind}.States()[tierState]
		if len(vals) != want {
			t.Errorf("%s: %d state values, want %d", kind, len(vals), want)
		}
		seen := map[uint64]bool{}
		for tier := 0; tier <= maxTier; tier++ {
			for _, on := range []bool{false, true} {
				if on && !animatedMachines[kind] {
					continue
				}
				m := Machine{Kind: kind, tier: tier, active: on}
				_, h := m.Hash()
				if seen[h] {
					t.Errorf("%s: tier %d on %v repeats a hash", kind, tier, on)
				}
				seen[h] = true
				if _, props := m.EncodeBlock(); props[tierState] != tierValue(kind, tier, on) {
					t.Errorf("%s: encoded %v", kind, props)
				}
			}
		}
		if len(Machine{Kind: kind}.Permutations()) != want-1 {
			t.Errorf("%s: %d permutations", kind, len(Machine{Kind: kind}.Permutations()))
		}
	}
	if !tierLook(KindCrusher) || tierLook(KindAutoSieve) || tierLook(KindTank) {
		t.Fatal("wrong machines show tiers")
	}
}
