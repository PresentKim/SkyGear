package fairy

import "github.com/presentkim/skygear/content/skygear/core"

// subscribe registers the fairy's second, join and touch steps.
func subscribe() {
	subscribeSecond()
	subscribeJoin()
	subscribeTouch()
}

// The fairy acts every second.
func subscribeSecond() {
	core.Second.On(40, "fairy.tick", func(e core.Tick) {
		Tick(e.S, e.Tx)
	})
}

// A joining player gets the fairy dust they are owed.
func subscribeJoin() {
	core.Join.On(20, "fairy.backfillDust", func(e core.Presence) {
		BackfillDust(e.S, e.P, e.Tx)
	})
}

// The fairy's time is told by the item it is used with; a visiting fairy
// gives its gift to a player who touches it, whichever way.
func subscribeTouch() {
	core.UseOnBlock.Handle(140, "fairy.time", func(e core.UseOnBlockEvent) bool {
		if UseFairyTime(e.S, e.P, e.Tx, e.Pos, e.Held) {
			e.Cancel()
			return true
		}
		return false
	})
	core.UseOnEntity.On(20, "fairy.touch", touchFairy)
	core.AttackEntity.On(20, "fairy.touch", touchFairy)
}

// touchFairy takes the gift of the fairy e.Target, if it is one.
func touchFairy(e core.EntityEvent) {
	if f, ok := Of(e.S, e.Target); ok {
		e.Cancel()
		f.Gift(e.Tx, e.S, e.P)
	}
}
