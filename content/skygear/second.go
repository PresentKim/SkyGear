package skygear

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/advancement"
)

// Second runs the crops, the fairy, the advancements and the operators'
// load lines (instance.Session.Second).
func (s *session) Second(tx *world.Tx) {
	s.growCrops(tx)
	held := advancement.InventoryNames(tx)
	s.checkAdvancementsHeld(tx, held)
	s.tickBarrels(tx)
	s.tickComposters(tx)
	s.tickFairy(tx)
	s.tickEnvironment(tx)
	s.tickWave(tx)
	s.tickToolEffects(tx)
	s.tickMonitor(tx)
	s.tickPlants(tx)
	s.updateBoards(tx)
	countAll(tx, "play_seconds", 1)
	grantCollection(held)
	if s.Dirty() {
		s.Flush()
	}
}
