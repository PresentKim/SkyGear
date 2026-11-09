package skygear

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log/slog"
	"maps"
	"sort"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/assets"
)

// probeNames are the names of the probe blocks.
func probeNames() []string {
	var out []string
	for _, c := range stateProbeNameCounts(stateProbeMode) {
		out = append(out, c.name)
	}
	return out
}

// TestStateProbeOff: baseline. With SKYGEAR_STATE_PROBE unset or off, no probe
// block and no layer probe block is registered, the helper says so, and no
// state is added.
func TestStateProbeOff(t *testing.T) {
	if stateProbeMode != assets.StateProbeOff {
		t.Skipf("probe mode %q", stateProbeMode)
	}
	if b, ok := stateProbeBlock(0, 1); ok || b != nil {
		t.Errorf("stateProbeBlock = %v, %v with the probe off", b, ok)
	}
	for _, b := range world.Blocks() {
		if _, ok := b.(StateProbe); ok {
			t.Fatalf("%v is registered with the probe off", b)
		}
		if name, _ := b.EncodeBlock(); strings.Contains(name, "state_probe") || strings.Contains(name, "layer_probe") {
			t.Fatalf("%s is registered with the probe off", name)
		}
	}
	for name := range world.DefaultBlockRegistry.CustomBlocks() {
		if strings.Contains(name, "state_probe") || strings.Contains(name, "layer_probe") {
			t.Fatalf("custom block %s with the probe off", name)
		}
	}
	if s, p := stateProbeCounts(stateProbeMode); s != 0 || p != 0 {
		t.Errorf("counts %d, %d with the probe off", s, p)
	}
}

// TestStateProbeRegistered: with pair2 on (go test with
// SKYGEAR_STATE_PROBE=pair2) both blocks have all their 37,249 states, in the
// order of their values, and the helper gives them.
func TestStateProbeRegistered(t *testing.T) {
	if stateProbeMode == assets.StateProbeOff {
		t.Skip("probe off")
	}
	states := map[string][]map[string]any{}
	for _, b := range world.Blocks() {
		if sp, ok := b.(StateProbe); ok {
			name, props := b.EncodeBlock()
			states[name] = append(states[name], props)
			if got, ok := stateProbeBlock(sp.Layer, sp.V); !ok || got != b {
				t.Errorf("stateProbeBlock(%d, %d) = %v, %v", sp.Layer, sp.V, got, ok)
			}
		}
	}
	sum := 0
	for l := range stateProbeLayers {
		name := stateProbeBlockName(l)
		sum += stateProbePair2Values
		if len(states[name]) != stateProbePair2Values {
			t.Errorf("%s: %d states, want %d", name, len(states[name]), stateProbePair2Values)
		}
		// the states of a name come in the order of States: 0, 1, 2 ...
		for i, props := range states[name] {
			if want := map[string]any{stateProbeState: int32(i)}; !maps.Equal(props, want) {
				t.Fatalf("%s: state %d is %v, want %v", name, i, props, want)
			}
		}
		if _, ok := stateProbeBlock(l, stateProbePair2Values); ok {
			t.Errorf("the value past the last is a probe block of layer %d", l)
		}
	}
	if _, ok := stateProbeBlock(stateProbeLayers, 0); ok {
		t.Error("layer 2 is a probe block")
	}
	wantStates, wantPerms := stateProbeCounts(stateProbeMode)
	if wantStates != sum {
		t.Errorf("%d states, want %d", wantStates, sum)
	}
	var perms int
	for _, name := range probeNames() {
		cb, ok := world.DefaultBlockRegistry.CustomBlocks()[name]
		if !ok {
			t.Fatalf("%s is not a custom block", name)
		}
		perms += len(cb.(block.Permutable).Permutations())
	}
	if perms != wantPerms {
		t.Errorf("%d permutations, want %d", perms, wantPerms)
	}
}

// probeFourGeometry reads state_probe_four.geo.json: the bones of each
// geometry and the material instance of each bone's faces.
func probeFourGeometry(t *testing.T) (bones map[string][]string, instance map[string]string) {
	t.Helper()
	var geo struct {
		Geometry []struct {
			Description struct {
				Identifier string `json:"identifier"`
			} `json:"description"`
			Bones []struct {
				Name  string `json:"name"`
				Cubes []struct {
					UV map[string]struct {
						MaterialInstance string `json:"material_instance"`
					} `json:"uv"`
				} `json:"cubes"`
			} `json:"bones"`
		} `json:"minecraft:geometry"`
	}
	readPackJSON(t, "models/blocks/state_probe_four.geo.json", &geo)
	bones, instance = map[string][]string{}, map[string]string{}
	for _, g := range geo.Geometry {
		for _, b := range g.Bones {
			bones[g.Description.Identifier] = append(bones[g.Description.Identifier], b.Name)
			for _, c := range b.Cubes {
				for _, f := range c.UV {
					if prev, ok := instance[b.Name]; ok && prev != f.MaterialInstance {
						t.Fatalf("bone %s has faces of %s and %s", b.Name, prev, f.MaterialInstance)
					}
					instance[b.Name] = f.MaterialInstance
				}
			}
		}
	}
	return bones, instance
}

