package platform

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/internal/db"
	"github.com/presentkim/skygear/internal/instance"
)

// showMembers lets the host manage the members of a save: host first, those
// who left last.
func (pf *Platform) showMembers(p *player.Player, saveID int64) { pf.showMembersAt(p, saveID, "") }

// showMembersAt shows the members with the account sel chosen (the first one
// when empty).
func (pf *Platform) showMembersAt(p *player.Player, saveID int64, sel string) {
	lang := i18n.Of(p)
	members, err := pf.db.Members(saveID)
	if err != nil {
		pf.fail(p, i18n.T(lang, "platform.members.loading"), err)
		return
	}
	if sel == "" && len(members) > 0 {
		sel = members[0].AccountID
	}
	var sp ui.ListSplit
	active := 0
	for _, m := range members {
		if !m.Left {
			active++
		}
		state := pf.status(lang, m.AccountID)
		switch {
		case m.Role == db.RoleHost:
			state = i18n.T(lang, "platform.members.host")
		case m.Left:
			state = i18n.T(lang, "platform.members.gone")
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
	sp.Body = fmt.Sprintf(i18n.T(lang, "platform.members.body"), active, MaxMembers)
	if pf.m.Location(instance.AccountID(p)) == nil {
		sp.Actions = append(sp.Actions, pf.lobbyBack(p, func(p *player.Player) { pf.showSaves(p, saveID) }))
	} else {
		sp.Actions = append(sp.Actions, ui.Back(i18n.T(lang, "platform.back"), func(p *player.Player, _ *world.Tx) { pf.menuBack(p, instance.ScreenMembers) }))
	}
	ui.ListSplitMenu(p, i18n.T(lang, "platform.members.title"), sp)
}

// memberActions are what the host does with a member: see the inventory and,
// for a guest still in the save, kick.
func (pf *Platform) memberActions(p *player.Player, saveID int64, m db.Member) []ui.Option {
	lang := i18n.Of(p)
	opts := []ui.Option{{Label: i18n.T(lang, "platform.members.invsee"), Icon: iconInvSee, Run: func(p *player.Player, tx *world.Tx) { pf.showInvSee(p, tx, saveID, m) }}}
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
		opts = append(opts, ui.Option{Label: i18n.T(lang, "platform.members.kick"), Icon: iconKick, Run: func(p *player.Player, _ *world.Tx) {
			pf.confirm(p, i18n.T(lang, "platform.members.kick.title"), m.Name+i18n.T(lang, "platform.members.kick.ask"), i18n.T(lang, "platform.members.kick.title"), func(p *player.Player, _ *world.Tx) {
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
		pf.fail(p, i18n.T(i18n.Of(p), "platform.members.kicking"), err)
		return
	}
	pf.invites.remove(m.AccountID, saveID)
	pf.with(m.AccountID, func(tp *player.Player) {
		if inst := pf.m.Location(m.AccountID); inst != nil && inst.Save.ID == saveID {
			ui.Tell(tp, "", i18n.T(i18n.Of(tp), "platform.members.kicked"))
			pf.m.ToLobby(tp)
			return
		}
		ui.Tell(tp, "", i18n.T(i18n.Of(tp), "platform.members.kickedOffline"))
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
		o.Error(i18n.T(i18n.Of(p), "platform.members.hostOnly"))
		return
	}
	c.pf.showMembers(p, inst.Save.ID)
}
