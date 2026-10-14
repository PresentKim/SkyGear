package platform

import (
	"errors"
	"fmt"
	"github.com/presentkim/dragonfactory/internal/i18n"
	"strings"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/db"
	"github.com/presentkim/dragonfactory/internal/instance"
	"github.com/presentkim/dragonfactory/internal/ui"
)

// showFriends lists friends with their status (docs/design/01-platform.md §4).
func (pf *Platform) showFriends(p *player.Player) {
	acc := instance.AccountID(p)
	friends, err := pf.db.Friends(acc)
	if err != nil {
		pf.fail(p, i18n.T(i18n.Of(p), "platform.friends.showFriends.1"), err)
		return
	}
	reqs, _ := pf.db.FriendRequests(acc)
	opts := []ui.Option{
		{Label: i18n.T(i18n.Of(p), "platform.friends.showFriends.2"), Icon: iconAdd, Run: func(p *player.Player, _ *world.Tx) { pf.askAddFriend(p) }},
		{Label: fmt.Sprintf(i18n.T(i18n.Of(p), "platform.friends.showFriends.3"), len(reqs)), Icon: iconRequests, Run: func(p *player.Player, _ *world.Tx) { pf.showRequests(p) }},
	}
	var b strings.Builder
	if len(friends) == 0 {
		b.WriteString(i18n.T(i18n.Of(p), "platform.friends.showFriends.4"))
	}
	for _, f := range friends {
		st := pf.status(f.ID)
		fmt.Fprintf(&b, "§f%s %s\n", f.Name, st)
		opts = append(opts, ui.Option{
			Label: fmt.Sprintf("%s\n%s", f.Name, st),
			Icon:  iconPlayer,
			Run:   func(p *player.Player, _ *world.Tx) { pf.showFriend(p, f) },
		})
	}
	opts = append(opts, ui.Option{Label: i18n.T(i18n.Of(p), "platform.friends.showFriends.5"), Icon: iconBlocked, Run: func(p *player.Player, _ *world.Tx) { pf.showBlocked(p) }})
	ui.Menu(p, i18n.T(i18n.Of(p), "platform.friends.showFriends.6"), b.String(), opts...)
}

func (pf *Platform) showFriend(p *player.Player, f db.Account) {
	ui.Menu(p, f.Name, fmt.Sprintf("%s\n%s", f.Name, pf.status(f.ID)),
		ui.Option{Label: i18n.T(i18n.Of(p), "platform.friends.showFriend.1"), Icon: iconDelete, Run: func(p *player.Player, _ *world.Tx) {
			ui.Confirm(p, i18n.T(i18n.Of(p), "platform.friends.showFriend.2"), f.Name+i18n.T(i18n.Of(p), "platform.friends.showFriend.3"), i18n.T(i18n.Of(p), "platform.friends.showFriend.4"), func(p *player.Player, _ *world.Tx) {
				if err := pf.db.RemoveFriend(instance.AccountID(p), f.ID); err != nil {
					pf.fail(p, i18n.T(i18n.Of(p), "platform.friends.showFriend.5"), err)
					return
				}
				ui.Flash(p, i18n.T(i18n.Of(p), "platform.friends.showFriend.6"), f.Name)
			})
		}},
		ui.Option{Label: i18n.T(i18n.Of(p), "platform.friends.showFriend.7"), Icon: iconBlocked, Run: func(p *player.Player, _ *world.Tx) { pf.askBlock(p, f) }},
		ui.Back(i18n.T(i18n.Of(p), "platform.friends.showFriend.8"), func(p *player.Player, _ *world.Tx) { pf.showFriends(p) }),
	)
}

func (pf *Platform) askAddFriend(p *player.Player) {
	ui.Input(p, i18n.T(i18n.Of(p), "platform.friends.askAddFriend.1"), i18n.T(i18n.Of(p), "platform.friends.askAddFriend.2"), "", func(p *player.Player, _ *world.Tx, name string) {
		pf.addFriend(p, strings.TrimSpace(name))
	})
}

// addFriend sends a friend request by name.
func (pf *Platform) addFriend(p *player.Player, name string) {
	target, err := pf.db.AccountByName(name)
	if errors.Is(err, db.ErrNotFound) {
		ui.Notify(p, i18n.T(i18n.Of(p), "platform.friends.addFriend.1"), name)
		return
	} else if err != nil {
		pf.fail(p, i18n.T(i18n.Of(p), "platform.friends.addFriend.2"), err)
		return
	}
	accepted, err := pf.db.SendFriendRequest(instance.AccountID(p), target.ID)
	switch {
	case errors.Is(err, db.ErrSelf):
		ui.Notify(p, "%s", i18n.T(i18n.Of(p), "platform.friends.addFriend.3"))
	case errors.Is(err, db.ErrAlreadyFriends):
		ui.Flash(p, i18n.T(i18n.Of(p), "platform.friends.addFriend.4"), target.Name)
	case errors.Is(err, db.ErrBlocked):
		ui.Notify(p, "%s", i18n.T(i18n.Of(p), "platform.friends.addFriend.5"))
	case err != nil:
		pf.fail(p, i18n.T(i18n.Of(p), "platform.friends.addFriend.6"), err)
	case accepted:
		ui.Flash(p, i18n.T(i18n.Of(p), "platform.friends.addFriend.7"), target.Name)
		pf.tell(target.ID, fmt.Sprintf(i18n.T(i18n.Of(p), "platform.friends.addFriend.8"), p.Name()))
	default:
		ui.Flash(p, i18n.T(i18n.Of(p), "platform.friends.addFriend.9"), target.Name)
		pf.tell(target.ID, fmt.Sprintf(i18n.T(i18n.Of(p), "platform.friends.addFriend.10"), p.Name()))
	}
}

