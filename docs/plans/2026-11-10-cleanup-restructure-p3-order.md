# Phase 3 reference: dispatch order, marks, scenarios

Reference for the machine plans p3a-p3e (not a machine plan, not linted). It is the transcription of `content/skygear/handler.go`, `second.go`, `game.go`, `lostfound.go` and `conduitload.go` at 541f654 into steps. A step is one future subscription: `<priority> <name>`. The name's prefix is the owning feature. `H` marks a step that may stop the chain (`Hook.Handle`); the others always let the chain go on (`Hook.On`). Lower priorities run first. Priorities are per hook, steps of 10 in today's line order, so a later feature can sit between two steps.

## 1. Rules

- R1 One step per owner and contiguous region of today's code. Two regions of one owner are one step only where the plan says why (type-exclusive cases).
- R2 Mark rule (pin tasks T35-T38): `core.Mark("<name>")` goes on the line before the statement that begins the step, so a mark is written exactly when no earlier step returned. Marks never change control flow. For the two type switches (`HandleBlockBreak`, `HandleBlockPlace`) the marks of the switch's steps go, in priority order, on the lines before the `switch`. `deny` gets a third parameter, the mark suffix (`"guard"` or `"guardSide"`), and writes `expand.<suffix>` on entry and `island.<suffix>` after the expansion check. In `HandleToggleSneak` and `HandleToggleSprint` every mark is inside `if after { ... }` (add that `if` around the first mark): the kernel emits `Sneak` and `Sprint` only when sneaking or sprinting starts, because nothing happens today when it ends.
- R3 Stop rule (bridge tasks): a step is `H` exactly where today's code returns before the end of the method. Its function returns true on that path only. A step that cancels at the end of the method (the drawer's front punch, whose `return` leaves the last statement; the void fall; `dimension.fallen`) calls `e.Cancel()` and stops nothing.
- R4 The kernel builds each event once, at the start of the handler method: `Block` is `tx.Block(pos)` for `BlockBreak` and the block being placed for `BlockPlace`; `Held` and `Off` are `p.HeldItems()`. Steps read these fields; they do not read them again, except where today's code reads after a change. These are reads only, so making them before the guards changes nothing.
- R5 Bridge files: the steps of feature `f` live in `content/skygear/legacy_<f>.go` (package `skygear`, an `init` that subscribes) from the bridge task that creates the step until the feature task that moves them to `<f>/feature.go` with the same names and priorities. The feature task deletes the file. Owner `core` uses `legacy_core.go`, owner `autofill` uses `legacy_autofill.go`.
- R6 Goldens under `content/skygear/testdata/pin/` are written once, by the task named in section 4, with `SKYGEAR_GOLDEN=update`. No later task owns them, so the checkpoint tool rejects a change. A pin that fails after a change means the change is wrong.
- R7 Speed: before the first edit and after the last, a task that owns `handler.go`, `second.go`, `game.go` or `core/hook.go` runs `go test ./content/skygear -run '^$' -bench 'Second$|HandleMove$|EnergyStep$' -benchmem -count 8`. Fail: allocs/op or B/op higher on any benchmark, or the median ns/op higher than the before median by more than the before runs' spread (max minus min). Report both tables.

## 2. Player hooks (23)

