package platform

import (
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/platform/account"
	"github.com/presentkim/skygear/platform/instance"
)

// showSettings is the settings screen: a switch per account.Settings entry.
func (pf *Platform) showSettings(p *player.Player) {
	lang := i18n.Of(p)
	var opts []ui.Option
	for _, s := range account.Settings() {
		opts = append(opts, ui.Toggle(s.Name(lang), account.On(p, s.Key), func(p *player.Player, _ *world.Tx) {
			if _, err := account.Toggle(p, s.Key); err != nil {
				pf.fail(p, "store setting", err)
			}
			pf.showSettings(p)
		}))
	}
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
