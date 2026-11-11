# Phase 3, parts 3-3 to 3-5: outline

This is an outline, not a machine plan. Parts 3-1 and 3-2 are built (`2026-11-10-cleanup-restructure-p3a.md` to `-p3e.md`, tasks T32 to T58). Part 3-3 is planned in full in `-p3f.md`, `-p3g.md` and `-p3h.md` (tasks T59 to T74) with the shared reference `-p3-capabilities.md`. Parts 3-4 and 3-5 below get their machine plans once 3-3 is built. Numbers are measured at 1ec765f unless marked.

## 3-3 in one paragraph

Sixteen tasks. The WAILA panel asks a block through `waila.Provider` (and `waila.Adapter` for vanilla blocks), so `look` imports `core` and `naming` only and sits in the core layer; `expand` hides the panel through `look.Hidden`, `tinker` and `wand` follow it through `look.Follow`. An item says its slot decoration through `slotdeco.Decorated`. The creative list is put together from `core.Feature.Creative` with explicit slots, in today's order. A feature registers its commands through `core.Feature.Commands`. The root's `names.go` goes to `naming`, autofill becomes its own package. Two new goldens (`look.golden`, 143 panels; `creative.golden`, 243 items) pin the output first. Left as they are, with reasons in the reference: the operator tools and `giveItems`, `itemsFeature`, the root's `hopper.go`, the copier's and the codex's knowledge of other features, the arena and screen helpers of `core`.

## Where 3-3 leaves the tree

