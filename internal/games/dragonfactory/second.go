package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/instance"
)

// Second runs the crops, the fairy and the operators' load lines
// (instance.SecondTicker).
func (s *session) Second(tx *world.Tx) {
	s.growCrops(tx)
	s.tickFairy(tx)
	s.updateLoadBoards(tx)
}

var _ instance.SecondTicker = (*session)(nil)
