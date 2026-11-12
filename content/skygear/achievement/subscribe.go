package achievement

import (
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// subscribe registers the achievement counters on the hooks they count. A
// counter records what happened and changes nothing: it is a Monitor
// subscriber.
func subscribe() {
	subscribeSimple()
	subscribeMoves()
	subscribeSeconds()
	subscribeBlocks()
}

// The achievement counters of the simple player events.
func subscribeSimple() {
	core.Sprint.On(core.Monitor, "achievement.sprint", func(e core.PlayerEvent) {
		Count(e.P, "sprint", 1)
	})
	core.Jump.On(core.Monitor, "achievement.jump", func(e core.PlayerEvent) {
		Count(e.P, "jump", 1)
	})
	core.ItemConsume.On(core.Monitor, "achievement.eat", func(e core.ItemEvent) {
		Count(e.P, "eat", 1)
	})
	core.ItemPickup.On(core.Monitor, "achievement.pickup", func(e core.ItemEvent) {
		Count(e.P, "pickup", int64(e.Pick.Count()))
	})
	core.Death.On(core.Monitor, "achievement.death", func(e core.PlayerEvent) {
		Count(e.P, "death", 1)
	})
	core.PunchAir.On(core.Monitor, "achievement.punchAir", func(e core.PlayerEvent) {
		Count(e.P, "punch_air", 1)
	})
	core.Chat.On(core.Monitor, "achievement.chat", func(e core.PlayerEvent) {
		Count(e.P, "chat", 1)
	})
	core.BlockPick.On(core.Monitor, "achievement.blockPick", func(e core.PlayerEvent) {
		Count(e.P, "block_pick", 1)
	})
	core.HeldSlot.On(core.Monitor, "achievement.heldSlot", func(e core.PlayerEvent) {
		Count(e.P, "hotbar", 1)
	})
	core.Command.On(core.Monitor, "achievement.command", func(e core.PlayerEvent) {
		Count(e.P, "command", 1)
	})
}

// The achievement counters of the walk, sneak, drop and hurt events. The walk
// counts from where the player stood when the event was built (e.From), so a
// step that moved the player meanwhile does not change it. Every sneak
// counts, whatever else it did (an elevator ride, the screen of island
// expansion mode).
func subscribeMoves() {
	core.Move.On(core.Monitor, "achievement.walk", func(e core.MoveEvent) {
		CountWalk(e.S, e.P, e.From, e.Pos)
	})
	core.Sneak.On(core.Monitor, "achievement.sneak", func(e core.PlayerEvent) {
		Count(e.P, "sneak", 1)
	})
	core.ItemDrop.On(core.Monitor, "achievement.drop", func(e core.ItemEvent) {
		Count(e.P, "item_drop", 1)
	})
	core.Hurt.On(core.Monitor, "achievement.hurt", func(e core.HurtEvent) {
		Count(e.P, "hurt", 1)
	})
}

// Every player's play time counts a second.
func subscribeSeconds() {
	core.Second.On(core.Monitor, "achievement.playSeconds", func(e core.Tick) {
		CountAll(e.Tx, "play_seconds", 1)
	})
}

// Breaking and placing blocks count.
func subscribeBlocks() {
	core.BlockBreak.On(core.Monitor, "achievement.break", func(e core.BlockBreakEvent) {
		CountBreak(e.P, e.Block, sgitem.IsHammer(e.Held))
	})
	core.BlockPlace.On(core.Monitor, "achievement.place", func(e core.BlockPlaceEvent) {
		Count(e.P, "block_place", 1)
	})
}
