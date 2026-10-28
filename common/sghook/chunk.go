package sghook

import (
	_ "unsafe" // go:linkname

	"github.com/df-mc/dragonfly/server/world"
)

// Column returns the world's chunk at pos in tx, loading it first if it is
// not loaded. Changes to its biomes are saved with the world only if the
// chunk is saved for another change.
func Column(tx *world.Tx, pos world.ChunkPos) *world.Column { return tx_chunk(tx, pos) }

//go:linkname tx_chunk github.com/df-mc/dragonfly/server/world.(*Tx).chunk
func tx_chunk(*world.Tx, world.ChunkPos) *world.Column
