package skygear

import (
	"strings"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/climate"
	"github.com/presentkim/skygear/content/skygear/machine"
)

// TestDaylight: the sun is full at noon, about a fifth at sunrise and dusk,
// and gone at night; rain and snow dim it.
func TestDaylight(t *testing.T) {
	for _, c := range []struct{ time, min, max int }{
		{6000, 100, 100}, {0, 15, 30}, {12000, 15, 30}, {13000, 0, 0}, {18000, 0, 0}, {23000, 0, 0}, {3000, 70, 95},
	} {
		if got := climate.Daylight(c.time); got < c.min || got > c.max {
			t.Errorf("daylight at %d is %d, want %d..%d", c.time, got, c.min, c.max)
		}
	}
	if climate.WeatherLight("") != 100 || climate.WeatherLight(climate.Rain) != climate.RainLight || climate.WeatherLight(climate.Snow) != climate.SnowLight {
		t.Error("weather light")
	}
}

// TestSolarGenerator: under the fixed day sun it makes its full output, a
// tier more; the rain dims it, the night and a roof stop it, glass does not.
func TestSolarGenerator(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	climate.StateOf(s.Session).Sun = climate.SunDay
	gen := placeTestMachine(s, machinePos, machine.KindSolarGenerator, "")
	gain := func(ticks int) int {
		before := gen.Energy
		tick(s, ticks)
		return gen.Energy - before
	}
	if got := gain(20); got != 20*machine.SolarOutput {
		t.Fatalf("made %d in 20 ticks of noon, want %d", got, 20*machine.SolarOutput)
	}
	if lines := strings.Join(machine.NatureLines(i18n.Default, gen), "\n"); !strings.Contains(lines, "RE/t") {
		t.Errorf("panel without its rate: %q", lines)
	}

	climate.StateOf(s.Session).Weather, climate.StateOf(s.Session).WeatherUntil = climate.Rain, time.Hour
	if got := gain(20); got != 20*machine.SolarOutput*climate.RainLight/100 {
		t.Errorf("made %d in 20 ticks of rain, want %d", got, 20*machine.SolarOutput*climate.RainLight/100)
	}
	climate.StateOf(s.Session).Weather, climate.StateOf(s.Session).WeatherUntil = "", 0

	climate.StateOf(s.Session).Sun = climate.SunNight
	if got := gain(20); got != 0 {
		t.Errorf("made %d at night", got)
	}
	climate.StateOf(s.Session).Sun = climate.SunDay

	above := machinePos.Add(cube.Pos{0, 3, 0})
	s.World.Do(func(tx *world.Tx) { tx.SetBlock(above, block.Glass{}, nil) })
	if gain(21); gain(20) != 20*machine.SolarOutput {
		t.Error("glass above stops it")
	}
	s.World.Do(func(tx *world.Tx) { tx.SetBlock(above, block.Stone{}, nil) })
	if gain(21); gain(20) != 0 {
		t.Error("it makes energy under a roof")
	}
	if lines := strings.Join(machine.NatureLines(i18n.Default, gen), "\n"); !strings.Contains(lines, i18n.T(i18n.Default, "naturegen.lines.noSky")) {
		t.Errorf("a roofed panel does not say why it stopped: %q", lines)
	}
	s.World.Do(func(tx *world.Tx) { tx.SetBlock(above, nil, nil) })

	gen.Tier = 1
	if gain(21); gen.GenRate() <= machine.SolarOutput || gain(20) != 20*gen.GenRate() {
		t.Error("a tier 1 panel makes no more")
	}
}

// TestHydroGenerator: still water makes nothing, flowing water by its depth
// and the faces it touches, up to the full output.
func TestHydroGenerator(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	// the water is set without block updates and the tests tick less than
	// the generator's recheck, so the flow they see is the one set
	place := func(pos cube.Pos, water map[cube.Face]block.Water) *machine.State {
		st := placeTestMachine(s, pos, machine.KindHydroGenerator, "")
		s.World.Do(func(tx *world.Tx) {
			for f, w := range water {
				tx.SetBlock(pos.Side(f), w, &world.SetOpts{DisableBlockUpdates: true, DisableLiquidDisplacement: true})
			}
		})
		return st
	}
	source := block.Water{Depth: 8, Still: true}
	falling := block.Water{Depth: 8, Falling: true}

	still := place(machinePos, map[cube.Face]block.Water{cube.FaceEast: source, cube.FaceWest: source})
	tick(s, 10)
	if still.Energy != 0 {
		t.Errorf("still water made %d", still.Energy)
	}
	if lines := strings.Join(machine.NatureLines(i18n.Default, still), "\n"); !strings.Contains(lines, i18n.T(i18n.Default, "naturegen.lines.stillWater")) {
		t.Errorf("no word of the still water: %q", lines)
	}

	full := place(machinePos.Add(cube.Pos{0, 0, 6}), map[cube.Face]block.Water{cube.FaceEast: falling, cube.FaceWest: falling, cube.FaceNorth: falling, cube.FaceSouth: falling})
	tick(s, 10)
	if full.Energy != 10*machine.HydroOutput {
		t.Errorf("four falling faces made %d in 10 ticks, want the full %d", full.Energy, 10*machine.HydroOutput)
	}

	// one face flowing at depth 6: 6/24 of the output, the fraction carried
	weak := place(machinePos.Add(cube.Pos{6, 0, 0}), map[cube.Face]block.Water{cube.FaceEast: {Depth: 6}})
	tick(s, 10)
	if want := 10 * machine.HydroOutput * machine.HydroShare(6) / 100; weak.Energy != want {
		t.Errorf("one face at depth 6 made %d in 10 ticks, want %d", weak.Energy, want)
	}

	dry := place(machinePos.Add(cube.Pos{-6, 0, 0}), nil)
	tick(s, 10)
	if dry.Energy != 0 || !strings.Contains(strings.Join(machine.NatureLines(i18n.Default, dry), "\n"), i18n.T(i18n.Default, "naturegen.lines.noWater")) {
		t.Error("a dry hydro generator makes energy or does not say why not")
	}
}
