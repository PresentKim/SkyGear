package sgitem

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/blockmodel"
)

// Compressed blocks are nine blocks of a kind crafted into one, nine of those
// into one of the next level, as the compressed blocks of the Overloaded and
// ExtraUtilities mods. One crafts back into nine. They keep the blocks that
// pile up (the cobblestone generator, the hammer) in few slots, and island
// chunks take compressed cobblestone as cobblestone (island.PayCobble).

// CompactBase is a kind of block that compresses, and how many levels it has.
type CompactBase struct {
	// Kind is the block name without "minecraft:", e.g. "cobblestone".
	Kind string
	// Block is the block nine of which make level 1.
	Block world.Item
	// Levels is the highest level (1 to CompactMaxLevel).
	Levels int
	// shovel: mined with a shovel (the soft blocks), else with a pickaxe.
	shovel bool
	// hardness is the hardness of the block itself.
	hardness float64
}

// CompactMaxLevel is the highest level of any compressed block: the slot
// pips show up to four (slotdeco.MaxTier).
const CompactMaxLevel = 4

// CompactBases are the blocks that compress. Cobblestone is the island's
// money and is made without end, so it has the most levels; gravel, sand and
// dirt are sieved away and need fewer.
var CompactBases = []CompactBase{
	{Kind: "cobblestone", Block: block.Cobblestone{}, Levels: 4, hardness: 2},
	{Kind: "gravel", Block: block.Gravel{}, Levels: 2, shovel: true, hardness: 0.6},
	{Kind: "sand", Block: block.Sand{}, Levels: 2, shovel: true, hardness: 0.5},
	{Kind: "dirt", Block: block.Dirt{}, Levels: 2, shovel: true, hardness: 0.5},
}

// CompactBaseOf is the base of a kind.
func CompactBaseOf(kind string) (CompactBase, bool) {
	for _, b := range CompactBases {
		if b.Kind == kind {
			return b, true
		}
	}
	return CompactBase{}, false
}

// CompactBlocks are every compressed block, by base, lowest level first.
func CompactBlocks() []CompactBlock {
	var out []CompactBlock
	for _, b := range CompactBases {
		for l := 1; l <= b.Levels; l++ {
			out = append(out, CompactBlock{Kind: b.Kind, Level: l})
		}
	}
	return out
}

var compactHashes = func() map[CompactBlock]uint64 {
	m := map[CompactBlock]uint64{}
	for _, c := range CompactBlocks() {
		m[c] = block.NextHash()
	}
	return m
}()

// CompactBlock is a compressed block of Kind at Level (1: nine blocks).
type CompactBlock struct {
	Kind  string
	Level int
}

// CompactWorth is how many base blocks one block of level is: 9 to the
// level, 1 for level 0 (the base block).
func CompactWorth(level int) int {
	n := 1
	for range level {
		n *= 9
	}
	return n
}

// Worth is how many base blocks the block is.
func (c CompactBlock) Worth() int { return CompactWorth(c.Level) }

// Below is the item nine of which make c: the block one level lower, or the
// base block for level 1.
func (c CompactBlock) Below() world.Item {
	if c.Level <= 1 {
		b, _ := CompactBaseOf(c.Kind)
		return b.Block
	}
	return CompactBlock{Kind: c.Kind, Level: c.Level - 1}
}

func (c CompactBlock) id() string { return fmt.Sprintf("compressed_%s_%d", c.Kind, c.Level) }

func (c CompactBlock) EncodeItem() (string, int16) { return "skygear:" + c.id(), 0 }

func (c CompactBlock) EncodeBlock() (string, map[string]any) { return "skygear:" + c.id(), nil }

func (c CompactBlock) Hash() (uint64, uint64) { return compactHashes[c], 0 }

func (CompactBlock) Model() world.BlockModel { return model.Solid{} }

func (c CompactBlock) Name() string {
	return i18n.T(i18n.Default, fmt.Sprintf("name.Compressed.%s.%d", c.Kind, c.Level))
}

func (CompactBlock) Category() category.Category { return category.Construction() }

func (c CompactBlock) Properties() customblock.Properties {
	return customblock.Properties{Cube: true, Textures: map[string]customblock.Material{"*": blockmodel.SolidMaterial("skygear_" + c.id())}}
}

// BreakInfo is that of the base block, harder each level; it drops itself.
func (c CompactBlock) BreakInfo() block.BreakInfo {
	b, _ := CompactBaseOf(c.Kind)
	h := b.hardness * (1 + 0.5*float64(c.Level))
	effective := blockmodel.EffectiveFor(item.TypePickaxe)
	harvestable := func(t item.Tool) bool { return t.ToolType() == item.TypePickaxe }
	if b.shovel {
		effective = blockmodel.ShovelEffective
		harvestable = func(item.Tool) bool { return true }
	}
	return block.BreakInfo{
		Hardness: h, BlastResistance: max(6, h*3),
		Harvestable: harvestable,
		Effective:   effective,
		Drops:       func(item.Tool, []item.Enchantment) []item.Stack { return []item.Stack{item.NewStack(c, 1)} },
	}
}

// CompactPayment is how a payment of base blocks is made from what an
// inventory holds, by level (0 the base block, 1 to CompactMaxLevel the
// compressed blocks).
type CompactPayment struct {
	// Take is how many of each level are taken.
	Take [CompactMaxLevel + 1]int
	// Change is how many of each level are given back: what a block broken
	// for the payment is worth over the cost, in the largest lower levels.
	Change [CompactMaxLevel + 1]int
}

// Net is what the inventory gains of each level (negative: loses).
func (p CompactPayment) Net() [CompactMaxLevel + 1]int {
	var out [CompactMaxLevel + 1]int
	for l := range out {
		out[l] = p.Change[l] - p.Take[l]
	}
	return out
}

// PlanCompactPayment plans paying cost base blocks from have (the count of
// each level held). The base blocks go first, then the compressed blocks
// from the lowest level, each only as far as it does not pay too much. When
// what is left is less than one block of the next level held, one of the
// lowest such block is broken and the difference comes back as change.
// ok is false when everything held is worth less than cost.
func PlanCompactPayment(have [CompactMaxLevel + 1]int, cost int) (CompactPayment, bool) {
	var p CompactPayment
	if cost <= 0 {
		return p, true
	}
	total := 0
	for l, n := range have {
		total += max(n, 0) * CompactWorth(l)
	}
	if total < cost {
		return p, false
	}
	rem := cost
	for l, n := range have {
		w := CompactWorth(l)
		t := min(max(n, 0), rem/w)
		p.Take[l] = t
		rem -= t * w
	}
	if rem == 0 {
		return p, true
	}
	// Every level with blocks left is now worth more than rem: break the
	// lowest one.
	for l, n := range have {
		if n-p.Take[l] <= 0 {
			continue
		}
		p.Take[l]++
		change := CompactWorth(l) - rem
		for j := l - 1; j >= 0; j-- {
			w := CompactWorth(j)
			p.Change[j] = change / w
			change %= w
		}
		break
	}
	return p, true
}
