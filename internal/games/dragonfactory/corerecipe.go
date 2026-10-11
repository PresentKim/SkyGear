package dragonfactory

import (
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

// Core recipes (docs/design/02-dragonfactory.md §3.1): the block under the
// core (its base) is the block it makes above itself. The base is not used
// up. Since the 13th playtest the core only makes the basic resources; the
// rest comes from crushing and sifting, recipes from research.

// CoreRecipe is one thing the core can make.
type CoreRecipe struct {
	ID string
	// Base is the block under the core.
	Base world.Block
	// Output is the block made above the core.
	Output world.Block
}

var oakLog = block.Log{Wood: block.OakWood()}

// coreRecipes are all core recipes.
var coreRecipes = []CoreRecipe{
	{ID: "oak_log", Base: oakLog, Output: oakLog},
	{ID: "cobblestone", Base: block.Cobblestone{}, Output: block.Cobblestone{}},
}

// blockKey identifies a block for recipes: its name without the namespace
// and any state (log axis, facing, ...).
func blockKey(b world.Block) string {
	name, _ := b.EncodeBlock()
	name = strings.TrimPrefix(name, "minecraft:")
	return strings.TrimPrefix(name, "flowing_")
}

// blockName is the Korean name of a block for recipe texts.
func blockName(b world.Block) string {
	k := blockKey(b)
	if n, ok := koreanNames[k]; ok {
		return n
	}
	return k
}

// matchRecipe returns the recipe of the core at pos: the one for its base,
// or nil.
func matchRecipe(tx *world.Tx, pos cube.Pos) *CoreRecipe {
	key := blockKey(tx.Block(pos.Side(cube.FaceDown)))
	for i := range coreRecipes {
		if blockKey(coreRecipes[i].Base) == key {
			return &coreRecipes[i]
		}
	}
	return nil
}

// text describes a recipe, e.g. "조약돌 → 조약돌".
func (c *CoreRecipe) text() string {
	return fmt.Sprintf("%s → %s", blockName(c.Base), blockName(c.Output))
}

// coreRecipeTexts lists every recipe for the core texts.
func coreRecipeTexts() string {
	var lines []string
	for i := range coreRecipes {
		lines = append(lines, "§f"+coreRecipes[i].text())
	}
	return strings.Join(lines, "\n")
}
