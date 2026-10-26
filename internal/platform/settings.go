package platform

import (
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/common/ui"
	"github.com/presentkim/dragonfactory/internal/account"
	"github.com/presentkim/dragonfactory/internal/instance"
)

// showSettings is the settings screen (docs/design/01-platform.md §5.1): a
// button for every player setting (account.Settings) with its name and
// whether it is on, then the menu style (uimode), and the back button to the
// menu it came from. Pressing a setting turns it over and shows the screen
// again.
func (pf *Platform) showSettings(p *player.Player) {
	lang := i18n.Of(p)
	row := func(name string, on bool, toggle func(p *player.Player)) ui.Option {
		state, icon := i18n.T(lang, "setting.off"), "textures/ui/cancel"
		if on {
			state, icon = i18n.T(lang, "setting.on"), "textures/ui/confirm"
		}
		return ui.Option{Label: name + "  " + state, Icon: icon, Run: func(p *player.Player, _ *world.Tx) {
			toggle(p)
			pf.showSettings(p)
		}}
	}
	var opts []ui.Option
	for _, s := range account.Settings() {
		opts = append(opts, row(s.Name(lang), account.On(p, s.Key), func(p *player.Player) {
			if _, err := account.Toggle(p, s.Key); err != nil {
				pf.fail(p, "store setting", err)
			}
		}))
	}
	_, json := jsonUI.Load(p.UUID())
	opts = append(opts, row(i18n.T(lang, "setting.uimode"), json, func(p *player.Player) { pf.setUIMode(p, !json) }))
	opts = append(opts, ui.Back(i18n.T(lang, "platform.back"), func(p *player.Player, _ *world.Tx) { pf.menuBack(p, instance.ScreenSettings) }))
	ui.Menu(p, i18n.T(lang, "setting.title"), "", opts...)
}

type settingsCmd struct {
	playerOnly
	pf *Platform
}

func (c settingsCmd) Run(src cmd.Source, _ *cmd.Output, _ *world.Tx) {
	c.pf.showSettings(src.(*player.Player))
}
