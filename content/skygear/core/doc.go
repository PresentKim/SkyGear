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
// A feature declares its parts as package variables and reads them with
// Of(s); it never adds fields to Session.
package core
