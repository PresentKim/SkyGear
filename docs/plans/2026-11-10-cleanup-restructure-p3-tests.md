# Phase 3 part 3-5 (feature tests leave the root package): reference

The machine plans `2026-11-10-cleanup-restructure-p3l.md`, `-p3m.md` and `-p3n.md` (tasks T88 to T107) cite this file as REF. It holds what a task needs word for word: the measurements, the decisions, the support packages with their whole API, the import rules, the rules a move follows, and per task the files, the declarations that stay, the helpers that are copied and the code of every new file.

Everything here was measured at cbb7d9b (49e1fba plus three documentation commits; the test files of `content/skygear` are those of 49e1fba). The whole part was built once in a scratch worktree, one commit per task in id order. After each of T88 to T107: `go build ./...`, `go vet ./...` and `go test ./...` pass, `gofmt -l` is clean, the nine goldens of `content/skygear/testdata/pin/`, the four of `content/skygear/wire/testdata` and `content/skygear/dimension/testdata/popup.golden` are byte for byte unchanged, and the union of test names over all packages is the one of the base (644 names; T107 adds its two guards). The combined patch is `/tmp/claude-0/-home-user-SkyGear/a71fa8d5-6f20-5c3f-ba20-eeb27e13ad8a/scratchpad/proto-3-5.patch` (it applies to cbb7d9b), with one patch per task in `/tmp/claude-0/-home-user-SkyGear/a71fa8d5-6f20-5c3f-ba20-eeb27e13ad8a/scratchpad/proto-3-5-tasks/`. A patch is a reference for the exact result of a task, not something to apply blindly: where this file and the code a developer finds disagree, the developer stops and reports (`NEEDS_CONTEXT`); it does not choose.

No production file changes in this part. There is no seam (section 5).

## 1. Measurements

### 1.1 What a test binary of a feature package lacks

The root package's `init` calls `core.Install(Features...)` and `hopper.Register()`, and the root links every package of the game. A test binary of `content/skygear/<feature>` links only what that package and its tests import. Measured by moving each candidate and running its package alone:

| Missing in the package binary | What fails | Tests it keeps in the root |
|---|---|---|
| The subscribers of the features (nothing is installed) | a test that reaches its feature through `s.Handler(p)`, `s.Fast`, the world handler, or a machine kind's registered behaviour | none: the package sets its own feature up (section 2, decision 4) |
| `catalog`: the game's recipes (`catalog/recipes.go`, registered in its `init`) and the research tree (`progress.ByID` finds nothing) | `0 elevator recipes`, `no compact_cobblestone research`, a nil `*progress.Research`, an empty `CrafterCandidates` | `TestElevatorRecipes`, `TestCompactRecipesAndResearch`, `TestMagicSeedsAndRecipes`, `TestLabScreen`, the three research tests of `fairy_test.go`, three of `autoequip_test.go`; whole files: `advancement`, `barkfiber`, `project`, `recipeshape`, `research`, `researchessence`, `logrecipes` |
| The game's hopper: the package `hopper` hooks the vanilla hopper's transfer in its `init` (`block.HopperTransfer`) and the root's `hopper.go` registers the flow hoppers; the binaries of `farm`, `storage` and `machine` do not link `hopper` | a hopper moves nothing into or out of a block of the game | `TestHandsFreeLine`, `TestDrawerHopperAndDisplay`, `TestHoppersFeedComposterAndBarrel`, `TestHopperFeedsComposter`, `hopper_test.go` |
| An item another package registers | `TestHuntRecords`: the fairy dust a clear gives is `sgitem.FairyDust`, registered by `fairy/fairymagic.go`; `dimension` does not import `fairy`, so in `dimension`'s binary the item is not registered, and the dust is gone from the hunter's inventory after the second hunt (`0 fairy dust after the second clear, want still 4`); in the root, where `fairy` is linked, the test passes | `TestHuntRecords` |
| A second feature | `TestSieveTopsUpTheHand` and `TestAutofill`, `TestAutofillPlace` need `autofill` beside `machine` and `wand`; `TestExpansionPlane` reads `screens.Boards` | those four |
| The root's unexported identifiers | does not compile | `TestVeinDiagonalSetting` (`s.g.db`), `TestWatchdog` (`s.Relocate`), `TestGiveRelics`, `TestPreviewGlassBlock`, `TestGiveItems` (`giveItems`); whole files, section 1.3 |

What a package binary does not lack: the block registry (it is closed by `coretest`), the entity types of the features that are set up, the account settings of a feature that is set up (`coretest.Setup` registers them), `account` without a database (it answers from memory when `account.Init` was not called).

### 1.2 The helpers of the root's tests

`helpers_test.go` (100 lines) gives `testSession` (67 of the 95 files use it), `loadAround` (64), `spawnPlayer` (50), `withPlayer` (50), `tick` (29) and `shapeAs` (1); `machine_test.go` gives `machinePos` (26) and `placeTestMachine` (24). Of `testSession`'s result a feature test uses only what the embedded `core.Lifecycle` has (`s.Session`, `s.World`, `s.Handler`, `s.Join`, `s.Second`, `s.Fast` …). The root's own fields and methods are used by nine files besides the helpers: `s.g` (achievements, debug_factory, hammer, look_branches, pin_lifecycle), `s.state` (sapling, persist, pin_lifecycle), the root's methods `Relocate`, `Spawn`, `Close` (dimension, sapling, netflush, pin_lifecycle).

Other helpers are defined in one test file and used by another:

| Helper | Defined in | Also used by | In this part |
|---|---|---|---|
| `droppedItems` | `handwork_test.go` | `research_test.go` (stays) | T90: to the root's `helpers_test.go`; the moved file keeps it |
| `droppedKeys` (the same function) | `witchwater_test.go` | `hammer_test.go`, `handwork_test.go` | T90: the one call in `handwork_test.go` becomes `droppedItems`; T92: a copy goes with `hammer_test.go`; T98: it moves with `witchwater_test.go` |
| `chestWithBundle` | `copier_test.go` | `pin_look_test.go`, `pin_use_test.go` | T91: to `helpers_test.go`; the moved file keeps it |
| `cropPos`, `plantAt`, `stageOf`, `advance` | `crops_test.go` | `fairytime_test.go` | T95: copies go with `fairytime_test.go`; T98: they move with `crops_test.go` |
| `always`, `count` | `crook_test.go` | `silkworm_test.go` | T98: both files move to `farm` |
| `useBlock` | `drawer_test.go` | `terminal_test.go` | T96: a copy goes with `terminal_test.go`; T97: it moves with `drawer_test.go` |
| `runSteps` | `energy_test.go` | `machine_test.go` (`TestHandsFreeLine`, stays) | T99: to `helpers_test.go`; the moved file keeps it |
| `wandWall`, `countStone` | `wand_test.go` | `pin_use_test.go`, the tests that stay | T102: in both halves |
| `press` | `screens_test.go` | `lab_test.go` (`TestLabScreen`, stays) | stays in the root's `screens_test.go` |
| `rangeBoxAt` | `autoequip_test.go` | `look_branches_test.go` | T106: in both halves |
| `fillLab` | `lab_test.go` | `pin_look_test.go` | T106: in both halves (with `countIn`) |
| `runSeconds` | `monitor_test.go` | `rest_perf_test.go` | T105: to `helpers_test.go`; the moved file keeps it |
| `hasLine` | `codex_test.go` | `autoequip_test.go` (only the tests that stay) | untouched |
| `recipeFor`, `eyeAt`, `registerVanillaRecipes`, `packDir`, `packLangFile`, `openPersisted`, `benchSession`, the pin harness | files that stay | files that stay | untouched |

### 1.3 Every test file of the root package

95 files, 18,754 lines, 436 tests and benchmarks at cbb7d9b. "Set up" is what the target package's binary sets up (section 2.4).

