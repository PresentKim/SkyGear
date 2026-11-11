package skygear

import (
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/core"
)

// The achievement counters of the simple player events.
func init() {
	core.Sprint.On(10, "achievement.sprint", func(e core.PlayerEvent) {
		achievement.Count(e.P, "sprint", 1)
	})
	core.Jump.On(10, "achievement.jump", func(e core.PlayerEvent) {
		achievement.Count(e.P, "jump", 1)
	})
	core.ItemConsume.On(10, "achievement.eat", func(e core.ItemEvent) {
		achievement.Count(e.P, "eat", 1)
	})
	core.ItemPickup.On(10, "achievement.pickup", func(e core.ItemEvent) {
		achievement.Count(e.P, "pickup", int64(e.Pick.Count()))
	})
	core.Death.On(10, "achievement.death", func(e core.PlayerEvent) {
		achievement.Count(e.P, "death", 1)
	})
	core.PunchAir.On(10, "achievement.punchAir", func(e core.PlayerEvent) {
		achievement.Count(e.P, "punch_air", 1)
	})
	core.Chat.On(10, "achievement.chat", func(e core.PlayerEvent) {
		achievement.Count(e.P, "chat", 1)
	})
	core.BlockPick.On(10, "achievement.blockPick", func(e core.PlayerEvent) {
		achievement.Count(e.P, "block_pick", 1)
	})
	core.HeldSlot.On(10, "achievement.heldSlot", func(e core.PlayerEvent) {
		achievement.Count(e.P, "hotbar", 1)
	})
	core.Command.On(10, "achievement.command", func(e core.PlayerEvent) {
		achievement.Count(e.P, "command", 1)
	})
}
