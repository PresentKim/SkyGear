package sievetable

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// The drop tables of the magic sieve by the essence in it.
// Chances are per mille at the string mesh; rare entries grow with the
// difficulty of the mob.

// ThemeDrop is a product of a theme.
type ThemeDrop struct {
	Item     item.Stack
	Permille int
	Rare     bool
}

// Theme is what an essence adds to the sieve: Adds by the key of the block
// sifted, and the inputs it Replaces the mesh table of.
type Theme struct {
	Adds    map[string][]ThemeDrop
	Replace map[string]bool
}

var (
	inGravel = naming.ItemKey(block.Gravel{})
	inSand   = naming.ItemKey(block.Sand{})
	inDirt   = naming.ItemKey(block.Dirt{})
	inDust   = naming.ItemKey(sgitem.Dust{})
)

func td(it item.Stack, permille int, rare bool) ThemeDrop { return ThemeDrop{it, permille, rare} }

// MagicThemes are the themes by mob.
var MagicThemes = map[string]Theme{
	sgitem.MobZombie: {
		Adds: map[string][]ThemeDrop{
			inGravel: {
				td(item.NewStack(item.RottenFlesh{}, 1), 200, false),
				td(item.NewStack(item.IronNugget{}, 1), 150, false),
				td(item.NewStack(item.IronIngot{}, 1), 12, true),
			},
			inSand: {
				td(item.NewStack(item.RottenFlesh{}, 1), 150, false),
			},
			inDirt: {
				td(item.NewStack(block.Carrot{}, 1), 200, false),
				td(item.NewStack(block.Potato{}, 1), 200, false),
				td(item.NewStack(item.RottenFlesh{}, 1), 250, false),
				td(item.NewStack(item.IronNugget{}, 1), 120, false),
				td(item.NewStack(item.GoldenCarrot{}, 1), 12, true),
			},
			inDust: {
				td(item.NewStack(item.RottenFlesh{}, 1), 200, false),
				td(item.NewStack(item.BoneMeal{}, 1), 100, false),
			},
		},
	},
	sgitem.MobSkeleton: {
		Adds: map[string][]ThemeDrop{
			inGravel: {
				td(item.NewStack(item.Bone{}, 1), 300, false),
				td(item.NewStack(item.Arrow{}, 1), 250, false),
				td(item.NewStack(block.Skull{Type: block.SkeletonSkull()}, 1), 10, true),
			},
			inSand: {
				td(item.NewStack(item.Bone{}, 1), 200, false),
				td(item.NewStack(item.BoneMeal{}, 1), 150, false),
			},
			inDirt: {
				td(item.NewStack(item.Bone{}, 1), 300, false),
				td(item.NewStack(item.Arrow{}, 1), 250, false),
				td(item.NewStack(item.BoneMeal{}, 1), 200, false),
			},
			inDust: {
				td(item.NewStack(item.Bone{}, 1), 300, false),
				td(item.NewStack(item.Arrow{}, 1), 250, false),
				td(item.NewStack(block.Skull{Type: block.SkeletonSkull()}, 1), 10, true),
			},
		},
	},
	sgitem.MobSpider: {
		Adds: map[string][]ThemeDrop{
			inGravel: {
				td(item.NewStack(block.String{}, 1), 300, false),
				td(item.NewStack(item.SpiderEye{}, 1), 200, false),
				td(item.NewStack(item.FermentedSpiderEye{}, 1), 15, true),
			},
			inSand: {
				td(item.NewStack(block.String{}, 1), 200, false),
				td(item.NewStack(block.Cobweb{}, 1), 100, false),
			},
			inDirt: {
				td(item.NewStack(block.String{}, 1), 300, false),
				td(item.NewStack(item.SpiderEye{}, 1), 200, false),
				td(item.NewStack(block.Cobweb{}, 1), 120, false),
			},
			inDust: {
				td(item.NewStack(block.String{}, 1), 300, false),
				td(item.NewStack(item.SpiderEye{}, 1), 200, false),
				td(item.NewStack(item.FermentedSpiderEye{}, 1), 15, true),
			},
		},
	},
	sgitem.MobCreeper: {
		Adds: map[string][]ThemeDrop{
			inGravel: {
				td(item.NewStack(item.Gunpowder{}, 1), 300, false),
				td(item.NewStack(block.TNT{}, 1), 12, true),
			},
			inSand: {
				td(item.NewStack(item.Gunpowder{}, 1), 150, false),
			},
			inDirt: {
				td(item.NewStack(item.Gunpowder{}, 1), 120, false),
			},
			inDust: {
				td(item.NewStack(item.Gunpowder{}, 1), 250, false),
				td(item.NewStack(item.DiscFragment{}, 1), 15, true),
			},
		},
	},
	sgitem.MobSlime: {
		Adds: map[string][]ThemeDrop{
			inGravel: {
				td(item.NewStack(item.Slimeball{}, 1), 150, false),
			},
			inSand: {
				td(item.NewStack(item.Slimeball{}, 1), 200, false),
			},
			inDirt: {
				td(item.NewStack(item.Slimeball{}, 1), 300, false),
				td(item.NewStack(item.ClayBall{}, 1), 200, false),
				td(item.NewStack(block.Slime{}, 1), 15, true),
			},
			inDust: {
				td(item.NewStack(item.Slimeball{}, 1), 200, false),
				td(item.NewStack(item.MagmaCream{}, 1), 15, true),
			},
		},
	},
	sgitem.MobDrowned: {
		Adds: map[string][]ThemeDrop{
			inGravel: {
				td(item.NewStack(item.PrismarineShard{}, 1), 100, false),
				td(item.NewStack(item.Cod{}, 1), 100, false),
				td(item.NewStack(item.ClayBall{}, 1), 150, false),
				td(item.NewStack(item.NautilusShell{}, 1), 20, true),
			},
			inSand: {
				td(item.NewStack(block.Kelp{}, 1), 400, false),
				td(item.NewStack(item.PrismarineShard{}, 1), 200, false),
				td(item.NewStack(item.Cod{}, 1), 140, false),
				td(item.NewStack(block.SeaPickle{}, 1), 120, false),
				td(item.NewStack(item.Salmon{}, 1), 80, false),
				td(item.NewStack(item.InkSac{}, 1), 80, false),
				td(item.NewStack(item.TropicalFish{}, 1), 60, false),
				td(item.NewStack(item.PrismarineCrystals{}, 1), 80, false),
				td(item.NewStack(block.Sponge{}, 1), 40, true),
				td(item.NewStack(item.NautilusShell{}, 1), 25, true),
			},
			inDirt: {
				td(item.NewStack(block.LilyPad{}, 1), 150, false),
				td(item.NewStack(block.Mud{}, 1), 120, false),
				td(item.NewStack(item.ClayBall{}, 1), 200, false),
				td(item.NewStack(block.SeaPickle{}, 1), 80, false),
			},
			inDust: {
				td(item.NewStack(item.Pufferfish{}, 1), 70, false),
				td(item.NewStack(item.InkSac{}, 1), 100, false),
				td(item.NewStack(item.HeartOfTheSea{}, 1), 8, true),
				td(item.NewStack(item.NautilusShell{}, 1), 15, true),
			},
		},
		Replace: map[string]bool{inSand: true},
	},
	sgitem.MobWitch: {
		Adds: map[string][]ThemeDrop{
			inGravel: {
				td(item.NewStack(item.Gunpowder{}, 1), 100, false),
				td(item.NewStack(item.RedstoneWire{}, 1), 120, false),
				td(item.NewStack(item.SpiderEye{}, 1), 60, false),
				td(item.NewStack(item.GoldenApple{}, 1), 10, true),
			},
			inSand: {
				td(item.NewStack(item.GlowstoneDust{}, 1), 100, false),
				td(item.NewStack(item.FermentedSpiderEye{}, 1), 50, false),
				td(item.NewStack(item.InkSac{}, 1), 60, false),
				td(item.NewStack(item.Gunpowder{}, 1), 80, false),
			},
			inDirt: {
				td(item.NewStack(item.Sugar{}, 1), 150, false),
				td(item.NewStack(item.RedstoneWire{}, 1), 150, false),
				td(item.NewStack(item.Stick{}, 1), 120, false),
				td(item.NewStack(item.SpiderEye{}, 1), 80, false),
				td(item.NewStack(item.GlassBottle{}, 1), 100, false),
			},
			inDust: {
				td(item.NewStack(item.GlowstoneDust{}, 1), 100, false),
				td(item.NewStack(item.RedstoneWire{}, 1), 150, false),
				td(item.NewStack(item.Sugar{}, 1), 100, false),
			},
		},
	},
	sgitem.MobPiglin: {
		Adds: map[string][]ThemeDrop{
			inGravel: {
				td(item.NewStack(block.Netherrack{}, 1), 400, false),
				td(item.NewStack(block.Blackstone{}, 1), 200, false),
				td(item.NewStack(item.GoldNugget{}, 1), 250, false),
				td(item.NewStack(item.NetherQuartz{}, 1), 140, false),
				td(item.NewStack(block.AncientDebris{}, 1), 12, true),
			},
			inSand: {
				td(item.NewStack(block.SoulSand{}, 1), 300, false),
				td(item.NewStack(block.SoulSoil{}, 1), 150, false),
				td(item.NewStack(item.GlowstoneDust{}, 1), 140, false),
				td(item.NewStack(block.Magma{}, 1), 80, false),
				td(item.NewStack(block.Basalt{}, 1), 100, false),
				td(item.NewStack(item.GhastTear{}, 1), 15, true),
			},
			inDirt: {
				td(item.NewStack(block.NetherWart{}, 1), 120, false),
				td(item.NewStack(block.NetherSprouts{}, 1), 120, false),
				td(item.NewStack(block.Shroomlight{}, 1), 18, true),
			},
			inDust: {
				td(item.NewStack(item.GoldNugget{}, 1), 150, false),
				td(item.NewStack(item.GlowstoneDust{}, 1), 120, false),
				td(item.NewStack(item.BlazePowder{}, 1), 30, false),
				td(item.NewStack(item.NetheriteScrap{}, 1), 6, true),
			},
		},
		Replace: map[string]bool{inGravel: true, inSand: true},
	},
	sgitem.MobEnder: {
		Adds: map[string][]ThemeDrop{
			inGravel: {
				td(item.NewStack(block.EndStone{}, 1), 400, false),
				td(item.NewStack(item.PoppedChorusFruit{}, 1), 90, false),
				td(item.NewStack(item.EnderPearl{}, 1), 50, false),
				td(item.NewStack(item.ShulkerShell{}, 1), 12, true),
			},
			inSand: {
				td(item.NewStack(block.Purpur{}, 1), 180, false),
				td(item.NewStack(block.EndBricks{}, 1), 120, false),
				td(item.NewStack(item.EnderPearl{}, 1), 60, false),
				td(item.NewStack(item.DragonBreath{}, 1), 25, true),
				td(item.NewStack(item.EndCrystal{}, 1), 6, true),
			},
			inDirt: {
				td(item.NewStack(item.PoppedChorusFruit{}, 1), 150, false),
				td(item.NewStack(block.EndStone{}, 1), 120, false),
			},
			inDust: {
				td(item.NewStack(item.EnderEye{}, 1), 40, false),
				td(item.NewStack(item.EnderPearl{}, 1), 80, false),
				td(item.NewStack(item.ShulkerShell{}, 1), 15, true),
			},
		},
		Replace: map[string]bool{inGravel: true, inSand: true},
	},
	sgitem.MobBlaze: {
		Adds: map[string][]ThemeDrop{
			inGravel: {
				td(item.NewStack(block.Magma{}, 1), 140, false),
				td(item.NewStack(item.FireCharge{}, 1), 100, false),
				td(item.NewStack(item.BlazeRod{}, 1), 25, true),
			},
			inSand: {
				td(item.NewStack(item.GlowstoneDust{}, 1), 150, false),
				td(item.NewStack(item.MagmaCream{}, 1), 80, false),
				td(item.NewStack(item.GhastTear{}, 1), 8, true),
			},
			inDirt: {
				td(item.NewStack(item.Coal{}, 1), 150, false),
				td(item.NewStack(block.Shroomlight{}, 1), 10, true),
			},
			inDust: {
				td(item.NewStack(item.BlazePowder{}, 1), 250, false),
				td(item.NewStack(item.FireCharge{}, 1), 80, false),
				td(item.NewStack(item.BlazeRod{}, 1), 20, true),
			},
		},
	},
	sgitem.MobWither: {
		Adds: map[string][]ThemeDrop{
			inGravel: {
				td(item.NewStack(item.Bone{}, 1), 200, false),
				td(item.NewStack(item.Coal{}, 1), 250, false),
				td(item.NewStack(item.NetheriteScrap{}, 1), 10, true),
			},
			inSand: {
				td(item.NewStack(block.SoulSand{}, 1), 200, false),
				td(item.NewStack(item.NetherQuartz{}, 1), 120, false),
			},
			inDirt: {
				td(item.NewStack(item.BoneMeal{}, 1), 200, false),
				td(item.NewStack(item.Coal{}, 1), 150, false),
			},
			inDust: {
				td(item.NewStack(item.Bone{}, 1), 300, false),
				td(item.NewStack(item.Coal{}, 1), 300, false),
				td(item.NewStack(block.Skull{Type: block.WitherSkeletonSkull()}, 1), 8, true),
				td(item.NewStack(item.NetheriteScrap{}, 1), 18, true),
			},
		},
	},
}

// MagicCharges is how many sifts one essence charges.
const MagicCharges = 24

// ThemeCap is the most a theme drop is likely to come out, per mille.
const ThemeCap = 950

// rareBoost is how much a mob's difficulty raises its rare drops: ×1 for the
// easiest, +0.2 a step.
func rareBoost(mob string) float64 { return 1 + 0.2*float64(sgitem.MobDifficulty[mob]-1) }

// meshBoost is how much a better mesh multiplies the chances of the theme
// drops of the magic sieve (percent).
var meshBoost = map[string]int{"flint": 130, "iron": 160, "gold": 200, "diamond": 250, "netherite": 300}

// ThemeChance is the chance, per mille, of a theme drop with a mesh fitted.
func ThemeChance(d ThemeDrop, mesh, mob string) float64 {
	boost := 1.0
	if percent, ok := meshBoost[mesh]; ok {
		boost = float64(percent) / 100
	}
	p := float64(d.Permille) * boost
	if d.Rare {
		p *= rareBoost(mob)
	}
	return min(p, ThemeCap)
}
