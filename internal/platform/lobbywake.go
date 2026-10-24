package platform

import (
	"strings"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/dfhook"
	"github.com/presentkim/dragonfactory/common/ui"
	"github.com/presentkim/dragonfactory/internal/instance"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// The lobby's main menu is open whenever no other form is (docs/design/
// 01-platform.md §1): when a form of the lobby is answered or closed and no
// form follows, the main menu comes back at once. Closing the main menu
// itself (Esc) leaves it closed for lobbyEscDelay, or until the player's next
// action: a click, a movement key, a jump, a sneak, the use key or a hotbar
// slot. The pack draws the menu on the HUD, so the screen looks the same
// either way. Movement is locked in the lobby, so the movement keys reach the
// server only as the raw move vector of the client's input packet, read by a
// packet hook.

// lobbyWakeGap is how often an input of a player can ask for the menu: the
// client sends its input every tick.
const lobbyWakeGap = 300 * time.Millisecond

// lobbyEscDelay is how long the main menu stays closed after the player
// closed it.
const lobbyEscDelay = 1500 * time.Millisecond

// lobbyIdle is ui.Idle: p answered a form and has none open. In the lobby the
// main menu opens again, at once or, when p closed the main menu itself,
// after lobbyEscDelay.
func (pf *Platform) lobbyIdle(p *player.Player, _ *world.Tx, title string, closed bool) {
	if pf.m.Location(instance.AccountID(p)) != nil {
		return
	}
	if closed && strings.Contains(title, ui.LauncherToken) {
		p.H().DoAfter(lobbyEscDelay, func(_ *world.Tx, e world.Entity) {
			if p := e.(*player.Player); !ui.FormOpen(p) {
				pf.wakeLobbyMenu(p)
			}
		})
		return
	}
	pf.showLobbyMenu(p)
}

// lobbyConns are the hooked connections of the players in the lobby, with
// their handle and the time of their last wake.
var lobbyConns = struct {
	sync.Mutex
	m map[*dfhook.Conn]*lobbyConn
}{m: map[*dfhook.Conn]*lobbyConn{}}

type lobbyConn struct {
	h    *world.EntityHandle
	w    *world.World
	last time.Time
}

// watchLobbyInput makes the input of p, who arrived in the lobby w, open the
// main menu.
func watchLobbyInput(p *player.Player, w *world.World) {
	c := dfhook.Of(p)
	if c == nil {
		return
	}
	lobbyConns.Lock()
	lobbyConns.m[c] = &lobbyConn{h: p.H(), w: w}
	lobbyConns.Unlock()
}

// lobbyPlatform is the platform the input hook wakes the menu with.
var lobbyPlatform *Platform

func init() {
	dfhook.OnClose(func(c *dfhook.Conn) {
		lobbyConns.Lock()
		delete(lobbyConns.m, c)
		lobbyConns.Unlock()
	})
	dfhook.OnClientPacket(func(c *dfhook.Conn, pk packet.Packet) bool {
		in, ok := pk.(*packet.PlayerAuthInput)
		if !ok || !lobbyInput(in) {
			return false
		}
		lobbyConns.Lock()
		lc := lobbyConns.m[c]
		now := time.Now()
		if lc == nil || now.Sub(lc.last) < lobbyWakeGap {
			lobbyConns.Unlock()
			return false
		}
		lc.last = now
		lobbyConns.Unlock()
		lc.w.Do(func(tx *world.Tx) {
			if e, ok := lc.h.Entity(tx); ok {
				if p, ok := e.(*player.Player); ok && lobbyPlatform != nil {
					lobbyPlatform.wakeLobbyMenu(p)
				}
			}
		})
		return false
	})
}

// lobbyInput reports whether an input packet holds an action of the player:
// a movement key, a jump or a sneak.
func lobbyInput(in *packet.PlayerAuthInput) bool {
	if in.RawMoveVector.Len() > 0.1 || in.MoveVector.Len() > 0.1 {
		return true
	}
	for _, f := range []int{packet.InputFlagJumpPressedRaw, packet.InputFlagSneakPressedRaw, packet.InputFlagStartJumping, packet.InputFlagStartSneaking} {
		if in.InputData.Load(f) {
			return true
		}
	}
	return false
}

// wakeLobbyMenu shows the main menu to p if p is still in the lobby. A menu
// that is already open is not sent again (ui's guard against repeated forms).
func (pf *Platform) wakeLobbyMenu(p *player.Player) {
	if pf.m.Location(instance.AccountID(p)) == nil {
		pf.showLobbyMenu(p)
	}
}
