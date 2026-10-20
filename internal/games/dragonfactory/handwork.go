package dragonfactory

import (
	"github.com/presentkim/dragonfactory/common/i18n"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/df-mc/dragonfly/server/world/sound"
)

// Hand-made resources of the start (docs/design/02-dragonfactory.md §3,
// 05-sky-progression.md §3). Chances are drafts.
//   - A stick used on grass or dirt always drops a pebble; four pebbles
//     make the stone they come from (research "pebbles").
//   - Stripping a log with a stone or better axe sometimes drops one or two
//     bark fibers; four fibers make string (research "string").
//   - Dust is sand crushed once more, sifted for its own drops.

const (
	// stickCooldown is the shortest time between two stick uses on the
	// ground (the client sends one click twice).
	stickCooldown = 250 * time.Millisecond
	// fiberChance is the percent chance stripping a log drops fibers.
	fiberChance = 50
)

// Pebble kinds. Other stones (andesite, diorite, granite) are added here
// when something uses them.
const (
	PebbleStone = "stone"
)

// pebbleKinds are the pebble kinds with the drop weight of each and the
// block four of them make.
var pebbleKinds = []struct {
	kind   string
	name   string
	weight int
	block  world.Item
}{
	{PebbleStone, i18n.T(i18n.Default, "dragonfactory.handwork.init.1"), 1, block.Cobblestone{}},
}

// Pebble is a small stone a stick knocks out of grass and dirt.
type Pebble struct{ Kind string }

func (p Pebble) EncodeItem() (string, int16) { return "dragonfactory:pebble_" + p.Kind, 0 }
func (p Pebble) Name() string {
	for _, k := range pebbleKinds {
		if k.kind == p.Kind {
			return k.name
		}
	}
	return i18n.T(i18n.Default, "name.Pebble")
}
func (Pebble) Category() category.Category { return category.Items() }

// BarkFiber is stripped from logs; three in a row make three string.
type BarkFiber struct{}

func (BarkFiber) EncodeItem() (string, int16) { return "dragonfactory:bark_fiber", 0 }
func (BarkFiber) Name() string                { return i18n.T(i18n.Default, "name.BarkFiber") }
func (BarkFiber) Category() category.Category { return category.Items() }

// CompostChance makes a fiber plant matter for the composter and the barrel
// (composter.go; the chance itself is not used, only that it is above 0).
func (BarkFiber) CompostChance() float64 { return 0.3 }

// Dust is sand crushed once more (hammer, crusher), sifted for its drops.
type Dust struct{}

var dustHash = block.NextHash()

func (Dust) EncodeItem() (string, int16)           { return "dragonfactory:dust", 0 }
func (Dust) EncodeBlock() (string, map[string]any) { return "dragonfactory:dust", nil }
func (Dust) Hash() (uint64, uint64)                { return dustHash, 0 }
func (Dust) Model() world.BlockModel               { return model.Solid{} }
func (Dust) Name() string                          { return i18n.T(i18n.Default, "name.Dust") }
func (Dust) Category() category.Category           { return category.Construction() }
func (Dust) Properties() customblock.Properties {
	return customblock.Properties{Cube: true, Textures: map[string]customblock.Material{"*": alphaMaterial("dragonfactory_dust")}}
}
func (d Dust) BreakInfo() block.BreakInfo {
	return block.BreakInfo{
		Hardness: 0.5, BlastResistance: 0.5,
		Harvestable: func(item.Tool) bool { return true },
		Effective:   customBlockEffective,
		Drops:       func(item.Tool, []item.Enchantment) []item.Stack { return []item.Stack{item.NewStack(d, 1)} },
	}
}

// rollPebble picks a pebble kind by weight.
func (s *session) rollPebble() Pebble {
	total := 0
	for _, k := range pebbleKinds {
		total += k.weight
	}
	n := s.rand.IntN(total)
	for _, k := range pebbleKinds {
		if n < k.weight {
			return Pebble{Kind: k.kind}
		}
		n -= k.weight
	}
	return Pebble{Kind: PebbleStone}
}

