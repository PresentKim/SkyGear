# The tinker owns its machine kind: reference for T108 to T113

Follow-up to part 3-4 (`2026-11-10-cleanup-restructure-p3-machine.md`, decision 6) and to row R9 of part 3-3. The machine plan is `2026-11-11-tinker-owns-its-kind-plan.md`. This file holds the decisions, the measurements, the proof and, per task, the exact code.

Everything here was measured at b49fc81 (first at cd8599f, then all six states again at b49fc81). The tasks were built in id order in a scratch worktree; after each of them `go build ./...`, `go vet ./...`, `gofmt -l` and `go test ./...` (62 packages with tests) were run, with the result section 5 lists. The patches of that build are in `/tmp/claude-0/-home-user-SkyGear/a71fa8d5-6f20-5c3f-ba20-eeb27e13ad8a/scratchpad/proto-tinker-tasks/` (`T108.patch` to `T113.patch`, made in id order on b49fc81 with `git diff --binary`, and `combined.patch`). They are a reference for a task's exact code, not something to apply whole: a task copies its blocks of section 6, and records its goldens and its fixture itself with the commands given there. Where a block and the code a developer finds disagree, the developer stops and reports (`NEEDS_CONTEXT`); it does not choose.

## 1. What changes

- The tinker table becomes the seventeenth kind another feature owns: `Owner: "tinker"` in `machine.Kinds`, the behaviour in the new `tinker/kinds.go` (`registerKinds`, called last in `tinker`'s `subscribe`), `machine/tinkertable.go` deleted. `machine` imports no package of the feature layer; `tinker` imports `machine`.
- The hopper's blocks and items are registered by the package that declares them: `hopper.Register` becomes that package's `init`, and the root's `hopper.go` is deleted.
- Both changes change the order custom items are registered in, so their numeric ids. Four goldens are re-recorded once (T112). Nothing else a client receives, nothing a save holds and no pin changes: sections 3.1 to 3.5 show it, and T108 and T109 put the proof into the repository before the change.

## 2. Decisions

They bind. One line of reason each.

1. `machine.Kinds[KindTinker]` gets `Owner: "tinker"`; `tinker` registers `Click` (debounce, `ShowTable`, true) and `Lines: machine.NoLines`, the two hooks `machine/tinkertable.go` set, with the same bodies. No frame code changes, so no benchmark is run: `Register` writes into the `*Def` the old `init` wrote into.
2. `registerKinds()` is the last call of `tinker`'s `subscribe()`, as in `storage`, `dimension`, `farm` and `netstore`.
3. `Features` in `content/skygear/features.go` is not edited: `tinker.Feature` stays before `autofill.Feature` and `machine.Feature`. Measured: with `tinker.Feature` after `machine.Feature`, `settings.golden` changes in lines 07 to 09 (the order of the settings a player sees) and `TestFeatureNames` fails. The list is the order of settings and steps, not of imports. The tinker's `Setup` therefore calls `machine.Register` before the machine feature's `Setup` has run; `Register` needs only `machine.Kinds`, a package variable, and `CheckKinds` runs after every `Setup`.
4. `layers` and `featureLayer` in `layers_test.go` say `"machine", "tinker"` (a package imports only the packages before it). `siblings`: the row `"machine": {"tinker"}` goes, the row `"tinker": {"machine"}` comes. `machineFeatureImports` goes. `TestMachineImportsNoFeature` stays and loses its exception: `machine` has no `siblings` row and imports no package of `featureLayer`.
5. `kindOwners` in the root's `kinds_test.go` gets `machine.KindTinker: "tinker"`.
6. `content/skygear/debug_ui.go` is not touched (`k != machine.KindTinker` is already listed in `docs/TODO.md`).
7. The hopper moves to its own `init` (section 3.7: where it belongs, and why it is clean). `hopper.Register` has one caller, the root's `hopper.go`; both go.
8. One re-record. T110 and T111 each leave the tree red in exactly the golden tests their sections name and are not committed; `SKYGEAR_GOLDEN=update` is forbidden in them. T112 re-records, and T110, T111 and T112 are one commit. Every commit is green: T108, T109, that one commit, T113.
9. The proof is in the repository, as tests that keep running: the goldens by name (T108) and the old save (T109). They are recorded from the code before the change and must not change afterwards. A one-off script would have to be trusted once; a golden that `git status` does not list after the re-record can be checked by anyone.
10. A text by name is made by `hooktest.CanonWith`, a hook for struct fields, not by replacing text: a map key that happens to be called `NetworkID` inside an item's NBT cannot be taken for an id.
11. The item id fields are `protocol.ItemEntry.RuntimeID`, `protocol.ItemType.NetworkID` and `protocol.DefaultItemDescriptor.NetworkID`. An id field that is not listed stays a number in the text by name; if a custom item's id were in one, the goldens by name would change with the numbering and the proof would fail, which is the safe side. Measured: they do not change.
12. `popup.golden` gets a twin by name too (`popup.byname.golden`): its two changed entries hold a block runtime id, not an item id (section 3.1).
13. No migration patch and no save version change: nothing stored holds a numeric id (section 3.3).

## 3. Measurements

### 3.1 The goldens: which files, which entries, what kind of change

`SKYGEAR_GOLDEN_DUMP` texts of the base and of the final state were compared with `diff -r`; the counts are entries (one line of the golden file each).

| File | Entries | After T110 (hopper) | After T111 (hopper and tinker) | What differs inside a changed entry |
|---|---|---|---|---|
| `wire/testdata/items.golden` | 2,778 (349 `registry/`, 2,429 `start/`) | 698 | 698 | one line, `RuntimeID: int16(N)` |
| `wire/testdata/recipes.golden` | 3,907 | 327 | 327 | one line, `NetworkID: int32(N)` of the result (ingredients are sent by name) |
| `wire/testdata/creative.golden` | 1,877 (127 groups, 1,749 items, `order/items`) | 243 | 243 | one line, `NetworkID: int32(N)` |
| `dimension/testdata/popup.golden` | 32 | 0 | 2 (`popup1/001 *packet.AddActor`, `popup2/001 *packet.AddActor`) | one line, `uint32(2): int32(20943)` becomes `int32(20934)` |
| `wire/testdata/blocks.golden` | 206 | 0 | 0 | |
| the four goldens by name (T108) | 2,779, 3,907, 1,877, 32 | 0 | 0 | |
| the nine files of `content/skygear/testdata/pin/` | | 0 | 0 | |
| `content/skygear/testdata/save/oldsave.golden` (T109) | 2,822 | 0 | 0 | |

No entry is added or removed in any file. `git diff --numstat` after the re-record: `698 698 items.golden`, `327 327 recipes.golden`, `243 243 creative.golden`, `2 2 popup.golden`. With the tinker change alone (measured at cd8599f, without the hopper) the counts are the 512, 274, 155 and 2 of the earlier reference.

Why these numbers. A custom item's runtime id is `len(itemNamesToRuntimeIDs)` at the moment `world.RegisterItem` is called (`third_party/dragonfly/server/world/item.go:42`), so the 349 custom items have the ids 2080 to 2428 in registration order.

| Run of items | Count | Ids before | Ids after | Shift |
|---|---|---|---|---|
| the hoppers (`wood_hopper`, `wood_hopper_reverse`, `iron_hopper_reverse`) | 3 | 2426 to 2428 | 2080 to 2082 | -346 |
| `skygear:hammer` to `skygear:research_table` (`sgitem`, `elevator`, `handwork`, `progress`) | 90 | 2080 to 2169 | 2083 to 2172 | +3 |
| the tinker's tools (`tool_pickaxe_flint_wood` to `tool_shovel_diamond_blaze`) | 63 | 2170 to 2232 | 2366 to 2428 | +196 |
| `skygear:essence_zombie` to `skygear:stack_upgrade` (`machine`, `conduit`, `dimension`, `farm`, `fairy`, `storage`, `copier`, `netstore`) | 193 | 2233 to 2425 | 2173 to 2365 | -60 |

All 349 ids change, the set of ids is the same (2080 to 2428), the 349 names are the same, and inside each run the order is the same. 698 is 349 twice (the registry packet and the start game data). 327 recipes give a custom item, 243 creative entries are custom items (the 243 of the pin `creative.golden`).

The two entries of `popup.golden` are of another kind. The number is the runtime id of the air block, which `common/sghook/entity.go:36` gives a damage number as its variant, in the test binary of `dimension`. A block's runtime id is its place among all registered block states sorted by the hash of the name (`block_registry.go`, `Finalize`), so it depends on which packages the binary links. Before the change `dimension` linked `tinker` through `machine`, and through `tinker` the package `preview`, nine of whose sixteen glass blocks sort before air; afterwards it does not, and air is nine lower. In the game binary nothing of this changes: all 26,377 block states have the same runtime id before and after (measured with a scratch test in the root package that wrote every state with its id; the two files are byte for byte equal). The test already asserts that the variant is air's id (`popup_wire_test.go`, "The number's metadata holds the air variant").

Not pinned before or after: the order of the 349 entries inside the item registry packet. It is the registration order (`server.go`, `makeItemComponents`) and changes with it; the goldens key the entries by name.

### 3.2 The proof: goldens by name

`hooktest.CanonWith(v, field)` is `Canon` with one addition: before it writes a struct field it asks `field(ownerType, fieldName, value)` for a text to write in the value's place. `wire/byname_test.go` uses it to write each item id as `item "<name>"`, from a table of the ids the client is told (the registry packet and the start game data; an id with two names or a name with two ids fails the test). The subtest `TestWire/by name` pins the items, the recipes and the creative list in that form in `items.byname.golden`, `recipes.byname.golden` and `creative.byname.golden`, and with the items the sorted ids of the custom items as `meta/custom_ids`. `popup.byname.golden` holds the popup packets with the variant written as `"block minecraft:air"`.

What it proves: the goldens by name are recorded in T108 from the code before the change. If they are byte for byte the same after it, then every entry is the same but for the numbers written as names, and the set of custom ids is the same; with the numbered goldens changed in exactly the entries of 3.1, the change is a renumbering and nothing else. A developer and a reviewer check it with `git status --short`: after T112 it lists the four numbered goldens and none of the files by name.

Measured, in the prototype:

- After T110 and after T111 the subtest `by name` passes while `items`, `recipes` and `creative` fail with the counts of 3.1; `popup.byname.golden` passes while `popup.golden` fails in 2 entries.
- After the re-record `git status` lists `items.golden`, `recipes.golden`, `creative.golden` and `popup.golden` and no `*.byname.golden`; the dumps differ in 698 + 327 + 243 + 2 entries of the numbered goldens and in 0 of those by name; every differing line is `RuntimeID: intN(N)`, `NetworkID: intN(N)` or the popup's `uint32(2): int32(N)`.
- It catches a real difference. With `MaxDurability` of `sgitem/hammer.go` raised by one, `TestWire/by name` fails with `golden testdata/items.byname.golden differs: 4 changed` and names `registry/skygear:hammer`, `registry/skygear:stone_hammer`, `start/skygear:hammer`, `start/skygear:stone_hammer` (the change was undone).
- The subtest keeps three checks of its own (`byNameSensitivity`): two custom items that trade ids change their two numbered entries and no entry by name; one changed component value changes exactly that item's entry by name; a recipe whose result becomes another item has another text by name.

`TestWire` cannot run twice in one process (`-count=2` fails at the base, `docs/TODO.md`); run it as separate processes.

### 3.3 Saves and stored data

Nothing stored holds a numeric item or block id. Every place something is kept across server starts:

| Stored | Written by | How an item or block is identified |
|---|---|---|
| chunk blocks (leveldb) | `third_party/dragonfly/server/world/chunk/encoding.go:98` (`RuntimeIDToState`), read at `:151` (`StateToRuntimeID`) | the palette entry `name`, `states`, `version`; runtime ids exist only in memory |
| items in any block entity, entity or inventory | `item.WriteNBT(st, true)`, `third_party/dragonfly/server/item/nbt.go:78` | `Name`, `Damage`, `Count`, for a block item `Block{name, states}` |
| machines, hoppers, drawers | `machine/machine.go:1348`, `:1355`, read `:1448`; `hopper/hopper.go:194`, `:211`; `storage/drawer.go:270`, `:282` | `item.WriteNBT(…, true)`; a machine's kind is the block name `skygear:<kind>` |
| a player's state (`save_players.state`) | `common/playerstate/state.go:78`, `:190`; `platform/invsee.go:158`, `:169` | `item.WriteNBT(…, true)` in little endian NBT |
| the save's state document (`save_state`, key `skygear.state`) | `core/saved.go` `marshalSaved`, ten saved parts | JSON by field name. Items appear as text: `produced` (item name, `machine/stats.go`), `favourites` (`naming.ItemKey`, name and metadata value), `net_disks` and `net_tasks` (`storagenet.Key`, the NBT of the stack by name, `core.EncodeItem`), `hunt_kits` and `hunt_stash` (that NBT, `dimension/kit.go:230`), `net_nodes` filters (strings) |
| account tables (`achievements`, `counters`, `account_settings`) | `platform/db` | text keys |
| the resource packs | static files under `assets/resourcepack`, loaded with `resourcepack.LoadFS`; `conf.DisableResourceBuilding = true` (`cmd/skygear/main.go:152`) | no pack file holds a numeric item id (searched: only the binding name `#item_id_aux`, which the client fills) |

Numeric ids are read in these places only, all for a packet being sent or a comparison in memory: `world.ItemRuntimeID` in `common/chestui/localize.go:77` and `common/internal/slottext/slottext.go:41` (the aux value in texts sent to the client: `common/ui/ui.go:395`, `common/waila/container.go:129`); `BlockRuntimeID` in `expand/expandmap.go:102`, `wand/use.go:119`, `preview/preview.go:212`, `common/sghook/view.go:156` and `:177`, `common/sghook/entity.go:36`. In Dragonfly, item runtime ids are used in `server.go`, `session/` and `world/item.go` only. `block.NextHash()` values and the chunk encoder's palette cache are per process.

One stored thing does change, in order only. `marshalSaved` writes the fields of the saved parts in the order the parts were declared, which is package initialisation order, and `codex` (one saved part, the field `favourites`) is initialised later than before (3.4). A save written after the change has `"favourites"` after the fields of `fairy` and `netstore` instead of before them. `loadSaved` parses the document into a map by field name, so the order is not read. Measured with a scratch test: the base writes `…"composters":{"a":1},"last_sapling":"…","favourites":{…},"next_fairy":5,"version":1}`, the final state `…"last_sapling":"…","next_fairy":5,"favourites":{…},"version":1}`, and each build loads the other's document with the same values. `core/saved_test.go` (`TestSavedRoundTrip`, `TestLoadIgnoresUnknownAndOtherFields`) pins the reading by name.

The existing persistence tests (`persist_test.go`, `vanilla_persist_test.go`, `drawer_reload_test.go`, `migrate`, `TestForeignKindsRoundTrip`) write and read in one process, so with one numbering: they cannot show that a save of the old order loads under the new one. T109 adds the test that does. `TestOldSaveLoads` (root package) loads `testdata/save/oldsave.zip`, a save recorded by the code before the change: the world directory mcdb wrote and the bytes of one player's state. It holds every state of every custom block (2,417 states of 107 blocks), every custom item in chests (349 in 13 chests), a crusher, a drawer and a wooden hopper that hold custom items, and a player who carries 36 custom items. The test lists what it reads, by name, and compares with `testdata/save/oldsave.golden` (2,433 block lines, 353 item lines, 36 player lines), which the recording build wrote from its own reading.

Measured: recorded at the base, the test passes unchanged after T110, after T111 and at the end, in separate processes and under `-race`. It catches a real difference: with `sgitem.Hammer`'s name changed to `skygear:hammer2` it fails and names `item/2,122,0/00` (`skygear:hammer:0 x77`, the drawer), `item/4,120,0/08` (a chest slot) and `player/inventory/12` (undone). When it records, it also checks that the recording build reads back every block under its name and every stack whole; a machine's properties may differ there, because its tier and a sieve's look are rebuilt from its stored data (the registered state `skygear:sieve{look=490}` reads back as `look=000`).

### 3.4 Package initialisation order

`GODEBUG=inittrace=1` on the binary of `cmd/skygear`, base and final: 231 packages with init work in both, and two of them change place.

| Package | Before | After | Why it is harmless |
|---|---|---|---|
| `tinker` | after `expand`, before `machine` | after `netstore`, before `codex` | it now imports `machine`. Its init work: 63 `world.RegisterItem` calls (the ids of 3.1) and three `core.NewPlayers` (ids of in-memory parts). No package initialised between its old and its new place (`machine`, `conduit`, `dimension`, `farm`, `fairy`, `storage`, `copier`, `netstore`) imports it or looks an item up by name while it initialises (`world.ItemByName` is called at run time only: `machine/stats.go:197`) |
| `codex` | after `farm`, before `fairy` | after `tinker`, before `catalog` | it imports `tinker`. Its init work: one `core.NewSaved` (the field order of 3.3) and two `core.NewPlayers`. It registers no block, item or recipe |

Go initialises, among the packages whose imports are done, the one whose path sorts first. `tinker` needs `machine` now, so it waits for it and then for everything that sorts before `tinker` and is ready: `conduit`, `dimension`, `farm`, `fairy`, `storage`, `copier`, `netstore`.

What guarantees the order the game needs: nothing needed "tools before machines" but the goldens. After the change `machine` cannot name a tinker symbol (no import), `tinker` initialises after `machine` by Go's rule, and the root still initialises last, so `core.Install` runs when every item is registered and every `Kinds` entry exists. The hopper move changes no package's place (the order of the 231 is the same after T110): the registration moves from the root's `init` to `hopper`'s, which already ran before `sgitem`'s.

### 3.5 The pins

The nine files of `content/skygear/testdata/pin/` are byte for byte unchanged in every state (`git status --short content/skygear/testdata/pin` empty; `TestPinOrder`, `TestPinSimple`, `TestPinBlocks`, `TestPinUse`, `TestPinLifecycle`, `TestPinSettings`, `TestPinLook`, `TestPinKinds`, `TestPinCreative` pass). In particular `kinds.golden` keeps `click/tinker_table: activate=true screens=tinker.ShowTable`, `look.golden` keeps `machine/tinker_table`, `creative.golden` keeps its 243 items in order (the creative list is built from `Feature.Creative` slots, not from registration order), and `order.golden` and `settings.golden` are untouched because `Features` is (decision 3). `wire/testdata/blocks.golden` is unchanged.

### 3.6 Tests that need the tinker table's behaviour

- The root's test binary installs every feature, so `TestPinKinds` (`click/tinker_table`), `TestForeignKindsRoundTrip` and `TestKindOwners` see the registered kind. Without the call of `registerKinds()` the binary does not start: `panic: core: Install: feature "machine": machine kinds without behaviour: tinker_table (feature tinker); …` (measured, undone).
- `tinker`'s tests: `tinker/setup_test.go` already calls `coretest.Setup(tinker.Feature)`, whose `Setup` now registers the kind too. T111 adds `TestTinkerTableKind` there. `TestTinkerTable` calls `tinker.MakeTool` and `tinker.ApplyModifier` directly and is not touched.
- `machine`'s tests set up `machine` and, in `netkinds_test.go`, `netstore`. They do not link `tinker` any more, so the tinker table has no behaviour there; no test of `machine` clicks one (`KindTinker` appears only in `TestFrameFlags`), and `coretest.Setup` runs no `Check`. Nothing is added.
- Test binaries change what they link: `catalog`, `climate`, `conduit`, `copier`, `decor`, `dimension`, `fairy`, `machine`, `netstore`, `storage` and `textfmt` no longer link `tinker` and `preview` (and `look`, which `farm` keeps); `tinker`'s links `machine`. The binaries of `catalog` and `screens` link `hopper` without the root and now have the hoppers registered. All their tests pass unchanged; the one golden that sees it is `popup.golden` (3.1).

### 3.7 The hopper

Where it belongs: in `hopper`'s own `init`. Every other package that declares a block or an item type registers it when it is initialised (`sgitem`, `machine`, `conduit`, `storage`, `farm`, `elevator`, `preview`, `progress`, `tinker`); the root's `hopper.go` was the one registration outside the declaring package, and row R9 of part 3-3 kept it for the goldens alone.

Why it is clean, measured after T110 alone: the full suite fails in `TestWire/items` (698), `TestWire/recipes` (327) and `TestWire/creative` (243) and nowhere else; `TestWire/by name`, `TestOldSaveLoads` and every pin pass; all 26,377 block runtime ids are the same; the package initialisation order is the same; `hopper.Register` has no other caller (`grep -rn 'hopper\.Register'`). The hopper's block hashes (`block.NextHash()`) are taken earlier, which shifts those of the blocks registered after it; they are keys of an in-memory table.

## 4. Names, imports and layers

```go
// common/sghook/hooktest (canon.go)
type FieldText func(owner reflect.Type, name string, v reflect.Value) (string, bool)
func CanonWith(v any, field FieldText) string
func Canon(v any) string // unchanged for its callers: CanonWith(v, nil)

// content/skygear/wire (test files)
func itemEntries(t testing.TB, canon func(any) string, registry, start []protocol.ItemEntry) map[string]string
func recipeEntries(t testing.TB, canon func(any) string, cd *packet.CraftingData) map[string]string
func creativeEntries(t testing.TB, canon func(any) string, cc *packet.CreativeContent) map[string]string
func diffEntries(a, b map[string]string) []string
var itemIDFields map[reflect.Type]string
func itemNames(t testing.TB, registry, start []protocol.ItemEntry) map[int64]string
func byName(names map[int64]string) func(any) string
func byNameGoldens(t *testing.T, registry, start []protocol.ItemEntry, cd *packet.CraftingData, cc *packet.CreativeContent)
func byNameSensitivity(t *testing.T, registry []protocol.ItemEntry, cd *packet.CraftingData)

// content/skygear/dimension (popup_wire_test.go)
func blockNamed(pk packet.Packet) packet.Packet

// content/skygear (oldsave_test.go)
func TestOldSaveLoads(t *testing.T)

// content/skygear/tinker
func registerKinds()                    // kinds.go
func TestTinkerTableKind(t *testing.T)  // kinds_test.go
```

Removed: `hopper.Register`, the root's `hopper.go`, `machine/tinkertable.go`, `machineFeatureImports`.

| Package | Imports before | Imports after |
|---|---|---|
| `machine` | …, `tinker` (in `tinkertable.go`) | no package of the feature layer |
| `tinker` | achievement, advancement, core, island, locales, look, naming, preview, progress, sgitem | the same and `machine` |
| the root | …, `hopper` (in `hopper.go`; test files keep importing it) | no non-test file imports `hopper` |

`layers` and `featureLayer` after T111: `"machine", "tinker", "conduit", "storage", "farm", "fairy", "dimension", "handwork", "expand", "copier", "wand", "netstore", "elevator"`. `siblings` after T111: `"tinker": {"machine"}`, `"conduit": {"machine"}`, `"storage": {"machine"}`, `"farm": {"machine"}`, `"fairy": {"farm", "machine"}`, `"dimension": {"machine"}`, `"copier": {"conduit", "machine", "storage"}`, `"netstore": {"conduit", "machine", "storage"}`.

## 5. Order and what was run

T108 and T109 share no file and may run at the same time; T110, T111, T112 and T113 run one after another in id order. The prototype applied the six tasks in id order on b49fc81:

| State | `go build`, `go vet`, `gofmt -l` | `go test ./...` | Goldens |
|---|---|---|---|
| base b49fc81 | clean | 62 ok | |
| after T108 | clean | 62 ok | four new files by name; the existing goldens untouched |
| after T109 | clean | 62 ok | `oldsave.zip`, `oldsave.golden` new |
| after T110 | clean | 61 ok, `wire` fails: `items` 698, `recipes` 327, `creative` 243 | by name, old save, pins: pass |
| after T111 | clean | 60 ok, `wire` fails as after T110, `dimension` fails: `popup.golden` 2 | by name, old save, pins: pass |
| after T112 | clean | 62 ok | four numbered goldens re-recorded; `git status` lists no file by name |
| after T113 | clean | 62 ok | documents only |

Also run on the final state: `go test -race` on `common/sghook/hooktest`, `wire`, `tinker`, the popup tests of `dimension`, and `TestOldSaveLoads`, `TestKindOwners`, `TestMachineImportsNoFeature`, `TestPinKinds` of the root, all green.

## 6. Code by task


### REF T108

Test files and the test support package `hooktest` only (nothing the server binary links), with the package comments of `hooktest` and `wire`. Copy the blocks, then record the four goldens by name with the two commands, once, from the unchanged code.

In `canon.go` the function `canon` gets a fifth parameter that every recursive call passes on; the struct case asks it first. In `wire_test.go` the three entry functions take the function that makes a text, the closure `diff` of `sensitivity` becomes the package function `diffEntries` (its body unchanged), and `TestWire` gets the subtest `by name` between `creative` and `sensitivity`.

Record and check:

```sh
SKYGEAR_GOLDEN=update go test ./content/skygear/wire/ -run 'TestWire$' -count=1
SKYGEAR_GOLDEN=update go test ./content/skygear/dimension/ -run 'TestPopupAddActor$' -count=1
git status --short content/skygear/wire/testdata content/skygear/dimension/testdata
wc -l content/skygear/wire/testdata/*.byname.golden content/skygear/dimension/testdata/popup.byname.golden
```

`git status` lists exactly four files, all new (`??`): `items.byname.golden`, `recipes.byname.golden`, `creative.byname.golden`, `popup.byname.golden`. The update also rewrites the five existing goldens with the bytes they already have; if `git status` shows one of them as modified, stop and report. `wc -l` at b49fc81: 2,779, 3,907, 1,877 and 32 lines (a different number means the game's items or recipes changed since; report it and go on only if the existing goldens are unmodified). Then run `go test ./content/skygear/wire/ ./content/skygear/dimension/ ./common/sghook/hooktest/ -count=1` three times as separate processes (never `-count=2`).


`common/sghook/hooktest/canon.go`

```diff
diff --git a/common/sghook/hooktest/canon.go b/common/sghook/hooktest/canon.go
index 00ff093..c75ea78 100644
--- a/common/sghook/hooktest/canon.go
+++ b/common/sghook/hooktest/canon.go
@@ -14,13 +14,24 @@ import (
 // too), pointers and interfaces followed, strings quoted, byte slices in
 // hex and other scalars as type(value). Equal content gives equal text
 // whatever the map order.
-func Canon(v any) string {
+func Canon(v any) string { return CanonWith(v, nil) }
+
+// FieldText is asked for every struct field CanonWith is about to write:
+// owner is the struct's type, name the field's name and v its value. It
+// returns the text to write in place of the value, or false to write the
+// value as Canon does.
+type FieldText func(owner reflect.Type, name string, v reflect.Value) (string, bool)
+
+// CanonWith is Canon, but writes a struct field as field says where field
+// says so (nil: nowhere). It is for a text that names what a number stands
+// for, so the text does not change when only the numbering does.
+func CanonWith(v any, field FieldText) string {
 	var b strings.Builder
-	canon(&b, reflect.ValueOf(v), 0, map[uintptr]bool{})
+	canon(&b, reflect.ValueOf(v), 0, map[uintptr]bool{}, field)
 	return b.String()
 }
 
-func canon(b *strings.Builder, v reflect.Value, depth int, seen map[uintptr]bool) {
+func canon(b *strings.Builder, v reflect.Value, depth int, seen map[uintptr]bool, field FieldText) {
 	if !v.IsValid() {
 		b.WriteString("nil")
 		return
@@ -37,14 +48,14 @@ func canon(b *strings.Builder, v reflect.Value, depth int, seen map[uintptr]bool
 			return
 		}
 		seen[p] = true
-		canon(b, v.Elem(), depth, seen)
+		canon(b, v.Elem(), depth, seen, field)
 		delete(seen, p)
 	case reflect.Interface:
 		if v.IsNil() {
 			b.WriteString("nil")
 			return
 		}
-		canon(b, v.Elem(), depth, seen)
+		canon(b, v.Elem(), depth, seen, field)
 	case reflect.Map:
 		if v.IsNil() {
 			b.WriteString("nil")
@@ -57,7 +68,7 @@ func canon(b *strings.Builder, v reflect.Value, depth int, seen map[uintptr]bool
 		kvs := make([]kv, 0, v.Len())
 		for it := v.MapRange(); it.Next(); {
 			var kb strings.Builder
-			canon(&kb, it.Key(), 0, seen)
+			canon(&kb, it.Key(), 0, seen, field)
 			kvs = append(kvs, kv{keyText(it.Key(), kb.String()), it.Value()})
 		}
 		sort.Slice(kvs, func(i, j int) bool { return kvs[i].key < kvs[j].key })
@@ -65,7 +76,7 @@ func canon(b *strings.Builder, v reflect.Value, depth int, seen map[uintptr]bool
 		for _, e := range kvs {
 			indent(b, depth+1)
 			b.WriteString(e.key + ": ")
-			canon(b, e.val, depth+1, seen)
+			canon(b, e.val, depth+1, seen, field)
 			b.WriteByte('\n')
 		}
 		closing(b, '}', len(kvs), depth)
@@ -82,7 +93,11 @@ func canon(b *strings.Builder, v reflect.Value, depth int, seen map[uintptr]bool
 		for _, n := range names {
 			indent(b, depth+1)
 			b.WriteString(n + ": ")
-			canon(b, v.Field(idx[n]), depth+1, seen)
+			if text, ok := fieldText(field, t, n, v.Field(idx[n])); ok {
+				b.WriteString(text)
+			} else {
+				canon(b, v.Field(idx[n]), depth+1, seen, field)
+			}
 			b.WriteByte('\n')
 		}
 		closing(b, '}', len(names), depth)
@@ -102,7 +117,7 @@ func canon(b *strings.Builder, v reflect.Value, depth int, seen map[uintptr]bool
 		open(b, '[', v.Len(), depth)
 		for i := range v.Len() {
 			indent(b, depth+1)
-			canon(b, v.Index(i), depth+1, seen)
+			canon(b, v.Index(i), depth+1, seen, field)
 			b.WriteByte('\n')
 		}
 		closing(b, ']', v.Len(), depth)
@@ -123,6 +138,14 @@ func canon(b *strings.Builder, v reflect.Value, depth int, seen map[uintptr]bool
 	}
 }
 
+// fieldText is field's text for a struct field, if field is set and has one.
+func fieldText(field FieldText, owner reflect.Type, name string, v reflect.Value) (string, bool) {
+	if field == nil {
+		return "", false
+	}
+	return field(owner, name, v)
+}
+
 // keyText is how a map key is written: strings bare, others as canon.
 func keyText(k reflect.Value, text string) string {
 	for k.Kind() == reflect.Interface && !k.IsNil() {
```

`common/sghook/hooktest/hooktest_test.go`

```diff
diff --git a/common/sghook/hooktest/hooktest_test.go b/common/sghook/hooktest/hooktest_test.go
index 9177b83..c394ec6 100644
--- a/common/sghook/hooktest/hooktest_test.go
+++ b/common/sghook/hooktest/hooktest_test.go
@@ -4,6 +4,7 @@ import (
 	"fmt"
 	"os"
 	"path/filepath"
+	"reflect"
 	"strconv"
 	"strings"
 	"sync/atomic"
@@ -104,6 +105,35 @@ func TestCanonTypes(t *testing.T) {
 	}
 }
 
+// TestCanonWith: a struct field is written as the function says wherever the
+// struct is (in a field, behind a pointer in a slice of interfaces); a field
+// the function passes, a value that is no struct field and a nil function
+// are written as Canon writes them.
+func TestCanonWith(t *testing.T) {
+	type id struct{ ID, Count int32 }
+	type holder struct {
+		One  id
+		Many []any
+	}
+	v := holder{One: id{ID: 7, Count: 2}, Many: []any{&id{ID: 9, Count: 1}, &id{ID: 7}, int32(7)}}
+	names := map[int64]string{7: "seven"}
+	got := CanonWith(v, func(owner reflect.Type, name string, f reflect.Value) (string, bool) {
+		if owner != reflect.TypeFor[id]() || name != "ID" {
+			return "", false
+		}
+		n, ok := names[f.Int()]
+		return "named " + strconv.Quote(n), ok
+	})
+	want := "{\n  Many: [\n    {\n      Count: int32(1)\n      ID: int32(9)\n    }\n    {\n      Count: int32(0)\n      ID: named \"seven\"\n    }\n    int32(7)\n  ]\n" +
+		"  One: {\n    Count: int32(2)\n    ID: named \"seven\"\n  }\n}"
+	if got != want {
+		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
+	}
+	if CanonWith(v, nil) != Canon(v) {
+		t.Fatal("a nil function changes the text")
+	}
+}
+
 func TestGoldenUpdateAndMismatch(t *testing.T) {
 	path := filepath.Join(t.TempDir(), "sub", "x.golden")
 	t.Setenv("SKYGEAR_GOLDEN", "update")
```

`common/sghook/hooktest/doc.go`

```diff
diff --git a/common/sghook/hooktest/doc.go b/common/sghook/hooktest/doc.go
index ea60eff..e6bedbe 100644
--- a/common/sghook/hooktest/doc.go
+++ b/common/sghook/hooktest/doc.go
@@ -4,10 +4,11 @@
 // listener, with one fake connection, and keeps every packet and the
 // start game data the server wrote to it.
 //
-// Canon turns any value into a stable text, Golden compares the hashes of
-// such texts to a file under testdata. A change of Dragonfly or of the
-// patch that alters what a client receives makes Golden fail and name the
-// entries that changed.
+// Canon turns any value into a stable text (CanonWith lets the caller write
+// a struct field its own way, to name what a numeric id stands for), Golden
+// compares the hashes of such texts to a file under testdata. A change of
+// Dragonfly or of the patch that alters what a client receives makes Golden
+// fail and name the entries that changed.
 //
 // Re-record a golden file after a deliberate change:
 //
```

`content/skygear/wire/wire_test.go`

```diff
diff --git a/content/skygear/wire/wire_test.go b/content/skygear/wire/wire_test.go
index 5c0c0d2..08e65fd 100644
--- a/content/skygear/wire/wire_test.go
+++ b/content/skygear/wire/wire_test.go
@@ -106,7 +106,7 @@ func TestWire(t *testing.T) {
 		t.Logf("%d of %d blocks have more than one property", multi, len(data.CustomBlocks))
 	})
 	t.Run("items", func(t *testing.T) {
-		entries := itemEntries(t, reg.Items, data.Items)
+		entries := itemEntries(t, hooktest.Canon, reg.Items, data.Items)
 		custom := 0
 		for _, e := range reg.Items {
 			if e.ComponentBased {
@@ -121,7 +121,7 @@ func TestWire(t *testing.T) {
 		hooktest.Golden(t, "testdata/items.golden", entries)
 	})
 	t.Run("recipes", func(t *testing.T) {
-		entries := recipeEntries(t, cd)
+		entries := recipeEntries(t, hooktest.Canon, cd)
 		if want := len(recipe.Recipes()) + 1; len(entries) != want {
 			t.Errorf("pinned %d recipe entries (with the clear flag), registry has %d", len(entries), want)
 		}
@@ -129,13 +129,16 @@ func TestWire(t *testing.T) {
 		hooktest.Golden(t, "testdata/recipes.golden", entries)
 	})
 	t.Run("creative", func(t *testing.T) {
-		entries := creativeEntries(t, cc)
+		entries := creativeEntries(t, hooktest.Canon, cc)
 		if len(cc.Groups) != len(creative.Groups()) || len(cc.Items) != len(creative.Items()) {
 			t.Errorf("packet has %d groups, %d items; registry has %d, %d", len(cc.Groups), len(cc.Items), len(creative.Groups()), len(creative.Items()))
 		}
 		t.Logf("creative: %d groups + %d items pinned; registry %d + %d", len(cc.Groups), len(cc.Items), len(creative.Groups()), len(creative.Items()))
 		hooktest.Golden(t, "testdata/creative.golden", entries)
 	})
+	// The same items, recipes and creative list with every item id written as
+	// the item's name (byname_test.go).
+	t.Run("by name", func(t *testing.T) { byNameGoldens(t, reg.Items, data.Items, cd, cc) })
 	t.Run("sensitivity", func(t *testing.T) { sensitivity(t, data.CustomBlocks, reg.Items) })
 }
 
@@ -227,8 +230,10 @@ func (s sorter) Less(i, j int) bool { return s.less(i, j) }
 func (s sorter) Swap(i, j int)      { s.swap(i, j) }
 
 // itemEntries returns registry/<name> for the item registry packet and
-// start/<name> for the start game data.
-func itemEntries(t testing.TB, registry, start []protocol.ItemEntry) map[string]string {
+// start/<name> for the start game data. canon makes an entry's text
+// (hooktest.Canon, or byName's for the text by name); recipeEntries and
+// creativeEntries take it too.
+func itemEntries(t testing.TB, canon func(any) string, registry, start []protocol.ItemEntry) map[string]string {
 	t.Helper()
 	out := make(map[string]string, len(registry)+len(start))
 	for prefix, list := range map[string][]protocol.ItemEntry{"registry/": registry, "start/": start} {
@@ -237,7 +242,7 @@ func itemEntries(t testing.TB, registry, start []protocol.ItemEntry) map[string]
 			if _, dup := out[k]; dup {
 				t.Errorf("duplicate item %s", k)
 			}
-			out[k] = hooktest.Canon(e)
+			out[k] = canon(e)
 		}
 	}
 	return out
@@ -245,14 +250,14 @@ func itemEntries(t testing.TB, registry, start []protocol.ItemEntry) map[string]
 
 // recipeEntries returns <kind>/<RecipeNetworkID> per recipe and <kind>/<index>
 // for the recipes without a network id.
-func recipeEntries(t testing.TB, cd *packet.CraftingData) map[string]string {
+func recipeEntries(t testing.TB, canon func(any) string, cd *packet.CraftingData) map[string]string {
 	t.Helper()
 	out := map[string]string{}
 	add := func(k string, v any) {
 		if _, dup := out[k]; dup {
 			t.Errorf("duplicate recipe %s", k)
 		}
-		out[k] = hooktest.Canon(v)
+		out[k] = canon(v)
 	}
 	for _, r := range cd.ShapedRecipes {
 		add(fmt.Sprintf("shaped/%d", r.RecipeNetworkID), r)
@@ -293,11 +298,11 @@ func recipeEntries(t testing.TB, cd *packet.CraftingData) map[string]string {
 
 // creativeEntries returns group/<index> and item/<CreativeItemNetworkID>, and
 // order/items, the network ids in the order the client shows them.
-func creativeEntries(t testing.TB, cc *packet.CreativeContent) map[string]string {
+func creativeEntries(t testing.TB, canon func(any) string, cc *packet.CreativeContent) map[string]string {
 	t.Helper()
 	out := map[string]string{}
 	for i, g := range cc.Groups {
-		out[fmt.Sprintf("group/%d", i)] = hooktest.Canon(g)
+		out[fmt.Sprintf("group/%d", i)] = canon(g)
 	}
 	var order []uint32
 	for _, it := range cc.Items {
@@ -305,10 +310,10 @@ func creativeEntries(t testing.TB, cc *packet.CreativeContent) map[string]string
 		if _, dup := out[k]; dup {
 			t.Errorf("duplicate creative item %s", k)
 		}
-		out[k] = hooktest.Canon(it)
+		out[k] = canon(it)
 		order = append(order, it.CreativeItemNetworkID)
 	}
-	out["order/items"] = hooktest.Canon(order)
+	out["order/items"] = canon(order)
 	return out
 }
 
@@ -319,21 +324,7 @@ func creativeEntries(t testing.TB, cc *packet.CreativeContent) map[string]string
 // block order, the permutations and the ids, which must change nothing.
 func sensitivity(t *testing.T, blocks []protocol.BlockEntry, items []protocol.ItemEntry) {
 	base := blockEntries(t, blocks)
-	diff := func(a, b map[string]string) []string {
-		var d []string
-		for k, v := range a {
-			if b[k] != v {
-				d = append(d, k)
-			}
-		}
-		for k := range b {
-			if _, ok := a[k]; !ok {
-				d = append(d, k)
-			}
-		}
-		sort.Strings(d)
-		return d
-	}
+	diff := diffEntries
 	clone := func(i int, mutate func(props map[string]any) bool) (map[string]string, string) {
 		cp := slices.Clone(blocks)
 		cp[i] = protocol.BlockEntry{Name: blocks[i].Name, Properties: cloneMap(blocks[i].Properties)}
@@ -455,7 +446,7 @@ func sensitivity(t *testing.T, blocks []protocol.BlockEntry, items []protocol.It
 	}
 
 	// Items: one component value, and one added component.
-	baseItems := itemEntries(t, items, nil)
+	baseItems := itemEntries(t, hooktest.Canon, items, nil)
 	tried := 0
 	for _, tc := range []string{"value", "added"} {
 		for i, e := range items {
@@ -474,7 +465,7 @@ func sensitivity(t *testing.T, blocks []protocol.BlockEntry, items []protocol.It
 				continue
 			}
 			tried++
-			if d := diff(baseItems, itemEntries(t, cp, nil)); !slices.Equal(d, []string{"registry/" + e.Name}) {
+			if d := diff(baseItems, itemEntries(t, hooktest.Canon, cp, nil)); !slices.Equal(d, []string{"registry/" + e.Name}) {
 				t.Errorf("item %s, %s: changed entries %v", e.Name, tc, d)
 			} else {
 				t.Logf("item %s, component %s: only registry/%s changed", e.Name, tc, e.Name)
@@ -487,6 +478,24 @@ func sensitivity(t *testing.T, blocks []protocol.BlockEntry, items []protocol.It
 	}
 }
 
+// diffEntries returns the sorted names of the entries that differ between a
+// and b, or are in only one of them.
+func diffEntries(a, b map[string]string) []string {
+	var d []string
+	for k, v := range a {
+		if b[k] != v {
+			d = append(d, k)
+		}
+	}
+	for k := range b {
+		if _, ok := a[k]; !ok {
+			d = append(d, k)
+		}
+	}
+	sort.Strings(d)
+	return d
+}
+
 // bumpFirstScalar changes the first number or string found under m (by
 // sorted key) and reports whether there was one.
 func bumpFirstScalar(m map[string]any) bool {
```

`content/skygear/wire/byname_test.go` (new)

```go
package wire

import (
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/presentkim/skygear/common/sghook/hooktest"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// A custom item's numeric id is its place in the order the items are
// registered in, which is the order the packages are initialised in. The
// goldens of the items, the recipes and the creative list pin those ids, so a
// change of that order alone (an import turned around, a registration moved)
// changes hundreds of their entries and nothing a player can tell.
//
// The goldens by name (testdata/*.byname.golden) hold the same entries with
// every item id written as the item's name, and the set of ids the custom
// items have. A change of the numbering alone leaves them byte for byte as
// they are: that is the proof that nothing but the numbering changed, and the
// lines that did change in the numbered goldens are then ids only.

// itemIDFields are the struct fields that hold a numeric item id.
var itemIDFields = map[reflect.Type]string{
	reflect.TypeFor[protocol.ItemEntry]():             "RuntimeID",
	reflect.TypeFor[protocol.ItemType]():              "NetworkID",
	reflect.TypeFor[protocol.DefaultItemDescriptor](): "NetworkID",
}

// itemNames returns the name of every numeric item id a client is told: the
// ids of the item registry packet and of the start game data. An id with two
// names or a name with two ids fails t.
func itemNames(t testing.TB, registry, start []protocol.ItemEntry) map[int64]string {
	t.Helper()
	names, ids := map[int64]string{}, map[string]int64{}
	for _, list := range [][]protocol.ItemEntry{registry, start} {
		for _, e := range list {
			id := int64(e.RuntimeID)
			if n, ok := names[id]; ok && n != e.Name {
				t.Errorf("item id %d is %s and %s", id, n, e.Name)
			}
			if i, ok := ids[e.Name]; ok && i != id {
				t.Errorf("item %s has the ids %d and %d", e.Name, i, id)
			}
			names[id], ids[e.Name] = e.Name, id
		}
	}
	return names
}

// byName returns hooktest.Canon with every item id of itemIDFields written as
// the item's name. An id names does not hold (0, the empty stack) stays a
// number.
func byName(names map[int64]string) func(any) string {
	return func(v any) string {
		return hooktest.CanonWith(v, func(owner reflect.Type, field string, f reflect.Value) (string, bool) {
			if itemIDFields[owner] != field {
				return "", false
			}
			name, ok := names[f.Int()]
			return "item " + strconv.Quote(name), ok
		})
	}
}

// byNameGoldens pins the items, the recipes and the creative list by name,
// and with the items the sorted ids of the custom items (meta/custom_ids).
func byNameGoldens(t *testing.T, registry, start []protocol.ItemEntry, cd *packet.CraftingData, cc *packet.CreativeContent) {
	canon := byName(itemNames(t, registry, start))
	items := itemEntries(t, canon, registry, start)
	ids := make([]int, 0, len(registry))
	for _, e := range registry {
		ids = append(ids, int(e.RuntimeID))
	}
	sort.Ints(ids)
	items["meta/custom_ids"] = hooktest.Canon(ids)
	hooktest.Golden(t, "testdata/items.byname.golden", items)
	hooktest.Golden(t, "testdata/recipes.byname.golden", recipeEntries(t, canon, cd))
	hooktest.Golden(t, "testdata/creative.byname.golden", creativeEntries(t, canon, cc))
	byNameSensitivity(t, registry, cd)
}

// byNameSensitivity shows what the texts by name leave out and what they
// keep: two custom items that trade their ids change their two entries of
// the numbered texts and no text by name; one changed component value changes
// exactly that item's text by name; a recipe whose result becomes another
// item has another text by name.
func byNameSensitivity(t *testing.T, registry []protocol.ItemEntry, cd *packet.CraftingData) {
	if len(registry) < 2 {
		t.Fatal("fewer than two custom items")
	}
	canon := byName(itemNames(t, registry, nil))
	base := itemEntries(t, canon, registry, nil)

	traded := slices.Clone(registry)
	traded[0].RuntimeID, traded[1].RuntimeID = registry[1].RuntimeID, registry[0].RuntimeID
	want := []string{"registry/" + registry[0].Name, "registry/" + registry[1].Name}
	sort.Strings(want)
	if d := diffEntries(itemEntries(t, hooktest.Canon, registry, nil), itemEntries(t, hooktest.Canon, traded, nil)); !slices.Equal(d, want) {
		t.Errorf("traded ids changed the numbered entries %v, want %v", d, want)
	}
	if d := diffEntries(base, itemEntries(t, byName(itemNames(t, traded, nil)), traded, nil)); len(d) != 0 {
		t.Errorf("traded ids changed the entries by name %v", d)
	}

	tried := false
	for i, e := range registry {
		cp := slices.Clone(registry)
		cp[i].Data = cloneMap(e.Data)
		comps, _ := cp[i].Data["components"].(map[string]any)
		if comps == nil || !bumpFirstScalar(comps) {
			continue
		}
		tried = true
		if d := diffEntries(base, itemEntries(t, canon, cp, nil)); !slices.Equal(d, []string{"registry/" + e.Name}) {
			t.Errorf("item %s, component value: changed entries by name %v", e.Name, d)
		}
		break
	}
	if !tried {
		t.Error("no item with a component to change")
	}

	tried = false
	for _, r := range cd.ShapedRecipes {
		if len(r.Output) == 0 || int64(r.Output[0].NetworkID) == int64(registry[0].RuntimeID) {
			continue
		}
		other := r
		other.Output = slices.Clone(r.Output)
		other.Output[0].NetworkID = int32(registry[0].RuntimeID)
		tried = true
		if got := canon(other); got == canon(r) || !strings.Contains(got, "NetworkID: item "+strconv.Quote(registry[0].Name)) {
			t.Errorf("a recipe that gives %s instead has the text by name:\n%s", registry[0].Name, got)
		}
		break
	}
	if !tried {
		t.Error("no shaped recipe to change")
	}
}
```

`content/skygear/wire/doc.go`

```diff
diff --git a/content/skygear/wire/doc.go b/content/skygear/wire/doc.go
index 1b59ec9..b067203 100644
--- a/content/skygear/wire/doc.go
+++ b/content/skygear/wire/doc.go
@@ -13,7 +13,18 @@
 //   - items.golden: the item registry packet (registry/<name>) and the start
 //     game items (start/<name>), with their components;
 //   - recipes.golden: the crafting data, <kind>/<network id>;
-//   - creative.golden: the creative groups and items.
+//   - creative.golden: the creative groups and items;
+//   - items.byname.golden, recipes.byname.golden, creative.byname.golden:
+//     the entries of the three files above with every item id written as the
+//     item's name, and the ids of the custom items as a set
+//     (meta/custom_ids).
+//
+// A custom item's id is its place in the order the items are registered in,
+// which follows the order the packages are initialised in. A change of that
+// order alone changes hundreds of entries of items.golden, recipes.golden
+// and creative.golden and leaves the three files by name byte for byte as
+// they are: after re-recording, git status then lists none of them, which
+// shows that nothing but the numbering changed.
 //
 // A golden changes only in a commit meant to change what clients get.
 // Re-record after such a change:
```

`content/skygear/dimension/popup_wire_test.go`

```diff
diff --git a/content/skygear/dimension/popup_wire_test.go b/content/skygear/dimension/popup_wire_test.go
index f217bd0..cf6fd51 100644
--- a/content/skygear/dimension/popup_wire_test.go
+++ b/content/skygear/dimension/popup_wire_test.go
@@ -3,6 +3,7 @@ package dimension
 import (
 	"context"
 	"fmt"
+	"maps"
 	"slices"
 	"testing"
 
@@ -96,6 +97,28 @@ func norm(t *testing.T, pk packet.Packet) (id uint64, out packet.Packet, ok bool
 	return 0, nil, false
 }
 
+// blockNamed returns pk with the variant of an add-actor packet, a block
+// runtime ID, written as the block's name. The runtime ID of a block depends
+// on which blocks the program registers, so it differs between this test
+// binary and the game and changes when this package's imports do; the name
+// does not.
+func blockNamed(pk packet.Packet) packet.Packet {
+	a, ok := pk.(*packet.AddActor)
+	if !ok {
+		return pk
+	}
+	rid, ok := a.EntityMetadata[protocol.EntityDataKeyVariant].(int32)
+	if !ok {
+		return pk
+	}
+	b, _ := world.DefaultBlockRegistry.BlockByRuntimeID(uint32(rid))
+	name, _ := b.EncodeBlock()
+	c := *a
+	c.EntityMetadata = maps.Clone(a.EntityMetadata)
+	c.EntityMetadata[protocol.EntityDataKeyVariant] = "block " + name
+	return &c
+}
+
 // TestPopupAddActor pins every packet a client is sent for the damage
 // numbers of a hunt and, for contrast, for a gem (a mark that is not
 // floating text): the add-actor packet with all its metadata, the moves
@@ -135,7 +158,7 @@ func TestPopupAddActor(t *testing.T) {
 		return ok && uint64(p.EntityUniqueID) == gemID
 	})
 
-	entries := map[string]string{}
+	entries, named := map[string]string{}, map[string]string{}
 	seq := map[string]int{} // packets so far per entity: the order between entities is not fixed
 	for _, pk := range c.Packets() {
 		id, cp, ok := norm(t, pk)
@@ -152,10 +175,15 @@ func TestPopupAddActor(t *testing.T) {
 			continue
 		}
 		seq[name]++
-		entries[fmt.Sprintf("%s/%03d %T", name, seq[name], pk)] = hooktest.Canon(cp)
+		key := fmt.Sprintf("%s/%03d %T", name, seq[name], pk)
+		entries[key] = hooktest.Canon(cp)
+		named[key] = hooktest.Canon(blockNamed(cp))
 	}
 	t.Logf("%d packets pinned", len(entries))
 	hooktest.Golden(t, "testdata/popup.golden", entries)
+	// The same packets with the variant's block by name: unlike its runtime
+	// ID, the name does not depend on the blocks this test binary registers.
+	hooktest.Golden(t, "testdata/popup.byname.golden", named)
 
 	// The number's metadata holds the air variant.
 	air := int32(world.DefaultBlockRegistry.BlockRuntimeID(block.Air{}))
```

### REF T109

One new test file and its two recorded files. `openPersisted`, `testSession`, `spawnPlayer`, `withPlayer` and `pinGolden` are the root's existing test helpers (`persist_test.go`, `helpers_test.go`, `pin_test.go`); they are used as they are.

Record and check:

```sh
SKYGEAR_GOLDEN=update go test ./content/skygear/ -run 'TestOldSaveLoads$' -count=1
cut -d/ -f1 content/skygear/testdata/save/oldsave.golden | sort | uniq -c
git status --short content/skygear/testdata
```

At b49fc81 the counts are `2433 block`, `353 item`, `36 player` (2,822 lines), and `git status` lists only `content/skygear/testdata/save/` as new; nothing under `testdata/pin` is modified. `oldsave.zip` is about 36 kB and holds `world/db/000001.log`, `world/db/CURRENT`, `world/db/MANIFEST-000000`, `world/level.dat`, `world/levelname.txt` and `player.nbt`; its bytes differ from one recording to the next, its content does not. Then run `go test ./content/skygear/ -run 'TestOldSaveLoads$' -count=1` three times without the variable: it passes each time. From then on the two files are frozen: T110 to T113 do not record them again.


`content/skygear/oldsave_test.go` (new)

```go
package skygear

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/playerstate"
	"github.com/presentkim/skygear/content/skygear/hopper"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/storage"
)

// The old save is a save a build of 2026-11-11 wrote, kept as a file:
// testdata/save/oldsave.zip holds its world directory and the stored state of
// one player (player.nbt, the bytes of save_players.state), and
// testdata/save/oldsave.golden lists what that build read back from it, by
// name. The world holds, in the chunk at the origin:
//
//   - every state of every custom block, from y 100 up, 256 to a layer;
//   - every custom item, one to a slot, in chests at y 120;
//   - at y 122 a crusher, a drawer and a wooden hopper that hold custom items.
//
// The player carries 36 custom items.
const (
	oldSaveZip    = "testdata/save/oldsave.zip"
	oldSaveGolden = "testdata/save/oldsave.golden"
	oldSaveBlockY = 100
	oldSaveChestY = 120
	oldSaveHeldY  = 122
	oldSaveTopY   = 124
)

// TestOldSaveLoads: this build reads the old save as the build that wrote it
// did. Every block is the block of the same name and properties and every
// stored item the item of the same name, metadata value and count, whatever
// numeric ids the blocks and items have in this build: a custom item's
// runtime id follows the order the packages are initialised in, and no stored
// byte holds one.
//
// SKYGEAR_GOLDEN=update writes the save anew with this build and records what
// it reads back. That is for a change of what the save holds; a change of the
// game that makes this test fail has changed what old saves load as.
func TestOldSaveLoads(t *testing.T) {
	var wrote map[string]string
	if os.Getenv("SKYGEAR_GOLDEN") == "update" {
		wrote = writeOldSave(t)
	}
	dir := t.TempDir()
	unzipOldSave(t, dir)
	lines := map[string]string{}

	s, closeAll := openPersisted(t, dir)
	s.World.Do(func(tx *world.Tx) { oldSaveWorldLines(tx, lines) })
	closeAll()

	blob, err := os.ReadFile(filepath.Join(dir, "player.nbt"))
	if err != nil {
		t.Fatal(err)
	}
	state, err := playerstate.Unmarshal(blob)
	if err != nil {
		t.Fatal(err)
	}
	ps := testSession(t)
	withPlayer(t, ps, spawnPlayer(ps), func(p *player.Player, _ *world.Tx) {
		playerstate.Apply(p, state)
		oldSavePlayerLines(p, lines)
	})

	// The build that writes the save reads back what it wrote: every block
	// under its name and every stack whole. A machine's properties may
	// differ: its tier and a sieve's look are rebuilt from its stored data.
	if wrote != nil {
		if len(wrote) != len(lines) {
			t.Fatalf("wrote %d lines, read back %d", len(wrote), len(lines))
		}
		for k, v := range wrote {
			a, _, _ := strings.Cut(v, "{")
			b, _, _ := strings.Cut(lines[k], "{")
			if a != b {
				t.Fatalf("%s: wrote %q, read back %q", k, v, lines[k])
			}
		}
	}
	pinGolden(t, oldSaveGolden, lines)
}

// oldSaveWorldLines adds to lines what the chunk of the old save holds: a
// line per block that is not air (block/x,y,z) and a line per stack a block
// holds (item/x,y,z/slot).
func oldSaveWorldLines(tx *world.Tx, lines map[string]string) {
	for y := oldSaveBlockY; y < oldSaveTopY; y++ {
		for x := range 16 {
			for z := range 16 {
				pos := cube.Pos{x, y, z}
				b := tx.Block(pos)
				if _, air := b.(block.Air); air {
					continue
				}
				at := fmt.Sprintf("%d,%d,%d", x, y, z)
				lines["block/"+at] = blockText(b)
				for slot, st := range heldStacks(tx, pos, b) {
					if !st.Empty() {
						lines[fmt.Sprintf("item/%s/%02d", at, slot)] = stackText(st)
					}
				}
			}
		}
	}
}

// oldSavePlayerLines adds to lines a line per stack p carries
// (player/inventory/slot).
func oldSavePlayerLines(p *player.Player, lines map[string]string) {
	for slot, st := range p.Inventory().Slots() {
		if !st.Empty() {
			lines[fmt.Sprintf("player/inventory/%02d", slot)] = stackText(st)
		}
	}
}

// blockText is a block by name: its name and its properties, sorted.
func blockText(b world.Block) string {
	name, props := b.EncodeBlock()
	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%#v", k, props[k]))
	}
	return name + "{" + strings.Join(parts, ",") + "}"
}

// stackText is a stack by name: the item's name and metadata value, and the
// count.
func stackText(st item.Stack) string {
	name, meta := st.Item().EncodeItem()
	return fmt.Sprintf("%s:%d x%d", name, meta, st.Count())
}

// heldStacks are the stacks the block at pos holds, by slot: a chest's and a
// hopper's slots, a machine's input and output slots, a drawer's stored item
// with its count.
func heldStacks(tx *world.Tx, pos cube.Pos, b world.Block) map[int]item.Stack {
	out := map[int]item.Stack{}
	switch b := b.(type) {
	case block.Chest:
		for slot, st := range b.Inventory(tx, pos).Slots() {
			out[slot] = st
		}
	case hopper.FlowHopper:
		if b.Inv != nil {
			for slot, st := range b.Inv.Slots() {
				out[slot] = st
			}
		}
	case machine.Machine:
		if b.State != nil {
			slots := b.State.Inv().Slots()
			for _, slot := range append(append([]int{}, b.State.Def.Input...), b.State.Def.Output...) {
				out[slot] = slots[slot]
			}
		}
	case storage.Drawer:
		if !b.Item.Empty() && b.Count > 0 {
			out[0] = b.Item.Grow(int(b.Count) - b.Item.Count())
		}
	}
	return out
}

// oldSaveItems are the custom items in the order of their names.
func oldSaveItems() []world.Item {
	var items []world.Item
	for _, it := range world.CustomItems() {
		items = append(items, it)
	}
	sort.Slice(items, func(i, j int) bool {
		a, _ := items[i].EncodeItem()
		b, _ := items[j].EncodeItem()
		return a < b
	})
	return items
}

// writeOldSave writes the old save with this build, packs it into oldSaveZip
// and returns the lines of what it wrote.
func writeOldSave(t *testing.T) map[string]string {
	t.Helper()
	wrote := map[string]string{}
	dir := t.TempDir()
	quiet := &world.SetOpts{DisableBlockUpdates: true, DisableLiquidDisplacement: true}
	items := oldSaveItems()

	// Every state of every custom block, in the order of name and properties.
	var states []world.Block
	for _, b := range world.Blocks() {
		if name, _ := b.EncodeBlock(); strings.HasPrefix(name, "skygear:") {
			states = append(states, b)
		}
	}
	sort.Slice(states, func(i, j int) bool { return blockText(states[i]) < blockText(states[j]) })
	if max := 256 * (oldSaveChestY - oldSaveBlockY); len(states) > max {
		t.Fatalf("%d custom block states, room for %d", len(states), max)
	}

	s, closeAll := openPersisted(t, dir)
	s.World.Do(func(tx *world.Tx) {
		for i, b := range states {
			tx.SetBlock(cube.Pos{i % 16, oldSaveBlockY + i/256, i / 16 % 16}, b, quiet)
		}
		for i, it := range items {
			pos := cube.Pos{i / 27 % 16, oldSaveChestY, i / 27 / 16}
			if i%27 == 0 {
				tx.SetBlock(pos, block.NewChest(), quiet)
			}
			_ = tx.Block(pos).(block.Chest).Inventory(tx, pos).SetItem(i%27, item.NewStack(it, 1))
		}
		m := machine.New(machine.KindCrusher)
		_ = m.State.Inv().SetItem(m.State.Def.Input[0], item.NewStack(items[0], 3))
		_ = m.State.Inv().SetItem(m.State.Def.Output[0], item.NewStack(items[len(items)/2], 2))
		tx.SetBlock(cube.Pos{0, oldSaveHeldY, 0}, m, quiet)
		tx.SetBlock(cube.Pos{2, oldSaveHeldY, 0}, storage.Drawer{Facing: cube.North, Item: item.NewStack(items[len(items)/3], 1), Count: 77}, quiet)
		h := hopper.NewFlowHopper("wood_hopper", cube.FaceDown)
		_ = h.Inv.SetItem(0, item.NewStack(items[len(items)-1], 5))
		tx.SetBlock(cube.Pos{4, oldSaveHeldY, 0}, h, quiet)
		oldSaveWorldLines(tx, wrote)
	})
	closeAll()

	var state playerstate.State
	ps := testSession(t)
	withPlayer(t, ps, spawnPlayer(ps), func(p *player.Player, _ *world.Tx) {
		for slot := range 36 {
			_ = p.Inventory().SetItem(slot, item.NewStack(items[slot*len(items)/36], 1+slot%3))
		}
		oldSavePlayerLines(p, wrote)
		state = playerstate.Capture(p)
	})
	blob, err := playerstate.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}

	// The world directory without leveldb's lock and its log of what it did,
	// and the player's state, in the order of the paths.
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	add := func(name string, data []byte) {
		w, err := zw.Create(name)
		if err == nil {
			_, err = w.Write(data)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	root := filepath.Join(dir, "world")
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() == "LOCK" || d.Name() == "LOG" {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		add(filepath.ToSlash(rel), data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	add("player.nbt", blob)
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(oldSaveZip), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(oldSaveZip, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return wrote
}

// unzipOldSave unpacks oldSaveZip into dir.
func unzipOldSave(t *testing.T, dir string) {
	t.Helper()
	zr, err := zip.OpenReader(oldSaveZip)
	if err != nil {
		t.Fatalf("%v\nrecord the old save with SKYGEAR_GOLDEN=update", err)
	}
	defer zr.Close()
	for _, f := range zr.File {
		path := filepath.Join(dir, filepath.FromSlash(f.Name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		r, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(r)
		_ = r.Close()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
```

### REF T110

Two edits: delete `content/skygear/hopper.go` (the whole file is `package skygear`, the import of `hopper` and `func init() { hopper.Register() }`), and turn `Register` into the package's `init`. The body of the function is not touched.

The tree is red afterwards, on purpose, in exactly these lines of `go test ./... 2>&1 | grep -E '^(FAIL|--- FAIL|    --- FAIL)|differs'`:

```
--- FAIL: TestWire (…)
    --- FAIL: TestWire/items (…)
        wire_test.go:121: golden testdata/items.golden differs: 698 changed, 0 added, 0 removed
    --- FAIL: TestWire/recipes (…)
        wire_test.go:129: golden testdata/recipes.golden differs: 327 changed, 0 added, 0 removed
    --- FAIL: TestWire/creative (…)
        wire_test.go:137: golden testdata/creative.golden differs: 243 changed, 0 added, 0 removed
FAIL
FAIL	github.com/presentkim/skygear/content/skygear/wire	…
FAIL
```

61 packages are `ok`. `go test ./content/skygear/wire/ -run 'TestWire$' -count=1 -v 2>&1 | grep -- '--- '` shows `PASS: TestWire/blocks`, `PASS: TestWire/by_name` and `PASS: TestWire/sensitivity`. Any other failure, another count, an added or removed entry, or a failing `by name`: the change is wrong or the base differs; stop and report. Do not record anything.


`content/skygear/hopper/hopper.go`

```diff
diff --git a/content/skygear/hopper/hopper.go b/content/skygear/hopper/hopper.go
index 829c11d..c115031 100644
--- a/content/skygear/hopper/hopper.go
+++ b/content/skygear/hopper/hopper.go
@@ -361,8 +361,8 @@ func HopperLines(lang string, h FlowHopper) []string {
 	return append(lines, fmt.Sprintf(i18n.T(lang, "hopper.lines.rate"), float64(h.Info().interval)/20))
 }
 
-// Register registers the hopper blocks and items.
-func Register() {
+// The hopper blocks and items are registered when the package is initialised.
+func init() {
 	for _, id := range HopperOrder {
 		k := hopperKinds[id]
 		k.hash = block.NextHash()
```

### REF T111

`content/skygear/machine/tinkertable.go` is deleted. The two hook bodies in `tinker/kinds.go` are those of the deleted file, with `tinker.ShowTable` now `ShowTable` and `NoLines` now `machine.NoLines`. `content/skygear/features.go` and `content/skygear/debug_ui.go` are not edited (decisions 3 and 6).

The tree stays red, on purpose, in the three lines of T110 and one more package:

```
--- FAIL: TestPopupAddActor (…)
    popup_wire_test.go:183: golden testdata/popup.golden differs: 2 changed, 0 added, 0 removed
FAIL
FAIL	github.com/presentkim/skygear/content/skygear/dimension	…
```

60 packages are `ok`; the counts of `wire` are still 698, 327 and 243. These pass, and are run by name: `go test ./content/skygear/ -run 'TestPin|TestKind|TestLayers|TestFeature|TestMachine|TestOldSaveLoads|TestForeignKindsRoundTrip' -count=1`, `go test ./content/skygear/tinker/ ./content/skygear/machine/ -count=1`, `TestWire/by_name` as in T110. `git status --short content/skygear/testdata content/skygear/wire/testdata content/skygear/dimension/testdata` is empty. Do not record anything.


`content/skygear/machine/machine.go`

```diff
diff --git a/content/skygear/machine/machine.go b/content/skygear/machine/machine.go
index 44e40ab..18d1b9b 100644
--- a/content/skygear/machine/machine.go
+++ b/content/skygear/machine/machine.go
@@ -286,9 +286,8 @@ type Def struct {
 	Owner      string
 	registered bool
 	// hooks is what the owner registered: the zero value until then and for
-	// the machine feature's own kinds, but for the tinker table, whose hooks
-	// tinkertable.go sets. Only Register writes it for a kind with an owner;
-	// Hooks reads it.
+	// the machine feature's own kinds. Only Register writes it; Hooks reads
+	// it.
 	hooks Behaviour
 }
 
@@ -351,7 +350,7 @@ var Kinds = map[string]*Def{
 		helpKeys: []string{"machine.drawer_interface.help.1", "machine.drawer_interface.help.2"},
 	},
 	KindTinker: {
-		ID: idTinker, nameKey: "machine.tinker.name", Interval: 1, Indicator: 10, NoDisplay: true,
+		ID: idTinker, Owner: "tinker", nameKey: "machine.tinker.name", Interval: 1, Indicator: 10, NoDisplay: true,
 		helpKeys: []string{"machine.tinker.help.1", "machine.tinker.help.2"},
 	},
 	KindCollector: {
```

`content/skygear/machine/kinds.go`

```diff
diff --git a/content/skygear/machine/kinds.go b/content/skygear/machine/kinds.go
index 1d5d447..db9c1bd 100644
--- a/content/skygear/machine/kinds.go
+++ b/content/skygear/machine/kinds.go
@@ -72,7 +72,7 @@ func Register(kind string, b Behaviour) {
 }
 
 // Hooks is the behaviour registered for the kind: the zero value for a kind
-// of the machine feature itself, but for the tinker table (tinkertable.go).
+// of the machine feature itself.
 func (k *Def) Hooks() Behaviour { return k.hooks }
 
 // CheckKinds reports the kinds whose owner did not register their behaviour.
```

`content/skygear/machine/doc.go`

```diff
diff --git a/content/skygear/machine/doc.go b/content/skygear/machine/doc.go
index 95b25fd..f27f006 100644
--- a/content/skygear/machine/doc.go
+++ b/content/skygear/machine/doc.go
@@ -25,10 +25,10 @@
 //     stall alerts.
 //
 // The kinds of other features (the drawer controller, the harvester, the
-// nodes of the storage network ...) are declared here like every kind, so
-// that the blocks and items register in one order, and name their feature as
-// Owner. The feature hands the kind's Behaviour (Click, Run, Insert, Peek,
-// Extract, Accepts, Lines, Info) to Register from its Setup; CheckKinds, the
-// machine feature's Check, stops the installation when one did not
-// (kinds.go). The tinker table is set in tinkertable.go.
+// tinker table, the nodes of the storage network ...) are declared here like
+// every kind, so that the blocks and items register in one order, and name
+// their feature as Owner. The feature hands the kind's Behaviour (Click, Run,
+// Insert, Peek, Extract, Accepts, Lines, Info) to Register from its Setup;
+// CheckKinds, the machine feature's Check, stops the installation when one
+// did not (kinds.go). This package imports none of those features.
 package machine
```

`content/skygear/tinker/kinds.go` (new)

```go
package tinker

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/machine"
)

// registerKinds hands the machine frame the behaviour of the tinker's machine
// kind: a click opens the tinker table, whose panel has no lines.
func registerKinds() {
	machine.Register(machine.KindTinker, machine.Behaviour{
		Click: func(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool {
			if !s.Debounced(p) {
				ShowTable(s, p, tx, pos)
			}
			return true
		},
		Lines: machine.NoLines,
	})
}
```

`content/skygear/tinker/subscribe.go`

```diff
diff --git a/content/skygear/tinker/subscribe.go b/content/skygear/tinker/subscribe.go
index 347cd2a..09b2da0 100644
--- a/content/skygear/tinker/subscribe.go
+++ b/content/skygear/tinker/subscribe.go
@@ -24,6 +24,7 @@ func subscribe() {
 	subscribeTools()
 	subscribeVeinToggle()
 	followLook()
+	registerKinds()
 }
 
 func subscribeFood() {
```

`content/skygear/tinker/kinds_test.go` (new)

```go
package tinker_test

import (
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/presentkim/skygear/content/skygear/machine"
)

// TestTinkerTableKind: the tinker's Setup (setup_test.go) registered the
// behaviour of its machine kind: the tinker table takes the click itself and
// its panel has no lines, and the machine feature's check does not miss it.
func TestTinkerTableKind(t *testing.T) {
	d := machine.Kinds[machine.KindTinker]
	if d.Owner != "tinker" {
		t.Fatalf("the tinker table's owner is %q", d.Owner)
	}
	h := d.Hooks()
	if h.Click == nil {
		t.Error("the tinker table has no click of its own: a click would open the machine window")
	}
	if h.Lines == nil || h.Lines("ko", nil, cube.Pos{}) != nil {
		t.Error("the tinker table's panel lines are not machine.NoLines")
	}
	if err := machine.CheckKinds(); err != nil && strings.Contains(err.Error(), machine.KindTinker) {
		t.Errorf("the check misses the tinker table: %v", err)
	}
}
```

`content/skygear/kinds_test.go`

```diff
diff --git a/content/skygear/kinds_test.go b/content/skygear/kinds_test.go
index a2ef2b3..6c0825c 100644
--- a/content/skygear/kinds_test.go
+++ b/content/skygear/kinds_test.go
@@ -11,6 +11,7 @@ import (
 // that is not here is the machine feature's own.
 var kindOwners = map[string]string{
 	machine.KindHarvester: "farm",
+	machine.KindTinker:    "tinker",
 	machine.KindCollector: "dimension", machine.KindRelicTable: "dimension",
 	machine.KindDrawerController: "storage", machine.KindDrawerTrim: "storage", machine.KindDrawerInterface: "storage",
 	machine.KindNetController: "netstore", machine.KindNetCable: "netstore", machine.KindDiskDrive: "netstore",
```

`content/skygear/layers_test.go`

```diff
diff --git a/content/skygear/layers_test.go b/content/skygear/layers_test.go
index 865f6c4..e3754d3 100644
--- a/content/skygear/layers_test.go
+++ b/content/skygear/layers_test.go
@@ -40,7 +40,7 @@ var layers = []string{
 	// progress
 	"progress", "island", "climate",
 	// features
-	"tinker", "machine", "conduit", "storage", "farm", "fairy", "dimension", "handwork", "expand", "copier", "wand",
+	"machine", "tinker", "conduit", "storage", "farm", "fairy", "dimension", "handwork", "expand", "copier", "wand",
 	"netstore", "elevator",
 	// content lists
 	"catalog",
@@ -89,7 +89,7 @@ func TestLayers(t *testing.T) {
 // imports another one of it only if its row lists it, and every listed import
 // exists. A feature without a row imports no sibling.
 var siblings = map[string][]string{
-	"machine":   {"tinker"},
+	"tinker":    {"machine"},
 	"conduit":   {"machine"},
 	"storage":   {"machine"},
 	"farm":      {"machine"},
@@ -101,7 +101,7 @@ var siblings = map[string][]string{
 
 // featureLayer is the "features" group of layers.
 var featureLayer = []string{
-	"tinker", "machine", "conduit", "storage", "farm", "fairy", "dimension", "handwork", "expand", "copier", "wand",
+	"machine", "tinker", "conduit", "storage", "farm", "fairy", "dimension", "handwork", "expand", "copier", "wand",
 	"netstore", "elevator",
 }
 
@@ -153,47 +153,22 @@ func TestFeatureSiblings(t *testing.T) {
 	}
 }
 
-// machineFeatureImports are the packages of the feature layer machine may
-// import. The features that own a machine kind import machine and register
-// the kind's behaviour (machine.Register); machine does not import them back.
-// tinker is the one exception, and only in machine/tinkertable.go: the tinker
-// feature cannot import machine, because the tinker's tools must register
-// before the machines (the client goldens pin the item ids), so machine sets
-// the tinker table's behaviour itself.
-var machineFeatureImports = map[string]string{"tinker": "tinkertable.go"}
-
 // TestMachineImportsNoFeature: machine imports no package of the feature
-// layer but those of machineFeatureImports, each only in its one file, and
-// its siblings row lists exactly those.
+// layer and has no siblings row. The features that own a machine kind import
+// machine and register the kind's behaviour (machine.Register); machine does
+// not import them back.
 func TestMachineImportsNoFeature(t *testing.T) {
 	const prefix = "github.com/presentkim/skygear/content/skygear/"
-	if row := siblings["machine"]; len(row) != len(machineFeatureImports) {
-		t.Errorf("the siblings row of machine is %v, want the %d of machineFeatureImports", row, len(machineFeatureImports))
-	}
-	for _, dep := range siblings["machine"] {
-		if _, ok := machineFeatureImports[dep]; !ok {
-			t.Errorf("siblings lists machine -> %s: the feature registers its kind's behaviour with machine.Register instead", dep)
-		}
+	if row, ok := siblings["machine"]; ok {
+		t.Errorf("siblings has the row machine -> %v: a feature registers its kind's behaviour with machine.Register instead", row)
 	}
-	files, err := os.ReadDir("machine")
+	pkg, err := build.Default.ImportDir("machine", 0)
 	if err != nil {
 		t.Fatal(err)
 	}
-	for _, f := range files {
-		if !strings.HasSuffix(f.Name(), ".go") || strings.HasSuffix(f.Name(), "_test.go") {
-			continue
-		}
-		src, err := os.ReadFile("machine/" + f.Name())
-		if err != nil {
-			t.Fatal(err)
-		}
-		for _, dep := range featureLayer {
-			if !strings.Contains(string(src), `"`+prefix+dep+`"`) {
-				continue
-			}
-			if machineFeatureImports[dep] != f.Name() {
-				t.Errorf("machine/%s imports the feature %s: the feature registers its kind's behaviour with machine.Register instead", f.Name(), dep)
-			}
+	for _, imp := range pkg.Imports {
+		if dep, ok := strings.CutPrefix(imp, prefix); ok && slices.Contains(featureLayer, dep) {
+			t.Errorf("machine imports the feature %s: the feature registers its kind's behaviour with machine.Register instead", dep)
 		}
 	}
 }
```

### REF T112

No source file changes. The four numbered goldens are recorded once, and the proof is checked.

```sh
SKYGEAR_GOLDEN=update go test ./content/skygear/wire/ -run 'TestWire$' -count=1
SKYGEAR_GOLDEN=update go test ./content/skygear/dimension/ -run 'TestPopupAddActor$' -count=1
git status --short content/skygear/wire/testdata content/skygear/dimension/testdata content/skygear/testdata
git diff --numstat -- content/skygear/wire/testdata content/skygear/dimension/testdata
```

`git status` lists exactly these four, modified, and nothing else (no `*.byname.golden`, nothing under `content/skygear/testdata`):

```
 M content/skygear/dimension/testdata/popup.golden
 M content/skygear/wire/testdata/creative.golden
 M content/skygear/wire/testdata/items.golden
 M content/skygear/wire/testdata/recipes.golden
```

`git diff --numstat` prints:

```
2	2	content/skygear/dimension/testdata/popup.golden
243	243	content/skygear/wire/testdata/creative.golden
698	698	content/skygear/wire/testdata/items.golden
327	327	content/skygear/wire/testdata/recipes.golden
```

If a file by name, `oldsave.golden`, `oldsave.zip` or a pin is listed, or a count differs: something other than the numbering changed. Undo the recording (`git checkout -- content/skygear/wire/testdata content/skygear/dimension/testdata` is the one checkout this task may run, and only for that), and report `BLOCKED` with the output.

What changed inside the entries is shown once, for the report. The texts of the tree before T110 are needed for it; the last commit is that tree, if it holds T108 and T109 (`git log -3 --oneline` shows their subjects; if it does not, leave this block out and say so in the report):

```sh
S=<scratchpad>/T112
git worktree add --detach $S/before HEAD
(cd $S/before && SKYGEAR_GOLDEN_DUMP=$S/a go test ./content/skygear/wire/ ./content/skygear/dimension/ -run 'TestWire$|TestPopupAddActor$' -count=1)
SKYGEAR_GOLDEN_DUMP=$S/b go test ./content/skygear/wire/ ./content/skygear/dimension/ -run 'TestWire$|TestPopupAddActor$' -count=1
diff -r $S/a $S/b | grep '^[<>]' | sed 's/[0-9]\+/N/g' | sort | uniq -c
for g in items recipes creative popup; do diff -rq $S/a/$g.byname.golden $S/b/$g.byname.golden | wc -l; done
git worktree remove --force $S/before && rm -rf $S
```

Expected: six kinds of line and no other, `698` each of `<   RuntimeID: intN(N)` and `>   RuntimeID: intN(N)`, `327` and `243` each of `NetworkID: intN(N)` (at two indentations), `2` each of `uintN(N): intN(N)`; and four times `0` for the texts by name.

Then the whole suite: `go build ./... && go vet ./... && go test ./...`, 62 packages `ok`. The commit holds the files of T110, T111 and T112 and its message carries the four `numstat` lines.


### REF T113

Documents only, Korean, saying what is. The text is the diffs below; copy it. `docs/TODO.md` needs no change: no line of it names the tinker table's exception or the root's `hopper.go` (its line on `debug_ui.go` and `machine.KindTinker` stays true).

Check afterwards: `grep -rn 'tinkertable\|machineFeatureImports\|hopper\.Register\|호퍼 등록\|루트가 등록하는 호퍼' docs/tech content/skygear/README.md` prints nothing, and `go test ./...` is green (62 packages `ok`).


`docs/tech/extension.md`

````diff
diff --git a/docs/tech/extension.md b/docs/tech/extension.md
index 5ba0bd6..7045b76 100644
--- a/docs/tech/extension.md
+++ b/docs/tech/extension.md
@@ -23,7 +23,7 @@ Dragonfly 자체를 확장하는 방법은 [dragonfly.md](dragonfly.md) 에 있
 - 데이터: `locales`, `naming`, `textfmt`, `formicon`, `blockmodel`, `netshape`, `sgitem`, `sievetable`, `wintext`, `machinerecipe`, `advancement`, `hopper`, `migrate`, `wire`
 - 핵심: `core`, `network`, `achievement`, `preview`, `look`, `autofill`, `decor`
 - 진행: `progress`, `island`, `climate`
-- 기능: `tinker`, `machine`, `conduit`, `storage`, `farm`, `fairy`, `dimension`, `handwork`, `expand`, `copier`, `wand`, `netstore`, `elevator`
+- 기능: `machine`, `tinker`, `conduit`, `storage`, `farm`, `fairy`, `dimension`, `handwork`, `expand`, `copier`, `wand`, `netstore`, `elevator`
 - 내용 목록: `catalog`
 - 화면: `codex`, `screens`
 
@@ -36,7 +36,6 @@ Dragonfly 자체를 확장하는 방법은 [dragonfly.md](dragonfly.md) 에 있
 | `features.go` | 기능 목록 `Features` 와 `core.Install`. 루트 자신의 기능 `itemsFeature` 는 아이템 항목의 이름을 번역 키로 바꾸고(`naming.LocalItemName`) 클라이언트가 빈 나무 양동이로 액체를 겨누게 한다(`sgitem.ClipWoodenBucket`) |
 | `game.go` | `Game` 과 `session`. 플랫폼의 `instance.Game`·`instance.Session` 을 구현한다. `New` 가 기능의 명령어(`core.Commands()`)와 창작 목록(`catalog.RegisterCreativeItems`)과 운영자 명령을 등록한다 |
 | `state.go` | 세이브 상태 문서의 버전(`State`, `core.NewSaved`) |
-| `hopper.go` | `hopper` 는 기능이 아닌 데이터 칸 패키지라 루트가 `init` 에서 호퍼 블록과 아이템을 등록한다(`hopper.Register`) |
 | `blockcounts.go` | 커스텀 블록 수와 상태 수(`CustomBlockCounts`, `LogCustomBlockCounts`). 서버가 시작할 때 로그에 남긴다 |
 | `admin.go` | 운영자 명령 `/sgadmin` |
 | `debug.go`, `debug_factory.go`, `debug_oreui.go`, `debug_ui.go` | 운영자 디버그 명령 `/sgdebug` 와 그 도구 |
@@ -242,7 +241,7 @@ Dragonfly 자체를 확장하는 방법은 [dragonfly.md](dragonfly.md) 에 있
    - 소유한 기능이 등록하지 않으면 게임이 시작하지 않는다. 기계 기능의 `Check` 인 `machine.CheckKinds` 가 `machine kinds without behaviour: …` 오류를 돌려주고 `Install` 이 그 오류로 패닉한다. 소유 기능이 기능 목록에 없을 때도 같다.
    - 기능 패키지의 테스트는 기능 목록 없이 돈다. 종류의 동작이 필요한 테스트는 테스트 파일의 `init` 에서 `coretest.Setup(<기능>.Feature)` 를 부른다(`netstore/keys_test.go`, 아래 "기능의 테스트").
 
-   팅커 테이블은 `Owner` 가 비어 있고 `machine/tinkertable.go` 가 동작을 정한다. `tinker` 가 `machine` 을 import 하면 `machine` 이 `tinker` 보다 먼저 초기화되어 기계 아이템이 팅커 도구보다 먼저 등록되고, 클라이언트가 받는 아이템 id 가 바뀌기 때문이다. 루트가 `hopper` 를 등록하는 것(`hopper.go`)과 같은 사정이다.
+   `machine` 은 기능 칸의 패키지를 하나도 import 하지 않는다. 종류를 소유한 기능이 `machine` 을 import 하므로 그 기능의 패키지는 `machine` 보다 나중에 초기화된다. 그래서 그 기능이 `init` 에서 등록하는 아이템은 기계 아이템보다 뒤의 id 를 받는다. import 를 더하거나 방향을 바꿔 초기화 순서가 달라지면 커스텀 아이템의 id 가 바뀐다(아래 "핀"의 "id 만 바뀌는 변경").
 6. 기능의 아이템에 등급이나 단계 표시가 있으면 그 아이템에 `Decor()` 메서드를 둔다(`slotdeco.Decorated`). 예는 `fairy/fairymagic.go` 의 요정 판이다.
 
    ```go
@@ -374,7 +373,7 @@ func init() { coretest.Setup(farm.Feature) }
 
 - 핀과 골든(`pin_*_test.go`), 구조 규칙(`layers_test.go`, `kinds_test.go`, `testsupport_test.go` 등), 벤치마크.
 - `catalog` 의 레시피와 연구, 도감, 리소스팩, 세이브(저장, 다시 열기, 마이그레이션)를 보는 테스트.
-- 기능 셋 이상이 설치되어야 통과하는 테스트, 루트가 등록하는 호퍼(`hopper.go`)가 필요한 테스트, 자기가 import 하지 않는 기능이 등록한 아이템이 필요한 테스트.
+- 기능 셋 이상이 설치되어야 통과하는 테스트, 자기가 import 하지 않는 기능이 등록한 아이템이 필요한 테스트.
 - 루트의 내보내지 않은 이름(`Game`, `session`, `giveItems` 등)을 쓰는 테스트와 런타임 id 를 보는 테스트.
 
 루트의 `helpers_test.go` 는 같은 도우미를 루트의 `session` 위에서 준다(`testSession`, `loadAround`, `spawnPlayer`, `withPlayer`, `tick`, `placeTestMachine`, `machinePos`). `testSession` 은 데이터베이스와 `Game` 도 만든다.
@@ -383,7 +382,7 @@ func init() { coretest.Setup(farm.Feature) }
 
 - 패키지의 테스트는 루트 패키지를 import 하지 않고, 내용 목록 아래 칸의 패키지 테스트는 `catalog`, `codex`, `screens` 를 import 하지 않는다(`TestPackageTestsStayBelow`).
 - 테스트 지원 패키지는 테스트 파일만 import 한다(`TestSupportPackagesAreTestOnly`).
-- `dimension` 의 테스트 바이너리에서는 `popup_wire_test.go` 가 서버에 접속해 `testdata/popup.golden` 의 아이템 id 를 본다. 이 패키지의 테스트에 블록이나 아이템을 등록하는 패키지를 새로 import 하면 id 가 바뀐다.
+- `dimension` 의 `popup_wire_test.go` 는 서버에 접속해 데미지 숫자의 패킷을 `testdata/popup.golden` 에 고정한다. 그 패킷에는 공기 블록의 런타임 id 가 들어 있다. 블록의 런타임 id 는 테스트 바이너리가 등록한 블록 전체로 정해진다. `dimension` 이나 그 테스트의 import 가 달라져 등록되는 블록이 늘거나 줄면 이 값이 바뀐다. 같은 패킷을 블록 이름으로 적은 `testdata/popup.byname.golden` 은 그때도 그대로다.
 
 ## 클라이언트 항목 고치기
 
@@ -413,7 +412,7 @@ func init() { coretest.Setup(farm.Feature) }
 | 기능 패키지는 루트를 import 하지 않는다(간접도). `core` 는 기능을 import 하지 않는다. `common` 은 `content/`, `platform/` 을 import 하지 않는다 | `TestFeaturesLiftable` |
 | 패키지는 `layers` 의 아래 칸만 import 하고, 같은 칸의 기능 import 는 `siblings` 가 허용한 것뿐이다. `featureLayer` 는 `layers` 의 기능 칸과 같다 | `TestLayers`, `TestFeatureSiblings` |
 | `look` 은 `core`, `naming` 만, `decor` 는 `naming` 만 import 한다. 블록과 아이템은 인터페이스로 자기를 설명한다 | `TestCapabilityImports` |
-| `machine` 은 기능 칸의 패키지를 import 하지 않는다. 예외는 `machine/tinkertable.go` 의 `tinker` 하나다(`machineFeatureImports`) | `TestMachineImportsNoFeature` |
+| `machine` 은 기능 칸의 패키지를 import 하지 않고 `siblings` 에 자기 줄이 없다 | `TestMachineImportsNoFeature` |
 | 다른 기능의 기계 종류는 `kindOwners` 의 기능이 소유하고, 그 기능은 목록에 있으며 `Setup` 에서 종류를 등록한다 | `TestKindOwners` |
 | 등록되지 않은 종류가 있으면 `machine.CheckKinds` 가 종류와 기능 이름을 든 오류를 돌려준다. `machine.Register` 는 없는 종류, 소유 기능이 없는 종류, 두 번째 등록, 검사가 끝난 뒤의 등록에 패닉한다 | `machine.TestCheckKindsNamesTheMissingOwner`, `machine.TestRegisterPanics` |
 | 기능의 `Check` 는 모든 기능의 `Setup` 이 돈 뒤에 목록 순서대로 돈다 | `core.TestInstallRunsChecksAfterEverySetup` |
@@ -452,10 +451,14 @@ go test -run xxx -bench Emit -benchmem -count 5 ./content/skygear/core
 | `content/skygear/testdata/pin/kinds.golden` | 기계 종류마다 패널 밖에서 하는 일: 서서·웅크려 클릭한 결과와 열린 화면, 창의 정보 아이콘, 창 27칸의 역할과 칸이 받는 견본 아이템, 145줄 | `TestPinKinds` |
 | `content/skygear/testdata/pin/creative.golden` | 게임이 더한 창작 목록 항목을 클라이언트가 보이는 순서대로 그룹과 함께, 243개 | `TestPinCreative` |
 | `content/skygear/wire/testdata/*.golden` | 접속한 플레이어가 받는 블록 항목, 아이템 등록, 제작 데이터, 창작 목록의 항목별 해시 | `TestWire` |
+| `content/skygear/wire/testdata/items.byname.golden`, `recipes.byname.golden`, `creative.byname.golden` | 위의 아이템 등록, 제작 데이터, 창작 목록과 같은 항목인데 아이템 id 자리에 아이템 이름을 적은 것. 커스텀 아이템 id 의 집합(`meta/custom_ids`)도 담는다 | `TestWire` |
+| `content/skygear/dimension/testdata/popup.golden`, `popup.byname.golden` | 이차원 데미지 숫자와 경험치 보석의 패킷. `byname` 은 블록 런타임 id 자리에 블록 이름을 적은 것 | `dimension.TestPopupAddActor` |
+| `content/skygear/testdata/save/oldsave.zip`, `oldsave.golden` | 2026-11-11 의 빌드가 쓴 세이브(모든 커스텀 블록 상태, 상자에 든 모든 커스텀 아이템, 아이템을 든 분쇄기·서랍·나무 호퍼, 플레이어 상태)와 그 세이브를 읽은 결과를 이름으로 적은 2,822줄 | `TestOldSaveLoads` |
 
 ```sh
 go test ./content/skygear -run 'TestPin' -count=1     # 구독 순서, 장면, 패널, 기계 종류, 창작 목록
 go test ./content/skygear/wire/                        # 클라이언트 데이터
+go test ./content/skygear -run 'TestOldSaveLoads' -count=1   # 옛 세이브 읽기
 ```
 
 골든이 달라지면 테스트가 어느 줄(장면)이 어떻게 달라졌는지 `want` 와 `got` 로 보여 준다. 디스패치 순서나 클라이언트 데이터가 바뀌면 골든 차이로 나타난다.
@@ -467,3 +470,14 @@ SKYGEAR_GOLDEN=update go test ./content/skygear/wire/
 ```
 
 다시 기록한 골든은 `git diff` 로 의도한 줄만 바뀌었는지 확인하고, 커밋 메시지에 왜 바꿨는지 적는다. 테스트를 통과시키려고 다시 기록하지 않는다.
+
+### id 만 바뀌는 변경
+
+커스텀 아이템의 숫자 id 는 아이템이 등록된 순서다. 등록 순서는 패키지가 초기화되는 순서를 따른다. 패키지 사이의 import 를 더하거나 방향을 바꾸거나 등록하는 곳을 옮기면 플레이어가 보는 것은 그대로인데 `items.golden`, `recipes.golden`, `creative.golden` 의 수백 줄이 바뀐다. 이런 변경은 다음과 같이 확인한다.
+
+1. 코드를 바꾼 뒤 다시 기록하지 않고 테스트를 돌린다. `TestWire` 의 `items`, `recipes`, `creative` 만 실패하고 `by name` 은 통과해야 한다. `TestPin…` 과 `TestOldSaveLoads` 도 통과해야 한다.
+2. `SKYGEAR_GOLDEN=update go test ./content/skygear/wire/ -run 'TestWire$' -count=1` 로 다시 기록한다. `popup.golden` 이 달라졌으면 `SKYGEAR_GOLDEN=update go test ./content/skygear/dimension/ -run 'TestPopupAddActor$' -count=1` 도 돌린다.
+3. `git status --short` 에 `*.byname.golden`, `testdata/save/`, `testdata/pin/` 의 파일이 없어야 한다. 이름으로 적은 골든이 그대로면 바뀐 것은 id 뿐이다. 하나라도 있으면 id 말고 다른 것이 바뀐 것이므로 다시 기록한 것을 되돌리고 원인을 찾는다.
+4. `git diff --numstat` 의 줄 수를 커밋 메시지에 적는다.
+
+세이브는 블록과 아이템을 이름으로 저장하므로 id 가 바뀌어도 그대로 읽힌다. `TestOldSaveLoads` 가 옛 빌드가 쓴 세이브를 지금 빌드로 읽어 이를 확인한다. `oldsave.zip` 과 `oldsave.golden` 은 세이브에 담을 것을 바꿀 때만 다시 기록한다. 게임을 바꾼 뒤 이 테스트가 실패하면 옛 세이브가 전과 다르게 읽힌다는 뜻이다.
````

`docs/tech/machines.md`

```diff
diff --git a/docs/tech/machines.md b/docs/tech/machines.md
index adc1001..3afa2d3 100644
--- a/docs/tech/machines.md
+++ b/docs/tech/machines.md
@@ -59,6 +59,7 @@
 | `farm` | `harvester` | `Run` |
 | `storage` | `drawer_controller`, `drawer_interface` | `Click`, `Insert`, `Peek`, `Extract`, `Lines` |
 | `storage` | `drawer_trim` | `Click`(아무 일도 안 함), `Lines: machine.NoLines` |
+| `tinker` | `tinker_table` | `Click`, `Lines: machine.NoLines` |
 | `dimension` | `essence_collector` | `Click`, `Info`(받는 몬스터) |
 | `dimension` | `relic_table` | `Click`, `Lines: machine.NoLines` |
 | `netstore` | `net_controller` | `Click`, `Run`, `Insert`, `Peek`, `Extract` |
@@ -69,9 +70,8 @@
 | `netstore` | `net_external` | `Click`, `Run` |
 | `netstore` | `pattern_encoder` | `Click`, `Accepts` |
 | `netstore` | `net_autocrafter` | `Run`, `Accepts` |
-| 없음 | `tinker_table` | `machine/tinkertable.go` 가 정한다(`Click`, `Lines: NoLines`) |
 
-팅커 테이블은 `tinker` 가 `machine` 을 import 할 수 없어서 `machine` 이 직접 정한다. 그렇게 하면 기계 아이템이 팅커 도구보다 먼저 등록되어 클라이언트가 받는 아이템 id 가 바뀐다. `machine` 이 기능 칸의 패키지를 import 하는 곳은 이 파일 하나다.
+`machine` 은 기능 칸의 패키지를 하나도 import 하지 않는다(`TestMachineImportsNoFeature`). 종류를 소유한 기능이 `machine` 을 import 한다.
 
 ### 클라이언트 모양
```

`docs/tech/dragonfly.md`

````diff
diff --git a/docs/tech/dragonfly.md b/docs/tech/dragonfly.md
index 7efc302..78b2b6c 100644
--- a/docs/tech/dragonfly.md
+++ b/docs/tech/dragonfly.md
@@ -47,7 +47,9 @@ Dragonfly 의 동작 자체를 바꿔야 하거나, 패치 없이 같은 결과
 
 ## 클라이언트 데이터 검증
 
-`content/skygear/wire` 의 `TestWire` 는 서버를 `cmd/skygear` 와 같은 방식으로 세워 가짜 클라이언트를 한 번 접속시키고, 받은 블록 항목·아이템 등록·제작 데이터·창작 목록을 항목마다 해시해 `testdata/*.golden` 과 맞춘다. 접속을 흉내 내는 도구는 `common/sghook/hooktest` 다(`Join`, `Canon`, `Golden`).
+`content/skygear/wire` 의 `TestWire` 는 서버를 `cmd/skygear` 와 같은 방식으로 세워 가짜 클라이언트를 한 번 접속시키고, 받은 블록 항목·아이템 등록·제작 데이터·창작 목록을 항목마다 해시해 `testdata/*.golden` 과 맞춘다. 접속을 흉내 내는 도구는 `common/sghook/hooktest` 다(`Join`, `Canon`, `CanonWith`, `Golden`).
+
+`items.byname.golden`, `recipes.byname.golden`, `creative.byname.golden` 은 같은 항목을 아이템 id 대신 아이템 이름으로 적은 것이다(`wire/byname_test.go`, `hooktest.CanonWith`). 커스텀 아이템의 id 는 등록 순서라서 패키지 초기화 순서만 바뀌어도 앞의 세 골든은 수백 줄이 바뀌지만 이 셋은 그대로다. 확인하는 순서는 [extension.md](extension.md) 의 "id 만 바뀌는 변경"에 있다.
 
 ```sh
 go test ./content/skygear/wire/                              # 골든과 비교
````

`content/skygear/README.md`

```diff
diff --git a/content/skygear/README.md b/content/skygear/README.md
index 70e6303..fe0e681 100644
--- a/content/skygear/README.md
+++ b/content/skygear/README.md
@@ -27,7 +27,7 @@
 | | `wintext` | 기계 창 아이템의 글(보는 사람 언어로) |
 | | `machinerecipe` | 합금 제련기·실험실·자동 제작기 레시피 |
 | | `advancement` | 발전과제 트리·장·배치 |
-| | `hopper` | 나무·역방향 호퍼 |
+| | `hopper` | 나무·역방향 호퍼와 그 등록 |
 | | `migrate` | 세이브 데이터를 실행 중인 릴리스 형식으로 올리는 패치(`Patches`)와, 이름 변경(DragonFactory → Sky Gear) 전 세이브의 한 번짜리 변환(`Namespace`) |
 | | `wire` | 접속한 플레이어가 받는 데이터(블록 항목, 아이템 등록, 제작 데이터, 크리에이티브 목록)의 골든 파일과 그 테스트 |
 | 핵심 | `core` | 기능 커널: 세이브 하나의 실행 상태(`Session`: 월드·시계·난수·세이브 정보), 기능이 자기 데이터를 두는 방법(`Part`·`Players`·`Saved`), 훅과 이벤트(`PlayerHandler`·`WorldHandler`·`Lifecycle` 이 내보낸다), 기능 서술자(`Feature`·`Install`), 창작 목록 자리(`CreativeItem`·`CreativeItems`), 기능의 명령어(`Commands`·`PlayerSession`), 공용 도우미(아이템, 메뉴 버튼, 디버그 도형) |
@@ -40,8 +40,8 @@
 | 진행 | `progress` | 연구·발전과제 규칙, 레시피 잠금, 연구대 블록 |
 | | `island` | 소유 청크, 시작 위치, 청크 구매, 보이드 복귀, 분실물 보관함 |
 | | `climate` | 날씨·해·성장 시계 |
-| 기능 | `tinker` | 팅커 도구(재료·레벨·개조), 팅커 테이블, 광맥 채굴, 자동 도구 |
-| | `machine` | 기계의 틀(블록, 창, 틱, 에너지·유체, 단계)과 모든 종류의 목록(`Kinds`, `Order`, 다른 기능의 종류는 `Owner` 와 `Register`), 체, 가공 기계, 생산 기록과 정지 알림 |
+| 기능 | `machine` | 기계의 틀(블록, 창, 틱, 에너지·유체, 단계)과 모든 종류의 목록(`Kinds`, `Order`, 다른 기능의 종류는 `Owner` 와 `Register`), 체, 가공 기계, 생산 기록과 정지 알림 |
+| | `tinker` | 팅커 도구(재료·레벨·개조), 팅커 테이블(기계 종류 `tinker_table` 의 동작), 광맥 채굴, 자동 도구 |
 | | `conduit` | 도관 묶음 블록, 연결망 잇기, 표시 엔티티, 연결판과 그 화면 |
 | | `storage` | 서랍과 아이템 거치대, 저장고 컨트롤러와 목록 화면 |
 | | `farm` | 작물, 묘목·나무, 잎, 누에와 누에 나뭇잎, 퇴비통, 배럴과 마녀물, 수확기, 마법 식물 |
@@ -56,7 +56,7 @@
 | 내용 목록 | `catalog` | 연구 트리, 게임 아이템 레시피, 공용 아이템(`sgitem`, `hopper`)의 크리에이티브 항목과 크리에이티브 목록 등록(`RegisterCreativeItems`) |
 | 화면 | `codex` | 도감(JEI 같은 레시피 보기) |
 | | `screens` | 게임 메뉴, 섬 관리, 분실물, 도움말, 연구·발전과제 화면, 사이드바와 퀘스트 판, 메뉴·메뉴 아이템·도감·도움말 명령어 |
-| 게임 | `skygear` | 기능 목록(`features.go`, `core.Install`)과 플랫폼 접착제: `Game`·`session`(플랫폼 인터페이스 구현), 세이브 상태 버전(`state.go`), 호퍼 등록(`hopper.go`), 커스텀 블록 수 기록(`blockcounts.go`), 운영자 도구(`admin.go`, `debug*.go`). 훅에 구독하는 단계와 플레이어 명령어는 두지 않는다 |
+| 게임 | `skygear` | 기능 목록(`features.go`, `core.Install`)과 플랫폼 접착제: `Game`·`session`(플랫폼 인터페이스 구현), 세이브 상태 버전(`state.go`), 커스텀 블록 수 기록(`blockcounts.go`), 운영자 도구(`admin.go`, `debug*.go`). 훅에 구독하는 단계와 플레이어 명령어는 두지 않는다 |
 
 ## 기능의 데이터
```
