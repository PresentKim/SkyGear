package tinker

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/look"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// keptFood is the food level each player died with, given back when it
// respawns: dying does not fill the food bar (it would make vein mining's
// food cost free).
var keptFood = core.NewPlayers[int]()

// KeptFood returns the food level of each player of s that died and has not
// respawned yet.
func KeptFood(s *core.Session) map[*world.EntityHandle]int { return keptFood.Of(s) }

// subscribe subscribes the tinker steps to the hooks.
func subscribe() {
	subscribeFood()
	subscribeEffects()
	subscribeTools()
	subscribeVeinToggle()
	followLook()
	registerKinds()
}

func subscribeFood() {
	// The iron hammer keeps whole: its vein mining costs food instead (vein.go).
	// It shapes the damage others read: Low.
	core.ItemDamage.On(core.Low, "tinker.ironHammer", func(e core.ItemDamageEvent) {
		if sgitem.IsIronHammer(e.Item) {
			*e.Damage = 0
		}
	})
	core.Death.On(core.Normal, "tinker.keepFood", func(e core.NoticeEvent) {
		keptFood.Of(e.S)[e.P.H()] = e.P.Food()
	})
	core.Respawn.On(core.Normal, "tinker.restoreFood", func(e core.RespawnEvent) {
		if food, ok := keptFood.Of(e.S)[e.P.H()]; ok {
			delete(keptFood.Of(e.S), e.P.H())
			e.P.SetFood(food)
		}
	})
}

// The worn tools give their effects every second.
func subscribeEffects() {
	core.Second.On(core.Normal, "tinker.effects", func(e core.Tick) {
		TickEffects(e.S, e.Tx)
	})
}

// The pickaxe in hand is chosen for the block being mined; breaking with a
// tool gives it experience, and a hammer crushes the block and mines the vein.
// The hammer's drop replaces the drops in place.
func subscribeTools() {
	core.StartBreak.On(core.Normal, "tinker.autoTool", func(e core.StartBreakEvent) {
		AutoTool(e.P, e.Tx.Block(e.Pos))
	})
	core.BlockBreak.On(core.Normal, "tinker.break", func(e core.BlockBreakEvent) {
		if _, ok := ToolOf(e.P); ok {
			GainXP(e.S, e.P, 1)
			Reinforce(e.S, e.P)
		}
		if sgitem.IsHammer(e.Held) {
			if into, ok := sgitem.HammerDrop(e.Block); ok {
				*e.Drops = []item.Stack{into}
			}
			VeinMine(e.S, e.P, e.Tx, e.Pos)
		}
	})
}

// The iron hammer used on the air while sneaking switches diagonal vein
// mining: the item's own Use (sgitem.Hammer), which Dragonfly calls unless a
// subscriber cancelled the use. The stone hammer does nothing there.
func subscribeVeinToggle() {
	sgitem.HammerUsed = func(tx *world.Tx, u item.User) {
		p, ok := u.(*player.Player)
		s, ok2 := core.Of(tx)
		if !ok || !ok2 {
			return
		}
		if held, _ := p.HeldItems(); sgitem.IsIronHammer(held) && p.Sneaking() && !s.Debounced(p) {
			ToggleVeinDiagonal(s, p)
		}
	}
}

// On the fast tick the vein outline follows the block a player looks at.
func followLook() {
	look.Follow("tinker.vein", func(t look.Target) {
		if !t.Tick {
			return
		}
		if t.Ok {
			ShowVeinOutline(t.S, t.P, t.Tx, t.Pos)
		} else {
			HideVeinOutline(t.S, t.P)
		}
	})
}
