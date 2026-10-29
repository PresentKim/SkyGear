package ui

import (
	"testing"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// TestPlaneKeys: the keys of a PlaneMenu are on the root of the layout as
// ids the screen runs; a key without a run is not answered.
func TestPlaneKeys(t *testing.T) {
	var got *oreLayout
	var layout any
	oreShown = func(l *oreLayout, lay any) { got, layout = l, lay }
	defer func() { oreShown = nil }()

	var pressed Key
	key := func(k Key) func(*player.Player, *world.Tx) {
		return func(*player.Player, *world.Tx) { pressed = k }
	}
	PlaneMenu(nil, Plane{Keys: map[Key]func(*player.Player, *world.Tx){KeyUp: key(KeyUp), KeyOK: key(KeyOK), KeyLeft: nil}})
	keys, _ := layout.(map[string]any)["keys"].(map[string]string)
	if len(keys) != 2 || keys["left"] != "" {
		t.Fatalf("keys %v, want up and ok", keys)
	}
	for _, k := range []Key{KeyUp, KeyOK} {
		got.runs[keys[string(k)]](nil, nil)
		if pressed != k {
			t.Errorf("key %s ran %q", k, pressed)
		}
	}
}