// probeNetworkHash is Dragonfly's network block hash of a state
// (world.networkBlockHash): FNV-1a 32 of its NBT, the states sorted by name.
func probeNetworkHash(name string, props map[string]any) uint32 {
	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var d []byte
	str := func(s string) { d = binary.LittleEndian.AppendUint16(d, uint16(len(s))); d = append(d, s...) }
	d = append(d, 10, 0, 0, 8)
	str("name")
	str(name)
	d = append(d, 10)
	str("states")
	for _, k := range keys {
		switch v := props[k].(type) {
		case string:
			d = append(d, 8)
			str(k)
			str(v)
		case uint8, int8, bool:
			b := byte(0)
			switch v := v.(type) {
			case uint8:
				b = v
			case int8:
				b = byte(v)
			case bool:
				if v {
					b = 1
				}
			}
			d = append(d, 1)
			str(k)
			d = append(d, b)
		case int16:
			d = append(d, 2)
			str(k)
			d = binary.LittleEndian.AppendUint16(d, uint16(v))
		case uint16:
			d = append(d, 2)
			str(k)
			d = binary.LittleEndian.AppendUint16(d, v)
		case int32:
			d = append(d, 3)
			str(k)
			d = binary.LittleEndian.AppendUint32(d, uint32(v))
		case uint32:
			d = append(d, 3)
			str(k)
			d = binary.LittleEndian.AppendUint32(d, v)
		default:
			panic(fmt.Sprintf("state %s of %s: %T", k, name, v))
		}
	}
	d = append(d, 0, 0)
	h := uint32(2166136261)
	for _, b := range d {
		h ^= uint32(b)
		h *= 16777619
	}
	return h
}

// TestStateProbeNetworkHashes: Dragonfly panics at start when two states share
// a network block hash. The states of the two pair2 blocks share none with each
// other or with the other blocks registered; if a new block meets one, change
// stateProbePair2Salt.
func TestStateProbeNetworkHashes(t *testing.T) {
	others := map[uint32]string{}
	for _, b := range world.Blocks() {
		if _, ok := b.(StateProbe); ok {
			continue
		}
		name, props := b.EncodeBlock()
		others[probeNetworkHash(name, props)] = name
	}
	if len(others) < 10000 {
		t.Fatalf("%d other states", len(others))
	}
	seen := map[uint32]bool{}
	for l := range stateProbeLayers {
		name := stateProbeBlockName(l)
		for v := range stateProbePair2Values {
			h := probeNetworkHash(name, map[string]any{stateProbeState: int32(v)})
			if other, ok := others[h]; ok || seen[h] {
				t.Fatalf("%s %d meets the network hash of %q (stateProbePair2Salt %d)", name, v, other, stateProbePair2Salt)
			}
			seen[h] = true
		}
	}
}

// logRecords runs f with a logger that writes its lines (message and
// attributes, level first) into the returned buffer.
func logRecords(f func(log *slog.Logger)) string {
	var buf bytes.Buffer
	f(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	return buf.String()
}

// TestStateProbeCapWarning: the log lists each probe block name with its
// states, and warns of each name over 65,536 states: pair2 has none over.
func TestStateProbeCapWarning(t *testing.T) {
	counts := stateProbeNameCounts(assets.StateProbePair2)
	out := logRecords(func(log *slog.Logger) { logBlockNameCounts(log, counts) })
	if warns, infos := strings.Count(out, "level=WARN"), strings.Count(out, "level=INFO"); warns != 0 || infos != len(counts) {
		t.Errorf("pair2: %d warnings and %d name lines, want 0 and %d:\n%s", warns, infos, len(counts), out)
	}
	// a block that is not a probe is warned of but not listed
	out = logRecords(func(log *slog.Logger) {
		logBlockNameCounts(log, []blockNameCount{{name: "skygear:big", states: 70000}, {name: "skygear:ok", states: 65536}})
	})
	if strings.Count(out, "level=WARN") != 1 || strings.Contains(out, "level=INFO") || !strings.Contains(out, "skygear:big") {
		t.Errorf("log:\n%s", out)
	}
	// the registry of this run: the custom blocks line, and no name over the cap
	out = logRecords(LogCustomBlockCounts)
	if !strings.Contains(out, "max_states_per_name=") {
		t.Errorf("custom blocks line has no max states per name:\n%s", out)
	}
	if strings.Contains(out, "level=WARN") {
		t.Errorf("a custom block is over the cap:\n%s", out)
	}
}
