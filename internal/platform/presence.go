package platform

import (
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/google/uuid"
	"github.com/presentkim/dragonfactory/internal/i18n"
	"github.com/presentkim/dragonfactory/internal/ui"
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

// tell sends a message to an account if it is online (ui.Tell: chat, or a
// toast in the lobby).
func (pf *Platform) tell(accountID, msg string) {
	if h, ok := pf.onlineHandle(accountID); ok {
		h.Do(func(_ *world.Tx, e world.Entity) { ui.Tell(e.(*player.Player), "", msg) })
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

// status describes where an account is: offline, lobby or a save.
func (pf *Platform) status(accountID string) string {
	if !pf.isOnline(accountID) {
		return i18n.T(i18n.Default, "platform.presence.status.1")
	}
	if inst := pf.m.Location(accountID); inst != nil {
		return "§a● '" + inst.Save.Name + i18n.T(i18n.Default, "platform.presence.status.2")
	}
	return i18n.T(i18n.Default, "platform.presence.status.3")
}
