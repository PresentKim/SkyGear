package platform

import (
	"fmt"
	"slices"
	"strings"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/account"
	"github.com/presentkim/dragonfactory/internal/instance"
	"github.com/presentkim/dragonfactory/internal/ui"
)

// Titles, name colours and achievements (docs/design/01-platform.md §5),
// from the lobby menu.

func (pf *Platform) showCosmetics(p *player.Player) {
	acc, err := pf.db.Account(instance.AccountID(p))
	if err != nil {
		pf.fail(p, "계정 불러오기", err)
		return
	}
	owned := account.Owned(acc.ID)
	title := "없음"
	if a, ok := account.Get(acc.Title); ok {
		title = a.Title
	}
	body := fmt.Sprintf("§f%s\n§7업적 %d/%d · 칭호 §e%s§7 · 이름 색 %s%s",
		account.DisplayName(p), len(owned), len(account.All()), title, account.ColorOf(acc.NameColor).Code, account.ColorOf(acc.NameColor).Name)
	ui.CardMenu(p, "칭호·치장", body,
		ui.Option{Label: "칭호 고르기", Icon: iconCode, Run: func(p *player.Player, _ *world.Tx) { pf.showTitles(p) }},
		ui.Option{Label: "이름 색", Icon: iconRename, Run: func(p *player.Player, _ *world.Tx) { pf.showColors(p) }},
		ui.Option{Label: "업적 목록", Icon: iconGuide, Run: func(p *player.Player, _ *world.Tx) { pf.showAchievements(p) }},
		ui.Option{Label: "뒤로", Icon: iconBack, Run: func(p *player.Player, _ *world.Tx) { pf.showLobbyMenu(p) }},
	)
}

func (pf *Platform) showTitles(p *player.Player) {
	acc := instance.AccountID(p)
	opts := []ui.Option{{Label: "§7칭호 없음", Icon: iconDelete, Run: func(p *player.Player, _ *world.Tx) { pf.setTitle(p, "") }}}
	for _, id := range account.Owned(acc) {
		if a, ok := account.Get(id); ok && a.Title != "" {
			opts = append(opts, ui.Option{Label: "§e[" + a.Title + "]\n§8" + a.Name, Icon: iconCode, Run: func(p *player.Player, _ *world.Tx) { pf.setTitle(p, a.ID) }})
		}
	}
	opts = append(opts, ui.Option{Label: "뒤로", Icon: iconBack, Run: func(p *player.Player, _ *world.Tx) { pf.showCosmetics(p) }})
	body := "업적을 달성하면 칭호를 얻습니다. 장착한 칭호는 채팅과 이름표 앞에 보입니다."
	if len(opts) == 2 {
		body += "\n\n§7아직 얻은 칭호가 없습니다."
	}
	ui.Menu(p, "칭호 고르기", body, opts...)
}

func (pf *Platform) setTitle(p *player.Player, id string) {
	acc := instance.AccountID(p)
	if id != "" && !slices.Contains(account.Owned(acc), id) {
		return
	}
	if err := pf.db.SetTitle(acc, id); err != nil {
		pf.fail(p, "칭호 장착", err)
		return
	}
	account.Refresh(p)
	p.Messagef("§a칭호를 바꿨습니다: %s", account.DisplayName(p))
	pf.showCosmetics(p)
}

func (pf *Platform) showColors(p *player.Player) {
	n := len(account.Owned(instance.AccountID(p)))
	var opts []ui.Option
	for _, c := range account.Colors {
		label := c.Code + c.Name
		if c.Need > n {
			label = fmt.Sprintf("§8%s\n§8업적 %d개 필요", c.Name, c.Need)
		}
		opts = append(opts, ui.Option{Label: label, Icon: iconRename, Run: func(p *player.Player, _ *world.Tx) {
			if c.Need > len(account.Owned(instance.AccountID(p))) {
				p.Messagef("§c업적을 %d개 달성하면 열립니다.", c.Need)
				pf.showColors(p)
				return
			}
			if err := pf.db.SetNameColor(instance.AccountID(p), c.ID); err != nil {
				pf.fail(p, "이름 색 변경", err)
				return
			}
			account.Refresh(p)
			pf.showCosmetics(p)
		}})
	}
	opts = append(opts, ui.Option{Label: "뒤로", Icon: iconBack, Run: func(p *player.Player, _ *world.Tx) { pf.showCosmetics(p) }})
	ui.CardMenu(p, "이름 색", fmt.Sprintf("§7업적 %d개 달성. 업적이 늘면 색이 더 열립니다.", n), opts...)
}

func (pf *Platform) showAchievements(p *player.Player) {
	owned := account.Owned(instance.AccountID(p))
	var b strings.Builder
	for _, a := range account.All() {
		mark := "§8□ §7"
		if slices.Contains(owned, a.ID) {
			mark = "§a■ §f"
		}
		fmt.Fprintf(&b, "%s%s §8- %s", mark, a.Name, a.Desc)
		if a.Title != "" {
			fmt.Fprintf(&b, " §7(칭호 [%s])", a.Title)
		}
		b.WriteString("\n")
	}
	ui.Menu(p, "업적 목록", b.String(), ui.Option{Label: "뒤로", Icon: iconBack, Run: func(p *player.Player, _ *world.Tx) { pf.showCosmetics(p) }})
}