| Hook (event) | Steps |
|---|---|
| `StartBreak` (`StartBreakEvent`) | 10 expand.guard H · 20 island.guard H · 30 tinker.autoTool |
| `BlockBreak` (`BlockBreakEvent`) | 10 expand.guard H · 20 island.guard H · 30 achievement.break · 40 tinker.break · 50 farm.break · 60 storage.break · 70 netstore.break · 80 machine.break · 90 conduit.break |
| `BlockPlace` (`BlockPlaceEvent`) | 10 expand.guard H · 20 island.guard H · 30 achievement.place · 40 autofill.place · 50 farm.plant · 60 storage.place · 70 machine.place · 80 netstore.place · 90 conduit.place · 100 progress.dragonEgg |
| `UseOnBlock` (`UseOnBlockEvent`) | 10 dimension.hunting H · 20 screens.menuItem H · 30 dimension.stone H · 40 expand.guard H · 50 island.guard H · 60 machine.noteFace · 70 copier.use H · 80 wand.use H · 90 storage.sneakUse H · 100 conduit.plate H · 110 handwork.stick H · 120 farm.composter H · 130 handwork.gather · 140 fairy.time H · 150 farm.crop H · 160 expand.guardSide H · 170 island.guardSide H |
| `UseOnEntity` (`EntityEvent`) | 10 dimension.hunting H · 20 fairy.touch |
| `AttackEntity` (`EntityEvent`) | 10 dimension.hunter H · 20 fairy.touch |
| `Use` (`UseEvent`) | 10 autofill.use · 20 screens.menuItem H · 30 dimension.hunting H · 40 dimension.stone H · 50 copier.clear · 60 wand.options · 70 tinker.veinToggle |
| `Move` (`MoveEvent`) | 10 look.update · 20 expand.hold H · 30 achievement.walk · 40 island.void |
| `Sneak` (`PlayerEvent`, start only) | 10 expand.show H · 20 achievement.sneak · 30 elevator.down H · 40 farm.crouchGrow |
| `Sprint` (`PlayerEvent`, start only) | 10 achievement.sprint |
| `Jump` (`PlayerEvent`) | 10 achievement.jump · 20 elevator.up |
| `ItemDamage` (`ItemDamageEvent`) | 10 tinker.ironHammer · 20 autofill.broken |
| `ItemConsume` (`ItemEvent`) | 10 achievement.eat · 20 autofill.consume |
| `ItemPickup` (`ItemEvent`) | 10 achievement.pickup |
| `ItemDrop` (`ItemEvent`) | 10 dimension.hunting H · 20 achievement.drop |
| `Death` (`PlayerEvent`) | 10 achievement.death · 20 dimension.died · 30 tinker.keepFood |
| `Respawn` (`RespawnEvent`) | 10 dimension.respawned · 20 tinker.restoreFood |
| `Hurt` (`HurtEvent`) | 10 dimension.frozen H · 20 achievement.hurt · 30 dimension.fallen |
| `PunchAir`, `Chat`, `BlockPick`, `HeldSlot`, `Command` (`PlayerEvent`) | 10 achievement.punchAir / .chat / .blockPick / .heldSlot / .command |

