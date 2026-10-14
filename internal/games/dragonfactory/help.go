package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/i18n"
	"github.com/presentkim/dragonfactory/internal/ui"
)

// helpTopics are the in-game help pages (menu → 도움말), in display order.
var helpTopics = []struct{ title, icon, text string }{
	{i18n.T(i18n.Default, "dragonfactory.help.init.1"), iconMission, i18n.T(i18n.Default, "dragonfactory.help.init.2")},
	{i18n.T(i18n.Default, "dragonfactory.help.init.3"), iconCores, i18n.T(i18n.Default, "dragonfactory.help.init.4")},
	{i18n.T(i18n.Default, "dragonfactory.help.init.5"), iconAll, i18n.T(i18n.Default, "dragonfactory.help.init.6")},
	{i18n.T(i18n.Default, "dragonfactory.help.init.7"), iconSell, i18n.T(i18n.Default, "dragonfactory.help.init.8")},
	{i18n.T(i18n.Default, "dragonfactory.help.init.9"), iconLost, i18n.T(i18n.Default, "dragonfactory.help.init.10")},
	{i18n.T(i18n.Default, "dragonfactory.help.init.11"), iconIsland, i18n.T(i18n.Default, "dragonfactory.help.init.12")},
	{i18n.T(i18n.Default, "dragonfactory.help.init.13"), iconMission, i18n.T(i18n.Default, "dragonfactory.help.init.14")},
}

// showHelp lists the help topics.
func (s *session) showHelp(p *player.Player) {
	var opts []ui.Option
	for _, t := range helpTopics {
		opts = append(opts, s.opt("§f"+t.title, t.icon, func(p *player.Player, _ *world.Tx) {
			ui.Menu(p, i18n.T(i18n.Of(p), "dragonfactory.help.showHelp.1")+t.title, t.text, s.opt(i18n.T(i18n.Of(p), "dragonfactory.help.showHelp.2"), iconBack, func(p *player.Player, _ *world.Tx) { s.showHelp(p) }))
		}))
	}
	ui.CardMenu(p, i18n.T(i18n.Of(p), "dragonfactory.help.showHelp.3"), i18n.T(i18n.Of(p), "dragonfactory.help.showHelp.4"), append(opts, s.backOption())...)
}
