# Phase 3 part 3-4 (the `machine` package gives the foreign kinds' behaviour back): reference

The machine plans `2026-11-10-cleanup-restructure-p3i.md`, `-p3j.md` and `-p3k.md` (tasks T75 to T87) cite this file as REF. It holds what a task needs word for word: the inventory, the decisions, every new name with its signature, the measured numbers and the code.

Everything here was measured at a31f0d6 (the non-test sources of `machine`, `storage`, `farm`, `dimension`, `tinker`, `netstore`, `decor` and `core` are those of bb72e2e but for one comment in `core/feature.go`). The code blocks were built and run in a scratch worktree, task by task in id order: after each of T75 to T86 `go build ./...`, `go vet ./...` and `go test ./...` pass (51 packages), `gofmt -l` is clean, and the eight goldens of `content/skygear/testdata/pin/`, the four of `content/skygear/wire/testdata` and `content/skygear/dimension/testdata/popup.golden` are byte for byte unchanged. The combined patch is `/tmp/claude-0/-home-user-SkyGear/a71fa8d5-6f20-5c3f-ba20-eeb27e13ad8a/scratchpad/proto-3-4.patch` (it applies to a31f0d6); it is a reference for a task's exact code, not something to apply whole. A task copies its blocks as they are. Where a block and the code a developer finds disagree, the developer stops and reports (`NEEDS_CONTEXT`); it does not choose.

## 1. Inventory

### 1.1 The kinds

`machine.Kinds` and `machine.Order` hold 36 kinds (`machine/machine.go:299-483`); `machine`'s 29 non-test files have 6,700 lines, `machine.go` 1,974 of them. Twenty kinds are the machine feature's own, sixteen belong to another feature by meaning: storage 3, tinker 1, dimension 2, farm 1, netstore 9. The review plan and the audit say 17; the count is 16.

