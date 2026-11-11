package machine

import (
	"fmt"
	"strings"
	"testing"
)

// ownedKind makes KindRelicTable a kind of the feature "dimension" that is
// not registered yet, for the length of the test.
func ownedKind(t *testing.T) *Def {
	t.Helper()
	d := Kinds[KindRelicTable]
	saved, checked := *d, kindsChecked
	t.Cleanup(func() { *d, kindsChecked = saved, checked })
	d.Owner, d.registered, d.Behaviour = "dimension", false, Behaviour{}
	kindsChecked = false
	return d
}

func panicOf(f func()) (msg string) {
	defer func() {
		if r := recover(); r != nil {
			msg = fmt.Sprint(r)
		}
	}()
	f()
	return ""
}

// TestCheckKindsNamesTheMissingOwner: a kind whose owner did not register it
// fails the check with the kind and the feature in the message, and passes
// once the owner registered it.
func TestCheckKindsNamesTheMissingOwner(t *testing.T) {
	d := ownedKind(t)
	err := CheckKinds()
	if err == nil || !strings.Contains(err.Error(), "relic_table (feature dimension)") {
		t.Fatalf("check of an unregistered kind: %v", err)
	}
	if kindsChecked {
		t.Fatal("a failed check closed the registry")
	}
	Register(KindRelicTable, Behaviour{Lines: NoLines})
	if !d.registered || d.Lines == nil {
		t.Fatal("Register did not set the behaviour")
	}
	if err := CheckKinds(); err != nil && strings.Contains(err.Error(), "relic_table") {
		t.Fatalf("check after the registration: %v", err)
	}
}

// TestRegisterPanics: Register refuses a name that is no kind, a kind of the
// machine feature itself, a second registration and one after the check
// passed.
func TestRegisterPanics(t *testing.T) {
	ownedKind(t)
	if msg := panicOf(func() { Register("no_such_kind", Behaviour{}) }); !strings.Contains(msg, "not a machine kind") {
		t.Errorf("unknown kind: %q", msg)
	}
	if msg := panicOf(func() { Register(KindCrusher, Behaviour{}) }); !strings.Contains(msg, "has no owner") {
		t.Errorf("the machine feature's own kind: %q", msg)
	}
	Register(KindRelicTable, Behaviour{})
	if msg := panicOf(func() { Register(KindRelicTable, Behaviour{}) }); !strings.Contains(msg, "twice") {
		t.Errorf("second registration: %q", msg)
	}
	Kinds[KindRelicTable].registered = false
	kindsChecked = true
	if msg := panicOf(func() { Register(KindRelicTable, Behaviour{}) }); !strings.Contains(msg, "after the features were installed") {
		t.Errorf("registration after the check: %q", msg)
	}
}

// TestOrderListsEveryKind: Order and Kinds hold the same kinds, each once.
func TestOrderListsEveryKind(t *testing.T) {
	seen := map[string]bool{}
	for _, k := range Order {
		if Kinds[k] == nil {
			t.Errorf("%s is in Order but not in Kinds", k)
		}
		if seen[k] {
			t.Errorf("%s is in Order twice", k)
		}
		seen[k] = true
	}
	if len(seen) != len(Kinds) {
		t.Errorf("Order has %d kinds, Kinds %d", len(seen), len(Kinds))
	}
}

// TestFrameFlags: the kinds without a display to keep, without a window and
// with a turning cycle are the ones the frame named by id before the flags.
func TestFrameFlags(t *testing.T) {
	want := map[string][]string{
		"NoDisplay": {KindEnergyCell, KindTank, KindCrucible, KindLavaGenerator, KindSolarGenerator, KindHydroGenerator, KindDrawerController, KindDrawerTrim, KindDrawerInterface, KindTinker, KindCollector, KindRelicTable},
		"NoWindow":  {KindNetCable},
		"Cycle":     {KindHarvester},
	}
	for flag, kinds := range want {
		in := map[string]bool{}
		for _, k := range kinds {
			in[k] = true
		}
		for _, k := range Order {
			d := Kinds[k]
			got := map[string]bool{"NoDisplay": d.NoDisplay, "NoWindow": d.NoWindow, "Cycle": d.Cycle}[flag]
			if got != in[k] {
				t.Errorf("%s: %s is %t, want %t", k, flag, got, in[k])
			}
		}
	}
	for _, k := range Order {
		if d := Kinds[k]; d.Cell && !d.NoDisplay {
			t.Errorf("%s stores energy but has a display: its tick would not offer the energy to the network", k)
		}
	}
}

// TestEveryKindHasOneShape: a kind is a sieve, a thin network block or a cube
// of exactly one of the three texture sets, so the properties of a cube that
// is in neither cubeMachines nor plainCubes are those of a frontMachines kind.
func TestEveryKindHasOneShape(t *testing.T) {
	for _, k := range Order {
		_, plain := plainCubes[k]
		n := 0
		for _, in := range []bool{IsSieve(k), NetShaped(k), plain, cubeMachines[k], frontMachines[k]} {
			if in {
				n++
			}
		}
		if n != 1 {
			t.Errorf("%s is in %d of the shape sets, want 1", k, n)
		}
		if cube := plain || cubeMachines[k] || frontMachines[k]; cube != (shapeOf(k) == shapeCube) {
			t.Errorf("%s: shapeOf says %d", k, shapeOf(k))
		}
	}
}
