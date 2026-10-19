package platform

import (
	"fmt"
	"github.com/presentkim/dragonfactory/internal/i18n"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/db"
	"github.com/presentkim/dragonfactory/internal/instance"
	"github.com/presentkim/dragonfactory/internal/ui"
)

// showMembers lets the host manage the members of a save
// (docs/design/01-platform.md §4, 참여자 관리).
func (pf *Platform) showMembers(p *player.Player, saveID int64) {
	members, err := pf.db.Members(saveID)
	if err != nil {
		pf.fail(p, i18n.T(i18n.Of(p), "platform.members.showMembers.1"), err)
		return
	}
	var opts []ui.Option
	for _, m := range members {
		state := pf.status(m.AccountID)
		switch {
		case m.Role == db.RoleHost:
			state = i18n.T(i18n.Of(p), "platform.members.showMembers.2")
		case m.Left:
			state = i18n.T(i18n.Of(p), "platform.members.showMembers.3")
		}
		opts = append(opts, ui.Option{Label: fmt.Sprintf("%s\n%s", m.Name, state), Icon: iconPlayer, Run: func(p *player.Player, _ *world.Tx) {
			pf.showMember(p, saveID, m)
		}})
	}
	n, _ := pf.db.ActiveMembers(saveID)
	ui.Menu(p, i18n.T(i18n.Of(p), "platform.members.showMembers.4"), fmt.Sprintf(i18n.T(i18n.Of(p), "platform.members.showMembers.5"), n, MaxMembers), opts...)
}

func (pf *Platform) showMember(p *player.Player, saveID int64, m db.Member) {
	opts := []ui.Option{{Label: i18n.T(i18n.Of(p), "platform.members.showMember.1"), Icon: iconInvSee, Run: func(p *player.Player, tx *world.Tx) { pf.showInvSee(p, tx, saveID, m) }}}
	if m.Role != db.RoleHost && !m.Left {
		opts = append(opts, ui.Option{Label: i18n.T(i18n.Of(p), "platform.members.showMember.2"), Icon: iconKick, Run: func(p *player.Player, _ *world.Tx) {
			ui.Confirm(p, i18n.T(i18n.Of(p), "platform.members.showMember.3"), m.Name+i18n.T(i18n.Of(p), "platform.members.showMember.4"), i18n.T(i18n.Of(p), "platform.members.showMember.5"), func(p *player.Player, _ *world.Tx) {
				pf.kick(p, saveID, m)
			})
		}})
	}
	opts = append(opts, ui.Back(i18n.T(i18n.Of(p), "platform.members.showMember.6"), func(p *player.Player, _ *world.Tx) { pf.showMembers(p, saveID) }))
	ui.Menu(p, m.Name, fmt.Sprintf("%s\n%s", m.Name, pf.status(m.AccountID)), opts...)
}

func (pf *Platform) kick(p *player.Player, saveID int64, m db.Member) {
	if err := pf.db.LeaveMember(saveID, m.AccountID); err != nil {
		pf.fail(p, i18n.T(i18n.Of(p), "platform.members.kick.1"), err)
		return
	}
	ui.Flash(p, i18n.T(i18n.Of(p), "platform.members.kick.2"), m.Name)
	pf.invites.remove(m.AccountID, saveID)
	pf.with(m.AccountID, func(tp *player.Player) {
		if inst := pf.m.Location(m.AccountID); inst != nil && inst.Save.ID == saveID {
			ui.Tell(tp, "", i18n.T(i18n.Of(tp), "platform.members.kick.3"))
			pf.m.ToLobby(tp)
			return
		}
		ui.Tell(tp, "", i18n.T(i18n.Of(tp), "platform.members.kick.4"))
	})
}

// membersCmd opens member management from inside the save (host only).
type membersCmd struct {
	playerOnly
	pf *Platform
}

func (c membersCmd) Run(src cmd.Source, o *cmd.Output, _ *world.Tx) {
	p := src.(*player.Player)
	inst := c.pf.m.Location(instance.AccountID(p))
	if inst == nil || inst.Host() != instance.AccountID(p) {
		o.Error(i18n.T(i18n.Of(p), "platform.members.Run.1"))
		return
	}
	c.pf.showMembers(p, inst.Save.ID)
}
