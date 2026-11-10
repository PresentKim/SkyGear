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
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// Silkworms and silkworm leaves, after Ex Nihilo Fabrico (SirLyle, MIT;
// InfestedHelper, InfestingLeavesBlockEntity, SilkWormItem):
//   - A silkworm used on leaves turns them into silkworm leaves at cocoon 0%.
//   - The cocoon grows CocoonStep (10%) every CocoonStepTime, on a scheduled
//     tick only, so it takes the same time everywhere (a random tick only
//     schedules the next step again where one got lost: an old tick that
//     comes for a state the leaves have left does nothing).
//   - From SpreadFrom on, every step spreads the worms at SpreadChance to
//     one of the leaves in the 3×3×3 around (only leaves are picked, so a
//     spread does not miss), on the island's own chunks only (never in a
//     world with no island session). At 100% they stop and drop string when
//     broken, more with a crook.
//   - Leaves look for their log through them (block.LeavesLike, a Dragonfly
//     patch). Silkworm leaves of leaves that grew on a tree decay like them
//     when no log is left (FindLog), at once on the neighbour update; those
//     of placed (persistent) leaves stay and pass the update on to the leaves
//     behind them (wakeLeaves).
//   - The block keeps the id of the infesting leaves of 26.11.7, whose
//     stages 0 to 3 read as 0% to 30%; the old infested leaves block turns
//     into 100% silkworm leaves on its next tick or update (legacyInfested).

func init() {
	for _, persistent := range []bool{false, true} {
		for p := 0; p <= CocoonMax; p++ {
			world.RegisterBlock(SilkwormLeaves{Cocoon: p, Persistent: persistent})
		}
	}
	world.RegisterItem(SilkwormLeaves{Cocoon: CocoonMax, Persistent: true})
	world.RegisterBlock(legacyInfested{})
	world.RegisterItem(legacyInfested{})
	world.RegisterItem(Silkworm{})
	world.RegisterItem(CookedSilkworm{})
	sgitem.CrookBlockNames = append(sgitem.CrookBlockNames, silkwormLeavesName, legacyInfestedName)
}

// Balancing drafts (docs/design/open-questions.md).
var (
	// CocoonStepTime is the time the cocoon takes to grow one step.
	CocoonStepTime = 12 * time.Second
	// SpreadChance is the chance a step from SpreadFrom on spreads the
	// worms to one of the leaves around.
	SpreadChance = 0.35
	SpreadFrom   = 2
	// StringByHand and StringByCrook are the rolls of string full silkworm
	// leaves drop: a string for each roll that comes up (Fabrico's crook
	// table; Fabrico's leaves drop nothing by hand).
	StringByHand  = []float64{0.5, 0.25}
	StringByCrook = []float64{1, 1, 0.5, 0.2, 0.1}
	// CookedSilkwormFood and CookedSilkwormSaturation are what a cooked
	// silkworm fills (Fabrico's: as cooked cod).
	CookedSilkwormFood       = 5
	CookedSilkwormSaturation = 6.0
)

// CocoonMax is the last step of the cocoon (100%); a step is CocoonStep %.
const (
	CocoonMax  = 10
	CocoonStep = 10
)

// CocoonTime is how long silkworm leaves take from 0% to 100%.
func CocoonTime() time.Duration { return CocoonStepTime * CocoonMax }

const (
	silkwormLeavesName = "skygear:infesting_leaves"
	legacyInfestedName = "skygear:infested_leaves"
	// cocoonState is the one block state: the cocoon step, "p" in front
	// for persistent leaves. The values of 26.11.7 ("0" to "3") stay valid.
	cocoonState = "skygear:infest_stage"
	// logReach is how far leaves look for their log.
	logReach = 4
)

var (
	silkwormLeavesHash = block.NextHash()
	legacyInfestedHash = block.NextHash()
)

// leafMaterial is the look of a leaves texture: cut out, culled against its
// neighbours (alpha test draws the back faces too, which fought with the
// faces of the leaves beside it) and shaded with ambient occlusion like
// vanilla leaves (without it the block looked flat and lit at night).
func leafMaterial(texture string) customblock.Material {
	return customblock.NewMaterial(texture, customblock.AlphaTestSingleSidedRenderMethod()).WithAmbientOcclusion()
}

