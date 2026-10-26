package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/advancement"
	"github.com/presentkim/dragonfactory/internal/instance"
)

// Second runs the crops, the fairy, the advancements and the operators'
// load lines (instance.SecondTicker).
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
	if s.stateDirty {
		s.flushState()
	}
}

var _ instance.SecondTicker = (*session)(nil)
