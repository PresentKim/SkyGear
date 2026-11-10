package conduit

import (
	"github.com/df-mc/dragonfly/server/world"
)

// Of returns the conduit of a kind ("energy", "item", "fluid" or "net") and
// tier; the network conduit has one tier.
func Of(kind string, tier int) world.Item {
	switch kind {
	case "net":
		return NetConduit{}
	case "energy":
		return EnergyConduit{Tier: tier}
	case "item":
		return ItemConduit{Tier: tier}
	}
	return FluidConduit{Tier: tier}
}
