package platform

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/platform/db"
	"github.com/presentkim/skygear/platform/instance"
)

// The tabs of the friends screen. The friend requests are notices
// (notify.go), whose screen the tab between them opens with the tabs kept.
const (
	friendsTab = iota
	noticesTab
	blockedTab
)

// showFriends is the friends screen: tabs for friends, the notices and
// blocked players. Inside a save the host invites a friend from here, also
// one who is offline (the invite waits in their notices).
func (pf *Platform) showFriends(p *player.Player) { pf.showFriendsAt(p, friendsTab, "") }

// showFriendsAt shows the tab with the row sel chosen (an account ID; the
// first one when empty).
func (pf *Platform) showFriendsAt(p *player.Player, tab int, sel string) {
	lang := i18n.Of(p)
	var sp ui.ListSplit
	var err error
	switch tab {
	case blockedTab:
		sp, err = pf.blockedList(p, sel)
	default:
		sp, err = pf.friendList(p, sel)
	}
	if err != nil {
		pf.fail(p, i18n.T(lang, "platform.friends.load"), err)
		return
	}
	sp.Actions = append(sp.Actions, pf.friendTabs(p, tab)...)
	sp.Actions = append(sp.Actions, ui.Back(i18n.T(lang, "platform.back"), func(p *player.Player, _ *world.Tx) { pf.menuBack(p, instance.ScreenFriends) }))
	ui.ListSplitMenu(p, i18n.T(lang, "platform.friends.title"), sp)
}

