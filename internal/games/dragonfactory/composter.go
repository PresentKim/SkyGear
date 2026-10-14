package dragonfactory

import (
	"fmt"

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
		s.giveItem(p, tx, pos.Side(cube.FaceUp).Vec3Middle(), item.NewStack(block.Dirt{}, 1))
		s.setComposterCount(pos, 0)
		tx.SetBlock(pos, block.Composter{}, nil)
		tx.PlaySound(pos.Vec3(), sound.ComposterEmpty{})
		s.storeState()
		return true
	case plantMatter(held):
		n++
		s.setComposterCount(pos, n)
		level := composterLevel(n)
		if level != c.Level {
			tx.SetBlock(pos, block.Composter{Level: level}, nil)
		}
		tx.AddParticle(pos.Vec3(), particle.BoneMeal{})
		switch {
		case n >= matterCapacity:
			tx.PlaySound(pos.Vec3(), sound.ComposterReady{})
			p.SendTip("§a퇴비가 가득 찼습니다. 우클릭하면 흙이 나옵니다.")
		case level != c.Level:
			tx.PlaySound(pos.Vec3(), sound.ComposterFillLayer{})
		default:
			tx.PlaySound(pos.Vec3(), sound.ComposterFill{})
		}
		if !p.GameMode().CreativeInventory() {
			_, off := p.HeldItems()
			p.SetHeldItems(held.Grow(-1), off)
		}
		s.storeState()
		return true
	}
	return false
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
		return []string{"§a가득 참: 우클릭하면 흙"}
	}
	return []string{fmt.Sprintf("§f식물 §e%d§7/%d", n, matterCapacity), "§7풀·잎·묘목·씨앗·작물을 넣으세요"}
}

// breakComposter forgets a composter that is broken; a full one drops dirt.
func (s *session) breakComposter(pos cube.Pos, c block.Composter, drops *[]item.Stack) {
	if s.composterCount(pos, c) >= matterCapacity {
		*drops = append(*drops, item.NewStack(block.Dirt{}, 1))
	}
	s.setComposterCount(pos, 0)
}
