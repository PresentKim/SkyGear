package sgitem

import (
	"github.com/df-mc/dragonfly/server/world"
)

// The items and blocks of this package register with the world on import.
func init() {
	world.RegisterItem(Hammer{})
	world.RegisterItem(Hammer{Stone: true})
	for _, k := range CrushedKinds {
		world.RegisterBlock(CrushedBlock{Kind: k})
		world.RegisterItem(CrushedBlock{Kind: k})
	}
	for _, metal := range NewOreMetals {
		world.RegisterBlock(OreBlock{Metal: metal})
		world.RegisterItem(OreBlock{Metal: metal})
		world.RegisterItem(RawOre{Metal: metal})
	}
	for _, k := range OreKinds {
		world.RegisterItem(OreChunk{Ore: k.Ore})
	}
	OreChunkRecipes()
	for _, m := range DustMetals {
		world.RegisterItem(MetalDust{Metal: m})
	}
	for _, m := range IngotMetals {
		world.RegisterItem(Ingot{Metal: m})
	}
	world.RegisterItem(WoodenBucket{})
	world.RegisterItem(WoodenBucket{Water: true})
}
