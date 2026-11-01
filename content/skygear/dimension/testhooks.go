package dimension

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/core"
)

// What the game's tests drive the hunts with.

// CraftRelic makes or raises kind at a relic table for p.
func CraftRelic(s *core.Session, p *player.Player, tx *world.Tx, kind string) {
	craftRelic(s, p, tx, kind)
}

// Carried is the highest grade of the relic kind p carries.
func Carried(p *player.Player, kind string) (int, bool) { return carried(p, kind) }

// SetKit puts a copy of it in slot i of p's kit.
func SetKit(s *core.Session, p *player.Player, i int, it item.Stack) { setKit(s, p, i, it) }

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

// KitAccepts reports whether slot i of a kit takes it.
func KitAccepts(i int, it item.Stack) bool { return kitAccepts(i, it) }

// KitPut moves one item from inventory slot from into kit slot i.
func KitPut(s *core.Session, tx *world.Tx, p *player.Player, i, from int) bool {
	it, _ := p.Inventory().Item(from)
	return kitPut(s, tx, p, i, from, it)
}

// KitTake gives the item of kit slot i back.
func KitTake(s *core.Session, tx *world.Tx, p *player.Player, i int) { kitTake(s, tx, p, i) }

// DropGem leaves a gem at at; Gems is how many lie about.
func DropGem(r *Run, tx *world.Tx, p *player.Player, at mgl64.Vec3) { r.gear.drop(tx, p, r, at) }
func Gems(r *Run) int                                               { return len(r.gear.gems) }

// Describe is what it does in a hunt.
func Describe(lang string, it item.Stack) string { return describe(lang, it) }
