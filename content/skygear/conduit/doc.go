// Package conduit is the conduits of the game, as in EnderIO: energy, item
// and fluid conduits of three tiers, placed into one block per position, the
// Bundle (bundle.go), which holds at most one conduit of each kind. Bundles
// next to each other join into the save's networks (network.Energy, Items,
// Fluids; conduit.go). A display entity draws each bundle (conduitvisual.go)
// and connector plates mark where an item conduit meets an inventory
// (conduitplate.go); a plate's screen sets what goes in and out of that side
// on which channel (conduitform.go).
package conduit
