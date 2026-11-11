// Package core is what every Sky Gear feature builds on: the Session of one
// running save and the ways a feature keeps its own data in it.
//
//   - Session: the save's world, clock, random source and ids, and helpers
//     every feature uses (dropping items, messaging everyone, debouncing
//     screens). Of finds the session of a transaction's world.
//   - Part: a feature's runtime data in each session (not saved), made on
//     first use.
//   - Players: a feature's data per player in each session, dropped when the
//     player leaves.
//   - Saved: a feature's saved data. All Saved parts are fields of one JSON
//     document per save (the "skygear.state" key of the save's state), so
//     each part owns some of its fields and a field belongs to one part only.
//
// Hooks are how a feature reacts to the server. A Hook is a typed event with
// subscribers in explicit priority order (On never stops the chain, Handle
// stops it by returning true). PlayerHandler and WorldHandler are the
// Dragonfly handlers that emit the player and world hooks, and Lifecycle makes
// the session calls of the instance (join, leave, new day, every second, fast,
// autosave, close) emit theirs. The events carry the arguments Dragonfly
// passed, including its pointers, and an Actor to cancel the event in
// Dragonfly; cancelling and stopping the chain are independent.
//
// A Feature is what a feature package hands to the game: its name, entity
// types, settings, item entry edit, creative items, commands and a Setup that
// subscribes its steps. The root package lists the features and passes them to
// Install, which runs each Setup under the feature's name; a subscriber is
// named "<feature>.<step>". EditEntries and CheckEntries compose the client
// entry edits of the features, CreativeItems orders the creative items of the
// features and of the content list by Slot, and Commands collects the
// features' commands (PlayerSession finds the player and session a command
// runs for). The kernel imports no feature package.
//
// A feature declares its parts as package variables and reads them with
// Of(s); it never adds fields to Session.
package core
