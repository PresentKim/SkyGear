package farm

import (
	"fmt"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/core"
)

// The composter is the vanilla block with the game's own rules: every plant
// item counts as one, no chance, and the 32nd fills it; a full one makes one
// dirt after composterDelay. Counts are saved in State.Composters. The
// Dragonfly patch (block.ComposterVanillaRules) turns off vanilla hopper
// feeding so composterIO handles it.

func init() { block.ComposterVanillaRules = false }

const (
	// matterCapacity is how many plant items fill a composter or a barrel.
	MatterCapacity = 32
	// composterLevels is the number of fill levels of the composter block;
	// the last one is a full composter. composterReadyLevel shows its dirt
	// is ready.
	ComposterLevels     = 7
	ComposterReadyLevel = 8
	// composterDelay is how long a full composter takes to make dirt.
	composterDelay = 30 * time.Second
)

// ComposterLevel is the fill level the block shows for a count of items.
func ComposterLevel(count int) int { return count * ComposterLevels / MatterCapacity }

// PlantMatter reports whether a held item can be put in a composter or a
// barrel: whatever vanilla composts.
func PlantMatter(st item.Stack) bool {
	if st.Empty() {
		return false
	}
	c, ok := st.Item().(item.Compostable)
	return ok && c.CompostChance() > 0
}

// ComposterCount is how many items the composter at pos holds. A composter
// the save knows nothing about (an older world) holds what its level shows.
func ComposterCount(s *core.Session, pos cube.Pos, c block.Composter) int {
	if n, ok := StateOf(s).Composters[CropKey(pos)]; ok {
		return n
	}
	return (c.Level*MatterCapacity + ComposterLevels - 1) / ComposterLevels
}

func SetComposterCount(s *core.Session, pos cube.Pos, n int) {
	if n <= 0 {
		delete(StateOf(s).Composters, CropKey(pos))
		return
	}
	if StateOf(s).Composters == nil {
		StateOf(s).Composters = map[string]int{}
	}
	StateOf(s).Composters[CropKey(pos)] = n
}

// UseComposter handles a player using the composter at pos with held: a full
// composter gives dirt, otherwise a plant item is put in. It returns true
// if the click was used (nothing is placed or used on the block then).
func UseComposter(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, held item.Stack) bool {
	c, ok := tx.Block(pos).(block.Composter)
	if !ok {
		return false
	}
	n := ComposterCount(s, pos, c)
	switch {
	case n >= MatterCapacity:
		if dirt := emptyComposter(s, tx, pos, c); !dirt.Empty() {
			s.GiveItem(p, tx, pos.Side(cube.FaceUp).Vec3Middle(), dirt)
			achievement.Count(p, "compost_dirt", 1)
		}
		return true
	case PlantMatter(held):
		if fillComposter(s, tx, pos, c, held, 1) == 0 {
			return false
		}
		achievement.Count(p, "compost_fill", 1)
		if ComposterCount(s, pos, c) >= MatterCapacity {
			p.SendTip(i18n.T(i18n.Of(p), "composter.full"))
		}
		if !p.GameMode().CreativeInventory() {
			_, off := p.HeldItems()
			p.SetHeldItems(held.Grow(-1), off)
		}
		return true
	}
	return false
}

// fillComposter puts up to n plant items of st into the composter at pos
// (whose block is c) and returns how many it took: it shows the new level
// and plays the fill sounds.
func fillComposter(s *core.Session, tx *world.Tx, pos cube.Pos, c block.Composter, st item.Stack, n int) int {
	have := ComposterCount(s, pos, c)
	n = min(n, MatterCapacity-have)
	if n <= 0 || !PlantMatter(st) {
		return 0
	}
	have += n
	SetComposterCount(s, pos, have)
	level := ComposterLevel(have)
	if level != c.Level {
		tx.SetBlock(pos, block.Composter{Level: level}, nil)
	}
	tx.AddParticle(pos.Vec3(), particle.BoneMeal{})
	switch {
	case have >= MatterCapacity:
		if StateOf(s).ComposterReady == nil {
			StateOf(s).ComposterReady = map[string]time.Duration{}
		}
		now, _, _ := s.Clock.Now()
		StateOf(s).ComposterReady[CropKey(pos)] = now + composterDelay
		tx.PlaySound(pos.Vec3(), sound.ComposterFillLayer{})
	case level != c.Level:
		tx.PlaySound(pos.Vec3(), sound.ComposterFillLayer{})
	default:
		tx.PlaySound(pos.Vec3(), sound.ComposterFill{})
	}
	s.MarkDirty()
	return n
}