| Kind (name in saves: `skygear:<kind>`) | Id | Owner | Its behaviour at HEAD |
|---|---|---|---|
| `crusher` | `idCrusher` | machine | `machine.go` `runMachine` 1783-1800, `CrusherOutput` 488 |
| `sieve`, `auto_sieve`, `magic_sieve` | `idSieve`, `idAutoSieve`, `idMagicSieve` | machine | `sieve.go` (485 lines), `magicsieve.go` (72), `mesh.go` (61) |
| `auto_miner` | `idMiner` | machine | `machine.go` `runMiner` 1899-1936 |
| `generator`, `energy_cell` | `idGenerator`, `idEnergyCell` | machine | `generator.go` (172) |
| `alloy_smelter` | `idAlloy` | machine | `alloy.go` (46) |
| `fluid_tank`, `ore_washer`, `crucible`, `lava_generator` | `idTank`, `idWasher`, `idCrucible`, `idLavaGenerator` | machine | `fluidmachine.go` (260) |
| `solar_generator`, `hydro_generator` | `idSolarGenerator`, `idHydroGenerator` | machine | `naturegen.go` (212) |
| `mob_simulator` | `idSimulator` | machine | `essence.go` (58) |
| `electric_furnace`, `item_vacuum`, `tree_feller` | `idFurnace`, `idVacuum`, `idFeller` | machine | `autoequip.go` (266) |
| `auto_crafter` | `idCrafter` | machine | `autocraft.go` (285) |
| `laboratory` | `idLab` | machine | `lab.go` (158), `labui.go` (216) |
| `harvester` | `idHarvester` | farm | work: `farm/harvester.go` (148; `init` at 148 sets `Run`); in `machine`: the cycle line of the panel (`look.go:121`), the cycle ring (`machine.go:1147`), its range numbers (`tier.go:66,154-166`) |
| `drawer_controller`, `drawer_interface` | `idDrawerController`, `idDrawerInterface` | storage | `storage/storage.go` (463; `init` at 452-463 sets `Click`, `Insert`, `Peek`, `Extract`, `Lines`); in `machine`: the flag `Def.Controller` (`machine.go:267`, read at `look.go:129,201`) |
| `drawer_trim` | `idDrawerTrim` | storage | nothing in `storage`; in `machine`: `Activate` returns true (`machine.go:1579`), no panel lines (`look.go:127`), its texture (`machine.go:1348`) |
| `tinker_table` | `idTinker` | tinker | the screen: `tinker/table.go` `ShowTable` (309); in `machine`: the click (`machine.go:1582-1587`), no panel lines (`look.go:127`) |
| `essence_collector` | `idCollector` | dimension | `dimension/collector.go` (239; `init` at 232-239 sets `Click`); in `machine`: the monster line of the info icon (`machine.go:782-788`) |
| `relic_table` | `idRelicTable` | dimension | `dimension/relic.go` (210; `init` sets `Click` at 85); in `machine`: no panel lines (`look.go:127`) |
| `net_controller` | `idNetController` | netstore | `netstore/network.go:314` (`Run`), `terminal.go:309,319-340` (`Click`, item I/O) |
| `net_cable` | `idNetCable` | netstore | nothing in `netstore`; in `machine`: a click does nothing (`machine.go:1606`), its thin shape (`netshape.go`) |
| `disk_drive` | `idDiskDrive` | netstore | `netstore/items.go:135` (`Accepts`) |
| `net_terminal` | `idNetTerminal` | netstore | `netstore/terminal.go:315,319-340` (`Click`, item I/O) |
| `net_importer`, `net_exporter` | `idNetImporter`, `idNetExporter` | netstore | `netstore/filterui.go:26-35` (`Click`), `io.go:128-130` (`Run`), `items.go:129-134` (`Accepts`); shape in `machine/netshape.go` |
| `net_external` | `idNetExternal` | netstore | `netstore/filterui.go:26-35` (`Click`), `io.go:128-130` (`Run`); shape in `machine/netshape.go` |
| `pattern_encoder` | `idPatternEncoder` | netstore | `netstore/encoder.go:44-63` (`Accepts`, `Click`, and it rewrites the kind's `Input` and `Output`) |
| `net_autocrafter` | `idNetAutocrafter` | netstore | `netstore/craft.go:139-144` (`Run`), `items.go:139` (`Accepts`) |

At HEAD the owners already hold almost all of this behaviour. What is left to do is the way it is wired (nineteen assignments into the exported map `machine.Kinds` from eleven `init` functions, with nothing that says which feature a kind waits for), the clauses `machine` still keeps for foreign kinds, and the two imports of `machine` that point at features.

### 1.2 Places in `machine` that branch on a kind or name a foreign kind

| Row | Where (HEAD) | What it does | Kinds | Choice |
|---|---|---|---|---|
| M1 | `machine.go:782-788` `State.InfoItem` | adds the collector's monster line to the info icon | `idCollector` | hook `Behaviour.Info`, filled by `dimension` (T81) |
| M2 | `machine.go:856-886` `State.Accepts`, `897` `takesCharge`, `1094` `Insert` | what an input takes | 11 own ids, `idFeller`, `idLab` | stays: the machine feature's own kinds |
| M3 | `machine.go:1142-1149` `progressLook` | ring or arrow and its text | 10 own ids, `idHarvester` | `idHarvester` becomes the data flag `Def.Cycle` (T78); the rest stays |
| M4 | `machine.go:1284-1285` `cubeMachines`, `frontMachines` | which texture set a kind has | 28 kinds, 10 of them foreign | stay as data: the client shape is identity (the owner's decision); they move to `shape.go` (T85) |
| M5 | `machine.go:1289-1361` `States`, `Permutations`, `ClientExtras`, `Properties` | the client definition; each branches by hand over sieve, tier look, animated, net shape, cube, front, and `KindDrawerTrim` (1329, 1348) | all | one classification `shapeOf` (T85); the trim's texture becomes the data table `plainCubes` |
| M6 | `machine.go:1472`, `1955` | the facing of an attachment or a miner is saved and set on placing | `netHeads`, `KindMiner` | stays: the codec and the shape frame |
| M7 | `machine.go:1508`, `1546`, `1588`, `1626`, `1368`, `1951`, `715` | laboratory, sieve and miner particulars | own | stays |
| M8 | `machine.go:1576-1608` `Machine.Activate` | the click: the hook `Click`, then `KindDrawerTrim` (1579), `KindTinker` (1582), the laboratory, the crafter, fluids, `Cell || KindNetCable` (1606), the window | 3 foreign | trim: `Click` of `storage` (T79); tinker table: `Click` set in `machine/tinkertable.go` (T83); cable: the data flag `Def.NoWindow` (T78) |
| M9 | `machine.go:1692` `Machine.Tick` | redraws a thin network block every 40 ticks | the 4 net shapes | stays: the shape frame |
| M10 | `machine.go:1731-1737` `Machine.Tick` | ends the tick before the window display for 12 ids | 6 own, 6 foreign | the data flag `Def.NoDisplay` on the same 12 kinds (T78) |
| M11 | `machine.go:1782-1835` `runMachine` | the work of one run: 16 clauses, the hook `Run` in the default | own | stays: the machine feature's own kinds |
| M12 | `look.go:112-150` `MachineLines` | the panel lines: 10 clauses | harvester (121), trim, tinker table, relic table (127), `Def.Controller` (129) | harvester: `Def.Cycle` (T78); the three: `Lines: machine.NoLines` from the owner (T79, T81, T83); `Controller`: the clause becomes "a kind with `Lines` of its own" (T78) |
| M13 | `look.go:201` `outputContainer`, `machine.go:265-267` `Def.Controller` | no output slots for a controller | 2 storage kinds | removed (T79): both kinds have no output slots, so `len(Output) == 0` already says it |
| M14 | `tier.go:56-68` `tierKinds`, `tierRanges`; `tier.go:164` `MaxOffset`; `machine.go:109` `harvestRadius` | which kinds take kits; the reach by tier; the harvester's lift | harvester among others | the tables stay (tier data, the owner's decision); the one code branch `kind == KindHarvester` becomes the data table `rangeLifts` (T78) |
| M15 | `animate.go:24-27` `AnimatedKinds`, `machinecount.go:19-22` `CycleOnReset` | which kinds have a working look, which count cycles on reset | harvester in the first | stay: data |
| M16 | `netshape.go` (272 lines) | `NetNodes`, `netHeads`, `NetKinds`, the shapes, `LinksNet`, `NetLinker`, `showNetShape`, `NeighbourUpdateTick`, `NotePlaceFace` | 8 netstore kinds | stays whole (decision 9) |
| M17 | `subscribe.go:35-40` `machine.noteFace` | remembers the clicked face for an attachment or a miner | net shapes, miner | stays: the step's name is in `order.golden` and `handler-use.golden` |
| M18 | `rangeview.go:26,36-42`, `screen.go:108,173,266`, `monitor.go:359`, `naturegen.go:159-184`, `sieve.go:34` | range, miner target, laboratory tool, statistics, nature generators, sieves | own | stays |

### 1.3 Places outside `machine` that branch on a machine's kind or assert `machine.Machine`

| Row | Where (HEAD) | What | Choice |
|---|---|---|---|
| O1 | `storage/storage.go:452-463` (`init`) | fills `Click`, `Insert`, `Peek`, `Extract`, `Lines` of two kinds | `machine.Register` from `storage`'s `Setup` (T79) |
| O2 | `storage/storage.go:130-132` `IsSetMachine`, `151-158` `findNetwork`, `173-176` `isInterface`, `217-221` `findInterfaceController`, `355-358` `IsDrawerController`; `storage/subscribe.go:34,42` | the storage recognises its own three kinds in the world | leave (decision 10) |
| O3 | `farm/harvester.go:148` (`init`) | fills `Run` | `machine.Register` from `farm`'s `Setup` (T80) |
| O4 | `dimension/collector.go:232-239`, `relic.go:85-90` (`init`) | fill `Click` of two kinds | `machine.Register` from `dimension`'s `Setup` (T81) |
| O5 | `dimension/collector.go:133-139` `collectorAt` | the dimension reads its own collector | leave (decision 10) |
| O6 | `netstore`: `filterui.go:26-35`, `terminal.go:308-341`, `network.go:313-315`, `craft.go:139-144`, `encoder.go:44-63`, `io.go:128-130`, `items.go:129-142` (seven `init` functions) | fill `Click`, `Run`, `Insert`, `Peek`, `Extract`, `Accepts` of eight kinds; `encoder.go:46-47` rewrites `Input` and `Output` of the encoder | `machine.Register` from `netstore`'s `Setup`; the encoder's slots are declared in `machine.Kinds` (T84) |
| O7 | `netstore/nodes.go:17` `nodeKinds = machine.NetNodes`, `34-43` `nodeUse`, `57-60` `nodeAt`; `netstore/subscribe.go:27,40` | the network recognises its own nodes | leave (decision 10) |
| O8 | `copier/copier.go:173,199,250-252` | copies the settings of any machine and of the crafter | leave: row R11 of the 3-3 reference (the copier is the top of the three it copies) |
| O9 | `fairy/fairytime.go:68` | winds the hourglass of any machine (`State.WindHourglass`) | leave: the frame's public API, no kind named |
| O10 | `conduit/bundle.go:224-235` | `machine.LinksNet`, `machine.NetLinker` | leave: the shape frame's public API |
| O11 | `catalog/recipes.go` (39 kind names), `codex/codexcards.go` and others (22 symbols), `screens/help.go` (10) | recipes, codex cards and help name kinds through `machine.Item(kind)` | leave: content lists (review plan, out of scope) |
| O12 | root `debug_ui.go:32`, `debug_factory.go`, `admin.go:84` | operator tools | leave; `debug_ui.go:32` changes one expression in T86 |
| O13 | `migrate` | nothing: it names no machine kind | |

By package, uses of `machine` symbols at HEAD: netstore 18 symbols in 178 places, storage 6 in 24, farm 5 in 22, dimension 6 in 11, copier 2 in 4, conduit 2 in 3, fairy 1; above them catalog 39 in 85, codex 22 in 76, screens 11 in 22.

### 1.4 What each side needs of the other

| Owner | What it calls in `machine` | What `machine` needs of it |
|---|---|---|
| storage | `Machine`, `New`, the three `Kind…` names, `Register`, `Behaviour`, `NoLines` | the click, item I/O and lines of the controller and the interface; the click and "no lines" of the trim |
| farm | `State` (`TickWork`, `RoomFor`, `Emit`, `Area`, `Centre`, `Progress`, `Done`, `Inv`), `KindHarvester`, `BucketMB`, `FluidOfBucket`, `Register` | the harvester's `Run` |
| dimension | `State` (`Mob`, `SetMob`, `Inv`), `Ticking`, `Machine`, two `Kind…` names, `Register`, `NoLines` | the click of both kinds, the collector's info line, "no lines" of the relic table |
| netstore | `Machine`, `State`, `New`, `Place`, `NetMask`, `NetNodes`, `Order`, `Kinds[k].Title`, nine `Kind…` names, `Register` | click, run, item I/O, slot rule of eight kinds; nothing for the cable |
| tinker | nothing (it cannot import `machine`, decision 6) | `tinker.ShowTable`, called from `machine/tinkertable.go` |

### 1.5 The shared frame that stays in `machine`

The block type `Machine` and its `State`; `Def`, `Kinds`, `Order` and the registration of every block and item in `init` (`machine.go:43-75`); the 27 slot window with its roles, the machine handler and the Ore UI screen (`screen.go`); item I/O through hoppers and conduits; the energy buffer and `TickWork` (`generator.go`), fluids (`fluidmachine.go`); tiers, kits and tier looks (`tier.go`, `tierlook.go`); the working look (`animate.go`); the range, its screen and the range box (`rangeview.go`); the thin network shape (`netshape.go`); the NBT codec (`EncodeNBT`, `DecodeNBT`, `persistState`); the statistics and the stall monitor (`stats.go`, `monitor.go`, `machinecount.go`); the panel frame (`look.go`: `Waila`, `MachineLines`, `ConsumerLines`, `BlockedLines`, `BoostLine`, `outputContainer`); the client definition (`States`, `Permutations`, `Properties`, `ClientExtras`); the creative run and the statistics command.

## 2. Decisions

They bind. One line of reason each.

### 2.1 Per foreign kind

(a) the behaviour is the owner's and is registered from its `Setup`; (b) it stays in `machine` as generic machinery; (c) it stays for now.

| Kind | Choice | What the owner registers (`machine.Behaviour`) | What stays in `machine` and why |
|---|---|---|---|
| `harvester` | (a) for the work, (b) for the cycle display | `Run: RunHarvester` | (b) the cycle line and ring become the data flag `Def.Cycle`: the frame draws every window and panel from the definition, and a hook that replaced the lines would have to rebuild the energy, blocked and boost lines of the frame. The reach by tier is tier data. |
| `drawer_controller`, `drawer_interface` | (a) | `Click` (`UseController`, true), `Insert`, `Peek`, `Extract` (`ControllerInsert`, `ControllerPeek`, `ControllerExtract`), `Lines` (`storage.Lines`) | the texture set (data) |
| `drawer_trim` | (a) | `Click` returning true, `Lines: machine.NoLines` | its one texture (data, `plainCubes`) |
| `tinker_table` | (c) | nothing: `Owner` stays empty | `machine/tinkertable.go` sets `Click` (debounce, `tinker.ShowTable`) and `Lines: NoLines` in an `init`. Measured: with `tinker` importing `machine` instead, the machines initialise before the tinker's tools and `wire/testdata/items.golden` changes in 512 entries, `recipes.golden` in 274, `creative.golden` in 155, `dimension/testdata/popup.golden` in 2. Goldens may not be re-recorded in this part. |
| `essence_collector` | (a) | `Click` (debounce, `ShowCollector`), `Info` (the monster line) | the window layout (data) |
| `relic_table` | (a) | `Click` (debounce, `ShowRelicTable`), `Lines: machine.NoLines` | nothing |
| `net_controller` | (a) | `Click`, `Run`, `Insert`, `Peek`, `Extract` | energy numbers (data) |
| `net_cable` | (a), empty | `machine.Behaviour{}` (the owner claims it) | (b) the thin shape; "a click does nothing" is the data flag `Def.NoWindow` |
| `disk_drive` | (a) | `Accepts` | slots (data) |
| `net_terminal` | (a) | `Click`, `Insert`, `Peek`, `Extract` | slots (data) |
| `net_importer`, `net_exporter` | (a) | `Click`, `Run`, `Accepts` | (b) the thin shape and the facing |
| `net_external` | (a) | `Click`, `Run` | (b) the thin shape and the facing |
| `pattern_encoder` | (a) | `Click`, `Accepts` | its slots, now declared in `machine.Kinds` (`Input` 0 to 12, `Output` 14) instead of rewritten by `netstore`'s `init` |
| `net_autocrafter` | (a) | `Run`, `Accepts` | slots (data) |

### 2.2 The rest

1. Identity stays (the owner's decision): `Kinds`, `Order`, the ids, the registration loop, the texture sets, the tier tables and the net shapes are in `machine`, in today's order. No task moves a `world.RegisterBlock` or `world.RegisterItem` call, and no golden is re-recorded.
2. The registry is the map `machine.Kinds`, keyed by the kind's name, as today. A kind of another feature names that feature in `Def.Owner`; the owner hands the kind's `machine.Behaviour` to `machine.Register(kind, b)` once, from its `Setup`. `Register` writes into the same `*Def` every `State` already points at, so the tick reads its hooks exactly as before (no map lookup, no interface, no allocation).
3. From `Setup`, not from `init`: a feature provides its capabilities when it is installed (as the panel adapters and the panel followers of 3-3), and only then can a missing owner be told from a present one. Cost: a test binary of a feature package that needs a kind's behaviour sets the owner up itself (`netstore/keys_test.go` and `machine/netkinds_test.go` call `<owner>.Feature.Setup()` in an `init`; no package test emits a kernel hook).
4. Complete when the features are installed: `core.Feature` gets `Check func() error`; `core.Install` calls every `Check` after every `Setup` and panics with the first error. `machine.Feature.Check` is `machine.CheckKinds`: every kind of `Order` with an `Owner` must be registered. The root's `init` calls `core.Install`, so a game or a test of the root whose feature list lacks an owner does not start: `core: Install: feature "machine": machine kinds without behaviour: net_cable (feature netstore), …`. After the check passed `Register` panics (nothing changes a kind while a world runs). `Register` also panics for a name that is no kind, a kind without an owner and a second registration. The kernel stays generic: `Check` names nothing.
5. During the migration a kind whose owner has not moved yet has `Owner == ""` and its hooks are still assigned by the owner's `init` (the fields are promoted through the embedded `Behaviour`, so the old assignments compile). Each owner's task sets `Owner` on its kinds, registers them and deletes the `init` assignments. T86, when no assignment is left, makes the hooks an unexported field (`Def.hooks`) read through `Def.Hooks()`: from then on `Register` is the only way in, enforced by the compiler.
6. `tinker` cannot own its kind in this part (table above). `machine` keeps the import of `tinker`, in one file with one purpose, and the frame's `Activate` and `MachineLines` name no tinker table any more. When the client goldens may be re-recorded, `tinker` registers the kind (`Owner: "tinker"`) and `machine/tinkertable.go` is deleted. The same holds for `hopper.Register()` (3-3 reference, row R9).
7. `machine` stops importing `decor` as a feature: `tinker.RarityColour` becomes `slotdeco.Rarity.Colour()` in `common/slotdeco` (a rarity's colour belongs with the rarity), `decor` then imports only `naming` and moves to the core group of `layers`, beside `look` and `autofill`. `netstore`, `codex` and `screens` keep importing it from above.
8. Three kind checks in the frame's code become data in the kind's definition, because they describe the kind and the frame reads them: `Def.NoDisplay` (the twelve kinds whose tick ended before the window display, six of them foreign), `Def.NoWindow` (the cable), `Def.Cycle` (the harvester). `MaxOffset`'s `kind == KindHarvester` becomes the table `rangeLifts`.
9. `machine/netshape.go` stays: the thin shape is the block's model, its states, permutations and collision boxes, registered shape by shape in `machine`'s `init`; `conduit` uses `LinksNet`. The step `machine.noteFace` stays under its name.
10. The switches of 3-3's rows R12 and R13 stay as they are: after this part they are the owner recognising its own kinds (`storage` its three, `netstore` its nodes, `dimension` its collector) by names from `machine`'s list, which is where identity lives by decision 1. No package above an owner switches on them. A capability interface would have to be implemented by `machine.Machine` for another feature's sake, which is the direction this part removes.
11. The subscribers `storage.break`, `storage.place`, `netstore.break`, `netstore.place`, `machine.break`, `machine.place` stay subscribers: their names and order are in `order.golden` and `handler-blocks.golden`. `Behaviour` has no break or place hook.
12. The six unexported fields of `Def` (`nameKey`, `helpKeys`, `fluidSource`, `fluidSink`, `fluidOnly`, `autoSieve`) stay unexported: no owner declares a `Def`.
13. `ClientExtras` is unified with `States`, `Permutations` and `Properties` through `shapeOf` (three shapes: cube, sieve, net). `EncodeBlock`, `Hash` and `Model` are not touched (they run on every block change). `wire/testdata/blocks.golden` is unchanged.
14. `runMachine`, `State.Accepts` and the other switches over the machine feature's own kinds stay: turning them into hooks would change nothing about who owns what.
15. Tests stay in the root (3-5 moves them). New tests: `pin_kinds_test.go` and `kinds_test.go` in the root, `machine/kinds_test.go`, two kernel tests, one `slotdeco` test, one case in the root's `decor_test.go`.

## 3. New names

```go
// core
Feature.Check func() error            // runs at the end of Install, after every Setup; Install panics with its error

// machine (kinds.go)
type Behaviour struct {
	Click   func(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool
	Run     func(s *core.Session, st *State, pos cube.Pos, tx *world.Tx)
	Insert  func(tx *world.Tx, pos cube.Pos, st item.Stack) int
	Peek    func(tx *world.Tx, pos cube.Pos, want func(item.Stack) bool) item.Stack
	Extract func(tx *world.Tx, pos cube.Pos, st item.Stack, n int) item.Stack
	Accepts func(slot int, st item.Stack) bool
	Lines   func(lang string, tx *world.Tx, pos cube.Pos) []string
	Info    func(st *State) []wintext.Text                          // from T81
}
func NoLines(string, *world.Tx, cube.Pos) []string                  // always nil
func Register(kind string, b Behaviour)                             // once per owned kind, from the owner's Setup
func CheckKinds() error                                             // machine.Feature.Check
func (k *Def) Hooks() Behaviour                                     // from T86

// machine.Def
Owner string                    // "" or the feature that registers the kind
NoDisplay, NoWindow, Cycle bool // from T78
Behaviour                       // embedded from T77; the unexported field `hooks Behaviour` from T86
// removed: Controller (T79); the seven hook fields are now Behaviour's

// machine (shape.go, tier.go, tinkertable.go), unexported
type shape int; const shapeCube, shapeSieve, shapeNet; func shapeOf(kind string) shape
var plainCubes map[string]string; func cubeProperties(kind string) customblock.Properties
var rangeLifts map[string]int

// common/slotdeco
func (r Rarity) Colour() string   // was tinker.RarityColour(r)

// owners, unexported
storage.registerKinds(), farm.registerKinds(), dimension.registerKinds(), dimension.collectorInfo(st *machine.State) []wintext.Text
netstore.registerKinds(), clickFilter, clickController, clickTerminal, insertAt, peekAt, extractAt, runAutocrafter, encoderAccepts, clickEncoder, acceptsUpgrade, acceptsDisk, acceptsPattern

// root tests
var kindOwners map[string]string            // kinds_test.go
var machineFeatureImports map[string]string // layers_test.go
func pinScreens(call func()) string         // pin_kinds_test.go
```

Removed: `tinker.RarityColour`, `machine.Def.Controller`, every assignment to a hook of `machine.Kinds[...]` outside `machine`.

## 4. Imports and layers, before and after

| Package | Before | After |
|---|---|---|
| `machine` | achievement, advancement, autofill, blockmodel, climate, core, **decor** (feature layer), formicon, island, machinerecipe, naming, netshape, network, progress, sgitem, sievetable, textfmt, **tinker**, wintext | the same list; `decor` is now in the core layer; `tinker` is imported by `machine/tinkertable.go` only |
| `decor` | naming, tinker | naming |
| `tinker` | (no change in its imports) | `RarityColour` is gone |
| `dimension` | … machine | … machine, wintext |
| `core` | (no new import) | |

`layers` after T82: the core group is `"core", "network", "achievement", "preview", "look", "autofill", "decor"`, the features group `"tinker", "machine", "conduit", "storage", "farm", "fairy", "dimension", "handwork", "expand", "copier", "wand", "netstore", "elevator"`; `featureLayer` is the same thirteen names. `siblings` after T82: the row `"decor"` is gone, `"machine": {"tinker"}`, `"netstore": {"conduit", "machine", "storage"}`; the other rows are unchanged. `onlyImports["decor"]` is `{"naming"}`. T83 adds `machineFeatureImports = {"tinker": "tinkertable.go"}` and `TestMachineImportsNoFeature`. The feature list (`Features`, `TestFeatureNames`) is not edited.

## 5. Measured numbers

Benchmarks in this container (`go test -run xxx -bench <names> -benchmem -count 6 ./content/skygear`), median ns/op, at HEAD with T75's two new benchmarks and with every block of this file applied:

| Benchmark | Before | After |
|---|---|---|
| `BenchmarkMachineTick` | 2,778 ns, 520 B, 8 allocs | 2,754 ns, 519 B, 8 allocs |
| `BenchmarkEnergyStep` | 3,604 ns, 1,573 B, 17 allocs | 3,737 ns, 1,573 B, 17 allocs |
| `BenchmarkSecond` | 27,844 ns, 10,557 B, 63 allocs | 26,142 ns, 10,557 B, 63 allocs |
| `BenchmarkLookMachine` | 6,988 ns, 4,232 B, 47 allocs | 6,023 ns, 4,232 B, 47 allocs |
| `BenchmarkLookInfoMachine` | 2,422 ns, 1,600 B, 23 allocs | 2,240 ns, 1,600 B, 23 allocs |
| `BenchmarkTickHarvester` (new, T75) | 2,384 ns, 200 B, 5 allocs | 2,213 ns, 200 B, 5 allocs |
| `BenchmarkTickDrawerController` (new, T75) | 2,182 ns, 200 B, 5 allocs | 2,102 ns, 200 B, 5 allocs |

The runs of one benchmark spread by about 15 percent in this container, and B/op of the tick benchmarks reads 200 or 201 and 519 or 520 from run to run (the world's own maps grow). The rule for T77 to T86: B/op and allocs/op equal T75's report; ns/op (median of 6) within 15 percent of it.

What pins each foreign kind (a task that moves a kind runs these and changes none):

| Kind | Panel | Click, info icon, slots | Work and item I/O | Saved state |
|---|---|---|---|---|
| harvester | `look.golden` `machine/harvester`, `/sneak` | `kinds.golden` | `harvester_test.go` (3 tests), `magic_test.go` (the plant harvest), `tier_test.go`, `BenchmarkTickHarvester` | `TestForeignKindsRoundTrip` (T75), `machine/rangeview_test.go` `TestRangeSettingIsSaved`, `TestOffsetIsCappedAndSaved` |
| drawer controller, interface | `look.golden` `machine/drawer_controller`, `machine/drawer_interface` | `kinds.golden` | `drawer_test.go` `TestDrawerController`, `TestStorageSearchAndDeposit`, `TestControllerSneakClickOpensNoList`, `TestControllerClickInsertsHeldItem`, `TestControllerDoubleClickInsertsInventory`, `TestDrawerInterfaceActsForItsController`, `TestTwoControllersStopTheSet`; `BenchmarkControllerPeek`; `netstore` `TestExternalDrawerController`; `storage/perf_test.go` | `TestForeignKindsRoundTrip`; the drawers themselves `drawer_reload_test.go` |
| drawer trim | `look.golden` `machine/drawer_trim` | `kinds.golden` (`click/drawer_trim: activate=true screens=-`) | `drawer_test.go` (sets joined through trims) | `TestForeignKindsRoundTrip` |
| tinker table | `look.golden` `machine/tinker_table` | `kinds.golden` (`screens=tinker.ShowTable`) | `tinker_test.go` `TestTinkerTable` | `TestForeignKindsRoundTrip` |
| essence collector | `look.golden` `machine/essence_collector`, `state/` lines | `kinds.golden` (`click/`, `info/essence_collector`, `info/essence_collector/zombie`) | `dimension_test.go` `TestCollectors` | `TestForeignKindsRoundTrip` (monster and outputs) |
| relic table | `look.golden` `machine/relic_table` | `kinds.golden` | `dimension_test.go` `TestRelicTable` | `TestForeignKindsRoundTrip` |
| the nine network kinds | `look.golden` `machine/net_…`, `machine/disk_drive`, `machine/pattern_encoder` | `kinds.golden` (`click/` standing and sneaking, `slots/`: the encoder's 13 inputs and its output, disks, patterns, upgrades) | the 60 tests of `netstore/*_test.go`, `terminal_test.go`, `netflush_test.go`, `machine/netkinds_test.go`, `machine/netshape_test.go`, the cable scenes of `handler-blocks.golden` and `handler-use.golden` | `TestForeignKindsRoundTrip` (slots, facing), `machine/netkinds_test.go` `TestDriveAcceptsOnlyDisks` (disk ids), `netstore/migrate_disks_test.go` |

Saves name a kind by its string: the block and the item are `skygear:<kind>`, the block entity id is `SkyGearMachine`, and `DecodeNBT` rebuilds the state with `New(m.Kind)` (`machine.go:1403-1556`). No name changes and the only slot layout that moves (the encoder's) moves as data with the same values, so no migration patch is needed. `persist_test.go` (`TestSieveProgressSurvivesReload`) holds the world round trip of a machine.

Other counts: 36 kinds; 16 foreign; 19 hook assignments in 11 `init` functions at HEAD, none after T84; 145 lines in `kinds.golden`; 93 root test files at HEAD, 95 after this part.

## 6. Order and what was run

T75 and T76 touch disjoint files and may run at the same time; T82 may run at any time before T83; the others run in id order (each edits `machine/machine.go`). The prototype applied the tasks cumulatively in id order and ran `go build ./...`, `go vet ./...`, `gofmt -l` and `go test ./...` after each of T75, T76 (kernel package), T77, T78, T79, T80, T81, T82, T83, T84, T85 and T86, first on bb72e2e and again, all twelve states, on 9f6571a: 51 packages pass and no golden changes in every state. The last state was run once more on a31f0d6 (the suite, the pin and guard tests in three processes, `TestPinKinds` under `-race`, the benchmarks). The check and the pin were shown to bite on the last state, and the change was undone: without the trim's registration the root's test binary does not start (`panic: core: Install: feature "machine": machine kinds without behaviour: drawer_trim (feature storage); …`), and with the trim's `Click` returning false `TestPinKinds` fails on `click/drawer_trim` (`screens=machine.ShowMachine`). `TestTipNameColour` (T82) passes on the unchanged code. One state failed on the way and shaped two tasks: after T79 as first written, `netstore`'s `TestExternalDrawerController` failed (the storage's kinds had no behaviour in that test binary), which is why T79 owns `netstore/keys_test.go`; and the first form of T83 (tinker importing machine) changed the client goldens, which is decision 6.

## 7. Code by task

A `go` block is the whole of a new file. A `diff` block is the change to an existing file as a unified diff against the state the previous task left (lines starting with a space are context and stay as they are). Tasks apply in id order.
### T75

Tests only. `content/skygear/testdata/pin/kinds.golden` is recorded by step 2 of the task (145 lines); it is not copied from here.

`content/skygear/pin_bench_test.go`

```diff
--- a/content/skygear/pin_bench_test.go
+++ b/content/skygear/pin_bench_test.go
@@ -140,3 +140,27 @@
 		}
 	})
 }
+
+// BenchmarkTickHarvester ticks a harvester with no crop in reach: the work of
+// its kind is the farm's, reached through the kind's definition.
+func BenchmarkTickHarvester(b *testing.B) {
+	s := benchSession(b)
+	placeTestMachine(s, machinePos, machine.KindHarvester)
+	b.ReportAllocs()
+	b.ResetTimer()
+	for range b.N {
+		s.World.AdvanceTick()
+	}
+}
+
+// BenchmarkTickDrawerController ticks a drawer controller: a kind that only
+// stands there, so the tick is the machine frame's alone.
+func BenchmarkTickDrawerController(b *testing.B) {
+	s := benchSession(b)
+	placeTestMachine(s, machinePos, machine.KindDrawerController)
+	b.ReportAllocs()
+	b.ResetTimer()
+	for range b.N {
+		s.World.AdvanceTick()
+	}
+}
```

`content/skygear/pin_kinds_test.go` (new)

```go
package skygear

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/netstore"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// pinScreens runs call and returns "screens=" with the functions that asked
// the UI for a screen while it ran (the innermost one outside common/ui, as
// TestPinUse names them), or "screens=-".
func pinScreens(call func()) string {
	var shown []string
	saved := ui.InLobby
	ui.InLobby = func(*player.Player) bool {
		pcs := make([]uintptr, 32)
		n := runtime.Callers(2, pcs)
		frames := runtime.CallersFrames(pcs[:n])
		viaScreen, name := false, ""
		for {
			f, more := frames.Next()
			if strings.HasSuffix(f.Function, "ui.showScreen") {
				viaScreen = true
			} else if viaScreen && name == "" && !strings.Contains(f.Function, "/common/ui.") {
				name = f.Function[strings.LastIndex(f.Function, "/skygear/")+len("/skygear/"):]
			}
			if !more {
				break
			}
		}
		if viaScreen {
			shown = append(shown, name)
		}
		return false
	}
	defer func() { ui.InLobby = saved }()
	call()
	if len(shown) == 0 {
		return "screens=-"
	}
	return "screens=" + strings.Join(shown, ",")
}

// pinKindSamples are the stacks TestPinKinds offers every kind's slots, by
// the name the golden shows.
func pinKindSamples() []struct {
	name string
	st   item.Stack
} {
	return []struct {
		name string
		st   item.Stack
	}{
		{"cobblestone", item.NewStack(block.Cobblestone{}, 1)},
		{"dirt", item.NewStack(block.Dirt{}, 1)},
		{"coal", item.NewStack(item.Coal{}, 1)},
		{"sapling", item.NewStack(block.Sapling{}, 1)},
		{"bonemeal", item.NewStack(item.BoneMeal{}, 1)},
		{"essence", item.NewStack(sgitem.Essence{Mob: sgitem.MobZombie}, 1)},
		{"disk", netstore.WithDiskID(item.NewStack(netstore.Disk{Tier: 1}, 1), "d1")},
		{"pattern", item.NewStack(netstore.Pattern{}, 1)},
		{"upgrade", item.NewStack(netstore.Upgrade{}, 1)},
	}
}

// TestPinKinds pins, for every machine kind in machine.Order, what the kind's
// definition and its owner's hooks decide outside the WAILA panel (which
// TestPinLook pins): what a click on the placed machine does, standing and
// sneaking (the result of Activate and the screens asked for); the info icon
// of its window; the role of each of its 27 window slots and which sample
// items its slots take. The golden was recorded from the code in which the
// owners set the hooks of machine.Kinds in their init and machine.Activate,
// machine.Tick and State.InfoItem named kinds of other features; it changes
// only when a kind is changed on purpose.
func TestPinKinds(t *testing.T) {
	s := testSession(t)
	island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
	loadAround(s)
	h := spawnPlayer(s)
	lines := map[string]string{}
	rec := func(name, line string) {
		t.Helper()
		if _, dup := lines[name]; dup {
			t.Fatalf("pin scenario %q recorded twice", name)
		}
		lines[name] = line
	}
	samples := pinKindSamples()
	at := cube.Pos{4, island.IslandY + 4, 4}
	for _, k := range machine.Order {
		for _, sneak := range []bool{false, true} {
			m := machine.New(k)
			withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
				tx.SetBlock(at, m, nil)
				p.SetHeldItems(item.Stack{}, item.Stack{})
				p.StopSneaking()
				if sneak {
					p.StartSneaking()
				}
				s.ForgetOpens()
				name := "click/" + k
				if sneak {
					name += "/sneak"
				}
				var took bool
				scr := pinScreens(func() { took = m.Activate(at, cube.FaceUp, tx, p, &item.UseContext{}) })
				rec(name, fmt.Sprintf("activate=%t %s", took, scr))
				p.StopSneaking()
				tx.SetBlock(at, nil, nil)
			})
		}

		st := machine.New(k).State
		info := st.InfoItem()
		rec("info/"+k, fmt.Sprintf("name=%q lore=%q", info.CustomName(), info.Lore()))

		roles := make([]byte, 27)
		for slot := range roles {
			roles[slot] = "-IORC"[st.Role(slot)]
		}
		var takes []string
		for _, sm := range samples {
			var slots []string
			for slot := range 27 {
				if st.SlotAccepts(slot, sm.st) {
					slots = append(slots, fmt.Sprint(slot))
				}
			}
			if len(slots) > 0 || st.Accepts(sm.st) {
				takes = append(takes, fmt.Sprintf("%s@%s/%t", sm.name, strings.Join(slots, ","), st.Accepts(sm.st)))
			}
		}
		rec("slots/"+k, fmt.Sprintf("roles=%s takes=%v", roles, takes))
	}
	// The essence collector's info icon names the monster it receives.
	st := machine.New(machine.KindCollector).State
	st.SetMob(sgitem.MobZombie)
	info, _ := st.Inv().Item(26)
	rec("info/"+machine.KindCollector+"/zombie", fmt.Sprintf("name=%q lore=%q", info.CustomName(), info.Lore()))

	pinGolden(t, "testdata/pin/kinds.golden", lines)
}

// TestForeignKindsRoundTrip: the saved state of every machine kind another
// feature owns survives EncodeNBT and DecodeNBT: the items of every slot with
// a role, the monster and charges, the energy, the progress and the facing.
// The kinds are saved by name ("skygear:<kind>" with the block entity id
// SkyGearMachine), so a kind keeps its saves as long as its name and its slot
// layout stay.
func TestForeignKindsRoundTrip(t *testing.T) {
	kinds := []string{
		machine.KindHarvester,
		machine.KindDrawerController, machine.KindDrawerTrim, machine.KindDrawerInterface,
		machine.KindTinker,
		machine.KindCollector, machine.KindRelicTable,
		machine.KindNetController, machine.KindNetCable, machine.KindDiskDrive, machine.KindNetTerminal,
		machine.KindNetImporter, machine.KindNetExporter, machine.KindNetExternal,
		machine.KindPatternEncoder, machine.KindNetAutocrafter,
	}
	for _, k := range kinds {
		m := machine.New(k)
		st := m.State
		filled := 0
		for slot := range 27 {
			if st.Role(slot) != machine.RoleNone {
				_ = st.Inv().SetItem(slot, item.NewStack(block.Dirt{}, slot+1))
				filled++
			}
		}
		st.Mob, st.Charges, st.Energy, st.Progress, st.Facing = sgitem.MobZombie, 2, 7, 1, cube.FaceEast
		saved := m.EncodeNBT()
		back := machine.Machine{Kind: k}.DecodeNBT(saved).(machine.Machine)
		if again := back.EncodeNBT(); !reflect.DeepEqual(again, saved) {
			t.Errorf("%s: saved %v, saved again %v", k, saved, again)
		}
		bst := back.State
		if bst.Mob != sgitem.MobZombie || bst.Charges != 2 || bst.Energy != 7 || bst.Progress != 1 {
			t.Errorf("%s: mob %q charges %d energy %d progress %d", k, bst.Mob, bst.Charges, bst.Energy, bst.Progress)
		}
		got := 0
		for slot := range 27 {
			if bst.Role(slot) == machine.RoleNone {
				continue
			}
			it, _ := bst.Inv().Item(slot)
			if it.Count() != slot+1 {
				t.Errorf("%s: slot %d holds %d, want %d", k, slot, it.Count(), slot+1)
			}
			got++
		}
		if got != filled {
			t.Errorf("%s: %d slots with a role after the reload, %d before", k, got, filled)
		}
	}
}
```

### T76

The comment of `Commands` above `Setup` is not part of this change.

`content/skygear/core/feature.go`

```diff
--- a/content/skygear/core/feature.go
+++ b/content/skygear/core/feature.go
@@ -37,6 +37,10 @@
 	// Setup subscribes the feature's steps to the hooks. It runs once, during
 	// Install, before any event is emitted.
 	Setup func()
+	// Check, if set, runs once at the end of Install, after the Setup of every
+	// feature: it returns an error when something another feature had to
+	// provide during its Setup is missing. Install panics with it.
+	Check func() error
 }
 
 var (
@@ -47,9 +51,10 @@
 
 // Install installs fs once per process. For each feature in the given order it
 // registers the settings (account.RegisterSetting) and runs Setup with the
-// feature's name as the owner of every subscriber it adds. A second call, an
-// empty or repeated feature name, or a nil feature panics before anything of
-// that call is installed.
+// feature's name as the owner of every subscriber it adds. When every Setup
+// has run it calls each feature's Check in the same order and panics with the
+// first error. A second call, an empty or repeated feature name, or a nil
+// feature panics before anything of that call is installed.
 func Install(fs ...*Feature) {
 	featuresMu.Lock()
 	defer featuresMu.Unlock()
@@ -81,6 +86,14 @@
 			installing = ""
 		}
 	}
+	for _, f := range fs {
+		if f.Check == nil {
+			continue
+		}
+		if err := f.Check(); err != nil {
+			panic(fmt.Sprintf("core: Install: feature %q: %v", f.Name, err))
+		}
+	}
 }
 
 // Features returns the installed features in installation order.
```

`content/skygear/core/feature_test.go`

```diff
--- a/content/skygear/core/feature_test.go
+++ b/content/skygear/core/feature_test.go
@@ -1,6 +1,7 @@
 package core
 
 import (
+	"errors"
 	"reflect"
 	"slices"
 	"strings"
@@ -52,6 +53,30 @@
 	Install(&Feature{Name: "b"})
 }
 
+// TestInstallRunsChecksAfterEverySetup: a feature's Check runs when the Setup
+// of every feature has run, in list order, so it sees what a feature listed
+// after it provided.
+func TestInstallRunsChecksAfterEverySetup(t *testing.T) {
+	ResetHooks()
+	var got []string
+	Install(
+		&Feature{Name: "a", Setup: func() { got = append(got, "setup a") }, Check: func() error { got = append(got, "check a"); return nil }},
+		&Feature{Name: "b", Setup: func() { got = append(got, "setup b") }},
+		&Feature{Name: "c", Check: func() error { got = append(got, "check c"); return nil }},
+	)
+	if want := []string{"setup a", "setup b", "check a", "check c"}; !slices.Equal(got, want) {
+		t.Fatalf("got %v, want %v", got, want)
+	}
+}
+
+// TestInstallCheckErrorPanics: Install panics with the failing feature's name
+// and the error of its Check.
+func TestInstallCheckErrorPanics(t *testing.T) {
+	ResetHooks()
+	defer mustPanic(t, `"a"`, "kind x has no behaviour")
+	Install(&Feature{Name: "a", Check: func() error { return errors.New("kind x has no behaviour") }})
+}
+
 func TestDuplicateFeatureNamePanics(t *testing.T) {
 	ResetHooks()
 	ran := 0
```

### T77

The seven hook fields move from `Def` into `Behaviour`, which `Def` embeds: `machine.Kinds[k].Click = …` in the owners and `st.Def.Run` in the frame compile unchanged. No kind has an `Owner` after this task.

`content/skygear/kinds_test.go` (new)

```go
package skygear

import (
	"testing"

	"github.com/presentkim/skygear/content/skygear/machine"
)

// kindOwners are the machine kinds whose behaviour another feature supplies
// (machine.Register), by kind: the feature named is the kind's Owner. A kind
// that is not here is the machine feature's own.
var kindOwners = map[string]string{}

// TestKindOwners: every kind of machine.Order has the owner kindOwners says,
// each owner is a listed feature, and every owned kind was registered by the
// time the features were installed (machine.CheckKinds, which core.Install
// ran as the machine feature's Check).
func TestKindOwners(t *testing.T) {
	listed := map[string]bool{}
	for _, f := range Features {
		listed[f.Name] = true
	}
	for _, k := range machine.Order {
		d := machine.Kinds[k]
		if d.Owner != kindOwners[k] {
			t.Errorf("kind %s has the owner %q, kindOwners says %q", k, d.Owner, kindOwners[k])
		}
		if d.Owner != "" && !listed[d.Owner] {
			t.Errorf("kind %s is owned by %q, which is not a listed feature", k, d.Owner)
		}
	}
	for k := range kindOwners {
		if machine.Kinds[k] == nil {
			t.Errorf("kindOwners names %s, which is not a machine kind", k)
		}
	}
	if err := machine.CheckKinds(); err != nil {
		t.Error(err)
	}
	if machine.Feature.Check == nil {
		t.Error("the machine feature has no Check: an unregistered kind would not stop the start")
	}
}
```

`content/skygear/machine/feature.go`

```diff
--- a/content/skygear/machine/feature.go
+++ b/content/skygear/machine/feature.go
@@ -6,4 +6,4 @@
 )
 
 // Feature is the machine feature, listed in the game's feature list.
-var Feature = &core.Feature{Name: "machine", Entities: []world.EntityType{RangeBoxType}, Creative: creativeItems, Commands: commands, Setup: subscribe}
+var Feature = &core.Feature{Name: "machine", Entities: []world.EntityType{RangeBoxType}, Creative: creativeItems, Commands: commands, Setup: subscribe, Check: CheckKinds}
```

`content/skygear/machine/kinds.go` (new)

```go
package machine

import (
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
)

// The kinds of other features. A kind's name, its place in Order, its block
// and item and its Def are declared here, so the registration order of the
// blocks and items does not depend on which features are installed. What a
// kind of another feature does is that feature's: its Def names the feature
// as Owner, and the feature hands the kind's Behaviour to Register from its
// Setup. CheckKinds, the machine feature's Check, fails the installation when
// an owner did not.

// Behaviour is what the feature that owns a kind supplies. Every field may be
// nil: the machine frame then does what it does for its own kinds.
type Behaviour struct {
	// Click is what a click on the placed machine does instead of opening
	// the machine window; it returns true if it took the click.
	Click func(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool
	// Run is the work of one run of the machine's tick.
	Run func(s *core.Session, st *State, pos cube.Pos, tx *world.Tx)
	// Insert, Peek and Extract replace the item I/O of the block (hoppers and
	// item conduits).
	Insert  func(tx *world.Tx, pos cube.Pos, st item.Stack) int
	Peek    func(tx *world.Tx, pos cube.Pos, want func(item.Stack) bool) item.Stack
	Extract func(tx *world.Tx, pos cube.Pos, st item.Stack, n int) item.Stack
	// Accepts says which items an input slot takes (slot is -1 for "any input
	// slot"); it replaces the machine's own rule, so a kind can hold items
	// with stored values (disks, patterns).
	Accepts func(slot int, st item.Stack) bool
	// Lines are the kind's panel lines in place of the machine's own
	// (Machine.Waila); NoLines is a kind without lines.
	Lines func(lang string, tx *world.Tx, pos cube.Pos) []string
}

// NoLines is the Lines of a kind whose panel has no lines.
func NoLines(string, *world.Tx, cube.Pos) []string { return nil }

// kindsChecked is set once CheckKinds found every kind registered: the
// features are installed and no kind changes any more.
var kindsChecked bool

// Register sets the behaviour of a kind another feature owns. The owner calls
// it once per kind from its Setup. It panics for a name that is no kind, for
// a kind without an Owner, for a second registration and after CheckKinds
// passed.
func Register(kind string, b Behaviour) {
	d, ok := Kinds[kind]
	switch {
	case !ok:
		panic(fmt.Sprintf("machine: Register: %q is not a machine kind", kind))
	case d.Owner == "":
		panic(fmt.Sprintf("machine: Register: kind %q has no owner: its behaviour is the machine feature's", kind))
	case d.registered:
		panic(fmt.Sprintf("machine: Register: kind %q registered twice", kind))
	case kindsChecked:
		panic(fmt.Sprintf("machine: Register: kind %q registered after the features were installed", kind))
	}
	d.Behaviour, d.registered = b, true
}

// CheckKinds reports the kinds whose owner did not register their behaviour.
// It is the machine feature's Check: core.Install calls it when the Setup of
// every feature has run and panics with its error, so a game that lists a
// kind's owner too late, or not at all, does not start.
func CheckKinds() error {
	var missing []string
	for _, k := range Order {
		if d := Kinds[k]; d.Owner != "" && !d.registered {
			missing = append(missing, fmt.Sprintf("%s (feature %s)", k, d.Owner))
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("machine kinds without behaviour: %s; list the feature in the game's features (its Setup calls machine.Register)", strings.Join(missing, ", "))
	}
	kindsChecked = true
	return nil
}
```

`content/skygear/machine/kinds_test.go` (new)

```go
package machine

import (
	"fmt"
	"strings"
	"testing"
)

// ownedKind makes KindRelicTable a kind of the feature "dimension" that is
// not registered yet, for the length of the test.
func ownedKind(t *testing.T) *Def {
	t.Helper()
	d := Kinds[KindRelicTable]
	saved, checked := *d, kindsChecked
	t.Cleanup(func() { *d, kindsChecked = saved, checked })
	d.Owner, d.registered, d.Behaviour = "dimension", false, Behaviour{}
	kindsChecked = false
	return d
}

func panicOf(f func()) (msg string) {
	defer func() {
		if r := recover(); r != nil {
			msg = fmt.Sprint(r)
		}
	}()
	f()
	return ""
}

// TestCheckKindsNamesTheMissingOwner: a kind whose owner did not register it
// fails the check with the kind and the feature in the message, and passes
// once the owner registered it.
func TestCheckKindsNamesTheMissingOwner(t *testing.T) {
	d := ownedKind(t)
	err := CheckKinds()
	if err == nil || !strings.Contains(err.Error(), "relic_table (feature dimension)") {
		t.Fatalf("check of an unregistered kind: %v", err)
	}
	if kindsChecked {
		t.Fatal("a failed check closed the registry")
	}
	Register(KindRelicTable, Behaviour{Lines: NoLines})
	if !d.registered || d.Lines == nil {
		t.Fatal("Register did not set the behaviour")
	}
	if err := CheckKinds(); err != nil && strings.Contains(err.Error(), "relic_table") {
		t.Fatalf("check after the registration: %v", err)
	}
}

// TestRegisterPanics: Register refuses a name that is no kind, a kind of the
// machine feature itself, a second registration and one after the check
// passed.
func TestRegisterPanics(t *testing.T) {
	ownedKind(t)
	if msg := panicOf(func() { Register("no_such_kind", Behaviour{}) }); !strings.Contains(msg, "not a machine kind") {
		t.Errorf("unknown kind: %q", msg)
	}
	if msg := panicOf(func() { Register(KindCrusher, Behaviour{}) }); !strings.Contains(msg, "has no owner") {
		t.Errorf("the machine feature's own kind: %q", msg)
	}
	Register(KindRelicTable, Behaviour{})
	if msg := panicOf(func() { Register(KindRelicTable, Behaviour{}) }); !strings.Contains(msg, "twice") {
		t.Errorf("second registration: %q", msg)
	}
	Kinds[KindRelicTable].registered = false
	kindsChecked = true
	if msg := panicOf(func() { Register(KindRelicTable, Behaviour{}) }); !strings.Contains(msg, "after the features were installed") {
		t.Errorf("registration after the check: %q", msg)
	}
}

// TestOrderListsEveryKind: Order and Kinds hold the same kinds, each once.
func TestOrderListsEveryKind(t *testing.T) {
	seen := map[string]bool{}
	for _, k := range Order {
		if Kinds[k] == nil {
			t.Errorf("%s is in Order but not in Kinds", k)
		}
		if seen[k] {
			t.Errorf("%s is in Order twice", k)
		}
		seen[k] = true
	}
	if len(seen) != len(Kinds) {
		t.Errorf("Order has %d kinds, Kinds %d", len(seen), len(Kinds))
	}
}
```

`content/skygear/machine/machine.go`

```diff
--- a/content/skygear/machine/machine.go
+++ b/content/skygear/machine/machine.go
@@ -279,21 +279,14 @@
 	Charge    []int
 	autoSieve bool
 
-	// The kinds whose behaviour is another package's set these (Register):
-	// click is what a click does instead of opening the window, true if
-	// it took the click; run is the work of one run; insert, peek and extract
-	// replace the item I/O of the block.
-	Click   func(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool
-	Run     func(s *core.Session, st *State, pos cube.Pos, tx *world.Tx)
-	Insert  func(tx *world.Tx, pos cube.Pos, st item.Stack) int
-	Peek    func(tx *world.Tx, pos cube.Pos, want func(item.Stack) bool) item.Stack
-	Extract func(tx *world.Tx, pos cube.Pos, st item.Stack, n int) item.Stack
-	// Accepts, if set, says which items an input slot takes (slot is -1 for
-	// "any input slot"); it replaces the machine's own rule, so the network
-	// kinds can hold items with stored values (disks, patterns).
-	Accepts func(slot int, st item.Stack) bool
-	// Lines, if set, are the kind's panel lines in place of the machine's own (Machine.Waila).
-	Lines func(lang string, tx *world.Tx, pos cube.Pos) []string
+	// Owner is the feature that supplies the kind's behaviour with Register
+	// ("" for a kind whose behaviour is this package's); registered is set
+	// once it did (kinds.go).
+	Owner      string
+	registered bool
+	// Behaviour is what the owner registered: the zero value until then and
+	// for the machine feature's own kinds.
+	Behaviour
 }
 
 var Kinds = map[string]*Def{
```

### T78

Fourteen entries of `Kinds` get a flag: `NoDisplay` on twelve, `NoWindow` on the cable, `Cycle` on the harvester.

`content/skygear/machine/kinds_test.go`

```diff
--- a/content/skygear/machine/kinds_test.go
+++ b/content/skygear/machine/kinds_test.go
@@ -87,3 +87,31 @@
 		t.Errorf("Order has %d kinds, Kinds %d", len(seen), len(Kinds))
 	}
 }
+
+// TestFrameFlags: the kinds without a display to keep, without a window and
+// with a turning cycle are the ones the frame named by id before the flags.
+func TestFrameFlags(t *testing.T) {
+	want := map[string][]string{
+		"NoDisplay": {KindEnergyCell, KindTank, KindCrucible, KindLavaGenerator, KindSolarGenerator, KindHydroGenerator, KindDrawerController, KindDrawerTrim, KindDrawerInterface, KindTinker, KindCollector, KindRelicTable},
+		"NoWindow":  {KindNetCable},
+		"Cycle":     {KindHarvester},
+	}
+	for flag, kinds := range want {
+		in := map[string]bool{}
+		for _, k := range kinds {
+			in[k] = true
+		}
+		for _, k := range Order {
+			d := Kinds[k]
+			got := map[string]bool{"NoDisplay": d.NoDisplay, "NoWindow": d.NoWindow, "Cycle": d.Cycle}[flag]
+			if got != in[k] {
+				t.Errorf("%s: %s is %t, want %t", k, flag, got, in[k])
+			}
+		}
+	}
+	for _, k := range Order {
+		if d := Kinds[k]; d.Cell && !d.NoDisplay {
+			t.Errorf("%s stores energy but has a display: its tick would not offer the energy to the network", k)
+		}
+	}
+}
```

`content/skygear/machine/look.go`

```diff
--- a/content/skygear/machine/look.go
+++ b/content/skygear/machine/look.go
@@ -118,7 +118,7 @@
 			return lines, nil
 		}
 		return lines[1:], itemLine(lang, item.NewStack(st.Lab.Out, st.Lab.OutCount))
-	case st.Def == Kinds[KindHarvester]:
+	case st.Def.Cycle:
 		lines = append(lines, i18n.T(lang, "look.machine.harvestCycle")+gauge)
 	case st.Def == Kinds[KindGenerator]:
 		return GeneratorLines(lang, st), nil
@@ -126,7 +126,8 @@
 		return NatureLines(lang, st), nil
 	case m.Kind == KindDrawerTrim, m.Kind == KindTinker, m.Kind == KindRelicTable:
 		return nil, nil
-	case st.Def.Controller:
+	case st.Def.Lines != nil:
+		// the kind's owner describes it (Machine.Waila)
 		return nil, nil
 	case st.Def.FluidCap > 0:
 		lines := FluidLines(lang, st)
```

`content/skygear/machine/machine.go`

```diff
--- a/content/skygear/machine/machine.go
+++ b/content/skygear/machine/machine.go
@@ -278,6 +278,13 @@
 	// marks the sieves that sift on their own (sieve.go).
 	Charge    []int
 	autoSieve bool
+	// NoDisplay marks a kind with no machine window to keep current: its tick
+	// ends before the window's progress, energy gauge and info icon are
+	// brought up to date. NoWindow marks a kind a click does nothing on (the
+	// click goes on to the held item). Cycle marks a kind that works in a
+	// cycle that turns whether or not there is work: its window and its panel
+	// show the cycle, not an input.
+	NoDisplay, NoWindow, Cycle bool
 
 	// Owner is the feature that supplies the kind's behaviour with Register
 	// ("" for a kind whose behaviour is this package's); registered is set
@@ -322,7 +329,7 @@
 	},
 	KindHarvester: {
 		ID: idHarvester, nameKey: "machine.harvester.name", Marker: "§d§m§3", Interval: 40, Output: []int{13, 14, 15, 16}, Indicator: 10,
-		Use: 10, EnergyCap: ConsumerBuffer,
+		Use: 10, EnergyCap: ConsumerBuffer, Cycle: true,
 		helpKeys: []string{"machine.harvester.help.1", "machine.harvester.help.2"},
 	},
 	KindGenerator: {
@@ -336,28 +343,28 @@
 		helpKeys: []string{"machine.alloy.help.1", "machine.alloy.help.2"},
 	},
 	KindDrawerController: {
-		ID: idDrawerController, nameKey: "machine.controller.name", Interval: 1, Indicator: 10, Controller: true,
+		ID: idDrawerController, nameKey: "machine.controller.name", Interval: 1, Indicator: 10, Controller: true, NoDisplay: true,
 		helpKeys: []string{"machine.controller.help.1", "machine.controller.help.2"},
 	},
 	KindDrawerTrim: {
-		ID: idDrawerTrim, nameKey: "machine.link.name", Interval: 1, Indicator: 10,
+		ID: idDrawerTrim, nameKey: "machine.link.name", Interval: 1, Indicator: 10, NoDisplay: true,
 		helpKeys: []string{"machine.link.help.1"},
 	},
 	KindDrawerInterface: {
-		ID: idDrawerInterface, nameKey: "machine.drawer_interface.name", Interval: 1, Indicator: 10, Controller: true,
+		ID: idDrawerInterface, nameKey: "machine.drawer_interface.name", Interval: 1, Indicator: 10, Controller: true, NoDisplay: true,
 		helpKeys: []string{"machine.drawer_interface.help.1", "machine.drawer_interface.help.2"},
 	},
 	KindTinker: {
-		ID: idTinker, nameKey: "machine.tinker.name", Interval: 1, Indicator: 10,
+		ID: idTinker, nameKey: "machine.tinker.name", Interval: 1, Indicator: 10, NoDisplay: true,
 		helpKeys: []string{"machine.tinker.help.1", "machine.tinker.help.2"},
 	},
 	KindCollector: {
 		// The harvester's window: four output slots, no energy.
-		ID: idCollector, nameKey: "machine.collector.name", Marker: "§d§m§3", Interval: 1, Output: []int{13, 14, 15, 16}, Indicator: 10,
+		ID: idCollector, nameKey: "machine.collector.name", Marker: "§d§m§3", Interval: 1, Output: []int{13, 14, 15, 16}, Indicator: 10, NoDisplay: true,
 		helpKeys: []string{"machine.collector.help.1", "machine.collector.help.2"},
 	},
 	KindRelicTable: {
-		ID: idRelicTable, nameKey: "machine.relictable.name", Interval: 1, Indicator: 10,
+		ID: idRelicTable, nameKey: "machine.relictable.name", Interval: 1, Indicator: 10, NoDisplay: true,
 		helpKeys: []string{"machine.relictable.help.1"},
 	},
 	KindNetController: {
@@ -366,7 +373,7 @@
 		helpKeys: []string{"machine.net_controller.help.1"},
 	},
 	KindNetCable: {
-		ID: idNetCable, nameKey: "machine.net_cable.name", Interval: 1, Indicator: 10,
+		ID: idNetCable, nameKey: "machine.net_cable.name", Interval: 1, Indicator: 10, NoWindow: true,
 		helpKeys: []string{"machine.net_cable.help.1"},
 	},
 	KindDiskDrive: {
@@ -406,29 +413,29 @@
 		helpKeys: []string{"machine.washer.help.1", "machine.washer.help.2"},
 	},
 	KindTank: {
-		ID: idTank, nameKey: "machine.tank.name", Interval: 1, Indicator: 10,
+		ID: idTank, nameKey: "machine.tank.name", Interval: 1, Indicator: 10, NoDisplay: true,
 		FluidCap: tankCapacity, fluidSource: true, fluidSink: true,
 		helpKeys: []string{"machine.tank.help.1", "machine.tank.help.2"},
 	},
 	KindCrucible: {
-		ID: idCrucible, nameKey: "machine.crucible.name", Interval: 20, Input: []int{10}, Indicator: 12,
+		ID: idCrucible, nameKey: "machine.crucible.name", Interval: 20, Input: []int{10}, Indicator: 12, NoDisplay: true,
 		Use: 30, EnergyCap: ConsumerBuffer,
 		FluidCap: 8000, fluidSource: true, fluidOnly: fluidnet.Lava,
 		helpKeys: []string{"machine.crucible.help.1", "machine.crucible.help.2"},
 	},
 	KindLavaGenerator: {
-		ID: idLavaGenerator, nameKey: "machine.lavagen.name", Interval: 1, Indicator: 10,
+		ID: idLavaGenerator, nameKey: "machine.lavagen.name", Interval: 1, Indicator: 10, NoDisplay: true,
 		Generates: LavaGenOutput, EnergyCap: generatorBuffer,
 		FluidCap: 4000, fluidSink: true, fluidOnly: fluidnet.Lava,
 		helpKeys: []string{"machine.lavagen.help.1", "machine.lavagen.help.2"},
 	},
 	KindSolarGenerator: {
-		ID: idSolarGenerator, nameKey: "machine.solargen.name", Interval: 1, Indicator: 10,
+		ID: idSolarGenerator, nameKey: "machine.solargen.name", Interval: 1, Indicator: 10, NoDisplay: true,
 		Generates: SolarOutput, EnergyCap: generatorBuffer,
 		helpKeys: []string{"machine.solargen.help.1", "machine.solargen.help.2"},
 	},
 	KindHydroGenerator: {
-		ID: idHydroGenerator, nameKey: "machine.hydrogen.name", Interval: 1, Indicator: 10,
+		ID: idHydroGenerator, nameKey: "machine.hydrogen.name", Interval: 1, Indicator: 10, NoDisplay: true,
 		Generates: HydroOutput, EnergyCap: generatorBuffer,
 		helpKeys: []string{"machine.hydrogen.help.1", "machine.hydrogen.help.2"},
 	},
@@ -438,7 +445,7 @@
 		helpKeys: []string{"machine.simulator.help.1", "machine.simulator.help.2"},
 	},
 	KindEnergyCell: {
-		ID: idEnergyCell, nameKey: "machine.cell.name", Interval: 1, Indicator: 10,
+		ID: idEnergyCell, nameKey: "machine.cell.name", Interval: 1, Indicator: 10, NoDisplay: true,
 		Cell: true, EnergyCap: CellCapacity,
 		helpKeys: []string{"machine.cell.help.1", "machine.cell.help.2"},
 	},
@@ -1132,13 +1139,14 @@
 // shows the fuel left in the one burning), and the text key of its tooltip,
 // which takes the percent when withPct is set.
 func (st *State) progressLook() (ring bool, key string, withPct bool) {
+	if st.Def.Cycle {
+		return true, "machine.progress.harvestCycle", false
+	}
 	switch st.Def.ID {
 	case idMiner, idAutoSieve, idMagicSieve, idSimulator, idWasher, idVacuum, idFeller, idCrafter, idLab:
 		return true, "machine.progress.mining", true
 	case idGenerator:
 		return true, "machine.progress.fuel", true
-	case idHarvester:
-		return true, "machine.progress.harvestCycle", false
 	}
 	return false, "machine.progress.percent", true
 }
@@ -1596,7 +1604,7 @@
 			return true
 		}
 	}
-	if m.State.Def.Cell || m.Kind == KindNetCable {
+	if m.State.Def.Cell || m.State.Def.NoWindow {
 		return false
 	}
 	if !s.Debounced(p) {
@@ -1721,9 +1729,8 @@
 		watchMachine(s, tx, pos, st)
 	}
 	st.showRange(tx, pos)
-	switch st.Def.ID {
-	case idEnergyCell, idTank, idCrucible, idLavaGenerator, idSolarGenerator, idHydroGenerator, idDrawerController, idDrawerTrim, idDrawerInterface, idTinker, idCollector, idRelicTable:
-		if st.Def.ID == idEnergyCell && st.Energy > 0 {
+	if st.Def.NoDisplay {
+		if st.Def.Cell && st.Energy > 0 {
 			network.Energy(s).AddSource(pos)
 		}
 		return
```

`content/skygear/machine/tier.go`

```diff
--- a/content/skygear/machine/tier.go
+++ b/content/skygear/machine/tier.go
@@ -155,14 +155,18 @@
 // moved: its area is one block high, so it has no reach of its own to go by.
 const harvesterLift = 2
 
+// rangeLifts are the kinds whose area moves up and down by a fixed number of
+// blocks instead of by its own height.
+var rangeLifts = map[string]int{KindHarvester: harvesterLift}
+
 // MaxOffset is how far the area of a machine of kind may be moved each way
 // on each axis: as far as its tier reaches sideways, and as far as it reaches
 // below or above.
 func (st *State) MaxOffset(kind string) cube.Pos {
 	most := st.MaxArea(kind)
 	y := max(most.Down, most.Up)
-	if kind == KindHarvester {
-		y = harvesterLift
+	if lift, ok := rangeLifts[kind]; ok {
+		y = lift
 	}
 	return cube.Pos{most.X, y, most.Z}
 }
```

### T79

The `init` at the end of `storage/storage.go` goes with its comment and the blank line before it; the file's imports stay as they are (all are still used).

`content/skygear/kinds_test.go`

```diff
--- a/content/skygear/kinds_test.go
+++ b/content/skygear/kinds_test.go
@@ -9,7 +9,9 @@
 // kindOwners are the machine kinds whose behaviour another feature supplies
 // (machine.Register), by kind: the feature named is the kind's Owner. A kind
 // that is not here is the machine feature's own.
-var kindOwners = map[string]string{}
+var kindOwners = map[string]string{
+	machine.KindDrawerController: "storage", machine.KindDrawerTrim: "storage", machine.KindDrawerInterface: "storage",
+}
 
 // TestKindOwners: every kind of machine.Order has the owner kindOwners says,
 // each owner is a listed feature, and every owned kind was registered by the
```

`content/skygear/machine/look.go`

```diff
--- a/content/skygear/machine/look.go
+++ b/content/skygear/machine/look.go
@@ -124,7 +124,7 @@
 		return GeneratorLines(lang, st), nil
 	case IsNatureGenerator(m.Kind):
 		return NatureLines(lang, st), nil
-	case m.Kind == KindDrawerTrim, m.Kind == KindTinker, m.Kind == KindRelicTable:
+	case m.Kind == KindTinker, m.Kind == KindRelicTable:
 		return nil, nil
 	case st.Def.Lines != nil:
 		// the kind's owner describes it (Machine.Waila)
@@ -195,11 +195,10 @@
 }
 
 // outputContainer is the output slots of the machine st as a chest's slots,
-// each item with its count under it, or nil for a machine with no output or
-// a drawer controller (whose lines count the drawers instead), and nil while
-// the output is empty.
+// each item with its count under it, or nil for a machine with no output, and
+// nil while the output is empty.
 func outputContainer(st *State) *waila.Container {
-	if st == nil || st.Def.Controller || len(st.Def.Output) == 0 {
+	if st == nil || len(st.Def.Output) == 0 {
 		return nil
 	}
 	var items []item.Stack
```

`content/skygear/machine/machine.go`

```diff
--- a/content/skygear/machine/machine.go
+++ b/content/skygear/machine/machine.go
@@ -262,9 +262,6 @@
 	FluidCap               int
 	fluidSource, fluidSink bool
 	fluidOnly              fluidnet.Kind
-	// Controller marks the drawer controller and the drawer interface, which
-	// show the controller's storage (storage.go).
-	Controller bool
 	// cell makes the machine an energy store: it takes energy from the network and
 	// gives it back, without working (energyCap is its capacity).
 	Cell bool
@@ -343,15 +340,15 @@
 		helpKeys: []string{"machine.alloy.help.1", "machine.alloy.help.2"},
 	},
 	KindDrawerController: {
-		ID: idDrawerController, nameKey: "machine.controller.name", Interval: 1, Indicator: 10, Controller: true, NoDisplay: true,
+		ID: idDrawerController, Owner: "storage", nameKey: "machine.controller.name", Interval: 1, Indicator: 10, NoDisplay: true,
 		helpKeys: []string{"machine.controller.help.1", "machine.controller.help.2"},
 	},
 	KindDrawerTrim: {
-		ID: idDrawerTrim, nameKey: "machine.link.name", Interval: 1, Indicator: 10, NoDisplay: true,
+		ID: idDrawerTrim, Owner: "storage", nameKey: "machine.link.name", Interval: 1, Indicator: 10, NoDisplay: true,
 		helpKeys: []string{"machine.link.help.1"},
 	},
 	KindDrawerInterface: {
-		ID: idDrawerInterface, nameKey: "machine.drawer_interface.name", Interval: 1, Indicator: 10, Controller: true, NoDisplay: true,
+		ID: idDrawerInterface, Owner: "storage", nameKey: "machine.drawer_interface.name", Interval: 1, Indicator: 10, NoDisplay: true,
 		helpKeys: []string{"machine.drawer_interface.help.1", "machine.drawer_interface.help.2"},
 	},
 	KindTinker: {
@@ -1577,9 +1574,6 @@
 	if c := m.State.Def.Click; c != nil && c(s, p, tx, pos) {
 		return true
 	}
-	if m.Kind == KindDrawerTrim {
-		return true
-	}
 	if m.Kind == KindTinker {
 		if !s.Debounced(p) {
 			tinker.ShowTable(s, p, tx, pos)
```

`content/skygear/netstore/keys_test.go`

```diff
--- a/content/skygear/netstore/keys_test.go
+++ b/content/skygear/netstore/keys_test.go
@@ -12,9 +12,16 @@
 	"github.com/df-mc/dragonfly/server/world"
 	"github.com/presentkim/skygear/common/storagenet"
 	"github.com/presentkim/skygear/content/skygear/core"
+	"github.com/presentkim/skygear/content/skygear/storage"
 )
 
-func init() { world.DefaultBlockRegistry.Finalize() }
+// The tests of this package run without the game's feature list: the storage
+// feature is set up here, so the drawer controller the external storage
+// reads has its behaviour (machine.Register).
+func init() {
+	world.DefaultBlockRegistry.Finalize()
+	storage.Feature.Setup()
+}
 
 func TestKeyOfCount(t *testing.T) {
 	a, b := item.NewStack(block.Stone{}, 5), item.NewStack(block.Stone{}, 1)
```

`content/skygear/storage/kinds.go` (new)

```go
package storage

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/machine"
)

// registerKinds hands the machine frame the behaviour of the storage's
// machine kinds. The drawer controller and the drawer interface stand for
// their drawer set: a click deposits into it, hoppers and conduits put into
// and take from it, and the panel counts it. The drawer trim only joins
// drawers: a click does nothing and its panel has no lines.
func registerKinds() {
	set := machine.Behaviour{
		Click: func(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool {
			UseController(s, p, tx, pos)
			return true
		},
		Insert: ControllerInsert, Peek: ControllerPeek, Extract: ControllerExtract,
		Lines: Lines,
	}
	machine.Register(machine.KindDrawerController, set)
	machine.Register(machine.KindDrawerInterface, set)
	machine.Register(machine.KindDrawerTrim, machine.Behaviour{
		Click: func(*core.Session, *player.Player, *world.Tx, cube.Pos) bool { return true },
		Lines: machine.NoLines,
	})
}
```

`content/skygear/storage/storage.go`

```diff
--- a/content/skygear/storage/storage.go
+++ b/content/skygear/storage/storage.go
@@ -447,17 +447,3 @@
 	}
 	return sorted
 }
-
-// The drawer controller and the drawer interface are machine kinds whose
-// clicks and item I/O are the storage's.
-func init() {
-	for _, k := range []string{machine.KindDrawerController, machine.KindDrawerInterface} {
-		def := machine.Kinds[k]
-		def.Click = func(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool {
-			UseController(s, p, tx, pos)
-			return true
-		}
-		def.Insert, def.Peek, def.Extract = ControllerInsert, ControllerPeek, ControllerExtract
-		def.Lines = Lines
-	}
-}
```

`content/skygear/storage/subscribe.go`

```diff
--- a/content/skygear/storage/subscribe.go
+++ b/content/skygear/storage/subscribe.go
@@ -10,6 +10,7 @@
 	subscribeDisplays()
 	subscribeBlocks()
 	subscribeSneakUse()
+	registerKinds()
 }
 
 // The drawer displays are removed when the save closes.
```

### T80

`kindOwners` is laid out by `gofmt`; copy the block as it is.

`content/skygear/farm/harvester.go`

```diff
--- a/content/skygear/farm/harvester.go
+++ b/content/skygear/farm/harvester.go
@@ -144,5 +144,8 @@
 	return block.Air{}, drops
 }
 
-// The harvester is a machine kind whose run is the farm's.
-func init() { machine.Kinds[machine.KindHarvester].Run = RunHarvester }
+// registerKinds hands the machine frame the work of the farm's machine kind,
+// the harvester.
+func registerKinds() {
+	machine.Register(machine.KindHarvester, machine.Behaviour{Run: RunHarvester})
+}
```

`content/skygear/farm/subscribe.go`

```diff
--- a/content/skygear/farm/subscribe.go
+++ b/content/skygear/farm/subscribe.go
@@ -19,6 +19,7 @@
 	subscribeUse()
 	registerComposterIO()
 	registerLook()
+	registerKinds()
 }
 
 // registerComposterIO lets itemio reach a composter (a vanilla block) through
```

`content/skygear/kinds_test.go`

```diff
--- a/content/skygear/kinds_test.go
+++ b/content/skygear/kinds_test.go
@@ -10,6 +10,7 @@
 // (machine.Register), by kind: the feature named is the kind's Owner. A kind
 // that is not here is the machine feature's own.
 var kindOwners = map[string]string{
+	machine.KindHarvester:        "farm",
 	machine.KindDrawerController: "storage", machine.KindDrawerTrim: "storage", machine.KindDrawerInterface: "storage",
 }
 
```

`content/skygear/machine/machine.go`

```diff
--- a/content/skygear/machine/machine.go
+++ b/content/skygear/machine/machine.go
@@ -325,7 +325,7 @@
 		helpKeys: []string{"machine.miner.help.1"},
 	},
 	KindHarvester: {
-		ID: idHarvester, nameKey: "machine.harvester.name", Marker: "§d§m§3", Interval: 40, Output: []int{13, 14, 15, 16}, Indicator: 10,
+		ID: idHarvester, Owner: "farm", nameKey: "machine.harvester.name", Marker: "§d§m§3", Interval: 40, Output: []int{13, 14, 15, 16}, Indicator: 10,
 		Use: 10, EnergyCap: ConsumerBuffer, Cycle: true,
 		helpKeys: []string{"machine.harvester.help.1", "machine.harvester.help.2"},
 	},
```

### T81

`dimension/relic.go` loses the imports `cube` and `machine` with the assignment.

`content/skygear/dimension/collector.go`

```diff
--- a/content/skygear/dimension/collector.go
+++ b/content/skygear/dimension/collector.go
@@ -227,13 +227,3 @@
 		Size:  "collector",
 	})
 }
-
-// A click opens the collector's window.
-func init() {
-	machine.Kinds[machine.KindCollector].Click = func(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool {
-		if !s.Debounced(p) {
-			ShowCollector(s, p, pos)
-		}
-		return true
-	}
-}
```

`content/skygear/dimension/kinds.go` (new)

```go
package dimension

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/wintext"
)

// registerKinds hands the machine frame the behaviour of the dimension's
// machine kinds: a click opens the essence collector's window, whose info
// icon names the monster it receives; a click opens the relic table, whose
// panel has no lines.
func registerKinds() {
	machine.Register(machine.KindCollector, machine.Behaviour{
		Click: func(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool {
			if !s.Debounced(p) {
				ShowCollector(s, p, pos)
			}
			return true
		},
		Info: collectorInfo,
	})
	machine.Register(machine.KindRelicTable, machine.Behaviour{
		Click: func(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool {
			if !s.Debounced(p) {
				ShowRelicTable(s, p, "")
			}
			return true
		},
		Lines: machine.NoLines,
	})
}

// collectorInfo is the line the collector's info icon adds: the monster it
// receives, or that none is chosen.
func collectorInfo(st *machine.State) []wintext.Text {
	if st.Mob != "" {
		return []wintext.Text{{wintext.Key("machine.collector.mob", wintext.Key("mob."+st.Mob))}}
	}
	return []wintext.Text{{wintext.Key("machine.collector.none")}}
}
```

`content/skygear/dimension/relic.go`

```diff
--- a/content/skygear/dimension/relic.go
+++ b/content/skygear/dimension/relic.go
@@ -5,7 +5,6 @@
 	"strings"
 
 	"github.com/df-mc/dragonfly/server/block"
-	"github.com/df-mc/dragonfly/server/block/cube"
 	"github.com/df-mc/dragonfly/server/item"
 	"github.com/df-mc/dragonfly/server/item/recipe"
 	"github.com/df-mc/dragonfly/server/player"
@@ -15,7 +14,6 @@
 	"github.com/presentkim/skygear/common/ui"
 	"github.com/presentkim/skygear/content/skygear/core"
 	"github.com/presentkim/skygear/content/skygear/formicon"
-	"github.com/presentkim/skygear/content/skygear/machine"
 	"github.com/presentkim/skygear/content/skygear/naming"
 	"github.com/presentkim/skygear/content/skygear/progress"
 	"github.com/presentkim/skygear/content/skygear/sgitem"
@@ -82,12 +80,6 @@
 		item.NewStack(sgitem.Ingot{Metal: sgitem.MetalSteel}, 1), item.NewStack(sgitem.Ingot{Metal: sgitem.MetalSteel}, 1),
 		item.NewStack(item.Stick{}, 1),
 	}, sgitem.RelicStack(sgitem.RelicBlade, 1), "crafting_table"))
-	machine.Kinds[machine.KindRelicTable].Click = func(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool {
-		if !s.Debounced(p) {
-			ShowRelicTable(s, p, "")
-		}
-		return true
-	}
 }
 
 // relicKinds are every relic, the weapons first.
```

`content/skygear/dimension/subscribe.go`

```diff
--- a/content/skygear/dimension/subscribe.go
+++ b/content/skygear/dimension/subscribe.go
@@ -14,6 +14,7 @@
 	subscribeTicks()
 	subscribePresence()
 	subscribeUse()
+	registerKinds()
 }
 
 // The second dimension's reaction to a death and a respawn.
```

`content/skygear/kinds_test.go`

```diff
--- a/content/skygear/kinds_test.go
+++ b/content/skygear/kinds_test.go
@@ -10,7 +10,8 @@
 // (machine.Register), by kind: the feature named is the kind's Owner. A kind
 // that is not here is the machine feature's own.
 var kindOwners = map[string]string{
-	machine.KindHarvester:        "farm",
+	machine.KindHarvester: "farm",
+	machine.KindCollector: "dimension", machine.KindRelicTable: "dimension",
 	machine.KindDrawerController: "storage", machine.KindDrawerTrim: "storage", machine.KindDrawerInterface: "storage",
 }
 
```

`content/skygear/machine/kinds.go`

```diff
--- a/content/skygear/machine/kinds.go
+++ b/content/skygear/machine/kinds.go
@@ -9,6 +9,7 @@
 	"github.com/df-mc/dragonfly/server/player"
 	"github.com/df-mc/dragonfly/server/world"
 	"github.com/presentkim/skygear/content/skygear/core"
+	"github.com/presentkim/skygear/content/skygear/wintext"
 )
 
 // The kinds of other features. A kind's name, its place in Order, its block
@@ -39,6 +40,9 @@
 	// Lines are the kind's panel lines in place of the machine's own
 	// (Machine.Waila); NoLines is a kind without lines.
 	Lines func(lang string, tx *world.Tx, pos cube.Pos) []string
+	// Info are the lines the kind adds to the info icon of its window, after
+	// the lines that explain it (State.InfoItem).
+	Info func(st *State) []wintext.Text
 }
 
 // NoLines is the Lines of a kind whose panel has no lines.
```

`content/skygear/machine/look.go`

```diff
--- a/content/skygear/machine/look.go
+++ b/content/skygear/machine/look.go
@@ -124,7 +124,7 @@
 		return GeneratorLines(lang, st), nil
 	case IsNatureGenerator(m.Kind):
 		return NatureLines(lang, st), nil
-	case m.Kind == KindTinker, m.Kind == KindRelicTable:
+	case m.Kind == KindTinker:
 		return nil, nil
 	case st.Def.Lines != nil:
 		// the kind's owner describes it (Machine.Waila)
```

`content/skygear/machine/machine.go`

```diff
--- a/content/skygear/machine/machine.go
+++ b/content/skygear/machine/machine.go
@@ -357,11 +357,11 @@
 	},
 	KindCollector: {
 		// The harvester's window: four output slots, no energy.
-		ID: idCollector, nameKey: "machine.collector.name", Marker: "§d§m§3", Interval: 1, Output: []int{13, 14, 15, 16}, Indicator: 10, NoDisplay: true,
+		ID: idCollector, Owner: "dimension", nameKey: "machine.collector.name", Marker: "§d§m§3", Interval: 1, Output: []int{13, 14, 15, 16}, Indicator: 10, NoDisplay: true,
 		helpKeys: []string{"machine.collector.help.1", "machine.collector.help.2"},
 	},
 	KindRelicTable: {
-		ID: idRelicTable, nameKey: "machine.relictable.name", Interval: 1, Indicator: 10, NoDisplay: true,
+		ID: idRelicTable, Owner: "dimension", nameKey: "machine.relictable.name", Interval: 1, Indicator: 10, NoDisplay: true,
 		helpKeys: []string{"machine.relictable.help.1"},
 	},
 	KindNetController: {
@@ -776,12 +776,8 @@
 	for _, k := range st.Def.helpKeys {
 		lore = append(lore, wintext.Text{wintext.Key(k)})
 	}
-	if st.Def.ID == idCollector {
-		if st.Mob != "" {
-			lore = append(lore, wintext.Text{wintext.Key("machine.collector.mob", wintext.Key("mob."+st.Mob))})
-		} else {
-			lore = append(lore, wintext.Text{wintext.Key("machine.collector.none")})
-		}
+	if st.Def.Info != nil {
+		lore = append(lore, st.Def.Info(st)...)
 	}
 	if st.Tier > 0 {
 		spec := st.spec()
@@ -801,7 +797,7 @@
 }
 
 // SetMob sets what the machine works with (the monster an essence
-// collector receives) and shows it in its window.
+// collector receives) and redraws the info icon of its window.
 func (st *State) SetMob(mob string) {
 	st.Mob = mob
 	_ = st.Inv().SetItem(infoSlot, st.InfoItem())
```

### T82

This task does not touch `machine`. `machine` keeps the sibling `tinker` in `siblings`.

`common/slotdeco/slotdeco.go`

```diff
--- a/common/slotdeco/slotdeco.go
+++ b/common/slotdeco/slotdeco.go
@@ -53,6 +53,21 @@
 	return rarityMarks[r]
 }
 
+// Colour returns the colour code of the rarity's name, "§f" for Common.
+func (r Rarity) Colour() string {
+	switch r {
+	case Uncommon:
+		return "§a"
+	case Rare:
+		return "§b"
+	case Epic:
+		return "§d"
+	case Legendary:
+		return "§6"
+	}
+	return "§f"
+}
+
 // TierMark returns the mark of the front image n (1 to MaxFront: tier pips
 // and badges), "" otherwise. It is put after the name; the colour code in it
 // is undone at once.
```

`common/slotdeco/slotdeco_test.go`

```diff
--- a/common/slotdeco/slotdeco_test.go
+++ b/common/slotdeco/slotdeco_test.go
@@ -9,6 +9,16 @@
 	"testing"
 )
 
+// TestRarityColour: the colour of a rarity's name; white for Common and for
+// a value that is no rarity.
+func TestRarityColour(t *testing.T) {
+	for r, want := range map[Rarity]string{Common: "§f", Uncommon: "§a", Rare: "§b", Epic: "§d", Legendary: "§6", Rarity(9): "§f", Rarity(-1): "§f"} {
+		if got := r.Colour(); got != want {
+			t.Errorf("rarity %d: colour %q, want %q", r, got, want)
+		}
+	}
+}
+
 func TestDecorateStripOf(t *testing.T) {
 	for _, tc := range []struct {
 		r    Rarity
```

`content/skygear/decor/decor.go`

```diff
--- a/content/skygear/decor/decor.go
+++ b/content/skygear/decor/decor.go
@@ -10,7 +10,6 @@
 	"github.com/df-mc/dragonfly/server/world"
 	"github.com/presentkim/skygear/common/slotdeco"
 	"github.com/presentkim/skygear/content/skygear/naming"
-	"github.com/presentkim/skygear/content/skygear/tinker"
 )
 
 // ItemDecor returns the rarity and the tier badge of an item type: Common and
@@ -31,5 +30,5 @@
 		return n // stacks with a name of their own carry their marks already
 	}
 	r, tier := ItemDecor(st.Item())
-	return slotdeco.Decorate(r, tier, tinker.RarityColour(r)+naming.UIItemName(lang, st))
+	return slotdeco.Decorate(r, tier, r.Colour()+naming.UIItemName(lang, st))
 }
```

`content/skygear/decor_test.go`

```diff
--- a/content/skygear/decor_test.go
+++ b/content/skygear/decor_test.go
@@ -8,6 +8,7 @@
 	"github.com/presentkim/skygear/common/slotdeco"
 	"github.com/presentkim/skygear/content/skygear/decor"
 	"github.com/presentkim/skygear/content/skygear/fairy"
+	"github.com/presentkim/skygear/content/skygear/naming"
 	"github.com/presentkim/skygear/content/skygear/sgitem"
 )
 
@@ -38,3 +39,25 @@
 		}
 	}
 }
+
+// TestTipNameColour: the name on a slot's tooltip is in the colour of the
+// item's rarity, between the rarity's marks.
+func TestTipNameColour(t *testing.T) {
+	for _, c := range []struct {
+		it     world.Item
+		colour string
+	}{
+		{item.Stick{}, "§f"},
+		{sgitem.Ingot{Metal: sgitem.MetalSteel}, "§a"},
+		{sgitem.Mesh{Tier: "gold"}, "§b"},
+		{sgitem.Relic{Kind: sgitem.RelicWeapons[0]}, "§d"},
+		{sgitem.Circuit{Level: 4}, "§6"},
+	} {
+		st := item.NewStack(c.it, 1)
+		r, tier := decor.ItemDecor(c.it)
+		want := slotdeco.Decorate(r, tier, c.colour+naming.UIItemName("ko", st))
+		if got := decor.TipName("ko", st); got != want {
+			t.Errorf("%T%+v: tip name %q, want %q", c.it, c.it, got, want)
+		}
+	}
+}
```

`content/skygear/layers_test.go`

```diff
--- a/content/skygear/layers_test.go
+++ b/content/skygear/layers_test.go
@@ -36,11 +36,11 @@
 	"locales", "naming", "textfmt", "formicon", "blockmodel", "netshape", "sgitem", "sievetable", "wintext", "machinerecipe",
 	"advancement", "hopper", "migrate", "wire",
 	// core
-	"core", "network", "achievement", "preview", "look", "autofill",
+	"core", "network", "achievement", "preview", "look", "autofill", "decor",
 	// progress
 	"progress", "island", "climate",
 	// features
-	"tinker", "decor", "machine", "conduit", "storage", "farm", "fairy", "dimension", "handwork", "expand", "copier", "wand",
+	"tinker", "machine", "conduit", "storage", "farm", "fairy", "dimension", "handwork", "expand", "copier", "wand",
 	"netstore", "elevator",
 	// content lists
 	"catalog",
@@ -89,20 +89,19 @@
 // imports another one of it only if its row lists it, and every listed import
 // exists. A feature without a row imports no sibling.
 var siblings = map[string][]string{
-	"decor":     {"tinker"},
-	"machine":   {"decor", "tinker"},
+	"machine":   {"tinker"},
 	"conduit":   {"machine"},
 	"storage":   {"machine"},
 	"farm":      {"machine"},
 	"fairy":     {"farm", "machine"},
 	"dimension": {"machine"},
 	"copier":    {"conduit", "machine", "storage"},
-	"netstore":  {"conduit", "decor", "machine", "storage"},
+	"netstore":  {"conduit", "machine", "storage"},
 }
 
 // featureLayer is the "features" group of layers.
 var featureLayer = []string{
-	"tinker", "decor", "machine", "conduit", "storage", "farm", "fairy", "dimension", "handwork", "expand", "copier", "wand",
+	"tinker", "machine", "conduit", "storage", "farm", "fairy", "dimension", "handwork", "expand", "copier", "wand",
 	"netstore", "elevator",
 }
 
@@ -261,7 +260,7 @@
 // and so know no feature and no item type.
 var onlyImports = map[string][]string{
 	"look":  {"core", "naming"},
-	"decor": {"naming", "tinker"},
+	"decor": {"naming"},
 }
 
 // TestCapabilityImports: the packages of onlyImports import only what their
```

`content/skygear/tinker/tinker.go`

```diff
--- a/content/skygear/tinker/tinker.go
+++ b/content/skygear/tinker/tinker.go
@@ -313,7 +313,7 @@
 		}
 	}
 	r := toolRarity(lvl)
-	return st.WithCustomName(slotdeco.Decorate(r, 0, "§r"+RarityColour(r)+t.Name()+fmt.Sprintf(" §7Lv.%d", lvl))).WithLore(lore...)
+	return st.WithCustomName(slotdeco.Decorate(r, 0, "§r"+r.Colour()+t.Name()+fmt.Sprintf(" §7Lv.%d", lvl))).WithLore(lore...)
 }
 
 // WithToolXP adds xp to a tool stack and returns it, with whether it leveled up.
@@ -339,21 +339,6 @@
 	return slotdeco.Common
 }
 
-// RarityColour is the colour code of a rarity's name ("§f" for Common).
-func RarityColour(r slotdeco.Rarity) string {
-	switch r {
-	case slotdeco.Uncommon:
-		return "§a"
-	case slotdeco.Rare:
-		return "§b"
-	case slotdeco.Epic:
-		return "§d"
-	case slotdeco.Legendary:
-		return "§6"
-	}
-	return "§f"
-}
-
 // The tools register as items: every kind with every head and handle.
 func init() {
 	for _, k := range ToolKinds {
```

### T83

`tinker` does not import `machine`. `KindTinker` gets no `Owner` and no row in `kindOwners`.

`content/skygear/layers_test.go`

```diff
--- a/content/skygear/layers_test.go
+++ b/content/skygear/layers_test.go
@@ -153,6 +153,51 @@
 	}
 }
 
+// machineFeatureImports are the packages of the feature layer machine may
+// import. The features that own a machine kind import machine and register
+// the kind's behaviour (machine.Register); machine does not import them back.
+// tinker is the one exception, and only in machine/tinkertable.go: the tinker
+// feature cannot import machine, because the tinker's tools must register
+// before the machines (the client goldens pin the item ids), so machine sets
+// the tinker table's behaviour itself.
+var machineFeatureImports = map[string]string{"tinker": "tinkertable.go"}
+
+// TestMachineImportsNoFeature: machine imports no package of the feature
+// layer but those of machineFeatureImports, each only in its one file, and
+// its siblings row lists exactly those.
+func TestMachineImportsNoFeature(t *testing.T) {
+	const prefix = "github.com/presentkim/skygear/content/skygear/"
+	if row := siblings["machine"]; len(row) != len(machineFeatureImports) {
+		t.Errorf("the siblings row of machine is %v, want the %d of machineFeatureImports", row, len(machineFeatureImports))
+	}
+	for _, dep := range siblings["machine"] {
+		if _, ok := machineFeatureImports[dep]; !ok {
+			t.Errorf("siblings lists machine -> %s: the feature registers its kind's behaviour with machine.Register instead", dep)
+		}
+	}
+	files, err := os.ReadDir("machine")
+	if err != nil {
+		t.Fatal(err)
+	}
+	for _, f := range files {
+		if !strings.HasSuffix(f.Name(), ".go") || strings.HasSuffix(f.Name(), "_test.go") {
+			continue
+		}
+		src, err := os.ReadFile("machine/" + f.Name())
+		if err != nil {
+			t.Fatal(err)
+		}
+		for _, dep := range featureLayer {
+			if !strings.Contains(string(src), `"`+prefix+dep+`"`) {
+				continue
+			}
+			if machineFeatureImports[dep] != f.Name() {
+				t.Errorf("machine/%s imports the feature %s: the feature registers its kind's behaviour with machine.Register instead", f.Name(), dep)
+			}
+		}
+	}
+}
+
 // TestLookExpandingWithTool: a player in island expansion mode who holds a
 // wand, or sneaks with the iron hammer, gets no preview from the panel update
 // its movement makes (the hidden panel returns first), but gets it from the
```

`content/skygear/machine/look.go`

```diff
--- a/content/skygear/machine/look.go
+++ b/content/skygear/machine/look.go
@@ -124,8 +124,6 @@
 		return GeneratorLines(lang, st), nil
 	case IsNatureGenerator(m.Kind):
 		return NatureLines(lang, st), nil
-	case m.Kind == KindTinker:
-		return nil, nil
 	case st.Def.Lines != nil:
 		// the kind's owner describes it (Machine.Waila)
 		return nil, nil
```

`content/skygear/machine/machine.go`

```diff
--- a/content/skygear/machine/machine.go
+++ b/content/skygear/machine/machine.go
@@ -35,7 +35,6 @@
 	"github.com/presentkim/skygear/content/skygear/progress"
 	"github.com/presentkim/skygear/content/skygear/sgitem"
 	"github.com/presentkim/skygear/content/skygear/sievetable"
-	"github.com/presentkim/skygear/content/skygear/tinker"
 	"github.com/presentkim/skygear/content/skygear/wintext"
 	"github.com/presentkim/skygear/platform/account"
 )
@@ -1570,12 +1569,6 @@
 	if c := m.State.Def.Click; c != nil && c(s, p, tx, pos) {
 		return true
 	}
-	if m.Kind == KindTinker {
-		if !s.Debounced(p) {
-			tinker.ShowTable(s, p, tx, pos)
-		}
-		return true
-	}
 	if m.Kind == KindLab {
 		if !s.Debounced(p) {
 			showLab(s, p, tx, pos, m.State)
```

`content/skygear/machine/tinkertable.go` (new)

```go
package machine

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/tinker"
)

// The tinker table is the tinker's block, but the tinker feature cannot
// register its kind (Register): it would have to import this package, and
// then the machines would be initialised, and their items registered, before
// the tinker's tools. The item ids the client receives follow the
// registration order. So this package, which imports tinker, sets the kind's
// behaviour itself: a click opens the tinker table, and the panel has no
// lines. This file is the only one of the package that imports a feature.
func init() {
	Kinds[KindTinker].Behaviour = Behaviour{
		Click: func(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool {
			if !s.Debounced(p) {
				tinker.ShowTable(s, p, tx, pos)
			}
			return true
		},
		Lines: NoLines,
	}
}
```

### T84

Every hook body is the old anonymous function's, unchanged. The encoder's `Input` and `Output` are the values `netstore/encoder.go`'s `init` used to set (`encBlank` 12, `encDone` 14).

`content/skygear/kinds_test.go`

```diff
--- a/content/skygear/kinds_test.go
+++ b/content/skygear/kinds_test.go
@@ -13,6 +13,9 @@
 	machine.KindHarvester: "farm",
 	machine.KindCollector: "dimension", machine.KindRelicTable: "dimension",
 	machine.KindDrawerController: "storage", machine.KindDrawerTrim: "storage", machine.KindDrawerInterface: "storage",
+	machine.KindNetController: "netstore", machine.KindNetCable: "netstore", machine.KindDiskDrive: "netstore",
+	machine.KindNetTerminal: "netstore", machine.KindNetImporter: "netstore", machine.KindNetExporter: "netstore",
+	machine.KindNetExternal: "netstore", machine.KindPatternEncoder: "netstore", machine.KindNetAutocrafter: "netstore",
 }
 
 // TestKindOwners: every kind of machine.Order has the owner kindOwners says,
```

`content/skygear/machine/machine.go`

```diff
--- a/content/skygear/machine/machine.go
+++ b/content/skygear/machine/machine.go
@@ -364,42 +364,43 @@
 		helpKeys: []string{"machine.relictable.help.1"},
 	},
 	KindNetController: {
-		ID: idNetController, nameKey: "machine.net_controller.name", Interval: 1, Indicator: 10,
+		ID: idNetController, Owner: "netstore", nameKey: "machine.net_controller.name", Interval: 1, Indicator: 10,
 		Use: 8, EnergyCap: ConsumerBuffer,
 		helpKeys: []string{"machine.net_controller.help.1"},
 	},
 	KindNetCable: {
-		ID: idNetCable, nameKey: "machine.net_cable.name", Interval: 1, Indicator: 10, NoWindow: true,
+		ID: idNetCable, Owner: "netstore", nameKey: "machine.net_cable.name", Interval: 1, Indicator: 10, NoWindow: true,
 		helpKeys: []string{"machine.net_cable.help.1"},
 	},
 	KindDiskDrive: {
-		ID: idDiskDrive, nameKey: "machine.disk_drive.name", Interval: 1, Input: []int{0, 1, 2, 3, 4, 5, 6, 7}, Indicator: 10,
+		ID: idDiskDrive, Owner: "netstore", nameKey: "machine.disk_drive.name", Interval: 1, Input: []int{0, 1, 2, 3, 4, 5, 6, 7}, Indicator: 10,
 		helpKeys: []string{"machine.disk_drive.help.1"},
 	},
 	KindNetTerminal: {
-		ID: idNetTerminal, nameKey: "machine.net_terminal.name", Interval: 1, Indicator: 10, Input: []int{0, 1, 2, 3, 4, 5, 6, 7, 8},
+		ID: idNetTerminal, Owner: "netstore", nameKey: "machine.net_terminal.name", Interval: 1, Indicator: 10, Input: []int{0, 1, 2, 3, 4, 5, 6, 7, 8},
 		helpKeys: []string{"machine.net_terminal.help.1"},
 	},
 	KindNetImporter: {
 		// four upgrade slots (netstore.Upgrade)
-		ID: idNetImporter, nameKey: "machine.net_importer.name", Interval: 1, Input: []int{0, 1, 2, 3}, Indicator: 10,
+		ID: idNetImporter, Owner: "netstore", nameKey: "machine.net_importer.name", Interval: 1, Input: []int{0, 1, 2, 3}, Indicator: 10,
 		helpKeys: []string{"machine.net_importer.help.1"},
 	},
 	KindNetExporter: {
-		ID: idNetExporter, nameKey: "machine.net_exporter.name", Interval: 1, Input: []int{0, 1, 2, 3}, Indicator: 10,
+		ID: idNetExporter, Owner: "netstore", nameKey: "machine.net_exporter.name", Interval: 1, Input: []int{0, 1, 2, 3}, Indicator: 10,
 		helpKeys: []string{"machine.net_exporter.help.1"},
 	},
 	KindNetExternal: {
-		ID: idNetExternal, nameKey: "machine.net_external.name", Interval: 1, Indicator: 10,
+		ID: idNetExternal, Owner: "netstore", nameKey: "machine.net_external.name", Interval: 1, Indicator: 10,
 		helpKeys: []string{"machine.net_external.help.1"},
 	},
 	KindPatternEncoder: {
-		// nine pattern slots and three sample slots
-		ID: idPatternEncoder, nameKey: "machine.pattern_encoder.name", Interval: 1, Input: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}, Indicator: 13,
+		// nine input samples, three output samples and the blank pattern (12);
+		// the engraved pattern comes out at 14 (netstore/encoder.go)
+		ID: idPatternEncoder, Owner: "netstore", nameKey: "machine.pattern_encoder.name", Interval: 1, Input: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}, Output: []int{14}, Indicator: 13,
 		helpKeys: []string{"machine.pattern_encoder.help.1"},
 	},
 	KindNetAutocrafter: {
-		ID: idNetAutocrafter, nameKey: "machine.net_autocrafter.name", Interval: 1, Input: []int{0, 1, 2, 3, 4, 5, 6, 7, 8}, Indicator: 13,
+		ID: idNetAutocrafter, Owner: "netstore", nameKey: "machine.net_autocrafter.name", Interval: 1, Input: []int{0, 1, 2, 3, 4, 5, 6, 7, 8}, Indicator: 13,
 		helpKeys: []string{"machine.net_autocrafter.help.1"},
 	},
 	KindWasher: {
```

`content/skygear/machine/netkinds_test.go`

```diff
--- a/content/skygear/machine/netkinds_test.go
+++ b/content/skygear/machine/netkinds_test.go
@@ -10,6 +10,10 @@
 	"github.com/presentkim/skygear/content/skygear/netstore"
 )
 
+// The network kinds get their behaviour from the netstore feature's Setup,
+// which no feature list runs in this package's tests.
+func init() { netstore.Feature.Setup() }
+
 var netKinds = []string{"net_controller", "net_cable", "disk_drive", "net_terminal", "net_importer", "net_exporter", "net_external", "pattern_encoder", "net_autocrafter"}
 
 func TestNetKindsRegistered(t *testing.T) {
```

`content/skygear/netstore/craft.go`

```diff
--- a/content/skygear/netstore/craft.go
+++ b/content/skygear/netstore/craft.go
@@ -136,11 +136,14 @@
 	CraftableHook = Craftable
 	RequestHook = func(_ *core.Session, p *player.Player, n *Network, k storagenet.Key) { RequestUI(p, p.Tx(), n, k) }
 	MonitorHook = func(_ *core.Session, p *player.Player, n *Network) { MonitorUI(p, p.Tx(), n) }
-	machine.Kinds[machine.KindNetAutocrafter].Run = func(s *core.Session, _ *machine.State, pos cube.Pos, tx *world.Tx) {
-		sweepLost(tx, s, false)
-		if cs := craftOf(At(tx, pos)); cs != nil {
-			cs.tick(tx)
-		}
+}
+
+// runAutocrafter is one run of an autocrafter's tick: the crafting tasks of
+// its network go on.
+func runAutocrafter(s *core.Session, _ *machine.State, pos cube.Pos, tx *world.Tx) {
+	sweepLost(tx, s, false)
+	if cs := craftOf(At(tx, pos)); cs != nil {
+		cs.tick(tx)
 	}
 }
 
```

`content/skygear/netstore/encoder.go`

```diff
--- a/content/skygear/netstore/encoder.go
+++ b/content/skygear/netstore/encoder.go
@@ -41,26 +41,26 @@
 	encoderPage = 36
 )
 
-func init() {
-	def := machine.Kinds[machine.KindPatternEncoder]
-	def.Input = []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, encBlank}
-	def.Output = []int{encDone}
-	def.Accepts = func(slot int, st item.Stack) bool {
-		if slot >= encIn && slot < encBlank {
-			return true
-		}
-		return isBlank(st)
-	}
-	def.Click = func(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool {
-		if !p.Sneaking() {
-			return false
-		}
-		m, ok := tx.Block(pos).(machine.Machine)
-		if ok && m.State != nil && !s.Debounced(p) {
-			showEncoder(s, p, m.State, 0)
-		}
+// encoderAccepts says what a slot of the encoder takes: any item as a sample,
+// a blank pattern on the blank slot.
+func encoderAccepts(slot int, st item.Stack) bool {
+	if slot >= encIn && slot < encBlank {
 		return true
 	}
+	return isBlank(st)
+}
+
+// clickEncoder is the click on an encoder: a sneaking click opens the recipe
+// list, any other the machine window.
+func clickEncoder(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool {
+	if !p.Sneaking() {
+		return false
+	}
+	m, ok := tx.Block(pos).(machine.Machine)
+	if ok && m.State != nil && !s.Debounced(p) {
+		showEncoder(s, p, m.State, 0)
+	}
+	return true
 }
 
 // isBlank reports whether a stack is a pattern with no recipe on it.
```

`content/skygear/netstore/filterui.go`

```diff
--- a/content/skygear/netstore/filterui.go
+++ b/content/skygear/netstore/filterui.go
@@ -23,15 +23,13 @@
 // with an item held takes that kind (nothing is used up), pressed with an
 // empty hand it empties; a shift click in the inventory adds the kind.
 
-func init() {
-	for _, k := range []string{machine.KindNetImporter, machine.KindNetExporter, machine.KindNetExternal} {
-		machine.Kinds[k].Click = func(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool {
-			if !s.Debounced(p) {
-				showFilter(s, p, pos, nodeAt(tx, pos))
-			}
-			return true
-		}
+// clickFilter is the click on an importer, exporter or external storage: it
+// opens the filter window.
+func clickFilter(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool {
+	if !s.Debounced(p) {
+		showFilter(s, p, pos, nodeAt(tx, pos))
 	}
+	return true
 }
 
 const (
```

`content/skygear/netstore/io.go`

```diff
--- a/content/skygear/netstore/io.go
+++ b/content/skygear/netstore/io.go
@@ -125,9 +125,6 @@
 		delete(rts, n)
 		rtMu.Unlock()
 	})
-	for _, k := range []string{machine.KindNetImporter, machine.KindNetExporter, machine.KindNetExternal} {
-		machine.Kinds[k].Run = nodeRun
-	}
 }
 
 func buildIO(tx *world.Tx, n *Network) {
```

`content/skygear/netstore/items.go`

```diff
--- a/content/skygear/netstore/items.go
+++ b/content/skygear/netstore/items.go
@@ -126,18 +126,21 @@
 	world.RegisterItem(Pattern{})
 	world.RegisterItem(Upgrade{})
 	world.RegisterItem(Upgrade{Stack: true})
-	for _, k := range []string{machine.KindNetImporter, machine.KindNetExporter} {
-		machine.Kinds[k].Accepts = func(_ int, st item.Stack) bool {
-			_, ok := st.Item().(Upgrade)
-			return ok
-		}
-	}
-	machine.Kinds[machine.KindDiskDrive].Accepts = func(_ int, st item.Stack) bool {
-		_, ok := st.Item().(Disk)
-		return ok
-	}
-	machine.Kinds[machine.KindNetAutocrafter].Accepts = func(_ int, st item.Stack) bool {
-		_, ok := st.Item().(Pattern)
-		return ok
-	}
+}
+
+// acceptsUpgrade, acceptsDisk and acceptsPattern say what the slots of an
+// importer or exporter, of a disk drive and of an autocrafter take.
+func acceptsUpgrade(_ int, st item.Stack) bool {
+	_, ok := st.Item().(Upgrade)
+	return ok
+}
+
+func acceptsDisk(_ int, st item.Stack) bool {
+	_, ok := st.Item().(Disk)
+	return ok
+}
+
+func acceptsPattern(_ int, st item.Stack) bool {
+	_, ok := st.Item().(Pattern)
+	return ok
 }
```

`content/skygear/netstore/keys_test.go`

```diff
--- a/content/skygear/netstore/keys_test.go
+++ b/content/skygear/netstore/keys_test.go
@@ -16,11 +16,13 @@
 )
 
 // The tests of this package run without the game's feature list: the storage
-// feature is set up here, so the drawer controller the external storage
-// reads has its behaviour (machine.Register).
+// feature and this one are set up here, so the drawer controller the external
+// storage reads and the network's own kinds have their behaviour
+// (machine.Register).
 func init() {
 	world.DefaultBlockRegistry.Finalize()
 	storage.Feature.Setup()
+	Feature.Setup()
 }
 
 func TestKeyOfCount(t *testing.T) {
```

`content/skygear/netstore/kinds.go` (new)

```go
package netstore

import "github.com/presentkim/skygear/content/skygear/machine"

// registerKinds hands the machine frame the behaviour of the storage
// network's machine kinds. The cable only joins nodes and has none.
func registerKinds() {
	machine.Register(machine.KindNetController, machine.Behaviour{
		Click: clickController, Run: runController,
		Insert: insertAt, Peek: peekAt, Extract: extractAt,
	})
	machine.Register(machine.KindNetCable, machine.Behaviour{})
	machine.Register(machine.KindDiskDrive, machine.Behaviour{Accepts: acceptsDisk})
	machine.Register(machine.KindNetTerminal, machine.Behaviour{
		Click:  clickTerminal,
		Insert: insertAt, Peek: peekAt, Extract: extractAt,
	})
	machine.Register(machine.KindNetImporter, machine.Behaviour{Click: clickFilter, Run: nodeRun, Accepts: acceptsUpgrade})
	machine.Register(machine.KindNetExporter, machine.Behaviour{Click: clickFilter, Run: nodeRun, Accepts: acceptsUpgrade})
	machine.Register(machine.KindNetExternal, machine.Behaviour{Click: clickFilter, Run: nodeRun})
	machine.Register(machine.KindPatternEncoder, machine.Behaviour{Click: clickEncoder, Accepts: encoderAccepts})
	machine.Register(machine.KindNetAutocrafter, machine.Behaviour{Run: runAutocrafter, Accepts: acceptsPattern})
}
```

`content/skygear/netstore/network.go`

```diff
--- a/content/skygear/netstore/network.go
+++ b/content/skygear/netstore/network.go
@@ -309,7 +309,3 @@
 		n.flush()
 	}
 }
-
-func init() {
-	machine.Kinds[machine.KindNetController].Run = runController
-}
```

`content/skygear/netstore/subscribe.go`

```diff
--- a/content/skygear/netstore/subscribe.go
+++ b/content/skygear/netstore/subscribe.go
@@ -10,6 +10,7 @@
 func subscribe() {
 	subscribeFlush()
 	subscribeBlocks()
+	registerKinds()
 }
 
 // Disk contents changed since the controller last flushed them are part of the
```

`content/skygear/netstore/terminal.go`

```diff
--- a/content/skygear/netstore/terminal.go
+++ b/content/skygear/netstore/terminal.go
@@ -305,37 +305,45 @@
 
 // A click opens the terminal window on a terminal and the network's node list
 // on a controller; both take and give items through Root.
-func init() {
-	machine.Kinds[machine.KindNetController].Click = func(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool {
-		if !s.Debounced(p) {
-			showController(s, p, tx, pos)
-		}
-		return true
+
+// clickController is the click on a controller: the network's node list.
+func clickController(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool {
+	if !s.Debounced(p) {
+		showController(s, p, tx, pos)
 	}
-	machine.Kinds[machine.KindNetTerminal].Click = func(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool {
-		ShowTerminal(s, p, tx, pos)
-		return true
+	return true
+}
+
+// clickTerminal is the click on a terminal: the terminal window.
+func clickTerminal(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool {
+	ShowTerminal(s, p, tx, pos)
+	return true
+}
+
+// insertAt puts st into the network of the terminal or controller at pos and
+// returns how many it took.
+func insertAt(tx *world.Tx, pos cube.Pos, st item.Stack) int {
+	return int(netInsert(At(tx, pos), st))
+}
+
+// peekAt is the first stack the network of the terminal or controller at pos
+// would give that want accepts.
+func peekAt(tx *world.Tx, pos cube.Pos, want func(item.Stack) bool) item.Stack {
+	n := At(tx, pos)
+	if n == nil || !n.Online() {
+		return item.Stack{}
 	}
-	for _, kind := range []string{machine.KindNetTerminal, machine.KindNetController} {
-		d := machine.Kinds[kind]
-		d.Insert = func(tx *world.Tx, pos cube.Pos, st item.Stack) int {
-			return int(netInsert(At(tx, pos), st))
-		}
-		d.Peek = func(tx *world.Tx, pos cube.Pos, want func(item.Stack) bool) item.Stack {
-			n := At(tx, pos)
-			if n == nil || !n.Online() {
-				return item.Stack{}
-			}
-			for _, e := range entriesOf(n) {
-				if want == nil || want(e.Item) {
-					return e.Item.Grow(int(min(e.Count, int64(e.Item.MaxCount()))) - 1)
-				}
-			}
-			return item.Stack{}
-		}
-		d.Extract = func(tx *world.Tx, pos cube.Pos, st item.Stack, cnt int) item.Stack {
-			n := At(tx, pos)
-			return netExtract(n, KeyOf(st), cnt)
+	for _, e := range entriesOf(n) {
+		if want == nil || want(e.Item) {
+			return e.Item.Grow(int(min(e.Count, int64(e.Item.MaxCount()))) - 1)
 		}
 	}
+	return item.Stack{}
+}
+
+// extractAt takes cnt of st out of the network of the terminal or controller
+// at pos.
+func extractAt(tx *world.Tx, pos cube.Pos, st item.Stack, cnt int) item.Stack {
+	n := At(tx, pos)
+	return netExtract(n, KeyOf(st), cnt)
 }
```

### T85

The block removed from `machine/machine.go` runs from the comment `// cubeMachines are the machines drawn as plain cubes` to the closing brace of `func (m Machine) Properties()`.

`content/skygear/machine/kinds_test.go`

```diff
--- a/content/skygear/machine/kinds_test.go
+++ b/content/skygear/machine/kinds_test.go
@@ -115,3 +115,24 @@
 		}
 	}
 }
+
+// TestEveryKindHasOneShape: a kind is a sieve, a thin network block or a cube
+// of exactly one of the three texture sets, so the properties of a cube that
+// is in neither cubeMachines nor plainCubes are those of a frontMachines kind.
+func TestEveryKindHasOneShape(t *testing.T) {
+	for _, k := range Order {
+		_, plain := plainCubes[k]
+		n := 0
+		for _, in := range []bool{IsSieve(k), NetShaped(k), plain, cubeMachines[k], frontMachines[k]} {
+			if in {
+				n++
+			}
+		}
+		if n != 1 {
+			t.Errorf("%s is in %d of the shape sets, want 1", k, n)
+		}
+		if cube := plain || cubeMachines[k] || frontMachines[k]; cube != (shapeOf(k) == shapeCube) {
+			t.Errorf("%s: shapeOf says %d", k, shapeOf(k))
+		}
+	}
+}
```

`content/skygear/machine/machine.go`

```diff
--- a/content/skygear/machine/machine.go
+++ b/content/skygear/machine/machine.go
@@ -9,7 +9,6 @@
 
 	"github.com/df-mc/dragonfly/server/block"
 	"github.com/df-mc/dragonfly/server/block/cube"
-	"github.com/df-mc/dragonfly/server/block/customblock"
 	"github.com/df-mc/dragonfly/server/block/model"
 	"github.com/df-mc/dragonfly/server/item"
 	"github.com/df-mc/dragonfly/server/item/category"
@@ -19,7 +18,6 @@
 	"github.com/df-mc/dragonfly/server/world/particle"
 	"github.com/go-gl/mathgl/mgl64"
 	"github.com/presentkim/skygear/common/chestui"
-	"github.com/presentkim/skygear/common/clientdef"
 	"github.com/presentkim/skygear/common/fluidnet"
 	"github.com/presentkim/skygear/common/i18n"
 	"github.com/presentkim/skygear/common/slotdeco"
@@ -1271,89 +1269,6 @@
 
 func (Machine) Category() category.Category { return category.Items() }
 
-// cubeMachines are the machines drawn as plain cubes with one side texture;
-// frontMachines have a front (south), back and side textures. The sieves
-// have models.
-var (
-	cubeMachines  = map[string]bool{KindMiner: true, KindHarvester: true, KindGenerator: true, KindEnergyCell: true, KindVacuum: true, KindFeller: true, KindTank: true, KindLavaGenerator: true, KindSolarGenerator: true, KindHydroGenerator: true, KindDrawerController: true, KindDrawerInterface: true, KindTinker: true, KindCollector: true, KindRelicTable: true, KindNetController: true, KindDiskDrive: true, KindNetTerminal: true, KindPatternEncoder: true}
-	frontMachines = map[string]bool{KindCrusher: true, KindWasher: true, KindAlloy: true, KindSimulator: true, KindCrucible: true, KindFurnace: true, KindCrafter: true, KindLab: true, KindNetAutocrafter: true}
-)
-
-// States implements block.Permutable: only the sieve has states.
-func (m Machine) States() map[string][]any {
-	if IsSieve(m.Kind) {
-		return SieveStates()
-	}
-	if TierLook(m.Kind) {
-		return tierStates(m.Kind)
-	}
-	if AnimatedKinds[m.Kind] {
-		return activeStates()
-	}
-	if NetShaped(m.Kind) {
-		return netStates(m.Kind)
-	}
-	return nil
-}
-
-// Permutations implements block.Permutable.
-func (m Machine) Permutations() []customblock.Permutation {
-	if IsSieve(m.Kind) {
-		return sievePermutations(m.Kind)
-	}
-	if TierLook(m.Kind) {
-		return tierPermutations(m.Kind)
-	}
-	if AnimatedKinds[m.Kind] {
-		return activePermutations(m.Kind)
-	}
-	if NetShaped(m.Kind) {
-		return netPermutations(m.Kind)
-	}
-	return nil
-}
-
-// ClientExtras implements clientdef.ExtraProperties for the machines whose
-// geometry has bones: the base follows Properties and the permutations follow
-// Permutations. A sieve shows its content bones, a network block its shape's
-// bones and collision boxes.
-func (m Machine) ClientExtras() (clientdef.Extras, map[string]clientdef.Extras) {
-	var base clientdef.Extras
-	switch {
-	case m.Kind == KindDrawerTrim || cubeMachines[m.Kind] || frontMachines[m.Kind]:
-	case NetShaped(m.Kind):
-		base, _ = netAllExtras(m.Kind)
-	default:
-		base, _ = sieveExtras()
-	}
-	switch {
-	case IsSieve(m.Kind):
-		_, perms := sieveExtras()
-		return base, perms
-	case TierLook(m.Kind), AnimatedKinds[m.Kind]:
-	case NetShaped(m.Kind):
-		_, perms := netAllExtras(m.Kind)
-		return base, perms
-	}
-	return base, nil
-}
-
-func (m Machine) Properties() customblock.Properties {
-	if m.Kind == KindDrawerTrim {
-		return customblock.Properties{Cube: true, Textures: map[string]customblock.Material{"*": blockmodel.AlphaMaterial("skygear_drawer_side")}}
-	}
-	if cubeMachines[m.Kind] {
-		return blockmodel.CubeProperties(m.Kind)
-	}
-	if frontMachines[m.Kind] {
-		return blockmodel.FrontCubeProperties(m.Kind)
-	}
-	if NetShaped(m.Kind) {
-		return netProperties(m.Kind, 0)
-	}
-	return sieveProperties(m.Kind)
-}
-
 // BreakInfo drops the machine and its contents.
 func (m Machine) BreakInfo() block.BreakInfo {
 	return block.BreakInfo{
```

`content/skygear/machine/shape.go` (new)

```go
package machine

import (
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/presentkim/skygear/common/clientdef"
	"github.com/presentkim/skygear/content/skygear/blockmodel"
)

// How the client draws a kind. A kind has one of three shapes, and its
// states, permutations, base properties and extra components all follow from
// that shape (shapeOf); the four methods below are what Dragonfly and
// clientdef ask a custom block.

// shape is how the client draws the blocks of a kind.
type shape int

const (
	// shapeCube is a full cube with textures by face; some kinds have a look
	// per tier or a working look as block states (tierlook.go, animate.go).
	shapeCube shape = iota
	// shapeSieve is the sieve model with a state for what lies on its mesh
	// (sieve.go).
	shapeSieve
	// shapeNet is a thin network block with a state for its arms and facing
	// (netshape.go).
	shapeNet
)

// shapeOf is the shape of a kind.
func shapeOf(kind string) shape {
	switch {
	case IsSieve(kind):
		return shapeSieve
	case NetShaped(kind):
		return shapeNet
	}
	return shapeCube
}

// The textures of the cubes. cubeMachines are drawn with one side texture
// and frontMachines with a front (south), back and side textures, both named
// after the kind; plainCubes have the one named texture on every face.
var (
	cubeMachines  = map[string]bool{KindMiner: true, KindHarvester: true, KindGenerator: true, KindEnergyCell: true, KindVacuum: true, KindFeller: true, KindTank: true, KindLavaGenerator: true, KindSolarGenerator: true, KindHydroGenerator: true, KindDrawerController: true, KindDrawerInterface: true, KindTinker: true, KindCollector: true, KindRelicTable: true, KindNetController: true, KindDiskDrive: true, KindNetTerminal: true, KindPatternEncoder: true}
	frontMachines = map[string]bool{KindCrusher: true, KindWasher: true, KindAlloy: true, KindSimulator: true, KindCrucible: true, KindFurnace: true, KindCrafter: true, KindLab: true, KindNetAutocrafter: true}
	plainCubes    = map[string]string{KindDrawerTrim: "skygear_drawer_side"}
)

// cubeProperties are the base properties of a cube kind.
func cubeProperties(kind string) customblock.Properties {
	if tex, ok := plainCubes[kind]; ok {
		return customblock.Properties{Cube: true, Textures: map[string]customblock.Material{"*": blockmodel.AlphaMaterial(tex)}}
	}
	if cubeMachines[kind] {
		return blockmodel.CubeProperties(kind)
	}
	return blockmodel.FrontCubeProperties(kind)
}

// States implements block.Permutable.
func (m Machine) States() map[string][]any {
	switch shapeOf(m.Kind) {
	case shapeSieve:
		return SieveStates()
	case shapeNet:
		return netStates(m.Kind)
	}
	if TierLook(m.Kind) {
		return tierStates(m.Kind)
	}
	if AnimatedKinds[m.Kind] {
		return activeStates()
	}
	return nil
}

// Permutations implements block.Permutable.
func (m Machine) Permutations() []customblock.Permutation {
	switch shapeOf(m.Kind) {
	case shapeSieve:
		return sievePermutations(m.Kind)
	case shapeNet:
		return netPermutations(m.Kind)
	}
	if TierLook(m.Kind) {
		return tierPermutations(m.Kind)
	}
	if AnimatedKinds[m.Kind] {
		return activePermutations(m.Kind)
	}
	return nil
}

// Properties are the base properties of the kind's block.
func (m Machine) Properties() customblock.Properties {
	switch shapeOf(m.Kind) {
	case shapeSieve:
		return sieveProperties(m.Kind)
	case shapeNet:
		return netProperties(m.Kind, 0)
	}
	return cubeProperties(m.Kind)
}

// ClientExtras implements clientdef.ExtraProperties for the kinds whose
// geometry has bones: a sieve shows its content bones, a network block its
// shape's bones and collision boxes. A cube has none.
func (m Machine) ClientExtras() (clientdef.Extras, map[string]clientdef.Extras) {
	switch shapeOf(m.Kind) {
	case shapeSieve:
		return sieveExtras()
	case shapeNet:
		return netAllExtras(m.Kind)
	}
	return clientdef.Extras{}, nil
}
```

### T86

Only the receiver path changes at each place (`Def.` to `Def.hooks.`); no condition and no argument changes.

`content/skygear/debug_ui.go`

```diff
--- a/content/skygear/debug_ui.go
+++ b/content/skygear/debug_ui.go
@@ -29,7 +29,7 @@
 func windowKinds() []string {
 	var out []string
 	for _, k := range machine.Order {
-		if d := machine.Kinds[k]; d.Marker != "" && d.Click == nil && k != machine.KindSieve && k != machine.KindTinker {
+		if d := machine.Kinds[k]; d.Marker != "" && d.Hooks().Click == nil && k != machine.KindSieve && k != machine.KindTinker {
 			out = append(out, k)
 		}
 	}
```

`content/skygear/machine/kinds.go`

```diff
--- a/content/skygear/machine/kinds.go
+++ b/content/skygear/machine/kinds.go
@@ -68,9 +68,13 @@
 	case kindsChecked:
 		panic(fmt.Sprintf("machine: Register: kind %q registered after the features were installed", kind))
 	}
-	d.Behaviour, d.registered = b, true
+	d.hooks, d.registered = b, true
 }
 
+// Hooks is the behaviour registered for the kind: the zero value for a kind
+// of the machine feature itself.
+func (k *Def) Hooks() Behaviour { return k.hooks }
+
 // CheckKinds reports the kinds whose owner did not register their behaviour.
 // It is the machine feature's Check: core.Install calls it when the Setup of
 // every feature has run and panics with its error, so a game that lists a
```

`content/skygear/machine/kinds_test.go`

```diff
--- a/content/skygear/machine/kinds_test.go
+++ b/content/skygear/machine/kinds_test.go
@@ -13,7 +13,7 @@
 	d := Kinds[KindRelicTable]
 	saved, checked := *d, kindsChecked
 	t.Cleanup(func() { *d, kindsChecked = saved, checked })
-	d.Owner, d.registered, d.Behaviour = "dimension", false, Behaviour{}
+	d.Owner, d.registered, d.hooks = "dimension", false, Behaviour{}
 	kindsChecked = false
 	return d
 }
@@ -41,7 +41,7 @@
 		t.Fatal("a failed check closed the registry")
 	}
 	Register(KindRelicTable, Behaviour{Lines: NoLines})
-	if !d.registered || d.Lines == nil {
+	if !d.registered || d.Hooks().Lines == nil {
 		t.Fatal("Register did not set the behaviour")
 	}
 	if err := CheckKinds(); err != nil && strings.Contains(err.Error(), "relic_table") {
```

`content/skygear/machine/look.go`

```diff
--- a/content/skygear/machine/look.go
+++ b/content/skygear/machine/look.go
@@ -15,7 +15,7 @@
 
 // Waila fills in a machine's panel (waila.Provider): its lines and item
 // line, the time boost left, the lines of a kind another package describes
-// (Def.Lines), what waits in its output as slots, and the cycle figures while
+// (Behaviour.Lines), what waits in its output as slots, and the cycle figures while
 // the viewer sneaks. A sneaking look also shows the machine's range for a
 // moment (PeekRange).
 func (m Machine) Waila(v waila.View, info waila.Info) waila.Info {
@@ -26,8 +26,8 @@
 	if l := BoostLine(v.Lang, m.State); l != "" {
 		info.Lines = append(info.Lines, l)
 	}
-	if m.State != nil && m.State.Def.Lines != nil {
-		info.Lines = m.State.Def.Lines(v.Lang, v.Tx, v.Pos)
+	if m.State != nil && m.State.Def.hooks.Lines != nil {
+		info.Lines = m.State.Def.hooks.Lines(v.Lang, v.Tx, v.Pos)
 	}
 	if c := outputContainer(m.State); c != nil {
 		info.Container = c
@@ -124,7 +124,7 @@
 		return GeneratorLines(lang, st), nil
 	case IsNatureGenerator(m.Kind):
 		return NatureLines(lang, st), nil
-	case st.Def.Lines != nil:
+	case st.Def.hooks.Lines != nil:
 		// the kind's owner describes it (Machine.Waila)
 		return nil, nil
 	case st.Def.FluidCap > 0:
```

`content/skygear/machine/machine.go`

```diff
--- a/content/skygear/machine/machine.go
+++ b/content/skygear/machine/machine.go
@@ -285,9 +285,10 @@
 	// once it did (kinds.go).
 	Owner      string
 	registered bool
-	// Behaviour is what the owner registered: the zero value until then and
-	// for the machine feature's own kinds.
-	Behaviour
+	// hooks is what the owner registered: the zero value until then and for
+	// the machine feature's own kinds. Only Register writes it for a kind
+	// with an owner; Hooks reads it.
+	hooks Behaviour
 }
 
 var Kinds = map[string]*Def{
@@ -774,8 +775,8 @@
 	for _, k := range st.Def.helpKeys {
 		lore = append(lore, wintext.Text{wintext.Key(k)})
 	}
-	if st.Def.Info != nil {
-		lore = append(lore, st.Def.Info(st)...)
+	if st.Def.hooks.Info != nil {
+		lore = append(lore, st.Def.hooks.Info(st)...)
 	}
 	if st.Tier > 0 {
 		spec := st.spec()
@@ -838,8 +839,8 @@
 // accepts reports whether it may go into the input of the machine.
 // Time-limited items are never processed or sold.
 func (st *State) Accepts(it item.Stack) bool {
-	if st.Def.Accepts != nil {
-		return !it.Empty() && st.Def.Accepts(-1, it)
+	if st.Def.hooks.Accepts != nil {
+		return !it.Empty() && st.Def.hooks.Accepts(-1, it)
 	}
 	if it.Empty() || !core.Plain(it) {
 		return false
@@ -898,8 +899,8 @@
 func (st *State) SlotAccepts(slot int, it item.Stack) bool {
 	switch st.Role(slot) {
 	case roleInput:
-		if st.Def.Accepts != nil {
-			return !it.Empty() && st.Def.Accepts(slot, it)
+		if st.Def.hooks.Accepts != nil {
+			return !it.Empty() && st.Def.hooks.Accepts(slot, it)
 		}
 		return st.Accepts(it) && !st.takesCharge(it)
 	case roleCharge:
@@ -1482,7 +1483,7 @@
 		useSieve(s, p, tx, pos, m.State)
 		return true
 	}
-	if c := m.State.Def.Click; c != nil && c(s, p, tx, pos) {
+	if c := m.State.Def.hooks.Click; c != nil && c(s, p, tx, pos) {
 		return true
 	}
 	if m.Kind == KindLab {
@@ -1517,8 +1518,8 @@
 // (it needs its mesh), so it takes one only while it is empty, and its look
 // follows.
 func (m Machine) InsertItems(tx *world.Tx, pos cube.Pos, face cube.Face, st item.Stack) int {
-	if m.State != nil && m.State.Def.Insert != nil {
-		return m.State.Def.Insert(tx, pos, st)
+	if m.State != nil && m.State.Def.hooks.Insert != nil {
+		return m.State.Def.hooks.Insert(tx, pos, st)
 	}
 	if m.State == nil || len(m.State.Def.Input) == 0 || face == cube.FaceDown {
 		return 0
@@ -1541,8 +1542,8 @@
 // (a hopper under the machine, a conduit anywhere); machines do not push it
 // into their neighbours.
 func (m Machine) PeekItems(tx *world.Tx, pos cube.Pos, _ cube.Face, want func(item.Stack) bool) item.Stack {
-	if m.State != nil && m.State.Def.Peek != nil {
-		return m.State.Def.Peek(tx, pos, want)
+	if m.State != nil && m.State.Def.hooks.Peek != nil {
+		return m.State.Def.hooks.Peek(tx, pos, want)
 	}
 	if m.State == nil {
 		return item.Stack{}
@@ -1557,8 +1558,8 @@
 
 // ExtractItems implements itemio.Extractor.
 func (m Machine) ExtractItems(tx *world.Tx, pos cube.Pos, _ cube.Face, st item.Stack, n int) item.Stack {
-	if m.State != nil && m.State.Def.Extract != nil {
-		return m.State.Def.Extract(tx, pos, st, n)
+	if m.State != nil && m.State.Def.hooks.Extract != nil {
+		return m.State.Def.hooks.Extract(tx, pos, st, n)
 	}
 	if m.State == nil {
 		return item.Stack{}
@@ -1728,8 +1729,8 @@
 	case idLab:
 		st.runLab()
 	default:
-		if st.Def.Run != nil {
-			st.Def.Run(s, st, pos, tx)
+		if st.Def.hooks.Run != nil {
+			st.Def.hooks.Run(s, st, pos, tx)
 		}
 	}
 }
```

`content/skygear/machine/tinkertable.go`

```diff
--- a/content/skygear/machine/tinkertable.go
+++ b/content/skygear/machine/tinkertable.go
@@ -16,7 +16,7 @@
 // behaviour itself: a click opens the tinker table, and the panel has no
 // lines. This file is the only one of the package that imports a feature.
 func init() {
-	Kinds[KindTinker].Behaviour = Behaviour{
+	Kinds[KindTinker].hooks = Behaviour{
 		Click: func(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool {
 			if !s.Debounced(p) {
 				tinker.ShowTable(s, p, tx, pos)
```

`content/skygear/netstore/integration_test.go`

```diff
--- a/content/skygear/netstore/integration_test.go
+++ b/content/skygear/netstore/integration_test.go
@@ -78,7 +78,7 @@
 		if out.Count() != 5 || n.Root.Count(KeyOf(dirtS)) != 0 {
 			t.Fatalf("took %v, left %d", out, n.Root.Count(KeyOf(dirtS)))
 		}
-		if got := machine.Kinds[machine.KindNetTerminal].Insert(tx, p(term), out); got != 5 {
+		if got := machine.Kinds[machine.KindNetTerminal].Hooks().Insert(tx, p(term), out); got != 5 {
 			t.Fatalf("deposit %d", got)
 		}
 
```

`content/skygear/netstore/io_test.go`

```diff
--- a/content/skygear/netstore/io_test.go
+++ b/content/skygear/netstore/io_test.go
@@ -359,7 +359,7 @@
 			t.Fatalf("online %v, mirrored %d", n.Online(), n.Root.Count(KeyOf(stoneS)))
 		}
 		// the controller's item I/O reaches the chest through At alone
-		got := machine.Kinds[machine.KindNetController].Extract(tx, p(0), stoneS, 4)
+		got := machine.Kinds[machine.KindNetController].Hooks().Extract(tx, p(0), stoneS, 4)
 		if got.Count() != 4 {
 			t.Fatalf("controller extract %d", got.Count())
 		}
```

`content/skygear/netstore/terminal_test.go`

```diff
--- a/content/skygear/netstore/terminal_test.go
+++ b/content/skygear/netstore/terminal_test.go
@@ -156,7 +156,7 @@
 func TestControllerItemIO(t *testing.T) {
 	netEnv(t, func(s *core.Session, tx *world.Tx) {
 		n := termNet(t, tx, 0)
-		d := machine.Kinds[machine.KindNetController]
+		d := machine.Kinds[machine.KindNetController].Hooks()
 		stone := item.NewStack(block.Stone{}, 20)
 		if got := d.Insert(tx, p(0), stone); got != 20 {
 			t.Fatalf("insert %d", got)
@@ -178,7 +178,7 @@
 			t.Fatalf("extract %v root %d", out, n.Root.Count(KeyOf(stone)))
 		}
 		// the terminal has the same hooks, and an offline network moves nothing
-		tm := machine.Kinds[machine.KindNetTerminal]
+		tm := machine.Kinds[machine.KindNetTerminal].Hooks()
 		if tm.Insert(tx, p(1), stone) != 20 {
 			t.Fatal("terminal insert")
 		}
```

### T87

`content/skygear/machine/doc.go`: the last paragraph of the package comment ("The kinds of other parts of the game … Insert, Peek and Extract.") becomes

```go
// The kinds of other features (the drawer controller, the harvester, the
// nodes of the storage network ...) are declared here like every kind, so
// that the blocks and items register in one order, and name their feature as
// Owner. The feature hands the kind's Behaviour (Click, Run, Insert, Peek,
// Extract, Accepts, Lines, Info) to Register from its Setup; CheckKinds, the
// machine feature's Check, stops the installation when one did not
// (kinds.go). The tinker table is set in tinkertable.go.
```

`content/skygear/core/doc.go`: in the paragraph that starts "A Feature is what a feature package hands to the game", "commands and a Setup that subscribes its steps" becomes "commands, a Setup that subscribes its steps and a Check that Install runs when every feature is set up".

The other documents are written from the code as built, as the task's steps say.
