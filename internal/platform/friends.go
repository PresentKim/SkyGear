package platform

import (
	"errors"
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/common/ui"
	"github.com/presentkim/dragonfactory/internal/db"
	"github.com/presentkim/dragonfactory/internal/instance"
)

// The tabs of the friends screen.
const (
	friendsTab = iota
	requestsTab
	blockedTab
)

// codeRow is the id of the row with the invite code, on the friends tab
// inside the host's save.
const codeRow = "+code"

// showFriends is the friends screen (docs/design/01-platform.md §4): tabs for
// the friends, the requests received and the blocked players, the list on the
// left and what can be done with the chosen one on the right. Inside a save
// the host invites an online friend from here.
func (pf *Platform) showFriends(p *player.Player) { pf.showFriendsAt(p, friendsTab, "") }

// showFriendsAt shows the tab with the row sel chosen (an account ID or
// codeRow; the first one when empty).
func (pf *Platform) showFriendsAt(p *player.Player, tab int, sel string) {
	lang := i18n.Of(p)
	acc := instance.AccountID(p)
	reqs, err := pf.db.FriendRequests(acc)
	if err != nil {
		pf.fail(p, i18n.T(lang, "platform.friends.load"), err)
		return
	}
	var sp ui.ListSplit
	switch tab {
	case requestsTab:
		sp = pf.requestList(p, reqs, sel)
	case blockedTab:
		sp, err = pf.blockedList(p, sel)
	default:
		sp, err = pf.friendList(p, sel)
	}
	if err != nil {
		pf.fail(p, i18n.T(lang, "platform.friends.load"), err)
		return
	}
	tabLabel := func(key string, n int) string {
		if n > 0 {
			return fmt.Sprintf("%s §e%d", i18n.T(lang, key), n)
		}
		return i18n.T(lang, key)
	}
	tabTo := func(t int) func(p *player.Player, _ *world.Tx) {
		return func(p *player.Player, _ *world.Tx) { pf.showFriendsAt(p, t, "") }
	}
	sp.Actions = append(sp.Actions,
		ui.Tab(i18n.T(lang, "platform.friends.tab.friends"), iconFriend, tab == friendsTab, tabTo(friendsTab)),
		ui.Tab(tabLabel("platform.friends.tab.requests", len(reqs)), iconRequests, tab == requestsTab, tabTo(requestsTab)),
		ui.Tab(i18n.T(lang, "platform.friends.tab.blocked"), iconBlocked, tab == blockedTab, tabTo(blockedTab)),
	)
	sp.Actions = append(sp.Actions, pf.lobbyBacks(p)...)
	ui.ListSplitMenu(p, i18n.T(lang, "platform.friends.title"), sp)
}

// hostedSave is the save p is hosting and playing in, nil in the lobby or
// as a guest.
func (pf *Platform) hostedSave(p *player.Player) *instance.Instance {
	inst := pf.m.Location(instance.AccountID(p))
	if inst == nil || inst.Host() != instance.AccountID(p) {
		return nil
	}
	return inst
}

// friendList is the friends tab: the invite code in the host's save, the
// row that adds a friend (pressing it asks for the name at once: no button on
// the right does the same), then the friends, online ones first.
func (pf *Platform) friendList(p *player.Player, sel string) (ui.ListSplit, error) {
	lang := i18n.Of(p)
	friends, err := pf.db.Friends(instance.AccountID(p))
	if err != nil {
		return ui.ListSplit{}, err
	}
	var online, offline []db.Account
	for _, f := range friends {
		if pf.isOnline(f.ID) {
			online = append(online, f)
		} else {
			offline = append(offline, f)
		}
	}
	friends = append(online, offline...)
	host := pf.hostedSave(p)
	if sel == "" {
		switch {
		case host != nil:
			sel = codeRow
		case len(friends) > 0:
			sel = friends[0].ID
		}
	}
	again := func(id string) func(p *player.Player, _ *world.Tx) {
		return func(p *player.Player, _ *world.Tx) { pf.showFriendsAt(p, friendsTab, id) }
	}
	var sp ui.ListSplit
	if host != nil {
		code, err := pf.inviteCode(host.Save, false)
		if err != nil {
			return ui.ListSplit{}, err
		}
		sp.Rows = append(sp.Rows, ui.ListRow{
			Label: ui.ListRowText(i18n.T(lang, "platform.friends.code"), "§e"+code), Icon: iconCode,
			Selected: sel == codeRow, Run: again(codeRow),
		})
	}
	sp.Rows = append(sp.Rows, ui.ListRow{
		Label: ui.ListRowText(i18n.T(lang, "platform.friends.add"), ""), Icon: iconAdd,
		Run: func(p *player.Player, _ *world.Tx) { pf.askAddFriend(p) },
	})
	var chosen db.Account
	for _, f := range friends {
		if f.ID == sel {
			chosen = f
		}
		sp.Rows = append(sp.Rows, ui.ListRow{
			Label: ui.ListRowText(f.Name, pf.status(lang, f.ID)), Icon: iconFriend,
			Selected: f.ID == sel, Run: again(f.ID),
		})
	}
	switch {
	case sel == codeRow && host != nil:
		saveID := host.Save.ID
		sp.Body = i18n.T(lang, "platform.friends.code.body")
		sp.Actions = []ui.Option{{Label: i18n.T(lang, "platform.invite.renew"), Icon: iconCode, Run: func(p *player.Player, _ *world.Tx) {
			pf.confirm(p, i18n.T(lang, "platform.invite.renew"), i18n.T(lang, "platform.invite.renew.ask"), i18n.T(lang, "platform.invite.renew"), func(p *player.Player, _ *world.Tx) {
				s, err := pf.db.Save(saveID)
				if err == nil {
					_, err = pf.inviteCode(s, true)
				}
				if err != nil {
					pf.fail(p, i18n.T(lang, "platform.invite.renew"), err)
					return
				}
				pf.showFriendsAt(p, friendsTab, codeRow)
			}, again(codeRow))
		}}}
	case chosen.ID != "":
		sp.Actions = pf.friendActions(p, chosen, host)
	default:
		sp.Body = i18n.T(lang, "platform.friends.none")
	}
	return sp, nil
}

