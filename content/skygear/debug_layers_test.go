package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// TestDebugLayers: each /sgdebug layers mode leaves the probe blocks on the
// layers it says, and alt, info and clear work on the last target.
func TestDebugLayers(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s) // eye at (0.5, 65.6, 2.5), facing south: the target is 0 65 4
	pos := cube.Pos{0, 65, 4}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		run := func(mode string) *cmd.Output {
			t.Helper()
			var o cmd.Output
			debugLayers{adminBase: adminBase{g: s.g}, Mode: layersEnum(mode)}.Run(p, &o, tx)
			if len(o.Errors()) > 0 {
				t.Fatalf("%s: %v", mode, o.Errors())
			}
			if len(o.Messages()) != 1 {
				t.Fatalf("%s: %d lines, want 1", mode, len(o.Messages()))
			}
			return &o
		}
		want := func(mode string, layers ...world.Block) {
			t.Helper()
			for l := 0; l < layerCount; l++ {
				var w world.Block = block.Air{}
				if l < len(layers) && layers[l] != nil {
					w = layers[l]
				}
				if got := tx.BlockLayer(pos, uint8(l)); got != w {
					t.Errorf("%s: layer %d = %#v, want %#v", mode, l, got, w)
				}
			}
		}
		probe := func(kind, alt int) world.Block { return LayerProbe{Kind: kind, Alt: alt} }

		// Before any target, alt, info and clear have nothing to work on.
		var o cmd.Output
		debugLayers{adminBase: adminBase{g: s.g}, Mode: "info"}.Run(p, &o, tx)
		if len(o.Errors()) != 1 {
			t.Fatalf("info without a target: %v", o.Errors())
		}

		run("a")
		want("a", probe(0, 0), probe(1, 0), probe(2, 0))
		run("c")
		want("c", probe(0, 0), nil, probe(2, 0), probe(1, 0))
		if n := tx.SubChunkLayers(pos); n < 4 {
			t.Errorf("c: %d layers, want >= 4", n)
		}
		run("alt")
		want("alt", probe(0, 1), nil, probe(2, 1), probe(1, 1))
		run("alt")
		want("alt again", probe(0, 0), nil, probe(2, 0), probe(1, 0))
		run("b")
		want("b", nil, probe(1, 0), probe(2, 0))
		run("d")
		want("d", probe(probeHidden, 0), probe(1, 0))
		if tx.Block(pos) == (block.Air{}) {
			t.Errorf("d: layer 0 is air, want a block that draws nothing")
		}
		run("alt1")
		want("alt1", probe(probeHidden, 0), probe(1, 1))
		run("alt1")
		want("alt1 again", probe(probeHidden, 0), probe(1, 0))
		run("a")
		run("info")
		want("info", probe(0, 0), probe(1, 0), probe(2, 0))
		run("clear")
		want("clear")
		if tx.Block(pos) != (block.Air{}) {
			t.Errorf("clear: layer 0 is %#v", tx.Block(pos))
		}
	})
}

// TestLayerProbeStates: every combination of the states is a registered
// block that encodes back to itself, and each has a look.
func TestLayerProbeStates(t *testing.T) {
	br := world.DefaultBlockRegistry
	n := 0
	for k := 0; k < probeKinds; k++ {
		for a := 0; a < probeAlts; a++ {
			b := LayerProbe{Kind: k, Alt: a}
			name, props := b.EncodeBlock()
			got, ok := br.BlockByName(name, props)
			if !ok || got != b {
				t.Errorf("kind %d alt %d: registered as %#v (%v)", k, a, got, ok)
			}
			n++
		}
	}
	if got := len(LayerProbe{}.Permutations()); got != n {
		t.Errorf("%d permutations, want %d", got, n)
	}
}
