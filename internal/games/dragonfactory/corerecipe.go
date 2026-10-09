package dragonfactory

import (
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/ui"
)

// Core recipes (docs/design/02-dragonfactory.md §3.2): the block under a
// core (its base) and the blocks around that base choose what the core
// makes. The base is not used up. Values are drafts.

// CoreRecipe is one thing a core can make.
type CoreRecipe struct {
	ID string
	// Tier is the lowest core tier that makes it.
	Tier int
	// Base is the block under the core.
	Base world.Block
	// Around are blocks needed among the four sides of the base.
	Around []AroundNeed
	// Output is the block made above the core.
	Output world.Block
	// Hidden recipes show only a hint in the core book until a core in the
	// save first makes them.
	Hidden bool
	Hint   string
}

// AroundNeed asks for at least Count of a block beside the base.
type AroundNeed struct {
	Block world.Block
	Count int
}

var (
	baseWater = block.Water{Still: true, Depth: 8}
	oakLog    = block.Log{Wood: block.OakWood()}
)

// coreRecipes are all recipes in core book order.
var coreRecipes = []CoreRecipe{
	{ID: "oak_log", Tier: 1, Base: oakLog, Output: oakLog},
	{ID: "cobblestone", Tier: 1, Base: block.Cobblestone{}, Output: block.Cobblestone{}},
	{ID: "dirt", Tier: 1, Base: block.Dirt{}, Output: block.Dirt{}},
	{ID: "sand", Tier: 1, Base: block.Sand{}, Output: block.Sand{}},
	{ID: "gravel", Tier: 1, Base: block.Gravel{}, Output: block.Gravel{}},
	{ID: "smooth_stone", Tier: 1, Base: block.Cobblestone{}, Around: []AroundNeed{{baseWater, 1}}, Output: block.Stone{},
		Hidden: true, Hint: "물에 닿은 조약돌은 매끄러워진다"},
	{ID: "mud", Tier: 2, Base: block.Dirt{}, Around: []AroundNeed{{baseWater, 1}}, Output: block.Mud{}},
	{ID: "coal_ore", Tier: 2, Base: block.Coal{}, Output: block.CoalOre{}},
	{ID: "clay", Tier: 2, Base: block.Clay{}, Output: block.Clay{}},
	{ID: "wet_sand", Tier: 2, Base: block.Sand{}, Around: []AroundNeed{{baseWater, 4}}, Output: block.Clay{},
		Hidden: true, Hint: "사방이 물에 잠긴 모래는 끈적해진다"},
	{ID: "pumpkin", Tier: 2, Base: block.Pumpkin{}, Output: block.Pumpkin{}},
	{ID: "melon", Tier: 2, Base: block.Melon{}, Output: block.Melon{}},
	{ID: "hay", Tier: 2, Base: block.HayBale{Axis: cube.Y}, Output: block.HayBale{Axis: cube.Y}},
	{ID: "iron_ore", Tier: 3, Base: block.Iron{}, Output: block.IronOre{}},
	{ID: "copper_ore", Tier: 3, Base: block.Copper{}, Output: block.CopperOre{}},
	{ID: "dried_kelp", Tier: 3, Base: block.DriedKelp{}, Output: block.DriedKelp{}},
	{ID: "iron_vein", Tier: 3, Base: block.Stone{}, Around: []AroundNeed{{block.Iron{}, 1}}, Output: block.IronOre{},
		Hidden: true, Hint: "철과 맞닿은 돌에 광맥이 스민다"},
	{ID: "gold_ore", Tier: 4, Base: block.Gold{}, Output: block.GoldOre{}},
}

// blockKey identifies a block for recipes: its name without the namespace,
// any state (log axis, facing, ...) and flowing/still water alike.
func blockKey(b world.Block) string {
	name, _ := b.EncodeBlock()
	name = strings.TrimPrefix(name, "minecraft:")
	return strings.TrimPrefix(name, "flowing_")
}

// keyAt is the recipe key of the block at pos; a liquid counts as itself.
func keyAt(tx *world.Tx, pos cube.Pos) string {
	if l, ok := tx.Liquid(pos); ok {
		return blockKey(l)
	}
	return blockKey(tx.Block(pos))
}

// blockName is the Korean name of a block for recipe texts.
func blockName(b world.Block) string {
	k := blockKey(b)
	if n, ok := koreanNames[k]; ok {
		return n
	}
	return k
}

