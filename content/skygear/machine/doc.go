// Package machine is the automation blocks of the game. Most run on energy
// (RE) and stop without it; each operation takes a fixed time. Items go in
// and out by hand in the machine window (an entity chest) or by hoppers: a
// hopper above or beside a machine inserts into its input, a hopper below
// extracts from its output.
//
// One block type, Machine, is every machine; its Kind names a Def in Kinds
// (machine.go): the window layout, the timing and the energy of the kind.
// A machine's State is the inventory and progress of one placed machine,
// with the fields every kind needs.
//
//   - Energy (generator.go): generators burn fuel into RE, energy cells store
//     it, the other machines spend it from their buffer (network.Energy).
//   - Fluids (fluidmachine.go): tanks, the crucible, the lava generator and
//     the ore washer hold a fluid (network.Fluids).
//   - The kinds that work on items: the crusher and the miner (machine.go),
//     the sieves and their meshes (sieve.go, magicsieve.go, mesh.go), the
//     alloy smelter, the electric furnace, the item vacuum and the tree
//     feller (alloy.go, autoequip.go), the auto crafter (autocraft.go), the
//     laboratory (lab.go, labui.go) and the mob simulator (essence.go).
//   - Tiers and looks (tier.go, tierlook.go, animate.go): tier kits make a
//     machine faster; some kinds show their tier and whether they work.
//   - What the island's machines did (machinecount.go, stats.go,
//     monitor.go): the achievement counts, the production record and the
//     stall alerts.
//
// The kinds of other parts of the game (the storage controller, the
// dimension gate, the harvester) set the hooks of their Def: Click, Run,
// Insert, Peek and Extract.
package machine
