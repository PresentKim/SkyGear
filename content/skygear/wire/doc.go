// Package wire pins the data a joining player is sent as golden files, so a
// change of Dragonfly or of its patch that alters what clients get fails a
// test that names the thing that changed.
//
// One join runs the server as cmd/skygear does (entity registry, item entry
// edit, skygear.New with all blocks, items, recipes and the creative list).
// testdata holds one line per entry, the sha256 of its canonical text:
//
//   - blocks.golden: the start game block entries with every component and
//     permutation, one per block (block_id is zeroed and permutations are
//     sorted by condition, as both follow map order per process; the set of
//     ids is meta/block_ids);
//   - items.golden: the item registry packet (registry/<name>) and the start
//     game items (start/<name>), with their components;
//   - recipes.golden: the crafting data, <kind>/<network id>;
//   - creative.golden: the creative groups and items.
//
// A golden changes only in a commit meant to change what clients get.
// Re-record after such a change:
//
//	SKYGEAR_GOLDEN=update go test ./content/skygear/wire/
//
// To see what changed inside an entry, dump the texts of both trees and
// compare them:
//
//	SKYGEAR_GOLDEN_DUMP=/tmp/a go test ./content/skygear/wire/
//	SKYGEAR_GOLDEN_DUMP=/tmp/b go test ./content/skygear/wire/
//	diff -r /tmp/a /tmp/b
//
// The test also shows the entries are sensitive: it changes one value of
// one block and one item and expects exactly that entry to change.
package wire
