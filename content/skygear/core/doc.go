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
// Hooks are how a feature reacts to the server. A hook is a typed event with
// subscribers in six tiers (Priority: Lowest to Monitor), each tier with one
// role; the subscribers of one tier have no order among themselves. No
// subscriber can stop the others: every subscriber of a Hook is called. A
// CancelHook is the hook of an event that can be cancelled (it embeds Actor):
// a subscriber cancels it with e.Cancel(), which keeps the action from
// happening in Dragonfly, and from then on only the subscribers registered
// with the HandleCancelled option are called. So a subscriber acts by its own
// conditions, and cancels only when the action or the later subscribers must
// not happen. PlayerHandler and WorldHandler are the Dragonfly handlers that
// emit the player and world hooks, and Lifecycle makes the session calls of
// the instance (join, leave, new day, every second, fast, autosave, close)
// emit theirs. The events carry the arguments Dragonfly passed, including its
// pointers.
//
// A Feature is what a feature package hands to the game: its name, entity
// types, settings, item entry edit, creative items, commands, a Setup that
// subscribes its steps and a Check that Install runs when every feature is set
// up. The root package lists the features and passes them to Install, which
// runs each Setup under the feature's name; a subscriber is named
// "<feature>.<step>". EditEntries and CheckEntries compose the client
// entry edits of the features, CreativeItems orders the creative items of the
// features and of the content list by Slot, and Commands collects the
// features' commands (PlayerSession finds the player and session a command
// runs for). The kernel imports no feature package.
//
// A feature declares its parts as package variables and reads them with
// Of(s); it never adds fields to Session.
package core
