package sghook

// Locator bar: the bar at the top of the screen that points at waypoints and
// other players.

import (
	"strings"
	_ "unsafe" // go:linkname

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// LocatorBarRule is the game rule that shows the locator bar. Dragonfly
// turns it off in the start game data.
const LocatorBarRule = "locatorBar"

// startLocatorRules turns the locator bar on in the start game data.
// Dragonfly sends it off there (as "locatorBar"), and turning it on later
// with GameRulesChanged did not show the bar in a client under any name, so
// the start data carries it on, under the lower case name the other game
// rules use and under Dragonfly's.
func startLocatorRules(rules []protocol.GameRule) []protocol.GameRule {
	out := rules[:0:0]
	for _, r := range rules {
		if !strings.EqualFold(r.Name, LocatorBarRule) {
			out = append(out, r)
		}
	}
	return append(out, protocol.GameRule{Name: "locatorbar", Value: true}, protocol.GameRule{Name: LocatorBarRule, Value: true})
}

// SetLocatorBar shows or hides p's locator bar (the game rule, for p only).
// Turning it off is sent only if it was turned on.
func SetLocatorBar(p *player.Player, on bool) {
	c := Of(p)
	if c == nil || c.locatorBar.Swap(on) == on && !on {
		return
	}
	write(p, &packet.GameRulesChanged{GameRules: []protocol.GameRule{{Name: "locatorbar", Value: on}, {Name: LocatorBarRule, Value: on}}})
}

// Waypoint is a mark on p's locator bar.
type Waypoint struct {
	// ID tells p's waypoints apart: showing a waypoint with an ID p already
	// has updates it.
	ID uuid.UUID
	// Pos is where it points (in the overworld).
	Pos mgl64.Vec3
	// Colour is the icon's RGB colour (0xRRGGBB).
	Colour int32
}

// ShowWaypoint adds w to p's locator bar, or moves it if p has it.
func ShowWaypoint(p *player.Player, w Waypoint, update bool) {
	action := uint8(protocol.WaypointActionAdd)
	if update {
		action = protocol.WaypointActionUpdate
	}
	write(p, &packet.LocatorBar{Waypoints: []protocol.LocatorBarWaypoint{{
		GroupHandle: w.ID,
		Action:      action,
		Waypoint: protocol.Waypoint{
			UpdateFlag:    protocol.WaypointUpdateFlagVisible | protocol.WaypointUpdateFlagPosition | protocol.WaypointUpdateFlagColour,
			Visible:       protocol.Option(true),
			WorldPosition: protocol.Option(protocol.WaypointWorldPosition{Position: vec32(w.Pos)}),
			Colour:        protocol.Option(w.Colour),
		},
	}}})
}

// HideWaypoint removes the waypoint id from p's locator bar.
func HideWaypoint(p *player.Player, id uuid.UUID) {
	write(p, &packet.LocatorBar{Waypoints: []protocol.LocatorBarWaypoint{{GroupHandle: id, Action: protocol.WaypointActionRemove}}})
}

// ShowPlayerLocation puts other at its position on p's locator bar, if p's
// client knows other.
func ShowPlayerLocation(p, other *player.Player) {
	if id := EntityRuntimeID(p, other); id != 0 {
		write(p, &packet.PlayerLocation{Type: packet.PlayerLocationTypeCoordinates, EntityUniqueID: int64(id), Position: vec32(other.Position())})
	}
}

// HidePlayerLocation takes other off p's locator bar.
func HidePlayerLocation(p, other *player.Player) {
	if id := EntityRuntimeID(p, other); id != 0 {
		write(p, &packet.PlayerLocation{Type: packet.PlayerLocationTypeHide, EntityUniqueID: int64(id)})
	}
}

// EntityRuntimeID returns the ID p's client knows e by (its runtime and
// unique ID in Dragonfly), or 0 if p's client does not know e.
func EntityRuntimeID(p *player.Player, e world.Entity) uint64 {
	s := Session(p)
	if s == session.Nop {
		return 0
	}
	if e.H() == p.H() {
		return selfRuntimeID
	}
	return session_handleRuntimeID(s, e.H())
}

//go:linkname session_handleRuntimeID github.com/df-mc/dragonfly/server/session.(*Session).handleRuntimeID
func session_handleRuntimeID(*session.Session, *world.EntityHandle) uint64
