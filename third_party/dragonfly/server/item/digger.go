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

// TagDigger is a custom item that tells clients how fast it mines blocks by
// their tags (the minecraft:digger item component's block tags), so one
// entry covers every vanilla and custom block that carries the tag.
// (skygear patch)
type TagDigger interface {
	// DestroySpeedsByTag are the item's mining speeds, each on the blocks a
	// Molang tag query matches (e.g.
	// "query.any_tag('minecraft:is_pickaxe_item_destructible')").
	DestroySpeedsByTag() []TagSpeed
}

// TagSpeed is a mining speed on the blocks a Molang tag query matches.
// (skygear patch)
type TagSpeed struct {
	Tags  string
	Speed int
}
