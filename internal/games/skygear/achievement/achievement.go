// Package achievement lists the game's account achievements: the counter and
// first-possession catalogs and the IDs of the other achievements. The game
// package registers and grants them.
package achievement

import (
	"fmt"
)

// Achievements of the game. Drafts.
const (
	FirstMachine  = "sg_first_machine"
	Chunks10      = "sg_chunks_10"
	FirstResearch = "sg_research_first"
)

// HaveDef is an item whose first possession is an achievement.
type HaveDef struct{ Item, Short string }

// achievements without a counter: the whole research tree and the whole
// advancement tree done.
const (
	AllResearch     = "sg_all_research"
	AllAdvancements = "sg_all_advancements"
)

// CounterDef is one counter: its key, the group its achievements are
// listed in and the counts they are granted at.
type CounterDef struct {
	Key, Group string
	Needs      []int64
}

// CounterID is the achievement a counter grants at a count.
func CounterID(key string, need int64) string { return fmt.Sprintf("sg_%s_%d", key, need) }

// Group is the name of a group of achievements.
func Group(g string) string { return "achgroup." + g }

// Achievements of the account that the game's milestones give; they open
// titles and, by their number, name colours.
const (
	Wave1  = "sg_wave_1"
	Wave8  = "sg_wave_8"
	Tinker = "sg_tinker"
	Bloom  = "sg_bloom"
	Dragon = "sg_dragon"
)
