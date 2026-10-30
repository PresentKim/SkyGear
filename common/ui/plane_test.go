package ui

import (
	"encoding/json"
	"testing"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// TestPlaneKeys (plane_close_keys): the keys of a PlaneMenu are on the root
// as ids the screen runs, a key without a run is not answered, and Close runs.
func TestPlaneKeys(t *testing.T) {
	var got *oreLayout
	var layout any
	oreShown = func(l *oreLayout, lay any) { got, layout = l, lay }
	defer func() { oreShown = nil }()

	var pressed Key
	key := func(k Key) func(*player.Player, *world.Tx) {
		return func(*player.Player, *world.Tx) { pressed = k }
	}
	closed := false
	spotRun := false
	PlaneMenu(nil, Plane{
		FOV: 70, Distance: 10,
		Spots:   []PlaneSpot{{X: 1, Z: 2, W: 3, H: 4, Tip: "t", Run: func(*player.Player, *world.Tx) { spotRun = true }}},
		Actions: []Option{{Label: "go", Run: nopRun}},
		Close:   func(*player.Player, *world.Tx) { closed = true },
		Keys:    map[Key]func(*player.Player, *world.Tx){KeyUp: key(KeyUp), KeyOK: key(KeyOK), KeyLeft: nil},
	})
	if name, _ := screenRoot(layout); name != "plane" {
		t.Fatalf("screen %q", name)
	}
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
	got.close(nil, nil)
	if !closed {
		t.Error("Close did not run")
	}
	b, _ := json.Marshal(layout.(map[string]any)["data"])
	var d planeData
	if err := json.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	if d.FOV != 70 || d.Dist != 10 || len(d.Spots) != 1 || d.Spots[0].At != [4]float64{1, 2, 3, 4} || len(d.Actions) != 1 {
		t.Fatalf("data %s", b)
	}
	got.runs[d.Spots[0].ID](nil, nil)
	if !spotRun {
		t.Error("spot did not run")
	}
}
