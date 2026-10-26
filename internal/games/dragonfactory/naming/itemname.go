package naming

import (
	"strings"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/slotdeco"
)

// ItemName returns a readable name of an item stack in lang.
func ItemName(lang string, s item.Stack) string {
	if n := s.CustomName(); n != "" {
		// The slot marks are for the slot only (docs/tech/slotdeco.md).
		return slotdeco.Strip(n)
	}
	if c, ok := s.Item().(world.CustomItem); ok {
		return CustomItemName(lang, c)
	}
	name, _ := s.Item().EncodeItem()
	name = strings.TrimPrefix(name, "minecraft:")
	if n, ok := VanillaName(lang, name); ok {
		return n
	}
	return name
}

// UnlockName is the item name of an unlock as items encode it.
func UnlockName(n string) string {
	if strings.Contains(n, ":") {
		return n
	}
	return "minecraft:" + n
}
