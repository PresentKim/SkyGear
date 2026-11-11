package achievement

import (
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// subscribe registers the achievement counters on the hooks they count.
func subscribe() {
	subscribeSimple()
	subscribeMoves()
	subscribeSeconds()
	subscribeBlocks()
}

// The achievement counters of the simple player events.
func subscribeSimple() {
	core.Sprint.On(10, "achievement.sprint", func(e core.PlayerEvent) {
		Count(e.P, "sprint", 1)
	})
	core.Jump.On(10, "achievement.jump", func(e core.PlayerEvent) {
		Count(e.P, "jump", 1)
	})
	core.ItemConsume.On(10, "achievement.eat", func(e core.ItemEvent) {
		Count(e.P, "eat", 1)
	})
	core.ItemPickup.On(10, "achievement.pickup", func(e core.ItemEvent) {
		Count(e.P, "pickup", int64(e.Pick.Count()))
	})
	core.Death.On(10, "achievement.death", func(e core.PlayerEvent) {
		Count(e.P, "death", 1)
	})
	core.PunchAir.On(10, "achievement.punchAir", func(e core.PlayerEvent) {
		Count(e.P, "punch_air", 1)
	})
	core.Chat.On(10, "achievement.chat", func(e core.PlayerEvent) {
		Count(e.P, "chat", 1)
	})
	core.BlockPick.On(10, "achievement.blockPick", func(e core.PlayerEvent) {
		Count(e.P, "block_pick", 1)
	})
	core.HeldSlot.On(10, "achievement.heldSlot", func(e core.PlayerEvent) {
		Count(e.P, "hotbar", 1)
	})
	core.Command.On(10, "achievement.command", func(e core.PlayerEvent) {
		Count(e.P, "command", 1)
	})
}

// The achievement counters of the walk, sneak, drop and hurt events.
func subscribeMoves() {
	core.Move.On(30, "achievement.walk", func(e core.MoveEvent) {
		CountWalk(e.S, e.P, e.Pos)
	})
	core.Sneak.On(20, "achievement.sneak", func(e core.PlayerEvent) {
		Count(e.P, "sneak", 1)
	})
	core.ItemDrop.On(20, "achievement.drop", func(e core.ItemEvent) {
		Count(e.P, "item_drop", 1)
	})
	core.Hurt.On(20, "achievement.hurt", func(e core.HurtEvent) {
		Count(e.P, "hurt", 1)
	})
}

// Every player's play time counts a second.
func subscribeSeconds() {
	core.Second.On(110, "achievement.playSeconds", func(e core.Tick) {
		CountAll(e.Tx, "play_seconds", 1)
	})
}

// Breaking and placing blocks count.
func subscribeBlocks() {
	core.BlockBreak.On(30, "achievement.break", func(e core.BlockBreakEvent) {
		CountBreak(e.P, e.Block, sgitem.IsHammer(e.Held))
	})
	core.BlockPlace.On(30, "achievement.place", func(e core.BlockPlaceEvent) {
		Count(e.P, "block_place", 1)
	})
}
