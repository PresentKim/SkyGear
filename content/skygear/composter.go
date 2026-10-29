package skygear

import (
	"fmt"
	"github.com/presentkim/skygear/common/i18n"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
)

// The composter is the vanilla block with the game's own rules: every plant
// item counts as one, no chance, and the 32nd fills it; a full one makes one
// dirt after composterDelay. Counts are saved in State.Composters. The
// Dragonfly patch (block.ComposterVanillaRules) turns off vanilla hopper
// feeding so composterIO handles it.

const (
	// matterCapacity is how many plant items fill a composter or a barrel.
	matterCapacity = 32
	// composterLevels is the number of fill levels of the composter block;
	// the last one is a full composter. composterReadyLevel shows its dirt
	// is ready.
	composterLevels     = 7
	composterReadyLevel = 8
	// composterDelay is how long a full composter takes to make dirt.
	composterDelay = 30 * time.Second
)

func init() { block.ComposterVanillaRules = false }

// composterLevel is the fill level the block shows for a count of items.
func composterLevel(count int) int { return count * composterLevels / matterCapacity }

// plantMatter reports whether a held item can be put in a composter or a
// barrel: whatever vanilla composts.
func plantMatter(st item.Stack) bool {
	if st.Empty() {
		return false
	}
	c, ok := st.Item().(item.Compostable)
	return ok && c.CompostChance() > 0
}

// composterCount is how many items the composter at pos holds. A composter
// the save knows nothing about (an older world) holds what its level shows.
func (s *session) composterCount(pos cube.Pos, c block.Composter) int {
	if n, ok := s.state.Composters[cropKey(pos)]; ok {
		return n
	}
	return (c.Level*matterCapacity + composterLevels - 1) / composterLevels
}

func (s *session) setComposterCount(pos cube.Pos, n int) {
	if n <= 0 {
		delete(s.state.Composters, cropKey(pos))
		return
	}
	if s.state.Composters == nil {
		s.state.Composters = map[string]int{}
	}
	s.state.Composters[cropKey(pos)] = n
}

