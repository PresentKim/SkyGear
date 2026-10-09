package platform

import (
	"time"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/scoreboard"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/perf"
)

// ShowLoad shows operators in the lobby a sidebar with the server load
// (internal/perf), refreshed every second until stop is closed. Games show
// the same lines on their own sidebars.
func ShowLoad(lobby *world.World, isOperator func(*player.Player) bool, stop <-chan struct{}) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			lobby.Do(func(tx *world.Tx) {
				for e := range tx.Players() {
					if p := e.(*player.Player); isOperator(p) {
						sendLoadBoard(p, lobby)
					}
				}
			})
		}
	}
}

func sendLoadBoard(p *player.Player, w *world.World) {
	b := scoreboard.New("§l§6서버 상태")
	for i, l := range perf.Lines(w) {
		b.Set(i, l)
	}
	p.SendScoreboard(b)
}
