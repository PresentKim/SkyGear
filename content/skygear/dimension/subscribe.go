package dimension

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
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
	core.Death.On(core.Normal, "dimension.died", func(e core.NoticeEvent) {
		Died(e.S, e.P.Tx(), e.P)
	})
	core.Respawn.On(core.Low, "dimension.respawned", func(e core.RespawnEvent) {
		Respawned(e.S, e.P, e.Pos)
	})
}

// The second dimension's reaction to a drop and a hurt. Whether a hit would
// kill the hunter is asked last, of the damage the other steps left: Highest.
func subscribeHurt() {
	core.ItemDrop.On(core.Lowest, "dimension.hunting", func(e core.ItemEvent) {
		if Hunting(e.S, e.P) {
			e.Cancel()
		}
	})
	core.Hurt.On(core.Lowest, "dimension.frozen", func(e core.HurtEvent) {
		if Frozen(e.S, e.P) {
			e.Cancel() // the hunt waits while the hunter chooses an upgrade
		}
	})
	core.Hurt.On(core.Highest, "dimension.fallen", func(e core.HurtEvent) {
		if Fallen(e.S, e.Tx, e.P, *e.Damage) {
			e.Cancel() // no hurt for the subscribers after it: the run ended instead
		}
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
// dimension.joined gives a player the server stopped under in the second
// dimension back what it carried, and a watcher its game mode. It makes the
// player ready before the join steps of the other features hand anything
// out: Low. What they give (the fairy dust the save is owed) goes into the
// inventory the player keeps.
func subscribePresence() {
	core.Join.On(core.Low, "dimension.joined", func(e core.Presence) {
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
	core.UseOnBlock.On(core.Lowest, "dimension.hunting", func(e core.UseOnBlockEvent) {
		if Hunting(e.S, e.P) {
			e.Cancel()
		}
	})
	core.UseOnBlock.On(core.Lowest, "dimension.stone", func(e core.UseOnBlockEvent) {
		useStone(e.Actor, e.Held)
	})
	core.Use.On(core.Lowest, "dimension.hunting", func(e core.UseEvent) {
		// the menu item in the air is the hunter's way out (screens.menuItem)
		if Hunting(e.S, e.P) && !sgitem.IsMenuItem(e.Held) {
			e.Cancel()
		}
	})
	// On the air no block is asked before the item: the stone opens its
	// window in its own Use (sgitem.DimensionStone), which Dragonfly calls
	// unless a subscriber cancelled the use (a hunter's: dimension.hunting).
	sgitem.StoneUsed = func(tx *world.Tx, u item.User) {
		p, ok := u.(*player.Player)
		s, ok2 := core.Of(tx)
		if ok && ok2 && !Hunting(s, p) && !s.Debounced(p) {
			ShowWindow(s, p, "")
		}
	}
	core.UseOnEntity.On(core.Lowest, "dimension.hunting", func(e core.EntityEvent) {
		if Hunting(e.S, e.P) {
			e.Cancel()
		}
	})
	core.AttackEntity.On(core.Lowest, "dimension.hunter", func(e core.EntityEvent) {
		if _, ok := e.Target.(*Monster); ok || Hunting(e.S, e.P) {
			e.Cancel()
		}
	})
}

// useStone opens the second dimension's window for a teleport stone held on a
// block, and cancels the click it took: the block under it must not open. A hunter's stone
// opens nothing: the hunter's use is refused (dimension.hunting).
func useStone(a core.Actor, held item.Stack) {
	if _, ok := held.Item().(sgitem.DimensionStone); !ok || Hunting(a.S, a.P) {
		return
	}
	a.Cancel()
	if !a.S.Debounced(a.P) {
		ShowWindow(a.S, a.P, "")
	}
}
