package platform

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/google/uuid"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/common/ui"
)

// Online looks up the entity handle of an online player by UUID (the
// server's Player method).
type Online func(id uuid.UUID) (*world.EntityHandle, bool)

// onlineHandle returns the handle of an online account.
func (pf *Platform) onlineHandle(accountID string) (*world.EntityHandle, bool) {
	id, err := uuid.Parse(accountID)
	if err != nil || pf.online == nil {
		return nil, false
	}
	return pf.online(id)
}

// isOnline reports if an account is connected.
func (pf *Platform) isOnline(accountID string) bool {
	_, ok := pf.onlineHandle(accountID)
	return ok
}

// tell sends the catalog text key, filled with args, to an account if it is
// online (ui.Tell: chat, or a toast in the lobby), in that player's language.
func (pf *Platform) tell(accountID, key string, args ...any) {
	if h, ok := pf.onlineHandle(accountID); ok {
		h.Do(func(_ *world.Tx, e world.Entity) {
			p := e.(*player.Player)
			ui.Tell(p, "", fmt.Sprintf(i18n.T(i18n.Of(p), key), args...))
		})
	}
}

// with runs f with an online account's player in its current world.
func (pf *Platform) with(accountID string, f func(p *player.Player)) bool {
	h, ok := pf.onlineHandle(accountID)
	if ok {
		h.Do(func(_ *world.Tx, e world.Entity) { f(e.(*player.Player)) })
	}
	return ok
}

// status describes in lang where an account is: offline, lobby or a save.
func (pf *Platform) status(lang, accountID string) string {
	return pf.statusOf(lang, accountID, pf.isOnline(accountID))
}

// statusOf is status for an account already known to be online or not.
func (pf *Platform) statusOf(lang, accountID string, online bool) string {
	if !online {
		return i18n.T(lang, "platform.presence.offline")
	}
	if inst := pf.m.Location(accountID); inst != nil {
		return "§a● '" + inst.Save.Name + i18n.T(lang, "platform.presence.playing")
	}
	return i18n.T(lang, "platform.presence.lobby")
}
