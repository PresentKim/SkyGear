package skygear

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/tinker"
)

// keptFood is the food level each player died with, given back when it
// respawns: dying does not fill the food bar (it would make vein mining's
// food cost free).
var keptFood = core.NewPlayers[int]()

func init() {
	// The iron hammer keeps whole: its vein mining costs food instead (vein.go).
	core.ItemDamage.On(10, "tinker.ironHammer", func(e core.ItemDamageEvent) {
		if sgitem.IsIronHammer(e.Item) {
			*e.Damage = 0
		}
	})
	core.Death.On(30, "tinker.keepFood", func(e core.PlayerEvent) {
		keptFood.Of(e.S)[e.P.H()] = e.P.Food()
	})
	core.Respawn.On(20, "tinker.restoreFood", func(e core.RespawnEvent) {
		if food, ok := keptFood.Of(e.S)[e.P.H()]; ok {
			delete(keptFood.Of(e.S), e.P.H())
			e.P.SetFood(food)
		}
	})
}

// The worn tools give their effects every second.
func init() {
	core.Second.On(70, "tinker.effects", func(e core.Tick) {
		tinker.TickEffects(e.S, e.Tx)
	})
}

// The pickaxe in hand is chosen for the block being mined; breaking with a
// tool gives it experience, and a hammer crushes the block and mines the vein.
// The hammer's drop replaces the drops in place.
func init() {
	core.StartBreak.On(30, "tinker.autoTool", func(e core.StartBreakEvent) {
		tinker.AutoTool(e.P, e.Tx.Block(e.Pos))
	})
	core.BlockBreak.On(40, "tinker.break", func(e core.BlockBreakEvent) {
		if _, ok := tinker.ToolOf(e.P); ok {
			tinker.GainXP(e.S, e.P, 1)
			tinker.Reinforce(e.S, e.P)
		}
		if sgitem.IsHammer(e.Held) {
			if into, ok := sgitem.HammerDrop(e.Block); ok {
				*e.Drops = []item.Stack{into}
			}
			tinker.VeinMine(e.S, e.P, e.Tx, e.Pos)
		}
	})
}

// The iron hammer used on the air while sneaking switches diagonal vein
// mining.
func init() {
	core.Use.On(70, "tinker.veinToggle", func(e core.UseEvent) {
		if sgitem.IsIronHammer(e.Held) && e.P.Sneaking() && !e.S.Debounced(e.P) {
			tinker.ToggleVeinDiagonal(e.S, e.P)
		}
	})
}