// stickDebounced reports whether p used a stick less than stickCooldown
// ago, and otherwise records this use.
func (s *session) stickDebounced(p *player.Player) bool {
	if s.lastStick == nil {
		s.lastStick = map[*world.EntityHandle]time.Time{}
	}
	if t, ok := s.lastStick[p.H()]; ok && time.Since(t) < stickCooldown {
		return true
	}
	s.lastStick[p.H()] = time.Now()
	return false
}

// useStick knocks a pebble out of grass or dirt. It reports
// whether the use was a stick on grass or dirt.
func (s *session) useStick(p *player.Player, tx *world.Tx, pos cube.Pos, face cube.Face, held item.Stack) bool {
	if _, ok := held.Item().(item.Stick); !ok {
		return false
	}
	switch tx.Block(pos).(type) {
	case block.Grass, block.Dirt:
	default:
		if !s.stickDebounced(p) {
			count(p, "stick_other", 1)
		}
		return false
	}
	if s.stickDebounced(p) {
		return true
	}
	if _, dirt := tx.Block(pos).(block.Dirt); dirt {
		count(p, "stick_dirt", 1)
	} else {
		count(p, "stick_grass", 1)
	}
	// Every use knocks a pebble out, with dirt chips and the small
	// breaking sound of the block.
	b := tx.Block(pos)
	tx.AddParticle(pos.Side(face).Vec3Centre(), particle.PunchBlock{Block: b, Face: face})
	tx.AddParticle(pos.Vec3Centre(), particle.BlockBreak{Block: b})
	tx.PlaySound(pos.Vec3Centre(), sound.BlockBreaking{Block: b})
	pebble := s.rollPebble()
	count(p, "pebble_"+pebble.Kind, 1)
	s.dropAt(tx, pos.Side(face).Vec3Centre(), item.NewStack(pebble, 1))
	return true
}

// stripFiber drops bark fibers by chance when a stone or better axe strips
// a log (Dragonfly strips it after this).
func (s *session) stripFiber(p *player.Player, tx *world.Tx, pos cube.Pos, face cube.Face, held item.Stack) {
	axe, ok := held.Item().(item.Axe)
	if !ok {
		return
	}
	if l, ok := tx.Block(pos).(block.Log); !ok || l.Stripped {
		return
	}
	count(p, "strip_log", 1)
	if axe.Tier.HarvestLevel < item.ToolTierStone.HarvestLevel {
		return
	}
	if s.rand.IntN(100) < fiberChance {
		s.dropAt(tx, pos.Side(face).Vec3Centre(), item.NewStack(BarkFiber{}, 1+s.rand.IntN(2)))
	}
}

// handworkRecipes registers the recipes of the hand-made resources (locked
// until researched like every other recipe).
func handworkRecipes() {
	for _, k := range pebbleKinds {
		p := item.NewStack(Pebble{Kind: k.kind}, 1)
		recipe.Register(recipe.NewShapeless([]recipe.Item{p, p, p, p}, item.NewStack(k.block, 1), "crafting_table"))
	}
	// Three fibers in a row make three string. Three wide, it does not fit
	// the 2x2 grid of the inventory: only a crafting table makes string.
	f := item.NewStack(BarkFiber{}, 1)
	recipe.Register(recipe.NewShaped([]recipe.Item{f, f, f}, item.NewStack(block.String{}, 3), recipe.NewShape(3, 1), "crafting_table"))
}

func init() {
	for _, k := range pebbleKinds {
		world.RegisterItem(Pebble{Kind: k.kind})
	}
	world.RegisterItem(BarkFiber{})
	world.RegisterBlock(Dust{})
	world.RegisterItem(Dust{})
	handworkRecipes()
}
