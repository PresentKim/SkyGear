// Package locator shows the island on the locator bar: the other players in
// the save and the island's spawn. It is a test of the locator bar packets
// and runs only while Enabled (platform.toml [Experimental] LocatorBar).
package locator

import (
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/google/uuid"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/island"
)

// Enabled turns the locator bar on in saves. Set it once at startup.
var Enabled bool

// spawnID is the waypoint of the island's spawn on every player's bar.
var spawnID = uuid.MustParse("6c0f3e3a-8d8e-4a9e-9e1b-2f6b5d0c1a01")

// spawnColour is the spawn waypoint's colour (gold).
const spawnColour = 0xFFC83D

// Join turns p's bar on and adds the island's spawn to it.
func Join(s *core.Session, p *player.Player) {
	sghook.SetLocatorBar(p, true)
	sghook.ShowWaypoint(p, sghook.Waypoint{ID: spawnID, Pos: island.Spawn(s), Colour: spawnColour}, false)
}

// Leave takes p off the bars of the others and clears p's bar: the spawn
// waypoint goes and the bar is turned off (the lobby has none).
func Leave(tx *world.Tx, p *player.Player) {
	for e := range tx.Players() {
		if o := e.(*player.Player); o.H() != p.H() {
			sghook.HidePlayerLocation(o, p)
		}
	}
	sghook.HideWaypoint(p, spawnID)
	sghook.SetLocatorBar(p, false)
}

// Update puts every player at its position on the bars of the others.
func Update(tx *world.Tx) {
	var ps []*player.Player
	for e := range tx.Players() {
		ps = append(ps, e.(*player.Player))
	}
	for _, viewer := range ps {
		for _, o := range ps {
			if o.H() != viewer.H() {
				sghook.ShowPlayerLocation(viewer, o)
			}
		}
	}
}
