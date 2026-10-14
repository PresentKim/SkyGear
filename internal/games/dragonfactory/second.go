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
	s.tickFairy(tx)
	s.tickWave(tx)
	s.updateLoadBoards(tx)
}

var _ instance.SecondTicker = (*session)(nil)
