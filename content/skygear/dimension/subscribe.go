package dimension

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// subscribe registers the takeovers and the steps of dimension on the kernel.
func subscribe() {
	core.RegisterTakeover("dimension.hunt", Hunting)
	core.RegisterTakeover("dimension.watch", Watching)
	subscribeLife()
	subscribeHurt()
	subscribeTicks()
	subscribePresence()
	subscribeUse()
	registerKinds()
}

// The second dimension's reaction to a death and a respawn.
func subscribeLife() {
	core.Death.On(core.Normal, "dimension.died", func(e core.PlayerEvent) {
		Died(e.S, e.P.Tx(), e.P)
	})
	core.Respawn.On(core.Low, "dimension.respawned", func(e core.RespawnEvent) {
		Respawned(e.S, e.P, e.Pos)
	})
}

// The second dimension's reaction to a drop and a hurt. Whether a hit would
// kill the hunter is asked last, of the damage the other steps left: Highest.
func subscribeHurt() {
	core.ItemDrop.Handle(core.Lowest, "dimension.hunting", func(e core.ItemEvent) bool {
		if Hunting(e.S, e.P) {
			e.Cancel()
			return true
		}
		return false
	})
	core.Hurt.Handle(core.Lowest, "dimension.frozen", func(e core.HurtEvent) bool {
		if Frozen(e.S, e.P) {
			e.Cancel() // the hunt waits while the hunter chooses an upgrade
			return true
		}
		return false
	})
	core.Hurt.Handle(core.Highest, "dimension.fallen", func(e core.HurtEvent) bool {
		if Fallen(e.S, e.Tx, e.P, *e.Damage) {
			e.Cancel() // no hurt for the subscribers after it: the run ended instead
			return true
		}
		return false
	})
}

// The collectors work and the items of the other dimension are retired every
// second; the watchdog looks at the hunt on every fast tick.
func subscribeTicks() {
	core.Second.On(core.Normal, "dimension.second", func(e core.Tick) {
		TickCollectors(e.S, e.Tx)
		for p := range e.Tx.Players() {
			RetireItems(p.(*player.Player))
		}
	})
	core.Fast.On(core.Normal, "dimension.watchdog", func(e core.Tick) {
		Watchdog(e.S, e.Tx)
	})
}

// The second dimension reacts to a player joining and leaving.
//
// dimension.joined is in the High tier, after the join steps of the other
// features, as it always ran: a hunter the server stopped under gets back
// what it carried, and what the earlier join steps put into the hunter's
// inventory meanwhile is gone with the hunting kit. That keeps today's
// behaviour; it is one of the listed exceptions (docs/tech/extension.md).
func subscribePresence() {
	core.Join.On(core.High, "dimension.joined", func(e core.Presence) {
		Joined(e.S, e.P)
		RetireItems(e.P)
	})
	core.Leave.On(core.Normal, "dimension.leave", func(e core.Presence) {
		Leave(e.S, e.Tx, e.P)
	})
}

// A hunter only moves: the use chain refuses it, and the weapons of a hunter
// strike on their own (gear.go). The menu item is the way out. The refusals
// and the stone's window come before everything else: Lowest. They exclude
// each other and the menu item's step by their own conditions.
func subscribeUse() {
	core.UseOnBlock.Handle(core.Lowest, "dimension.hunting", func(e core.UseOnBlockEvent) bool {
		if Hunting(e.S, e.P) {
			e.Cancel()
			return true
		}
		return false
	})
	core.UseOnBlock.Handle(core.Lowest, "dimension.stone", func(e core.UseOnBlockEvent) bool {
		return useStone(e.Actor, e.Held)
	})
	core.Use.Handle(core.Lowest, "dimension.hunting", func(e core.UseEvent) bool {
		// the menu item in the air is the hunter's way out (screens.menuItem)
		if Hunting(e.S, e.P) && !sgitem.IsMenuItem(e.Held) {
			e.Cancel()
			return true
		}
		return false
	})
	core.Use.Handle(core.Lowest, "dimension.stone", func(e core.UseEvent) bool {
		return useStone(e.Actor, e.Held)
	})
	core.UseOnEntity.Handle(core.Lowest, "dimension.hunting", func(e core.EntityEvent) bool {
		if Hunting(e.S, e.P) {
			e.Cancel()
			return true
		}
		return false
	})
	core.AttackEntity.Handle(core.Lowest, "dimension.hunter", func(e core.EntityEvent) bool {
		if _, ok := e.Target.(*Monster); ok || Hunting(e.S, e.P) {
			e.Cancel()
			return true
		}
		return false
	})
}

// useStone opens the second dimension's window for a held teleport stone,
// on a block or the air; it reports whether it took the click. A hunter's
// stone opens nothing: the hunter's use is refused (dimension.hunting).
func useStone(a core.Actor, held item.Stack) bool {
	if _, ok := held.Item().(sgitem.DimensionStone); !ok || Hunting(a.S, a.P) {
		return false
	}
	a.Cancel()
	if !a.S.Debounced(a.P) {
		ShowWindow(a.S, a.P, "")
	}
	return true
}
