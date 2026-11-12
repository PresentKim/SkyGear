# Hook tiers: reference (measurements, order dependences, exceptions, design, the code of each task)

Reference for the machine plan `docs/plans/2026-11-11-hook-tiers-plan.md` (not a machine plan, not linted). Everything here was measured or built in a scratch worktree at 6016b31; the per-task prototype patches are in `/tmp/claude-0/-home-user-SkyGear/a71fa8d5-6f20-5c3f-ba20-eeb27e13ad8a/scratchpad/proto-tiers-tasks/` (`T120.patch` to `T125.patch`, in id order on that commit, and `combined.patch`).

## 1. The decision and the frame

The owner's decision: the exact numeric priorities go; a subscriber stands in one of six tiers, and code and documents define only each tier's role, in these words.

| Tier | Role (the owner's words) |
|---|---|
| LOWEST | 전처리 - 먼저 이벤트 자체를 가공하거나 차단해야 할 때 사용합니다. |
| LOW | 전처리 이후 처리되어도 될 이벤트 가공이나 차단에 사용합니다. |
| NORMAL | 전처리 이후에 처리되어도 될 일반 우선순위에 사용합니다. |
| HIGH | 후처리에 해당하진 않지만, 우선순위가 낮은 작업에 사용합니다. |
| HIGHEST | 이벤트를 가공할 수 있는 마지막 후처리 작업에 사용합니다. |
| MONITOR | 이벤트를 가공하거나 취소하지 않는 결과 처리(기록, 업적 처리 등)에 사용합니다. |

The owner's addition: 이벤트가 취소되어도 수신할지를 선택할 수 있다 (전통적인 handleCancelled 옵션). Default: a subscriber does not receive a cancelled event; opt-in per subscriber.

The coordinator's frame, all of it kept: `core.Priority` with `Lowest … Monitor`; `On` and `Handle` take a `Priority`; inside a tier the order is deterministic (installation order of the features, then call order) and no subscriber may depend on it, which a test guards by permuting; `Handle` at `Monitor` is refused; a stopped walk skips later subscribers, Monitor included, unless they opted in; results of every scenario pin byte for byte; `Emit` keeps its shape. One measurement speaks against one point of the frame as first written ("Monitor never sees a stopped event"): see E1 in section 6. The addition (the option) answers it only in part.

## 2. Measurement: the two things a "cancel" can mean

The kernel has two: (1) the walk is stopped: a `Handle` subscriber returned true, later subscribers do not run, `Emit` returns true; (2) Dragonfly's context is cancelled (`e.Cancel()`), the walk goes on.

Of the 123 subscribers at b49fc81 (31 `Handle`, 92 `On`):

| What the subscriber does | Count | Who |
|---|---|---|
| stops and cancels on every path that returns true | 29 | `expand.guard` ×4, `expand.guardSide`, `island.guard` ×4, `island.guardSide`, `dimension.hunting` ×4 (`UseOnBlock`, `Use`, `UseOnEntity`, `ItemDrop`), `dimension.hunter`, `dimension.frozen`, `dimension.stone` ×2, `screens.menuItem` ×2, `copier.use`, `wand.use`, `storage.sneakUse`, `conduit.plate`, `handwork.stick`, `farm.composter`, `fairy.time`, `farm.crop`, `expand.hold` |
| stops without cancelling | 2 | `Sneak`: `expand.show`, `elevator.down` (cancelling a sneak toggle would keep the player from sneaking) |
| cancels without stopping (`On`) | 5 registrations, 4 steps | `storage.break` (a drawer hit on its front), `island.void`, `dimension.fallen`, `fairy.touch` ×2 |
| neither | 87 | every other `On` subscriber |
| reads `Ctx.Cancelled()` before acting | 0 | none (`wand/use.go` reads it of a `BlockPlace` it fires itself, not of its own event) |

Consequences for the option:

- The default "does not run once the event is cancelled" is true today only of meaning 1. After a cancel of meaning 2 every later subscriber runs today (after `storage.break` cancels, `netstore.break`, `machine.break`, `conduit.break` and the break counter still run). Skipping on meaning 2 would change behaviour.
- **One meaning for the option: an event is cancelled when the walk was stopped (a `Handle` subscriber returned true).** A subscriber registered with the option still runs after that and is told so. In case 2 (context cancelled, walk goes on) nothing is skipped for anybody; any subscriber can ask `e.Cancelled()` (new, `Actor.Cancelled`).
- How an opted-in subscriber learns the state, with events passed by value and no allocation: a second callback shape. `Hook.Always(prio, name, func(e E, cancelled bool))`. `cancelled` is false for a live event, true when an earlier subscriber stopped the walk.
- A dedicated method, not a variadic option on `On`: the flag needs another function type anyway; the call site reads as what it is (`core.Move.Always(core.Monitor, "look.update", …)`); the common call stays as short as today; and a `Handle` subscriber with the option cannot be written at all (an `Always` function returns nothing), so "what does its return value mean after a stop" needs no rule.
- Hooks with nothing to cancel (`Join`, `Leave`, `Second`, `Fast`, `Close` …): the option is not refused there. Its meaning is the walk's stop, which exists on every hook (`Handle` can be registered on any hook); on a hook without a `Handle` subscriber the flag is always false. Refusing would need the kernel to know which hooks will ever get a `Handle` subscriber.
- `Subscriptions()` reports it (`Subscription.Always`) and `order.golden` shows it (` always` after the name).

## 3. The 123 subscribers

"Today" is the number at b49fc81. Does: R reads only (counts, records), D shows (panel, sidebar), S shapes the event (writes through one of its pointers), C cancels in Dragonfly, H may stop the walk (`Handle`), W changes other state (world, inventory, parts, saved data).

| Hook | Subscriber | Today | Does | Tier | Why |
|---|---|---|---|---|---|
| StartBreak | expand.guard | 10 | H C: in island expansion mode cancels, shows the map chunk again | LOWEST | a mode that holds the player refuses before anything else looks |
| StartBreak | island.guard | 20 | H C: outside the island cancels, tip | LOW | a refusal that may come after the modes |
| StartBreak | tinker.autoTool | 30 | W: picks the pickaxe in hand | NORMAL | the feature's work |
| BlockBreak | expand.guard | 10 | H C | LOWEST | as above |
| BlockBreak | island.guard | 20 | H C | LOW | as above |
| BlockBreak | achievement.break | 30 | R | MONITOR | a counter |
| BlockBreak | tinker.break | 40 | S (hammer drop) W (tool xp, vein) | NORMAL | the feature's work |
| BlockBreak | farm.break | 50 | S (crook leaves, composter and barrel contents) | NORMAL | the feature's work |
| BlockBreak | storage.break | 60 | C (drawer hit on its front) W (display, networks) | NORMAL | the feature's work; nobody reads its cancel |
| BlockBreak | netstore.break | 70 | W (deferred) | NORMAL | the feature's work |
| BlockBreak | machine.break | 80 | W | NORMAL | the feature's work |
| BlockBreak | conduit.break | 90 | W | NORMAL | the feature's work |
| BlockPlace | expand.guard | 10 | H C | LOWEST | as above |
| BlockPlace | island.guard | 20 | H C | LOW | as above |
| BlockPlace | achievement.place | 30 | R | MONITOR | a counter |
| BlockPlace | autofill.place | 40 | W (deferred refill of the hand) | HIGH | follows what the other steps made of the event |
| BlockPlace | farm.plant | 50 | W and R (plants the crop, counts it) | NORMAL | the feature's work; the count records its own act |
| BlockPlace | storage.place | 60 | W | NORMAL | the feature's work |
| BlockPlace | machine.place | 70 | W (sets the block in a deferred call) | NORMAL | the feature's work |
| BlockPlace | netstore.place | 80 | W (deferred) | HIGH | after the machine is set up (F23) |
| BlockPlace | conduit.place | 90 | R | MONITOR | a counter |
| BlockPlace | progress.dragonEgg | 100 | W (the advancement that ends the game) | MONITOR | handles the result, shapes nothing (업적 처리) |
| UseOnBlock | dimension.hunting | 10 | H C | LOWEST | a mode refuses first |
| UseOnBlock | screens.menuItem | 20 | H C: opens the menu | LOWEST | the item's click is taken before anything else |
| UseOnBlock | dimension.stone | 30 | H C: opens the window | LOWEST | as the menu item |
| UseOnBlock | expand.guard | 40 | H C | LOWEST | a mode refuses first |
| UseOnBlock | island.guard | 50 | H C | LOW | a refusal after the modes |
| UseOnBlock | machine.noteFace | 60 | W (notes the face for the placement) | NORMAL | the feature's work |
| UseOnBlock | copier.use | 70 | H C | LOW | a held tool takes the click from what the block would do |
| UseOnBlock | wand.use | 80 | H C | LOW | as the copier |
| UseOnBlock | storage.sneakUse | 90 | H C | NORMAL | the feature's work |
| UseOnBlock | conduit.plate | 100 | H C | NORMAL | the feature's work |
| UseOnBlock | handwork.stick | 110 | H C, and R (counts `stick_other` when it takes nothing) | NORMAL | the feature's work (E2) |
| UseOnBlock | farm.composter | 120 | H C | HIGH | what the block does comes after what the held item does |
| UseOnBlock | handwork.gather | 130 | W | NORMAL | the feature's work |
| UseOnBlock | fairy.time | 140 | H C | NORMAL | the feature's work (E3) |
| UseOnBlock | farm.crop | 150 | H C | HIGH | as the composter |
| UseOnBlock | expand.guardSide | 160 | H C (never fires, section 9) | HIGHEST | the last refusal |
| UseOnBlock | island.guardSide | 170 | H C | HIGHEST | the last refusal, when no step took the click |
| UseOnEntity | dimension.hunting | 10 | H C | LOWEST | a mode refuses first |
| UseOnEntity | fairy.touch | 20 | C W | NORMAL | the feature's work |
| AttackEntity | dimension.hunter | 10 | H C | LOWEST | a mode refuses first |
| AttackEntity | fairy.touch | 20 | C W | NORMAL | the feature's work |
| Use | autofill.use | 10 | W (deferred refill) | HIGH | follows what the other steps made of the event |
| Use | screens.menuItem | 20 | H C | LOWEST | as on a block |
| Use | dimension.hunting | 30 | H C | LOWEST | a mode refuses first |
| Use | dimension.stone | 40 | H C | LOWEST | as on a block |
| Use | copier.clear | 50 | W | NORMAL | the feature's work |
| Use | wand.options | 60 | D | NORMAL | the feature's work |
| Use | tinker.veinToggle | 70 | W | NORMAL | the feature's work |
| Move | look.update | 10 | D | MONITOR, Always | shows the result; must run for a refused move too (F16) |
| Move | expand.hold | 20 | H C | LOWEST | a mode refuses first |
| Move | achievement.walk | 30 | R | MONITOR | a counter |
| Move | island.void | 40 | C W (teleports back, counts) | HIGHEST | the last step that may cancel the move |
| Sneak | expand.show | 10 | H (no cancel) | LOWEST | a mode takes the event first |
| Sneak | achievement.sneak | 20 | R | LOW | a counter outside MONITOR: exception E1 |
| Sneak | elevator.down | 30 | H (no cancel) W | NORMAL | the feature's work |
| Sneak | farm.crouchGrow | 40 | W | HIGH | only when nothing else used the sneak |
| Sprint | achievement.sprint | 10 | R | MONITOR | a counter |
| Jump | achievement.jump | 10 | R | MONITOR | a counter |
| Jump | elevator.up | 20 | W | NORMAL | the feature's work |
| ItemDamage | tinker.ironHammer | 10 | S (damage to 0) | LOW | shapes what others read |
| ItemDamage | autofill.broken | 20 | W (deferred refill; reads the damage) | HIGH | follows the shaped event |
| ItemConsume | achievement.eat | 10 | R | MONITOR | a counter |
| ItemConsume | autofill.consume | 20 | W (deferred refill) | HIGH | as above |
| ItemPickup | achievement.pickup | 10 | R | MONITOR | a counter |
| ItemDrop | dimension.hunting | 10 | H C | LOWEST | a mode refuses first |
| ItemDrop | achievement.drop | 20 | R | MONITOR | a counter |
| Death | achievement.death | 10 | R | MONITOR | a counter |
| Death | dimension.died | 20 | W | NORMAL | the feature's work |
| Death | tinker.keepFood | 30 | W | NORMAL | the feature's work |
| Respawn | dimension.respawned | 10 | S (the respawn position) W | LOW | shapes the event |
| Respawn | tinker.restoreFood | 20 | W | NORMAL | the feature's work |
| Hurt | dimension.frozen | 10 | H C | LOWEST | a mode refuses first |
| Hurt | achievement.hurt | 20 | R | MONITOR | a counter |
| Hurt | dimension.fallen | 30 | C W (reads the damage left) | HIGHEST | the last step that may cancel, on the final damage |
| PunchAir, Chat, BlockPick, HeldSlot, Command | achievement.punchAir, .chat, .blockPick, .heldSlot, .command | 10 each | R | MONITOR | counters |
| Load | farm.loadCrops | 10 | W | NORMAL | the feature's work |
| Start | island.keepLoaded | 10 | W | NORMAL | the feature's work |
| Start | climate.apply | 20 | W | NORMAL | the feature's work |
| Join | screens.menuItem | 10 | W (gives the menu item, renames it) | NORMAL | the feature's work |
| Join | fairy.backfillDust | 20 | W (gives dust) | NORMAL | the feature's work |
| Join | climate.showWeather | 30 | D | NORMAL | the feature's work |
| Join | core.shapes | 40 | D | NORMAL | the kernel's one subscriber (7.3) |
| Join | island.purgeLost | 50 | W | NORMAL | the feature's work |
| Join | progress.sendRecipes | 60 | D | NORMAL | the feature's work |
| Join | screens.board | 70 | D | MONITOR | shows what the join steps made |
| Join | dimension.joined | 80 | W (game mode, the stranded hunter's inventory, relics) | HIGH | keeps today's order: exception E4 |
| Leave | dimension.leave | 10 | W | NORMAL | the feature's work |
| Leave | expand.stop | 15 | W D (redraws the sidebar) | NORMAL | the feature's work |
| Leave | core.parts | 20 | W | not a subscriber | the tail of `Lifecycle.Leave` (7.3) |
| Leave | look.clear | 40 | D | HIGHEST | the last clean-up (F19) |
| Leave | core.shapes | 50 | D | not a subscriber | the tail of `Lifecycle.Leave` (7.3) |
| Leave | screens.board | 60 | D | HIGHEST | the last clean-up (F19) |
| Leave | progress.recipes | 70 | D | HIGHEST | the last clean-up (F19) |
| NewDay | island.purgeLost | 10 | W | NORMAL | the feature's work |
| Second | farm.growCrops | 10 | W | NORMAL | the feature's work |
| Second | progress.checkHeld | 20 | R W (reads what players hold, advances) | LOW | reads the state before the second's work changes it (F21) |
| Second | farm.barrels | 30 | W | NORMAL | the feature's work |
| Second | fairy.tick | 40 | W | NORMAL | the feature's work |
| Second | climate.tick | 50 | W | NORMAL | the feature's work |
| Second | dimension.second | 60 | W (collectors, relics in inventories) | NORMAL | the feature's work |
| Second | tinker.effects | 70 | W | NORMAL | the feature's work |
| Second | machine.monitor | 80 | W | NORMAL | the feature's work |
| Second | farm.plants | 90 | W | NORMAL | the feature's work |
| Second | screens.boards | 100 | D | MONITOR | shows the result |
| Second | achievement.playSeconds | 110 | R | MONITOR | a counter |
| Second | progress.collection | 120 | R (grants from the reading of checkHeld) | MONITOR | 업적 처리 |
| Fast | network.step | 10 | W | NORMAL | the feature's work |
| Fast | dimension.watchdog | 20 | W | NORMAL | the feature's work |
| Fast | look.updateAll | 30 | D | MONITOR | shows the result |
| Autosave | netstore.flush | 10 | W | NORMAL | the feature's work |
| Autosave | conduit.shares | 20 | W | NORMAL | the feature's work |
| Close | storage.displays | 10 | W | NORMAL | the feature's work |
| Close | conduit.close | 20 | W | NORMAL | the feature's work |
| Close | netstore.flush | 30 | W | NORMAL | the feature's work |
| ColumnLoad, ColumnUnload | conduit.load, conduit.unload | 10 | W | NORMAL | the feature's work |
| EntityDespawn | island.lost | 10 | W | NORMAL | the feature's work |

Counts after the plan (121 subscribers; the two kernel steps of `Leave` are no subscribers any more): LOWEST 16, LOW 10, NORMAL 56, HIGH 9, HIGHEST 7, MONITOR 23. One subscriber has the option (`look.update`).

The largest tiers, which is what the guard has to prove most about: `Second` NORMAL 8 of 12, `UseOnBlock` NORMAL 6 of 17, `Join` NORMAL 6 of 8, `BlockBreak` NORMAL 6 of 9, `UseOnBlock` LOWEST 4, then groups of 3 (`UseOnBlock` LOW, `Use` LOWEST and NORMAL, `BlockPlace` NORMAL and MONITOR, `Leave` HIGHEST, `Close` NORMAL, `Second` MONITOR). Twelve hooks have a single subscriber.

## 4. Order dependences found

Method. With the seam of T120 the root test binary was run with the planned tiers simulated on today's numbers and (a) one kept pair flipped (the second moved to just before the first), (b) one subscriber moved to another tier, (c) every tier reversed, (d) every tier shuffled with 16 seeds. Each run dumped, per pin scenario, the result part of the line and the achievement counters the scenario counted; a dump that differs from the run in today's order is a hit. Every hit was then read in the code. What the dumps cannot see (tips, messages, sounds, client packets: the test players have no connection; state no scenario reads) was found by reading only, and is marked so.

| Id | Hook | Before → after | Why the order matters | Shown by | Kind |
|---|---|---|---|---|---|
| F1 | UseOnBlock | dimension.hunting → screens.menuItem | a hunter holding the menu item clicks a block: the hunt refuses, no menu | flip: `UseOnBlock/huntingMenuItem` `screens=-` becomes `screens=dimension.ShowRunMenu` | (a) |
| F2 | UseOnBlock, Use | dimension.hunting → dimension.stone | a hunter holding the stone: refused, no window | flip in `Use`: `Use/huntingStone` `screens=-` becomes `dimension.ShowWindow`; on a block by reading (no scenario) | (a) |
| F3 | UseOnBlock | screens.menuItem, dimension.stone → expand.guard | in island expansion mode the two items still open their screen | flip: `UseOnBlock/menuItemExpanding` (`screens=expand.showExpandPlane`, counter `menu_open+1`) and `stoneExpanding` lose the screen | (a) |
| F4 | UseOnBlock | screens.menuItem, dimension.stone → island.guard | outside the island the two items still open their screen | island.guard moved to LOWEST and reversed: `menuItemOutside`, `stoneOutside` lose the screen | (c) |
| F5 | StartBreak, BlockBreak, BlockPlace, UseOnBlock | expand.guard → island.guard | expansion mode and a block outside the island: the mode refuses (map chunk shown again, no "outside" tip) | reading (tip and map are not observable) | (c) |
| F6 | UseOnBlock | dimension.hunting → island.guard | a hunter's click on a block that is not the island's: no tip | reading | (c) |
| F7 | UseOnBlock | island.guard → copier.use, wand.use | outside the island the guard refuses with its tip; the tools do nothing | reading (flip is silent: no result differs) | (a) with (c) |
| F8 | UseOnBlock | copier.use, wand.use → storage.sneakUse, conduit.plate, farm.composter, farm.crop | the tool in hand takes the click on a drawer, a plate, a composter, a crop | reading (no scenario holds a tool on those) | (c) |
| F9 | UseOnBlock | storage.sneakUse, conduit.plate → handwork.stick | `handwork.stick` counts `stick_other` when it takes nothing; a click the drawer menu or a plate took does not reach it | flip: `UseOnBlock/drawerSneak` and `drawerSneakAfterWindow` count `stick_other+1`; the plate by reading | (e) E2 |
| F10 | UseOnBlock | conduit.plate → fairy.time | an hourglass on the connector plate of a machine opens the plate's settings | reading | (e) E3 |
| F11 | UseOnBlock | handwork.stick, fairy.time, conduit.plate → farm.composter, farm.crop | the fairy's bone meal acts on a crop before the crop takes the click; a stick on a crop or a full composter is counted first; a plate on a composter wins | farm's two moved to NORMAL and reversed: `UseOnBlock/fairyTime` keeps its bone meal (`0x2` becomes `0x3`); the rest by reading | (c) |
| F12 | UseOnBlock | every step → expand.guardSide, island.guardSide | the side is refused only when no step took the click (a grown crop at the border is harvested) | pins `cropGrownSideOutside`, `sideOutside` | (c) |
| F13 | Use | screens.menuItem → dimension.hunting | in the air the menu item is the hunter's way out | flip: `Use/menuItemHunting` loses `dimension.ShowRunMenu` | (a) |
| F14 | Use | autofill.use first | nothing: the refill is deferred and reads the hand later | moved to HIGH: no dump differs | independent |
| F15 | Move | achievement.walk → island.void | the walk counter read the player's place, which the void's teleport changes | `TestWalkCountsTheMoveIntoTheVoid` with the old code and the planned tiers: 2 counted for a move of one block | (d) |
| F16 | Move | look.update → expand.hold | the panel is cleared on a move the mode refuses | reading (the fast tick clears it 0.2 s later, no test sees the difference) | (f) |
| F17 | Sneak | expand.show → achievement.sneak → elevator.down → farm.crouchGrow | a sneak counts when the elevator rides and not in expansion mode; saplings do not grow on a ride | counter moved to MONITOR: `Sneak/elevator` loses `sneak+1`; crouchGrow by reading (chance) | (c), and (e) E1 for the counter |
| F18 | Leave | dimension.leave, expand.stop → core.parts | the player's entries hold the game mode to restore | flip: `Leave/expanding` `mode=world.survival` becomes `expand.ViewOnlyMode` | (d) |
| F19 | Leave | dimension.leave, expand.stop → look.clear, screens.board, progress.recipes, core.shapes | `expand.Stop` redraws the sidebar, a hunt that ends may advance and send recipes: the clean-up must come after | reading | (c), (d) for the kernel's |
| F20 | Join | fairy.backfillDust, screens.menuItem → dimension.joined | a hunter the server stopped under gets its inventory back: what the earlier steps gave or renamed is wiped with the kit | reading | (e) E4 |
| F21 | Second | progress.checkHeld → dimension.second; progress.checkHeld → progress.collection | the held items are read before relics are retired; the collection grants from that reading | reading (no scenario holds a retired relic) | (c) |
| F22 | ItemDamage | tinker.ironHammer → autofill.broken | the refill reads the damage the hammer set to 0 | reading (silent: the iron hammer never nears its last use) | (c) |
| F23 | BlockPlace | machine.place → netstore.place | the code says the network is dropped after the machine's deferred set | reversed: nothing differs; `Invalidate` only drops networks and they are rebuilt at the next use, so no effect was found | (c), kept as documented |
| F24 | Hurt | dimension.frozen → the rest; dimension.fallen last | the freeze refuses; the fall reads the damage left | pins `Hurt/frozen`, `Hurt/fallen` | (c) |
| F25 | every hook with guards | the guard → every later step | a refused event reaches no step | pins (`…/outside`, `…/expanding`, `ItemDrop/hunting`) | (c) |

Checked and independent (reversed, shuffled, and read): the shapers of `Drops` in `BlockBreak` (hammer and crook exclude each other by the held item, composter and barrel by the block); `storage.break`, `netstore.break`, `machine.break` on a machine block (separate state); the three placers of `BlockPlace` (block types); `Death` (the run's end does not touch food); `Respawn`; `Join` NORMAL except F20; `Second` NORMAL (separate state; see V5 for the random source); `Fast`, `Autosave`, `Close`, `Start`; `machine.noteFace` against the steps of its tier (a note a plate's click leaves is overwritten or consumed by the next placement); the item steps of `Use` (held items exclude each other).

The claim chains ("first claimant wins"):

- `UseOnBlock` (15 `Handle`): by their own conditions exclusive are menu item / stone (held item), the hunt / expansion mode (a player is in one mode at most, `core.Takeover`), copier / wand / stick / fairy items (held item), `sneakUse` / plate, composter, crop (sneaking), composter / crop (block). They overlap in F1 to F12.
- `BlockPlace`, `BlockBreak`, `StartBreak` (2 `Handle` each, the guards): overlap F5.
- `Use` (3 `Handle`): F2, F13.

## 5. Resolutions

- (a) a condition made explicit in the subscriber, behaviour unchanged:
  - F1: `screens.menuItem` on a block does not take a hunter's click (T121).
  - F2: `dimension.stone` takes no hunter's click, on a block and in the air (T121).
  - F13: `dimension.hunting` in the air leaves the menu item (T121). `IsMenuItem` moves to `sgitem` so `dimension` can ask.
  - F3: `expand.guard` on `UseOnBlock` leaves an item that opens its screen (`sgitem.OpensScreen`: the menu item, the stone) (T121).
  - F7: `copier.use` and `wand.use` take the click only where `island.CanBuild` (T122).
- (b) merging two steps into one subscriber: not used. No hit was two steps of one feature.
- (c) a role-faithful tier: F4 to F6 (the modes LOWEST, `island.guard` LOW), F8 (the held tools LOW), F11 (farm's two block interactions HIGH), F12 (the side guards HIGHEST), F17 (`expand.show` LOWEST, `elevator.down` NORMAL, `farm.crouchGrow` HIGH), F19 (the three clean-ups of `Leave` HIGHEST), F21 (`progress.checkHeld` LOW, `progress.collection` MONITOR), F22 (LOW, HIGH), F23 (`netstore.place` HIGH), F24, F25.
- (d) the kernel provides it by construction: F18 and F19 (clearing the shapes and dropping the player's entries are the tail of `Lifecycle.Leave`, after every subscriber and also after a stop); F15 (`MoveEvent.From`, the place the player stood when the event was built).
- (f) the later subscriber takes handleCancelled: F16 (`look.update` is `Always` in MONITOR: it runs for the move expansion mode refused, as it does today from the first place).

What the tiers cost, in one line: five explicit conditions in four packages, one event field, one tail in the kernel, one use of the option, and four exceptions.

## 6. EXCEPTIONS (the owner decides)

None of (a) to (d) or (f) resolves these without bending a tier's role or changing behaviour. The plan keeps today's behaviour for each by the most conservative option, named first; no task implements an alternative.

### E1 `Sneak`: `achievement.sneak` between `expand.show` and `elevator.down`

- What they do: `expand.show` (stops, no cancel) shows the expansion screen again in island expansion mode; `achievement.sneak` counts a sneak; `elevator.down` (stops, no cancel) rides down; `farm.crouchGrow` grows saplings by chance.
- Why order matters: today the counter runs after the first and before the second. A sneak in expansion mode is not counted; a sneak that rides the elevator is. By role the counter is MONITOR, but there it does not run when the elevator stops the walk.
- The other way round for a player: with the counter in MONITOR (default delivery) a sneak that rides an elevator down no longer counts toward the sneak achievements.
- Why the option does not settle it: with `Always` the counter runs after both stops and is told `cancelled=true` in both; neither stop cancels in Dragonfly, so nothing tells them apart.
- Kept by (the plan): the counter stays an `On` subscriber in LOW, between LOWEST and NORMAL. A record outside MONITOR: the role is bent, the behaviour is exact.
- Alternatives: (1) MONITOR with `Always`, counting every sneak: one more count per sneak pressed in expansion mode (the player did sneak; arguably the truer record). (2) The option tells where the walk stopped (the flag becomes the tier of the stopping subscriber): the counter counts unless the stop came from LOWEST or LOW; exact, role-faithful, a wider API. (3) `expand.show` also cancels in Dragonfly and the counter reads `e.Cancelled()`: changes whether an expanding player is sneaking; not advised.
- Recommendation: (1).

### E2 `UseOnBlock`: `storage.sneakUse`, `conduit.plate` before `handwork.stick` (all NORMAL)

- What they do: `sneakUse` opens the drawer menu for a sneaking player with a non-block item; `plate` opens a connector plate's settings; `handwork.stick` takes a stick's click on grass, dirt or (sneaking) a chest, and otherwise counts `stick_other` and takes nothing.
- Why order matters: the count is a side effect of a step that takes nothing. It happens when the click reaches the step: not when the drawer menu or a plate took it first, but also when a composter or a crop takes it afterwards.
- The other way round for a player: a stick used to open a drawer's menu or a plate's settings also counts one `stick_other`.
- Kept by: the order inside the tier. `conduit` and `storage` are installed before `handwork`, so they run first; the guard keeps exactly these pairs when it permutes (`orderKept`). A known dependence on the place inside a tier, listed in one table.
- Alternatives: (1) accept the other way round (count whenever a stick is used on a block that is no grass or dirt): the record becomes its own MONITOR subscriber. (2) An explicit "after <name>" constraint in the API. (3) An extra tier between NORMAL and HIGH for this step alone (not advised).
- Recommendation: (1).

### E3 `UseOnBlock`: `conduit.plate` before `fairy.time` (both NORMAL)

- What they do: `plate` as above; `fairy.time` uses a held fairy item (the hourglass winds a machine).
- Why order matters: both can take a click with the hourglass on the centre of a plate that sits on a machine's face. Today the plate's settings open and the hourglass is not used.
- The other way round for a player: the hourglass winds the machine; the plate's settings open only when there is nothing to wind.
- Kept by: the order inside the tier (`conduit` is installed before `fairy`), kept by the guard (`orderKept`).
- Alternatives: (1) the hourglass first, by tier: `fairy.time` joins the held tools in LOW with the condition `island.CanBuild` as the copier has it ("the item in hand takes the click"). (2) an explicit "after" constraint.
- Recommendation: (1).

### E4 `Join`: `dimension.joined` after the join steps that touch the inventory

- What they do: `dimension.joined` gives a hunter the server stopped under what it carried (it clears the inventory, which holds the hunting kit, and restores the stash), fixes the game mode and retires relics. `fairy.backfillDust` gives a save's first player after the update the fairy dust of past nights; `screens.menuItem` renames menu items into the player's language.
- Why order matters: for such a stranded hunter today's order gives the dust into the kit and renames the kit's menu item, and then wipes both with the kit: the dust is lost (it is given once per save) and the restored menu item keeps its old name. This looks like a defect of today's order, in a rare state.
- The other way round for a player: the stranded hunter keeps the dust and gets the menu item renamed.
- Kept by: `dimension.joined` in HIGH, after the NORMAL join steps. By role (restoring the player's own state before the join steps act) it belongs in LOW; HIGH keeps what happens today.
- Alternatives: (1) LOW: the two losses go away (a behaviour change, a fix).
- Recommendation: (1).

## 7. Design

### 7.1 API

```go
type Priority uint8
const ( Lowest Priority = iota; Low; Normal; High; Highest; Monitor )

func (h *Hook[E]) On(prio Priority, name string, f func(E))
func (h *Hook[E]) Handle(prio Priority, name string, f func(E) bool)          // panics at Monitor
func (h *Hook[E]) Always(prio Priority, name string, f func(e E, cancelled bool))
func (a Actor) Cancelled() bool                                              // Dragonfly's cancel
type Subscription struct { Hook, Name, Feature string; Tier Priority; Always bool }
func Reorder(order func(hook string, subs []Subscription) []int)             // test seam
```

No numeric priority remains. Registration panics: nil function, a name twice in one hook, after the first `Emit`, a name outside the installing feature, a value above `Monitor`, `Handle` at `Monitor`. Two subscribers in one tier is the normal case.

### 7.2 Inside a tier

The order is the order of registration: the features in the order of `features.go` (`core.Install`), each feature's calls in the order `subscribe()` makes them. The kernel inserts a new step after the last step of its tier. That order makes a run repeatable and nothing else: **no subscriber may depend on it.** The exceptions E2 and E3 do, knowingly; they are the three entries of `orderKept`.

### 7.3 The kernel's own work

- `Join`: `core.shapes` (the session's shapes are sent to the joining player) stays a subscriber, in NORMAL. `core` is installed first, so it is the first of its tier; it is bound by the rule like any other and is independent of the other join steps.
- `Leave`: clearing the client's shapes and dropping the player's `Players` entries are no subscribers. `Lifecycle.Leave` does both after `Emit`, marked `core.shapes` and `core.parts` for the traces. They follow every subscriber of every tier, and a subscriber that stops the walk cannot keep them from running (today a `Handle` on `Leave` returning true would have leaked the entries).
- `Second`, `Autosave`, `Close`, `NewDay` keep their tails.

### 7.4 Monitor

A Monitor subscriber handles the result and neither shapes nor cancels. Enforced: `Handle` at `Monitor` panics at registration (`core.TestHandleAtMonitorPanics`); the root's `TestMonitorSubscribersDoNotShape` reads the function literal of every subscriber registered with `core.Monitor` and fails on a call of `Cancel` or an assignment through one of the event's pointers (it sees the literal, not what the literal calls). Not enforceable at registration: what a function does at run time.

Difference from the Bukkit tradition, after the option: the role is the same, and with `Always` a Monitor subscriber sees cancelled events and is told so. The default is the other way round: without `Always` a Monitor subscriber, like every other, does not run for a cancelled event. That default keeps today's behaviour: every counter that today stands after a refusing step (`ItemDrop`, `Hurt`, `Move`, the block hooks) keeps not counting a refused event.

### 7.5 Emit

`steps` stays one slice sorted by tier, walked in order; the step struct keeps its size (56 bytes: the tier and a flag in place of the number). The loop is today's loop with `break` in place of `return true`; after it, one comparison decides whether any `Always` subscriber stands after the stopping step (`lastAlways`, kept up to date at registration), and only then a second, not inlined function walks them (`after`, a small slice of the `Always` subscribers in run order). An `Always` subscriber is stored as an `On` step whose function calls it with false, so the loop has no third case. A hook without an `Always` subscriber returns at the stop as today. Nothing allocates (`TestEmitAllocatesNothing`, `TestEmitAllocatesNothingAfterCancel`). Putting the call of the second walk inside the loop cost about 20 ns per `Use` event in the measurement (the event is copied for the call); with the `break` it costs nothing measurable (section 8).

### 7.6 The guard

`core.Reorder` (T120) is the seam: before the first `Emit` it lets a test give every hook a new order; from T124 it panics when the new order moves a subscriber out of its tier. `coretest.ReorderHooks` (T124) turns the environment variable `SKYGEAR_HOOK_ORDER` (`reverse`, `shuffle:<seed>`) into such an order for a whole test binary; `coretest.Session` calls it, the root's `TestMain` calls it with the exceptions.

`TestTierOrder` (root) runs the root test binary six times in processes of their own: once in the order of registration and once each with `reverse`, `shuffle:1` to `shuffle:4`. It compares exactly this:

1. Every root test passes in every reordered run (their own assertions; about 190 test functions). In a reordered run a pin test compares the result part of each scenario line with the golden (the part after the first ` | `, byte for byte) and ignores the trace; `TestPinOrder` compares the lines of one tier of one hook as a set; the pin tests' own assertions on traces are off.
2. Every pin scenario (234 in the four scenario goldens; the lines of the other goldens that go through the same helper are compared too) dumps `<scenario> | <results> | <counters>`: the counters are what the achievement counters of every player the tests spawned rose by during the scenario (read through `account.Progress` from a store the reordered process opens for itself). Each reordered run's dump must equal the dump of the run in the order of registration, line for line.

What it cannot see: tips, chat lines, sounds and packets (the test players have no connection), and state that no scenario's outcome and no counter reads. Dependences of that kind were found by reading (section 4) and are covered by a tier, a condition with its own test (`TestHunterStoneOpensNothing`, `TestToolsLeaveTheClickOutsideTheIsland`, `TestWalkCountsTheMoveIntoTheVoid`: each asserts something that holds in every order and fails in some order without the condition), or the exception list.

Until T124 the kernel has no tiers: the guard of T120 simulates them. `plannedTiers` gives every subscriber its planned tier; a reordered run sorts each hook by planned tier (mode `tiers`), then reverses or shuffles each tier, and keeps the pairs of `orderKept`. `orderKept` starts with 14 entries, the dependences tiers alone do not carry; T121 removes six, T122 two, T124 three (the kernel's tail and `Always`), three stay (E2, E3).

The same variable reorders the tests of every feature package: `SKYGEAR_HOOK_ORDER=reverse go test -count=1 ./content/skygear/...` (28 packages) passes with `reverse` and `shuffle:1` to `shuffle:4` on the final prototype.

## 8. Benchmarks

Before is 6016b31, after is the final prototype state; six alternating runs each on the same machine, median (min-max) ns/op.

| Benchmark | Before | After | Median | B/op | allocs/op |
|---|---|---|---|---|---|
| `HandleMove` | 172.80 (164.20-180.80) | 165.35 (162.60-178.10) | -4% | 24 -> 24 | 0 -> 0 |
| `Second` | 24841.50 (24239.00-26468.00) | 25135.50 (24450.00-26049.00) | +1% | 10557 -> 10557 | 63 -> 63 |
| `UseOnBlock` | 553.80 (535.30-599.30) | 605.30 (527.90-610.10) | +9% | 0 -> 0 | 0 -> 0 |
| `Use` | 140.30 (127.70-146.50) | 134.65 (130.60-145.90) | -4% | 0 -> 0 | 0 -> 0 |
| `Emit/handle=false/subs=0` | 2.75 (2.49-3.02) | 2.84 (2.50-3.06) | +3% | 0 -> 0 | 0 -> 0 |
| `Emit/handle=false/subs=1` | 4.85 (4.19-5.05) | 5.06 (4.55-5.50) | +4% | 0 -> 0 | 0 -> 0 |
| `Emit/handle=false/subs=4` | 12.27 (10.37-13.75) | 12.63 (11.21-13.68) | +3% | 0 -> 0 | 0 -> 0 |
| `Emit/handle=false/subs=8` | 19.55 (18.19-21.25) | 21.98 (18.49-23.68) | +12% | 0 -> 0 | 0 -> 0 |
| `Emit/handle=false/subs=32` | 73.22 (64.05-78.45) | 78.70 (70.26-81.96) | +7% | 0 -> 0 | 0 -> 0 |
| `Emit/handle=true/subs=0` | 2.86 (2.70-3.07) | 2.70 (2.28-2.90) | -6% | 0 -> 0 | 0 -> 0 |
| `Emit/handle=true/subs=1` | 5.18 (4.42-5.51) | 5.60 (4.42-5.89) | +8% | 0 -> 0 | 0 -> 0 |
| `Emit/handle=true/subs=4` | 12.45 (11.87-13.81) | 13.82 (12.68-14.45) | +11% | 0 -> 0 | 0 -> 0 |
| `Emit/handle=true/subs=8` | 21.64 (19.94-25.71) | 22.90 (22.50-24.87) | +6% | 0 -> 0 | 0 -> 0 |
| `Emit/handle=true/subs=32` | 80.26 (78.08-89.99) | 90.72 (87.40-100.20) | +13% | 0 -> 0 | 0 -> 0 |
| `EmitMove` | 14.25 (12.50-14.71) | 15.46 (12.86-16.36) | +8% | 0 -> 0 | 0 -> 0 |

Allocations are unchanged everywhere. The ranges overlap in every row; the machine's noise between two runs of one binary is about 10 percent. `EmitMove` is not the same benchmark before and after (one of its four subscribers is an `Always` subscriber now). `UseOnBlock` carries the conditions of T121 on every click (`expand.guard` asks `sgitem.OpensScreen`, one value lookup on the held stack). The move event grew by `From` (24 bytes); `CountWalk` no longer reads the position itself. A first form of `Emit` that called the walk of the `Always` subscribers from inside the loop measured `Use` 16 percent slower (about 160 against 138 ns, six alternating runs); the form of REF T124 does not.

## 9. Outside this change (not fixed)

- `expand.guardSide` (`UseOnBlock`) can never fire: it has the condition of `expand.guard`, which stops the walk before it. It is kept, in HIGHEST.
- A hunter's menu item behaves differently on a block (nothing opens) and in the air (the run menu opens). Kept by the two conditions of T121.
- `BlockBreak`: a drawer hit on its front cancels the break, and the break is still counted (`block_break`), the tool still gains experience and a hammer still vein-mines: the steps before and after `storage.break` do not read the cancel. Unchanged.
- E4 describes a loss for a hunter the server stopped under (fairy dust given once per save, the menu item's name).
- In island expansion mode a click on the map with the menu item in hand does not show the map chunk again (`screens.menuItem` takes the click before `expand.guard` would call `RestoreMapNear`). Unchanged.
- `netstore.place` says it must run after `machine.place`'s deferred set; no effect of the order was found (F23).
- The dump of `TestTierOrder` counts every line that goes through `pinGolden` (3597 with the save fixture's golden), of which 234 are hook scenarios.

## 10. Tasks

| Task | What | Goldens | Commit subject |
|---|---|---|---|
| T120 | the seam `core.Reorder` and the guard on today's numbers, planned tiers simulated, 14 kept pairs | none | `test(core): guard that no subscriber depends on its place inside its planned tier` |
| T121 | F1, F2, F3, F13: four conditions, `sgitem.IsMenuItem`, `sgitem.OpensScreen` | none | `refactor(screens): the menu item, the stone and the hunt exclude each other by their own conditions` |
| T122 | F7: the copier and the wand where the island allows building | none | `refactor(copier): the copier and the wand leave a click outside the island to the island's guard` |
| T123 | F15: `MoveEvent.From` | none | `refactor(core): a move carries where the player stood, the walk counter reads it` |
| T124 | `Priority`, `Always`, the tail of `Leave`, 120 call sites, the guard on real tiers, `coretest.ReorderHooks`, the Monitor guard | `order.golden` (format, 121 lines); traces of `handler-simple`, `handler-blocks`, `handler-use`, `lifecycle` | `refactor(core): six tiers replace the hook priorities, a subscriber may receive cancelled events` |
| T125 | documents | none | `docs: the six hook tiers, the handleCancelled option and the order exceptions` |

## REF T120

Prototype patch: `T120.patch`.

Tests and one seam. `core/reorder.go` and `tierorder_test.go` are new and given whole; the rest are edits.

`orderKept` starts with these 14 entries: `UseOnBlock dimension.hunting screens.menuItem`, `UseOnBlock dimension.hunting dimension.stone`, `UseOnBlock screens.menuItem expand.guard`, `UseOnBlock dimension.stone expand.guard`, `UseOnBlock island.guard copier.use`, `UseOnBlock island.guard wand.use`, `UseOnBlock storage.sneakUse handwork.stick`, `UseOnBlock conduit.plate handwork.stick`, `UseOnBlock conduit.plate fairy.time`, `Use screens.menuItem dimension.hunting`, `Use dimension.hunting dimension.stone`, `Move look.update expand.hold`, `Leave dimension.leave core.parts`, `Leave expand.stop core.parts`.

Two planned tiers are provisional on purpose, so that today's behaviour holds until the task that removes the dependence: `Move HIGH achievement.walk` (MONITOR from T123) and `Move LOWEST look.update` with its kept pair (MONITOR with `Always` from T124). The three kernel lines (`Join NORMAL core.shapes`, `Leave NORMAL core.parts`, `Leave NORMAL core.shapes`) simulate the kernel's steps as subscribers until T124.

In a run without `SKYGEAR_HOOK_ORDER` nothing changes: `TestPinOrder` and the trace assertions bind as before. `TestTierOrder` returns at once inside a run it started itself (either variable set).

### `content/skygear/core/hook.go`

~~~~diff
--- a/content/skygear/core/hook.go
+++ b/content/skygear/core/hook.go
@@ -56,6 +56,7 @@ type Hook[E any] struct {
 
 type anyHook interface {
 	subscriptions() []Subscription
+	reorder(order func(hook string, subs []Subscription) []int)
 	reset()
 }
 
~~~~

### `content/skygear/core/reorder.go` (new file)

~~~~go
package core

import "fmt"

// Reorder is a seam for the tests that prove no subscriber depends on its
// place among the others. It changes the order the subscribers of every hook
// run in: order is called once per hook that has subscribers, with the hook's
// name and its subscribers in the order they run, and returns the new order
// as indexes into subs.
//
// Call it after Install and before the first Emit. It panics when a hook has
// emitted already or when order does not return every index once. Nothing
// outside tests calls it.
func Reorder(order func(hook string, subs []Subscription) []int) {
	hooksMu.Lock()
	defer hooksMu.Unlock()
	for _, h := range hooks {
		h.reorder(order)
	}
}

func (h *Hook[E]) reorder(order func(hook string, subs []Subscription) []int) {
	if len(h.steps) == 0 {
		return
	}
	if h.sealed.Load() {
		panic(fmt.Sprintf("core: Reorder: hook %q has emitted", h.name))
	}
	idx := order(h.name, h.subscriptions())
	if len(idx) != len(h.steps) {
		panic(fmt.Sprintf("core: Reorder: hook %q: %d indexes for %d subscribers", h.name, len(idx), len(h.steps)))
	}
	seen := make([]bool, len(h.steps))
	steps := make([]step[E], len(h.steps))
	for to, from := range idx {
		if from < 0 || from >= len(h.steps) || seen[from] {
			panic(fmt.Sprintf("core: Reorder: hook %q: index %d is out of range or repeated", h.name, from))
		}
		seen[from] = true
		steps[to] = h.steps[from]
	}
	h.steps = steps
}
~~~~

### `content/skygear/core/reorder_test.go` (new file)

~~~~go
package core

import (
	"slices"
	"testing"
)

// TestReorderChangesTheRunOrder: Reorder hands every hook's subscribers in
// run order and the hook then runs them in the order of the returned indexes.
func TestReorderChangesTheRunOrder(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("reorder")
	empty := NewHook[int]("reorder-empty")
	var ran []string
	h.On(10, "a", func(int) { ran = append(ran, "a") })
	h.On(20, "b", func(int) { ran = append(ran, "b") })
	h.On(30, "c", func(int) { ran = append(ran, "c") })
	var asked []string
	Reorder(func(hook string, subs []Subscription) []int {
		asked = append(asked, hook)
		if hook != "reorder" {
			t.Fatalf("asked for the order of %q, a hook without subscribers", hook)
		}
		if len(subs) != 3 || subs[0].Name != "a" || subs[2].Name != "c" {
			t.Fatalf("subscribers %+v", subs)
		}
		return []int{2, 0, 1}
	})
	h.Emit(0)
	empty.Emit(0)
	if !slices.Equal(ran, []string{"c", "a", "b"}) {
		t.Fatalf("ran %v", ran)
	}
	if !slices.Contains(asked, "reorder") {
		t.Fatalf("asked %v", asked)
	}
	var names []string
	for _, s := range Subscriptions() {
		if s.Hook == "reorder" {
			names = append(names, s.Name)
		}
	}
	if !slices.Equal(names, []string{"c", "a", "b"}) {
		t.Fatalf("Subscriptions lists %v", names)
	}
}

func TestReorderAfterEmitPanics(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("reorder-late")
	h.On(10, "a", func(int) {})
	h.Emit(0)
	defer mustPanic(t, "Reorder", "reorder-late", "emitted")
	Reorder(func(string, []Subscription) []int { return []int{0} })
}

func TestReorderWantsEveryIndexOnce(t *testing.T) {
	for _, bad := range [][]int{{0}, {0, 0}, {0, 2}, {1, 0, 0}} {
		ResetHooks()
		h := NewHook[int]("reorder-bad")
		h.On(10, "a", func(int) {})
		h.On(20, "b", func(int) {})
		func() {
			defer mustPanic(t, "Reorder", "reorder-bad")
			Reorder(func(string, []Subscription) []int { return bad })
		}()
	}
}
~~~~

### `content/skygear/helpers_test.go`

~~~~diff
--- a/content/skygear/helpers_test.go
+++ b/content/skygear/helpers_test.go
@@ -53,7 +53,11 @@ func init() { world.DefaultBlockRegistry.Finalize() }
 
 func loadAround(s *session) { coretest.LoadAround(s.Lifecycle) }
 
-func spawnPlayer(s *session) *world.EntityHandle { return coretest.SpawnPlayer(s.Lifecycle) }
+func spawnPlayer(s *session) *world.EntityHandle {
+	h := coretest.SpawnPlayer(s.Lifecycle)
+	pinNotePlayer(h)
+	return h
+}
 
 // withPlayer runs f with the player in a transaction. A panic in f fails
 // the test (Dragonfly would only log it).
~~~~

### `content/skygear/pin_blocks_test.go`

~~~~diff
--- a/content/skygear/pin_blocks_test.go
+++ b/content/skygear/pin_blocks_test.go
@@ -280,19 +280,19 @@ func TestPinBlocks(t *testing.T) {
 		"BlockPlace/dirt":  "expand.guard island.guard achievement.place autofill.place farm.plant storage.place machine.place netstore.place conduit.place progress.dragonEgg",
 	}
 	for name, row := range rows {
-		if got := lines[name]; !strings.HasPrefix(got, row+" | ") {
+		if got := lines[name]; hookOrder == "" && !strings.HasPrefix(got, row+" | ") {
 			t.Errorf("%s: %q does not start with the row %q", name, got, row)
 		}
 	}
 	for _, name := range []string{"StartBreak", "BlockBreak", "BlockPlace"} {
-		if got, want := lines[name+"/outside"], "expand.guard island.guard | cancelled=true"; !strings.HasPrefix(got, want) {
+		if got, want := lines[name+"/outside"], "expand.guard island.guard | cancelled=true"; hookOrder == "" && !strings.HasPrefix(got, want) {
 			t.Errorf("%s/outside: %q, want prefix %q", name, got, want)
 		}
-		if got, want := lines[name+"/expanding"], "expand.guard | cancelled=true"; !strings.HasPrefix(got, want) {
+		if got, want := lines[name+"/expanding"], "expand.guard | cancelled=true"; hookOrder == "" && !strings.HasPrefix(got, want) {
 			t.Errorf("%s/expanding: %q, want prefix %q", name, got, want)
 		}
 	}
-	if got := lines["BlockBreak/drawerFront"]; !strings.HasPrefix(got, rows["BlockBreak/stone"]+" | cancelled=true") {
+	if got := lines["BlockBreak/drawerFront"]; hookOrder == "" && !strings.HasPrefix(got, rows["BlockBreak/stone"]+" | cancelled=true") {
 		t.Errorf("BlockBreak/drawerFront: %q", got)
 	}
 	if got := lines["BlockBreak/hammerCobble"]; !strings.Contains(got, "drops="+naming.ItemKey(block.Gravel{})+"x1") {
~~~~

### `content/skygear/pin_order_test.go`

~~~~diff
--- a/content/skygear/pin_order_test.go
+++ b/content/skygear/pin_order_test.go
@@ -32,6 +32,9 @@ func TestPinOrder(t *testing.T) {
 		fmt.Fprintf(&b, "%s %d %s\n", s.Hook, s.Prio, s.Name)
 	}
 	got := b.String()
+	if hookOrder != "" {
+		return // a reordered run (TestTierOrder) lists another order on purpose
+	}
 	const file = "testdata/pin/order.golden"
 	if os.Getenv("SKYGEAR_GOLDEN") == "update" {
 		if err := os.WriteFile(file, []byte(got), 0o644); err != nil {
~~~~

### `content/skygear/pin_test.go`

~~~~diff
--- a/content/skygear/pin_test.go
+++ b/content/skygear/pin_test.go
@@ -63,6 +63,7 @@ func pinRun(t *testing.T, lines map[string]string, scenario string, outcome func
 		t.Fatalf("pin scenario %q recorded twice", scenario)
 	}
 	var k pinSink
+	counted := pinCounters()
 	core.Trace = k.add
 	func() {
 		defer func() { core.Trace = nil }()
@@ -73,6 +74,11 @@ func pinRun(t *testing.T, lines map[string]string, scenario string, outcome func
 		out = outcome()
 	}
 	lines[scenario] = strings.Join(k.take(), " ") + " | " + out
+	if pinDump != "" {
+		pinDumpMu.Lock()
+		pinDumpLines[scenario] = pinCountersSince(counted)
+		pinDumpMu.Unlock()
+	}
 }
 
 // pinGolden compares lines with the text file (one "scenario: names |
@@ -90,6 +96,7 @@ func pinGolden(t *testing.T, file string, lines map[string]string) {
 	for _, name := range keys {
 		fmt.Fprintf(&b, "%s: %s\n", name, lines[name])
 	}
+	pinWriteDump(t, lines)
 	if os.Getenv("SKYGEAR_GOLDEN") == "update" {
 		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
 			t.Fatal(err)
@@ -120,6 +127,8 @@ func pinGolden(t *testing.T, file string, lines map[string]string) {
 		case !ok:
 			t.Errorf("golden %s: scenario %s is not recorded any more\nwant: %s", file, name, want[name])
 			failed = true
+		case hookOrder != "" && pinResults(got) == pinResults(want[name]):
+			// a reordered run: the trace may differ, the results may not
 		case got != want[name]:
 			t.Errorf("golden %s: scenario %s\nwant: %s\ngot:  %s", file, name, want[name], got)
 			failed = true
~~~~

### `content/skygear/pin_use_test.go`

~~~~diff
--- a/content/skygear/pin_use_test.go
+++ b/content/skygear/pin_use_test.go
@@ -700,7 +700,7 @@ func TestPinUse(t *testing.T) {
 		"Use/plain":        "autofill.use screens.menuItem dimension.hunting dimension.stone copier.clear wand.options tinker.veinToggle",
 	}
 	for name, row := range rows {
-		if got := lines[name]; !strings.HasPrefix(got, row+" | ") {
+		if got := lines[name]; hookOrder == "" && !strings.HasPrefix(got, row+" | ") {
 			t.Errorf("%s: %q does not start with the row %q", name, got, row)
 		}
 	}
@@ -714,7 +714,7 @@ func TestPinUse(t *testing.T) {
 		"Use/menuItemHunting":     "autofill.use screens.menuItem | cancelled=true",
 		"AttackEntity/monster":    "dimension.hunter | cancelled=true",
 	} {
-		if got := lines[name]; !strings.Contains(got, want) {
+		if got := lines[name]; hookOrder == "" && !strings.Contains(got, want) {
 			t.Errorf("%s: %q lacks %q", name, got, want)
 		}
 	}
~~~~

### `content/skygear/tierorder_test.go` (new file)

~~~~go
package skygear

import (
	"fmt"
	"hash/fnv"
	"log/slog"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/platform/account"
	"github.com/presentkim/skygear/platform/db"
)

// The tier order guard. A subscriber's place is its tier; inside a tier no
// subscriber may depend on where it stands. TestTierOrder proves it: it runs
// this test binary again with the subscribers of every hook reordered inside
// their tiers (SKYGEAR_HOOK_ORDER) and compares what every pin scenario did.
//
// Until the kernel has tiers, the tier of each subscriber is the one planned
// for it (plannedTiers), and a reordered run first sorts each hook by planned
// tier, keeping today's order inside a tier ("tiers"), then reverses
// ("reverse") or shuffles ("shuffle:<seed>") every tier.

// hookOrder is the reordering of this process: "" (the order of the code),
// "tiers", "reverse" or "shuffle:<seed>".
var hookOrder = os.Getenv("SKYGEAR_HOOK_ORDER")

// pinDump, when set, is the file every pin test appends its scenarios to:
// one line "<scenario> | <results> | <counters>" each (pinDumpLines).
var pinDump = os.Getenv("SKYGEAR_PIN_DUMP")

// tierNames are the tiers in the order they run.
var tierNames = []string{"LOWEST", "LOW", "NORMAL", "HIGH", "HIGHEST", "MONITOR"}

// plannedTiers is the tier planned for every subscriber, one
// "<Hook> <TIER> <name>" per line, in today's order.
const plannedTiers = `
StartBreak LOWEST expand.guard
StartBreak LOW island.guard
StartBreak NORMAL tinker.autoTool
BlockBreak LOWEST expand.guard
BlockBreak LOW island.guard
BlockBreak MONITOR achievement.break
BlockBreak NORMAL tinker.break
BlockBreak NORMAL farm.break
BlockBreak NORMAL storage.break
BlockBreak NORMAL netstore.break
BlockBreak NORMAL machine.break
BlockBreak NORMAL conduit.break
BlockPlace LOWEST expand.guard
BlockPlace LOW island.guard
BlockPlace MONITOR achievement.place
BlockPlace HIGH autofill.place
BlockPlace NORMAL farm.plant
BlockPlace NORMAL storage.place
BlockPlace NORMAL machine.place
BlockPlace HIGH netstore.place
BlockPlace MONITOR conduit.place
BlockPlace MONITOR progress.dragonEgg
UseOnBlock LOWEST dimension.hunting
UseOnBlock LOWEST screens.menuItem
UseOnBlock LOWEST dimension.stone
UseOnBlock LOWEST expand.guard
UseOnBlock LOW island.guard
UseOnBlock NORMAL machine.noteFace
UseOnBlock LOW copier.use
UseOnBlock LOW wand.use
UseOnBlock NORMAL storage.sneakUse
UseOnBlock NORMAL conduit.plate
UseOnBlock NORMAL handwork.stick
UseOnBlock HIGH farm.composter
UseOnBlock NORMAL handwork.gather
UseOnBlock NORMAL fairy.time
UseOnBlock HIGH farm.crop
UseOnBlock HIGHEST expand.guardSide
UseOnBlock HIGHEST island.guardSide
UseOnEntity LOWEST dimension.hunting
UseOnEntity NORMAL fairy.touch
AttackEntity LOWEST dimension.hunter
AttackEntity NORMAL fairy.touch
Use HIGH autofill.use
Use LOWEST screens.menuItem
Use LOWEST dimension.hunting
Use LOWEST dimension.stone
Use NORMAL copier.clear
Use NORMAL wand.options
Use NORMAL tinker.veinToggle
Move LOWEST look.update
Move LOWEST expand.hold
Move HIGH achievement.walk
Move HIGHEST island.void
Sneak LOWEST expand.show
Sneak LOW achievement.sneak
Sneak NORMAL elevator.down
Sneak HIGH farm.crouchGrow
Sprint MONITOR achievement.sprint
Jump MONITOR achievement.jump
Jump NORMAL elevator.up
ItemDamage LOW tinker.ironHammer
ItemDamage HIGH autofill.broken
ItemConsume MONITOR achievement.eat
ItemConsume HIGH autofill.consume
ItemPickup MONITOR achievement.pickup
ItemDrop LOWEST dimension.hunting
ItemDrop MONITOR achievement.drop
Death MONITOR achievement.death
Death NORMAL dimension.died
Death NORMAL tinker.keepFood
Respawn LOW dimension.respawned
Respawn NORMAL tinker.restoreFood
Hurt LOWEST dimension.frozen
Hurt MONITOR achievement.hurt
Hurt HIGHEST dimension.fallen
PunchAir MONITOR achievement.punchAir
Chat MONITOR achievement.chat
BlockPick MONITOR achievement.blockPick
HeldSlot MONITOR achievement.heldSlot
Command MONITOR achievement.command
Load NORMAL farm.loadCrops
Start NORMAL island.keepLoaded
Start NORMAL climate.apply
Join NORMAL screens.menuItem
Join NORMAL fairy.backfillDust
Join NORMAL climate.showWeather
Join NORMAL core.shapes
Join NORMAL island.purgeLost
Join NORMAL progress.sendRecipes
Join MONITOR screens.board
Join HIGH dimension.joined
Leave NORMAL dimension.leave
Leave NORMAL expand.stop
Leave NORMAL core.parts
Leave HIGHEST look.clear
Leave NORMAL core.shapes
Leave HIGHEST screens.board
Leave HIGHEST progress.recipes
NewDay NORMAL island.purgeLost
Second NORMAL farm.growCrops
Second LOW progress.checkHeld
Second NORMAL farm.barrels
Second NORMAL fairy.tick
Second NORMAL climate.tick
Second NORMAL dimension.second
Second NORMAL tinker.effects
Second NORMAL machine.monitor
Second NORMAL farm.plants
Second MONITOR screens.boards
Second MONITOR achievement.playSeconds
Second MONITOR progress.collection
Fast NORMAL network.step
Fast NORMAL dimension.watchdog
Fast MONITOR look.updateAll
Autosave NORMAL netstore.flush
Autosave NORMAL conduit.shares
Close NORMAL storage.displays
Close NORMAL conduit.close
Close NORMAL netstore.flush
ColumnLoad NORMAL conduit.load
ColumnUnload NORMAL conduit.unload
EntityDespawn NORMAL island.lost
`

// orderKept are the pairs that still depend on their order inside one tier:
// "<Hook> <first> <second>". A reordered run keeps first before second. Each
// entry is a dependency to remove (or, for the last ones, an exception the
// owner decides on); the list only shrinks.
var orderKept = []string{
	// the hunting ground refuses the hunter's click before the menu item and the stone
	"UseOnBlock dimension.hunting screens.menuItem",
	"UseOnBlock dimension.hunting dimension.stone",
	// the menu item and the stone open their screen in island expansion mode
	"UseOnBlock screens.menuItem expand.guard",
	"UseOnBlock dimension.stone expand.guard",
	// outside the island the copier and the wand do nothing
	"UseOnBlock island.guard copier.use",
	"UseOnBlock island.guard wand.use",
	// a click the drawer menu or a plate takes counts no stick use
	"UseOnBlock storage.sneakUse handwork.stick",
	"UseOnBlock conduit.plate handwork.stick",
	// a plate's settings open before the hourglass winds the machine
	"UseOnBlock conduit.plate fairy.time",
	// in the air the menu item is the hunter's way out, the stone is not
	"Use screens.menuItem dimension.hunting",
	"Use dimension.hunting dimension.stone",
	// the panel updates on the move island expansion mode refuses
	"Move look.update expand.hold",
	// the player's entries are dropped after the features used them
	"Leave dimension.leave core.parts",
	"Leave expand.stop core.parts",
}

func tierIndex(name string) int {
	i := slices.Index(tierNames, name)
	if i < 0 {
		panic("unknown tier " + name)
	}
	return i
}

// plannedTierOf returns the planned tier of every subscriber, keyed
// "<Hook> <name>".
func plannedTierOf() map[string]int {
	out := map[string]int{}
	for _, ln := range strings.Split(strings.TrimSpace(plannedTiers), "\n") {
		f := strings.Fields(ln)
		out[f[0]+" "+f[2]] = tierIndex(f[1])
	}
	return out
}

// reorderHooks applies mode to the kernel's hooks (core.Reorder).
func reorderHooks(mode string) {
	tiers := plannedTierOf()
	kind, arg, _ := strings.Cut(mode, ":")
	var seed uint64
	if kind == "shuffle" {
		if _, err := fmt.Sscan(arg, &seed); err != nil {
			panic("SKYGEAR_HOOK_ORDER: " + mode)
		}
	} else if kind != "tiers" && kind != "reverse" {
		panic("SKYGEAR_HOOK_ORDER: " + mode)
	}
	core.Reorder(func(hook string, subs []core.Subscription) []int {
		groups := make([][]int, len(tierNames))
		for i, s := range subs {
			t, ok := tiers[hook+" "+s.Name]
			if !ok {
				panic(fmt.Sprintf("plannedTiers has no line for %s %s", hook, s.Name))
			}
			groups[t] = append(groups[t], i)
		}
		h := fnv.New64a()
		h.Write([]byte(hook))
		r := rand.New(rand.NewPCG(seed, h.Sum64()))
		var idx []int
		for _, g := range groups {
			switch kind {
			case "reverse":
				slices.Reverse(g)
			case "shuffle":
				r.Shuffle(len(g), func(i, j int) { g[i], g[j] = g[j], g[i] })
			}
			idx = append(idx, g...)
		}
		// Put the kept pairs back in their order: a first that ended up after
		// its second moves to just before it.
		for moved := true; moved; {
			moved = false
			for _, k := range orderKept {
				f := strings.Fields(k)
				if f[0] != hook {
					continue
				}
				a := slices.IndexFunc(idx, func(i int) bool { return subs[i].Name == f[1] })
				b := slices.IndexFunc(idx, func(i int) bool { return subs[i].Name == f[2] })
				if a < 0 || b < 0 {
					panic("orderKept names no subscriber: " + k)
				}
				if a > b {
					v := idx[a]
					idx = slices.Delete(idx, a, a+1)
					idx = slices.Insert(idx, b, v)
					moved = true
				}
			}
		}
		return idx
	})
}

// pinAccounts are the accounts of the players the tests spawned, in a run
// that dumps (spawnPlayer).
var (
	pinAccountsMu sync.Mutex
	pinAccounts   []string
	pinStoreDB    *db.DB
	pinDumpMu     sync.Mutex
	pinDumpLines  = map[string]string{}
	pinCounterIDs []string
)

func TestMain(m *testing.M) {
	if hookOrder != "" {
		reorderHooks(hookOrder)
	}
	if pinDump != "" {
		dir, err := os.MkdirTemp("", "pindump")
		if err != nil {
			panic(err)
		}
		d, err := db.Open(filepath.Join(dir, "db.sqlite"))
		if err != nil {
			panic(err)
		}
		pinStoreDB = d
		code := m.Run()
		_ = d.Close()
		_ = os.RemoveAll(dir)
		os.Exit(code)
	}
	os.Exit(m.Run())
}

// pinNotePlayer remembers the account of a spawned player in a run that
// dumps, and makes the account exist in the dump's store.
func pinNotePlayer(h *world.EntityHandle) {
	if pinDump == "" {
		return
	}
	acc := h.UUID().String()
	_, _ = pinStoreDB.TouchAccount(acc, "", "p")
	pinAccountsMu.Lock()
	pinAccounts = append(pinAccounts, acc)
	pinAccountsMu.Unlock()
}

// pinCounters returns the count of every counter, summed over the accounts of
// the players spawned so far, keyed by the counter's first achievement id. In
// a run that does not dump it returns nil.
func pinCounters() map[string]int64 {
	if pinDump == "" {
		return nil
	}
	// A test that used its own account store put none back.
	account.Init(pinStoreDB, slog.New(slog.DiscardHandler))
	if pinCounterIDs == nil {
		seen := map[string]bool{}
		for _, a := range account.All() {
			if _, _, ok := account.Progress("", a.ID); !ok {
				continue
			}
			key := a.ID[:strings.LastIndex(a.ID, "_")]
			if !seen[key] {
				seen[key] = true
				pinCounterIDs = append(pinCounterIDs, a.ID)
			}
		}
	}
	pinAccountsMu.Lock()
	accs := slices.Clone(pinAccounts)
	pinAccountsMu.Unlock()
	out := map[string]int64{}
	for _, acc := range accs {
		for _, id := range pinCounterIDs {
			if have, _, _ := account.Progress(acc, id); have != 0 {
				out[id[:strings.LastIndex(id, "_")]] += have
			}
		}
	}
	return out
}

// pinCountersSince is what was counted since before, as
// "<counter>+<n>" sorted and joined by commas, "-" for nothing.
func pinCountersSince(before map[string]int64) string {
	var out []string
	for key, n := range pinCounters() {
		if d := n - before[key]; d != 0 {
			out = append(out, fmt.Sprintf("%s+%d", strings.TrimPrefix(key, "sg_"), d))
		}
	}
	if len(out) == 0 {
		return "-"
	}
	sort.Strings(out)
	return strings.Join(out, ",")
}

// pinResults is the part of a pin line after the trace.
func pinResults(line string) string {
	_, res, ok := strings.Cut(line, " | ")
	if !ok {
		return line
	}
	return res
}

// pinWriteDump appends the dump lines of the scenarios of lines to pinDump.
func pinWriteDump(t *testing.T, lines map[string]string) {
	t.Helper()
	if pinDump == "" {
		return
	}
	f, err := os.OpenFile(pinDump, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	pinDumpMu.Lock()
	defer pinDumpMu.Unlock()
	for name, ln := range lines {
		fmt.Fprintf(f, "%s | %s | %s\n", name, pinResults(ln), pinDumpLines[name])
	}
}

// tierOrders are the reorderings TestTierOrder runs.
var tierOrders = []string{"tiers", "reverse", "shuffle:1", "shuffle:2", "shuffle:3"}

// TestTierOrder runs every test of this package once per reordering of
// tierOrders, in a process of its own, and once in the order of the code. A
// reordered run must pass, which includes that every pin scenario has the
// results of its golden (the trace before the first " | " may differ), and
// must count exactly what the run in the order of the code counted in every
// scenario (the achievement counters of the players, which no golden holds).
func TestTierOrder(t *testing.T) {
	if hookOrder != "" || pinDump != "" {
		return // this is a run TestTierOrder started
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	run := func(mode string) (dump string, out []byte, err error) {
		dump = filepath.Join(dir, strings.ReplaceAll(mode+".dump", ":", "-"))
		cmd := exec.Command(exe, "-test.count=1")
		cmd.Env = append(os.Environ(), "SKYGEAR_HOOK_ORDER="+mode, "SKYGEAR_PIN_DUMP="+dump)
		out, err = cmd.CombinedOutput()
		return dump, out, err
	}
	read := func(file string) map[string]string {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		out := map[string]string{}
		for _, ln := range strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n") {
			name, rest, _ := strings.Cut(ln, " | ")
			out[name] = rest
		}
		return out
	}
	type result struct {
		mode, dump string
		out        []byte
		err        error
	}
	modes := append([]string{""}, tierOrders...)
	results := make([]result, len(modes))
	var wg sync.WaitGroup
	for i, mode := range modes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			dump, out, err := run(mode)
			results[i] = result{mode, dump, out, err}
		}()
	}
	wg.Wait()
	for _, r := range results {
		if r.err != nil {
			t.Fatalf("order %q: the tests fail: %v\n%s", r.mode, r.err, failLines(r.out))
		}
	}
	want := read(results[0].dump)
	if len(want) < 200 {
		t.Fatalf("the run in the order of the code dumped %d scenarios", len(want))
	}
	for _, r := range results[1:] {
		got := read(r.dump)
		for name, w := range want {
			if g, ok := got[name]; !ok || g != w {
				t.Errorf("order %q: scenario %s\nwant: %s\ngot:  %s", r.mode, name, w, g)
			}
		}
		if len(got) != len(want) {
			t.Errorf("order %q: %d scenarios, want %d", r.mode, len(got), len(want))
		}
	}
	t.Logf("%d orders, %d scenarios each", len(results)-1, len(want))
}

// failLines keeps the lines of a test run's output that say what failed.
func failLines(out []byte) string {
	var keep []string
	for _, ln := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(ln, "20") && strings.Contains(ln, " INFO ") {
			continue
		}
		keep = append(keep, ln)
	}
	if len(keep) > 80 {
		keep = keep[:80]
	}
	return strings.Join(keep, "\n")
}
~~~~

## REF T121

Prototype patch: `T121.patch`.

Four conditions; each makes two subscribers exclusive and changes no scenario. `screens.IsMenuItem` keeps its name. Remove exactly these six entries of `orderKept` (with their two comments): `UseOnBlock dimension.hunting screens.menuItem`, `UseOnBlock dimension.hunting dimension.stone`, `UseOnBlock screens.menuItem expand.guard`, `UseOnBlock dimension.stone expand.guard`, `Use screens.menuItem dimension.hunting`, `Use dimension.hunting dimension.stone`.

### `content/skygear/dimension/subscribe.go`

~~~~diff
--- a/content/skygear/dimension/subscribe.go
+++ b/content/skygear/dimension/subscribe.go
@@ -91,7 +91,8 @@ func subscribeUse() {
 		return useStone(e.Actor, e.Held)
 	})
 	core.Use.Handle(30, "dimension.hunting", func(e core.UseEvent) bool {
-		if Hunting(e.S, e.P) {
+		// the menu item in the air is the hunter's way out (screens.menuItem)
+		if Hunting(e.S, e.P) && !sgitem.IsMenuItem(e.Held) {
 			e.Cancel()
 			return true
 		}
@@ -117,9 +118,10 @@ func subscribeUse() {
 }
 
 // useStone opens the second dimension's window for a held teleport stone,
-// on a block or the air; it reports whether held is one.
+// on a block or the air; it reports whether it took the click. A hunter's
+// stone opens nothing: the hunter's use is refused (dimension.hunting).
 func useStone(a core.Actor, held item.Stack) bool {
-	if _, ok := held.Item().(sgitem.DimensionStone); !ok {
+	if _, ok := held.Item().(sgitem.DimensionStone); !ok || Hunting(a.S, a.P) {
 		return false
 	}
 	a.Cancel()
~~~~

### `content/skygear/expand/subscribe.go`

~~~~diff
--- a/content/skygear/expand/subscribe.go
+++ b/content/skygear/expand/subscribe.go
@@ -5,6 +5,7 @@ import (
 	"github.com/df-mc/dragonfly/server/player"
 	"github.com/presentkim/skygear/content/skygear/core"
 	"github.com/presentkim/skygear/content/skygear/look"
+	"github.com/presentkim/skygear/content/skygear/sgitem"
 )
 
 // subscribe registers the takeover and the steps of expand on the kernel.
@@ -73,9 +74,14 @@ func guard(s *core.Session, ctx *player.Context, pos cube.Pos) bool {
 	return false
 }
 
-// The block clicked and the block next to its face are guarded alike.
+// The block clicked and the block next to its face are guarded alike. An
+// item that opens its screen wherever it is used (the menu item, the dimension
+// stone) keeps its click: its own step takes it.
 func subscribeSide() {
 	core.UseOnBlock.Handle(40, "expand.guard", func(e core.UseOnBlockEvent) bool {
+		if sgitem.OpensScreen(e.Held) {
+			return false
+		}
 		return guard(e.S, e.Ctx, e.Pos)
 	})
 	core.UseOnBlock.Handle(160, "expand.guardSide", func(e core.UseOnBlockEvent) bool {
~~~~

### `content/skygear/screenitems_test.go` (new file)

~~~~go
package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/dimension"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// TestHunterStoneOpensNothing: a hunter's use is the hunting ground's to
// refuse, on a block and in the air, also with the dimension stone in hand:
// the use is cancelled and no window opens, wherever dimension.stone stands
// among the subscribers.
func TestHunterStoneOpensNothing(t *testing.T) {
	s := testSession(t)
	island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
	loadAround(s)
	h := spawnPlayer(s)
	t.Cleanup(func() {
		// the hunt schedules the player's landing: out before the world closes
		s.World.Do(func(tx *world.Tx) {
			if e, ok := h.Entity(tx); ok {
				tx.RemoveEntity(e)
			}
		})
	})
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		if !dimension.Begin(s.Session, tx, p, sgitem.MobSlime) {
			t.Fatal("the hunt did not begin")
		}
		p.SetHeldItems(item.NewStack(sgitem.DimensionStone{}, 1), item.Stack{})
		hd := s.Handler(p)
		for name, use := range map[string]func(ctx *player.Context){
			"on a block": func(ctx *player.Context) {
				hd.HandleItemUseOnBlock(ctx, cube.PosFromVec3(p.Position()).Side(cube.FaceDown), cube.FaceUp, mgl64.Vec3{0.5, 1, 0.5})
			},
			"in the air": func(ctx *player.Context) { hd.HandleItemUse(ctx) },
		} {
			ctx := player.NewEventContext(tx, p)
			if shown := pinScreens(func() { use(ctx) }); shown != "screens=-" {
				t.Errorf("%s: the hunter's stone opened %s", name, shown)
			}
			if !ctx.Cancelled() {
				t.Errorf("%s: the hunter's use was not cancelled", name)
			}
		}
	})
}
~~~~

### `content/skygear/screens/menu.go`

~~~~diff
--- a/content/skygear/screens/menu.go
+++ b/content/skygear/screens/menu.go
@@ -4,15 +4,14 @@ import (
 	"github.com/df-mc/dragonfly/server/item"
 	"github.com/df-mc/dragonfly/server/player"
 	"github.com/presentkim/skygear/common/i18n"
+	"github.com/presentkim/skygear/content/skygear/sgitem"
 )
 
-const menuItemKey = "skygear_menu"
-
 // MenuItem opens the game menu when used. Its name is written on it in lang
 // (a name of a stack is text, not a translation key); renameMenuItems names
 // the menu items a player has in the player's language again.
 func MenuItem(lang string) item.Stack {
-	return item.NewStack(item.NetherStar{}, 1).WithCustomName(i18n.T(lang, "menu.item.name")).WithValue(menuItemKey, true)
+	return item.NewStack(item.NetherStar{}, 1).WithCustomName(i18n.T(lang, "menu.item.name")).WithValue(sgitem.MenuItemKey, true)
 }
 
 // RenameMenuItems names the menu items in p's inventory in p's language, as
@@ -28,7 +27,4 @@ func RenameMenuItems(p *player.Player) {
 	}
 }
 
-func IsMenuItem(s item.Stack) bool {
-	_, ok := s.Value(menuItemKey)
-	return ok
-}
+func IsMenuItem(s item.Stack) bool { return sgitem.IsMenuItem(s) }
~~~~

### `content/skygear/screens/subscribe.go`

~~~~diff
--- a/content/skygear/screens/subscribe.go
+++ b/content/skygear/screens/subscribe.go
@@ -39,10 +39,12 @@ func subscribePresence() {
 	})
 }
 
-// A held menu item opens the menu on a block and in the air.
+// A held menu item opens the menu on a block and in the air. A hunter's click
+// on a block is the hunting ground's to refuse (dimension.hunting), whatever
+// the hunter holds; in the air the menu item is the hunter's way out.
 func subscribeUse() {
 	core.UseOnBlock.Handle(20, "screens.menuItem", func(e core.UseOnBlockEvent) bool {
-		if IsMenuItem(e.Held) {
+		if IsMenuItem(e.Held) && !dimension.Hunting(e.S, e.P) {
 			e.Cancel()
 			useHeldScreen(e.S, e.P)
 			return true
~~~~

### `content/skygear/sgitem/screenitem.go` (new file)

~~~~go
package sgitem

import "github.com/df-mc/dragonfly/server/item"

// MenuItemKey is the value the menu item carries: the game menu's item is a
// plain item marked with it (screens.MenuItem).
const MenuItemKey = "skygear_menu"

// IsMenuItem reports whether s is the menu item.
func IsMenuItem(s item.Stack) bool {
	_, ok := s.Value(MenuItemKey)
	return ok
}

// OpensScreen reports whether s is an item that opens its screen wherever it
// is used: the menu item or the dimension stone. A step that refuses a click
// because of where the player is leaves the click of such an item alone.
func OpensScreen(s item.Stack) bool {
	if IsMenuItem(s) {
		return true
	}
	_, stone := s.Item().(DimensionStone)
	return stone
}
~~~~

### `content/skygear/sgitem/screenitem_test.go` (new file)

~~~~go
package sgitem

import (
	"testing"

	"github.com/df-mc/dragonfly/server/item"
)

func TestOpensScreen(t *testing.T) {
	menu := item.NewStack(item.NetherStar{}, 1).WithValue(MenuItemKey, true)
	for _, c := range []struct {
		name       string
		it         item.Stack
		menu, open bool
	}{
		{"menu item", menu, true, true},
		{"dimension stone", item.NewStack(DimensionStone{}, 1), false, true},
		{"plain nether star", item.NewStack(item.NetherStar{}, 1), false, false},
		{"copier", item.NewStack(ConfigCopier{}, 1), false, false},
		{"empty hand", item.Stack{}, false, false},
	} {
		if got := IsMenuItem(c.it); got != c.menu {
			t.Errorf("%s: IsMenuItem = %t", c.name, got)
		}
		if got := OpensScreen(c.it); got != c.open {
			t.Errorf("%s: OpensScreen = %t", c.name, got)
		}
	}
}
~~~~

### `content/skygear/tierorder_test.go`

~~~~diff
--- a/content/skygear/tierorder_test.go
+++ b/content/skygear/tierorder_test.go
@@ -174,12 +174,6 @@ EntityDespawn NORMAL island.lost
 // entry is a dependency to remove (or, for the last ones, an exception the
 // owner decides on); the list only shrinks.
 var orderKept = []string{
-	// the hunting ground refuses the hunter's click before the menu item and the stone
-	"UseOnBlock dimension.hunting screens.menuItem",
-	"UseOnBlock dimension.hunting dimension.stone",
-	// the menu item and the stone open their screen in island expansion mode
-	"UseOnBlock screens.menuItem expand.guard",
-	"UseOnBlock dimension.stone expand.guard",
 	// outside the island the copier and the wand do nothing
 	"UseOnBlock island.guard copier.use",
 	"UseOnBlock island.guard wand.use",
@@ -188,9 +182,6 @@ var orderKept = []string{
 	"UseOnBlock conduit.plate handwork.stick",
 	// a plate's settings open before the hourglass winds the machine
 	"UseOnBlock conduit.plate fairy.time",
-	// in the air the menu item is the hunter's way out, the stone is not
-	"Use screens.menuItem dimension.hunting",
-	"Use dimension.hunting dimension.stone",
 	// the panel updates on the move island expansion mode refuses
 	"Move look.update expand.hold",
 	// the player's entries are dropped after the features used them
~~~~

## REF T122

Prototype patch: `T122.patch`.

Remove the two entries `UseOnBlock island.guard copier.use` and `UseOnBlock island.guard wand.use` (with their comment). The test asserts what holds in every order: the island's guard is reached, is the last step and the click is cancelled.

### `content/skygear/copier/subscribe.go`

~~~~diff
--- a/content/skygear/copier/subscribe.go
+++ b/content/skygear/copier/subscribe.go
@@ -2,6 +2,7 @@ package copier
 
 import (
 	"github.com/presentkim/skygear/content/skygear/core"
+	"github.com/presentkim/skygear/content/skygear/island"
 	"github.com/presentkim/skygear/content/skygear/sgitem"
 )
 
@@ -11,9 +12,11 @@ func subscribe() {
 }
 
 // The configuration copier copies and pastes settings and does nothing else.
+// Where the island does not allow building it leaves the click alone: the
+// island's guard refuses it.
 func subscribeUse() {
 	core.UseOnBlock.Handle(70, "copier.use", func(e core.UseOnBlockEvent) bool {
-		if _, ok := e.Held.Item().(sgitem.ConfigCopier); ok {
+		if _, ok := e.Held.Item().(sgitem.ConfigCopier); ok && island.CanBuild(e.S, e.Pos) {
 			e.Cancel()
 			_, off := e.P.HeldItems()
 			UseCopier(e.S, e.P, e.Tx, e.Pos, e.Face, e.Click, e.Held, off)
~~~~

### `content/skygear/outsidetools_test.go` (new file)

~~~~go
package skygear

import (
	"slices"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// TestToolsLeaveTheClickOutsideTheIsland: with the configuration copier or
// the construction wand in hand, a click on a block outside the island is
// refused by the island's guard as any other click is. The guard is reached
// wherever the tools' steps stand among the subscribers: they do not take a
// click where the island does not allow building.
func TestToolsLeaveTheClickOutsideTheIsland(t *testing.T) {
	outside := cube.Pos{84, island.IslandY + 2, 4}
	for name, held := range map[string]item.Stack{
		"copier": item.NewStack(sgitem.ConfigCopier{}, 1),
		"wand":   item.NewStack(sgitem.NewWand(sgitem.WandStone), 1),
	} {
		s := testSession(t)
		island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
		loadAround(s)
		h := spawnPlayer(s)
		withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
			p.SetHeldItems(held, item.Stack{})
			ctx := player.NewEventContext(tx, p)
			var ran []string
			core.Trace = func(step string) { ran = append(ran, step) }
			func() {
				defer func() { core.Trace = nil }()
				s.Handler(p).HandleItemUseOnBlock(ctx, outside, cube.FaceUp, mgl64.Vec3{0.5, 1, 0.5})
			}()
			if !slices.Contains(ran, "island.guard") {
				t.Errorf("%s: the island's guard was not reached: %v", name, ran)
			}
			if last := ran[len(ran)-1]; last != "island.guard" {
				t.Errorf("%s: the click went on to %s after the guard", name, last)
			}
			if !ctx.Cancelled() {
				t.Errorf("%s: the click outside the island was not cancelled", name)
			}
		})
	}
}
~~~~

### `content/skygear/tierorder_test.go`

~~~~diff
--- a/content/skygear/tierorder_test.go
+++ b/content/skygear/tierorder_test.go
@@ -174,9 +174,6 @@ EntityDespawn NORMAL island.lost
 // entry is a dependency to remove (or, for the last ones, an exception the
 // owner decides on); the list only shrinks.
 var orderKept = []string{
-	// outside the island the copier and the wand do nothing
-	"UseOnBlock island.guard copier.use",
-	"UseOnBlock island.guard wand.use",
 	// a click the drawer menu or a plate takes counts no stick use
 	"UseOnBlock storage.sneakUse handwork.stick",
 	"UseOnBlock conduit.plate handwork.stick",
~~~~

### `content/skygear/wand/subscribe.go`

~~~~diff
--- a/content/skygear/wand/subscribe.go
+++ b/content/skygear/wand/subscribe.go
@@ -2,6 +2,7 @@ package wand
 
 import (
 	"github.com/presentkim/skygear/content/skygear/core"
+	"github.com/presentkim/skygear/content/skygear/island"
 	"github.com/presentkim/skygear/content/skygear/look"
 	"github.com/presentkim/skygear/content/skygear/sgitem"
 )
@@ -13,10 +14,11 @@ func subscribe() {
 }
 
 // The construction wand builds out the face clicked and does nothing else;
-// used on the air while sneaking it shows its settings.
+// used on the air while sneaking it shows its settings. Where the island does
+// not allow building it leaves the click alone: the island's guard refuses it.
 func subscribeUse() {
 	core.UseOnBlock.Handle(80, "wand.use", func(e core.UseOnBlockEvent) bool {
-		if _, ok := sgitem.WandOf(e.Held); ok {
+		if _, ok := sgitem.WandOf(e.Held); ok && island.CanBuild(e.S, e.Pos) {
 			e.Cancel()
 			Use(e.S, e.P, e.Tx, e.Pos, e.Face, e.Held)
 			return true
~~~~

## REF T123

Prototype patch: `T123.patch`.

`plannedTiers`: the line `Move HIGH achievement.walk` becomes `Move MONITOR achievement.walk`. With the old `CountWalk` and that line the new test fails in the reordered runs (2 counted for a move of one block): that is the dependence this task removes.

### `content/skygear/achievement/counter.go`

~~~~diff
--- a/content/skygear/achievement/counter.go
+++ b/content/skygear/achievement/counter.go
@@ -68,11 +68,10 @@ func CountBreak(p *player.Player, b world.Block, hammer bool) {
 // block was counted.
 var Walked = core.NewPlayers[float64]()
 
-// CountWalk counts the blocks p walks to newPos: the horizontal distance of
-// a move, kept as a fraction until it adds up to a block. A jump of more
-// than a few blocks is a teleport, not walking.
-func CountWalk(s *core.Session, p *player.Player, newPos mgl64.Vec3) {
-	from := p.Position()
+// CountWalk counts the blocks p walks from from to newPos: the horizontal
+// distance of a move, kept as a fraction until it adds up to a block. A jump
+// of more than a few blocks is a teleport, not walking.
+func CountWalk(s *core.Session, p *player.Player, from, newPos mgl64.Vec3) {
 	d := math.Hypot(newPos[0]-from[0], newPos[2]-from[2])
 	if d == 0 || d > 4 {
 		return
~~~~

### `content/skygear/achievement/subscribe.go`

~~~~diff
--- a/content/skygear/achievement/subscribe.go
+++ b/content/skygear/achievement/subscribe.go
@@ -50,7 +50,7 @@ func subscribeSimple() {
 // The achievement counters of the walk, sneak, drop and hurt events.
 func subscribeMoves() {
 	core.Move.On(30, "achievement.walk", func(e core.MoveEvent) {
-		CountWalk(e.S, e.P, e.Pos)
+		CountWalk(e.S, e.P, e.From, e.Pos)
 	})
 	core.Sneak.On(20, "achievement.sneak", func(e core.PlayerEvent) {
 		Count(e.P, "sneak", 1)
~~~~

### `content/skygear/core/events.go`

~~~~diff
--- a/content/skygear/core/events.go
+++ b/content/skygear/core/events.go
@@ -36,9 +36,10 @@ func (a Actor) Cancel() {
 // Pointer fields (Drops, XP, Damage, Pos, Pick) are the pointers Dragonfly
 // passed to the handler, not copies: a subscriber that writes through one
 // changes what Dragonfly does, exactly as a direct handler would. The other
-// fields (Held, Off, Block, Item) are copies taken when the event was built:
-// when a subscriber changes the hand or the world, later subscribers still see
-// the old value and must ask the player or the Tx for the new one.
+// fields (Held, Off, Block, Item, From) are copies taken when the event was
+// built: when a subscriber changes the hand, the world or the player's place,
+// later subscribers still see the old value and must ask the player or the Tx
+// for the new one.
 
 // StartBreakEvent is a player starting to break the block at Pos.
 type StartBreakEvent struct {
@@ -89,11 +90,14 @@ type UseEvent struct {
 	Held, Off item.Stack
 }
 
-// MoveEvent is a player moving to Pos, turned to Rot.
+// MoveEvent is a player moving from From to Pos, turned to Rot. From is
+// where the player stood when the event was built: a subscriber that moves
+// the player (a teleport) does not change it.
 type MoveEvent struct {
 	Actor
-	Pos mgl64.Vec3
-	Rot cube.Rotation
+	From mgl64.Vec3
+	Pos  mgl64.Vec3
+	Rot  cube.Rotation
 }
 
 // PlayerEvent is an event that has no arguments of its own (Sneak, Sprint,
~~~~

### `content/skygear/core/handler.go`

~~~~diff
--- a/content/skygear/core/handler.go
+++ b/content/skygear/core/handler.go
@@ -71,7 +71,8 @@ func (h *PlayerHandler) HandleItemUse(ctx *player.Context) {
 }
 
 func (h *PlayerHandler) HandleMove(ctx *player.Context, newPos mgl64.Vec3, newRot cube.Rotation) {
-	Move.Emit(MoveEvent{Actor: h.actor(ctx), Pos: newPos, Rot: newRot})
+	a := h.actor(ctx)
+	Move.Emit(MoveEvent{Actor: a, From: a.P.Position(), Pos: newPos, Rot: newRot})
 }
 
 // HandleToggleSneak emits Sneak when sneaking starts; nothing happens when it
~~~~

### `content/skygear/core/handler_test.go`

~~~~diff
--- a/content/skygear/core/handler_test.go
+++ b/content/skygear/core/handler_test.go
@@ -177,6 +177,7 @@ func rows() []row {
 		rowOf("Move", Move, func(x *fix) { x.h.HandleMove(x.ctx, vec, rot) },
 			func(t *testing.T, x *fix, e MoveEvent) {
 				actorIs(t, x, e.Actor, true)
+				is(t, "From", e.From, x.p.Position())
 				is(t, "Pos", e.Pos, vec)
 				is(t, "Rot", e.Rot, rot)
 			}),
~~~~

### `content/skygear/tierorder_test.go`

~~~~diff
--- a/content/skygear/tierorder_test.go
+++ b/content/skygear/tierorder_test.go
@@ -96,7 +96,7 @@ Use NORMAL wand.options
 Use NORMAL tinker.veinToggle
 Move LOWEST look.update
 Move LOWEST expand.hold
-Move HIGH achievement.walk
+Move MONITOR achievement.walk
 Move HIGHEST island.void
 Sneak LOWEST expand.show
 Sneak LOW achievement.sneak
~~~~

### `content/skygear/walkvoid_test.go` (new file)

~~~~go
package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/platform/account"
	"github.com/presentkim/skygear/platform/instance"
)

// TestWalkCountsTheMoveIntoTheVoid: the move that takes a player below the
// island counts the distance of that move, from where the player stood to
// where it moved, although the island brings the player back in the same
// event. It counts the same wherever achievement.walk stands relative to
// island.void: the event carries where the player stood (MoveEvent.From).
func TestWalkCountsTheMoveIntoTheVoid(t *testing.T) {
	d := bridgeAccount(t)
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		acc := instance.AccountID(p)
		if _, err := d.TouchAccount(acc, "", p.Name()); err != nil {
			t.Fatal(err)
		}
		// The only block around: the player comes back above it, two blocks
		// from where the move ends.
		tx.SetBlock(cube.Pos{3, 60, 2}, block.Stone{}, nil)
		from := p.Position()
		to := mgl64.Vec3{from.X() + 1, island.VoidY - 10, from.Z()}
		ctx := player.NewEventContext(tx, p)
		s.Handler(p).HandleMove(ctx, to, cube.Rotation{})
		if !ctx.Cancelled() {
			t.Fatal("the move into the void was not cancelled")
		}
		if got := p.Position(); got.X() != 3.5 || got.Y() != island.VoidReturnY {
			t.Fatalf("the player was put at %v", got)
		}
		if got := account.Counter(acc, "walk"); got != 1 {
			t.Errorf("walk counted %d for a move of one block, want 1", got)
		}
	})
}
~~~~

## REF T124

Prototype patch: `T124.patch`.

The switch. `core/hook.go` and `core/coretest/order.go` are given whole (the first as a diff of the whole file). The 120 call sites change the first argument only, by the table of section 3; the diff below has every one of them, with the comments that say why a tier is not the obvious one.

The four scenario goldens are not listed: re-record them (global constraints of the plan) and run this check from `content/skygear`; it must print `result diffs []` four times:

~~~~python
import subprocess
for f in ['handler-simple', 'handler-blocks', 'handler-use', 'lifecycle']:
    path = 'testdata/pin/%s.golden' % f
    old = subprocess.run(['git', 'show', 'HEAD:content/skygear/' + path], capture_output=True, text=True).stdout.rstrip('\n').split('\n')
    new = open(path).read().rstrip('\n').split('\n')
    def parse(lines):
        d = {}
        for ln in lines:
            name, rest = ln.split(': ', 1)
            trace, results = rest.split(' | ', 1)
            d[name] = (trace, results)
        return d
    o, n = parse(old), parse(new)
    print(f, len(o), len(n), set(o) == set(n), 'result diffs', [k for k in o if o[k][1] != n[k][1]])
~~~~

Expected: 29, 33, 123 and 49 scenarios, the same names, no result difference. `order.golden` after the re-record is exactly:

~~~~
StartBreak LOWEST expand.guard
StartBreak LOW island.guard
StartBreak NORMAL tinker.autoTool
BlockBreak LOWEST expand.guard
BlockBreak LOW island.guard
BlockBreak NORMAL tinker.break
BlockBreak NORMAL machine.break
BlockBreak NORMAL conduit.break
BlockBreak NORMAL storage.break
BlockBreak NORMAL farm.break
BlockBreak NORMAL netstore.break
BlockBreak MONITOR achievement.break
BlockPlace LOWEST expand.guard
BlockPlace LOW island.guard
BlockPlace NORMAL machine.place
BlockPlace NORMAL storage.place
BlockPlace NORMAL farm.plant
BlockPlace HIGH autofill.place
BlockPlace HIGH netstore.place
BlockPlace MONITOR achievement.place
BlockPlace MONITOR progress.dragonEgg
BlockPlace MONITOR conduit.place
UseOnBlock LOWEST dimension.hunting
UseOnBlock LOWEST dimension.stone
UseOnBlock LOWEST expand.guard
UseOnBlock LOWEST screens.menuItem
UseOnBlock LOW island.guard
UseOnBlock LOW copier.use
UseOnBlock LOW wand.use
UseOnBlock NORMAL machine.noteFace
UseOnBlock NORMAL conduit.plate
UseOnBlock NORMAL storage.sneakUse
UseOnBlock NORMAL fairy.time
UseOnBlock NORMAL handwork.stick
UseOnBlock NORMAL handwork.gather
UseOnBlock HIGH farm.composter
UseOnBlock HIGH farm.crop
UseOnBlock HIGHEST island.guardSide
UseOnBlock HIGHEST expand.guardSide
UseOnEntity LOWEST dimension.hunting
UseOnEntity NORMAL fairy.touch
AttackEntity LOWEST dimension.hunter
AttackEntity NORMAL fairy.touch
Use LOWEST dimension.hunting
Use LOWEST dimension.stone
Use LOWEST screens.menuItem
Use NORMAL tinker.veinToggle
Use NORMAL copier.clear
Use NORMAL wand.options
Use HIGH autofill.use
Move LOWEST expand.hold
Move HIGHEST island.void
Move MONITOR achievement.walk
Move MONITOR look.update always
Sneak LOWEST expand.show
Sneak LOW achievement.sneak
Sneak NORMAL elevator.down
Sneak HIGH farm.crouchGrow
Sprint MONITOR achievement.sprint
Jump NORMAL elevator.up
Jump MONITOR achievement.jump
ItemDamage LOW tinker.ironHammer
ItemDamage HIGH autofill.broken
ItemConsume HIGH autofill.consume
ItemConsume MONITOR achievement.eat
ItemPickup MONITOR achievement.pickup
ItemDrop LOWEST dimension.hunting
ItemDrop MONITOR achievement.drop
Death NORMAL tinker.keepFood
Death NORMAL dimension.died
Death MONITOR achievement.death
Respawn LOW dimension.respawned
Respawn NORMAL tinker.restoreFood
Hurt LOWEST dimension.frozen
Hurt HIGHEST dimension.fallen
Hurt MONITOR achievement.hurt
PunchAir MONITOR achievement.punchAir
Chat MONITOR achievement.chat
BlockPick MONITOR achievement.blockPick
HeldSlot MONITOR achievement.heldSlot
Command MONITOR achievement.command
Load NORMAL farm.loadCrops
Start NORMAL island.keepLoaded
Start NORMAL climate.apply
Join NORMAL core.shapes
Join NORMAL progress.sendRecipes
Join NORMAL island.purgeLost
Join NORMAL climate.showWeather
Join NORMAL fairy.backfillDust
Join NORMAL screens.menuItem
Join HIGH dimension.joined
Join MONITOR screens.board
Leave NORMAL dimension.leave
Leave NORMAL expand.stop
Leave HIGHEST progress.recipes
Leave HIGHEST look.clear
Leave HIGHEST screens.board
NewDay NORMAL island.purgeLost
Second LOW progress.checkHeld
Second NORMAL climate.tick
Second NORMAL tinker.effects
Second NORMAL machine.monitor
Second NORMAL farm.growCrops
Second NORMAL farm.barrels
Second NORMAL farm.plants
Second NORMAL fairy.tick
Second NORMAL dimension.second
Second MONITOR achievement.playSeconds
Second MONITOR progress.collection
Second MONITOR screens.boards
Fast NORMAL network.step
Fast NORMAL dimension.watchdog
Fast MONITOR look.updateAll
Autosave NORMAL conduit.shares
Autosave NORMAL netstore.flush
Close NORMAL conduit.close
Close NORMAL storage.displays
Close NORMAL netstore.flush
ColumnLoad NORMAL conduit.load
ColumnUnload NORMAL conduit.unload
EntityDespawn NORMAL island.lost
~~~~

### `content/skygear/achievement/subscribe.go`

~~~~diff
--- a/content/skygear/achievement/subscribe.go
+++ b/content/skygear/achievement/subscribe.go
@@ -5,7 +5,9 @@ import (
 	"github.com/presentkim/skygear/content/skygear/sgitem"
 )
 
-// subscribe registers the achievement counters on the hooks they count.
+// subscribe registers the achievement counters on the hooks they count. A
+// counter records what happened and changes nothing: it is a Monitor
+// subscriber, except the sneak counter (subscribeMoves).
 func subscribe() {
 	subscribeSimple()
 	subscribeMoves()
@@ -15,67 +17,75 @@ func subscribe() {
 
 // The achievement counters of the simple player events.
 func subscribeSimple() {
-	core.Sprint.On(10, "achievement.sprint", func(e core.PlayerEvent) {
+	core.Sprint.On(core.Monitor, "achievement.sprint", func(e core.PlayerEvent) {
 		Count(e.P, "sprint", 1)
 	})
-	core.Jump.On(10, "achievement.jump", func(e core.PlayerEvent) {
+	core.Jump.On(core.Monitor, "achievement.jump", func(e core.PlayerEvent) {
 		Count(e.P, "jump", 1)
 	})
-	core.ItemConsume.On(10, "achievement.eat", func(e core.ItemEvent) {
+	core.ItemConsume.On(core.Monitor, "achievement.eat", func(e core.ItemEvent) {
 		Count(e.P, "eat", 1)
 	})
-	core.ItemPickup.On(10, "achievement.pickup", func(e core.ItemEvent) {
+	core.ItemPickup.On(core.Monitor, "achievement.pickup", func(e core.ItemEvent) {
 		Count(e.P, "pickup", int64(e.Pick.Count()))
 	})
-	core.Death.On(10, "achievement.death", func(e core.PlayerEvent) {
+	core.Death.On(core.Monitor, "achievement.death", func(e core.PlayerEvent) {
 		Count(e.P, "death", 1)
 	})
-	core.PunchAir.On(10, "achievement.punchAir", func(e core.PlayerEvent) {
+	core.PunchAir.On(core.Monitor, "achievement.punchAir", func(e core.PlayerEvent) {
 		Count(e.P, "punch_air", 1)
 	})
-	core.Chat.On(10, "achievement.chat", func(e core.PlayerEvent) {
+	core.Chat.On(core.Monitor, "achievement.chat", func(e core.PlayerEvent) {
 		Count(e.P, "chat", 1)
 	})
-	core.BlockPick.On(10, "achievement.blockPick", func(e core.PlayerEvent) {
+	core.BlockPick.On(core.Monitor, "achievement.blockPick", func(e core.PlayerEvent) {
 		Count(e.P, "block_pick", 1)
 	})
-	core.HeldSlot.On(10, "achievement.heldSlot", func(e core.PlayerEvent) {
+	core.HeldSlot.On(core.Monitor, "achievement.heldSlot", func(e core.PlayerEvent) {
 		Count(e.P, "hotbar", 1)
 	})
-	core.Command.On(10, "achievement.command", func(e core.PlayerEvent) {
+	core.Command.On(core.Monitor, "achievement.command", func(e core.PlayerEvent) {
 		Count(e.P, "command", 1)
 	})
 }
 
-// The achievement counters of the walk, sneak, drop and hurt events.
+// The achievement counters of the walk, sneak, drop and hurt events. The walk
+// counts from where the player stood when the event was built (e.From), so a
+// step that moved the player meanwhile does not change it.
+//
+// The sneak counter is in the Low tier, not in Monitor: a sneak counts when
+// the elevator then rides (elevator.down, Normal, cancels the event for the
+// later tiers) and does not count in island expansion mode (expand.show,
+// Lowest). No tier after Normal sees the first and no option tells the two
+// apart, so the counter stands between them.
 func subscribeMoves() {
-	core.Move.On(30, "achievement.walk", func(e core.MoveEvent) {
+	core.Move.On(core.Monitor, "achievement.walk", func(e core.MoveEvent) {
 		CountWalk(e.S, e.P, e.From, e.Pos)
 	})
-	core.Sneak.On(20, "achievement.sneak", func(e core.PlayerEvent) {
+	core.Sneak.On(core.Low, "achievement.sneak", func(e core.PlayerEvent) {
 		Count(e.P, "sneak", 1)
 	})
-	core.ItemDrop.On(20, "achievement.drop", func(e core.ItemEvent) {
+	core.ItemDrop.On(core.Monitor, "achievement.drop", func(e core.ItemEvent) {
 		Count(e.P, "item_drop", 1)
 	})
-	core.Hurt.On(20, "achievement.hurt", func(e core.HurtEvent) {
+	core.Hurt.On(core.Monitor, "achievement.hurt", func(e core.HurtEvent) {
 		Count(e.P, "hurt", 1)
 	})
 }
 
 // Every player's play time counts a second.
 func subscribeSeconds() {
-	core.Second.On(110, "achievement.playSeconds", func(e core.Tick) {
+	core.Second.On(core.Monitor, "achievement.playSeconds", func(e core.Tick) {
 		CountAll(e.Tx, "play_seconds", 1)
 	})
 }
 
 // Breaking and placing blocks count.
 func subscribeBlocks() {
-	core.BlockBreak.On(30, "achievement.break", func(e core.BlockBreakEvent) {
+	core.BlockBreak.On(core.Monitor, "achievement.break", func(e core.BlockBreakEvent) {
 		CountBreak(e.P, e.Block, sgitem.IsHammer(e.Held))
 	})
-	core.BlockPlace.On(30, "achievement.place", func(e core.BlockPlaceEvent) {
+	core.BlockPlace.On(core.Monitor, "achievement.place", func(e core.BlockPlaceEvent) {
 		Count(e.P, "block_place", 1)
 	})
 }
~~~~

### `content/skygear/autofill/feature.go`

~~~~diff
--- a/content/skygear/autofill/feature.go
+++ b/content/skygear/autofill/feature.go
@@ -18,23 +18,24 @@ var Feature = &core.Feature{
 
 // A broken tool and a consumed item are replaced from the inventory, the last
 // block of a stack placed is replaced, and a thrown snowball, egg or pearl may
-// be the last of the stack.
+// be the last of the stack. Refilling follows what the other steps made of the
+// event (the damage the tool takes, a use another step took): the High tier.
 func subscribe() {
-	core.ItemDamage.On(20, "autofill.broken", func(e core.ItemDamageEvent) {
+	core.ItemDamage.On(core.High, "autofill.broken", func(e core.ItemDamageEvent) {
 		// a held tool that breaks is replaced by another of its kind
 		if held, _ := e.P.HeldItems(); *e.Damage > 0 && e.Item.Durability() >= 0 && *e.Damage >= e.Item.Durability() && sameItemName(held, e.Item) {
 			RefillHandLater(e.P, e.Tx, e.Item)
 		}
 	})
-	core.ItemConsume.On(20, "autofill.consume", func(e core.ItemEvent) {
+	core.ItemConsume.On(core.High, "autofill.consume", func(e core.ItemEvent) {
 		RefillHandLater(e.P, e.Tx, e.Item)
 	})
-	core.BlockPlace.On(40, "autofill.place", func(e core.BlockPlaceEvent) {
+	core.BlockPlace.On(core.High, "autofill.place", func(e core.BlockPlaceEvent) {
 		if e.Held.Count() == 1 {
 			RefillHandLater(e.P, e.Tx, e.Held)
 		}
 	})
-	core.Use.On(10, "autofill.use", func(e core.UseEvent) {
+	core.Use.On(core.High, "autofill.use", func(e core.UseEvent) {
 		if e.Held.Count() == 1 {
 			RefillHandLater(e.P, e.Tx, e.Held)
 		}
~~~~

### `content/skygear/climate/subscribe.go`

~~~~diff
--- a/content/skygear/climate/subscribe.go
+++ b/content/skygear/climate/subscribe.go
@@ -10,17 +10,17 @@ func subscribe() {
 
 // The weather advances every second.
 func subscribeSecond() {
-	core.Second.On(50, "climate.tick", func(e core.Tick) {
+	core.Second.On(core.Normal, "climate.tick", func(e core.Tick) {
 		Tick(e.S, e.Tx)
 	})
 }
 
 // The weather is shown to a joining player and applied when the session starts.
 func subscribeSession() {
-	core.Join.On(30, "climate.showWeather", func(e core.Presence) {
+	core.Join.On(core.Normal, "climate.showWeather", func(e core.Presence) {
 		ShowWeather(e.S, e.P)
 	})
-	core.Start.On(20, "climate.apply", func(e core.Tick) {
+	core.Start.On(core.Normal, "climate.apply", func(e core.Tick) {
 		Apply(e.S, e.Tx)
 	})
 }
~~~~

### `content/skygear/conduit/subscribe.go`

~~~~diff
--- a/content/skygear/conduit/subscribe.go
+++ b/content/skygear/conduit/subscribe.go
@@ -16,29 +16,30 @@ func subscribe() {
 // at close; a loaded column puts its bundles into the grids and an unloaded one
 // takes them out, before the column is saved.
 func subscribeShares() {
-	core.Autosave.On(20, "conduit.shares", func(e core.Tick) {
+	core.Autosave.On(core.Normal, "conduit.shares", func(e core.Tick) {
 		WriteShares(e.S, e.Tx)
 	})
-	core.Close.On(20, "conduit.close", func(e core.Tick) {
+	core.Close.On(core.Normal, "conduit.close", func(e core.Tick) {
 		WriteShares(e.S, e.Tx)
 		RemoveVisuals(e.S, e.Tx)
 	})
-	core.ColumnLoad.On(10, "conduit.load", func(e core.ColumnEvent) {
+	core.ColumnLoad.On(core.Normal, "conduit.load", func(e core.ColumnEvent) {
 		LoadColumn(e.S, e.Tx, e.Blocks)
 	})
-	core.ColumnUnload.On(10, "conduit.unload", func(e core.ColumnEvent) {
+	core.ColumnUnload.On(core.Normal, "conduit.unload", func(e core.ColumnEvent) {
 		UnloadColumn(e.S, e.Tx, e.Blocks)
 	})
 }
 
-// A conduit bundle breaking loses its visual; a placed one counts.
+// A conduit bundle breaking loses its visual; a placed one counts (a record:
+// Monitor).
 func subscribeBlocks() {
-	core.BlockBreak.On(90, "conduit.break", func(e core.BlockBreakEvent) {
+	core.BlockBreak.On(core.Normal, "conduit.break", func(e core.BlockBreakEvent) {
 		if _, ok := e.Block.(Bundle); ok {
 			RemoveVisual(e.S, e.Tx, e.Pos)
 		}
 	})
-	core.BlockPlace.On(90, "conduit.place", func(e core.BlockPlaceEvent) {
+	core.BlockPlace.On(core.Monitor, "conduit.place", func(e core.BlockPlaceEvent) {
 		if _, ok := e.Block.(Bundle); ok {
 			achievement.Count(e.P, "conduit_place", 1)
 		}
@@ -49,8 +50,13 @@ func subscribeBlocks() {
 // opens its settings, whatever they hold, unless the held conduit goes into
 // that conduit (a click on the conduit itself is ConduitBundle.Activate).
 // Sneaking places blocks instead.
+//
+// conduit.plate is registered before handwork.stick and fairy.time, which are
+// in the same tier, and takes the click first: a stick on a plate counts no
+// stick use and an hourglass on a plate opens the settings. That order is one
+// of the exceptions the tier order guard keeps (the root's orderKept).
 func subscribePlate() {
-	core.UseOnBlock.Handle(100, "conduit.plate", func(e core.UseOnBlockEvent) bool {
+	core.UseOnBlock.Handle(core.Normal, "conduit.plate", func(e core.UseOnBlockEvent) bool {
 		if !e.P.Sneaking() && UsePlate(e.S, e.P, e.Tx, e.Pos, e.Face, e.Click, e.Held) {
 			e.Cancel()
 			return true
~~~~

### `content/skygear/copier/subscribe.go`

~~~~diff
--- a/content/skygear/copier/subscribe.go
+++ b/content/skygear/copier/subscribe.go
@@ -13,9 +13,10 @@ func subscribe() {
 
 // The configuration copier copies and pastes settings and does nothing else.
 // Where the island does not allow building it leaves the click alone: the
-// island's guard refuses it.
+// island's guard refuses it. A held tool takes the click away from what the
+// block itself would do with it: the Low tier.
 func subscribeUse() {
-	core.UseOnBlock.Handle(70, "copier.use", func(e core.UseOnBlockEvent) bool {
+	core.UseOnBlock.Handle(core.Low, "copier.use", func(e core.UseOnBlockEvent) bool {
 		if _, ok := e.Held.Item().(sgitem.ConfigCopier); ok && island.CanBuild(e.S, e.Pos) {
 			e.Cancel()
 			_, off := e.P.HeldItems()
@@ -24,7 +25,7 @@ func subscribeUse() {
 		}
 		return false
 	})
-	core.Use.On(50, "copier.clear", func(e core.UseEvent) {
+	core.Use.On(core.Normal, "copier.clear", func(e core.UseEvent) {
 		if _, ok := e.Held.Item().(sgitem.ConfigCopier); ok {
 			ClearCopier(e.S, e.P, e.Held, e.Off)
 		}
~~~~

### `content/skygear/core/coretest/coretest.go`

~~~~diff
--- a/content/skygear/core/coretest/coretest.go
+++ b/content/skygear/core/coretest/coretest.go
@@ -81,9 +81,13 @@ func entityTypes() []world.EntityType {
 // The result is the session as the instance sees it: its Session field is the
 // core session, its methods are the calls the instance makes (Join, Leave,
 // Second, Fast, NewDay, Autosave, Close) and Handler is a player's handler.
+//
+// The first call reorders the subscribers when the environment asks for it
+// (ReorderHooks).
 func Session(t testing.TB) core.Lifecycle {
 	t.Helper()
 	world.DefaultBlockRegistry.Finalize()
+	ReorderHooks()
 	w := world.Config{Synchronous: true, Entities: entity.DefaultRegistry.Config().New(entityTypes())}.New()
 	t.Cleanup(func() { _ = w.Close() })
 	c := core.New(core.Config{
~~~~

### `content/skygear/core/coretest/order.go` (new file)

~~~~go
package coretest

import (
	"fmt"
	"hash/fnv"
	"math/rand/v2"
	"os"
	"slices"
	"strings"
	"sync"

	"github.com/presentkim/skygear/content/skygear/core"
)

// HookOrderEnv names the environment variable that reorders the subscribers
// of every hook inside their tiers for a whole test binary: "reverse", or
// "shuffle:<seed>". A subscriber must not depend on its place inside its
// tier, so every test passes under every such order:
//
//	SKYGEAR_HOOK_ORDER=reverse go test ./content/skygear/...
const HookOrderEnv = "SKYGEAR_HOOK_ORDER"

var reorder sync.Once

// ReorderHooks reorders the subscribers of every hook inside their tiers as
// the environment asks (HookOrderEnv), once per test binary; without the
// variable it does nothing. kept are the pairs "<Hook> <first> <second>" of
// one tier that keep their order (the exceptions the root package lists).
//
// Session calls it, so the tests of a feature package run reordered as soon
// as the variable is set. The root package calls it from TestMain with its
// exceptions, before any event.
func ReorderHooks(kept ...string) {
	reorder.Do(func() {
		if mode := os.Getenv(HookOrderEnv); mode != "" {
			core.Reorder(orderFor(mode, kept))
		}
	})
}

// orderFor returns the reordering of mode for core.Reorder.
func orderFor(mode string, kept []string) func(hook string, subs []core.Subscription) []int {
	kind, arg, _ := strings.Cut(mode, ":")
	var seed uint64
	switch kind {
	case "reverse":
	case "shuffle":
		if _, err := fmt.Sscan(arg, &seed); err != nil {
			panic(HookOrderEnv + ": " + mode)
		}
	default:
		panic(HookOrderEnv + ": " + mode + " (want reverse or shuffle:<seed>)")
	}
	return func(hook string, subs []core.Subscription) []int {
		h := fnv.New64a()
		h.Write([]byte(hook))
		r := rand.New(rand.NewPCG(seed, h.Sum64()))
		idx := make([]int, 0, len(subs))
		for from := 0; from < len(subs); {
			to := from
			for to < len(subs) && subs[to].Tier == subs[from].Tier {
				to++
			}
			tier := make([]int, 0, to-from)
			for i := from; i < to; i++ {
				tier = append(tier, i)
			}
			if kind == "reverse" {
				slices.Reverse(tier)
			} else {
				r.Shuffle(len(tier), func(i, j int) { tier[i], tier[j] = tier[j], tier[i] })
			}
			idx = append(idx, tier...)
			from = to
		}
		// Put the kept pairs back in their order: a first that ended up after
		// its second moves to just before it.
		for moved := true; moved; {
			moved = false
			for _, k := range kept {
				f := strings.Fields(k)
				if len(f) != 3 {
					panic("coretest: kept pair " + k)
				}
				if f[0] != hook {
					continue
				}
				a := slices.IndexFunc(idx, func(i int) bool { return subs[i].Name == f[1] })
				b := slices.IndexFunc(idx, func(i int) bool { return subs[i].Name == f[2] })
				if a < 0 || b < 0 {
					continue // a test binary that did not set up both features
				}
				if subs[idx[a]].Tier != subs[idx[b]].Tier {
					panic("coretest: kept pair " + k + ": the two are not in one tier, their tiers order them")
				}
				if a > b {
					v := idx[a]
					idx = slices.Delete(idx, a, a+1)
					idx = slices.Insert(idx, b, v)
					moved = true
				}
			}
		}
		return idx
	}
}
~~~~

### `content/skygear/core/doc.go`

~~~~diff
--- a/content/skygear/core/doc.go
+++ b/content/skygear/core/doc.go
@@ -13,13 +13,16 @@
 //     each part owns some of its fields and a field belongs to one part only.
 //
 // Hooks are how a feature reacts to the server. A Hook is a typed event with
-// subscribers in explicit priority order (On never stops the chain, Handle
-// stops it by returning true). PlayerHandler and WorldHandler are the
-// Dragonfly handlers that emit the player and world hooks, and Lifecycle makes
-// the session calls of the instance (join, leave, new day, every second, fast,
-// autosave, close) emit theirs. The events carry the arguments Dragonfly
-// passed, including its pointers, and an Actor to cancel the event in
-// Dragonfly; cancelling and stopping the chain are independent.
+// subscribers in six tiers (Priority: Lowest to Monitor), each tier with one
+// role; inside a tier no subscriber depends on its place. On never cancels
+// the event for the subscribers that follow, Handle does by returning true,
+// and Always also receives an event that was cancelled so. PlayerHandler and
+// WorldHandler are the Dragonfly handlers that emit the player and world
+// hooks, and Lifecycle makes the session calls of the instance (join, leave,
+// new day, every second, fast, autosave, close) emit theirs. The events carry
+// the arguments Dragonfly passed, including its pointers, and an Actor to
+// cancel the event in Dragonfly, which is independent of cancelling it for
+// the subscribers.
 //
 // A Feature is what a feature package hands to the game: its name, entity
 // types, settings, item entry edit, creative items, commands, a Setup that
~~~~

### `content/skygear/core/events.go`

~~~~diff
--- a/content/skygear/core/events.go
+++ b/content/skygear/core/events.go
@@ -14,11 +14,12 @@ import (
 // Ctx is nil for the events Dragonfly cannot cancel (Jump, Death, Respawn);
 // Cancel then does nothing.
 //
-// Cancelling and stopping are two things. Cancel is Dragonfly's: the action
-// the player asked for does not happen, and the chain of subscribers goes on.
-// Stopping is the hook's: a Handle subscriber that returns true ends the
-// chain, and the event is not cancelled by it. A subscriber does either, both
-// or neither.
+// Cancelling has two sides. Cancel is Dragonfly's: the action the player
+// asked for does not happen, and the subscribers go on (Cancelled tells). The
+// hook's is a Handle subscriber returning true: the event is cancelled for
+// the subscribers that follow, which do not run unless they were registered
+// with Always, and Dragonfly's action is not cancelled by it. A subscriber
+// does either, both or neither.
 type Actor struct {
 	S   *Session
 	P   *player.Player
@@ -26,13 +27,20 @@ type Actor struct {
 	Ctx *player.Context
 }
 
-// Cancel cancels the event in Dragonfly. It does not stop the hook's chain.
+// Cancel cancels the event in Dragonfly. It does not keep the later
+// subscribers from running.
 func (a Actor) Cancel() {
 	if a.Ctx != nil {
 		a.Ctx.Cancel()
 	}
 }
 
+// Cancelled reports whether the event is cancelled in Dragonfly. It is false
+// for the events Dragonfly cannot cancel.
+func (a Actor) Cancelled() bool {
+	return a.Ctx != nil && a.Ctx.Cancelled()
+}
+
 // Pointer fields (Drops, XP, Damage, Pos, Pick) are the pointers Dragonfly
 // passed to the handler, not copies: a subscriber that writes through one
 // changes what Dragonfly does, exactly as a direct handler would. The other
@@ -173,8 +181,8 @@ type DespawnEvent struct {
 	Entity world.Entity
 }
 
-// Player hooks, one per method of PlayerHandler. Their subscribers run in
-// ascending priority; see Hook for stopping.
+// Player hooks, one per method of PlayerHandler. Their subscribers run tier
+// by tier; see Hook for cancelling.
 var (
 	StartBreak   = NewHook[StartBreakEvent]("StartBreak")
 	BlockBreak   = NewHook[BlockBreakEvent]("BlockBreak")
~~~~

### `content/skygear/core/feature_test.go`

~~~~diff
--- a/content/skygear/core/feature_test.go
+++ b/content/skygear/core/feature_test.go
@@ -31,10 +31,10 @@ func TestInstallRunsSetupUnderName(t *testing.T) {
 	ResetHooks()
 	h := NewHook[int]("T")
 	Install(
-		&Feature{Name: "a", Setup: func() { h.On(10, "a.x", func(int) {}) }},
-		&Feature{Name: "b", Setup: func() { h.On(20, "b.x", func(int) {}) }},
+		&Feature{Name: "a", Setup: func() { h.On(Normal, "a.x", func(int) {}) }},
+		&Feature{Name: "b", Setup: func() { h.On(Normal, "b.x", func(int) {}) }},
 	)
-	want := []Subscription{{Hook: "T", Prio: 10, Name: "a.x", Feature: "a"}, {Hook: "T", Prio: 20, Name: "b.x", Feature: "b"}}
+	want := []Subscription{{Hook: "T", Tier: Normal, Name: "a.x", Feature: "a"}, {Hook: "T", Tier: Normal, Name: "b.x", Feature: "b"}}
 	if got := Subscriptions(); !reflect.DeepEqual(got, want) {
 		t.Fatalf("got %v", got)
 	}
@@ -43,7 +43,7 @@ func TestInstallRunsSetupUnderName(t *testing.T) {
 	}
 	defer mustPanic(t, "b.x", "a")
 	ResetHooks()
-	Install(&Feature{Name: "a", Setup: func() { h.On(10, "b.x", func(int) {}) }})
+	Install(&Feature{Name: "a", Setup: func() { h.On(Normal, "b.x", func(int) {}) }})
 }
 
 func TestInstallTwicePanics(t *testing.T) {
~~~~

### `content/skygear/core/handler_test.go`

~~~~diff
--- a/content/skygear/core/handler_test.go
+++ b/content/skygear/core/handler_test.go
@@ -1,6 +1,7 @@
 package core
 
 import (
+	"slices"
 	"strings"
 	"sync"
 	"testing"
@@ -75,7 +76,7 @@ func rowOf[E any](name string, h *Hook[E], call func(x *fix), check func(t *test
 		call: call,
 		arm: func() func(*testing.T, *fix) {
 			var got []E
-			h.On(10, "t.rec", func(e E) { got = append(got, e) })
+			h.On(Normal, "t.rec", func(e E) { got = append(got, e) })
 			return func(t *testing.T, x *fix) {
 				t.Helper()
 				if len(got) != 1 {
@@ -84,7 +85,7 @@ func rowOf[E any](name string, h *Hook[E], call func(x *fix), check func(t *test
 				check(t, x, got[0])
 			}
 		},
-		nop: func() { h.On(10, "t.nop", func(E) {}) },
+		nop: func() { h.On(Normal, "t.nop", func(E) {}) },
 	}
 }
 
@@ -362,8 +363,8 @@ func TestEventsEmitAllocateNothing(t *testing.T) {
 func TestSneakAndSprintEmitOnStartOnly(t *testing.T) {
 	withFix(t, nil, func(x *fix) {
 		var sneaks, sprints int
-		Sneak.On(10, "t.sneak", func(PlayerEvent) { sneaks++ })
-		Sprint.On(10, "t.sprint", func(PlayerEvent) { sprints++ })
+		Sneak.On(Normal, "t.sneak", func(PlayerEvent) { sneaks++ })
+		Sprint.On(Normal, "t.sprint", func(PlayerEvent) { sprints++ })
 		x.h.HandleToggleSneak(x.ctx, false)
 		x.h.HandleToggleSprint(x.ctx, false)
 		if sneaks != 0 || sprints != 0 {
@@ -383,13 +384,13 @@ func TestCancelAndStopAreIndependent(t *testing.T) {
 	for _, c := range []struct{ cancel, stop bool }{{false, false}, {true, false}, {false, true}, {true, true}} {
 		withFix(t, nil, func(x *fix) {
 			ran := false
-			UseOnBlock.Handle(10, "t.first", func(e UseOnBlockEvent) bool {
+			UseOnBlock.Handle(Normal, "t.first", func(e UseOnBlockEvent) bool {
 				if c.cancel {
 					e.Cancel()
 				}
 				return c.stop
 			})
-			UseOnBlock.On(20, "t.second", func(UseOnBlockEvent) { ran = true })
+			UseOnBlock.On(Normal, "t.second", func(UseOnBlockEvent) { ran = true })
 			x.h.HandleItemUseOnBlock(x.ctx, pos, cube.FaceUp, vec)
 			if got := x.ctx.Cancelled(); got != c.cancel {
 				t.Errorf("cancel=%v stop=%v: cancelled = %v", c.cancel, c.stop, got)
@@ -401,11 +402,59 @@ func TestCancelAndStopAreIndependent(t *testing.T) {
 	}
 }
 
+// TestLeaveDropsThePlayersEntriesLast: Leave drops the player's Players
+// entries when every subscriber has run, so a subscriber of any tier finds
+// them, and a subscriber that cancels the event does not keep them from
+// being dropped.
+func TestLeaveDropsThePlayersEntriesLast(t *testing.T) {
+	part := NewPlayers[int]()
+	for _, stop := range []bool{false, true} {
+		withFix(t, nil, func(x *fix) {
+			part.Of(x.s)[x.p.H()] = 7
+			var seen []int
+			var traced []string
+			see := func(e Presence) { seen = append(seen, part.Of(e.S)[e.P.H()]) }
+			Leave.On(Lowest, "t.first", see)
+			Leave.Handle(Normal, "t.stop", func(e Presence) bool { see(e); return stop })
+			Leave.Always(Monitor, "t.last", func(e Presence, _ bool) { see(e) })
+			traceInto(&traced)
+			x.l.Leave(x.p)
+			Trace = nil
+			if !slices.Equal(seen, []int{7, 7, 7}) {
+				t.Errorf("stop=%t: the subscribers saw %v", stop, seen)
+			}
+			if _, ok := part.Of(x.s)[x.p.H()]; ok {
+				t.Errorf("stop=%t: the player's entry was not dropped", stop)
+			}
+			if want := []string{"t.first", "t.stop", "t.last", "core.shapes", "core.parts"}; !slices.Equal(traced, want) {
+				t.Errorf("stop=%t: traced %v, want %v", stop, traced, want)
+			}
+		})
+	}
+}
+
+// TestCancelledTellsTheContext: Cancelled reports Dragonfly's cancel, and is
+// false for an event that has no context.
+func TestCancelledTellsTheContext(t *testing.T) {
+	withFix(t, nil, func(x *fix) {
+		var before, after, jump bool
+		Sneak.On(Low, "t.before", func(e PlayerEvent) { before = e.Cancelled() })
+		Sneak.On(Normal, "t.cancel", func(e PlayerEvent) { e.Cancel() })
+		Sneak.On(High, "t.after", func(e PlayerEvent) { after = e.Cancelled() })
+		Jump.On(Normal, "t.jump", func(e PlayerEvent) { e.Cancel(); jump = e.Cancelled() })
+		x.h.HandleToggleSneak(x.ctx, true)
+		x.h.HandleJump(x.p)
+		if before || !after || jump {
+			t.Fatalf("Cancelled: before=%t after=%t jump=%t", before, after, jump)
+		}
+	})
+}
+
 // TestCancelWithoutContext: Cancel on an event Dragonfly cannot cancel does
 // nothing.
 func TestCancelWithoutContext(t *testing.T) {
 	withFix(t, nil, func(x *fix) {
-		Jump.On(10, "t.cancel", func(e PlayerEvent) { e.Cancel() })
+		Jump.On(Normal, "t.cancel", func(e PlayerEvent) { e.Cancel() })
 		x.h.HandleJump(x.p)
 	})
 }
@@ -414,11 +463,11 @@ func TestCancelWithoutContext(t *testing.T) {
 // field reaches the variable Dragonfly passed.
 func TestEventsCarryDragonflyPointers(t *testing.T) {
 	withFix(t, nil, func(x *fix) {
-		BlockBreak.On(10, "t.drops", func(e BlockBreakEvent) { *e.Drops = nil; *e.XP = 99 })
-		ItemDamage.On(10, "t.damage", func(e ItemDamageEvent) { *e.Damage = 0 })
-		Hurt.On(10, "t.hurt", func(e HurtEvent) { *e.Damage = 0.5 })
-		Respawn.On(10, "t.respawn", func(e RespawnEvent) { *e.Pos = mgl64.Vec3{1, 2, 3} })
-		ItemPickup.On(10, "t.pick", func(e ItemEvent) { *e.Pick = item.Stack{} })
+		BlockBreak.On(Normal, "t.drops", func(e BlockBreakEvent) { *e.Drops = nil; *e.XP = 99 })
+		ItemDamage.On(Normal, "t.damage", func(e ItemDamageEvent) { *e.Damage = 0 })
+		Hurt.On(Normal, "t.hurt", func(e HurtEvent) { *e.Damage = 0.5 })
+		Respawn.On(Normal, "t.respawn", func(e RespawnEvent) { *e.Pos = mgl64.Vec3{1, 2, 3} })
+		ItemPickup.On(Normal, "t.pick", func(e ItemEvent) { *e.Pick = item.Stack{} })
 
 		drops, xp := []item.Stack{stack}, 1
 		x.h.HandleBlockBreak(x.ctx, pos, &drops, &xp)
@@ -454,7 +503,7 @@ func TestLifecycleKernelStepsRunLast(t *testing.T) {
 			x.s.FlushInterval = time.Nanosecond
 			x.s.MarkDirty()
 			var during bool
-			Second.On(10, "t.step", func(Tick) { during = x.s.Dirty() })
+			Second.On(Normal, "t.step", func(Tick) { during = x.s.Dirty() })
 			x.l.Second(x.tx)
 			if !during {
 				t.Error("Second flushed before its subscribers")
@@ -468,7 +517,7 @@ func TestLifecycleKernelStepsRunLast(t *testing.T) {
 	t.Run("NewDay", func(t *testing.T) {
 		withFix(t, store, func(x *fix) {
 			var seen []string
-			NewDay.On(10, "t.step", func(DayEvent) {})
+			NewDay.On(Normal, "t.step", func(DayEvent) {})
 			Trace = func(name string) { seen = append(seen, name) }
 			defer func() { Trace = nil }()
 			x.l.NewDay(x.tx, 3)
@@ -481,7 +530,7 @@ func TestLifecycleKernelStepsRunLast(t *testing.T) {
 		withFix(t, store, func(x *fix) {
 			x.s.MarkDirty()
 			var during bool
-			Autosave.On(10, "t.step", func(Tick) { during = x.s.Dirty() })
+			Autosave.On(Normal, "t.step", func(Tick) { during = x.s.Dirty() })
 			x.l.Autosave(x.tx)
 			if !during {
 				t.Error("Autosave flushed before its subscribers")
@@ -497,7 +546,7 @@ func TestLifecycleKernelStepsRunLast(t *testing.T) {
 			before := storedNow()
 			var registered bool
 			var storedDuring int
-			Close.On(10, "t.step", func(e Tick) {
+			Close.On(Normal, "t.step", func(e Tick) {
 				_, registered = OfWorld(x.w)
 				storedDuring = storedNow()
 			})
~~~~

### `content/skygear/core/hook.go`

~~~~diff
--- a/content/skygear/core/hook.go
+++ b/content/skygear/core/hook.go
@@ -24,24 +24,83 @@ func Mark(name string) {
 // is set, every subscriber name must start with it plus a dot.
 var installing string
 
+// Priority is the tier a subscriber runs in. A hook runs its tiers in the
+// order of the constants. Each tier has one role; a subscriber takes the tier
+// of what it does, not a place before or after another subscriber.
+type Priority uint8
+
+const (
+	// Lowest is preprocessing: for a subscriber that must shape or block the
+	// event itself before anything else.
+	Lowest Priority = iota
+	// Low is for shaping or blocking the event that may happen after the
+	// preprocessing.
+	Low
+	// Normal is the general tier, for what may happen after the
+	// preprocessing.
+	Normal
+	// High is for work of low priority that is not postprocessing.
+	High
+	// Highest is the last postprocessing that may still shape the event.
+	Highest
+	// Monitor is for handling the result without shaping or cancelling the
+	// event: records, achievements and the like.
+	Monitor
+)
+
+var priorityNames = [...]string{"LOWEST", "LOW", "NORMAL", "HIGH", "HIGHEST", "MONITOR"}
+
+// String returns the tier's name in capitals, as the order pin writes it.
+func (p Priority) String() string {
+	if int(p) < len(priorityNames) {
+		return priorityNames[p]
+	}
+	return fmt.Sprintf("Priority(%d)", uint8(p))
+}
+
 // Subscription is one registered subscriber, as Subscriptions lists it.
+// Always is set for a subscriber that also receives a cancelled event
+// (Hook.Always).
 type Subscription struct {
 	Hook    string
 	Name    string
 	Feature string
-	Prio    int
+	Tier    Priority
+	Always  bool
 }
 
+// A step has exactly one of on and handle. An Always step has on (the
+// subscriber told that the event is live) and always set; what it is called
+// with for a cancelled event is in Hook.after.
 type step[E any] struct {
-	prio    int
+	tier    Priority
+	always  bool
 	name    string
 	feature string
 	on      func(E)
 	handle  func(E) bool
 }
 
-// Hook is a typed event with subscribers in explicit priority order. The event
-// is passed by value, so Emit allocates nothing.
+// afterStep is an Always subscriber as the walk after a cancel calls it: at is
+// its index in Hook.steps.
+type afterStep[E any] struct {
+	at   int
+	name string
+	f    func(E, bool)
+}
+
+// Hook is a typed event with subscribers in tiers. The event is passed by
+// value, so Emit allocates nothing.
+//
+// The tiers run in the order of the Priority constants. Inside a tier the
+// subscribers run in the order they were registered (the features in
+// installation order, each feature's calls in the order it made them). That
+// order only makes a run repeatable: a subscriber must not depend on its
+// place inside its tier.
+//
+// An event is cancelled, as far as its hook is concerned, when a Handle
+// subscriber returned true. The subscribers after it do not run, except those
+// registered with Always.
 //
 // Subscribers are registered at startup, before the first Emit; registering
 // afterwards panics. Once registered, Emit only reads the hook, so any number
@@ -49,9 +108,16 @@ type step[E any] struct {
 // panic in a subscriber propagates to the caller unchanged and later
 // subscribers do not run.
 type Hook[E any] struct {
-	name   string
-	steps  []step[E] // ascending prio
-	sealed atomic.Bool
+	name  string
+	steps []step[E] // ascending tier, registration order inside a tier
+	// alwaysF are the functions of the Always subscribers by name; after are
+	// the same subscribers in the order they run, and lastAlways is the index
+	// in steps of the last of them, -1 when there is none (index keeps the two
+	// up to date).
+	alwaysF    map[string]func(E, bool)
+	after      []afterStep[E]
+	lastAlways int
+	sealed     atomic.Bool
 }
 
 type anyHook interface {
@@ -69,73 +135,106 @@ var (
 // from package-level variable declarations, so hooks list in declaration
 // order.
 func NewHook[E any](name string) *Hook[E] {
-	h := &Hook[E]{name: name}
+	h := &Hook[E]{name: name, lastAlways: -1}
 	hooksMu.Lock()
 	hooks = append(hooks, h)
 	hooksMu.Unlock()
 	return h
 }
 
-// On subscribes f at priority prio. A subscriber added with On never stops the
-// event.
-func (h *Hook[E]) On(prio int, name string, f func(E)) {
+// On subscribes f in tier prio. A subscriber added with On never cancels the
+// event, and does not run for an event an earlier subscriber cancelled.
+func (h *Hook[E]) On(prio Priority, name string, f func(E)) {
 	if f == nil {
 		panic(fmt.Sprintf("core: hook %q: subscriber %q has a nil func", h.name, name))
 	}
-	h.add(step[E]{prio: prio, name: name, on: f})
+	h.add(step[E]{tier: prio, name: name, on: f}, nil)
 }
 
-// Handle subscribes f at priority prio. When f returns true the event is
-// handled: later subscribers do not run and Emit returns true.
-func (h *Hook[E]) Handle(prio int, name string, f func(E) bool) {
+// Handle subscribes f in tier prio. When f returns true the event is
+// cancelled: later subscribers do not run, except those registered with
+// Always, and Emit returns true. Handle panics for the Monitor tier, whose
+// subscribers do not cancel.
+func (h *Hook[E]) Handle(prio Priority, name string, f func(E) bool) {
 	if f == nil {
 		panic(fmt.Sprintf("core: hook %q: subscriber %q has a nil func", h.name, name))
 	}
-	h.add(step[E]{prio: prio, name: name, handle: f})
+	if prio == Monitor {
+		panic(fmt.Sprintf("core: hook %q: subscriber %q: a Monitor subscriber does not cancel the event; use On or Always", h.name, name))
+	}
+	h.add(step[E]{tier: prio, name: name, handle: f}, nil)
 }
 
-func (h *Hook[E]) add(s step[E]) {
+// Always subscribes f in tier prio like On, and f also receives the event
+// when an earlier subscriber cancelled it (the handleCancelled option of
+// other event systems): cancelled then is true, and false for a live event.
+// Like On, f never cancels the event itself.
+func (h *Hook[E]) Always(prio Priority, name string, f func(e E, cancelled bool)) {
+	if f == nil {
+		panic(fmt.Sprintf("core: hook %q: subscriber %q has a nil func", h.name, name))
+	}
+	h.add(step[E]{tier: prio, always: true, name: name, on: func(e E) { f(e, false) }}, f)
+}
+
+// add registers s; always is the function of an Always subscriber, nil for
+// the others.
+func (h *Hook[E]) add(s step[E], always func(E, bool)) {
 	if h.sealed.Load() {
 		panic(fmt.Sprintf("core: hook %q: subscriber %q registered after the first Emit", h.name, s.name))
 	}
+	if s.tier > Monitor {
+		panic(fmt.Sprintf("core: hook %q: subscriber %q: %v is not a tier", h.name, s.name, s.tier))
+	}
 	if installing != "" {
 		if !strings.HasPrefix(s.name, installing+".") {
 			panic(fmt.Sprintf("core: hook %q: subscriber %q of feature %q must be named %q.<step>", h.name, s.name, installing, installing))
 		}
 		s.feature = installing
 	}
-	i := 0
-	for ; i < len(h.steps); i++ {
+	// After the last subscriber of its tier: registration order inside a tier.
+	at := len(h.steps)
+	for i := range h.steps {
 		o := &h.steps[i]
 		if o.name == s.name {
 			panic(fmt.Sprintf("core: hook %q: subscriber name %q registered twice", h.name, s.name))
 		}
-		if o.prio == s.prio {
-			panic(fmt.Sprintf("core: hook %q: priority %d taken by %q, cannot add %q", h.name, s.prio, o.name, s.name))
-		}
-	}
-	// Names were all checked above; find the sorted position.
-	at := len(h.steps)
-	for i := range h.steps {
-		if h.steps[i].prio > s.prio {
+		if o.tier > s.tier && at == len(h.steps) {
 			at = i
-			break
 		}
 	}
 	h.steps = append(h.steps, step[E]{})
 	copy(h.steps[at+1:], h.steps[at:])
 	h.steps[at] = s
+	if always != nil {
+		if h.alwaysF == nil {
+			h.alwaysF = map[string]func(E, bool){}
+		}
+		h.alwaysF[s.name] = always
+	}
+	h.index()
+}
+
+// index lists the Always steps in the order they run.
+func (h *Hook[E]) index() {
+	h.after, h.lastAlways = nil, -1
+	for i := range h.steps {
+		if s := &h.steps[i]; s.always {
+			h.after = append(h.after, afterStep[E]{at: i, name: s.name, f: h.alwaysF[s.name]})
+			h.lastAlways = i
+		}
+	}
 }
 
-// Emit runs the subscribers in ascending priority and reports whether a Handle
-// subscriber stopped the walk. It is safe to call from many goroutines at
-// once.
+// Emit runs the subscribers tier by tier and reports whether a Handle
+// subscriber cancelled the event. After a cancel only the Always subscribers
+// still to come run. It is safe to call from many goroutines at once.
 func (h *Hook[E]) Emit(e E) bool {
 	if !h.sealed.Load() {
 		h.sealed.Store(true)
 	}
 	tr := Trace
-	for i := range h.steps {
+	i := 0
+	for ; i < len(h.steps); i++ {
 		s := &h.steps[i]
 		if tr != nil {
 			tr(s.name)
@@ -143,22 +242,47 @@ func (h *Hook[E]) Emit(e E) bool {
 		if s.on != nil {
 			s.on(e)
 		} else if s.handle(e) {
-			return true
+			break
+		}
+	}
+	if i == len(h.steps) {
+		return false
+	}
+	if i < h.lastAlways {
+		h.emitCancelled(e, i)
+	}
+	return true
+}
+
+// emitCancelled runs the Always subscribers that stand after the step at
+// index stop, for an event that step cancelled.
+//
+//go:noinline
+func (h *Hook[E]) emitCancelled(e E, stop int) {
+	tr := Trace
+	for i := range h.after {
+		s := &h.after[i]
+		if s.at <= stop {
+			continue
+		}
+		if tr != nil {
+			tr(s.name)
 		}
+		s.f(e, true)
 	}
-	return false
 }
 
 func (h *Hook[E]) subscriptions() []Subscription {
 	out := make([]Subscription, len(h.steps))
 	for i, s := range h.steps {
-		out[i] = Subscription{Hook: h.name, Name: s.name, Feature: s.feature, Prio: s.prio}
+		out[i] = Subscription{Hook: h.name, Name: s.name, Feature: s.feature, Tier: s.tier, Always: s.always}
 	}
 	return out
 }
 
 func (h *Hook[E]) reset() {
-	h.steps = nil
+	h.steps, h.alwaysF, h.after = nil, nil, nil
+	h.lastAlways = -1
 	h.sealed.Store(false)
 }
 
~~~~

### `content/skygear/core/hook_test.go`

~~~~diff
--- a/content/skygear/core/hook_test.go
+++ b/content/skygear/core/hook_test.go
@@ -31,29 +31,97 @@ func traceInto(got *[]string) {
 	Trace = func(name string) { *got = append(*got, name) }
 }
 
-func TestRunsInPriorityOrder(t *testing.T) {
+// TestRunsInTierOrder: the tiers run in the order of the constants, whatever
+// the order of registration, with every kind of subscriber.
+func TestRunsInTierOrder(t *testing.T) {
 	ResetHooks()
 	h := NewHook[int]("order")
 	var got []string
-	h.On(30, "c", func(int) { got = append(got, "c") })
-	h.Handle(10, "a", func(int) bool { got = append(got, "a"); return false })
-	h.On(20, "b", func(int) { got = append(got, "b") })
+	rec := func(name string) func(int) { return func(int) { got = append(got, name) } }
+	h.On(Monitor, "monitor", rec("monitor"))
+	h.Always(Highest, "highest", func(int, bool) { got = append(got, "highest") })
+	h.On(High, "high", rec("high"))
+	h.Handle(Normal, "normal", func(int) bool { got = append(got, "normal"); return false })
+	h.On(Low, "low", rec("low"))
+	h.Handle(Lowest, "lowest", func(int) bool { got = append(got, "lowest"); return false })
 	if h.Emit(1) {
-		t.Fatal("Emit reported a stop")
+		t.Fatal("Emit reported a cancel")
 	}
-	if want := []string{"a", "b", "c"}; !slices.Equal(got, want) {
+	if want := []string{"lowest", "low", "normal", "high", "highest", "monitor"}; !slices.Equal(got, want) {
 		t.Fatalf("got %v want %v", got, want)
 	}
 }
 
+// TestSameTierRunsInRegistrationOrder: inside a tier the subscribers run in
+// the order they were registered, and a later registration in an earlier tier
+// still runs before them.
+func TestSameTierRunsInRegistrationOrder(t *testing.T) {
+	ResetHooks()
+	h := NewHook[int]("tier")
+	var got []string
+	rec := func(name string) func(int) { return func(int) { got = append(got, name) } }
+	h.On(Normal, "b.x", rec("b.x"))
+	h.On(Normal, "a.x", rec("a.x"))
+	h.On(Monitor, "m.x", rec("m.x"))
+	h.On(Normal, "c.x", rec("c.x"))
+	h.On(Low, "l.x", rec("l.x"))
+	h.Emit(0)
+	want := []string{"l.x", "b.x", "a.x", "c.x", "m.x"}
+	if !slices.Equal(got, want) {
+		t.Fatalf("got %v want %v", got, want)
+	}
+	var listed []string
+	for _, s := range Subscriptions() {
+		if s.Hook == "tier" {
+			listed = append(listed, s.Name)
+		}
+	}
+	if !slices.Equal(listed, want) {
+		t.Fatalf("Subscriptions lists %v, want %v", listed, want)
+	}
+}
+
+func TestPriorityNames(t *testing.T) {
+	var got []string
+	for p := Lowest; p <= Monitor; p++ {
+		got = append(got, p.String())
+	}
+	if want := []string{"LOWEST", "LOW", "NORMAL", "HIGH", "HIGHEST", "MONITOR"}; !slices.Equal(got, want) {
+		t.Fatalf("got %v", got)
+	}
+	if got := Priority(9).String(); got != "Priority(9)" {
+		t.Fatalf("got %q", got)
+	}
+}
+
+func TestNotATierPanics(t *testing.T) {
+	ResetHooks()
+	h := NewHook[int]("notier")
+	defer mustPanic(t, "notier", "a.x", "Priority(6)", "not a tier")
+	h.On(Monitor+1, "a.x", func(int) {})
+}
+
+// TestHandleAtMonitorPanics: a Monitor subscriber does not cancel, so it is
+// registered with On or Always.
+func TestHandleAtMonitorPanics(t *testing.T) {
+	ResetHooks()
+	h := NewHook[int]("monitor")
+	h.On(Monitor, "a.x", func(int) {})
+	h.Always(Monitor, "b.x", func(int, bool) {})
+	defer mustPanic(t, "monitor", "c.x", "Monitor", "does not cancel")
+	h.Handle(Monitor, "c.x", func(int) bool { return false })
+}
+
 func TestStopSkipsLaterSteps(t *testing.T) {
 	ResetHooks()
 	h := NewHook[int]("stop")
 	var traced, ran []string
 	traceInto(&traced)
 	defer func() { Trace = nil }()
-	h.Handle(10, "a.x", func(int) bool { ran = append(ran, "a.x"); return true })
-	h.On(20, "b.x", func(int) { ran = append(ran, "b.x") })
+	h.Handle(Normal, "a.x", func(int) bool { ran = append(ran, "a.x"); return true })
+	h.On(Normal, "b.x", func(int) { ran = append(ran, "b.x") })
+	h.Handle(High, "c.x", func(int) bool { ran = append(ran, "c.x"); return false })
+	h.On(Monitor, "d.x", func(int) { ran = append(ran, "d.x") })
 	if !h.Emit(1) {
 		t.Fatal("Emit did not report the stop")
 	}
@@ -62,55 +130,182 @@ func TestStopSkipsLaterSteps(t *testing.T) {
 	}
 }
 
+// alwaysCase is one arrangement of TestAlways: the tier of the subscriber
+// that may cancel, and the tier of the two subscribers under test (one
+// registered with On, one with Always), registered before or after it.
+type alwaysCase struct {
+	stopTier, tier Priority
+	before         bool
+}
+
+// TestAlways: an On subscriber and an Always subscriber, in every tier,
+// before and after a Handle subscriber in every tier but Monitor, for an
+// event that is not cancelled, one that the Handle subscriber cancels and one
+// whose Dragonfly context a subscriber cancelled while the walk went on.
+//
+//   - a live event: both run, the Always subscriber is told cancelled=false;
+//   - a cancelled event: a subscriber that stands before the cancelling one
+//     ran already (told false); one that stands after it runs only when it
+//     was registered with Always, and is told true;
+//   - a cancelled context without a cancelled walk is a live event for the
+//     hook: both run and are told false; Actor.Cancelled tells them.
+func TestAlways(t *testing.T) {
+	var cases []alwaysCase
+	for stop := Lowest; stop <= Highest; stop++ {
+		for tier := Lowest; tier <= Monitor; tier++ {
+			if tier != stop {
+				cases = append(cases, alwaysCase{stop, tier, tier < stop})
+				continue
+			}
+			cases = append(cases, alwaysCase{stop, tier, true}, alwaysCase{stop, tier, false})
+		}
+	}
+	for _, c := range cases {
+		for _, mode := range []string{"live", "cancelled", "context"} {
+			withFix(t, nil, func(x *fix) {
+				var ran []string
+				var onSaw, alwaysSaw bool
+				under := func() {
+					Sneak.On(c.tier, "t.on", func(e PlayerEvent) {
+						ran = append(ran, "on")
+						onSaw = e.Cancelled()
+					})
+					Sneak.Always(c.tier, "t.always", func(e PlayerEvent, cancelled bool) {
+						ran = append(ran, fmt.Sprint("always=", cancelled))
+						alwaysSaw = e.Cancelled()
+					})
+				}
+				if c.before {
+					under()
+				}
+				Sneak.Handle(c.stopTier, "t.stop", func(e PlayerEvent) bool {
+					ran = append(ran, "stop")
+					if mode == "context" {
+						e.Cancel()
+					}
+					return mode == "cancelled"
+				})
+				if !c.before {
+					under()
+				}
+				stopped := Sneak.Emit(PlayerEvent{Actor{S: x.s, P: x.p, Tx: x.tx, Ctx: x.ctx}})
+				if stopped != (mode == "cancelled") {
+					t.Errorf("%+v %s: Emit = %t", c, mode, stopped)
+				}
+				var want []string
+				switch {
+				case c.before:
+					want = []string{"on", "always=false", "stop"}
+				case mode == "cancelled":
+					want = []string{"stop", "always=true"}
+				default:
+					want = []string{"stop", "on", "always=false"}
+				}
+				if !slices.Equal(ran, want) {
+					t.Errorf("%+v %s: ran %v, want %v", c, mode, ran, want)
+				}
+				// The context is cancelled for those that run after the cancel.
+				if wantSaw := mode == "context" && !c.before; alwaysSaw != wantSaw || (onSaw != wantSaw) {
+					t.Errorf("%+v %s: Cancelled() was %t for On and %t for Always, want %t", c, mode, onSaw, alwaysSaw, wantSaw)
+				}
+			})
+		}
+	}
+	if len(cases) != 35 {
+		t.Fatalf("%d arrangements", len(cases))
+	}
+}
+
+// TestAlwaysAfterCancelRunsOnlyAlways: after a cancel the Always subscribers
+// still to come run in order, traced like any other, whatever stands between
+// them; a second Handle subscriber is not asked, and Emit still reports the
+// cancel.
+func TestAlwaysAfterCancelRunsOnlyAlways(t *testing.T) {
+	ResetHooks()
+	h := NewHook[int]("always")
+	var traced, ran []string
+	traceInto(&traced)
+	defer func() { Trace = nil }()
+	rec := func(name string) func(int, bool) {
+		return func(_ int, cancelled bool) { ran = append(ran, fmt.Sprint(name, "=", cancelled)) }
+	}
+	h.Always(Lowest, "a", rec("a"))
+	h.Handle(Low, "stop", func(int) bool { ran = append(ran, "stop"); return true })
+	h.Always(Low, "b", rec("b"))
+	h.On(Normal, "on", func(int) { ran = append(ran, "on") })
+	h.Handle(Normal, "handle", func(int) bool { ran = append(ran, "handle"); return true })
+	h.Always(High, "c", rec("c"))
+	h.On(Monitor, "monitor", func(int) { ran = append(ran, "monitor") })
+	h.Always(Monitor, "d", rec("d"))
+	if !h.Emit(0) {
+		t.Fatal("Emit did not report the cancel")
+	}
+	if want := []string{"a=false", "stop", "b=true", "c=true", "d=true"}; !slices.Equal(ran, want) {
+		t.Fatalf("ran %v, want %v", ran, want)
+	}
+	if want := []string{"a", "stop", "b", "c", "d"}; !slices.Equal(traced, want) {
+		t.Fatalf("traced %v, want %v", traced, want)
+	}
+	var always []string
+	for _, s := range Subscriptions() {
+		if s.Hook == "always" && s.Always {
+			always = append(always, s.Name)
+		}
+	}
+	if want := []string{"a", "b", "c", "d"}; !slices.Equal(always, want) {
+		t.Fatalf("Subscriptions marks %v as Always, want %v", always, want)
+	}
+}
+
 func TestOnNeverStops(t *testing.T) {
 	ResetHooks()
 	h := NewHook[int]("onstop")
 	n := 0
-	h.On(1, "a", func(int) { n++ })
-	h.Handle(2, "b", func(int) bool { n++; return false })
-	h.On(3, "c", func(int) { n++ })
+	h.On(Lowest, "a", func(int) { n++ })
+	h.Handle(Normal, "b", func(int) bool { n++; return false })
+	h.On(High, "c", func(int) { n++ })
 	if h.Emit(0) || n != 3 {
 		t.Fatalf("stopped=%v n=%d", h.Emit(0), n)
 	}
 }
 
-func TestSamePriorityPanics(t *testing.T) {
-	ResetHooks()
-	h := NewHook[int]("prio")
-	h.On(10, "a.x", func(int) {})
-	defer mustPanic(t, "prio", "10", "a.x", "b.x")
-	h.On(10, "b.x", func(int) {})
-}
-
 func TestSameNamePanics(t *testing.T) {
 	ResetHooks()
 	h := NewHook[int]("name")
-	h.On(10, "a.x", func(int) {})
+	h.On(Normal, "a.x", func(int) {})
 	defer mustPanic(t, "name", "a.x")
-	h.Handle(20, "a.x", func(int) bool { return false })
+	h.Handle(High, "a.x", func(int) bool { return false })
 }
 
 func TestNilFuncPanics(t *testing.T) {
 	ResetHooks()
 	h := NewHook[int]("nilf")
+	func() {
+		defer mustPanic(t, "nilf", "h")
+		h.Handle(Normal, "h", nil)
+	}()
+	func() {
+		defer mustPanic(t, "nilf", "w")
+		h.Always(Normal, "w", nil)
+	}()
 	defer mustPanic(t, "nilf", "a")
-	h.On(1, "a", nil)
+	h.On(Normal, "a", nil)
 }
 
 func TestInstallingPrefix(t *testing.T) {
 	ResetHooks()
 	h := NewHook[int]("pfx")
 	SetInstalling("feat")
-	h.On(1, "feat.ok", func(int) {})
+	h.On(Normal, "feat.ok", func(int) {})
 	func() {
 		defer mustPanic(t, "pfx", "other.bad", "feat")
-		h.On(2, "other.bad", func(int) {})
+		h.On(Normal, "other.bad", func(int) {})
 	}()
 	found := false
 	for _, s := range Subscriptions() {
 		if s.Hook == "pfx" {
 			found = true
-			if s.Feature != "feat" || s.Name != "feat.ok" || s.Prio != 1 {
+			if s.Feature != "feat" || s.Name != "feat.ok" || s.Tier != Normal || s.Always {
 				t.Fatalf("subscription %+v", s)
 			}
 		}
@@ -123,10 +318,10 @@ func TestInstallingPrefix(t *testing.T) {
 func TestRegisterAfterEmitPanics(t *testing.T) {
 	ResetHooks()
 	h := NewHook[int]("late")
-	h.On(1, "a", func(int) {})
+	h.On(Normal, "a", func(int) {})
 	h.Emit(0)
 	defer mustPanic(t, "late", "b")
-	h.On(2, "b", func(int) {})
+	h.On(Normal, "b", func(int) {})
 }
 
 func TestEmptyEmitStillSealsRegistration(t *testing.T) {
@@ -134,21 +329,21 @@ func TestEmptyEmitStillSealsRegistration(t *testing.T) {
 	h := NewHook[int]("early")
 	h.Emit(0) // no steps: an empty emit still counts as emission
 	defer mustPanic(t, "early")
-	h.On(1, "a", func(int) {})
+	h.On(Normal, "a", func(int) {})
 }
 
 func TestSubscriptionsRunOrder(t *testing.T) {
 	ResetHooks()
 	h := NewHook[int]("T")
-	h.On(20, "b.x", func(int) {})
-	h.On(10, "a.x", func(int) {})
+	h.Always(High, "b.x", func(int, bool) {})
+	h.On(Low, "a.x", func(int) {})
 	var got []Subscription
 	for _, s := range Subscriptions() {
 		if s.Hook == "T" {
 			got = append(got, s)
 		}
 	}
-	want := []Subscription{{Hook: "T", Prio: 10, Name: "a.x"}, {Hook: "T", Prio: 20, Name: "b.x"}}
+	want := []Subscription{{Hook: "T", Tier: Low, Name: "a.x"}, {Hook: "T", Tier: High, Name: "b.x", Always: true}}
 	if !slices.Equal(got, want) {
 		t.Fatalf("got %+v want %+v", got, want)
 	}
@@ -158,8 +353,8 @@ func TestSubscriptionsHookDeclarationOrder(t *testing.T) {
 	ResetHooks()
 	h1 := NewHook[int]("decl1")
 	h2 := NewHook[int]("decl2")
-	h2.On(1, "z", func(int) {})
-	h1.On(5, "y", func(int) {})
+	h2.On(Lowest, "z", func(int) {})
+	h1.On(Monitor, "y", func(int) {})
 	var names []string
 	for _, s := range Subscriptions() {
 		if s.Hook == "decl1" || s.Hook == "decl2" {
@@ -175,8 +370,8 @@ func TestTraceNamesInOrder(t *testing.T) {
 	ResetHooks()
 	h := NewHook[int]("trace")
 	var traced, ran []string
-	h.On(2, "b", func(int) { ran = append(ran, fmt.Sprint(len(traced), ":b")) })
-	h.On(1, "a", func(int) { ran = append(ran, fmt.Sprint(len(traced), ":a")) })
+	h.On(Normal, "b", func(int) { ran = append(ran, fmt.Sprint(len(traced), ":b")) })
+	h.On(Low, "a", func(int) { ran = append(ran, fmt.Sprint(len(traced), ":a")) })
 	traceInto(&traced)
 	defer func() { Trace = nil }()
 	h.Emit(0)
@@ -206,8 +401,8 @@ func TestSubscriberPanicPropagates(t *testing.T) {
 	ResetHooks()
 	h := NewHook[int]("boom")
 	later := false
-	h.On(1, "a", func(int) { panic("boom!") })
-	h.On(2, "b", func(int) { later = true })
+	h.On(Normal, "a", func(int) { panic("boom!") })
+	h.On(Normal, "b", func(int) { later = true })
 	func() {
 		defer func() {
 			if r := recover(); r != "boom!" {
@@ -226,7 +421,7 @@ func TestEmitConcurrent(t *testing.T) {
 	h := NewHook[int]("conc")
 	var n atomic.Int64
 	for i := 0; i < 8; i++ {
-		h.On(i, fmt.Sprint("s", i), func(e int) { n.Add(int64(e)) })
+		h.On(Priority(i%6), fmt.Sprint("s", i), func(e int) { n.Add(int64(e)) })
 	}
 	var wg sync.WaitGroup
 	for g := 0; g < 16; g++ {
@@ -244,13 +439,15 @@ func TestEmitConcurrent(t *testing.T) {
 	}
 }
 
+// fillHook gives h n subscribers spread over the tiers (Handle subscribers
+// over the five tiers that may cancel).
 func fillHook(h *Hook[bigEvent], n int, handle bool) {
 	for i := 0; i < n; i++ {
 		name := fmt.Sprint("s", i)
 		if handle {
-			h.Handle(i, name, func(e bigEvent) bool { return e.A[0] == 99 })
+			h.Handle(Priority(i%5), name, func(e bigEvent) bool { return e.A[0] == 99 })
 		} else {
-			h.On(i, name, func(e bigEvent) { _ = e.A[0] })
+			h.On(Priority(i%6), name, func(e bigEvent) { _ = e.A[0] })
 		}
 	}
 }
@@ -272,6 +469,34 @@ func TestEmitAllocatesNothing(t *testing.T) {
 	}
 }
 
+// TestEmitAllocatesNothingAfterCancel: the walk of the Always subscribers
+// after a cancel allocates nothing either.
+func TestEmitAllocatesNothingAfterCancel(t *testing.T) {
+	ResetHooks()
+	h := NewHook[bigEvent]("alloc-always")
+	seen := 0
+	h.Handle(Lowest, "stop", func(e bigEvent) bool { return e.A[0] == 1 })
+	h.On(Normal, "on", func(bigEvent) {})
+	h.Always(Normal, "a", func(_ bigEvent, cancelled bool) {
+		if cancelled {
+			seen++
+		}
+	})
+	h.Always(Monitor, "b", func(_ bigEvent, cancelled bool) {
+		if cancelled {
+			seen++
+		}
+	})
+	var e bigEvent
+	e.A[0] = 1
+	if got := testing.AllocsPerRun(200, func() { h.Emit(e) }); got != 0 {
+		t.Errorf("%v allocs", got)
+	}
+	if seen == 0 || seen%2 != 0 {
+		t.Fatalf("the Always subscribers saw %d cancelled events", seen)
+	}
+}
+
 func BenchmarkEmit(b *testing.B) {
 	for _, handle := range []bool{false, true} {
 		for _, n := range []int{0, 1, 4, 8, 32} {
@@ -292,15 +517,15 @@ func BenchmarkEmit(b *testing.B) {
 
 var moveSink float64
 
-// BenchmarkEmitMove emits a 72-byte MoveEvent to four mixed subscribers, the
-// shape of the move hook: On, Handle (not stopping), On, On.
+// BenchmarkEmitMove emits a MoveEvent to four mixed subscribers, the shape of
+// the move hook: Handle (not cancelling), On, On, Always.
 func BenchmarkEmitMove(b *testing.B) {
 	ResetHooks()
 	h := NewHook[MoveEvent]("bench-move")
-	h.On(10, "a", func(e MoveEvent) { moveSink += e.Rot[0] + e.Pos[0] })
-	h.Handle(20, "b", func(e MoveEvent) bool { return e.P == nil && e.Pos[1] > 1e9 })
-	h.On(30, "c", func(e MoveEvent) { moveSink += e.Pos[2] })
-	h.On(40, "d", func(e MoveEvent) {
+	h.Always(Monitor, "a", func(e MoveEvent, _ bool) { moveSink += e.Rot[0] + e.Pos[0] })
+	h.Handle(Lowest, "b", func(e MoveEvent) bool { return e.P == nil && e.Pos[1] > 1e9 })
+	h.On(Monitor, "c", func(e MoveEvent) { moveSink += e.Pos[2] })
+	h.On(Highest, "d", func(e MoveEvent) {
 		if e.Pos[1] < -1e9 {
 			e.Cancel()
 		}
~~~~

### `content/skygear/core/kernel.go`

~~~~diff
--- a/content/skygear/core/kernel.go
+++ b/content/skygear/core/kernel.go
@@ -1,20 +1,16 @@
 package core
 
-// KernelFeature is the kernel's own feature (the shapes and parts steps of
-// the lifecycle); the game lists it first.
+// KernelFeature is the kernel's own feature; the game lists it first, so its
+// subscriber is the first of its tier. It is a subscriber like any other: it
+// does not depend on its place inside the tier.
 var KernelFeature = &Feature{Name: "core", Setup: subscribeKernel}
 
-// Shapes are sent when a player joins and cleared, client-side, before the
-// player leaves the instance world; the core session drops the player's
-// per-player parts.
+// Shapes are sent when a player joins. What the kernel does when a player
+// leaves (clearing the shapes client-side, dropping the player's per-player
+// parts) is not a subscriber: Lifecycle.Leave does it when every subscriber
+// has run.
 func subscribeKernel() {
-	Join.On(40, "core.shapes", func(e Presence) {
+	Join.On(Normal, "core.shapes", func(e Presence) {
 		e.S.SendShapes(e.P)
 	})
-	Leave.On(20, "core.parts", func(e Presence) {
-		e.S.Leave(e.P)
-	})
-	Leave.On(50, "core.shapes", func(e Presence) {
-		e.P.RemoveAllDebugShapes()
-	})
 }
~~~~

### `content/skygear/core/lifecycle.go`

~~~~diff
--- a/content/skygear/core/lifecycle.go
+++ b/content/skygear/core/lifecycle.go
@@ -21,10 +21,16 @@ func (l Lifecycle) Join(p *player.Player, first bool) {
 	Join.Emit(Presence{S: l.Session, P: p, Tx: p.Tx(), First: first})
 }
 
-// Leave emits Leave. The player's Players entries are dropped by a subscriber
-// calling Session.Leave.
+// Leave emits Leave, then clears the shapes on the player's client and drops
+// the player's Players entries (Session.Leave). Both come after every
+// subscriber, whatever its tier and whether or not one cancelled the event: a
+// subscriber still finds the player's entries, and what it drew is cleared.
 func (l Lifecycle) Leave(p *player.Player) {
 	Leave.Emit(Presence{S: l.Session, P: p, Tx: p.Tx()})
+	Mark("core.shapes")
+	p.RemoveAllDebugShapes()
+	Mark("core.parts")
+	l.Session.Leave(p)
 }
 
 // Handler is the handler of a player who entered the save.
~~~~

### `content/skygear/core/newfeature_test.go`

~~~~diff
--- a/content/skygear/core/newfeature_test.go
+++ b/content/skygear/core/newfeature_test.go
@@ -13,7 +13,7 @@ func TestNewFeatureNeedsOnlyTheList(t *testing.T) {
 	defer ResetHooks()
 	var got []PlayerEvent
 	fake := &Feature{Name: "fake", Setup: func() {
-		Jump.On(10, "fake.jump", func(e PlayerEvent) { got = append(got, e) })
+		Jump.On(Normal, "fake.jump", func(e PlayerEvent) { got = append(got, e) })
 	}}
 	Install(fake)
 
@@ -22,7 +22,7 @@ func TestNewFeatureNeedsOnlyTheList(t *testing.T) {
 		t.Fatalf("the fake feature received %d events, want 1", len(got))
 	}
 	subs := Subscriptions()
-	if len(subs) != 1 || subs[0] != (Subscription{Hook: "Jump", Name: "fake.jump", Feature: "fake", Prio: 10}) {
+	if len(subs) != 1 || subs[0] != (Subscription{Hook: "Jump", Name: "fake.jump", Feature: "fake", Tier: Normal}) {
 		t.Fatalf("subscriptions %v", subs)
 	}
 	if fs := Features(); len(fs) != 1 || fs[0] != fake {
~~~~

### `content/skygear/core/reorder.go`

~~~~diff
--- a/content/skygear/core/reorder.go
+++ b/content/skygear/core/reorder.go
@@ -3,14 +3,15 @@ package core
 import "fmt"
 
 // Reorder is a seam for the tests that prove no subscriber depends on its
-// place among the others. It changes the order the subscribers of every hook
+// place inside its tier. It changes the order the subscribers of every hook
 // run in: order is called once per hook that has subscribers, with the hook's
 // name and its subscribers in the order they run, and returns the new order
-// as indexes into subs.
+// as indexes into subs. The new order may only move a subscriber inside its
+// tier.
 //
 // Call it after Install and before the first Emit. It panics when a hook has
-// emitted already or when order does not return every index once. Nothing
-// outside tests calls it.
+// emitted already, when order does not return every index once or when the
+// new order does not keep the tiers in order. Nothing outside tests calls it.
 func Reorder(order func(hook string, subs []Subscription) []int) {
 	hooksMu.Lock()
 	defer hooksMu.Unlock()
@@ -38,6 +39,10 @@ func (h *Hook[E]) reorder(order func(hook string, subs []Subscription) []int) {
 		}
 		seen[from] = true
 		steps[to] = h.steps[from]
+		if to > 0 && steps[to].tier < steps[to-1].tier {
+			panic(fmt.Sprintf("core: Reorder: hook %q: %q (%v) after %q (%v): a subscriber moves inside its tier only", h.name, steps[to].name, steps[to].tier, steps[to-1].name, steps[to-1].tier))
+		}
 	}
 	h.steps = steps
+	h.index()
 }
~~~~

### `content/skygear/core/reorder_test.go`

~~~~diff
--- a/content/skygear/core/reorder_test.go
+++ b/content/skygear/core/reorder_test.go
@@ -12,9 +12,9 @@ func TestReorderChangesTheRunOrder(t *testing.T) {
 	h := NewHook[int]("reorder")
 	empty := NewHook[int]("reorder-empty")
 	var ran []string
-	h.On(10, "a", func(int) { ran = append(ran, "a") })
-	h.On(20, "b", func(int) { ran = append(ran, "b") })
-	h.On(30, "c", func(int) { ran = append(ran, "c") })
+	h.On(Normal, "a", func(int) { ran = append(ran, "a") })
+	h.On(Normal, "b", func(int) { ran = append(ran, "b") })
+	h.On(Normal, "c", func(int) { ran = append(ran, "c") })
 	var asked []string
 	Reorder(func(hook string, subs []Subscription) []int {
 		asked = append(asked, hook)
@@ -48,7 +48,7 @@ func TestReorderChangesTheRunOrder(t *testing.T) {
 func TestReorderAfterEmitPanics(t *testing.T) {
 	ResetHooks()
 	h := NewHook[int]("reorder-late")
-	h.On(10, "a", func(int) {})
+	h.On(Normal, "a", func(int) {})
 	h.Emit(0)
 	defer mustPanic(t, "Reorder", "reorder-late", "emitted")
 	Reorder(func(string, []Subscription) []int { return []int{0} })
@@ -58,11 +58,40 @@ func TestReorderWantsEveryIndexOnce(t *testing.T) {
 	for _, bad := range [][]int{{0}, {0, 0}, {0, 2}, {1, 0, 0}} {
 		ResetHooks()
 		h := NewHook[int]("reorder-bad")
-		h.On(10, "a", func(int) {})
-		h.On(20, "b", func(int) {})
+		h.On(Normal, "a", func(int) {})
+		h.On(Normal, "b", func(int) {})
 		func() {
 			defer mustPanic(t, "Reorder", "reorder-bad")
 			Reorder(func(string, []Subscription) []int { return bad })
 		}()
 	}
 }
+
+// TestReorderKeepsTheTiers: a subscriber moves inside its tier only.
+func TestReorderKeepsTheTiers(t *testing.T) {
+	ResetHooks()
+	h := NewHook[int]("reorder-tier")
+	h.On(Low, "a", func(int) {})
+	h.On(Normal, "b", func(int) {})
+	defer mustPanic(t, "Reorder", "reorder-tier", "a", "b", "inside its tier")
+	Reorder(func(string, []Subscription) []int { return []int{1, 0} })
+}
+
+// TestReorderKeepsAlways: after a reordering the Always subscribers still
+// receive a cancelled event, wherever they stand now.
+func TestReorderKeepsAlways(t *testing.T) {
+	ResetHooks()
+	h := NewHook[int]("reorder-always")
+	var ran []string
+	h.Always(Normal, "w", func(_ int, cancelled bool) {
+		if cancelled {
+			ran = append(ran, "w")
+		}
+	})
+	h.Handle(Normal, "stop", func(int) bool { return true })
+	h.On(Normal, "on", func(int) { ran = append(ran, "on") })
+	Reorder(func(string, []Subscription) []int { return []int{1, 2, 0} })
+	if !h.Emit(0) || !slices.Equal(ran, []string{"w"}) {
+		t.Fatalf("ran %v", ran)
+	}
+}
~~~~

### `content/skygear/dimension/subscribe.go`

~~~~diff
--- a/content/skygear/dimension/subscribe.go
+++ b/content/skygear/dimension/subscribe.go
@@ -21,31 +21,32 @@ func subscribe() {
 
 // The second dimension's reaction to a death and a respawn.
 func subscribeLife() {
-	core.Death.On(20, "dimension.died", func(e core.PlayerEvent) {
+	core.Death.On(core.Normal, "dimension.died", func(e core.PlayerEvent) {
 		Died(e.S, e.P.Tx(), e.P)
 	})
-	core.Respawn.On(10, "dimension.respawned", func(e core.RespawnEvent) {
+	core.Respawn.On(core.Low, "dimension.respawned", func(e core.RespawnEvent) {
 		Respawned(e.S, e.P, e.Pos)
 	})
 }
 
-// The second dimension's reaction to a drop and a hurt.
+// The second dimension's reaction to a drop and a hurt. Whether a hit would
+// kill the hunter is asked last, of the damage the other steps left: Highest.
 func subscribeHurt() {
-	core.ItemDrop.Handle(10, "dimension.hunting", func(e core.ItemEvent) bool {
+	core.ItemDrop.Handle(core.Lowest, "dimension.hunting", func(e core.ItemEvent) bool {
 		if Hunting(e.S, e.P) {
 			e.Cancel()
 			return true
 		}
 		return false
 	})
-	core.Hurt.Handle(10, "dimension.frozen", func(e core.HurtEvent) bool {
+	core.Hurt.Handle(core.Lowest, "dimension.frozen", func(e core.HurtEvent) bool {
 		if Frozen(e.S, e.P) {
 			e.Cancel() // the hunt waits while the hunter chooses an upgrade
 			return true
 		}
 		return false
 	})
-	core.Hurt.On(30, "dimension.fallen", func(e core.HurtEvent) {
+	core.Hurt.On(core.Highest, "dimension.fallen", func(e core.HurtEvent) {
 		if Fallen(e.S, e.Tx, e.P, *e.Damage) {
 			e.Cancel()
 		}
@@ -55,42 +56,50 @@ func subscribeHurt() {
 // The collectors work and the items of the other dimension are retired every
 // second; the watchdog looks at the hunt on every fast tick.
 func subscribeTicks() {
-	core.Second.On(60, "dimension.second", func(e core.Tick) {
+	core.Second.On(core.Normal, "dimension.second", func(e core.Tick) {
 		TickCollectors(e.S, e.Tx)
 		for p := range e.Tx.Players() {
 			RetireItems(p.(*player.Player))
 		}
 	})
-	core.Fast.On(20, "dimension.watchdog", func(e core.Tick) {
+	core.Fast.On(core.Normal, "dimension.watchdog", func(e core.Tick) {
 		Watchdog(e.S, e.Tx)
 	})
 }
 
 // The second dimension reacts to a player joining and leaving.
+//
+// dimension.joined is in the High tier, after the join steps of the other
+// features, as it always ran: a hunter the server stopped under gets back
+// what it carried, and what the earlier join steps put into the hunter's
+// inventory meanwhile is gone with the hunting kit. That keeps today's
+// behaviour; it is one of the listed exceptions (docs/tech/extension.md).
 func subscribePresence() {
-	core.Join.On(80, "dimension.joined", func(e core.Presence) {
+	core.Join.On(core.High, "dimension.joined", func(e core.Presence) {
 		Joined(e.S, e.P)
 		RetireItems(e.P)
 	})
-	core.Leave.On(10, "dimension.leave", func(e core.Presence) {
+	core.Leave.On(core.Normal, "dimension.leave", func(e core.Presence) {
 		Leave(e.S, e.Tx, e.P)
 	})
 }
 
 // A hunter only moves: the use chain refuses it, and the weapons of a hunter
-// strike on their own (gear.go). The menu item is the way out.
+// strike on their own (gear.go). The menu item is the way out. The refusals
+// and the stone's window come before everything else: Lowest. They exclude
+// each other and the menu item's step by their own conditions.
 func subscribeUse() {
-	core.UseOnBlock.Handle(10, "dimension.hunting", func(e core.UseOnBlockEvent) bool {
+	core.UseOnBlock.Handle(core.Lowest, "dimension.hunting", func(e core.UseOnBlockEvent) bool {
 		if Hunting(e.S, e.P) {
 			e.Cancel()
 			return true
 		}
 		return false
 	})
-	core.UseOnBlock.Handle(30, "dimension.stone", func(e core.UseOnBlockEvent) bool {
+	core.UseOnBlock.Handle(core.Lowest, "dimension.stone", func(e core.UseOnBlockEvent) bool {
 		return useStone(e.Actor, e.Held)
 	})
-	core.Use.Handle(30, "dimension.hunting", func(e core.UseEvent) bool {
+	core.Use.Handle(core.Lowest, "dimension.hunting", func(e core.UseEvent) bool {
 		// the menu item in the air is the hunter's way out (screens.menuItem)
 		if Hunting(e.S, e.P) && !sgitem.IsMenuItem(e.Held) {
 			e.Cancel()
@@ -98,17 +107,17 @@ func subscribeUse() {
 		}
 		return false
 	})
-	core.Use.Handle(40, "dimension.stone", func(e core.UseEvent) bool {
+	core.Use.Handle(core.Lowest, "dimension.stone", func(e core.UseEvent) bool {
 		return useStone(e.Actor, e.Held)
 	})
-	core.UseOnEntity.Handle(10, "dimension.hunting", func(e core.EntityEvent) bool {
+	core.UseOnEntity.Handle(core.Lowest, "dimension.hunting", func(e core.EntityEvent) bool {
 		if Hunting(e.S, e.P) {
 			e.Cancel()
 			return true
 		}
 		return false
 	})
-	core.AttackEntity.Handle(10, "dimension.hunter", func(e core.EntityEvent) bool {
+	core.AttackEntity.Handle(core.Lowest, "dimension.hunter", func(e core.EntityEvent) bool {
 		if _, ok := e.Target.(*Monster); ok || Hunting(e.S, e.P) {
 			e.Cancel()
 			return true
~~~~

### `content/skygear/elevator/subscribe.go`

~~~~diff
--- a/content/skygear/elevator/subscribe.go
+++ b/content/skygear/elevator/subscribe.go
@@ -10,14 +10,15 @@ func subscribe() {
 
 // A jump on an elevator rides it up.
 func subscribeJump() {
-	core.Jump.On(20, "elevator.up", func(e core.PlayerEvent) {
+	core.Jump.On(core.Normal, "elevator.up", func(e core.PlayerEvent) {
 		Up(e.S, e.P, e.Tx)
 	})
 }
 
-// Sneaking on an elevator rides it down.
+// Sneaking on an elevator rides it down. A ride cancels the event for the
+// later tiers: the sneak was used.
 func subscribeSneak() {
-	core.Sneak.Handle(30, "elevator.down", func(e core.PlayerEvent) bool {
+	core.Sneak.Handle(core.Normal, "elevator.down", func(e core.PlayerEvent) bool {
 		return Down(e.S, e.P, e.Tx)
 	})
 }
~~~~

### `content/skygear/expand/expand_test.go`

~~~~diff
--- a/content/skygear/expand/expand_test.go
+++ b/content/skygear/expand/expand_test.go
@@ -136,8 +136,8 @@ func TestExpansionMap(t *testing.T) {
 }
 
 // TestLeaveEndsExpansionMode: a player who leaves the session in expansion
-// mode gets back the game mode it had before, through the kernel's Leave
-// with the kernel's own steps (core.parts drops the player's entries).
+// mode gets back the game mode it had before, through the kernel's Leave,
+// which drops the player's entries when the subscribers have run.
 func TestLeaveEndsExpansionMode(t *testing.T) {
 	s := coretest.Session(t)
 	coretest.LoadAround(s)
~~~~

### `content/skygear/expand/setup_test.go`

~~~~diff
--- a/content/skygear/expand/setup_test.go
+++ b/content/skygear/expand/setup_test.go
@@ -8,6 +8,5 @@ import (
 
 // This package's test binary links no feature list, so no feature is
 // installed in it. The tests that reach a feature through the kernel's hooks
-// have it set up here, once, with the kernel's own steps (core.parts drops a
-// leaving player's entries).
+// have it set up here, once, with the kernel's own feature.
 func init() { coretest.Setup(core.KernelFeature, expand.Feature) }
~~~~

### `content/skygear/expand/subscribe.go`

~~~~diff
--- a/content/skygear/expand/subscribe.go
+++ b/content/skygear/expand/subscribe.go
@@ -24,14 +24,14 @@ func subscribe() {
 // A player in island expansion mode stays in place and sees the expansion
 // screen again when sneaking.
 func subscribeHold() {
-	core.Move.Handle(20, "expand.hold", func(e core.MoveEvent) bool {
+	core.Move.Handle(core.Lowest, "expand.hold", func(e core.MoveEvent) bool {
 		if _, expanding := Of(e.S, e.P); expanding {
 			e.Cancel()
 			return true
 		}
 		return false
 	})
-	core.Sneak.Handle(10, "expand.show", func(e core.PlayerEvent) bool {
+	core.Sneak.Handle(core.Lowest, "expand.show", func(e core.PlayerEvent) bool {
 		if _, ok := Of(e.S, e.P); ok {
 			Show(e.S, e.P)
 			return true
@@ -40,25 +40,27 @@ func subscribeHold() {
 	})
 }
 
-// Island expansion mode ends when the player leaves. It runs before
-// core.parts drops the player's entries: Stop finds the expansion there and
-// restores the saved game mode, the movement and the camera.
+// Island expansion mode ends when the player leaves. The player's entries are
+// dropped when every Leave subscriber has run (core.Lifecycle.Leave), so Stop
+// finds the expansion there and restores the saved game mode, the movement
+// and the camera.
 func subscribeLeave() {
-	core.Leave.On(15, "expand.stop", func(e core.Presence) {
+	core.Leave.On(core.Normal, "expand.stop", func(e core.Presence) {
 		Stop(e.S, e.P, e.Tx)
 	})
 }
 
 // No building or clicking in island expansion mode. A click on the map makes
-// the server resend the real blocks, so the map chunk is shown again.
+// the server resend the real blocks, so the map chunk is shown again. The mode
+// refuses before anything else looks at the event: Lowest.
 func subscribeGuard() {
-	core.StartBreak.Handle(10, "expand.guard", func(e core.StartBreakEvent) bool {
+	core.StartBreak.Handle(core.Lowest, "expand.guard", func(e core.StartBreakEvent) bool {
 		return guard(e.S, e.Ctx, e.Pos)
 	})
-	core.BlockBreak.Handle(10, "expand.guard", func(e core.BlockBreakEvent) bool {
+	core.BlockBreak.Handle(core.Lowest, "expand.guard", func(e core.BlockBreakEvent) bool {
 		return guard(e.S, e.Ctx, e.Pos)
 	})
-	core.BlockPlace.Handle(10, "expand.guard", func(e core.BlockPlaceEvent) bool {
+	core.BlockPlace.Handle(core.Lowest, "expand.guard", func(e core.BlockPlaceEvent) bool {
 		return guard(e.S, e.Ctx, e.Pos)
 	})
 }
@@ -76,15 +78,16 @@ func guard(s *core.Session, ctx *player.Context, pos cube.Pos) bool {
 
 // The block clicked and the block next to its face are guarded alike. An
 // item that opens its screen wherever it is used (the menu item, the dimension
-// stone) keeps its click: its own step takes it.
+// stone) keeps its click: its own step takes it. The side is refused last,
+// when no step took the click: Highest.
 func subscribeSide() {
-	core.UseOnBlock.Handle(40, "expand.guard", func(e core.UseOnBlockEvent) bool {
+	core.UseOnBlock.Handle(core.Lowest, "expand.guard", func(e core.UseOnBlockEvent) bool {
 		if sgitem.OpensScreen(e.Held) {
 			return false
 		}
 		return guard(e.S, e.Ctx, e.Pos)
 	})
-	core.UseOnBlock.Handle(160, "expand.guardSide", func(e core.UseOnBlockEvent) bool {
+	core.UseOnBlock.Handle(core.Highest, "expand.guardSide", func(e core.UseOnBlockEvent) bool {
 		return guard(e.S, e.Ctx, e.Pos.Side(e.Face))
 	})
 }
~~~~

### `content/skygear/fairy/subscribe.go`

~~~~diff
--- a/content/skygear/fairy/subscribe.go
+++ b/content/skygear/fairy/subscribe.go
@@ -11,30 +11,35 @@ func subscribe() {
 
 // The fairy acts every second.
 func subscribeSecond() {
-	core.Second.On(40, "fairy.tick", func(e core.Tick) {
+	core.Second.On(core.Normal, "fairy.tick", func(e core.Tick) {
 		Tick(e.S, e.Tx)
 	})
 }
 
 // A joining player gets the fairy dust they are owed.
 func subscribeJoin() {
-	core.Join.On(20, "fairy.backfillDust", func(e core.Presence) {
+	core.Join.On(core.Normal, "fairy.backfillDust", func(e core.Presence) {
 		BackfillDust(e.S, e.P, e.Tx)
 	})
 }
 
 // The fairy's time is told by the item it is used with; a visiting fairy
 // gives its gift to a player who touches it, whichever way.
+//
+// conduit.plate is in the same tier, is registered before fairy.time and
+// takes a click on a connector plate first, hourglass in hand or not: that
+// order is one of the exceptions the tier order guard keeps (the root's
+// orderKept).
 func subscribeTouch() {
-	core.UseOnBlock.Handle(140, "fairy.time", func(e core.UseOnBlockEvent) bool {
+	core.UseOnBlock.Handle(core.Normal, "fairy.time", func(e core.UseOnBlockEvent) bool {
 		if UseFairyTime(e.S, e.P, e.Tx, e.Pos, e.Held) {
 			e.Cancel()
 			return true
 		}
 		return false
 	})
-	core.UseOnEntity.On(20, "fairy.touch", touchFairy)
-	core.AttackEntity.On(20, "fairy.touch", touchFairy)
+	core.UseOnEntity.On(core.Normal, "fairy.touch", touchFairy)
+	core.AttackEntity.On(core.Normal, "fairy.touch", touchFairy)
 }
 
 // touchFairy takes the gift of the fairy e.Target, if it is one.
~~~~

### `content/skygear/farm/subscribe.go`

~~~~diff
--- a/content/skygear/farm/subscribe.go
+++ b/content/skygear/farm/subscribe.go
@@ -35,30 +35,31 @@ func registerComposterIO() {
 	})
 }
 
-// Sneaking grows saplings nearby by chance.
+// Sneaking grows saplings nearby by chance, unless another step used the
+// sneak (an elevator ride cancels the event): work of low priority, High.
 func subscribeSneak() {
-	core.Sneak.On(40, "farm.crouchGrow", func(e core.PlayerEvent) {
+	core.Sneak.On(core.High, "farm.crouchGrow", func(e core.PlayerEvent) {
 		CrouchGrow(e.S, e.P, e.Tx)
 	})
 }
 
 // The crops, barrels, composters and plants grow every second.
 func subscribeSecond() {
-	core.Second.On(10, "farm.growCrops", func(e core.Tick) {
+	core.Second.On(core.Normal, "farm.growCrops", func(e core.Tick) {
 		GrowCrops(e.S, e.Tx)
 	})
-	core.Second.On(30, "farm.barrels", func(e core.Tick) {
+	core.Second.On(core.Normal, "farm.barrels", func(e core.Tick) {
 		TickBarrels(e.S, e.Tx)
 		TickComposters(e.S, e.Tx)
 	})
-	core.Second.On(90, "farm.plants", func(e core.Tick) {
+	core.Second.On(core.Normal, "farm.plants", func(e core.Tick) {
 		TickPlants(e.S, e.Tx)
 	})
 }
 
 // The crops are loaded before the world transaction of the session start.
 func subscribeLoad() {
-	core.Load.On(10, "farm.loadCrops", func(e core.Tick) {
+	core.Load.On(core.Normal, "farm.loadCrops", func(e core.Tick) {
 		LoadCrops(e.S)
 	})
 }
@@ -67,7 +68,7 @@ func subscribeLoad() {
 // silkworm (silkworm leaves drop their string themselves); a broken composter
 // and a broken barrel give back what they hold. A placed crop is planted.
 func subscribeBlocks() {
-	core.BlockBreak.On(50, "farm.break", func(e core.BlockBreakEvent) {
+	core.BlockBreak.On(core.Normal, "farm.break", func(e core.BlockBreakEvent) {
 		if c, ok := sgitem.CrookOf(e.Held); ok {
 			if l, ok := e.Block.(block.Leaves); ok {
 				*e.Drops = CrookLeafDrops(l, c, e.S.Rand.Float64)
@@ -80,7 +81,7 @@ func subscribeBlocks() {
 			BreakBarrel(e.S, e.Pos, e.Drops)
 		}
 	})
-	core.BlockPlace.On(50, "farm.plant", func(e core.BlockPlaceEvent) {
+	core.BlockPlace.On(core.Normal, "farm.plant", func(e core.BlockPlaceEvent) {
 		if kind, _, ok := CropOf(e.Block); ok {
 			achievement.Count(e.P, "crop_plant", 1)
 			PlantCrop(e.S, e.Pos, kind)
@@ -90,16 +91,17 @@ func subscribeBlocks() {
 
 // Using a composter or a crop with a free hand acts on it; sneaking uses the
 // item as usual (placing blocks, bone meal). A crop shows its time or
-// harvests, and places nothing.
+// harvests, and places nothing. What the block does with a click comes after
+// what the held item does with it (Normal: a stick, the fairy's items): High.
 func subscribeUse() {
-	core.UseOnBlock.Handle(120, "farm.composter", func(e core.UseOnBlockEvent) bool {
+	core.UseOnBlock.Handle(core.High, "farm.composter", func(e core.UseOnBlockEvent) bool {
 		if !e.P.Sneaking() && UseComposter(e.S, e.P, e.Tx, e.Pos, e.Held) {
 			e.Cancel()
 			return true
 		}
 		return false
 	})
-	core.UseOnBlock.Handle(150, "farm.crop", func(e core.UseOnBlockEvent) bool {
+	core.UseOnBlock.Handle(core.High, "farm.crop", func(e core.UseOnBlockEvent) bool {
 		if !e.P.Sneaking() && UseCrop(e.S, e.P, e.Tx, e.Pos) {
 			e.Cancel()
 			return true
~~~~

### `content/skygear/handwork/subscribe.go`

~~~~diff
--- a/content/skygear/handwork/subscribe.go
+++ b/content/skygear/handwork/subscribe.go
@@ -9,15 +9,21 @@ func subscribe() {
 
 // A stick opens or fills a chest and strips; a bare hand strips fibre and
 // tills seeds.
+//
+// handwork.stick also counts a stick used on any other block, when the click
+// reaches it. storage.sneakUse and conduit.plate are in the same tier, are
+// registered before it and take such a click first, so it counts nothing:
+// that order is one of the exceptions the tier order guard keeps (the root's
+// orderKept).
 func subscribeUse() {
-	core.UseOnBlock.Handle(110, "handwork.stick", func(e core.UseOnBlockEvent) bool {
+	core.UseOnBlock.Handle(core.Normal, "handwork.stick", func(e core.UseOnBlockEvent) bool {
 		if UseStickOnChest(e.S, e.P, e.Tx, e.Pos, e.Held) || UseStick(e.S, e.P, e.Tx, e.Pos, e.Face, e.Held) {
 			e.Cancel()
 			return true
 		}
 		return false
 	})
-	core.UseOnBlock.On(130, "handwork.gather", func(e core.UseOnBlockEvent) {
+	core.UseOnBlock.On(core.Normal, "handwork.gather", func(e core.UseOnBlockEvent) {
 		StripFiber(e.S, e.P, e.Tx, e.Pos, e.Face, e.Held)
 		TillSeeds(e.S, e.P, e.Tx, e.Pos, e.Face, e.Held)
 	})
~~~~

### `content/skygear/island/subscribe.go`

~~~~diff
--- a/content/skygear/island/subscribe.go
+++ b/content/skygear/island/subscribe.go
@@ -16,9 +16,10 @@ func subscribe() {
 	subscribeSide()
 }
 
-// A player that falls below the island returns from the void.
+// A player that falls below the island returns from the void: the last step
+// that may cancel the move, Highest.
 func subscribeVoid() {
-	core.Move.On(40, "island.void", func(e core.MoveEvent) {
+	core.Move.On(core.Highest, "island.void", func(e core.MoveEvent) {
 		if e.Pos.Y() < VoidY {
 			e.Cancel()
 			ReturnFromVoid(e.S, e.P, e.Tx)
@@ -31,18 +32,18 @@ func subscribeVoid() {
 // loaded; Dragonfly closes an item entity below the world's lowest height,
 // while a picked-up or expired item despawns above it.
 func subscribeLost() {
-	core.Join.On(50, "island.purgeLost", func(e core.Presence) {
+	core.Join.On(core.Normal, "island.purgeLost", func(e core.Presence) {
 		PurgeLost(e.S)
 	})
-	core.NewDay.On(10, "island.purgeLost", func(e core.DayEvent) {
+	core.NewDay.On(core.Normal, "island.purgeLost", func(e core.DayEvent) {
 		PurgeLost(e.S)
 	})
-	core.Start.On(10, "island.keepLoaded", func(e core.Tick) {
+	core.Start.On(core.Normal, "island.keepLoaded", func(e core.Tick) {
 		for _, c := range StateOf(e.S).Chunks {
 			KeepLoaded(e.S, e.Tx, c)
 		}
 	})
-	core.EntityDespawn.On(10, "island.lost", func(e core.DespawnEvent) {
+	core.EntityDespawn.On(core.Normal, "island.lost", func(e core.DespawnEvent) {
 		ent, ok := e.Entity.(*entity.Ent)
 		if !ok || e.Entity.Position().Y() >= float64(e.Tx.Range()[0]) {
 			return
@@ -53,15 +54,17 @@ func subscribeLost() {
 	})
 }
 
-// Building is only allowed on the island's own ground.
+// Building is only allowed on the island's own ground. The guard refuses after
+// the modes that hold a player (Lowest) and before any step works on the
+// block: Low.
 func subscribeGuard() {
-	core.StartBreak.Handle(20, "island.guard", func(e core.StartBreakEvent) bool {
+	core.StartBreak.Handle(core.Low, "island.guard", func(e core.StartBreakEvent) bool {
 		return guard(e.S, e.Ctx, e.Pos)
 	})
-	core.BlockBreak.Handle(20, "island.guard", func(e core.BlockBreakEvent) bool {
+	core.BlockBreak.Handle(core.Low, "island.guard", func(e core.BlockBreakEvent) bool {
 		return guard(e.S, e.Ctx, e.Pos)
 	})
-	core.BlockPlace.Handle(20, "island.guard", func(e core.BlockPlaceEvent) bool {
+	core.BlockPlace.Handle(core.Low, "island.guard", func(e core.BlockPlaceEvent) bool {
 		return guard(e.S, e.Ctx, e.Pos)
 	})
 }
@@ -80,12 +83,13 @@ func guard(s *core.Session, ctx *player.Context, pos cube.Pos) bool {
 	return true
 }
 
-// The block clicked and the block next to its face are guarded alike.
+// The block clicked and the block next to its face are guarded alike. The side
+// is refused last, when no step took the click: Highest.
 func subscribeSide() {
-	core.UseOnBlock.Handle(50, "island.guard", func(e core.UseOnBlockEvent) bool {
+	core.UseOnBlock.Handle(core.Low, "island.guard", func(e core.UseOnBlockEvent) bool {
 		return guard(e.S, e.Ctx, e.Pos)
 	})
-	core.UseOnBlock.Handle(170, "island.guardSide", func(e core.UseOnBlockEvent) bool {
+	core.UseOnBlock.Handle(core.Highest, "island.guardSide", func(e core.UseOnBlockEvent) bool {
 		return guard(e.S, e.Ctx, e.Pos.Side(e.Face))
 	})
 }
~~~~

### `content/skygear/look/subscribe.go`

~~~~diff
--- a/content/skygear/look/subscribe.go
+++ b/content/skygear/look/subscribe.go
@@ -12,23 +12,27 @@ func subscribe() {
 	subscribeLeave()
 }
 
-// A movement updates where the player looks.
+// A movement updates where the player looks. It runs for every move, also
+// one that was cancelled for the other subscribers (island expansion mode
+// holds the player in place): Update then clears the hidden panel.
 func subscribeMove() {
-	core.Move.On(10, "look.update", func(e core.MoveEvent) {
+	core.Move.Always(core.Monitor, "look.update", func(e core.MoveEvent, _ bool) {
 		Update(e.S, e.P, e.Tx, e.Pos, e.Rot, false)
 	})
 }
 
-// Every player's panel updates on every fast tick.
+// Every player's panel updates on every fast tick, when the tick's work is
+// done: it shows the result (Monitor).
 func subscribeFast() {
-	core.Fast.On(30, "look.updateAll", func(e core.Tick) {
+	core.Fast.On(core.Monitor, "look.updateAll", func(e core.Tick) {
 		UpdateAll(e.S, e.Tx)
 	})
 }
 
-// What the player looked at is cleared when the player leaves.
+// What the player looked at is cleared when the player leaves, after the
+// features' own leave steps: the last clean-up, Highest.
 func subscribeLeave() {
-	core.Leave.On(40, "look.clear", func(e core.Presence) {
+	core.Leave.On(core.Highest, "look.clear", func(e core.Presence) {
 		waila.Clear(e.P)
 	})
 }
~~~~

### `content/skygear/machine/subscribe.go`

~~~~diff
--- a/content/skygear/machine/subscribe.go
+++ b/content/skygear/machine/subscribe.go
@@ -12,19 +12,19 @@ func subscribe() {
 
 // The monitor reads its machines every second.
 func subscribeMonitor() {
-	core.Second.On(80, "machine.monitor", func(e core.Tick) {
+	core.Second.On(core.Normal, "machine.monitor", func(e core.Tick) {
 		TickMonitor(e.S, e.Tx)
 	})
 }
 
 // A machine breaking or placed is taken down or set up.
 func subscribeBlocks() {
-	core.BlockBreak.On(80, "machine.break", func(e core.BlockBreakEvent) {
+	core.BlockBreak.On(core.Normal, "machine.break", func(e core.BlockBreakEvent) {
 		if b, ok := e.Block.(Machine); ok {
 			Break(e.S, e.Tx, e.Pos, b)
 		}
 	})
-	core.BlockPlace.On(70, "machine.place", func(e core.BlockPlaceEvent) {
+	core.BlockPlace.On(core.Normal, "machine.place", func(e core.BlockPlaceEvent) {
 		if b, ok := e.Block.(Machine); ok {
 			Place(e.S, e.P, e.Tx, e.Pos, b.Kind)
 		}
@@ -33,7 +33,7 @@ func subscribeBlocks() {
 
 // A net-shaped machine or a miner in hand remembers the face it is placed on.
 func subscribeFace() {
-	core.UseOnBlock.On(60, "machine.noteFace", func(e core.UseOnBlockEvent) {
+	core.UseOnBlock.On(core.Normal, "machine.noteFace", func(e core.UseOnBlockEvent) {
 		if m, ok := e.Held.Item().(Machine); ok && (NetShaped(m.Kind) || m.Kind == KindMiner) {
 			NotePlaceFace(e.S, e.P, e.Face)
 		}
~~~~

### `content/skygear/monitor_test.go` (new file)

~~~~go
package skygear

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/presentkim/skygear/content/skygear/core"
)

// TestMonitorSubscribersDoNotShape: a Monitor subscriber handles the result
// of an event and neither shapes nor cancels it. The kernel refuses Handle in
// the Monitor tier (core.TestHandleAtMonitorPanics); this test reads the
// function literal of every subscriber registered with core.Monitor in a
// feature package and fails when it calls Cancel or assigns through one of the
// event's pointers (*e.Drops, *e.XP, *e.Damage, *e.Pos, *e.Pick). It sees the
// literal only, not what the literal calls.
func TestMonitorSubscribersDoNotShape(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("*", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) != 3 {
				return true
			}
			if tier, ok := call.Args[0].(*ast.SelectorExpr); !ok || tier.Sel.Name != "Monitor" {
				return true
			}
			name := strings.Trim(call.Args[1].(*ast.BasicLit).Value, `"`)
			found++
			lit, ok := call.Args[2].(*ast.FuncLit)
			if !ok {
				t.Errorf("%s: %s: a Monitor subscriber is a function literal, so this test can read it", fset.Position(call.Pos()), name)
				return true
			}
			ast.Inspect(lit.Body, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.CallExpr:
					if sel, ok := n.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Cancel" {
						t.Errorf("%s: %s cancels the event; a Monitor subscriber does not (use an earlier tier)", fset.Position(n.Pos()), name)
					}
				case *ast.AssignStmt:
					for _, lhs := range n.Lhs {
						if star, ok := lhs.(*ast.StarExpr); ok {
							if sel, ok := star.X.(*ast.SelectorExpr); ok {
								t.Errorf("%s: %s writes *%s.%s; a Monitor subscriber does not shape the event (use an earlier tier)", fset.Position(n.Pos()), name, exprName(sel.X), sel.Sel.Name)
							}
						}
					}
				}
				return true
			})
			return true
		})
	}
	want := 0
	for _, s := range core.Subscriptions() {
		if s.Tier == core.Monitor {
			want++
		}
	}
	if found != want {
		t.Errorf("read %d Monitor subscribers in the feature packages, the kernel lists %d", found, want)
	}
}

func exprName(e ast.Expr) string {
	if id, ok := e.(*ast.Ident); ok {
		return id.Name
	}
	return "?"
}
~~~~

### `content/skygear/netstore/subscribe.go`

~~~~diff
--- a/content/skygear/netstore/subscribe.go
+++ b/content/skygear/netstore/subscribe.go
@@ -16,19 +16,20 @@ func subscribe() {
 // Disk contents changed since the controller last flushed them are part of the
 // save.
 func subscribeFlush() {
-	core.Autosave.On(10, "netstore.flush", func(e core.Tick) {
+	core.Autosave.On(core.Normal, "netstore.flush", func(e core.Tick) {
 		Disks(e.S).Flush()
 	})
-	core.Close.On(30, "netstore.flush", func(e core.Tick) {
+	core.Close.On(core.Normal, "netstore.flush", func(e core.Tick) {
 		Disks(e.S).Flush()
 	})
 }
 
 // A network node breaking drops its network once the block is gone; a placed
 // node forgets the setting a node broken by an explosion left and drops the
-// network after machine.place's deferred set.
+// network after machine.place's deferred set: netstore.place is in the High
+// tier, after the machine is set up (Normal).
 func subscribeBlocks() {
-	core.BlockBreak.On(70, "netstore.break", func(e core.BlockBreakEvent) {
+	core.BlockBreak.On(core.Normal, "netstore.break", func(e core.BlockBreakEvent) {
 		if b, ok := e.Block.(machine.Machine); ok && IsNode(b.Kind) {
 			// The block goes after this handler: drop the network once it is gone.
 			s, pos := e.S, e.Pos
@@ -39,7 +40,7 @@ func subscribeBlocks() {
 			})
 		}
 	})
-	core.BlockPlace.On(80, "netstore.place", func(e core.BlockPlaceEvent) {
+	core.BlockPlace.On(core.High, "netstore.place", func(e core.BlockPlaceEvent) {
 		if b, ok := e.Block.(machine.Machine); ok && IsNode(b.Kind) {
 			Forget(e.S, e.Pos) // a node broken by an explosion left its setting
 			// machine.Place sets the block in a deferred call; this one runs after it.
~~~~

### `content/skygear/network/subscribe.go`

~~~~diff
--- a/content/skygear/network/subscribe.go
+++ b/content/skygear/network/subscribe.go
@@ -20,7 +20,7 @@ func subscribe() {
 
 // The conduits move on every fast tick.
 func subscribeFast() {
-	core.Fast.On(10, "network.step", func(e core.Tick) {
+	core.Fast.On(core.Normal, "network.step", func(e core.Tick) {
 		// Energy moves every FastInterval (four ticks) over the conduits.
 		Energy(e.S).Step(e.Tx, 4)
 		// Items move every itemnet.StepTicks ticks and fluids every
~~~~

### `content/skygear/pin_blocks_test.go`

~~~~diff
--- a/content/skygear/pin_blocks_test.go
+++ b/content/skygear/pin_blocks_test.go
@@ -273,11 +273,12 @@ func TestPinBlocks(t *testing.T) {
 	})
 
 	// The shapes of the rows: a scenario that stops nowhere equals its hook's
-	// row of the order file; the guards stop where they should.
+	// lines of order.golden; the guards stop where they should. A run that
+	// reorders the subscribers (TestTierOrder) has other traces.
 	rows := map[string]string{
 		"StartBreak/own":   "expand.guard island.guard tinker.autoTool",
-		"BlockBreak/stone": "expand.guard island.guard achievement.break tinker.break farm.break storage.break netstore.break machine.break conduit.break",
-		"BlockPlace/dirt":  "expand.guard island.guard achievement.place autofill.place farm.plant storage.place machine.place netstore.place conduit.place progress.dragonEgg",
+		"BlockBreak/stone": "expand.guard island.guard tinker.break machine.break conduit.break storage.break farm.break netstore.break achievement.break",
+		"BlockPlace/dirt":  "expand.guard island.guard machine.place storage.place farm.plant autofill.place netstore.place achievement.place progress.dragonEgg conduit.place",
 	}
 	for name, row := range rows {
 		if got := lines[name]; hookOrder == "" && !strings.HasPrefix(got, row+" | ") {
~~~~

### `content/skygear/pin_order_test.go`

~~~~diff
--- a/content/skygear/pin_order_test.go
+++ b/content/skygear/pin_order_test.go
@@ -15,29 +15,35 @@ import (
 	"testing"
 
 	"github.com/presentkim/skygear/content/skygear/core"
+	"github.com/presentkim/skygear/content/skygear/core/coretest"
 )
 
 // TestPinOrder compares the kernel's subscriptions with
-// testdata/pin/order.golden: one line "<Hook> <priority> <name>" per
-// subscriber, hooks in declaration order, each hook's subscribers in the
-// order they run. It names no feature, so it does not change when a step moves
-// to its feature.
+// testdata/pin/order.golden: one line "<Hook> <TIER> <name>" per subscriber,
+// with " always" after the name of a subscriber that also receives a
+// cancelled event (Hook.Always), hooks in declaration order, each hook's
+// subscribers in the order they run. It names no feature, so it does not
+// change when a step moves to its feature.
 //
-// The golden was recorded once from the bridged code. Re-record it
-// (SKYGEAR_GOLDEN=update) only when a step is added, removed or reordered on
-// purpose.
+// Re-record it (SKYGEAR_GOLDEN=update) only when a step is added, removed or
+// moved to another tier on purpose. In a run that reorders the subscribers
+// inside their tiers (TestTierOrder) the lines of one tier of one hook are
+// compared as a set.
 func TestPinOrder(t *testing.T) {
-	var b strings.Builder
+	var lines []string
 	for _, s := range core.Subscriptions() {
-		fmt.Fprintf(&b, "%s %d %s\n", s.Hook, s.Prio, s.Name)
-	}
-	got := b.String()
-	if hookOrder != "" {
-		return // a reordered run (TestTierOrder) lists another order on purpose
+		ln := fmt.Sprintf("%s %s %s", s.Hook, s.Tier, s.Name)
+		if s.Always {
+			ln += " always"
+		}
+		lines = append(lines, ln)
 	}
 	const file = "testdata/pin/order.golden"
 	if os.Getenv("SKYGEAR_GOLDEN") == "update" {
-		if err := os.WriteFile(file, []byte(got), 0o644); err != nil {
+		if hookOrder != "" {
+			t.Fatal("record order.golden in the order of registration, without " + coretest.HookOrderEnv)
+		}
+		if err := os.WriteFile(file, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
 			t.Fatal(err)
 		}
 		return
@@ -46,20 +52,39 @@ func TestPinOrder(t *testing.T) {
 	if err != nil {
 		t.Fatal(err)
 	}
-	if want := string(raw); got != want {
-		wl, gl := strings.Split(want, "\n"), strings.Split(got, "\n")
-		for i := 0; i < len(wl) || i < len(gl); i++ {
-			var w, g string
-			if i < len(wl) {
-				w = wl[i]
-			}
-			if i < len(gl) {
-				g = gl[i]
-			}
-			if w != g {
-				t.Fatalf("%s line %d:\nwant: %q\ngot:  %q", file, i+1, w, g)
-			}
+	wl := strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n")
+	if hookOrder != "" {
+		// The tiers stay in order; inside "<Hook> <TIER>" the order is free.
+		sortTiers(wl)
+		sortTiers(lines)
+	}
+	for i := 0; i < len(wl) || i < len(lines); i++ {
+		var w, g string
+		if i < len(wl) {
+			w = wl[i]
+		}
+		if i < len(lines) {
+			g = lines[i]
+		}
+		if w != g {
+			t.Fatalf("%s line %d:\nwant: %q\ngot:  %q", file, i+1, w, g)
+		}
+	}
+}
+
+// sortTiers sorts each run of lines that share "<Hook> <TIER>".
+func sortTiers(lines []string) {
+	tier := func(ln string) string {
+		f := strings.SplitN(ln, " ", 3)
+		return f[0] + " " + f[1]
+	}
+	for from := 0; from < len(lines); {
+		to := from
+		for to < len(lines) && tier(lines[to]) == tier(lines[from]) {
+			to++
 		}
+		slices.Sort(lines[from:to])
+		from = to
 	}
 }
 
@@ -100,7 +125,7 @@ func rootSources(t *testing.T) map[string]*ast.File {
 }
 
 // TestRootSubscribesToNothing: no non-test file of the root package calls a
-// hook's On or Handle, calls core.Mark, declares a Handle* method or is named
+// hook's On, Handle or Always, calls core.Mark, declares a Handle* method or is named
 // legacy_*.go. A step belongs in the Setup of the feature that owns it.
 func TestRootSubscribesToNothing(t *testing.T) {
 	const hint = "a step belongs in the Setup of its feature package (content/skygear/<feature>/feature.go); the root package only lists features and holds platform glue"
@@ -122,8 +147,8 @@ func TestRootSubscribesToNothing(t *testing.T) {
 				if id, ok := sel.X.(*ast.Ident); ok && id.Name == "core" && sel.Sel.Name == "Mark" {
 					t.Errorf("%s: core.Mark call: %s", name, hint)
 				}
-				if (sel.Sel.Name == "On" || sel.Sel.Name == "Handle") && len(n.Args) == 3 {
-					t.Errorf("%s: %s(prio, name, f) subscribes to a hook: %s", name, sel.Sel.Name, hint)
+				if (sel.Sel.Name == "On" || sel.Sel.Name == "Handle" || sel.Sel.Name == "Always") && len(n.Args) == 3 {
+					t.Errorf("%s: %s(tier, name, f) subscribes to a hook: %s", name, sel.Sel.Name, hint)
 				}
 			}
 			return true
~~~~

### `content/skygear/pin_use_test.go`

~~~~diff
--- a/content/skygear/pin_use_test.go
+++ b/content/skygear/pin_use_test.go
@@ -696,8 +696,8 @@ func TestPinUse(t *testing.T) {
 	pinCheckReached(t, names, lines)
 
 	rows := map[string]string{
-		"UseOnBlock/plain": "dimension.hunting screens.menuItem dimension.stone expand.guard island.guard machine.noteFace copier.use wand.use storage.sneakUse conduit.plate handwork.stick farm.composter handwork.gather fairy.time farm.crop expand.guardSide island.guardSide",
-		"Use/plain":        "autofill.use screens.menuItem dimension.hunting dimension.stone copier.clear wand.options tinker.veinToggle",
+		"UseOnBlock/plain": "dimension.hunting dimension.stone expand.guard screens.menuItem island.guard copier.use wand.use machine.noteFace conduit.plate storage.sneakUse fairy.time handwork.stick handwork.gather farm.composter farm.crop island.guardSide expand.guardSide",
+		"Use/plain":        "dimension.hunting dimension.stone screens.menuItem tinker.veinToggle copier.clear wand.options autofill.use",
 	}
 	for name, row := range rows {
 		if got := lines[name]; hookOrder == "" && !strings.HasPrefix(got, row+" | ") {
@@ -706,12 +706,12 @@ func TestPinUse(t *testing.T) {
 	}
 	for name, want := range map[string]string{
 		"UseOnBlock/hunting":      "dimension.hunting | cancelled=true",
-		"UseOnBlock/menuItem":     "dimension.hunting screens.menuItem | cancelled=true",
-		"UseOnBlock/stone":        "dimension.hunting screens.menuItem dimension.stone | cancelled=true",
-		"UseOnBlock/expanding":    "dimension.hunting screens.menuItem dimension.stone expand.guard | cancelled=true",
-		"UseOnBlock/outside":      "dimension.hunting screens.menuItem dimension.stone expand.guard island.guard | cancelled=true",
-		"UseOnBlock/cropSneaking": "farm.crop expand.guardSide island.guardSide",
-		"Use/menuItemHunting":     "autofill.use screens.menuItem | cancelled=true",
+		"UseOnBlock/menuItem":     "dimension.hunting dimension.stone expand.guard screens.menuItem | cancelled=true",
+		"UseOnBlock/stone":        "dimension.hunting dimension.stone | cancelled=true",
+		"UseOnBlock/expanding":    "dimension.hunting dimension.stone expand.guard | cancelled=true",
+		"UseOnBlock/outside":      "dimension.hunting dimension.stone expand.guard screens.menuItem island.guard | cancelled=true",
+		"UseOnBlock/cropSneaking": "farm.crop island.guardSide expand.guardSide",
+		"Use/menuItemHunting":     "dimension.hunting dimension.stone screens.menuItem | cancelled=true",
 		"AttackEntity/monster":    "dimension.hunter | cancelled=true",
 	} {
 		if got := lines[name]; hookOrder == "" && !strings.Contains(got, want) {
~~~~

### `content/skygear/progress/subscribe.go`

~~~~diff
--- a/content/skygear/progress/subscribe.go
+++ b/content/skygear/progress/subscribe.go
@@ -18,33 +18,38 @@ func subscribe() {
 	subscribeDragon()
 }
 
-// The advancements and the collection read the items players hold.
+// The advancements and the collection read the items players hold. The items
+// are read before the second's work changes them (Low); the collection is
+// granted from that reading when the work is done (a record: Monitor).
 func subscribeHeld() {
-	core.Second.On(20, "progress.checkHeld", func(e core.Tick) {
+	core.Second.On(core.Low, "progress.checkHeld", func(e core.Tick) {
 		names := advancement.InventoryNames(e.Tx)
 		*held.Of(e.S) = names
 		CheckHeld(e.S, e.Tx, names)
 	})
-	core.Second.On(120, "progress.collection", func(e core.Tick) {
+	core.Second.On(core.Monitor, "progress.collection", func(e core.Tick) {
 		achievement.GrantCollection(*held.Of(e.S))
 	})
 }
 
 // A joining player is sent the recipes they know; the recipe filter, which
-// belongs to this island, is cleared on leave.
+// belongs to this island, is cleared on leave, after the features' own leave
+// steps, which may send recipes again (a hunt that ends may advance): the
+// last clean-up, Highest.
 func subscribeRecipes() {
-	core.Join.On(60, "progress.sendRecipes", func(e core.Presence) {
+	core.Join.On(core.Normal, "progress.sendRecipes", func(e core.Presence) {
 		SendRecipesTo(e.S, e.P)
 	})
-	core.Leave.On(70, "progress.recipes", func(e core.Presence) {
+	core.Leave.On(core.Highest, "progress.recipes", func(e core.Presence) {
 		// Locked recipes belong to this island (recipes.go).
 		e.P.SetRecipeFilter(nil)
 	})
 }
 
-// Placing the dragon egg finishes the game.
+// Placing the dragon egg finishes the game: an advancement of the result,
+// Monitor.
 func subscribeDragon() {
-	core.BlockPlace.On(100, "progress.dragonEgg", func(e core.BlockPlaceEvent) {
+	core.BlockPlace.On(core.Monitor, "progress.dragonEgg", func(e core.BlockPlaceEvent) {
 		if _, ok := e.Block.(block.DragonEgg); ok {
 			CompleteDragon(e.S, e.P, e.Tx, e.Pos)
 		}
~~~~

### `content/skygear/screens/subscribe.go`

~~~~diff
--- a/content/skygear/screens/subscribe.go
+++ b/content/skygear/screens/subscribe.go
@@ -14,36 +14,40 @@ func subscribe() {
 	subscribeUse()
 }
 
-// The scoreboards update every second.
+// The scoreboards update every second, when the second's work is done: they
+// show the result (Monitor).
 func subscribeBoards() {
-	core.Second.On(100, "screens.boards", func(e core.Tick) {
+	core.Second.On(core.Monitor, "screens.boards", func(e core.Tick) {
 		UpdateBoards(e.S, e.Tx)
 	})
 }
 
-// The menu item and the scoreboard are given on join; the scoreboard goes on
-// leave.
+// The menu item and the scoreboard are given on join (the scoreboard shows
+// what the other join steps made: Monitor); the scoreboard goes on leave,
+// after the features' own leave steps, which may draw it again (island
+// expansion mode redraws it when it ends): the last clean-up, Highest.
 func subscribePresence() {
-	core.Join.On(10, "screens.menuItem", func(e core.Presence) {
+	core.Join.On(core.Normal, "screens.menuItem", func(e core.Presence) {
 		if e.First {
 			// Bare hands: only the menu item.
 			_ = e.P.Inventory().SetItem(8, MenuItem(i18n.Of(e.P)))
 		}
 		RenameMenuItems(e.P)
 	})
-	core.Join.On(70, "screens.board", func(e core.Presence) {
+	core.Join.On(core.Monitor, "screens.board", func(e core.Presence) {
 		UpdateBoard(e.S, e.P)
 	})
-	core.Leave.On(60, "screens.board", func(e core.Presence) {
+	core.Leave.On(core.Highest, "screens.board", func(e core.Presence) {
 		e.P.RemoveScoreboard()
 	})
 }
 
 // A held menu item opens the menu on a block and in the air. A hunter's click
 // on a block is the hunting ground's to refuse (dimension.hunting), whatever
-// the hunter holds; in the air the menu item is the hunter's way out.
+// the hunter holds; in the air the menu item is the hunter's way out. The menu
+// item takes its click before anything else looks at it: Lowest.
 func subscribeUse() {
-	core.UseOnBlock.Handle(20, "screens.menuItem", func(e core.UseOnBlockEvent) bool {
+	core.UseOnBlock.Handle(core.Lowest, "screens.menuItem", func(e core.UseOnBlockEvent) bool {
 		if IsMenuItem(e.Held) && !dimension.Hunting(e.S, e.P) {
 			e.Cancel()
 			useHeldScreen(e.S, e.P)
@@ -51,7 +55,7 @@ func subscribeUse() {
 		}
 		return false
 	})
-	core.Use.Handle(20, "screens.menuItem", func(e core.UseEvent) bool {
+	core.Use.Handle(core.Lowest, "screens.menuItem", func(e core.UseEvent) bool {
 		if IsMenuItem(e.Held) {
 			e.Cancel()
 			useHeldScreen(e.S, e.P)
~~~~

### `content/skygear/storage/subscribe.go`

~~~~diff
--- a/content/skygear/storage/subscribe.go
+++ b/content/skygear/storage/subscribe.go
@@ -15,7 +15,7 @@ func subscribe() {
 
 // The drawer displays are removed when the save closes.
 func subscribeDisplays() {
-	core.Close.On(10, "storage.displays", func(e core.Tick) {
+	core.Close.On(core.Normal, "storage.displays", func(e core.Tick) {
 		RemoveDisplays(e.S, e.Tx)
 	})
 }
@@ -24,7 +24,7 @@ func subscribeDisplays() {
 // loses its display. A set machine breaking or placed invalidates the
 // networks; a placed drawer is set up.
 func subscribeBlocks() {
-	core.BlockBreak.On(60, "storage.break", func(e core.BlockBreakEvent) {
+	core.BlockBreak.On(core.Normal, "storage.break", func(e core.BlockBreakEvent) {
 		switch b := e.Block.(type) {
 		case Drawer:
 			if lp, ok := LastPunch.Of(e.S)[e.P.H()]; ok && lp.Pos == e.Pos && lp.Front {
@@ -38,7 +38,7 @@ func subscribeBlocks() {
 			}
 		}
 	})
-	core.BlockPlace.On(60, "storage.place", func(e core.BlockPlaceEvent) {
+	core.BlockPlace.On(core.Normal, "storage.place", func(e core.BlockPlaceEvent) {
 		switch b := e.Block.(type) {
 		case machine.Machine:
 			if IsSetMachine(b.Kind) {
@@ -50,9 +50,12 @@ func subscribeBlocks() {
 	})
 }
 
-// A sneaking player uses a drawer with a held item that is no block.
+// A sneaking player uses a drawer with a held item that is no block. It is
+// registered before handwork.stick, in the same tier, and takes the click
+// first (a stick opens the drawer's menu and counts no stick use): that order
+// is one of the exceptions the tier order guard keeps (the root's orderKept).
 func subscribeSneakUse() {
-	core.UseOnBlock.Handle(90, "storage.sneakUse", func(e core.UseOnBlockEvent) bool {
+	core.UseOnBlock.Handle(core.Normal, "storage.sneakUse", func(e core.UseOnBlockEvent) bool {
 		if SneakUse(e.S, e.P, e.Tx, e.Pos, e.Held) {
 			e.Cancel()
 			return true
~~~~

### `content/skygear/tierorder_test.go`

~~~~diff
--- a/content/skygear/tierorder_test.go
+++ b/content/skygear/tierorder_test.go
@@ -2,9 +2,7 @@ package skygear
 
 import (
 	"fmt"
-	"hash/fnv"
 	"log/slog"
-	"math/rand/v2"
 	"os"
 	"os/exec"
 	"path/filepath"
@@ -15,7 +13,7 @@ import (
 	"testing"
 
 	"github.com/df-mc/dragonfly/server/world"
-	"github.com/presentkim/skygear/content/skygear/core"
+	"github.com/presentkim/skygear/content/skygear/core/coretest"
 	"github.com/presentkim/skygear/platform/account"
 	"github.com/presentkim/skygear/platform/db"
 )
@@ -23,246 +21,31 @@ import (
 // The tier order guard. A subscriber's place is its tier; inside a tier no
 // subscriber may depend on where it stands. TestTierOrder proves it: it runs
 // this test binary again with the subscribers of every hook reordered inside
-// their tiers (SKYGEAR_HOOK_ORDER) and compares what every pin scenario did.
-//
-// Until the kernel has tiers, the tier of each subscriber is the one planned
-// for it (plannedTiers), and a reordered run first sorts each hook by planned
-// tier, keeping today's order inside a tier ("tiers"), then reverses
-// ("reverse") or shuffles ("shuffle:<seed>") every tier.
+// their tiers (coretest.HookOrderEnv: reversed, and shuffled with a few seeds)
+// and compares what every pin scenario did.
 
-// hookOrder is the reordering of this process: "" (the order of the code),
-// "tiers", "reverse" or "shuffle:<seed>".
-var hookOrder = os.Getenv("SKYGEAR_HOOK_ORDER")
+// hookOrder is the reordering of this process: "" (the order of
+// registration), "reverse" or "shuffle:<seed>".
+var hookOrder = os.Getenv(coretest.HookOrderEnv)
 
 // pinDump, when set, is the file every pin test appends its scenarios to:
 // one line "<scenario> | <results> | <counters>" each (pinDumpLines).
 var pinDump = os.Getenv("SKYGEAR_PIN_DUMP")
 
-// tierNames are the tiers in the order they run.
-var tierNames = []string{"LOWEST", "LOW", "NORMAL", "HIGH", "HIGHEST", "MONITOR"}
-
-// plannedTiers is the tier planned for every subscriber, one
-// "<Hook> <TIER> <name>" per line, in today's order.
-const plannedTiers = `
-StartBreak LOWEST expand.guard
-StartBreak LOW island.guard
-StartBreak NORMAL tinker.autoTool
-BlockBreak LOWEST expand.guard
-BlockBreak LOW island.guard
-BlockBreak MONITOR achievement.break
-BlockBreak NORMAL tinker.break
-BlockBreak NORMAL farm.break
-BlockBreak NORMAL storage.break
-BlockBreak NORMAL netstore.break
-BlockBreak NORMAL machine.break
-BlockBreak NORMAL conduit.break
-BlockPlace LOWEST expand.guard
-BlockPlace LOW island.guard
-BlockPlace MONITOR achievement.place
-BlockPlace HIGH autofill.place
-BlockPlace NORMAL farm.plant
-BlockPlace NORMAL storage.place
-BlockPlace NORMAL machine.place
-BlockPlace HIGH netstore.place
-BlockPlace MONITOR conduit.place
-BlockPlace MONITOR progress.dragonEgg
-UseOnBlock LOWEST dimension.hunting
-UseOnBlock LOWEST screens.menuItem
-UseOnBlock LOWEST dimension.stone
-UseOnBlock LOWEST expand.guard
-UseOnBlock LOW island.guard
-UseOnBlock NORMAL machine.noteFace
-UseOnBlock LOW copier.use
-UseOnBlock LOW wand.use
-UseOnBlock NORMAL storage.sneakUse
-UseOnBlock NORMAL conduit.plate
-UseOnBlock NORMAL handwork.stick
-UseOnBlock HIGH farm.composter
-UseOnBlock NORMAL handwork.gather
-UseOnBlock NORMAL fairy.time
-UseOnBlock HIGH farm.crop
-UseOnBlock HIGHEST expand.guardSide
-UseOnBlock HIGHEST island.guardSide
-UseOnEntity LOWEST dimension.hunting
-UseOnEntity NORMAL fairy.touch
-AttackEntity LOWEST dimension.hunter
-AttackEntity NORMAL fairy.touch
-Use HIGH autofill.use
-Use LOWEST screens.menuItem
-Use LOWEST dimension.hunting
-Use LOWEST dimension.stone
-Use NORMAL copier.clear
-Use NORMAL wand.options
-Use NORMAL tinker.veinToggle
-Move LOWEST look.update
-Move LOWEST expand.hold
-Move MONITOR achievement.walk
-Move HIGHEST island.void
-Sneak LOWEST expand.show
-Sneak LOW achievement.sneak
-Sneak NORMAL elevator.down
-Sneak HIGH farm.crouchGrow
-Sprint MONITOR achievement.sprint
-Jump MONITOR achievement.jump
-Jump NORMAL elevator.up
-ItemDamage LOW tinker.ironHammer
-ItemDamage HIGH autofill.broken
-ItemConsume MONITOR achievement.eat
-ItemConsume HIGH autofill.consume
-ItemPickup MONITOR achievement.pickup
-ItemDrop LOWEST dimension.hunting
-ItemDrop MONITOR achievement.drop
-Death MONITOR achievement.death
-Death NORMAL dimension.died
-Death NORMAL tinker.keepFood
-Respawn LOW dimension.respawned
-Respawn NORMAL tinker.restoreFood
-Hurt LOWEST dimension.frozen
-Hurt MONITOR achievement.hurt
-Hurt HIGHEST dimension.fallen
-PunchAir MONITOR achievement.punchAir
-Chat MONITOR achievement.chat
-BlockPick MONITOR achievement.blockPick
-HeldSlot MONITOR achievement.heldSlot
-Command MONITOR achievement.command
-Load NORMAL farm.loadCrops
-Start NORMAL island.keepLoaded
-Start NORMAL climate.apply
-Join NORMAL screens.menuItem
-Join NORMAL fairy.backfillDust
-Join NORMAL climate.showWeather
-Join NORMAL core.shapes
-Join NORMAL island.purgeLost
-Join NORMAL progress.sendRecipes
-Join MONITOR screens.board
-Join HIGH dimension.joined
-Leave NORMAL dimension.leave
-Leave NORMAL expand.stop
-Leave NORMAL core.parts
-Leave HIGHEST look.clear
-Leave NORMAL core.shapes
-Leave HIGHEST screens.board
-Leave HIGHEST progress.recipes
-NewDay NORMAL island.purgeLost
-Second NORMAL farm.growCrops
-Second LOW progress.checkHeld
-Second NORMAL farm.barrels
-Second NORMAL fairy.tick
-Second NORMAL climate.tick
-Second NORMAL dimension.second
-Second NORMAL tinker.effects
-Second NORMAL machine.monitor
-Second NORMAL farm.plants
-Second MONITOR screens.boards
-Second MONITOR achievement.playSeconds
-Second MONITOR progress.collection
-Fast NORMAL network.step
-Fast NORMAL dimension.watchdog
-Fast MONITOR look.updateAll
-Autosave NORMAL netstore.flush
-Autosave NORMAL conduit.shares
-Close NORMAL storage.displays
-Close NORMAL conduit.close
-Close NORMAL netstore.flush
-ColumnLoad NORMAL conduit.load
-ColumnUnload NORMAL conduit.unload
-EntityDespawn NORMAL island.lost
-`
-
-// orderKept are the pairs that still depend on their order inside one tier:
-// "<Hook> <first> <second>". A reordered run keeps first before second. Each
-// entry is a dependency to remove (or, for the last ones, an exception the
-// owner decides on); the list only shrinks.
+// orderKept are the exceptions: the pairs of one tier whose order still
+// matters, "<Hook> <first> <second>". The kernel runs first before second
+// because first's feature is installed earlier (features.go), and a reordered
+// run keeps them so. Each is a known dependency on the place inside a tier,
+// kept until it is decided how to remove it (docs/tech/extension.md); the
+// list does not grow.
 var orderKept = []string{
-	// a click the drawer menu or a plate takes counts no stick use
+	// a click the drawer menu or a connector plate takes counts no stick use
+	// (handwork.stick counts "stick_other" when the click reaches it)
 	"UseOnBlock storage.sneakUse handwork.stick",
 	"UseOnBlock conduit.plate handwork.stick",
-	// a plate's settings open before the hourglass winds the machine
+	// a connector plate's settings open before the hourglass in hand winds
+	// the machine the plate is on
 	"UseOnBlock conduit.plate fairy.time",
-	// the panel updates on the move island expansion mode refuses
-	"Move look.update expand.hold",
-	// the player's entries are dropped after the features used them
-	"Leave dimension.leave core.parts",
-	"Leave expand.stop core.parts",
-}
-
-func tierIndex(name string) int {
-	i := slices.Index(tierNames, name)
-	if i < 0 {
-		panic("unknown tier " + name)
-	}
-	return i
-}
-
-// plannedTierOf returns the planned tier of every subscriber, keyed
-// "<Hook> <name>".
-func plannedTierOf() map[string]int {
-	out := map[string]int{}
-	for _, ln := range strings.Split(strings.TrimSpace(plannedTiers), "\n") {
-		f := strings.Fields(ln)
-		out[f[0]+" "+f[2]] = tierIndex(f[1])
-	}
-	return out
-}
-
-// reorderHooks applies mode to the kernel's hooks (core.Reorder).
-func reorderHooks(mode string) {
-	tiers := plannedTierOf()
-	kind, arg, _ := strings.Cut(mode, ":")
-	var seed uint64
-	if kind == "shuffle" {
-		if _, err := fmt.Sscan(arg, &seed); err != nil {
-			panic("SKYGEAR_HOOK_ORDER: " + mode)
-		}
-	} else if kind != "tiers" && kind != "reverse" {
-		panic("SKYGEAR_HOOK_ORDER: " + mode)
-	}
-	core.Reorder(func(hook string, subs []core.Subscription) []int {
-		groups := make([][]int, len(tierNames))
-		for i, s := range subs {
-			t, ok := tiers[hook+" "+s.Name]
-			if !ok {
-				panic(fmt.Sprintf("plannedTiers has no line for %s %s", hook, s.Name))
-			}
-			groups[t] = append(groups[t], i)
-		}
-		h := fnv.New64a()
-		h.Write([]byte(hook))
-		r := rand.New(rand.NewPCG(seed, h.Sum64()))
-		var idx []int
-		for _, g := range groups {
-			switch kind {
-			case "reverse":
-				slices.Reverse(g)
-			case "shuffle":
-				r.Shuffle(len(g), func(i, j int) { g[i], g[j] = g[j], g[i] })
-			}
-			idx = append(idx, g...)
-		}
-		// Put the kept pairs back in their order: a first that ended up after
-		// its second moves to just before it.
-		for moved := true; moved; {
-			moved = false
-			for _, k := range orderKept {
-				f := strings.Fields(k)
-				if f[0] != hook {
-					continue
-				}
-				a := slices.IndexFunc(idx, func(i int) bool { return subs[i].Name == f[1] })
-				b := slices.IndexFunc(idx, func(i int) bool { return subs[i].Name == f[2] })
-				if a < 0 || b < 0 {
-					panic("orderKept names no subscriber: " + k)
-				}
-				if a > b {
-					v := idx[a]
-					idx = slices.Delete(idx, a, a+1)
-					idx = slices.Insert(idx, b, v)
-					moved = true
-				}
-			}
-		}
-		return idx
-	})
 }
 
 // pinAccounts are the accounts of the players the tests spawned, in a run
@@ -277,9 +60,7 @@ var (
 )
 
 func TestMain(m *testing.M) {
-	if hookOrder != "" {
-		reorderHooks(hookOrder)
-	}
+	coretest.ReorderHooks(orderKept...)
 	if pinDump != "" {
 		dir, err := os.MkdirTemp("", "pindump")
 		if err != nil {
@@ -391,14 +172,16 @@ func pinWriteDump(t *testing.T, lines map[string]string) {
 }
 
 // tierOrders are the reorderings TestTierOrder runs.
-var tierOrders = []string{"tiers", "reverse", "shuffle:1", "shuffle:2", "shuffle:3"}
+var tierOrders = []string{"reverse", "shuffle:1", "shuffle:2", "shuffle:3", "shuffle:4"}
 
 // TestTierOrder runs every test of this package once per reordering of
-// tierOrders, in a process of its own, and once in the order of the code. A
-// reordered run must pass, which includes that every pin scenario has the
-// results of its golden (the trace before the first " | " may differ), and
-// must count exactly what the run in the order of the code counted in every
-// scenario (the achievement counters of the players, which no golden holds).
+// tierOrders, in a process of its own, and once in the order of
+// registration. A reordered run must pass, which includes that every pin
+// scenario has the results of its golden (the trace before the first " | "
+// may differ) and that order.golden lists the same subscribers in every tier,
+// and it must count exactly what the run in the order of registration counted
+// in every scenario (the achievement counters of the players, which no golden
+// holds).
 func TestTierOrder(t *testing.T) {
 	if hookOrder != "" || pinDump != "" {
 		return // this is a run TestTierOrder started
@@ -411,7 +194,7 @@ func TestTierOrder(t *testing.T) {
 	run := func(mode string) (dump string, out []byte, err error) {
 		dump = filepath.Join(dir, strings.ReplaceAll(mode+".dump", ":", "-"))
 		cmd := exec.Command(exe, "-test.count=1")
-		cmd.Env = append(os.Environ(), "SKYGEAR_HOOK_ORDER="+mode, "SKYGEAR_PIN_DUMP="+dump)
+		cmd.Env = append(os.Environ(), coretest.HookOrderEnv+"="+mode, "SKYGEAR_PIN_DUMP="+dump)
 		out, err = cmd.CombinedOutput()
 		return dump, out, err
 	}
@@ -451,7 +234,7 @@ func TestTierOrder(t *testing.T) {
 	}
 	want := read(results[0].dump)
 	if len(want) < 200 {
-		t.Fatalf("the run in the order of the code dumped %d scenarios", len(want))
+		t.Fatalf("the run in the order of registration dumped %d scenarios", len(want))
 	}
 	for _, r := range results[1:] {
 		got := read(r.dump)
~~~~

### `content/skygear/tinker/subscribe.go`

~~~~diff
--- a/content/skygear/tinker/subscribe.go
+++ b/content/skygear/tinker/subscribe.go
@@ -29,15 +29,16 @@ func subscribe() {
 
 func subscribeFood() {
 	// The iron hammer keeps whole: its vein mining costs food instead (vein.go).
-	core.ItemDamage.On(10, "tinker.ironHammer", func(e core.ItemDamageEvent) {
+	// It shapes the damage others read: Low.
+	core.ItemDamage.On(core.Low, "tinker.ironHammer", func(e core.ItemDamageEvent) {
 		if sgitem.IsIronHammer(e.Item) {
 			*e.Damage = 0
 		}
 	})
-	core.Death.On(30, "tinker.keepFood", func(e core.PlayerEvent) {
+	core.Death.On(core.Normal, "tinker.keepFood", func(e core.PlayerEvent) {
 		keptFood.Of(e.S)[e.P.H()] = e.P.Food()
 	})
-	core.Respawn.On(20, "tinker.restoreFood", func(e core.RespawnEvent) {
+	core.Respawn.On(core.Normal, "tinker.restoreFood", func(e core.RespawnEvent) {
 		if food, ok := keptFood.Of(e.S)[e.P.H()]; ok {
 			delete(keptFood.Of(e.S), e.P.H())
 			e.P.SetFood(food)
@@ -47,7 +48,7 @@ func subscribeFood() {
 
 // The worn tools give their effects every second.
 func subscribeEffects() {
-	core.Second.On(70, "tinker.effects", func(e core.Tick) {
+	core.Second.On(core.Normal, "tinker.effects", func(e core.Tick) {
 		TickEffects(e.S, e.Tx)
 	})
 }
@@ -56,10 +57,10 @@ func subscribeEffects() {
 // tool gives it experience, and a hammer crushes the block and mines the vein.
 // The hammer's drop replaces the drops in place.
 func subscribeTools() {
-	core.StartBreak.On(30, "tinker.autoTool", func(e core.StartBreakEvent) {
+	core.StartBreak.On(core.Normal, "tinker.autoTool", func(e core.StartBreakEvent) {
 		AutoTool(e.P, e.Tx.Block(e.Pos))
 	})
-	core.BlockBreak.On(40, "tinker.break", func(e core.BlockBreakEvent) {
+	core.BlockBreak.On(core.Normal, "tinker.break", func(e core.BlockBreakEvent) {
 		if _, ok := ToolOf(e.P); ok {
 			GainXP(e.S, e.P, 1)
 			Reinforce(e.S, e.P)
@@ -76,7 +77,7 @@ func subscribeTools() {
 // The iron hammer used on the air while sneaking switches diagonal vein
 // mining.
 func subscribeVeinToggle() {
-	core.Use.On(70, "tinker.veinToggle", func(e core.UseEvent) {
+	core.Use.On(core.Normal, "tinker.veinToggle", func(e core.UseEvent) {
 		if sgitem.IsIronHammer(e.Held) && e.P.Sneaking() && !e.S.Debounced(e.P) {
 			ToggleVeinDiagonal(e.S, e.P)
 		}
~~~~

### `content/skygear/wand/subscribe.go`

~~~~diff
--- a/content/skygear/wand/subscribe.go
+++ b/content/skygear/wand/subscribe.go
@@ -16,8 +16,10 @@ func subscribe() {
 // The construction wand builds out the face clicked and does nothing else;
 // used on the air while sneaking it shows its settings. Where the island does
 // not allow building it leaves the click alone: the island's guard refuses it.
+// A held tool takes the click away from what the block itself would do with
+// it: the Low tier.
 func subscribeUse() {
-	core.UseOnBlock.Handle(80, "wand.use", func(e core.UseOnBlockEvent) bool {
+	core.UseOnBlock.Handle(core.Low, "wand.use", func(e core.UseOnBlockEvent) bool {
 		if _, ok := sgitem.WandOf(e.Held); ok && island.CanBuild(e.S, e.Pos) {
 			e.Cancel()
 			Use(e.S, e.P, e.Tx, e.Pos, e.Face, e.Held)
@@ -25,7 +27,7 @@ func subscribeUse() {
 		}
 		return false
 	})
-	core.Use.On(60, "wand.options", func(e core.UseEvent) {
+	core.Use.On(core.Normal, "wand.options", func(e core.UseEvent) {
 		if _, ok := sgitem.WandOf(e.Held); ok && e.P.Sneaking() && !e.S.Debounced(e.P) {
 			ShowOptions(e.S, e.P)
 		}
~~~~

## REF T125

Prototype patch: `T125.patch`.

Documents only, by hand. Each `-` block of the diff is the old text to find and each `+` block the text that replaces it; the six role sentences and the sentence on the option are the owner's and are copied letter for letter. If the old text is not found as given (another change touched the paragraph), stop and return `NEEDS_CONTEXT`.

### `CONTRIBUTING.md`

~~~~diff
--- a/CONTRIBUTING.md
+++ b/CONTRIBUTING.md
@@ -59,7 +59,8 @@ Thanks for your interest. These are the rules the project follows.
   `Settings`, `ItemEntry`, `Creative`, `Commands`, `Setup`) and subscribes
   its steps to the kernel's hooks in `Setup`, in `subscribe.go`, with
   subscriber names
-  `<feature>.<step>` and a priority unique within the hook. It keeps its data in
+  `<feature>.<step>` and the tier (`core.Lowest` to `core.Monitor`) of what the
+  step does; inside a tier no step may depend on its place. It keeps its data in
   `core.NewPart`, `core.NewPlayers` and `core.NewSaved` parts, never in a field
   of `Session`, and never imports the root package. Its blocks fill in their
   own WAILA lines (`Waila`, in `look.go`) and its items their slot decoration
~~~~

### `content/skygear/README.md`

~~~~diff
--- a/content/skygear/README.md
+++ b/content/skygear/README.md
@@ -8,7 +8,7 @@
 기능마다 패키지가 하나다. 패키지는 아래 표에서 **자기보다 위 칸**의 패키지만 import 한다(같은 칸끼리는 표의 순서대로).
 서버 이벤트(블록 캐기·놓기·사용, 이동, 입장·퇴장, 매초와 빠른 틱 …)는 `core` 의 훅으로 기능에 닿고, 기능은 자기 패키지의 `feature.go`
 (`core.Feature`)의 `Setup` 에서 구독한다. 블록의 WAILA 줄(`Waila` 메서드), 아이템의 칸 장식(`Decor` 메서드), 창작 목록 항목(`Creative`), 명령어(`Commands`), 설정(`Settings`)도
-기능이 자기 패키지에서 선언한다. 기능은 루트 패키지의 `features.go` 목록에 한 줄로 오른다. 훅, 우선순위, 새 기능을 더하는 순서는
+기능이 자기 패키지에서 선언한다. 기능은 루트 패키지의 `features.go` 목록에 한 줄로 오른다. 훅, 구독자의 티어, 새 기능을 더하는 순서는
 [docs/tech/extension.md](../../docs/tech/extension.md) 에 있다.
 위 칸이 아래 칸을 불러야 할 때 중 이벤트가 아닌 것(메뉴 열기, 사이드바 갱신)은 위 칸이 함수 변수를 두고,
 아래 칸이 `init` 에서 채운다(예: `core.ShowMenu` 는 `screens` 가 채운다). 다른 기능의 기계 종류는 `machine` 이 이름과 정의를 두고,
~~~~

### `docs/design/03-architecture.md`

~~~~diff
--- a/docs/design/03-architecture.md
+++ b/docs/design/03-architecture.md
@@ -102,7 +102,7 @@ type Session interface {
   저장 데이터 `core.NewSaved`(모든 저장 파트가 세이브 상태 문서 하나의 필드라 기능을 옮겨도 세이브 형식이 그대로다).
 - 패키지는 자기보다 아래 층만 import 한다(데이터 → 핵심 → 진행 → 기능 → 내용 목록 → 화면 → 게임). 위로 불러야 할 때는
   아래 패키지가 함수 변수를 두고 위 패키지가 `init` 에서 채운다. `layers_test.go` 가 이 방향을 검사한다.
-- 이벤트는 훅으로 기능에 닿는다. Dragonfly 가 `core.PlayerHandler`·`core.WorldHandler` 를 부르면 같은 이름의 훅이 우선순위 차례로 기능의 구독자를 부르고,
+- 이벤트는 훅으로 기능에 닿는다. Dragonfly 가 `core.PlayerHandler`·`core.WorldHandler` 를 부르면 같은 이름의 훅이 티어 차례로 기능의 구독자를 부르고,
   플랫폼이 부르는 세션 호출(`Join`, `Leave`, `NewDay`, `Second`, `Fast`, `Autosave`, `Close`)은 `core.Lifecycle` 이 같은 식으로 훅에 내보낸다.
   기능은 자기 패키지의 `feature.go` 에서 `core.Feature` 를 선언하고 `Setup` 에서 구독자를 단다.
 - 루트 패키지 `skygear` 는 기능 목록(`features.go`, `core.Install`)과 플랫폼 접착제(`Game`, `session`)이다. 플랫폼 인터페이스를 구현하고 운영자 도구(`/sgadmin`, `/sgdebug`)를 두며, 훅에 구독하지 않는다. 플레이어 명령어는 루트에 없고 기능이 `core.Feature` 의 `Commands` 로 더한다.
~~~~

### `docs/tech/extension.md`

~~~~diff
--- a/docs/tech/extension.md
+++ b/docs/tech/extension.md
@@ -62,20 +62,53 @@ Dragonfly 자체를 확장하는 방법은 [dragonfly.md](dragonfly.md) 에 있
 
 세션 타입은 `core.Lifecycle` 을 품고 있어 이 호출들이 `Lifecycle` 의 메서드로 간다. 메서드마다 이벤트를 만들어 훅의 `Emit` 을 한 번 부르고, 아래 꼬리 일을 한다.
 
+- `Leave`: 구독자가 끝나면 플레이어 클라이언트의 도형을 지우고(`RemoveAllDebugShapes`) 플레이어별 파트 항목을 지운다(`Session.Leave`). 구독자가 이벤트를 멈춰도 한다.
 - `Second`: 구독자가 끝나면 바뀐 상태를 저장 주기에 맞을 때만 쓴다(`FlushIfDue`).
 - `Autosave`: 구독자가 끝나면 바뀐 상태를 쓴다(`Flush`).
 - `Close`: 구독자가 끝나면 저장 파트를 쓰고 쓰기 고루틴을 멈추고(`Session.Close`), 세션 등록을 지운다(`Unregister`). 루트의 `session.Close` 는 그 뒤에 `Game` 의 세션 표에서도 지운다.
 - `NewDay`: 구독자가 끝나면 모든 플레이어에게 `game.newDay` 를 알린다.
 
-`Emit` 은 구독자를 우선순위가 작은 것부터 차례로 부른다.
+`Emit` 은 구독자를 티어 순서대로 부른다.
 
-### On 과 Handle
+### 티어
 
-- `On(prio, name, f)` 로 단 구독자는 이벤트를 멈추지 못한다.
-- `Handle(prio, name, f)` 로 단 구독자가 true 를 돌려주면 뒤의 구독자는 돌지 않는다. `Emit` 도 true 를 돌려주지만 핸들러와 수명 호출은 그 값을 쓰지 않는다.
+구독자는 티어(`core.Priority`)에 선다. 티어는 여섯이고 표의 순서대로 돈다. 티어가 정하는 것은 역할뿐이다. 구독자는 다른 구독자의 앞이나 뒤가 아니라 자기가 하는 일에 맞는 티어를 고른다.
 
-멈춤과 취소는 다르다. 멈춤은 훅의 일이고 뒤의 구독자만 건너뛴다. 취소는 Dragonfly 의 일이고 플레이어가 한 행동이 일어나지 않게 한다. `Actor.Cancel()`(`e.Cancel()`)이 취소하며, 취소해도 뒤의 구독자는 계속 돈다.
-구독자는 둘 중 하나만 하거나 둘 다 하거나 아무것도 안 할 수 있다. `Jump`, `Death`, `Respawn` 은 Dragonfly 가 취소할 수 없는 이벤트라 `Ctx` 가 nil 이고 `Cancel()` 은 아무 일도 하지 않는다.
+| 티어 | 상수 | 역할 |
+|---|---|---|
+| LOWEST | `core.Lowest` | 전처리 - 먼저 이벤트 자체를 가공하거나 차단해야 할 때 사용합니다. |
+| LOW | `core.Low` | 전처리 이후 처리되어도 될 이벤트 가공이나 차단에 사용합니다. |
+| NORMAL | `core.Normal` | 전처리 이후에 처리되어도 될 일반 우선순위에 사용합니다. |
+| HIGH | `core.High` | 후처리에 해당하진 않지만, 우선순위가 낮은 작업에 사용합니다. |
+| HIGHEST | `core.Highest` | 이벤트를 가공할 수 있는 마지막 후처리 작업에 사용합니다. |
+| MONITOR | `core.Monitor` | 이벤트를 가공하거나 취소하지 않는 결과 처리(기록, 업적 처리 등)에 사용합니다. |
+
+이벤트가 취소되어도 수신할지를 선택할 수 있다(전통적인 handleCancelled 옵션). 기본은 받지 않는 것이고, `Always` 로 단 구독자만 받는다. 여기서 취소된 이벤트는 앞의 `Handle` 구독자가 멈춘 이벤트를 말한다(다음 절).
+
+같은 훅의 같은 티어 안에서는 등록한 순서대로 돈다. 기능 목록(`features.go`)의 순서이고, 한 기능 안에서는 `subscribe()` 가 단 순서다. 이 순서는 실행이 매번 같게 할 뿐이다. 구독자는 같은 티어 안의 자리에 기대면 안 된다. 같은 티어의 두 구독자는 어느 쪽이 먼저 돌아도 결과가 같아야 한다. 순서가 필요하면 티어를 나누거나, 구독자의 조건에 그 사실을 적거나, 한 구독자로 합친다. `TestTierOrder` 가 이 규칙을 지킨다(아래 "핀").
+
+MONITOR 구독자는 이벤트를 가공하지도 취소하지도 않는다. MONITOR 에 `Handle` 을 달면 패닉이다. MONITOR 구독자의 함수가 `Cancel()` 을 부르거나 이벤트의 포인터에 쓰면 `TestMonitorSubscribersDoNotShape` 가 실패한다. Bukkit 의 MONITOR 와 역할은 같고 기본값이 다르다. 여기서는 MONITOR 구독자도 멈춘 이벤트를 받지 않는다. 받으려면 `Always` 로 단다.
+
+수명 훅(`Join`, `Leave`, `Second`, `Fast`, `Autosave`, `Close` 따위)에는 가공하거나 취소할 것이 없다. 거기서는 티어를 일의 차례로 읽는다.
+
+- LOWEST, LOW: 뒤의 일이 바꾸기 전에 상태를 읽어 두는 일(`Second` 의 `progress.checkHeld`).
+- NORMAL: 기능이 그때 하는 일.
+- HIGH: 다른 기능의 일이 끝난 뒤에 해도 되는 일.
+- HIGHEST: 마지막 정리.
+- MONITOR: 결과를 보여 주거나 세는 일(사이드바, 패널, 업적).
+
+틱 훅(`Second`, `Fast`)에서 같은 티어의 구독자는 서로의 이번 틱 결과를 읽지 않는다. 읽어야 하면 티어를 나눈다.
+
+### On, Handle, Always
+
+- `On(티어, name, f)` 로 단 구독자는 이벤트를 멈추지 못하고, 멈춘 이벤트를 받지 않는다.
+- `Handle(티어, name, f)` 로 단 구독자가 true 를 돌려주면 이벤트가 멈춘다. 뒤의 구독자는 돌지 않고(`Always` 로 단 구독자는 돈다) `Emit` 이 true 를 돌려준다. 핸들러와 수명 호출은 그 값을 쓰지 않는다. MONITOR 에는 달 수 없다.
+- `Always(티어, name, f)` 로 단 구독자는 `On` 과 같되 앞에서 멈춘 이벤트도 받는다. 함수는 `func(e E, cancelled bool)` 이고, 멈춘 이벤트면 `cancelled` 가 true 다. 스스로 멈추지는 못한다. 어느 훅, 어느 티어에나 달 수 있고, `Handle` 구독자가 없는 훅에서는 `cancelled` 가 늘 false 다. 지금은 `Move` 의 `look.update` 가 쓴다. 섬 확장 모드가 이동을 멈춰도 패널을 갱신한다.
+
+멈춤과 취소는 다르다. 멈춤은 훅의 일이고 뒤의 구독자를 건너뛴다. 취소는 Dragonfly 의 일이고 플레이어가 한 행동이 일어나지 않게 한다. `Actor.Cancel()`(`e.Cancel()`)이 취소하며, 취소만 하면 뒤의 구독자는 계속 돈다. `e.Cancelled()` 가 취소됐는지 알려 준다.
+구독자는 둘 중 하나만 하거나 둘 다 하거나 아무것도 안 할 수 있다. `Jump`, `Death`, `Respawn` 은 Dragonfly 가 취소할 수 없는 이벤트라 `Ctx` 가 nil 이고 `Cancel()` 은 아무 일도 하지 않으며 `Cancelled()` 는 false 다.
+
+handleCancelled 옵션이 멈춤을 기준으로 삼는 까닭은 구독자들이 실제로 그렇게 쓰기 때문이다. `Handle` 구독자 31개 가운데 29개는 멈추면서 `Cancel()` 도 한다. 2개(`Sneak` 의 `expand.show`, `elevator.down`)는 멈추기만 한다. 웅크리기를 취소하면 플레이어가 웅크리지 못하기 때문이다. `Cancel()` 만 하고 멈추지 않는 구독자는 넷(`storage.break`, `island.void`, `dimension.fallen`, `fairy.touch`)이고, 그 뒤의 구독자는 `On` 으로 달았어도 그대로 돈다. `Cancelled()` 를 보고 일을 건너뛰는 구독자는 없다.
 
 ### 이벤트의 값
 
@@ -157,13 +190,13 @@ Dragonfly 자체를 확장하는 방법은 [dragonfly.md](dragonfly.md) 에 있
    }
 
    func subscribeJump() {
-       core.Jump.On(20, "elevator.up", func(e core.PlayerEvent) {
+       core.Jump.On(core.Normal, "elevator.up", func(e core.PlayerEvent) {
            Up(e.S, e.P, e.Tx)
        })
    }
 
    func subscribeSneak() {
-       core.Sneak.Handle(30, "elevator.down", func(e core.PlayerEvent) bool {
+       core.Sneak.Handle(core.Normal, "elevator.down", func(e core.PlayerEvent) bool {
            return Down(e.S, e.P, e.Tx)
        })
    }
@@ -298,32 +331,65 @@ Dragonfly 자체를 확장하는 방법은 [dragonfly.md](dragonfly.md) 에 있
 11. 커스텀 아이템 항목을 고치는 함수는 서술자의 `ItemEntry` 에 둔다(아래 "클라이언트 항목 고치기"). 지금은 루트의 `itemsFeature`(`features.go`)만 쓴다. 클라이언트가 받는 항목이 바뀌므로 `content/skygear/wire/testdata/items.golden`(`TestWire`)이 달라진다.
 12. 루트의 `content/skygear/features.go` 의 `Features` 에 `<기능>.Feature` 를 한 줄 더하고 import 를 더한다. 설정, 아이템 항목 고치기, 창작 항목, 명령어는 목록 순서대로 모이고 `Setup` 도 그 순서대로 돈다.
 13. `content/skygear/layers_test.go` 에 기능을 알린다. `layers` 에서 칸을 정하고(자기보다 아래 칸에 든 패키지만 import 할 수 있다), 기능 칸이면 `featureLayer` 에도 이름을 더한다(`featureLayer` 가 `layers` 의 기능 칸과 다르면 `TestFeatureSiblings` 가 실패한다). 같은 칸의 기능을 import 하면 `siblings` 에 행을 더한다. 기능의 순서는 `TestFeatureNames` 의 `want` 목록에도 적혀 있으니 `Features` 에 더한 자리에 같이 더한다.
-14. 구독자를 달았으면 골든이 달라진다. `order.golden` 에는 새 줄(`<훅> <우선순위> <이름>`)이 생기고, 그 훅을 거치는 장면이 있으면 `handler-simple.golden`, `handler-blocks.golden`, `handler-use.golden`, `lifecycle.golden` 의 그 장면 줄에도 새 이름이 들어간다. 의도한 변경이므로 `SKYGEAR_GOLDEN=update go test ./content/skygear -run TestPin -count=1` 로 다시 기록하고 `git diff` 로 새 단계의 이름만 늘었는지 확인한다. 커밋 메시지에 어느 단계를 더했는지 적는다.
+14. 구독자를 달았으면 골든이 달라진다. `order.golden` 에는 새 줄(`<훅> <티어> <이름>`)이 생기고, 그 훅을 거치는 장면이 있으면 `handler-simple.golden`, `handler-blocks.golden`, `handler-use.golden`, `lifecycle.golden` 의 그 장면 줄에도 새 이름이 들어간다. 의도한 변경이므로 `SKYGEAR_GOLDEN=update go test ./content/skygear -run TestPin -count=1` 로 다시 기록하고 `git diff` 로 새 단계의 이름만 늘었는지 확인한다. 커밋 메시지에 어느 단계를 더했는지 적는다.
 15. `go test ./content/skygear -run 'TestLayers|TestFeature|TestCapability|TestLook|TestCreative|TestItemDecor|TestEvery|TestRoot|TestPin|TestHandlerMethodsOneWay|TestKind|TestMachine' -count=1` 로 확인한다. 위 순서를 빠뜨리면 이 테스트들이 `TestLayers`(`package <기능> is not in layers`), `TestFeatureNames`, `TestPinOrder`, `TestEverySubscriberHasItsFeature`, `TestPinSimple`, `TestCreativeListNamesNoFeature`, `TestFeatureCommands` 따위로 어긋난 곳을 알려 준다.
 
-### 우선순위 고르기
+### 티어 고르기
+
+이름은 한 훅 안에서 겹치면 안 된다(훅이 다르면 같아도 된다). 겹치면 `Install` 이 패닉한다. 티어는 겹쳐도 된다.
+다른 구독자의 자리를 보고 고르지 않는다. 내 구독자가 하는 일을 보고 고른다.
+
+1. 이벤트를 가공하지도 취소하지도 않고 결과를 세거나 보여 주기만 한다: MONITOR. 멈춘 이벤트도 봐야 하면 `Always` 로 단다.
+2. 플레이어를 붙잡은 모드처럼 무엇보다 먼저 막아야 한다: LOWEST. 다른 구독자가 읽을 값을 고치거나 그 뒤에 막아도 된다: LOW.
+3. 다른 구독자가 다 고친 뒤의 값을 보고 마지막으로 가공하거나 취소한다: HIGHEST.
+4. 다른 구독자가 이벤트를 쓰고 남았을 때만 하는 일, 다른 기능이 일을 마친 뒤에 하는 일: HIGH.
+5. 그 밖의 일: NORMAL.
+
+같은 티어에 이미 있는 구독자와 겹치는지 본다. 같은 클릭을 둘 다 가져갈 수 있으면 누가 먼저 돌아도 결과가 같게 조건을 적는다(든 아이템, 블록 종류, 웅크림, `island.CanBuild`). 조건으로 가를 수 없으면 티어를 나눈다. 구독자를 달고 `go test ./content/skygear -run TestTierOrder -count=1` 을 돌린다.
 
-우선순위는 한 훅 안에서 겹치면 안 되고, 이름도 한 훅 안에서 겹치면 안 된다(이름은 훅이 다르면 같아도 된다). 겹치면 `Install` 이 패닉한다.
-기존 단계는 10, 20, 30처럼 10씩 띄워 두었다. 둘 사이에 끼우려면 그 사이의 수를 쓴다(15 등). 현재 순서는 `order.golden` 에 훅별로 모여 있으니 먼저 거기서 이웃 단계를 확인한다.
-무엇보다 앞에서 도는지, 어느 단계가 멈추면 내 단계가 건너뛰어도 되는지를 본다(다음 절).
+### 순서를 지키는 것
 
-### 순서가 중요한 곳
+지금 순서가 필요한 곳과 그것을 지키는 방법이다. 같은 티어 안의 자리나 역할에 맞지 않는 티어로 지키는 곳은 "예외"의 넷뿐이다.
 
-지금 순서에서 알아 둘 사실이다.
+티어가 지키는 것:
 
-- `StartBreak`, `BlockBreak`, `BlockPlace` 에서는 `expand.guard`(10)와 `island.guard`(20)가 맨 앞이다. 월드를 바꾸는 단계는 모두 그 뒤에 선다. 가드는 `Handle` 로 멈추면서 `Cancel()` 도 한다.
-- `UseOnBlock` 에서는 `dimension.hunting`(10), `screens.menuItem`(20), `dimension.stone`(30)이 두 가드(`expand.guard` 40, `island.guard` 50)보다 앞이고, 블록을 쓰는 단계(`copier.use` 70부터)는 가드 뒤다. `fairy.time`(140) 다음에 `farm.crop`(150)이 돌고, 옆 칸 가드(`expand.guardSide` 160, `island.guardSide` 170)는 그 뒤다.
-- `UseOnBlock` 에서는 사냥 검사(`dimension.hunting`)가 메뉴 아이템(`screens.menuItem`)보다 앞이고, `Use` 에서는 메뉴 아이템(20)이 사냥 검사(30)보다 앞이다. `Use` 의 `autofill.use`(10)는 메뉴 아이템보다 앞이다.
-- `Second` 에서 `progress.checkHeld`(20)는 `dimension.second`(60)보다 앞이다. 앞의 것이 플레이어가 든 아이템을 읽어 파트에 두고, 뒤의 것이 `RetireItems` 로 아이템을 정리한다. `progress.collection`(120)이 그 파트를 읽는다.
-- `Leave` 에서 `dimension.leave`(10)와 `expand.stop`(15)은 `core.parts`(20)보다 앞이다. `expand.stop` 은 `expanding` 항목에 둔 게임 모드를 되돌리므로 항목이 지워지기 전에 돈다. `core.parts` 는 `Players` 의 플레이어 항목을 지우므로, 그 뒤(`look.clear` 40 등)에 도는 구독자는 `Players` 에서 그 플레이어의 항목을 찾지 못한다. 플레이어별 데이터를 읽어야 하는 퇴장 단계는 20 앞에 둔다.
-- `Close` 에서는 구독자 뒤에 저장 파트 쓰기와 세션 등록 해제가 온다. 구독자는 세션이 아직 등록된 상태에서 돈다.
+- `StartBreak`, `BlockBreak`, `BlockPlace`, `UseOnBlock`: `expand.guard`(LOWEST)가 `island.guard`(LOW)보다 먼저다. 확장 모드에서 섬 밖을 누르면 확장 모드가 막고 지도를 다시 보여 준다. 월드를 바꾸는 단계는 모두 NORMAL 이후다. 가드는 `Handle` 로 멈추면서 `Cancel()` 도 한다.
+- `UseOnBlock`: 든 도구가 클릭을 가져가는 단계(`copier.use`, `wand.use`, LOW)가 블록에 하는 일(NORMAL, HIGH)보다 먼저다. 든 아이템을 쓰는 단계(`handwork.stick`, `fairy.time`, NORMAL)가 블록 자체의 동작(`farm.composter`, `farm.crop`, HIGH)보다 먼저다. 옆 칸 가드(`expand.guardSide`, `island.guardSide`, HIGHEST)는 아무도 클릭을 가져가지 않았을 때 마지막에 막는다.
+- `Sneak`: `expand.show`(LOWEST), `achievement.sneak`(LOW), `elevator.down`(NORMAL), `farm.crouchGrow`(HIGH) 순서다. 엘리베이터가 웅크리기를 쓰면 묘목은 자라지 않는다.
+- `Hurt`: `dimension.frozen`(LOWEST)이 먼저 막고, `dimension.fallen`(HIGHEST)이 남은 피해로 사냥꾼이 쓰러지는지 본다. `Move` 의 `island.void`(HIGHEST)도 마지막에 취소한다.
+- `ItemDamage`: `tinker.ironHammer`(LOW)가 피해를 0 으로 고친 뒤에 `autofill.broken`(HIGH)이 읽는다.
+- `BlockPlace`: `netstore.place`(HIGH)는 `machine.place`(NORMAL)가 블록 설치를 미뤄 둔 뒤에 망을 버린다.
+- `Leave`: 기능이 퇴장 때 하는 일(`dimension.leave`, `expand.stop`, NORMAL)이 클라이언트에 남은 것을 지우는 정리(`look.clear`, `screens.board`, `progress.recipes`, HIGHEST)보다 먼저다. 확장 모드가 끝나면 사이드바를 다시 그리고, 사냥이 끝나면 레시피를 다시 보낼 수 있다.
+- `Second`: `progress.checkHeld`(LOW)가 플레이어가 든 아이템을 읽어 파트에 둔 뒤에 `dimension.second`(NORMAL)가 `RetireItems` 로 아이템을 정리한다. `progress.collection`(MONITOR)이 그 파트를 읽는다.
+
+구독자의 조건이 지키는 것:
+
+- 사냥 중인 플레이어가 블록을 누르면 무엇을 들었든 `dimension.hunting` 이 막는다. `screens.menuItem`(`UseOnBlock`)과 `dimension.stone` 은 사냥 중이면 클릭을 가져가지 않는다. 공중에서는 메뉴 아이템이 사냥에서 나가는 길이라 `dimension.hunting`(`Use`)이 메뉴 아이템을 든 클릭을 막지 않는다.
+- 메뉴 아이템과 이차원석은 어디서나 자기 화면을 연다. `expand.guard`(`UseOnBlock`)는 그런 아이템(`sgitem.OpensScreen`)을 든 클릭을 막지 않는다.
+- `copier.use` 와 `wand.use` 는 섬이 짓기를 허락하는 곳(`island.CanBuild`)에서만 클릭을 가져간다. 섬 밖에서는 `island.guard` 가 막는다.
+
+커널이 지키는 것:
+
+- `Leave`: 플레이어 클라이언트의 도형 지우기와 `Players` 항목 지우기는 구독자가 아니다. `Lifecycle.Leave` 가 모든 구독자 뒤에 한다. 퇴장 구독자는 어느 티어에 서든 플레이어별 데이터를 읽을 수 있다.
+- `Second`, `Autosave`, `Close`, `NewDay` 의 꼬리 일도 구독자 뒤다. `Close` 의 구독자는 세션이 아직 등록된 상태에서 돈다.
+- `Move`: 이벤트의 `From` 은 이벤트를 만들 때 플레이어가 서 있던 곳이다. `island.void` 가 플레이어를 옮긴 뒤에 돌아도 `achievement.walk` 는 같은 거리를 센다.
+- 커널의 구독자는 `Join` 의 `core.shapes` 하나다(NORMAL). `core` 가 목록의 처음이라 그 티어에서 먼저 돌지만, 다른 구독자와 같은 규칙을 따른다.
+
+### 예외
+
+여섯 티어로 풀리지 않아 같은 티어 안의 자리(기능 목록 순서)나 역할에 맞지 않는 티어로 지키는 곳이다. 자리로 지키는 쌍은 루트의 `orderKept` 에 있고 `TestTierOrder` 가 그 쌍만 순서를 지킨 채 섞는다. 목록은 늘리지 않는다.
+
+- `UseOnBlock` 의 `storage.sneakUse`, `conduit.plate` 다음 `handwork.stick`(모두 NORMAL): `handwork.stick` 은 흙·잔디가 아닌 블록에 막대기를 쓴 횟수(`stick_other`)도 센다. 서랍 메뉴나 도관 판이 클릭을 가져가면 세지 않는다.
+- `UseOnBlock` 의 `conduit.plate` 다음 `fairy.time`(둘 다 NORMAL): 기계에 붙은 도관 판을 요정의 모래시계로 누르면 판 설정이 열리고 모래시계는 쓰이지 않는다.
+- `Sneak` 의 `achievement.sneak`(LOW): 세기만 하는 구독자라 역할은 MONITOR 다. 엘리베이터를 탄 웅크리기는 세고 섬 확장 모드의 웅크리기는 세지 않으려고 `expand.show` 와 `elevator.down` 사이에 둔다.
+- `Join` 의 `dimension.joined`(HIGH): 서버가 멈출 때 사냥 중이던 플레이어의 소지품을 되돌리는 일이라 역할로는 다른 입장 단계보다 먼저(LOW)다. 지금 순서를 지키려고 뒤에 둔다. 앞의 입장 단계가 그 플레이어의 인벤토리에 넣거나 고친 것(`fairy.backfillDust` 의 가루, `screens.menuItem` 이 고친 메뉴 아이템 이름)은 사냥 장비와 함께 사라진다.
 
 ## 기능의 데이터
 
 기능은 `Session` 에 필드를 더하지 않고 `core` 의 파트를 패키지 변수로 둔다.
 
 - `core.NewPart[T](mk)`: 세션마다 하나, 처음 `Of(s)` 를 부를 때 만든다. 저장하지 않는다. 예: `progress` 의 `held`.
-- `core.NewPlayers[T]()`: 세션 안에서 플레이어마다 하나. `Of(s)` 는 `map[*world.EntityHandle]T` 를 돌려준다. 플레이어가 나갈 때 `Leave` 의 `core.parts` 단계(`Session.Leave`)가 항목을 지운다. 예: `tinker` 의 `keptFood`.
+- `core.NewPlayers[T]()`: 세션 안에서 플레이어마다 하나. `Of(s)` 는 `map[*world.EntityHandle]T` 를 돌려준다. 플레이어가 나갈 때 `Lifecycle.Leave` 가 구독자가 모두 돈 뒤에 항목을 지운다(`Session.Leave`). 예: `tinker` 의 `keptFood`.
 - `core.NewSaved[T](mk)`: 저장하는 데이터. 모든 `Saved` 파트는 세이브 상태 문서(`skygear.state`) 하나의 JSON 필드이고, 필드 이름은 파트끼리 겹치지 않는다.
 
 ```go
@@ -425,8 +491,14 @@ func init() { coretest.Setup(farm.Feature) }
 | 테스트 지원 패키지(`core/coretest`, `machine/machinetest`)는 테스트 파일만 import 한다 | `TestSupportPackagesAreTestOnly` |
 | 패키지의 테스트는 루트를 import 하지 않는다(`wire` 는 예외). 내용 목록 아래 칸의 테스트는 `catalog`, `codex`, `screens` 를 import 하지 않는다. `core` 밖의 테스트는 `core.Install` 이나 기능의 `Setup` 을 직접 부르지 않고 `coretest.Setup` 을 쓴다 | `TestPackageTestsStayBelow` |
 | 플레이어 핸들러의 모든 메서드가 `core.PlayerHandler` 에 있고 훅마다 구독자가 있으며, 루트가 같은 이름의 메서드를 두지 않는다 | `TestHandlerMethodsOneWay` |
-| 핫 경로는 할당하지 않는다. 이벤트마다 `AllocsPerRun` 으로 0 을 확인한다 | `core.TestEmitAllocatesNothing`, `core.TestEventsEmitAllocateNothing` |
-| 구독자는 `Install` 동안에만 단다. 첫 `Emit` 뒤의 등록, 같은 훅 안의 이름·우선순위 겹침은 패닉이다 | `core.TestRegisterAfterEmitPanics`, `core.TestSamePriorityPanics`, `core.TestSameNamePanics` |
+| 핫 경로는 할당하지 않는다. 이벤트마다 `AllocsPerRun` 으로 0 을 확인한다. 멈춘 뒤 `Always` 구독자를 도는 길도 같다 | `core.TestEmitAllocatesNothing`, `core.TestEmitAllocatesNothingAfterCancel`, `core.TestEventsEmitAllocateNothing` |
+| 구독자는 `Install` 동안에만 단다. 첫 `Emit` 뒤의 등록, 같은 훅 안의 이름 겹침, 없는 티어는 패닉이다 | `core.TestRegisterAfterEmitPanics`, `core.TestSameNamePanics`, `core.TestNotATierPanics` |
+| 티어는 상수의 순서대로 돌고, 같은 티어 안에서는 등록한 순서대로 돈다 | `core.TestRunsInTierOrder`, `core.TestSameTierRunsInRegistrationOrder` |
+| 구독자는 같은 티어 안의 자리에 기대지 않는다. 티어 안의 순서를 뒤집고 섞어도 모든 테스트가 통과하고 장면의 결과와 업적 카운터가 같다 | `TestTierOrder` |
+| MONITOR 구독자는 멈추지도 취소하지도 가공하지도 않는다 | `core.TestHandleAtMonitorPanics`, `TestMonitorSubscribersDoNotShape` |
+| `Always` 구독자는 멈춘 이벤트도 받고 `cancelled` 로 구별한다. `On` 구독자는 받지 않는다. Dragonfly 의 취소만으로는 아무도 건너뛰지 않는다 | `core.TestAlways`, `core.TestAlwaysAfterCancelRunsOnlyAlways`, `core.TestCancelledTellsTheContext` |
+| `Leave` 의 구독자가 모두 돈 뒤에, 멈췄어도, 플레이어별 파트 항목을 지운다 | `core.TestLeaveDropsThePlayersEntriesLast` |
+| 구독자의 조건이 지키는 순서: 사냥꾼의 이차원석, 섬 밖의 복사기와 지팡이, 공허로 떨어진 걸음 | `TestHunterStoneOpensNothing`, `TestToolsLeaveTheClickOutsideTheIsland`, `TestWalkCountsTheMoveIntoTheVoid` |
 
 핫 경로(이동, 매초, 빠른 틱)를 건드리면 벤치마크로 재 본다. 이벤트 하나의 비용은 `core` 의 `BenchmarkEmit`, `BenchmarkEmitMove` 와 루트의 `BenchmarkHandleMove`, `BenchmarkSecond`, `BenchmarkUseOnBlock`, `BenchmarkUse` 가 잰다. 패널 하나의 비용은 루트의 `BenchmarkLookMachine`(시선 추적과 패널 보내기), `BenchmarkLookInfoMachine`, `BenchmarkLookInfoStone`(패널 만들기)이 잰다. 다른 기능이 동작을 등록한 기계의 틱 한 번은 `BenchmarkTickHarvester`, `BenchmarkTickDrawerController` 가 잰다.
 
@@ -443,7 +515,7 @@ go test -run xxx -bench Emit -benchmem -count 5 ./content/skygear/core
 
 | 파일 | 담는 것 | 테스트 |
 |---|---|---|
-| `content/skygear/testdata/pin/order.golden` | 모든 훅의 구독자를 도는 순서대로 `<훅> <우선순위> <이름>` | `TestPinOrder` |
+| `content/skygear/testdata/pin/order.golden` | 모든 훅의 구독자를 도는 순서대로 `<훅> <티어> <이름>`. `Always` 로 단 구독자는 끝에 ` always` 가 붙는다 | `TestPinOrder` |
 | `content/skygear/testdata/pin/handler-simple.golden`, `handler-blocks.golden`, `handler-use.golden` | 장면마다 이벤트가 거친 구독자 이름과 결과(취소 여부, 효과) | `TestPinSimple`, `TestPinBlocks`, `TestPinUse` |
 | `content/skygear/testdata/pin/lifecycle.golden` | 입장·퇴장·매초·빠른 틱·새 날·자동 저장·닫기·시작의 구독자 이름과 결과 | `TestPinLifecycle` |
 | `content/skygear/testdata/pin/settings.golden` | 설정이 등록된 순서 | `TestPinSettings` |
@@ -462,7 +534,22 @@ go test ./content/skygear -run 'TestOldSaveLoads' -count=1   # 옛 세이브 읽
 ```
 
 골든이 달라지면 테스트가 어느 줄(장면)이 어떻게 달라졌는지 `want` 와 `got` 로 보여 준다. 디스패치 순서나 클라이언트 데이터가 바뀌면 골든 차이로 나타난다.
-골든은 동작을 바꾸려는 변경에서만 다시 기록한다.
+
+장면 골든의 한 줄은 `<장면>: <거친 구독자 이름> | <결과>` 다. `|` 앞은 등록한 순서로 돌았을 때의 자취이고 뒤가 결과다.
+`TestTierOrder` 는 루트의 테스트 바이너리를 여섯 번 다시 돌린다. 한 번은 등록한 순서 그대로, 다섯 번은 모든 훅의 구독자를 티어 안에서 뒤집거나 섞어서다(`SKYGEAR_HOOK_ORDER=reverse`, `shuffle:1` 부터 `shuffle:4`). 섞은 실행에서 확인하는 것은 셋이다.
+
+- 루트의 모든 테스트가 통과한다. 장면 골든은 `|` 뒤의 결과만 견주고, `order.golden` 은 한 훅의 한 티어에 든 줄을 집합으로 견준다.
+- 장면 234개의 결과가 골든과 글자 하나까지 같다.
+- 장면마다 플레이어의 업적 카운터가 늘어난 양이 등록한 순서로 돈 실행과 같다. 카운터는 골든에 없어서 따로 견준다.
+
+플레이어가 받는 팁·글·소리와 어느 장면도 읽지 않는 상태는 이 테스트가 보지 못한다. "예외"의 쌍(`orderKept`)은 섞어도 순서를 지킨다.
+기능 패키지의 테스트도 같은 변수로 섞어서 돌릴 수 있다. `coretest.Session` 이 처음 불릴 때 섞는다.
+
+```sh
+go test ./content/skygear -run TestTierOrder -count=1 -v
+SKYGEAR_HOOK_ORDER=reverse go test ./content/skygear/... -count=1
+```
+골든은 동작을 바꾸려는 변경에서만 다시 기록한다. 같은 티어에 구독자를 더하면 그 훅을 거치는 장면의 자취(`|` 앞)가 달라지므로 그때도 다시 기록하고, `|` 뒤가 그대로인지 `git diff` 로 본다.
 
 ```sh
 SKYGEAR_GOLDEN=update go test ./content/skygear -run 'TestPin' -count=1
~~~~
