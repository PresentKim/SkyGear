package skygear

import (
	"github.com/df-mc/dragonfly/server/player"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/dimension"
)

// The second dimension's reaction to a death and a respawn.
func init() {
	core.Death.On(20, "dimension.died", func(e core.PlayerEvent) {
		dimension.Died(e.S, e.P.Tx(), e.P)
	})
	core.Respawn.On(10, "dimension.respawned", func(e core.RespawnEvent) {
		dimension.Respawned(e.S, e.P, e.Pos)
	})
}

// The second dimension's reaction to a drop and a hurt.
func init() {
	core.ItemDrop.Handle(10, "dimension.hunting", func(e core.ItemEvent) bool {
		if dimension.Hunting(e.S, e.P) {
			e.Cancel()
			return true
		}
		return false
	})
	core.Hurt.Handle(10, "dimension.frozen", func(e core.HurtEvent) bool {
		if dimension.Frozen(e.S, e.P) {
			e.Cancel() // the hunt waits while the hunter chooses an upgrade
			return true
		}
		return false
	})
	core.Hurt.On(30, "dimension.fallen", func(e core.HurtEvent) {
		if dimension.Fallen(e.S, e.Tx, e.P, *e.Damage) {
			e.Cancel()
		}
	})
}

// The collectors work and the items of the other dimension are retired every
// second; the watchdog looks at the hunt on every fast tick.
func init() {
	core.Second.On(60, "dimension.second", func(e core.Tick) {
		dimension.TickCollectors(e.S, e.Tx)
		for p := range e.Tx.Players() {
			dimension.RetireItems(p.(*player.Player))
		}
	})
	core.Fast.On(20, "dimension.watchdog", func(e core.Tick) {
		dimension.Watchdog(e.S, e.Tx)
	})
}
