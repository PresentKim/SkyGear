package dimension

import (
	"github.com/df-mc/dragonfly/server/player"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// RetireItems fixes the relics p carries (session.Second and Join).
func RetireItems(p *player.Player) {
	inv := p.Inventory()
	for slot, it := range inv.Slots() {
		if fixed := sgitem.FixRelic(it); !it.Empty() && !fixed.Equal(it) {
			_ = inv.SetItem(slot, fixed)
		}
	}
}