What each step holds (today's lines in `handler.go`):

- `expand.guard`, `island.guard` (and `...Side`): the two halves of `deny` (41-59). `expand.*` cancels, calls `RestoreMapNear` and stops when the player is in expansion mode. `island.*` passes when `island.CanBuild`, else cancels, sends the tip unless `core.InArena`, and stops. The `Side` pair checks `pos.Side(face)` (218).
- `tinker.break`: tool XP and reinforce (75-78), then for a hammer the hammer drop and vein mining (79-84). One step: adjacent, one owner.
- `farm.break`: crook on leaves (85-91), composter (112-113), barrel (114-115). One step at the crook's place: the three act on different block types, and the steps between them (storage, netstore, machine) act only on drawers and machines, so no order between them can be seen.
- `storage.break`: a drawer hit on its front cancels and removes nothing, another drawer loses its display (93-99); a set machine invalidates the networks (101-103). `netstore.break`: the deferred invalidate and forget (104-110). `machine.break`: `machine.Break` (111). `conduit.break`: `RemoveVisual` (116-117). Today's order inside the machine case is storage, netstore, machine: priorities 60, 70, 80.
- `BlockPlace`: `achievement.place` 124, `autofill.place` 125-127, `farm.plant` 128-131 (with its `crop_plant` count), `storage.place` 135-137 and 144-145, `machine.place` 138, `netstore.place` 139-143, `conduit.place` 146-147 (the `conduit_place` count), `progress.dragonEgg` 148-149. Today's order inside the machine case is storage, machine, netstore: 60, 70, 80.
- `UseOnBlock`: 153-218 top to bottom. `screens.menuItem` is 158-162 with `useHeldScreen` (295-308); `dimension.stone` is `useStone` (282-291); `handwork.stick` is 198-201; `handwork.gather` is 206-207; the `!p.Sneaking()` tests of 194, 202 and 214 move into their steps.
- `Use`: 248-277 top to bottom. `HandleItemUseOnEntity` 224-228, `HandleAttackEntity` 232-237, `touchFairy` 240-245.
- `Death`/`Respawn`: `tinker.keepFood` and `tinker.restoreFood` are the food kept through death (391-394, 398-401). The map becomes `core.NewPlayers[int]()`, so the `delete` in `session.Leave` goes (the `core.parts` step drops it).
- `ItemDamage`: `tinker.ironHammer` 357-359, `autofill.broken` 361-363 with `sameItemName`.

## 3. Lifecycle hooks (12)

| Hook (event) | Steps | After the steps, in the kernel |
|---|---|---|
| `Load` (`Tick`, no transaction: `Tx` is nil) | 10 farm.loadCrops | |
| `Start` (`Tick`, inside the `World.Do` of `Game.Start`) | 10 island.keepLoaded · 20 climate.apply | |
| `Join` (`Presence`) | 10 screens.menuItem · 20 fairy.backfillDust · 30 climate.showWeather · 40 core.shapes · 50 island.purgeLost · 60 progress.sendRecipes · 70 screens.board · 80 dimension.joined | |
| `Leave` (`Presence`) | 10 dimension.leave · 20 core.parts · 30 expand.stop · 40 look.clear · 50 core.shapes · 60 screens.board · 70 progress.recipes | |
| `NewDay` (`DayEvent`) | 10 island.purgeLost · 20 climate.newDay | |
| `Second` (`Tick`) | 10 farm.growCrops · 20 progress.checkHeld · 30 farm.barrels · 40 fairy.tick · 50 climate.tick · 60 dimension.second · 70 tinker.effects · 80 machine.monitor · 90 farm.plants · 100 screens.boards · 110 achievement.playSeconds · 120 progress.collection | `FlushIfDue(time.Now())` |
| `Fast` (`Tick`) | 10 network.step · 20 dimension.watchdog · 30 look.updateAll | |
| `Autosave` (`Tick`) | 10 netstore.flush · 20 conduit.shares | `Flush()` |
| `Close` (`Tick`) | 10 storage.displays · 20 conduit.close · 30 netstore.flush | `Session.Close()`, `Unregister` |
| `ColumnLoad`, `ColumnUnload` (`ColumnEvent`) | 10 conduit.load / conduit.unload | |
| `EntityDespawn` (`DespawnEvent`) | 10 island.lost | |

- `Second`: `progress.checkHeld` takes the `advancement.InventoryNames(tx)` snapshot, keeps it in a `core.NewPart` of `progress` and runs `CheckHeld`; `progress.collection` hands the same snapshot to `achievement.GrantCollection`. The snapshot is taken before `dimension.second` retires items, as today. `farm.barrels` is `TickBarrels` then `TickComposters`; `dimension.second` is `TickCollectors` then `RetireItems` for every player.
- `Fast`: `network.step` is the energy step, the counter and the item and fluid steps (`second.go` 54-63); the counter becomes a `core.NewPart[int]` of `network`.
- `Join`: `screens.menuItem` gives the menu item on the first join and renames menu items; `dimension.joined` is `Joined` then `RetireItems`. `Leave`: `core.parts` is today's `core.Session.Leave`; `look.clear` is `waila.Clear`; `core.shapes` is `RemoveAllDebugShapes` (join: `SendShapes`); `screens.board` is `RemoveScoreboard` (join: `UpdateBoard`); `progress.recipes` is `SetRecipeFilter(nil)`.
- `Close`: `conduit.close` is `WriteShares` then `RemoveVisuals`. `Autosave` and `Close` are called with a nil transaction by `netflush_test.go`; the steps pass it on as today.
- `Load` and `Start`: `Game.Start` keeps the order of its statements. `farm.LoadCrops` runs synchronously, before the transaction is queued and before the session is registered; the island and the climate run in the queued transaction, which is not waited for; the world handler and `core.Register` follow the queueing. Loading the crops inside the transaction would let a world tick see a registered session with an empty crop queue, so there are two hooks: `Load` is emitted where `farm.LoadCrops` is called, `Start` inside the `World.Do`. `island.SetKeep` stays in `Game.Start` (it hands the instance to the island).
- `island.lost` holds the whole test of `lostfound.go` (an item entity below the world's lowest height).

Counts: 81 player steps and 43 lifecycle steps, 124 in all, over 22 owners: achievement 17, dimension 15, island 10, farm 9, expand 8, tinker 7, conduit 7, screens 6, progress 5, fairy 5, autofill 4, storage 4, netstore 4, machine 4, climate 4, core 3, look 3, copier 2, wand 2, handwork 2, elevator 2, network 1.

## 4. Pins

A pin line is `<scenario>: <names reached, space separated> | <outcome>`. The outcome is `cancelled=<bool>` where the method has a context, plus `drops=<name>x<count>,...` for `BlockBreak`, `damage=<n>` for `ItemDamage`, `food=<n>` for `Respawn`. The harness sets `core.Trace` to a sink that appends names, calls the method through `s.Handler(p)` (or the session method), and clears the sink. Test players have no handler of their own, so one call writes one hook's names. One test session and one player per scenario (`testSession`, `loadAround`, `spawnPlayer`, `withPlayer`); no server join anywhere in the pins.

| Golden | Written by | Scenarios |
|---|---|---|
| `handler-simple.golden` | T35 | every method of section 2 not listed below |
| `handler-blocks.golden` | T36 | `StartBreak`, `BlockBreak`, `BlockPlace` |
| `handler-use.golden` | T37 | `UseOnBlock`, `Use`, `UseOnEntity`, `AttackEntity` |
| `lifecycle.golden`, `settings.golden` | T38 | section 3, and the keys of `account.Settings()` in order |
| `order.golden` | T45 | one line `<Hook> <priority> <name>` per `core.Subscriptions()` entry |

Scenarios (fixture: the root test that already builds the state):

- `StartBreak`: `own`, `outside` (a chunk not owned), `arena` (`core.ArenaCentre`), `expanding` (`screens_test.go` 76).
- `BlockBreak`: `stone`, `outside`, `expanding`, `hammerCobble` (`hammer_test.go` 33), `crookLeaves` (`crook_test.go` 117), `drawerFront` and `drawer` (`drawer_test.go` 306), `machine` (crusher), `setMachine` (drawer controller), `netNode` (network cable), `composter`, `barrel`, `bundle`.
- `BlockPlace`: `dirt`, `outside`, `expanding`, `lastOfStack` (one held), `crop` (`crops_test.go` 158), `machine`, `setMachine`, `netNode`, `drawer`, `bundle`, `dragonEgg` (`project_test.go`).
- `UseOnBlock`: `plain`, `hunting` (`dimension_test.go` 68), `menuItem`, `stone`, `expanding`, `outside`, `heldNetMachine`, `copier` (`copier_test.go`), `wand` (`wand_test.go` 190), `drawerSneak` (`drawer_test.go` 551), `plate` (`energy_test.go` 550), `stickChest` (`stickchest_test.go`), `stick` (`handwork_test.go`), `composter` (`composter_test.go`), `fiber` (`barkfiber_test.go`), `fairyTime` (`fairytime_test.go`), `crop` and `cropSneaking` (`cropinfo_test.go`), `sideOutside` (own block at the chunk border, the side not owned).
- `Use`: `plain`, `lastSnowball`, `menuItem`, `menuItemHunting`, `hunting`, `stone`, `copier`, `wandSneak`, `ironHammerSneak`.
- `UseOnEntity`: `other`, `hunting`, `fairy` (`fairy_test.go` 145). `AttackEntity`: `other`, `monster`, `hunting`, `fairy`.
- `Move`: `walk`, `expanding`, `void`. `Sneak`: `start`, `end`, `expanding`, `elevator` (`elevator_test.go`). `Sprint`: `start`, `end`. `Jump`: `plain`, `elevator`.
- `ItemDamage`: `plain`, `ironHammer`, `lastUse`. `ItemConsume`, `ItemPickup`, `PunchAir`, `Chat`, `BlockPick`, `HeldSlot`, `Command`: `plain`. `ItemDrop`: `plain`, `hunting`. `Death`: `plain`. `Respawn`: `afterDeath` (`hammer_test.go` 258). `Hurt`: `plain`, `frozen` (`dimension_test.go` 486), `fallen`.
- Lifecycle, by calling the session: `Join/first`, `Join/again`, `Second`, `Fast`, `NewDay`, `Autosave`, `Leave`, `Close`, and the world handler's `ColumnLoad`, `ColumnUnload`, `EntityDespawn/void`, `EntityDespawn/pickedUp` (above the lowest height: nothing). `Start` cannot be called without an instance: its line comes from one `instance.NewManager` with `&Game{log, db}` and one player entering a new save (the fixture of `platform/instance/manager_test.go`), the sink keeping only the three names of the `Load` and `Start` rows, as one line `Start` (the instance's own ticks write other names meanwhile).

Review step for T35-T38: every line whose scenario stops nowhere must equal its hook's row in sections 2 and 3, name for name.

## 5. Feature list

`content/skygear/features.go` after T39, in layer order; settings and item entry edits apply in this order:

`core.KernelFeature`, `achievement`, `progress`, `island`, `climate`, `network`, `tinker`, `core.AutofillFeature`, `machine`, `conduit`, `storage`, `farm`, `fairy`, `dimension`, `handwork`, `expand`, `copier`, `wand`, `netstore`, `elevator`, `look`, `screens`, `items` (the root's own: the item entry edit of `names.go`).

Sibling imports among the feature layer at 541f654, the table `TestFeatureSiblings` starts from: decor → tinker; machine → decor, tinker; conduit → machine; storage → machine; farm → machine; fairy → farm, machine; dimension → machine; copier → conduit, machine, storage; netstore → conduit, decor, machine, storage. None for tinker, handwork, expand, wand, elevator.
