package dragonfactory

import (
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

// Requirement is an item asked for: a research's cost, an expansion's
// price.
type Requirement struct {
	Item  world.Item
	Count int
}

var oakLog = block.Log{Wood: block.OakWood()}

// itemKey identifies an item type: its name and metadata.
func itemKey(it world.Item) string {
	name, meta := it.EncodeItem()
	return fmt.Sprintf("%s:%d", name, meta)
}

// plain reports whether a stack is a plain item: no added NBT (custom
// name, lore, values such as a time-limited item's expiry). Research,
// machine recipes and prices ask for plain items; anything else is a
// different item to them.
func plain(st item.Stack) bool { return st.Comparable(item.NewStack(st.Item(), 1)) }

// blockKey identifies a block: its name without the namespace and any
// state (log axis, facing, ...).
func blockKey(b world.Block) string {
	name, _ := b.EncodeBlock()
	name = strings.TrimPrefix(name, "minecraft:")
	return strings.TrimPrefix(name, "flowing_")
}

// blockName is the Korean name of a block.
func blockName(b world.Block) string {
	k := blockKey(b)
	if n, ok := koreanNames[k]; ok {
		return n
	}
	return k
}

// blockNameKey is the client translation key of a block's name: the client
// shows it in its own language, from its vanilla texts or the pack's
// texts/*.lang for our blocks ("tile.dragonfactory:sieve.name").
func blockNameKey(b world.Block) string {
	name, _ := b.EncodeBlock()
	name = strings.TrimPrefix(name, "minecraft:")
	return "tile." + strings.TrimPrefix(name, "flowing_") + ".name"
}

// formatInt formats a number with thousands separators, e.g. 12,400.
func formatInt(v int64) string {
	s := fmt.Sprint(v)
	neg := v < 0
	if neg {
		s = s[1:]
	}
	var out []byte
	for i := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, s[i])
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}
