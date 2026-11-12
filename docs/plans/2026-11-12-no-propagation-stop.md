# No propagation stop: reference (measurements, the 31 subscribers, behaviour changes, design, the code of each task)

Reference for the machine plans `docs/plans/2026-11-12-no-propagation-stop-plan-a.md` (T130 to T134) and `docs/plans/2026-11-12-no-propagation-stop-plan-b.md` (T135 to T139). Not a machine plan, not linted. Everything here was measured or built in a scratch worktree at 5a8815b; the per-task prototype patches are in `/tmp/claude-0/-home-user-SkyGear/a71fa8d5-6f20-5c3f-ba20-eeb27e13ad8a/scratchpad/proto-nostop-tasks/` (`T130.patch` to `T139.patch`, in id order on that commit, and `combined.patch`).

## 1. The decision and the frame

The owner's decision, in the owner's words: a "propagation stop" that handles the stop-without-cancel case is not the flow I want. Making other things not work because one thing worked makes later handling harder. As far as possible go in one of two directions: a subscriber says precisely "only in this state do I handle this, and I cancel" (어떤 상태에서만 처리하고 취소), or "I do my function but do not cancel the event when cancelling is not necessary" (기능은 처리하되 불필요하면 취소하지 않음).

Read as the traditional event model:

- No subscriber can stop the walk. `Handle` and the stop go away. Every subscriber of a hook is called, tier by tier.
- The only thing that keeps a subscriber from being called: the event is cancelled and the subscriber did not opt in to cancelled events (the handleCancelled option). "Cancelled" is the event's own cancel state: the Dragonfly context of the event, cancelled by an earlier subscriber or by Dragonfly.
- A subscriber acts by its own conditions (its state, the item in hand, the block, the mode the player is in), not by "nobody before me took it". It cancels only when the vanilla action or the later default subscribers must not happen.

Two confirmations of the owner that arrived while this was built, both folded in:

1. The handleCancelled model is as first stated: a subscriber that does not give the option is NOT called once the event is cancelled (the default); a subscriber opts in. The option is named after the tradition.
2. Subscribers of one tier have NO guaranteed order, and everything is handled on that premise. `orderKept` goes away: the plan ends with no kept pair. A same-tier pair in which one cancels and the other is a default subscriber that could act on the same event is order-dependent by definition: such pairs are exclusive by their own conditions or stand in different tiers.

One point of the coordinator's frame is argued against by a measurement, and changed: "the handleCancelled subscriber keeps a callback that receives the cancel state". With the stop gone, the cancel state is the event's own (`e.Cancelled()`), which a subscriber reads from the event it is handed. A second callback shape (`func(e E, cancelled bool)`) would repeat that state in a parameter, needs a wrapper closure per subscriber and a second walk in `Emit`, and its one user (`look.update`) ignores the flag. The option is therefore an argument of `On`, and the callback is the same `func(E)` (section 9).

One point of the expected task order is changed by a measurement: the behaviour changes come BEFORE the kernel switch, not after it (section 3.3).

## 2. Measurement at 5a8815b

121 subscribers: 31 `Handle`, 89 `On`, 1 `Always` (`look.update`). After the plan: 123 (`storage.front` and `handwork.stickOther` are new), all `On`, one of them with the option.

| What a subscriber does today | Count | Who |
|---|---|---|
| stops and cancels on every path that returns true | 29 | the rows of section 3.1 marked A |
| stops without cancelling | 2 | `Sneak`: `expand.show`, `elevator.down` |
| cancels without stopping (`On`) | 5 registrations | `storage.break` (a drawer hit on its front), `island.void`, `dimension.fallen`, `fairy.touch` ×2 |
| reads `e.Cancelled()` | 0 | |

What this means for the new model, checked by running the prototype (not by reading alone):

- For the 29, "stops and cancels" and "cancels, and the later default subscribers are skipped" are the same thing: with the kernel switched and the 29 changed from `Handle … return true` to `On … e.Cancel()`, every one of the 234 scenario lines of the four scenario goldens is byte for byte the same, TRACE INCLUDED, and so are the achievement counters and the inventories of every scenario (the dump of section 10). No later subscriber needed the option to keep what it does, except `look.update`, which has it today.
- The 2 on `Sneak` have nothing to fall back on: nothing cancels a sneak (cancelling the toggle keeps the player from sneaking). Each `Sneak` subscriber needs its own condition (section 4).
- The 5 change by definition: today the subscribers after them run, in the new model the default ones do not (section 5).
- Three same-tier pairs relied on the registration order (section 6); they are resolved, none is kept.

## 3. The 31 former `Handle` subscribers

### 3.1 The table

A = "handles in its state and cancels": the cancel stays; the later default subscribers are skipped because the event is cancelled. B = "acts and does not cancel".

"After it" is what runs after the subscriber acted. Today: nothing, it stops (on `Move` the `Always` subscriber `look.update`). New model: for A the same (nothing but a handleCancelled subscriber, which is `look.update` on `Move` and nobody elsewhere); for B every later subscriber.

| # | Hook | Subscriber | Tier | The state it reacts to (its own condition) | Choice | Cancel before → after | Why | Condition or change added |
|---|---|---|---|---|---|---|---|---|
| 1-3 | StartBreak, BlockBreak, BlockPlace | expand.guard | LOWEST | the player is in island expansion mode | A | yes → yes | the mode refuses building; the client's predicted block must be put back | none |
| 4 | UseOnBlock | expand.guard | LOWEST | expansion mode, and the held item is not one that opens its screen (`sgitem.OpensScreen`) | A | yes → yes | as above | none |
| 5 | UseOnBlock | expand.guardSide | HIGHEST | expansion mode (never reached: 4, 17 or 19 cancelled before) | A | yes → yes | unchanged; see 12.3 | none |
| 6-8 | StartBreak, BlockBreak, BlockPlace | island.guard | LOW | the block is not the island's to build on (`!island.CanBuild`) | A | yes → yes | no building outside the island | none |
| 9 | UseOnBlock | island.guard | LOW | as 6 | A | yes → yes | as 6 | none |
| 10 | UseOnBlock | island.guardSide | HIGHEST | the block next to the clicked face is not the island's | A | yes → yes | a block would be placed outside | none |
| 11 | UseOnBlock | dimension.hunting | LOWEST | the player hunts | A | yes → yes | a hunter only moves | none |
| 12 | Use | dimension.hunting | LOWEST | the player hunts and does not hold the menu item | A | yes → yes | as 11 | none |
| 13 | UseOnEntity | dimension.hunting | LOWEST | the player hunts | A | yes → yes | as 11 | none |
| 14 | ItemDrop | dimension.hunting | LOWEST | the player hunts | A | yes → yes | the kit is not dropped | none |
| 15 | AttackEntity | dimension.hunter | LOWEST | the target is a monster of the hunt, or the player hunts | A | yes → yes | the hunter's weapons strike on their own | none |
| 16 | Hurt | dimension.frozen | LOWEST | the hunter's hunt waits for its upgrade choice | A | yes → yes | no damage while the hunt waits | none |
| 17 | UseOnBlock | dimension.stone | LOWEST | holds the dimension stone and does not hunt | A | yes → yes | the block under the click must not be activated | none |
| 18 | Use | dimension.stone | LOWEST | as 17 | A | yes → yes | kept as today; the cancel is not needed for vanilla (12.1) | none |
| 19 | UseOnBlock | screens.menuItem | LOWEST | holds the menu item and does not hunt | A | yes → yes | as 17 | none |
| 20 | Use | screens.menuItem | LOWEST | holds the menu item | A | yes → yes | as 18 | none |
| 21 | UseOnBlock | copier.use | LOW | holds the configuration copier where the island allows building | A | yes → yes | the tool uses the click; the block must not open | none |
| 22 | UseOnBlock | wand.use | LOW | holds a construction wand where the island allows building | A | yes → yes | as 21 | none |
| 23 | UseOnBlock | storage.sneakUse | NORMAL | sneaking, a drawer, a held item that is no block, the island allows building | A | yes → yes | the held item must not be used on the drawer | none |
| 24 | UseOnBlock | conduit.plate | NORMAL | not sneaking, the click is on a connector plate, the held conduit does not go into that bundle | A | yes → yes | the block behind the plate must not open | none |
| 25 | UseOnBlock | handwork.stick | NORMAL | a stick on grass or dirt; a sneaking stick on a chest | A | yes → yes | kept as today (12.1); the side guard must not also refuse the tap with its tip | the record `stick_other` leaves it and becomes `handwork.stickOther`, MONITOR (T135) |
| 26 | UseOnBlock | farm.composter | HIGH | not sneaking, a composter that is full or takes the held plant item | A | yes → yes | the held item must not be placed or used | none |
| 27 | UseOnBlock | fairy.time | NORMAL → LOW | holds fairy bone meal on something it grows, a fairy hourglass on a machine it can wind, or plain bone meal on a timed crop while sneaking | A | yes → yes | the item is spent; vanilla bone meal must not act | `island.CanBuild(e.S, e.Pos)`, the tier of the held tools (T136) |
| 28 | UseOnBlock | farm.crop | HIGH | not sneaking, a crop | A | yes → yes | nothing is placed on a crop | none |
| 29 | Move | expand.hold | LOWEST | expansion mode | A | yes → yes | the body stays where the mode put it | none |
| 30 | Sneak | expand.show | LOWEST → NORMAL | expansion mode | B | no → no | the sneak is the mode's key for "show the screen again"; nothing vanilla is to be prevented, and a cancelled toggle would leave the player not sneaking | none of its own; the others say what they lack (T131, T132) |
| 31 | Sneak | elevator.down | NORMAL | the player stands on an elevator that has one below, the cooldown passed | B | no → no | the ride needs no cancel | not for a player a takeover holds: `core.Takeover(e.S, e.P) == ""` (T131) |

29 A, 2 B. Every A keeps a cancel it has today; no cancel is added and none is dropped.

### 3.2 What runs after each, checked

- A (rows 1 to 29): the scenario traces of the prototype after the switch equal the traces before it, line for line (0 of 234 traces changed in T138). So "what runs after it" did not change for any A in any scenario: the later default subscribers are skipped exactly where the stop skipped them. The order guard confirms it under every permutation.
- B, row 30 (`expand.show`): after it run `elevator.down` (does nothing: a takeover holds the player; and the mode moved the body to the map, away from any elevator), `farm.crouchGrow` (does nothing: same condition) and `achievement.sneak` (counts: behaviour change C1). Trace `Sneak/expanding`: `expand.show elevator.down farm.crouchGrow achievement.sneak`.
- B, row 31 (`elevator.down`): after a ride run `farm.crouchGrow` (acts where the player arrived: behaviour change C2) and `achievement.sneak` (counted today as well). Trace `Sneak/elevator`: `expand.show elevator.down farm.crouchGrow achievement.sneak`.

### 3.3 Why the behaviour changes come before the switch

The expected order was: conditions, the switch with results byte for byte, then the behaviour changes. Measured: every behaviour change that the new model forces can be written on TODAY's kernel, as a small task with its own test:

- "a cancel also keeps the later subscribers away" is today's `Handle` returning true next to the cancel (`island.void`, `dimension.fallen`, the drawer's front);
- "nothing stops a sneak" is today's `On`;
- the three same-tier pairs are tier moves and one new subscriber.

Done in that order, the switch itself (T138) changes nothing a test can see: results AND traces of all four scenario goldens byte for byte, the dump of counters and inventories equal line for line before and after. That is a far stronger proof for a commit that rewrites the kernel and 33 call sites than "results only", and it is why the order was changed.

The other way round does not exist: for the cancel-without-stop cases there is no form in the new kernel that keeps today's behaviour (section 5), so with the switch first those changes would have to be inside the big commit.

What "accept or drop" means for each change is said per change in section 7: the forced ones can only be traded for the alternative named there.

## 4. The hooks without a cancel

15 hooks have an event that cannot be cancelled: `Jump`, `Death`, `Respawn` (Dragonfly passes no context) and the session and world hooks `Load`, `Start`, `Join`, `Leave`, `NewDay`, `Second`, `Fast`, `Autosave`, `Close`, `ColumnLoad`, `ColumnUnload`, `EntityDespawn`. On these every subscriber is always called; `Hook.On` takes no option and the events have no `Cancel` (section 9). `Sneak` can be cancelled but nobody cancels it, so in practice it belongs here too.

Each was checked for same-tier pairs that work only by order (by reading, and by the guard with the inventories of section 10):

