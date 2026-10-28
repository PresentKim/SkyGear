package block

import (
	"math/rand/v2"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// Leaves are blocks that grow as part of trees which mainly drop saplings and sticks.
type Leaves struct {
	leaves
	sourceWaterDisplacer

	// Type is the type of the leaves.
	Type LeavesType
	// Persistent specifies if the leaves are persistent, meaning they will not decay as a result of no wood
	// being nearby.
	Persistent bool

	ShouldUpdate bool
}

// UseOnBlock makes leaves persistent when they are placed so that they don't decay.
func (l Leaves) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) (used bool) {
	pos, _, used = firstReplaceable(tx, pos, face, l)
	if !used {
		return
	}
	l.Persistent = true

	place(tx, pos, l, user, ctx)
	return placed(ctx)
}

// findLog returns whether a log is within four blocks of the leaves at the position passed, counted through the
// leaves between them. The blocks around it are searched a step at a time, so each is reached the shortest way.
func findLog(pos cube.Pos, tx *world.Tx) bool {
	visited, step := map[cube.Pos]struct{}{pos: {}}, []cube.Pos{pos}
	for distance := 0; len(step) > 0; distance++ {
		var next []cube.Pos
		for _, at := range step {
			switch tx.Block(at).(type) {
			case Log, Wood:
				return true
			}
			if _, ok := tx.Block(at).(Leaves); !ok || distance >= 4 {
				continue
			}
			at.Neighbours(func(neighbour cube.Pos) {
				if _, ok := visited[neighbour]; ok {
					return
				}
				visited[neighbour] = struct{}{}
				next = append(next, neighbour)
			}, tx.Range())
		}
		step = next
	}
	return false
}

// LeavesDecayDelay is the time the game gives leaves that lost their log to
// decay, as a minimum and a random extra, via a scheduled tick (skygear
// patch). Both zero keeps vanilla: decay waits for a random tick.
var (
	LeavesDecayDelayMin    time.Duration
	LeavesDecayDelayJitter time.Duration
)

// decay removes the leaves at pos if no log is near, dropping what leaves
// drop; with a log near they stay (skygear patch: shared by the random
// tick and the scheduled tick).
func (l Leaves) decay(pos cube.Pos, tx *world.Tx) {
	if l.Persistent || !l.ShouldUpdate {
		return
	}
	if findLog(pos, tx) {
		l.ShouldUpdate = false
		tx.SetBlock(pos, l, nil)
		return
	}
	ctx := tx.Event()
	if tx.World().Handler().HandleLeavesDecay(ctx, pos); ctx.Cancelled() {
		// Prevent immediate re-updating.
		l.ShouldUpdate = false
		tx.SetBlock(pos, l, nil)
		return
	}
	tx.SetBlock(pos, nil, nil)
	for _, drop := range l.BreakInfo().Drops(item.ToolNone{}, nil) {
		dropItem(tx, drop, pos.Vec3Centre())
	}
}

// RandomTick ...
func (l Leaves) RandomTick(pos cube.Pos, tx *world.Tx, _ *rand.Rand) {
	l.decay(pos, tx)
}

// ScheduledTick decays leaves whose log is gone (skygear patch).
func (l Leaves) ScheduledTick(pos cube.Pos, tx *world.Tx, _ *rand.Rand) {
	if cur, ok := tx.Block(pos).(Leaves); ok {
		cur.decay(pos, tx)
	}
}

// NeighbourUpdateTick ...
func (l Leaves) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	if !l.Persistent && !l.ShouldUpdate {
		l.ShouldUpdate = true
		tx.SetBlock(pos, l, nil)
		if LeavesDecayDelayMin > 0 || LeavesDecayDelayJitter > 0 {
			delay := LeavesDecayDelayMin
			if LeavesDecayDelayJitter > 0 {
				delay += time.Duration(rand.Int64N(int64(LeavesDecayDelayJitter)))
			}
			tx.ScheduleBlockUpdate(pos, l, delay)
		}
	}
}

// FlammabilityInfo ...
func (l Leaves) FlammabilityInfo() FlammabilityInfo {
	return newFlammabilityInfo(30, 60, true)
}

// LeavesSaplingChances are the chances of leaves dropping a sapling when
// broken or decayed, by fortune level 0 to 3 (vanilla values). skygear
// patch: variables instead of constants.
var LeavesSaplingChances = []float64{0.05, 0.0625, 0.083333336, 0.1}

// LeavesStickChances are the chances of leaves dropping sticks, by fortune
// level 0 to 3 (vanilla values; skygear patch: a variable).
var LeavesStickChances = []float64{0.02, 0.022222222, 0.025, 0.033333333}

// JungleLeavesSaplingChances are LeavesSaplingChances for jungle leaves.
var JungleLeavesSaplingChances = []float64{0.025, 0.027777778, 0.03125, 0.041666668}

// BreakInfo ...
func (l Leaves) BreakInfo() BreakInfo {
	return newBreakInfo(0.2, alwaysHarvestable, func(t item.Tool) bool {
		return t.ToolType() == item.TypeShears || t.ToolType() == item.TypeHoe
	}, func(t item.Tool, enchantments []item.Enchantment) []item.Stack {
		if t.ToolType() == item.TypeShears || hasSilkTouch(enchantments) {
			return []item.Stack{item.NewStack(l, 1)}
		}
		fortune := fortuneLevel(enchantments)
		var drops []item.Stack

		if sapling, ok := l.Type.Sapling(); ok {
			// skygear patch: the chances are variables the game can raise.
			chances := LeavesSaplingChances
			if l.Type == JungleLeaves() {
				chances = JungleLeavesSaplingChances
			}
			if rand.Float64() < chances[min(fortune, 3)] {
				drops = append(drops, item.NewStack(Sapling{Type: sapling}, 1))
			}
		}
		if rand.Float64() < LeavesStickChances[min(fortune, 3)] {
			drops = append(drops, item.NewStack(item.Stick{}, rand.IntN(2)+1))
		}
		if wood, ok := l.Type.Wood(); ok && (wood == OakWood() || wood == DarkOakWood()) {
			appleChances := []float64{0.005, 0.005555556, 0.00625, 0.008333333}
			if rand.Float64() < appleChances[min(fortune, 3)] {
				drops = append(drops, item.NewStack(item.Apple{}, 1))
			}
		}
		return drops
	})
}

// CompostChance ...
func (Leaves) CompostChance() float64 {
	return 0.3
}

// EncodeItem ...
func (l Leaves) EncodeItem() (name string, meta int16) {
	return "minecraft:" + l.Type.String(), 0
}

// LightDiffusionLevel ...
func (Leaves) LightDiffusionLevel() uint8 {
	return 1
}

// SideClosed ...
func (Leaves) SideClosed(cube.Pos, cube.Pos, *world.Tx) bool {
	return false
}

// EncodeBlock ...
func (l Leaves) EncodeBlock() (name string, properties map[string]any) {
	return "minecraft:" + l.Type.String(), map[string]any{"persistent_bit": l.Persistent, "update_bit": l.ShouldUpdate}
}

// allLeaves returns a list of all possible leaves states.
func allLeaves() (leaves []world.Block) {
	f := func(persistent, update bool) {
		for _, t := range LeavesTypes() {
			leaves = append(leaves, Leaves{Type: t, Persistent: persistent, ShouldUpdate: update})
		}
	}
	f(true, true)
	f(true, false)
	f(false, true)
	f(false, false)
	return
}
