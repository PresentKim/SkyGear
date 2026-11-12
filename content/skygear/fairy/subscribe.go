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
	core.Second.On(core.Normal, "fairy.tick", func(e core.Tick) {
		Tick(e.S, e.Tx)
	})
}

// A joining player gets the fairy dust they are owed.
func subscribeJoin() {
	core.Join.On(core.Normal, "fairy.backfillDust", func(e core.Presence) {
		BackfillDust(e.S, e.P, e.Tx)
	})
}

// The fairy's time is told by the item it is used with; a visiting fairy
// gives its gift to a player who touches it, whichever way.
//
// conduit.plate is in the same tier, is registered before fairy.time and
// takes a click on a connector plate first, hourglass in hand or not: that
// order is one of the exceptions the tier order guard keeps (the root's
// orderKept).
func subscribeTouch() {
	core.UseOnBlock.Handle(core.Normal, "fairy.time", func(e core.UseOnBlockEvent) bool {
		if UseFairyTime(e.S, e.P, e.Tx, e.Pos, e.Held) {
			e.Cancel()
			return true
		}
		return false
	})
	core.UseOnEntity.On(core.Normal, "fairy.touch", touchFairy)
	core.AttackEntity.On(core.Normal, "fairy.touch", touchFairy)
}

// touchFairy takes the gift of the fairy e.Target, if it is one.
func touchFairy(e core.EntityEvent) {
	if f, ok := Of(e.S, e.Target); ok {
		e.Cancel()
		f.Gift(e.Tx, e.S, e.P)
	}
}
