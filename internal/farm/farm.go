// Package farm contains farming progression: harvest detection, farming
// levels, perks and crop sell prices.
package farm

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

// Harvest describes a fully grown crop that was broken.
type Harvest struct {
	// Name is a stable key for statistics.
	Name string
	XP   int
	// Crop is the main produce, used for bonus drops.
	Crop world.Item
	// Seed is the item consumed from the drops to replant the crop, and Replant
	// the block placed back. Replant is nil for crops that can't be replanted.
	Seed    world.Item
	Replant world.Block
}

// HarvestOf returns the harvest information if b is a fully grown crop.
// Only age based crops count, so placing and breaking blocks can't farm XP.
func HarvestOf(b world.Block) (Harvest, bool) {
	switch c := b.(type) {
	case block.WheatSeeds:
		if c.Growth == 7 {
			return Harvest{Name: "wheat", XP: 2, Crop: item.Wheat{}, Seed: block.WheatSeeds{}, Replant: block.WheatSeeds{}}, true
		}
	case block.Carrot:
		if c.Growth == 7 {
			return Harvest{Name: "carrot", XP: 2, Crop: block.Carrot{}, Seed: block.Carrot{}, Replant: block.Carrot{}}, true
		}
	case block.Potato:
		if c.Growth == 7 {
			return Harvest{Name: "potato", XP: 2, Crop: block.Potato{}, Seed: block.Potato{}, Replant: block.Potato{}}, true
		}
	case block.BeetrootSeeds:
		if c.Growth == 7 {
			return Harvest{Name: "beetroot", XP: 3, Crop: item.Beetroot{}, Seed: block.BeetrootSeeds{}, Replant: block.BeetrootSeeds{}}, true
		}
	case block.NetherWart:
		if c.Age == 3 {
			return Harvest{Name: "nether_wart", XP: 4, Crop: block.NetherWart{}, Seed: block.NetherWart{}, Replant: block.NetherWart{}}, true
		}
	case block.CocoaBean:
		if c.Age == 2 {
			return Harvest{Name: "cocoa", XP: 3, Crop: block.CocoaBean{}, Seed: block.CocoaBean{}, Replant: block.CocoaBean{Facing: c.Facing}}, true
		}
	}
	return Harvest{}, false
}

// MaxLevel is the highest farming level.
const MaxLevel = 50

// XPForLevel returns the total XP required to reach level l.
func XPForLevel(l int) int {
	// Level 1: 0, level 2: 50, level 3: 150, level 4: 300 ...
	return 25 * l * (l - 1)
}

// Level returns the farming level for a total amount of XP, starting at 1.
func Level(xp int) int {
	l := 1
	for l < MaxLevel && xp >= XPForLevel(l+1) {
		l++
	}
	return l
}

// BonusChance is the chance a harvest yields an extra crop at level l.
func BonusChance(l int) float64 { return float64(l) * 0.02 }

// ReplantLevel is the level at which auto replanting is unlocked.
const ReplantLevel = 5
