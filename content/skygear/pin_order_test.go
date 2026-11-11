package skygear

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/presentkim/skygear/content/skygear/core"
)

// TestPinOrder compares the kernel's subscriptions with
// testdata/pin/order.golden: one line "<Hook> <priority> <name>" per
// subscriber, hooks in declaration order, each hook's subscribers in the
// order they run. It names no feature, so it does not change when a step moves
// to its feature.
//
// The golden was recorded once from the bridged code. Re-record it
// (SKYGEAR_GOLDEN=update) only when a step is added, removed or reordered on
// purpose.
func TestPinOrder(t *testing.T) {
	var b strings.Builder
	for _, s := range core.Subscriptions() {
		fmt.Fprintf(&b, "%s %d %s\n", s.Hook, s.Prio, s.Name)
	}
	got := b.String()
	const file = "testdata/pin/order.golden"
	if os.Getenv("SKYGEAR_GOLDEN") == "update" {
		if err := os.WriteFile(file, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if want := string(raw); got != want {
		wl, gl := strings.Split(want, "\n"), strings.Split(got, "\n")
		for i := 0; i < len(wl) || i < len(gl); i++ {
			var w, g string
			if i < len(wl) {
				w = wl[i]
			}
			if i < len(gl) {
				g = gl[i]
			}
			if w != g {
				t.Fatalf("%s line %d:\nwant: %q\ngot:  %q", file, i+1, w, g)
			}
		}
	}
}