// cocoonTexture is the texture of a cocoon step: four pictures of the silk
// spreading, then the full one.
func cocoonTexture(p int) string {
	switch {
	case p >= CocoonMax:
		return "skygear_infested_leaves"
	case p >= 6:
		return "skygear_infesting_leaves_3"
	}
	return fmt.Sprintf("skygear_infesting_leaves_%d", p/2)
}

// SilkwormLeaves are leaves silkworms live in, spinning their cocoons.
type SilkwormLeaves struct {
	// Cocoon is the step of the cocoon, 0 to CocoonMax.
	Cocoon int
	// Persistent is kept from the leaves: placed leaves never decay.
	Persistent bool
}

// CocoonPercent is how far the cocoon is.
func (l SilkwormLeaves) CocoonPercent() int { return l.Cocoon * CocoonStep }

func (SilkwormLeaves) EncodeItem() (string, int16) { return silkwormLeavesName, 0 }

func (l SilkwormLeaves) stateValue() string {
	if l.Persistent {
		return fmt.Sprintf("p%d", l.Cocoon)
	}
	return fmt.Sprint(l.Cocoon)
}

func (l SilkwormLeaves) EncodeBlock() (string, map[string]any) {
	return silkwormLeavesName, map[string]any{cocoonState: l.stateValue()}
}

func (l SilkwormLeaves) Hash() (uint64, uint64) {
	h := uint64(l.Cocoon)
	if l.Persistent {
		h |= 1 << 8
	}
	return silkwormLeavesHash, h
}

func (SilkwormLeaves) Model() world.BlockModel { return model.Leaves{} }

func (SilkwormLeaves) Name() string { return i18n.T(i18n.Default, "name.SilkwormLeaves") }

func (SilkwormLeaves) Category() category.Category { return category.Nature() }

func (SilkwormLeaves) Properties() customblock.Properties {
	return customblock.Properties{Cube: true, Textures: map[string]customblock.Material{"*": leafMaterial(cocoonTexture(0))}}
}

// States implements block.Permutable.
func (SilkwormLeaves) States() map[string][]any {
	var values []any
	for _, persistent := range []bool{false, true} {
		for p := 0; p <= CocoonMax; p++ {
			values = append(values, SilkwormLeaves{Cocoon: p, Persistent: persistent}.stateValue())
		}
	}
	return map[string][]any{cocoonState: values}
}

// Permutations implements block.Permutable: the texture of each step.
func (SilkwormLeaves) Permutations() []customblock.Permutation {
	var out []customblock.Permutation
	for p := 1; p <= CocoonMax; p++ {
		if cocoonTexture(p) == cocoonTexture(0) {
			continue
		}
		cond := fmt.Sprintf("q.block_state('%[1]s') == '%[2]d' || q.block_state('%[1]s') == 'p%[2]d'", cocoonState, p)
		out = append(out, customblock.Permutation{
			Condition:  cond,
			Properties: customblock.Properties{Cube: true, Textures: map[string]customblock.Material{"*": leafMaterial(cocoonTexture(p))}},
		})
	}
	return out
}

// BreakInfo: full cocoons give string, more with a crook; leaves still
// spinning give nothing (the worms are lost).
func (l SilkwormLeaves) BreakInfo() block.BreakInfo {
	return leavesBreakInfo(func(t item.Tool, _ []item.Enchantment) []item.Stack {
		if l.Cocoon < CocoonMax {
			return nil
		}
		return CocoonDrops(t, rand.Float64)
	})
}

func (SilkwormLeaves) LightDiffusionLevel() uint8 { return 1 }

func (SilkwormLeaves) SideClosed(cube.Pos, cube.Pos, *world.Tx) bool { return false }

func (SilkwormLeaves) FlammabilityInfo() block.FlammabilityInfo {
	return block.FlammabilityInfo{Encouragement: 30, Flammability: 60, LavaFlammable: true}
}

// LeavesLike implements block.LeavesLike: leaves find their log through it.
func (SilkwormLeaves) LeavesLike() bool { return true }

// NeighbourUpdateTick decays the leaves when no log is left, as leaves do
// (at once: the decay that follows passes on to the next); placed leaves
// stay and pass the update on.
func (l SilkwormLeaves) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	if !l.Persistent && !FindLog(tx, pos) {
		tx.SetBlock(pos, nil, nil)
		for _, d := range l.BreakInfo().Drops(item.ToolNone{}, nil) {
			if s, ok := core.Of(tx); ok {
				s.DropAt(tx, pos.Vec3Centre(), d)
			}
		}
		return
	}
	wakeLeaves(tx, pos)
}

