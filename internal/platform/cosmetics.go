package platform

import (
	"fmt"
	"github.com/presentkim/dragonfactory/internal/i18n"
	"slices"
	"strings"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/account"
	"github.com/presentkim/dragonfactory/internal/instance"
	"github.com/presentkim/dragonfactory/internal/ui"
)

// Titles, name colours and achievements (docs/design/01-platform.md §5),
// from the lobby menu, /title, or a game's menu (which runs /title).

func (pf *Platform) showCosmetics(p *player.Player) {
	acc, err := pf.db.Account(instance.AccountID(p))
	if err != nil {
		pf.fail(p, i18n.T(i18n.Of(p), "platform.cosmetics.showCosmetics.1"), err)
		return
	}
	owned := account.Owned(acc.ID)
	title := i18n.T(i18n.Of(p), "platform.cosmetics.showCosmetics.2")
	if a, ok := account.Get(acc.Title); ok {
		title = a.Title
	}
	body := fmt.Sprintf(i18n.T(i18n.Of(p), "platform.cosmetics.showCosmetics.3"),
		account.DisplayName(p), len(owned), len(account.All()), title, account.ColorOf(acc.NameColor).Code, account.ColorOf(acc.NameColor).Name)
	ui.CardMenu(p, i18n.T(i18n.Of(p), "platform.cosmetics.showCosmetics.4"), body,
		ui.Option{Label: i18n.T(i18n.Of(p), "platform.cosmetics.showCosmetics.5"), Icon: iconCode, Run: func(p *player.Player, _ *world.Tx) { pf.showTitles(p) }},
		ui.Option{Label: i18n.T(i18n.Of(p), "platform.cosmetics.showCosmetics.6"), Icon: iconRename, Run: func(p *player.Player, _ *world.Tx) { pf.showColors(p) }},
		ui.Option{Label: i18n.T(i18n.Of(p), "platform.cosmetics.showCosmetics.7"), Icon: iconGuide, Run: func(p *player.Player, _ *world.Tx) { pf.showAchievements(p) }},
		ui.Back(i18n.T(i18n.Of(p), "platform.cosmetics.showCosmetics.8"), func(p *player.Player, _ *world.Tx) { pf.cosmeticsBack(p) }),
	)
}

// cosmeticsBack goes back to the menu the cosmetics came from: the lobby
// menu in the lobby, the game's menu (/menu) in a save.
func (pf *Platform) cosmeticsBack(p *player.Player) {
	if pf.m.Location(instance.AccountID(p)) == nil {
		pf.showLobbyMenu(p)
		return
	}
	p.ExecuteCommand("/menu")
}

func (pf *Platform) showTitles(p *player.Player) {
	acc := instance.AccountID(p)
	opts := []ui.Option{{Label: i18n.T(i18n.Of(p), "platform.cosmetics.showTitles.1"), Icon: iconDelete, Run: func(p *player.Player, _ *world.Tx) { pf.setTitle(p, "") }}}
	for _, id := range account.Owned(acc) {
		if a, ok := account.Get(id); ok && a.Title != "" {
			opts = append(opts, ui.Option{Label: "§e[" + a.Title + "]\n§8" + a.Name, Icon: iconCode, Run: func(p *player.Player, _ *world.Tx) { pf.setTitle(p, a.ID) }})
		}
	}
	opts = append(opts, ui.Back(i18n.T(i18n.Of(p), "platform.cosmetics.showTitles.2"), func(p *player.Player, _ *world.Tx) { pf.showCosmetics(p) }))
	body := i18n.T(i18n.Of(p), "platform.cosmetics.showTitles.3")
	if len(opts) == 2 {
		body += i18n.T(i18n.Of(p), "platform.cosmetics.showTitles.4")
	}
	ui.Menu(p, i18n.T(i18n.Of(p), "platform.cosmetics.showTitles.5"), body, opts...)
}

func (pf *Platform) setTitle(p *player.Player, id string) {
	acc := instance.AccountID(p)
	if id != "" && !slices.Contains(account.Owned(acc), id) {
		return
	}
	if err := pf.db.SetTitle(acc, id); err != nil {
		pf.fail(p, i18n.T(i18n.Of(p), "platform.cosmetics.setTitle.1"), err)
		return
	}
	account.Refresh(p)
	ui.Flash(p, i18n.T(i18n.Of(p), "platform.cosmetics.setTitle.2"), account.DisplayName(p))
	pf.showCosmetics(p)
}

func (pf *Platform) showColors(p *player.Player) {
	n := len(account.Owned(instance.AccountID(p)))
	var opts []ui.Option
	for _, c := range account.Colors {
		label := c.Code + c.Name
		if c.Need > n {
			label = fmt.Sprintf(i18n.T(i18n.Of(p), "platform.cosmetics.showColors.1"), c.Name, c.Need)
		}
		opts = append(opts, ui.Option{Label: label, Icon: iconRename, Run: func(p *player.Player, _ *world.Tx) {
			if c.Need > len(account.Owned(instance.AccountID(p))) {
				ui.Notify(p, i18n.T(i18n.Of(p), "platform.cosmetics.showColors.2"), c.Need)
				pf.showColors(p)
				return
			}
			if err := pf.db.SetNameColor(instance.AccountID(p), c.ID); err != nil {
				pf.fail(p, i18n.T(i18n.Of(p), "platform.cosmetics.showColors.3"), err)
				return
			}
			account.Refresh(p)
			pf.showCosmetics(p)
		}})
	}
	opts = append(opts, ui.Back(i18n.T(i18n.Of(p), "platform.cosmetics.showColors.4"), func(p *player.Player, _ *world.Tx) { pf.showCosmetics(p) }))
	ui.CardMenu(p, i18n.T(i18n.Of(p), "platform.cosmetics.showColors.5"), fmt.Sprintf(i18n.T(i18n.Of(p), "platform.cosmetics.showColors.6"), n), opts...)
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
			fmt.Fprintf(&b, i18n.T(i18n.Of(p), "platform.cosmetics.showAchievements.1"), a.Title)
		}
		b.WriteString("\n")
	}
	ui.Menu(p, i18n.T(i18n.Of(p), "platform.cosmetics.showAchievements.2"), b.String(), ui.Back(i18n.T(i18n.Of(p), "platform.cosmetics.showAchievements.3"), func(p *player.Player, _ *world.Tx) { pf.showCosmetics(p) }))
}
