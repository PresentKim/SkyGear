// Package hooktest records what a joining client is sent. It runs a real
// Dragonfly server (its block and item encoding, start game data, item
// registry, crafting data and creative content) behind sghook's wrapped
// listener, with one fake connection, and keeps every packet and the
// start game data the server wrote to it.
//
// Canon turns any value into a stable text, Golden compares the hashes of
// such texts to a file under testdata. A change of Dragonfly or of the
// patch that alters what a client receives makes Golden fail and name the
// entries that changed.
//
// Re-record a golden file after a deliberate change:
//
//	SKYGEAR_GOLDEN=update go test ./path/to/package
//
// To see what changed inside an entry, dump the texts from two trees and
// compare them:
//
//	SKYGEAR_GOLDEN_DUMP=/tmp/a go test ./path/to/package   # tree one
//	SKYGEAR_GOLDEN_DUMP=/tmp/b go test ./path/to/package   # tree two
//	diff -r /tmp/a /tmp/b
//
// The package is test support. The server binary does not import it.
package hooktest
