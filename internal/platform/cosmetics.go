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

// The cosmetics are one window with three tabs: titles, name colours and
// achievements (docs/tech/ui-style.md).
const (
	cosmeticTitles = iota
	cosmeticColors
	cosmeticAchievements
)

// cosmeticTabs is the tab strip with cur selected.
func (pf *Platform) cosmeticTabs(p *player.Player, cur int) []ui.Option {
	lang := i18n.Of(p)
	tab := func(key, icon string, t int, show func(p *player.Player)) ui.Option {
		return ui.Tab(i18n.T(lang, key), icon, t == cur, func(p *player.Player, _ *world.Tx) { show(p) })
	}
	return []ui.Option{
		tab("platform.cosmetics.tab.titles", iconCode, cosmeticTitles, pf.showTitles),
		tab("platform.cosmetics.tab.colors", iconRename, cosmeticColors, pf.showColors),
		tab("platform.cosmetics.tab.achievements", iconGuide, cosmeticAchievements, pf.showAchievements),
		ui.Back(i18n.T(lang, "platform.cosmetics.showCosmetics.8"), func(p *player.Player, _ *world.Tx) { pf.cosmeticsBack(p) }),
	}
}

// showCosmetics opens the cosmetics window on its first tab.
func (pf *Platform) showCosmetics(p *player.Player) { pf.showTitles(p) }

