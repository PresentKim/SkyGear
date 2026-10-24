package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/instance"
)

// Second runs the crops, the fairy, the advancements and the operators'
// load lines (instance.SecondTicker).
func (s *session) Second(tx *world.Tx) {
	s.growCrops(tx)
	s.checkAdvancements(tx)
	s.tickBarrels(tx)
	s.tickComposters(tx)
	s.tickFairy(tx)
	s.tickWave(tx)
	s.tickToolEffects(tx)
	s.tickMonitor(tx)
	s.tickPlants(tx)
	s.updateLoadBoards(tx)
	countAll(tx, "play_seconds", 1)
	grantCollection(tx)
}

var _ instance.SecondTicker = (*session)(nil)
