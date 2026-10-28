package sgitem

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
)

// SimCharges is how many runs one essence gives.
const SimCharges = 8

// SimDrop is a possible drop of a run: Min to Max of Item with a Chance percent.
type SimDrop struct {
	Item     world.Item
	Min, Max int
	Chance   int
}

// SimDrops are the drops of each mob, drafts.
var SimDrops = map[string][]SimDrop{
	MobZombie:   {{item.RottenFlesh{}, 1, 3, 100}},
	MobSkeleton: {{item.Bone{}, 1, 2, 100}, {item.Arrow{}, 1, 2, 60}},
	MobCreeper:  {{item.Gunpowder{}, 1, 2, 100}},
	MobSpider:   {{block.String{}, 1, 2, 100}, {item.SpiderEye{}, 1, 1, 30}},
	MobSlime:    {{item.Slimeball{}, 1, 2, 100}},
	MobDrowned:  {{item.RottenFlesh{}, 1, 2, 100}, {item.CopperIngot{}, 1, 1, 30}, {item.NautilusShell{}, 1, 1, 6}},
	MobWitch:    {{item.Sugar{}, 1, 2, 60}, {item.Stick{}, 1, 2, 60}, {item.RedstoneWire{}, 1, 2, 60}, {item.GlowstoneDust{}, 1, 2, 40}, {item.SpiderEye{}, 1, 1, 30}, {item.GlassBottle{}, 1, 1, 30}},
	MobPiglin:   {{item.GoldNugget{}, 1, 3, 100}, {item.GoldIngot{}, 1, 1, 25}, {item.Porkchop{}, 1, 1, 40}},
	MobEnder:    {{item.EnderPearl{}, 1, 1, 40}},
	MobBlaze:    {{item.BlazeRod{}, 1, 1, 40}},
	MobWither:   {{item.Bone{}, 1, 2, 100}, {item.Coal{}, 1, 2, 80}, {block.Skull{Type: block.WitherSkeletonSkull()}, 1, 1, 4}},
}

// Essence is the essence of a mob.
type Essence struct{ Mob string }

func (e Essence) EncodeItem() (string, int16) { return "skygear:essence_" + e.Mob, 0 }

func (e Essence) Name() string {
	return i18n.T(i18n.Default, "mob."+e.Mob) + i18n.T(i18n.Default, "name.Essence.suffix")
}

func (Essence) Category() category.Category { return category.Items() }

// MaxCount keeps essences in stacks of 16.
func (Essence) MaxCount() int { return 16 }

// AcceptsEssence reports whether the simulator takes it.
func AcceptsEssence(it item.Stack) bool {
	_, ok := it.Item().(Essence)
	return ok
}
