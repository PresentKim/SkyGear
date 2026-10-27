package ui

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

func TestProbeScreenCarriesNodesAndReport(t *testing.T) {
	var got *oreLayout
	var layout any
	oreShown = func(l *oreLayout, lay any) { got, layout = l, lay }
	defer func() { oreShown = nil }()

	var line string
	ProbeScreen(nil, "probe", []any{ProbeNode("restest", "js", "x")}, func(_ *player.Player, _ *world.Tx, _, l string) { line = l }, nil)
	b, _ := json.Marshal(layout)
	for _, want := range []string{`"t":"restest"`, `"bind":"probeInfo"`} {
		if !strings.Contains(string(b), want) {
			t.Errorf("layout lacks %s", want)
		}
	}
	got.probe(nil, nil, "res", "ok")
	if line != "ok" {
		t.Fatalf("report got %q", line)
	}
}
