package waila

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

// View is who looks at a block and where the block is.
type View struct {
	// Lang is the viewer's language.
	Lang string
	// Sneaking is whether the viewer sneaks: blocks add their details then.
	Sneaking bool
	// Tx and Pos are the transaction and the position of the block.
	Tx  *world.Tx
	Pos cube.Pos
}

// Provider is a block that fills in its own panel.
type Provider interface {
	// Waila returns info, which holds the block's name, icon, tools and the
	// contents of a vanilla container, with what the block adds to it.
	Waila(v View, info Info) Info
}

// Adapter fills in the panel of blocks that have no Waila method of their
// own, such as vanilla blocks; it returns info unchanged for any other block.
type Adapter func(v View, b world.Block, info Info) Info

var adapters []Adapter

// RegisterAdapter adds an adapter. Adapters run in the order they were added,
// after the block's own Waila.
func RegisterAdapter(a Adapter) { adapters = append(adapters, a) }

// Describe returns info with what block b adds to its panel: its Waila if it
// is a Provider, then every adapter.
func Describe(v View, b world.Block, info Info) Info {
	if p, ok := b.(Provider); ok {
		info = p.Waila(v, info)
	}
	for _, a := range adapters {
		info = a(v, b, info)
	}
	return info
}
