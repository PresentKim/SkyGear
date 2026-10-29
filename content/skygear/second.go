package skygear

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/tinker"
)

// Second runs the crops, the fairy, the advancements and the operators'
// load lines (instance.Session.Second).
func (s *session) Second(tx *world.Tx) {
	s.growCrops(tx)
	held := advancement.InventoryNames(tx)
	progress.CheckHeld(s.Session, tx, held)
	s.tickBarrels(tx)
	s.tickComposters(tx)
	s.tickFairy(tx)
	s.tickEnvironment(tx)
	s.tickWave(tx)
	tinker.TickEffects(s.Session, tx)
	s.tickMonitor(tx)
	s.tickPlants(tx)
	s.updateBoards(tx)
	achievement.CountAll(tx, "play_seconds", 1)
	achievement.GrantCollection(held)
	if s.Dirty() {
		s.Flush()
	}
}
