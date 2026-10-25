// Package slottext has the text pieces that the form slots, the WAILA panel
// and the HUD rows build in the same way.
package slottext

import (
	"strings"
	"unicode/utf8"

	"github.com/df-mc/dragonfly/server/world"
)

// Pad cuts s to at most n bytes (whole characters) and fills it with '~' up
// to n bytes; '~' in s is dropped.
func Pad(s string, n int) string {
	var b strings.Builder
	for _, r := range s {
		if r == '~' {
			continue
		}
		if b.Len()+utf8.RuneLen(r) > n {
			break
		}
		b.WriteRune(r)
	}
	return b.String() + strings.Repeat("~", n-b.Len())
}

// Unique returns s with invisible "§r"s added until it is not in seen, and
// adds the result to seen.
func Unique(seen map[string]bool, s string) string {
	for seen[s] {
		s += "§r"
	}
	seen[s] = true
	return s
}

// ItemAux is the item's runtime ID times 65536, how the client's item
// renderer and the packs read an item; false for an unregistered item.
func ItemAux(it world.Item) (int64, bool) {
	rid, _, ok := world.ItemRuntimeID(it)
	if !ok {
		return 0, false
	}
	return int64(rid) * 65536, true
}
