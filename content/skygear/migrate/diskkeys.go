package migrate

import (
	"encoding/base64"
	"encoding/json"
	"math"
	"slices"
	"strconv"

	"github.com/presentkim/skygear/common/nbtcanon"
	"github.com/presentkim/skygear/common/nbtrewrite"
)

// ItemKey, if set, is the storage network key of the stack whose little
// endian item NBT is itemNBT, as the running server computes it (netstore
// sets it); ok is false for NBT it cannot read. Without it a rewritten key is
// the canonical form of the rewritten NBT, which is the server's key for a
// stack with no block or other data the server adds when it writes the stack.
var ItemKey func(itemNBT []byte) (key string, ok bool)

// disksField is the save state field holding the storage network disks: disk
// id -> item key -> count. A key is the base64 of the canonical form
// (package nbtcanon) of the NBT of one item, so a renamed item needs a new
// key, and keys that become one add their counts.
const disksField = "net_disks"

// rewriteDisks rewrites the item keys of the disks in doc, a decoded save
// state document, and reports whether any changed. A disk that cannot be
// rewritten without losing items stays as it is.
func rewriteDisks(doc any, r renamer) bool {
	top, ok := doc.(map[string]any)
	if !ok {
		return false
	}
	disks, ok := top[disksField].(map[string]any)
	if !ok {
		return false
	}
	changed := false
	for id, d := range disks {
		if m, ok := d.(map[string]any); ok {
			if nm, ok := rewriteDisk(m, r); ok {
				disks[id] = nm
				changed = true
			}
		}
	}
	return changed
}

// rewriteDisk is the disk m with its keys rewritten and the counts of keys
// that became one added. ok is false, and the disk to be left as it is, when
// no key changes or a count is not an integer or the sum would overflow. The
// total of a disk never changes, so its capacity, which the save does not
// record, still holds exactly what it held.
func rewriteDisk(m map[string]any, r renamer) (map[string]any, bool) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	out := make(map[string]any, len(m))
	counts := make(map[string]int64, len(m))
	renamed := false
	for _, k := range keys {
		n, ok := m[k].(json.Number)
		if !ok {
			return nil, false
		}
		c, err := n.Int64()
		if err != nil {
			return nil, false
		}
		nk, ok := r.diskKey(k)
		if ok {
			renamed = true
		} else {
			nk = k
		}
		prev, dup := counts[nk]
		if !dup {
			out[nk], counts[nk] = n, c
			continue
		}
		// Only positive counts hold items (a disk drops the others when it
		// loads), so only they are added.
		a, b := max(prev, 0), max(c, 0)
		if a > math.MaxInt64-b {
			return nil, false
		}
		counts[nk] = a + b
		out[nk] = json.Number(strconv.FormatInt(a+b, 10))
	}
	return out, renamed
}

// diskKey is the key a disk key becomes under the renamer; ok is false when it
// stays (not a key, or no patched name in the item).
func (r renamer) diskKey(k string) (string, bool) {
	raw, err := base64.RawStdEncoding.DecodeString(k)
	if err != nil {
		return "", false
	}
	enc, err := nbtcanon.NBT(raw)
	if err != nil || !r.mentions(enc) {
		return "", false
	}
	nb, err := nbtrewrite.NBTStacks(enc, r.rep, r.key, r.plain)
	if err != nil {
		return "", false
	}
	nk := ""
	if ItemKey != nil {
		nk, _ = ItemKey(nb)
	}
	if nk == "" {
		nk = base64.RawStdEncoding.EncodeToString(nbtcanon.Encode(nb))
	}
	return nk, nk != k
}
