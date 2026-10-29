package skygear

import (
	"fmt"
	"time"

	"github.com/presentkim/skygear/content/skygear/climate"

	"github.com/presentkim/skygear/content/skygear/core"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/formicon"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/textfmt"
)

// The spells of the fairy board: the
// island's weather never changes by itself; a fairy emerald brings rain,
// which grows the timed crops faster, or snow, which runs the energy machines
// faster, for spellDays instance days. The sun follows the instance clock
// unless a spell fixes it at day, dusk or night. Values are drafts.

// Use opens the spells of the fairy board.
func (FairyBoard) Use(tx *world.Tx, u item.User, _ *item.UseContext) bool {
	p, ok := u.(*player.Player)
	if !ok {
		return false
	}
	if s, ok := activeGame.sessionOf(tx.World()); ok {
		s.showFairyBoard(p)
	}
	return false
}

// showFairyBoard shows the spells: the emeralds, the weather and the sun at
// the top, then the weathers and the suns, the current ones marked, each
// casting for a fairy emerald from p's inventory.
func (s *session) showFairyBoard(p *player.Player) {
	lang := i18n.Of(p)
	now, _, _ := s.Clock.Now()
	emeralds := core.CountPlain(p.Inventory(), sgitem.FairyEmerald{})
	weather := climate.WeatherNow(s.Session, now)
	cast := func(do func(tx *world.Tx) bool) func(p *player.Player, tx *world.Tx) {
		return func(p *player.Player, tx *world.Tx) {
			if core.CountPlain(p.Inventory(), sgitem.FairyEmerald{}) > 0 && do(tx) {
				core.TakePlain(p.Inventory(), sgitem.FairyEmerald{}, 1)
				tx.PlaySound(p.Position(), sound.Experience{})
			}
			s.showFairyBoard(p)
		}
	}
	choice := func(name, info string, icon item.Stack, on bool, do func(tx *world.Tx) bool) ui.Choice {
		return ui.Choice{Label: i18n.T(lang, name), Info: i18n.T(lang, info), Icon: formicon.Of(icon), Badge: i18n.T(lang, "fairy.board.now"),
			On: on, Dim: emeralds == 0, Run: cast(do)}
	}
	weathers := ui.ChoiceGroup{Title: i18n.T(lang, "fairy.board.weathers")}
	var weatherIcon string
	for _, w := range []struct {
		weather string
		icon    item.Stack
	}{
		{"", item.NewStack(block.DoubleFlower{Type: block.Sunflower()}, 1)},
		{climate.Rain, item.NewStack(item.Bucket{Content: item.LiquidBucketContent(block.Water{Still: true, Depth: 8})}, 1)},
		{climate.Snow, item.NewStack(item.Snowball{}, 1)},
	} {
		k := climate.WeatherKey(w.weather)
		if weather == w.weather {
			weatherIcon = formicon.Of(w.icon)
		}
		weathers.Choices = append(weathers.Choices, choice("fairy.weather."+k, "fairy.board.weather."+k+".info", w.icon, weather == w.weather,
			func(tx *world.Tx) bool { return climate.CastWeather(s.Session, tx, w.weather) }))
	}
	suns := ui.ChoiceGroup{Title: i18n.T(lang, "fairy.board.suns")}
	var sunIcon string
	for _, m := range []struct {
		sun  string
		icon item.Stack
	}{
		{"", item.NewStack(item.Clock{}, 1)},
		{climate.SunDay, item.NewStack(block.Glowstone{}, 1)},
		{climate.SunDusk, item.NewStack(block.Torch{}, 1)},
		{climate.SunNight, item.NewStack(item.EnderPearl{}, 1)},
	} {
		k := climate.SunKey(m.sun)
		if climate.StateOf(s.Session).Sun == m.sun {
			sunIcon = formicon.Of(m.icon)
		}
		suns.Choices = append(suns.Choices, choice("fairy.board.sun."+k, "fairy.board.sun."+k+".info", m.icon, climate.StateOf(s.Session).Sun == m.sun,
			func(tx *world.Tx) bool { return climate.CastSun(s.Session, tx, m.sun) }))
	}
	weatherText := i18n.T(lang, "fairy.weather."+climate.WeatherKey(weather))
	if weather != "" {
		weatherText = fmt.Sprintf(i18n.T(lang, "fairy.board.left"), weatherText, textfmt.Duration(lang, (climate.StateOf(s.Session).WeatherUntil-now).Round(time.Second)))
	}
	ui.ChoiceMenu(p, i18n.T(lang, "fairy.board.title"), ui.ChoiceBoard{
		Stats: []ui.Stat{
			{Icon: formicon.Of(item.NewStack(sgitem.FairyEmerald{}, 1)), Label: i18n.T(lang, "fairy.board.emeralds"), Value: fmt.Sprintf(i18n.T(lang, "fairy.board.count"), emeralds)},
			{Icon: weatherIcon, Label: i18n.T(lang, "fairy.board.weather"), Value: weatherText},
			{Icon: sunIcon, Label: i18n.T(lang, "fairy.board.sun"), Value: i18n.T(lang, "fairy.sun."+climate.SunKey(climate.StateOf(s.Session).Sun))},
		},
		Note:   i18n.T(lang, "fairy.board.note"),
		Groups: []ui.ChoiceGroup{weathers, suns},
	})
}
