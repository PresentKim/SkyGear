package platform

import (
	"errors"
	"fmt"
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
		pf.fail(p, "친구 목록 불러오기", err)
		return
	}
	reqs, _ := pf.db.FriendRequests(acc)
	opts := []ui.Option{
		{Label: "친구 추가", Icon: iconAdd, Run: func(p *player.Player, _ *world.Tx) { pf.askAddFriend(p) }},
		{Label: fmt.Sprintf("받은 친구 요청 §8(%d)", len(reqs)), Icon: iconRequests, Run: func(p *player.Player, _ *world.Tx) { pf.showRequests(p) }},
	}
	var b strings.Builder
	if len(friends) == 0 {
		b.WriteString("§7아직 친구가 없습니다. 친구 추가로 요청을 보내 보세요.")
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
	opts = append(opts, ui.Option{Label: "차단 목록", Icon: iconBlocked, Run: func(p *player.Player, _ *world.Tx) { pf.showBlocked(p) }})
	ui.Menu(p, "친구", b.String(), opts...)
}

func (pf *Platform) showFriend(p *player.Player, f db.Account) {
	ui.Menu(p, f.Name, fmt.Sprintf("%s\n%s", f.Name, pf.status(f.ID)),
		ui.Option{Label: "친구 삭제", Icon: iconDelete, Run: func(p *player.Player, _ *world.Tx) {
			ui.Confirm(p, "친구 삭제", f.Name+" 님을 친구에서 삭제할까요?", "삭제", func(p *player.Player, _ *world.Tx) {
				if err := pf.db.RemoveFriend(instance.AccountID(p), f.ID); err != nil {
					pf.fail(p, "친구 삭제", err)
					return
				}
				ui.Flash(p, "§e%s 님을 친구에서 삭제했습니다.", f.Name)
			})
		}},
		ui.Option{Label: "§c차단", Icon: iconBlocked, Run: func(p *player.Player, _ *world.Tx) { pf.askBlock(p, f) }},
		ui.Back("뒤로", func(p *player.Player, _ *world.Tx) { pf.showFriends(p) }),
	)
}

func (pf *Platform) askAddFriend(p *player.Player) {
	ui.Input(p, "친구 추가", "플레이어 이름", "", func(p *player.Player, _ *world.Tx, name string) {
		pf.addFriend(p, strings.TrimSpace(name))
	})
}

// addFriend sends a friend request by name.
func (pf *Platform) addFriend(p *player.Player, name string) {
	target, err := pf.db.AccountByName(name)
	if errors.Is(err, db.ErrNotFound) {
		ui.Notify(p, "§c'%s' 님은 이 서버에 접속한 적이 없습니다.", name)
		return
	} else if err != nil {
		pf.fail(p, "친구 요청", err)
		return
	}
	accepted, err := pf.db.SendFriendRequest(instance.AccountID(p), target.ID)
	switch {
	case errors.Is(err, db.ErrSelf):
		ui.Notify(p, "§c자기 자신에게는 요청할 수 없습니다.")
	case errors.Is(err, db.ErrAlreadyFriends):
		ui.Flash(p, "§e%s 님과 이미 친구입니다.", target.Name)
	case errors.Is(err, db.ErrBlocked):
		ui.Notify(p, "§c이 플레이어에게 친구 요청을 보낼 수 없습니다.")
	case err != nil:
		pf.fail(p, "친구 요청", err)
	case accepted:
		ui.Flash(p, "§a%s 님과 친구가 되었습니다.", target.Name)
		pf.tell(target.ID, fmt.Sprintf("§a%s 님과 친구가 되었습니다.", p.Name()))
	default:
		ui.Flash(p, "§a%s 님에게 친구 요청을 보냈습니다.", target.Name)
		pf.tell(target.ID, fmt.Sprintf("§e%s 님이 친구 요청을 보냈습니다. §7/friends", p.Name()))
	}
}

func (pf *Platform) showRequests(p *player.Player) {
	reqs, err := pf.db.FriendRequests(instance.AccountID(p))
	if err != nil {
		pf.fail(p, "친구 요청 불러오기", err)
		return
	}
	var opts []ui.Option
	for _, r := range reqs {
		opts = append(opts, ui.Option{Label: r.Name, Icon: iconPlayer, Run: func(p *player.Player, _ *world.Tx) {
			ui.Menu(p, "친구 요청", r.Name+" 님의 친구 요청",
				ui.Option{Label: "§a수락", Icon: iconAccept, Run: func(p *player.Player, _ *world.Tx) {
					if err := pf.db.AcceptFriend(instance.AccountID(p), r.ID); err != nil {
						pf.fail(p, "친구 수락", err)
						return
					}
					ui.Flash(p, "§a%s 님과 친구가 되었습니다.", r.Name)
					pf.tell(r.ID, fmt.Sprintf("§a%s 님이 친구 요청을 수락했습니다.", p.Name()))
				}},
				ui.Option{Label: "거절", Icon: iconDecline, Run: func(p *player.Player, _ *world.Tx) {
					_ = pf.db.DeclineFriend(instance.AccountID(p), r.ID)
					pf.showRequests(p)
				}},
				ui.Option{Label: "§c차단", Icon: iconBlocked, Run: func(p *player.Player, _ *world.Tx) { pf.askBlock(p, r) }},
			)
		}})
	}
	body := "받은 친구 요청입니다."
	if len(reqs) == 0 {
		body = "§7받은 친구 요청이 없습니다."
	}
	ui.Menu(p, "받은 친구 요청", body, opts...)
}

func (pf *Platform) askBlock(p *player.Player, target db.Account) {
	ui.Confirm(p, "차단", target.Name+" 님을 차단할까요?\n친구 관계와 요청이 사라지고, 이 플레이어의 요청·초대를 받지 않습니다.", "§c차단", func(p *player.Player, _ *world.Tx) {
		if err := pf.db.Block(instance.AccountID(p), target.ID); err != nil {
			pf.fail(p, "차단", err)
			return
		}
		ui.Flash(p, "§e%s 님을 차단했습니다.", target.Name)
	})
}

func (pf *Platform) showBlocked(p *player.Player) {
	list, err := pf.db.BlockedAccounts(instance.AccountID(p))
	if err != nil {
		pf.fail(p, "차단 목록 불러오기", err)
		return
	}
	var opts []ui.Option
	for _, a := range list {
		opts = append(opts, ui.Option{Label: a.Name + "\n§8눌러서 차단 해제", Icon: iconPlayer, Run: func(p *player.Player, _ *world.Tx) {
			_ = pf.db.Unblock(instance.AccountID(p), a.ID)
			ui.Flash(p, "§e%s 님의 차단을 해제했습니다.", a.Name)
		}})
	}
	body := "차단한 플레이어"
	if len(list) == 0 {
		body = "§7차단한 플레이어가 없습니다."
	}
	ui.Menu(p, "차단 목록", body, opts...)
}

// notifyRequests tells p about pending friend requests on join.
func (pf *Platform) notifyRequests(p *player.Player) {
	if reqs, err := pf.db.FriendRequests(instance.AccountID(p)); err == nil && len(reqs) > 0 {
		p.Messagef("§e받은 친구 요청이 %d개 있습니다. §7/friends", len(reqs))
	}
}
