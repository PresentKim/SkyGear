package platform

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/common/ui"
	"github.com/presentkim/dragonfactory/internal/db"
	"github.com/presentkim/dragonfactory/internal/instance"
)

// showMembers lets the host manage the members of a save
// (docs/design/01-platform.md §4, 참여자 관리): the members on the left, the
// host first and those who left last, and what can be done with the chosen
// one on the right.
func (pf *Platform) showMembers(p *player.Player, saveID int64) { pf.showMembersAt(p, saveID, "") }

// showMembersAt shows the members with the account sel chosen (the first one
// when empty).
func (pf *Platform) showMembersAt(p *player.Player, saveID int64, sel string) {
	lang := i18n.Of(p)
	members, err := pf.db.Members(saveID)
	if err != nil {
		pf.fail(p, i18n.T(lang, "platform.members.showMembers.1"), err)
		return
	}
	if sel == "" && len(members) > 0 {
		sel = members[0].AccountID
	}
	var sp ui.ListSplit
	for _, m := range members {
		state := pf.status(m.AccountID)
		switch {
		case m.Role == db.RoleHost:
			state = i18n.T(lang, "platform.members.showMembers.2")
		case m.Left:
			state = i18n.T(lang, "platform.members.showMembers.3")
		}
		sp.Rows = append(sp.Rows, ui.ListRow{
			Label: ui.ListRowText(m.Name, state), Icon: iconFriend,
			Selected: m.AccountID == sel,
			Run:      func(p *player.Player, _ *world.Tx) { pf.showMembersAt(p, saveID, m.AccountID) },
		})
		if m.AccountID == sel {
			sp.Actions = pf.memberActions(p, saveID, m)
		}
	}
	n, _ := pf.db.ActiveMembers(saveID)
	sp.Body = fmt.Sprintf(i18n.T(lang, "platform.members.showMembers.5"), n, MaxMembers)
	if pf.m.Location(instance.AccountID(p)) == nil {
		sp.Actions = append(sp.Actions, pf.lobbyBack(p, func(p *player.Player) { pf.showSaves(p, saveID) }))
	}
	ui.ListSplitMenu(p, i18n.T(lang, "platform.members.showMembers.4"), sp)
}

// memberActions are what the host does with a member: see the inventory and,
// for a guest still in the save, kick.
func (pf *Platform) memberActions(p *player.Player, saveID int64, m db.Member) []ui.Option {
	lang := i18n.Of(p)
	opts := []ui.Option{{Label: i18n.T(lang, "platform.members.showMember.1"), Icon: iconInvSee, Run: func(p *player.Player, tx *world.Tx) { pf.showInvSee(p, tx, saveID, m) }}}
	// a member in the same save as p: go to them, or bring them here
	if here := pf.m.Location(instance.AccountID(p)); here != nil && here.Save.ID == saveID && m.AccountID != instance.AccountID(p) && pf.m.Location(m.AccountID) == here {
		opts = append(opts,
			ui.Option{Label: i18n.T(lang, "platform.members.goTo"), Icon: iconEnter, Run: func(p *player.Player, tx *world.Tx) {
				if t, ok := playerIn(tx, m.AccountID); ok {
					p.Teleport(t.Position())
				}
			}},
			ui.Option{Label: i18n.T(lang, "platform.members.bring"), Icon: iconInvite, Run: func(p *player.Player, tx *world.Tx) {
				if t, ok := playerIn(tx, m.AccountID); ok {
					t.Teleport(p.Position())
					ui.Notify(t, i18n.T(i18n.Of(t), "platform.members.brought"), p.Name())
				}
			}})
	}
	if m.Role != db.RoleHost && !m.Left {
		opts = append(opts, ui.Option{Label: i18n.T(lang, "platform.members.showMember.2"), Icon: iconKick, Run: func(p *player.Player, _ *world.Tx) {
			pf.confirm(p, i18n.T(lang, "platform.members.showMember.3"), m.Name+i18n.T(lang, "platform.members.showMember.4"), i18n.T(lang, "platform.members.showMember.3"), func(p *player.Player, _ *world.Tx) {
				pf.kick(p, saveID, m)
				pf.showMembersAt(p, saveID, m.AccountID)
			}, func(p *player.Player, _ *world.Tx) { pf.showMembersAt(p, saveID, m.AccountID) })
		}})
	}
	return opts
}

// playerIn is the player of the account in tx's world.
func playerIn(tx *world.Tx, accountID string) (*player.Player, bool) {
	for e := range tx.Players() {
		if p := e.(*player.Player); instance.AccountID(p) == accountID {
			return p, true
		}
	}
	return nil, false
}

func (pf *Platform) kick(p *player.Player, saveID int64, m db.Member) {
	if err := pf.db.LeaveMember(saveID, m.AccountID); err != nil {
		pf.fail(p, i18n.T(i18n.Of(p), "platform.members.kick.1"), err)
		return
	}
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

// managedSave is the save p manages from inside it: the one it hosts, or as
// an operator (platform.toml [Admin]) any save it is in; nil otherwise.
func (pf *Platform) managedSave(p *player.Player) *instance.Instance {
	if inst := pf.hostedSave(p); inst != nil {
		return inst
	}
	if pf.isOperator != nil && pf.isOperator(p) {
		return pf.m.Location(instance.AccountID(p))
	}
	return nil
}

// usePlayer opens the management of target for p when p sneaks and
// right-clicks it in a save it manages (managedSave): the member list with
// target chosen, its inventory and kick on the right. It reports whether it
// took the click.
func (pf *Platform) usePlayer(p, target *player.Player) bool {
	if !p.Sneaking() {
		return false
	}
	inst := pf.managedSave(p)
	if inst == nil || pf.m.Location(instance.AccountID(target)) != inst {
		return false
	}
	pf.showMembersAt(p, inst.Save.ID, instance.AccountID(target))
	return true
}

// membersCmd opens member management from inside the save (host, or an
// operator in any save).
type membersCmd struct {
	playerOnly
	pf *Platform
}

func (c membersCmd) Run(src cmd.Source, o *cmd.Output, _ *world.Tx) {
	p := src.(*player.Player)
	inst := c.pf.managedSave(p)
	if inst == nil {
		o.Error(i18n.T(i18n.Of(p), "platform.members.Run.1"))
		return
	}
	c.pf.showMembers(p, inst.Save.ID)
}
