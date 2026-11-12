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
//
// Namespace is the one-time conversion of saves from before the rename from
// DragonFactory to Sky Gear (namespace.go); it runs before Run.
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
	// Plain is Rename for retired items, with one more change: every item
	// stack of such a name becomes a plain stack of the new item (same
	// count, no custom name, lore, damage, values or other item NBT, so it
	// stacks with a new one) wherever a stack is saved: inventories,
	// containers, item entities, lost items. Other strings of the name are
	// renamed as with Rename. A name is in Rename or in Plain, not both.
	Plain map[string]string
	// Blocks replaces whole block palette entries of sub chunks (every
	// layer): an entry whose name is a key gets Name and States, keeping
	// its version, instead of Rename (a block that changes into another
	// needs its states, or it gets the first registered ones). Only sub
	// chunk palettes are touched.
	Blocks map[string]Block
}

// Block is the name and block states of a palette entry.
type Block struct {
	Name   string
	States map[string]any
	// DropEntities says the old block had a block entity the new block has
	// no use for: the block entities at the positions of the replaced
	// blocks are removed from the world (see dropentities.go).
	DropEntities bool
}

// block returns the replacement of the palette entry name.
func (p Patch) block(name string) (Block, bool) {
	b, ok := p.Blocks[name]
	return b, ok
}

func (p Patch) rename(s string) string {
	if n, ok := p.Rename[s]; ok {
		return n
	}
	if n, ok := p.Plain[s]; ok {
		return n
	}
	if i := strings.LastIndexByte(s, ':'); i > 0 {
		if n, ok := p.Rename[s[:i]]; ok {
			return n + s[i:]
		}
		if n, ok := p.Plain[s[:i]]; ok {
			return n + s[i:]
		}
	}
	return s
}

// plain reports whether the item stacks named s become plain stacks.
func (p Patch) plain(s string) bool {
	_, ok := p.Plain[s]
	return ok
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

// renamer applies a rename to the strings of saved data: rep to tag names
// and string values, key to the value keys of item stacks, mentions says
// whether bytes hold anything to rename. lost, if set, is the renamer of the
// item NBT that JSON keeps as base64 (the lost items); nil means r itself.
type renamer struct {
	rep      func(string) string
	key      func(string) string
	mentions func([]byte) bool
	lost     *renamer
	// blocks, if set, replaces whole sub chunk palette entries.
	blocks func(name string) (string, map[string]any, bool)
	// drops, if set, says whether the block entities of a block of the
	// palette entry name are dropped.
	drops func(name string) bool
	// plain, if set, says whether the item stacks named name become plain
	// stacks of the item they are renamed to.
	plain func(name string) bool
	// disks says the item keys of the storage network disks in a save state
	// document are rewritten as well (diskkeys.go).
	disks bool
	warn  bool // log the values that mention a name but could not be rewritten
}

func (r renamer) embedded() renamer {
	if r.lost != nil {
		return *r.lost
	}
	return r
}

// newRenamer applies the patches in order.
func newRenamer(ps []Patch) renamer {
	var old [][]byte
	hasBlocks, hasPlain := false, false
	for _, p := range ps {
		for k := range p.Rename {
			old = append(old, []byte(k))
		}
		for k := range p.Plain {
			old = append(old, []byte(k))
			hasPlain = true
		}
		for k := range p.Blocks {
			old = append(old, []byte(k))
		}
		if len(p.Blocks) > 0 {
			hasBlocks = true
		}
	}
	rep := func(s string) string {
		for _, p := range ps {
			s = p.rename(s)
		}
		return s
	}
	r := renamer{rep: rep, key: rep, disks: true, mentions: func(b []byte) bool {
		for _, o := range old {
			if bytes.Contains(b, o) {
				return true
			}
		}
		return false
	}}
	if hasPlain {
		r.plain = func(name string) bool {
			for _, p := range ps {
				if p.plain(name) {
					return true
				}
				name = p.rename(name)
			}
			return false
		}
	}
	if hasBlocks {
		// Each patch in turn: Blocks if it names the entry, else Rename.
		r.blocks = func(name string) (string, map[string]any, bool) {
			var states map[string]any
			replaced := false
			for _, p := range ps {
				if b, ok := p.block(name); ok {
					name, states, replaced = b.Name, b.States, true
				} else {
					name = p.rename(name)
				}
			}
			return name, states, replaced
		}
		r.drops = func(name string) bool {
			for _, p := range ps {
				if b, ok := p.block(name); ok {
					if b.DropEntities {
						return true
					}
					name = b.Name
				} else {
					name = p.rename(name)
				}
			}
			return false
		}
	}
	return r
}
