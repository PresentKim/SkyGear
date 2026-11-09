package wand

import (
	"math/rand/v2"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// similarBlocks are the blocks the similar match treats as one (the mod's
// default list: the kinds of dirt).
var similarBlocks = map[string]bool{
	"minecraft:dirt": true, "minecraft:grass_block": true, "minecraft:coarse_dirt": true,
	"minecraft:podzol": true, "minecraft:mycelium": true, "minecraft:farmland": true,
	"minecraft:grass_path": true, "minecraft:dirt_with_roots": true,
}

// similarItems are the items of similarBlocks that are blocks, in a fixed order.
var similarItems = []string{"minecraft:dirt", "minecraft:grass_block", "minecraft:coarse_dirt", "minecraft:podzol", "minecraft:mycelium", "minecraft:dirt_with_roots"}

func blockName(b world.Block) string {
	name, _ := b.EncodeBlock()
	return name
}

func isAir(b world.Block) bool {
	_, ok := b.(block.Air)
	return ok
}

// matches reports whether a block behind a free spot counts as the clicked one.
func (o Options) matches(target, b world.Block) bool {
	switch o.Match {
	case MatchExact:
		return blockName(target) == blockName(b)
	case MatchAny:
		return !isAir(target) && !isAir(b)
	}
	n1, n2 := blockName(target), blockName(b)
	return n1 == n2 || (similarBlocks[n1] && similarBlocks[n2])
}

// Placeable reports whether the wand may place b: a block with a collision
// box that is not one of Sky Gear's own (machines, conduits, drawers keep
// state of their own), has no block entity and is not one half of a two
// block tall block.
func Placeable(b world.Block) bool {
	if b == nil || isAir(b) {
		return false
	}
	switch b.(type) {
	case world.Liquid, world.NBTer, block.WoodDoor, block.CopperDoor, block.Bed, block.DoubleFlower, block.DoubleTallGrass, block.Composter:
		return false
	case sgitem.CrushedBlock, sgitem.OreBlock:
		return true
	case world.CustomBlock:
		return false
	}
	return len(b.Model().BBox(cube.Pos{}, airSource{})) > 0
}

// airSource is a world of air, for the models that look at their neighbours
// (stairs, fences, panes).
type airSource struct{}

func (airSource) Block(cube.Pos) world.Block { return block.Air{} }

// blockItem is the item of the block b the wand would place, if it may.
func blockItem(it world.Item) (world.Block, bool) {
	if it == nil {
		return nil, false
	}
	b, ok := it.(world.Block)
	if !ok || !Placeable(b) {
		return nil, false
	}
	return b, true
}

// canonical returns the item of a block as it is in an inventory (a block
// placed in a world can hold state its item has not).
func canonical(b world.Block) (world.Item, bool) {
	it, ok := b.(world.Item)
	if !ok {
		return nil, false
	}
	name, meta := it.EncodeItem()
	out, ok := world.ItemByName(name, meta)
	return out, ok
}

// supply is where the blocks come from: the items in the order they are
// used and how many of each the player has (creative: no end).
type supply struct {
	items    []world.Item
	counts   []int
	creative bool
	random   *rand.Rand
}

// newSupply finds the blocks for a click on target: the block in the off
// hand, else the target's item and, unless the match is exact, the items of
// the blocks similar to it; or, at random, every block of the hotbar.
func newSupply(p *player.Player, target world.Block, o Options, r *rand.Rand) *supply {
	s := &supply{creative: p.GameMode().CreativeInventory(), random: r}
	_, off := p.HeldItems()
	add := func(it world.Item) {
		if _, ok := blockItem(it); !ok {
			return
		}
		for _, have := range s.items {
			if sameItem(have, it) {
				return
			}
		}
		n := countItem(p, it)
		if s.creative {
			n = 1 << 30
		}
		if n > 0 {
			s.items, s.counts = append(s.items, it), append(s.counts, n)
		}
	}
	switch {
	case o.Random:
		inv := p.Inventory()
		for slot := 0; slot < 9; slot++ {
			if st, err := inv.Item(slot); err == nil && !st.Empty() {
				add(st.Item())
			}
		}
		if !off.Empty() {
			add(off.Item())
		}
	case !off.Empty():
		if _, ok := blockItem(off.Item()); ok {
			add(off.Item())
			break
		}
		fallthrough
	default:
		if it, ok := canonical(target); ok {
			add(it)
		}
		if o.Match != MatchExact && similarBlocks[blockName(target)] {
			for _, name := range similarItems {
				if it, ok := world.ItemByName(name, 0); ok {
					add(it)
				}
			}
		}
	}
	return s
}

// draw uses the item peek gave.
func (s *supply) draw() (world.Item, bool) {
	var left []int
	for i, n := range s.counts {
		if n > 0 {
			left = append(left, i)
		}
	}
	if len(left) == 0 {
		return nil, false
	}
	i := left[0]
	s.counts[i]--
	return s.items[i], true
}

func sameItem(a, b world.Item) bool {
	return item.NewStack(a, 1).Comparable(item.NewStack(b, 1))
}

// countItem counts the items it p has in the inventory and the off hand.
func countItem(p *player.Player, it world.Item) int {
	want := item.NewStack(it, 1)
	n := 0
	for _, st := range p.Inventory().Slots() {
		if !st.Empty() && st.Comparable(want) {
			n += st.Count()
		}
	}
	if _, off := p.HeldItems(); !off.Empty() && off.Comparable(want) {
		n += off.Count()
	}
	return n
}

// takeItem takes one it from p: the inventory first, then the off hand.
func takeItem(p *player.Player, it world.Item) bool {
	want := item.NewStack(it, 1)
	cmp := func(st item.Stack) bool { return st.Comparable(want) }
	if p.Inventory().ContainsItemFunc(1, cmp) {
		return p.Inventory().RemoveItemFunc(1, cmp) == nil
	}
	held, off := p.HeldItems()
	if !off.Empty() && off.Comparable(want) {
		p.SetHeldItems(held, off.Grow(-1))
		return true
	}
	return false
}

// Placement is one block a click places.
type Placement struct {
	Pos   cube.Pos
	Block world.Block
	Item  world.Item
}

// toPlace is the block for item it placed in front of support: the support
// itself when it is the same block, so the copy faces the same way (a double
// slab gives a single one), else the item's block.
func toPlace(it world.Item, support world.Block) world.Block {
	b, _ := it.(world.Block)
	if blockName(b) != blockName(support) {
		return b
	}
	if s, ok := support.(block.Slab); ok && s.Double {
		s.Double = false
		return s
	}
	return support
}

// free reports whether b may be placed at pos: the spot is air or, if the
// options let it, a block b replaces, and nothing but items stands in it.
func free(tx *world.Tx, pos cube.Pos, b world.Block, o Options) bool {
	if !pos.OutOfBounds(tx.Range()) {
		cur := tx.Block(pos)
		if !isAir(cur) {
			r, ok := cur.(block.Replaceable)
			if o.NoReplace || !ok || !r.ReplaceableBy(b) {
				return false
			}
		}
		if _, ok := tx.Liquid(pos); ok && o.NoReplace {
			return false
		}
		return !obstructed(tx, pos, b)
	}
	return false
}

// obstructed reports whether a living thing stands where b would be at pos.
func obstructed(tx *world.Tx, pos cube.Pos, b world.Block) bool {
	boxes := b.Model().BBox(pos, tx)
	for e := range tx.EntitiesWithin(cube.Box(-2, -2, -2, 3, 3, 3).Translate(pos.Vec3())) {
		switch e.H().Type() {
		case entity.ItemType, entity.ArrowType, entity.ExperienceOrbType:
			continue
		}
		eb := e.H().Type().BBox(e).Translate(e.Position())
		for _, bb := range boxes {
			if eb.IntersectsWith(bb.Translate(pos.Vec3())) {
				return true
			}
		}
	}
	return false
}

// Plan finds the blocks a click on the face of the block at pos places, at
// most limit: from the spot in front of the clicked block, spreading as the
// lock allows to every spot in front of a matching block. can reports
// whether the player may build at a spot.
func Plan(tx *world.Tx, p *player.Player, pos cube.Pos, face cube.Face, o Options, limit int, can func(cube.Pos) bool, r *rand.Rand) []Placement {
	target := tx.Block(pos)
	offsets, start := o.spread(face)
	if !start || limit <= 0 {
		return nil
	}
	sup := newSupply(p, target, o, r)
	if len(sup.items) == 0 {
		return nil
	}
	if !o.Random {
		sup.random = nil
	}
	var out []Placement
	queue := []cube.Pos{pos.Side(face)}
	seen := map[cube.Pos]bool{}
	for len(queue) > 0 && len(out) < limit {
		at := queue[0]
		queue = queue[1:]
		support := tx.Block(at.Side(face.Opposite()))
		if seen[at] || !o.matches(target, support) {
			continue
		}
		seen[at] = true
		if !can(at) {
			continue
		}
		it, ok := sup.peek()
		if !ok {
			break
		}
		b := toPlace(it, support)
		if !free(tx, at, b, o) {
			continue
		}
		sup.draw()
		out = append(out, Placement{Pos: at, Block: b, Item: it})
		for _, off := range offsets {
			queue = append(queue, at.Add(off))
		}
	}
	return out
}

// peek is the item draw would give, without using it: for a random supply
// the item is drawn on peek and kept for the next draw.
func (s *supply) peek() (world.Item, bool) {
	if s.random == nil {
		for i, n := range s.counts {
			if n > 0 {
				return s.items[i], true
			}
		}
		return nil, false
	}
	return s.drawRandom()
}

// drawRandom draws a random item and puts it first, so draw takes it.
func (s *supply) drawRandom() (world.Item, bool) {
	var left []int
	for i, n := range s.counts {
		if n > 0 {
			left = append(left, i)
		}
	}
	if len(left) == 0 {
		return nil, false
	}
	i := left[s.random.IntN(len(left))]
	s.items[0], s.items[i] = s.items[i], s.items[0]
	s.counts[0], s.counts[i] = s.counts[i], s.counts[0]
	return s.items[0], true
}
