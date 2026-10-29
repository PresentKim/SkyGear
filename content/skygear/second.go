package skygear

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/fluidnet"
	"github.com/presentkim/skygear/common/itemnet"
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/content/skygear/climate"
	"github.com/presentkim/skygear/content/skygear/fairy"
	"github.com/presentkim/skygear/content/skygear/farm"
	"github.com/presentkim/skygear/content/skygear/look"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/network"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/screens"
	"github.com/presentkim/skygear/content/skygear/tinker"
	"github.com/presentkim/skygear/content/skygear/wave"
)

// Second runs the crops, the fairy, the advancements and the operators'
// load lines (instance.Session.Second).
func (s *session) Second(tx *world.Tx) {
	farm.GrowCrops(s.Session, tx)
	held := advancement.InventoryNames(tx)
	progress.CheckHeld(s.Session, tx, held)
	farm.TickBarrels(s.Session, tx)
	farm.TickComposters(s.Session, tx)
	fairy.Tick(s.Session, tx)
	climate.Tick(s.Session, tx)
	wave.Tick(s.Session, tx)
	tinker.TickEffects(s.Session, tx)
	machine.TickMonitor(s.Session, tx)
	farm.TickPlants(s.Session, tx)
	screens.UpdateBoards(s.Session, tx)
	achievement.CountAll(tx, "play_seconds", 1)
	achievement.GrantCollection(held)
	if s.Dirty() {
		s.Flush()
	}
}

// Fast updates every player's panel (instance.Session.Fast, every 0.2
// seconds: gauges and time left change while a player looks; Show sends only
// what changed).
func (s *session) Fast(tx *world.Tx) {
	// Energy moves every FastInterval (four ticks) over the conduits.
	network.Energy(s.Session).Step(tx, 4)
	// Items move every itemnet.StepTicks ticks and fluids every
	// fluidnet.StepTicks ticks (a Fast call is four ticks).
	s.fastCount++
	if s.fastCount%(itemnet.StepTicks/4) == 0 {
		network.Items(s.Session).Step(tx)
	}
	if s.fastCount%(fluidnet.StepTicks/4) == 0 {
		network.Fluids(s.Session).Step(tx)
	}
	wave.EmergeMonsters(s.Session, tx)
	look.UpdateAll(s.Session, tx)
}
