// Package climate is the island's weather, sun and growth clock. The
// weather never changes by itself: a fairy spell (the fairy package) brings
// rain, which grows the timed crops faster (GrowNow), or snow, which runs the
// energy machines faster (SnowReps), for spellDays instance days. The sun
// follows the instance clock unless a spell fixes it at day, dusk or night.
// Values are drafts.
package climate

import (
	"time"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// The weathers of State.Weather; clear is "".
const (
	Rain = "rain"
	Snow = "snow"
)

// The sun modes of State.Sun; flowing with the instance clock is "".
const (
	SunDay   = "day"
	SunDusk  = "dusk"
	SunNight = "night"
)

// SpellDays is how many instance days rain and snow last, and add when cast
// again.
const SpellDays = 3

// sunTimes are the world times of the fixed suns (the vanilla day of 24000
// ticks starts at sunrise).
var sunTimes = map[string]int{SunDay: 6000, SunDusk: 12000, SunNight: 18000}

// WeatherNow is the weather at instance time now: clear once a rain or snow
// has run out.
func WeatherNow(s *core.Session, now time.Duration) string {
	if now >= StateOf(s).WeatherUntil {
		return ""
	}
	return StateOf(s).Weather
}

// GrowNow is the growth clock: it runs with the instance clock, half as fast
// again while it rains. Crops and magic plants count their time on it
// (crops.go, magic.go). It starts at the instance time, so records made
// before it keep their times.
func GrowNow(s *core.Session) time.Duration {
	now, _, _ := s.Clock.Now()
	if !localOf(s).growInit {
		localOf(s).growInit = true
		if StateOf(s).GrowClock == 0 {
			StateOf(s).GrowClock = now
		}
		localOf(s).growAt = now
	}
	if d := now - localOf(s).growAt; d > 0 {
		var rain time.Duration
		if StateOf(s).Weather == Rain {
			rain = max(0, min(now, StateOf(s).WeatherUntil)-localOf(s).growAt)
		}
		StateOf(s).GrowClock += d + rain/2
		localOf(s).growAt = now
	}
	return StateOf(s).GrowClock
}

// RealLeft turns growth time left into instance time at the current rate.
func RealLeft(s *core.Session, d time.Duration) time.Duration {
	if now, _, _ := s.Clock.Now(); WeatherNow(s, now) == Rain {
		return d * 2 / 3
	}
	return d
}

// SnowReps is how many runs a machine that runs reps times a tick makes on
// top of them while it snows: half as many again, an odd half every other
// tick. The extra runs spend no energy (machineState.free).
func SnowReps(s *core.Session, reps int, tick int64) int {
	if !localOf(s).snow {
		return 0
	}
	return reps/2 + reps%2*int(tick%2)
}

// Tick ends a weather that ran out, keeps the rain or snow falling and
// the sun up (Second).
func Tick(s *core.Session, tx *world.Tx) {
	now, _, _ := s.Clock.Now()
	if StateOf(s).Weather != "" && WeatherNow(s, now) == "" {
		GrowNow(s)
		StateOf(s).Weather, StateOf(s).WeatherUntil = "", 0
		s.MarkDirty()
		applyWeather(s, tx)
	}
	tickFall(s, tx, now)
	ApplySun(s, tx)
}

// Apply shows the save's weather and sun (session start).
func Apply(s *core.Session, tx *world.Tx) {
	restoreBiome(s, tx)
	applyWeather(s, tx)
	ApplySun(s, tx)
}

// applyWeather shows the weather to everyone on the island: rain as falling
// raindrops, snow as a white fog and falling snowflakes (showWeather,
// tickFall). The world itself never rains; the client draws only what the
// server sends.
func applyWeather(s *core.Session, tx *world.Tx) {
	now, _, _ := s.Clock.Now()
	w, weather := tx.World(), WeatherNow(s, now)
	w.StopWeatherCycle()
	w.StopRaining()
	localOf(s).rain = weather == Rain
	localOf(s).snow = weather == Snow
	for e := range tx.Players() {
		ShowWeather(s, e.(*player.Player))
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

// ShowWeather sets p's fog for the weather.
func ShowWeather(s *core.Session, p *player.Player) {
	stack := []string{}
	if localOf(s).snow {
		stack = []string{snowFog}
	}
	sghook.WritePacket(sghook.Session(p), &packet.PlayerFog{Stack: stack})
}

// tickFall starts the falling snowflakes or raindrops around every player
// while it snows or rains. The client draws them; the server sends a packet a
// player every fallEvery seconds.
func tickFall(s *core.Session, tx *world.Tx, now time.Duration) {
	name := ""
	switch {
	case localOf(s).snow:
		name = snowfall
	case localOf(s).rain:
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
func restoreBiome(s *core.Session, tx *world.Tx) {
	if StateOf(s).BaseBiome == nil {
		return
	}
	want, r := *StateOf(s).BaseBiome, tx.Range()
	for _, c := range island.StateOf(s).Chunks {
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
	StateOf(s).BaseBiome = nil
	s.MarkDirty()
}

// ApplySun keeps the world's sun on the instance clock, or at the fixed
// time.
func ApplySun(s *core.Session, tx *world.Tx) {
	_, day, dayTime := s.Clock.Now()
	w := tx.World()
	base := (day - 1) * 24000
	if t, fixed := sunTimes[StateOf(s).Sun]; fixed {
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

// CastWeather casts a weather: the same rain or snow adds spellDays, another
// one replaces it. It reports false if there is nothing to change (clear
// when it is clear), which costs nothing.
func CastWeather(s *core.Session, tx *world.Tx, weather string) bool {
	now, _, _ := s.Clock.Now()
	cur := WeatherNow(s, now)
	if weather == "" && cur == "" {
		return false
	}
	GrowNow(s)
	span := SpellDays * s.Clock.DayLength()
	switch {
	case weather == "":
		StateOf(s).Weather, StateOf(s).WeatherUntil = "", 0
	case weather == cur:
		StateOf(s).WeatherUntil += span
	default:
		StateOf(s).Weather, StateOf(s).WeatherUntil = weather, now+span
	}
	s.MarkDirty()
	applyWeather(s, tx)
	return true
}

// CastSun fixes the sun or lets it flow. It reports false if the sun is
// already so.
func CastSun(s *core.Session, tx *world.Tx, mode string) bool {
	if StateOf(s).Sun == mode {
		return false
	}
	StateOf(s).Sun = mode
	s.MarkDirty()
	ApplySun(s, tx)
	return true
}

// WeatherKey and sunKey name a weather and a sun mode in the catalog.
func WeatherKey(w string) string {
	if w == "" {
		return "clear"
	}
	return w
}

func SunKey(m string) string {
	if m == "" {
		return "flow"
	}
	return m
}

// State is what a save keeps of its climate: its fields are part of the
// save's state document (core.Saved).
type State struct {
	// Weather is the weather a fairy spell brought ("", rain or snow) and
	// WeatherUntil the instance time it ends (fairyspell.go).
	Weather      string        `json:"weather,omitempty"`
	WeatherUntil time.Duration `json:"weather_until,omitempty"`
	// Sun is the fixed sun ("" flows with the instance clock).
	Sun string `json:"sun,omitempty"`
	// GrowClock is the growth clock the crops and magic plants count on
	// (growNow).
	GrowClock time.Duration `json:"grow_clock,omitempty"`
	// BaseBiome is the island's biome an old save's snow changed, given back
	// at the next start (restoreBiome).
	BaseBiome *uint32 `json:"base_biome,omitempty"`
}

var saved = core.NewSaved[State](nil)

// StateOf is what the save of s keeps of its climate.
func StateOf(s *core.Session) *State { return saved.Of(s) }

// local is what a running save keeps for its climate.
type local struct {
	// growInit and growAt are whether the growth clock started and the
	// instance time it last ran to; rain and snow are whether it rains or
	// snows now (fairyspell.go).
	growInit bool
	growAt   time.Duration
	rain     bool
	snow     bool
}

var localPart = core.NewPart(func() *local { return &local{} })

// localOf is what the save of s keeps for its climate.
func localOf(s *core.Session) *local { return localPart.Of(s) }

// Snowing reports whether it snows on the island now.
func Snowing(s *core.Session) bool { return localOf(s).snow }