// friendTabs are the tabs of the friends screen with on chosen; the notices
// tab shows how many there are.
func (pf *Platform) friendTabs(p *player.Player, on int) []ui.Option {
	lang := i18n.Of(p)
	notices := i18n.T(lang, "platform.notify.title")
	if n := len(pf.notifies(p)); n > 0 {
		notices = fmt.Sprintf("%s §e%d", notices, n)
	}
	tabTo := func(t int) func(p *player.Player, _ *world.Tx) {
		return func(p *player.Player, _ *world.Tx) { pf.showFriendsAt(p, t, "") }
	}
	return []ui.Option{
		ui.Tab(i18n.T(lang, "platform.friends.tab.friends"), iconFriend, on == friendsTab, tabTo(friendsTab)),
		ui.Tab(notices, iconRequests, on == noticesTab, func(p *player.Player, _ *world.Tx) { pf.openNotify(p, true) }),
		ui.Tab(i18n.T(lang, "platform.friends.tab.blocked"), iconBlocked, on == blockedTab, tabTo(blockedTab)),
	}
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

// friendList is the friends tab: the row that adds a friend (pressing it
// asks for the name at once: no button on the right does the same), then the
// friends, close friends first and online ones before offline ones.
func (pf *Platform) friendList(p *player.Player, sel string) (ui.ListSplit, error) {
	lang := i18n.Of(p)
	me := instance.AccountID(p)
	friends, err := pf.db.Friends(me)
	if err != nil {
		return ui.ListSplit{}, err
	}
	closeOf, err := pf.db.CloseFriends(me)
	if err != nil {
		return ui.ListSplit{}, err
	}
	isOnline := map[string]bool{}
	for _, f := range friends {
		isOnline[f.ID] = pf.isOnline(f.ID)
	}
	rank := func(f db.Account) int {
		r := 0
		if !closeOf[f.ID] {
			r += 2
		}
		if !isOnline[f.ID] {
			r++
		}
		return r
	}
	sort.SliceStable(friends, func(i, j int) bool { return rank(friends[i]) < rank(friends[j]) })
	host := pf.hostedSave(p)
	if sel == "" && len(friends) > 0 {
		sel = friends[0].ID
	}
	again := func(id string) func(p *player.Player, _ *world.Tx) {
		return func(p *player.Player, _ *world.Tx) { pf.showFriendsAt(p, friendsTab, id) }
	}
	sp := ui.ListSplit{Rows: []ui.ListRow{{
		Label: ui.ListRowText(i18n.T(lang, "platform.friends.add"), ""), Icon: iconAdd,
		Run: func(p *player.Player, _ *world.Tx) { pf.askAddFriend(p) },
	}}}
	var chosen db.Account
	for _, f := range friends {
		if f.ID == sel {
			chosen = f
		}
		name := f.Name
		if closeOf[f.ID] {
			name = i18n.T(lang, "platform.friends.close.mark") + name
		}
		sp.Rows = append(sp.Rows, ui.ListRow{
			Label: ui.ListRowText(name, pf.statusOf(lang, f.ID, isOnline[f.ID])), Icon: iconFriend,
			Selected: f.ID == sel, Run: again(f.ID),
		})
	}
	if chosen.ID == "" {
		sp.Body = i18n.T(lang, "platform.friends.none")
		return sp, nil
	}
	if closeOf[chosen.ID] {
		sp.Body = fmt.Sprintf(i18n.T(lang, "platform.friends.close.body"), chosen.Name)
	}
	sp.Actions = pf.friendActions(p, chosen, host, closeOf[chosen.ID])
	return sp, nil
}

// friendActions are what can be done with a friend: invite them to the save
// p hosts (when they are not in it; one who is offline finds the invite
// later), enter the save they play in when they made p their close friend
// (in the lobby), make them a close friend or not, remove and block.
func (pf *Platform) friendActions(p *player.Player, f db.Account, host *instance.Instance, isClose bool) []ui.Option {
	lang := i18n.Of(p)
	me := instance.AccountID(p)
	back := func(p *player.Player, _ *world.Tx) { pf.showFriendsAt(p, friendsTab, f.ID) }
	var opts []ui.Option
	if host != nil && pf.m.Location(f.ID) != host {
		opts = append(opts, ui.Option{Label: i18n.T(lang, "platform.friends.invite"), Icon: iconInvite, Run: func(p *player.Player, _ *world.Tx) {
			pf.sendInvite(p, host, f)
			pf.showFriendsAt(p, friendsTab, f.ID)
		}})
	}
	if pf.m.Location(me) == nil && pf.openCloseFriendSave(p, f.ID) != nil {
		opts = append(opts, ui.Option{Label: i18n.T(lang, "platform.friends.close.join"), Icon: iconEnter, Run: func(p *player.Player, _ *world.Tx) {
			pf.joinCloseFriend(p, f.ID)
		}})
	}
	closeLabel, closeIcon := "platform.friends.close.set", iconAccept
	if isClose {
		closeLabel, closeIcon = "platform.friends.close.unset", iconDecline
	}
	opts = append(opts,
		ui.Option{Label: i18n.T(lang, closeLabel), Icon: closeIcon, Run: func(p *player.Player, _ *world.Tx) {
			pf.setCloseFriend(p, f, !isClose)
			pf.showFriendsAt(p, friendsTab, f.ID)
		}},
		ui.Option{Label: i18n.T(lang, "platform.friends.remove"), Icon: iconDelete, Run: func(p *player.Player, _ *world.Tx) {
			pf.confirm(p, i18n.T(lang, "platform.friends.remove"), fmt.Sprintf(i18n.T(lang, "platform.friends.remove.ask"), f.Name), i18n.T(lang, "platform.friends.remove"), func(p *player.Player, _ *world.Tx) {
				if err := pf.db.RemoveFriend(instance.AccountID(p), f.ID); err != nil {
					pf.fail(p, i18n.T(lang, "platform.friends.remove"), err)
					return
				}
				pf.showFriendsAt(p, friendsTab, "")
			}, back)
		}},
		pf.blockOption(lang, f, func(p *player.Player, _ *world.Tx) { pf.showFriendsAt(p, friendsTab, "") },
			func(p *player.Player, _ *world.Tx) { pf.showFriendsAt(p, friendsTab, f.ID) }),
	)
	return opts
}

// setCloseFriend makes f p's close friend, or not. A close friend may enter
// p's save while p plays in it, without an invite, and becomes a member
// then; f is told.
func (pf *Platform) setCloseFriend(p *player.Player, f db.Account, on bool) {
	lang := i18n.Of(p)
	if err := pf.db.SetCloseFriend(instance.AccountID(p), f.ID, on); err != nil {
		pf.fail(p, i18n.T(lang, "platform.friends.close.set"), err)
		return
	}
	if on {
		ui.Flash(p, i18n.T(lang, "platform.friends.close.done"), f.Name)
		pf.notified(f.ID, "platform.friends.close.received", p.Name())
		return
	}
	ui.Flash(p, i18n.T(lang, "platform.friends.close.undone"), f.Name)
}

// blockOption blocks a after asking; done runs after blocking, back on
// cancel.
func (pf *Platform) blockOption(lang string, a db.Account, done, back func(p *player.Player, tx *world.Tx)) ui.Option {
	return ui.Option{Label: i18n.T(lang, "platform.friends.block"), Icon: iconBlocked, Run: func(p *player.Player, _ *world.Tx) {
		pf.confirm(p, i18n.T(lang, "platform.friends.block"), fmt.Sprintf(i18n.T(lang, "platform.friends.block.ask"), a.Name), i18n.T(lang, "platform.friends.block"), func(p *player.Player, tx *world.Tx) {
			if err := pf.db.Block(instance.AccountID(p), a.ID); err != nil {
				pf.fail(p, i18n.T(lang, "platform.friends.block"), err)
				return
			}
			done(p, tx)
		}, back)
	}}
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
		pf.tell(target.ID, "platform.friends.accepted", p.Name())
	default:
		ui.Flash(p, i18n.T(lang, "platform.friends.add.sent"), target.Name)
		pf.notified(target.ID, "platform.friends.add.received", p.Name())
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

// notifyPending tells p on join how many notices wait.
func (pf *Platform) notifyPending(p *player.Player) {
	if n := len(pf.notifies(p)); n > 0 {
		ui.Tell(p, i18n.T(i18n.Of(p), "platform.notify.title"), fmt.Sprintf(i18n.T(i18n.Of(p), "platform.notify.pending"), n))
	}
}