- Root package, non-test: `features.go` (the list, `itemsFeature`, `Install`), `game.go` (`Game`, `session`, registering `core.Commands()`), `hopper.go` (one `init`), `state.go`, `blockcounts.go`, and the operator tools `admin.go`, `debug*.go`. 10 files, about 1,930 lines.
- A new feature touches: its own package, one line in `features.go`, the `layers` row, and the goldens that list what it adds on purpose (`order.golden`, `creative.golden` with the wire golden). Still central by decision: `catalog/recipes.go`, `catalog/research.go` with `progress/essence.go`, the codex cards, the help, the locales, the pack's index files.
- `machine` gained the panel code (`machine/look.go`, with the ten-clause switch of `MachineLines` now over its own package's kinds), one more definition hook (`Def.Lines`, filled by `storage`), its creative run (slot 50, all 36 kinds in `Order`) and its command.
- The root has 92 test files (88 at HEAD plus `pin_look_test.go`, `pin_creative_test.go`, `commands_test.go`, `decor_test.go`).

## 3-4 Machine kinds go home

Goal: `machine` keeps the frame (block, state, window, energy, fluid, tiers, statistics, monitor) and its own kinds; what a kind of another feature does is defined by that feature.

Measured:

- `machine/machine.go` is 1,972 of the package's 6,427 non-test lines. `machine.Kinds` and `machine.Order` hold 36 kinds: 20 of `machine`, 3 of storage (drawer controller, trim, interface), 1 of tinker (tinker table), 2 of dimension (collector, relic table), 1 of farm (harvester), 9 of netstore. Sixteen belong elsewhere (the review plan and the audit say 17).
- What the owners use of `machine` today: netstore 18 symbols in 178 places, storage 6 in 24, farm 5 in 22, dimension 6 in 11, fairy 2, conduit 2, copier 2. Above them `catalog` uses 40 symbols, `codex` 22, `screens` 11. The most used are `Machine` (133 places in all), `Item` (82), `Kinds` (68), `New` (45), `OutputCount` (42), `State` (34).
- The definition already has hook fields other packages fill from their `init`: `Click`, `Run`, `Insert`, `Peek`, `Extract`, `Accepts`, and `Lines` after 3-3. Nineteen assignment statements in eleven files of storage, dimension, farm and netstore fill them. Six fields a foreign owner would need are unexported: `nameKey`, `helpKeys`, `fluidSource`, `fluidSink`, `fluidOnly`, `autoSieve`.
- Kind knowledge that stays in `machine` beside the hooks: `Machine.Activate` (line 1557: sieve, drawer trim, tinker table, lab, crafter, tank, cell, net cable), `Machine.Tick` (1680: an early return for 12 ids), `runMachine` (1779: 16 clauses, hooks only in the default), `MachineLines` (10 clauses), and the kind sets `cubeMachines` (19 kinds), `frontMachines`, `AnimatedKinds`, the net-shape set, `TierLook`, `IsSieve`.
- `States`, `Permutations`, `ClientExtras` and `Properties` (lines 1287 to 1360) each branch over the same sets by hand. One description of a kind's client shape should feed all four; `blocks.golden` proves the result.
- `machine` imports `tinker` for one call (`tinker.ShowTable` in `Activate`) and `decor` for tool tips in three files; `decor` imports `tinker` for `RarityColour`. The first edge turns around when the tinker table's click is the tinker's; `RarityColour` then belongs in `common/slotdeco`. `TestFeatureSiblings` and `onlyImports` (the `decor` row) must shrink with it.
- Saves name a kind by its string: the block and the item are `skygear:<kind>`, the state is read back with `New(m.Kind)`, statistics are keyed by item. None of this may change.

The finding that shapes the part: item registration order is in the wire goldens. Moving one registration call (`hopper.Register()` from the root's `init` into the package's own) changed 698 entries of `items.golden`, 327 of `recipes.golden` and 243 of `creative.golden`. `machine` registers every kind in its own `init`, in `Order`, before its owners are initialised. A kind therefore cannot be registered by its owner's `init` without re-recording the client goldens, and those may not be re-recorded in this phase. Two ways out, the owner chooses:

1. The identity of a kind (name, id, place in `Order`, block and item registration, client shape) stays declared in `machine`; only behaviour goes home (click, run, item I/O, lines, window, the `Tick` and `Activate` exceptions as definition fields). No golden changes. `machine` still names the 16 kinds, but as data.
2. Kinds are registered by their owners, and the wire goldens are re-recorded once with a proof that only runtime ids moved (the dumps of both trees, entries compared by name). This is the liftable result; it needs the owner's word because it lifts the rule of this phase.

Order of work under way 1: (a) export the six fields and add the missing hooks (`Lines` with a state argument, a "no work" flag for the early return of `Tick`, a "no window" flag for `Activate`), no kind touched; (b) one owner per task, smallest first (farm, tinker, dimension, storage, netstore last), each task moving the kind's clauses out of `Activate`, `Tick`, `runMachine` and `MachineLines` into the owner's `init`; (c) the four client-shape functions from one description; (d) `RarityColour` down, the `tinker` and `decor` edges of `machine` removed. The look pin of 3-3 (`machine/<kind>` and `state/*` lines) holds the panel text through all of it; `machine_test.go`, `terminal_test.go`, `drawer_test.go`, `harvester_test.go` hold the behaviour.

Not part of the move, questions for the owner: the duplicate `conduit.RemoveVisual` on break and the cleanup that runs only on a player's break (audit D2).

Re-measure before planning: every reader of `Kinds[...]` by kind; which clauses of the four functions each foreign kind takes; the `Order` readers (the creative run, the codex, the statistics, `giveItems`).

## 3-5 Feature tests leave the root package

Goal: a feature's tests live in its package and run without the root package.

Measured:

- The root has 88 test files with 17,613 lines at HEAD (the review plan says 81); 92 after 3-3. Sixteen sub-packages have 47.
- The helpers: `testSession` is used by 65 files, `loadAround` 61, `spawnPlayer` 47, `withPlayer` 47, `tick` 31, `machinePos` 26, `placeTestMachine` 23; a dozen smaller ones (`chestWithBundle`, `plantAt`, `fillLab`, `runSeconds`, `hasLine`) are shared by two or three files each. `helpers_test.go` is 100 lines; `placeTestMachine` and `machinePos` are in `machine_test.go`.
- 15 files drive the player handler and 17 call the session's lifecycle methods; those need their feature installed.
- By the feature packages a file imports: 8 files import none, 16 one, 26 two, 18 three, 20 four or more. The files with one or two are the candidates; a file with four or more tests the composed game.
- Stay in the root because they test the wiring: the eight `pin_*_test.go` and the goldens, `layers_test.go`, `helpers_test.go`, the benchmarks (`bench_test.go`, `pin_bench_test.go`, `netgraph_bench_test.go`, `rest_perf_test.go`), `handler_dispatch_test.go`, `bridge_effects_test.go`, `pack_test.go`, `persist_test.go`, `vanilla_persist_test.go`, `migrate_retired_test.go`, `energy_test.go`, `creative_test.go`, `commands_test.go`, `decor_test.go`, `names_test.go`, `admingive_test.go`, `blockcounts_test.go`, the two `debug_*_test.go`. About 28 files.

Needs: `core/coretest` with a session over a synchronous world (registered, so `core.At` finds it: the panel providers of 3-3 depend on that), a player, ticks, and installing a named set of features once per test binary. Three things are process-wide and never reset: `core.Install`, the adapter list of `common/waila`, the followers of `look`. A feature's test binary installs what it needs in one `TestMain`. The block registry is finalised in an `init` of the root's helpers; `coretest` must do the same.

Order: after 3-4 (the code's final homes), `coretest` first, then one feature per task, leaf features first (elevator, handwork, copier, wand, fairy), `machine` and `netstore` last. A task moves files and changes call sites, never an assertion.

Risks: a test that reaches an unexported root identifier; a test that passes only because every feature is installed (found by running the moved package alone); one server join per process (`wire` stays alone); `TestPinLook` and its helpers stay in the root while single-feature panel tests may move.

Re-measure before planning: per root test file, the feature packages it imports and the root identifiers it uses, after 3-4.

## Left out unless the owner asks

The duplicate session map (`Game.sessions` beside `core`'s registry), `core/arena.go` moving to `dimension`, thinning `sgitem` (its feature-owned item types, with their creative runs in `catalog`, going to their features), locales per feature, an `admin` package for the operator tools, deriving `giveItems`, a per-connection state helper in `sghook`, lifting `core` to `platform/`. Moving `hopper.Register()` is not on this list any more: it cannot be done without re-recording the client goldens.

## Claims that do not hold at HEAD

- Audit, step 2: moving `hopper.Register()` "should have no effect". It changes the client goldens (item ids follow registration order).
- Audit D10 and 4.2: the creative list can be derived from `Category()`. 45 of 243 entries are in another tab than their category, 106 of 349 custom items are not listed, and the order interleaves owners.
- Audit 4.2: the panel interface defined by `look` with a context pointer. `hopper` cannot import `look`, and a pointer through an interface allocates; the interface is in `common/waila` and passes the panel by value.
- The earlier outline: machine lines move "together with 3-4's kind definition" (they move whole in 3-3); look providers as a descriptor field (they are methods and adapters); `decor` would pull `tinker` down (it does not: the interface is in `common/slotdeco`).
- Counts: 16 foreign kinds, not 17; 88 root test files, not 79 or 81; `look` imports 14 game packages, of which 10 are features or `hopper`.
- `content/skygear/README.md` and `docs/tech/extension.md` still list the creative list, the look lines and the commands as central lists; T74 corrects them.
