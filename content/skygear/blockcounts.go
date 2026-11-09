package skygear

import (
	"log/slog"
	"sort"
	"strings"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/world"
)

// blockNameCount is a custom block name with its block states and permutations.
type blockNameCount struct {
	name                 string
	states, permutations int
}

// customBlockNameCounts are the custom blocks registered by name (sorted), each
// with its block states (for a block with states, the combinations of its
// values) and its permutations.
func customBlockNameCounts() []blockNameCount {
	var out []blockNameCount
	for name, b := range world.CustomBlocks() {
		c := blockNameCount{name: name, states: 1}
		if p, ok := b.(block.Permutable); ok {
			for _, values := range p.States() {
				c.states *= len(values)
			}
			c.permutations = len(p.Permutations())
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

// CustomBlockCounts are the custom blocks registered, their block states (for a
// block with states, the combinations of its values) and their permutations:
// what the client has to load, which grows with both.
func CustomBlockCounts() (blocks, states, permutations int) {
	for _, c := range customBlockNameCounts() {
		blocks++
		states += c.states
		permutations += c.permutations
	}
	return blocks, states, permutations
}

// LogCustomBlockCounts logs CustomBlockCounts once, for comparing client
// loading times with the state probe on and off, with the most states of a name, a
// line for each state probe block name and a warning for each name over
// stateProbeNameCap states.
func LogCustomBlockCounts(log *slog.Logger) {
	counts := customBlockNameCounts()
	var blocks, states, permutations, most int
	var mostName string
	for _, c := range counts {
		blocks++
		states += c.states
		permutations += c.permutations
		if c.states > most {
			most, mostName = c.states, c.name
		}
	}
	log.Info("custom blocks", "blocks", blocks, "states", states, "permutations", permutations,
		"max_states_per_name", most, "max_name", mostName)
	logBlockNameCounts(log, counts)
}

// logBlockNameCounts logs a line for each state probe block name and a warning
// for each name with more states than the client is thought to know.
func logBlockNameCounts(log *slog.Logger, counts []blockNameCount) {
	for _, c := range counts {
		if strings.HasPrefix(c.name, "skygear:state_probe") {
			log.Info("state probe block", "name", c.name, "states", c.states, "permutations", c.permutations)
		}
		if c.states > stateProbeNameCap {
			log.Warn("custom block has more states than the client may know", "name", c.name, "states", c.states, "cap", stateProbeNameCap)
		}
	}
}
