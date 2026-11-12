package core

import (
	"fmt"
	"sync"

	"github.com/df-mc/dragonfly/server/player"
)

// A takeover is a mode in which one feature holds a player's game mode, camera
// or body and gives them back when the mode ends (island expansion mode,
// hunting, watching a hunt). Two takeovers must not nest: each saved what to
// give back when it began. A feature registers its takeover in Setup and asks
// Takeover before it begins one.

type takeover struct {
	name  string
	holds func(s *Session, p *player.Player) bool
}

var (
	takeoversMu sync.Mutex
	// takeovers are in registration order. They are registered in Setup,
	// before any player is asked about, so Takeover reads them without a lock.
	takeovers []takeover
)

// RegisterTakeover adds a takeover. name is "<feature>" or "<feature>.<mode>";
// holds reports whether p is in it. It panics on an empty or repeated name or
// a nil holds.
func RegisterTakeover(name string, holds func(s *Session, p *player.Player) bool) {
	takeoversMu.Lock()
	defer takeoversMu.Unlock()
	switch {
	case name == "":
		panic("core: RegisterTakeover: empty name")
	case holds == nil:
		panic(fmt.Sprintf("core: RegisterTakeover: takeover %q has a nil holds", name))
	}
	for _, t := range takeovers {
		if t.name == name {
			panic(fmt.Sprintf("core: RegisterTakeover: takeover %q registered twice", name))
		}
	}
	takeovers = append(takeovers, takeover{name: name, holds: holds})
}

// Takeover returns the name of the takeover that holds p, or "" when none
// does. When several hold p, the first registered is named.
func Takeover(s *Session, p *player.Player) string {
	for i := range takeovers {
		if takeovers[i].holds(s, p) {
			return takeovers[i].name
		}
	}
	return ""
}