// cosmeticsStatus is the line above the first tab: the display name, how many
// achievements are owned, the title and the name colour now.
func (pf *Platform) cosmeticsStatus(p *player.Player) string {
	acc, err := pf.db.Account(instance.AccountID(p))
	if err != nil {
		return ""
	}
	title := i18n.T(i18n.Of(p), "platform.cosmetics.showCosmetics.2")
	if a, ok := account.Get(acc.Title); ok {
		title = a.Title
	}
	return fmt.Sprintf(i18n.T(i18n.Of(p), "platform.cosmetics.showCosmetics.3"),
		account.DisplayName(p), len(account.Owned(acc.ID)), len(account.All()), title, account.ColorOf(acc.NameColor).Code, account.ColorOf(acc.NameColor).Name)
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

// pageSize is how many buttons a page of titles has.
const pageSize = 24

// groupOf is the group an achievement is listed under.
func groupOf(p *player.Player, a account.Achievement) string {
	if a.Group == "" {
		return i18n.T(i18n.Of(p), "platform.cosmetics.group.other")
	}
	return a.Group
}

// groupsOf lists the groups of the achievements in order, with how many
// achievements each has and how many of them the account owns.
func groupsOf(p *player.Player, owned []string) (names []string, total, have map[string]int) {
	total, have = map[string]int{}, map[string]int{}
	for _, g := range account.Groups() {
		if g == "" {
			g = i18n.T(i18n.Of(p), "platform.cosmetics.group.other")
		}
		names = append(names, g)
	}
	for _, a := range account.All() {
		g := groupOf(p, a)
		total[g]++
		if slices.Contains(owned, a.ID) {
			have[g]++
		}
	}
	return names, total, have
}

func (pf *Platform) showTitles(p *player.Player) {
	acc := instance.AccountID(p)
	owned := account.Owned(acc)
	opts := []ui.Option{{Label: i18n.T(i18n.Of(p), "platform.cosmetics.showTitles.1"), Icon: iconDelete, Run: func(p *player.Player, _ *world.Tx) { pf.setTitle(p, "") }}}
	names, _, _ := groupsOf(p, owned)
	titles := map[string]int{}
	for _, a := range account.All() {
		if a.Title != "" && slices.Contains(owned, a.ID) {
			titles[groupOf(p, a)]++
		}
	}
	for _, g := range names {
		if titles[g] > 0 {
			opts = append(opts, ui.Option{Label: fmt.Sprintf(i18n.T(i18n.Of(p), "platform.cosmetics.groupLabel"), g, titles[g]), Icon: iconCode, Run: func(p *player.Player, _ *world.Tx) { pf.showTitleGroup(p, g, 0) }})
		}
	}
	opts = append(opts, pf.cosmeticTabs(p, cosmeticTitles)...)
	body := pf.cosmeticsStatus(p) + "\n\n" + i18n.T(i18n.Of(p), "platform.cosmetics.showTitles.3")
	if len(titles) == 0 {
		body += i18n.T(i18n.Of(p), "platform.cosmetics.showTitles.4")
	}
	ui.Menu(p, i18n.T(i18n.Of(p), "platform.cosmetics.showTitles.5"), body, opts...)
}

// showTitleGroup lists the titles of a group, a page at a time.
func (pf *Platform) showTitleGroup(p *player.Player, group string, page int) {
	owned := account.Owned(instance.AccountID(p))
	var list []account.Achievement
	for _, a := range account.All() {
		if a.Title != "" && groupOf(p, a) == group && slices.Contains(owned, a.ID) {
			list = append(list, a)
		}
	}
	page = max(0, min(page, (len(list)-1)/pageSize))
	var opts []ui.Option
	for _, a := range list[page*pageSize : min(len(list), (page+1)*pageSize)] {
		opts = append(opts, ui.Option{Label: "§e[" + a.Title + "]\n§8" + a.Name, Icon: iconCode, Run: func(p *player.Player, _ *world.Tx) { pf.setTitle(p, a.ID) }})
	}
	opts = append(opts, ui.Pager(p, page, max(1, (len(list)+pageSize-1)/pageSize), func(p *player.Player, _ *world.Tx, to int) { pf.showTitleGroup(p, group, to) })...)
	opts = append(opts, ui.Back(i18n.T(i18n.Of(p), "platform.cosmetics.showTitles.2"), func(p *player.Player, _ *world.Tx) { pf.showTitles(p) }))
	ui.Menu(p, ui.SubTitle(i18n.T(i18n.Of(p), "platform.cosmetics.showCosmetics.4"), group), i18n.T(i18n.Of(p), "platform.cosmetics.showTitles.3"), opts...)
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
	opts = append(opts, pf.cosmeticTabs(p, cosmeticColors)...)
	ui.CardMenu(p, i18n.T(i18n.Of(p), "platform.cosmetics.showColors.5"), fmt.Sprintf(i18n.T(i18n.Of(p), "platform.cosmetics.showColors.6"), n), opts...)
}

func (pf *Platform) showAchievements(p *player.Player) {
	owned := account.Owned(instance.AccountID(p))
	names, total, have := groupsOf(p, owned)
	var opts []ui.Option
	for _, g := range names {
		label := fmt.Sprintf(i18n.T(i18n.Of(p), "platform.cosmetics.achGroupLabel"), g, have[g], total[g])
		opts = append(opts, ui.Option{Label: label, Icon: iconGuide, Run: func(p *player.Player, _ *world.Tx) { pf.showAchievementGroup(p, g) }})
	}
	opts = append(opts, pf.cosmeticTabs(p, cosmeticAchievements)...)
	ui.Menu(p, i18n.T(i18n.Of(p), "platform.cosmetics.showAchievements.2"), fmt.Sprintf(i18n.T(i18n.Of(p), "platform.cosmetics.achTotal"), len(owned), len(account.All())), opts...)
}

// showAchievementGroup lists the achievements of a group: the ones earned
// with their titles, the others with how far the counter is.
func (pf *Platform) showAchievementGroup(p *player.Player, group string) {
	acc := instance.AccountID(p)
	owned := account.Owned(acc)
	var b strings.Builder
	for _, a := range account.All() {
		if groupOf(p, a) != group {
			continue
		}
		mark := "§8□ §7"
		if slices.Contains(owned, a.ID) {
			mark = "§a■ §f"
		}
		fmt.Fprintf(&b, "%s%s §8- %s", mark, a.Name, a.Desc)
		if slices.Contains(owned, a.ID) {
			if a.Title != "" {
				fmt.Fprintf(&b, i18n.T(i18n.Of(p), "platform.cosmetics.showAchievements.1"), a.Title)
			}
		} else if have, need, ok := account.Progress(acc, a.ID); ok {
			fmt.Fprintf(&b, i18n.T(i18n.Of(p), "platform.cosmetics.progress"), commas(have), commas(need))
		}
		b.WriteString("\n")
	}
	ui.Menu(p, ui.SubTitle(i18n.T(i18n.Of(p), "platform.cosmetics.showCosmetics.4"), group), b.String(), ui.Back(i18n.T(i18n.Of(p), "platform.cosmetics.showAchievements.3"), func(p *player.Player, _ *world.Tx) { pf.showAchievements(p) }))
}

// commas writes n with thousands separators.
func commas(n int64) string {
	s := fmt.Sprint(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}
