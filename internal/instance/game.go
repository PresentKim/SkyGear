package instance

import (
	"time"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// Game is Sky Gear as the Manager sees it: the skygear package implements
// it, and this interface keeps the instance package from importing it.
// The game knows nothing about lobbies, accounts or persistence of players;
// the Manager handles those.
type Game interface {
	// Init builds the world of a brand new save. It runs once, on the
	// first activation of the save.
	Init(tx *world.Tx, inst *Instance)
	// Start begins the game logic of an activated instance.
	Start(inst *Instance) Session
}

// FastInterval is how often Session.Fast is called.
const FastInterval = 200 * time.Millisecond

// Session is the running game logic of one active instance.
type Session interface {
	// Spawn is where players without a stored position are placed.
	Spawn() mgl64.Vec3
	// Join is called in the instance world after p entered it and its state
	// was restored. first is true the first time p enters this save.
	Join(p *player.Player, first bool)
	// Leave is called in the instance world before p's state is captured.
	Leave(p *player.Player)
	// Handler returns the game's handler for p's events while in the
	// instance. It may return nil.
	Handler(p *player.Player) player.Handler
	// NewDay is called in the instance world when the instance day changes.
	NewDay(tx *world.Tx, day int)
	// Second is called in the instance world about once a second while the
	// instance is active, after the clock advanced.
	Second(tx *world.Tx)
	// Fast is called in the instance world every FastInterval while the
	// instance is active, for what players watch change (such as the WAILA
	// panel's gauges).
	Fast(tx *world.Tx)
	// MenuBack shows p, in the instance world, the game screen that the
	// platform screen from (ScreenFriends, ...) was opened from, for that
	// screen's back button.
	MenuBack(p *player.Player, from string)
	// Autosave is called in the instance world on every autosave, after the
	// clock and player states were stored. The game stores its state here.
	Autosave(tx *world.Tx)
	// Close is called in the instance world before it is saved and closed.
	Close(tx *world.Tx)
}

// The platform screens that also open inside an instance, as
// Session.MenuBack names them.
const (
	ScreenFriends  = "friends"
	ScreenMembers  = "members"
	ScreenTitles   = "titles"
	ScreenSettings = "settings"
	ScreenNotify   = "notify"
)
