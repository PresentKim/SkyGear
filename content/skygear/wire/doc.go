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
//   - creative.golden: the creative groups and items;
//   - items.byname.golden, recipes.byname.golden, creative.byname.golden:
//     the entries of the three files above with every item id written as the
//     item's name, and the ids of the custom items as a set
//     (meta/custom_ids).
//
// A custom item's id is its place in the order the items are registered in,
// which follows the order the packages are initialised in. A change of that
// order alone changes hundreds of entries of items.golden, recipes.golden
// and creative.golden and leaves the three files by name byte for byte as
// they are: after re-recording, git status then lists none of them, which
// shows that nothing but the numbering changed.
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