// friendActions are what can be done with a friend: invite them to the save
// p hosts (when they are online and not in it), remove and block.
func (pf *Platform) friendActions(p *player.Player, f db.Account, host *instance.Instance) []ui.Option {
	lang := i18n.Of(p)
	back := func(p *player.Player, _ *world.Tx) { pf.showFriendsAt(p, friendsTab, f.ID) }
	var opts []ui.Option
	if host != nil && pf.isOnline(f.ID) && pf.m.Location(f.ID) != host {
		opts = append(opts, ui.Option{Label: i18n.T(lang, "platform.friends.invite"), Icon: iconInvite, Run: func(p *player.Player, _ *world.Tx) {
			pf.sendInvite(p, host, f)
			pf.showFriendsAt(p, friendsTab, f.ID)
		}})
	}
	opts = append(opts,
		ui.Option{Label: i18n.T(lang, "platform.friends.remove"), Icon: iconDelete, Run: func(p *player.Player, _ *world.Tx) {
			pf.confirm(p, i18n.T(lang, "platform.friends.remove"), fmt.Sprintf(i18n.T(lang, "platform.friends.remove.ask"), f.Name), i18n.T(lang, "platform.friends.remove"), func(p *player.Player, _ *world.Tx) {
				if err := pf.db.RemoveFriend(instance.AccountID(p), f.ID); err != nil {
					pf.fail(p, i18n.T(lang, "platform.friends.remove"), err)
					return
				}
				pf.showFriendsAt(p, friendsTab, "")
			}, back)
		}},
		pf.blockOption(p, f, friendsTab),
	)
	return opts
}

// blockOption blocks the account after asking, then shows the tab again.
func (pf *Platform) blockOption(p *player.Player, a db.Account, tab int) ui.Option {
	lang := i18n.Of(p)
	return ui.Option{Label: i18n.T(lang, "platform.friends.block"), Icon: iconBlocked, Run: func(p *player.Player, _ *world.Tx) {
		pf.confirm(p, i18n.T(lang, "platform.friends.block"), fmt.Sprintf(i18n.T(lang, "platform.friends.block.ask"), a.Name), i18n.T(lang, "platform.friends.block"), func(p *player.Player, _ *world.Tx) {
			if err := pf.db.Block(instance.AccountID(p), a.ID); err != nil {
				pf.fail(p, i18n.T(lang, "platform.friends.block"), err)
				return
			}
			pf.showFriendsAt(p, tab, "")
		}, func(p *player.Player, _ *world.Tx) { pf.showFriendsAt(p, tab, a.ID) })
	}}
}

// requestList is the requests tab: accept, decline or block the chosen one.
func (pf *Platform) requestList(p *player.Player, reqs []db.Account, sel string) ui.ListSplit {
	lang := i18n.Of(p)
	var sp ui.ListSplit
	var chosen db.Account
	for i, r := range reqs {
		if r.ID == sel || (sel == "" && i == 0) {
			chosen = r
		}
	}
	for _, r := range reqs {
		sp.Rows = append(sp.Rows, ui.ListRow{
			Label: ui.ListRowText(r.Name, pf.status(lang, r.ID)), Icon: iconFriend,
			Selected: r.ID == chosen.ID,
			Run:      func(p *player.Player, _ *world.Tx) { pf.showFriendsAt(p, requestsTab, r.ID) },
		})
	}
	if chosen.ID == "" {
		sp.Body = i18n.T(lang, "platform.friends.requests.none")
		return sp
	}
	sp.Actions = []ui.Option{
		{Label: i18n.T(lang, "platform.friends.accept"), Icon: iconAccept, Run: func(p *player.Player, _ *world.Tx) {
			if err := pf.db.AcceptFriend(instance.AccountID(p), chosen.ID); err != nil {
				pf.fail(p, i18n.T(lang, "platform.friends.accept"), err)
				return
			}
			pf.tell(chosen.ID, fmt.Sprintf(i18n.T(lang, "platform.friends.accepted"), p.Name()))
			pf.showFriendsAt(p, requestsTab, "")
		}},
		{Label: i18n.T(lang, "platform.friends.decline"), Icon: iconDecline, Run: func(p *player.Player, _ *world.Tx) {
			_ = pf.db.DeclineFriend(instance.AccountID(p), chosen.ID)
			pf.showFriendsAt(p, requestsTab, "")
		}},
		pf.blockOption(p, chosen, requestsTab),
	}
	return sp
}

