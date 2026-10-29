package platform

import (
	"fmt"
	"sort"
	"time"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/internal/instance"
)

// A player's notices are everything waiting for them to answer or read: the
// friend requests and save invites they got, and later announcements and the
// like. Each kind is a noticeSource that lists its notices when the notices
// are shown; the notice screen, the main screen's bell (the count) and its
// feed (the newest lines) read them all through notices, so a new kind is
// one more source in noticeSources.

// Notice is one notice of a player.
type Notice struct {
	// Key tells the notice from the others of the player ("<kind>:<id>"); the
	// notice screen keeps it chosen.
	Key string
	// Icon, Name and Info are its row: the picture, the name and the small
	// line under it.
	Icon, Name, Info string
	// Body is the text beside the actions.
	Body string
	// Feed is its line in the main screen's feed ("": none).
	Feed string
	// At is when it came: the newest come first.
	At time.Time
	// Actions are its buttons. A run that shows no screen of its own shows
	// the notices again (showNotices).
	Actions []ui.Option
}

// noticeSource lists the notices of one kind for an account, in lang.
type noticeSource func(pf *Platform, accountID, lang string) []Notice

// noticeSources are the kinds of notices, in no order (notices sorts them).
func noticeSources() []noticeSource { return []noticeSource{friendRequestNotices, inviteNotices} }

// notices are p's notices, newest first.
func (pf *Platform) notices(p *player.Player) []Notice {
	return pf.noticesOf(instance.AccountID(p), i18n.Of(p))
}

// noticesOf are the notices of an account in lang, newest first.
func (pf *Platform) noticesOf(accountID, lang string) []Notice {
	var out []Notice
	for _, src := range noticeSources() {
		out = append(out, src(pf, accountID, lang)...)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].At.After(out[j].At) })
	return out
}

// showNotices is the notice screen: the notices on the left, newest first,
// and the chosen one (sel, a Key; the newest when not found) with its
// actions on the right.
func (pf *Platform) showNotices(p *player.Player, sel string) {
	lang := i18n.Of(p)
	list := pf.notices(p)
	var sp ui.ListSplit
	chosen := -1
	for i, n := range list {
		if n.Key == sel {
			chosen = i
		}
	}
	if chosen < 0 && len(list) > 0 {
		chosen = 0
	}
	for i, n := range list {
		sp.Rows = append(sp.Rows, ui.ListRow{
			Label: ui.ListRowText(n.Name, n.Info), Icon: n.Icon, Selected: i == chosen,
			Run: func(p *player.Player, _ *world.Tx) { pf.showNotices(p, n.Key) },
		})
	}
	if chosen < 0 {
		sp.Body = i18n.T(lang, "platform.notice.none")
	} else {
		sp.Body = list[chosen].Body
		sp.Actions = list[chosen].Actions
	}
	sp.Actions = append(sp.Actions, ui.Back(i18n.T(lang, "platform.back"), func(p *player.Player, _ *world.Tx) { pf.menuBack(p, instance.ScreenNotices) }))
	ui.ListSplitMenu(p, i18n.T(lang, "platform.notice.title"), sp)
}

// noticed tells p, online, about a new notice: a toast with line, and the
// main screen drawn again for its count when p is looking at it.
func (pf *Platform) noticed(accountID, key string, args ...any) {
	pf.with(accountID, func(p *player.Player) {
		ui.Tell(p, i18n.T(i18n.Of(p), "platform.notice.title"), fmt.Sprintf(i18n.T(i18n.Of(p), key), args...))
		pf.refreshHome(p)
	})
}

// noticeAgo is the "when" of a notice's row in lang.
func noticeAgo(lang string, at time.Time) string {
	return i18n.ClientAgo(at, at.Format("2006-01-02 15:04"))
}

// friendRequestNotices are the friend requests p got: accept, decline or
// block the one who sent it.
func friendRequestNotices(pf *Platform, accountID, lang string) []Notice {
	reqs, err := pf.db.FriendRequestsTo(accountID)
	if err != nil {
		pf.log.Error("load friend requests", "account", accountID, "err", err)
		return nil
	}
	again := func(p *player.Player, _ *world.Tx) { pf.showNotices(p, "") }
	var out []Notice
	for _, r := range reqs {
		from := r.From
		out = append(out, Notice{
			Key:  "friend:" + from.ID,
			Icon: iconAdd,
			Name: from.Name,
			Info: fmt.Sprintf(i18n.T(lang, "platform.notice.friend.info"), noticeAgo(lang, r.At)),
			Body: fmt.Sprintf(i18n.T(lang, "platform.notice.friend.body"), from.Name, pf.status(lang, from.ID)),
			Feed: fmt.Sprintf(i18n.T(lang, "platform.notice.friend.feed"), from.Name),
			At:   r.At,
			Actions: []ui.Option{
				{Label: i18n.T(lang, "platform.friends.accept"), Icon: iconAccept, Run: func(p *player.Player, tx *world.Tx) {
					if err := pf.db.AcceptFriend(instance.AccountID(p), from.ID); err != nil {
						pf.fail(p, i18n.T(lang, "platform.friends.accept"), err)
						return
					}
					pf.tell(from.ID, "platform.friends.accepted", p.Name())
					ui.Flash(p, i18n.T(lang, "platform.friends.accepted"), from.Name)
					again(p, tx)
				}},
				{Label: i18n.T(lang, "platform.friends.decline"), Icon: iconDecline, Run: func(p *player.Player, tx *world.Tx) {
					_ = pf.db.DeclineFriend(instance.AccountID(p), from.ID)
					again(p, tx)
				}},
				pf.blockOption(lang, from, again, func(p *player.Player, _ *world.Tx) { pf.showNotices(p, "friend:"+from.ID) }),
			},
		})
	}
	return out
}

// inviteNotices are the save invites p got: join the save, or decline.
func inviteNotices(pf *Platform, accountID, lang string) []Notice {
	var out []Notice
	for _, in := range pf.invites.list(accountID) {
		left := max(1, int((inviteTTL-time.Since(in.at)).Minutes()+0.5))
		out = append(out, Notice{
			Key:  fmt.Sprintf("invite:%d", in.saveID),
			Icon: iconInvite,
			Name: in.saveName,
			Info: fmt.Sprintf(i18n.T(lang, "platform.invite.row"), in.from, left),
			Body: fmt.Sprintf(i18n.T(lang, "platform.invite.body"), in.from, in.saveName),
			Feed: fmt.Sprintf(i18n.T(lang, "platform.notice.invite.feed"), in.from, in.saveName),
			At:   in.at,
			Actions: []ui.Option{
				{Label: i18n.T(lang, "platform.invite.accept"), Icon: iconEnter, Run: func(p *player.Player, _ *world.Tx) {
					if pf.m.Location(instance.AccountID(p)) != nil {
						ui.Notify(p, "%s", i18n.T(lang, "platform.lobbyOnly"))
						pf.showNotices(p, "")
						return
					}
					pf.join(p, in.saveID)
				}},
				{Label: i18n.T(lang, "platform.invite.decline"), Icon: iconDecline, Run: func(p *player.Player, _ *world.Tx) {
					pf.invites.remove(instance.AccountID(p), in.saveID)
					pf.showNotices(p, "")
				}},
			},
		})
	}
	return out
}

// noticesCmd opens the notice screen: /notices.
type noticesCmd struct {
	playerOnly
	pf *Platform
}

func (c noticesCmd) Run(src cmd.Source, _ *cmd.Output, _ *world.Tx) {
	c.pf.showNotices(src.(*player.Player), "")
}
