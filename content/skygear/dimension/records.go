package dimension

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// The island's records of the hunts: per kind of monster the fastest clear,
// the kills and the clears, and from them three stars (a clear, a clear in
// the star time, enough kills in all). The three stars of a monster let the
// next one be opened, for a fairy emerald. The first monster is open from the
// start.

// Record is what the save keeps of the hunts of one kind of monster.
type Record struct {
	// Open is whether the monster was opened (the first one always is).
	Open bool `json:"open,omitempty"`
	// Best is the fastest clear in seconds, 0 before the first.
	Best   int `json:"best,omitempty"`
	Kills  int `json:"kills,omitempty"`
	Clears int `json:"clears,omitempty"`
	// Given are the stars whose first-time reward was given (bit n: star n).
	Given int `json:"given,omitempty"`
}

type dimensionState struct {
	// Hunts are the records by mob.
	Hunts map[string]*Record `json:"hunts,omitempty"`
	// Pool are the essences waiting for the collectors, by mob
	// (collector.go).
	Pool map[string]float64 `json:"essence_pool,omitempty"`
	// Kits are the hunting kits by account (kit.go): encoded stacks, empty
	// where a slot is unset. Stash is what each hunter carried before it went
	// in, until it is given back.
	Kits  map[string][][]byte `json:"hunt_kits,omitempty"`
	Stash map[string]stash    `json:"hunt_stash,omitempty"`
}

var dimensionSaved = core.NewSaved[dimensionState](nil)

// RecordOf is the record of mob in the save of s.
func RecordOf(s *core.Session, mob string) *Record {
	st := dimensionSaved.Of(s)
	if st.Hunts == nil {
		st.Hunts = map[string]*Record{}
	}
	r, ok := st.Hunts[mob]
	if !ok {
		r = &Record{}
		st.Hunts[mob] = r
	}
	return r
}

// Stars are the three stars of a monster: cleared, cleared within its star
// time, killed often enough.
func Stars(s *core.Session, mob string) [3]bool {
	h, _ := huntOf(mob)
	r := RecordOf(s, mob)
	return [3]bool{r.Clears > 0, r.Best > 0 && r.Best <= h.Star2, r.Kills >= h.Star3}
}

// StarCount is how many of the stars of mob the save has.
func StarCount(s *core.Session, mob string) int {
	n := 0
	for _, ok := range Stars(s, mob) {
		if ok {
			n++
		}
	}
	return n
}

// huntIndex is mob's place in the order of the hunts, -1 when it has none.
func huntIndex(mob string) int {
	for i, h := range Hunts {
		if h.Mob == mob {
			return i
		}
	}
	return -1
}

// IsOpen reports whether mob may be hunted in the save of s.
func IsOpen(s *core.Session, mob string) bool {
	i := huntIndex(mob)
	return i == 0 || (i > 0 && RecordOf(s, mob).Open)
}

// CanOpen reports whether mob is closed and the monster before it has all
// three stars.
func CanOpen(s *core.Session, mob string) bool {
	i := huntIndex(mob)
	return i > 0 && !IsOpen(s, mob) && StarCount(s, Hunts[i-1].Mob) == 3
}

// Open opens mob for a fairy emerald from p's inventory.
func Open(s *core.Session, p *player.Player, mob string) bool {
	if !CanOpen(s, mob) || !core.TakePlain(p.Inventory(), sgitem.FairyEmerald{}, 1) {
		return false
	}
	RecordOf(s, mob).Open = true
	s.MarkDirty()
	return true
}

const (
	// killEssence is the percent chance a kill gives the monster's essence.
	killEssence = 5
	// firstClearEssence is how many essences the first clear gives.
	firstClearEssence = 8
	// starDust is how much fairy dust a star gives the first time.
	starDust = 2
)

// record puts a finished run in the island's records and works out its
// rewards: an essence for some kills, essences for the first clear, dust for
// each new star.
func record(s *core.Session, res *result) {
	rec := RecordOf(s, res.Mob)
	before := Stars(s, res.Mob)
	rec.Kills += res.Kills
	if res.How == endClear {
		if rec.Clears == 0 {
			res.Essences += firstClearEssence
			res.FirstClear = true
		}
		rec.Clears++
		if rec.Best == 0 || res.Seconds < rec.Best {
			rec.Best = res.Seconds
			res.NewBest = true
		}
	}
	after := Stars(s, res.Mob)
	for i := range after {
		if after[i] && !before[i] {
			res.NewStars++
		}
		if after[i] && rec.Given&(1<<i) == 0 {
			rec.Given |= 1 << i
			res.Dust += starDust
		}
	}
	s.MarkDirty()
}

// deliver gives the hunter the rewards of its run.
func deliver(s *core.Session, tx *world.Tx, p *player.Player, res result) {
	if res.Essences > 0 {
		s.GiveItem(p, tx, p.Position(), item.NewStack(sgitem.Essence{Mob: res.Mob}, res.Essences))
	}
	if res.Dust > 0 {
		s.GiveItem(p, tx, p.Position(), item.NewStack(sgitem.FairyDust{}, res.Dust))
	}
}
