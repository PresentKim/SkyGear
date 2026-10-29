package skygear

import (
	"testing"

	"github.com/presentkim/skygear/content/skygear/machine"

	"github.com/presentkim/skygear/common/i18n"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/machinerecipe"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// TestTierNumbers: a better tier spends less energy on the same work, the
// buffer grows with the tier, and a generator makes more.
func TestTierNumbers(t *testing.T) {
	crusher := &machine.State{Def: machine.Kinds[machine.KindCrusher]}
	use := crusher.Def.Use
	var perWork []float64
	for tier := range machine.MaxTier + 1 {
		crusher.Tier = tier
		total := crusher.UseTotal()
		if tier == 0 && total != use {
			t.Fatalf("basic uses %d, want %d", total, use)
		}
		perWork = append(perWork, float64(total)/float64(crusher.Reps()))
		if crusher.EnergyCap() != crusher.Def.EnergyCap*machine.TierSpecs[tier].Buffer {
			t.Fatalf("tier %d buffer %d", tier, crusher.EnergyCap())
		}
	}
	for i := 1; i < len(perWork); i++ {
		if perWork[i] >= perWork[i-1] {
			t.Fatalf("energy per work did not fall: %v", perWork)
		}
	}
	gen := &machine.State{Def: machine.Kinds[machine.KindGenerator], Tier: 3}
	if gen.GenRate() != 8*machine.GeneratorOutput || gen.Reps() != 1 || gen.BurnRate() != 3 {
		t.Fatalf("tier 3 generator: %d RE/t, %d runs, burn %d", gen.GenRate(), gen.Reps(), gen.BurnRate())
	}
	cell := &machine.State{Def: machine.Kinds[machine.KindEnergyCell], Tier: 2}
	if cell.EnergyCap() != 16*machine.CellCapacity {
		t.Fatalf("tier 2 cell holds %d", cell.EnergyCap())
	}
}

// TestTierSpeedsUpWork: a tier 3 crusher does eight times the work of a basic
// one in the same ticks.
func TestTierSpeedsUpWork(t *testing.T) {
	run := func(tier int) int {
		s := testSession(t)
		loadAround(s)
		st := placeTestMachine(s, machinePos, machine.KindCrusher, "")
		st.Tier = tier
		_ = st.Inv().SetItem(11, item.NewStack(block.Stone{}, 64))
		tick(s, 800)
		return machine.FactoryOf(st.Sess).Stats.Buckets[machine.FactoryOf(st.Sess).Stats.Head].Total
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
	st := placeTestMachine(s, machinePos, machine.KindCrusher, "")
	st.Unlimited = false
	st.Tier = 2
	st.Energy = st.EnergyCap()
	_ = st.Inv().SetItem(11, item.NewStack(block.Stone{}, 64))
	tick(s, 10)
	if got, want := st.EnergyCap()-st.Energy, 10*st.UseTotal(); got != want {
		t.Fatalf("spent %d RE in 10 ticks, want %d", got, want)
	}
	st.Energy = 0
	before := st.Progress
	tick(s, 5)
	if st.Progress != before {
		t.Fatal("a machine without energy went on working")
	}
}

// TestUseKit: the next kit raises a machine, others change nothing, and
// breaking the machine gives the kits back.
func TestUseKit(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, machine.KindCrusher, "")
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		m := machine.Machine{Kind: machine.KindCrusher, State: st}
		kit := func(l int) (item.Stack, item.Stack) { return item.NewStack(sgitem.TierKit{Level: l}, 1), item.Stack{} }
		held, off := kit(2)
		machine.UseKit(s.Session, p, tx, machinePos, m, sgitem.TierKit{Level: 2}, held, off)
		if st.Tier != 0 {
			t.Fatal("the second kit raised a basic machine")
		}
		for l := 1; l <= machine.MaxTier; l++ {
			held, off = kit(l)
			machine.UseKit(s.Session, p, tx, machinePos, m, sgitem.TierKit{Level: l}, held, off)
			if st.Tier != l {
				t.Fatalf("kit %d: tier %d", l, st.Tier)
			}
			if got := tx.Block(machinePos).(machine.Machine); got.Tier != l {
				t.Fatalf("kit %d: the block shows tier %d", l, got.Tier)
			}
		}
		held, off = kit(3)
		machine.UseKit(s.Session, p, tx, machinePos, m, sgitem.TierKit{Level: 3}, held, off)
		if st.Tier != machine.MaxTier {
			t.Fatal("the tier went past the top")
		}
		tank := placeTestMachine(s, machinePos.Add(cube.Pos{2, 0, 0}), machine.KindTank, "")
		held, off = kit(1)
		machine.UseKit(s.Session, p, tx, machinePos, machine.Machine{Kind: machine.KindTank, State: tank}, sgitem.TierKit{Level: 1}, held, off)
		if tank.Tier != 0 {
			t.Fatal("a tank took a kit")
		}
	})
	drops := machine.Machine{Kind: machine.KindCrusher, State: st}.BreakInfo().Drops(nil, nil)
	kits := 0
	for _, d := range drops {
		if _, ok := d.Item().(sgitem.TierKit); ok {
			kits++
		}
	}
	if kits != machine.MaxTier {
		t.Fatalf("%d kits came back, want %d", kits, machine.MaxTier)
	}
}

