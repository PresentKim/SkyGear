package item

// Digger is a custom item that tells clients how fast it mines certain
// blocks (the minecraft:digger item component). Without it, clients mine
// with a custom item at bare-hand speed whatever the server computes.
// (skygear patch)
type Digger interface {
	// DestroySpeeds maps block names (e.g. "minecraft:stone") to the mining
	// speed of the item on them (a tool tier efficiency, e.g. 4 for stone).
	DestroySpeeds() map[string]int
}
