package dragonfactory

import (
	"time"

	"github.com/df-mc/dragonfly/server/block"
)

// Leaves (docs/design/02-dragonfactory.md §2). The block stays Dragonfly's
// Leaves (trees, shears, the textures and old worlds all depend on it); the
// dragonfactory patch makes its numbers variables the game sets here.
//   - Saplings drop far more often than vanilla's 5%: without a sapling the
//     start is stuck, and a sapling grows into a tree whose leaves give
//     more. Fortune raises it a little; jungle leaves drop half as often.
//   - No sticks: a stick can only come after its research.
//   - Leaves that lost their log decay within about a second instead of
//     waiting for a random tick, and the decay passes on to the leaves
//     beside them, so a felled tree's crown is gone in a few seconds.
func init() {
	block.LeavesSaplingChances = []float64{0.2, 0.22, 0.25, 0.3}
	block.JungleLeavesSaplingChances = []float64{0.1, 0.11, 0.125, 0.15}
	block.LeavesStickChances = []float64{0, 0, 0, 0}
	block.LeavesDecayDelayMin = 250 * time.Millisecond
	block.LeavesDecayDelayJitter = 750 * time.Millisecond
}
