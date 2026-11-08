package assets

import (
	"io/fs"
	"os"
	"strings"
	"sync"
)

// StateProbeEnv names the environment variable that turns on the state count
// probe for a manual client test (content/skygear/stateprobe.go): unset or
// "off" is the game as it is, "perm" adds about 1,150 conduit-like block
// states with one permutation each, "bones" the same states with a few
// permutations that switch the bones of one geometry; "single4", "same4" and
// "all4" model four conduit kinds (network added) with the bones technique,
// all4 drawing any two kinds in one block (443,906 states); "all4split" is
// all4 with the states split over blocks of at most 65,536 states each.
const StateProbeEnv = "SKYGEAR_STATE_PROBE"

// The probe modes.
const (
	StateProbeOff   = ""
	StateProbePerm  = "perm"
	StateProbeBones = "bones"
	// the 4-kind modes: one kind a block value, same-faces pairs on layer 1,
	// any pairs on both layers
	StateProbeSingle4 = "single4"
	StateProbeSame4   = "same4"
	StateProbeAll4    = "all4"
	// StateProbeAll4Split is all4 with each layer split into 7 block names of
	// at most 65,536 states (the client may know no more of a name).
	StateProbeAll4Split = "all4split"
)

// stateProbeFiles are the pack files of the probe by mode, written by
// tools/stateprobegeo. They sit in the skygear pack's models/blocks, and
// Pack leaves them out unless the mode needs them. The 4-kind modes share one.
var stateProbeFiles = map[string]string{
	StateProbePerm:    "models/blocks/state_probe_shape.geo.json",
	StateProbeBones:   "models/blocks/state_probe_bones.geo.json",
	StateProbeSingle4: "models/blocks/state_probe_four.geo.json",
	StateProbeSame4:   "models/blocks/state_probe_four.geo.json",
	StateProbeAll4:    "models/blocks/state_probe_four.geo.json",

	StateProbeAll4Split: "models/blocks/state_probe_four.geo.json",
}

var stateProbeMode = sync.OnceValue(func() string { return parseStateProbeMode(os.Getenv(StateProbeEnv)) })

// StateProbeMode is the mode of the state count probe, read once from
// SKYGEAR_STATE_PROBE: StateProbeOff (also for any other value),
// StateProbePerm, StateProbeBones, StateProbeSingle4, StateProbeSame4,
// StateProbeAll4 or StateProbeAll4Split.
func StateProbeMode() string { return stateProbeMode() }

func parseStateProbeMode(v string) string {
	switch v = strings.ToLower(strings.TrimSpace(v)); v {
	case StateProbePerm, StateProbeBones, StateProbeSingle4, StateProbeSame4, StateProbeAll4, StateProbeAll4Split:
		return v
	}
	return StateProbeOff
}

// withoutProbeFiles is fsys without the probe files the mode does not ship:
// all of them when the probe is off, the other modes' when it is on.
func withoutProbeFiles(fsys fs.FS, mode string) fs.FS {
	hidden := map[string]bool{}
	for _, name := range stateProbeFiles {
		if name != stateProbeFiles[mode] {
			hidden[name] = true
		}
	}
	return hideFS{fsys, hidden}
}

// hideFS hides files of an fs.FS from opening, stat and listing.
type hideFS struct {
	fs.FS
	hidden map[string]bool
}

func (h hideFS) Open(name string) (fs.File, error) {
	if h.hidden[name] {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return h.FS.Open(name)
}

func (h hideFS) Stat(name string) (fs.FileInfo, error) {
	if h.hidden[name] {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
	}
	return fs.Stat(h.FS, name)
}

func (h hideFS) ReadFile(name string) ([]byte, error) {
	if h.hidden[name] {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return fs.ReadFile(h.FS, name)
}

func (h hideFS) ReadDir(name string) ([]fs.DirEntry, error) {
	entries, err := fs.ReadDir(h.FS, name)
	if err != nil {
		return nil, err
	}
	kept := entries[:0:0]
	for _, e := range entries {
		p := e.Name()
		if name != "." {
			p = name + "/" + p
		}
		if !h.hidden[p] {
			kept = append(kept, e)
		}
	}
	return kept, nil
}
