package skygear

import (
	"log/slog"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/world"
)

// CustomBlockCounts are the custom blocks registered, their block states (for a
// block with states, the combinations of its values) and their permutations:
// what the client has to load, which grows with both.
func CustomBlockCounts() (blocks, states, permutations int) {
	for _, b := range world.CustomBlocks() {
		blocks++
		p, ok := b.(block.Permutable)
		if !ok {
			states++
			continue
		}
		n := 1
		for _, values := range p.States() {
			n *= len(values)
		}
		states += n
		permutations += len(p.Permutations())
	}
	return blocks, states, permutations
}

// LogCustomBlockCounts logs CustomBlockCounts once, for comparing client
// loading times between state probe modes.
func LogCustomBlockCounts(log *slog.Logger) {
	blocks, states, permutations := CustomBlockCounts()
	log.Info("custom blocks", "blocks", blocks, "states", states, "permutations", permutations)
}
