package assets

import (
	"io/fs"
	"os"
	"strings"
	"sync"
)

// StateProbeEnv names the environment variable that turns on the state count
// probe for a manual client test (content/skygear/stateprobe.go): unset or
// "off" is the game as it is, "pair2" is two blocks of two conduit kinds each
// (energy + item, fluid + network; any tier and faces), 37,249 states a block,
// the layout chosen for conduits drawn as blocks.
const StateProbeEnv = "SKYGEAR_STATE_PROBE"

// The probe modes.
const (
	StateProbeOff = ""
	// StateProbePair2 is two fixed pair blocks (energy + item, fluid + network),
	// each drawing its two kinds with any tier and faces (37,249 states a block).
	StateProbePair2 = "pair2"
)

// stateProbeFiles are the pack files of the probe by mode, written by
// tools/stateprobegeo. They sit in the skygear pack's models/blocks, and
// Pack leaves them out unless the mode needs them.
var stateProbeFiles = map[string]string{
	StateProbePair2: "models/blocks/state_probe_four.geo.json",
}

var stateProbeMode = sync.OnceValue(func() string { return parseStateProbeMode(os.Getenv(StateProbeEnv)) })

// StateProbeMode is the mode of the state count probe, read once from
// SKYGEAR_STATE_PROBE: StateProbeOff (also for any other value) or
// StateProbePair2.
func StateProbeMode() string { return stateProbeMode() }

func parseStateProbeMode(v string) string {
	if v = strings.ToLower(strings.TrimSpace(v)); v == StateProbePair2 {
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
