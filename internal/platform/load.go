package platform

import (
	"time"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/hud"
	"github.com/presentkim/dragonfactory/internal/perf"
)

// ShowLoad shows operators in the lobby who turned it on (/dfadmin tps,
// perf.Toggle) the server load at the top left of the HUD, refreshed every
// second until stop is closed. Games show the same line on their own HUD.
func ShowLoad(lobby *world.World, isOperator func(*player.Player) bool, stop <-chan struct{}) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	// showing are the players the line was last sent to, to clear it once
	// when turned off.
	showing := map[string]bool{}
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			lobby.Do(func(tx *world.Tx) {
				for e := range tx.Players() {
					p := e.(*player.Player)
					on := isOperator(p) && perf.Shown(p.Name())
					if !on && !showing[p.Name()] {
						continue
					}
					var b hud.Board
					if on {
						b.Add(hud.TopLeft, perf.Line(lobby))
					}
					b.Send(p)
					showing[p.Name()] = on
				}
			})
		}
	}
}
