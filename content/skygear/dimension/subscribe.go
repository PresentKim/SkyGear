package dimension

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// subscribe registers the steps of dimension on the kernel's hooks.
func subscribe() {
	subscribeLife()
	subscribeHurt()
	subscribeTicks()
	subscribePresence()
	subscribeUse()
}

// The second dimension's reaction to a death and a respawn.
func subscribeLife() {
	core.Death.On(20, "dimension.died", func(e core.PlayerEvent) {
		Died(e.S, e.P.Tx(), e.P)
	})
	core.Respawn.On(10, "dimension.respawned", func(e core.RespawnEvent) {
		Respawned(e.S, e.P, e.Pos)
	})
}

// The second dimension's reaction to a drop and a hurt.
func subscribeHurt() {
	core.ItemDrop.Handle(10, "dimension.hunting", func(e core.ItemEvent) bool {
		if Hunting(e.S, e.P) {
			e.Cancel()
			return true
		}
		return false
	})
	core.Hurt.Handle(10, "dimension.frozen", func(e core.HurtEvent) bool {
		if Frozen(e.S, e.P) {
			e.Cancel() // the hunt waits while the hunter chooses an upgrade
			return true
		}
		return false
	})
	core.Hurt.On(30, "dimension.fallen", func(e core.HurtEvent) {
		if Fallen(e.S, e.Tx, e.P, *e.Damage) {
			e.Cancel()
		}
	})
}

// The collectors work and the items of the other dimension are retired every
// second; the watchdog looks at the hunt on every fast tick.
func subscribeTicks() {
	core.Second.On(60, "dimension.second", func(e core.Tick) {
		TickCollectors(e.S, e.Tx)
		for p := range e.Tx.Players() {
			RetireItems(p.(*player.Player))
		}
	})
	core.Fast.On(20, "dimension.watchdog", func(e core.Tick) {
		Watchdog(e.S, e.Tx)
	})
}

// The second dimension reacts to a player joining and leaving.
func subscribePresence() {
	core.Join.On(80, "dimension.joined", func(e core.Presence) {
		Joined(e.S, e.P)
		RetireItems(e.P)
	})
	core.Leave.On(10, "dimension.leave", func(e core.Presence) {
		Leave(e.S, e.Tx, e.P)
	})
}

// A hunter only moves: the use chain refuses it, and the weapons of a hunter
// strike on their own (gear.go). The menu item is the way out.
func subscribeUse() {
	core.UseOnBlock.Handle(10, "dimension.hunting", func(e core.UseOnBlockEvent) bool {
		if Hunting(e.S, e.P) {
			e.Cancel()
			return true
		}
		return false
	})
	core.UseOnBlock.Handle(30, "dimension.stone", func(e core.UseOnBlockEvent) bool {
		return useStone(e.Actor, e.Held)
	})
	core.Use.Handle(30, "dimension.hunting", func(e core.UseEvent) bool {
		if Hunting(e.S, e.P) {
			e.Cancel()
			return true
		}
		return false
	})
	core.Use.Handle(40, "dimension.stone", func(e core.UseEvent) bool {
		return useStone(e.Actor, e.Held)
	})
	core.UseOnEntity.Handle(10, "dimension.hunting", func(e core.EntityEvent) bool {
		if Hunting(e.S, e.P) {
			e.Cancel()
			return true
		}
		return false
	})
	core.AttackEntity.Handle(10, "dimension.hunter", func(e core.EntityEvent) bool {
		if _, ok := e.Target.(*Monster); ok || Hunting(e.S, e.P) {
			e.Cancel()
			return true
		}
		return false
	})
}

// useStone opens the second dimension's window for a held teleport stone,
// on a block or the air; it reports whether held is one.
func useStone(a core.Actor, held item.Stack) bool {
	if _, ok := held.Item().(sgitem.DimensionStone); !ok {
		return false
	}
	a.Cancel()
	if !a.S.Debounced(a.P) {
		ShowWindow(a.S, a.P, "")
	}
	return true
}
