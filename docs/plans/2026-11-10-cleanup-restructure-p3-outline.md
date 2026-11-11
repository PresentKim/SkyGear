# Phase 3, parts 3-3 to 3-5: outline

This is an outline, not a machine plan. Parts 3-1 and 3-2 are planned in full in `2026-11-10-cleanup-restructure-p3a.md` to `-p3e.md` (tasks T32-T58, with the step table in `-p3-order.md`). The three parts below get their own machine plans once 3-2 is built, because their shape depends on how the kernel lands. Numbers are measured at 541f654 unless marked "audit".

## Where 3-2 leaves the tree

- `content/skygear/core` holds the hooks, the handlers and the `Feature` descriptor (name, entities, settings, item entry edit, `Setup`). Every feature has a `feature.go`; `content/skygear/features.go` is the one list.
- The root package still holds: `game.go` (platform glue: `Spawn`, `Relocate`, `MenuBack`, `OpenScreen`, `Game.Start`'s setup), `commands.go` (12 commands, 6 distinct, of screens, codex, farm and machine), `admin.go` (465 lines), `debug*.go` (1,100 lines), `names.go`, `hopper.go`, `blockcounts.go`, `state.go`, and 79 test files plus the pins.
- Central places a new feature still touches after 3-2: `catalog/creative.go`, `catalog/recipes.go`, `catalog/research.go` with `progress/essence.go`, `look/look.go`, `codex/codexcards.go`, `screens/help.go`, the locales, `layers_test.go`, the pack's index files. Recipes, research, codex cards, help and the pack stay central by decision (design data across features). 3-3 removes `creative.go` and `look.go` from this list.

## 3-3 Type switches become capabilities or derived lists

Goal: a layer above no longer switches over the concrete types of the features below it. Each switch is replaced by an interface the value implements (the way `common/itemio` already works) or by a list derived from the feature descriptors.

Verified facts:

- `look/look.go` (420 lines, imports 14 game packages): `Info` switches over 8 block types at lines 156-198 (`machine.Machine`, `storage.Drawer`, `block.Composter`, `conduit.Bundle`, `farm.SilkwormLeaves`, `farm.Barrel`, `hopper.FlowHopper`, `farm.MagicPlant`) plus the crops; `MachineLines` switches over machine definitions in 10 branches at 318-354. The two switches over vanilla containers (256, 283) know no feature and stay.
- `decor/decor.go` (81 lines): one switch, 9 cases over `sgitem` types. It is why feature item types live in `sgitem`.
- `catalog/creative.go` (151 lines, 75 references to feature symbols): a hand-written list. Its order is pinned: `content/skygear/wire/testdata/creative.golden` has an `order/items` entry, and re-recording is forbidden in this phase.
- `admin.go`: 41 references to `sgitem`; the give list is already partly derived from `world.Items()`.
- A vanilla block cannot get a method (composter, crops): it needs an adapter the owning feature registers, as `itemio.RegisterAdapter` does.

Order: (1) a golden of `look.Info` for every block type and machine kind, recorded from the current code (WAILA text is player-visible; `look_test.go` and `cropinfo_test.go` cover only part); (2) `decor` to an item method; (3) `look` blocks to a method or adapter per owner, one owner per task; (4) machine lines, together with 3-4's kind definition; (5) the creative list derived from the descriptors with an explicit order that reproduces today's list exactly; (6) commands and the admin give list through the descriptor.

The descriptor grows here, one field per list that is actually derived: items for the creative list, commands, look providers. Blocks and locales only if a task needs them.

Risks: the creative order (if it cannot be reproduced from declarations, the hand list stays and the part is dropped: say so, do not re-record); WAILA text drift; `decor` sits below `machine`, so the item method must not pull `tinker` down (`tinker.RarityColour` is used by `decor`).

Re-measure before planning: the cases of each switch; which `sgitem` types exist only because `decor` or `catalog` switch on them; whether `creative.golden` still pins order; what `look_test.go` covers per block type; the descriptor as built.

## 3-4 Machine kinds go home

Goal: `machine` keeps the frame (block, state, window, energy, fluid, tiers, monitor) and its own kinds; a kind that belongs to another feature is defined and registered by that feature.

Verified facts:

- `machine/machine.go` is 1,972 lines. `machine.Kinds` has 36 kinds and `machine.Order` 36 (the audit's 38 and 37 predate phase 1). 16 belong elsewhere: storage 3 (drawer controller, trim, interface), tinker 1 (tinker table), dimension 2 (collector, relic table), farm 1 (harvester), netstore 9.
- Other packages already fill hook fields of their kinds from their own `init` (9 assignments in 7 files of dimension, farm and netstore), but the definitions sit in one map literal and four places branch by kind: `Machine.Activate` (1557), `Machine.Tick` (1680), `runMachine` (1779) and `look.MachineLines`.
- `Machine.ClientExtras` (1324) repeats the branching of `Properties` (1345) and `Permutations` (1304): three functions must agree by hand. One description of a kind's client shape should feed all three; `blocks.golden` proves the result.
- `machine` imports `tinker` and `decor`; both edges turn around when the tinker table is registered by `tinker`. `TestFeatureSiblings` (T39) lists them and must shrink.
- `machine.Order` orders the creative list, the codex and the statistics: it becomes an explicit sort value per kind.

Order: (1) `machine.Register(kind, Def)` with the fields a foreign owner needs exported, no kind moved yet; (2) one owner per task, smallest first (farm, tinker, dimension, storage, netstore last); (3) the kind branches in `Activate`, `Tick` and `runMachine` become definition fields as their last user leaves; (4) `ClientExtras`, `Properties` and `Permutations` from one description.

Risks: block registration order (kinds are registered in loops of `machine`'s `init`; a moved kind registers at another time; `blocks.golden` ignores entry order, but the creative order and the root tests' registry finalization do not); `netstore` uses 17 symbols of `machine` in 81 places (audit: re-measure); kind names and block names must not change (saves and statistics keys); the duplicate `conduit.RemoveVisual` on break and the cleanup that only runs on a player's break (audit D2) are behaviour questions for the owner, not part of the move.

Re-measure before planning: kinds and owners; every reader of `machine.Kinds` and `machine.Order`; unexported `Def` fields; external uses of `machine.State`.

## 3-5 Feature tests leave the root package

Goal: a feature's tests live in its package and run without the root package, so the feature can be lifted out with its tests.

Verified facts:

- The root package has 79 test files (14,749 lines); 16 sub-packages have 42. `helpers_test.go` (102 lines) builds the session (`testSession`, `loadAround`, `spawnPlayer`, `withPlayer`, `tick`); `machine_test.go` and `monitor_test.go` add helpers other files use (`placeTestMachine`, `machinePos`, `runSeconds`).
- Nine root test files build the handler (12 call sites) and more call the session's tick methods; after 3-2 these go through hooks, so a moved test needs its feature installed.
- `core.Install` runs once per process, and `helpers_test.go` finalizes the block registry in `init`.

Needs: `core/coretest` (a session over a synchronous world, a player, ticks, and installing a named set of features once per test binary). Tests that span features (energy, persistence, migration, the pack, the pins, `wire`) stay in the root package: they test the composed game.

Order: after 3-3 and 3-4 (the code's final homes), `coretest` first, then one feature per task, leaf features first. Mechanical work; a task moves files and changes call sites, never an assertion.

Risks: a test that reaches an unexported root identifier; tests that pass only because every feature is installed; one server join per process (`wire` stays alone).

Re-measure before planning: per root test file, the feature packages it imports and the root identifiers it uses; which files can move unchanged.

## Left out unless the owner asks

Moving `hopper.Register()` out of the root's `init` (it draws block hashes in call order; the audit expects no effect, the wire goldens and the persistence tests would have to show it), the duplicate session map (`Game.sessions` beside `core`'s registry), `core/arena.go` and `core/refill.go` moving to their features, thinning `sgitem`, locales per feature, an `admin` package for the operator tools, a per-connection state helper in `sghook`, and lifting `core` to `platform/`. None is needed for the review plan's success checks.

## Audit claims that no longer hold

- Kind counts (38, 37, 17) are 36, 36, 16; root tests are 79 files, not 81; the root's non-test code is 2,763 lines in 18 files, not 3,610.
- `Emit(e *E)` with the event "on the stack": a pointer passed to a function value escapes to the heap. Events are passed by value.
- Subscribing in each feature's `init` and emitting at the end of the legacy method cannot place a step in the middle of a chain and makes order depend on imports. The plan subscribes through `core.Install` with numbered priorities and bridges whole methods.
- A static `Order()` test cannot record today's order, because today's code has no hooks: the pins use marks in the legacy code.
- The five priority bands do not fit today's order.
- A derived creative list was said to change the order, to be checked by eye: the order is pinned by a golden now and may not change.
- One `Start` hook inside the transaction: the crops load before the transaction is queued, so there are two hooks, `Load` and `Start`.
