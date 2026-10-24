package platform

import (
	"fmt"
	"github.com/presentkim/dragonfactory/common/i18n"
	"time"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/ui"
	"github.com/presentkim/dragonfactory/internal/instance"
)

// GuideVersion is increased when the guide changes; players see it again once.
const GuideVersion = 3

func (pf *Platform) guideText(lang string) string {
	return fmt.Sprintf(i18n.T(lang, "platform.guide.guideText.1"), pf.game.Name(lang), Slots, MaxMembers)
}

// showGuide shows the server guide. With first set, closing it records that
// the account has seen the current version.
func (pf *Platform) showGuide(p *player.Player, first bool) {
	ok := ui.Option{Label: i18n.T(i18n.Of(p), "platform.guide.showGuide.2"), Icon: iconEnter, Run: func(p *player.Player, _ *world.Tx) {
		if first {
			if err := pf.db.SetGuideVersion(instance.AccountID(p), GuideVersion); err != nil {
				pf.log.Error("set guide version", "err", err)
			}
			pf.showSaves(p, 0)
			return
		}
		if pf.m.Location(instance.AccountID(p)) == nil {
			pf.showLobbyMenu(p)
		}
	}}
	ui.Menu(p, i18n.T(i18n.Of(p), "platform.guide.showGuide.1"), pf.guideText(i18n.Of(p)), ok)
}

func formatDuration(lang string, d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf(i18n.T(lang, "platform.guide.formatDuration.1"), h, m)
	}
	return fmt.Sprintf(i18n.T(lang, "platform.guide.formatDuration.2"), m)
}
