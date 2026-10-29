package skygear

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/content/skygear/climate"
	"github.com/presentkim/skygear/content/skygear/farm"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/tinker"
)

// Second runs the crops, the fairy, the advancements and the operators'
// load lines (instance.Session.Second).
func (s *session) Second(tx *world.Tx) {
	farm.GrowCrops(s.Session, tx)
	held := advancement.InventoryNames(tx)
	progress.CheckHeld(s.Session, tx, held)
	farm.TickBarrels(s.Session, tx)
	farm.TickComposters(s.Session, tx)
	s.tickFairy(tx)
	climate.Tick(s.Session, tx)
	s.tickWave(tx)
	tinker.TickEffects(s.Session, tx)
	machine.TickMonitor(s.Session, tx)
	farm.TickPlants(s.Session, tx)
	s.updateBoards(tx)
	achievement.CountAll(tx, "play_seconds", 1)
	achievement.GrantCollection(held)
	if s.Dirty() {
		s.Flush()
	}
}
