package farm

import (
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/blockmodel"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// Silkworms and infested leaves, after Ex Nihilo Fabrico (SirLyle, MIT;
// InfestedHelper, InfestingLeavesBlockEntity, SilkWormItem):
//   - A silkworm used on leaves turns them into infesting leaves.
//   - Infesting leaves go up a stage every InfestStageTime (a scheduled
//     tick; a random tick also moves them on, so leaves whose tick got lost
//     still finish). From the second stage on every step tries to spread
//     the worms to one random block of the 3×3×3 around (InfestSpreadChance).
//     After the last stage they are infested leaves.
//   - Infested leaves keep trying to spread on their random ticks
//     (InfestedSpreadChance) and drop string when broken, more with a crook.
//   - Neither ever decays: Dragonfly's leaves look for their log through
//     them (block.LeavesLike, a Dragonfly patch), so infesting the middle of
//     a crown keeps its outer leaves, and a felled tree leaves its infested
//     leaves hanging for the string.
//   - The worms only spread onto the island's own chunks.

func init() {
	for stage := range InfestStages {
		world.RegisterBlock(InfestingLeaves{Stage: stage})
	}
	world.RegisterItem(InfestingLeaves{})
	world.RegisterBlock(InfestedLeaves{})
	world.RegisterItem(InfestedLeaves{})
	world.RegisterItem(Silkworm{})
	sgitem.CrookBlockNames = append(sgitem.CrookBlockNames, infestingName, infestedName)
}

// Balancing drafts (docs/design/open-questions.md).
var (
	// InfestStageTime is the time one stage of infesting leaves takes, plus
	// up to InfestStageJitter.
	InfestStageTime   = 30 * time.Second
	InfestStageJitter = 10 * time.Second
	// InfestSpreadChance is the chance a stage step of infesting leaves (from
	// stage 1 on) spreads the worms, InfestedSpreadChance that a random tick
	// of infested leaves does. A spread picks one block of the 3×3×3 around;
	// only leaves catch it.
	InfestSpreadChance   = 0.25
	InfestedSpreadChance = 0.25
	// StringByHand and StringByCrook are the rolls of string infested
	// leaves drop: a string for each roll that comes up (Fabrico's crook
	// table; Fabrico's leaves drop nothing by hand).
	StringByHand  = []float64{0.5, 0.25}
	StringByCrook = []float64{1, 1, 0.5, 0.2, 0.1}
)

// InfestStages are the stages of infesting leaves (0 to InfestStages-1).
const InfestStages = 4

const (
	infestingName = "skygear:infesting_leaves"
	infestedName  = "skygear:infested_leaves"
	// infestStageState is the one block state of infesting leaves.
	infestStageState = "skygear:infest_stage"
)

var (
	infestingHash = block.NextHash()
	infestedHash  = block.NextHash()
)

// leafMaterial is the look of a leaves texture: cut out like leaves.
func leafMaterial(texture string) customblock.Material {
	return blockmodel.AlphaMaterial(texture)
}

// InfestingLeaves are leaves the silkworms are eating their way through.
type InfestingLeaves struct{ Stage int }

func (InfestingLeaves) EncodeItem() (string, int16) { return infestingName, 0 }

func (l InfestingLeaves) EncodeBlock() (string, map[string]any) {
	return infestingName, map[string]any{infestStageState: fmt.Sprint(l.Stage)}
}

func (l InfestingLeaves) Hash() (uint64, uint64) { return infestingHash, uint64(l.Stage) }

func (InfestingLeaves) Model() world.BlockModel { return model.Leaves{} }

func (InfestingLeaves) Name() string { return i18n.T(i18n.Default, "name.InfestingLeaves") }

func (InfestingLeaves) Category() category.Category { return category.Nature() }

func infestingTexture(stage int) string { return fmt.Sprintf("skygear_infesting_leaves_%d", stage) }

func (InfestingLeaves) Properties() customblock.Properties {
	return customblock.Properties{Cube: true, Textures: map[string]customblock.Material{"*": leafMaterial(infestingTexture(0))}}
}

// States implements block.Permutable.
func (InfestingLeaves) States() map[string][]any {
	var values []any
	for i := range InfestStages {
		values = append(values, fmt.Sprint(i))
	}
	return map[string][]any{infestStageState: values}
}

// Permutations implements block.Permutable: one texture per stage.
func (InfestingLeaves) Permutations() []customblock.Permutation {
	var out []customblock.Permutation
	for stage := 1; stage < InfestStages; stage++ {
		out = append(out, customblock.Permutation{
			Condition: fmt.Sprintf("q.block_state('%s') == '%d'", infestStageState, stage),
			Properties: customblock.Properties{
				Cube:     true,
				Textures: map[string]customblock.Material{"*": leafMaterial(infestingTexture(stage))},
			},
		})
	}
	return out
}

// BreakInfo: leaves still being eaten give nothing; the worms are lost.
func (InfestingLeaves) BreakInfo() block.BreakInfo {
	return leavesBreakInfo(func(item.Tool, []item.Enchantment) []item.Stack { return nil })
}

func (InfestingLeaves) LightDiffusionLevel() uint8 { return 1 }

func (InfestingLeaves) SideClosed(cube.Pos, cube.Pos, *world.Tx) bool { return false }

func (InfestingLeaves) FlammabilityInfo() block.FlammabilityInfo {
	return block.FlammabilityInfo{Encouragement: 30, Flammability: 60, LavaFlammable: true}
}

// LeavesLike implements block.LeavesLike: leaves find their log through it.
func (InfestingLeaves) LeavesLike() bool { return true }

// ScheduledTick moves the leaves a stage on.
func (l InfestingLeaves) ScheduledTick(pos cube.Pos, tx *world.Tx, r *rand.Rand) {
	if cur, ok := tx.Block(pos).(InfestingLeaves); ok {
		AdvanceInfesting(tx, pos, cur, r)
	}
}

// RandomTick moves the leaves a stage on too: the stage also goes on for
// leaves whose scheduled tick was lost (an infesting leaves item placed).
func (l InfestingLeaves) RandomTick(pos cube.Pos, tx *world.Tx, r *rand.Rand) {
	AdvanceInfesting(tx, pos, l, r)
}

// AdvanceInfesting moves infesting leaves l at pos one stage on, spreading
// the worms from stage 1 on, and schedules the next stage; after the last
// stage they become infested leaves.
func AdvanceInfesting(tx *world.Tx, pos cube.Pos, l InfestingLeaves, r *rand.Rand) {
	if l.Stage >= 1 && r.Float64() < InfestSpreadChance {
		TrySpread(tx, pos, r)
	}
	if l.Stage+1 >= InfestStages {
		tx.SetBlock(pos, InfestedLeaves{}, nil)
		return
	}
	l.Stage++
	tx.SetBlock(pos, l, nil)
	scheduleInfest(tx, pos, l, r)
}

// scheduleInfest schedules the next stage of the infesting leaves at pos.
func scheduleInfest(tx *world.Tx, pos cube.Pos, l InfestingLeaves, r *rand.Rand) {
	delay := InfestStageTime
	if InfestStageJitter > 0 {
		delay += time.Duration(r.Int64N(int64(InfestStageJitter)))
	}
	tx.ScheduleBlockUpdate(pos, l, delay)
}

// InfestedLeaves are leaves the silkworms have turned into silk.
type InfestedLeaves struct{}

func (InfestedLeaves) EncodeItem() (string, int16) { return infestedName, 0 }

func (InfestedLeaves) EncodeBlock() (string, map[string]any) { return infestedName, nil }

func (InfestedLeaves) Hash() (uint64, uint64) { return infestedHash, 0 }

func (InfestedLeaves) Model() world.BlockModel { return model.Leaves{} }

func (InfestedLeaves) Name() string { return i18n.T(i18n.Default, "name.InfestedLeaves") }

func (InfestedLeaves) Category() category.Category { return category.Nature() }

func (InfestedLeaves) Properties() customblock.Properties {
	return customblock.Properties{Cube: true, Textures: map[string]customblock.Material{"*": leafMaterial("skygear_infested_leaves")}}
}

// BreakInfo: string, more with a crook.
func (InfestedLeaves) BreakInfo() block.BreakInfo {
	return leavesBreakInfo(func(t item.Tool, _ []item.Enchantment) []item.Stack {
		return InfestedDrops(t, rand.Float64)
	})
}

func (InfestedLeaves) LightDiffusionLevel() uint8 { return 1 }

func (InfestedLeaves) SideClosed(cube.Pos, cube.Pos, *world.Tx) bool { return false }

func (InfestedLeaves) FlammabilityInfo() block.FlammabilityInfo {
	return block.FlammabilityInfo{Encouragement: 30, Flammability: 60, LavaFlammable: true}
}

// LeavesLike implements block.LeavesLike.
func (InfestedLeaves) LeavesLike() bool { return true }

// RandomTick tries to spread the worms.
func (InfestedLeaves) RandomTick(pos cube.Pos, tx *world.Tx, r *rand.Rand) {
	if r.Float64() < InfestedSpreadChance {
		TrySpread(tx, pos, r)
	}
}

// InfestedDrops are the drops of infested leaves broken with t: a string
// for each roll of the table that comes up (roll is rand.Float64 in play).
func InfestedDrops(t item.Tool, roll func() float64) []item.Stack {
	rolls := StringByHand
	if _, crook := t.(sgitem.Crook); crook {
		rolls = StringByCrook
	}
	n := 0
	for _, c := range rolls {
		if roll() < c {
			n++
		}
	}
	if n == 0 {
		return nil
	}
	return []item.Stack{item.NewStack(block.String{}, n)}
}

// leavesBreakInfo is the break info of the infested leaves: as soft as
// leaves, quickest with a crook (a hoe to the server) or shears.
func leavesBreakInfo(drops func(item.Tool, []item.Enchantment) []item.Stack) block.BreakInfo {
	return block.BreakInfo{
		Hardness: 0.2, BlastResistance: 0.2,
		Harvestable: func(item.Tool) bool { return true },
		Effective: func(t item.Tool) bool {
			return t.ToolType() == item.TypeHoe || t.ToolType() == item.TypeShears
		},
		Drops: drops,
	}
}

// Infest turns the leaves at pos into infesting leaves and schedules their
// first stage, reporting whether there were leaves to infest.
func Infest(tx *world.Tx, pos cube.Pos, r *rand.Rand) bool {
	if _, ok := tx.Block(pos).(block.Leaves); !ok {
		return false
	}
	l := InfestingLeaves{}
	tx.SetBlock(pos, l, nil)
	scheduleInfest(tx, pos, l, r)
	return true
}

// TrySpread spreads the worms from pos to one random block of the 3×3×3
// around it: leaves there start infesting, anything else is missed. The
// worms stay on the island's own chunks.
func TrySpread(tx *world.Tx, pos cube.Pos, r *rand.Rand) bool {
	at := pos.Add(cube.Pos{r.IntN(3) - 1, r.IntN(3) - 1, r.IntN(3) - 1})
	if at == pos || at.OutOfBounds(tx.Range()) {
		return false
	}
	if s, ok := core.Of(tx); ok && !island.CanBuild(s, at) {
		return false
	}
	return Infest(tx, at, r)
}

// CrookLeafDrops are the drops of leaves l broken with crook c: a sapling at
// the crook's chance (half for jungle leaves), a silkworm at its chance and
// the leaves' own other drops (apples). roll is rand.Float64 in play.
func CrookLeafDrops(l block.Leaves, c sgitem.Crook, roll func() float64) []item.Stack {
	var out []item.Stack
	for _, d := range l.BreakInfo().Drops(item.ToolNone{}, nil) {
		if _, sapling := d.Item().(block.Sapling); !sapling {
			out = append(out, d)
		}
	}
	tier := c.Stats()
	if sapling, ok := l.Type.Sapling(); ok {
		chance := tier.Sapling
		if l.Type == block.JungleLeaves() {
			chance /= 2
		}
		if roll() < chance {
			out = append(out, item.NewStack(block.Sapling{Type: sapling}, 1))
		}
	}
	if roll() < tier.Silkworm {
		out = append(out, item.NewStack(Silkworm{}, 1))
	}
	return out
}

// Silkworm infests the leaves it is used on. Crooks pull them out of leaves.
type Silkworm struct{}

func (Silkworm) EncodeItem() (string, int16) { return "skygear:silkworm", 0 }

func (Silkworm) Name() string { return i18n.T(i18n.Default, "name.Silkworm") }

func (Silkworm) Category() category.Category { return category.Items() }

// UseOnBlock implements item.UsableOnBlock: leaves turn into infesting leaves.
func (Silkworm) UseOnBlock(pos cube.Pos, _ cube.Face, _ mgl64.Vec3, tx *world.Tx, _ item.User, ctx *item.UseContext) bool {
	r := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	if s, ok := core.Of(tx); ok {
		r = s.Rand
	}
	if !Infest(tx, pos, r) {
		return false
	}
	tx.PlaySound(pos.Vec3Centre(), sound.ItemUseOn{Block: block.Leaves{Type: block.OakLeaves()}})
	ctx.SubtractFromCount(1)
	return true
}
