package tinker

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/platform/account"
)

// AutoToolSetting is the player setting (account.Setting) that puts the
// tool that fits a block in the player's hand when they start breaking it
// with one that does not.
const AutoToolSetting = "auto_tool"

// fitsBlock reports whether held is a tool that mines b faster than a hand
// and gets its drops. A hammer fits the blocks it crushes.
func fitsBlock(b world.Block, held item.Stack) bool {
	if _, crushable := sgitem.HammerDrop(b); crushable && sgitem.IsHammer(held) {
		return true
	}
	t, ok := held.Item().(item.Tool)
	if !ok {
		return false
	}
	br, ok := b.(block.Breakable)
	if !ok {
		return false
	}
	info := br.BreakInfo()
	return info.Effective != nil && info.Harvestable != nil && info.Effective(t) && info.Harvestable(t)
}

// AutoTool runs equipToolFor when p turned the setting on.
func AutoTool(p *player.Player, b world.Block) {
	if account.On(p, AutoToolSetting) {
		EquipToolFor(p, b)
	}
}