// ScheduledTick grows the cocoon a step.
func (SilkwormLeaves) ScheduledTick(pos cube.Pos, tx *world.Tx, r *rand.Rand) {
	if cur, ok := tx.Block(pos).(SilkwormLeaves); ok {
		GrowCocoon(tx, pos, cur, r)
	}
}

// RandomTick schedules the next step again, for leaves whose tick got lost
// (an item placed, a tick that did not survive): it never grows the cocoon.
func (l SilkwormLeaves) RandomTick(pos cube.Pos, tx *world.Tx, _ *rand.Rand) {
	if l.Cocoon < CocoonMax {
		tx.ScheduleBlockUpdate(pos, l, CocoonStepTime)
	}
}

// GrowCocoon grows the cocoon of l at pos a step, spreading the worms from
// SpreadFrom on, and schedules the next step until it is full.
func GrowCocoon(tx *world.Tx, pos cube.Pos, l SilkwormLeaves, r *rand.Rand) {
	if l.Cocoon >= CocoonMax {
		return
	}
	if l.Cocoon >= SpreadFrom && r.Float64() < SpreadChance {
		TrySpread(tx, pos, r)
	}
	l.Cocoon++
	tx.SetBlock(pos, l, nil)
	if l.Cocoon < CocoonMax {
		tx.ScheduleBlockUpdate(pos, l, CocoonStepTime)
	}
}

// legacyInfested is the infested leaves block of 26.11.7: full silkworm
// leaves, which it turns into on its next tick or update.
type legacyInfested struct{}

func (legacyInfested) EncodeItem() (string, int16) { return legacyInfestedName, 0 }

func (legacyInfested) EncodeBlock() (string, map[string]any) { return legacyInfestedName, nil }

func (legacyInfested) Hash() (uint64, uint64) { return legacyInfestedHash, 0 }

func (legacyInfested) Model() world.BlockModel { return model.Leaves{} }

func (legacyInfested) Name() string { return i18n.T(i18n.Default, "name.SilkwormLeaves") }

func (legacyInfested) Category() category.Category { return category.Nature() }

func (legacyInfested) Properties() customblock.Properties {
	return customblock.Properties{Cube: true, Textures: map[string]customblock.Material{"*": leafMaterial(cocoonTexture(CocoonMax))}}
}

func (legacyInfested) BreakInfo() block.BreakInfo {
	return SilkwormLeaves{Cocoon: CocoonMax}.BreakInfo()
}

func (legacyInfested) LightDiffusionLevel() uint8 { return 1 }

func (legacyInfested) SideClosed(cube.Pos, cube.Pos, *world.Tx) bool { return false }

func (legacyInfested) LeavesLike() bool { return true }

func (legacyInfested) upgrade(pos cube.Pos, tx *world.Tx) {
	tx.SetBlock(pos, SilkwormLeaves{Cocoon: CocoonMax}, nil)
}

func (b legacyInfested) RandomTick(pos cube.Pos, tx *world.Tx, _ *rand.Rand) { b.upgrade(pos, tx) }

func (b legacyInfested) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) { b.upgrade(pos, tx) }

// FindLog reports whether a log is within logReach of pos, counted through
// leaves and blocks like them (block.LeavesLike), as leaves look for theirs.
func FindLog(tx *world.Tx, pos cube.Pos) bool {
	visited, step := map[cube.Pos]struct{}{pos: {}}, []cube.Pos{pos}
	for distance := 0; len(step) > 0; distance++ {
		var next []cube.Pos
		for _, at := range step {
			b := tx.Block(at)
			switch b.(type) {
			case block.Log, block.Wood:
				return true
			}
			if !leafy(b) || distance >= logReach {
				continue
			}
			at.Neighbours(func(n cube.Pos) {
				if _, ok := visited[n]; !ok {
					visited[n] = struct{}{}
					next = append(next, n)
				}
			}, tx.Range())
		}
		step = next
	}
	return false
}

// leafy reports whether b is leaves or counts as leaves.
func leafy(b world.Block) bool {
	if _, ok := b.(block.Leaves); ok {
		return true
	}
	l, ok := b.(block.LeavesLike)
	return ok && l.LeavesLike()
}