| File | Lines | Tests | Outcome | Task | Reason, or what the package sets up |
|---|---|---|---|---|---|
| `achievements_test.go` | 50 | 2 | stays | | root identifier: `s.g.db`; the research of `catalog` |
| `admingive_test.go` | 70 | 2 | stays | | root identifiers: `giveItems`, `resolveGive`, `giveEnum` |
| `advancement_test.go` | 259 | 9 | stays | | content list: the advancement tree names items of every feature and the research of `catalog` |
| `animate_test.go` | 49 | 1 | moves to `machine/animate_test.go` | T105 | set up: machine (and netstore, as before) |
| `autoequip_test.go` | 345 | 11 | split: 8 to `machine/autoequip_test.go`; 3 stay | T106 | stay: `TestAutoCrafterMakesARecipe`: recipes of `catalog`; `TestAutoCrafterOnlyMakesResearchedRecipes`: recipes and research of `catalog`; `TestMachineStatusLines`: recipes of `catalog` (`CrafterCandidates`). Set up: machine (and netstore, as before) |
| `autotool_test.go` | 62 | 1 | moves to `tinker/autotool_test.go` | T92 | set up: tinker |
| `barkfiber_test.go` | 112 | 4 | stays | | content lists: `codex.Uses`, the recipes and the research of `catalog` |
| `barrel_test.go` | 451 | 9 | moves to `farm/barrel_test.go` | T98 | set up: farm |
| `bench_test.go` | 142 | 6 | stays | | benchmarks of the composed game (`benchSession`, the root's `session`) |
| `blockcounts_test.go` | 42 | 2 | stays | | root code: `CustomBlockCounts` |
| `bridge_effects_test.go` | 149 | 4 | stays | | the composed game: join, second and fast tick over the installed order (`TestBridge*`) |
| `codex_test.go` | 373 | 11 | stays | | `codex` is no feature: the cards over all features and the recipes |
| `commands_test.go` | 21 | 1 | stays | | the feature list: `core.Commands()` in list order |
| `compact_test.go` | 214 | 6 | split: 5 to `island/compact_test.go`; 1 stay | T93 | stay: `TestCompactRecipesAndResearch`: research of `catalog`. Set up: island |
| `composter_test.go` | 163 | 5 | split: 3 to `farm/composter_test.go`; 2 stay | T98 | stay: `TestHoppersFeedComposterAndBarrel`: the root's hopper; `TestHopperFeedsComposter`: the root's hopper. Set up: farm |
| `conduitload_test.go` | 65 | 2 | stays | | saves: `openPersisted` (`persist_test.go`) |
| `copier_test.go` | 217 | 5 | moves to `copier/copier_test.go` | T91 | set up: copier |
| `counter_test.go` | 108 | 2 | moves to `achievement/counter_test.go` | T94 | set up: achievement |
| `counterhooks_test.go` | 126 | 2 | stays | | guard: reads the source of every package |
| `creative_test.go` | 82 | 2 | stays | | the creative list of `catalog` over all features |
| `crook_test.go` | 161 | 4 | moves to `farm/crook_test.go` | T98 | set up: farm |
| `cropinfo_test.go` | 50 | 1 | moves to `farm/cropinfo_test.go` | T98 | set up: farm |
| `crops_test.go` | 163 | 6 | moves to `farm/crops_test.go` | T98 | set up: farm |
| `debug_factory_test.go` | 85 | 3 | stays | | root code: the operator commands (`debugDense`, `debugPlace`, `debugFloor`) |
| `debug_oreui_test.go` | 25 | 1 | stays | | root code: `oreResServers` |
| `decor_test.go` | 63 | 2 | moves to `decor/decor_test.go` | T94 | set up: nothing (no session) |
| `dimension_test.go` | 735 | 15 | split: 12 to `dimension/dimension_test.go`; 3 stay | T100 | stay: `TestHuntRecords`: the fairy dust item is registered by `fairy`, which `dimension` does not link; `TestWatchdog`: root identifier `s.Relocate`; `TestGiveRelics`: root identifier `giveItems`. Set up: dimension |
| `drawer_reload_test.go` | 52 | 1 | stays | | saves: `openPersisted` |
| `drawer_test.go` | 757 | 23 | split: 21 to `storage/drawer_test.go`; 2 stay | T97 | stay: `TestDrawerHopperAndDisplay`: the root's hopper; `TestDrawerStatesMatchRuntimeIDs`: asserts runtime ids. Set up: storage |
| `elevator_test.go` | 107 | 2 | split: 1 to `elevator/elevator_test.go`; 1 stay | T89 | stay: `TestElevatorRecipes`: recipes of `catalog`. Set up: elevator |
| `energy_test.go` | 646 | 17 | moves to `conduit/energy_test.go` | T99 | set up: conduit, network |
| `fairy_test.go` | 258 | 10 | split: 7 to `fairy/fairy_test.go`; 3 stay | T95 | stay: `TestGiftsComeFromDoneResearch`: research of `catalog`; `TestGiftsSkipProjectsAndResearchDoneOutOfOrder`: research of `catalog`; `TestDeeperResearchIsDeeper`: research of `catalog`. Set up: fairy |
| `fairyspell_test.go` | 143 | 5 | moves to `climate/fairyspell_test.go` | T101 | set up: climate |
| `fairytime_test.go` | 123 | 4 | moves to `fairy/fairytime_test.go` | T95 | set up: fairy |
| `hammer_test.go` | 276 | 9 | split: 8 to `tinker/hammer_test.go`; 1 stay | T92 | stay: `TestVeinDiagonalSetting`: root identifier `s.g.db`. Set up: tinker |
| `handler_dispatch_test.go` | 90 | 1 | stays | | guard: the handler's methods and the root's source |
| `handwork_test.go` | 217 | 6 | moves to `handwork/handwork_test.go` | T90 | set up: handwork |
| `harvester_test.go` | 103 | 3 | moves to `farm/harvester_test.go` | T98 | set up: farm |
| `helpers_test.go` | 100 | 0 | stays | | the root's helpers (they become wrappers in T88) |
| `holder_test.go` | 28 | 1 | moves to `storage/holder_test.go` | T97 | set up: storage |
| `hopper_test.go` | 178 | 7 | stays | | the root registers the hopper (`hopper.go`); the turn recipes are `catalog`'s; `hopper` is no feature |
| `island_test.go` | 95 | 4 | moves to `island/island_test.go` | T93 | set up: island |
| `kinds_test.go` | 50 | 1 | stays | | guard: `kindOwners` against the feature list |
| `lab_test.go` | 306 | 10 | split: 9 to `machine/lab_test.go`; 1 stay | T106 | stay: `TestLabScreen`: research of `catalog`. Set up: machine (and netstore, as before) |
| `layers_test.go` | 334 | 8 | stays | | guards; `TestLookExpandingWithTool` needs `expand`, `look`, `wand` and `tinker` |
| `leaves_test.go` | 66 | 3 | moves to `farm/leaves_test.go` | T98 | set up: farm |
| `logrecipes_test.go` | 60 | 1 | stays | | content list: the recipes of `catalog` |
| `look_branches_test.go` | 177 | 6 | stays | | root identifiers: `s.g.sessionOf`, `EntityTypes`; the panel over `machine` and `storage` |
| `look_test.go` | 265 | 8 | stays | | the panel over the blocks of four features (`farm`, `machine`, `storage`, `tinker`), whose adapters their `Setup` registers |
| `machine_test.go` | 756 | 28 | split: 27 to `machine/machine_test.go`; 1 stay | T106 | stay: `TestHandsFreeLine`: the root's hopper. Set up: machine (and netstore, as before) |
| `magic_test.go` | 181 | 4 | split: 3 to `farm/magic_test.go`; 1 stay | T98 | stay: `TestMagicSeedsAndRecipes`: recipes and research of `catalog`. Set up: farm |
| `magicsieve_test.go` | 178 | 5 | moves to `machine/magicsieve_test.go` | T105 | set up: machine (and netstore, as before) |
| `migrate_retired_test.go` | 828 | 6 | stays | | saves: the migration of a whole save through the database |
| `minetool_test.go` | 89 | 2 | stays | | one table over the blocks of six packages (`storage`, `progress`, `farm`, `hopper`, `machine`, `sgitem`) |
| `monitor_test.go` | 167 | 5 | moves to `machine/monitor_test.go` | T105 | set up: machine (and netstore, as before) |
| `names_test.go` | 112 | 4 | stays | | the pack's names (`packDir`) and `giveItems` |
| `naturegen_test.go` | 128 | 3 | moves to `machine/naturegen_test.go` | T105 | set up: machine (and netstore, as before) |
| `netflush_test.go` | 36 | 1 | stays | | root identifiers: `Game`, `session.Close` |
| `netgraph_bench_test.go` | 243 | 4 | stays | | benchmarks (`benchSession`) |
| `pack_test.go` | 674 | 12 | stays | | the resource pack against every registered block and item |
| `persist_test.go` | 157 | 2 | stays | | saves: `Game`, `session`, `savedState` |
| `pin_bench_test.go` | 166 | 9 | stays | | benchmarks of the pinned paths |
| `pin_blocks_test.go` | 317 | 1 | stays | | pin: `handler-blocks.golden` |
| `pin_creative_test.go` | 33 | 1 | stays | | pin: `creative.golden` |
| `pin_kinds_test.go` | 208 | 2 | stays | | pin: `kinds.golden` |
| `pin_lifecycle_test.go` | 648 | 2 | stays | | pin: `lifecycle.golden`, `settings.golden` |
| `pin_look_test.go` | 296 | 1 | stays | | pin: `look.golden` |
| `pin_order_test.go` | 301 | 5 | stays | | pin: `order.golden`, and the guards over the root's source |
| `pin_simple_test.go` | 200 | 1 | stays | | pin: `handler-simple.golden` |
| `pin_test.go` | 182 | 0 | stays | | the pin harness |
| `pin_use_test.go` | 723 | 1 | stays | | pin: `handler-use.golden` |
| `project_test.go` | 56 | 2 | stays | | content list: the projects of the research tree |
| `quickbar_test.go` | 70 | 1 | stays | | the resource pack's `ui/sg_machine.json` |
| `quickwins_test.go` | 95 | 3 | stays | | three tests of three features (`dimension`, `conduit`, `machine`) |
| `recipeshape_test.go` | 198 | 5 | stays | | content list: the recipes of `catalog` |
| `research_split_test.go` | 44 | 1 | moves to `screens/research_split_test.go` | T104 | set up: screens |
| `research_test.go` | 187 | 5 | stays | | content list: the research tree of `catalog` |
| `researchessence_test.go` | 151 | 4 | stays | | content list: the essence costs in the research tree |
| `researchtext_test.go` | 156 | 5 | moves to `screens/researchtext_test.go` | T104 | set up: screens |
| `rest_perf_test.go` | 42 | 2 | stays | | a performance bound of the composed second |
| `sapling_test.go` | 91 | 3 | stays | | root identifiers: `buildIsland`, `newState`, `s.state`, `s.Spawn` |
| `screens_test.go` | 231 | 6 | split: 2 to `screens/menu_test.go`; 2 to `expand/expand_test.go`; 2 stay | T103, T104 | stay: `TestGiveItems`: root identifier `giveItems`; `TestExpansionPlane`: `expand` and `screens` together (`screens.Boards`). Set up: expand, screens |
| `sidebar_test.go` | 117 | 4 | moves to `screens/sidebar_test.go` | T104 | set up: screens |
| `sieve_test.go` | 322 | 8 | split: 6 to `machine/sieve_test.go`; 2 stay | T106 | stay: `TestSieveTopsUpTheHand`: needs `autofill` beside `machine`; `TestSieveTakesCrushedBlocks`: `codex.SieveInputs`. Set up: machine (and netstore, as before) |
| `silkworm_test.go` | 254 | 6 | moves to `farm/silkworm_test.go` | T98 | set up: farm |
| `stickchest_test.go` | 94 | 2 | moves to `handwork/stickchest_test.go` | T90 | set up: handwork |
| `terminal_test.go` | 31 | 1 | moves to `netstore/terminalclick_test.go` | T96 | set up: storage, netstore (as before) |
| `tier_test.go` | 237 | 8 | moves to `machine/tier_test.go` | T105 | set up: machine (and netstore, as before) |
| `timed_test.go` | 95 | 3 | stays | | time-limited items across `storage`, `machine`, `progress` and `island`; an item of `catalog` |
| `tinker_test.go` | 114 | 3 | moves to `tinker/tinker_test.go` | T92 | set up: tinker |
| `vanilla_persist_test.go` | 127 | 2 | stays | | saves: `openPersisted` |
| `vanilla_test.go` | 25 | 0 | stays | | helper of `codex_test.go` (`registerVanillaRecipes`) |
| `wand_test.go` | 382 | 9 | split: 6 to `wand/wand_test.go`; 3 stay | T102 | stay: `TestAutofill`: needs `autofill` beside `wand`; `TestAutofillPlace`: needs `autofill` beside `wand`; `TestPreviewGlassBlock`: root identifier `giveItems`. Set up: wand |
| `windowtext_test.go` | 47 | 2 | stays | | the pack (`packDir`) |
| `witchwater_test.go` | 152 | 3 | moves to `farm/witchwater_test.go` | T98 | set up: farm |

### 1.4 Numbers

| | Before | After |
|---|---|---|
| Test files of the root package | 95 (18,754 lines) | 67 (10,576 lines): 52 that keep all their tests, 14 with the tests that stay, `testsupport_test.go` new |
| Tests and benchmarks in the root | 436 | 198 (196 that stay and the two guards of T107) |
| Files that leave the root whole | | 29 (120 tests) |
| Files that are split | | 14 (120 tests move, 26 stay) |
| Tests in the other packages of the game | 208 in 16 packages | 448 in 27 packages |
| Test names over all packages | 644 | 646 (the two guards) |
| Support packages | | `core/coretest` 145 lines, `machine/machinetest` 28 lines |
| Production seams | | none |

Files that keep all their tests, by reason (52): the pins, their harness and `pin_bench_test.go` 10; structure guards and the feature list 6 (`layers`, `kinds`, `handler_dispatch`, `counterhooks`, `commands`, `creative`); the helpers 1; benchmarks 3 (`bench`, `netgraph_bench`, `rest_perf`); unexported root identifiers 9 (`achievements`, `admingive`, `blockcounts`, `bridge_effects`, `debug_factory`, `debug_oreui`, `look_branches`, `netflush`, `sapling`); saves 5 (`persist`, `vanilla_persist`, `conduitload`, `drawer_reload`, `migrate_retired`); the pack 4 (`pack`, `quickbar`, `windowtext`, `names`); the content lists 9 (`advancement`, `barkfiber`, `codex`, `logrecipes`, `project`, `recipeshape`, `research`, `researchessence`, `vanilla`); three or more features, or the root's hopper 5 (`hopper`, `look`, `minetool`, `quickwins`, `timed`).

### 1.5 Where the outline does not hold

- The outline moves whole files. Measured: 14 files mix tests that move with one to three that cannot (section 1.1). With whole files only, 29 files and 120 tests would move, and `machine_test.go` (28 tests), `drawer_test.go` (23), `dimension_test.go` (15), `autoequip_test.go` (11), `lab_test.go` (10) and `fairy_test.go` (10) would stay for one to three tests each; `elevator`, `dimension`, `wand` and `expand` would get nothing. The plan splits those 14 files by top-level declaration (section 2.2).
- "The eleven files that import `catalog`, `codex` or `screens` stay; `screens`' tests need the whole list installed": `research_split_test.go`, `researchtext_test.go`, `sidebar_test.go` and two tests of `screens_test.go` pass in `screens` alone with nothing set up, because `screens` imports `catalog` and `codex` itself. They move (T104).
- `decor_test.go` "needs every item registered in the game's order": it passes alone in `decor`. It moves (T94).
- Targets by the package a file calls most, not by the highest feature it imports: `lab_test.go` and `machine_test.go` go to `machine` (38 and 111 references to `machine`, 4 and 1 to `farm`); `fairyspell_test.go` to `climate` (35 references, 3 to `machine`); `leaves_test.go` and `silkworm_test.go` to `farm` (the leaf code is `farm/leaves.go`; 36 references to `farm`, 2 to `fairy`); `counter_test.go` to `achievement`; `compact_test.go` to `island`; three tests of `screens_test.go` are `expand`'s.
- The outline's lists by feature held files that cannot move: `sapling_test.go` (the root's `buildIsland`, `newState`, `state`, `Spawn`), `minetool_test.go` (blocks of six packages in one table), `conduitload_test.go` and `drawer_reload_test.go` (`openPersisted`), `netflush_test.go` (`Game`, `session.Close`), `quickwins_test.go` (three tests of three features).
- "A test that drives the handler needs every feature whose step runs before it (the guards, `autofill`, `copier` before `wand`)": no test that moves needs another feature's step. The two that need `autofill` stay.
- New: `energy_test.go` needs the `network` feature (the energy, item and fluid step on `Fast`), which is neither its own feature nor a kind owner. `conduit`'s binary sets it up.
- New: a support package cannot close the block registry in its `init` (section 6, risk 1).
- Counts at cbb7d9b: `tick` is used by 29 files (33), `placeTestMachine` by 24 (23), `loadAround` by 64 (63), `spawnPlayer` and `withPlayer` by 50 (49); `.Handler(` is called in 17 files (15), the lifecycle's `Join` in 2 (12).

## 2. Decisions

