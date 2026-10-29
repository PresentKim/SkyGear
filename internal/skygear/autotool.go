package skygear

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/internal/account"
	"github.com/presentkim/skygear/internal/skygear/sgitem"
)

// autoToolSetting is the player setting (account.Setting) that puts the
// tool that fits a block in the player's hand when they start breaking it
// with one that does not.
const autoToolSetting = "auto_tool"

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

// equipToolFor puts the first tool in p's inventory that fits b in p's hand,
// moving what p held to that tool's slot, when the held item does not fit b.
func equipToolFor(p *player.Player, b world.Block) {
	held, off := p.HeldItems()
	if fitsBlock(b, held) {
		return
	}
	inv := p.Inventory()
	for slot, st := range inv.Slots() {
		if st.Empty() || !fitsBlock(b, st) {
			continue
		}
		if sgitem.IsHammer(st) {
			// A hammer turns the drops into something else: never pick it
			// for the player.
			continue
		}
		_ = inv.SetItem(slot, held)
		p.SetHeldItems(st, off)
		return
	}
}

// autoTool runs equipToolFor when p turned the setting on.
func autoTool(p *player.Player, b world.Block) {
	if account.On(p, autoToolSetting) {
		equipToolFor(p, b)
	}
}
