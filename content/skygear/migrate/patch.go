// Package migrate brings saved data up to the format of the running release.
// Each save records the SkyGear release its data is in (save_state
// VersionKey), and the data outside saves (accounts, achievements, counters)
// records one in data_versions. At startup Run applies to each the Patches
// newer than its record, oldest first, after copying it to the backup
// folder, and records the running release.
//
// A change to saved data (a renamed block or item, a moved key) gets a new
// patch at the end of Patches with the release it ships in. A released
// patch is never edited.
package migrate

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// Patch is one change to saved data.
type Patch struct {
	// Version is the release (year.month.build) that brings the change:
	// data recorded at an older release gets the patch.
	Version string
	Name    string
	// Rename maps saved strings to their new values: a string equal to a
	// key, or a key followed by ":" and an item meta, in block palettes,
	// NBT, JSON and text anywhere in the worlds and the database.
	Rename map[string]string
}

func (p Patch) rename(s string) string {
	if n, ok := p.Rename[s]; ok {
		return n
	}
	if i := strings.LastIndexByte(s, ':'); i > 0 {
		if n, ok := p.Rename[s[:i]]; ok {
			return n + s[i:]
		}
	}
	return s
}

// Latest is the version of the newest patch.
func Latest() string {
	if len(Patches) == 0 {
		return ""
	}
	return Patches[len(Patches)-1].Version
}

// Stamp is the version recorded for data written by release. A build made
// between releases reports the last release but already writes the data of
// the coming one, so the newest patch's version wins when it is newer.
func Stamp(release string) string {
	if Compare(Latest(), release) > 0 {
		return Latest()
	}
	return release
}

// pending are the patches newer than version, oldest first.
func pending(version string) []Patch {
	var out []Patch
	for _, p := range Patches {
		if Compare(p.Version, version) > 0 {
			out = append(out, p)
		}
	}
	return out
}

// Compare orders two year.month.build versions; "" is older than any.
func Compare(a, b string) int {
	pa, pb := parts(a), parts(b)
	for i := range 3 {
		if pa[i] != pb[i] {
			if pa[i] < pb[i] {
				return -1
			}
			return 1
		}
	}
	return 0
}

func parts(v string) [3]int {
	var out [3]int
	for i, s := range strings.SplitN(v, ".", 3) {
		out[i], _ = strconv.Atoi(s)
	}
	return out
}

// valid reports whether v is year.month.build.
func valid(v string) error {
	p := strings.Split(v, ".")
	if len(p) != 3 {
		return fmt.Errorf("version %q is not year.month.build", v)
	}
	for _, s := range p {
		if n, err := strconv.Atoi(s); err != nil || n <= 0 {
			return fmt.Errorf("version %q is not year.month.build", v)
		}
	}
	return nil
}

// renamer applies a run of patches to one string.
type renamer struct {
	patches []Patch
	old     [][]byte
}

func newRenamer(ps []Patch) renamer {
	r := renamer{patches: ps}
	for _, p := range ps {
		for k := range p.Rename {
			r.old = append(r.old, []byte(k))
		}
	}
	return r
}

func (r renamer) rep(s string) string {
	for _, p := range r.patches {
		s = p.rename(s)
	}
	return s
}

// mentions reports whether b holds any of the renamed strings.
func (r renamer) mentions(b []byte) bool {
	for _, o := range r.old {
		if bytes.Contains(b, o) {
			return true
		}
	}
	return false
}
