package dimension

import "math"

// The spinning blades (gear.go): blades turn around the hunter and hurt every
// monster they pass.

const (
	// bladeInner and bladeOuter are where a blade starts and ends, from the
	// hunter; bladeTurn is how far the blades turn a tick (a turn in 1.5 s).
	bladeInner, bladeOuter = 1.2, 2.6
	bladeTurn              = 2 * math.Pi / 30
	// bladeReach is how close to a blade a monster is hit.
	bladeReach = 0.8
	// bladeDamage is a hit's damage at grade 1, level 1; a monster is hit
	// again after bladeCooldown ticks at the soonest.
	bladeDamage   = 4.0
	bladeCooldown = 10
)
