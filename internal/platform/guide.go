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
const GuideVersion = 4

// guideTopics are the pages of the server guide, in display order. Their
// texts are "platform.guide.topic.<id>.title" and ".text" in the catalog.
var guideTopics = []struct{ id, icon string }{
	{"saves", "textures/ui/df/icon_cube"},
	{"together", "textures/ui/df/icon_people"},
	{"friends", "textures/ui/df/icon_person"},
	{"cosmetics", "textures/ui/df/icon_crown"},
	{"rules", "textures/ui/df/icon_help"},
}

// guideTopicText is the text of a guide page, with the server's numbers.
func (pf *Platform) guideTopicText(lang, id string) string {
	text := i18n.T(lang, "platform.guide.topic."+id+".text")
	switch id {
	case "saves":
		return fmt.Sprintf(text, Slots)
	case "together":
		return fmt.Sprintf(text, MaxMembers)
	}
	return text
}

// showGuide shows the server guide: a short welcome, the start button first
// and a button per topic (and the back button to the lobby menu when it is
// opened from there). With first set, the start button records that the
// account has seen the current version and goes on to the saves.
func (pf *Platform) showGuide(p *player.Player, first bool) {
	lang := i18n.Of(p)
	opts := []ui.Option{{Label: i18n.T(lang, "platform.guide.start"), Icon: iconEnter, Run: func(p *player.Player, _ *world.Tx) {
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
	}}}
	for _, t := range guideTopics {
		opts = append(opts, ui.Option{Label: i18n.T(lang, "platform.guide.topic."+t.id+".title"), Icon: t.icon, Run: func(p *player.Player, _ *world.Tx) {
			pf.showGuideTopic(p, t.id, first)
		}})
	}
	// opened again from the lobby menu, it goes back there
	if !first && pf.m.Location(instance.AccountID(p)) == nil {
		opts = append(opts, pf.lobbyBack(p, pf.showLobbyMenu))
	}
	body := fmt.Sprintf(i18n.T(lang, "platform.guide.intro"), pf.game.Name(lang))
	ui.Menu(p, i18n.T(lang, "platform.guide.showGuide.1"), body, opts...)
}

// showGuideTopic shows one page of the guide; back returns to its list.
func (pf *Platform) showGuideTopic(p *player.Player, id string, first bool) {
	lang := i18n.Of(p)
	title := ui.SubTitle(i18n.T(lang, "platform.guide.showGuide.1"), i18n.T(lang, "platform.guide.topic."+id+".title"))
	ui.Menu(p, title, pf.guideTopicText(lang, id), ui.Back(i18n.T(lang, "platform.guide.back"), func(p *player.Player, _ *world.Tx) {
		pf.showGuide(p, first)
	}))
}

func formatDuration(lang string, d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf(i18n.T(lang, "platform.guide.formatDuration.1"), h, m)
	}
	return fmt.Sprintf(i18n.T(lang, "platform.guide.formatDuration.2"), m)
}
