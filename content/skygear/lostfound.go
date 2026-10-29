package skygear

import (
	"github.com/presentkim/skygear/content/skygear/island"

	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/world"
)

// worldHandler handles events of the instance world.
type worldHandler struct {
	world.NopHandler
	s *session
}

// HandleEntityDespawn collects items that fell into the void: Dragonfly
// closes an item entity below the world's lowest height, while a picked-up or
// expired item despawns above it.
func (h *worldHandler) HandleEntityDespawn(tx *world.Tx, e world.Entity) {
	ent, ok := e.(*entity.Ent)
	if !ok || e.Position().Y() >= float64(tx.Range()[0]) {
		return
	}
	if ib, ok := ent.Behaviour().(*entity.ItemBehaviour); ok {
		island.AddLost(h.s.Session, ib.Item())
	}
}
