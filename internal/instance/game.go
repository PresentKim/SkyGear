package instance

import (
	"time"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// Game is a game type that saves are created for (e.g. dragonfactory).
// Games know nothing about lobbies, accounts or persistence of players;
// the Manager handles those.
type Game interface {
	// ID is the stable identifier stored in saves.game_type.
	ID() string
	// Name is the display name in lang.
	Name(lang string) string
	// Init builds the world of a brand new save. It runs once, on the
	// first activation of the save.
	Init(tx *world.Tx, inst *Instance)
	// Start begins the game logic of an activated instance.
	Start(inst *Instance) Session
}

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
	// Autosave is called in the instance world on every autosave, after the
	// clock and player states were stored. Games store their state here.
	Autosave(tx *world.Tx)
	// Close is called in the instance world before it is saved and closed.
	Close(tx *world.Tx)
}

// SecondTicker is an optional Session interface: Second is called in the
// instance world about once a second while the instance is active, after
// the clock advanced.
type SecondTicker interface {
	Second(tx *world.Tx)
}

// FastInterval is how often FastTicker.Fast is called.
const FastInterval = 200 * time.Millisecond

// FastTicker is an optional Session interface: Fast is called in the
// instance world every FastInterval while the instance is active, for what
// players watch change (such as the WAILA panel's gauges).
type FastTicker interface {
	Fast(tx *world.Tx)
}