| Hook | Subscribers by tier | Same-tier pairs | Result |
|---|---|---|---|
| Sneak | NORMAL `expand.show`, `elevator.down`; HIGH `farm.crouchGrow`; MONITOR `achievement.sneak` | `expand.show` / `elevator.down` | exclusive: the first acts only in expansion mode, the second only when no takeover holds the player (T131). `farm.crouchGrow` reads where the player is after a ride: another tier (HIGH), not an order inside one. `achievement.sneak` counts every sneak (T132). |
| Jump | NORMAL `elevator.up`; MONITOR `achievement.jump` | none | see 12.3 for a jump in a mode |
| Death | NORMAL `tinker.keepFood`, `dimension.died`; MONITOR `achievement.death` | `keepFood` / `died` | independent: the run's end does not touch the food level (guard: reversed and shuffled, same results) |
| Respawn | LOW `dimension.respawned`; NORMAL `tinker.restoreFood` | none | |
| Load, NewDay, ColumnLoad, ColumnUnload, EntityDespawn | one subscriber each | none | |
| Start | NORMAL `island.keepLoaded`, `climate.apply` | one | independent (chunks kept loaded; the weather applied) |
| Join | LOWEST `screens.menuItem`; NORMAL `core.shapes`, `progress.sendRecipes`, `island.purgeLost`, `climate.showWeather`, `fairy.backfillDust`; HIGH `dimension.joined`; MONITOR `screens.board` | the five of NORMAL | only `fairy.backfillDust` touches the inventory (`island.purgeLost` prunes the list of lost items, it gives nothing); the others send packets. No pair works by order. The slot of the menu item is a tier separation (LOWEST) and stays. `dimension.joined` after them is a tier, not an in-tier order: E4, section 8. |
| Leave | NORMAL `dimension.leave`, `expand.stop`; HIGHEST `progress.recipes`, `look.clear`, `screens.board` | two groups | NORMAL: exclusive, a player is in one takeover at most. HIGHEST: three independent clean-ups (recipe filter, panel, scoreboard). |
| Second | LOW `progress.checkHeld`; NORMAL 8; MONITOR `achievement.playSeconds`, `progress.collection`, `screens.boards` | NORMAL, MONITOR | NORMAL: separate state per feature (they share the session's random source: the draws differ with the order, no result depends on which feature draws first). MONITOR: the board shows the clock and the quest rows, neither of which the other two write. |
| Fast | NORMAL `network.step`, `dimension.watchdog`; MONITOR `look.updateAll` | one | independent |
| Autosave | NORMAL `conduit.shares`, `netstore.flush` | one | independent (separate saved parts) |
| Close | NORMAL `conduit.close`, `storage.displays`, `netstore.flush` | three | independent |

## 5. The cancel-without-stop registrations

Today the subscribers after them still run. In the new model a default subscriber after them is skipped; to keep running it would need handleCancelled, and then it would also run after every OTHER cancel of its hook (the guards), which it does not today. No subscriber can tell who cancelled. So "keep exactly today's behaviour" does not exist for these in the new model; the choice per later subscriber is "skipped" or "sees every cancelled event".

| Registration | What runs after its cancel today | New model | Wanted? |
|---|---|---|---|
| `storage.break` (BlockBreak, NORMAL): a drawer hit on its front | same tier: `tinker.break` (ran before or after it by registration order: tool experience, reinforcement, a hammer's vein mining), `machine.break`, `conduit.break`, `farm.break`, `netstore.break` (none acts on a drawer); MONITOR `achievement.break` (counts `block_break`) | the refusal becomes its own subscriber `storage.front` in LOW (a cancel in NORMAL next to `tinker.break` would be order-dependent); every later default subscriber is skipped | yes: the earlier reference called "the break is cancelled, yet it is counted, the tool gains experience and a hammer still vein-mines" a defect. Change C4. |
| `island.void` (Move, HIGHEST) | MONITOR `achievement.walk` (adds the horizontal part of the cancelled move), `look.update` (`Always`) | `achievement.walk` is a default subscriber: skipped. `look.update` has handleCancelled: runs | yes: a cancelled move is no walk. Change C5. Alternative: handleCancelled on the walk counter, which then also counts the moves island expansion mode refuses. |
| `dimension.fallen` (Hurt, HIGHEST): the hit that would kill a hunter | MONITOR `achievement.hurt` | skipped | yes: the hit is cancelled, the hunter takes no damage; the run's end is recorded on its own (`wave_fail`). Change C6. Alternative: handleCancelled on the hurt counter, which then also counts every hit a frozen hunter does not take. |
| `fairy.touch` (UseOnEntity, NORMAL) | nothing: it is the last subscriber of the hook | nothing | no change |
| `fairy.touch` (AttackEntity, NORMAL) | nothing | nothing | no change |

No later subscriber takes handleCancelled because of these. The option keeps its one user, `look.update`.

## 6. Pairs that were order-dependent inside a tier, and how each was resolved

The kept list (`orderKept`) is empty and the mechanism is removed (`coretest.ReorderHooks()` takes no pairs; T136). `TestTierOrder` runs under every permutation with no exception.

| Pair (tier) | Why the order mattered | Resolution | Seen by a player |
|---|---|---|---|
| `storage.sneakUse` / `handwork.stick` (UseOnBlock NORMAL), was E2 | `handwork.stick` counted `stick_other` as a side effect when a click reached it and it took nothing; the drawer menu, registered first, stopped before it | the record is split out: `handwork.stickOther`, MONITOR, default delivery. `handwork.stick` no longer acts on anything but grass, dirt and (sneaking) a chest, so it is exclusive with the drawer by the block (T135) | C7 |
| `conduit.plate` / `handwork.stick` (UseOnBlock NORMAL), was E2 | as above, for a stick on a connector plate | as above; a plate is never on grass or dirt | C7 |
| `conduit.plate` / `fairy.time` (UseOnBlock NORMAL), was E3 | both can take an hourglass click on the plate of a machine | by role: `fairy.time` joins the held tools in LOW with the condition `island.CanBuild`, which makes it exclusive with `island.guard` (LOW, acts where `!CanBuild`); against `copier.use` and `wand.use` the held item decides (T136) | C8 |
| `machine.noteFace` / `conduit.plate` (UseOnBlock NORMAL), new finding | `noteFace` writes a note for the next placement; with a machine item in hand on a plate the note was written or not by registration order (written today). A cancel in the same tier next to a default subscriber that acts on the same click | by role: the note belongs to the click that goes through to the placement. `machine.noteFace` moves to MONITOR, default delivery: it is called when nobody cancelled (T137) | C9 (latent) |
| `storage.break` / `tinker.break` (BlockBreak NORMAL), new finding | the drawer's front cancel stood in the tier of the tool's work | the refusal is `storage.front` in LOW; in LOW it is exclusive with `island.guard` by `island.CanBuild` (T134) | C4 |
| `expand.show` / `achievement.sneak` / `elevator.down` (Sneak), was E1 | a tier was bent (the counter in LOW) to stand between two stops | no stop: the counter is in MONITOR and counts every sneak; `expand.show` and `elevator.down` share NORMAL and exclude each other by `core.Takeover` (T131, T132) | C1 |

Checked and exclusive by their own conditions, nothing to change (every remaining same-tier group in which one subscriber cancels):

- UseOnBlock LOWEST `dimension.hunting`, `dimension.stone`, `expand.guard`, `screens.menuItem`: hunter / not hunter with the stone / expansion mode without a screen item / not hunter with the menu item; a player is in one takeover at most.
- UseOnBlock LOW `island.guard`, `fairy.time`, `copier.use`, `wand.use`: `!CanBuild` / `CanBuild` and the held item.
- UseOnBlock NORMAL `conduit.plate`, `storage.sneakUse`, `handwork.stick`, `handwork.gather` (never cancels): standing on a plate / sneaking on a drawer / stick on grass, dirt, sneaking on a chest / axe on a log, hoe on dirt.
- UseOnBlock HIGH `farm.composter`, `farm.crop`: the block.
- UseOnBlock HIGHEST `island.guardSide`, `expand.guardSide`: the second never acts (12.3).
- BlockBreak LOW `island.guard`, `storage.front`: `!CanBuild` / `CanBuild`.
- Use LOWEST `dimension.hunting`, `dimension.stone`, `screens.menuItem`: hunter without the menu item / not hunter with the stone / the menu item.
- StartBreak, BlockPlace, UseOnEntity, AttackEntity, Move, ItemDrop, Hurt: every canceller is alone in its tier.

## 7. BEHAVIOUR CHANGES

What a player can see or an achievement counts. "Forced" means the owner's direction leaves no form that keeps today's behaviour; the alternative is the other form the direction allows. Each is a task of its own with its own test; the scenario goldens change in their TRACE part only. NO RESULT PART (after ` | `) of any scenario line changes in this plan: none of the changed behaviours is in a result part (they are counters, tool experience, a wound machine, a note); each task's test asserts its change instead, and the dump of the order guard shows the counter changes (given per change).

| Id | Situation | Before | After | Caused by | Task | Forced? Alternative |
|---|---|---|---|---|---|---|
| C1 | a player in island expansion mode presses sneak | the expansion screen shows again; the sneak is not counted | the screen shows again; the sneak counts toward the sneak achievements | no stop (decision 2 of the brief; E1 disappears) | T132 | forced: without a stop the counter has no condition of its own that excludes a mode. Alternative: `core.Takeover(e.S, e.P) != "expand"` in the counter (a record that names another feature's mode; also not counting a hunter's sneaks would be a new change). Dump: `Sneak/expanding` counters `-` → `sneak+1`. |
| C2 | a player sneaks on an elevator and rides down next to saplings | the ride; the saplings get no growth chance from that sneak | the ride; the saplings around the place the player ARRIVES at get the usual 30 percent chance each | no stop: `elevator.down` acts and does not cancel | T132 | forced. Alternative: `farm.crouchGrow` must learn that the sneak rode (it cannot ask `elevator`: sibling features; it would need a new kernel seam). Not built. |
| C3 | (none visible) `elevator.down` and `farm.crouchGrow` leave a player a takeover holds | in expansion mode neither ran (the stop); a hunter or a watcher could in theory ride or grow | in a mode neither acts | a condition replaces the stop | T131 | behaviour-preserving in every reachable state: the modes move the body to the map or the hunting ground, where there is no elevator and no sapling. Listed for completeness. |
| C4 | a player hits a drawer's front until the break completes (the break is cancelled, items came out) | `block_break` counts; a tinker tool in hand gains experience and may reinforce; a hammer vein-mines from the drawer | nothing but the cancel | cancel means cancelled (decision 3) | T134 | forced, and the wanted outcome (a defect before). Alternative: none that keeps the count without counting every refused break. Dump: `BlockBreak/drawerFront` counters `block_break+1` → `-`. |
| C5 | a player falls below the island | the move that crosses the void line adds its horizontal distance to the walk counter (under 4 blocks, usually well under 1) | that move is not counted | cancel means cancelled | T133 | forced. Alternative: `core.HandleCancelled` on `achievement.walk`: counts it as today and also counts moves expansion mode refuses. No scenario's dump changes (`Move/void` moves 0 blocks horizontally); `TestMoveIntoTheVoidIsNotWalked` asserts it. |
| C6 | a hunter takes the hit that would kill it (the run ends, the hunter goes back unharmed) | `hurt` counts | `hurt` does not count; `wave_fail` counts as before | cancel means cancelled | T133 | forced. Alternative: `core.HandleCancelled` on `achievement.hurt`: counts it and every hit a frozen hunter does not take. Dump: `Hurt/fallen` counters `hurt+1,wave_fail+1` → `wave_fail+1`. |
| C7 | a stick is used on a block that is neither grass nor dirt | `stick_other` counts unless the drawer's sneak menu or a connector plate took the click; it counts when a crop, a full composter or the side guard takes the click afterwards | `stick_other` counts when nobody cancelled the click: not for the drawer menu or a plate (as before), and no longer for a crop (not sneaking), a full composter, or a click the side guard refuses | no order inside a tier (confirmation 2; E2) | T135 | a judgment between two forms, both order-free. Built: the record is a default MONITOR subscriber ("a stick tap that went through"). Alternative: `core.HandleCancelled` plus `core.Takeover(e.S, e.P) == ""`: counts every stick tap on another block, also when the drawer menu or plate settings opened and when the island's guard refused it (three new counts instead of three lost ones). The earlier recommendation was the alternative; see 12.2. The same click sent twice within 100 ms after a tap another feature took is no longer debounced by that tap. No existing scenario's dump changes; `TestStickOtherCountsTheTapNothingTook` asserts it. |
| C8 | a fairy hourglass is used on the centre of a machine's face where a connector plate sits | the plate's settings open; the hourglass is not used | the hourglass winds the machine (and is spent); the plate's settings open only when there is nothing to wind | no order inside a tier (E3): the held tool is in LOW | T136 | a judgment; the earlier recommendation. Alternative: none found that keeps "plate first" without a kept pair (the plate in LOW would collide with the copier and the wand; `fairy.time` in HIGH collides with `farm.crop`). No scenario changes; `TestHourglassWindsTheMachineBehindAPlate` asserts it. |
| C9 | a player holds a network attachment or a miner, clicks something that takes or refuses the click (a plate, a crop, a full composter, the side guard), and the next machine is placed without a click of its own (the wand places it) | the placed attachment or miner faces by the stale click | it faces its default | no order inside a tier: the note is a result (MONITOR) | T137 | latent: every hand placement notes its own click. Alternative: none needed. `TestRefusedClickNotesNoFace` asserts it. |

Not changed although the direction would allow it (kept by choice A, section 3.1; the owner may decide otherwise, 12.1): the cancels that vanilla does not need (the menu item and the stone in the air, a stick on grass or dirt).

## 8. The exceptions E1 to E4 and `orderKept`, re-judged

| | Was | Now |
|---|---|---|
| E1 `Sneak`: the counter in LOW between two stops | a bent tier | GONE. No stop: `achievement.sneak` is in MONITOR and counts every sneak (C1). |
| E2 `storage.sneakUse`, `conduit.plate` before `handwork.stick` | two kept pairs | GONE, solved by splitting the record out (`handwork.stickOther`, MONITOR): C7. The form of the record is the owner's to confirm (12.2). |
| E3 `conduit.plate` before `fairy.time` | one kept pair | GONE, solved by role: `fairy.time` in LOW with `island.CanBuild` (C8). |
| E4 `Join`: `dimension.joined` in HIGH | a bent tier (its role is LOW) that keeps today's loss for a hunter the server stopped under | REMAINS, unchanged. It is not an order inside a tier and the new model does not touch `Join` (nothing to cancel), so nothing forces it; it is a further judgment. Options: (1) LOW: the stranded hunter keeps the backfilled fairy dust; the restored menu item still keeps its old name, because `screens.menuItem` (LOWEST) ran before the restore. (2) as it is. Recommendation: (1), as its own task with a scenario of a stranded hunter (none exists; the state needs a second session over the same save). Not in this plan. |
| `orderKept` | 3 entries | empty; the variable, the `kept` parameter of `coretest.ReorderHooks` and the code that put pairs back are removed (T136). |

## 9. Design

### 9.1 API

```go
// an event that cannot be cancelled: every subscriber is called
type Hook[E any]
func NewHook[E any](name string) *Hook[E]
func (h *Hook[E]) On(prio Priority, name string, f func(E))
func (h *Hook[E]) Emit(e E)

// an event that can be cancelled
type Cancellable interface { Cancel(); Cancelled() bool; cancelState() *world.Context }
type CancelHook[E Cancellable] struct{ /* a Hook[E], not exported: only On and Emit below are its methods */ }
func NewCancelHook[E Cancellable](name string) *CancelHook[E]
func (h *CancelHook[E]) On(prio Priority, name string, f func(E), opts ...Option)
func (h *CancelHook[E]) Emit(e E)

type Option uint8
const HandleCancelled Option = 1

type Who struct { S *Session; P *player.Player; Tx *world.Tx }   // no Cancel
type Actor struct { Who; Ctx *player.Context }
func (a Actor) Cancel()
func (a Actor) Cancelled() bool

type Subscription struct { Hook, Name, Feature string; Tier Priority; HandleCancelled bool }
```

- The default subscriber: `core.Move.On(core.Monitor, "achievement.walk", f)`. It is not called for a cancelled event.
- The handleCancelled subscriber: `core.Move.On(core.Monitor, "look.update", f, core.HandleCancelled)`. Same function type; it reads `e.Cancelled()` if it cares.
- Cancelling: `e.Cancel()`. Reading: `e.Cancelled()`.
- `Emit` returns nothing. The handler in `core/handler.go` needs nothing from it: Dragonfly reads the context it passed. The "stopped" meaning is gone everywhere (no `bool` result, no `Handle`, no `Always`, no second walk).

Why the option is an argument of `On`, named `HandleCancelled`:

- It is the tradition's word (PocketMine's `@handleCancelled`, the mirror of Bukkit's `ignoreCancelled`), which the owner used; a reader who knows either recognises it at the call site.
- It is a property of the registration, like the tier, not another verb. A method `HandleCancelled(...)` or `OnCancelled(...)` reads as "called only for cancelled events", which is wrong.
- The common call stays as short as today (three arguments).
- The function type is the same, so a subscriber changes its delivery by adding or removing one argument, and `Emit` has one loop and no wrapper closure.

Which events can be cancelled, and how the compiler knows:

- Cancellable (20 hooks, `CancelHook`): `StartBreak`, `BlockBreak`, `BlockPlace`, `UseOnBlock`, `UseOnEntity`, `AttackEntity`, `Use`, `Move`, `Sneak`, `Sprint`, `ItemDamage`, `ItemConsume`, `ItemPickup`, `ItemDrop`, `Hurt`, `PunchAir`, `Chat`, `BlockPick`, `HeldSlot`, `Command`. Their events embed `Actor`, which carries the Dragonfly context and has `Cancel` and `Cancelled`.
- Not cancellable (15 hooks, `Hook`): `Jump`, `Death` (event `NoticeEvent{Who}`), `Respawn` (`RespawnEvent{Who; Pos}`), and the 12 session and world hooks. Their events have no `Ctx`, no `Cancel` and no `Cancelled`: `e.Cancel()` on one does not compile. `Hook.On` has no option parameter: `core.Join.On(core.Normal, "x.y", f, core.HandleCancelled)` does not compile either. `NewCancelHook[Tick]` does not compile (`Tick` is not `Cancellable`).
- Before, `Jump`, `Death` and `Respawn` carried an `Actor` with a nil `Ctx`, and `Cancel()` silently did nothing: the run-time surprise the brief names. `core.TestOnlyActorEventsCancel` pins the split.
- An event built by a test without a context (`Actor{}`) is never cancelled: `Cancel` does nothing, `Emit` calls everybody.

### 9.2 `Emit`

`Hook.Emit` is the old loop without the `handle` case. `CancelHook.Emit` asks the event once for the place its cancel state lives (`cancelState()`, one call through the type parameter's dictionary per `Emit`), then per step reads that one bool directly and skips a default subscriber when it is set:

```go
c := e.cancelState()
...
steps := h.hook.steps
for i := range steps {
	s := &steps[i]
	if c.Cancelled() && !s.cancelled {
		continue
	}
	...
	s.f(e)
}
```

A skipped subscriber is not traced. Nothing allocates (`TestEmitAllocatesNothing`, `TestEmitAllocatesNothingAfterCancel`, `TestEventsEmitAllocateNothing`). The step struct lost a function pointer (48 bytes).

### 9.3 Monitor

A Monitor subscriber never cancels and never changes the event; with `HandleCancelled` it sees cancelled events. Kept: `TestMonitorSubscribersDoNotShape` (reads the function literal for `Cancel` calls and writes through the event's pointers). The reviewer's finding was that it reads the literal only. Improved cheaply for the cancel half: while `core.Trace` is set, which is exactly while a test traces (every pin scenario and every test that sets it), `CancelHook.Emit` panics when a Monitor subscriber cancelled the event, whatever helper did it (`core.TestMonitorCancelPanicsWhileTraced`). It costs nothing outside tests (the check is on the traced path only). Writes through the event's pointers below the literal are still not seen (kept in the list of open work).

`Handle` at `Monitor` no longer exists, so `TestHandleAtMonitorPanics` goes with it.

### 9.4 Inside a tier: no order, and what the kernel says about it

The kernel's comments and `docs/tech/extension.md` stop promising an order inside a tier: "the subscribers of one tier have no order among themselves; nothing may be written against the order they happen to run in". What happens is unchanged (registration order: the features in the order of `features.go`), because a repeatable run is worth having: the scenario traces are goldens.

`order.golden` follows: the lines of one tier of one hook are sorted by name, in the file and in the comparison, in every run. The file no longer changes when the feature list is reordered, and it no longer suggests an order to rely on.

A mode that shuffles inside tiers by default in tests was considered and NOT built: the trace part of 234 scenario lines is compared in the order of registration, so a default shuffle would end those goldens or need a second, unshuffled mode for them, and a failure would depend on a seed the developer has to carry around. Instead the shuffles are a test of their own that always runs (`TestTierOrder`: reverse and four seeds, every root test, plus the dump of section 10), and the same variable shuffles any package on demand: `SKYGEAR_HOOK_ORDER=shuffle:7 go test ./content/skygear/...`.

### 9.5 The `Leave` tail and `core.Takeover`

Both stay. `Lifecycle.Leave` clears the shapes and drops the player's entries after `Emit`; nothing can stop the walk before it any more, so the remark "also after a stop" is gone. `core.Takeover` is the seam for "the player is held by another mode": used by `elevator.down` and `farm.crouchGrow` (T131).

## 10. The guard

`TestTierOrder` runs the root test binary six times (the order of registration, `reverse`, `shuffle:1` to `shuffle:4`), with no kept pair, and compares per scenario line:

1. every root test passes in every run (in a reordered run the pin tests compare the result part of each scenario line);
2. the achievement counters the scenario's players gained (as before);
3. NEW (T130): what the players of the scenario's world hold after it, slot by slot (`<slot>=<item>x<count>`), for every player the test ran with (`withPlayer`). This is the gap the last review found (the first join whose backfilled dust was overwritten by the menu item): a join step that fills or wipes a slot differently in another order now changes the dump.

It logs `5 orders, 3597 scenarios each` and takes about 16 seconds.

The same dump is the proof of T138. Dumped in the order of registration before and after the switch, it is equal line for line (3597 lines: results, counters, inventories); the commands are in REF T138. Per earlier task the dump differs from the one before it in exactly the lines section 7 names:

| Task | Dump lines that differ from the task before |
|---|---|
| T130 | (the first dump with inventories) |
| T131 | none |
| T132 | `Sneak/expanding`: counters `-` → `sneak+1` |
| T133 | `Hurt/fallen`: counters `hurt+1,wave_fail+1` → `wave_fail+1` |
| T134 | `BlockBreak/drawerFront`: counters `block_break+1` → `-` |
| T135, T136, T137 | none |
| T138 | none |

What it cannot see is unchanged: tips, chat, sounds, packets, and state no scenario and no counter reads. Changes of that kind are covered by the task's own test.

`SKYGEAR_HOOK_ORDER=reverse` and `shuffle:1` to `shuffle:4` over `./content/skygear/...` (28 packages with tests) pass on the final prototype.

## 11. Benchmarks

"5a8815b" is the base with the two benchmarks T130 adds (`BenchmarkBlockBreak`, `BenchmarkBlockPlace`), "final" the final prototype; eight alternating runs of each binary on the same machine (4 cores, shared), median (min-max) ns/op. The handler table and the `Emit` table are from two such series.

| Handler benchmark | 5a8815b | after T130 (tests only) | after T137 (old kernel) | final | final vs 5a8815b | final vs T137 | B/op | allocs/op |
|---|---|---|---|---|---|---|---|---|
| `HandleMove` | 175.8 (162.0-194.5) | 186.5 (168.8-197.5) | 185.1 (171.7-203.6) | 188.2 (159.4-202.4) | +7% | +2% | 24 -> 24 | 0 -> 0 |
| `UseOnBlock` | 593.1 (567.7-639.9) | 613.5 (547.3-647.2) | 654.5 (633.6-711.4) | 629.2 (581.3-684.3) | +6% | -4% | 0 -> 0 | 0 -> 0 |
| `Use` | 142.0 (129.5-152.6) | 140.1 (128.9-146.9) | 153.1 (142.5-201.5) | 148.9 (134.3-162.8) | +5% | -3% | 0 -> 0 | 0 -> 0 |
| `Second` | 26623.5 (25553.0-28720.0) | 27896.5 (26447.0-29490.0) | 27839.0 (25945.0-30460.0) | 25587.0 (24379.0-30055.0) | -4% | -8% | 10557 -> 10557 | 63 -> 63 |
| `BlockBreak` | 390.4 (364.6-406.2) | 381.0 (358.3-432.1) | 381.8 (364.4-503.5) | 397.7 (390.6-406.4) | +2% | +4% | 48 -> 48 | 1 -> 1 |
| `BlockPlace` | 267.5 (254.2-280.9) | 254.4 (241.7-279.8) | 270.6 (249.6-302.6) | 262.5 (245.9-295.2) | -2% | -3% | 48 -> 48 | 1 -> 1 |

| `Emit` benchmark (before -> after name) | 5a8815b | final | Median | B/op | allocs/op |
|---|---|---|---|---|---|
| `Emit/handle=false/subs=0 -> hook=plain/subs=0` | 3.0 (2.7-4.4) | 3.0 (2.5-3.7) | +1% (+0.0 ns) | 0 -> 0 | 0 -> 0 |
| `Emit/handle=false/subs=1 -> hook=plain/subs=1` | 5.4 (4.7-6.7) | 5.5 (4.8-7.2) | +2% (+0.1 ns) | 0 -> 0 | 0 -> 0 |
| `Emit/handle=false/subs=4 -> hook=plain/subs=4` | 13.3 (12.2-17.0) | 13.7 (12.8-17.2) | +2% (+0.3 ns) | 0 -> 0 | 0 -> 0 |
| `Emit/handle=false/subs=8 -> hook=plain/subs=8` | 24.1 (20.9-28.7) | 25.2 (24.2-26.1) | +5% (+1.1 ns) | 0 -> 0 | 0 -> 0 |
| `Emit/handle=false/subs=32 -> hook=plain/subs=32` | 83.1 (73.1-104.7) | 94.9 (84.8-114.7) | +14% (+11.8 ns) | 0 -> 0 | 0 -> 0 |
| `Emit/handle=true/subs=0 -> hook=cancel/subs=0` | 3.2 (2.8-4.1) | 6.5 (5.9-8.1) | +104% (+3.3 ns) | 0 -> 0 | 0 -> 0 |
| `Emit/handle=true/subs=1 -> hook=cancel/subs=1` | 6.1 (5.3-7.3) | 8.9 (7.7-10.5) | +46% (+2.8 ns) | 0 -> 0 | 0 -> 0 |
| `Emit/handle=true/subs=4 -> hook=cancel/subs=4` | 14.4 (12.4-16.1) | 17.2 (15.5-22.8) | +19% (+2.8 ns) | 0 -> 0 | 0 -> 0 |
| `Emit/handle=true/subs=8 -> hook=cancel/subs=8` | 25.4 (23.2-30.5) | 30.2 (28.2-32.3) | +19% (+4.7 ns) | 0 -> 0 | 0 -> 0 |
| `Emit/handle=true/subs=32 -> hook=cancel/subs=32` | 92.6 (77.9-110.7) | 96.6 (88.4-108.6) | +4% (+3.9 ns) | 0 -> 0 | 0 -> 0 |
| `EmitMove` | 16.7 (15.4-17.8) | 18.2 (16.5-19.7) | +9% (+1.5 ns) | 0 -> 0 | 0 -> 0 |

Allocations are equal everywhere. Reading the table:

- The handler benchmarks are given in four states so that each step's share can be read: 5a8815b; after T130 (test files only: the production code is that of 5a8815b); after T137 (every behaviour task, the OLD kernel); final. During the measurement the machine's spread between two runs of one binary was 15 to 30 percent of the median, so a median of eight carries roughly 5 to 10 percent. No handler benchmark is more than 15 percent slower; the largest are `HandleMove` at +7 percent (-2 and -1 percent in two earlier series; no task touches a subscriber of `Move` except the kernel switch) and `UseOnBlock` at +6 percent.
- **A finding on the way, fixed inside the plan.** The first measurement of the prototype showed `Use` 20 to 26 percent slower than 5a8815b and `UseOnBlock` up to 19 percent. It was not the kernel: the state after T130, which changes test files only, already measured `Use` at 168 ns against 149, and the kernel switch added 3 to 8 percent on top. Neutralising the kernel differences one by one (no per-step check, no call through the type parameter, a flat `Actor`) and randomising the function layout (`-ldflags=-randlayout`, four seeds) changed nothing. The cause was the first form of T130's test helper: `withPlayer` wrapped the test's function in one more closure to note the player's inventory, so every benchmark body ran one frame deeper, which shifts how the large events are copied on the stack (`UseEvent` is 256 bytes, copied once per subscriber): `Use` +13 percent, `UseOnBlock` +7, `BlockBreak` +8, `BlockPlace` -10, with identical production code. T130 as delivered wraps only in a run that dumps (`SKYGEAR_PIN_DUMP` set); otherwise `withPlayer` is the call it was. With that, the table above: `Use` after T130 140 ns against 142.
- What the plan's production changes cost, 5a8815b to final: `UseOnBlock` +6 percent (one subscriber more, `handwork.stickOther`; `fairy.time` asks the held item first); `Use` +5 percent; `BlockBreak` +2 percent (one subscriber more, `storage.front`); `BlockPlace`, `Second` within the spread.
- What the switch itself costs is in the `Emit` rows. A `Hook` (no cancel) is the old loop without the `handle` branch: within the spread (the row of 32 subscribers measured +8, +22 and +14 percent in three series, each with overlapping ranges; nothing in the loop costs more per subscriber than before). A `CancelHook` pays a fixed 3 ns per `Emit` (one call through the type parameter's dictionary to find the event's cancel state, with one copy of the event) and a fraction of a nanosecond per subscriber (the state is read before every default subscriber). `EmitMove` shows the fixed part. It could be removed by letting the handler hand the context to `Emit`; measured on `Use` that changes nothing a handler benchmark can see, so the plan keeps the one-argument `Emit`.
- All 20 player hooks with a context go through `CancelHook.Emit`, also those where nothing cancels (`Sprint`, `Chat`, …): 3 ns per event.
- If `Use` should ever need to be cheaper: its event carries `Off`, which one subscriber reads (`copier.clear`); without it the event is 120 bytes smaller and each of the seven copies per use that much shorter. A change of the kernel's events, outside this plan.

Hot path rules the plan keeps: a condition tests the cheapest thing first (the held item's type before a map lookup or a block read: `fairy.time` asks `TimeItem(e.Held)` before `island.CanBuild`; `storage.front` asks the block's type before the punch map); no subscriber allocates; `Emit` allocates nothing.

## 12. The owner's decisions, and what is outside this change

### 12.1 Cancels that vanilla does not need (candidates for "act and do not cancel")

Kept as A because the result is what a player sees today and the direction allows "handles in its state and cancels". The owner's second direction would also allow B for:

- `screens.menuItem` and `dimension.stone` on `Use` (in the air): Dragonfly does nothing with a nether star or the stone in the air, and every later subscriber of `Use` excludes them by the held item. B would change only the `cancelled=` of 10 result lines (`Use/menuItem…`, `Use/stone…`, `Use/lastMenuItem`), nothing a player sees.
- `handwork.stick` on grass or dirt and sneaking on a chest: Dragonfly does nothing with a stick there. B would let `island.guardSide` run: a tap at the island's edge would knock the pebble out AND show the "outside the island" tip; and the client would no longer get the block resent after each tap. Result lines `UseOnBlock/stick…` would change `cancelled=`.

Recommendation: leave them (no gain a player sees; the second has a visible side effect). Not in the plan.

### 12.2 The form of the `stick_other` record (C7)

Built: counts a stick used on a block that is neither grass nor dirt when nobody cancelled the click. Alternative: count every such tap (`core.HandleCancelled`, and `core.Takeover(e.S, e.P) == ""` so a click on the expansion map does not count). The achievement's text says "tap a block that is neither dirt nor grass with a stick"; its titles ("the one who taps although it will not work") suggest the tap that did nothing. The built form is the default subscriber and needs no option and no mode condition; the alternative is two lines in `handwork/subscribe.go` and one changed assertion in `stickother_test.go` (the drawer and the crop then count). Recommendation: the built form.

### 12.3 Outside this change (not fixed)

- E4 (section 8).
- `expand.guardSide` (`UseOnBlock`, HIGHEST) can never act: a click in expansion mode is cancelled in LOWEST by `expand.guard`, or by the subscriber of the screen item in hand. Kept.
- `elevator.up` (`Jump`) has no takeover condition: if a client in island expansion mode reports a jump, the body would ride an elevator it stands on. In practice the mode moves the body to the map. `elevator.down` has the condition from T131; the jump was left alone because it is not part of the stop.
- `handwork.gather` (NORMAL) drops bark fibre when an axe is used on a log, then Dragonfly strips the log. At the island's edge, on a face that looks outside, `island.guardSide` (HIGHEST) cancels the click afterwards: fibre drops, the log is not stripped, and the click can be repeated. Found by reading; today's behaviour, unchanged by this plan. The fix would be the tier of the click that goes through (as `machine.noteFace` in T137).
- A hunter's menu item behaves differently on a block (nothing opens) and in the air (the run menu opens). Unchanged.
- In island expansion mode a click on the map with the menu item in hand does not show the map chunk again. Unchanged.
- `-count=2` in one process and the `-race` findings of `docs/TODO.md` are unchanged.

## 13. Tasks

Base 5a8815b. One chain; plan a is T130 to T134, plan b is T135 to T139.

| Task | What | Goldens | Commit subject |
|---|---|---|---|
| T130 | the order guard compares the players' inventories; `BenchmarkBlockBreak`, `BenchmarkBlockPlace` | none | `test(skygear): the tier order guard compares what the players hold` |
| T131 | condition: `elevator.down` and `farm.crouchGrow` leave a player a takeover holds (preserving) | none | `refactor(elevator): the elevator and the saplings leave a sneak a mode holds` |
| T132 | `Sneak` without a stop: C1, C2 | `order.golden` (3 lines), traces of 3 lines of `handler-simple.golden` | `feat(core): no subscriber stops a sneak, every sneak counts` |
| T133 | a cancelled move and a cancelled hit reach no later subscriber: C5, C6 | traces of 2 lines of `handler-simple.golden` | `fix(achievement): a cancelled move and a cancelled hit are not counted` |
| T134 | the drawer's front refuses the break in LOW: C4 | `order.golden` (+1 line), traces of 15 lines of `handler-blocks.golden` | `fix(storage): a drawer hit on its front is no break for the other subscribers` |
| T135 | the stick record is its own MONITOR subscriber: C7; two kept pairs go | `order.golden` (+1 line), traces of 28 lines of `handler-use.golden` | `refactor(handwork): the stick record is a Monitor subscriber of the click that went through` |
| T136 | `fairy.time` joins the held tools: C8; the last kept pair and the kept-pair mechanism go | `order.golden` (1 line moves), traces of 62 lines of `handler-use.golden` | `feat(fairy): a fairy item takes the click like the other held tools` |
| T137 | `machine.noteFace` notes the click that goes through: C9 | `order.golden` (1 line moves), traces of 51 lines of `handler-use.golden` | `refactor(machine): the placement face is noted from the click that goes through` |
| T138 | the kernel switch: no `Handle`, no `Always`, `CancelHook`, `core.HandleCancelled`, `Who`/`Actor`, 33 call sites | `order.golden` only (sorted inside tiers, ` handleCancelled`); the four scenario goldens byte for byte | `refactor(core): no subscriber stops the others, a cancelled event reaches only handleCancelled subscribers` |
| T139 | documents | none | `docs: no propagation stop, the handleCancelled option, no order inside a tier` |

Goldens frozen in every task: `wire/testdata`, `*.byname.golden`, the save fixture, `kinds.golden`, `look.golden`, `creative.golden`, `settings.golden`, `popup*.golden`, `lifecycle.golden` (no lifecycle trace changes in any task).

How a task re-records and proves it, the same in every task that names goldens:

```sh
S=/tmp/claude-0/-home-user-SkyGear/a71fa8d5-6f20-5c3f-ba20-eeb27e13ad8a/scratchpad/<task id>
mkdir -p $S && go test -c -o $S/root.test ./content/skygear
(cd content/skygear && SKYGEAR_GOLDEN=update $S/root.test -test.count=1 -test.run '^TestPin(<the tests the task names>)$')
git status --short content/skygear/testdata content/skygear/wire/testdata content/skygear/dimension/testdata   # only the goldens the task names
python3 - <<'PY'      # the check script: prints RESULT for every scenario whose key or result part changed
import subprocess
for g in ['handler-simple', 'handler-blocks', 'handler-use', 'lifecycle']:
    f = 'content/skygear/testdata/pin/%s.golden' % g
    def read(text):
        out = {}
        for ln in text.splitlines():
            key, rest = ln.split(': ', 1)
            trace, _, res = rest.partition(' | ')
            out[key] = (trace, res)
        return out
    o = read(subprocess.run(['git', 'show', 'HEAD:' + f], capture_output=True, text=True, check=True).stdout)
    n = read(open(f, encoding='utf-8').read())
    for k in sorted(set(o) | set(n)):
        if k not in o: print('RESULT %s: new scenario %s' % (g, k))
        elif k not in n: print('RESULT %s: scenario %s is gone' % (g, k))
        elif o[k][1] != n[k][1]: print('RESULT %s: %s\n  was: %s\n  now: %s' % (g, k, o[k][1], n[k][1]))
    print('%s: %d scenarios, %d traces changed' % (g, len(n), sum(1 for k in o if k in n and o[k][0] != n[k][0])))
PY
```

It must print no `RESULT` line, and the counts of changed traces the task's REF gives. Any `RESULT` line, another golden changed, or another count: restore the goldens (`git checkout -- content/skygear/testdata`) and report `NEEDS_CONTEXT`.

The dump of a state (used by T138, and useful anywhere): `(cd content/skygear && SKYGEAR_PIN_DUMP=$S/dump.txt $S/root.test -test.count=1 >/dev/null 2>&1); sort $S/dump.txt > $S/dump.sorted`.

## REF T130

The order guard compares what the players hold. Prototype patch: `T130.patch`.

Tests and benchmarks only. The dump line of the order guard becomes `<scenario> | <results> | <counters> | <inventories>`; the inventories are those of the players the test ran with (`withPlayer`) in the world of the last such player, so each scenario's players and no others. Nothing runs when `SKYGEAR_PIN_DUMP` is unset: `withPlayer` then calls `coretest.WithPlayer` with the test's function as it did, without a wrapper. This matters: a wrapper on that path makes every handler benchmark run one frame deeper, which measured up to 13 percent on `BenchmarkUse` (section 11). `BenchmarkBlockBreak` and `BenchmarkBlockPlace` are the "stone" and "dirt" scenarios of `TestPinBlocks` as benchmarks; they are what T138 measures.

Proof: `TestTierOrder` passes and logs `5 orders, 3597 scenarios each`; `git status --short` shows the four test files only.

### `content/skygear/helpers_test.go`

~~~~diff
--- a/content/skygear/helpers_test.go
+++ b/content/skygear/helpers_test.go
@@ -63,7 +63,15 @@ func spawnPlayer(s *session) *world.EntityHandle {
 // the test (Dragonfly would only log it).
 func withPlayer(t testing.TB, s *session, h *world.EntityHandle, f func(p *player.Player, tx *world.Tx)) {
 	t.Helper()
-	coretest.WithPlayer(t, s.Lifecycle, h, f)
+	if pinDump == "" {
+		coretest.WithPlayer(t, s.Lifecycle, h, f)
+		return
+	}
+	// A run that dumps (TestTierOrder) also notes what the player holds.
+	coretest.WithPlayer(t, s.Lifecycle, h, func(p *player.Player, tx *world.Tx) {
+		pinNoteInventory(p)
+		f(p, tx)
+	})
 }
 
 func tick(s *session, n int) { coretest.Tick(s.Lifecycle, n) }
~~~~

### `content/skygear/pin_bench_test.go`

~~~~diff
--- a/content/skygear/pin_bench_test.go
+++ b/content/skygear/pin_bench_test.go
@@ -87,6 +87,50 @@ func BenchmarkUse(b *testing.B) {
 	})
 }
 
+// BenchmarkBlockBreak breaks a stone block of the island with a stack of stone
+// in hand through the session's handler: every subscriber is reached and none
+// cancels (the "stone" scenario of TestPinBlocks).
+func BenchmarkBlockBreak(b *testing.B) {
+	s := benchSession(b)
+	island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
+	h := spawnPlayer(s)
+	b.ReportAllocs()
+	withPlayer(b, s, h, func(p *player.Player, tx *world.Tx) {
+		at := cube.Pos{4, island.IslandY + 2, 4}
+		tx.SetBlock(at, block.Stone{}, nil)
+		p.SetHeldItems(item.NewStack(block.Stone{}, 3), item.Stack{})
+		hd := s.Handler(p)
+		ctx := player.NewEventContext(tx, p)
+		drops := []item.Stack{item.NewStack(block.Stone{}, 1)}
+		xp := 0
+		b.ResetTimer()
+		for range b.N {
+			hd.HandleBlockBreak(ctx, at, &drops, &xp)
+		}
+	})
+}
+
+// BenchmarkBlockPlace places dirt on the island from a stack of three through
+// the session's handler: every subscriber is reached and none cancels (the
+// "dirt" scenario of TestPinBlocks).
+func BenchmarkBlockPlace(b *testing.B) {
+	s := benchSession(b)
+	island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
+	h := spawnPlayer(s)
+	b.ReportAllocs()
+	withPlayer(b, s, h, func(p *player.Player, tx *world.Tx) {
+		at := cube.Pos{4, island.IslandY + 2, 4}
+		p.SetHeldItems(item.NewStack(block.Dirt{}, 3), item.Stack{})
+		hd := s.Handler(p)
+		ctx := player.NewEventContext(tx, p)
+		var dirt world.Block = block.Dirt{}
+		b.ResetTimer()
+		for range b.N {
+			hd.HandleBlockPlace(ctx, at, dirt)
+		}
+	})
+}
+
 // BenchmarkLookMachine updates the WAILA panel of a player who looks at a
 // working crusher: the trace to the block, the panel's lines and the title.
 func BenchmarkLookMachine(b *testing.B) {
~~~~

### `content/skygear/pin_test.go`

~~~~diff
--- a/content/skygear/pin_test.go
+++ b/content/skygear/pin_test.go
@@ -75,8 +75,9 @@ func pinRun(t *testing.T, lines map[string]string, scenario string, outcome func
 	}
 	lines[scenario] = strings.Join(k.take(), " ") + " | " + out
 	if pinDump != "" {
+		dump := pinCountersSince(counted) + " | " + pinInventoryDump()
 		pinDumpMu.Lock()
-		pinDumpLines[scenario] = pinCountersSince(counted)
+		pinDumpLines[scenario] = dump
 		pinDumpMu.Unlock()
 	}
 }
~~~~

### `content/skygear/tierorder_test.go`

~~~~diff
--- a/content/skygear/tierorder_test.go
+++ b/content/skygear/tierorder_test.go
@@ -12,8 +12,11 @@ import (
 	"sync"
 	"testing"
 
+	"github.com/df-mc/dragonfly/server/item/inventory"
+	"github.com/df-mc/dragonfly/server/player"
 	"github.com/df-mc/dragonfly/server/world"
 	"github.com/presentkim/skygear/content/skygear/core/coretest"
+	"github.com/presentkim/skygear/content/skygear/naming"
 	"github.com/presentkim/skygear/platform/account"
 	"github.com/presentkim/skygear/platform/db"
 )
@@ -29,7 +32,8 @@ import (
 var hookOrder = os.Getenv(coretest.HookOrderEnv)
 
 // pinDump, when set, is the file every pin test appends its scenarios to:
-// one line "<scenario> | <results> | <counters>" each (pinDumpLines).
+// one line "<scenario> | <results> | <counters> | <inventories>" each
+// (pinDumpLines).
 var pinDump = os.Getenv("SKYGEAR_PIN_DUMP")
 
 // orderKept are the exceptions: the pairs of one tier whose order still
@@ -144,6 +148,56 @@ func pinCountersSince(before map[string]int64) string {
 	return strings.Join(out, ",")
 }
 
+// pinInventories are the inventories of the players of pinInventoryWorld,
+// the world of the last player a test ran with (withPlayer), in a run that
+// dumps.
+var (
+	pinInventories    []*inventory.Inventory
+	pinInventoryWorld *world.World
+)
+
+// pinNoteInventory remembers the inventory of a player a test runs with, in
+// a run that dumps. A player of another world than the last starts the list
+// again: each scenario has a world of its own.
+func pinNoteInventory(p *player.Player) {
+	if pinDump == "" {
+		return
+	}
+	pinDumpMu.Lock()
+	defer pinDumpMu.Unlock()
+	if w := p.Tx().World(); w != pinInventoryWorld {
+		pinInventoryWorld, pinInventories = w, nil
+	}
+	if inv := p.Inventory(); !slices.Contains(pinInventories, inv) {
+		pinInventories = append(pinInventories, inv)
+	}
+}
+
+// pinInventoryDump is what the players of the last scenario's world hold, one
+// "<slot>=<item>x<count>" per used slot, the players separated by "/"; "-"
+// for a player that holds nothing.
+func pinInventoryDump() string {
+	pinDumpMu.Lock()
+	defer pinDumpMu.Unlock()
+	var players []string
+	for _, inv := range pinInventories {
+		var slots []string
+		for slot, it := range inv.Slots() {
+			if !it.Empty() {
+				slots = append(slots, fmt.Sprintf("%d=%sx%d", slot, naming.ItemKey(it.Item()), it.Count()))
+			}
+		}
+		if len(slots) == 0 {
+			slots = []string{"-"}
+		}
+		players = append(players, strings.Join(slots, ","))
+	}
+	if len(players) == 0 {
+		return "-"
+	}
+	return strings.Join(players, "/")
+}
+
 // pinResults is the part of a pin line after the trace.
 func pinResults(line string) string {
 	_, res, ok := strings.Cut(line, " | ")
@@ -181,7 +235,8 @@ var tierOrders = []string{"reverse", "shuffle:1", "shuffle:2", "shuffle:3", "shu
 // may differ) and that order.golden lists the same subscribers in every tier,
 // and it must count exactly what the run in the order of registration counted
 // in every scenario (the achievement counters of the players, which no golden
-// holds).
+// holds) and leave the players the scenario ran with holding exactly the
+// same items in the same slots.
 func TestTierOrder(t *testing.T) {
 	if hookOrder != "" || pinDump != "" {
 		return // this is a run TestTierOrder started
~~~~

## REF T131

The elevator and the saplings leave a player a mode holds. Prototype patch: `T131.patch`.

Two conditions, behaviour-preserving (change C3 of section 7: nothing reachable changes). Each test registers a takeover of its own in the test binary of its package (no other feature is set up there), holds the player with it and sneaks through the handler.

Before the change the tests fail with `a held player rode the elevator to [10.3 67 8.7]` and `a held player's sneaks grew the sapling: block.Log{…}`.

The condition is asked before `Down`, which has the side effect. No golden changes; the dump of the order guard is equal to the one of T130.

### `content/skygear/elevator/held_test.go` (new file)

~~~~go
package elevator_test

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/core/coretest"
	"github.com/presentkim/skygear/content/skygear/elevator"
	"github.com/presentkim/skygear/content/skygear/island"
)

// held are the players the test's takeover holds.
var held = map[*world.EntityHandle]bool{}

func init() {
	core.RegisterTakeover("elevatortest", func(_ *core.Session, p *player.Player) bool { return held[p.H()] })
}

// TestHeldPlayerDoesNotRideDown: a sneak on an elevator rides it down, but
// not for a player a takeover holds (island expansion mode, a hunt): the mode
// has the player's body, and its own use for the sneak.
func TestHeldPlayerDoesNotRideDown(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	h := coretest.SpawnPlayer(s)
	x, z := island.Centre+2, island.Centre
	base := island.IslandY + 2
	red := elevator.Elevator{Colour: item.ColourRed()}
	coretest.WithPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		tx.SetBlock(cube.Pos{x, base, z}, red, nil)
		tx.SetBlock(cube.Pos{x, base + 12, z}, red, nil)
		top := mgl64.Vec3{float64(x) + 0.3, float64(base + 13), float64(z) + 0.7}
		p.Teleport(top)
		hd := s.Handler(p).(*core.PlayerHandler)
		sneak := func() { hd.HandleToggleSneak(player.NewEventContext(tx, p), true) }

		held[p.H()] = true
		sneak()
		if got := p.Position(); got != top {
			t.Fatalf("a held player rode the elevator to %v", got)
		}
		delete(held, p.H())
		sneak()
		if got := p.Position().Y(); got != float64(base+1) {
			t.Fatalf("a free player is at y %v after the sneak, want %d", got, base+1)
		}
	})
}
~~~~

### `content/skygear/elevator/subscribe.go`

~~~~diff
--- a/content/skygear/elevator/subscribe.go
+++ b/content/skygear/elevator/subscribe.go
@@ -16,9 +16,10 @@ func subscribeJump() {
 }
 
 // Sneaking on an elevator rides it down. A ride cancels the event for the
-// later tiers: the sneak was used.
+// later tiers: the sneak was used. A player a takeover holds (island
+// expansion mode, a hunt) does not ride: that mode has the player's body.
 func subscribeSneak() {
 	core.Sneak.Handle(core.Normal, "elevator.down", func(e core.PlayerEvent) bool {
-		return Down(e.S, e.P, e.Tx)
+		return core.Takeover(e.S, e.P) == "" && Down(e.S, e.P, e.Tx)
 	})
 }
~~~~

### `content/skygear/farm/held_test.go` (new file)

~~~~go
package farm_test

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/core/coretest"
)

// held are the players the test's takeover holds.
var held = map[*world.EntityHandle]bool{}

func init() {
	core.RegisterTakeover("farmtest", func(_ *core.Session, p *player.Player) bool { return held[p.H()] })
}

// TestHeldPlayerGrowsNoSapling: sneaking next to a sapling grows it by
// chance, but not for a player a takeover holds (island expansion mode, a
// hunt): that sneak belongs to the mode.
func TestHeldPlayerGrowsNoSapling(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	h := coretest.SpawnPlayer(s)
	coretest.WithPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		pos := cube.PosFromVec3(p.Position()).Add(cube.Pos{1, 0, 0})
		tx.SetBlock(pos.Side(cube.FaceDown), block.Dirt{}, nil)
		tx.SetBlock(pos, block.Sapling{Type: block.OakSapling()}, nil)
		hd := s.Handler(p).(*core.PlayerHandler)
		sneak := func() { hd.HandleToggleSneak(player.NewEventContext(tx, p), true) }
		sapling := func() bool {
			_, ok := tx.Block(pos).(block.Sapling)
			return ok
		}

		held[p.H()] = true
		for range 200 {
			sneak()
		}
		if b, ok := tx.Block(pos).(block.Sapling); !ok || b.Aged {
			t.Fatalf("a held player's sneaks grew the sapling: %#v", tx.Block(pos))
		}
		delete(held, p.H())
		for range 200 {
			if sneak(); !sapling() {
				break
			}
		}
		if sapling() {
			t.Fatal("a free player's sneaks did not grow the sapling")
		}
	})
}
~~~~

### `content/skygear/farm/subscribe.go`

~~~~diff
--- a/content/skygear/farm/subscribe.go
+++ b/content/skygear/farm/subscribe.go
@@ -36,10 +36,14 @@ func registerComposterIO() {
 }
 
 // Sneaking grows saplings nearby by chance, unless another step used the
-// sneak (an elevator ride cancels the event): work of low priority, High.
+// sneak (an elevator ride cancels the event): work of low priority, High. A
+// player a takeover holds (island expansion mode, a hunt) grows none: that
+// sneak belongs to the mode.
 func subscribeSneak() {
 	core.Sneak.On(core.High, "farm.crouchGrow", func(e core.PlayerEvent) {
-		CrouchGrow(e.S, e.P, e.Tx)
+		if core.Takeover(e.S, e.P) == "" {
+			CrouchGrow(e.S, e.P, e.Tx)
+		}
 	})
 }
 
~~~~

## REF T132

No subscriber stops a sneak. Prototype patch: `T132.patch`.

Change C1 and C2. On today's kernel: the two `Handle` subscribers of `Sneak` become `On`, `expand.show` moves to NORMAL (it no longer blocks anything: the mode's work on the sneak), `achievement.sneak` moves to MONITOR, `farm.crouchGrow` stays in HIGH (it reads where the player is when the ride is done).

Before the change `sneak_test.go` fails with `a sneak in island expansion mode counted 0, want 1`, `a sneak that rides the elevator: 1 sneaks counted, want 2` and `200 sneaks that rode the elevator down grew no sapling where they arrived`.

Goldens: re-record `^TestPin(Order|Simple)$`. `order.golden`: the four `Sneak` lines become

    Sneak NORMAL expand.show
    Sneak NORMAL elevator.down
    Sneak HIGH farm.crouchGrow
    Sneak MONITOR achievement.sneak

`handler-simple.golden`: the check script prints `handler-simple: 29 scenarios, 3 traces changed` and 0 for the other three; the three lines are

    Sneak/elevator: expand.show elevator.down farm.crouchGrow achievement.sneak | cancelled=false
    Sneak/expanding: expand.show elevator.down farm.crouchGrow achievement.sneak | cancelled=false
    Sneak/start: expand.show elevator.down farm.crouchGrow achievement.sneak | cancelled=false

### `content/skygear/achievement/subscribe.go`

~~~~diff
--- a/content/skygear/achievement/subscribe.go
+++ b/content/skygear/achievement/subscribe.go
@@ -7,7 +7,7 @@ import (
 
 // subscribe registers the achievement counters on the hooks they count. A
 // counter records what happened and changes nothing: it is a Monitor
-// subscriber, except the sneak counter (subscribeMoves).
+// subscriber.
 func subscribe() {
 	subscribeSimple()
 	subscribeMoves()
@@ -51,18 +51,14 @@ func subscribeSimple() {
 
 // The achievement counters of the walk, sneak, drop and hurt events. The walk
 // counts from where the player stood when the event was built (e.From), so a
-// step that moved the player meanwhile does not change it.
-//
-// The sneak counter is in the Low tier, not in Monitor: a sneak counts when
-// the elevator then rides (elevator.down, Normal, cancels the event for the
-// later tiers) and does not count in island expansion mode (expand.show,
-// Lowest). No tier after Normal sees the first and no option tells the two
-// apart, so the counter stands between them.
+// step that moved the player meanwhile does not change it. Every sneak
+// counts, whatever else it did (an elevator ride, the screen of island
+// expansion mode).
 func subscribeMoves() {
 	core.Move.On(core.Monitor, "achievement.walk", func(e core.MoveEvent) {
 		CountWalk(e.S, e.P, e.From, e.Pos)
 	})
-	core.Sneak.On(core.Low, "achievement.sneak", func(e core.PlayerEvent) {
+	core.Sneak.On(core.Monitor, "achievement.sneak", func(e core.PlayerEvent) {
 		Count(e.P, "sneak", 1)
 	})
 	core.ItemDrop.On(core.Monitor, "achievement.drop", func(e core.ItemEvent) {
~~~~

### `content/skygear/elevator/subscribe.go`

~~~~diff
--- a/content/skygear/elevator/subscribe.go
+++ b/content/skygear/elevator/subscribe.go
@@ -15,11 +15,14 @@ func subscribeJump() {
 	})
 }
 
-// Sneaking on an elevator rides it down. A ride cancels the event for the
-// later tiers: the sneak was used. A player a takeover holds (island
-// expansion mode, a hunt) does not ride: that mode has the player's body.
+// Sneaking on an elevator rides it down. The sneak is not cancelled: the
+// player goes on sneaking and the other subscribers of the sneak run. A
+// player a takeover holds (island expansion mode, a hunt) does not ride: that
+// mode has the player's body.
 func subscribeSneak() {
-	core.Sneak.Handle(core.Normal, "elevator.down", func(e core.PlayerEvent) bool {
-		return core.Takeover(e.S, e.P) == "" && Down(e.S, e.P, e.Tx)
+	core.Sneak.On(core.Normal, "elevator.down", func(e core.PlayerEvent) {
+		if core.Takeover(e.S, e.P) == "" {
+			Down(e.S, e.P, e.Tx)
+		}
 	})
 }
~~~~

### `content/skygear/expand/subscribe.go`

~~~~diff
--- a/content/skygear/expand/subscribe.go
+++ b/content/skygear/expand/subscribe.go
@@ -22,7 +22,9 @@ func subscribe() {
 }
 
 // A player in island expansion mode stays in place and sees the expansion
-// screen again when sneaking.
+// screen again when sneaking. The sneak stays a sneak for everyone else: it
+// is not cancelled, and the subscribers that have no business with a player
+// in a mode say so themselves (core.Takeover).
 func subscribeHold() {
 	core.Move.Handle(core.Lowest, "expand.hold", func(e core.MoveEvent) bool {
 		if _, expanding := Of(e.S, e.P); expanding {
@@ -31,12 +33,10 @@ func subscribeHold() {
 		}
 		return false
 	})
-	core.Sneak.Handle(core.Lowest, "expand.show", func(e core.PlayerEvent) bool {
+	core.Sneak.On(core.Normal, "expand.show", func(e core.PlayerEvent) {
 		if _, ok := Of(e.S, e.P); ok {
 			Show(e.S, e.P)
-			return true
 		}
-		return false
 	})
 }
 
~~~~

### `content/skygear/farm/subscribe.go`

~~~~diff
--- a/content/skygear/farm/subscribe.go
+++ b/content/skygear/farm/subscribe.go
@@ -35,10 +35,12 @@ func registerComposterIO() {
 	})
 }
 
-// Sneaking grows saplings nearby by chance, unless another step used the
-// sneak (an elevator ride cancels the event): work of low priority, High. A
-// player a takeover holds (island expansion mode, a hunt) grows none: that
-// sneak belongs to the mode.
+// Sneaking grows saplings around the player by chance. It is work of low
+// priority (High): it reads where the player is when the sneak's other work
+// is done, so after an elevator ride (elevator.down, Normal) the saplings
+// around the place the player arrived at get the chance. A player a takeover
+// holds (island expansion mode, a hunt) grows none: that sneak belongs to the
+// mode.
 func subscribeSneak() {
 	core.Sneak.On(core.High, "farm.crouchGrow", func(e core.PlayerEvent) {
 		if core.Takeover(e.S, e.P) == "" {
~~~~

### `content/skygear/sneak_test.go` (new file)

~~~~go
package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/elevator"
	"github.com/presentkim/skygear/content/skygear/expand"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/platform/account"
	"github.com/presentkim/skygear/platform/instance"
)

// sneakElevators puts two red elevators 12 blocks apart at the island's
// centre and returns where a player stands on the upper and on the lower one.
func sneakElevators(tx *world.Tx) (top, bottom mgl64.Vec3) {
	x, z := island.Centre+2, island.Centre
	base := island.IslandY + 2
	red := elevator.Elevator{Colour: item.ColourRed()}
	tx.SetBlock(cube.Pos{x, base, z}, red, nil)
	tx.SetBlock(cube.Pos{x, base + 12, z}, red, nil)
	return mgl64.Vec3{float64(x) + 0.3, float64(base + 13), float64(z) + 0.7}, mgl64.Vec3{float64(x) + 0.3, float64(base + 1), float64(z) + 0.7}
}

// TestEverySneakCounts: no subscriber of Sneak keeps another from running.
// A sneak counts whatever else it does: in island expansion mode, where it
// shows the expansion screen again and the body stays where the mode put it,
// and on an elevator, which it rides down.
func TestEverySneakCounts(t *testing.T) {
	d := bridgeAccount(t)
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		acc := instance.AccountID(p)
		if _, err := d.TouchAccount(acc, "", p.Name()); err != nil {
			t.Fatal(err)
		}
		top, bottom := sneakElevators(tx)
		sneak := func() { s.Handler(p).HandleToggleSneak(player.NewEventContext(tx, p), true) }

		expand.Show(s.Session, p)
		body := p.Position()
		sneak()
		if got := account.Counter(acc, "sneak"); got != 1 {
			t.Errorf("a sneak in island expansion mode counted %d, want 1", got)
		}
		if got := p.Position(); got != body {
			t.Errorf("a sneak in island expansion mode moved the player from %v to %v", body, got)
		}
		if _, ok := expand.Of(s.Session, p); !ok {
			t.Error("the sneak ended island expansion mode")
		}
		expand.Stop(s.Session, p, tx)

		p.Teleport(top)
		sneak()
		if got := account.Counter(acc, "sneak"); got != 2 {
			t.Errorf("a sneak that rides the elevator: %d sneaks counted, want 2", got)
		}
		if got := p.Position(); got != bottom {
			t.Errorf("the sneak on the elevator left the player at %v, want %v", got, bottom)
		}
	})
}

// TestSneakRideGrowsSaplingsWhereItArrives: a sneak that rides an elevator
// down is still a sneak for the saplings: those around the place the player
// arrives at get their chance to grow.
func TestSneakRideGrowsSaplingsWhereItArrives(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		top, bottom := sneakElevators(tx)
		at := cube.PosFromVec3(bottom).Add(cube.Pos{1, 0, 0})
		tx.SetBlock(at.Side(cube.FaceDown), block.Dirt{}, nil)
		tx.SetBlock(at, block.Sapling{Type: block.OakSapling()}, nil)
		for range 200 {
			p.Teleport(top)
			elevator.ResetCooldown(s.Session, p)
			s.Handler(p).HandleToggleSneak(player.NewEventContext(tx, p), true)
			if got := p.Position(); got != bottom {
				t.Fatalf("the sneak left the player at %v, want %v", got, bottom)
			}
			if _, ok := tx.Block(at).(block.Sapling); !ok {
				return
			}
		}
		t.Fatal("200 sneaks that rode the elevator down grew no sapling where they arrived")
	})
}
~~~~

## REF T133

A cancelled move and a cancelled hit are not counted. Prototype patch: `T133.patch`.

Change C5 and C6. On today's kernel "a cancel also keeps the later subscribers away" is `Handle` returning true where it cancels. `look.update` (`Always`) still runs for the void move.

`TestWalkCountsTheMoveIntoTheVoid` asserted the opposite of C5 (the void move counts 1): it is renamed `TestMoveIntoTheVoidIsNotWalked` and rewritten (the void move counts 0, the next move of one block counts 1). This is the one existing test the plan changes. Before the change the two tests fail with `walk counted 1 for the cancelled move into the void, want 0` and `hurt counted 2 after the cancelled felling hit, want 1`.

Goldens: re-record `^TestPinSimple$`. The check script prints `handler-simple: 29 scenarios, 2 traces changed`; the two lines are

    Hurt/fallen: dimension.frozen dimension.fallen | cancelled=true
    Move/void: expand.hold island.void look.update | cancelled=true

### `content/skygear/dimension/subscribe.go`

~~~~diff
--- a/content/skygear/dimension/subscribe.go
+++ b/content/skygear/dimension/subscribe.go
@@ -46,10 +46,12 @@ func subscribeHurt() {
 		}
 		return false
 	})
-	core.Hurt.On(core.Highest, "dimension.fallen", func(e core.HurtEvent) {
+	core.Hurt.Handle(core.Highest, "dimension.fallen", func(e core.HurtEvent) bool {
 		if Fallen(e.S, e.Tx, e.P, *e.Damage) {
-			e.Cancel()
+			e.Cancel() // no hurt for the subscribers after it: the run ended instead
+			return true
 		}
+		return false
 	})
 }
 
~~~~

### `content/skygear/fellinghit_test.go` (new file)

~~~~go
package skygear

import (
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/dimension"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/platform/account"
	"github.com/presentkim/skygear/platform/instance"
)

// TestFellingHitIsNotAHurt: the hit that would kill a hunter is cancelled
// (the run ends instead and the hunter goes back unharmed), and a cancelled
// hit is not a hurt: the hurt counter is not called for it. A hit the hunter
// takes counts.
func TestFellingHitIsNotAHurt(t *testing.T) {
	d := bridgeAccount(t)
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	t.Cleanup(func() {
		s.World.Do(func(tx *world.Tx) {
			if e, ok := h.Entity(tx); ok {
				tx.RemoveEntity(e)
			}
		})
	})
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		acc := instance.AccountID(p)
		if _, err := d.TouchAccount(acc, "", p.Name()); err != nil {
			t.Fatal(err)
		}
		if !dimension.Begin(s.Session, tx, p, sgitem.MobSlime) {
			t.Fatal("the hunt did not begin")
		}
		hurt := func(dmg float64) *player.Context {
			ctx := player.NewEventContext(tx, p)
			var immunity time.Duration
			s.Handler(p).HandleHurt(ctx, &dmg, false, &immunity, nil)
			return ctx
		}
		if hurt(1).Cancelled() {
			t.Fatal("a hit the hunter survives was cancelled")
		}
		if got := account.Counter(acc, "hurt"); got != 1 {
			t.Errorf("hurt counted %d for a hit the hunter took, want 1", got)
		}
		if !hurt(p.Health() + 100).Cancelled() {
			t.Fatal("the hit that would kill the hunter was not cancelled")
		}
		if dimension.Hunting(s.Session, p) {
			t.Fatal("the run did not end")
		}
		if got := account.Counter(acc, "hurt"); got != 1 {
			t.Errorf("hurt counted %d after the cancelled felling hit, want 1", got)
		}
	})
}
~~~~

### `content/skygear/island/subscribe.go`

~~~~diff
--- a/content/skygear/island/subscribe.go
+++ b/content/skygear/island/subscribe.go
@@ -17,13 +17,16 @@ func subscribe() {
 }
 
 // A player that falls below the island returns from the void: the last step
-// that may cancel the move, Highest.
+// that may cancel the move, Highest. The cancelled move is no move for the
+// subscribers after it (the walk counter).
 func subscribeVoid() {
-	core.Move.On(core.Highest, "island.void", func(e core.MoveEvent) {
+	core.Move.Handle(core.Highest, "island.void", func(e core.MoveEvent) bool {
 		if e.Pos.Y() < VoidY {
 			e.Cancel()
 			ReturnFromVoid(e.S, e.P, e.Tx)
+			return true
 		}
+		return false
 	})
 }
 
~~~~

### `content/skygear/walkvoid_test.go`

~~~~diff
--- a/content/skygear/walkvoid_test.go
+++ b/content/skygear/walkvoid_test.go
@@ -13,12 +13,11 @@ import (
 	"github.com/presentkim/skygear/platform/instance"
 )
 
-// TestWalkCountsTheMoveIntoTheVoid: the move that takes a player below the
-// island counts the distance of that move, from where the player stood to
-// where it moved, although the island brings the player back in the same
-// event. It counts the same wherever achievement.walk stands relative to
-// island.void: the event carries where the player stood (MoveEvent.From).
-func TestWalkCountsTheMoveIntoTheVoid(t *testing.T) {
+// TestMoveIntoTheVoidIsNotWalked: the move that takes a player below the
+// island is cancelled (the island brings the player back), and a cancelled
+// move is not a walk: the walk counter is not called for it. The next move
+// counts from where the island put the player.
+func TestMoveIntoTheVoidIsNotWalked(t *testing.T) {
 	d := bridgeAccount(t)
 	s := testSession(t)
 	loadAround(s)
@@ -38,11 +37,19 @@ func TestWalkCountsTheMoveIntoTheVoid(t *testing.T) {
 		if !ctx.Cancelled() {
 			t.Fatal("the move into the void was not cancelled")
 		}
-		if got := p.Position(); got.X() != 3.5 || got.Y() != island.VoidReturnY {
-			t.Fatalf("the player was put at %v", got)
+		back := p.Position()
+		if back.X() != 3.5 || back.Y() != island.VoidReturnY {
+			t.Fatalf("the player was put at %v", back)
 		}
+		if got := account.Counter(acc, "walk"); got != 0 {
+			t.Errorf("walk counted %d for the cancelled move into the void, want 0", got)
+		}
+		if got := account.Counter(acc, "void_return"); got != 1 {
+			t.Errorf("void_return counted %d, want 1", got)
+		}
+		s.Handler(p).HandleMove(player.NewEventContext(tx, p), back.Add(mgl64.Vec3{1, 0, 0}), cube.Rotation{})
 		if got := account.Counter(acc, "walk"); got != 1 {
-			t.Errorf("walk counted %d for a move of one block, want 1", got)
+			t.Errorf("walk counted %d after a move of one block from where the island put the player, want 1", got)
 		}
 	})
 }
~~~~

## REF T134

A drawer hit on its front is no break. Prototype patch: `T134.patch`.

Change C4. The refusal of a break that ends a hit on a drawer's front becomes its own subscriber `storage.front` in LOW (`Handle`, cancels and stops); `storage.break` (NORMAL) keeps the rest. In LOW it is exclusive with `island.guard` by `island.CanBuild`. The condition tests the block's type first (the hot path: every break).

Before the change `drawerfront_test.go` fails with `block_break counted 1 for the cancelled break, want 0` and `the tool gained 1 experience from the cancelled break, want 0`.

`pin_blocks_test.go`: the row of `BlockBreak/stone` gains `storage.front` after `island.guard`, and the assertion on `BlockBreak/drawerFront` becomes the prefix `expand.guard island.guard storage.front | cancelled=true`.

Goldens: re-record `^TestPin(Order|Blocks)$`. `order.golden` gains the line `BlockBreak LOW storage.front` after `BlockBreak LOW island.guard`. The check script prints `handler-blocks: 33 scenarios, 15 traces changed`: `storage.front` appears after `island.guard` in the 14 `BlockBreak` lines that are not refused before it, and

    BlockBreak/drawerFront: expand.guard island.guard storage.front | cancelled=true drops=minecraft:stone:0x1 xp=0

### `content/skygear/drawerfront_test.go` (new file)

~~~~go
package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/storage"
	"github.com/presentkim/skygear/content/skygear/tinker"
	"github.com/presentkim/skygear/platform/account"
	"github.com/presentkim/skygear/platform/instance"
)

// TestDrawerFrontHitIsNoBreak: a break that ends a hit on a drawer's front is
// cancelled (the hit took items out), and a cancelled break is no break for
// the subscribers after it: it is not counted and the tool in hand gains no
// experience. A break after a hit on another face is a break.
func TestDrawerFrontHitIsNoBreak(t *testing.T) {
	d := bridgeAccount(t)
	s := testSession(t)
	island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
	loadAround(s)
	h := spawnPlayer(s)
	pos := cube.Pos{4, island.IslandY + 2, 4}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		acc := instance.AccountID(p)
		if _, err := d.TouchAccount(acc, "", p.Name()); err != nil {
			t.Fatal(err)
		}
		tx.SetBlock(pos, storage.Drawer{Facing: cube.South}, nil)
		p.SetHeldItems(item.NewStack(tinker.Tool{Kind: tinker.ToolPickaxe, Head: "iron", Handle: "wood"}, 1), item.Stack{})
		hit := func(face cube.Face) *player.Context {
			tx.Block(pos).(storage.Drawer).Punch(pos, face, tx, p)
			drops, xp := []item.Stack{item.NewStack(block.Stone{}, 1)}, 0
			ctx := player.NewEventContext(tx, p)
			s.Handler(p).HandleBlockBreak(ctx, pos, &drops, &xp)
			return ctx
		}
		toolXP := func() int {
			held, _ := p.HeldItems()
			return tinker.ToolXP(held)
		}

		if !hit(cube.South.Face()).Cancelled() {
			t.Fatal("the break after a hit on the drawer's front was not cancelled")
		}
		if got := account.Counter(acc, "block_break"); got != 0 {
			t.Errorf("block_break counted %d for the cancelled break, want 0", got)
		}
		if got := toolXP(); got != 0 {
			t.Errorf("the tool gained %d experience from the cancelled break, want 0", got)
		}

		if hit(cube.North.Face()).Cancelled() {
			t.Fatal("the break after a hit on the drawer's back was cancelled")
		}
		if got := account.Counter(acc, "block_break"); got != 1 {
			t.Errorf("block_break counted %d after the break, want 1", got)
		}
		if got := toolXP(); got != 1 {
			t.Errorf("the tool has %d experience after the break, want 1", got)
		}
	})
}
~~~~

### `content/skygear/pin_blocks_test.go`

~~~~diff
--- a/content/skygear/pin_blocks_test.go
+++ b/content/skygear/pin_blocks_test.go
@@ -277,7 +277,7 @@ func TestPinBlocks(t *testing.T) {
 	// reorders the subscribers (TestTierOrder) has other traces.
 	rows := map[string]string{
 		"StartBreak/own":   "expand.guard island.guard tinker.autoTool",
-		"BlockBreak/stone": "expand.guard island.guard tinker.break machine.break conduit.break storage.break farm.break netstore.break achievement.break",
+		"BlockBreak/stone": "expand.guard island.guard storage.front tinker.break machine.break conduit.break storage.break farm.break netstore.break achievement.break",
 		"BlockPlace/dirt":  "expand.guard island.guard machine.place storage.place farm.plant autofill.place netstore.place achievement.place progress.dragonEgg conduit.place",
 	}
 	for name, row := range rows {
@@ -293,8 +293,8 @@ func TestPinBlocks(t *testing.T) {
 			t.Errorf("%s/expanding: %q, want prefix %q", name, got, want)
 		}
 	}
-	if got := lines["BlockBreak/drawerFront"]; hookOrder == "" && !strings.HasPrefix(got, rows["BlockBreak/stone"]+" | cancelled=true") {
-		t.Errorf("BlockBreak/drawerFront: %q", got)
+	if got, want := lines["BlockBreak/drawerFront"], "expand.guard island.guard storage.front | cancelled=true"; hookOrder == "" && !strings.HasPrefix(got, want) {
+		t.Errorf("BlockBreak/drawerFront: %q, want prefix %q", got, want)
 	}
 	if got := lines["BlockBreak/hammerCobble"]; !strings.Contains(got, "drops="+naming.ItemKey(block.Gravel{})+"x1") {
 		t.Errorf("BlockBreak/hammerCobble: %q lacks the hammer's drop", got)
~~~~

### `content/skygear/storage/subscribe.go`

~~~~diff
--- a/content/skygear/storage/subscribe.go
+++ b/content/skygear/storage/subscribe.go
@@ -2,6 +2,7 @@ package storage
 
 import (
 	"github.com/presentkim/skygear/content/skygear/core"
+	"github.com/presentkim/skygear/content/skygear/island"
 	"github.com/presentkim/skygear/content/skygear/machine"
 )
 
@@ -20,17 +21,25 @@ func subscribeDisplays() {
 	})
 }
 
-// Hitting a drawer's front takes items out and never mines it; another drawer
-// loses its display. A set machine breaking or placed invalidates the
-// networks; a placed drawer is set up.
+// Hitting a drawer's front takes items out and never mines it: the break
+// that ends such a hit is refused before any step works on the block (Low),
+// where the island allows building (elsewhere the island's guard refuses it).
+// A drawer that does break loses its display. A set machine breaking or
+// placed invalidates the networks; a placed drawer is set up.
 func subscribeBlocks() {
+	core.BlockBreak.Handle(core.Low, "storage.front", func(e core.BlockBreakEvent) bool {
+		if _, ok := e.Block.(Drawer); !ok {
+			return false
+		}
+		if lp, ok := LastPunch.Of(e.S)[e.P.H()]; ok && lp.Pos == e.Pos && lp.Front && island.CanBuild(e.S, e.Pos) {
+			e.Cancel()
+			return true
+		}
+		return false
+	})
 	core.BlockBreak.On(core.Normal, "storage.break", func(e core.BlockBreakEvent) {
 		switch b := e.Block.(type) {
 		case Drawer:
-			if lp, ok := LastPunch.Of(e.S)[e.P.H()]; ok && lp.Pos == e.Pos && lp.Front {
-				e.Cancel()
-				return
-			}
 			RemoveDisplay(e.S, e.Tx, e.Pos)
 		case machine.Machine:
 			if IsSetMachine(b.Kind) {
~~~~

## REF T135

The stick record is a Monitor subscriber. Prototype patch: `T135.patch`.

Change C7; the two kept pairs of E2 go. `UseStick` no longer counts; `handwork.CountStickOther` does, from the new subscriber `handwork.stickOther` in MONITOR with default delivery: it is called only when nobody cancelled the click.

Before the change `stickother_test.go` fails with `stick_other counted 2 after the crop took the stick's click, want 1` (the stone and the drawer assertions hold before and after).

`tierorder_test.go`: the two `handwork.stick` entries leave `orderKept`. `pin_use_test.go`: the row of `UseOnBlock/plain` gains ` handwork.stickOther` at its end.

Goldens: re-record `^TestPin(Order|Use)$`. `order.golden` gains `UseOnBlock MONITOR handwork.stickOther` as the last `UseOnBlock` line. The check script prints `handler-use: 123 scenarios, 28 traces changed`: ` handwork.stickOther` at the end of the 28 `UseOnBlock` traces that are not cancelled.

### `content/skygear/conduit/subscribe.go`

~~~~diff
--- a/content/skygear/conduit/subscribe.go
+++ b/content/skygear/conduit/subscribe.go
@@ -51,10 +51,9 @@ func subscribeBlocks() {
 // that conduit (a click on the conduit itself is ConduitBundle.Activate).
 // Sneaking places blocks instead.
 //
-// conduit.plate is registered before handwork.stick and fairy.time, which are
-// in the same tier, and takes the click first: a stick on a plate counts no
-// stick use and an hourglass on a plate opens the settings. That order is one
-// of the exceptions the tier order guard keeps (the root's orderKept).
+// conduit.plate is registered before fairy.time, which is in the same tier,
+// and takes the click first: an hourglass on a plate opens the settings. That
+// order is the exception the tier order guard keeps (the root's orderKept).
 func subscribePlate() {
 	core.UseOnBlock.Handle(core.Normal, "conduit.plate", func(e core.UseOnBlockEvent) bool {
 		if !e.P.Sneaking() && UsePlate(e.S, e.P, e.Tx, e.Pos, e.Face, e.Click, e.Held) {
~~~~

### `content/skygear/handwork/handwork.go`

~~~~diff
--- a/content/skygear/handwork/handwork.go
+++ b/content/skygear/handwork/handwork.go
@@ -97,9 +97,6 @@ func UseStick(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, fac
 	switch tx.Block(pos).(type) {
 	case block.Grass, block.Dirt:
 	default:
-		if !stickDebounced(s, p) {
-			achievement.Count(p, "stick_other", 1)
-		}
 		return false
 	}
 	if stickDebounced(s, p) {
@@ -122,6 +119,21 @@ func UseStick(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, fac
 	return true
 }
 
+// CountStickOther records a stick used on a block that is neither grass nor
+// dirt ("stick_other"); the same click sent twice counts once.
+func CountStickOther(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, held item.Stack) {
+	if _, ok := held.Item().(item.Stick); !ok {
+		return
+	}
+	switch tx.Block(pos).(type) {
+	case block.Grass, block.Dirt:
+		return
+	}
+	if !stickDebounced(s, p) {
+		achievement.Count(p, "stick_other", 1)
+	}
+}
+
 // StripFiber drops bark fibers by chance when a bark stripper (any axe, or
 // an item that is a sgitem.BarkStripper) strips a log (Dragonfly strips it
 // after this).
~~~~

### `content/skygear/handwork/subscribe.go`

~~~~diff
--- a/content/skygear/handwork/subscribe.go
+++ b/content/skygear/handwork/subscribe.go
@@ -7,14 +7,10 @@ func subscribe() {
 	subscribeUse()
 }
 
-// A stick opens or fills a chest and strips; a bare hand strips fibre and
-// tills seeds.
-//
-// handwork.stick also counts a stick used on any other block, when the click
-// reaches it. storage.sneakUse and conduit.plate are in the same tier, are
-// registered before it and take such a click first, so it counts nothing:
-// that order is one of the exceptions the tier order guard keeps (the root's
-// orderKept).
+// A stick fills a chest (sneaking) and knocks pebbles out of grass and dirt;
+// a bare hand strips fibre and tills seeds. A stick used on any other block
+// is recorded when the click went through (nothing cancelled it): the record
+// handles the result, Monitor.
 func subscribeUse() {
 	core.UseOnBlock.Handle(core.Normal, "handwork.stick", func(e core.UseOnBlockEvent) bool {
 		if UseStickOnChest(e.S, e.P, e.Tx, e.Pos, e.Held) || UseStick(e.S, e.P, e.Tx, e.Pos, e.Face, e.Held) {
@@ -27,4 +23,7 @@ func subscribeUse() {
 		StripFiber(e.S, e.P, e.Tx, e.Pos, e.Face, e.Held)
 		TillSeeds(e.S, e.P, e.Tx, e.Pos, e.Face, e.Held)
 	})
+	core.UseOnBlock.On(core.Monitor, "handwork.stickOther", func(e core.UseOnBlockEvent) {
+		CountStickOther(e.S, e.P, e.Tx, e.Pos, e.Held)
+	})
 }
~~~~

### `content/skygear/pin_use_test.go`

~~~~diff
--- a/content/skygear/pin_use_test.go
+++ b/content/skygear/pin_use_test.go
@@ -696,7 +696,7 @@ func TestPinUse(t *testing.T) {
 	pinCheckReached(t, names, lines)
 
 	rows := map[string]string{
-		"UseOnBlock/plain": "dimension.hunting dimension.stone expand.guard screens.menuItem island.guard copier.use wand.use machine.noteFace conduit.plate storage.sneakUse fairy.time handwork.stick handwork.gather farm.composter farm.crop island.guardSide expand.guardSide",
+		"UseOnBlock/plain": "dimension.hunting dimension.stone expand.guard screens.menuItem island.guard copier.use wand.use machine.noteFace conduit.plate storage.sneakUse fairy.time handwork.stick handwork.gather farm.composter farm.crop island.guardSide expand.guardSide handwork.stickOther",
 		"Use/plain":        "dimension.hunting dimension.stone screens.menuItem tinker.veinToggle copier.clear wand.options autofill.use",
 	}
 	for name, row := range rows {
~~~~

### `content/skygear/stickother_test.go` (new file)

~~~~go
package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/handwork"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/storage"
	"github.com/presentkim/skygear/platform/account"
	"github.com/presentkim/skygear/platform/instance"
)

// TestStickOtherCountsTheTapNothingTook: "stick_other" records a stick used
// on a block that is neither grass nor dirt, when the click went through:
// no subscriber cancelled it. A click another feature took (the menu of a
// drawer, a crop) is not such a tap, wherever the record's subscriber stands
// among the others.
func TestStickOtherCountsTheTapNothingTook(t *testing.T) {
	d := bridgeAccount(t)
	s := testSession(t)
	island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
	loadAround(s)
	h := spawnPlayer(s)
	pos := cube.Pos{4, island.IslandY + 2, 4}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		acc := instance.AccountID(p)
		if _, err := d.TouchAccount(acc, "", p.Name()); err != nil {
			t.Fatal(err)
		}
		p.SetHeldItems(item.NewStack(item.Stick{}, 1), item.Stack{})
		tap := func() *player.Context {
			delete(handwork.LastStick.Of(s.Session), p.H()) // not the same click sent twice
			ctx := player.NewEventContext(tx, p)
			s.Handler(p).HandleItemUseOnBlock(ctx, pos, cube.FaceUp, mgl64.Vec3{0.5, 0.5, 0.5})
			return ctx
		}
		counted := func() int64 { return account.Counter(acc, "stick_other") }

		tx.SetBlock(pos, block.Stone{}, nil)
		if tap().Cancelled() {
			t.Fatal("a stick on stone was cancelled")
		}
		if got := counted(); got != 1 {
			t.Errorf("stick_other counted %d after a stick on stone, want 1", got)
		}

		tx.SetBlock(pos, storage.Drawer{Facing: cube.South}, nil)
		p.StartSneaking()
		if !tap().Cancelled() {
			t.Fatal("a sneaking stick on a drawer did not take the click")
		}
		p.StopSneaking()
		if got := counted(); got != 1 {
			t.Errorf("stick_other counted %d after the stick opened a drawer's menu, want 1", got)
		}

		tx.SetBlock(pos.Side(cube.FaceDown), block.Farmland{Hydration: 7}, nil)
		tx.SetBlock(pos, block.WheatSeeds{}, nil)
		if !tap().Cancelled() {
			t.Fatal("a crop did not take the click")
		}
		if got := counted(); got != 1 {
			t.Errorf("stick_other counted %d after the crop took the stick's click, want 1", got)
		}
	})
}
~~~~

### `content/skygear/storage/subscribe.go`

~~~~diff
--- a/content/skygear/storage/subscribe.go
+++ b/content/skygear/storage/subscribe.go
@@ -59,10 +59,7 @@ func subscribeBlocks() {
 	})
 }
 
-// A sneaking player uses a drawer with a held item that is no block. It is
-// registered before handwork.stick, in the same tier, and takes the click
-// first (a stick opens the drawer's menu and counts no stick use): that order
-// is one of the exceptions the tier order guard keeps (the root's orderKept).
+// A sneaking player uses a drawer with a held item that is no block.
 func subscribeSneakUse() {
 	core.UseOnBlock.Handle(core.Normal, "storage.sneakUse", func(e core.UseOnBlockEvent) bool {
 		if SneakUse(e.S, e.P, e.Tx, e.Pos, e.Held) {
~~~~

### `content/skygear/tierorder_test.go`

~~~~diff
--- a/content/skygear/tierorder_test.go
+++ b/content/skygear/tierorder_test.go
@@ -43,10 +43,6 @@ var pinDump = os.Getenv("SKYGEAR_PIN_DUMP")
 // kept until it is decided how to remove it (docs/tech/extension.md); the
 // list does not grow.
 var orderKept = []string{
-	// a click the drawer menu or a connector plate takes counts no stick use
-	// (handwork.stick counts "stick_other" when the click reaches it)
-	"UseOnBlock storage.sneakUse handwork.stick",
-	"UseOnBlock conduit.plate handwork.stick",
 	// a connector plate's settings open before the hourglass in hand winds
 	// the machine the plate is on
 	"UseOnBlock conduit.plate fairy.time",
~~~~

## REF T136

A fairy item takes the click like the other held tools. Prototype patch: `T136.patch`.

Change C8; the last kept pair (E3) and the kept-pair mechanism go. `fairy.time` moves to LOW and asks `TimeItem(e.Held)` (new, the cheap test), then `island.CanBuild`, then `UseFairyTime`.

Before the change `hourglassplate_test.go` fails with `the machine is wound by 0 ticks, want 12000` and `2 hourglasses left in hand, want 1`.

`tierorder_test.go`: `orderKept` and its comment are removed, `TestMain` calls `coretest.ReorderHooks()`. `core/coretest/order.go`: `ReorderHooks` and `orderFor` lose the `kept` parameter and the loop that put pairs back. `pin_use_test.go`: in the row of `UseOnBlock/plain`, `fairy.time` moves to just after `island.guard`.

Goldens: re-record `^TestPin(Order|Use)$`. `order.golden`: the line `UseOnBlock NORMAL fairy.time` goes, `UseOnBlock LOW fairy.time` comes after `UseOnBlock LOW island.guard`. The check script prints `handler-use: 123 scenarios, 62 traces changed`.

### `content/skygear/conduit/subscribe.go`

~~~~diff
--- a/content/skygear/conduit/subscribe.go
+++ b/content/skygear/conduit/subscribe.go
@@ -49,11 +49,7 @@ func subscribeBlocks() {
 // A player who clicks a conduit's connector plate on the block it touches
 // opens its settings, whatever they hold, unless the held conduit goes into
 // that conduit (a click on the conduit itself is ConduitBundle.Activate).
-// Sneaking places blocks instead.
-//
-// conduit.plate is registered before fairy.time, which is in the same tier,
-// and takes the click first: an hourglass on a plate opens the settings. That
-// order is the exception the tier order guard keeps (the root's orderKept).
+// Sneaking places blocks instead. A held tool (Low) took its click before.
 func subscribePlate() {
 	core.UseOnBlock.Handle(core.Normal, "conduit.plate", func(e core.UseOnBlockEvent) bool {
 		if !e.P.Sneaking() && UsePlate(e.S, e.P, e.Tx, e.Pos, e.Face, e.Click, e.Held) {
~~~~

### `content/skygear/core/coretest/order.go`

~~~~diff
--- a/content/skygear/core/coretest/order.go
+++ b/content/skygear/core/coretest/order.go
@@ -24,22 +24,22 @@ var reorder sync.Once
 
 // ReorderHooks reorders the subscribers of every hook inside their tiers as
 // the environment asks (HookOrderEnv), once per test binary; without the
-// variable it does nothing. kept are the pairs "<Hook> <first> <second>" of
-// one tier that keep their order (the exceptions the root package lists).
+// variable it does nothing. No pair of subscribers keeps its order: the
+// subscribers of one tier have none.
 //
 // Session calls it, so the tests of a feature package run reordered as soon
-// as the variable is set. The root package calls it from TestMain with its
-// exceptions, before any event.
-func ReorderHooks(kept ...string) {
+// as the variable is set. The root package calls it from TestMain, before any
+// event.
+func ReorderHooks() {
 	reorder.Do(func() {
 		if mode := os.Getenv(HookOrderEnv); mode != "" {
-			core.Reorder(orderFor(mode, kept))
+			core.Reorder(orderFor(mode))
 		}
 	})
 }
 
 // orderFor returns the reordering of mode for core.Reorder.
-func orderFor(mode string, kept []string) func(hook string, subs []core.Subscription) []int {
+func orderFor(mode string) func(hook string, subs []core.Subscription) []int {
 	kind, arg, _ := strings.Cut(mode, ":")
 	var seed uint64
 	switch kind {
@@ -73,34 +73,6 @@ func orderFor(mode string, kept []string) func(hook string, subs []core.Subscrip
 			idx = append(idx, tier...)
 			from = to
 		}
-		// Put the kept pairs back in their order: a first that ended up after
-		// its second moves to just before it.
-		for moved := true; moved; {
-			moved = false
-			for _, k := range kept {
-				f := strings.Fields(k)
-				if len(f) != 3 {
-					panic("coretest: kept pair " + k)
-				}
-				if f[0] != hook {
-					continue
-				}
-				a := slices.IndexFunc(idx, func(i int) bool { return subs[i].Name == f[1] })
-				b := slices.IndexFunc(idx, func(i int) bool { return subs[i].Name == f[2] })
-				if a < 0 || b < 0 {
-					continue // a test binary that did not set up both features
-				}
-				if subs[idx[a]].Tier != subs[idx[b]].Tier {
-					panic("coretest: kept pair " + k + ": the two are not in one tier, their tiers order them")
-				}
-				if a > b {
-					v := idx[a]
-					idx = slices.Delete(idx, a, a+1)
-					idx = slices.Insert(idx, b, v)
-					moved = true
-				}
-			}
-		}
 		return idx
 	}
 }
~~~~

### `content/skygear/fairy/fairytime.go`

~~~~diff
--- a/content/skygear/fairy/fairytime.go
+++ b/content/skygear/fairy/fairytime.go
@@ -48,6 +48,16 @@ func FairyTimeRecipes() {
 	recipe.Register(recipe.NewShaped([]recipe.Item{e, g, e, d, s, d, e, g, e}, item.NewStack(sgitem.FairyHourglass{}, 1), recipe.NewShape(3, 3), "crafting_table"))
 }
 
+// TimeItem reports whether held is an item UseFairyTime looks at: fairy bone
+// meal, a fairy hourglass or plain bone meal.
+func TimeItem(held item.Stack) bool {
+	switch held.Item().(type) {
+	case item.BoneMeal, sgitem.FairyBoneMeal, sgitem.FairyHourglass:
+		return true
+	}
+	return false
+}
+
 // UseFairyTime uses a held fairy bone meal or hourglass, or keeps plain
 // bone meal off a timed crop (a click without sneaking is useCrop's). It
 // reports whether the click was taken: an item that would change nothing
~~~~

### `content/skygear/fairy/subscribe.go`

~~~~diff
--- a/content/skygear/fairy/subscribe.go
+++ b/content/skygear/fairy/subscribe.go
@@ -1,6 +1,9 @@
 package fairy
 
-import "github.com/presentkim/skygear/content/skygear/core"
+import (
+	"github.com/presentkim/skygear/content/skygear/core"
+	"github.com/presentkim/skygear/content/skygear/island"
+)
 
 // subscribe registers the fairy's second, join and touch steps.
 func subscribe() {
@@ -23,16 +26,15 @@ func subscribeJoin() {
 	})
 }
 
-// The fairy's time is told by the item it is used with; a visiting fairy
-// gives its gift to a player who touches it, whichever way.
-//
-// conduit.plate is in the same tier, is registered before fairy.time and
-// takes a click on a connector plate first, hourglass in hand or not: that
-// order is one of the exceptions the tier order guard keeps (the root's
-// orderKept).
+// The fairy's time is told by the item it is used with. Like the other held
+// tools, a fairy item takes the click away from what the block itself would
+// do with it (an hourglass on a machine's connector plate winds the machine):
+// the Low tier. Where the island does not allow building it leaves the click
+// alone: the island's guard refuses it. A visiting fairy gives its gift to a
+// player who touches it, whichever way.
 func subscribeTouch() {
-	core.UseOnBlock.Handle(core.Normal, "fairy.time", func(e core.UseOnBlockEvent) bool {
-		if UseFairyTime(e.S, e.P, e.Tx, e.Pos, e.Held) {
+	core.UseOnBlock.Handle(core.Low, "fairy.time", func(e core.UseOnBlockEvent) bool {
+		if TimeItem(e.Held) && island.CanBuild(e.S, e.Pos) && UseFairyTime(e.S, e.P, e.Tx, e.Pos, e.Held) {
 			e.Cancel()
 			return true
 		}
~~~~

### `content/skygear/hourglassplate_test.go` (new file)

~~~~go
package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/itemnet"
	"github.com/presentkim/skygear/content/skygear/conduit"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// TestHourglassWindsTheMachineBehindAPlate: a fairy hourglass used on the
// connector plate of a machine winds the machine, as it does anywhere else on
// it: the item in hand takes the click before the block's own parts do (the
// held tools are in the Low tier, the plate in Normal). With any other item
// the plate takes the click and nothing is wound.
func TestHourglassWindsTheMachineBehindAPlate(t *testing.T) {
	s := testSession(t)
	island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
	loadAround(s)
	h := spawnPlayer(s)
	pos := cube.Pos{4, island.IslandY + 2, 4}
	click := mgl64.Vec3{1, 0.5, 0.5}
	st := placeTestMachine(s, pos, machine.KindCrusher)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		b := conduit.Bundle{Item: 1}
		b.Modes[cube.FaceWest] = itemnet.Input
		at := pos.Side(cube.FaceEast)
		tx.SetBlock(at, b, nil)
		b.Refresh(at, tx)
		if _, _, _, ok := conduit.PlateHit(tx, pos, cube.FaceEast, click); !ok {
			t.Fatal("the click is not on a connector plate")
		}
		use := func() *player.Context {
			ctx := player.NewEventContext(tx, p)
			s.Handler(p).HandleItemUseOnBlock(ctx, pos, cube.FaceEast, click)
			return ctx
		}

		if !use().Cancelled() {
			t.Fatal("the plate did not take an empty hand's click")
		}
		if st.Boost != 0 {
			t.Fatalf("an empty hand wound the machine: %d", st.Boost)
		}
		p.SetHeldItems(item.NewStack(sgitem.FairyHourglass{}, 2), item.Stack{})
		if !use().Cancelled() {
			t.Fatal("the hourglass did not take the click")
		}
		if st.Boost != machine.HourglassTicks {
			t.Errorf("the machine is wound by %d ticks, want %d", st.Boost, machine.HourglassTicks)
		}
		if held, _ := p.HeldItems(); held.Count() != 1 {
			t.Errorf("%d hourglasses left in hand, want 1", held.Count())
		}
	})
}
~~~~

### `content/skygear/pin_use_test.go`

~~~~diff
--- a/content/skygear/pin_use_test.go
+++ b/content/skygear/pin_use_test.go
@@ -696,7 +696,7 @@ func TestPinUse(t *testing.T) {
 	pinCheckReached(t, names, lines)
 
 	rows := map[string]string{
-		"UseOnBlock/plain": "dimension.hunting dimension.stone expand.guard screens.menuItem island.guard copier.use wand.use machine.noteFace conduit.plate storage.sneakUse fairy.time handwork.stick handwork.gather farm.composter farm.crop island.guardSide expand.guardSide handwork.stickOther",
+		"UseOnBlock/plain": "dimension.hunting dimension.stone expand.guard screens.menuItem island.guard fairy.time copier.use wand.use machine.noteFace conduit.plate storage.sneakUse handwork.stick handwork.gather farm.composter farm.crop island.guardSide expand.guardSide handwork.stickOther",
 		"Use/plain":        "dimension.hunting dimension.stone screens.menuItem tinker.veinToggle copier.clear wand.options autofill.use",
 	}
 	for name, row := range rows {
~~~~

### `content/skygear/tierorder_test.go`

~~~~diff
--- a/content/skygear/tierorder_test.go
+++ b/content/skygear/tierorder_test.go
@@ -25,7 +25,8 @@ import (
 // subscriber may depend on where it stands. TestTierOrder proves it: it runs
 // this test binary again with the subscribers of every hook reordered inside
 // their tiers (coretest.HookOrderEnv: reversed, and shuffled with a few seeds)
-// and compares what every pin scenario did.
+// and compares what every pin scenario did. No pair of subscribers is kept in
+// its order: there is no list of exceptions.
 
 // hookOrder is the reordering of this process: "" (the order of
 // registration), "reverse" or "shuffle:<seed>".
@@ -36,18 +37,6 @@ var hookOrder = os.Getenv(coretest.HookOrderEnv)
 // (pinDumpLines).
 var pinDump = os.Getenv("SKYGEAR_PIN_DUMP")
 
-// orderKept are the exceptions: the pairs of one tier whose order still
-// matters, "<Hook> <first> <second>". The kernel runs first before second
-// because first's feature is installed earlier (features.go), and a reordered
-// run keeps them so. Each is a known dependency on the place inside a tier,
-// kept until it is decided how to remove it (docs/tech/extension.md); the
-// list does not grow.
-var orderKept = []string{
-	// a connector plate's settings open before the hourglass in hand winds
-	// the machine the plate is on
-	"UseOnBlock conduit.plate fairy.time",
-}
-
 // pinAccounts are the accounts of the players the tests spawned, in a run
 // that dumps (spawnPlayer).
 var (
@@ -60,7 +49,7 @@ var (
 )
 
 func TestMain(m *testing.M) {
-	coretest.ReorderHooks(orderKept...)
+	coretest.ReorderHooks()
 	if pinDump != "" {
 		dir, err := os.MkdirTemp("", "pindump")
 		if err != nil {
~~~~

## REF T137

The placement face is noted from the click that goes through. Prototype patch: `T137.patch`.

Change C9. `machine.noteFace` moves to MONITOR, default delivery.

Before the change `noteface_test.go` fails with `a miner placed after the plate took a click faces west, want down as without a click`.

`pin_use_test.go`: in the row of `UseOnBlock/plain`, `machine.noteFace` moves to just before `handwork.stickOther`.

Goldens: re-record `^TestPin(Order|Use)$`. `order.golden`: `UseOnBlock NORMAL machine.noteFace` goes, `UseOnBlock MONITOR machine.noteFace` comes before `UseOnBlock MONITOR handwork.stickOther`. The check script prints `handler-use: 123 scenarios, 51 traces changed`.

### `content/skygear/machine/subscribe.go`

~~~~diff
--- a/content/skygear/machine/subscribe.go
+++ b/content/skygear/machine/subscribe.go
@@ -32,8 +32,11 @@ func subscribeBlocks() {
 }
 
 // A net-shaped machine or a miner in hand remembers the face it is placed on.
+// The note is of the click that goes through to the placement, when every
+// subscriber that may take or refuse the click has run: it handles the
+// result, Monitor.
 func subscribeFace() {
-	core.UseOnBlock.On(core.Normal, "machine.noteFace", func(e core.UseOnBlockEvent) {
+	core.UseOnBlock.On(core.Monitor, "machine.noteFace", func(e core.UseOnBlockEvent) {
 		if m, ok := e.Held.Item().(Machine); ok && (NetShaped(m.Kind) || m.Kind == KindMiner) {
 			NotePlaceFace(e.S, e.P, e.Face)
 		}
~~~~

### `content/skygear/noteface_test.go` (new file)

~~~~go
package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/itemnet"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/machine"
)

// TestRefusedClickNotesNoFace: the face a miner is placed by is noted from
// the click that goes through to the placement. A click another subscriber
// took (here a connector plate opens its settings) places nothing and leaves
// no note for a later placement, wherever machine.noteFace stands among the
// subscribers that take clicks.
func TestRefusedClickNotesNoFace(t *testing.T) {
	s := testSession(t)
	island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
	loadAround(s)
	h := spawnPlayer(s)
	pos := cube.Pos{4, island.IslandY + 2, 4}
	at := cube.Pos{6, island.IslandY + 2, 6}
	place := func() cube.Face {
		withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
			s.Handler(p).HandleBlockPlace(player.NewEventContext(tx, p), at, machine.New(machine.KindMiner))
		})
		var facing cube.Face
		withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
			m, ok := tx.Block(at).(machine.Machine)
			if !ok || m.State == nil {
				t.Fatal("no miner was placed")
			}
			facing = m.State.Facing
		})
		return facing
	}
	unnoted := place() // no click before it

	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		chestWithBundle(tx, pos, itemnet.Input, 5, 9)
		p.SetHeldItems(item.NewStack(machine.New(machine.KindMiner), 1), item.Stack{})
		ctx := player.NewEventContext(tx, p)
		s.Handler(p).HandleItemUseOnBlock(ctx, pos, cube.FaceEast, mgl64.Vec3{1, 0.5, 0.5})
		if !ctx.Cancelled() {
			t.Fatal("the connector plate did not take the click")
		}
	})
	if got := place(); got != unnoted {
		t.Errorf("a miner placed after the plate took a click faces %v, want %v as without a click", got, unnoted)
	}

	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		ctx := player.NewEventContext(tx, p)
		s.Handler(p).HandleItemUseOnBlock(ctx, at.Side(cube.FaceDown), cube.FaceEast, mgl64.Vec3{0.5, 0.5, 0.5})
		if ctx.Cancelled() {
			t.Fatal("a click on a plain block was cancelled")
		}
	})
	if got := place(); got != cube.FaceWest {
		t.Errorf("a miner placed by a click on an east face faces %v, want west", got)
	}
}
~~~~

### `content/skygear/pin_use_test.go`

~~~~diff
--- a/content/skygear/pin_use_test.go
+++ b/content/skygear/pin_use_test.go
@@ -696,7 +696,7 @@ func TestPinUse(t *testing.T) {
 	pinCheckReached(t, names, lines)
 
 	rows := map[string]string{
-		"UseOnBlock/plain": "dimension.hunting dimension.stone expand.guard screens.menuItem island.guard fairy.time copier.use wand.use machine.noteFace conduit.plate storage.sneakUse handwork.stick handwork.gather farm.composter farm.crop island.guardSide expand.guardSide handwork.stickOther",
+		"UseOnBlock/plain": "dimension.hunting dimension.stone expand.guard screens.menuItem island.guard fairy.time copier.use wand.use conduit.plate storage.sneakUse handwork.stick handwork.gather farm.composter farm.crop island.guardSide expand.guardSide machine.noteFace handwork.stickOther",
 		"Use/plain":        "dimension.hunting dimension.stone screens.menuItem tinker.veinToggle copier.clear wand.options autofill.use",
 	}
 	for name, row := range rows {
~~~~

## REF T138

No subscriber stops the others. Prototype patch: `T138.patch`.

The switch. Nothing a test can see changes: after T130 to T137 every `Handle` subscriber cancels wherever it returned true, and the one `Always` subscriber is `look.update`.

What changes, mechanically:

- `core/hook.go` whole (below): `Hook[E]`, `CancelHook[E Cancellable]`, `Option`, `HandleCancelled`; no `Handle`, no `Always`, `Emit` returns nothing.
- `core/events.go`: `Who`, `Actor{Who; Ctx}`, `NoticeEvent`, `RespawnEvent{Who; Pos}`; the 20 cancellable player hooks are `NewCancelHook`.
- `core/handler.go`: `who`, `actor`; `Jump`, `Death`, `Respawn` build their events from `who`.
- The 33 registrations: `X.Handle(tier, name, func(e E) bool { … e.Cancel(); …; return true … return false })` becomes `X.On(tier, name, func(e E) { … e.Cancel(); … })`: the body keeps every statement but the returns (a `return false` guard at the top becomes a plain `return`, an `if cond { …; return true }; return false` becomes `if cond { … }`). `look.update` becomes `On(…, core.HandleCancelled)` with a `func(e core.MoveEvent)`.
- The five subscribers of `Jump` and `Death` take `core.NoticeEvent`.
- `guard` of `expand` and of `island` take `(a core.Actor, pos cube.Pos)` and return nothing; `useStone` returns nothing.
- Root tests: `pin_order_test.go` (the line format, the tiers always sorted), `monitor_test.go` (a registration has 3 or 4 arguments).

Proofs, all of them in the report:

```sh
S=/tmp/claude-0/-home-user-SkyGear/a71fa8d5-6f20-5c3f-ba20-eeb27e13ad8a/scratchpad/T138
mkdir -p $S && go test -c -o $S/before.test ./content/skygear
(cd content/skygear && SKYGEAR_PIN_DUMP=$S/before.dump $S/before.test -test.count=1 >/dev/null 2>&1); sort $S/before.dump > $S/before.sorted
# ... the change ...
go test -c -o $S/root.test ./content/skygear
(cd content/skygear && SKYGEAR_GOLDEN=update $S/root.test -test.count=1 -test.run '^TestPinOrder$')
git status --short content/skygear/testdata    # exactly: M content/skygear/testdata/pin/order.golden
(cd content/skygear && SKYGEAR_PIN_DUMP=$S/after.dump $S/root.test -test.count=1 >/dev/null 2>&1); sort $S/after.dump > $S/after.sorted
cmp $S/before.sorted $S/after.sorted && wc -l < $S/after.sorted      # equal; 3597
(cd content/skygear && $S/root.test -test.count=1)                    # PASS, with TestTierOrder
SKYGEAR_HOOK_ORDER=reverse go test -count=1 ./content/skygear/...
go test -count=1 ./...
```

`order.golden` after the re-record, whole (123 lines; inside a tier sorted by name):

~~~~
StartBreak LOWEST expand.guard
StartBreak LOW island.guard
StartBreak NORMAL tinker.autoTool
BlockBreak LOWEST expand.guard
BlockBreak LOW island.guard
BlockBreak LOW storage.front
BlockBreak NORMAL conduit.break
BlockBreak NORMAL farm.break
BlockBreak NORMAL machine.break
BlockBreak NORMAL netstore.break
BlockBreak NORMAL storage.break
BlockBreak NORMAL tinker.break
BlockBreak MONITOR achievement.break
BlockPlace LOWEST expand.guard
BlockPlace LOW island.guard
BlockPlace NORMAL farm.plant
BlockPlace NORMAL machine.place
BlockPlace NORMAL storage.place
BlockPlace HIGH autofill.place
BlockPlace HIGH netstore.place
BlockPlace MONITOR achievement.place
BlockPlace MONITOR conduit.place
BlockPlace MONITOR progress.dragonEgg
UseOnBlock LOWEST dimension.hunting
UseOnBlock LOWEST dimension.stone
UseOnBlock LOWEST expand.guard
UseOnBlock LOWEST screens.menuItem
UseOnBlock LOW copier.use
UseOnBlock LOW fairy.time
UseOnBlock LOW island.guard
UseOnBlock LOW wand.use
UseOnBlock NORMAL conduit.plate
UseOnBlock NORMAL handwork.gather
UseOnBlock NORMAL handwork.stick
UseOnBlock NORMAL storage.sneakUse
UseOnBlock HIGH farm.composter
UseOnBlock HIGH farm.crop
UseOnBlock HIGHEST expand.guardSide
UseOnBlock HIGHEST island.guardSide
UseOnBlock MONITOR handwork.stickOther
UseOnBlock MONITOR machine.noteFace
UseOnEntity LOWEST dimension.hunting
UseOnEntity NORMAL fairy.touch
AttackEntity LOWEST dimension.hunter
AttackEntity NORMAL fairy.touch
Use LOWEST dimension.hunting
Use LOWEST dimension.stone
Use LOWEST screens.menuItem
Use NORMAL copier.clear
Use NORMAL tinker.veinToggle
Use NORMAL wand.options
Use HIGH autofill.use
Move LOWEST expand.hold
Move HIGHEST island.void
Move MONITOR achievement.walk
Move MONITOR look.update handleCancelled
Sneak NORMAL elevator.down
Sneak NORMAL expand.show
Sneak HIGH farm.crouchGrow
Sneak MONITOR achievement.sneak
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
Death NORMAL dimension.died
Death NORMAL tinker.keepFood
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
Start NORMAL climate.apply
Start NORMAL island.keepLoaded
Join LOWEST screens.menuItem
Join NORMAL climate.showWeather
Join NORMAL core.shapes
Join NORMAL fairy.backfillDust
Join NORMAL island.purgeLost
Join NORMAL progress.sendRecipes
Join HIGH dimension.joined
Join MONITOR screens.board
Leave NORMAL dimension.leave
Leave NORMAL expand.stop
Leave HIGHEST look.clear
Leave HIGHEST progress.recipes
Leave HIGHEST screens.board
NewDay NORMAL island.purgeLost
Second LOW progress.checkHeld
Second NORMAL climate.tick
Second NORMAL dimension.second
Second NORMAL fairy.tick
Second NORMAL farm.barrels
Second NORMAL farm.growCrops
Second NORMAL farm.plants
Second NORMAL machine.monitor
Second NORMAL tinker.effects
Second MONITOR achievement.playSeconds
Second MONITOR progress.collection
Second MONITOR screens.boards
Fast NORMAL dimension.watchdog
Fast NORMAL network.step
Fast MONITOR look.updateAll
Autosave NORMAL conduit.shares
Autosave NORMAL netstore.flush
Close NORMAL conduit.close
Close NORMAL netstore.flush
Close NORMAL storage.displays
ColumnLoad NORMAL conduit.load
ColumnUnload NORMAL conduit.unload
EntityDespawn NORMAL island.lost
~~~~

### `content/skygear/achievement/subscribe.go`

~~~~diff
--- a/content/skygear/achievement/subscribe.go
+++ b/content/skygear/achievement/subscribe.go
@@ -7,7 +7,8 @@ import (
 
 // subscribe registers the achievement counters on the hooks they count. A
 // counter records what happened and changes nothing: it is a Monitor
-// subscriber.
+// subscriber. It is not called for a cancelled event: what was refused did
+// not happen.
 func subscribe() {
 	subscribeSimple()
 	subscribeMoves()
@@ -20,7 +21,7 @@ func subscribeSimple() {
 	core.Sprint.On(core.Monitor, "achievement.sprint", func(e core.PlayerEvent) {
 		Count(e.P, "sprint", 1)
 	})
-	core.Jump.On(core.Monitor, "achievement.jump", func(e core.PlayerEvent) {
+	core.Jump.On(core.Monitor, "achievement.jump", func(e core.NoticeEvent) {
 		Count(e.P, "jump", 1)
 	})
 	core.ItemConsume.On(core.Monitor, "achievement.eat", func(e core.ItemEvent) {
@@ -29,7 +30,7 @@ func subscribeSimple() {
 	core.ItemPickup.On(core.Monitor, "achievement.pickup", func(e core.ItemEvent) {
 		Count(e.P, "pickup", int64(e.Pick.Count()))
 	})
-	core.Death.On(core.Monitor, "achievement.death", func(e core.PlayerEvent) {
+	core.Death.On(core.Monitor, "achievement.death", func(e core.NoticeEvent) {
 		Count(e.P, "death", 1)
 	})
 	core.PunchAir.On(core.Monitor, "achievement.punchAir", func(e core.PlayerEvent) {
~~~~

### `content/skygear/conduit/subscribe.go`

~~~~diff
--- a/content/skygear/conduit/subscribe.go
+++ b/content/skygear/conduit/subscribe.go
@@ -51,11 +51,9 @@ func subscribeBlocks() {
 // that conduit (a click on the conduit itself is ConduitBundle.Activate).
 // Sneaking places blocks instead. A held tool (Low) took its click before.
 func subscribePlate() {
-	core.UseOnBlock.Handle(core.Normal, "conduit.plate", func(e core.UseOnBlockEvent) bool {
+	core.UseOnBlock.On(core.Normal, "conduit.plate", func(e core.UseOnBlockEvent) {
 		if !e.P.Sneaking() && UsePlate(e.S, e.P, e.Tx, e.Pos, e.Face, e.Click, e.Held) {
 			e.Cancel()
-			return true
 		}
-		return false
 	})
 }
~~~~

### `content/skygear/copier/subscribe.go`

~~~~diff
--- a/content/skygear/copier/subscribe.go
+++ b/content/skygear/copier/subscribe.go
@@ -16,14 +16,12 @@ func subscribe() {
 // island's guard refuses it. A held tool takes the click away from what the
 // block itself would do with it: the Low tier.
 func subscribeUse() {
-	core.UseOnBlock.Handle(core.Low, "copier.use", func(e core.UseOnBlockEvent) bool {
+	core.UseOnBlock.On(core.Low, "copier.use", func(e core.UseOnBlockEvent) {
 		if _, ok := e.Held.Item().(sgitem.ConfigCopier); ok && island.CanBuild(e.S, e.Pos) {
 			e.Cancel()
 			_, off := e.P.HeldItems()
 			UseCopier(e.S, e.P, e.Tx, e.Pos, e.Face, e.Click, e.Held, off)
-			return true
 		}
-		return false
 	})
 	core.Use.On(core.Normal, "copier.clear", func(e core.UseEvent) {
 		if _, ok := e.Held.Item().(sgitem.ConfigCopier); ok {
~~~~

### `content/skygear/core/doc.go`

~~~~diff
--- a/content/skygear/core/doc.go
+++ b/content/skygear/core/doc.go
@@ -12,17 +12,20 @@
 //     document per save (the "skygear.state" key of the save's state), so
 //     each part owns some of its fields and a field belongs to one part only.
 //
-// Hooks are how a feature reacts to the server. A Hook is a typed event with
+// Hooks are how a feature reacts to the server. A hook is a typed event with
 // subscribers in six tiers (Priority: Lowest to Monitor), each tier with one
-// role; inside a tier no subscriber depends on its place. On never cancels
-// the event for the subscribers that follow, Handle does by returning true,
-// and Always also receives an event that was cancelled so. PlayerHandler and
-// WorldHandler are the Dragonfly handlers that emit the player and world
-// hooks, and Lifecycle makes the session calls of the instance (join, leave,
-// new day, every second, fast, autosave, close) emit theirs. The events carry
-// the arguments Dragonfly passed, including its pointers, and an Actor to
-// cancel the event in Dragonfly, which is independent of cancelling it for
-// the subscribers.
+// role; the subscribers of one tier have no order among themselves. No
+// subscriber can stop the others: every subscriber of a Hook is called. A
+// CancelHook is the hook of an event that can be cancelled (it embeds Actor):
+// a subscriber cancels it with e.Cancel(), which keeps the action from
+// happening in Dragonfly, and from then on only the subscribers registered
+// with the HandleCancelled option are called. So a subscriber acts by its own
+// conditions, and cancels only when the action or the later subscribers must
+// not happen. PlayerHandler and WorldHandler are the Dragonfly handlers that
+// emit the player and world hooks, and Lifecycle makes the session calls of
+// the instance (join, leave, new day, every second, fast, autosave, close)
+// emit theirs. The events carry the arguments Dragonfly passed, including its
+// pointers.
 //
 // A Feature is what a feature package hands to the game: its name, entity
 // types, settings, item entry edit, creative items, commands, a Setup that
~~~~

### `content/skygear/core/events.go`

~~~~diff
--- a/content/skygear/core/events.go
+++ b/content/skygear/core/events.go
@@ -8,39 +8,52 @@ import (
 	"github.com/go-gl/mathgl/mgl64"
 )
 
-// Actor is what every player event carries: the session, the player, the
-// player's transaction and the Dragonfly context of the event.
-//
-// Ctx is nil for the events Dragonfly cannot cancel (Jump, Death, Respawn);
-// Cancel then does nothing.
+// Who is the session, the player and the player's transaction. It is all a
+// player event carries that Dragonfly does not let a handler cancel (Jump,
+// Death, Respawn): such an event has no Cancel.
+type Who struct {
+	S  *Session
+	P  *player.Player
+	Tx *world.Tx
+}
+
+// Actor is what a player event that can be cancelled carries: Who and the
+// Dragonfly context of the event.
 //
-// Cancelling has two sides. Cancel is Dragonfly's: the action the player
-// asked for does not happen, and the subscribers go on (Cancelled tells). The
-// hook's is a Handle subscriber returning true: the event is cancelled for
-// the subscribers that follow, which do not run unless they were registered
-// with Always, and Dragonfly's action is not cancelled by it. A subscriber
-// does either, both or neither.
+// Cancel cancels the event: the action the player asked for does not happen
+// in Dragonfly, and the subscribers that follow are not called for it, except
+// those registered with the HandleCancelled option (CancelHook). Cancelled
+// tells the state. A subscriber cancels only when the action or the later
+// subscribers must not happen; otherwise it does its work and leaves the
+// event alone.
 type Actor struct {
-	S   *Session
-	P   *player.Player
-	Tx  *world.Tx
+	Who
 	Ctx *player.Context
 }
 
-// Cancel cancels the event in Dragonfly. It does not keep the later
-// subscribers from running.
+// Cancel cancels the event. It does nothing for an event built without a
+// context (a test).
 func (a Actor) Cancel() {
 	if a.Ctx != nil {
 		a.Ctx.Cancel()
 	}
 }
 
-// Cancelled reports whether the event is cancelled in Dragonfly. It is false
-// for the events Dragonfly cannot cancel.
+// Cancelled reports whether the event is cancelled: by a subscriber that ran
+// before, or by Dragonfly.
 func (a Actor) Cancelled() bool {
 	return a.Ctx != nil && a.Ctx.Cancelled()
 }
 
+// cancelState is where the event's cancel state is kept, nil for an event
+// built without a context.
+func (a Actor) cancelState() *world.Context {
+	if a.Ctx == nil {
+		return nil
+	}
+	return a.Ctx.Context
+}
+
 // Pointer fields (Drops, XP, Damage, Pos, Pick) are the pointers Dragonfly
 // passed to the handler, not copies: a subscriber that writes through one
 // changes what Dragonfly does, exactly as a direct handler would. The other
@@ -108,12 +121,18 @@ type MoveEvent struct {
 	Rot  cube.Rotation
 }
 
-// PlayerEvent is an event that has no arguments of its own (Sneak, Sprint,
-// Jump, Death, PunchAir, Chat, BlockPick, HeldSlot, Command).
+// PlayerEvent is a cancellable event that has no arguments of its own (Sneak,
+// Sprint, PunchAir, Chat, BlockPick, HeldSlot, Command).
 type PlayerEvent struct {
 	Actor
 }
 
+// NoticeEvent is a player event Dragonfly only gives notice of (Jump, Death):
+// it cannot be cancelled.
+type NoticeEvent struct {
+	Who
+}
+
 // ItemEvent is a player consuming, picking up or dropping Item. For ItemPickup
 // Pick is the stack Dragonfly will add, which a subscriber may change; it is
 // nil for the other hooks.
@@ -136,9 +155,9 @@ type HurtEvent struct {
 	Damage *float64
 }
 
-// RespawnEvent is a player respawning at *Pos.
+// RespawnEvent is a player respawning at *Pos. It cannot be cancelled.
 type RespawnEvent struct {
-	Actor
+	Who
 	Pos *mgl64.Vec3
 }
 
@@ -182,35 +201,37 @@ type DespawnEvent struct {
 }
 
 // Player hooks, one per method of PlayerHandler. Their subscribers run tier
-// by tier; see Hook for cancelling.
+// by tier. A CancelHook is an event Dragonfly lets a handler cancel; Jump,
+// Death and Respawn are not.
 var (
-	StartBreak   = NewHook[StartBreakEvent]("StartBreak")
-	BlockBreak   = NewHook[BlockBreakEvent]("BlockBreak")
-	BlockPlace   = NewHook[BlockPlaceEvent]("BlockPlace")
-	UseOnBlock   = NewHook[UseOnBlockEvent]("UseOnBlock")
-	UseOnEntity  = NewHook[EntityEvent]("UseOnEntity")
-	AttackEntity = NewHook[EntityEvent]("AttackEntity")
-	Use          = NewHook[UseEvent]("Use")
-	Move         = NewHook[MoveEvent]("Move")
+	StartBreak   = NewCancelHook[StartBreakEvent]("StartBreak")
+	BlockBreak   = NewCancelHook[BlockBreakEvent]("BlockBreak")
+	BlockPlace   = NewCancelHook[BlockPlaceEvent]("BlockPlace")
+	UseOnBlock   = NewCancelHook[UseOnBlockEvent]("UseOnBlock")
+	UseOnEntity  = NewCancelHook[EntityEvent]("UseOnEntity")
+	AttackEntity = NewCancelHook[EntityEvent]("AttackEntity")
+	Use          = NewCancelHook[UseEvent]("Use")
+	Move         = NewCancelHook[MoveEvent]("Move")
 	// Sneak and Sprint are emitted when sneaking or sprinting starts only.
-	Sneak       = NewHook[PlayerEvent]("Sneak")
-	Sprint      = NewHook[PlayerEvent]("Sprint")
-	Jump        = NewHook[PlayerEvent]("Jump")
-	ItemDamage  = NewHook[ItemDamageEvent]("ItemDamage")
-	ItemConsume = NewHook[ItemEvent]("ItemConsume")
-	ItemPickup  = NewHook[ItemEvent]("ItemPickup")
-	ItemDrop    = NewHook[ItemEvent]("ItemDrop")
-	Death       = NewHook[PlayerEvent]("Death")
+	Sneak       = NewCancelHook[PlayerEvent]("Sneak")
+	Sprint      = NewCancelHook[PlayerEvent]("Sprint")
+	Jump        = NewHook[NoticeEvent]("Jump")
+	ItemDamage  = NewCancelHook[ItemDamageEvent]("ItemDamage")
+	ItemConsume = NewCancelHook[ItemEvent]("ItemConsume")
+	ItemPickup  = NewCancelHook[ItemEvent]("ItemPickup")
+	ItemDrop    = NewCancelHook[ItemEvent]("ItemDrop")
+	Death       = NewHook[NoticeEvent]("Death")
 	Respawn     = NewHook[RespawnEvent]("Respawn")
-	Hurt        = NewHook[HurtEvent]("Hurt")
-	PunchAir    = NewHook[PlayerEvent]("PunchAir")
-	Chat        = NewHook[PlayerEvent]("Chat")
-	BlockPick   = NewHook[PlayerEvent]("BlockPick")
-	HeldSlot    = NewHook[PlayerEvent]("HeldSlot")
-	Command     = NewHook[PlayerEvent]("Command")
+	Hurt        = NewCancelHook[HurtEvent]("Hurt")
+	PunchAir    = NewCancelHook[PlayerEvent]("PunchAir")
+	Chat        = NewCancelHook[PlayerEvent]("Chat")
+	BlockPick   = NewCancelHook[PlayerEvent]("BlockPick")
+	HeldSlot    = NewCancelHook[PlayerEvent]("HeldSlot")
+	Command     = NewCancelHook[PlayerEvent]("Command")
 )
 
-// Session and world hooks, emitted by Lifecycle and WorldHandler.
+// Session and world hooks, emitted by Lifecycle and WorldHandler. None of
+// their events can be cancelled: every subscriber is called.
 var (
 	// Load is emitted before the save's first transaction, Start inside it
 	// (Tx is nil for Load).
~~~~

### `content/skygear/core/handler.go`

~~~~diff
--- a/content/skygear/core/handler.go
+++ b/content/skygear/core/handler.go
@@ -21,17 +21,20 @@ var (
 // handles into the hook of the same name. S is read as a field, so an event
 // costs no lookup. The other Handle methods are Dragonfly's no-ops.
 //
-// Each method builds its event once, before the first subscriber runs, and
-// ignores Emit's result: a stop ends the chain, and only a subscriber's
-// Cancel cancels the event.
+// Each method builds its event once, before the first subscriber runs. Emit
+// returns nothing: a subscriber cancels through the event, and Dragonfly
+// reads the context it passed.
 type PlayerHandler struct {
 	player.NopHandler
 	S *Session
 }
 
+func (h *PlayerHandler) who(p *player.Player) Who {
+	return Who{S: h.S, P: p, Tx: p.Tx()}
+}
+
 func (h *PlayerHandler) actor(ctx *player.Context) Actor {
-	p := ctx.Player()
-	return Actor{S: h.S, P: p, Tx: p.Tx(), Ctx: ctx}
+	return Actor{Who: h.who(ctx.Player()), Ctx: ctx}
 }
 
 func (h *PlayerHandler) HandleStartBreak(ctx *player.Context, pos cube.Pos) {
@@ -92,7 +95,7 @@ func (h *PlayerHandler) HandleToggleSprint(ctx *player.Context, after bool) {
 }
 
 func (h *PlayerHandler) HandleJump(p *player.Player) {
-	Jump.Emit(PlayerEvent{Actor{S: h.S, P: p, Tx: p.Tx()}})
+	Jump.Emit(NoticeEvent{h.who(p)})
 }
 
 func (h *PlayerHandler) HandleItemDamage(ctx *player.Context, it item.Stack, damage *int) {
@@ -116,11 +119,11 @@ func (h *PlayerHandler) HandleItemPickup(ctx *player.Context, i *item.Stack) {
 }
 
 func (h *PlayerHandler) HandleDeath(p *player.Player, _ world.DamageSource, _ *bool) {
-	Death.Emit(PlayerEvent{Actor{S: h.S, P: p, Tx: p.Tx()}})
+	Death.Emit(NoticeEvent{h.who(p)})
 }
 
 func (h *PlayerHandler) HandleRespawn(p *player.Player, pos *mgl64.Vec3, _ **world.World) {
-	Respawn.Emit(RespawnEvent{Actor: Actor{S: h.S, P: p, Tx: p.Tx()}, Pos: pos})
+	Respawn.Emit(RespawnEvent{Who: h.who(p), Pos: pos})
 }
 
 func (h *PlayerHandler) HandleBlockPick(ctx *player.Context, _ cube.Pos, _ world.Block) {
~~~~

### `content/skygear/core/handler_test.go`

~~~~diff
--- a/content/skygear/core/handler_test.go
+++ b/content/skygear/core/handler_test.go
@@ -70,6 +70,7 @@ type row struct {
 	nop   func()
 }
 
+// rowOf makes the row of the hook h; for a CancelHook h is the Hook inside it.
 func rowOf[E any](name string, h *Hook[E], call func(x *fix), check func(t *testing.T, x *fix, e E)) row {
 	return row{
 		name: name,
@@ -103,16 +104,17 @@ func sameStack(t *testing.T, what string, got, want item.Stack) {
 	}
 }
 
-func actorIs(t *testing.T, x *fix, a Actor, ctx bool) {
+func whoIs(t *testing.T, x *fix, w Who) {
 	t.Helper()
-	is(t, "S", a.S, x.s)
-	is(t, "P", a.P, x.p)
-	is(t, "Tx", a.Tx, x.tx)
-	if ctx {
-		is(t, "Ctx", a.Ctx, x.ctx)
-	} else {
-		is(t, "Ctx", a.Ctx, (*player.Context)(nil))
-	}
+	is(t, "S", w.S, x.s)
+	is(t, "P", w.P, x.p)
+	is(t, "Tx", w.Tx, x.tx)
+}
+
+func actorIs(t *testing.T, x *fix, a Actor) {
+	t.Helper()
+	whoIs(t, x, a.Who)
+	is(t, "Ctx", a.Ctx, x.ctx)
 }
 
 var (
@@ -133,106 +135,106 @@ func rows() []row {
 	respawn := mgl64.Vec3{9, 9, 9}
 	picked := stack
 	return []row{
-		rowOf("StartBreak", StartBreak, func(x *fix) { x.h.HandleStartBreak(x.ctx, pos) },
-			func(t *testing.T, x *fix, e StartBreakEvent) { actorIs(t, x, e.Actor, true); is(t, "Pos", e.Pos, pos) }),
-		rowOf("BlockBreak", BlockBreak, func(x *fix) { x.h.HandleBlockBreak(x.ctx, pos, &drops, &xp) },
+		rowOf("StartBreak", &StartBreak.hook, func(x *fix) { x.h.HandleStartBreak(x.ctx, pos) },
+			func(t *testing.T, x *fix, e StartBreakEvent) { actorIs(t, x, e.Actor); is(t, "Pos", e.Pos, pos) }),
+		rowOf("BlockBreak", &BlockBreak.hook, func(x *fix) { x.h.HandleBlockBreak(x.ctx, pos, &drops, &xp) },
 			func(t *testing.T, x *fix, e BlockBreakEvent) {
-				actorIs(t, x, e.Actor, true)
+				actorIs(t, x, e.Actor)
 				is(t, "Pos", e.Pos, pos)
 				is(t, "Block", e.Block, x.tx.Block(pos))
 				sameStack(t, "Held", e.Held, stack)
 				is(t, "Drops", e.Drops, &drops)
 				is(t, "XP", e.XP, &xp)
 			}),
-		rowOf("BlockPlace", BlockPlace, func(x *fix) { x.h.HandleBlockPlace(x.ctx, pos, stone) },
+		rowOf("BlockPlace", &BlockPlace.hook, func(x *fix) { x.h.HandleBlockPlace(x.ctx, pos, stone) },
 			func(t *testing.T, x *fix, e BlockPlaceEvent) {
-				actorIs(t, x, e.Actor, true)
+				actorIs(t, x, e.Actor)
 				is(t, "Pos", e.Pos, pos)
 				is(t, "Block", e.Block, stone)
 				sameStack(t, "Held", e.Held, stack)
 			}),
-		rowOf("UseOnBlock", UseOnBlock, func(x *fix) { x.h.HandleItemUseOnBlock(x.ctx, pos, cube.FaceEast, vec) },
+		rowOf("UseOnBlock", &UseOnBlock.hook, func(x *fix) { x.h.HandleItemUseOnBlock(x.ctx, pos, cube.FaceEast, vec) },
 			func(t *testing.T, x *fix, e UseOnBlockEvent) {
-				actorIs(t, x, e.Actor, true)
+				actorIs(t, x, e.Actor)
 				is(t, "Pos", e.Pos, pos)
 				is(t, "Face", e.Face, cube.FaceEast)
 				is(t, "Click", e.Click, vec)
 				sameStack(t, "Held", e.Held, stack)
 			}),
-		rowOf("UseOnEntity", UseOnEntity, func(x *fix) { x.h.HandleItemUseOnEntity(x.ctx, x.p) },
+		rowOf("UseOnEntity", &UseOnEntity.hook, func(x *fix) { x.h.HandleItemUseOnEntity(x.ctx, x.p) },
 			func(t *testing.T, x *fix, e EntityEvent) {
-				actorIs(t, x, e.Actor, true)
+				actorIs(t, x, e.Actor)
 				is[world.Entity](t, "Target", e.Target, x.p)
 			}),
-		rowOf("AttackEntity", AttackEntity, func(x *fix) { x.h.HandleAttackEntity(x.ctx, x.p, nil, nil, nil) },
+		rowOf("AttackEntity", &AttackEntity.hook, func(x *fix) { x.h.HandleAttackEntity(x.ctx, x.p, nil, nil, nil) },
 			func(t *testing.T, x *fix, e EntityEvent) {
-				actorIs(t, x, e.Actor, true)
+				actorIs(t, x, e.Actor)
 				is[world.Entity](t, "Target", e.Target, x.p)
 			}),
-		rowOf("Use", Use, func(x *fix) { x.h.HandleItemUse(x.ctx) },
+		rowOf("Use", &Use.hook, func(x *fix) { x.h.HandleItemUse(x.ctx) },
 			func(t *testing.T, x *fix, e UseEvent) {
-				actorIs(t, x, e.Actor, true)
+				actorIs(t, x, e.Actor)
 				sameStack(t, "Held", e.Held, stack)
 				sameStack(t, "Off", e.Off, off)
 			}),
-		rowOf("Move", Move, func(x *fix) { x.h.HandleMove(x.ctx, vec, rot) },
+		rowOf("Move", &Move.hook, func(x *fix) { x.h.HandleMove(x.ctx, vec, rot) },
 			func(t *testing.T, x *fix, e MoveEvent) {
-				actorIs(t, x, e.Actor, true)
+				actorIs(t, x, e.Actor)
 				is(t, "From", e.From, x.p.Position())
 				is(t, "Pos", e.Pos, vec)
 				is(t, "Rot", e.Rot, rot)
 			}),
-		rowOf("Sneak", Sneak, func(x *fix) { x.h.HandleToggleSneak(x.ctx, true) },
-			func(t *testing.T, x *fix, e PlayerEvent) { actorIs(t, x, e.Actor, true) }),
-		rowOf("Sprint", Sprint, func(x *fix) { x.h.HandleToggleSprint(x.ctx, true) },
-			func(t *testing.T, x *fix, e PlayerEvent) { actorIs(t, x, e.Actor, true) }),
+		rowOf("Sneak", &Sneak.hook, func(x *fix) { x.h.HandleToggleSneak(x.ctx, true) },
+			func(t *testing.T, x *fix, e PlayerEvent) { actorIs(t, x, e.Actor) }),
+		rowOf("Sprint", &Sprint.hook, func(x *fix) { x.h.HandleToggleSprint(x.ctx, true) },
+			func(t *testing.T, x *fix, e PlayerEvent) { actorIs(t, x, e.Actor) }),
 		rowOf("Jump", Jump, func(x *fix) { x.h.HandleJump(x.p) },
-			func(t *testing.T, x *fix, e PlayerEvent) { actorIs(t, x, e.Actor, false) }),
-		rowOf("ItemDamage", ItemDamage, func(x *fix) { x.h.HandleItemDamage(x.ctx, stack, &dmg) },
+			func(t *testing.T, x *fix, e NoticeEvent) { whoIs(t, x, e.Who) }),
+		rowOf("ItemDamage", &ItemDamage.hook, func(x *fix) { x.h.HandleItemDamage(x.ctx, stack, &dmg) },
 			func(t *testing.T, x *fix, e ItemDamageEvent) {
-				actorIs(t, x, e.Actor, true)
+				actorIs(t, x, e.Actor)
 				sameStack(t, "Item", e.Item, stack)
 				is(t, "Damage", e.Damage, &dmg)
 			}),
-		rowOf("ItemConsume", ItemConsume, func(x *fix) { x.h.HandleItemConsume(x.ctx, stack) },
+		rowOf("ItemConsume", &ItemConsume.hook, func(x *fix) { x.h.HandleItemConsume(x.ctx, stack) },
 			func(t *testing.T, x *fix, e ItemEvent) {
-				actorIs(t, x, e.Actor, true)
+				actorIs(t, x, e.Actor)
 				sameStack(t, "Item", e.Item, stack)
 				is(t, "Pick", e.Pick, (*item.Stack)(nil))
 			}),
-		rowOf("ItemPickup", ItemPickup, func(x *fix) { x.h.HandleItemPickup(x.ctx, &picked) },
+		rowOf("ItemPickup", &ItemPickup.hook, func(x *fix) { x.h.HandleItemPickup(x.ctx, &picked) },
 			func(t *testing.T, x *fix, e ItemEvent) {
-				actorIs(t, x, e.Actor, true)
+				actorIs(t, x, e.Actor)
 				sameStack(t, "Item", e.Item, stack)
 				is(t, "Pick", e.Pick, &picked)
 			}),
-		rowOf("ItemDrop", ItemDrop, func(x *fix) { x.h.HandleItemDrop(x.ctx, stack) },
+		rowOf("ItemDrop", &ItemDrop.hook, func(x *fix) { x.h.HandleItemDrop(x.ctx, stack) },
 			func(t *testing.T, x *fix, e ItemEvent) {
-				actorIs(t, x, e.Actor, true)
+				actorIs(t, x, e.Actor)
 				sameStack(t, "Item", e.Item, stack)
 			}),
 		rowOf("Death", Death, func(x *fix) { x.h.HandleDeath(x.p, nil, nil) },
-			func(t *testing.T, x *fix, e PlayerEvent) { actorIs(t, x, e.Actor, false) }),
+			func(t *testing.T, x *fix, e NoticeEvent) { whoIs(t, x, e.Who) }),
 		rowOf("Respawn", Respawn, func(x *fix) { x.h.HandleRespawn(x.p, &respawn, nil) },
 			func(t *testing.T, x *fix, e RespawnEvent) {
-				actorIs(t, x, e.Actor, false)
+				whoIs(t, x, e.Who)
 				is(t, "Pos", e.Pos, &respawn)
 			}),
-		rowOf("Hurt", Hurt, func(x *fix) { x.h.HandleHurt(x.ctx, &hurt, false, nil, nil) },
+		rowOf("Hurt", &Hurt.hook, func(x *fix) { x.h.HandleHurt(x.ctx, &hurt, false, nil, nil) },
 			func(t *testing.T, x *fix, e HurtEvent) {
-				actorIs(t, x, e.Actor, true)
+				actorIs(t, x, e.Actor)
 				is(t, "Damage", e.Damage, &hurt)
 			}),
-		rowOf("PunchAir", PunchAir, func(x *fix) { x.h.HandlePunchAir(x.ctx) },
-			func(t *testing.T, x *fix, e PlayerEvent) { actorIs(t, x, e.Actor, true) }),
-		rowOf("Chat", Chat, func(x *fix) { x.h.HandleChat(x.ctx, nil) },
-			func(t *testing.T, x *fix, e PlayerEvent) { actorIs(t, x, e.Actor, true) }),
-		rowOf("BlockPick", BlockPick, func(x *fix) { x.h.HandleBlockPick(x.ctx, pos, stone) },
-			func(t *testing.T, x *fix, e PlayerEvent) { actorIs(t, x, e.Actor, true) }),
-		rowOf("HeldSlot", HeldSlot, func(x *fix) { x.h.HandleHeldSlotChange(x.ctx, 1, 2) },
-			func(t *testing.T, x *fix, e PlayerEvent) { actorIs(t, x, e.Actor, true) }),
-		rowOf("Command", Command, func(x *fix) { x.h.HandleCommandExecution(x.ctx, cmd.Command{}, nil) },
-			func(t *testing.T, x *fix, e PlayerEvent) { actorIs(t, x, e.Actor, true) }),
+		rowOf("PunchAir", &PunchAir.hook, func(x *fix) { x.h.HandlePunchAir(x.ctx) },
+			func(t *testing.T, x *fix, e PlayerEvent) { actorIs(t, x, e.Actor) }),
+		rowOf("Chat", &Chat.hook, func(x *fix) { x.h.HandleChat(x.ctx, nil) },
+			func(t *testing.T, x *fix, e PlayerEvent) { actorIs(t, x, e.Actor) }),
+		rowOf("BlockPick", &BlockPick.hook, func(x *fix) { x.h.HandleBlockPick(x.ctx, pos, stone) },
+			func(t *testing.T, x *fix, e PlayerEvent) { actorIs(t, x, e.Actor) }),
+		rowOf("HeldSlot", &HeldSlot.hook, func(x *fix) { x.h.HandleHeldSlotChange(x.ctx, 1, 2) },
+			func(t *testing.T, x *fix, e PlayerEvent) { actorIs(t, x, e.Actor) }),
+		rowOf("Command", &Command.hook, func(x *fix) { x.h.HandleCommandExecution(x.ctx, cmd.Command{}, nil) },
+			func(t *testing.T, x *fix, e PlayerEvent) { actorIs(t, x, e.Actor) }),
 
 		rowOf("Load", Load, func(x *fix) { Load.Emit(Tick{S: x.s}) },
 			func(t *testing.T, x *fix, e Tick) { is(t, "S", e.S, x.s); is(t, "Tx", e.Tx, (*world.Tx)(nil)) }),
@@ -378,25 +380,25 @@ func TestSneakAndSprintEmitOnStartOnly(t *testing.T) {
 	})
 }
 
-// TestCancelAndStopAreIndependent: a subscriber may cancel without stopping
-// and stop without cancelling, and neither implies the other.
-func TestCancelAndStopAreIndependent(t *testing.T) {
-	for _, c := range []struct{ cancel, stop bool }{{false, false}, {true, false}, {false, true}, {true, true}} {
+// TestCancelIsDragonflysCancel: a subscriber's Cancel cancels the context
+// Dragonfly passed, and that one state is what keeps the later default
+// subscribers from being called; without a Cancel all of them run.
+func TestCancelIsDragonflysCancel(t *testing.T) {
+	for _, cancel := range []bool{false, true} {
 		withFix(t, nil, func(x *fix) {
 			ran := false
-			UseOnBlock.Handle(Normal, "t.first", func(e UseOnBlockEvent) bool {
-				if c.cancel {
+			UseOnBlock.On(Low, "t.first", func(e UseOnBlockEvent) {
+				if cancel {
 					e.Cancel()
 				}
-				return c.stop
 			})
 			UseOnBlock.On(Normal, "t.second", func(UseOnBlockEvent) { ran = true })
 			x.h.HandleItemUseOnBlock(x.ctx, pos, cube.FaceUp, vec)
-			if got := x.ctx.Cancelled(); got != c.cancel {
-				t.Errorf("cancel=%v stop=%v: cancelled = %v", c.cancel, c.stop, got)
+			if got := x.ctx.Cancelled(); got != cancel {
+				t.Errorf("cancel=%v: the context is cancelled = %v", cancel, got)
 			}
-			if ran == c.stop {
-				t.Errorf("cancel=%v stop=%v: second subscriber ran = %v", c.cancel, c.stop, ran)
+			if ran == cancel {
+				t.Errorf("cancel=%v: the second subscriber ran = %v", cancel, ran)
 			}
 		})
 	}
@@ -404,61 +406,47 @@ func TestCancelAndStopAreIndependent(t *testing.T) {
 
 // TestLeaveDropsThePlayersEntriesLast: Leave drops the player's Players
 // entries when every subscriber has run, so a subscriber of any tier finds
-// them, and a subscriber that cancels the event does not keep them from
-// being dropped.
+// them.
 func TestLeaveDropsThePlayersEntriesLast(t *testing.T) {
 	part := NewPlayers[int]()
-	for _, stop := range []bool{false, true} {
-		withFix(t, nil, func(x *fix) {
-			part.Of(x.s)[x.p.H()] = 7
-			var seen []int
-			var traced []string
-			see := func(e Presence) { seen = append(seen, part.Of(e.S)[e.P.H()]) }
-			Leave.On(Lowest, "t.first", see)
-			Leave.Handle(Normal, "t.stop", func(e Presence) bool { see(e); return stop })
-			Leave.Always(Monitor, "t.last", func(e Presence, _ bool) { see(e) })
-			traceInto(&traced)
-			x.l.Leave(x.p)
-			Trace = nil
-			if !slices.Equal(seen, []int{7, 7, 7}) {
-				t.Errorf("stop=%t: the subscribers saw %v", stop, seen)
-			}
-			if _, ok := part.Of(x.s)[x.p.H()]; ok {
-				t.Errorf("stop=%t: the player's entry was not dropped", stop)
-			}
-			if want := []string{"t.first", "t.stop", "t.last", "core.shapes", "core.parts"}; !slices.Equal(traced, want) {
-				t.Errorf("stop=%t: traced %v, want %v", stop, traced, want)
-			}
-		})
-	}
+	withFix(t, nil, func(x *fix) {
+		part.Of(x.s)[x.p.H()] = 7
+		var seen []int
+		var traced []string
+		see := func(e Presence) { seen = append(seen, part.Of(e.S)[e.P.H()]) }
+		Leave.On(Lowest, "t.first", see)
+		Leave.On(Normal, "t.middle", see)
+		Leave.On(Monitor, "t.last", see)
+		traceInto(&traced)
+		x.l.Leave(x.p)
+		Trace = nil
+		if !slices.Equal(seen, []int{7, 7, 7}) {
+			t.Errorf("the subscribers saw %v", seen)
+		}
+		if _, ok := part.Of(x.s)[x.p.H()]; ok {
+			t.Error("the player's entry was not dropped")
+		}
+		if want := []string{"t.first", "t.middle", "t.last", "core.shapes", "core.parts"}; !slices.Equal(traced, want) {
+			t.Errorf("traced %v, want %v", traced, want)
+		}
+	})
 }
 
-// TestCancelledTellsTheContext: Cancelled reports Dragonfly's cancel, and is
-// false for an event that has no context.
+// TestCancelledTellsTheContext: Cancelled reports the cancel of the context
+// Dragonfly passed, for a subscriber that still runs after it.
 func TestCancelledTellsTheContext(t *testing.T) {
 	withFix(t, nil, func(x *fix) {
-		var before, after, jump bool
+		var before, after bool
 		Sneak.On(Low, "t.before", func(e PlayerEvent) { before = e.Cancelled() })
 		Sneak.On(Normal, "t.cancel", func(e PlayerEvent) { e.Cancel() })
-		Sneak.On(High, "t.after", func(e PlayerEvent) { after = e.Cancelled() })
-		Jump.On(Normal, "t.jump", func(e PlayerEvent) { e.Cancel(); jump = e.Cancelled() })
+		Sneak.On(High, "t.after", func(e PlayerEvent) { after = e.Cancelled() }, HandleCancelled)
 		x.h.HandleToggleSneak(x.ctx, true)
-		x.h.HandleJump(x.p)
-		if before || !after || jump {
-			t.Fatalf("Cancelled: before=%t after=%t jump=%t", before, after, jump)
+		if before || !after || !x.ctx.Cancelled() {
+			t.Fatalf("Cancelled: before=%t after=%t context=%t", before, after, x.ctx.Cancelled())
 		}
 	})
 }
 
-// TestCancelWithoutContext: Cancel on an event Dragonfly cannot cancel does
-// nothing.
-func TestCancelWithoutContext(t *testing.T) {
-	withFix(t, nil, func(x *fix) {
-		Jump.On(Normal, "t.cancel", func(e PlayerEvent) { e.Cancel() })
-		x.h.HandleJump(x.p)
-	})
-}
-
 // TestEventsCarryDragonflyPointers: a subscriber's write through a pointer
 // field reaches the variable Dragonfly passed.
 func TestEventsCarryDragonflyPointers(t *testing.T) {
~~~~

### `content/skygear/core/hook.go` (the whole file)

~~~~go
package core

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/df-mc/dragonfly/server/world"
)

// Trace, when set, is called with a subscriber's name just before Emit runs
// it, and with every Mark. It is nil outside tests and costs one nil check per
// Emit. Set it only while no Emit is running.
//
// While Trace is set, a CancelHook also checks its Monitor subscribers: Emit
// panics when one of them cancelled the event.
var Trace func(name string)

// Mark reports a step that is not a subscriber to Trace. It does nothing when
// Trace is nil.
func Mark(name string) {
	if t := Trace; t != nil {
		t(name)
	}
}

// installing is the feature whose subscriptions are being registered. While it
// is set, every subscriber name must start with it plus a dot.
var installing string

// Priority is the tier a subscriber runs in. A hook runs its tiers in the
// order of the constants. Each tier has one role; a subscriber takes the tier
// of what it does, not a place before or after another subscriber.
type Priority uint8

const (
	// Lowest is preprocessing: for a subscriber that must shape or block the
	// event itself before anything else.
	Lowest Priority = iota
	// Low is for shaping or blocking the event that may happen after the
	// preprocessing.
	Low
	// Normal is the general tier, for what may happen after the
	// preprocessing.
	Normal
	// High is for work of low priority that is not postprocessing.
	High
	// Highest is the last postprocessing that may still shape the event.
	Highest
	// Monitor is for handling the result without shaping or cancelling the
	// event: records, achievements and the like.
	Monitor
)

var priorityNames = [...]string{"LOWEST", "LOW", "NORMAL", "HIGH", "HIGHEST", "MONITOR"}

// String returns the tier's name in capitals, as the order pin writes it.
func (p Priority) String() string {
	if int(p) < len(priorityNames) {
		return priorityNames[p]
	}
	return fmt.Sprintf("Priority(%d)", uint8(p))
}

// Option changes how a subscriber of a CancelHook is called.
type Option uint8

// HandleCancelled makes a subscriber also receive a cancelled event (the
// handleCancelled option of other event systems). Without it a subscriber is
// not called once the event is cancelled. The subscriber reads the state from
// the event: e.Cancelled().
const HandleCancelled Option = 1

// Subscription is one registered subscriber, as Subscriptions lists it.
// HandleCancelled is set for a subscriber that also receives a cancelled
// event (the HandleCancelled option).
type Subscription struct {
	Hook            string
	Name            string
	Feature         string
	Tier            Priority
	HandleCancelled bool
}

// A step is one subscriber. cancelled is set when it also receives a
// cancelled event.
type step[E any] struct {
	tier      Priority
	cancelled bool
	name      string
	feature   string
	f         func(E)
}

// Hook is a typed event with subscribers in tiers, for an event that cannot
// be cancelled: every subscriber is called for every event. The event is
// passed by value, so Emit allocates nothing. CancelHook is the hook of an
// event that can be cancelled.
//
// The tiers run in the order of the Priority constants. The subscribers of
// one tier have no order among themselves: nothing may be written against
// the order they happen to run in (it is the order of registration, which
// only makes a run repeatable, and the tests run them in others). What must
// come before or after belongs in another tier, by its role.
//
// No subscriber can keep another from being called. A subscriber acts when
// its own conditions hold (its state, the item in hand, the block, the mode
// the player is in), not because no subscriber before it acted.
//
// Subscribers are registered at startup, before the first Emit; registering
// afterwards panics. Once registered, Emit only reads the hook, so any number
// of goroutines may Emit at once without a lock. Emit does not recover: a
// panic in a subscriber propagates to the caller unchanged and later
// subscribers do not run.
type Hook[E any] struct {
	name   string
	steps  []step[E] // ascending tier, registration order inside a tier
	sealed atomic.Bool
}

type anyHook interface {
	subscriptions() []Subscription
	reorder(order func(hook string, subs []Subscription) []int)
	reset()
}

var (
	hooksMu sync.Mutex
	hooks   []anyHook
)

// NewHook returns a hook named name and lists it for Subscriptions. Call it
// from package-level variable declarations, so hooks list in declaration
// order.
func NewHook[E any](name string) *Hook[E] {
	h := &Hook[E]{name: name}
	hooksMu.Lock()
	hooks = append(hooks, h)
	hooksMu.Unlock()
	return h
}

// On subscribes f in tier prio. f is called for every event.
func (h *Hook[E]) On(prio Priority, name string, f func(E)) {
	h.add(step[E]{tier: prio, name: name, f: f})
}

// add registers s.
func (h *Hook[E]) add(s step[E]) {
	if s.f == nil {
		panic(fmt.Sprintf("core: hook %q: subscriber %q has a nil func", h.name, s.name))
	}
	if h.sealed.Load() {
		panic(fmt.Sprintf("core: hook %q: subscriber %q registered after the first Emit", h.name, s.name))
	}
	if s.tier > Monitor {
		panic(fmt.Sprintf("core: hook %q: subscriber %q: %v is not a tier", h.name, s.name, s.tier))
	}
	if installing != "" {
		if !strings.HasPrefix(s.name, installing+".") {
			panic(fmt.Sprintf("core: hook %q: subscriber %q of feature %q must be named %q.<step>", h.name, s.name, installing, installing))
		}
		s.feature = installing
	}
	// After the last subscriber of its tier.
	at := len(h.steps)
	for i := range h.steps {
		o := &h.steps[i]
		if o.name == s.name {
			panic(fmt.Sprintf("core: hook %q: subscriber name %q registered twice", h.name, s.name))
		}
		if o.tier > s.tier && at == len(h.steps) {
			at = i
		}
	}
	h.steps = append(h.steps, step[E]{})
	copy(h.steps[at+1:], h.steps[at:])
	h.steps[at] = s
}

// Emit calls every subscriber, tier by tier. It is safe to call from many
// goroutines at once.
func (h *Hook[E]) Emit(e E) {
	if !h.sealed.Load() {
		h.sealed.Store(true)
	}
	tr := Trace
	for i := range h.steps {
		s := &h.steps[i]
		if tr != nil {
			tr(s.name)
		}
		s.f(e)
	}
}

func (h *Hook[E]) subscriptions() []Subscription {
	out := make([]Subscription, len(h.steps))
	for i, s := range h.steps {
		out[i] = Subscription{Hook: h.name, Name: s.name, Feature: s.feature, Tier: s.tier, HandleCancelled: s.cancelled}
	}
	return out
}

func (h *Hook[E]) reset() {
	h.steps = nil
	h.sealed.Store(false)
}

// Cancellable is an event that can be cancelled: one that carries the
// Dragonfly context of the event, which every event that embeds Actor does.
// Cancel and Cancelled exist on these events only, so cancelling an event
// that cannot be cancelled does not compile.
type Cancellable interface {
	Cancel()
	Cancelled() bool
	cancelState() *world.Context
}

// CancelHook is the hook of an event that can be cancelled. It is a Hook
// whose subscribers are not called once the event is cancelled, unless they
// asked for cancelled events with the HandleCancelled option.
//
// An event is cancelled when its Dragonfly context is: a subscriber called
// e.Cancel(), in this walk or before it. Cancelling says two things at once:
// the action the player asked for does not happen in Dragonfly, and the
// subscribers that follow are not called for it. There is no other way to
// keep a subscriber from being called. A subscriber cancels only when that is
// what it means ("in this state I handle this, and nothing else may"); when
// the action may go on, it does its work and leaves the event alone.
//
// Because cancelling skips the subscribers of the same tier that happen to
// run later, two subscribers of one tier that could act on the same event,
// one of them cancelling, must exclude each other by their own conditions or
// stand in different tiers.
type CancelHook[E Cancellable] struct {
	hook Hook[E]
}

// NewCancelHook returns a hook for a cancellable event, named name, and lists
// it for Subscriptions like NewHook.
func NewCancelHook[E Cancellable](name string) *CancelHook[E] {
	h := &CancelHook[E]{}
	h.hook.name = name
	hooksMu.Lock()
	hooks = append(hooks, &h.hook)
	hooksMu.Unlock()
	return h
}

// On subscribes f in tier prio. f is not called for a cancelled event, unless
// the HandleCancelled option is given: then it is called for every event and
// reads e.Cancelled() if it cares. A Monitor subscriber does not cancel.
func (h *CancelHook[E]) On(prio Priority, name string, f func(E), opts ...Option) {
	s := step[E]{tier: prio, name: name, f: f}
	for _, o := range opts {
		if o != HandleCancelled {
			panic(fmt.Sprintf("core: hook %q: subscriber %q: unknown option %d", h.hook.name, name, o))
		}
		s.cancelled = true
	}
	h.hook.add(s)
}

// Emit calls the subscribers tier by tier. Once the event is cancelled, only
// the subscribers registered with HandleCancelled are still called. It is
// safe to call from many goroutines at once.
func (h *CancelHook[E]) Emit(e E) {
	c := e.cancelState()
	if c == nil {
		// An event built without a context (a test) is never cancelled.
		h.hook.Emit(e)
		return
	}
	if !h.hook.sealed.Load() {
		h.hook.sealed.Store(true)
	}
	tr := Trace
	steps := h.hook.steps
	for i := range steps {
		s := &steps[i]
		if c.Cancelled() && !s.cancelled {
			continue
		}
		if tr == nil {
			s.f(e)
			continue
		}
		tr(s.name)
		was := c.Cancelled()
		s.f(e)
		if s.tier == Monitor && !was && c.Cancelled() {
			panic(fmt.Sprintf("core: hook %q: Monitor subscriber %q cancelled the event", h.hook.name, s.name))
		}
	}
}

// Subscriptions lists every subscriber: hooks in declaration order, then each
// hook's subscribers in the order they run.
func Subscriptions() []Subscription {
	hooksMu.Lock()
	defer hooksMu.Unlock()
	var out []Subscription
	for _, h := range hooks {
		out = append(out, h.subscriptions()...)
	}
	return out
}
~~~~

### `content/skygear/core/hook_test.go` (the whole file)

~~~~go
package core

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

type bigEvent struct {
	A [12]uint64 // 96 bytes
}

// bigCancel is bigEvent as an event that can be cancelled.
type bigCancel struct {
	Actor
	A [12]uint64
}

// newContext returns a context nobody cancelled, as Dragonfly passes one.
func newContext() *player.Context { return &player.Context{Context: &world.Context{}} }

func newBigCancel() bigCancel { return bigCancel{Actor: Actor{Ctx: newContext()}} }

func mustPanic(t *testing.T, want ...string) {
	t.Helper()
	r := recover()
	if r == nil {
		t.Fatal("expected a panic")
	}
	msg := fmt.Sprint(r)
	for _, w := range want {
		if !strings.Contains(msg, w) {
			t.Fatalf("panic %q lacks %q", msg, w)
		}
	}
}

func traceInto(got *[]string) {
	Trace = func(name string) { *got = append(*got, name) }
}

// TestRunsInTierOrder: the tiers run in the order of the constants, whatever
// the order of registration, on both kinds of hook and with the option.
func TestRunsInTierOrder(t *testing.T) {
	ResetHooks()
	want := []string{"lowest", "low", "normal", "high", "highest", "monitor"}
	h := NewHook[int]("order")
	var got []string
	rec := func(name string) func(int) { return func(int) { got = append(got, name) } }
	h.On(Monitor, "monitor", rec("monitor"))
	h.On(Highest, "highest", rec("highest"))
	h.On(High, "high", rec("high"))
	h.On(Normal, "normal", rec("normal"))
	h.On(Low, "low", rec("low"))
	h.On(Lowest, "lowest", rec("lowest"))
	h.Emit(1)
	if !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}

	c := NewCancelHook[bigCancel]("order-cancel")
	got = nil
	recC := func(name string) func(bigCancel) { return func(bigCancel) { got = append(got, name) } }
	c.On(Monitor, "monitor", recC("monitor"), HandleCancelled)
	c.On(Highest, "highest", recC("highest"))
	c.On(High, "high", recC("high"), HandleCancelled)
	c.On(Normal, "normal", recC("normal"))
	c.On(Low, "low", recC("low"))
	c.On(Lowest, "lowest", recC("lowest"), HandleCancelled)
	c.Emit(newBigCancel())
	if !slices.Equal(got, want) {
		t.Fatalf("cancel hook: got %v want %v", got, want)
	}
}

// TestSameTierRunsInRegistrationOrder: what happens inside a tier is the
// order of registration, which makes a run repeatable (no subscriber may be
// written against it); a later registration in an earlier tier still runs
// before them.
func TestSameTierRunsInRegistrationOrder(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("tier")
	var got []string
	rec := func(name string) func(int) { return func(int) { got = append(got, name) } }
	h.On(Normal, "b.x", rec("b.x"))
	h.On(Normal, "a.x", rec("a.x"))
	h.On(Monitor, "m.x", rec("m.x"))
	h.On(Normal, "c.x", rec("c.x"))
	h.On(Low, "l.x", rec("l.x"))
	h.Emit(0)
	want := []string{"l.x", "b.x", "a.x", "c.x", "m.x"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
	var listed []string
	for _, s := range Subscriptions() {
		if s.Hook == "tier" {
			listed = append(listed, s.Name)
		}
	}
	if !slices.Equal(listed, want) {
		t.Fatalf("Subscriptions lists %v, want %v", listed, want)
	}
}

func TestPriorityNames(t *testing.T) {
	var got []string
	for p := Lowest; p <= Monitor; p++ {
		got = append(got, p.String())
	}
	if want := []string{"LOWEST", "LOW", "NORMAL", "HIGH", "HIGHEST", "MONITOR"}; !slices.Equal(got, want) {
		t.Fatalf("got %v", got)
	}
	if got := Priority(9).String(); got != "Priority(9)" {
		t.Fatalf("got %q", got)
	}
}

func TestNotATierPanics(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("notier")
	defer mustPanic(t, "notier", "a.x", "Priority(6)", "not a tier")
	h.On(Monitor+1, "a.x", func(int) {})
}

// TestUnknownOptionPanics: HandleCancelled is the only option.
func TestUnknownOptionPanics(t *testing.T) {
	ResetHooks()
	h := NewCancelHook[bigCancel]("option")
	defer mustPanic(t, "option", "a.x", "unknown option")
	h.On(Normal, "a.x", func(bigCancel) {}, Option(7))
}

// TestNoSubscriberStopsAnother: every subscriber of a hook is called; on a
// hook whose event cannot be cancelled there is nothing that skips one.
func TestNoSubscriberStopsAnother(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("all")
	var traced, ran []string
	traceInto(&traced)
	defer func() { Trace = nil }()
	for i, tier := range []Priority{Lowest, Low, Normal, Normal, High, Highest, Monitor} {
		name := fmt.Sprint("s", i)
		h.On(tier, name, func(int) { ran = append(ran, name) })
	}
	h.Emit(1)
	want := []string{"s0", "s1", "s2", "s3", "s4", "s5", "s6"}
	if !slices.Equal(ran, want) || !slices.Equal(traced, want) {
		t.Fatalf("ran %v traced %v", ran, traced)
	}
}

// cancelCase is one arrangement of TestCancelSkipsDefaultSubscribers: the
// tier of the subscriber that may cancel, and the tier of the two
// subscribers under test (one default, one with HandleCancelled), registered
// before or after it.
type cancelCase struct {
	cancelTier, tier Priority
	before           bool
}

// TestCancelSkipsDefaultSubscribers: a default subscriber and one with
// HandleCancelled, in every tier, before and after a subscriber that may
// cancel in every tier but Monitor, for an event nobody cancels, one that
// subscriber cancels and one that was cancelled before Emit.
//
//   - a live event: both run and read Cancelled() == false;
//   - an event cancelled in the walk: a subscriber that stands before the
//     cancelling one ran already (and read false); one that stands after it
//     runs only with HandleCancelled, and reads true;
//   - an event cancelled before the walk: only the HandleCancelled
//     subscriber runs, and reads true; the cancelling subscriber, a default
//     one, is not called either.
func TestCancelSkipsDefaultSubscribers(t *testing.T) {
	var cases []cancelCase
	for cancel := Lowest; cancel <= Highest; cancel++ {
		for tier := Lowest; tier <= Monitor; tier++ {
			if tier != cancel {
				cases = append(cases, cancelCase{cancel, tier, tier < cancel})
				continue
			}
			cases = append(cases, cancelCase{cancel, tier, true}, cancelCase{cancel, tier, false})
		}
	}
	for _, c := range cases {
		for _, mode := range []string{"live", "cancelled", "before"} {
			withFix(t, nil, func(x *fix) {
				var ran []string
				under := func() {
					Sneak.On(c.tier, "t.default", func(e PlayerEvent) {
						ran = append(ran, fmt.Sprint("default=", e.Cancelled()))
					})
					Sneak.On(c.tier, "t.cancelled", func(e PlayerEvent) {
						ran = append(ran, fmt.Sprint("cancelled=", e.Cancelled()))
					}, HandleCancelled)
				}
				if c.before {
					under()
				}
				Sneak.On(c.cancelTier, "t.cancel", func(e PlayerEvent) {
					ran = append(ran, "cancel")
					if mode == "cancelled" {
						e.Cancel()
					}
				})
				if !c.before {
					under()
				}
				if mode == "before" {
					x.ctx.Cancel()
				}
				Sneak.Emit(PlayerEvent{x.h.actor(x.ctx)})
				var want []string
				switch {
				case mode == "before":
					want = []string{"cancelled=true"}
				case c.before:
					want = []string{"default=false", "cancelled=false", "cancel"}
				case mode == "cancelled":
					want = []string{"cancel", "cancelled=true"}
				default:
					want = []string{"cancel", "default=false", "cancelled=false"}
				}
				if !slices.Equal(ran, want) {
					t.Errorf("%+v %s: ran %v, want %v", c, mode, ran, want)
				}
				if got := x.ctx.Cancelled(); got != (mode != "live") {
					t.Errorf("%+v %s: the context is cancelled = %t", c, mode, got)
				}
			})
		}
	}
	if len(cases) != 35 {
		t.Fatalf("%d arrangements", len(cases))
	}
}

// TestAfterCancelOnlyHandleCancelledRuns: after a cancel the subscribers
// with HandleCancelled still to come run in order, traced like any other,
// whatever stands between them; a default subscriber that would cancel too is
// not called, and a skipped subscriber is not traced.
func TestAfterCancelOnlyHandleCancelledRuns(t *testing.T) {
	ResetHooks()
	h := NewCancelHook[bigCancel]("after")
	var traced, ran []string
	traceInto(&traced)
	defer func() { Trace = nil }()
	rec := func(name string) func(bigCancel) {
		return func(e bigCancel) { ran = append(ran, fmt.Sprint(name, "=", e.Cancelled())) }
	}
	h.On(Lowest, "a", rec("a"), HandleCancelled)
	h.On(Low, "cancel", func(e bigCancel) { ran = append(ran, "cancel"); e.Cancel() })
	h.On(Low, "b", rec("b"), HandleCancelled)
	h.On(Normal, "on", rec("on"))
	h.On(Normal, "cancel2", func(e bigCancel) { ran = append(ran, "cancel2"); e.Cancel() })
	h.On(High, "c", rec("c"), HandleCancelled)
	h.On(Monitor, "monitor", rec("monitor"))
	h.On(Monitor, "d", rec("d"), HandleCancelled)
	e := newBigCancel()
	h.Emit(e)
	if !e.Cancelled() {
		t.Fatal("the event is not cancelled")
	}
	if want := []string{"a=false", "cancel", "b=true", "c=true", "d=true"}; !slices.Equal(ran, want) {
		t.Fatalf("ran %v, want %v", ran, want)
	}
	if want := []string{"a", "cancel", "b", "c", "d"}; !slices.Equal(traced, want) {
		t.Fatalf("traced %v, want %v", traced, want)
	}
	var opted []string
	for _, s := range Subscriptions() {
		if s.Hook == "after" && s.HandleCancelled {
			opted = append(opted, s.Name)
		}
	}
	if want := []string{"a", "b", "c", "d"}; !slices.Equal(opted, want) {
		t.Fatalf("Subscriptions marks %v as HandleCancelled, want %v", opted, want)
	}
}

// TestEventWithoutContextIsNeverCancelled: an event built without a context
// (a test that emits by hand) reaches every subscriber; Cancel does nothing.
func TestEventWithoutContextIsNeverCancelled(t *testing.T) {
	ResetHooks()
	h := NewCancelHook[bigCancel]("nocontext")
	n := 0
	h.On(Low, "a", func(e bigCancel) { n++; e.Cancel() })
	h.On(Normal, "b", func(e bigCancel) {
		if n++; e.Cancelled() {
			t.Error("an event without a context is cancelled")
		}
	})
	h.On(Monitor, "c", func(bigCancel) { n++ }, HandleCancelled)
	h.Emit(bigCancel{})
	if n != 3 {
		t.Fatalf("%d subscribers ran, want 3", n)
	}
}

// TestMonitorCancelPanicsWhileTraced: a Monitor subscriber does not cancel.
// While Trace is set (the tests that trace) Emit panics when one did; a
// Monitor subscriber that receives an event cancelled before it is fine.
func TestMonitorCancelPanicsWhileTraced(t *testing.T) {
	ResetHooks()
	h := NewCancelHook[bigCancel]("monitor")
	var traced []string
	cancel := false
	h.On(Normal, "first", func(e bigCancel) {
		if cancel {
			e.Cancel()
		}
	})
	h.On(Monitor, "bad.x", func(e bigCancel) { e.Cancel() }, HandleCancelled)
	h.Emit(newBigCancel()) // not traced: no check
	traceInto(&traced)
	defer func() { Trace = nil }()
	cancel = true
	h.Emit(newBigCancel()) // cancelled before the Monitor subscriber: its Cancel changes nothing
	cancel = false
	defer mustPanic(t, "monitor", "bad.x", "Monitor subscriber", "cancelled the event")
	h.Emit(newBigCancel())
}

// TestOnlyActorEventsCancel: Cancel and Cancelled exist exactly on the
// events that carry the Dragonfly context (they embed Actor); the events of
// Jump, Death, Respawn and of the session and world hooks have neither, so a
// subscriber that tried to cancel one would not compile.
func TestOnlyActorEventsCancel(t *testing.T) {
	cancellable := []any{StartBreakEvent{}, BlockBreakEvent{}, BlockPlaceEvent{}, UseOnBlockEvent{}, EntityEvent{}, UseEvent{}, MoveEvent{}, PlayerEvent{}, ItemEvent{}, ItemDamageEvent{}, HurtEvent{}}
	for _, e := range cancellable {
		if _, ok := e.(Cancellable); !ok {
			t.Errorf("%T cannot be cancelled", e)
		}
	}
	for _, e := range []any{NoticeEvent{}, RespawnEvent{}, Tick{}, DayEvent{}, Presence{}, ColumnEvent{}, DespawnEvent{}, Who{}} {
		typ := reflect.TypeOf(e)
		for _, m := range []string{"Cancel", "Cancelled"} {
			if _, has := typ.MethodByName(m); has {
				t.Errorf("%v has %s: an event that cannot be cancelled must not have it", typ, m)
			}
		}
		if _, ok := typ.FieldByName("Ctx"); ok {
			t.Errorf("%v has a Ctx field", typ)
		}
	}
}

func TestSameNamePanics(t *testing.T) {
	ResetHooks()
	h := NewCancelHook[bigCancel]("name")
	h.On(Normal, "a.x", func(bigCancel) {})
	defer mustPanic(t, "name", "a.x")
	h.On(High, "a.x", func(bigCancel) {}, HandleCancelled)
}

func TestNilFuncPanics(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("nilf")
	c := NewCancelHook[bigCancel]("nilc")
	func() {
		defer mustPanic(t, "nilc", "h")
		c.On(Normal, "h", nil)
	}()
	func() {
		defer mustPanic(t, "nilc", "w")
		c.On(Normal, "w", nil, HandleCancelled)
	}()
	defer mustPanic(t, "nilf", "a")
	h.On(Normal, "a", nil)
}

func TestInstallingPrefix(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("pfx")
	SetInstalling("feat")
	h.On(Normal, "feat.ok", func(int) {})
	func() {
		defer mustPanic(t, "pfx", "other.bad", "feat")
		h.On(Normal, "other.bad", func(int) {})
	}()
	found := false
	for _, s := range Subscriptions() {
		if s.Hook == "pfx" {
			found = true
			if s.Feature != "feat" || s.Name != "feat.ok" || s.Tier != Normal || s.HandleCancelled {
				t.Fatalf("subscription %+v", s)
			}
		}
	}
	if !found {
		t.Fatal("subscription missing")
	}
}

func TestRegisterAfterEmitPanics(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("late")
	h.On(Normal, "a", func(int) {})
	h.Emit(0)
	defer mustPanic(t, "late", "b")
	h.On(Normal, "b", func(int) {})
}

// TestRegisterAfterEmitPanicsOnACancelHook: after the first Emit, of a live
// event, a cancelled one or one without a context, a CancelHook refuses a
// subscriber with and without the option, and refuses neither before it.
func TestRegisterAfterEmitPanicsOnACancelHook(t *testing.T) {
	kinds := map[string]func(h *CancelHook[bigCancel], name string){
		"default":   func(h *CancelHook[bigCancel], name string) { h.On(Normal, name, func(bigCancel) {}) },
		"cancelled": func(h *CancelHook[bigCancel], name string) { h.On(Monitor, name, func(bigCancel) {}, HandleCancelled) },
	}
	events := map[string]func() bigCancel{
		"live":      newBigCancel,
		"cancelled": func() bigCancel { e := newBigCancel(); e.Cancel(); return e },
		"nocontext": func() bigCancel { return bigCancel{} },
	}
	for _, event := range events {
		for kind, add := range kinds {
			ResetHooks()
			h := NewCancelHook[bigCancel]("late")
			for other, add := range kinds {
				add(h, "before."+other)
			}
			h.Emit(event())
			func() {
				defer mustPanic(t, "late", "after."+kind, "after the first Emit")
				add(h, "after."+kind)
			}()
		}
	}
}

// TestEmitConcurrentWithCancel: many goroutines emit events of their own
// that a subscriber cancels; for every event the subscribers before the
// cancel run once, the HandleCancelled subscribers after it run once and
// read true, the default subscribers after it never run.
func TestEmitConcurrentWithCancel(t *testing.T) {
	ResetHooks()
	h := NewCancelHook[bigCancel]("conccancel")
	var before, optedBeforeLive, optedBeforeCancelled, after, optedAfterLive, optedAfterCancelled atomic.Int64
	count := func(live, cancelled *atomic.Int64) func(bigCancel) {
		return func(e bigCancel) {
			if e.Cancelled() {
				cancelled.Add(1)
			} else {
				live.Add(1)
			}
		}
	}
	h.On(Lowest, "on.before", func(bigCancel) { before.Add(1) })
	h.On(Low, "opted.before", count(&optedBeforeLive, &optedBeforeCancelled), HandleCancelled)
	h.On(Normal, "cancel", func(e bigCancel) { e.Cancel() })
	h.On(Normal, "on.after", func(bigCancel) { after.Add(1) })
	h.On(High, "opted.after", count(&optedAfterLive, &optedAfterCancelled), HandleCancelled)
	h.On(Monitor, "monitor.after", func(bigCancel) { after.Add(1) })
	const goroutines, emits = 16, 1000
	var notCancelled atomic.Int64
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < emits; i++ {
				e := newBigCancel()
				if h.Emit(e); !e.Cancelled() {
					notCancelled.Add(1)
				}
			}
		}()
	}
	wg.Wait()
	const n = goroutines * emits
	if notCancelled.Load() != 0 || before.Load() != n || optedBeforeLive.Load() != n || optedBeforeCancelled.Load() != 0 ||
		after.Load() != 0 || optedAfterLive.Load() != 0 || optedAfterCancelled.Load() != n {
		t.Fatalf("not cancelled %d, before %d, opted before %d/%d (live/cancelled), default after %d, opted after %d/%d",
			notCancelled.Load(), before.Load(), optedBeforeLive.Load(), optedBeforeCancelled.Load(),
			after.Load(), optedAfterLive.Load(), optedAfterCancelled.Load())
	}
}

func TestEmptyEmitStillSealsRegistration(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("early")
	h.Emit(0) // no steps: an empty emit still counts as emission
	defer mustPanic(t, "early")
	h.On(Normal, "a", func(int) {})
}

func TestSubscriptionsRunOrder(t *testing.T) {
	ResetHooks()
	h := NewCancelHook[bigCancel]("T")
	h.On(High, "b.x", func(bigCancel) {}, HandleCancelled)
	h.On(Low, "a.x", func(bigCancel) {})
	var got []Subscription
	for _, s := range Subscriptions() {
		if s.Hook == "T" {
			got = append(got, s)
		}
	}
	want := []Subscription{{Hook: "T", Tier: Low, Name: "a.x"}, {Hook: "T", Tier: High, Name: "b.x", HandleCancelled: true}}
	if !slices.Equal(got, want) {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestSubscriptionsHookDeclarationOrder(t *testing.T) {
	ResetHooks()
	h1 := NewHook[int]("decl1")
	h2 := NewCancelHook[bigCancel]("decl2")
	h2.On(Lowest, "z", func(bigCancel) {})
	h1.On(Monitor, "y", func(int) {})
	var names []string
	for _, s := range Subscriptions() {
		if s.Hook == "decl1" || s.Hook == "decl2" {
			names = append(names, s.Name)
		}
	}
	if !slices.Equal(names, []string{"y", "z"}) {
		t.Fatalf("got %v", names)
	}
}

func TestTraceNamesInOrder(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("trace")
	var traced, ran []string
	h.On(Normal, "b", func(int) { ran = append(ran, fmt.Sprint(len(traced), ":b")) })
	h.On(Low, "a", func(int) { ran = append(ran, fmt.Sprint(len(traced), ":a")) })
	traceInto(&traced)
	defer func() { Trace = nil }()
	h.Emit(0)
	if !slices.Equal(traced, []string{"a", "b"}) {
		t.Fatalf("traced %v", traced)
	}
	// Trace ran just before each step: 1 name seen by a, 2 by b.
	if !slices.Equal(ran, []string{"1:a", "2:b"}) {
		t.Fatalf("ran %v", ran)
	}
}

func TestMarkCallsTrace(t *testing.T) {
	ResetHooks()
	Mark("nobody listens") // nil Trace is fine
	var got []string
	traceInto(&got)
	defer func() { Trace = nil }()
	Mark("m1")
	Mark("m2")
	if !slices.Equal(got, []string{"m1", "m2"}) {
		t.Fatalf("got %v", got)
	}
}

func TestSubscriberPanicPropagates(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("boom")
	later := false
	h.On(Normal, "a", func(int) { panic("boom!") })
	h.On(Normal, "b", func(int) { later = true })
	func() {
		defer func() {
			if r := recover(); r != "boom!" {
				t.Fatalf("recovered %v", r)
			}
		}()
		h.Emit(0)
	}()
	if later {
		t.Fatal("later step ran after a panic")
	}
}

func TestEmitConcurrent(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("conc")
	var n atomic.Int64
	for i := 0; i < 8; i++ {
		h.On(Priority(i%6), fmt.Sprint("s", i), func(e int) { n.Add(int64(e)) })
	}
	var wg sync.WaitGroup
	for g := 0; g < 16; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				h.Emit(1)
			}
		}()
	}
	wg.Wait()
	if n.Load() != 16*1000*8 {
		t.Fatalf("n=%d", n.Load())
	}
}

// fillHook gives h n subscribers spread over the tiers.
func fillHook(h *Hook[bigEvent], n int) {
	for i := 0; i < n; i++ {
		h.On(Priority(i%6), fmt.Sprint("s", i), func(e bigEvent) { _ = e.A[0] })
	}
}

// fillCancelHook gives h n subscribers spread over the tiers that may cancel,
// none of which cancels the events the tests emit, every fourth with
// HandleCancelled.
func fillCancelHook(h *CancelHook[bigCancel], n int) {
	for i := 0; i < n; i++ {
		f := func(e bigCancel) {
			if e.A[0] == 99 {
				e.Cancel()
			}
		}
		if i%4 == 3 {
			h.On(Priority(i%5), fmt.Sprint("s", i), f, HandleCancelled)
		} else {
			h.On(Priority(i%5), fmt.Sprint("s", i), f)
		}
	}
}

func TestEmitAllocatesNothing(t *testing.T) {
	for _, n := range []int{0, 1, 4, 8, 32} {
		ResetHooks()
		h := NewHook[bigEvent](fmt.Sprintf("alloc-%d", n))
		fillHook(h, n)
		var e bigEvent
		e.A[0] = 1
		if got := testing.AllocsPerRun(200, func() { h.Emit(e) }); got != 0 {
			t.Errorf("hook, %d subscribers: %v allocs", n, got)
		}
		c := NewCancelHook[bigCancel](fmt.Sprintf("alloc-cancel-%d", n))
		fillCancelHook(c, n)
		ce := newBigCancel()
		ce.A[0] = 1
		if got := testing.AllocsPerRun(200, func() { c.Emit(ce) }); got != 0 {
			t.Errorf("cancel hook, %d subscribers: %v allocs", n, got)
		}
	}
}

// TestEmitAllocatesNothingAfterCancel: the walk over a cancelled event, in
// which the default subscribers are skipped, allocates nothing either.
func TestEmitAllocatesNothingAfterCancel(t *testing.T) {
	ResetHooks()
	h := NewCancelHook[bigCancel]("alloc-cancelled")
	seen, skipped := 0, 0
	h.On(Lowest, "cancel", func(e bigCancel) { e.Cancel() })
	h.On(Normal, "on", func(bigCancel) { skipped++ })
	h.On(Normal, "a", func(e bigCancel) {
		if e.Cancelled() {
			seen++
		}
	}, HandleCancelled)
	h.On(Monitor, "b", func(e bigCancel) {
		if e.Cancelled() {
			seen++
		}
	}, HandleCancelled)
	e := newBigCancel()
	if got := testing.AllocsPerRun(200, func() { h.Emit(e) }); got != 0 {
		t.Errorf("%v allocs", got)
	}
	if seen == 0 || seen%2 != 0 || skipped != 0 {
		t.Fatalf("the HandleCancelled subscribers saw %d cancelled events, the default one ran %d times", seen, skipped)
	}
}

// BenchmarkEmit emits to n subscribers: of a hook whose event cannot be
// cancelled (hook=plain) and of a CancelHook whose event nobody cancels
// (hook=cancel), which asks the context before every default subscriber.
func BenchmarkEmit(b *testing.B) {
	for _, n := range []int{0, 1, 4, 8, 32} {
		b.Run(fmt.Sprintf("hook=plain/subs=%d", n), func(b *testing.B) {
			ResetHooks()
			h := NewHook[bigEvent](fmt.Sprintf("bench-%d", n))
			fillHook(h, n)
			var e bigEvent
			e.A[0] = 1
			b.ReportAllocs()
			for b.Loop() {
				h.Emit(e)
			}
		})
	}
	for _, n := range []int{0, 1, 4, 8, 32} {
		b.Run(fmt.Sprintf("hook=cancel/subs=%d", n), func(b *testing.B) {
			ResetHooks()
			h := NewCancelHook[bigCancel](fmt.Sprintf("bench-cancel-%d", n))
			fillCancelHook(h, n)
			e := newBigCancel()
			e.A[0] = 1
			b.ReportAllocs()
			for b.Loop() {
				h.Emit(e)
			}
		})
	}
}

var moveSink float64

// BenchmarkEmitMove emits a MoveEvent to four subscribers, the shape of the
// move hook: one that may cancel first, one that may cancel last, a record,
// and a record with HandleCancelled.
func BenchmarkEmitMove(b *testing.B) {
	ResetHooks()
	h := NewCancelHook[MoveEvent]("bench-move")
	h.On(Monitor, "a", func(e MoveEvent) { moveSink += e.Rot[0] + e.Pos[0] }, HandleCancelled)
	h.On(Lowest, "b", func(e MoveEvent) {
		if e.P == nil && e.Pos[1] > 1e9 {
			e.Cancel()
		}
	})
	h.On(Monitor, "c", func(e MoveEvent) { moveSink += e.Pos[2] })
	h.On(Highest, "d", func(e MoveEvent) {
		if e.Pos[1] < -1e9 {
			e.Cancel()
		}
	})
	e := MoveEvent{Actor: Actor{Ctx: newContext()}, Pos: [3]float64{1, 64, 2}}
	b.ReportAllocs()
	for b.Loop() {
		h.Emit(e)
	}
}
~~~~

### `content/skygear/core/kernel.go`

~~~~diff
--- a/content/skygear/core/kernel.go
+++ b/content/skygear/core/kernel.go
@@ -1,8 +1,7 @@
 package core
 
-// KernelFeature is the kernel's own feature; the game lists it first, so its
-// subscriber is the first of its tier. It is a subscriber like any other: it
-// does not depend on its place inside the tier.
+// KernelFeature is the kernel's own feature; the game lists it first. Its
+// subscriber is one like any other: the subscribers of its tier have no order.
 var KernelFeature = &Feature{Name: "core", Setup: subscribeKernel}
 
 // Shapes are sent when a player joins. What the kernel does when a player
~~~~

### `content/skygear/core/lifecycle.go`

~~~~diff
--- a/content/skygear/core/lifecycle.go
+++ b/content/skygear/core/lifecycle.go
@@ -23,8 +23,8 @@ func (l Lifecycle) Join(p *player.Player, first bool) {
 
 // Leave emits Leave, then clears the shapes on the player's client and drops
 // the player's Players entries (Session.Leave). Both come after every
-// subscriber, whatever its tier and whether or not one cancelled the event: a
-// subscriber still finds the player's entries, and what it drew is cleared.
+// subscriber, whatever its tier: a subscriber still finds the player's
+// entries, and what it drew is cleared.
 func (l Lifecycle) Leave(p *player.Player) {
 	Leave.Emit(Presence{S: l.Session, P: p, Tx: p.Tx()})
 	Mark("core.shapes")
~~~~

### `content/skygear/core/newfeature_test.go`

~~~~diff
--- a/content/skygear/core/newfeature_test.go
+++ b/content/skygear/core/newfeature_test.go
@@ -11,13 +11,13 @@ import "testing"
 func TestNewFeatureNeedsOnlyTheList(t *testing.T) {
 	ResetHooks()
 	defer ResetHooks()
-	var got []PlayerEvent
+	var got []NoticeEvent
 	fake := &Feature{Name: "fake", Setup: func() {
-		Jump.On(Normal, "fake.jump", func(e PlayerEvent) { got = append(got, e) })
+		Jump.On(Normal, "fake.jump", func(e NoticeEvent) { got = append(got, e) })
 	}}
 	Install(fake)
 
-	Jump.Emit(PlayerEvent{})
+	Jump.Emit(NoticeEvent{})
 	if len(got) != 1 {
 		t.Fatalf("the fake feature received %d events, want 1", len(got))
 	}
~~~~

### `content/skygear/core/reorder.go`

~~~~diff
--- a/content/skygear/core/reorder.go
+++ b/content/skygear/core/reorder.go
@@ -2,8 +2,8 @@ package core
 
 import "fmt"
 
-// Reorder is a seam for the tests that prove no subscriber depends on its
-// place inside its tier. It changes the order the subscribers of every hook
+// Reorder is a seam for the tests that prove the subscribers of one tier
+// have no order. It changes the order the subscribers of every hook
 // run in: order is called once per hook that has subscribers, with the hook's
 // name and its subscribers in the order they run, and returns the new order
 // as indexes into subs. The new order may only move a subscriber inside its
@@ -44,5 +44,4 @@ func (h *Hook[E]) reorder(order func(hook string, subs []Subscription) []int) {
 		}
 	}
 	h.steps = steps
-	h.index()
 }
~~~~

### `content/skygear/core/reorder_test.go`

~~~~diff
--- a/content/skygear/core/reorder_test.go
+++ b/content/skygear/core/reorder_test.go
@@ -77,21 +77,23 @@ func TestReorderKeepsTheTiers(t *testing.T) {
 	Reorder(func(string, []Subscription) []int { return []int{1, 0} })
 }
 
-// TestReorderKeepsAlways: after a reordering the Always subscribers still
-// receive a cancelled event, wherever they stand now.
-func TestReorderKeepsAlways(t *testing.T) {
+// TestReorderKeepsHandleCancelled: after a reordering a subscriber with
+// HandleCancelled still receives a cancelled event, wherever it stands now,
+// and a default one that now stands after the cancel does not.
+func TestReorderKeepsHandleCancelled(t *testing.T) {
 	ResetHooks()
-	h := NewHook[int]("reorder-always")
+	h := NewCancelHook[bigCancel]("reorder-cancelled")
 	var ran []string
-	h.Always(Normal, "w", func(_ int, cancelled bool) {
-		if cancelled {
+	h.On(Normal, "w", func(e bigCancel) {
+		if e.Cancelled() {
 			ran = append(ran, "w")
 		}
-	})
-	h.Handle(Normal, "stop", func(int) bool { return true })
-	h.On(Normal, "on", func(int) { ran = append(ran, "on") })
+	}, HandleCancelled)
+	h.On(Normal, "cancel", func(e bigCancel) { e.Cancel() })
+	h.On(Normal, "on", func(bigCancel) { ran = append(ran, "on") })
 	Reorder(func(string, []Subscription) []int { return []int{1, 2, 0} })
-	if !h.Emit(0) || !slices.Equal(ran, []string{"w"}) {
+	e := newBigCancel()
+	if h.Emit(e); !e.Cancelled() || !slices.Equal(ran, []string{"w"}) {
 		t.Fatalf("ran %v", ran)
 	}
 }
~~~~

### `content/skygear/dimension/subscribe.go`

~~~~diff
--- a/content/skygear/dimension/subscribe.go
+++ b/content/skygear/dimension/subscribe.go
@@ -21,7 +21,7 @@ func subscribe() {
 
 // The second dimension's reaction to a death and a respawn.
 func subscribeLife() {
-	core.Death.On(core.Normal, "dimension.died", func(e core.PlayerEvent) {
+	core.Death.On(core.Normal, "dimension.died", func(e core.NoticeEvent) {
 		Died(e.S, e.P.Tx(), e.P)
 	})
 	core.Respawn.On(core.Low, "dimension.respawned", func(e core.RespawnEvent) {
@@ -32,26 +32,20 @@ func subscribeLife() {
 // The second dimension's reaction to a drop and a hurt. Whether a hit would
 // kill the hunter is asked last, of the damage the other steps left: Highest.
 func subscribeHurt() {
-	core.ItemDrop.Handle(core.Lowest, "dimension.hunting", func(e core.ItemEvent) bool {
+	core.ItemDrop.On(core.Lowest, "dimension.hunting", func(e core.ItemEvent) {
 		if Hunting(e.S, e.P) {
 			e.Cancel()
-			return true
 		}
-		return false
 	})
-	core.Hurt.Handle(core.Lowest, "dimension.frozen", func(e core.HurtEvent) bool {
+	core.Hurt.On(core.Lowest, "dimension.frozen", func(e core.HurtEvent) {
 		if Frozen(e.S, e.P) {
 			e.Cancel() // the hunt waits while the hunter chooses an upgrade
-			return true
 		}
-		return false
 	})
-	core.Hurt.Handle(core.Highest, "dimension.fallen", func(e core.HurtEvent) bool {
+	core.Hurt.On(core.Highest, "dimension.fallen", func(e core.HurtEvent) {
 		if Fallen(e.S, e.Tx, e.P, *e.Damage) {
 			e.Cancel() // no hurt for the subscribers after it: the run ended instead
-			return true
 		}
-		return false
 	})
 }
 
@@ -91,53 +85,44 @@ func subscribePresence() {
 // and the stone's window come before everything else: Lowest. They exclude
 // each other and the menu item's step by their own conditions.
 func subscribeUse() {
-	core.UseOnBlock.Handle(core.Lowest, "dimension.hunting", func(e core.UseOnBlockEvent) bool {
+	core.UseOnBlock.On(core.Lowest, "dimension.hunting", func(e core.UseOnBlockEvent) {
 		if Hunting(e.S, e.P) {
 			e.Cancel()
-			return true
 		}
-		return false
 	})
-	core.UseOnBlock.Handle(core.Lowest, "dimension.stone", func(e core.UseOnBlockEvent) bool {
-		return useStone(e.Actor, e.Held)
+	core.UseOnBlock.On(core.Lowest, "dimension.stone", func(e core.UseOnBlockEvent) {
+		useStone(e.Actor, e.Held)
 	})
-	core.Use.Handle(core.Lowest, "dimension.hunting", func(e core.UseEvent) bool {
+	core.Use.On(core.Lowest, "dimension.hunting", func(e core.UseEvent) {
 		// the menu item in the air is the hunter's way out (screens.menuItem)
 		if Hunting(e.S, e.P) && !sgitem.IsMenuItem(e.Held) {
 			e.Cancel()
-			return true
 		}
-		return false
 	})
-	core.Use.Handle(core.Lowest, "dimension.stone", func(e core.UseEvent) bool {
-		return useStone(e.Actor, e.Held)
+	core.Use.On(core.Lowest, "dimension.stone", func(e core.UseEvent) {
+		useStone(e.Actor, e.Held)
 	})
-	core.UseOnEntity.Handle(core.Lowest, "dimension.hunting", func(e core.EntityEvent) bool {
+	core.UseOnEntity.On(core.Lowest, "dimension.hunting", func(e core.EntityEvent) {
 		if Hunting(e.S, e.P) {
 			e.Cancel()
-			return true
 		}
-		return false
 	})
-	core.AttackEntity.Handle(core.Lowest, "dimension.hunter", func(e core.EntityEvent) bool {
+	core.AttackEntity.On(core.Lowest, "dimension.hunter", func(e core.EntityEvent) {
 		if _, ok := e.Target.(*Monster); ok || Hunting(e.S, e.P) {
 			e.Cancel()
-			return true
 		}
-		return false
 	})
 }
 
 // useStone opens the second dimension's window for a held teleport stone,
-// on a block or the air; it reports whether it took the click. A hunter's
-// stone opens nothing: the hunter's use is refused (dimension.hunting).
-func useStone(a core.Actor, held item.Stack) bool {
+// on a block or the air, and cancels the click it took. A hunter's stone
+// opens nothing: the hunter's use is refused (dimension.hunting).
+func useStone(a core.Actor, held item.Stack) {
 	if _, ok := held.Item().(sgitem.DimensionStone); !ok || Hunting(a.S, a.P) {
-		return false
+		return
 	}
 	a.Cancel()
 	if !a.S.Debounced(a.P) {
 		ShowWindow(a.S, a.P, "")
 	}
-	return true
 }
~~~~

### `content/skygear/elevator/subscribe.go`

~~~~diff
--- a/content/skygear/elevator/subscribe.go
+++ b/content/skygear/elevator/subscribe.go
@@ -10,7 +10,7 @@ func subscribe() {
 
 // A jump on an elevator rides it up.
 func subscribeJump() {
-	core.Jump.On(core.Normal, "elevator.up", func(e core.PlayerEvent) {
+	core.Jump.On(core.Normal, "elevator.up", func(e core.NoticeEvent) {
 		Up(e.S, e.P, e.Tx)
 	})
 }
~~~~

### `content/skygear/expand/subscribe.go`

~~~~diff
--- a/content/skygear/expand/subscribe.go
+++ b/content/skygear/expand/subscribe.go
@@ -26,12 +26,10 @@ func subscribe() {
 // is not cancelled, and the subscribers that have no business with a player
 // in a mode say so themselves (core.Takeover).
 func subscribeHold() {
-	core.Move.Handle(core.Lowest, "expand.hold", func(e core.MoveEvent) bool {
+	core.Move.On(core.Lowest, "expand.hold", func(e core.MoveEvent) {
 		if _, expanding := Of(e.S, e.P); expanding {
 			e.Cancel()
-			return true
 		}
-		return false
 	})
 	core.Sneak.On(core.Normal, "expand.show", func(e core.PlayerEvent) {
 		if _, ok := Of(e.S, e.P); ok {
@@ -54,41 +52,38 @@ func subscribeLeave() {
 // the server resend the real blocks, so the map chunk is shown again. The mode
 // refuses before anything else looks at the event: Lowest.
 func subscribeGuard() {
-	core.StartBreak.Handle(core.Lowest, "expand.guard", func(e core.StartBreakEvent) bool {
-		return guard(e.S, e.Ctx, e.Pos)
+	core.StartBreak.On(core.Lowest, "expand.guard", func(e core.StartBreakEvent) {
+		guard(e.Actor, e.Pos)
 	})
-	core.BlockBreak.Handle(core.Lowest, "expand.guard", func(e core.BlockBreakEvent) bool {
-		return guard(e.S, e.Ctx, e.Pos)
+	core.BlockBreak.On(core.Lowest, "expand.guard", func(e core.BlockBreakEvent) {
+		guard(e.Actor, e.Pos)
 	})
-	core.BlockPlace.Handle(core.Lowest, "expand.guard", func(e core.BlockPlaceEvent) bool {
-		return guard(e.S, e.Ctx, e.Pos)
+	core.BlockPlace.On(core.Lowest, "expand.guard", func(e core.BlockPlaceEvent) {
+		guard(e.Actor, e.Pos)
 	})
 }
 
-// guard cancels the action at pos and reports true for a player in island
-// expansion mode.
-func guard(s *core.Session, ctx *player.Context, pos cube.Pos) bool {
-	if _, expanding := Of(s, ctx.Player()); expanding {
-		ctx.Cancel()
-		RestoreMapNear(s, ctx.Player(), pos)
-		return true
+// guard cancels the action at pos for a player in island expansion mode and
+// shows the map chunk there again.
+func guard(a core.Actor, pos cube.Pos) {
+	if _, expanding := Of(a.S, a.P); expanding {
+		a.Cancel()
+		RestoreMapNear(a.S, a.P, pos)
 	}
-	return false
 }
 
 // The block clicked and the block next to its face are guarded alike. An
 // item that opens its screen wherever it is used (the menu item, the dimension
 // stone) keeps its click: its own step takes it. The side is refused last,
-// when no step took the click: Highest.
+// when no subscriber cancelled the click: Highest.
 func subscribeSide() {
-	core.UseOnBlock.Handle(core.Lowest, "expand.guard", func(e core.UseOnBlockEvent) bool {
-		if sgitem.OpensScreen(e.Held) {
-			return false
+	core.UseOnBlock.On(core.Lowest, "expand.guard", func(e core.UseOnBlockEvent) {
+		if !sgitem.OpensScreen(e.Held) {
+			guard(e.Actor, e.Pos)
 		}
-		return guard(e.S, e.Ctx, e.Pos)
 	})
-	core.UseOnBlock.Handle(core.Highest, "expand.guardSide", func(e core.UseOnBlockEvent) bool {
-		return guard(e.S, e.Ctx, e.Pos.Side(e.Face))
+	core.UseOnBlock.On(core.Highest, "expand.guardSide", func(e core.UseOnBlockEvent) {
+		guard(e.Actor, e.Pos.Side(e.Face))
 	})
 }
 
~~~~

### `content/skygear/fairy/subscribe.go`

~~~~diff
--- a/content/skygear/fairy/subscribe.go
+++ b/content/skygear/fairy/subscribe.go
@@ -33,12 +33,10 @@ func subscribeJoin() {
 // alone: the island's guard refuses it. A visiting fairy gives its gift to a
 // player who touches it, whichever way.
 func subscribeTouch() {
-	core.UseOnBlock.Handle(core.Low, "fairy.time", func(e core.UseOnBlockEvent) bool {
+	core.UseOnBlock.On(core.Low, "fairy.time", func(e core.UseOnBlockEvent) {
 		if TimeItem(e.Held) && island.CanBuild(e.S, e.Pos) && UseFairyTime(e.S, e.P, e.Tx, e.Pos, e.Held) {
 			e.Cancel()
-			return true
 		}
-		return false
 	})
 	core.UseOnEntity.On(core.Normal, "fairy.touch", touchFairy)
 	core.AttackEntity.On(core.Normal, "fairy.touch", touchFairy)
~~~~

### `content/skygear/farm/subscribe.go`

~~~~diff
--- a/content/skygear/farm/subscribe.go
+++ b/content/skygear/farm/subscribe.go
@@ -100,18 +100,14 @@ func subscribeBlocks() {
 // harvests, and places nothing. What the block does with a click comes after
 // what the held item does with it (Normal: a stick, the fairy's items): High.
 func subscribeUse() {
-	core.UseOnBlock.Handle(core.High, "farm.composter", func(e core.UseOnBlockEvent) bool {
+	core.UseOnBlock.On(core.High, "farm.composter", func(e core.UseOnBlockEvent) {
 		if !e.P.Sneaking() && UseComposter(e.S, e.P, e.Tx, e.Pos, e.Held) {
 			e.Cancel()
-			return true
 		}
-		return false
 	})
-	core.UseOnBlock.Handle(core.High, "farm.crop", func(e core.UseOnBlockEvent) bool {
+	core.UseOnBlock.On(core.High, "farm.crop", func(e core.UseOnBlockEvent) {
 		if !e.P.Sneaking() && UseCrop(e.S, e.P, e.Tx, e.Pos) {
 			e.Cancel()
-			return true
 		}
-		return false
 	})
 }
~~~~

### `content/skygear/handwork/subscribe.go`

~~~~diff
--- a/content/skygear/handwork/subscribe.go
+++ b/content/skygear/handwork/subscribe.go
@@ -12,12 +12,10 @@ func subscribe() {
 // is recorded when the click went through (nothing cancelled it): the record
 // handles the result, Monitor.
 func subscribeUse() {
-	core.UseOnBlock.Handle(core.Normal, "handwork.stick", func(e core.UseOnBlockEvent) bool {
+	core.UseOnBlock.On(core.Normal, "handwork.stick", func(e core.UseOnBlockEvent) {
 		if UseStickOnChest(e.S, e.P, e.Tx, e.Pos, e.Held) || UseStick(e.S, e.P, e.Tx, e.Pos, e.Face, e.Held) {
 			e.Cancel()
-			return true
 		}
-		return false
 	})
 	core.UseOnBlock.On(core.Normal, "handwork.gather", func(e core.UseOnBlockEvent) {
 		StripFiber(e.S, e.P, e.Tx, e.Pos, e.Face, e.Held)
~~~~

### `content/skygear/island/subscribe.go`

~~~~diff
--- a/content/skygear/island/subscribe.go
+++ b/content/skygear/island/subscribe.go
@@ -3,7 +3,6 @@ package island
 import (
 	"github.com/df-mc/dragonfly/server/block/cube"
 	"github.com/df-mc/dragonfly/server/entity"
-	"github.com/df-mc/dragonfly/server/player"
 	"github.com/presentkim/skygear/common/i18n"
 	"github.com/presentkim/skygear/content/skygear/core"
 )
@@ -18,15 +17,13 @@ func subscribe() {
 
 // A player that falls below the island returns from the void: the last step
 // that may cancel the move, Highest. The cancelled move is no move for the
-// subscribers after it (the walk counter).
+// subscribers after it (the walk counter is not called).
 func subscribeVoid() {
-	core.Move.Handle(core.Highest, "island.void", func(e core.MoveEvent) bool {
+	core.Move.On(core.Highest, "island.void", func(e core.MoveEvent) {
 		if e.Pos.Y() < VoidY {
 			e.Cancel()
 			ReturnFromVoid(e.S, e.P, e.Tx)
-			return true
 		}
-		return false
 	})
 }
 
@@ -61,38 +58,37 @@ func subscribeLost() {
 // the modes that hold a player (Lowest) and before any step works on the
 // block: Low.
 func subscribeGuard() {
-	core.StartBreak.Handle(core.Low, "island.guard", func(e core.StartBreakEvent) bool {
-		return guard(e.S, e.Ctx, e.Pos)
+	core.StartBreak.On(core.Low, "island.guard", func(e core.StartBreakEvent) {
+		guard(e.Actor, e.Pos)
 	})
-	core.BlockBreak.Handle(core.Low, "island.guard", func(e core.BlockBreakEvent) bool {
-		return guard(e.S, e.Ctx, e.Pos)
+	core.BlockBreak.On(core.Low, "island.guard", func(e core.BlockBreakEvent) {
+		guard(e.Actor, e.Pos)
 	})
-	core.BlockPlace.Handle(core.Low, "island.guard", func(e core.BlockPlaceEvent) bool {
-		return guard(e.S, e.Ctx, e.Pos)
+	core.BlockPlace.On(core.Low, "island.guard", func(e core.BlockPlaceEvent) {
+		guard(e.Actor, e.Pos)
 	})
 }
 
-// guard reports whether the action at pos is refused because pos is not the
-// island's to build on; it cancels the action then.
-func guard(s *core.Session, ctx *player.Context, pos cube.Pos) bool {
-	if CanBuild(s, pos) {
-		return false
+// guard cancels the action at pos when pos is not the island's to build on,
+// and tells the player so.
+func guard(a core.Actor, pos cube.Pos) {
+	if CanBuild(a.S, pos) {
+		return
 	}
-	ctx.Cancel()
+	a.Cancel()
 	if core.InArena(pos.Vec3()) {
-		return true // the hunting ground is no one's to build on
+		return // the hunting ground is no one's to build on
 	}
-	ctx.Player().SendTip(i18n.T(i18n.Default, "island.outside"))
-	return true
+	a.P.SendTip(i18n.T(i18n.Default, "island.outside"))
 }
 
 // The block clicked and the block next to its face are guarded alike. The side
-// is refused last, when no step took the click: Highest.
+// is refused last, when no subscriber cancelled the click: Highest.
 func subscribeSide() {
-	core.UseOnBlock.Handle(core.Low, "island.guard", func(e core.UseOnBlockEvent) bool {
-		return guard(e.S, e.Ctx, e.Pos)
+	core.UseOnBlock.On(core.Low, "island.guard", func(e core.UseOnBlockEvent) {
+		guard(e.Actor, e.Pos)
 	})
-	core.UseOnBlock.Handle(core.Highest, "island.guardSide", func(e core.UseOnBlockEvent) bool {
-		return guard(e.S, e.Ctx, e.Pos.Side(e.Face))
+	core.UseOnBlock.On(core.Highest, "island.guardSide", func(e core.UseOnBlockEvent) {
+		guard(e.Actor, e.Pos.Side(e.Face))
 	})
 }
~~~~

### `content/skygear/look/subscribe.go`

~~~~diff
--- a/content/skygear/look/subscribe.go
+++ b/content/skygear/look/subscribe.go
@@ -12,13 +12,14 @@ func subscribe() {
 	subscribeLeave()
 }
 
-// A movement updates where the player looks. It runs for every move, also
-// one that was cancelled for the other subscribers (island expansion mode
-// holds the player in place): Update then clears the hidden panel.
+// A movement updates where the player looks. It is called for every move,
+// also a cancelled one (island expansion mode holds the player in place, the
+// island brings a player back from the void): Update then clears the hidden
+// panel.
 func subscribeMove() {
-	core.Move.Always(core.Monitor, "look.update", func(e core.MoveEvent, _ bool) {
+	core.Move.On(core.Monitor, "look.update", func(e core.MoveEvent) {
 		Update(e.S, e.P, e.Tx, e.Pos, e.Rot, false)
-	})
+	}, core.HandleCancelled)
 }
 
 // Every player's panel updates on every fast tick, when the tick's work is
~~~~

### `content/skygear/monitor_test.go`

~~~~diff
--- a/content/skygear/monitor_test.go
+++ b/content/skygear/monitor_test.go
@@ -12,12 +12,14 @@ import (
 )
 
 // TestMonitorSubscribersDoNotShape: a Monitor subscriber handles the result
-// of an event and neither shapes nor cancels it. The kernel refuses Handle in
-// the Monitor tier (core.TestHandleAtMonitorPanics); this test reads the
-// function literal of every subscriber registered with core.Monitor in a
-// feature package and fails when it calls Cancel or assigns through one of the
+// of an event and neither shapes nor cancels it. This test reads the function
+// literal of every subscriber registered with core.Monitor in a feature
+// package and fails when it calls Cancel or assigns through one of the
 // event's pointers (*e.Drops, *e.XP, *e.Damage, *e.Pos, *e.Pick). It sees the
-// literal only, not what the literal calls.
+// literal only, not what the literal calls. A cancel anywhere below the
+// literal is caught when it happens: while a test traces (every pin
+// scenario), the kernel's Emit panics when a Monitor subscriber cancelled the
+// event (core.TestMonitorCancelPanicsWhileTraced).
 func TestMonitorSubscribersDoNotShape(t *testing.T) {
 	files, err := filepath.Glob(filepath.Join("*", "*.go"))
 	if err != nil {
@@ -35,7 +37,7 @@ func TestMonitorSubscribersDoNotShape(t *testing.T) {
 		}
 		ast.Inspect(f, func(n ast.Node) bool {
 			call, ok := n.(*ast.CallExpr)
-			if !ok || len(call.Args) != 3 {
+			if !ok || (len(call.Args) != 3 && len(call.Args) != 4) {
 				return true
 			}
 			if tier, ok := call.Args[0].(*ast.SelectorExpr); !ok || tier.Sel.Name != "Monitor" {
~~~~

### `content/skygear/pin_order_test.go`

~~~~diff
--- a/content/skygear/pin_order_test.go
+++ b/content/skygear/pin_order_test.go
@@ -15,34 +15,30 @@ import (
 	"testing"
 
 	"github.com/presentkim/skygear/content/skygear/core"
-	"github.com/presentkim/skygear/content/skygear/core/coretest"
 )
 
 // TestPinOrder compares the kernel's subscriptions with
 // testdata/pin/order.golden: one line "<Hook> <TIER> <name>" per subscriber,
-// with " always" after the name of a subscriber that also receives a
-// cancelled event (Hook.Always), hooks in declaration order, each hook's
-// subscribers in the order they run. It names no feature, so it does not
-// change when a step moves to its feature.
+// with " handleCancelled" after the name of a subscriber that also receives
+// a cancelled event (core.HandleCancelled), hooks in declaration order, each
+// hook's tiers in order. The subscribers of one tier have no order: their
+// lines are sorted by name, in the file and in the comparison. It names no
+// feature, so it does not change when a step moves to its feature.
 //
 // Re-record it (SKYGEAR_GOLDEN=update) only when a step is added, removed or
-// moved to another tier on purpose. In a run that reorders the subscribers
-// inside their tiers (TestTierOrder) the lines of one tier of one hook are
-// compared as a set.
+// moved to another tier on purpose.
 func TestPinOrder(t *testing.T) {
 	var lines []string
 	for _, s := range core.Subscriptions() {
 		ln := fmt.Sprintf("%s %s %s", s.Hook, s.Tier, s.Name)
-		if s.Always {
-			ln += " always"
+		if s.HandleCancelled {
+			ln += " handleCancelled"
 		}
 		lines = append(lines, ln)
 	}
+	sortTiers(lines)
 	const file = "testdata/pin/order.golden"
 	if os.Getenv("SKYGEAR_GOLDEN") == "update" {
-		if hookOrder != "" {
-			t.Fatal("record order.golden in the order of registration, without " + coretest.HookOrderEnv)
-		}
 		if err := os.WriteFile(file, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
 			t.Fatal(err)
 		}
@@ -53,11 +49,6 @@ func TestPinOrder(t *testing.T) {
 		t.Fatal(err)
 	}
 	wl := strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n")
-	if hookOrder != "" {
-		// The tiers stay in order; inside "<Hook> <TIER>" the order is free.
-		sortTiers(wl)
-		sortTiers(lines)
-	}
 	for i := 0; i < len(wl) || i < len(lines); i++ {
 		var w, g string
 		if i < len(wl) {
@@ -125,7 +116,7 @@ func rootSources(t *testing.T) map[string]*ast.File {
 }
 
 // TestRootSubscribesToNothing: no non-test file of the root package calls a
-// hook's On, Handle or Always, calls core.Mark, declares a Handle* method or is named
+// hook's On, calls core.Mark, declares a Handle* method or is named
 // legacy_*.go. A step belongs in the Setup of the feature that owns it.
 func TestRootSubscribesToNothing(t *testing.T) {
 	const hint = "a step belongs in the Setup of its feature package (content/skygear/<feature>/feature.go); the root package only lists features and holds platform glue"
@@ -147,8 +138,8 @@ func TestRootSubscribesToNothing(t *testing.T) {
 				if id, ok := sel.X.(*ast.Ident); ok && id.Name == "core" && sel.Sel.Name == "Mark" {
 					t.Errorf("%s: core.Mark call: %s", name, hint)
 				}
-				if (sel.Sel.Name == "On" || sel.Sel.Name == "Handle" || sel.Sel.Name == "Always") && len(n.Args) == 3 {
-					t.Errorf("%s: %s(tier, name, f) subscribes to a hook: %s", name, sel.Sel.Name, hint)
+				if sel.Sel.Name == "On" && (len(n.Args) == 3 || len(n.Args) == 4) {
+					t.Errorf("%s: On(tier, name, f) subscribes to a hook: %s", name, hint)
 				}
 			}
 			return true
~~~~

### `content/skygear/screens/subscribe.go`

~~~~diff
--- a/content/skygear/screens/subscribe.go
+++ b/content/skygear/screens/subscribe.go
@@ -49,21 +49,17 @@ func subscribePresence() {
 // the hunter holds; in the air the menu item is the hunter's way out. The menu
 // item takes its click before anything else looks at it: Lowest.
 func subscribeUse() {
-	core.UseOnBlock.Handle(core.Lowest, "screens.menuItem", func(e core.UseOnBlockEvent) bool {
+	core.UseOnBlock.On(core.Lowest, "screens.menuItem", func(e core.UseOnBlockEvent) {
 		if IsMenuItem(e.Held) && !dimension.Hunting(e.S, e.P) {
 			e.Cancel()
 			useHeldScreen(e.S, e.P)
-			return true
 		}
-		return false
 	})
-	core.Use.Handle(core.Lowest, "screens.menuItem", func(e core.UseEvent) bool {
+	core.Use.On(core.Lowest, "screens.menuItem", func(e core.UseEvent) {
 		if IsMenuItem(e.Held) {
 			e.Cancel()
 			useHeldScreen(e.S, e.P)
-			return true
 		}
-		return false
 	})
 }
 
~~~~

### `content/skygear/storage/subscribe.go`

~~~~diff
--- a/content/skygear/storage/subscribe.go
+++ b/content/skygear/storage/subscribe.go
@@ -27,15 +27,13 @@ func subscribeDisplays() {
 // A drawer that does break loses its display. A set machine breaking or
 // placed invalidates the networks; a placed drawer is set up.
 func subscribeBlocks() {
-	core.BlockBreak.Handle(core.Low, "storage.front", func(e core.BlockBreakEvent) bool {
+	core.BlockBreak.On(core.Low, "storage.front", func(e core.BlockBreakEvent) {
 		if _, ok := e.Block.(Drawer); !ok {
-			return false
+			return
 		}
 		if lp, ok := LastPunch.Of(e.S)[e.P.H()]; ok && lp.Pos == e.Pos && lp.Front && island.CanBuild(e.S, e.Pos) {
 			e.Cancel()
-			return true
 		}
-		return false
 	})
 	core.BlockBreak.On(core.Normal, "storage.break", func(e core.BlockBreakEvent) {
 		switch b := e.Block.(type) {
@@ -61,11 +59,9 @@ func subscribeBlocks() {
 
 // A sneaking player uses a drawer with a held item that is no block.
 func subscribeSneakUse() {
-	core.UseOnBlock.Handle(core.Normal, "storage.sneakUse", func(e core.UseOnBlockEvent) bool {
+	core.UseOnBlock.On(core.Normal, "storage.sneakUse", func(e core.UseOnBlockEvent) {
 		if SneakUse(e.S, e.P, e.Tx, e.Pos, e.Held) {
 			e.Cancel()
-			return true
 		}
-		return false
 	})
 }
~~~~

### `content/skygear/tinker/subscribe.go`

~~~~diff
--- a/content/skygear/tinker/subscribe.go
+++ b/content/skygear/tinker/subscribe.go
@@ -35,7 +35,7 @@ func subscribeFood() {
 			*e.Damage = 0
 		}
 	})
-	core.Death.On(core.Normal, "tinker.keepFood", func(e core.PlayerEvent) {
+	core.Death.On(core.Normal, "tinker.keepFood", func(e core.NoticeEvent) {
 		keptFood.Of(e.S)[e.P.H()] = e.P.Food()
 	})
 	core.Respawn.On(core.Normal, "tinker.restoreFood", func(e core.RespawnEvent) {
~~~~

### `content/skygear/wand/subscribe.go`

~~~~diff
--- a/content/skygear/wand/subscribe.go
+++ b/content/skygear/wand/subscribe.go
@@ -19,13 +19,11 @@ func subscribe() {
 // A held tool takes the click away from what the block itself would do with
 // it: the Low tier.
 func subscribeUse() {
-	core.UseOnBlock.Handle(core.Low, "wand.use", func(e core.UseOnBlockEvent) bool {
+	core.UseOnBlock.On(core.Low, "wand.use", func(e core.UseOnBlockEvent) {
 		if _, ok := sgitem.WandOf(e.Held); ok && island.CanBuild(e.S, e.Pos) {
 			e.Cancel()
 			Use(e.S, e.P, e.Tx, e.Pos, e.Face, e.Held)
-			return true
 		}
-		return false
 	})
 	core.Use.On(core.Normal, "wand.options", func(e core.UseEvent) {
 		if _, ok := sgitem.WandOf(e.Held); ok && e.P.Sneaking() && !e.S.Debounced(e.P) {
~~~~

## REF T139

The documents. Prototype patch: `T139.patch`.

Documents only. The patch applies the text below; by hand, replace in the order given.

`docs/tech/extension.md`:

1. In the layer table, the `core` row: `훅(\`Hook\`)과 이벤트` becomes `훅(\`Hook\`, \`CancelHook\`)과 이벤트`.
2. In "이벤트의 길", the `Leave` bullet loses its last sentence `구독자가 이벤트를 멈춰도 한다.`
3. The sections "### 티어" and "### On, Handle, Always" (up to, not including, "### 이벤트의 값") are replaced by the new "### 티어" and "### 구독자와 취소" of the diff.
4. "훅 목록": the sentence on the column, the column head (`멈춤` becomes `취소`), every row's last cell (`됨` for the 20 hooks of `NewCancelHook`, `안 됨` for the 15 others), the event of `Jump` and `Death` (`NoticeEvent`), and the paragraph on `PlayerEvent`.
5. "새 기능 더하기": the example's `subscribeJump` and `subscribeSneak`.
6. The sections "### 티어 고르기", "### 순서를 지키는 것" and "### 예외" (up to, not including, "## 기능의 데이터") are replaced by the new "### 티어 고르기", "### 순서를 지키는 것" and "### 역할과 다른 티어".
7. "구조를 지키는 규칙": the rows on the root's subscriptions, the hot path, the tiers, the guard, Monitor, the option, `Leave` and the conditions, as the diff; the benchmark sentence and command.
8. "핀": the row of `order.golden` and the three bullets and the sentence after them on `TestTierOrder`.

The other four files: one passage each, as the diff.

### `CONTRIBUTING.md`

~~~~diff
--- a/CONTRIBUTING.md
+++ b/CONTRIBUTING.md
@@ -60,7 +60,10 @@ Thanks for your interest. These are the rules the project follows.
   its steps to the kernel's hooks in `Setup`, in `subscribe.go`, with
   subscriber names
   `<feature>.<step>` and the tier (`core.Lowest` to `core.Monitor`) of what the
-  step does; inside a tier no step may depend on its place. It keeps its data in
+  step does; the steps of one tier have no order. No step stops another: a step
+  acts by its own conditions and cancels the event (`e.Cancel()`) only when the
+  action or the later steps must not happen; a cancelled event reaches only the
+  steps registered with `core.HandleCancelled`. It keeps its data in
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
-기능이 자기 패키지에서 선언한다. 기능은 루트 패키지의 `features.go` 목록에 한 줄로 오른다. 훅, 구독자의 티어, 새 기능을 더하는 순서는
+기능이 자기 패키지에서 선언한다. 기능은 루트 패키지의 `features.go` 목록에 한 줄로 오른다. 훅, 구독자의 티어와 취소, 새 기능을 더하는 순서는
 [docs/tech/extension.md](../../docs/tech/extension.md) 에 있다.
 위 칸이 아래 칸을 불러야 할 때 중 이벤트가 아닌 것(메뉴 열기, 사이드바 갱신)은 위 칸이 함수 변수를 두고,
 아래 칸이 `init` 에서 채운다(예: `core.ShowMenu` 는 `screens` 가 채운다). 다른 기능의 기계 종류는 `machine` 이 이름과 정의를 두고,
~~~~

### `docs/TODO.md`

~~~~diff
--- a/docs/TODO.md
+++ b/docs/TODO.md
@@ -17,16 +17,17 @@
 
 ### 정리
 - Sky Gear 이름 변경 세이브 변환(`content/skygear/migrate/namespace.go`와 그 테스트)과 그 호출(`cmd/skygear/main.go`)을 모든 서버가 한 번씩 돌린 뒤 지운다. 데이터 폴더의 `skygear-namespace.done` 플래그와 `<데이터 폴더>.before-skygear` 사본도 함께 지운다.
-- `UseOnBlock` 의 두 번째 확장 모드 검사 `expand.guardSide`(HIGHEST)는 멈출 수 없다. 확장 모드의 플레이어는 이미 `expand.guard`(LOWEST)에서 멈춘다. 단계를 지우거나 옆 칸 검사가 의미를 갖게 고친다. 지우면 `order.golden` 과 `handler-use.golden` 에서 그 줄이 빠진다.
-- 훅 순서의 예외 넷([extension.md](tech/extension.md) "예외")을 어떻게 풀지 정한다. 지금은 모두 예전 동작 그대로 둔다. 풀면 플레이어가 보는 동작이 바뀌므로 장면 골든을 다시 기록한다. 쌍을 지우면 `orderKept`(루트 `tierorder_test.go`)에서도 뺀다.
-  - `Sneak` 의 `achievement.sneak`(LOW): MONITOR 에 `Always` 로 달아 웅크리기를 모두 센다(확장 모드의 웅크리기도 센다). 아니면 멈춘 구독자의 티어를 `Always` 구독자에게 알려 LOWEST·LOW 에서 멈춘 것만 빼고 센다(API 가 넓어진다).
-  - `UseOnBlock` 의 `storage.sneakUse`, `conduit.plate` 다음 `handwork.stick`: 흙·잔디가 아닌 블록에 막대기를 쓰면 언제나 `stick_other` 를 세고, 세는 일을 MONITOR 구독자로 뗀다(서랍 메뉴나 판 설정을 막대기로 열어도 센다). 아니면 API 에 "누구 뒤" 제약을 더한다.
-  - `UseOnBlock` 의 `conduit.plate` 다음 `fairy.time`: `fairy.time` 을 든 도구처럼 LOW 에 두고 `island.CanBuild` 조건을 단다(모래시계가 판보다 먼저 기계를 감는다). 아니면 "누구 뒤" 제약을 더한다.
-  - `Join` 의 `dimension.joined`(HIGH): 역할대로 LOW 로 옮긴다. 서버가 멈출 때 사냥 중이던 플레이어가 요정 가루를 잃지 않고 메뉴 아이템 이름도 고쳐진다.
-- 티어 안 순서 가드(`TestTierOrder`)는 장면의 결과와 업적 카운터만 견준다. 채팅·팁·소리·패킷, 장면이 닿지 않는 상태는 보지 못한다. 첫 접속 때 메뉴 아이템이 소급 요정 가루를 덮어쓰던 문제(`TestFirstJoinKeepsBackfilledDust`)가 그렇게 빠져나갔다. 인벤토리를 건드리는 접속·퇴장 구독자마다 장면을 더한다.
+- `UseOnBlock` 의 두 번째 확장 모드 검사 `expand.guardSide`(HIGHEST)는 일할 때가 없다. 확장 모드의 클릭은 `expand.guard`(LOWEST)나 화면을 여는 아이템의 구독자가 이미 취소한다. 구독자를 지우거나 옆 칸 검사가 의미를 갖게 고친다. 지우면 `order.golden` 과 `handler-use.golden` 에서 그 줄이 빠진다.
+- 훅에 남은 결정([extension.md](tech/extension.md) "구독자와 취소", "역할과 다른 티어"). 정하면 플레이어가 보는 동작이 바뀌므로 테스트를 더하고 장면 골든을 다시 기록한다.
+  - `Join` 의 `dimension.joined`(HIGH): 역할대로 LOW 로 옮긴다. 서버가 멈출 때 사냥 중이던 플레이어가 소급 요정 가루를 잃지 않는다. 메뉴 아이템 이름은 그대로 남는다(`screens.menuItem` 이 LOWEST 에서 먼저 돈다).
+  - 취소가 필요 없는데 취소하는 구독자: 공중에서 연 메뉴 아이템과 이차원석(`Use` 의 `screens.menuItem`, `dimension.stone`), 흙·잔디를 두드린 막대기(`handwork.stick`). 취소하지 않게 바꾸면 장면 골든의 `cancelled=` 가 달라지고, 막대기는 섬 가장자리에서 옆 칸 가드의 "섬 밖" 팁이 함께 뜬다.
+  - `stick_other` 는 아무도 취소하지 않은 클릭만 센다. 막대기로 누른 다른 블록을 모두 세려면 `handwork.stickOther` 에 `core.HandleCancelled` 를 주고 `core.Takeover` 조건을 단다(섬 밖, 서랍 메뉴, 도관 판, 작물을 누른 것도 센다).
+  - `Jump` 의 `elevator.up` 은 모드에 붙잡힌 플레이어도 태운다. `elevator.down` 처럼 `core.Takeover` 조건을 달지 정한다.
+  - 섬 가장자리의 원목을 섬 밖을 향한 면으로 누르면 `handwork.gather`(NORMAL)가 섬유를 떨군 뒤 `island.guardSide`(HIGHEST)가 클릭을 취소해 원목이 벗겨지지 않는다. 되풀이할 수 있다. `handwork.gather` 가 끝까지 간 클릭만 받게 옮길지 정한다.
+- 티어 안 순서 가드(`TestTierOrder`)는 장면의 결과, 업적 카운터, 장면의 플레이어가 든 아이템을 견준다. 채팅·팁·소리·패킷, 장면이 닿지 않는 상태는 보지 못한다. 첫 접속 때 메뉴 아이템이 소급 요정 가루를 덮어쓰던 문제(`TestFirstJoinKeepsBackfilledDust`)는 그런 장면이 없어서 빠져나갔다. 인벤토리를 건드리는 접속·퇴장 구독자마다 장면을 더한다.
 - `Join` 의 `screens.board`(MONITOR)가 이제 `dimension.joined`(HIGH) 뒤에 돈다. 서버가 멈출 때 사냥 중이던 플레이어의 퀘스트 비용 줄이 다음 갱신까지(1초 안) 장비가 아니라 되돌린 소지품을 센다. 그대로 둘지 정한다.
 - 같은 MONITOR 안에서는 채팅 순서가 설치 순서를 따른다(`achievement.place`·`conduit.place`·드래곤 알림, `playSeconds` 와 수집 보상). 기록을 MONITOR 로 옮기면서 업적 메시지가 다른 구독자의 메시지 뒤로 간 곳이 있다(`Death`, `BlockPlace`). 순서가 중요한 메시지가 있으면 한 구독자로 묶는다.
-- MONITOR 가드(`TestMonitorSubscribersDoNotShape`)는 구독자 함수 본문만 읽는다. 구독자가 부르는 도우미 안의 취소나 변경, 포인터를 변수에 옮겨 쓰는 경우는 보지 못한다. 지금의 23개는 읽어서 확인했다. 가드를 넓히거나 MONITOR 구독자에 읽기 전용 이벤트를 넘긴다.
+- MONITOR 가드(`TestMonitorSubscribersDoNotShape`)는 구독자 함수 본문만 읽는다. 도우미 안의 취소는 장면 테스트에서 `Emit` 이 잡지만(추적 중일 때 패닉), 도우미 안에서 이벤트의 포인터에 쓰는 경우와 포인터를 변수에 옮겨 쓰는 경우는 보지 못한다. 지금의 MONITOR 구독자는 읽어서 확인했다. 가드를 넓히거나 MONITOR 구독자에 읽기 전용 이벤트를 넘긴다.
 - `TestHuntClear`(`content/skygear/dimension/dimension_test.go`)와 `TestKitAndLevels` 가 `TestHuntFails` 와 같은 시간 의존 방식이라 `-race` 에서 `TestHuntFails` 처럼 불안정해질 수 있다. 같은 방식으로 시간에 기대지 않게 고친다.
 - `-race` 에서 `TestPinUse` 가 가끔 테스트에만 있는 데이터 경합을 보고한다. 동기 테스트 월드가 미뤄 둔 일을 월드 큐 밖에서 돌리기 때문이다(`fairy.wander` 와 `placeAt`·`Teleport`). 테스트 월드가 미룬 일을 월드 큐에서 돌리게 하거나, 시나리오가 그 일이 끝나기를 기다리게 고친다.
 - `machine.Register` 는 부르는 기능이 그 종류의 `Owner` 인지 확인하지 않는다. 다른 기능이 남의 종류를 등록해도 `TestKindOwners` 와 골든은 통과한다. 커널이 지금 설치 중인 기능 이름을 알려 주면 막을 수 있다. 기능 패키지의 테스트는 `core.Install` 없이 `coretest.Setup` 으로 설치하므로(`netstore/keys_test.go`) 그 경우를 어떻게 볼지 먼저 정한다.
~~~~

### `docs/design/03-architecture.md`

~~~~diff
--- a/docs/design/03-architecture.md
+++ b/docs/design/03-architecture.md
@@ -102,7 +102,7 @@ type Session interface {
   저장 데이터 `core.NewSaved`(모든 저장 파트가 세이브 상태 문서 하나의 필드라 기능을 옮겨도 세이브 형식이 그대로다).
 - 패키지는 자기보다 아래 층만 import 한다(데이터 → 핵심 → 진행 → 기능 → 내용 목록 → 화면 → 게임). 위로 불러야 할 때는
   아래 패키지가 함수 변수를 두고 위 패키지가 `init` 에서 채운다. `layers_test.go` 가 이 방향을 검사한다.
-- 이벤트는 훅으로 기능에 닿는다. Dragonfly 가 `core.PlayerHandler`·`core.WorldHandler` 를 부르면 같은 이름의 훅이 티어 차례로 기능의 구독자를 부르고,
+- 이벤트는 훅으로 기능에 닿는다. Dragonfly 가 `core.PlayerHandler`·`core.WorldHandler` 를 부르면 같은 이름의 훅이 티어 차례로 기능의 구독자를 모두 부르고(취소된 이벤트는 받겠다고 한 구독자만 받는다),
   플랫폼이 부르는 세션 호출(`Join`, `Leave`, `NewDay`, `Second`, `Fast`, `Autosave`, `Close`)은 `core.Lifecycle` 이 같은 식으로 훅에 내보낸다.
   기능은 자기 패키지의 `feature.go` 에서 `core.Feature` 를 선언하고 `Setup` 에서 구독자를 단다.
 - 루트 패키지 `skygear` 는 기능 목록(`features.go`, `core.Install`)과 플랫폼 접착제(`Game`, `session`)이다. 플랫폼 인터페이스를 구현하고 운영자 도구(`/sgadmin`, `/sgdebug`)를 두며, 훅에 구독하지 않는다. 플레이어 명령어는 루트에 없고 기능이 `core.Feature` 의 `Commands` 로 더한다.
~~~~

### `docs/tech/extension.md`

~~~~diff
--- a/docs/tech/extension.md
+++ b/docs/tech/extension.md
@@ -14,7 +14,7 @@ Dragonfly 자체를 확장하는 방법은 [dragonfly.md](dragonfly.md) 에 있
 | Dragonfly 패치 | `third_party/dragonfly` | Dragonfly 의 동작 자체. 패치 규칙은 [dragonfly.md](dragonfly.md) |
 | `sghook` | `common/sghook` | 패킷 훅, 세션 내부 접근, 클라이언트에 보내는 아이템·블록 항목 고치기(`EditCustomItems`, `EditCustomBlocks`, `ApplyEntryEdits`). Dragonfly 내부는 여기서만 건드린다 |
 | `clientdef` | `common/clientdef` | 커스텀 아이템·블록이 클라이언트에 보내는 컴포넌트. 공개 레지스트리만 쓰고 게임을 모른다 |
-| `core` | `content/skygear/core` | 기능의 커널. `Session`과 그 파트(`Part`, `Players`, `Saved`), 훅(`Hook`)과 이벤트, 핸들러(`PlayerHandler`, `WorldHandler`), 수명 호출(`Lifecycle`), 기능 서술자(`Feature`)와 `Install`, 창작 목록 순서(`CreativeItem`, `CreativeItems`), 기능의 명령어(`Commands`, `PlayerSession`), 플레이어를 붙잡는 모드(`RegisterTakeover`, `Takeover`) |
+| `core` | `content/skygear/core` | 기능의 커널. `Session`과 그 파트(`Part`, `Players`, `Saved`), 훅(`Hook`, `CancelHook`)과 이벤트, 핸들러(`PlayerHandler`, `WorldHandler`), 수명 호출(`Lifecycle`), 기능 서술자(`Feature`)와 `Install`, 창작 목록 순서(`CreativeItem`, `CreativeItems`), 기능의 명령어(`Commands`, `PlayerSession`), 플레이어를 붙잡는 모드(`RegisterTakeover`, `Takeover`) |
 | 기능 패키지 | `content/skygear/<기능>` | 한 기능의 블록·아이템·화면·규칙과, 그 기능이 훅에 거는 구독자 |
 | 루트 | `content/skygear` | 기능 목록(`features.go`)과 플랫폼 접착제(`Game`, `session`), 운영자 도구 |
 
@@ -62,7 +62,7 @@ Dragonfly 자체를 확장하는 방법은 [dragonfly.md](dragonfly.md) 에 있
 
 세션 타입은 `core.Lifecycle` 을 품고 있어 이 호출들이 `Lifecycle` 의 메서드로 간다. 메서드마다 이벤트를 만들어 훅의 `Emit` 을 한 번 부르고, 아래 꼬리 일을 한다.
 
-- `Leave`: 구독자가 끝나면 플레이어 클라이언트의 도형을 지우고(`RemoveAllDebugShapes`) 플레이어별 파트 항목을 지운다(`Session.Leave`). 구독자가 이벤트를 멈춰도 한다.
+- `Leave`: 구독자가 끝나면 플레이어 클라이언트의 도형을 지우고(`RemoveAllDebugShapes`) 플레이어별 파트 항목을 지운다(`Session.Leave`).
 - `Second`: 구독자가 끝나면 바뀐 상태를 저장 주기에 맞을 때만 쓴다(`FlushIfDue`).
 - `Autosave`: 구독자가 끝나면 바뀐 상태를 쓴다(`Flush`).
 - `Close`: 구독자가 끝나면 저장 파트를 쓰고 쓰기 고루틴을 멈추고(`Session.Close`), 세션 등록을 지운다(`Unregister`). 루트의 `session.Close` 는 그 뒤에 `Game` 의 세션 표에서도 지운다.
@@ -83,11 +83,9 @@ Dragonfly 자체를 확장하는 방법은 [dragonfly.md](dragonfly.md) 에 있
 | HIGHEST | `core.Highest` | 이벤트를 가공할 수 있는 마지막 후처리 작업에 사용합니다. |
 | MONITOR | `core.Monitor` | 이벤트를 가공하거나 취소하지 않는 결과 처리(기록, 업적 처리 등)에 사용합니다. |
 
-이벤트가 취소되어도 수신할지를 선택할 수 있다(전통적인 handleCancelled 옵션). 기본은 받지 않는 것이고, `Always` 로 단 구독자만 받는다. 여기서 취소된 이벤트는 앞의 `Handle` 구독자가 멈춘 이벤트를 말한다(다음 절).
+같은 훅의 같은 티어에 선 구독자 사이에는 순서가 없다. 실제로는 등록한 순서(기능 목록 `features.go` 의 순서, 한 기능 안에서는 `subscribe()` 가 단 순서)로 돌지만, 그 순서는 실행이 매번 같게 할 뿐이고 거기에 기대는 코드는 쓰지 않는다. 같은 티어의 두 구독자는 어느 쪽이 먼저 돌아도 결과가 같아야 한다. 앞뒤가 필요하면 역할에 맞게 티어를 나누거나, 구독자의 조건에 그 사실을 적거나, 한 구독자로 합친다. 순서를 지켜 주는 예외 목록은 없다. `TestTierOrder` 가 이 규칙을 지킨다(아래 "핀").
 
-같은 훅의 같은 티어 안에서는 등록한 순서대로 돈다. 기능 목록(`features.go`)의 순서이고, 한 기능 안에서는 `subscribe()` 가 단 순서다. 이 순서는 실행이 매번 같게 할 뿐이다. 구독자는 같은 티어 안의 자리에 기대면 안 된다. 같은 티어의 두 구독자는 어느 쪽이 먼저 돌아도 결과가 같아야 한다. 순서가 필요하면 티어를 나누거나, 구독자의 조건에 그 사실을 적거나, 한 구독자로 합친다. `TestTierOrder` 가 이 규칙을 지킨다(아래 "핀").
-
-MONITOR 구독자는 이벤트를 가공하지도 취소하지도 않는다. MONITOR 에 `Handle` 을 달면 패닉이다. MONITOR 구독자의 함수가 `Cancel()` 을 부르거나 이벤트의 포인터에 쓰면 `TestMonitorSubscribersDoNotShape` 가 실패한다. Bukkit 의 MONITOR 와 역할은 같고 기본값이 다르다. 여기서는 MONITOR 구독자도 멈춘 이벤트를 받지 않는다. 받으려면 `Always` 로 단다.
+MONITOR 구독자는 이벤트를 가공하지도 취소하지도 않는다. MONITOR 구독자의 함수가 `Cancel()` 을 부르거나 이벤트의 포인터에 쓰면 `TestMonitorSubscribersDoNotShape` 가 실패한다. 함수가 부른 도우미 안에서 취소하면 장면을 도는 테스트에서 `Emit` 이 패닉한다(`core.Trace` 가 켜진 동안만 검사한다). MONITOR 구독자도 기본으로는 취소된 이벤트를 받지 않는다. 막힌 일은 일어나지 않은 일이라 세지 않는다. 취소된 이벤트도 봐야 하면 `core.HandleCancelled` 를 준다(다음 절).
 
 수명 훅(`Join`, `Leave`, `Second`, `Fast`, `Autosave`, `Close` 따위)에는 가공하거나 취소할 것이 없다. 거기서는 티어를 일의 차례로 읽는다.
 
@@ -99,16 +97,25 @@ MONITOR 구독자는 이벤트를 가공하지도 취소하지도 않는다. MON
 
 틱 훅(`Second`, `Fast`)에서 같은 티어의 구독자는 서로의 이번 틱 결과를 읽지 않는다. 읽어야 하면 티어를 나눈다.
 
-### On, Handle, Always
+### 구독자와 취소
+
+어느 구독자도 다른 구독자를 멈추지 못한다. 훅의 구독자는 티어 순서대로 모두 불린다. 구독자가 불리지 않는 경우는 하나뿐이다. 이벤트가 취소됐고, 그 구독자가 취소된 이벤트도 받겠다고 하지 않았을 때다.
+
+- `On(티어, name, f)` 로 단 구독자는 취소된 이벤트를 받지 않는다. 이것이 기본이다.
+- `On(티어, name, f, core.HandleCancelled)` 로 단 구독자는 취소된 이벤트도 받는다(전통적인 handleCancelled 옵션). 함수 모양은 같고, 취소됐는지는 `e.Cancelled()` 로 읽는다. 지금은 `Move` 의 `look.update` 가 쓴다. 섬 확장 모드가 이동을 취소해도, 섬이 공허에서 플레이어를 되돌려도 패널을 갱신한다.
+- 취소는 이벤트로 한다. `e.Cancel()` 을 부르면 플레이어가 한 행동이 Dragonfly 에서 일어나지 않고, 뒤에 선 기본 구독자가 불리지 않는다. 취소된 이벤트는 Dragonfly 가 준 컨텍스트가 취소된 이벤트다. 그 밖의 뜻은 없고, 취소를 되돌릴 수도 없다.
+- `Emit` 은 값을 돌려주지 않는다. Dragonfly 는 자기가 준 컨텍스트를 읽는다.
+
+취소할 수 있는 이벤트는 `Actor`(`S`, `P`, `Tx`, `Ctx`)를 품은 이벤트이고, 그 훅은 `core.CancelHook` 이다. `Jump`, `Death`, `Respawn` 과 세션·월드 훅의 이벤트는 취소할 수 없다. 이 이벤트에는 `Cancel()` 과 `Cancelled()` 가 없고(`Jump`·`Death` 의 `NoticeEvent` 와 `RespawnEvent` 는 `Who` 만 품는다), 훅이 `core.Hook` 이라 `On` 이 옵션을 받지 않는다. 취소할 수 없는 이벤트를 취소하는 코드와 그런 훅에 handleCancelled 를 다는 코드는 컴파일되지 않는다. 이런 훅에서는 모든 구독자가 언제나 불린다.
 
-- `On(티어, name, f)` 로 단 구독자는 이벤트를 멈추지 못하고, 멈춘 이벤트를 받지 않는다.
-- `Handle(티어, name, f)` 로 단 구독자가 true 를 돌려주면 이벤트가 멈춘다. 뒤의 구독자는 돌지 않고(`Always` 로 단 구독자는 돈다) `Emit` 이 true 를 돌려준다. 핸들러와 수명 호출은 그 값을 쓰지 않는다. MONITOR 에는 달 수 없다.
-- `Always(티어, name, f)` 로 단 구독자는 `On` 과 같되 앞에서 멈춘 이벤트도 받는다. 함수는 `func(e E, cancelled bool)` 이고, 멈춘 이벤트면 `cancelled` 가 true 다. 스스로 멈추지는 못한다. 어느 훅, 어느 티어에나 달 수 있고, `Handle` 구독자가 없는 훅에서는 `cancelled` 가 늘 false 다. 지금은 `Move` 의 `look.update` 가 쓴다. 섬 확장 모드가 이동을 멈춰도 패널을 갱신한다.
+구독자는 다음 둘 중 하나로 쓴다.
 
-멈춤과 취소는 다르다. 멈춤은 훅의 일이고 뒤의 구독자를 건너뛴다. 취소는 Dragonfly 의 일이고 플레이어가 한 행동이 일어나지 않게 한다. `Actor.Cancel()`(`e.Cancel()`)이 취소하며, 취소만 하면 뒤의 구독자는 계속 돈다. `e.Cancelled()` 가 취소됐는지 알려 준다.
-구독자는 둘 중 하나만 하거나 둘 다 하거나 아무것도 안 할 수 있다. `Jump`, `Death`, `Respawn` 은 Dragonfly 가 취소할 수 없는 이벤트라 `Ctx` 가 nil 이고 `Cancel()` 은 아무 일도 하지 않으며 `Cancelled()` 는 false 다.
+- 어떤 상태에서만 처리하고 취소한다. 자기 조건(기능의 상태, 든 아이템, 블록, 플레이어가 든 모드)이 맞을 때만 일하고 `e.Cancel()` 을 부른다. 바닐라 동작이나 뒤의 기본 구독자가 일어나면 안 될 때 쓴다. 든 복사기가 클릭을 쓰면 블록이 열리면 안 되고(`copier.use`), 섬 밖에는 짓지 못한다(`island.guard`).
+- 기능은 처리하되 불필요하면 취소하지 않는다. 막을 것이 없으면 일만 하고 이벤트는 그대로 둔다. 다른 구독자도 계속 일한다. 엘리베이터는 웅크리면 내려가지만 웅크리기를 취소하지 않는다(`elevator.down`). 그래서 그 웅크리기도 세어지고 묘목도 자랄 기회를 얻는다.
 
-handleCancelled 옵션이 멈춤을 기준으로 삼는 까닭은 구독자들이 실제로 그렇게 쓰기 때문이다. `Handle` 구독자 31개 가운데 29개는 멈추면서 `Cancel()` 도 한다. 2개(`Sneak` 의 `expand.show`, `elevator.down`)는 멈추기만 한다. 웅크리기를 취소하면 플레이어가 웅크리지 못하기 때문이다. `Cancel()` 만 하고 멈추지 않는 구독자는 넷(`storage.break`, `island.void`, `dimension.fallen`, `fairy.touch`)이고, 그 뒤의 구독자는 `On` 으로 달았어도 그대로 돈다. `Cancelled()` 를 보고 일을 건너뛰는 구독자는 없다.
+"앞에서 아무도 가져가지 않았으니 내가 한다"는 조건은 쓰지 않는다. 구독자는 자기 조건으로 정한다. 다른 모드가 플레이어를 붙잡고 있을 때 하지 않는 일은 `core.Takeover(s, p) == ""` 로 적는다(`elevator.down`, `farm.crouchGrow`).
+
+같은 티어에서 한 구독자가 취소하고 다른 기본 구독자가 같은 이벤트에 일할 수 있으면, 도는 순서에 따라 뒤 구독자가 불리기도 하고 불리지 않기도 한다. 그런 쌍은 조건이 겹치지 않게 적거나(든 아이템, 블록 종류, 웅크림, `island.CanBuild`) 역할에 맞게 다른 티어에 둔다.
 
 ### 이벤트의 값
 
@@ -122,47 +129,47 @@ handleCancelled 옵션이 멈춤을 기준으로 삼는 까닭은 구독자들
 ## 훅 목록
 
 훅은 35개이고 `content/skygear/core/events.go` 에 선언되어 있다. 플레이어 훅 23개와 세션·월드 훅 12개다.
-"멈춤" 칸은 지금 그 훅에 `Handle` 구독자가 있는지를 나타낸다. 없는 훅에도 `Handle` 을 달 수 있다.
+"취소" 칸은 그 훅의 이벤트를 취소할 수 있는지를 나타낸다. "됨" 인 훅은 `core.CancelHook` 이고 나머지는 `core.Hook` 이다.
 
-| 훅 | 내보내는 때 | 이벤트 | 멈춤 |
+| 훅 | 내보내는 때 | 이벤트 | 취소 |
 |---|---|---|---|
-| `StartBreak` | 블록 캐기를 시작할 때 | `StartBreakEvent` | 있음 |
-| `BlockBreak` | 블록을 캘 때 | `BlockBreakEvent` | 있음 |
-| `BlockPlace` | 블록을 놓을 때 | `BlockPlaceEvent` | 있음 |
-| `UseOnBlock` | 블록에 아이템을 쓸 때 | `UseOnBlockEvent` | 있음 |
-| `UseOnEntity` | 엔티티에 아이템을 쓸 때 | `EntityEvent` | 있음 |
-| `AttackEntity` | 엔티티를 때릴 때 | `EntityEvent` | 있음 |
-| `Use` | 공중에서 아이템을 쓸 때 | `UseEvent` | 있음 |
-| `Move` | 움직일 때 | `MoveEvent` | 있음 |
-| `Sneak` | 웅크리기를 시작할 때 | `PlayerEvent` | 있음 |
-| `Sprint` | 달리기를 시작할 때 | `PlayerEvent` | 없음 |
-| `Jump` | 점프할 때 | `PlayerEvent` | 없음 |
-| `ItemDamage` | 아이템이 내구도를 잃을 때 | `ItemDamageEvent` | 없음 |
-| `ItemConsume` | 아이템을 먹거나 마실 때 | `ItemEvent` | 없음 |
-| `ItemPickup` | 아이템을 주울 때 | `ItemEvent` | 없음 |
-| `ItemDrop` | 아이템을 버릴 때 | `ItemEvent` | 있음 |
-| `Death` | 죽을 때 | `PlayerEvent` | 없음 |
-| `Respawn` | 되살아날 때 | `RespawnEvent` | 없음 |
-| `Hurt` | 피해를 입기 전 | `HurtEvent` | 있음 |
-| `PunchAir` | 허공을 칠 때 | `PlayerEvent` | 없음 |
-| `Chat` | 채팅을 보낼 때 | `PlayerEvent` | 없음 |
-| `BlockPick` | 블록을 집어 들 때 | `PlayerEvent` | 없음 |
-| `HeldSlot` | 들고 있는 칸을 바꿀 때 | `PlayerEvent` | 없음 |
-| `Command` | 명령어를 실행할 때 | `PlayerEvent` | 없음 |
-| `Load` | 세이브의 첫 트랜잭션 앞(`Tx` 는 nil) | `Tick` | 없음 |
-| `Start` | 세이브의 첫 트랜잭션 안 | `Tick` | 없음 |
-| `Join` | 플레이어가 입장한 뒤 | `Presence` | 없음 |
-| `Leave` | 플레이어가 나가기 전 | `Presence` | 없음 |
-| `NewDay` | 인스턴스 날짜가 바뀔 때 | `DayEvent` | 없음 |
-| `Second` | 약 1초마다 | `Tick` | 없음 |
-| `Fast` | 0.2초마다 | `Tick` | 없음 |
-| `Autosave` | 자동 저장 주기마다 | `Tick` | 없음 |
-| `Close` | 세이브를 닫을 때 | `Tick` | 없음 |
-| `ColumnLoad` | 청크 칼럼이 월드에 더해질 때 | `ColumnEvent` | 없음 |
-| `ColumnUnload` | 청크 칼럼이 월드에서 빠질 때 | `ColumnEvent` | 없음 |
-| `EntityDespawn` | 엔티티가 월드를 떠날 때 | `DespawnEvent` | 없음 |
-
-`PlayerEvent` 는 인자가 없는 이벤트이고, `Actor`(`S`, `P`, `Tx`, `Ctx`)만 든다. 위치와 방향, 대상 같은 인자는 이벤트 종류마다 `core/events.go` 에서 본다.
+| `StartBreak` | 블록 캐기를 시작할 때 | `StartBreakEvent` | 됨 |
+| `BlockBreak` | 블록을 캘 때 | `BlockBreakEvent` | 됨 |
+| `BlockPlace` | 블록을 놓을 때 | `BlockPlaceEvent` | 됨 |
+| `UseOnBlock` | 블록에 아이템을 쓸 때 | `UseOnBlockEvent` | 됨 |
+| `UseOnEntity` | 엔티티에 아이템을 쓸 때 | `EntityEvent` | 됨 |
+| `AttackEntity` | 엔티티를 때릴 때 | `EntityEvent` | 됨 |
+| `Use` | 공중에서 아이템을 쓸 때 | `UseEvent` | 됨 |
+| `Move` | 움직일 때 | `MoveEvent` | 됨 |
+| `Sneak` | 웅크리기를 시작할 때 | `PlayerEvent` | 됨 |
+| `Sprint` | 달리기를 시작할 때 | `PlayerEvent` | 됨 |
+| `Jump` | 점프할 때 | `NoticeEvent` | 안 됨 |
+| `ItemDamage` | 아이템이 내구도를 잃을 때 | `ItemDamageEvent` | 됨 |
+| `ItemConsume` | 아이템을 먹거나 마실 때 | `ItemEvent` | 됨 |
+| `ItemPickup` | 아이템을 주울 때 | `ItemEvent` | 됨 |
+| `ItemDrop` | 아이템을 버릴 때 | `ItemEvent` | 됨 |
+| `Death` | 죽을 때 | `NoticeEvent` | 안 됨 |
+| `Respawn` | 되살아날 때 | `RespawnEvent` | 안 됨 |
+| `Hurt` | 피해를 입기 전 | `HurtEvent` | 됨 |
+| `PunchAir` | 허공을 칠 때 | `PlayerEvent` | 됨 |
+| `Chat` | 채팅을 보낼 때 | `PlayerEvent` | 됨 |
+| `BlockPick` | 블록을 집어 들 때 | `PlayerEvent` | 됨 |
+| `HeldSlot` | 들고 있는 칸을 바꿀 때 | `PlayerEvent` | 됨 |
+| `Command` | 명령어를 실행할 때 | `PlayerEvent` | 됨 |
+| `Load` | 세이브의 첫 트랜잭션 앞(`Tx` 는 nil) | `Tick` | 안 됨 |
+| `Start` | 세이브의 첫 트랜잭션 안 | `Tick` | 안 됨 |
+| `Join` | 플레이어가 입장한 뒤 | `Presence` | 안 됨 |
+| `Leave` | 플레이어가 나가기 전 | `Presence` | 안 됨 |
+| `NewDay` | 인스턴스 날짜가 바뀔 때 | `DayEvent` | 안 됨 |
+| `Second` | 약 1초마다 | `Tick` | 안 됨 |
+| `Fast` | 0.2초마다 | `Tick` | 안 됨 |
+| `Autosave` | 자동 저장 주기마다 | `Tick` | 안 됨 |
+| `Close` | 세이브를 닫을 때 | `Tick` | 안 됨 |
+| `ColumnLoad` | 청크 칼럼이 월드에 더해질 때 | `ColumnEvent` | 안 됨 |
+| `ColumnUnload` | 청크 칼럼이 월드에서 빠질 때 | `ColumnEvent` | 안 됨 |
+| `EntityDespawn` | 엔티티가 월드를 떠날 때 | `DespawnEvent` | 안 됨 |
+
+`PlayerEvent` 는 인자가 없는 취소할 수 있는 이벤트이고 `Actor`(`S`, `P`, `Tx`, `Ctx`)만 든다. `NoticeEvent` 는 Dragonfly 가 알리기만 하는 이벤트(`Jump`, `Death`)이고 `Who`(`S`, `P`, `Tx`)만 든다. 위치와 방향, 대상 같은 인자는 이벤트 종류마다 `core/events.go` 에서 본다.
 이벤트에 없는 Dragonfly 인자(채팅 글, 사망의 `keepInv`, 공격의 힘과 치명타 따위)는 구독자가 받지 못한다.
 
 ## 새 기능 더하기
@@ -190,14 +197,16 @@ handleCancelled 옵션이 멈춤을 기준으로 삼는 까닭은 구독자들
    }
 
    func subscribeJump() {
-       core.Jump.On(core.Normal, "elevator.up", func(e core.PlayerEvent) {
+       core.Jump.On(core.Normal, "elevator.up", func(e core.NoticeEvent) {
            Up(e.S, e.P, e.Tx)
        })
    }
 
    func subscribeSneak() {
-       core.Sneak.Handle(core.Normal, "elevator.down", func(e core.PlayerEvent) bool {
-           return Down(e.S, e.P, e.Tx)
+       core.Sneak.On(core.Normal, "elevator.down", func(e core.PlayerEvent) {
+           if core.Takeover(e.S, e.P) == "" {
+               Down(e.S, e.P, e.Tx)
+           }
        })
    }
    ```
@@ -339,24 +348,27 @@ handleCancelled 옵션이 멈춤을 기준으로 삼는 까닭은 구독자들
 이름은 한 훅 안에서 겹치면 안 된다(훅이 다르면 같아도 된다). 겹치면 `Install` 이 패닉한다. 티어는 겹쳐도 된다.
 다른 구독자의 자리를 보고 고르지 않는다. 내 구독자가 하는 일을 보고 고른다.
 
-1. 이벤트를 가공하지도 취소하지도 않고 결과를 세거나 보여 주기만 한다: MONITOR. 멈춘 이벤트도 봐야 하면 `Always` 로 단다.
-2. 플레이어를 붙잡은 모드처럼 무엇보다 먼저 막아야 한다: LOWEST. 다른 구독자가 읽을 값을 고치거나 그 뒤에 막아도 된다: LOW.
+1. 이벤트를 가공하지도 취소하지도 않고 결과를 세거나 보여 주기만 한다: MONITOR. 취소된 이벤트도 봐야 하면 `core.HandleCancelled` 를 준다.
+2. 플레이어를 붙잡은 모드처럼 무엇보다 먼저 막아야 한다: LOWEST. 다른 구독자가 읽을 값을 고치거나 그 뒤에 막아도 된다: LOW. 든 도구가 블록보다 먼저 클릭을 쓰는 일도 LOW 다.
 3. 다른 구독자가 다 고친 뒤의 값을 보고 마지막으로 가공하거나 취소한다: HIGHEST.
-4. 다른 구독자가 이벤트를 쓰고 남았을 때만 하는 일, 다른 기능이 일을 마친 뒤에 하는 일: HIGH.
+4. 다른 구독자의 일이 끝난 뒤의 상태를 읽어서 하는 일: HIGH.
 5. 그 밖의 일: NORMAL.
 
-같은 티어에 이미 있는 구독자와 겹치는지 본다. 같은 클릭을 둘 다 가져갈 수 있으면 누가 먼저 돌아도 결과가 같게 조건을 적는다(든 아이템, 블록 종류, 웅크림, `island.CanBuild`). 조건으로 가를 수 없으면 티어를 나눈다. 구독자를 달고 `go test ./content/skygear -run TestTierOrder -count=1` 을 돌린다.
+그다음 취소할지 정한다. 내 일 때문에 바닐라 동작이나 뒤의 구독자가 일어나면 안 되면, 조건이 맞을 때만 일하고 취소한다. 그렇지 않으면 취소하지 않는다.
+
+마지막으로 같은 티어에 이미 있는 구독자와 겹치는지 본다. 한쪽이 취소하고 다른 쪽이 같은 이벤트에 일할 수 있으면 조건이 겹치지 않게 적는다(든 아이템, 블록 종류, 웅크림, `island.CanBuild`, `core.Takeover`). 조건으로 가를 수 없으면 티어를 나눈다. 구독자를 달고 `go test ./content/skygear -run TestTierOrder -count=1` 을 돌린다.
 
 ### 순서를 지키는 것
 
-지금 순서가 필요한 곳과 그것을 지키는 방법이다. 같은 티어 안의 자리나 역할에 맞지 않는 티어로 지키는 곳은 "예외"의 넷뿐이다.
+지금 순서가 필요한 곳과 그것을 지키는 방법이다. 같은 티어 안의 자리로 지키는 곳은 없다.
 
 티어가 지키는 것:
 
-- `StartBreak`, `BlockBreak`, `BlockPlace`, `UseOnBlock`: `expand.guard`(LOWEST)가 `island.guard`(LOW)보다 먼저다. 확장 모드에서 섬 밖을 누르면 확장 모드가 막고 지도를 다시 보여 준다. 월드를 바꾸는 단계는 모두 NORMAL 이후다. 가드는 `Handle` 로 멈추면서 `Cancel()` 도 한다.
-- `UseOnBlock`: 든 도구가 클릭을 가져가는 단계(`copier.use`, `wand.use`, LOW)가 블록에 하는 일(NORMAL, HIGH)보다 먼저다. 든 아이템을 쓰는 단계(`handwork.stick`, `fairy.time`, NORMAL)가 블록 자체의 동작(`farm.composter`, `farm.crop`, HIGH)보다 먼저다. 옆 칸 가드(`expand.guardSide`, `island.guardSide`, HIGHEST)는 아무도 클릭을 가져가지 않았을 때 마지막에 막는다.
-- `Sneak`: `expand.show`(LOWEST), `achievement.sneak`(LOW), `elevator.down`(NORMAL), `farm.crouchGrow`(HIGH) 순서다. 엘리베이터가 웅크리기를 쓰면 묘목은 자라지 않는다.
-- `Hurt`: `dimension.frozen`(LOWEST)이 먼저 막고, `dimension.fallen`(HIGHEST)이 남은 피해로 사냥꾼이 쓰러지는지 본다. `Move` 의 `island.void`(HIGHEST)도 마지막에 취소한다.
+- `StartBreak`, `BlockBreak`, `BlockPlace`, `UseOnBlock`: `expand.guard`(LOWEST)가 `island.guard`(LOW)보다 먼저다. 확장 모드에서 섬 밖을 누르면 확장 모드가 막고 지도를 다시 보여 준다. 월드를 바꾸는 단계는 모두 NORMAL 이후다. 가드는 조건이 맞으면 취소하고, 취소된 이벤트는 뒤의 기본 구독자에게 가지 않는다.
+- `BlockBreak`: `storage.front`(LOW)가 서랍 앞면을 친 캐기를 취소한다. 취소된 캐기는 세지 않고 도구 경험치도 광맥 캐기도 없다.
+- `UseOnBlock`: 든 도구가 클릭을 쓰는 단계(`copier.use`, `wand.use`, `fairy.time`, LOW)가 블록에 하는 일(NORMAL, HIGH)보다 먼저다. 요정의 모래시계는 기계에 붙은 도관 판 위를 눌러도 기계를 감는다. 든 아이템을 쓰는 단계(`handwork.stick`, NORMAL)가 블록 자체의 동작(`farm.composter`, `farm.crop`, HIGH)보다 먼저다. 옆 칸 가드(`expand.guardSide`, `island.guardSide`, HIGHEST)는 아무도 취소하지 않았을 때 마지막에 막는다. `machine.noteFace` 와 `handwork.stickOther`(MONITOR)는 취소되지 않고 끝까지 간 클릭만 받는다.
+- `Sneak`: `expand.show`, `elevator.down`(NORMAL), `farm.crouchGrow`(HIGH), `achievement.sneak`(MONITOR) 순서다. 웅크리기는 아무도 취소하지 않으므로 모두 불린다. 엘리베이터가 플레이어를 옮긴 뒤에 `farm.crouchGrow` 가 플레이어 둘레의 묘목을 본다.
+- `Hurt`: `dimension.frozen`(LOWEST)이 먼저 취소하고, `dimension.fallen`(HIGHEST)이 남은 피해로 사냥꾼이 쓰러지는지 보고 취소한다. `Move` 의 `island.void`(HIGHEST)도 마지막에 취소한다. 취소된 피해와 이동은 `achievement.hurt`, `achievement.walk`(MONITOR)가 세지 않는다.
 - `ItemDamage`: `tinker.ironHammer`(LOW)가 피해를 0 으로 고친 뒤에 `autofill.broken`(HIGH)이 읽는다.
 - `BlockPlace`: `netstore.place`(HIGH)는 `machine.place`(NORMAL)가 블록 설치를 미뤄 둔 뒤에 망을 버린다.
 - `Leave`: 기능이 퇴장 때 하는 일(`dimension.leave`, `expand.stop`, NORMAL)이 클라이언트에 남은 것을 지우는 정리(`look.clear`, `screens.board`, `progress.recipes`, HIGHEST)보다 먼저다. 확장 모드가 끝나면 사이드바를 다시 그리고, 사냥이 끝나면 레시피를 다시 보낼 수 있다.
@@ -365,24 +377,24 @@ handleCancelled 옵션이 멈춤을 기준으로 삼는 까닭은 구독자들
 
 구독자의 조건이 지키는 것:
 
-- 사냥 중인 플레이어가 블록을 누르면 무엇을 들었든 `dimension.hunting` 이 막는다. `screens.menuItem`(`UseOnBlock`)과 `dimension.stone` 은 사냥 중이면 클릭을 가져가지 않는다. 공중에서는 메뉴 아이템이 사냥에서 나가는 길이라 `dimension.hunting`(`Use`)이 메뉴 아이템을 든 클릭을 막지 않는다.
-- 메뉴 아이템과 이차원석은 어디서나 자기 화면을 연다. `expand.guard`(`UseOnBlock`)는 그런 아이템(`sgitem.OpensScreen`)을 든 클릭을 막지 않는다.
-- `copier.use` 와 `wand.use` 는 섬이 짓기를 허락하는 곳(`island.CanBuild`)에서만 클릭을 가져간다. 섬 밖에서는 `island.guard` 가 막는다.
+- 사냥 중인 플레이어가 블록을 누르면 무엇을 들었든 `dimension.hunting` 이 취소한다. `screens.menuItem`(`UseOnBlock`)과 `dimension.stone` 은 사냥 중이면 클릭을 쓰지 않는다. 공중에서는 메뉴 아이템이 사냥에서 나가는 길이라 `dimension.hunting`(`Use`)이 메뉴 아이템을 든 클릭을 취소하지 않는다.
+- 메뉴 아이템과 이차원석은 어디서나 자기 화면을 연다. `expand.guard`(`UseOnBlock`)는 그런 아이템(`sgitem.OpensScreen`)을 든 클릭을 취소하지 않는다.
+- `copier.use`, `wand.use`, `fairy.time` 은 섬이 짓기를 허락하는 곳(`island.CanBuild`)에서만 클릭을 쓴다. 섬 밖에서는 `island.guard` 가 취소한다. `storage.front` 도 같은 조건을 단다.
+- `UseOnBlock` 의 NORMAL 구독자는 블록과 든 아이템과 웅크림으로 갈린다. `conduit.plate` 는 서서 누른 도관 판, `storage.sneakUse` 는 웅크리고 누른 서랍, `handwork.stick` 은 막대기로 누른 흙·잔디와 웅크리고 누른 상자, `handwork.gather` 는 도끼로 누른 원목과 괭이로 누른 흙이다.
+- `Sneak` 의 `expand.show` 는 섬 확장 모드에서만 화면을 다시 보여 준다. `elevator.down` 과 `farm.crouchGrow` 는 다른 모드가 플레이어를 붙잡고 있으면(`core.Takeover`) 일하지 않는다.
+- `handwork.stickOther` 는 막대기로 흙도 잔디도 아닌 블록을 눌렀고 아무도 그 클릭을 취소하지 않았을 때 `stick_other` 를 센다. 서랍 메뉴, 도관 판, 작물, 가득 찬 퇴비통이 쓴 클릭은 세지 않는다.
 
 커널이 지키는 것:
 
 - `Leave`: 플레이어 클라이언트의 도형 지우기와 `Players` 항목 지우기는 구독자가 아니다. `Lifecycle.Leave` 가 모든 구독자 뒤에 한다. 퇴장 구독자는 어느 티어에 서든 플레이어별 데이터를 읽을 수 있다.
 - `Second`, `Autosave`, `Close`, `NewDay` 의 꼬리 일도 구독자 뒤다. `Close` 의 구독자는 세션이 아직 등록된 상태에서 돈다.
-- `Move`: 이벤트의 `From` 은 이벤트를 만들 때 플레이어가 서 있던 곳이다. `island.void` 가 플레이어를 옮긴 뒤에 돌아도 `achievement.walk` 는 같은 거리를 센다.
-- 커널의 구독자는 `Join` 의 `core.shapes` 하나다(NORMAL). `core` 가 목록의 처음이라 그 티어에서 먼저 돌지만, 다른 구독자와 같은 규칙을 따른다.
+- `Move`: 이벤트의 `From` 은 이벤트를 만들 때 플레이어가 서 있던 곳이다. 앞의 구독자가 플레이어를 옮겨도 `achievement.walk` 는 같은 거리를 센다.
+- 커널의 구독자는 `Join` 의 `core.shapes` 하나다(NORMAL). 다른 구독자와 같은 규칙을 따른다.
 
-### 예외
+### 역할과 다른 티어
 
-여섯 티어로 풀리지 않아 같은 티어 안의 자리(기능 목록 순서)나 역할에 맞지 않는 티어로 지키는 곳이다. 자리로 지키는 쌍은 루트의 `orderKept` 에 있고 `TestTierOrder` 가 그 쌍만 순서를 지킨 채 섞는다. 목록은 늘리지 않는다.
+역할에 맞지 않는 티어에 둔 구독자는 하나다. 예전 동작을 지키려고 둔 것이고, 목록은 늘리지 않는다.
 
-- `UseOnBlock` 의 `storage.sneakUse`, `conduit.plate` 다음 `handwork.stick`(모두 NORMAL): `handwork.stick` 은 흙·잔디가 아닌 블록에 막대기를 쓴 횟수(`stick_other`)도 센다. 서랍 메뉴나 도관 판이 클릭을 가져가면 세지 않는다.
-- `UseOnBlock` 의 `conduit.plate` 다음 `fairy.time`(둘 다 NORMAL): 기계에 붙은 도관 판을 요정의 모래시계로 누르면 판 설정이 열리고 모래시계는 쓰이지 않는다.
-- `Sneak` 의 `achievement.sneak`(LOW): 세기만 하는 구독자라 역할은 MONITOR 다. 엘리베이터를 탄 웅크리기는 세고 섬 확장 모드의 웅크리기는 세지 않으려고 `expand.show` 와 `elevator.down` 사이에 둔다.
 - `Join` 의 `dimension.joined`(HIGH): 서버가 멈출 때 사냥 중이던 플레이어의 소지품을 되돌리는 일이라 역할로는 다른 입장 단계보다 먼저(LOW)다. 지금 순서를 지키려고 뒤에 둔다. 앞의 입장 단계가 그 플레이어의 인벤토리에 넣거나 고친 것(`fairy.backfillDust` 의 가루, `screens.menuItem` 이 고친 메뉴 아이템 이름)은 사냥 장비와 함께 사라진다.
 
 ## 기능의 데이터
@@ -473,7 +485,7 @@ func init() { coretest.Setup(farm.Feature) }
 
 | 규칙 | 테스트 |
 |---|---|
-| 루트는 아무것도 구독하지 않는다. 루트 패키지의 비테스트 파일에는 훅의 `On`·`Handle` 호출, `core.Mark` 호출, `Handle...` 메서드가 없다 | `TestRootSubscribesToNothing` |
+| 루트는 아무것도 구독하지 않는다. 루트 패키지의 비테스트 파일에는 훅의 `On` 호출, `core.Mark` 호출, `Handle...` 메서드가 없다 | `TestRootSubscribesToNothing` |
 | 모든 구독자는 목록에 오른 기능이 `Setup` 에서 단다. 이름의 접두어가 그 기능 이름이다 | `TestEverySubscriberHasItsFeature`, `core.TestInstallingPrefix` |
 | `feature.go` 가 있는 패키지는 `Features` 에 정확히 한 번 오르고 `Name` 이 패키지 이름이다 | `TestEveryFeatureListed`, `TestFeatureNames` |
 | 기능 패키지는 루트를 import 하지 않는다(간접도). `core` 는 기능을 import 하지 않는다. `common` 은 `content/`, `platform/` 을 import 하지 않는다 | `TestFeaturesLiftable` |
@@ -492,19 +504,22 @@ func init() { coretest.Setup(farm.Feature) }
 | 테스트 지원 패키지(`core/coretest`, `machine/machinetest`)는 테스트 파일만 import 한다 | `TestSupportPackagesAreTestOnly` |
 | 패키지의 테스트는 루트를 import 하지 않는다(`wire` 는 예외). 내용 목록 아래 칸의 테스트는 `catalog`, `codex`, `screens` 를 import 하지 않는다. `core` 밖의 테스트는 `core.Install` 이나 기능의 `Setup` 을 직접 부르지 않고 `coretest.Setup` 을 쓴다 | `TestPackageTestsStayBelow` |
 | 플레이어 핸들러의 모든 메서드가 `core.PlayerHandler` 에 있고 훅마다 구독자가 있으며, 루트가 같은 이름의 메서드를 두지 않는다 | `TestHandlerMethodsOneWay` |
-| 핫 경로는 할당하지 않는다. 이벤트마다 `AllocsPerRun` 으로 0 을 확인한다. 멈춘 뒤 `Always` 구독자를 도는 길도 같다 | `core.TestEmitAllocatesNothing`, `core.TestEmitAllocatesNothingAfterCancel`, `core.TestEventsEmitAllocateNothing` |
+| 핫 경로는 할당하지 않는다. 이벤트마다 `AllocsPerRun` 으로 0 을 확인한다. 취소된 이벤트를 도는 길도 같다 | `core.TestEmitAllocatesNothing`, `core.TestEmitAllocatesNothingAfterCancel`, `core.TestEventsEmitAllocateNothing` |
 | 구독자는 `Install` 동안에만 단다. 첫 `Emit` 뒤의 등록, 같은 훅 안의 이름 겹침, 없는 티어는 패닉이다 | `core.TestRegisterAfterEmitPanics`, `core.TestSameNamePanics`, `core.TestNotATierPanics` |
-| 티어는 상수의 순서대로 돌고, 같은 티어 안에서는 등록한 순서대로 돈다 | `core.TestRunsInTierOrder`, `core.TestSameTierRunsInRegistrationOrder` |
-| 구독자는 같은 티어 안의 자리에 기대지 않는다. 티어 안의 순서를 뒤집고 섞어도 모든 테스트가 통과하고 장면의 결과와 업적 카운터가 같다 | `TestTierOrder` |
-| MONITOR 구독자는 멈추지도 취소하지도 가공하지도 않는다 | `core.TestHandleAtMonitorPanics`, `TestMonitorSubscribersDoNotShape` |
-| `Always` 구독자는 멈춘 이벤트도 받고 `cancelled` 로 구별한다. `On` 구독자는 받지 않는다. Dragonfly 의 취소만으로는 아무도 건너뛰지 않는다 | `core.TestAlways`, `core.TestAlwaysAfterCancelRunsOnlyAlways`, `core.TestCancelledTellsTheContext` |
-| `Leave` 의 구독자가 모두 돈 뒤에, 멈췄어도, 플레이어별 파트 항목을 지운다 | `core.TestLeaveDropsThePlayersEntriesLast` |
-| 구독자의 조건이 지키는 순서: 사냥꾼의 이차원석, 섬 밖의 복사기와 지팡이, 공허로 떨어진 걸음 | `TestHunterStoneOpensNothing`, `TestToolsLeaveTheClickOutsideTheIsland`, `TestWalkCountsTheMoveIntoTheVoid` |
-
-핫 경로(이동, 매초, 빠른 틱)를 건드리면 벤치마크로 재 본다. 이벤트 하나의 비용은 `core` 의 `BenchmarkEmit`, `BenchmarkEmitMove` 와 루트의 `BenchmarkHandleMove`, `BenchmarkSecond`, `BenchmarkUseOnBlock`, `BenchmarkUse` 가 잰다. 패널 하나의 비용은 루트의 `BenchmarkLookMachine`(시선 추적과 패널 보내기), `BenchmarkLookInfoMachine`, `BenchmarkLookInfoStone`(패널 만들기)이 잰다. 다른 기능이 동작을 등록한 기계의 틱 한 번은 `BenchmarkTickHarvester`, `BenchmarkTickDrawerController` 가 잰다.
+| 티어는 상수의 순서대로 돈다. 같은 티어 안에서는 등록한 순서로 돌지만 구독자는 그 순서에 기대지 않는다 | `core.TestRunsInTierOrder`, `core.TestSameTierRunsInRegistrationOrder` |
+| 같은 티어의 구독자 사이에는 순서가 없다. 티어 안의 순서를 뒤집고 섞어도 모든 테스트가 통과하고 장면의 결과, 업적 카운터, 플레이어의 인벤토리가 같다. 순서를 지켜 주는 쌍은 없다 | `TestTierOrder` |
+| 어느 구독자도 다른 구독자를 멈추지 못한다. 취소할 수 없는 이벤트의 훅에서는 모두 불린다 | `core.TestNoSubscriberStopsAnother`, `core.TestOnlyActorEventsCancel` |
+| 취소된 이벤트는 기본 구독자에게 가지 않고 `core.HandleCancelled` 구독자에게만 간다. 취소는 Dragonfly 의 컨텍스트 하나다 | `core.TestCancelSkipsDefaultSubscribers`, `core.TestAfterCancelOnlyHandleCancelledRuns`, `core.TestCancelIsDragonflysCancel`, `core.TestCancelledTellsTheContext` |
+| MONITOR 구독자는 취소하지도 가공하지도 않는다 | `TestMonitorSubscribersDoNotShape`, `core.TestMonitorCancelPanicsWhileTraced` |
+| `Leave` 의 구독자가 모두 돈 뒤에 플레이어별 파트 항목을 지운다 | `core.TestLeaveDropsThePlayersEntriesLast` |
+| 구독자의 조건이 지키는 것: 사냥꾼의 이차원석, 섬 밖의 복사기와 지팡이, 모드에 붙잡힌 플레이어의 웅크리기, 끝까지 간 클릭만 적는 놓을 면 | `TestHunterStoneOpensNothing`, `TestToolsLeaveTheClickOutsideTheIsland`, `elevator.TestHeldPlayerDoesNotRideDown`, `farm.TestHeldPlayerGrowsNoSapling`, `TestRefusedClickNotesNoFace` |
+| 취소된 일은 일어나지 않은 일이다: 공허로 떨어진 걸음, 사냥꾼을 쓰러뜨린 타격, 서랍 앞면을 친 캐기는 세지 않는다. 웅크리기는 언제나 센다 | `TestMoveIntoTheVoidIsNotWalked`, `TestFellingHitIsNotAHurt`, `TestDrawerFrontHitIsNoBreak`, `TestEverySneakCounts`, `TestSneakRideGrowsSaplingsWhereItArrives` |
+| 든 도구가 블록보다 먼저 클릭을 쓴다. 막대기로 다른 블록을 두드린 기록은 아무도 쓰지 않은 클릭만 센다 | `TestHourglassWindsTheMachineBehindAPlate`, `TestStickOtherCountsTheTapNothingTook` |
+
+핫 경로(이동, 매초, 빠른 틱)를 건드리면 벤치마크로 재 본다. 이벤트 하나의 비용은 `core` 의 `BenchmarkEmit`, `BenchmarkEmitMove` 와 루트의 `BenchmarkHandleMove`, `BenchmarkSecond`, `BenchmarkUseOnBlock`, `BenchmarkUse`, `BenchmarkBlockBreak`, `BenchmarkBlockPlace` 가 잰다. 구독자는 취소되지 않은 이벤트마다 모두 불리므로, 조건은 싼 것부터 본다(든 아이템의 종류, 그다음 맵 조회나 블록 읽기). 패널 하나의 비용은 루트의 `BenchmarkLookMachine`(시선 추적과 패널 보내기), `BenchmarkLookInfoMachine`, `BenchmarkLookInfoStone`(패널 만들기)이 잰다. 다른 기능이 동작을 등록한 기계의 틱 한 번은 `BenchmarkTickHarvester`, `BenchmarkTickDrawerController` 가 잰다.
 
 ```sh
-go test -run xxx -bench 'HandleMove|Second|UseOnBlock|Use$|LookMachine|LookInfo|TickHarvester|TickDrawerController' -benchmem -count 5 ./content/skygear
+go test -run xxx -bench 'HandleMove|Second|UseOnBlock|Use$|BlockBreak|BlockPlace|LookMachine|LookInfo|TickHarvester|TickDrawerController' -benchmem -count 5 ./content/skygear
 go test -run xxx -bench Emit -benchmem -count 5 ./content/skygear/core
 ```
 
@@ -516,7 +531,7 @@ go test -run xxx -bench Emit -benchmem -count 5 ./content/skygear/core
 
 | 파일 | 담는 것 | 테스트 |
 |---|---|---|
-| `content/skygear/testdata/pin/order.golden` | 모든 훅의 구독자를 도는 순서대로 `<훅> <티어> <이름>`. `Always` 로 단 구독자는 끝에 ` always` 가 붙는다 | `TestPinOrder` |
+| `content/skygear/testdata/pin/order.golden` | 모든 훅의 구독자를 훅과 티어 순서대로 `<훅> <티어> <이름>`. 한 티어 안은 이름순이다(순서가 없으므로). `core.HandleCancelled` 로 단 구독자는 끝에 ` handleCancelled` 가 붙는다 | `TestPinOrder` |
 | `content/skygear/testdata/pin/handler-simple.golden`, `handler-blocks.golden`, `handler-use.golden` | 장면마다 이벤트가 거친 구독자 이름과 결과(취소 여부, 효과) | `TestPinSimple`, `TestPinBlocks`, `TestPinUse` |
 | `content/skygear/testdata/pin/lifecycle.golden` | 입장·퇴장·매초·빠른 틱·새 날·자동 저장·닫기·시작의 구독자 이름과 결과 | `TestPinLifecycle` |
 | `content/skygear/testdata/pin/settings.golden` | 설정이 등록된 순서 | `TestPinSettings` |
@@ -539,11 +554,11 @@ go test ./content/skygear -run 'TestOldSaveLoads' -count=1   # 옛 세이브 읽
 장면 골든의 한 줄은 `<장면>: <거친 구독자 이름> | <결과>` 다. `|` 앞은 등록한 순서로 돌았을 때의 자취이고 뒤가 결과다.
 `TestTierOrder` 는 루트의 테스트 바이너리를 여섯 번 다시 돌린다. 한 번은 등록한 순서 그대로, 다섯 번은 모든 훅의 구독자를 티어 안에서 뒤집거나 섞어서다(`SKYGEAR_HOOK_ORDER=reverse`, `shuffle:1` 부터 `shuffle:4`). 섞은 실행에서 확인하는 것은 셋이다.
 
-- 루트의 모든 테스트가 통과한다. 장면 골든은 `|` 뒤의 결과만 견주고, `order.golden` 은 한 훅의 한 티어에 든 줄을 집합으로 견준다.
+- 루트의 모든 테스트가 통과한다. 장면 골든은 `|` 뒤의 결과만 견준다.
 - 장면 234개의 결과가 골든과 글자 하나까지 같다.
-- 장면마다 플레이어의 업적 카운터가 늘어난 양이 등록한 순서로 돈 실행과 같다. 카운터는 골든에 없어서 따로 견준다.
+- 장면마다 플레이어의 업적 카운터가 늘어난 양과, 장면의 플레이어가 든 아이템(칸, 종류, 개수)이 등록한 순서로 돈 실행과 같다. 둘 다 골든에 없어서 따로 견준다.
 
-플레이어가 받는 팁·글·소리와 어느 장면도 읽지 않는 상태는 이 테스트가 보지 못한다. "예외"의 쌍(`orderKept`)은 섞어도 순서를 지킨다.
+플레이어가 받는 팁·글·소리와 어느 장면도 읽지 않는 상태는 이 테스트가 보지 못한다. 순서를 지켜 주는 쌍은 없다. 모든 구독자를 티어 안에서 섞는다.
 기능 패키지의 테스트도 같은 변수로 섞어서 돌릴 수 있다. `coretest.Session` 이 처음 불릴 때 섞는다.
 
 ```sh
~~~~