// wakeLeaves gives the leaves reached from pos through silkworm leaves (up
// to logReach steps) a neighbour update, as leaves pass one on to each
// other: the leaves behind placed silkworm leaves still hear that their log
// was felled. The silkworm leaves themselves are not set again, so they do
// not keep waking each other.
func wakeLeaves(tx *world.Tx, pos cube.Pos) {
	visited, step := map[cube.Pos]struct{}{pos: {}}, []cube.Pos{pos}
	for distance := 0; distance < logReach && len(step) > 0; distance++ {
		var next []cube.Pos
		for _, at := range step {
			at.Neighbours(func(n cube.Pos) {
				if _, ok := visited[n]; ok {
					return
				}
				visited[n] = struct{}{}
				switch b := tx.Block(n).(type) {
				case block.Leaves:
					b.NeighbourUpdateTick(n, at, tx)
				case block.LeavesLike:
					if b.LeavesLike() {
						next = append(next, n)
					}
				}
			}, tx.Range())
		}
		step = next
	}
}

// CocoonDrops are the drops of full silkworm leaves broken with t: a string
// for each roll of the table that comes up (roll is rand.Float64 in play).
func CocoonDrops(t item.Tool, roll func() float64) []item.Stack {
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

// leavesBreakInfo is the break info of silkworm leaves: as soft as leaves,
// quickest with a crook (a hoe to the server) or shears.
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

// Infest turns the leaves at pos into silkworm leaves at 0% and schedules
// the first step, reporting whether there were leaves to infest.
func Infest(tx *world.Tx, pos cube.Pos) bool {
	leaves, ok := tx.Block(pos).(block.Leaves)
	if !ok {
		return false
	}
	l := SilkwormLeaves{Persistent: leaves.Persistent}
	tx.SetBlock(pos, l, nil)
	tx.ScheduleBlockUpdate(pos, l, CocoonStepTime)
	return true
}

// TrySpread spreads the worms from pos to one of the leaves in the 3×3×3
// around it, picked at random, on the island's own chunks; it reports
// whether any caught them.
func TrySpread(tx *world.Tx, pos cube.Pos, r *rand.Rand) bool {
	s, ok := core.Of(tx)
	if !ok {
		return false
	}
	var leaves []cube.Pos
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			for dz := -1; dz <= 1; dz++ {
				at := pos.Add(cube.Pos{dx, dy, dz})
				if at == pos || at.OutOfBounds(tx.Range()) || !island.CanBuild(s, at) {
					continue
				}
				if _, ok := tx.Block(at).(block.Leaves); ok {
					leaves = append(leaves, at)
				}
			}
		}
	}
	if len(leaves) == 0 {
		return false
	}
	return Infest(tx, leaves[r.IntN(len(leaves))])
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

// UseOnBlock implements item.UsableOnBlock: leaves turn into silkworm leaves.
func (Silkworm) UseOnBlock(pos cube.Pos, _ cube.Face, _ mgl64.Vec3, tx *world.Tx, _ item.User, ctx *item.UseContext) bool {
	if !Infest(tx, pos) {
		return false
	}
	tx.PlaySound(pos.Vec3Centre(), sound.ItemUseOn{Block: block.Leaves{Type: block.OakLeaves()}})
	ctx.SubtractFromCount(1)
	return true
}

// SmeltInfo implements item.Smeltable: a furnace, smoker or campfire cooks a
// silkworm into a cooked silkworm.
func (Silkworm) SmeltInfo() item.SmeltInfo {
	return item.SmeltInfo{Product: item.NewStack(CookedSilkworm{}, 1), Experience: 0.1, Food: true}
}

// CookedSilkworm is a silkworm cooked in a furnace: food for the time before
// the first farm (Fabrico's silkworm_cooked).
type CookedSilkworm struct{}

func (CookedSilkworm) EncodeItem() (string, int16) { return "skygear:cooked_silkworm", 0 }

func (CookedSilkworm) Name() string { return i18n.T(i18n.Default, "name.CookedSilkworm") }

func (CookedSilkworm) Category() category.Category { return category.Items() }

// AlwaysConsumable, ConsumeDuration and Consume implement item.Consumable.
func (CookedSilkworm) AlwaysConsumable() bool { return false }

func (CookedSilkworm) ConsumeDuration() time.Duration { return item.DefaultConsumeDuration }

func (CookedSilkworm) Consume(_ *world.Tx, c item.Consumer) item.Stack {
	c.Saturate(CookedSilkwormFood, CookedSilkwormSaturation)
	return item.Stack{}
}
