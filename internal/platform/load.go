package platform

import (
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/hud"
	"github.com/presentkim/skygear/common/perf"
	"github.com/presentkim/skygear/common/ui"
)

// lobbyHUD is what each lobby player's HUD was last sent, by name, so a
// board is sent only when it changes.
var lobbyHUD = struct {
	sync.Mutex
	sent map[string]string
}{sent: map[string]string{}}

// sendLobbyHUD shows p the lobby's launcher frame (hud.Board.Frame; not to a
// player who sees DDUI screens, whose main screen draws its own) and, if load
// is not empty, the server load line at the top left.
func sendLobbyHUD(p *player.Player, load string) {
	b := hud.Board{Frame: !ui.OreUIFor(p)}
	if load != "" {
		b.Add(hud.TopLeft, load)
	}
	key := b.Key()
	lobbyHUD.Lock()
	same := lobbyHUD.sent[p.Name()] == key
	lobbyHUD.sent[p.Name()] = key
	lobbyHUD.Unlock()
	if !same {
		b.Send(p)
	}
}

// forgetLobbyHUD drops what p was sent, so the next board is sent again
// (p left the lobby, or arrives with another HUD on screen).
func forgetLobbyHUD(p *player.Player) {
	lobbyHUD.Lock()
	delete(lobbyHUD.sent, p.Name())
	lobbyHUD.Unlock()
}

// ShowLoad updates the lobby HUD and scene every second until stop closes;
// operators who turned it on (/sgadmin tps) also get the server load line.
func ShowLoad(lobby *world.World, isOperator func(*player.Player) bool, stop <-chan struct{}) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			lobby.Do(func(tx *world.Tx) {
				now := time.Now()
				for e := range tx.Players() {
					p := e.(*player.Player)
					moveLobbyScene(p, now)
					load := ""
					if isOperator(p) && perf.Shown(p.Name()) {
						load = perf.Line(lobby)
					}
					sendLobbyHUD(p, load)
				}
			})
		}
	}
}
