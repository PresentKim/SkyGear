package naming

import (
	"strings"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/slotdeco"
)

// ItemName returns a readable name of an item stack in lang.
func ItemName(lang string, s item.Stack) string {
	if n := s.CustomName(); n != "" {
		// The slot marks are for the slot only.
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

// ItemNameKey is the client translation key of the name of an item stack:
// "item.<id>.name" for ours (the pack's texts/*.lang), the official key of a
// vanilla item, "" for a stack with a name of its own or a vanilla item the
// official texts do not name.
func ItemNameKey(s item.Stack) string {
	if s.CustomName() != "" {
		return ""
	}
	name, _ := s.Item().EncodeItem()
	if _, ok := s.Item().(world.CustomItem); ok {
		return "item." + name + ".name"
	}
	name = strings.TrimPrefix(name, "minecraft:")
	if key, ok := vanillaKeys[name]; ok {
		return key
	}
	if !i18n.Has("vanilla." + name) {
		return ""
	}
	return "tile." + name + ".name"
}

// UIItemName is ItemName as a client span (i18n.ClientKey) for the forms and
// DDUI screens: the Ore UI page shows the client's own name of the item, in
// any language the game has.
func UIItemName(lang string, s item.Stack) string {
	return i18n.ClientKey(ItemNameKey(s), ItemName(lang, s))
}

// UnlockName is the item name of an unlock as items encode it.
func UnlockName(n string) string {
	if strings.Contains(n, ":") {
		return n
	}
	return "minecraft:" + n
}
