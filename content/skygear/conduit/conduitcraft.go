package conduit

import (
	"github.com/df-mc/dragonfly/server/world"
)

// Of returns the conduit of a kind ("energy", "item" or "fluid") and tier.
func Of(kind string, tier int) world.Item {
	switch kind {
	case "energy":
		return EnergyConduit{Tier: tier}
	case "item":
		return ItemConduit{Tier: tier}
	}
	return FluidConduit{Tier: tier}
}