// useComposter handles a player using the composter at pos with held: a full
// composter gives dirt, otherwise a plant item is put in. It returns true
// if the click was used (nothing is placed or used on the block then).
func (s *session) useComposter(p *player.Player, tx *world.Tx, pos cube.Pos, held item.Stack) bool {
	c, ok := tx.Block(pos).(block.Composter)
	if !ok {
		return false
	}
	n := s.composterCount(pos, c)
	switch {
	case n >= matterCapacity:
		if dirt := s.emptyComposter(tx, pos, c); !dirt.Empty() {
			s.giveItem(p, tx, pos.Side(cube.FaceUp).Vec3Middle(), dirt)
			count(p, "compost_dirt", 1)
		}
		return true
	case plantMatter(held):
		if s.fillComposter(tx, pos, c, held, 1) == 0 {
			return false
		}
		count(p, "compost_fill", 1)
		if s.composterCount(pos, c) >= matterCapacity {
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
func (s *session) fillComposter(tx *world.Tx, pos cube.Pos, c block.Composter, st item.Stack, n int) int {
	have := s.composterCount(pos, c)
	n = min(n, matterCapacity-have)
	if n <= 0 || !plantMatter(st) {
		return 0
	}
	have += n
	s.setComposterCount(pos, have)
	level := composterLevel(have)
	if level != c.Level {
		tx.SetBlock(pos, block.Composter{Level: level}, nil)
	}
	tx.AddParticle(pos.Vec3(), particle.BoneMeal{})
	switch {
	case have >= matterCapacity:
		if s.state.ComposterReady == nil {
			s.state.ComposterReady = map[string]time.Duration{}
		}
		now, _, _ := s.Clock.Now()
		s.state.ComposterReady[cropKey(pos)] = now + composterDelay
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
func (s *session) composterReady(pos cube.Pos, c block.Composter) bool {
	if s.composterCount(pos, c) < matterCapacity {
		return false
	}
	at, ok := s.state.ComposterReady[cropKey(pos)]
	now, _, _ := s.Clock.Now()
	return !ok || now >= at
}

// tickComposters shows the dirt of the full composters whose time is up and
// forgets composters that are gone.
func (s *session) tickComposters(tx *world.Tx) {
	for key, at := range s.state.ComposterReady {
		pos, ok := parseCropKey(key)
		if !ok {
			delete(s.state.ComposterReady, key)
			continue
		}
		c, ok := tx.Block(pos).(block.Composter)
		if !ok {
			delete(s.state.ComposterReady, key)
			continue
		}
		if now, _, _ := s.Clock.Now(); now >= at && c.Level != composterReadyLevel {
			tx.SetBlock(pos, block.Composter{Level: composterReadyLevel}, nil)
			tx.PlaySound(pos.Vec3(), sound.ComposterReady{})
		}
	}
}

// emptyComposter takes the dirt out of a composter at pos and empties it; it
// returns the dirt (empty when the dirt is not ready).
func (s *session) emptyComposter(tx *world.Tx, pos cube.Pos, c block.Composter) item.Stack {
	if !s.composterReady(pos, c) {
		return item.Stack{}
	}
	s.setComposterCount(pos, 0)
	delete(s.state.ComposterReady, cropKey(pos))
	tx.SetBlock(pos, block.Composter{}, nil)
	tx.PlaySound(pos.Vec3(), sound.ComposterEmpty{})
	s.MarkDirty()
	return item.NewStack(block.Dirt{}, 1)
}

// composterIO is the itemio side of the composter (the vanilla block has no
// methods of ours): any face takes plant items, the bottom gives the dirt.
type composterIO struct{ s *session }

func (c composterIO) InsertItems(tx *world.Tx, pos cube.Pos, _ cube.Face, st item.Stack) int {
	comp, ok := tx.Block(pos).(block.Composter)
	if !ok {
		return 0
	}
	return c.s.fillComposter(tx, pos, comp, st, st.Count())
}

func (c composterIO) PeekItems(tx *world.Tx, pos cube.Pos, face cube.Face, want func(item.Stack) bool) item.Stack {
	comp, ok := tx.Block(pos).(block.Composter)
	if !ok || face != cube.FaceDown || !c.s.composterReady(pos, comp) {
		return item.Stack{}
	}
	dirt := item.NewStack(block.Dirt{}, 1)
	if want != nil && !want(dirt) {
		return item.Stack{}
	}
	return dirt
}

func (c composterIO) ExtractItems(tx *world.Tx, pos cube.Pos, face cube.Face, st item.Stack, n int) item.Stack {
	comp, ok := tx.Block(pos).(block.Composter)
	if !ok || face != cube.FaceDown || n < 1 || !st.Comparable(item.NewStack(block.Dirt{}, 1)) {
		return item.Stack{}
	}
	return c.s.emptyComposter(tx, pos, comp)
}

// giveItem gives p an item, dropping what does not fit at pos.
func (s *session) giveItem(p *player.Player, tx *world.Tx, pos mgl64.Vec3, it item.Stack) {
	if n, _ := p.Inventory().AddItem(it); n < it.Count() {
		s.DropAt(tx, pos, it.Grow(-n))
	}
}

// composterLines are the lines of the composter's WAILA panel.
func (s *session) composterLines(lang string, pos cube.Pos, c block.Composter) []string {
	n := s.composterCount(pos, c)
	if s.composterReady(pos, c) {
		return []string{i18n.T(lang, "composter.lines.done")}
	}
	if n >= matterCapacity {
		now, _, _ := s.Clock.Now()
		left := max(s.state.ComposterReady[cropKey(pos)]-now, 0).Round(time.Second)
		return []string{fmt.Sprintf(i18n.T(lang, "composter.lines.making"), int(left.Minutes()), int(left.Seconds())%60)}
	}
	return []string{fmt.Sprintf(i18n.T(lang, "composter.lines.plants"), n, matterCapacity)}
}

// breakComposter forgets a composter that is broken; a full one drops dirt.
func (s *session) breakComposter(pos cube.Pos, c block.Composter, drops *[]item.Stack) {
	if s.composterReady(pos, c) {
		*drops = append(*drops, item.NewStack(block.Dirt{}, 1))
	}
	s.setComposterCount(pos, 0)
	delete(s.state.ComposterReady, cropKey(pos))
}
