package dragonfactory

import (
	"fmt"
	"github.com/presentkim/dragonfactory/internal/i18n"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
)

// The composter (docs/design/02-dragonfactory.md §3.6) is the vanilla block
// with the game's own rules: no chance, no bone meal. Every plant item put
// in (anything vanilla composts: leaves, saplings, seeds, crops, ... and the
// bark fiber) counts as one, and the 32nd fills it. The seven fill levels
// of the block are at fixed counts (the level is count * 7 / 32), and using
// a full composter gives one dirt and empties it. The count of each
// composter is in the save (State.Composters, by position); hoppers do not
// feed it (Dragonfly patch block.ComposterVanillaRules).

const (
	// matterCapacity is how many plant items fill a composter or a barrel.
	matterCapacity = 32
	// composterLevels is the number of fill levels of the composter block;
	// the last one is a full composter.
	composterLevels = 7
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
		s.giveItem(p, tx, pos.Side(cube.FaceUp).Vec3Middle(), s.emptyComposter(tx, pos, c))
		return true
	case plantMatter(held):
		if s.fillComposter(tx, pos, c, held, 1) == 0 {
			return false
		}
		if s.composterCount(pos, c) >= matterCapacity {
			p.SendTip(i18n.T(i18n.Of(p), "dragonfactory.composter.useComposter.1"))
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
		tx.PlaySound(pos.Vec3(), sound.ComposterReady{})
	case level != c.Level:
		tx.PlaySound(pos.Vec3(), sound.ComposterFillLayer{})
	default:
		tx.PlaySound(pos.Vec3(), sound.ComposterFill{})
	}
	s.storeState()
	return n
}

// emptyComposter takes the dirt out of a full composter at pos and empties
// it; it returns the dirt (empty when the composter is not full).
func (s *session) emptyComposter(tx *world.Tx, pos cube.Pos, c block.Composter) item.Stack {
	if s.composterCount(pos, c) < matterCapacity {
		return item.Stack{}
	}
	s.setComposterCount(pos, 0)
	tx.SetBlock(pos, block.Composter{}, nil)
	tx.PlaySound(pos.Vec3(), sound.ComposterEmpty{})
	s.storeState()
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
	if !ok || face != cube.FaceDown || c.s.composterCount(pos, comp) < matterCapacity {
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
		s.dropAt(tx, pos, it.Grow(n-it.Count()))
	}
}

// composterLines are the lines of the composter's WAILA panel.
func (s *session) composterLines(pos cube.Pos, c block.Composter) []string {
	n := s.composterCount(pos, c)
	if n >= matterCapacity {
		return []string{i18n.T(i18n.Default, "dragonfactory.composter.composterLines.1")}
	}
	return []string{fmt.Sprintf(i18n.T(i18n.Default, "dragonfactory.composter.composterLines.2"), n, matterCapacity), i18n.T(i18n.Default, "dragonfactory.composter.composterLines.3")}
}

// breakComposter forgets a composter that is broken; a full one drops dirt.
func (s *session) breakComposter(pos cube.Pos, c block.Composter, drops *[]item.Stack) {
	if s.composterCount(pos, c) >= matterCapacity {
		*drops = append(*drops, item.NewStack(block.Dirt{}, 1))
	}
	s.setComposterCount(pos, 0)
}
