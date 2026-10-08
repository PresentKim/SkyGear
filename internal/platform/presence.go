package platform

import (
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/google/uuid"
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

// tell sends a chat message to an account if it is online.
func (pf *Platform) tell(accountID, msg string) {
	if h, ok := pf.onlineHandle(accountID); ok {
		h.Do(func(_ *world.Tx, e world.Entity) { e.(*player.Player).Message(msg) })
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
		return "§7● 오프라인"
	}
	if inst := pf.m.Location(accountID); inst != nil {
		return "§a● '" + inst.Save.Name + "' 플레이 중"
	}
	return "§a● 로비"
}
