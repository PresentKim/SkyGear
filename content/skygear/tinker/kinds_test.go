package tinker_test

import (
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/presentkim/skygear/content/skygear/machine"
)

// TestTinkerTableKind: the tinker's Setup (setup_test.go) registered the
// behaviour of its machine kind: the tinker table takes the click itself and
// its panel has no lines, and the machine feature's check does not miss it.
func TestTinkerTableKind(t *testing.T) {
	d := machine.Kinds[machine.KindTinker]
	if d.Owner != "tinker" {
		t.Fatalf("the tinker table's owner is %q", d.Owner)
	}
	h := d.Hooks()
	if h.Click == nil {
		t.Error("the tinker table has no click of its own: a click would open the machine window")
	}
	if h.Lines == nil || h.Lines("ko", nil, cube.Pos{}) != nil {
		t.Error("the tinker table's panel lines are not machine.NoLines")
	}
	if err := machine.CheckKinds(); err != nil && strings.Contains(err.Error(), machine.KindTinker) {
		t.Errorf("the check misses the tinker table: %v", err)
	}
}
