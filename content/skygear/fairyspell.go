package skygear

import (
	"fmt"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/formicon"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/textfmt"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// The spells of the fairy board: the
// island's weather never changes by itself; a fairy emerald brings rain,
// which grows the timed crops faster, or snow, which runs the energy machines
// faster, for spellDays instance days. The sun follows the instance clock
// unless a spell fixes it at day, dusk or night. Values are drafts.

// The weathers of State.Weather; clear is "".
const (
	weatherRain = "rain"
	weatherSnow = "snow"
)

// The sun modes of State.Sun; flowing with the instance clock is "".
const (
	sunDay   = "day"
	sunDusk  = "dusk"
	sunNight = "night"
)

// spellDays is how many instance days rain and snow last, and add when cast
// again.
const spellDays = 3

// sunTimes are the world times of the fixed suns (the vanilla day of 24000
// ticks starts at sunrise).
var sunTimes = map[string]int{sunDay: 6000, sunDusk: 12000, sunNight: 18000}

// weatherNow is the weather at instance time now: clear once a rain or snow
// has run out.
func (s *session) weatherNow(now time.Duration) string {
	if now >= s.state.WeatherUntil {
		return ""
	}
	return s.state.Weather
}

// growNow is the growth clock: it runs with the instance clock, half as fast
// again while it rains. Crops and magic plants count their time on it
// (crops.go, magic.go). It starts at the instance time, so records made
// before it keep their times.
func (s *session) growNow() time.Duration {
	now, _, _ := s.Clock.Now()
	if !s.growInit {
		s.growInit = true
		if s.state.GrowClock == 0 {
			s.state.GrowClock = now
		}
		s.growAt = now
	}
	if d := now - s.growAt; d > 0 {
		var rain time.Duration
		if s.state.Weather == weatherRain {
			rain = max(0, min(now, s.state.WeatherUntil)-s.growAt)
		}
		s.state.GrowClock += d + rain/2
		s.growAt = now
	}
	return s.state.GrowClock
}

// realLeft turns growth time left into instance time at the current rate.
func (s *session) realLeft(d time.Duration) time.Duration {
	if now, _, _ := s.Clock.Now(); s.weatherNow(now) == weatherRain {
		return d * 2 / 3
	}
	return d
}

// snowReps is how many runs a machine that runs reps times a tick makes on
// top of them while it snows: half as many again, an odd half every other
// tick. The extra runs spend no energy (machineState.free).
func (s *session) snowReps(reps int, tick int64) int {
	if !s.snow {
		return 0
	}
	return reps/2 + reps%2*int(tick%2)
}

// tickEnvironment ends a weather that ran out, keeps the rain or snow falling and
// the sun up (Second).
func (s *session) tickEnvironment(tx *world.Tx) {
	now, _, _ := s.Clock.Now()
	if s.state.Weather != "" && s.weatherNow(now) == "" {
		s.growNow()
		s.state.Weather, s.state.WeatherUntil = "", 0
		s.MarkDirty()
		s.applyWeather(tx)
	}
	s.tickFall(tx, now)
	s.applySun(tx)
}

// applyEnvironment shows the save's weather and sun (session start).
func (s *session) applyEnvironment(tx *world.Tx) {
	s.restoreBiome(tx)
	s.applyWeather(tx)
	s.applySun(tx)
}

// applyWeather shows the weather to everyone on the island: rain as falling
// raindrops, snow as a white fog and falling snowflakes (showWeather,
// tickFall). The world itself never rains; the client draws only what the
// server sends.
func (s *session) applyWeather(tx *world.Tx) {
	now, _, _ := s.Clock.Now()
	w, weather := tx.World(), s.weatherNow(now)
	w.StopWeatherCycle()
	w.StopRaining()
	s.rain = weather == weatherRain
	s.snow = weather == weatherSnow
	for e := range tx.Players() {
		s.showWeather(e.(*player.Player))
	}
}

// snowFog is the fog of the snow (pack fogs/snow_fog.json); snowfall and
// rainfall are the particle effects of falling snowflakes and raindrops
// around a player (particles/snowfall.json, rainfall.json), which last
// fallEvery.
const (
	snowFog   = "skygear:snow_fog"
	snowfall  = "skygear:snowfall"
	rainfall  = "skygear:rainfall"
	fallEvery = 2 // seconds
)

// showWeather sets p's fog for the weather.
func (s *session) showWeather(p *player.Player) {
	stack := []string{}
	if s.snow {
		stack = []string{snowFog}
	}
	sghook.WritePacket(sghook.Session(p), &packet.PlayerFog{Stack: stack})
}

// tickFall starts the falling snowflakes or raindrops around every player
// while it snows or rains. The client draws them; the server sends a packet a
// player every fallEvery seconds.
func (s *session) tickFall(tx *world.Tx, now time.Duration) {
	name := ""
	switch {
	case s.snow:
		name = snowfall
	case s.rain:
		name = rainfall
	}
	if name == "" || now/time.Second%fallEvery != 0 {
		return
	}
	for e := range tx.Players() {
		p := e.(*player.Player)
		sghook.WritePacket(sghook.Session(p), &packet.SpawnParticleEffect{
			EntityUniqueID: -1,
			Position:       mgl32.Vec3{float32(p.Position()[0]), float32(p.Position()[1]), float32(p.Position()[2])},
			ParticleName:   name,
		})
	}
}

// restoreBiome gives back the island's biome an old save's snow changed
// (State.BaseBiome), once.
func (s *session) restoreBiome(tx *world.Tx) {
	if s.state.BaseBiome == nil {
		return
	}
	want, r := *s.state.BaseBiome, tx.Range()
	for _, c := range s.state.Chunks {
		for dx := int32(-1); dx <= 1; dx++ {
			for dz := int32(-1); dz <= 1; dz++ {
				ch := sghook.Column(tx, world.ChunkPos{c[0] + dx, c[1] + dz}).Chunk
				for y := r[0]; y <= r[1]; y++ {
					for x := uint8(0); x < 16; x++ {
						for z := uint8(0); z < 16; z++ {
							ch.SetBiome(x, int16(y), z, want)
						}
					}
				}
			}
		}
	}
	s.state.BaseBiome = nil
	s.MarkDirty()
}

// applySun keeps the world's sun on the instance clock, or at the fixed
// time.
func (s *session) applySun(tx *world.Tx) {
	_, day, dayTime := s.Clock.Now()
	w := tx.World()
	base := (day - 1) * 24000
	if t, fixed := sunTimes[s.state.Sun]; fixed {
		if w.TimeCycle() {
			w.StopTime()
		}
		if w.Time() != base+t {
			w.SetTime(base + t)
		}
		return
	}
	if !w.TimeCycle() {
		w.StartTime()
	}
	want := base
	if l := s.Clock.DayLength(); l > 0 {
		want += int(dayTime * 24000 / l)
	}
	// the world ticks the time on by itself; set it when it drifts more
	// than a second
	if d := w.Time() - want; d > 20 || d < -20 {
		w.SetTime(want)
	}
}

// castWeather casts a weather: the same rain or snow adds spellDays, another
// one replaces it. It reports false if there is nothing to change (clear
// when it is clear), which costs nothing.
func (s *session) castWeather(tx *world.Tx, weather string) bool {
	now, _, _ := s.Clock.Now()
	cur := s.weatherNow(now)
	if weather == "" && cur == "" {
		return false
	}
	s.growNow()
	span := spellDays * s.Clock.DayLength()
	switch {
	case weather == "":
		s.state.Weather, s.state.WeatherUntil = "", 0
	case weather == cur:
		s.state.WeatherUntil += span
	default:
		s.state.Weather, s.state.WeatherUntil = weather, now+span
	}
	s.MarkDirty()
	s.applyWeather(tx)
	return true
}

// castSun fixes the sun or lets it flow. It reports false if the sun is
// already so.
func (s *session) castSun(tx *world.Tx, mode string) bool {
	if s.state.Sun == mode {
		return false
	}
	s.state.Sun = mode
	s.MarkDirty()
	s.applySun(tx)
	return true
}

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
	emeralds := countPlain(p.Inventory(), sgitem.FairyEmerald{})
	weather := s.weatherNow(now)
	cast := func(do func(tx *world.Tx) bool) func(p *player.Player, tx *world.Tx) {
		return func(p *player.Player, tx *world.Tx) {
			if countPlain(p.Inventory(), sgitem.FairyEmerald{}) > 0 && do(tx) {
				takePlain(p.Inventory(), sgitem.FairyEmerald{}, 1)
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
		{weatherRain, item.NewStack(item.Bucket{Content: item.LiquidBucketContent(block.Water{Still: true, Depth: 8})}, 1)},
		{weatherSnow, item.NewStack(item.Snowball{}, 1)},
	} {
		k := weatherKey(w.weather)
		if weather == w.weather {
			weatherIcon = formicon.Of(w.icon)
		}
		weathers.Choices = append(weathers.Choices, choice("fairy.weather."+k, "fairy.board.weather."+k+".info", w.icon, weather == w.weather,
			func(tx *world.Tx) bool { return s.castWeather(tx, w.weather) }))
	}
	suns := ui.ChoiceGroup{Title: i18n.T(lang, "fairy.board.suns")}
	var sunIcon string
	for _, m := range []struct {
		sun  string
		icon item.Stack
	}{
		{"", item.NewStack(item.Clock{}, 1)},
		{sunDay, item.NewStack(block.Glowstone{}, 1)},
		{sunDusk, item.NewStack(block.Torch{}, 1)},
		{sunNight, item.NewStack(item.EnderPearl{}, 1)},
	} {
		k := sunKey(m.sun)
		if s.state.Sun == m.sun {
			sunIcon = formicon.Of(m.icon)
		}
		suns.Choices = append(suns.Choices, choice("fairy.board.sun."+k, "fairy.board.sun."+k+".info", m.icon, s.state.Sun == m.sun,
			func(tx *world.Tx) bool { return s.castSun(tx, m.sun) }))
	}
	weatherText := i18n.T(lang, "fairy.weather."+weatherKey(weather))
	if weather != "" {
		weatherText = fmt.Sprintf(i18n.T(lang, "fairy.board.left"), weatherText, textfmt.Duration(lang, (s.state.WeatherUntil-now).Round(time.Second)))
	}
	ui.ChoiceMenu(p, i18n.T(lang, "fairy.board.title"), ui.ChoiceBoard{
		Stats: []ui.Stat{
			{Icon: formicon.Of(item.NewStack(sgitem.FairyEmerald{}, 1)), Label: i18n.T(lang, "fairy.board.emeralds"), Value: fmt.Sprintf(i18n.T(lang, "fairy.board.count"), emeralds)},
			{Icon: weatherIcon, Label: i18n.T(lang, "fairy.board.weather"), Value: weatherText},
			{Icon: sunIcon, Label: i18n.T(lang, "fairy.board.sun"), Value: i18n.T(lang, "fairy.sun."+sunKey(s.state.Sun))},
		},
		Note:   i18n.T(lang, "fairy.board.note"),
		Groups: []ui.ChoiceGroup{weathers, suns},
	})
}

// weatherKey and sunKey name a weather and a sun mode in the catalog.
func weatherKey(w string) string {
	if w == "" {
		return "clear"
	}
	return w
}

func sunKey(m string) string {
	if m == "" {
		return "flow"
	}
	return m
}
