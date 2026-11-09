package skygear

import "testing"

func TestCustomBlockCounts(t *testing.T) {
	blocks, states, perms := CustomBlockCounts()
	if blocks == 0 || states < blocks || perms < 0 {
		t.Fatalf("blocks %d, states %d, permutations %d", blocks, states, perms)
	}
	if mode := stateProbeMode; mode != "" {
		ps, pp := stateProbeCounts(mode)
		if states < ps || perms < pp {
			t.Fatalf("counts %d/%d miss the probe's %d/%d", states, perms, ps, pp)
		}
	}
}
