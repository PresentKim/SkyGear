package ui

import (
	"testing"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// TestProbeScreenCarriesNodesAndReport: the nodes go in the data, the report
// stays on the layout, and SetProbeInfo shows the screen again with the info.
func TestProbeScreenCarriesNodesAndReport(t *testing.T) {
	var got *oreLayout
	var root any
	oreShown = func(l *oreLayout, r any) { got, root = l, r }
	defer func() { oreShown = nil }()

	var line string
	ProbeScreen(nil, "probe", []any{ProbeNode("restest", "js", "x")}, func(_ *player.Player, _ *world.Tx, _, l string) { line = l }, nil)
	name, data := screenRoot(root)
	nodes, _ := data["nodes"].([]any)
	if name != "probe" || len(nodes) != 1 || nodes[0].(map[string]any)["t"] != "restest" {
		t.Errorf("%s %v", name, data)
	}
	got.probe(nil, nil, "res", "ok")
	if line != "ok" {
		t.Fatalf("report got %q", line)
	}
	SetProbeInfo(nil, "hello")
	_, data = screenRoot(root)
	if data["info"] != "hello" || len(data["nodes"].([]any)) != 1 {
		t.Errorf("after SetProbeInfo %v", data)
	}
}
