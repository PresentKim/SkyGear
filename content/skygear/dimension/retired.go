package dimension

import (
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/farm"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// RetireItems turns the seeds of essence blooms p carries into redstone bloom
// seeds and fixes the relics (session.Second and Join).
func RetireItems(_ *core.Session, _ *world.Tx, p *player.Player) {
	farm.RetireSeeds(p)
	inv := p.Inventory()
	for slot, it := range inv.Slots() {
		if fixed := sgitem.FixRelic(it); !it.Empty() && !fixed.Equal(it) {
			_ = inv.SetItem(slot, fixed)
		}
	}
}
