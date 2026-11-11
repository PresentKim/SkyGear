package skygear

import (
	"testing"

	"github.com/presentkim/skygear/content/skygear/machine"
)

// kindOwners are the machine kinds whose behaviour another feature supplies
// (machine.Register), by kind: the feature named is the kind's Owner. A kind
// that is not here is the machine feature's own.
var kindOwners = map[string]string{
	machine.KindDrawerController: "storage", machine.KindDrawerTrim: "storage", machine.KindDrawerInterface: "storage",
}

// TestKindOwners: every kind of machine.Order has the owner kindOwners says,
// each owner is a listed feature, and every owned kind was registered by the
// time the features were installed (machine.CheckKinds, which core.Install
// ran as the machine feature's Check).
func TestKindOwners(t *testing.T) {
	listed := map[string]bool{}
	for _, f := range Features {
		listed[f.Name] = true
	}
	for _, k := range machine.Order {
		d := machine.Kinds[k]
		if d.Owner != kindOwners[k] {
			t.Errorf("kind %s has the owner %q, kindOwners says %q", k, d.Owner, kindOwners[k])
		}
		if d.Owner != "" && !listed[d.Owner] {
			t.Errorf("kind %s is owned by %q, which is not a listed feature", k, d.Owner)
		}
	}
	for k := range kindOwners {
		if machine.Kinds[k] == nil {
			t.Errorf("kindOwners names %s, which is not a machine kind", k)
		}
	}
	if err := machine.CheckKinds(); err != nil {
		t.Error(err)
	}
	if machine.Feature.Check == nil {
		t.Error("the machine feature has no Check: an unregistered kind would not stop the start")
	}
}
