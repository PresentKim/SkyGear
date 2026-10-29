package skygear

import (
	"testing"
	"time"

	"github.com/presentkim/skygear/content/skygear/machine"

	"github.com/presentkim/skygear/content/skygear/island"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/biome"
)

// cast casts a weather in the session's world.
func cast(t *testing.T, s *session, weather string) bool {
	t.Helper()
	var ok bool
	s.World.Do(func(tx *world.Tx) { ok = s.castWeather(tx, weather) })
	return ok
}

// TestRainGrowsFaster: the growth clock runs half as fast again while it
// rains, and as the instance clock once the rain ends.
func TestRainGrowsFaster(t *testing.T) {
	s := testSession(t)
	start := s.growNow()
	s.Clock.Advance(time.Minute)
	if got := s.growNow() - start; got != time.Minute {
		t.Fatalf("clear: grew %v in a minute", got)
	}
	if !cast(t, s, weatherRain) {
		t.Fatal("rain was not cast")
	}
	start = s.growNow()
	s.Clock.Advance(time.Minute)
	if got := s.growNow() - start; got != 90*time.Second {
		t.Fatalf("rain: grew %v in a minute", got)
	}
	// the rain ends within the next stretch
	now, _, _ := s.Clock.Now()
	left := s.state.WeatherUntil - now
	start = s.growNow()
	s.Clock.Advance(left + time.Minute)
	if got, want := s.growNow()-start, left*3/2+time.Minute; got != want {
		t.Fatalf("rain running out: grew %v, want %v", got, want)
	}
}

// TestWeatherSpells: the same weather adds its days, another replaces it,
// clear ends it, and clear when clear costs nothing.
func TestWeatherSpells(t *testing.T) {
	s := testSession(t)
	span := spellDays * s.Clock.DayLength()
	if cast(t, s, "") {
		t.Fatal("clear was cast on a clear day")
	}
	cast(t, s, weatherRain)
	cast(t, s, weatherRain)
	if now, _, _ := s.Clock.Now(); s.state.WeatherUntil != now+2*span {
		t.Fatalf("rain twice ends at %v", s.state.WeatherUntil)
	}
	cast(t, s, weatherSnow)
	now, _, _ := s.Clock.Now()
	if s.weatherNow(now) != weatherSnow || s.state.WeatherUntil != now+span || !s.snow {
		t.Fatalf("snow over rain: %q until %v", s.state.Weather, s.state.WeatherUntil)
	}
	if !cast(t, s, "") || s.weatherNow(now) != "" || s.snow {
		t.Fatal("clear did not end the snow")
	}
	cast(t, s, weatherSnow)
	s.Clock.Advance(span)
	s.World.Do(func(tx *world.Tx) { s.tickEnvironment(tx) })
	if s.state.Weather != "" || s.snow {
		t.Fatal("the snow did not run out")
	}
}

// TestSnowRunsMachinesFaster: in the snow a crusher works half as fast
// again for the same energy a tick.
func TestSnowRunsMachinesFaster(t *testing.T) {
	run := func(snow bool) (progress, spent int) {
		s := testSession(t)
		loadAround(s)
		if snow {
			cast(t, s, weatherSnow)
		}
		st := placeTestMachine(s, machinePos, machine.KindCrusher, "")
		st.Unlimited = false
		st.Energy = 1000
		st.Insert(item.NewStack(block.Cobblestone{}, 1))
		tick(s, machine.Kinds[machine.KindCrusher].Interval/2)
		return st.Progress, 1000 - st.Energy
	}
	p, e := run(false)
	sp, se := run(true)
	if p == 0 || sp != p*3/2 || se != e {
		t.Errorf("clear: progress %d for %d RE; snow: progress %d for %d RE", p, e, sp, se)
	}
}

// TestSunSpells: the sun follows the instance clock unless fixed, and a
// spell for the sun it already has costs nothing.
func TestSunSpells(t *testing.T) {
	s := testSession(t)
	s.Clock.Advance(s.Clock.DayLength() + s.Clock.DayLength()/2)
	s.World.Do(func(tx *world.Tx) {
		s.applySun(tx)
		if got := tx.World().Time(); got != 24000+12000 || !tx.World().TimeCycle() {
			t.Errorf("flowing sun at %d", got)
		}
		if s.castSun(tx, "") {
			t.Error("the flowing sun was cast again")
		}
		if !s.castSun(tx, sunNight) || tx.World().Time() != 24000+18000 || tx.World().TimeCycle() {
			t.Errorf("night sun at %d", tx.World().Time())
		}
	})
}

// TestRestoreBiome: a biome the snow changed comes back once.
func TestRestoreBiome(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
	snowy, plains := uint32(biome.SnowyPlains{}.EncodeBiome()), uint32(biome.Plains{}.EncodeBiome())
	s.state.BaseBiome = &plains
	s.World.Do(func(tx *world.Tx) {
		tx.SetBiome(cube.Pos{3, island.IslandY, 3}, biome.SnowyPlains{})
		tx.SetBiome(cube.Pos{-5, island.IslandY, 20}, biome.SnowyPlains{})
		s.applyEnvironment(tx)
		for _, pos := range []cube.Pos{{3, island.IslandY, 3}, {-5, island.IslandY, 20}} {
			if got := uint32(tx.Biome(pos).EncodeBiome()); got != plains || got == snowy {
				t.Errorf("biome at %v is %d", pos, got)
			}
		}
	})
	if s.state.BaseBiome != nil {
		t.Error("the biome was not marked given back")
	}
}
