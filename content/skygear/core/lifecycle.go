package core

import (
	"time"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// Lifecycle is the session's part of instance.Session: the calls the instance
// makes when players join and leave, when the day changes, every second, every
// fifth of a second, on autosave and at close. Each emits its hook. A session
// type embeds Lifecycle, so the calls reach these methods before the core
// Session's own Leave and Close.
type Lifecycle struct {
	*Session
}

// Join emits Join.
func (l Lifecycle) Join(p *player.Player, first bool) {
	Join.Emit(Presence{S: l.Session, P: p, Tx: p.Tx(), First: first})
}

// Leave emits Leave, then clears the shapes on the player's client and drops
// the player's Players entries (Session.Leave). Both come after every
// subscriber, whatever its tier and whether or not one cancelled the event: a
// subscriber still finds the player's entries, and what it drew is cleared.
func (l Lifecycle) Leave(p *player.Player) {
	Leave.Emit(Presence{S: l.Session, P: p, Tx: p.Tx()})
	Mark("core.shapes")
	p.RemoveAllDebugShapes()
	Mark("core.parts")
	l.Session.Leave(p)
}

// Handler is the handler of a player who entered the save.
func (l Lifecycle) Handler(*player.Player) player.Handler {
	return &PlayerHandler{S: l.Session}
}

// NewDay emits NewDay, then tells every player the day (game.newDay).
func (l Lifecycle) NewDay(tx *world.Tx, day int) {
	NewDay.Emit(DayEvent{S: l.Session, Tx: tx, Day: day})
	Mark("game.newDay")
	l.Broadcast(tx, "game.newDay", day)
}

// Second emits Second, then stores the changes when they are due: they are
// stored every FlushInterval, not every second; Autosave and Close store
// whatever is left.
func (l Lifecycle) Second(tx *world.Tx) {
	Second.Emit(Tick{S: l.Session, Tx: tx})
	l.FlushIfDue(time.Now())
}

// Fast emits Fast.
func (l Lifecycle) Fast(tx *world.Tx) {
	Fast.Emit(Tick{S: l.Session, Tx: tx})
}

// Autosave emits Autosave, then stores the changes.
func (l Lifecycle) Autosave(tx *world.Tx) {
	Autosave.Emit(Tick{S: l.Session, Tx: tx})
	l.Flush()
}

// Close emits Close, then stores the saved parts, stops the writer and
// unregisters the session.
func (l Lifecycle) Close(tx *world.Tx) {
	Close.Emit(Tick{S: l.Session, Tx: tx})
	l.Session.Close()
	Unregister(l.Session)
}
