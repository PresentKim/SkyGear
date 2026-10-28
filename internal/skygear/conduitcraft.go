package skygear

import "github.com/df-mc/dragonfly/server/world"

// conduitOf returns the conduit of a kind ("energy", "item" or "fluid") and tier.
func conduitOf(kind string, tier int) world.Item {
	switch kind {
	case "energy":
		return EnergyConduit{Tier: tier}
	case "item":
		return ItemConduit{Tier: tier}
	}
	return FluidConduit{Tier: tier}
}