func (pf *Platform) showRequests(p *player.Player) {
	reqs, err := pf.db.FriendRequests(instance.AccountID(p))
	if err != nil {
		pf.fail(p, i18n.T(i18n.Of(p), "platform.friends.showRequests.1"), err)
		return
	}
	var opts []ui.Option
	for _, r := range reqs {
		opts = append(opts, ui.Option{Label: r.Name, Icon: iconPlayer, Run: func(p *player.Player, _ *world.Tx) {
			ui.Menu(p, i18n.T(i18n.Of(p), "platform.friends.showRequests.2"), r.Name+i18n.T(i18n.Of(p), "platform.friends.showRequests.3"),
				ui.Option{Label: i18n.T(i18n.Of(p), "platform.friends.showRequests.4"), Icon: iconAccept, Run: func(p *player.Player, _ *world.Tx) {
					if err := pf.db.AcceptFriend(instance.AccountID(p), r.ID); err != nil {
						pf.fail(p, i18n.T(i18n.Of(p), "platform.friends.showRequests.5"), err)
						return
					}
					ui.Flash(p, i18n.T(i18n.Of(p), "platform.friends.showRequests.6"), r.Name)
					pf.tell(r.ID, fmt.Sprintf(i18n.T(i18n.Of(p), "platform.friends.showRequests.7"), p.Name()))
				}},
				ui.Option{Label: i18n.T(i18n.Of(p), "platform.friends.showRequests.8"), Icon: iconDecline, Run: func(p *player.Player, _ *world.Tx) {
					_ = pf.db.DeclineFriend(instance.AccountID(p), r.ID)
					pf.showRequests(p)
				}},
				ui.Option{Label: i18n.T(i18n.Of(p), "platform.friends.showRequests.9"), Icon: iconBlocked, Run: func(p *player.Player, _ *world.Tx) { pf.askBlock(p, r) }},
			)
		}})
	}
	body := i18n.T(i18n.Of(p), "platform.friends.showRequests.10")
	if len(reqs) == 0 {
		body = i18n.T(i18n.Of(p), "platform.friends.showRequests.11")
	}
	ui.Menu(p, i18n.T(i18n.Of(p), "platform.friends.showRequests.12"), body, opts...)
}

func (pf *Platform) askBlock(p *player.Player, target db.Account) {
	ui.Confirm(p, i18n.T(i18n.Of(p), "platform.friends.askBlock.1"), target.Name+i18n.T(i18n.Of(p), "platform.friends.askBlock.2"), i18n.T(i18n.Of(p), "platform.friends.askBlock.3"), func(p *player.Player, _ *world.Tx) {
		if err := pf.db.Block(instance.AccountID(p), target.ID); err != nil {
			pf.fail(p, i18n.T(i18n.Of(p), "platform.friends.askBlock.4"), err)
			return
		}
		ui.Flash(p, i18n.T(i18n.Of(p), "platform.friends.askBlock.5"), target.Name)
	})
}

func (pf *Platform) showBlocked(p *player.Player) {
	list, err := pf.db.BlockedAccounts(instance.AccountID(p))
	if err != nil {
		pf.fail(p, i18n.T(i18n.Of(p), "platform.friends.showBlocked.1"), err)
		return
	}
	var opts []ui.Option
	for _, a := range list {
		opts = append(opts, ui.Option{Label: a.Name + i18n.T(i18n.Of(p), "platform.friends.showBlocked.2"), Icon: iconPlayer, Run: func(p *player.Player, _ *world.Tx) {
			_ = pf.db.Unblock(instance.AccountID(p), a.ID)
			ui.Flash(p, i18n.T(i18n.Of(p), "platform.friends.showBlocked.3"), a.Name)
		}})
	}
	body := i18n.T(i18n.Of(p), "platform.friends.showBlocked.4")
	if len(list) == 0 {
		body = i18n.T(i18n.Of(p), "platform.friends.showBlocked.5")
	}
	ui.Menu(p, i18n.T(i18n.Of(p), "platform.friends.showBlocked.6"), body, opts...)
}

// notifyRequests tells p about pending friend requests on join.
func (pf *Platform) notifyRequests(p *player.Player) {
	if reqs, err := pf.db.FriendRequests(instance.AccountID(p)); err == nil && len(reqs) > 0 {
		p.Messagef(i18n.T(i18n.Of(p), "platform.friends.notifyRequests.1"), len(reqs))
	}
}
