package machine

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/climate"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/network"
)

// The generators that burn nothing (docs/design/06-power-and-conduits.md §2):
// the solar generator makes energy from the sunlight on its top while it sees
// the sky (the instance clock's sun or the fixed one, and the weather), the
// hydro generator from the flowing water on its faces. They make a share of
// their full output (Def.Generates, raised by the tier) a tick; the share's
// fraction carries over to the next tick. Values are drafts.
const (
	// SolarOutput is the RE per tick of a solar generator at noon under a
	// clear sky.
	SolarOutput = 8
	// HydroOutput is the RE per tick of a hydro generator at full flow, and
	// HydroFull the flow that makes it (waterFlow: three faces of falling
	// water).
	HydroOutput = 10
	HydroFull   = 24
	// natureRecheck is how many ticks apart a generator looks at the sky or
	// the water around it again.
	natureRecheck = 20
)

// natureLook is what a solar or hydro generator last saw around it.
type natureLook struct {
	// known is whether it looked yet, at the tick it last looked.
	known bool
	at    int64
	// sky is whether the solar generator sees the sky; flow is the flow of
	// the water on the hydro generator's faces, faces the faces with
	// flowing water and still those with only still water (sources).
	sky                bool
	flow, faces, still int
	// pct is the share of the full output it made in its last tick, frac
	// the hundredths of RE carried over.
	pct, frac int
}

// SkyOpen reports whether the block at pos sees the sky: nothing but air and
// glass above it.
func SkyOpen(tx *world.Tx, pos cube.Pos) bool {
	top := tx.HighestBlock(pos.X(), pos.Z())
	for y := pos.Y() + 1; y <= top; y++ {
		switch tx.Block(cube.Pos{pos.X(), y, pos.Z()}).(type) {
		case block.Air, block.Glass, block.StainedGlass, block.GlassPane, block.StainedGlassPane:
		default:
			return false
		}
	}
	return true
}

// waterFlow is how strongly a water block flows past a face: falling water
// the most (8), flowing water by its depth (7 next to its source, down to 1
// at its end), a still source not at all.
func waterFlow(w block.Water) int {
	switch {
	case w.Falling:
		return 8
	case w.Depth >= 8:
		return 0
	}
	return max(0, w.Depth)
}

// HydroFlow sums the flow of the water on the six faces of pos; faces counts
// the faces with flowing water and still those with still water only.
func HydroFlow(tx *world.Tx, pos cube.Pos) (flow, faces, still int) {
	for _, f := range cube.Faces() {
		liq, ok := tx.Liquid(pos.Side(f))
		if !ok {
			continue
		}
		w, ok := liq.(block.Water)
		if !ok {
			continue
		}
		if v := waterFlow(w); v > 0 {
			flow += v
			faces++
		} else {
			still++
		}
	}
	return flow, faces, still
}

// HydroShare is the share of the full output a flow makes, in percent.
func HydroShare(flow int) int { return min(100, flow*100/HydroFull) }

// natureDue reports whether the generator looks around again: natureRecheck
// ticks passed since it last did.
func (st *State) natureDue() bool {
	return !st.nature.known || st.curTick-st.nature.at >= natureRecheck || st.curTick < st.nature.at
}

func (st *State) natureLooked() { st.nature.known, st.nature.at = true, st.curTick }

// runSolar makes the solar generator's energy of a tick.
func runSolar(s *core.Session, st *State, pos cube.Pos, tx *world.Tx) {
	defer network.Energy(s).AddSource(pos)
	if st.natureDue() {
		st.nature.sky = SkyOpen(tx, pos)
		st.natureLooked()
	}
	pct := 0
	if st.nature.sky {
		pct = climate.SunLight(s)
	}
	st.makeShare(pct)
}

// runHydro makes the hydro generator's energy of a tick.
func runHydro(s *core.Session, st *State, pos cube.Pos, tx *world.Tx) {
	defer network.Energy(s).AddSource(pos)
	if st.natureDue() {
		st.nature.flow, st.nature.faces, st.nature.still = HydroFlow(tx, pos)
		st.natureLooked()
	}
	st.makeShare(HydroShare(st.nature.flow))
}

// makeShare adds pct percent of the generator's full output to its buffer,
// carrying the fraction, while the buffer has room.
func (st *State) makeShare(pct int) {
	st.nature.pct = pct
	room := st.EnergyCap() - st.Energy
	if pct <= 0 || room <= 0 {
		return
	}
	st.nature.frac += st.GenRate() * pct
	e := min(room, st.nature.frac/100)
	st.nature.frac %= 100
	st.Energy += e
	st.made += e
	st.markWorked()
}

// NatureRate is the RE per tick the generator made in its last tick (with
// the fraction).
func (st *State) NatureRate() float64 {
	return float64(st.GenRate()*st.nature.pct) / 100
}

// IsNatureGenerator reports whether a kind is the solar or the hydro
// generator.
func IsNatureGenerator(kind string) bool {
	return kind == KindSolarGenerator || kind == KindHydroGenerator
}

// NatureLines are the lines of a solar or hydro generator's WAILA panel: its
// energy, what it makes and why it does not.
func NatureLines(lang string, st *State) []string {
	lines := []string{fmt.Sprintf(i18n.T(lang, "generator.lines.energy"), st.Energy, st.EnergyCap())}
	n := st.nature
	full := st.Energy >= st.EnergyCap()
	switch st.Def.ID {
	case idSolarGenerator:
		switch {
		case !n.known:
		case !n.sky:
			lines = append(lines, i18n.T(lang, "naturegen.lines.noSky"))
		case n.pct == 0:
			lines = append(lines, i18n.T(lang, "naturegen.lines.night"))
		case full:
			lines = append(lines, i18n.T(lang, "naturegen.lines.full"))
		default:
			lines = append(lines, fmt.Sprintf(i18n.T(lang, "naturegen.lines.making"), st.NatureRate()))
		}
		if n.known && n.sky && st.Sess != nil {
			lines = append(lines, solarWeatherLine(lang, st.Sess))
		}
	case idHydroGenerator:
		switch {
		case !n.known:
		case n.flow == 0 && n.still > 0:
			lines = append(lines, i18n.T(lang, "naturegen.lines.stillWater"))
		case n.flow == 0:
			lines = append(lines, i18n.T(lang, "naturegen.lines.noWater"))
		case full:
			lines = append(lines, i18n.T(lang, "naturegen.lines.full"))
		default:
			lines = append(lines, fmt.Sprintf(i18n.T(lang, "naturegen.lines.making"), st.NatureRate()))
		}
		if n.flow > 0 {
			lines = append(lines, fmt.Sprintf(i18n.T(lang, "naturegen.lines.flow"), n.faces, HydroShare(n.flow)))
		}
	}
	return lines
}

// solarWeatherLine is the sunlight the solar generator gets: its share, and
// the weather when it dims it.
func solarWeatherLine(lang string, s *core.Session) string {
	now, _, _ := s.Clock.Now()
	line := fmt.Sprintf(i18n.T(lang, "naturegen.lines.sun"), climate.SunLight(s))
	if w := climate.WeatherNow(s, now); w != "" {
		line += fmt.Sprintf(i18n.T(lang, "naturegen.lines.weather"), i18n.T(lang, "fairy.weather."+climate.WeatherKey(w)), climate.WeatherLight(w))
	}
	return line
}
