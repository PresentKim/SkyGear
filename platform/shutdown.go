package platform

import (
	"iter"

	"github.com/df-mc/dragonfly/server/player"
)

// DisconnectAll kicks every player with the text the player's disconnect
// screen shows. The screen shows text as it is (no translation keys, no
// colour codes). Disconnect runs once per player and takes the player out of
// the server, so the server's own close, which would send the untranslated
// default, finds nobody left. Call it right before Server.Close.
func DisconnectAll(players iter.Seq[*player.Player], text func(*player.Player) string) {
	for p := range players {
		p.Disconnect(text(p))
	}
}