1. **A moved test is in the external test package** `package <pkg>_test`. The files already name the feature's identifiers with the package qualifier, so no body changes; `machinetest` imports `machine`, so `machine`'s own package could not import it; and an external test package may import a package that imports the package under test (`farm_test` imports `fairy`, `machine_test` imports `storage`, `conduit`, `farm`), so no import cycle can arise. The existing in-package test files keep their package clause.
2. **A file is split by top-level declaration when a few of its tests cannot move.** The declarations of "Stays in the root" stay in the root file of the same name; everything else moves; a helper both halves use is in both, word for word. Nothing inside a function changes but the call sites of section 7.1.
3. **What stays in the root**: a test that needs `catalog`, `codex`, the root's hopper, a second feature set up, an item of a package its feature does not import, or a root identifier; a test that asserts a runtime id or a golden; the benchmarks; the guards. No moved test gets a blank import to make it pass.
4. **What a package sets up.** Every package that gets a test with a session has `setup_test.go` with `func init() { coretest.Setup(<own>.Feature) }`, also where no moved test fails without it: a test that drives the handler then takes the subscriber's path as in the game, not a fallback (measured: `TestPlateClicksWithAnythingInHand` passes without `conduit`'s subscriber through `Activate`). Beyond its own feature: `conduit` also sets up `network`; `netstore/keys_test.go` already sets up `storage` and `netstore`, `machine/netkinds_test.go` `netstore` (both since 3-4, changed in T88 to go through `coretest.Setup`). No package sets up three features. `decor` has no session and no setup.

   | Package | `coretest.Setup(…)` | Moved tests that fail without it |
   |---|---|---|
   | `elevator`, `handwork`, `copier`, `achievement`, `dimension`, `climate`, `wand`, `screens`, `machine` | own feature | none |
   | `tinker` | `tinker.Feature` | `TestHammerDropsCrushedBlock`, `TestDeathKeepsFood` |
   | `island` | `island.Feature` | `TestLostAndFound` |
   | `fairy` | `fairy.Feature` | `TestHittingTheFairyGivesTheGift` |
   | `storage` | `storage.Feature` | seven tests of `drawer_test.go` (the punches, the controller's clicks) |
   | `farm` | `farm.Feature` | `TestCrookBreaksLeavesInPlay`, `TestUsingCrops`, `TestPlantingRecordsCrop`, `TestHarvesterHarvestsGrownCrops`, `TestHarvesterPicksBloomingPlants` |
   | `conduit` | `conduit.Feature, network.Feature` | without `network`: `TestGeneratorOutOfFuelPushesStoredEnergy`, `TestPowerLine`, `TestItemConduitMovesItems` |
   | `expand` | `expand.Feature` | `TestExpansionMode` |
   | `netstore` | `storage.Feature, Feature` (in `keys_test.go`) | as before |

5. **Shared helpers are copied, not exported.** A helper a staying root file needs stays in the root: in the root file when that file remains, else in `helpers_test.go`. The moved file has its own copy. `coretest` holds only what `helpers_test.go` held.
6. **`coretest.Session` returns `core.Lifecycle`**, the type the root's `session` embeds, so every expression a moved test wrote on `s` compiles unchanged. It has no database: `account` works without one, and a test that needs one opens it (`counter_test.go` does).
7. **The root's helpers keep their names** and call the support packages, so no test that stays changes in T88.
8. **`tinker_test.go` gets its own `machinePos`** instead of `machinetest.Pos`: `machine` imports `tinker`, and the tinker's test binary should not link `machine` for one position.
9. **One package per task**, from the leaves up, `machine` last; `achievement` and `decor` share a task (one file each, no helper, no split), and `machine` takes two (T105 the files that move whole, T106 the four that are split: one task would be past the size a developer's task may have).

## 3. The support packages

### 3.1 `content/skygear/core/coretest`

| Name | Signature | What it does |
|---|---|---|
| `Setup` | `func Setup(fs ...*core.Feature)` | closes the block registry (`world.DefaultBlockRegistry.Finalize()`); for each feature not yet set up in this binary and not installed by `core.Install`: registers its `Settings` and calls its `Setup`. Runs no `Check`. Remembers the features, so a second mention is skipped. Called from a test file's `init` only: a hook takes no subscriber after its first event |
| `Session` | `func Session(t testing.TB) core.Lifecycle` | closes the block registry; a synchronous world whose entity registry has the default types, those of the installed features and those of the features `Setup` set up; a `core.Session` with save id 1, name `test`, host `host`, the clock `instance.NewClock(0, 1, 0, 20*time.Minute)`, `Rand` seeded `PCG(1, 2)`; registered for `core.Of`; the world's handler is `core.WorldHandler`. `t.Cleanup` closes the world, closes the session, unregisters it |
| `LoadAround` | `func LoadAround(s core.Lifecycle)` | loads the chunks within 2 of the origin |
| `SpawnPlayer` | `func SpawnPlayer(s core.Lifecycle) *world.EntityHandle` | a player `p`, Korean locale, at (0.5, 64, 2.5) |
| `WithPlayer` | `func WithPlayer(t testing.TB, s core.Lifecycle, h *world.EntityHandle, f func(p *player.Player, tx *world.Tx))` | runs `f` in a transaction; a panic in `f` fails the test |
| `Tick` | `func Tick(s core.Lifecycle, n int)` | `n` world ticks |
| `ShapeAs` | `func ShapeAs(s core.Lifecycle, key any) debug.Shape` | the debug shape the session keeps under `key`, or nil |

It imports `core`, `platform/account`, `platform/instance` and Dragonfly; no feature. In the root's binary `Setup` is never called and `Session` takes the entity types from `core.EntityTypes()`, which is what `EntityTypes()` of the root appends to the default ones.

### 3.2 `content/skygear/machine/machinetest`

| Name | Signature | What it does |
|---|---|---|
| `Pos` | `var Pos = cube.Pos{3, island.IslandY, 3}` | where the tests place their machine |
| `Place` | `func Place(s core.Lifecycle, pos cube.Pos, kind string) *machine.State` | `machine.New(kind)` with `Unlimited` set and, for a sieve, the string mesh; set at `pos`; returns the state |

It imports `core`, `island` and `machine`.

## 4. Imports of a moved test, and the guards

- The guards read non-test files only. `TestLayers`, `TestFeatureSiblings`, `TestCapabilityImports` and `TestFeaturesLiftable` use `build.Default.ImportDir(p, 0).Imports`, which leaves out `TestImports` and `XTestImports`; `TestMachineImportsNoFeature`, `TestRootSubscribesToNothing`, `TestHandlerMethodsOneWay` and `TestEveryCounterIsCounted` skip `_test.go`. So a test file may import a package its own package may not: a sibling without a `siblings` row, a package of a higher row of `layers`.
- `TestLayers` lists the direct sub-directories of `content/skygear` only. `core/coretest` and `machine/machinetest` are one level deeper, so they need no row in `layers`. `TestEveryCounterIsCounted` does read their string literals: neither package has a literal that ends in `_`.
- A moved test imports what its file imported in the root, plus `core/coretest`, `machine/machinetest` and `core` where the rules of section 7.1 put their names in. It never imports the root package, and in a package below `catalog` in `layers` never `catalog`, `codex` or `screens` (`TestPackageTestsStayBelow`, T107).
- `dimension`: its binary also runs `popup_wire_test.go`, which joins a server and compares item ids with `testdata/popup.golden`. A package that registers blocks or items and is new to that binary would change the ids. The moved `dimension_test.go` adds `coretest` and `machinetest` only; both register nothing and everything they import is linked there already (measured with `go list -deps -test ./content/skygear/dimension/`: the two support packages are the only new entries). No task adds another import to a test of `dimension`.
- `wire` and `dimension/popup_wire_test.go` are not touched: one server join per process.
- No test file outside `core` calls `core.Install` or a feature's `Setup` directly (`TestPackageTestsStayBelow`): `Setup` twice panics on the repeated subscriber name, and `coretest.Setup` is what remembers.

## 5. Production seams

None. No non-test file of an existing package changes in T88 to T106; T107 changes documents and adds one test file. The two support packages are new non-test files that only tests import (`TestSupportPackagesAreTestOnly`, T107): nothing of the server links them.

## 6. Risks met while prototyping

1. **Closing the block registry too early.** Go initialises packages in import-path order where no import orders them. `…/core/coretest` sorts before `…/farm`, `…/machine` and the rest, so a `Finalize` in `coretest`'s `init` runs before their blocks register and the binary panics. `coretest` has no `init`; `Setup` and `Session` close the registry. A package's `setup_test.go` calls `Setup` from its `init`, which runs after every imported package.
2. **Passing for another reason.** A test that drives the handler can pass without its feature's subscriber through a fallback. Decision 4: the own feature is always set up.
3. **`Setup` twice.** `netstore/keys_test.go` and `machine/netkinds_test.go` called `Feature.Setup()` directly; with a `setup_test.go` beside them a feature could be set up twice. T88 routes both through `coretest.Setup`.
4. **A hidden registration.** `TestHuntRecords` passes in the root only because `fairy` is linked. A test like that stays; no blank import (it would change `popup.golden`'s ids in `dimension`).
5. **`go test -count=2` is not a stability check here.** At the base already, `netstore`'s `TestTerminalDepositGoesToTask` (`craft hooks not set`) and `wire`'s `TestWire` fail on the second run in one process. Run a package several times as separate processes with `-count=1`.
6. **A test of two features in a file of one.** `TestExpansionPlane` passed in `expand` because the file imported `screens`; the guard of T107 found it. It stays in the root.
7. **A file name taken.** `netstore/terminal_test.go` exists; the root's `terminal_test.go` becomes `netstore/terminalclick_test.go`. No test name of a moved file exists in its target package (checked: no name is listed twice in one package).
8. **Unused copies.** A helper is copied only where the moved half calls it (`hasLine` is used by the three tests of `autoequip_test.go` that stay, so `machine/autoequip_test.go` has no copy).
9. **The time-dependent hunt tests** (`TestHuntClear`, `TestKitAndLevels`, `TestHuntFails`; `docs/TODO.md`) now run in `dimension`'s binary beside the server join. Eight separate runs passed.

## 7. How a file moves

### 7.1 The rules of a move (R1 to R9)

Applied to the moved file only, to whole identifiers that are not preceded by a dot:

| | In the root | In the package |
|---|---|---|
| R1 | `package skygear` | `package <pkg>_test` |
| R2 | `testSession(` | `coretest.Session(` |
| R3 | `loadAround(` | `coretest.LoadAround(` |
| R4 | `spawnPlayer(` | `coretest.SpawnPlayer(` |
| R5 | `withPlayer(` | `coretest.WithPlayer(` |
| R6 | `tick(` | `coretest.Tick(` |
| R7 | `shapeAs(` | `coretest.ShapeAs(` |
| R8 | `machinePos` and `placeTestMachine(` | `machinetest.Pos` and `machinetest.Place(` |
| R9 | the type `*session` (in a helper's parameters) | `core.Lifecycle` |

Nothing else inside a function changes, with the two exceptions the tasks name (T90: one call; T92: `machinePos` in `tinker_test.go` stays as it is). `runTick(` and `s.Tick` are not `tick(`: match the whole identifier.

Then the imports of the moved file: add `github.com/presentkim/skygear/content/skygear/core/coretest`, `…/machine/machinetest` and `…/core` where R2 to R9 put the name in and the file does not import it yet; delete the imports no line uses any more; `gofmt`. The table of each task lists the result against the root file's import list ("+" added, "−" removed; a path without a host is relative to `github.com/presentkim/skygear/content/skygear/`, `common/…` and `platform/…` to the module), so nothing is left to judge.

### 7.2 A whole file

`git mv content/skygear/<file> content/skygear/<pkg>/<file>` (or the other name the task gives), then R1 to R9 and the imports.

### 7.3 A split file

1. Copy the root file to `content/skygear/<pkg>/<file>`.
2. In the copy: delete every top-level declaration of the column "Stays in the root", each with the comment lines directly above it. Apply R1 to R9. Fix the imports.
3. In the root file: delete every top-level declaration that is neither in "Stays in the root" nor in "In both", each with its comment. Delete the imports no line uses any more. Nothing else changes in it.

A declaration keeps its text, its comment and its place among the others in both files.

### 7.4 A copied helper

"Copy of `x` from `<file>`" means: the declaration with its comment lines, appended at the end of the moved file after one blank line, in the order given, with R2 to R9 applied.

### 7.5 `setup_test.go`

```go
package <pkg>_test

import (
	"github.com/presentkim/skygear/content/skygear/core/coretest"
	"github.com/presentkim/skygear/content/skygear/<pkg>"
)

// This package's test binary links no feature list, so no feature is
// installed in it. The tests that reach a feature through the kernel's hooks
// have it set up here, once.
func init() { coretest.Setup(<pkg>.Feature) }
```

### 7.6 What a developer checks and reports

Before the first edit and after the last, for the root and for the target package (a package without test files prints nothing):

```sh
go test -list '.*' ./content/skygear/ > <scratch>/root.before
go test -list '.*' ./content/skygear/<pkg>/ > <scratch>/pkg.before
```

(If the session's hook refuses a bare `go test`, compile with `go test -c -o <scratch>/<name>.test <package>` and run `<scratch>/<name>.test -test.list '.*'` from inside the package's directory.) The names that left the root's list are exactly the names that joined the package's list, they are the names the task's section lists, and no name is in a list twice. The report gives the four counts.

Then: `go build ./... && go vet ./...`; `gofmt -l` on the files touched prints nothing; the target package alone, three times as separate processes: `go test -count=1 ./content/skygear/<pkg>/`; `go test -count=1 ./content/skygear/ -run 'TestPin|TestKind|TestLayers|TestFeature|TestCapability|TestEvery|TestRoot|TestMachine|TestHandlerMethodsOneWay|TestBridge|TestCreative'`; `go test -count=1 ./content/skygear/wire/ ./content/skygear/dimension/`; `git status --short content/skygear/testdata content/skygear/wire/testdata content/skygear/dimension/testdata` prints nothing; then the full suite through `superlite-cli run_tests --full --background` and `run_tests --wait`.

### 7.7 Rules for developers

1. The plan is decision-complete. The task text and this file say which file goes where, which declarations stay, which helper is copied and what each new file holds. Use those lists exactly. A test the lists do not name does not move; a test that fails in its package after the move is reported, not repaired, not given an import, not sent back by your own choice.
2. A task moves files, changes the call sites of section 7.1 and package clauses. It never changes an assertion, a test's name, a constant a test compares with, or a production file. If a file can only compile in its package with another change, stop and report (`NEEDS_CONTEXT`).
3. Every golden is frozen: the nine of `content/skygear/testdata/pin/`, the four of `content/skygear/wire/testdata/`, `content/skygear/dimension/testdata/popup.golden`. `SKYGEAR_GOLDEN=update` is forbidden in this part.
4. A package's tests do not call `core.Install` and do not call a feature's `Setup` themselves; they name features in `coretest.Setup`, in an `init`, in the one place this file gives.
5. Do not touch `content/skygear/wire`, `content/skygear/dimension/popup_wire_test.go`, any `testdata`, or a non-test file.
6. The prototype patch of your task is a reference for the exact result. Check it against the task text and this file (they win; report differences). If the root file you find differs from the one the patch expects, apply the rules of section 7 to the file you find and say so in the report.
7. Never commit, push, `git add`, stash or check out tracked files. Scratch files under the scratchpad's `<task id>/`, deleted before the report. Never `go clean -cache`.

Report (at most 230 words): status; the exact paths changed; the four counts of section 7.6 and whether the two sets of names are equal; goldens unchanged yes or no; the full suite's result; differences between the task text, this file and the patch; anything outside Owns; open questions. No code.

## 8. Order, and what may run side by side

The tasks in id order are one valid sequence, and the one the prototype ran. The edges that bind:

- T88 before everything.
- The root's `helpers_test.go` is edited by T88, T90, T91, T99 and T105: these run in that order.
- T92 and T95 copy helpers out of root files that T98 moves away (`witchwater_test.go`, `crops_test.go`): both before T98.
- T96 copies `useBlock` out of `drawer_test.go`, which T97 splits: T96 before T97.
- T103 and T104 both cut the root's `screens_test.go`: T103 before T104.
- T105 and T106 (`machine`) are the last moves, T105 first (it adds `machine/setup_test.go`); T107 after every move (its guard reads the moved files).

T89, T93, T94, T100, T101 and T102 share no file with any other task. They can run beside any task but T88 and T107, if each developer's checks can tell its own files from another's unfinished edit; in one checkout the coordinator runs one task at a time.

## 9. Code by task

### T88

Two new packages, and the root's helpers as wrappers over them. No test moves. The two test files that call a feature's `Setup` themselves go through `coretest.Setup`, so that a `setup_test.go` beside them cannot set a feature up twice.

`content/skygear/core/coretest/coretest.go`, the whole file:

```go
// Package coretest gives the tests of a game package what the root package's
// tests have: a running save on a synchronous world, a player in it, and the
// features the tests need, set up once in the test binary. A test binary of a
// package other than the root links no feature list, so nothing is installed
// in it (core.Install runs in the root package only): its tests name the
// features they need in Setup. Only test files import this package.
package coretest

import (
	"log/slog"
	"math/rand/v2"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/debug"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/platform/account"
	"github.com/presentkim/skygear/platform/instance"
	"golang.org/x/text/language"
)

var (
	mu sync.Mutex
	// setUp are the features Setup has set up in this test binary.
	setUp []*core.Feature
)

// Setup closes the block registry (every block is registered when a test
// file's init runs) and sets up each feature of fs that this test binary has
// not set up yet: it registers the feature's settings and runs its Setup, as
// core.Install does, but runs no Check, so a feature may be set up without
// the others. A feature core.Install installed (the root package's tests) or
// an earlier call set up is skipped, so naming a feature twice is safe.
//
// Call it from the init function of a test file, never from a test: a hook
// takes no subscriber after its first event.
func Setup(fs ...*core.Feature) {
	world.DefaultBlockRegistry.Finalize()
	mu.Lock()
	defer mu.Unlock()
	installed := core.Features()
	for _, f := range fs {
		if slices.Contains(setUp, f) || slices.Contains(installed, f) {
			continue
		}
		setUp = append(setUp, f)
		if len(f.Settings) > 0 {
			account.RegisterSetting(f.Settings...)
		}
		if f.Setup != nil {
			f.Setup()
		}
	}
}

// entityTypes are the default entity types, those of the installed features
// and those of the features Setup set up.
func entityTypes() []world.EntityType {
	types := append(entity.DefaultRegistry.Types(), core.EntityTypes()...)
	mu.Lock()
	defer mu.Unlock()
	for _, f := range setUp {
		types = append(types, f.Entities...)
	}
	return types
}

// Session returns the session of a new save (id 1, named "test", owned by the
// account "host") over a synchronous world, without an instance manager and
// without a database. core.Of finds the session, the world's handler is the
// kernel's, and the random source is seeded the same in every run. The world
// and the session are closed when the test ends.
//
// The result is the session as the instance sees it: its Session field is the
// core session, its methods are the calls the instance makes (Join, Leave,
// Second, Fast, NewDay, Autosave, Close) and Handler is a player's handler.
func Session(t testing.TB) core.Lifecycle {
	t.Helper()
	world.DefaultBlockRegistry.Finalize()
	w := world.Config{Synchronous: true, Entities: entity.DefaultRegistry.Config().New(entityTypes())}.New()
	t.Cleanup(func() { _ = w.Close() })
	c := core.New(core.Config{
		World: w, Clock: instance.NewClock(0, 1, 0, 20*time.Minute),
		SaveID: 1, SaveName: "test", HostID: "host", Log: slog.Default(),
	})
	c.Rand = rand.New(rand.NewPCG(1, 2))
	t.Cleanup(c.Close)
	core.Register(c)
	t.Cleanup(func() { core.Unregister(c) })
	w.Handle(&core.WorldHandler{S: c})
	return core.Lifecycle{Session: c}
}

// LoadAround loads the chunks around the origin of s's world.
func LoadAround(s core.Lifecycle) {
	s.World.Do(func(tx *world.Tx) {
		l := world.NewLoader(2, s.World, world.NopViewer{})
		l.Move(tx, mgl64.Vec3{})
		l.Load(tx, 50)
	})
}

// SpawnPlayer adds a player named "p" with the Korean locale to s's world, at
// (0.5, 64, 2.5), and returns its handle.
func SpawnPlayer(s core.Lifecycle) *world.EntityHandle {
	id := uuid.New()
	h := world.EntitySpawnOpts{ID: id}.New(player.Type, player.Config{Name: "p", UUID: id, Locale: language.Korean, Position: mgl64.Vec3{0.5, 64, 2.5}})
	s.World.Do(func(tx *world.Tx) { tx.AddEntity(h) })
	return h
}

// WithPlayer runs f with the player of h in a transaction. A panic in f fails
// the test (Dragonfly would only log it).
func WithPlayer(t testing.TB, s core.Lifecycle, h *world.EntityHandle, f func(p *player.Player, tx *world.Tx)) {
	t.Helper()
	finished := false
	s.World.Do(func(tx *world.Tx) {
		e, _ := h.Entity(tx)
		f(e.(*player.Player), tx)
		finished = true
	})
	if !finished {
		t.Fatal("transaction did not finish (panicked)")
	}
}

// Tick advances s's world by n ticks.
func Tick(s core.Lifecycle, n int) {
	for range n {
		s.World.AdvanceTick()
	}
}

// ShapeAs is the shape s keeps under key (nil if none).
func ShapeAs(s core.Lifecycle, key any) debug.Shape {
	sh, _ := s.Shape(key)
	return sh
}
```

`content/skygear/machine/machinetest/machinetest.go`, the whole file:

```go
// Package machinetest places machines for tests. Only test files import it;
// a test file of the machine package itself must be in the package
// machine_test to do so.
package machinetest

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/machine"
)

// Pos is where the tests place their machine: on the island's level, inside
// the chunks coretest.LoadAround loads.
var Pos = cube.Pos{3, island.IslandY, 3}

// Place places a machine of kind at pos in s's world and returns its state.
// The machine needs no energy, and a sieve has the string mesh fitted.
func Place(s core.Lifecycle, pos cube.Pos, kind string) *machine.State {
	m := machine.New(kind)
	m.State.Unlimited = true
	if machine.IsSieve(kind) {
		m.State.Mesh = "string"
	}
	s.World.Do(func(tx *world.Tx) { tx.SetBlock(pos, m, nil) })
	return m.State
}
```

`content/skygear/helpers_test.go`, the whole file:

```go
package skygear

import (
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/debug"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core/coretest"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/machine/machinetest"
	"github.com/presentkim/skygear/platform/db"
)

// The helpers below are the root package's names for the test support
// packages (core/coretest, machine/machinetest), over the root's session.

// testSession returns a session over a synchronous world, without an
// instance manager: coretest.Session with the game, a database and the save
// the database holds for it.
func testSession(t testing.TB) *session {
	t.Helper()
	d, err := db.Open(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	if _, err := d.TouchAccount("host", "", "host"); err != nil {
		t.Fatal(err)
	}
	save, err := d.CreateSave("host", 1, "test")
	if err != nil {
		t.Fatal(err)
	}
	l := coretest.Session(t)
	l.SaveID, l.SaveName = save.ID, save.Name
	g := &Game{log: slog.Default(), db: d}
	s := &session{Lifecycle: l, g: g}
	s.state = savedState.Of(l.Session)
	g.sessions.Store(l.World, s)
	return s
}

func init() { world.DefaultBlockRegistry.Finalize() }

func loadAround(s *session) { coretest.LoadAround(s.Lifecycle) }

func spawnPlayer(s *session) *world.EntityHandle { return coretest.SpawnPlayer(s.Lifecycle) }

// withPlayer runs f with the player in a transaction. A panic in f fails
// the test (Dragonfly would only log it).
func withPlayer(t testing.TB, s *session, h *world.EntityHandle, f func(p *player.Player, tx *world.Tx)) {
	t.Helper()
	coretest.WithPlayer(t, s.Lifecycle, h, f)
}

func tick(s *session, n int) { coretest.Tick(s.Lifecycle, n) }

// shapeAs is the shape s keeps under key (nil if none).
func shapeAs(s *session, key any) debug.Shape { return coretest.ShapeAs(s.Lifecycle, key) }

var machinePos = machinetest.Pos

// placeTestMachine places a machine and returns its state.
func placeTestMachine(s *session, pos cube.Pos, kind string) *machine.State {
	return machinetest.Place(s.Lifecycle, pos, kind)
}
```

`content/skygear/machine_test.go`:

```diff
diff --git a/content/skygear/machine_test.go b/content/skygear/machine_test.go
index 7ce6fd3..57c32c0 100644
--- a/content/skygear/machine_test.go
+++ b/content/skygear/machine_test.go
@@ -30,21 +30,6 @@ import (
 	"github.com/presentkim/skygear/content/skygear/storage"
 )
 
-var machinePos = cube.Pos{3, island.IslandY, 3}
-
-// placeTestMachine places a machine and returns its state.
-func placeTestMachine(s *session, pos cube.Pos, kind string) *machine.State {
-	m := machine.New(kind)
-	// These tests are about the work itself, not about energy (energy_test.go).
-	m.State.Unlimited = true
-	if machine.IsSieve(kind) {
-		// The tests sift with the string mesh fitted.
-		m.State.Mesh = "string"
-	}
-	s.World.Do(func(tx *world.Tx) { tx.SetBlock(pos, m, nil) })
-	return m.State
-}
-
 func TestMachineIDsDistinct(t *testing.T) {
 	seen := map[machine.ID]string{}
 	for kind, k := range machine.Kinds {
```

`content/skygear/machine/netkinds_test.go`:

```diff
diff --git a/content/skygear/machine/netkinds_test.go b/content/skygear/machine/netkinds_test.go
index d7f4d41..8393ccd 100644
--- a/content/skygear/machine/netkinds_test.go
+++ b/content/skygear/machine/netkinds_test.go
@@ -6,13 +6,14 @@ import (
 
 	"github.com/df-mc/dragonfly/server/block"
 	"github.com/df-mc/dragonfly/server/item"
+	"github.com/presentkim/skygear/content/skygear/core/coretest"
 	"github.com/presentkim/skygear/content/skygear/machine"
 	"github.com/presentkim/skygear/content/skygear/netstore"
 )
 
 // The network kinds get their behaviour from the netstore feature's Setup,
 // which no feature list runs in this package's tests.
-func init() { netstore.Feature.Setup() }
+func init() { coretest.Setup(netstore.Feature) }
 
 var netKinds = []string{"net_controller", "net_cable", "disk_drive", "net_terminal", "net_importer", "net_exporter", "net_external", "pattern_encoder", "net_autocrafter"}
 
```

`content/skygear/netstore/keys_test.go`:

```diff
diff --git a/content/skygear/netstore/keys_test.go b/content/skygear/netstore/keys_test.go
index 4ba41c9..3fa6c2a 100644
--- a/content/skygear/netstore/keys_test.go
+++ b/content/skygear/netstore/keys_test.go
@@ -12,6 +12,7 @@ import (
 	"github.com/df-mc/dragonfly/server/world"
 	"github.com/presentkim/skygear/common/storagenet"
 	"github.com/presentkim/skygear/content/skygear/core"
+	"github.com/presentkim/skygear/content/skygear/core/coretest"
 	"github.com/presentkim/skygear/content/skygear/storage"
 )
 
@@ -21,8 +22,7 @@ import (
 // (machine.Register).
 func init() {
 	world.DefaultBlockRegistry.Finalize()
-	storage.Feature.Setup()
-	Feature.Setup()
+	coretest.Setup(storage.Feature, Feature)
 }
 
 func TestKeyOfCount(t *testing.T) {
```

`machine_test.go` keeps its import list: its tests still use `cube`, `island`, `world` and `machine`. The line `world.DefaultBlockRegistry.Finalize()` in `keys_test.go` stays (it is harmless and the task changes only what it must).

After the task: `go test -list` prints the same 644 names (436 in the root, 32 in `machine`, 62 in `netstore`); 51 packages pass; `go vet ./...` is clean; `TestLayers` passes without a row for the two new directories (section 4).

### T89

`elevator`: the ride test moves, the recipe test stays (section 7.3).

| Root file | Goes to | Mode | Stays in the root | In both | Imports of the moved file | Imports the root file loses |
|---|---|---|---|---|---|---|
| `elevator_test.go` | `elevator/elevator_test.go` | split | `TestElevatorRecipes` |  | +`core` +`core/coretest` −`github.com/df-mc/dragonfly/server/item/recipe` | `github.com/df-mc/dragonfly/server/block/cube`, `github.com/df-mc/dragonfly/server/player`, `github.com/df-mc/dragonfly/server/world`, `github.com/go-gl/mathgl/mgl64`, `island` |

New file `content/skygear/elevator/setup_test.go`:

```go
package elevator_test

import (
	"github.com/presentkim/skygear/content/skygear/core/coretest"
	"github.com/presentkim/skygear/content/skygear/elevator"
)

// This package's test binary links no feature list, so no feature is
// installed in it. The tests that reach a feature through the kernel's hooks
// have it set up here, once.
func init() { coretest.Setup(elevator.Feature) }
```

Names that go from the root to `elevator` (1): `TestElevatorRides`.

Counts of `go test -list`: root 436 to 435; `elevator` 0 to 1.

### T90

`handwork`: both files move whole. `droppedItems` is also used by `research_test.go`, which stays, so the root gets it in `helpers_test.go`; the moved file keeps its own.

| Root file | Goes to | Mode | Stays in the root | In both | Imports of the moved file | Imports the root file loses |
|---|---|---|---|---|---|---|
| `handwork_test.go` | `handwork/handwork_test.go` | whole |  |  | +`core/coretest` |  |
| `stickchest_test.go` | `handwork/stickchest_test.go` | whole |  |  | +`core/coretest` |  |

In the moved `handwork/handwork_test.go`, `TestGrassSeeds` calls the twin of that helper, which stays behind in `witchwater_test.go`. The one line changes (the two helpers are the same function):

```diff
-		if n := droppedKeys(tx)[naming.ItemKey(sgitem.GrassSeeds{})]; n < 5 || n > 40 {
+		if n := droppedItems(tx)[naming.ItemKey(sgitem.GrassSeeds{})]; n < 5 || n > 40 {
```

New file `content/skygear/handwork/setup_test.go`:

```go
package handwork_test

import (
	"github.com/presentkim/skygear/content/skygear/core/coretest"
	"github.com/presentkim/skygear/content/skygear/handwork"
)

// This package's test binary links no feature list, so no feature is
// installed in it. The tests that reach a feature through the kernel's hooks
// have it set up here, once.
func init() { coretest.Setup(handwork.Feature) }
```

`content/skygear/helpers_test.go`:

```diff
diff --git a/content/skygear/helpers_test.go b/content/skygear/helpers_test.go
index a4e4f61..f6dd145 100644
--- a/content/skygear/helpers_test.go
+++ b/content/skygear/helpers_test.go
@@ -6,12 +6,14 @@ import (
 	"testing"
 
 	"github.com/df-mc/dragonfly/server/block/cube"
+	"github.com/df-mc/dragonfly/server/entity"
 	"github.com/df-mc/dragonfly/server/player"
 	"github.com/df-mc/dragonfly/server/player/debug"
 	"github.com/df-mc/dragonfly/server/world"
 	"github.com/presentkim/skygear/content/skygear/core/coretest"
 	"github.com/presentkim/skygear/content/skygear/machine"
 	"github.com/presentkim/skygear/content/skygear/machine/machinetest"
+	"github.com/presentkim/skygear/content/skygear/naming"
 	"github.com/presentkim/skygear/platform/db"
 )
 
@@ -68,3 +70,16 @@ var machinePos = machinetest.Pos
 func placeTestMachine(s *session, pos cube.Pos, kind string) *machine.State {
 	return machinetest.Place(s.Lifecycle, pos, kind)
 }
+
+// droppedItems counts the item entities in the world by item key.
+func droppedItems(tx *world.Tx) map[string]int {
+	out := map[string]int{}
+	for e := range tx.Entities() {
+		if it, ok := e.(*entity.Ent); ok {
+			if b, ok := it.Behaviour().(*entity.ItemBehaviour); ok {
+				out[naming.ItemKey(b.Item().Item())] += b.Item().Count()
+			}
+		}
+	}
+	return out
+}
```

Names that go from the root to `handwork` (8): `TestStickKnocksPebbles`, `TestAxeStripsFiber`, `TestTinkerAxeStripsFiber`, `TestDropsHaveMotion`, `TestNoFiberString`, `TestGrassSeeds`, `TestSneakingStickFillsChestWithWhatItHolds`, `TestThreeSneakingStickUsesSortChest`.

Counts of `go test -list`: root 435 to 427; `handwork` 0 to 8.

### T91

`copier`: the file moves whole. `chestWithBundle` is also used by `pin_look_test.go` and `pin_use_test.go`, so the root gets it in `helpers_test.go`; the moved file keeps its own.

| Root file | Goes to | Mode | Stays in the root | In both | Imports of the moved file | Imports the root file loses |
|---|---|---|---|---|---|---|
| `copier_test.go` | `copier/copier_test.go` | whole |  |  | +`core/coretest` +`machine/machinetest` |  |

New file `content/skygear/copier/setup_test.go`:

```go
package copier_test

import (
	"github.com/presentkim/skygear/content/skygear/copier"
	"github.com/presentkim/skygear/content/skygear/core/coretest"
)

// This package's test binary links no feature list, so no feature is
// installed in it. The tests that reach a feature through the kernel's hooks
// have it set up here, once.
func init() { coretest.Setup(copier.Feature) }
```

`content/skygear/helpers_test.go`:

```diff
diff --git a/content/skygear/helpers_test.go b/content/skygear/helpers_test.go
index f6dd145..36a9280 100644
--- a/content/skygear/helpers_test.go
+++ b/content/skygear/helpers_test.go
@@ -5,11 +5,14 @@ import (
 	"path/filepath"
 	"testing"
 
+	"github.com/df-mc/dragonfly/server/block"
 	"github.com/df-mc/dragonfly/server/block/cube"
 	"github.com/df-mc/dragonfly/server/entity"
 	"github.com/df-mc/dragonfly/server/player"
 	"github.com/df-mc/dragonfly/server/player/debug"
 	"github.com/df-mc/dragonfly/server/world"
+	"github.com/presentkim/skygear/common/itemnet"
+	"github.com/presentkim/skygear/content/skygear/conduit"
 	"github.com/presentkim/skygear/content/skygear/core/coretest"
 	"github.com/presentkim/skygear/content/skygear/machine"
 	"github.com/presentkim/skygear/content/skygear/machine/machinetest"
@@ -83,3 +86,16 @@ func droppedItems(tx *world.Tx) map[string]int {
 	}
 	return out
 }
+
+// chestWithBundle puts a chest at pos and an item conduit bundle east of it
+// whose west side is set.
+func chestWithBundle(tx *world.Tx, pos cube.Pos, mode itemnet.Mode, extract, insert int) cube.Pos {
+	tx.SetBlock(pos, block.NewChest(), nil)
+	b := conduit.Bundle{Item: 1}
+	b.Modes[cube.FaceWest] = mode
+	b.Extract[cube.FaceWest], b.Insert[cube.FaceWest] = extract, insert
+	at := pos.Add(cube.Pos{1, 0, 0})
+	tx.SetBlock(at, b, nil)
+	b.Refresh(at, tx)
+	return at
+}
```

Names that go from the root to `copier` (5): `TestCopierCardOnItem`, `TestCopierConduitSide`, `TestCopierWholeBundle`, `TestCopierCrafterAndDrawer`, `TestCopierHandler`.

Counts of `go test -list`: root 427 to 422; `copier` 0 to 5.

### T92

`tinker`: two files move whole, `hammer_test.go` is split.

| Root file | Goes to | Mode | Stays in the root | In both | Imports of the moved file | Imports the root file loses |
|---|---|---|---|---|---|---|
| `autotool_test.go` | `tinker/autotool_test.go` | whole |  |  | +`core/coretest` |  |
| `hammer_test.go` | `tinker/hammer_test.go` | split | `TestVeinDiagonalSetting` |  | +`core/coretest` −`log/slog` −`platform/account` −`platform/instance` | `core`, `github.com/df-mc/dragonfly/server/entity`, `github.com/df-mc/dragonfly/server/item`, `github.com/go-gl/mathgl/mgl64`, `naming`, `sgitem`, `sievetable` |
| `tinker_test.go` | `tinker/tinker_test.go` | whole |  |  | +`core/coretest` +`github.com/df-mc/dragonfly/server/block/cube` +`island` |  |

`tinker/tinker_test.go`: R8 is not applied to this file (decision 8). `machinePos` keeps its name and is declared in the file, directly after the import block, with the imports `github.com/df-mc/dragonfly/server/block/cube` and `github.com/presentkim/skygear/content/skygear/island`:

```go
// machinePos is where the tinker table of these tests stands.
var machinePos = cube.Pos{3, island.IslandY, 3}
```

`tinker/hammer_test.go`: copy of `droppedKeys` from the root's `witchwater_test.go` (section 7.4), used by `TestVeinMine` and `TestVeinMineFood`:

```go
// droppedKeys are the item keys of the item entities lying in tx.
func droppedKeys(tx *world.Tx) map[string]int {
	out := map[string]int{}
	for e := range tx.Entities() {
		if ent, ok := e.(*entity.Ent); ok {
			if ib, ok := ent.Behaviour().(*entity.ItemBehaviour); ok {
				out[naming.ItemKey(ib.Item().Item())] += ib.Item().Count()
			}
		}
	}
	return out
}
```

The root's `hammer_test.go` keeps `TestVeinDiagonalSetting` only; `witchwater_test.go` is not touched.

New file `content/skygear/tinker/setup_test.go`:

```go
package tinker_test

import (
	"github.com/presentkim/skygear/content/skygear/core/coretest"
	"github.com/presentkim/skygear/content/skygear/tinker"
)

// This package's test binary links no feature list, so no feature is
// installed in it. The tests that reach a feature through the kernel's hooks
// have it set up here, once.
func init() { coretest.Setup(tinker.Feature) }
```

Names that go from the root to `tinker` (12): `TestEquipToolFor`, `TestHammerDropsCrushedBlock`, `TestDustFromCrushedSand`, `TestCrushedStones`, `TestHammerOre`, `TestHammerTiers`, `TestVeinMine`, `TestVeinMineFood`, `TestDeathKeepsFood`, `TestToolStats`, `TestToolStackValues`, `TestTinkerTable`.

Counts of `go test -list`: root 422 to 410; `tinker` 0 to 12.

### T93

`island`: `island_test.go` moves whole, `compact_test.go` is split; the helper `compact` is in both halves, `levels` and `worth` go with the tests that use them.

| Root file | Goes to | Mode | Stays in the root | In both | Imports of the moved file | Imports the root file loses |
|---|---|---|---|---|---|---|
| `compact_test.go` | `island/compact_test.go` | split | `TestCompactRecipesAndResearch` | `compact` | +`core/coretest` −`progress` | `core`, `github.com/df-mc/dragonfly/server/item`, `github.com/df-mc/dragonfly/server/item/inventory`, `github.com/df-mc/dragonfly/server/player`, `island` |
| `island_test.go` | `island/island_test.go` | whole |  |  | +`core/coretest` |  |

New file `content/skygear/island/setup_test.go`:

```go
package island_test

import (
	"github.com/presentkim/skygear/content/skygear/core/coretest"
	"github.com/presentkim/skygear/content/skygear/island"
)

// This package's test binary links no feature list, so no feature is
// installed in it. The tests that reach a feature through the kernel's hooks
// have it set up here, once.
func init() { coretest.Setup(island.Feature) }
```

Names that go from the root to `island` (9): `TestPlanCompactPayment`, `TestPlanCompactPaymentAlwaysExact`, `TestPayTakesCompressedCobblestone`, `TestPayChangeThatDoesNotFit`, `TestBuyChunkWithCompressedCobblestone`, `TestChunkRules`, `TestLostAndFound`, `TestNearestBlockColumn`, `TestFormatInt`.

Counts of `go test -list`: root 410 to 401; `island` 0 to 9.

### T94

`achievement` and `decor`: one file each, whole. `decor` has no session and gets no `setup_test.go`. `counter_test.go` opens its own database (`account.Init`) as before.

| Root file | Goes to | Mode | Stays in the root | In both | Imports of the moved file | Imports the root file loses |
|---|---|---|---|---|---|---|
| `counter_test.go` | `achievement/counter_test.go` | whole |  |  | +`core/coretest` |  |
| `decor_test.go` | `decor/decor_test.go` | whole |  |  | unchanged |  |

New file `content/skygear/achievement/setup_test.go`:

```go
package achievement_test

import (
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/core/coretest"
)

// This package's test binary links no feature list, so no feature is
// installed in it. The tests that reach a feature through the kernel's hooks
// have it set up here, once.
func init() { coretest.Setup(achievement.Feature) }
```

Names that go from the root to `achievement` (2): `TestCounterCatalog`, `TestCountersCount`.

Names that go from the root to `decor` (2): `TestItemDecor`, `TestTipNameColour`.

Counts of `go test -list`: root 401 to 397; `achievement` 0 to 2; `decor` 0 to 2.

### T95

`fairy`: `fairytime_test.go` moves whole, `fairy_test.go` is split (the three tests over the research tree stay).

| Root file | Goes to | Mode | Stays in the root | In both | Imports of the moved file | Imports the root file loses |
|---|---|---|---|---|---|---|
| `fairy_test.go` | `fairy/fairy_test.go` | split | `TestGiftsComeFromDoneResearch`, `TestGiftsSkipProjectsAndResearchDoneOutOfOrder`, `TestDeeperResearchIsDeeper` |  | +`core/coretest` −`github.com/df-mc/dragonfly/server/item` −`progress` | `core`, `github.com/df-mc/dragonfly/server/block`, `github.com/df-mc/dragonfly/server/block/cube`, `github.com/df-mc/dragonfly/server/entity`, `github.com/df-mc/dragonfly/server/player`, `github.com/df-mc/dragonfly/server/world`, `naming`, `sgitem`, `time` |
| `fairytime_test.go` | `fairy/fairytime_test.go` | whole |  |  | +`core` +`core/coretest` +`island` |  |

`fairy/fairytime_test.go`: copies of `cropPos`, `plantAt`, `stageOf` and `advance` from the root's `crops_test.go` (section 7.4), which is not touched:

```go
var cropPos = cube.Pos{4, island.IslandY, 4}

// plantAt puts a crop on farmland and records it as planted now.
func plantAt(s core.Lifecycle, pos cube.Pos, b world.Block) {
	s.World.Do(func(tx *world.Tx) {
		tx.SetBlock(pos.Side(cube.FaceDown), block.Farmland{Hydration: 7}, nil)
		tx.SetBlock(pos, b, nil)
		kind, _, _ := farm.CropOf(b)
		farm.PlantCrop(s.Session, pos, kind)
	})
}

func stageOf(t *testing.T, s core.Lifecycle, pos cube.Pos) int {
	t.Helper()
	stage := -1
	s.World.Do(func(tx *world.Tx) {
		if _, g, ok := farm.CropOf(tx.Block(pos)); ok {
			stage = g
		}
	})
	return stage
}

// advance moves instance time on and runs the per-second update.
func advance(s core.Lifecycle, d time.Duration) {
	s.Clock.Advance(d)
	s.World.Do(func(tx *world.Tx) { farm.GrowCrops(s.Session, tx) })
}
```

New file `content/skygear/fairy/setup_test.go`:

```go
package fairy_test

import (
	"github.com/presentkim/skygear/content/skygear/core/coretest"
	"github.com/presentkim/skygear/content/skygear/fairy"
)

// This package's test binary links no feature list, so no feature is
// installed in it. The tests that reach a feature through the kernel's hooks
// have it set up here, once.
func init() { coretest.Setup(fairy.Feature) }
```

Names that go from the root to `fairy` (11): `TestGiftsAreBasics`, `TestFairyVisitsAndGives`, `TestHittingTheFairyGivesTheGift`, `TestFairyComesEveryNight`, `TestFairyGiftHasDust`, `TestFairyScattersTheGift`, `TestFairyDustBackfill`, `TestFairyBoneMealGrowsCropsAhead`, `TestFairyBoneMealBringsFruitSooner`, `TestFairyBoneMealOnPlantsAndSaplings`, `TestFairyHourglass`.

Counts of `go test -list`: root 397 to 386; `fairy` 0 to 11.

### T96

`netstore`: the one test of `terminal_test.go` moves. `netstore/terminal_test.go` exists, so the moved file is `netstore/terminalclick_test.go`. The package gets no `setup_test.go`: the `init` of `keys_test.go` sets up `storage` and `netstore` (T88).

| Root file | Goes to | Mode | Stays in the root | In both | Imports of the moved file | Imports the root file loses |
|---|---|---|---|---|---|---|
| `terminal_test.go` | `netstore/terminalclick_test.go` | whole |  |  | +`core` +`core/coretest` +`github.com/go-gl/mathgl/mgl64` +`machine/machinetest` |  |

`netstore/terminalclick_test.go`: copy of `useBlock` from the root's `drawer_test.go` (section 7.4), which is not touched:

```go
func useBlock(s core.Lifecycle, tx *world.Tx, p *player.Player, pos cube.Pos, held item.Stack, sneak bool) bool {
	p.SetHeldItems(held, item.Stack{})
	if sneak {
		p.StartSneaking()
	} else {
		p.StopSneaking()
	}
	ctx := player.NewEventContext(tx, p)
	s.Handler(p).HandleItemUseOnBlock(ctx, pos, cube.FaceUp, mgl64.Vec3{0.5, 1, 0.5})
	if ctx.Cancelled() {
		return true
	}
	held, _ = p.HeldItems()
	act, ok := tx.Block(pos).(block.Activatable)
	return ok && (!sneak || held.Empty()) && act.Activate(pos, cube.FaceUp, tx, p, &item.UseContext{})
}
```

Names that go from the root to `netstore` (1): `TestSneakingClickOpensTheTerminalOverEmptySlots`.

Counts of `go test -list`: root 386 to 385; `netstore` 62 to 63.

### T97

`storage`: `holder_test.go` moves whole, `drawer_test.go` is split. `useBlock` goes with the tests that use it (no test that stays calls it); `shapeAs` becomes `coretest.ShapeAs` in the moved half and stays `shapeAs` in `TestDrawerHopperAndDisplay`.

| Root file | Goes to | Mode | Stays in the root | In both | Imports of the moved file | Imports the root file loses |
|---|---|---|---|---|---|---|
| `drawer_test.go` | `storage/drawer_test.go` | split | `TestDrawerHopperAndDisplay`, `TestDrawerStatesMatchRuntimeIDs` |  | +`core` +`core/coretest` +`machine/machinetest` | `common/i18n`, `common/itemio`, `github.com/df-mc/dragonfly/server/entity`, `github.com/df-mc/dragonfly/server/player`, `github.com/go-gl/mathgl/mgl64`, `island`, `machine`, `naming`, `sgitem`, `slices`, `strings`, `textfmt` |
| `holder_test.go` | `storage/holder_test.go` | whole |  |  | unchanged |  |

New file `content/skygear/storage/setup_test.go`:

```go
package storage_test

import (
	"github.com/presentkim/skygear/content/skygear/core/coretest"
	"github.com/presentkim/skygear/content/skygear/storage"
)

// This package's test binary links no feature list, so no feature is
// installed in it. The tests that reach a feature through the kernel's hooks
// have it set up here, once.
func init() { coretest.Setup(storage.Feature) }
```

Names that go from the root to `storage` (22): `TestDrawerStorageRules`, `TestDrawerItemKeepsContents`, `TestUseOnBlockConsumesHeldItems`, `TestDrawerToolsUpgradesAndKey`, `TestDrawerUpgradeMarks`, `TestDrawerDoubleUseAndPunch`, `TestDrawerFrontPunchIsPerPlayer`, `TestFaceTextRotation`, `TestDrawerPunchFullInventoryDrops`, `TestLockedEmptyDrawer`, `TestDrawerMenuSlots`, `TestDrawerGivesThroughAnyFace`, `TestDrawerController`, `TestStorageSearchAndDeposit`, `TestControllerSneakClickOpensNoList`, `TestDrawerSneakMenuOnlyWithoutABlockHeld`, `TestControllerClickInsertsHeldItem`, `TestStorageOrders`, `TestControllerDoubleClickInsertsInventory`, `TestDrawerInterfaceActsForItsController`, `TestTwoControllersStopTheSet`, `TestHolderPoseRoundTrip`.

Counts of `go test -list`: root 385 to 363; `storage` 2 to 24.

### T98

`farm`: eight files move whole, `composter_test.go` and `magic_test.go` are split. `always` and `count` of `crook_test.go` and their user `silkworm_test.go` arrive together; the crop helpers and `droppedKeys` move with `crops_test.go` and `witchwater_test.go` (their other users took copies in T92 and T95). `farm/leaves_test.go` names nothing of `farm` and does not import it: the package under test is linked into its own test binary.

| Root file | Goes to | Mode | Stays in the root | In both | Imports of the moved file | Imports the root file loses |
|---|---|---|---|---|---|---|
| `barrel_test.go` | `farm/barrel_test.go` | whole |  |  | +`core/coretest` +`machine/machinetest` |  |
| `composter_test.go` | `farm/composter_test.go` | split | `TestHoppersFeedComposterAndBarrel`, `TestHopperFeedsComposter` |  | +`core/coretest` | `github.com/df-mc/dragonfly/server/player` |
| `crook_test.go` | `farm/crook_test.go` | whole |  |  | +`core/coretest` |  |
| `cropinfo_test.go` | `farm/cropinfo_test.go` | whole |  |  | +`core/coretest` |  |
| `crops_test.go` | `farm/crops_test.go` | whole |  |  | +`core` +`core/coretest` |  |
| `harvester_test.go` | `farm/harvester_test.go` | whole |  |  | +`core/coretest` +`machine/machinetest` |  |
| `leaves_test.go` | `farm/leaves_test.go` | whole |  |  | +`core/coretest` |  |
| `magic_test.go` | `farm/magic_test.go` | split | `TestMagicSeedsAndRecipes` |  | +`core/coretest` +`machine/machinetest` | `advancement`, `github.com/df-mc/dragonfly/server/block`, `github.com/df-mc/dragonfly/server/block/cube`, `github.com/df-mc/dragonfly/server/item`, `github.com/df-mc/dragonfly/server/player`, `github.com/df-mc/dragonfly/server/world`, `github.com/go-gl/mathgl/mgl64`, `look`, `machine`, `sgitem`, `sievetable` |
| `silkworm_test.go` | `farm/silkworm_test.go` | whole |  |  | +`core/coretest` |  |
| `witchwater_test.go` | `farm/witchwater_test.go` | whole |  |  | +`core/coretest` |  |

New file `content/skygear/farm/setup_test.go`:

```go
package farm_test

import (
	"github.com/presentkim/skygear/content/skygear/core/coretest"
	"github.com/presentkim/skygear/content/skygear/farm"
)

// This package's test binary links no feature list, so no feature is
// installed in it. The tests that reach a feature through the kernel's hooks
// have it set up here, once.
func init() { coretest.Setup(farm.Feature) }
```

Names that go from the root to `farm` (41): `TestBarrelStages`, `TestBarrelMakesWater`, `TestBarrelRecordIsForgotten`, `TestStoneBarrelMeltsLikeACrucible`, `TestStoneBarrelOldSave`, `TestDustMakesClayInWater`, `TestEmptyBarrelTakesItsLiquidBack`, `TestBarrelClayByHopper`, `TestWoodenBucket`, `TestComposterLevels`, `TestComposterFillsToDirt`, `TestBarkFiberIsPlantMatter`, `TestCrookLeafDrops`, `TestCrookIsQuickOnLeaves`, `TestCrookBreaksLeavesInPlay`, `TestCookedSilkworm`, `TestUsingCrops`, `TestCropsGrowOnInstanceTime`, `TestRandomTicksDoNotGrowCrops`, `TestBoneMealAndRemovedCrops`, `TestStemGrowsFruit`, `TestCropsLoadFromState`, `TestPlantingRecordsCrop`, `TestHarvesterHarvestsGrownCrops`, `TestHarvestedCropsAreYoung`, `TestReplantUsesSeeds`, `TestLeavesDropSaplingsOften`, `TestLeavesDropNoSticks`, `TestLeavesDecayOnScheduledTick`, `TestMagicPlantLifecycle`, `TestHarvesterPicksBloomingPlants`, `TestNoEssenceBloomRecipes`, `TestCocoonDropsString`, `TestCocoonGrowsOnTime`, `TestSilkwormsSpreadInPlay`, `TestSpreadStaysOnTheIsland`, `TestSilkwormLeavesDecayWithTheTree`, `TestOldInfestedLeavesUpgrade`, `TestWitchWater`, `TestMyceliumStaysUnderBarrel`, `TestBarrelLiquidIcon`.

Counts of `go test -list`: root 363 to 322; `farm` 3 to 44.

### T99

`conduit`: the file moves whole. `runSteps` is also used by `TestHandsFreeLine` of `machine_test.go`, which stays, so the root gets it in `helpers_test.go`; the moved file keeps its own. The package sets up `network` beside its own feature (decision 4).

| Root file | Goes to | Mode | Stays in the root | In both | Imports of the moved file | Imports the root file loses |
|---|---|---|---|---|---|---|
| `energy_test.go` | `conduit/energy_test.go` | whole |  |  | +`core` +`core/coretest` +`machine/machinetest` |  |

New file `content/skygear/conduit/setup_test.go`:

```go
package conduit_test

import (
	"github.com/presentkim/skygear/content/skygear/conduit"
	"github.com/presentkim/skygear/content/skygear/core/coretest"
	"github.com/presentkim/skygear/content/skygear/network"
)

// This package's test binary links no feature list, so no feature is
// installed in it. The tests that reach a feature through the kernel's hooks
// have it set up here, once. The network feature runs the energy, item and
// fluid step the conduit tests wait for.
func init() { coretest.Setup(conduit.Feature, network.Feature) }
```

`content/skygear/helpers_test.go`:

```diff
diff --git a/content/skygear/helpers_test.go b/content/skygear/helpers_test.go
index 36a9280..dd16206 100644
--- a/content/skygear/helpers_test.go
+++ b/content/skygear/helpers_test.go
@@ -99,3 +99,12 @@ func chestWithBundle(tx *world.Tx, pos cube.Pos, mode itemnet.Mode, extract, ins
 	b.Refresh(at, tx)
 	return at
 }
+
+// run advances n steps of four ticks, calling Fast (which moves the energy)
+// after each, as the instance does.
+func runSteps(s *session, n int) {
+	for range n {
+		tick(s, 4)
+		s.World.Do(func(tx *world.Tx) { s.Fast(tx) })
+	}
+}
```

Names that go from the root to `conduit` (17): `TestGeneratorBurnsFuelIntoEnergy`, `TestGeneratorOutOfFuelPushesStoredEnergy`, `TestPowerLine`, `TestConduitsOnlyJoinEnergyBlocks`, `TestConduitLookCodes`, `TestItemConduitMovesItems`, `TestConduitsShareABundle`, `TestConduitVisualFollowsTheBundle`, `TestConduitPlacementWays`, `TestConduitPlates`, `TestExtractAndInsertFlags`, `TestPlateFormsOpen`, `TestConduitBlockSizes`, `TestPlateClicksWithAnythingInHand`, `TestChannelItems`, `TestNextPlateSide`, `TestChestOpensUnderConduit`.

Counts of `go test -list`: root 322 to 305; `conduit` 14 to 31.

### T100

`dimension`: the file is split. `clearHunt` stays with `TestHuntRecords`, its only caller; `runTick` and `runMonsters` are in both halves; `withHeldPlayer` goes with the tests that use it. No import is added to the moved file beyond `coretest` and `machinetest` (section 4: `popup.golden`).

| Root file | Goes to | Mode | Stays in the root | In both | Imports of the moved file | Imports the root file loses |
|---|---|---|---|---|---|---|
| `dimension_test.go` | `dimension/dimension_test.go` | split | `clearHunt`, `TestHuntRecords`, `TestWatchdog`, `TestGiveRelics` | `runTick`, `runMonsters` | +`core/coretest` +`machine/machinetest` −`fmt` −`island` | `github.com/df-mc/dragonfly/server/block`, `github.com/df-mc/dragonfly/server/block/cube`, `github.com/df-mc/dragonfly/server/entity`, `github.com/go-gl/mathgl/mgl64`, `machine`, `math`, `progress`, `time` |

New file `content/skygear/dimension/setup_test.go`:

```go
package dimension_test

import (
	"github.com/presentkim/skygear/content/skygear/core/coretest"
	"github.com/presentkim/skygear/content/skygear/dimension"
)

// This package's test binary links no feature list, so no feature is
// installed in it. The tests that reach a feature through the kernel's hooks
// have it set up here, once.
func init() { coretest.Setup(dimension.Feature) }
```

Names that go from the root to `dimension` (12): `TestHuntClear`, `TestHuntFails`, `TestBladesHit`, `TestVariants`, `TestWatch`, `TestCollectors`, `TestRelicTable`, `TestKitAndLevels`, `TestKitMoves`, `TestGemsMagnet`, `TestDescribe`, `TestOldKit`.

Counts of `go test -list`: root 305 to 293; `dimension` 2 to 14.

### T101

`climate`: the file moves whole.

| Root file | Goes to | Mode | Stays in the root | In both | Imports of the moved file | Imports the root file loses |
|---|---|---|---|---|---|---|
| `fairyspell_test.go` | `climate/fairyspell_test.go` | whole |  |  | +`core` +`core/coretest` +`machine/machinetest` |  |

New file `content/skygear/climate/setup_test.go`:

```go
package climate_test

import (
	"github.com/presentkim/skygear/content/skygear/climate"
	"github.com/presentkim/skygear/content/skygear/core/coretest"
)

// This package's test binary links no feature list, so no feature is
// installed in it. The tests that reach a feature through the kernel's hooks
// have it set up here, once.
func init() { coretest.Setup(climate.Feature) }
```

Names that go from the root to `climate` (5): `TestRainGrowsFaster`, `TestWeatherSpells`, `TestSnowRunsMachinesFaster`, `TestSunSpells`, `TestRestoreBiome`.

Counts of `go test -list`: root 293 to 288; `climate` 0 to 5.

### T102

`wand`: the file is split. `wandWall` and `countStone` are in both halves (the tests that stay and `pin_use_test.go` use them); `sentViews` goes with the preview tests.

| Root file | Goes to | Mode | Stays in the root | In both | Imports of the moved file | Imports the root file loses |
|---|---|---|---|---|---|---|
| `wand_test.go` | `wand/wand_test.go` | split | `TestAutofill`, `TestAutofillPlace`, `TestPreviewGlassBlock` | `wandWall`, `countStone` | +`core/coretest` −`autofill` −`core` −`platform/account` −`strings` | `github.com/go-gl/mathgl/mgl64`, `machine`, `tinker` |

New file `content/skygear/wand/setup_test.go`:

```go
package wand_test

import (
	"github.com/presentkim/skygear/content/skygear/core/coretest"
	"github.com/presentkim/skygear/content/skygear/wand"
)

// This package's test binary links no feature list, so no feature is
// installed in it. The tests that reach a feature through the kernel's hooks
// have it set up here, once.
func init() { coretest.Setup(wand.Feature) }
```

Names that go from the root to `wand` (6): `TestWandBuildsFace`, `TestWandLock`, `TestWandPlaceable`, `TestWandPreview`, `TestPreviewOverlap`, `TestWandNextOption`.

Counts of `go test -list`: root 288 to 282; `wand` 0 to 6.

### T103

`expand`: two tests of the root's `screens_test.go` are the expansion's. They go to a new file `expand/expand_test.go`; everything else stays in `screens_test.go` for T104.

| Root file | Goes to | Mode | Stays in the root | In both | Imports of the moved file | Imports the root file loses |
|---|---|---|---|---|---|---|
| `screens_test.go` | `expand/expand_test.go` | split | `press`, `TestMainMenuHasIcons`, `TestLostMenu`, `TestGiveItems`, `TestExpansionPlane` |  | +`core/coretest` −`common/hud` −`common/i18n` −`common/ui` −`machine` −`screens` −`strings` | `github.com/df-mc/dragonfly/server/block`, `github.com/df-mc/dragonfly/server/block/cube`, `github.com/go-gl/mathgl/mgl64` |

New file `content/skygear/expand/setup_test.go`:

```go
package expand_test

import (
	"github.com/presentkim/skygear/content/skygear/core/coretest"
	"github.com/presentkim/skygear/content/skygear/expand"
)

// This package's test binary links no feature list, so no feature is
// installed in it. The tests that reach a feature through the kernel's hooks
// have it set up here, once.
func init() { coretest.Setup(expand.Feature) }
```

Names that go from the root to `expand` (2): `TestExpansionMode`, `TestExpansionMap`.

Counts of `go test -list`: root 282 to 280; `expand` 0 to 2.

### T104

`screens`: three files move whole; the two menu tests of what T103 left in `screens_test.go` go to `screens/menu_test.go`. The root's `screens_test.go` keeps `press` (used by `TestLabScreen`), `TestGiveItems` and `TestExpansionPlane`. `screens` imports `catalog` and `codex`, so its binary has the content lists.

| Root file | Goes to | Mode | Stays in the root | In both | Imports of the moved file | Imports the root file loses |
|---|---|---|---|---|---|---|
| `research_split_test.go` | `screens/research_split_test.go` | whole |  |  | +`core/coretest` |  |
| `researchtext_test.go` | `screens/researchtext_test.go` | whole |  |  | +`core/coretest` |  |
| `screens_test.go` | `screens/menu_test.go` | split | `press`, `TestGiveItems`, `TestExpansionPlane` |  | +`core/coretest` −`common/hud` −`expand` −`machine` | `github.com/df-mc/dragonfly/server/item` |
| `sidebar_test.go` | `screens/sidebar_test.go` | whole |  |  | +`core/coretest` |  |

New file `content/skygear/screens/setup_test.go`:

```go
package screens_test

import (
	"github.com/presentkim/skygear/content/skygear/core/coretest"
	"github.com/presentkim/skygear/content/skygear/screens"
)

// This package's test binary links no feature list, so no feature is
// installed in it. The tests that reach a feature through the kernel's hooks
// have it set up here, once.
func init() { coretest.Setup(screens.Feature) }
```

Names that go from the root to `screens` (12): `TestResearchCopiesFillTheTabs`, `TestEveryResearchIsExplained`, `TestResearchListHidesLockedResearches`, `TestResearchPreviewButton`, `TestResearchListSections`, `TestResearchScreen`, `TestMainMenuHasIcons`, `TestLostMenu`, `TestCostRowsCountTheInventory`, `TestHUDQuestFollowsAdvancements`, `TestQuestShowsResearchStep`, `TestQuestPanelSetting`.

Counts of `go test -list`: root 280 to 268; `screens` 1 to 13.

### T105

`machine`, the five files that move whole. `runSeconds` of `monitor_test.go` is also used by `rest_perf_test.go`, so the root gets it in `helpers_test.go`; the moved file keeps its own. Every file is in the package `machine_test`. `machine/netkinds_test.go` is not touched; its `init` and the new `setup_test.go` both go through `coretest.Setup`.

| Root file | Goes to | Mode | Stays in the root | In both | Imports of the moved file | Imports the root file loses |
|---|---|---|---|---|---|---|
| `animate_test.go` | `machine/animate_test.go` | whole |  |  | +`core/coretest` +`machine/machinetest` |  |
| `magicsieve_test.go` | `machine/magicsieve_test.go` | whole |  |  | +`core/coretest` +`machine/machinetest` |  |
| `monitor_test.go` | `machine/monitor_test.go` | whole |  |  | +`core` +`core/coretest` +`machine/machinetest` |  |
| `naturegen_test.go` | `machine/naturegen_test.go` | whole |  |  | +`core/coretest` +`machine/machinetest` |  |
| `tier_test.go` | `machine/tier_test.go` | whole |  |  | +`core/coretest` +`machine/machinetest` |  |

New file `content/skygear/machine/setup_test.go`:

```go
package machine_test

import (
	"github.com/presentkim/skygear/content/skygear/core/coretest"
	"github.com/presentkim/skygear/content/skygear/machine"
)

// This package's test binary links no feature list, so no feature is
// installed in it. The tests that reach a feature through the kernel's hooks
// have it set up here, once.
func init() { coretest.Setup(machine.Feature) }
```

`content/skygear/helpers_test.go`:

```diff
diff --git a/content/skygear/helpers_test.go b/content/skygear/helpers_test.go
index dd16206..da343ba 100644
--- a/content/skygear/helpers_test.go
+++ b/content/skygear/helpers_test.go
@@ -108,3 +108,10 @@ func runSteps(s *session, n int) {
 		s.World.Do(func(tx *world.Tx) { s.Fast(tx) })
 	}
 }
+
+// runSeconds runs the monitor's second n times.
+func runSeconds(s *session, n int) {
+	for range n {
+		s.World.Do(func(tx *world.Tx) { machine.TickMonitor(s.Session, tx) })
+	}
+}
```

Names that go from the root to `machine` (22): `TestActiveLook`, `TestMagicThemesAreComplete`, `TestThemeChance`, `TestMagicSieveCharging`, `TestMagicThemeReplaces`, `TestMagicSieveWorks`, `TestProductionRecorded`, `TestMinuteWindows`, `TestStallAlert`, `TestStarvedIsQuiet`, `TestStatsScreens`, `TestDaylight`, `TestSolarGenerator`, `TestHydroGenerator`, `TestTierNumbers`, `TestTierSpeedsUpWork`, `TestTierEnergy`, `TestUseKit`, `TestTierSaved`, `TestTierRanges`, `TestTierAlloyRecipes`, `TestTierLooks`.

Counts of `go test -list`: root 268 to 246; `machine` 32 to 54.

### T106

`machine`, the four files that are split. `rangeBoxAt` (used by `look_branches_test.go`), `fillLab` (used by `pin_look_test.go`) and `countIn` are in both halves. `hasLine` (`codex_test.go`) is called only by the tests of `autoequip_test.go` that stay: the moved half has no copy. `runSteps` (in the root's `helpers_test.go` since T99) is called only by `TestHandsFreeLine`, which stays.

| Root file | Goes to | Mode | Stays in the root | In both | Imports of the moved file | Imports the root file loses |
|---|---|---|---|---|---|---|
| `autoequip_test.go` | `machine/autoequip_test.go` | split | `TestAutoCrafterMakesARecipe`, `TestAutoCrafterOnlyMakesResearchedRecipes`, `TestMachineStatusLines` | `rangeBoxAt` | +`core` +`core/coretest` +`machine/machinetest` −`common/i18n` −`github.com/df-mc/dragonfly/server/event` −`github.com/df-mc/dragonfly/server/item/inventory` −`naming` −`progress` | `github.com/df-mc/dragonfly/server/entity` |
| `lab_test.go` | `machine/lab_test.go` | split | `TestLabScreen` | `fillLab`, `countIn` | +`core/coretest` +`machine/machinetest` −`progress` | `farm`, `github.com/df-mc/dragonfly/server/block`, `strings` |
| `machine_test.go` | `machine/machine_test.go` | split | `TestHandsFreeLine` |  | +`core/coretest` +`machine/machinetest` −`common/itemnet` | `common/energy`, `common/fluidnet`, `common/slotdeco`, `farm`, `fmt`, `github.com/df-mc/dragonfly/server/event`, `github.com/df-mc/dragonfly/server/item/inventory`, `github.com/df-mc/dragonfly/server/player`, `github.com/go-gl/mathgl/mgl64`, `machinerecipe`, `naming`, `network`, `sgitem`, `sievetable`, `storage`, `strings`, `time` |
| `sieve_test.go` | `machine/sieve_test.go` | split | `TestSieveTopsUpTheHand`, `TestSieveTakesCrushedBlocks` |  | +`core/coretest` +`machine/machinetest` −`codex` −`core` | `progress`, `time` |

Names that go from the root to `machine` (50): `TestElectricFurnace`, `TestItemVacuum`, `TestItemVacuumTakesItemsOnItsFloor`, `TestItemVacuumOffset`, `TestTreeFeller`, `TestTreeFellerBoneMealsSaplings`, `TestTreeFellerPlantsOnAnySoilInReach`, `TestRangeBoxFollowsAMovedCentre`, `TestLabRecipes`, `TestLabBrews`, `TestLabWaits`, `TestLabStopsWithoutRoom`, `TestLabAccepts`, `TestFarmingIngredientsGrow`, `TestLabNoticesAnIngredientTakenAway`, `TestLabFeedLeavesRoom`, `TestLabSaved`, `TestMachineIDsDistinct`, `TestCrusher`, `TestAutoSieveKeepsProductsAndStopsWhenFull`, `TestHandSieveDoesNotSiftAlone`, `TestAutoMinerKeepsItsOutput`, `TestAutoMinerMinesTheBlockItFaces`, `TestMachineWindowAndNBT`, `TestMineable`, `TestCustomBlockBareHandTime`, `TestServerFinishesCustomBlockBreak`, `TestServerFinishesCustomBlockBreakWithTool`, `TestMinerLegacySlots`, `TestHopperFillsHandSieve`, `TestRingGaugesInTheWindows`, `TestEnergyGauge`, `TestAutoSieveWithManyPossibleDrops`, `TestCrusherOre`, `TestCrusherBonusRolls`, `TestAlloySmelter`, `TestEnergyCell`, `TestMobSimulator`, `TestFluidLine`, `TestOreWasher`, `TestTankBuckets`, `TestMachineFillTakesOnlyWhatFits`, `TestQuickMoveIntoMachineGoesToInput`, `TestMachineInfoWarnsWhenStalled`, `TestSieveByHand`, `TestSieveNeedsAMesh`, `TestSieveTablesGiveOreChunks`, `TestSieveActsOn5x5`, `TestEmptyHandTakesTheMeshOut`, `TestSieveShowsItsMesh`.

Counts of `go test -list`: root 246 to 196; `machine` 54 to 104.

### T107

The documents and two guards. `testsupport_test.go` is a new file of the root package; it uses `layers` (`layers_test.go`) and `modulePrefix` (`pin_order_test.go`).

`content/skygear/testsupport_test.go`, the whole file:

```go
package skygear

import (
	"go/build"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// testSupport are the packages that exist for tests (module-relative paths):
// they give a package's tests a session, a player and the features they need
// without the root package.
var testSupport = []string{"content/skygear/core/coretest", "content/skygear/machine/machinetest"}

// TestSupportPackagesAreTestOnly: no non-test file of the module imports a
// package of testSupport, so the server never links one.
func TestSupportPackagesAreTestOnly(t *testing.T) {
	err := filepath.WalkDir(filepath.Join("..", ".."), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "third_party":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(token.NewFileSet(), p, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, im := range f.Imports {
			path, _ := strconv.Unquote(im.Path.Value)
			if rel, ok := strings.CutPrefix(path, modulePrefix); ok && slices.Contains(testSupport, rel) {
				t.Errorf("%s imports %s: only test files import a test support package", p, rel)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestPackageTestsStayBelow: the tests of a package run without the composed
// game. No package's tests import the root package (wire excepted: it tests
// what the composed game sends a client), and the tests of a package below
// the content lists import nothing from the content lists up (catalog, codex,
// screens): a test that needs the game's recipes, research or screens belongs
// in the root package. A package's tests set features up through
// coretest.Setup, which remembers what it set up: outside core, no test file
// calls core.Install or a feature's Setup itself.
func TestPackageTestsStayBelow(t *testing.T) {
	const root = modulePrefix + "content/skygear"
	top := slices.Index(layers, "catalog")
	if top < 0 {
		t.Fatal("layers has no catalog")
	}
	for i, p := range layers {
		pkg, err := build.Default.ImportDir(p, 0)
		if err != nil {
			t.Errorf("%s: %v", p, err)
			continue
		}
		for _, imp := range slices.Concat(pkg.TestImports, pkg.XTestImports) {
			if imp == root && p != "wire" {
				t.Errorf("the tests of %s import the root package: a test of the composed game belongs in the root package", p)
			}
			if dep, ok := strings.CutPrefix(imp, root+"/"); ok && i < top && slices.Contains(layers[top:], dep) {
				t.Errorf("the tests of %s import %s: a test that needs the content lists or the screens belongs in the root package", p, dep)
			}
		}
		if p == "core" {
			continue
		}
		for _, name := range slices.Concat(pkg.TestGoFiles, pkg.XTestGoFiles) {
			src, err := os.ReadFile(filepath.Join(p, name))
			if err != nil {
				t.Fatal(err)
			}
			for _, call := range []string{"core.Install(", "Feature.Setup()"} {
				if strings.Contains(string(src), call) {
					t.Errorf("%s/%s calls %s: a package's tests set features up with coretest.Setup", p, name, call)
				}
			}
		}
	}
}
```

`docs/tech/extension.md`:

```diff
diff --git a/docs/tech/extension.md b/docs/tech/extension.md
index cbf9c24..93da8bc 100644
--- a/docs/tech/extension.md
+++ b/docs/tech/extension.md
@@ -240,7 +240,7 @@ Dragonfly 자체를 확장하는 방법은 [dragonfly.md](dragonfly.md) 에 있
 
    - `content/skygear/kinds_test.go` 의 `kindOwners` 에 종류와 기능을 한 줄 더한다. `TestKindOwners` 가 이 표와 `Def.Owner` 를 맞춰 본다.
    - 소유한 기능이 등록하지 않으면 게임이 시작하지 않는다. 기계 기능의 `Check` 인 `machine.CheckKinds` 가 `machine kinds without behaviour: …` 오류를 돌려주고 `Install` 이 그 오류로 패닉한다. 소유 기능이 기능 목록에 없을 때도 같다.
-   - 기능 패키지의 테스트는 기능 목록 없이 돈다. 종류의 동작이 필요한 테스트는 `init` 에서 `<기능>.Feature.Setup()` 을 부른다(`netstore/keys_test.go`).
+   - 기능 패키지의 테스트는 기능 목록 없이 돈다. 종류의 동작이 필요한 테스트는 테스트 파일의 `init` 에서 `coretest.Setup(<기능>.Feature)` 를 부른다(`netstore/keys_test.go`, 아래 "기능의 테스트").
 
    팅커 테이블은 `Owner` 가 비어 있고 `machine/tinkertable.go` 가 동작을 정한다. `tinker` 가 `machine` 을 import 하면 `machine` 이 `tinker` 보다 먼저 초기화되어 기계 아이템이 팅커 도구보다 먼저 등록되고, 클라이언트가 받는 아이템 id 가 바뀌기 때문이다. 루트가 `hopper` 를 등록하는 것(`hopper.go`)과 같은 사정이다.
 6. 기능의 아이템에 등급이나 단계 표시가 있으면 그 아이템에 `Decor()` 메서드를 둔다(`slotdeco.Decorated`). 예는 `fairy/fairymagic.go` 의 요정 판이다.
@@ -335,6 +335,48 @@ core.Death.On(30, "tinker.keepFood", func(e core.PlayerEvent) {
 })
 ```
 
+## 기능의 테스트
+
+기능의 테스트는 기능 패키지 안에 둔다. 파일의 패키지 이름은 `<기능>_test` 다(바깥 테스트 패키지). 바깥 테스트 패키지는 그 기능을 import 하는 패키지도 import 할 수 있어서 순환이 생기지 않는다. 한 패키지만 돌려 볼 수 있다: `go test ./content/skygear/farm`.
+
+기능 패키지의 테스트 바이너리에는 루트 패키지가 없으므로 기능 목록이 설치되지 않는다(`core.Install` 은 루트의 `init` 에서만 돈다). 구독자도, 설정도, 다른 기능이 등록하는 기계 종류의 동작도 없는 상태다. 테스트에 필요한 것은 지원 패키지 둘이 준다.
+
+- `content/skygear/core/coretest`
+  - `coretest.Setup(fs ...*core.Feature)`: 블록 등록을 닫고, 이 바이너리에서 아직 준비하지 않은 기능마다 설정을 등록하고 `Setup` 을 부른다. `Check` 는 부르지 않는다. 이미 준비한 기능과 `Install` 로 설치된 기능은 건너뛰므로 같은 기능을 두 번 적어도 된다. 훅은 첫 이벤트 뒤에 구독자를 받지 않으므로 테스트 함수가 아니라 테스트 파일의 `init` 에서 부른다.
+  - `coretest.Session(t)`: 동기 월드 위의 세션을 `core.Lifecycle` 로 돌려준다. `s.Session` 이 코어 세션이고, `s.Join`·`s.Second`·`s.Fast` 같은 메서드가 인스턴스가 하는 호출이며, `s.Handler(p)` 가 플레이어 핸들러다. `core.Of` 가 이 세션을 찾고, 월드의 핸들러는 `core.WorldHandler` 다. 데이터베이스는 없다.
+  - `coretest.LoadAround(s)`, `coretest.SpawnPlayer(s)`, `coretest.WithPlayer(t, s, h, f)`, `coretest.Tick(s, n)`, `coretest.ShapeAs(s, key)`: 청크 읽기, 플레이어 넣기, 플레이어와 함께 트랜잭션 돌리기, 틱 진행, 세션이 둔 디버그 도형 읽기.
+- `content/skygear/machine/machinetest`: `machinetest.Place(s, pos, kind)` 가 기계를 놓고 상태를 돌려준다(에너지가 필요 없고, 체에는 실 메쉬가 끼워져 있다). `machinetest.Pos` 는 테스트가 기계를 놓는 자리다. `machine` 을 import 하므로 `machine` 패키지에서는 `machine_test` 패키지의 파일만 쓸 수 있다.
+
+테스트가 세션을 만드는 패키지에는 `setup_test.go` 가 있고, 그 패키지의 테스트에 필요한 기능을 준비한다. 기본은 자기 기능 하나다.
+
+```go
+package farm_test
+
+import (
+	"github.com/presentkim/skygear/content/skygear/core/coretest"
+	"github.com/presentkim/skygear/content/skygear/farm"
+)
+
+func init() { coretest.Setup(farm.Feature) }
+```
+
+자기 기능 말고 더 준비하는 곳은 셋이다. `conduit` 는 연결망 단계가 돌도록 `network` 를 같이 준비한다. `netstore` 는 `setup_test.go` 없이 `keys_test.go` 의 `init` 이 자기 기능과 `storage` 를 준비한다(외부 저장소가 서랍 컨트롤러를 읽는다). `machine` 은 `netkinds_test.go` 의 `init` 이 저장망 종류의 동작 때문에 `netstore` 를 더 준비한다.
+
+루트 패키지(`content/skygear`)에는 조립된 게임을 보는 테스트를 둔다.
+
+- 핀과 골든(`pin_*_test.go`), 구조 규칙(`layers_test.go`, `kinds_test.go`, `testsupport_test.go` 등), 벤치마크.
+- `catalog` 의 레시피와 연구, 도감, 리소스팩, 세이브(저장, 다시 열기, 마이그레이션)를 보는 테스트.
+- 기능 셋 이상이 설치되어야 통과하는 테스트, 루트가 등록하는 호퍼(`hopper.go`)가 필요한 테스트, 자기가 import 하지 않는 기능이 등록한 아이템이 필요한 테스트.
+- 루트의 내보내지 않은 이름(`Game`, `session`, `giveItems` 등)을 쓰는 테스트와 런타임 id 를 보는 테스트.
+
+루트의 `helpers_test.go` 는 같은 도우미를 루트의 `session` 위에서 준다(`testSession`, `loadAround`, `spawnPlayer`, `withPlayer`, `tick`, `placeTestMachine`, `machinePos`). `testSession` 은 데이터베이스와 `Game` 도 만든다.
+
+지킬 것이 셋 있다.
+
+- 패키지의 테스트는 루트 패키지를 import 하지 않고, 내용 목록 아래 칸의 패키지 테스트는 `catalog`, `codex`, `screens` 를 import 하지 않는다(`TestPackageTestsStayBelow`).
+- 테스트 지원 패키지는 테스트 파일만 import 한다(`TestSupportPackagesAreTestOnly`).
+- `dimension` 의 테스트 바이너리에서는 `popup_wire_test.go` 가 서버에 접속해 `testdata/popup.golden` 의 아이템 id 를 본다. 이 패키지의 테스트에 블록이나 아이템을 등록하는 패키지를 새로 import 하면 id 가 바뀐다.
+
 ## 클라이언트 항목 고치기
 
 클라이언트가 받는 커스텀 아이템·블록 항목은 `core.EditEntries(srv)` 가 한 번에 정한다. 서버를 만든 뒤 플레이어가 들어오기 전에 한 번 부른다.
@@ -373,6 +415,8 @@ core.Death.On(30, "tinker.keepFood", func(e core.PlayerEvent) {
 | 기능의 명령어가 기능 목록 순서대로 모인다 | `TestFeatureCommands`, `core.TestCommandsInListOrder` |
 | 아이템이 자기 칸 장식을 말하고, 말하지 않는 아이템은 일반 등급이다 | `TestItemDecor` |
 | 기능 하나가 목록과 자기 패키지만으로 이벤트를 받는다 | `core.TestNewFeatureNeedsOnlyTheList` |
+| 테스트 지원 패키지(`core/coretest`, `machine/machinetest`)는 테스트 파일만 import 한다 | `TestSupportPackagesAreTestOnly` |
+| 패키지의 테스트는 루트를 import 하지 않는다(`wire` 는 예외). 내용 목록 아래 칸의 테스트는 `catalog`, `codex`, `screens` 를 import 하지 않는다. `core` 밖의 테스트는 `core.Install` 이나 기능의 `Setup` 을 직접 부르지 않고 `coretest.Setup` 을 쓴다 | `TestPackageTestsStayBelow` |
 | 플레이어 핸들러의 모든 메서드가 `core.PlayerHandler` 에 있고 훅마다 구독자가 있으며, 루트가 같은 이름의 메서드를 두지 않는다 | `TestHandlerMethodsOneWay` |
 | 핫 경로는 할당하지 않는다. 이벤트마다 `AllocsPerRun` 으로 0 을 확인한다 | `core.TestEmitAllocatesNothing`, `core.TestEventsEmitAllocateNothing` |
 | 구독자는 `Install` 동안에만 단다. 첫 `Emit` 뒤의 등록, 같은 훅 안의 이름·우선순위 겹침은 패닉이다 | `core.TestRegisterAfterEmitPanics`, `core.TestSamePriorityPanics`, `core.TestSameNamePanics` |
```

`content/skygear/README.md`:

```diff
diff --git a/content/skygear/README.md b/content/skygear/README.md
index 84a6bfe..70e6303 100644
--- a/content/skygear/README.md
+++ b/content/skygear/README.md
@@ -67,6 +67,12 @@
 - `core.NewSaved`: 저장되는 데이터. 모든 `Saved` 파트는 세이브 상태 문서(`skygear.state`) 하나의 필드이고,
   필드 이름은 파트끼리 겹치지 않는다(겹치면 시작할 때 패닉). 그래서 기능을 옮겨도 세이브 형식은 그대로다.
 
+## 테스트
+
+기능의 테스트는 기능 패키지 안에 `<기능>_test` 패키지로 있고, 세션과 플레이어는 `core/coretest`, 기계는 `machine/machinetest` 에서 얻는다.
+루트 패키지(`content/skygear`)의 테스트는 조립된 게임을 본다: 핀과 골든, 구조 규칙, 레시피·연구·도감·리소스팩·세이브, 기능 여럿이 함께 있어야 하는 장면.
+어디에 두고 기능을 어떻게 준비하는지는 [docs/tech/extension.md](../../docs/tech/extension.md) 의 "기능의 테스트"에 있다.
+
 ## 파일
 
 문서에 나오는 `sieve.go`, `codexsearch.go` 같은 파일은 그 기능의 패키지 안에 있다
```

`CONTRIBUTING.md`:

```diff
diff --git a/CONTRIBUTING.md b/CONTRIBUTING.md
index 21eba16..9bfb8a1 100644
--- a/CONTRIBUTING.md
+++ b/CONTRIBUTING.md
@@ -71,6 +71,13 @@ Thanks for your interest. These are the rules the project follows.
   the features and holds the platform glue (`Game`, `session`) and the
   operator tools (`/sgadmin`, `/sgdebug`); it holds no player command and
   subscribes to nothing. See `docs/tech/extension.md`.
+- A feature's tests live in its package, as `package <feature>_test`. They get
+  a session and a player from `content/skygear/core/coretest` and machines
+  from `content/skygear/machine/machinetest`, and name the features they need
+  in `coretest.Setup`, called from the `init` of the package's `setup_test.go`.
+  Tests of the composed game (pins and goldens, the structure rules, recipes,
+  research, the pack, saves, scenes that need several features) stay in
+  `content/skygear`.
 - `go test ./content/skygear -run TestPin -count=1` compares the order of every
   hook's subscribers, the steps each scenario reaches, the WAILA panels
   (`look.golden`) and the creative list (`creative.golden`) with the goldens in
```

`docs/tech/machines.md`:

```diff
diff --git a/docs/tech/machines.md b/docs/tech/machines.md
index 0e4085d..adc1001 100644
--- a/docs/tech/machines.md
+++ b/docs/tech/machines.md
@@ -96,7 +96,7 @@
 
 ## 검증
 
-- 단위: `machine_test.go`(분쇄·자동 체·채굴·창 조작 제한·NBT 왕복·부술 때 드롭·무인 라인), `sieve_test.go`, `harvester_test.go`, `hammer_test.go`.
+- 단위: `machine/machine_test.go`(분쇄·자동 체·채굴·창 조작 제한·NBT 왕복·부술 때 드롭), `machine/sieve_test.go`, `farm/harvester_test.go`, `tinker/hammer_test.go`. 호퍼가 낀 무인 라인(`TestHandsFreeLine`)과 도감·자동 채움이 낀 체 테스트는 루트의 `machine_test.go`, `sieve_test.go` 에 있다.
 - 종류 등록: `machine/kinds_test.go` 가 등록되지 않은 종류의 오류, `Register` 가 패닉하는 경우, `Order` 와 `Kinds` 가 같은지, 플래그가 붙은 종류, 종류마다 모양이 하나인지 본다. 루트의 `kinds_test.go`(`TestKindOwners`)는 종류마다 소유 기능이 `kindOwners` 와 같고 목록에 있으며 모두 등록되었는지 본다.
 - 핀: 루트의 `pin_kinds_test.go`(`TestPinKinds`, `testdata/pin/kinds.golden`)가 종류마다 서서·웅크려 클릭한 결과, 창의 정보 아이콘, 창 칸의 역할과 받는 아이템을 고정한다. `TestForeignKindsRoundTrip` 은 다른 기능의 종류가 이름으로 저장되고 다시 읽히는지 본다.
 - 실기 확인 항목은 [TODO](../TODO.md).
```

`docs/design/11-research-essences.md`:

```diff
diff --git a/docs/design/11-research-essences.md b/docs/design/11-research-essences.md
index 09752da..e88ad1c 100644
--- a/docs/design/11-research-essences.md
+++ b/docs/design/11-research-essences.md
@@ -142,7 +142,7 @@
 | 연구 트리·정수 추출기 조합법 | `catalog/research.go` (`baseResearchTree`, `laboratory`) |
 | 도감·도움말·발전과제·카운터 | `codex/codex.go`, `screens/help.go`(`helpEssenceText`), `progress/advancement.go`, `achievement/counter.go`·`achievement/catalog.go` |
 | 텍스처 | 손으로 관리하는 PNG(`assets/resourcepack/skygear/textures/`) |
-| 검사 | `lab_test.go`, `researchessence_test.go`(정수 비용 표, 만들 수 있는지의 순서 점검) |
+| 검사 | `machine/lab_test.go`, 루트의 `lab_test.go`(실험실 화면), `researchessence_test.go`(정수 비용 표, 만들 수 있는지의 순서 점검) |
 
 연구 트리의 정수 비용은 기본 트리에 코드로 더하므로, 비용을 바꾸는 일은 `progress.EssenceCosts` 한 곳에서 한다. 새 정수는 종류 상수, 정수 추출기 레시피, 아이템 이름·텍스처, 비용 표, `makers`(`TestEssencesCanBeMade`)를 더하면 된다.
 
```

Both guards pass on the tree T106 leaves. `TestPackageTestsStayBelow` fails when a test of a package below `catalog` imports `catalog`, `codex` or `screens`, when a package's test imports the root (but `wire`), and when a test file outside `core` contains `core.Install(` or `Feature.Setup()`. `TestSupportPackagesAreTestOnly` fails when a non-test file of the module imports `core/coretest` or `machine/machinetest`.

Documents under `docs/` and the README are Korean and say what is, without history. Every identifier and file a document names exists (`grep`).

Not in this task: the open-items file under `docs/` names `content/skygear/dimension_test.go` for `TestHuntClear`; after T100 the file is `content/skygear/dimension/dimension_test.go`. The plan lint refuses that file's name in a plan, so the coordinator corrects the one path.
