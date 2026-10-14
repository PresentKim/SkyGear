package dragonfactory

import (
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

// Hand-made resources of the start (docs/design/02-dragonfactory.md §3.3,
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

// Pebble kinds.
const (
	PebbleStone    = "stone"
	PebbleAndesite = "andesite"
	PebbleDiorite  = "diorite"
	PebbleGranite  = "granite"
)

// pebbleKinds are the pebble kinds with the drop weight of each and the
// block four of them make.
var pebbleKinds = []struct {
	kind   string
	name   string
	weight int
	block  world.Item
}{
	{PebbleStone, "돌 돌멩이", 70, block.Cobblestone{}},
	{PebbleAndesite, "안산암 돌멩이", 10, block.Andesite{}},
	{PebbleDiorite, "섬록암 돌멩이", 10, block.Diorite{}},
	{PebbleGranite, "화강암 돌멩이", 10, block.Granite{}},
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
	return "돌멩이"
}
func (Pebble) Category() category.Category { return category.Items() }

// BarkFiber is stripped from logs; four make string.
type BarkFiber struct{}

func (BarkFiber) EncodeItem() (string, int16) { return "dragonfactory:bark_fiber", 0 }
func (BarkFiber) Name() string                { return "나무껍질 섬유" }
func (BarkFiber) Category() category.Category { return category.Items() }

// Dust is sand crushed once more (hammer, crusher), sifted for its drops.
type Dust struct{}

var dustHash = block.NextHash()

func (Dust) EncodeItem() (string, int16)           { return "dragonfactory:dust", 0 }
func (Dust) EncodeBlock() (string, map[string]any) { return "dragonfactory:dust", nil }
func (Dust) Hash() (uint64, uint64)                { return dustHash, 0 }
func (Dust) Model() world.BlockModel               { return model.Solid{} }
func (Dust) Name() string                          { return "먼지" }
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
		return false
	}
	if s.stickDebounced(p) {
		return true
	}
	// Every use knocks a pebble out, with dirt chips and the small
	// breaking sound of the block.
	b := tx.Block(pos)
	tx.AddParticle(pos.Side(face).Vec3Centre(), particle.PunchBlock{Block: b, Face: face})
	tx.AddParticle(pos.Vec3Centre(), particle.BlockBreak{Block: b})
	tx.PlaySound(pos.Vec3Centre(), sound.BlockBreaking{Block: b})
	s.dropAt(tx, pos.Side(face).Vec3Centre(), item.NewStack(s.rollPebble(), 1))
	return true
}

// stripFiber drops bark fibers by chance when a stone or better axe strips
// a log (Dragonfly strips it after this).
func (s *session) stripFiber(tx *world.Tx, pos cube.Pos, face cube.Face, held item.Stack) {
	axe, ok := held.Item().(item.Axe)
	if !ok || axe.Tier.HarvestLevel < item.ToolTierStone.HarvestLevel {
		return
	}
	if l, ok := tx.Block(pos).(block.Log); !ok || l.Stripped {
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
	f := item.NewStack(BarkFiber{}, 1)
	recipe.Register(recipe.NewShapeless([]recipe.Item{f, f, f, f}, item.NewStack(block.String{}, 1), "crafting_table"))
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