// blockedList is the blocked tab: unblock the chosen one.
func (pf *Platform) blockedList(p *player.Player, sel string) (ui.ListSplit, error) {
	lang := i18n.Of(p)
	list, err := pf.db.BlockedAccounts(instance.AccountID(p))
	if err != nil {
		return ui.ListSplit{}, err
	}
	var sp ui.ListSplit
	var chosen db.Account
	for i, a := range list {
		if a.ID == sel || (sel == "" && i == 0) {
			chosen = a
		}
	}
	for _, a := range list {
		sp.Rows = append(sp.Rows, ui.ListRow{
			Label: ui.ListRowText(a.Name, ""), Icon: iconBlocked,
			Selected: a.ID == chosen.ID,
			Run:      func(p *player.Player, _ *world.Tx) { pf.showFriendsAt(p, blockedTab, a.ID) },
		})
	}
	if chosen.ID == "" {
		sp.Body = i18n.T(lang, "platform.friends.blocked.none")
		return sp, nil
	}
	sp.Body = i18n.T(lang, "platform.friends.blocked.body")
	sp.Actions = []ui.Option{{Label: i18n.T(lang, "platform.friends.unblock"), Icon: iconAccept, Run: func(p *player.Player, _ *world.Tx) {
		_ = pf.db.Unblock(instance.AccountID(p), chosen.ID)
		pf.showFriendsAt(p, blockedTab, "")
	}}}
	return sp, nil
}

// askAddFriend asks for a name and sends the request; closing or going back
// shows the friends tab again.
func (pf *Platform) askAddFriend(p *player.Player) {
	lang := i18n.Of(p)
	ui.Input(p, i18n.T(lang, "platform.friends.add"), i18n.T(lang, "platform.friends.add.label"), "", func(p *player.Player, _ *world.Tx, name string) {
		pf.addFriend(p, strings.TrimSpace(name))
		pf.showFriendsAt(p, friendsTab, "")
	}, func(p *player.Player, _ *world.Tx) { pf.showFriendsAt(p, friendsTab, "") })
}

// addFriend sends a friend request by name. What the screen shows next does
// not tell whether it was sent, so the result is a toast.
func (pf *Platform) addFriend(p *player.Player, name string) {
	lang := i18n.Of(p)
	target, err := pf.db.AccountByName(name)
	if errors.Is(err, db.ErrNotFound) {
		ui.Notify(p, i18n.T(lang, "platform.friends.add.unknown"), name)
		return
	} else if err != nil {
		pf.fail(p, i18n.T(lang, "platform.friends.add"), err)
		return
	}
	accepted, err := pf.db.SendFriendRequest(instance.AccountID(p), target.ID)
	switch {
	case errors.Is(err, db.ErrSelf):
		ui.Notify(p, "%s", i18n.T(lang, "platform.friends.add.self"))
	case errors.Is(err, db.ErrAlreadyFriends):
		ui.Notify(p, i18n.T(lang, "platform.friends.add.already"), target.Name)
	case errors.Is(err, db.ErrBlocked):
		ui.Notify(p, "%s", i18n.T(lang, "platform.friends.add.blocked"))
	case err != nil:
		pf.fail(p, i18n.T(lang, "platform.friends.add"), err)
	case accepted:
		// they had asked p already: friends now
		pf.tell(target.ID, fmt.Sprintf(i18n.T(lang, "platform.friends.accepted"), p.Name()))
	default:
		ui.Flash(p, i18n.T(lang, "platform.friends.add.sent"), target.Name)
		pf.tell(target.ID, fmt.Sprintf(i18n.T(lang, "platform.friends.add.received"), p.Name()))
	}
}

// confirm asks before f; the other button runs back, so cancelling returns
// to the screen instead of closing it.
func (pf *Platform) confirm(p *player.Player, title, body, yes string, f, back func(p *player.Player, tx *world.Tx)) {
	ui.Menu(p, title, body,
		ui.Option{Label: "§c" + yes, Icon: "textures/ui/confirm", Run: f},
		ui.Option{Label: i18n.T(i18n.Of(p), "platform.confirm.cancel"), Icon: "textures/ui/cancel", Run: back},
	)
}

// notifyRequests tells p about pending friend requests on join.
func (pf *Platform) notifyRequests(p *player.Player) {
	if reqs, err := pf.db.FriendRequests(instance.AccountID(p)); err == nil && len(reqs) > 0 {
		ui.Tell(p, "", fmt.Sprintf(i18n.T(i18n.Of(p), "platform.friends.notifyRequests.1"), len(reqs)))
	}
}
