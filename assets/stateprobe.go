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
// permutations that switch the bones of one geometry.
const StateProbeEnv = "SKYGEAR_STATE_PROBE"

// The probe modes.
const (
	StateProbeOff   = ""
	StateProbePerm  = "perm"
	StateProbeBones = "bones"
)

// stateProbeFiles are the pack files of the probe by mode, written by
// tools/stateprobegeo. They sit in the skygear pack's models/blocks, and
// Pack leaves them out unless the mode needs them.
var stateProbeFiles = map[string]string{
	StateProbePerm:  "models/blocks/state_probe_shape.geo.json",
	StateProbeBones: "models/blocks/state_probe_bones.geo.json",
}

var stateProbeMode = sync.OnceValue(func() string { return parseStateProbeMode(os.Getenv(StateProbeEnv)) })

// StateProbeMode is the mode of the state count probe, read once from
// SKYGEAR_STATE_PROBE: StateProbeOff (also for any other value),
// StateProbePerm or StateProbeBones.
func StateProbeMode() string { return stateProbeMode() }

func parseStateProbeMode(v string) string {
	switch v = strings.ToLower(strings.TrimSpace(v)); v {
	case StateProbePerm, StateProbeBones:
		return v
	}
	return StateProbeOff
}

// withoutProbeFiles is fsys without the probe files the mode does not ship:
// all of them when the probe is off, the other mode's when it is on.
func withoutProbeFiles(fsys fs.FS, mode string) fs.FS {
	hidden := map[string]bool{}
	for m, name := range stateProbeFiles {
		if m != mode {
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