// TestTierSaved: the tier is saved with the machine and the machine's name and
// info show it.
func TestTierSaved(t *testing.T) {
	m := machine.New(machine.KindCrusher, "", "")
	m.State.Tier = 2
	back := machine.Machine{Kind: machine.KindCrusher}.DecodeNBT(m.EncodeNBT()).(machine.Machine)
	if back.State.Tier != 2 {
		t.Fatalf("tier %d after a round trip", back.State.Tier)
	}
	if back.Name() == machine.Kinds[machine.KindCrusher].Title(i18n.Default) {
		t.Fatal("the name does not show the tier")
	}
	if back.State.EnergyCap() != 4*machine.ConsumerBuffer {
		t.Fatalf("buffer %d", back.State.EnergyCap())
	}
	// a machine that takes no kits keeps tier 0 whatever is saved
	tank := machine.New(machine.KindTank, "", "")
	data := tank.EncodeNBT()
	data["Tier"] = int32(3)
	if got := (machine.Machine{Kind: machine.KindTank}).DecodeNBT(data).(machine.Machine); got.State.Tier != 0 {
		t.Fatal("a tank got a tier")
	}
}

// TestTierRanges: the harvester reaches further with its tier.
func TestTierRanges(t *testing.T) {
	st := &machine.State{Def: machine.Kinds[machine.KindHarvester]}
	var last int
	for tier := range machine.MaxTier + 1 {
		st.Tier = tier
		if r := st.Reach(machine.KindHarvester); r <= last {
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
	for _, kind := range machine.Order {
		if !machine.TierLook(kind) {
			if _, ok := (machine.Machine{Kind: kind}).States()[machine.TierState]; ok {
				t.Errorf("%s has tier states", kind)
			}
			continue
		}
		want := machine.MaxTier + 1
		if machine.AnimatedKinds[kind] {
			want *= 2
		}
		vals := machine.Machine{Kind: kind}.States()[machine.TierState]
		if len(vals) != want {
			t.Errorf("%s: %d state values, want %d", kind, len(vals), want)
		}
		seen := map[uint64]bool{}
		for tier := 0; tier <= machine.MaxTier; tier++ {
			for _, on := range []bool{false, true} {
				if on && !machine.AnimatedKinds[kind] {
					continue
				}
				m := machine.Machine{Kind: kind, Tier: tier, Active: on}
				_, h := m.Hash()
				if seen[h] {
					t.Errorf("%s: tier %d on %v repeats a hash", kind, tier, on)
				}
				seen[h] = true
				if _, props := m.EncodeBlock(); props[machine.TierState] != machine.TierValue(kind, tier, on) {
					t.Errorf("%s: encoded %v", kind, props)
				}
			}
		}
		if len(machine.Machine{Kind: kind}.Permutations()) != want-1 {
			t.Errorf("%s: %d permutations", kind, len(machine.Machine{Kind: kind}.Permutations()))
		}
	}
	if !machine.TierLook(machine.KindCrusher) || machine.TierLook(machine.KindAutoSieve) || machine.TierLook(machine.KindTank) {
		t.Fatal("wrong machines show tiers")
	}
}
