package dimension

import (
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
)

// What the game's tests drive the hunts with.

// CraftRelic makes or raises kind at a relic table for p.
func CraftRelic(s *core.Session, p *player.Player, tx *world.Tx, kind string) {
	craftRelic(s, p, tx, kind)
}

// Carried is the highest grade of the relic kind p carries.
func Carried(p *player.Player, kind string) (int, bool) { return carried(p, kind) }

// ToggleLoadout puts kind in p's loadout or takes it out.
func ToggleLoadout(s *core.Session, p *player.Player, kind string) { toggleLoadout(s, p, kind) }

// GiveGems hands p n experience gems at once.
func GiveGems(r *Run, tx *world.Tx, p *player.Player, n int) {
	for range n {
		r.gear.drop(tx, p, r, p.Position())
	}
}

// Paused reports whether the run waits for a level-up choice; Ticks is how
// far it went.
func (r *Run) Paused() bool { return r.paused }
func (r *Run) Ticks() int   { return r.ticks }

// ChooseFirst takes the first upgrade offered.
func ChooseFirst(r *Run, tx *world.Tx, p *player.Player) { r.choose(tx, p, r.choices[0]) }
