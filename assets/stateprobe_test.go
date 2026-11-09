package assets

import (
	"bytes"
	"io/fs"
	"strings"
	"testing"
)

func fileList(t *testing.T, fsys fs.FS) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			out[p] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestParseStateProbeMode(t *testing.T) {
	for in, want := range map[string]string{
		"": StateProbeOff, "off": StateProbeOff, "perm": StateProbePerm, " Bones ": StateProbeBones, "x": StateProbeOff,
		"single4": StateProbeSingle4, "SAME4": StateProbeSame4, "all4": StateProbeAll4, "all": StateProbeOff, "ALL4Split": StateProbeAll4Split, " Pair2": StateProbePair2,
		"pair2ab": StateProbePair2AB, "PAIR2BITS": StateProbePair2Bits, "pair2a": StateProbeOff,
	} {
		if got := parseStateProbeMode(in); got != want {
			t.Errorf("parseStateProbeMode(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestStateProbeFilesOnlyInTheirMode: baseline. With the probe off the
// skygear pack has no state_probe file and every other file is the embedded
// one, byte for byte; a mode adds only its own geometry file; other packs
// are never filtered.
func TestStateProbeFilesOnlyInTheirMode(t *testing.T) {
	raw, err := fs.Sub(files, "resourcepack/skygear")
	if err != nil {
		t.Fatal(err)
	}
	all := fileList(t, raw)
	probeFiles := map[string]bool{}
	for _, name := range stateProbeFiles {
		if !all[name] {
			t.Fatalf("%s is not in the pack folder", name)
		}
		probeFiles[name] = true
	}
	if len(probeFiles) != 3 {
		t.Fatalf("probe files %v", probeFiles)
	}
	rest := map[string]bool{}
	for p := range all {
		if !strings.Contains(p, "state_probe") {
			rest[p] = true
		}
	}
	if len(rest) != len(all)-len(probeFiles) {
		t.Fatalf("state_probe files other than %v", stateProbeFiles)
	}
	for _, mode := range []string{StateProbeOff, StateProbePerm, StateProbeBones, StateProbeSingle4, StateProbeSame4, StateProbeAll4, StateProbeAll4Split, StateProbePair2, StateProbePair2AB, StateProbePair2Bits} {
		pack, err := packFor("skygear", mode)
		if err != nil {
			t.Fatal(err)
		}
		got := fileList(t, pack)
		want := map[string]bool{}
		for p := range rest {
			want[p] = true
		}
		if name, ok := stateProbeFiles[mode]; ok {
			want[name] = true
		}
		if len(got) != len(want) {
			t.Errorf("mode %q: %d files, want %d", mode, len(got), len(want))
		}
		for p := range want {
			if !got[p] {
				t.Errorf("mode %q: %s is missing", mode, p)
				continue
			}
			a, _ := fs.ReadFile(pack, p)
			b, _ := fs.ReadFile(raw, p)
			if !bytes.Equal(a, b) {
				t.Errorf("mode %q: %s differs from the embedded file", mode, p)
			}
		}
		for p := range got {
			if !want[p] {
				t.Errorf("mode %q: %s should not ship", mode, p)
			}
		}
		for _, name := range stateProbeFiles {
			if _, err := fs.Stat(pack, name); (err == nil) != (name == stateProbeFiles[mode]) {
				t.Errorf("mode %q: Stat(%s) = %v", mode, name, err)
			}
		}
	}
	other, err := packFor("sgui", StateProbeOff)
	if err != nil {
		t.Fatal(err)
	}
	if len(fileList(t, other)) == 0 {
		t.Error("sgui pack is empty")
	}
}