// composterReady reports whether the composter at pos holds ready dirt: full
// and its time is up (a full one the save has no time for is ready).
func composterReady(s *core.Session, pos cube.Pos, c block.Composter) bool {
	if ComposterCount(s, pos, c) < MatterCapacity {
		return false
	}
	at, ok := StateOf(s).ComposterReady[CropKey(pos)]
	now, _, _ := s.Clock.Now()
	return !ok || now >= at
}

// TickComposters shows the dirt of the full composters whose time is up and
// forgets composters that are gone.
func TickComposters(s *core.Session, tx *world.Tx) {
	for key, at := range StateOf(s).ComposterReady {
		pos, ok := parseCropKey(key)
		if !ok {
			delete(StateOf(s).ComposterReady, key)
			continue
		}
		c, ok := tx.Block(pos).(block.Composter)
		if !ok {
			delete(StateOf(s).ComposterReady, key)
			continue
		}
		if now, _, _ := s.Clock.Now(); now >= at && c.Level != ComposterReadyLevel {
			tx.SetBlock(pos, block.Composter{Level: ComposterReadyLevel}, nil)
			tx.PlaySound(pos.Vec3(), sound.ComposterReady{})
		}
	}
}

// emptyComposter takes the dirt out of a composter at pos and empties it; it
// returns the dirt (empty when the dirt is not ready).
func emptyComposter(s *core.Session, tx *world.Tx, pos cube.Pos, c block.Composter) item.Stack {
	if !composterReady(s, pos, c) {
		return item.Stack{}
	}
	SetComposterCount(s, pos, 0)
	delete(StateOf(s).ComposterReady, CropKey(pos))
	tx.SetBlock(pos, block.Composter{}, nil)
	tx.PlaySound(pos.Vec3(), sound.ComposterEmpty{})
	s.MarkDirty()
	return item.NewStack(block.Dirt{}, 1)
}

// ComposterIO is the itemio side of the composter (the vanilla block has no
// methods of ours): any face takes plant items, the bottom gives the dirt.
type ComposterIO struct{ S *core.Session }

func (c ComposterIO) InsertItems(tx *world.Tx, pos cube.Pos, _ cube.Face, st item.Stack) int {
	comp, ok := tx.Block(pos).(block.Composter)
	if !ok {
		return 0
	}
	return fillComposter(c.S, tx, pos, comp, st, st.Count())
}

func (c ComposterIO) PeekItems(tx *world.Tx, pos cube.Pos, face cube.Face, want func(item.Stack) bool) item.Stack {
	comp, ok := tx.Block(pos).(block.Composter)
	if !ok || face != cube.FaceDown || !composterReady(c.S, pos, comp) {
		return item.Stack{}
	}
	dirt := item.NewStack(block.Dirt{}, 1)
	if want != nil && !want(dirt) {
		return item.Stack{}
	}
	return dirt
}

func (c ComposterIO) ExtractItems(tx *world.Tx, pos cube.Pos, face cube.Face, st item.Stack, n int) item.Stack {
	comp, ok := tx.Block(pos).(block.Composter)
	if !ok || face != cube.FaceDown || n < 1 || !st.Comparable(item.NewStack(block.Dirt{}, 1)) {
		return item.Stack{}
	}
	return emptyComposter(c.S, tx, pos, comp)
}

// ComposterLines are the lines of the composter's WAILA panel.
func ComposterLines(s *core.Session, lang string, pos cube.Pos, c block.Composter) []string {
	n := ComposterCount(s, pos, c)
	if composterReady(s, pos, c) {
		return []string{i18n.T(lang, "composter.lines.done")}
	}
	if n >= MatterCapacity {
		now, _, _ := s.Clock.Now()
		left := max(StateOf(s).ComposterReady[CropKey(pos)]-now, 0).Round(time.Second)
		return []string{fmt.Sprintf(i18n.T(lang, "composter.lines.making"), int(left.Minutes()), int(left.Seconds())%60)}
	}
	return []string{fmt.Sprintf(i18n.T(lang, "composter.lines.plants"), n, MatterCapacity)}
}

// BreakComposter forgets a composter that is broken; a full one drops dirt.
func BreakComposter(s *core.Session, pos cube.Pos, c block.Composter, drops *[]item.Stack) {
	if composterReady(s, pos, c) {
		*drops = append(*drops, item.NewStack(block.Dirt{}, 1))
	}
	SetComposterCount(s, pos, 0)
	delete(StateOf(s).ComposterReady, CropKey(pos))
}