// matchRecipe returns the recipe a core of the tier at pos makes: of the
// recipes its base and surroundings satisfy, the one with the most
// surrounding blocks. If none fits the tier, need is the lowest tier that
// would make something (0 if nothing would).
func matchRecipe(tx *world.Tx, pos cube.Pos, tier int) (r *CoreRecipe, need int) {
	base := pos.Side(cube.FaceDown)
	key := keyAt(tx, base)
	var around map[string]int
	for i := range coreRecipes {
		c := &coreRecipes[i]
		if blockKey(c.Base) != key {
			continue
		}
		if len(c.Around) > 0 && around == nil {
			around = map[string]int{}
			for _, f := range cube.HorizontalFaces() {
				around[keyAt(tx, base.Side(f))]++
			}
		}
		if !c.satisfied(around) {
			continue
		}
		if c.Tier > tier {
			if need == 0 || c.Tier < need {
				need = c.Tier
			}
			continue
		}
		if r == nil || len(c.Around) > len(r.Around) {
			r = c
		}
	}
	if r != nil {
		need = 0
	}
	return r, need
}

func (c *CoreRecipe) satisfied(around map[string]int) bool {
	for _, n := range c.Around {
		if around[blockKey(n.Block)] < n.Count {
			return false
		}
	}
	return true
}

// text describes a recipe, e.g. "조약돌 + 둘레 물 1개 이상 → 돌".
func (c *CoreRecipe) text() string {
	var b strings.Builder
	b.WriteString(blockName(c.Base))
	for _, n := range c.Around {
		if n.Count >= 4 {
			fmt.Fprintf(&b, " + 사방 %s", blockName(n.Block))
		} else {
			fmt.Fprintf(&b, " + 둘레 %s %d개 이상", blockName(n.Block), n.Count)
		}
	}
	fmt.Fprintf(&b, " → %s", blockName(c.Output))
	return b.String()
}

func recipeByID(id string) *CoreRecipe {
	for i := range coreRecipes {
		if coreRecipes[i].ID == id {
			return &coreRecipes[i]
		}
	}
	return nil
}

// known reports whether the save's players may see the recipe.
func (s *session) known(r *CoreRecipe) bool { return !r.Hidden || s.state.Discovered[r.ID] }

// discover records a hidden recipe the first time a core makes it and tells
// everyone in the save.
func (s *session) discover(tx *world.Tx, r *CoreRecipe) {
	if !r.Hidden || s.state.Discovered[r.ID] {
		return
	}
	if s.state.Discovered == nil {
		s.state.Discovered = map[string]bool{}
	}
	s.state.Discovered[r.ID] = true
	s.storeState()
	for e := range tx.Players() {
		e.(*player.Player).Messagef("§d새 코어 레시피를 발견했습니다! §f%s §7(코어 도감)", r.text())
	}
}

// tierOpenLevel is the level from which cores of a tier are in the shop.
func tierOpenLevel(tier int) int {
	for _, c := range coreShop {
		if coreTypes[c.Type].Tier == tier {
			return c.Level
		}
	}
	return 1
}

// showCoreBook shows every recipe: known ones in full, hidden ones as ???
// with a hint once their tier is open.
func (s *session) showCoreBook(p *player.Player) {
	body, opts := s.coreBook()
	ui.Menu(p, "코어 도감", body, opts...)
}

func (s *session) coreBook() (string, []ui.Option) {
	var b strings.Builder
	found, hidden := 0, 0
	for _, r := range coreRecipes {
		if r.Hidden {
			hidden++
			if s.state.Discovered[r.ID] {
				found++
			}
		}
	}
	fmt.Fprintf(&b, "§7코어 아래 받침 블록(과 그 둘레)이 만들 것을 정합니다. 받침은 닳지 않습니다.\n§7숨은 레시피 발견 §d%d/%d", found, hidden)
	for tier := 1; tier <= len(coreTiers); tier++ {
		fmt.Fprintf(&b, "\n\n§e[%d티어]", tier)
		for i := range coreRecipes {
			r := &coreRecipes[i]
			switch {
			case r.Tier != tier:
			case s.known(r) && r.Hidden:
				fmt.Fprintf(&b, "\n§d★ §f%s", r.text())
			case s.known(r):
				fmt.Fprintf(&b, "\n§f%s", r.text())
			case s.state.Level >= tierOpenLevel(tier):
				fmt.Fprintf(&b, "\n§8??? §7단서: %s", r.Hint)
			default:
				fmt.Fprintf(&b, "\n§8??? §7(Lv.%d에 단서 공개)", tierOpenLevel(tier))
			}
		}
	}
	return b.String(), []ui.Option{
		s.opt("§b코어 관리", iconCores, func(p *player.Player, tx *world.Tx) { s.showCores(p, tx) }),
		s.backOption(),
	}
}
