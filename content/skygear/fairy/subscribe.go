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

// Fairy bone meal grows the plant it is used on, and plain bone meal is kept
// off a timed crop: fairy.time acts before the crop's own click (farm.crop,
// High) and before the block is asked. It acts on crops, saplings and magic
// plants only, none of which carries a connector plate, so it and
// conduit.plate never act on the same click. A visiting fairy gives its gift
// to a player who touches it, whichever way.
func subscribeTouch() {
	core.UseOnBlock.On(core.Normal, "fairy.time", func(e core.UseOnBlockEvent) {
		if UseFairyTime(e.S, e.P, e.Tx, e.Pos, e.Held) {
			e.Cancel()
		}
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
