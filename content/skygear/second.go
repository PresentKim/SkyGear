package skygear

import (
	"time"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/fluidnet"
	"github.com/presentkim/skygear/common/itemnet"
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/content/skygear/climate"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/dimension"
	"github.com/presentkim/skygear/content/skygear/fairy"
	"github.com/presentkim/skygear/content/skygear/farm"
	"github.com/presentkim/skygear/content/skygear/look"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/network"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/screens"
	"github.com/presentkim/skygear/content/skygear/tinker"
)

// Second runs the crops, the fairy, the advancements and the operators'
// load lines (instance.Session.Second).
func (s *session) Second(tx *world.Tx) {
	core.Mark("farm.growCrops")
	farm.GrowCrops(s.Session, tx)
	core.Mark("progress.checkHeld")
	held := advancement.InventoryNames(tx)
	progress.CheckHeld(s.Session, tx, held)
	core.Mark("farm.barrels")
	farm.TickBarrels(s.Session, tx)
	farm.TickComposters(s.Session, tx)
	core.Mark("fairy.tick")
	fairy.Tick(s.Session, tx)
	core.Mark("climate.tick")
	climate.Tick(s.Session, tx)
	core.Mark("dimension.second")
	dimension.TickCollectors(s.Session, tx)
	for e := range tx.Players() {
		dimension.RetireItems(e.(*player.Player))
	}
	core.Mark("tinker.effects")
	tinker.TickEffects(s.Session, tx)
	core.Mark("machine.monitor")
	machine.TickMonitor(s.Session, tx)
	core.Mark("farm.plants")
	farm.TickPlants(s.Session, tx)
	core.Mark("screens.boards")
	screens.UpdateBoards(s.Session, tx)
	core.Mark("achievement.playSeconds")
	achievement.CountAll(tx, "play_seconds", 1)
	core.Mark("progress.collection")
	achievement.GrantCollection(held)
	// Changes are stored every FlushInterval, not every second; Autosave and
	// Close store whatever is left.
	s.FlushIfDue(time.Now())
}

// Fast updates every player's panel (instance.Session.Fast, every 0.2
// seconds: gauges and time left change while a player looks; Show sends only
// what changed).
func (s *session) Fast(tx *world.Tx) {
	// Energy moves every FastInterval (four ticks) over the conduits.
	core.Mark("network.step")
	network.Energy(s.Session).Step(tx, 4)
	// Items move every itemnet.StepTicks ticks and fluids every
	// fluidnet.StepTicks ticks (a Fast call is four ticks).
	s.fastCount++
	if s.fastCount%(itemnet.StepTicks/4) == 0 {
		core.Mark("network.items")
		network.Items(s.Session).Step(tx)
	}
	if s.fastCount%(fluidnet.StepTicks/4) == 0 {
		core.Mark("network.fluids")
		network.Fluids(s.Session).Step(tx)
	}
	core.Mark("dimension.watchdog")
	dimension.Watchdog(s.Session, tx)
	core.Mark("look.updateAll")
	look.UpdateAll(s.Session, tx)
}
