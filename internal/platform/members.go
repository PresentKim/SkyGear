package platform

import (
	"fmt"

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
		pf.fail(p, "참여자 불러오기", err)
		return
	}
	var opts []ui.Option
	for _, m := range members {
		state := pf.status(m.AccountID)
		switch {
		case m.Role == db.RoleHost:
			state = "§6호스트"
		case m.Left:
			state = "§8탈퇴·추방됨"
		}
		opts = append(opts, ui.Option{Label: fmt.Sprintf("%s\n%s", m.Name, state), Icon: iconPlayer, Run: func(p *player.Player, _ *world.Tx) {
			pf.showMember(p, saveID, m)
		}})
	}
	n, _ := pf.db.ActiveMembers(saveID)
	ui.Menu(p, "참여자 관리", fmt.Sprintf("참여자 %d/%d명\n탈퇴·추방된 플레이어의 인벤토리와 돈도 세이브에 남아 있습니다.", n, MaxMembers), opts...)
}

func (pf *Platform) showMember(p *player.Player, saveID int64, m db.Member) {
	opts := []ui.Option{{Label: "인벤토리 보기", Icon: iconInvSee, Run: func(p *player.Player, tx *world.Tx) { pf.showInvSee(p, tx, saveID, m) }}}
	if m.Role != db.RoleHost && !m.Left {
		opts = append(opts, ui.Option{Label: "§c추방", Icon: iconKick, Run: func(p *player.Player, _ *world.Tx) {
			ui.Confirm(p, "추방", m.Name+" 님을 이 세이브에서 추방할까요?\n인벤토리와 돈은 세이브에 남습니다.", "§c추방", func(p *player.Player, _ *world.Tx) {
				pf.kick(p, saveID, m)
			})
		}})
	}
	opts = append(opts, ui.Option{Label: "뒤로", Icon: iconBack, Run: func(p *player.Player, _ *world.Tx) { pf.showMembers(p, saveID) }})
	ui.Menu(p, m.Name, fmt.Sprintf("%s\n%s", m.Name, pf.status(m.AccountID)), opts...)
}

func (pf *Platform) kick(p *player.Player, saveID int64, m db.Member) {
	if err := pf.db.LeaveMember(saveID, m.AccountID); err != nil {
		pf.fail(p, "추방", err)
		return
	}
	p.Messagef("§e%s 님을 추방했습니다.", m.Name)
	pf.invites.remove(m.AccountID, saveID)
	pf.with(m.AccountID, func(tp *player.Player) {
		if inst := pf.m.Location(m.AccountID); inst != nil && inst.Save.ID == saveID {
			tp.Message("§c세이브에서 추방되어 로비로 이동합니다.")
			pf.m.ToLobby(tp)
			return
		}
		tp.Message("§c참여 중이던 세이브에서 추방되었습니다.")
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
		o.Error("자신의 세이브 안에서 호스트만 사용할 수 있습니다.")
		return
	}
	c.pf.showMembers(p, inst.Save.ID)
}
