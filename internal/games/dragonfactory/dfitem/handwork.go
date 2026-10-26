package dfitem

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/blockmodel"
)

// Pebble kinds. Other stones (andesite, diorite, granite) are added here
// when something uses them.
const (
	PebbleStone      = "stone"
	PebbleDeepslate  = "deepslate"
	PebbleBlackstone = "blackstone"
	PebbleBasalt     = "basalt"
	PebbleTuff       = "tuff"
	PebbleCalcite    = "calcite"
	PebbleAndesite   = "andesite"
	PebbleGranite    = "granite"
	PebbleDiorite    = "diorite"
)

// PebbleKinds are the pebble kinds with the weight a stick knocks each out
// of the ground with (0: only sieves give it) and the block four of them
// make.
var PebbleKinds = []struct {
	Kind   string
	name   string
	Weight int
	Block  world.Item
}{
	{PebbleStone, i18n.T(i18n.Default, "pebble.stone"), 1, block.Cobblestone{}},
	{PebbleDeepslate, i18n.T(i18n.Default, "pebble.deepslate"), 0, block.Deepslate{Type: block.CobbledDeepslate()}},
	{PebbleBlackstone, i18n.T(i18n.Default, "pebble.blackstone"), 0, block.Blackstone{}},
	{PebbleBasalt, i18n.T(i18n.Default, "pebble.basalt"), 0, block.Basalt{}},
	{PebbleTuff, i18n.T(i18n.Default, "pebble.tuff"), 0, block.Tuff{}},
	{PebbleCalcite, i18n.T(i18n.Default, "pebble.calcite"), 0, block.Calcite{}},
	{PebbleAndesite, i18n.T(i18n.Default, "pebble.andesite"), 0, block.Andesite{}},
	{PebbleGranite, i18n.T(i18n.Default, "pebble.granite"), 0, block.Granite{}},
	{PebbleDiorite, i18n.T(i18n.Default, "pebble.diorite"), 0, block.Diorite{}},
}

// Pebble is a small stone a stick knocks out of grass and dirt.
type Pebble struct{ Kind string }

func (p Pebble) EncodeItem() (string, int16) { return "dragonfactory:pebble_" + p.Kind, 0 }

func (p Pebble) Name() string {
	for _, k := range PebbleKinds {
		if k.Kind == p.Kind {
			return k.name
		}
	}
	return i18n.T(i18n.Default, "name.Pebble")
}

func (Pebble) Category() category.Category { return category.Items() }

// BarkFiber is stripped from logs; three in a row make three string.
type BarkFiber struct{}

func (BarkFiber) EncodeItem() (string, int16) { return "dragonfactory:bark_fiber", 0 }

func (BarkFiber) Name() string { return i18n.T(i18n.Default, "name.BarkFiber") }

func (BarkFiber) Category() category.Category { return category.Items() }

// CompostChance makes a fiber plant matter for the composter and the barrel
// (composter.go; the chance itself is not used, only that it is above 0).
func (BarkFiber) CompostChance() float64 { return 0.3 }

// GrassSeeds turn the dirt they are used on into grass. Tilling dirt with a
// hoe sometimes drops them (grassSeedChance): an island whose grass is all
// broken gets it back.
type GrassSeeds struct{}

func (GrassSeeds) EncodeItem() (string, int16) { return "dragonfactory:grass_seeds", 0 }

func (GrassSeeds) Name() string { return i18n.T(i18n.Default, "name.GrassSeeds") }

func (GrassSeeds) Category() category.Category { return category.Items() }

// UseOnBlock implements item.UsableOnBlock.
func (GrassSeeds) UseOnBlock(pos cube.Pos, _ cube.Face, _ mgl64.Vec3, tx *world.Tx, _ item.User, ctx *item.UseContext) bool {
	if d, ok := tx.Block(pos).(block.Dirt); !ok || d.Coarse {
		return false
	}
	tx.SetBlock(pos, block.Grass{}, nil)
	tx.PlaySound(pos.Vec3Centre(), sound.ItemUseOn{Block: block.Grass{}})
	ctx.SubtractFromCount(1)
	return true
}

// Dust is sand crushed once more (hammer, crusher), sifted for its drops.
type Dust struct{}

var dustHash = block.NextHash()

func (Dust) EncodeItem() (string, int16) { return "dragonfactory:dust", 0 }

func (Dust) EncodeBlock() (string, map[string]any) { return "dragonfactory:dust", nil }

func (Dust) Hash() (uint64, uint64) { return dustHash, 0 }

func (Dust) Model() world.BlockModel { return model.Solid{} }

func (Dust) Name() string { return i18n.T(i18n.Default, "name.Dust") }

func (Dust) Category() category.Category { return category.Construction() }

func (Dust) Properties() customblock.Properties {
	return customblock.Properties{Cube: true, Textures: map[string]customblock.Material{"*": blockmodel.SolidMaterial("dragonfactory_dust")}}
}

func (d Dust) BreakInfo() block.BreakInfo {
	return block.BreakInfo{
		Hardness: 0.5, BlastResistance: 0.5,
		Harvestable: func(item.Tool) bool { return true },
		Effective:   blockmodel.CustomBlockEffective,
		Drops:       func(item.Tool, []item.Enchantment) []item.Stack { return []item.Stack{item.NewStack(d, 1)} },
	}
}

// HandworkRecipes registers the recipes of the hand-made resources (locked
// until researched like every other recipe).
func HandworkRecipes() {
	for _, k := range PebbleKinds {
		p := item.NewStack(Pebble{Kind: k.Kind}, 1)
		recipe.Register(recipe.NewShapeless([]recipe.Item{p, p, p, p}, item.NewStack(k.Block, 1), "crafting_table"))
	}
	// Three fibers in a row make three string. Three wide, it does not fit
	// the 2x2 grid of the inventory: only a crafting table makes string.
	f := item.NewStack(BarkFiber{}, 1)
	recipe.Register(recipe.NewShaped([]recipe.Item{f, f, f}, item.NewStack(block.String{}, 3), recipe.NewShape(3, 1), "crafting_table"))
}
