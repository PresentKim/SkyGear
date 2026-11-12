# No propagation stop, part 2: reference (hooks and mediators, E4, the kernel switch rebuilt, the items used on the air, the death in the second dimension, the block clean-up; T140 to T150)

Reference for the machine plans `docs/plans/2026-11-12-no-propagation-stop-plan-c.md` (T140 to T143), `-plan-d.md` (T144 to T146) and `-plan-e.md` (T147 to T150). Not a machine plan, not linted. It continues REF1, `docs/plans/2026-11-12-no-propagation-stop.md`: where a section of REF1 still holds it is named by its heading and only the differences are given here. Everything was built and run in a scratch worktree on 3848f7f (T135); the per-task prototype patches are in `/tmp/claude-0/-home-user-SkyGear/a71fa8d5-6f20-5c3f-ba20-eeb27e13ad8a/scratchpad/proto-nostop2-tasks/` (`T140.patch` to `T150.patch`, in id order on that commit, `git diff --binary`, and `combined.patch`).

## 1. The ruling and what it does to plan B

The owner's ruling after the review of click delivery (`scratchpad/review-clicks.md`): where Dragonfly's own mediator (`block.Activatable`, `item.UsableOnBlock`, `item.Usable`, `block.Punchable`, the block's own change and break callbacks) can carry a click, a hook is a wrong case of event-isation and the mediator is used. Where no mediator can carry it, the hook stays, and a held tool goes first like the other held tools.

T130 to T135 are committed as planned (3031de0, 6b6d59d, 6463645, 1c17cc6, 6f762a4, 3848f7f). T136 to T139 of plan B are superseded:

| Plan B | What happens to it | Here |
|---|---|---|
| T136 `fairy.time` joins the held tools in LOW (C8) | NOT built. Its tier move, `fairy.TimeItem`, the `island.CanBuild` condition, `hourglassplate_test.go` and behaviour change C8 are dropped. What stays of it: `orderKept` and the kept-pair mechanism go (`coretest.ReorderHooks()` takes no pairs). | T140 |
| T137 `machine.noteFace` to MONITOR (C9) | NOT built. The subscriber is removed instead: the machine item carries the face (section 3). C9 is replaced by C13. | T142 |
| T138 the kernel switch | rebuilt on the new chain; the design of REF1 section 9 is unchanged. Two registrations and `order.golden` differ from the old patch. | T143 |
| T139 the documents | rebuilt: the text of REF1 "REF T139" plus the section on hooks and mediators, the death and the list of real-client checks; E4 is no longer an exception. | T146 |
| (REF1 section 8, E4, "not in this plan") | built: `dimension.joined` in LOW (C12). | T141 |
| (review P3) | the items used on the air are `item.Usable` and do their work in their own `Use`, by the owner's later answer (no client check first). | T144 |
| (review question 2, T133's half on `dimension.fallen`) | the owner chose design (ii): the hit that kills a hunter lands, the hunter dies for real and respawns at once. C6 is replaced. | T145 |
| (review P4, P5) | the block clean-up through the blocks' own callbacks, last. | T147 to T150 |

Two of the owner's answers arrived while this was built (the death design, the items used on the air) and are folded in as T144 and T145, after the kernel switch (section 3.7 says why after).

## 2. Dragonfly's order, and the rule

Read in `third_party/dragonfly/server/player/player.go` (`UseItemOnBlock`, `BreakBlock`); the review has the line numbers.

1. The handler (`HandleItemUseOnBlock`): every subscriber of `UseOnBlock`, tier by tier. Cancelled: the blocks are resent and nothing else happens.
2. The clicked block's `Activate`, unless the player sneaks with an item in hand. true: the arm swings, the `item.UseContext` is applied to the hand, the click is over.
3. The held item's `UseOnBlock`. true: the arm swings, the context is applied.
4. Otherwise a held block is placed (`HandleBlockPlace`).

A click on the air (`UseItem`): the handler (`HandleItemUse`), cancelled: nothing else; then the held item's `Use` (`item.Usable`). true: the arm swings and Dragonfly writes the stack it read BEFORE `Use` back into the hand, so a `Use` that changes the hand itself returns false.

A finished break: the drops are computed, the handler (`HandleBlockBreak`), the block is removed, then `BreakInfo().BreakHandler(pos, tx, user)`. An explosion and a liquid that washes a block away remove the block and call the same `BreakHandler` with a nil user; they raise no handler event.

The rule that follows, as it is written into `docs/tech/extension.md` by T146:

- A tool that works on one block type of this repository is used by that block's `Activate`, plus the item's `UseOnBlock` if it must also work while sneaking: the tier kit, the drawer key and upgrade, the bucket on a tank, and now the fairy hourglass.
- A subscriber stays where no mediator can carry the work: a guard (any block, any item); a held tool that must act before the clicked block's `Activate` on any block (copier, wand: LOW; menu item, dimension stone: LOWEST); a vanilla block or item this repository cannot add a method to (stick, axe, bone meal, crops, composter), or a clicked block that is not the feature's (the connector plate is drawn on the neighbour of the bundle); a record.
- Every subscriber runs before every mediator. So a mediator is refused wherever a guard refuses the click: this is C10.
- The pin scenarios call the handler only. Work a mediator does is invisible to them and has its own test through Dragonfly (`Player.UseItemOnBlock`, `Player.BreakBlock`, an explosion).

## 3. What this plan decides

### 3.1 The hourglass (T140), and what `fairy.time` still needs

Built as the review prototyped it. `Machine.Activate` winds the machine beside the tier kit; `sgitem.FairyHourglass.UseOnBlock` winds it for a sneaking player through the one-method interface `sgitem.HourglassWound`, which `Machine` satisfies. On the plate, standing: `conduit.plate` cancels in the handler, so neither the machine nor the item is asked (ruling 2: the plate first, as today).

`fairy.time` keeps the two bone meals and needs NOTHING of T136:

- No tier move: it stays in NORMAL. It must act before `farm.crop` (HIGH), which is unchanged.
- No `island.CanBuild` condition: T136 added it only to make `fairy.time` exclusive with `island.guard` inside LOW. In NORMAL the guard (LOW) has cancelled a click outside the island before `fairy.time` is reached.
- No kept pair: `conduit.plate` and `fairy.time` never act on the same click. `fairy.time` acts on a crop, a sapling or a magic plant (`UseFairyTime`, `BoneMeal`); a connector plate exists only on the face of a block that gives or takes energy, items or fluid (`PlateKinds`, `itemio.IO`), which none of those is. `TestTierOrder` passes in all its orders with no pair, and `TestHourglassThroughTheMachine` passes in the order of registration, reversed and in four shuffles.

A form without C10 at no cost was looked for and not found. The difference comes from Dragonfly's order itself (every subscriber, also `island.guardSide` in HIGHEST, before every mediator). Removing it means either keeping the hourglass a subscriber in front of the side guard (what the ruling ends) or teaching the guard to let a held hourglass on a machine through (a guard that names another feature's item, and a second rule for the same face on which the machine's window is refused). Neither is free; C10 is built as accepted.

A second visible difference the review did not list: C11 (the arm swing). It also follows from Dragonfly's order and has no cheaper form.

A mechanical consequence the review did not run into: `fairy` no longer imports `machine`, so the `siblings` row in `layers_test.go` becomes `"fairy": {"farm"}` (`TestFeatureSiblings` fails otherwise).

### 3.2 `machine.noteFace`: replaced now (T142), not moved to MONITOR first

Chosen: "replace T137 by the move now". T137 would move one subscriber, add a test and re-record 51 traces, all of which the ruling throws away when the subscriber goes, and it would not even close C9: a click that passes every subscriber can still be taken by the clicked block's `Activate` (a chest opens), and the note written in MONITOR stays behind (run: `T137`'s form on `TestMachinePlacedWithoutAClickFacesItsDefault` fails with `a miner placed after a click that opened a chest faces east, want down`). The move is built and run here on the full suite and in every order; its risk (the item repeats Dragonfly's placement) is the same before or after the switch, and doing it first makes the switch one subscriber smaller.

How it is built: `Machine` is an `item.UsableOnBlock`. Its `UseOnBlock` finds the place exactly as `Player.UseItemOnBlock` does for a block item (the clicked block if it is replaceable by the machine, else the block beside the clicked face; replaceable and inside the world), sets two unexported fields on the block it hands to `block.Placer.PlaceBlock` (`clicked`, `placedOn`) and returns whether an item was spent. `machine.place` hands the block to `Place`, which turns a miner or a network attachment to face the clicked block when `clicked` is set. The fields live on the transient value only: the world stores a block by its hash and NBT, and `Place` sets the machine with its new state in a deferred call.

Why the face is not carried in the block's `State`: the construction wand hands `HandleBlockPlace` the clicked block itself when it extends a row of the same kind (`wand.toPlace` returns `support`), a `Machine` with the State of a machine that stands in the world. `Place` must never read that State (two machines would share it, or a wand-placed machine would copy its neighbour's facing). It reads the two fields and nothing else.

The pair `machine.noteFace` / `conduit.plate` no longer exists: there is no subscriber.

### 3.3 E4 (T141): what moves with `dimension.joined`

`Join` after T141: LOWEST `screens.menuItem`; LOW `dimension.joined`; NORMAL `core.shapes`, `progress.sendRecipes`, `island.purgeLost`, `climate.showWeather`, `fairy.backfillDust`; MONITOR `screens.board`. `dimension.joined` is alone in its tier.

It moves in front of exactly the five NORMAL steps. Read one by one for what they do with the player's inventory, armour, off hand or game mode (the things `dimension.Joined` and `RetireItems` change): `core.shapes` sends shapes, `progress.sendRecipes` sets the recipe filter of the island, `island.purgeLost` prunes the island's list of lost items, `climate.showWeather` sends the weather. Only `fairy.backfillDust` touches the player: it puts the dust into the inventory (`Session.GiveItem`: what does not fit is dropped at the player's feet).

| What | Before (HIGH) | After (LOW) |
|---|---|---|
| a player the server stopped under in the second dimension, on a save still owed its fairy dust | the dust goes into the kit's inventory and is cleared with it when the carried items come back; the save is marked as paid | the carried items come back first; the dust goes into that inventory and stays (C12) |
| the same, when the returned inventory is full | (the dust was lost anyway) | the dust that does not fit lies at the player's feet |
| a watcher the server stopped under | the game mode is set back after the NORMAL steps | before them; no NORMAL step reads the game mode |
| `RetireItems` (fixes old relics in the inventory) | after the NORMAL steps | before them; no NORMAL step gives a relic, and `dimension.second` fixes every player every second |
| the menu item of the returned inventory | keeps the name it had (`screens.menuItem`, LOWEST, renamed the kit's copy before the restore) | the same: LOWEST is still before LOW |
| the side board (`screens.board`, MONITOR) | drawn after the restore | the same |

So the behaviour change is C12 alone. `lifecycle.golden` moves in its trace part only: the five `Join/` lines (`dimension.joined` stands after `screens.menuItem` instead of after `fairy.backfillDust`); their result parts are byte for byte the same (the check script prints `lifecycle: 49 scenarios, 5 traces changed` and no `RESULT`).

The scenario needs the state "the save says the player is in the second dimension, no run is under way": a second session over the state document the first one stored. `coretest.SessionOver(t, state, store)` is new for it (`coretest.Session(t)` is `SessionOver(t, nil, nil)`).

### 3.4 The items used on the air (T144)

`copier.clear`, `wand.options`, `tinker.veinToggle` and the air half of `dimension.stone` leave the `Use` hook. The copier, the wands (`Wand`, `InfinityWand`), the hammers (`Hammer`, stone and iron) and the teleport stone are `item.Usable` and do the work in their own `Use`.

The item types live in `sgitem`, a data layer below `core` (`layers_test.go`), so a method there cannot call a feature. Built: `sgitem/airuse.go` has one function variable per item (`CopierUsed`, `WandUsed`, `HammerUsed`, `StoneUsed`), which the feature sets from its `Setup` (in `subscribe.go`, where the subscriber was); an item whose function is not set does nothing. It is the form the repository already has for a call upwards (`conduit.NetChanged`, `core.ShowMenu`). The alternative, moving each type into its feature as `fairy.Board` is, moves six registered item types and everything that names them (the creative list, recipes, `naming`, the tinker's and the wand's helpers in `sgitem`): not done.

Every `Use` returns false, whatever the function did: the copier's clear changes the stack in hand, and for a true result Dragonfly writes the stack it read before the call back over it (and swings the arm).

What is kept, each asserted through `Player.UseItem` (`airuse_test.go`):

- the conditions of each use: the copier is emptied when it holds a card; a wand shows its settings while sneaking; the iron hammer switches diagonal vein mining while sneaking, the stone hammer does nothing; the stone opens its window; all four behind the session's 500 ms debounce, as before;
- the guard: a hunter's use on the air is cancelled by `dimension.hunting` (LOWEST) whatever is held but the menu item, so Dragonfly never calls `Use`. A hunter's copier, wand, hammer and stone do nothing;
- island expansion mode has no guard on a use on the air, today or after: the four uses work there as on the island (a kept fact, asserted, not a change). A use on the air has no position, so the island's border is not concerned.

What differs:

- `dimension.stone` no longer cancels the use on the air (it did, to no effect a player sees: REF1 section 12.1). `autofill.use` (HIGH) now runs for a stone and does nothing. The three `Use/stone…` scenario lines say `cancelled=false`.
- The client entries of seven items gain `minecraft:use_modifiers` (`clientdef.ItemEntry` gives it to every custom `item.Usable`): `skygear:config_copier`, `skygear:stone_wand`, `skygear:iron_wand`, `skygear:diamond_wand`, `skygear:infinity_wand`, `skygear:hammer`, `skygear:stone_hammer`. `skygear:dimension_stone` had it. Proven line by line: `items.golden` and `items.byname.golden` change in exactly 14 lines each (`registry/<name>` and `start/<name>` of the seven), and the dump of the entries (`SKYGEAR_GOLDEN_DUMP`) differs in those 28 entry files by exactly the added block `minecraft:use_modifiers: { use_duration: float32(0.05) movement_modifier: float32(1) }`. This is behaviour change C14; what a client does with it is unverified (section 9, real-client checks).

The pin scenarios of `Use/` go on, after the handler, to the held item's own `Use` unless the use was cancelled, as `Player.UseItem` does (the scenarios of the thrown items already imitate what the game does after the handler). So the result parts of the copier, wand and hammer lines stay byte for byte: the golden itself shows the work is still done.

### 3.5 The death in the second dimension (T145)

The owner chose design (ii) of the review: the hit that kills a hunter is not cancelled; the hunter dies for real and respawns at once.

**What the server does, in order** (proven by `seconddeath_test.go` through a real `Player.Hurt`, except where "read" is said):

1. `Player.Hurt`: the handler's subscribers of `Hurt`. `dimension.frozen` (LOWEST) still cancels a hit while the hunter chooses an upgrade. `dimension.fallen` is gone. `achievement.hurt` counts the hit (C16).
2. Dragonfly sets the health to zero and kills the player (`kill`): the death is shown to the players around, then `HandleDeath`.
3. The platform's handler sets keep-inventory for every death, lobby and instance, before the game's handler (`platform/instance/handler.go`, `HandleDeath`). Pinned now by `instance.TestDeathKeepsTheInventoryEverywhere`. The game's tests do not run the platform's handler; the death tests wrap the game's handler with the same one line (`keepsInventory`).
4. `core.Death`: `tinker.keepFood` and `dimension.died` (NORMAL), `achievement.death` (MONITOR, C17). `dimension.Died` ends the run (`Run.end`, `endDied`): the monsters and the gate go, the hunter's speed, health limit and experience come back (`gear.undress`), what the hunter carried comes back and the kit goes (`undressFromHunt`: the stash leaves the save), the camera is reset, the run is recorded, `wave_fail` counts, the rewards go into the inventory, and the result and the return place wait for the respawn (`runLocal.fallen`). `Run.end` returns at once when the run is no longer the save's run, so nothing ends it twice: `wave_fail` is 1 after a second `Died` and after `Leave`.
5. Back in Dragonfly: with keep-inventory nothing is dropped and the experience is not reset (Dragonfly resets it only where it drops the items); every effect is removed.
6. The respawn (`Player.Respawn`, read: it needs a connected client, so no test of this repository can run it; the tests call the respawn handler as it does): health full, hunger reset, `HandleRespawn`. The platform sets the island's spawn and the instance's world; `dimension.respawned` (LOW) sets the place the hunter went in from (`Run.Return`, the same place the other ends of a run use) and takes the game rule back; `tinker.restoreFood` puts the food level of the moment of death back. Dragonfly takes the player out of the world, adds it again and teleports it there.
7. `landAfter` (50 ms) later the hunter is held on safe ground there until the client has the chunks (`instance.PlaceAt`), as at the end of a run it survived; 500 ms later the result shows.

**What had to be added** to the path that already existed (`dimension.died`, `dimension.respawned` were the fallback nothing reached): the return place (it was the island's spawn), the landing, the game rule, the guard on the monster's hit effect. **What goes:** `dimension.fallen`, `dimension.Fallen`, the "healed without dying" branch of `Run.end`. No dead code is left: `Died` and `Respawned` are the real path.

**"At once".** `Player.EnableInstantRespawn()` sends the packet `GameRulesChanged` with `doimmediaterespawn = true` to that one player's connection (`third_party/dragonfly/server/session/player.go`, `EnableInstantRespawn`). It is per player, not per world. It is sent in `dimension.Begin` when the run starts and taken back (`DisableInstantRespawn`) wherever the run ends for a living hunter (`Run.end`) and in `Respawned` for a dead one. A hunter that disconnects loses it with the connection. So a death on the island keeps the death screen: the rule is on only between the start of a run and its end or the respawn. This is by reading: a test player has no connection and no test can see the packet. Whether a client honours a rule sent in the middle of a session, and how fast, is a real-client check (section 9).

**The fixes the review said (ii) needs:**

| Point of the review | Built |
|---|---|
| the landing: the respawn only sets a position | `Respawned` schedules `instance.PlaceAt` for the respawned player (`landAfter`) |
| the place: the death path went to the island's spawn | the return place of the run is kept with the result and handed to the respawn |
| the hit's effect: the monster adds its slowness or poison to the player it just killed, and a respawn does not clear effects | the melee hit adds knock-back and effect only when the hit landed and the target is alive (`TestHitThatKillsLeavesNoEffect`; without the guard: `the dead hunter has 1 effects`) |
| a client that stays on the death screen | nothing to build: the run is over and the rewards are in at the death; the watchdog leaves a dead player alone; the respawn button brings the hunter back |
| the rule left on | taken back at every end (above) |
| keep-inventory rests on one line of the platform | pinned by a test of the platform |

**A disconnect between death and respawn.** Dragonfly respawns a dead player before it lets it quit (`Player.close`): the respawn handler runs (`Respawned`: the return place, the waiting result forgotten), then the quit (`Leave`: no run, nothing to end). The player is saved on the island with what it carried. `TestDeadHunterWhoLeavesIsBackOnTheIsland` does the two calls in that order and then joins again: nothing changes.

**How it meets E4's stranded player (T141).** A stranded player is one whose stash is still in the save: the server stopped under a LIVING hunter. A hunter that died has no stash any more (the death gave the carried items back and took the stash out of the save), so whether it respawned, quit on the death screen or the server stopped right then, the next join finds nothing to restore, and `dimension.joined` does nothing for it. If the server stops in the 1.1 seconds between the death and Dragonfly moving the dead player's saved position, the saved position is in the second dimension and the session's `Relocate` sends the player to the island on the next join, as for the stranded player.

**The felling blast's knock-back (the owner's second answer).** Under (ii) the defect is gone without a change to the blast: the blast's `Hurt` kills the hunter, and Dragonfly gives a dead player no knock-back (`Player.KnockBack`). `TestBlastThatKillsDoesNotThrowTheHunter` asserts it; before T145 it fails with `the blast that felled the hunter threw it: velocity [-0.56 0.4 -0.56]` (the hunter was already back on the island, alive). The fix in "the same form" (knock back only when the hit landed) is therefore not built. What is left of it is not the reported defect: the blast still throws a hunter it did not hurt because of the half second of immunity after an earlier hit, where the melee hit does not. A hunter choosing an upgrade cannot be blasted at all (the monsters do not tick while the run is paused). Left as it is and listed as a decision.

### 3.6 The block clean-up (T147 to T150)

| Subscriber | Moves to | Hand break | Other removals |
|---|---|---|---|
| `conduit.break` | nothing: deleted. `Bundle.BlockChanged` already removes the display for every `SetBlock` that takes the bundle away. | the same | the same (it worked before) |
| `machine.break` | `Machine.BreakInfo().BreakHandler` calls `machine.Break` | the same; the take-down runs after the block is removed instead of just before, in the same transaction | NEW: an explosion takes the machine down too (C20) |
| `storage.break` | a drawer: `Drawer.BreakInfo().BreakHandler` calls `RemoveDisplay`; a set machine: `machine.Behaviour.Broken`, a new field the kind's owner registers, called by `machine.Break` | the same | NEW: C21 |
| `netstore.break` | the node kinds' `machine.Behaviour.Broken`: `Invalidate` and `Forget`, directly (the block is gone when it is called; the `Tx.Defer` goes) | the same | NEW: C22 |

`BreakHandler` rather than `BlockChanged` for the machines and the drawer: a machine and a drawer set their own block again whenever their look or count changes, so `BlockChanged` would run on every such `SetBlock` and would have to tell "the same machine with another look" from "gone"; `BreakHandler` is called exactly when Dragonfly broke the block. It does not cover a block replaced by a plain `SetBlock` (the wand's undo); `netstore.place` therefore still forgets a setting left at the position.

No allocation is added to `BreakInfo`, which the look panel and the break timer call often: the machine's handler is a function made once per machine state (`State.broken`, set in `New`), the drawer's handler captures nothing. Measured: `BenchmarkLookMachine` 47 allocs/op before and after, `BenchmarkLookInfoMachine` 23, `BenchmarkBlockBreak` 1, `BenchmarkMachineTick` 8.

**The reason T148 to T150 wait behind a decision.** For a break by hand nothing a player sees changes. But TNT can be had in this game (the magic sieve drops it, `sievetable/magicsieve_tables.go`), and an explosion removes machines and drawers (blast resistance 6 and 2.5). Today an exploded drawer leaves its floating display and label behind, an exploded machine stays in the monitor's list and keeps its windows open, an exploded network node keeps its setting. After T148 to T150 they are cleaned up. That is a change a player can see (C20 to C22), each proven by a test that fails before. The coordinator's instruction for a move that changes what a player sees is "do not plan it: report it"; it is reported (section 9, decision 1), and plan E is written so that it can be dispatched the moment the owner says yes. T147 changes nothing and may run at once.

### 3.7 Why T144 and T145 stand after the kernel switch

The switch comes after every behaviour change the new model FORCES (REF1 section 3.3): those are C1 to C7, committed. Neither T144 nor T145 is forced by the model: both can be written on either kernel.

- T145 removes a cancel (`dimension.fallen`), and with it the forced choice of C6. It is also the one task whose point ("at once", no death screen) only a real client can show. After the switch it is the last behaviour commit of the chain and can be held back or reverted alone if the client check fails; before the switch, the kernel commit would sit on top of it. The cost of this order is three lines: T143 converts `dimension.fallen` to `On` and T145 deletes it.
- T144 takes four registrations of `Use` away, one of them a `Handle` the switch converts (`dimension.stone`). Before the switch it would move T143's hunks in the `subscribe.go` of `dimension`, `copier`, `wand` and `tinker` for no gain.
- With both after it, T143 is, file by file, the patch that was proven for plan B (section "REF T143"), and its proof (the dump equal before and after) is not touched by them.

## 4. Same-tier groups after the plan

REF1 section 6 with the changes of this plan. Every group in which one subscriber cancels:

- `UseOnBlock` LOWEST `dimension.hunting`, `dimension.stone`, `expand.guard`, `screens.menuItem`: unchanged.
- `UseOnBlock` LOW `island.guard`, `copier.use`, `wand.use`: `!CanBuild` / `CanBuild` and the held item. (`fairy.time` is not here.)
- `UseOnBlock` NORMAL `conduit.plate`, `storage.sneakUse`, `fairy.time`, `handwork.stick`, `handwork.gather` (never cancels): standing on a plate / sneaking on a drawer / fairy bone meal on a crop, a sapling or a magic plant, plain bone meal sneaking on a crop / a stick on grass or dirt, sneaking on a chest / an axe on a log, a hoe on dirt. A plate is on none of the blocks `fairy.time` and `handwork.stick` act on; a drawer is none of them.
- `UseOnBlock` HIGH, HIGHEST, MONITOR: unchanged (`machine.noteFace` is gone from the hook altogether).
- `Use` after T144: LOWEST `dimension.hunting`, `screens.menuItem` (a hunter without the menu item / the menu item); HIGH `autofill.use`.
- `Hurt` after T145: LOWEST `dimension.frozen` is the only canceller; MONITOR `achievement.hurt`.
- `BlockBreak` NORMAL after plan E: `tinker.break`, `farm.break`. Neither cancels.

## 5. BEHAVIOUR CHANGES

What a player can see or an achievement counts. Numbered on from REF1 section 7. Each has its own test that fails before the change for the reason given, except C14 and the client half of C15, which only a real client can show.

| Id | Situation | Before | After | Task | Alternative |
|---|---|---|---|---|---|
| C10 | a fairy hourglass is used on the face of a machine that looks out of the island (a machine in the island's last row), standing or sneaking | the hourglass winds the machine and is spent | the side guard refuses the click with the "outside the island" tip; nothing is wound or spent. Opening that machine on that face is refused the same way today | T140 | provisionally accepted by the owner. To remove it: an exception in `island.guardSide` for a held hourglass on a machine (not built, section 3.1). Test: `TestHourglassOnTheOutwardFaceIsRefused`, before `wound by 12000 ticks with 4 hourglasses left, want 0 and 5` |
| C11 | a fairy hourglass winds a machine | the players around see nothing (the click was cancelled by `fairy.time`; Dragonfly swings no arm for a cancelled click) | they see the player's arm swing, as for a tier kit or an opened window (Dragonfly swings the arm when `Activate` or `UseOnBlock` took the click) | T140 | none without a subscriber. Test: `TestHourglassWindShowsTheArmSwing`, before `the players around saw no arm swing` |
| C12 | the server stops while a player is in the second dimension; the save was never paid its fairy dust; the player joins again | the dust is given and lost with the kit | the player has the carried items back and keeps the dust | T141 | leave `dimension.joined` in HIGH (REF1 E4). Test: `TestStrandedPlayerKeepsBackfilledDust`, before `0 fairy dust in the inventory after the join, want 6` |
| C13 | a player clicks with a miner or a network attachment in hand and nothing is placed (a connector plate, a crop, the side guard or a full composter takes or refuses the click, or the clicked block opens), and the next machine is placed without a click of its own (the construction wand) | it faces by the earlier click | it faces its default. Replaces C9: T137 would have covered the cancelled clicks only | T142 | T137 as planned (section 3.2). Test: `TestMachinePlacedWithoutAClickFacesItsDefault`, before `faces west, want down` and `faces east, want down` |
| C14 | a player holds the copier, a construction wand or a hammer | the client is not told the item is usable; it may never send a click on the air for it (then clearing the copier, the wand's settings and the hammer's switch never worked on a real client) | the client entry of the seven items has `minecraft:use_modifiers` (use duration 0.05 s, movement modifier 1), as the teleport stone and the fairy board have. On the server every use does what it did | T144 | UNVERIFIED on a client: whether the click now arrives, a use animation, a slowdown while the button is held (real-client checks). Alternative: keep the subscribers (then nothing is sent differently). Tests of the server half: `TestAirItemsAreUsable` (before `sgitem.ConfigCopier is not usable on the air` and four more), the wire goldens line by line |
| C15 | a hunter takes a hit it does not survive | the hit is cancelled: no flash, no sound; the hunter is on the island at once, healed, the result up | the hit lands and the hunter dies; its client is told at the start of the run to respawn without the death screen; on the respawn the hunter stands where it went in from, healed, and the result shows half a second later. The run ends at the death, once | T145 | the owner's choice of design (ii) over (i) (today) and (iii) (the hit lands without killing). The client half is UNVERIFIED (real-client checks). Tests: `TestHunterDiesAndComesBack`, before `the hit that kills the hunter did not land`; `TestDeadHunterWhoLeavesIsBackOnTheIsland`, before `the hunter did not die` |
| C16 | the same hit | `hurt` does not count (C6, T133) | `hurt` counts: the hit landed. Replaces C6; the void half of T133 (C5) stands | T145 | none: nothing is cancelled. Asserted in `TestHunterDiesAndComesBack` (`hurt, death, wave_fail` = 2, 1, 1 after one light and one killing hit) |
| C17 | a run lost by a death | `death` does not count; `wave_fail` +1 | `death` +1 and `wave_fail` +1 (the `death` achievements have tiers at 1, 10, 50 and 200) | T145 | not counted: two lines in `achievement/subscribe.go`, `if core.InArena(e.P.Position()) { return }` at the top of `achievement.death` (the dead hunter is still in the second dimension when the death is counted). Not built |
| C18 | the same death | the returned hunter keeps the effects it had, except slowness and speed | every effect is gone (a death clears them; a potion drunk before the run is lost with a lost run). The killing hit of a slowing or poisoning monster leaves no effect | T145 | none with a real death. Test: `TestHitThatKillsLeavesNoEffect`, before `the monster did not kill the hunter`; without the guard `the dead hunter has 1 effects` |
| C19 | the hit that fells the hunter is a creeper's blast | the hunter, already back on the island, is thrown by the blast (the defect of the review) | not thrown: a dead player takes no knock-back | T145 | none needed. Test: `TestBlastThatKillsDoesNotThrowTheHunter`, before `the blast that felled the hunter threw it` |
| C20 | TNT blows up a machine | the monitor lists it until it goes stale (10 s); its windows stay open | the machine is taken down at once, as when a player breaks it | T148 | leave `machine.break`. Test: `TestBrokenMachineIsTakenDown`, before `blown up: the monitor still lists the machine`. The windows cannot be seen by a test (the test players have no connection); they close in the same call |
| C21 | TNT blows up a drawer, or a machine of a drawer set | the drawer's floating display and label stay; the set's remembered network lists are dropped only when their tick (50 ms) is over | the display and the label go; the lists are dropped at once | T149 | leave `storage.break`. Tests: `TestBrokenDrawerLosesItsDisplay`, before `blown up: the drawer's display is still there`; `TestBrokenTrimCutsTheDrawerSet`, before `the controller still reaches 1 drawers` |
| C22 | TNT blows up a storage network node | the node's setting stays in the save until a node is placed there again; its network is not dropped | the network is dropped and the setting forgotten at once | T150 | leave `netstore.break`. Test: `TestBrokenNodeLeavesItsNetwork`, before `the cable left behind is not a network of its own` and `the broken node's setting is still there` |

Of REF1's changes: C1 to C5 and C7 stand as committed. **C6 is replaced by C16** (T145 takes T133's change to `dimension.fallen` away with the subscriber; T133's change to `island.void` stays). **C8 is gone** (the plate keeps the click of a standing player; the hourglass does not win on the plate). **C9 is replaced by C13** (the same outcome for the cancelled clicks, and also for a click a block's `Activate` takes). T147 changes nothing.

## 6. Goldens, and the result parts

How a task re-records and checks is REF1 section 13 ("How a task re-records and proves it"): the same commands and the same check script, with the task's scratch directory. Goldens frozen in every task: `wire/testdata` except the 28 lines of T144, the recipe and creative goldens, the save fixture, `kinds.golden`, `look.golden`, `creative.golden`, `settings.golden`, `popup*.golden`.

| Task | `order.golden` | Scenario goldens (check script) | Other goldens |
|---|---|---|---|
| T140 | none | none: 0 traces changed in all four | none |
| T141 | `Join HIGH dimension.joined` goes, `Join LOW dimension.joined` comes after `Join LOWEST screens.menuItem` | `lifecycle: 49 scenarios, 5 traces changed`, no `RESULT` | none |
| T142 | `UseOnBlock NORMAL machine.noteFace` goes | `handler-use: 123 scenarios, 53 traces changed` and two `RESULT` lines | none |
| T143 | re-recorded whole: sorted by name inside a tier, ` handleCancelled` on `look.update`; 122 lines (REF T143) | none: 0 traces changed in all four | none |
| T144 | `Use LOWEST dimension.stone`, `Use NORMAL copier.clear`, `Use NORMAL tinker.veinToggle`, `Use NORMAL wand.options` go (118 lines) | `handler-use: 123 scenarios, 27 traces changed` and three `RESULT` lines | `wire/testdata/items.golden`, `items.byname.golden`: 14 lines each (REF T144) |
| T145 | `Hurt HIGHEST dimension.fallen` goes (117 lines) | `handler-simple: 29 scenarios, 2 traces changed` and one `RESULT` line | none |
| T147, T148, T149 | one `BlockBreak NORMAL` line goes each | `handler-blocks: 33 scenarios, 14 traces changed`, no `RESULT` | none |
| T150 | `BlockBreak NORMAL netstore.break` goes (113 lines) | `handler-blocks: 33 scenarios, 14 traces changed` and one `RESULT` line | none |

**The rule on result parts has seven named exceptions.** REF1 kept every result part byte for byte. A scenario delivers its event to the handler; when the work a result part read moves out of the handler's subscribers, or a cancel goes, the line says so. The effect itself is asserted by the task's test through Dragonfly. These are all of them; any other `RESULT` line is a defect:

```
RESULT handler-use: UseOnBlock/heldMiner          (T142)
  was: cancelled=false screens=- placedFacing=west
  now: cancelled=false screens=- placedFacing=down
RESULT handler-use: UseOnBlock/heldNetCable       (T142)
  was: cancelled=false screens=- placedFacing=west
  now: cancelled=false screens=- placedFacing=down
RESULT handler-use: Use/stone                     (T144)
  was: cancelled=true screens=dimension.ShowWindow held=skygear:dimension_stone:0x1
  now: cancelled=false screens=dimension.ShowWindow held=skygear:dimension_stone:0x1
RESULT handler-use: Use/stoneAfterWindow          (T144)
  was: cancelled=true screens=dimension.ShowWindow held=skygear:dimension_stone:0x1
  now: cancelled=false screens=dimension.ShowWindow held=skygear:dimension_stone:0x1
RESULT handler-use: Use/stoneRepeat               (T144)
  was: cancelled=true screens=- held=skygear:dimension_stone:0x1
  now: cancelled=false screens=- held=skygear:dimension_stone:0x1
RESULT handler-simple: Hurt/fallen                (T145)
  was: cancelled=true
  now: cancelled=false
RESULT handler-blocks: BlockBreak/netNode         (T150)
  was: cancelled=false drops=minecraft:stone:0x1 xp=0 filter=0
  now: cancelled=false drops=minecraft:stone:0x1 xp=0 filter=1
```

- T142: the scenario clicks through the handler and then places a miner through the handler; no subscriber notes a face any more, so the miner faces down. `TestMachineItemPlacesAndFaces` asserts `west` for the real click.
- T144: the stone opens its window in its own `Use` (the scenario goes on to it, section 3.4) and nothing cancels the use.
- T145: the hit that kills is not cancelled. The scenario calls the handler alone, so nobody dies in it; the death is `seconddeath_test.go`'s.
- T150: the scenario breaks a node through the handler; the setting is forgotten by the block's break handler, which the handler alone does not reach. `TestBrokenNodeLeavesItsNetwork` asserts it for a real break.

## 7. The guard and the dump

`TestTierOrder` passes at every task state with no kept pair (`5 orders, 3597 scenarios each`). `SKYGEAR_HOOK_ORDER=reverse` and `shuffle:1` to `shuffle:4` over `./content/skygear/...` (28 packages with tests) pass after T143 and on the final state.

The dump of a state (REF1 section 13: `SKYGEAR_PIN_DUMP`, sorted, 3597 lines: results, counters, inventories) differs from the dump of the state before it in exactly these lines:

| Task | Dump lines that differ |
|---|---|
| T140 | none (equal to 3848f7f) |
| T141 | none |
| T142 | `UseOnBlock/heldMiner`, `UseOnBlock/heldNetCable`: the result part of section 6; counters and inventories equal |
| T143 | none: `cmp` of the sorted dumps before and after the switch is equal |
| T144 | `Use/stone`, `Use/stoneAfterWindow`, `Use/stoneRepeat`: the result part of section 6; counters and inventories equal |
| T145 | `Hurt/fallen`: the result part of section 6, and the counters `wave_fail+1` → `hurt+1` (the handler alone: the hit counts and nobody dies) |
| T146 | (documents) |
| T147, T148, T149 | none |
| T150 | `BlockBreak/netNode`: the result part of section 6; counters and inventories equal |

## 8. Benchmarks

Eight alternating runs of each binary on the same machine (4 cores, shared), median (min-max) ns/op. "3848f7f" is the base, "after T142" every behaviour task before the switch on the OLD kernel, "after T143" the switch.

| Handler benchmark | 3848f7f | after T142 (old kernel) | after T143 | T143 vs T142 | T143 vs 3848f7f | B/op | allocs/op |
|---|---|---|---|---|---|---|---|
| `HandleMove` | 188.6 (168.8-220.5) | 174.8 (166.0-198.3) | 188.1 (178.0-198.5) | +8% | -0% | 24 -> 24 -> 24 | 0 -> 0 -> 0 |
| `Second` | 28431.5 (24471.0-31184.0) | 25953.5 (24843.0-28072.0) | 26951.0 (25139.0-28641.0) | +4% | -5% | 10557 -> 10557 -> 10557 | 63 -> 63 -> 63 |
| `UseOnBlock` | 606.0 (568.2-676.2) | 605.0 (543.7-658.8) | 612.5 (597.2-637.5) | +1% | +1% | 0 -> 0 -> 0 | 0 -> 0 -> 0 |
| `Use` | 144.8 (128.9-172.6) | 147.2 (131.2-163.1) | 154.4 (130.3-178.2) | +5% | +7% | 0 -> 0 -> 0 | 0 -> 0 -> 0 |
| `BlockBreak` | 395.6 (379.3-443.1) | 394.6 (355.6-468.8) | 403.7 (358.6-447.7) | +2% | +2% | 48 -> 48 -> 48 | 1 -> 1 -> 1 |
| `BlockPlace` | 274.9 (239.3-288.3) | 260.5 (239.7-269.1) | 269.9 (259.6-300.8) | +4% | -2% | 48 -> 48 -> 48 | 1 -> 1 -> 1 |

| `Emit` benchmark (before -> after name) | after T142 | after T143 | Median | B/op | allocs/op |
|---|---|---|---|---|---|
| `Emit/handle=false/subs=0 -> hook=plain/subs=0` | 3.0 (2.6-4.0) | 3.0 (2.4-3.5) | +0% (+0.0 ns) | 0 -> 0 | 0 -> 0 |
| `Emit/handle=false/subs=1 -> hook=plain/subs=1` | 5.6 (5.2-6.8) | 5.6 (4.6-6.4) | -1% (-0.1 ns) | 0 -> 0 | 0 -> 0 |
| `Emit/handle=false/subs=4 -> hook=plain/subs=4` | 13.9 (12.0-18.0) | 14.0 (12.5-15.3) | +1% (+0.1 ns) | 0 -> 0 | 0 -> 0 |
| `Emit/handle=false/subs=8 -> hook=plain/subs=8` | 22.7 (19.6-32.9) | 24.1 (22.1-30.3) | +6% (+1.4 ns) | 0 -> 0 | 0 -> 0 |
| `Emit/handle=false/subs=32 -> hook=plain/subs=32` | 85.2 (71.1-96.0) | 89.7 (83.7-96.1) | +5% (+4.5 ns) | 0 -> 0 | 0 -> 0 |
| `Emit/handle=true/subs=0 -> hook=cancel/subs=0` | 3.2 (2.5-4.1) | 6.7 (5.9-7.0) | +107% (+3.4 ns) | 0 -> 0 | 0 -> 0 |
| `Emit/handle=true/subs=1 -> hook=cancel/subs=1` | 6.1 (5.6-7.3) | 8.8 (8.1-12.1) | +44% (+2.7 ns) | 0 -> 0 | 0 -> 0 |
| `Emit/handle=true/subs=4 -> hook=cancel/subs=4` | 14.8 (12.2-17.7) | 16.7 (12.9-20.6) | +13% (+1.9 ns) | 0 -> 0 | 0 -> 0 |
| `Emit/handle=true/subs=8 -> hook=cancel/subs=8` | 25.5 (23.7-28.3) | 30.4 (27.6-37.4) | +20% (+5.0 ns) | 0 -> 0 | 0 -> 0 |
| `Emit/handle=true/subs=32 -> hook=cancel/subs=32` | 88.3 (83.6-103.2) | 104.7 (91.6-112.2) | +18% (+16.3 ns) | 0 -> 0 | 0 -> 0 |
| `EmitMove` | 16.4 (15.1-18.4) | 18.7 (16.9-22.9) | +14% (+2.4 ns) | 0 -> 0 | 0 -> 0 |

Allocations are equal everywhere. The reading is that of REF1 section 11 and the numbers agree with it: the handler benchmarks move inside the machine's spread (15 to 30 percent between two runs of one binary); a `Hook` without a cancel is the old loop; a `CancelHook` pays a fixed 3 ns per `Emit` and a fraction of a nanosecond per subscriber. `UseOnBlock` after T142 is not slower than at 3848f7f although the machine and the hourglass moved (one subscriber less; `fairy.time` has one case less).

The commands, for T143's report (REF1 "REF T138" and plan B's hot path line): `-run xxx -bench 'HandleMove$|Second$|UseOnBlock$|Use$|BlockBreak$|BlockPlace$' -benchmem` on `./content/skygear` and `-bench Emit` on `./content/skygear/core`, each binary built once (`go test -c`), run alternately at least six times.

T144 and T145 take subscribers away from `Use` (four) and `Hurt` (one) and add none. Measured on the final state against the state after T143, three runs each: `Use` 153.7 (136.5-156.9) -> 108.6 (103.5-119.7) ns/op, 0 allocs/op both; `HandleMove`, `UseOnBlock` and `BlockBreak` inside the spread with equal allocations (0, 0, 1). Their reports give `-bench 'Use$|HandleMove$' -benchmem` before and after, for the allocations.

## 9. Real-client checks, decisions for the owner, and what is outside this change

### 9.1 Real-client checks (UNVERIFIED: no test of this repository has a connected client)

T146 writes this list into `docs/TODO.md` under "실기 확인".

The death (T145):

1. With `doimmediaterespawn` sent at the start of the run: no death screen, the client asks to respawn by itself. How long from the hit to standing on the island.
2. Whether a client honours a game rule that arrives in the middle of a session at once.
3. If it does not: the death screen shows on every lost run (the screen the owner removed on 2026-11-01). The run is over and the rewards are in; the respawn button must bring the hunter to where it went in from.
4. The red flash, the hurt sound and the death animation of the last hit; how the change of place across the distance to the island looks.
5. The landing: no fall through chunks the client does not have in the tick before `instance.PlaceAt` holds the player; the result window half a second later.
6. After the run, a death on the island shows the death screen as before (the rule is off again).
7. Disconnecting on the death screen and joining again: on the island, with what was carried.

The items used on the air (T144):

8. Whether a click on the air with the copier, a wand or a hammer now reaches the server (it may not have before).
9. Whether `minecraft:use_modifiers` shows a use animation or slows the player while the button is held, and whether mining with a hammer or building with a wand feels different.
10. The copier is emptied, a sneaking wand shows its settings, a sneaking iron hammer switches diagonal vein mining, once per press (repeats within 0.5 s are ignored).

### 9.2 Decisions

1. **Plan E, T148 to T150 (C20 to C22).** The block clean-up through `BreakHandler` also cleans up after an explosion, which a player can see. Accept, and plan E runs; or leave the three subscribers (then T147 alone runs, and `docs/TODO.md` keeps the item T146 writes).
2. **C17: a lost run counts as a death.** Built as chosen (a real death is a death). The two-line alternative is in the table.
3. **C10 and C11** are built as provisionally accepted (C10) and as found (C11: the others see the arm swing when an hourglass winds a machine). Both follow from "the machine uses the hourglass".
4. **The seven result parts of section 6.** Built: the lines stay and say what the handler alone now does. Alternative for T142 and T150: drop `placedFacing` and `filter` from those scenarios.
5. **The blast and the immunity window** (section 3.5): a blast throws a hunter it did not hurt because of the half second after an earlier hit; a melee hit does not. Left as it is.
6. **Where the air-used item types live** (section 3.4): built with function variables in `sgitem`. Moving the types into their features is the alternative.
7. **The returned menu item keeps its old name** for a stranded player (section 3.3). A name is only old if the player changed language between the two sessions.
8. Open from REF1 section 12, unchanged: the menu item's cancel on the air, a stick on grass or dirt, the form of the `stick_other` record, `elevator.up` in a mode, `handwork.gather` at the island's edge, `expand.guardSide`.

### 9.3 Found on the way, not fixed

- The review counted 15 `BlockBreak/` trace lines for the removal of a break subscriber; it is 14 (17 lines, 3 of them cancelled before NORMAL).
- `docs/TODO.md` (the line on `/sgdebug hunt`), `content/skygear/README.md:49` and `docs/design/03-architecture.md:94` call the second dimension's ground "사냥터". T146 writes 이차원 in everything it adds and leaves these three lines alone.
- `machine.Ticking` lists a machine for ten seconds after it was removed by anything but a break (a plain `SetBlock`): unchanged.
- A test player that dies is closed by Dragonfly 1.1 seconds later (`finishDying`, no connection): a test that kills must finish its checks in the same transaction, as `seconddeath_test.go` does.
- The session hook that blocks bare `go test` on the root package also blocks a package path when the `-run` pattern holds a `|`; build the binary.

## 10. Tasks

Base 3848f7f. One chain; plan C is T140 to T143, plan D is T144 to T146, plan E is T147 to T150.

| Task | What | Goldens | Commit subject |
|---|---|---|---|
| T140 | the hourglass is used by the machine (`Activate`) and by the item (`UseOnBlock`): C10, C11; `orderKept` and its mechanism go | none | `refactor(machine): the fairy hourglass is used through the machine, no subscriber order is kept` |
| T141 | `dimension.joined` in LOW: C12; `coretest.SessionOver` | `order.golden` (1 line moves), traces of 5 lines of `lifecycle.golden` | `fix(dimension): a player the server stopped under is restored before the join steps hand out items` |
| T142 | a machine item places itself and carries its face; `machine.noteFace` goes: C13 | `order.golden` (-1 line), 53 traces and 2 result parts of `handler-use.golden` | `refactor(machine): a machine item carries the face it is placed on` |
| T143 | the kernel switch (REF1 "REF T138") | `order.golden` only | `refactor(core): no subscriber stops the others, a cancelled event reaches only handleCancelled subscribers` |
| T144 | the copier, the wands, the hammers and the teleport stone are used on the air in their own `Use`: C14 | `order.golden` (-4 lines), 27 traces and 3 result parts of `handler-use.golden`, 14 lines each of `items.golden` and `items.byname.golden` | `refactor(sgitem): the items used on the air do their work in their own Use` |
| T145 | a hunter that takes a killing hit dies and respawns at once: C15 to C19; `dimension.fallen` goes | `order.golden` (-1 line), 2 traces and 1 result part of `handler-simple.golden` | `feat(dimension): a hunter that falls dies for real and respawns at once` |
| T146 | documents, with the real-client checks | none | `docs: no propagation stop, hooks and the methods of blocks and items, the death in the second dimension` |
| T147 | `conduit.break` is deleted | `order.golden` (-1 line), 14 traces of `handler-blocks.golden` | `refactor(conduit): a broken bundle is cleaned up by its own BlockChanged alone` |
| T148 | a machine takes itself down in its `BreakInfo`: C20 | the same | `refactor(machine): a broken machine takes itself down, also after an explosion` |
| T149 | a drawer and the set machines clean up in the block: C21; `machine.Behaviour.Broken` | the same | `refactor(storage): a broken drawer and a broken set machine clean up in the block` |
| T150 | a network node cleans up in the block: C22 | the same, and 1 result part | `refactor(netstore): a broken node drops its network and its setting in the block` |

## REF T140

The fairy hourglass is used through the machine; no subscriber order is kept. Prototype patch: `T140.patch`. Replaces T136.

Behaviour changes C10 and C11; everything else as before (the plate takes a standing player's click, the hourglass winds for a sneaking one). No golden changes: no scenario uses an hourglass on a machine.

What changes:

- `machine/machine.go`: `Activate` winds the machine when the held item is a fairy hourglass the machine takes, next to the tier kit; one that winds nothing leaves the click to the machine (the window opens). `Machine.WindHourglass(pos, tx)` implements `sgitem.HourglassWound` for the item.
- `sgitem/fairy.go`: `HourglassWound` and `FairyHourglass.UseOnBlock`, which Dragonfly calls when `Activate` did not take the click (a sneaking player).
- `fairy/fairytime.go`: the hourglass case of `UseFairyTime` goes, with the imports of `machine` and `sound`. `fairy/subscribe.go`: `fairy.time` is unchanged (`Handle`, `core.Normal`); its comment says why it and the plate never meet.
- `conduit/subscribe.go`: the comment of `conduit.plate`.
- `core/coretest/order.go`, `tierorder_test.go`: `orderKept`, the `kept` parameter and the loop that put pairs back go (the same change as in REF1 "REF T136").
- `layers_test.go`: the `siblings` row of `fairy` loses `machine`.

Before the change (3848f7f) the new tests fail like this:

- `TestHourglassOnTheOutwardFaceIsRefused`: `the outward face, sneaking=false: wound by 12000 ticks with 4 hourglasses left, want 0 and 5` (and the same for `sneaking=true`).
- `TestHourglassWindShowsTheArmSwing`: `sneaking=false: the players around saw no arm swing when the hourglass wound the machine` (and `sneaking=true`).
- `machine.TestHourglassIsUsedByTheMachine`: `Activate with an hourglass: wound by 0 ticks, 0 spent, want 12000 and 1` and `the fairy hourglass is not used on a block`.
- `TestHourglassThroughTheMachine` passes in the order of registration (the behaviour is kept). With the pair `UseOnBlock conduit.plate fairy.time` removed from `orderKept` it fails under `SKYGEAR_HOOK_ORDER=reverse`, `shuffle:2`, `shuffle:3` and `shuffle:4` with `the plate, standing: wound by 12000 ticks with 4 hourglasses left, want 0 and 5`: the dependency on the order that the task removes.

After the change, run the root's new tests in every order (a binary, from `content/skygear`):

```sh
for o in "" reverse shuffle:1 shuffle:2 shuffle:3 shuffle:4; do SKYGEAR_HOOK_ORDER=$o $S/root.test -test.count=1 -test.run 'TestHourglass'; done   # PASS six times
```

Goldens: none. `git status --short content/skygear/testdata` is empty and the check script prints 0 traces changed four times. The sorted dump equals that of 3848f7f.

### `content/skygear/conduit/subscribe.go`

~~~~diff
--- a/content/skygear/conduit/subscribe.go
+++ b/content/skygear/conduit/subscribe.go
@@ -49,11 +49,9 @@ func subscribeBlocks() {
 // A player who clicks a conduit's connector plate on the block it touches
 // opens its settings, whatever they hold, unless the held conduit goes into
 // that conduit (a click on the conduit itself is ConduitBundle.Activate).
-// Sneaking places blocks instead.
-//
-// conduit.plate is registered before fairy.time, which is in the same tier,
-// and takes the click first: an hourglass on a plate opens the settings. That
-// order is the exception the tier order guard keeps (the root's orderKept).
+// Sneaking places blocks instead. The plate takes the click before the block
+// behind it is asked (Activate) and before the held item is used: a tier kit
+// or a fairy hourglass on a plate opens the settings.
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
@@ -10,11 +10,9 @@ import (
 	"github.com/df-mc/dragonfly/server/player"
 	"github.com/df-mc/dragonfly/server/world"
 	"github.com/df-mc/dragonfly/server/world/particle"
-	"github.com/df-mc/dragonfly/server/world/sound"
 	"github.com/presentkim/skygear/content/skygear/climate"
 	"github.com/presentkim/skygear/content/skygear/core"
 	"github.com/presentkim/skygear/content/skygear/farm"
-	"github.com/presentkim/skygear/content/skygear/machine"
 	"github.com/presentkim/skygear/content/skygear/sgitem"
 )
 
@@ -48,10 +46,11 @@ func FairyTimeRecipes() {
 	recipe.Register(recipe.NewShaped([]recipe.Item{e, g, e, d, s, d, e, g, e}, item.NewStack(sgitem.FairyHourglass{}, 1), recipe.NewShape(3, 3), "crafting_table"))
 }
 
-// UseFairyTime uses a held fairy bone meal or hourglass, or keeps plain
-// bone meal off a timed crop (a click without sneaking is useCrop's). It
-// reports whether the click was taken: an item that would change nothing
-// is not spent and leaves the click to the block.
+// UseFairyTime uses a held fairy bone meal, or keeps plain bone meal off a
+// timed crop (a click without sneaking is useCrop's). It reports whether the
+// click was taken: an item that would change nothing is not spent and leaves
+// the click to the block. The fairy hourglass is not used here: the machine
+// it winds takes it (machine.Machine.Activate, sgitem.FairyHourglass).
 func UseFairyTime(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, held item.Stack) bool {
 	switch held.Item().(type) {
 	case item.BoneMeal:
@@ -64,14 +63,6 @@ func UseFairyTime(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos,
 		spendHeld(p, held)
 		tx.AddParticle(pos.Vec3(), particle.BoneMeal{})
 		return true
-	case sgitem.FairyHourglass:
-		m, ok := tx.Block(pos).(machine.Machine)
-		if !ok || m.State == nil || !m.State.WindHourglass() {
-			return false
-		}
-		spendHeld(p, held)
-		tx.PlaySound(pos.Vec3Centre(), sound.ItemAdd{})
-		return true
 	}
 	return false
 }
~~~~

### `content/skygear/fairy/subscribe.go`

~~~~diff
--- a/content/skygear/fairy/subscribe.go
+++ b/content/skygear/fairy/subscribe.go
@@ -23,13 +23,12 @@ func subscribeJoin() {
 	})
 }
 
-// The fairy's time is told by the item it is used with; a visiting fairy
-// gives its gift to a player who touches it, whichever way.
-//
-// conduit.plate is in the same tier, is registered before fairy.time and
-// takes a click on a connector plate first, hourglass in hand or not: that
-// order is one of the exceptions the tier order guard keeps (the root's
-// orderKept).
+// Fairy bone meal grows the plant it is used on, and plain bone meal is kept
+// off a timed crop: fairy.time acts before the crop's own click (farm.crop,
+// High) and before the block is asked. It acts on crops, saplings and magic
+// plants only, none of which carries a connector plate, so it and
+// conduit.plate never act on the same click. A visiting fairy gives its gift
+// to a player who touches it, whichever way.
 func subscribeTouch() {
 	core.UseOnBlock.Handle(core.Normal, "fairy.time", func(e core.UseOnBlockEvent) bool {
 		if UseFairyTime(e.S, e.P, e.Tx, e.Pos, e.Held) {
~~~~

### `content/skygear/hourglass_test.go` (new file)

~~~~go
package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
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

// hourglassClick sends one real click of p on the block at pos with n fairy
// hourglasses in hand (Player.UseItemOnBlock: the handler's subscribers, then
// the block's Activate unless p sneaks, then the held item's UseOnBlock) and
// returns how many are left.
func hourglassClick(p *player.Player, pos cube.Pos, face cube.Face, click mgl64.Vec3, sneak bool, n int) int {
	p.SetHeldItems(item.NewStack(sgitem.FairyHourglass{}, n), item.Stack{})
	if sneak {
		p.StartSneaking()
	} else {
		p.StopSneaking()
	}
	p.UseItemOnBlock(pos, face, click)
	held, _ := p.HeldItems()
	return held.Count()
}

// TestHourglassThroughTheMachine: a fairy hourglass winds the machine it is
// used on, standing (the machine's Activate takes it) or sneaking (the
// hourglass's own UseOnBlock). A connector plate on the clicked face takes a
// standing player's click before the machine is asked, and leaves a sneaking
// player's click alone. No subscriber of the click takes part in the winding,
// so the result is the same in every order of the subscribers of one tier.
func TestHourglassThroughTheMachine(t *testing.T) {
	s := testSession(t)
	island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
	loadAround(s)
	h := spawnPlayer(s)
	pos := cube.Pos{4, island.IslandY + 2, 4}
	gen := cube.Pos{4, island.IslandY + 2, 8}
	onPlate, offPlate, top := mgl64.Vec3{1, 0.5, 0.5}, mgl64.Vec3{1, 0.95, 0.95}, mgl64.Vec3{0.5, 1, 0.5}
	st := placeTestMachine(s, pos, machine.KindCrusher)
	genState := placeTestMachine(s, gen, machine.KindGenerator)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		b := conduit.Bundle{Item: 1}
		b.Modes[cube.FaceWest] = itemnet.Input
		at := pos.Side(cube.FaceEast)
		tx.SetBlock(at, b, nil)
		b.Refresh(at, tx)
		if _, _, _, ok := conduit.PlateHit(tx, pos, cube.FaceEast, onPlate); !ok {
			t.Fatal("the click is not on a connector plate")
		}
		if _, _, _, ok := conduit.PlateHit(tx, pos, cube.FaceEast, offPlate); ok {
			t.Fatal("the click beside the plate is on it")
		}
		p.Handle(s.Handler(p))
		p.Teleport(pos.Vec3Centre().Add(mgl64.Vec3{2, 1, 0}))

		for _, c := range []struct {
			name  string
			sneak bool
			face  cube.Face
			click mgl64.Vec3
			wound bool
		}{
			{"the plate, standing", false, cube.FaceEast, onPlate, false},
			{"the plate, sneaking", true, cube.FaceEast, onPlate, true},
			{"beside the plate, standing", false, cube.FaceEast, offPlate, true},
			{"beside the plate, sneaking", true, cube.FaceEast, offPlate, true},
			{"the top, standing", false, cube.FaceUp, top, true},
			{"the top, sneaking", true, cube.FaceUp, top, true},
		} {
			st.Boost = 0
			left := hourglassClick(p, pos, c.face, c.click, c.sneak, 5)
			wantBoost, wantLeft := 0, 5
			if c.wound {
				wantBoost, wantLeft = machine.HourglassTicks, 4
			}
			if st.Boost != wantBoost || left != wantLeft {
				t.Errorf("%s: wound by %d ticks with %d hourglasses left, want %d and %d", c.name, st.Boost, left, wantBoost, wantLeft)
			}
		}

		// A machine that takes no more keeps its click: nothing is spent.
		for _, sneak := range []bool{false, true} {
			st.Boost = machine.HourglassMaxTicks
			if left := hourglassClick(p, pos, cube.FaceUp, top, sneak, 5); left != 5 || st.Boost != machine.HourglassMaxTicks {
				t.Errorf("a wound-up machine, sneaking=%v: %d ticks, %d hourglasses left, want %d and 5", sneak, st.Boost, left, machine.HourglassMaxTicks)
			}
		}
		// A generator is not wound.
		p.Teleport(gen.Vec3Centre().Add(mgl64.Vec3{2, 1, 0}))
		for _, sneak := range []bool{false, true} {
			if left := hourglassClick(p, gen, cube.FaceUp, top, sneak, 5); left != 5 || genState.Boost != 0 {
				t.Errorf("a generator, sneaking=%v: %d ticks, %d hourglasses left, want 0 and 5", sneak, genState.Boost, left)
			}
		}
		// In creative the hourglass winds and is not spent.
		p.Teleport(pos.Vec3Centre().Add(mgl64.Vec3{2, 1, 0}))
		p.SetGameMode(world.GameModeCreative)
		for _, sneak := range []bool{false, true} {
			st.Boost = 0
			if left := hourglassClick(p, pos, cube.FaceUp, top, sneak, 5); left != 5 || st.Boost != machine.HourglassTicks {
				t.Errorf("creative, sneaking=%v: %d ticks, %d hourglasses left, want %d and 5", sneak, st.Boost, left, machine.HourglassTicks)
			}
		}
	})
}

// TestHourglassOnTheOutwardFaceIsRefused: on the face of a machine that looks
// out of the island (a machine in the island's last row) the side guard
// refuses the click, as it refuses opening that machine there: the hourglass
// is used by the machine and by the item, which Dragonfly asks only when no
// subscriber refused the click. The inward faces of the same machine wind it.
func TestHourglassOnTheOutwardFaceIsRefused(t *testing.T) {
	s := testSession(t)
	island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
	loadAround(s)
	h := spawnPlayer(s)
	edge := cube.Pos{15, island.IslandY + 2, 4}
	st := placeTestMachine(s, edge, machine.KindCrusher)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		if !island.CanBuild(s.Session, edge) || island.CanBuild(s.Session, edge.Side(cube.FaceEast)) {
			t.Fatal("the machine is not in the island's last row")
		}
		p.Handle(s.Handler(p))
		p.Teleport(edge.Vec3Centre().Add(mgl64.Vec3{-2, 1, 0}))
		for _, sneak := range []bool{false, true} {
			st.Boost = 0
			if left := hourglassClick(p, edge, cube.FaceEast, mgl64.Vec3{1, 0.5, 0.5}, sneak, 5); left != 5 || st.Boost != 0 {
				t.Errorf("the outward face, sneaking=%v: wound by %d ticks with %d hourglasses left, want 0 and 5", sneak, st.Boost, left)
			}
			st.Boost = 0
			if left := hourglassClick(p, edge, cube.FaceWest, mgl64.Vec3{0, 0.5, 0.5}, sneak, 5); left != 4 || st.Boost != machine.HourglassTicks {
				t.Errorf("an inward face, sneaking=%v: wound by %d ticks with %d hourglasses left, want %d and 4", sneak, st.Boost, left, machine.HourglassTicks)
			}
		}
	})
}

// armSwings counts the arm swings a viewer of the world is shown.
type armSwings struct {
	world.NopViewer
	n int
}

func (v *armSwings) ViewEntityAction(_ world.Entity, a world.EntityAction) {
	if _, ok := a.(entity.SwingArmAction); ok {
		v.n++
	}
}

// TestHourglassWindShowsTheArmSwing: winding a machine is a click the machine
// (or the hourglass) took, and Dragonfly shows the arm swing of such a click
// to the players around, as it does for a tier kit. A click a subscriber
// cancels is not shown: the plate's.
func TestHourglassWindShowsTheArmSwing(t *testing.T) {
	s := testSession(t)
	island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
	loadAround(s)
	h := spawnPlayer(s)
	pos := cube.Pos{4, island.IslandY + 2, 4}
	top := mgl64.Vec3{0.5, 1, 0.5}
	st := placeTestMachine(s, pos, machine.KindCrusher)
	seen := &armSwings{}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		l := world.NewLoader(2, s.World, seen)
		l.Move(tx, pos.Vec3Centre())
		l.Load(tx, 50)
		p.Handle(s.Handler(p))
		p.Teleport(pos.Vec3Centre().Add(mgl64.Vec3{2, 1, 0}))
		for _, sneak := range []bool{false, true} {
			st.Boost, seen.n = 0, 0
			if left := hourglassClick(p, pos, cube.FaceUp, top, sneak, 5); left != 4 {
				t.Fatalf("sneaking=%v: the hourglass was not used", sneak)
			}
			if seen.n == 0 {
				t.Errorf("sneaking=%v: the players around saw no arm swing when the hourglass wound the machine", sneak)
			}
		}
	})
}
~~~~

### `content/skygear/layers_test.go`

~~~~diff
--- a/content/skygear/layers_test.go
+++ b/content/skygear/layers_test.go
@@ -93,7 +93,7 @@ var siblings = map[string][]string{
 	"conduit":   {"machine"},
 	"storage":   {"machine"},
 	"farm":      {"machine"},
-	"fairy":     {"farm", "machine"},
+	"fairy":     {"farm"},
 	"dimension": {"machine"},
 	"copier":    {"conduit", "machine", "storage"},
 	"netstore":  {"conduit", "machine", "storage"},
~~~~

### `content/skygear/machine/hourglass_test.go` (new file)

~~~~go
package machine_test

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/core/coretest"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/machine/machinetest"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// TestHourglassIsUsedByTheMachine: the machine's Activate winds it with the
// hourglass in hand and asks for one to be spent, and the hourglass's
// UseOnBlock does the same for the click Activate is not handed (a sneaking
// player's). One the machine does not take is not spent, and the hourglass
// used on another block does nothing.
func TestHourglassIsUsedByTheMachine(t *testing.T) {
	s := coretest.Session(t)
	island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
	coretest.LoadAround(s)
	st := machinetest.Place(s, machinetest.Pos, machine.KindCrusher)
	h := coretest.SpawnPlayer(s)
	coretest.WithPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		pos := machinetest.Pos
		m := tx.Block(pos).(machine.Machine)
		p.SetHeldItems(item.NewStack(sgitem.FairyHourglass{}, 2), item.Stack{})

		ctx := &item.UseContext{}
		if !m.Activate(pos, cube.FaceUp, tx, p, ctx) || st.Boost != machine.HourglassTicks || ctx.CountSub != 1 {
			t.Errorf("Activate with an hourglass: wound by %d ticks, %d spent, want %d and 1", st.Boost, ctx.CountSub, machine.HourglassTicks)
		}

		use, ok := world.Item(sgitem.FairyHourglass{}).(item.UsableOnBlock)
		if !ok {
			t.Fatal("the fairy hourglass is not used on a block")
		}
		st.Boost = 0
		ctx = &item.UseContext{}
		if !use.UseOnBlock(pos, cube.FaceUp, mgl64.Vec3{}, tx, p, ctx) || st.Boost != machine.HourglassTicks || ctx.CountSub != 1 {
			t.Errorf("UseOnBlock on a machine: wound by %d ticks, %d spent, want %d and 1", st.Boost, ctx.CountSub, machine.HourglassTicks)
		}

		st.Boost = machine.HourglassMaxTicks
		ctx = &item.UseContext{}
		if use.UseOnBlock(pos, cube.FaceUp, mgl64.Vec3{}, tx, p, ctx) || ctx.CountSub != 0 || st.Boost != machine.HourglassMaxTicks {
			t.Errorf("UseOnBlock on a wound-up machine: %d ticks, %d spent", st.Boost, ctx.CountSub)
		}
		ctx = &item.UseContext{}
		if use.UseOnBlock(pos.Side(cube.FaceDown), cube.FaceUp, mgl64.Vec3{}, tx, p, ctx) || ctx.CountSub != 0 {
			t.Error("UseOnBlock on a block that is no machine took the hourglass")
		}
	})
}
~~~~

### `content/skygear/machine/machine.go`

~~~~diff
--- a/content/skygear/machine/machine.go
+++ b/content/skygear/machine/machine.go
@@ -16,6 +16,7 @@ import (
 	"github.com/df-mc/dragonfly/server/player"
 	"github.com/df-mc/dragonfly/server/world"
 	"github.com/df-mc/dragonfly/server/world/particle"
+	"github.com/df-mc/dragonfly/server/world/sound"
 	"github.com/go-gl/mathgl/mgl64"
 	"github.com/presentkim/skygear/common/chestui"
 	"github.com/presentkim/skygear/common/fluidnet"
@@ -1465,7 +1466,8 @@ func (m Machine) DecodeNBT(data map[string]any) any {
 	return n
 }
 
-// Activate opens the machine window, or uses a sieve.
+// Activate opens the machine window, or uses a sieve. A held tier kit or
+// fairy hourglass is used on the machine instead.
 func (m Machine) Activate(pos cube.Pos, _ cube.Face, tx *world.Tx, u item.User, ctx *item.UseContext) bool {
 	p, ok := u.(*player.Player)
 	s, ok2 := core.Of(tx)
@@ -1478,6 +1480,13 @@ func (m Machine) Activate(pos cube.Pos, _ cube.Face, tx *world.Tx, u item.User,
 			UseKit(s, p, tx, pos, m, kit, held, off)
 			return true
 		}
+		// A fairy hourglass winds the machine; one the machine does not take
+		// leaves the click to the machine.
+		if _, ok := held.Item().(sgitem.FairyHourglass); ok && m.State.WindHourglass() {
+			ctx.SubtractFromCount(1)
+			tx.PlaySound(pos.Vec3Centre(), sound.ItemAdd{})
+			return true
+		}
 	}
 	if IsSieve(m.Kind) {
 		useSieve(s, p, tx, pos, m.State)
@@ -1872,3 +1881,11 @@ func Break(s *core.Session, tx *world.Tx, pos cube.Pos, m Machine) {
 }
 
 func Item(kind string) item.Stack { return item.NewStack(Machine{Kind: kind}, 1) }
+
+// WindHourglass implements sgitem.HourglassWound: the hourglass of a sneaking
+// player, whose click Dragonfly does not hand to Activate, winds the machine
+// through the item.
+func (m Machine) WindHourglass(pos cube.Pos, tx *world.Tx) bool {
+	s, ok := core.Of(tx)
+	return ok && island.CanBuild(s, pos) && m.State != nil && m.State.WindHourglass()
+}
~~~~

### `content/skygear/sgitem/fairy.go`

~~~~diff
--- a/content/skygear/sgitem/fairy.go
+++ b/content/skygear/sgitem/fairy.go
@@ -1,7 +1,12 @@
 package sgitem
 
 import (
+	"github.com/df-mc/dragonfly/server/block/cube"
+	"github.com/df-mc/dragonfly/server/item"
 	"github.com/df-mc/dragonfly/server/item/category"
+	"github.com/df-mc/dragonfly/server/world"
+	"github.com/df-mc/dragonfly/server/world/sound"
+	"github.com/go-gl/mathgl/mgl64"
 	"github.com/presentkim/skygear/common/i18n"
 )
 
@@ -33,7 +38,9 @@ func (FairyBoneMeal) Name() string { return i18n.T(i18n.Default, "name.FairyBone
 
 func (FairyBoneMeal) Category() category.Category { return category.Items() }
 
-// FairyHourglass runs an energy machine faster for a while (fairytime.go).
+// FairyHourglass runs an energy machine faster for a while. It is used on the
+// machine: the machine's Activate takes it from a standing player, UseOnBlock
+// below from a sneaking one.
 type FairyHourglass struct{}
 
 func (FairyHourglass) EncodeItem() (string, int16) { return "skygear:fairy_hourglass", 0 }
@@ -41,3 +48,23 @@ func (FairyHourglass) EncodeItem() (string, int16) { return "skygear:fairy_hourg
 func (FairyHourglass) Name() string { return i18n.T(i18n.Default, "name.FairyHourglass") }
 
 func (FairyHourglass) Category() category.Category { return category.Items() }
+
+// HourglassWound is a block a fairy hourglass winds.
+type HourglassWound interface {
+	// WindHourglass adds the time of one hourglass to the block at pos and
+	// reports whether the block took it.
+	WindHourglass(pos cube.Pos, tx *world.Tx) bool
+}
+
+// UseOnBlock implements item.UsableOnBlock: the hourglass winds the block it
+// is used on. Dragonfly calls it when the block's Activate did not take the
+// click, which is while the player sneaks.
+func (FairyHourglass) UseOnBlock(pos cube.Pos, _ cube.Face, _ mgl64.Vec3, tx *world.Tx, _ item.User, ctx *item.UseContext) bool {
+	w, ok := tx.Block(pos).(HourglassWound)
+	if !ok || !w.WindHourglass(pos, tx) {
+		return false
+	}
+	ctx.SubtractFromCount(1)
+	tx.PlaySound(pos.Vec3Centre(), sound.ItemAdd{})
+	return true
+}
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

## REF T141

A player the server stopped under in the second dimension is restored before the join steps hand out items. Prototype patch: `T141.patch`. E4 of REF1 section 8.

Behaviour change C12. `dimension.joined` moves from HIGH to LOW; its body is unchanged. Section 3.3 has what else moves (nothing a player sees).

`core/coretest/coretest.go`: `SessionOver(t, state, store)` is `Session` with the save's state document and the store function handed to `core.New`; `Session(t)` calls it with nil, nil.

`strandedjoin_test.go` runs two sessions over one state document: in the first a player joins, carries three stone and goes into the second dimension (`dimension.Begin`), and the session is closed with the run still on; the second session reads the document, is three nights old and owed its dust, and the player comes back holding the kit. Before the change it fails with `0 fairy dust in the inventory after the join, want 6` (and `0 fairy dust after joining again, want 6`).

Goldens: re-record `^TestPin(Order|Lifecycle)$`. `order.golden`: `Join HIGH dimension.joined` goes, `Join LOW dimension.joined` comes after `Join LOWEST screens.menuItem`. `lifecycle.golden`: the five lines `Join/again`, `Join/first`, `Join/purge`, `Join/spectator`, `Join/withOther` have `dimension.joined` after `screens.menuItem` instead of after `fairy.backfillDust`; the part after ` | ` of every line is unchanged. The check script prints `lifecycle: 49 scenarios, 5 traces changed` and no `RESULT`.

### `content/skygear/core/coretest/coretest.go`

~~~~diff
--- a/content/skygear/core/coretest/coretest.go
+++ b/content/skygear/core/coretest/coretest.go
@@ -85,6 +85,15 @@ func entityTypes() []world.EntityType {
 // The first call reorders the subscribers when the environment asks for it
 // (ReorderHooks).
 func Session(t testing.TB) core.Lifecycle {
+	t.Helper()
+	return SessionOver(t, nil, nil)
+}
+
+// SessionOver is Session over the save's state document state, as a server
+// that starts again reads it (nil: a new save). store is handed every
+// document the session stores (nil: none is kept); Session.Close stores the
+// last one before it returns.
+func SessionOver(t testing.TB, state []byte, store func([]byte) error) core.Lifecycle {
 	t.Helper()
 	world.DefaultBlockRegistry.Finalize()
 	ReorderHooks()
@@ -93,6 +102,7 @@ func Session(t testing.TB) core.Lifecycle {
 	c := core.New(core.Config{
 		World: w, Clock: instance.NewClock(0, 1, 0, 20*time.Minute),
 		SaveID: 1, SaveName: "test", HostID: "host", Log: slog.Default(),
+		State: state, Store: store,
 	})
 	c.Rand = rand.New(rand.NewPCG(1, 2))
 	t.Cleanup(c.Close)
~~~~

### `content/skygear/dimension/subscribe.go`

~~~~diff
--- a/content/skygear/dimension/subscribe.go
+++ b/content/skygear/dimension/subscribe.go
@@ -71,13 +71,13 @@ func subscribeTicks() {
 
 // The second dimension reacts to a player joining and leaving.
 //
-// dimension.joined is in the High tier, after the join steps of the other
-// features, as it always ran: a hunter the server stopped under gets back
-// what it carried, and what the earlier join steps put into the hunter's
-// inventory meanwhile is gone with the hunting kit. That keeps today's
-// behaviour; it is one of the listed exceptions (docs/tech/extension.md).
+// dimension.joined gives a player the server stopped under in the second
+// dimension back what it carried, and a watcher its game mode. It makes the
+// player ready before the join steps of the other features hand anything
+// out: Low. What they give (the fairy dust the save is owed) goes into the
+// inventory the player keeps.
 func subscribePresence() {
-	core.Join.On(core.High, "dimension.joined", func(e core.Presence) {
+	core.Join.On(core.Low, "dimension.joined", func(e core.Presence) {
 		Joined(e.S, e.P)
 		RetireItems(e.P)
 	})
~~~~

### `content/skygear/strandedjoin_test.go` (new file)

~~~~go
package skygear

import (
	"math/rand/v2"
	"slices"
	"sync"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/core/coretest"
	"github.com/presentkim/skygear/content/skygear/dimension"
	"github.com/presentkim/skygear/content/skygear/fairy"
	"github.com/presentkim/skygear/content/skygear/screens"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"golang.org/x/text/language"
)

// TestStrandedPlayerKeepsBackfilledDust: the server stops while a player is
// in the second dimension, so the save still says the player is there: what
// the player carried is put away in the save and the player holds the kit of
// the run. When the server runs again and the player joins, the player gets
// back what it carried before the join steps of the other features hand
// anything out, so the fairy dust the save is owed stays in the inventory
// the player keeps.
func TestStrandedPlayerKeepsBackfilledDust(t *testing.T) {
	// The first run of the server: a player goes into the second dimension
	// carrying three stone, and the server stops.
	var (
		mu  sync.Mutex
		doc []byte
	)
	first := coretest.SessionOver(t, nil, func(b []byte) error {
		mu.Lock()
		defer mu.Unlock()
		doc = slices.Clone(b)
		return nil
	})
	coretest.LoadAround(first)
	h := coretest.SpawnPlayer(first)
	carried := item.NewStack(block.Stone{}, 3)
	var (
		id     uuid.UUID
		kit    []item.Stack
		armour []item.Stack
	)
	coretest.WithPlayer(t, first, h, func(p *player.Player, tx *world.Tx) {
		first.Join(p, true)
		_ = p.Inventory().SetItem(0, carried)
		if !dimension.Begin(first.Session, tx, p, sgitem.MobSlime) {
			t.Fatal("the run did not begin")
		}
		if n := core.CountPlain(p.Inventory(), block.Stone{}); n != 0 {
			t.Fatalf("the player still carries %d stone in the second dimension", n)
		}
		id, kit, armour = p.UUID(), slices.Clone(p.Inventory().Slots()), slices.Clone(p.Armour().Slots())
	})
	first.Session.Close() // the server stops: the save is stored as it is
	mu.Lock()
	saved := slices.Clone(doc)
	mu.Unlock()
	if len(saved) == 0 {
		t.Fatal("the first run stored no state document")
	}

	// The server runs again, nights later, on a save that never got its dust.
	const nights = 3
	second := coretest.SessionOver(t, saved, nil)
	coretest.LoadAround(second)
	day := second.Clock.DayLength()
	second.Clock.Advance((nights-1)*day + day*fairy.FairyTo/100)
	if n := fairy.PastNights(second.Session); n != nights {
		t.Fatalf("%d nights past, want %d", n, nights)
	}
	const seed1, seed2 = 1, 2
	second.Rand = rand.New(rand.NewPCG(seed1, seed2))
	r := rand.New(rand.NewPCG(seed1, seed2))
	want := 0
	for range nights {
		want += fairy.FairyDustMin + r.IntN(fairy.FairyDustMax-fairy.FairyDustMin+1)
	}
	if want == 0 {
		t.Fatal("the save is owed no dust")
	}

	// The player comes back as the server stored it: holding the kit.
	inv := inventory.New(36, nil)
	for slot, it := range kit {
		_ = inv.SetItem(slot, it)
	}
	arm := inventory.NewArmour(nil)
	arm.Set(armour[0], armour[1], armour[2], armour[3])
	back := world.EntitySpawnOpts{ID: id}.New(player.Type, player.Config{
		Name: "p", UUID: id, Locale: language.Korean, Position: mgl64.Vec3{0.5, 64, 2.5},
		Inventory: inv, Armour: arm,
	})
	second.World.Do(func(tx *world.Tx) { tx.AddEntity(back) })
	coretest.WithPlayer(t, second, back, func(p *player.Player, tx *world.Tx) {
		if dimension.Hunting(second.Session, p) {
			t.Fatal("a run is under way after the server started again")
		}
		second.Join(p, false)
		if got, _ := p.Inventory().Item(0); !got.Equal(carried) {
			t.Errorf("slot 0 holds %v after the join, want the three stone the player carried", got)
		}
		if it, _ := p.Inventory().Item(8); !screens.IsMenuItem(it) {
			t.Errorf("slot 8 holds %v, want the menu item", it)
		}
		for _, it := range p.Armour().Slots() {
			if !it.Empty() {
				t.Errorf("the player still wears %v of the kit", it)
			}
		}
		if got := core.CountPlain(p.Inventory(), sgitem.FairyDust{}); got != want {
			t.Errorf("%d fairy dust in the inventory after the join, want %d", got, want)
		}

		// A second join gives nothing more and takes nothing away.
		second.Join(p, false)
		if got := core.CountPlain(p.Inventory(), sgitem.FairyDust{}); got != want {
			t.Errorf("%d fairy dust after joining again, want %d", got, want)
		}
		if got, _ := p.Inventory().Item(0); !got.Equal(carried) {
			t.Errorf("slot 0 holds %v after joining again", got)
		}
	})
}
~~~~

## REF T142

A machine item carries the face it is placed on. Prototype patch: `T142.patch`. Replaces T137.

Behaviour change C13. Section 3.2 has the design and why the face is not read from the block's `State`.

What changes:

- `machine/machine.go`: the fields `clicked` and `placedOn` of `Machine`; `Machine.UseOnBlock`; `Place` takes the placed block (`b Machine`) instead of the kind and reads the two fields.
- `machine/subscribe.go`: `machine.noteFace` and `subscribeFace` go; `machine.place` hands the block to `Place`.
- `machine/netshape.go`: `placeFaces`, `NotePlaceFace` and `takePlaceFace` go, with the imports of `player` and `core`.
- `pin_use_test.go`: `machine.noteFace` leaves the row of `UseOnBlock/plain`; two comments.

`UseOnBlock` must stay the mirror of the block-item case of `Player.UseItemOnBlock` (`third_party/dragonfly/server/player/player.go`): the clicked block if it is `block.Replaceable` by the machine, else the block beside the face; that block replaceable and not out of bounds; then `PlaceBlock`, which checks the reach, the game mode and entities in the way, raises `HandleBlockPlace`, sets the block and counts the item (`ctx.CountSub`). Dragonfly spends the item from the context (not in creative).

Before the change `machineface_test.go` fails with `a miner placed after a connector plate took a click faces west, want down` and `a miner placed after a click that opened a chest faces east, want down`. `TestMachineItemPlacesAndFaces` passes before and after: the placing a player sees is kept.

Goldens: re-record `^TestPin(Order|Use)$`. `order.golden`: the line `UseOnBlock NORMAL machine.noteFace` goes. The check script prints `handler-use: 123 scenarios, 53 traces changed` and exactly the two `RESULT` lines of section 6 (`UseOnBlock/heldMiner`, `UseOnBlock/heldNetCable`: `placedFacing=west` becomes `placedFacing=down`). Any other `RESULT` line: restore and report.

### `content/skygear/machine/machine.go`

~~~~diff
--- a/content/skygear/machine/machine.go
+++ b/content/skygear/machine/machine.go
@@ -518,6 +518,12 @@ type Machine struct {
 	Tier int
 	// Shape is the look of a thin network block (netshape.go).
 	Shape int
+	// clicked and placedOn are whether a machine item was used on a block to
+	// place this one, and that block's face. The item sets them on the block
+	// it hands to the placement (UseOnBlock) and Place reads them; a machine
+	// in the world does not carry them.
+	clicked  bool
+	placedOn cube.Face
 }
 
 // State is the inventory and progress of a placed machine. It is
@@ -1851,8 +1857,12 @@ func viewCrack(tx *world.Tx, pos cube.Pos, a world.BlockAction) {
 	}
 }
 
-// Place sets up a placed machine with a fresh state.
-func Place(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, kind string) {
+// Place sets up a placed machine with a fresh state. b is the block that was
+// placed. One a machine item placed says which face the item was used on
+// (UseOnBlock): a network attachment or a miner faces that block. A machine
+// placed any other way faces its default.
+func Place(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, b Machine) {
+	kind := b.Kind
 	m := New(kind)
 	account.Grant(p, achievement.FirstMachine)
 	achievement.Count(p, "machine_place", 1)
@@ -1860,9 +1870,8 @@ func Place(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, kind s
 	if m.Kind != KindSieve {
 		progress.Advance(s, tx, advancement.Machine)
 	}
-	face, clicked := takePlaceFace(s, p)
-	if _, head := netHeads[kind]; (head || kind == KindMiner) && clicked {
-		m.State.Facing = face.Opposite()
+	if _, head := netHeads[kind]; (head || kind == KindMiner) && b.clicked {
+		m.State.Facing = b.placedOn.Opposite()
 	}
 	tx.Defer(func(tx *world.Tx) {
 		if NetShaped(kind) {
@@ -1882,6 +1891,27 @@ func Break(s *core.Session, tx *world.Tx, pos cube.Pos, m Machine) {
 
 func Item(kind string) item.Stack { return item.NewStack(Machine{Kind: kind}, 1) }
 
+// UseOnBlock implements item.UsableOnBlock: a machine item places its block
+// where Dragonfly places a block item (in the clicked block if that gives
+// way, else beside the clicked face) and hands the placement the face it was
+// used on (Place).
+func (m Machine) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) bool {
+	at := pos
+	if r, ok := tx.Block(at).(block.Replaceable); !ok || !r.ReplaceableBy(m) {
+		at = pos.Side(face)
+	}
+	if r, ok := tx.Block(at).(block.Replaceable); !ok || !r.ReplaceableBy(m) || at.OutOfBounds(tx.Range()) {
+		return false
+	}
+	placer, ok := user.(block.Placer)
+	if !ok {
+		return false
+	}
+	m.clicked, m.placedOn = true, face
+	placer.PlaceBlock(at, m, ctx)
+	return ctx.CountSub > 0
+}
+
 // WindHourglass implements sgitem.HourglassWound: the hourglass of a sneaking
 // player, whose click Dragonfly does not hand to Activate, winds the machine
 // through the item.
~~~~

### `content/skygear/machine/netshape.go`

~~~~diff
--- a/content/skygear/machine/netshape.go
+++ b/content/skygear/machine/netshape.go
@@ -6,11 +6,9 @@ import (
 
 	"github.com/df-mc/dragonfly/server/block/cube"
 	"github.com/df-mc/dragonfly/server/block/customblock"
-	"github.com/df-mc/dragonfly/server/player"
 	"github.com/df-mc/dragonfly/server/world"
 	"github.com/presentkim/skygear/common/clientdef"
 	"github.com/presentkim/skygear/content/skygear/blockmodel"
-	"github.com/presentkim/skygear/content/skygear/core"
 	"github.com/presentkim/skygear/content/skygear/netshape"
 )
 
@@ -34,22 +32,6 @@ func NetShaped(kind string) bool {
 	return head || kind == KindNetCable
 }
 
-// Faces the network attachments act on: an attachment faces the block
-// clicked to place it (NotePlaceFace); without one it faces south.
-var placeFaces = core.NewPlayers[cube.Face]()
-
-// NotePlaceFace keeps the face of the block p clicked with a machine, which
-// the attachment placed by that click faces.
-func NotePlaceFace(s *core.Session, p *player.Player, face cube.Face) {
-	placeFaces.Of(s)[p.H()] = face
-}
-
-func takePlaceFace(s *core.Session, p *player.Player) (cube.Face, bool) {
-	f, ok := placeFaces.Of(s)[p.H()]
-	delete(placeFaces.Of(s), p.H())
-	return f, ok
-}
-
 // netRaw is a shape as the arm mask, plus 64 times the facing for an
 // attachment.
 func netRaw(kind string, facing cube.Face, mask int) int {
~~~~

### `content/skygear/machine/subscribe.go`

~~~~diff
--- a/content/skygear/machine/subscribe.go
+++ b/content/skygear/machine/subscribe.go
@@ -6,7 +6,6 @@ import "github.com/presentkim/skygear/content/skygear/core"
 func subscribe() {
 	subscribeMonitor()
 	subscribeBlocks()
-	subscribeFace()
 	registerLook()
 }
 
@@ -26,16 +25,7 @@ func subscribeBlocks() {
 	})
 	core.BlockPlace.On(core.Normal, "machine.place", func(e core.BlockPlaceEvent) {
 		if b, ok := e.Block.(Machine); ok {
-			Place(e.S, e.P, e.Tx, e.Pos, b.Kind)
-		}
-	})
-}
-
-// A net-shaped machine or a miner in hand remembers the face it is placed on.
-func subscribeFace() {
-	core.UseOnBlock.On(core.Normal, "machine.noteFace", func(e core.UseOnBlockEvent) {
-		if m, ok := e.Held.Item().(Machine); ok && (NetShaped(m.Kind) || m.Kind == KindMiner) {
-			NotePlaceFace(e.S, e.P, e.Face)
+			Place(e.S, e.P, e.Tx, e.Pos, b)
 		}
 	})
 }
~~~~

### `content/skygear/machineface_test.go` (new file)

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
	"github.com/presentkim/skygear/common/itemnet"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/machine"
)

// machineAt returns the machine at pos, set up (with a state).
func machineAt(t *testing.T, tx *world.Tx, pos cube.Pos) machine.Machine {
	t.Helper()
	m, ok := tx.Block(pos).(machine.Machine)
	if !ok || m.State == nil {
		t.Fatalf("no machine was set up at %v: %T", pos, tx.Block(pos))
	}
	return m
}

// TestMachineItemPlacesAndFaces: a machine item is placed by real clicks
// (Player.UseItemOnBlock: the handler's subscribers, the clicked block's
// Activate, then the held item's UseOnBlock, which places the machine). The
// item hands the placement the face it was used on: a miner and a network
// attachment face the clicked block, another machine does not turn. A click
// places like Dragonfly places a block item: beside the clicked face, one
// item spent, none in creative, nothing where the place is taken or the
// island's guard refuses.
func TestMachineItemPlacesAndFaces(t *testing.T) {
	s := testSession(t)
	island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
	loadAround(s)
	h := spawnPlayer(s)
	base := cube.Pos{4, island.IslandY + 2, 4}
	side, top := mgl64.Vec3{1, 0.5, 0.5}, mgl64.Vec3{0.5, 1, 0.5}
	place := func(p *player.Player, kind string, n int, pos cube.Pos, face cube.Face, click mgl64.Vec3) int {
		p.SetHeldItems(item.NewStack(machine.Machine{Kind: kind}, n), item.Stack{})
		p.UseItemOnBlock(pos, face, click)
		held, _ := p.HeldItems()
		return held.Count()
	}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		p.Handle(s.Handler(p))
		p.Teleport(base.Vec3Centre().Add(mgl64.Vec3{0, 3, 3}))
		for i := range 4 {
			tx.SetBlock(base.Add(cube.Pos{0, 0, i * 2}), block.Stone{}, nil)
		}
		if left := place(p, machine.KindMiner, 3, base, cube.FaceEast, side); left != 2 {
			t.Errorf("%d miners left after placing one, want 2", left)
		}
		if left := place(p, machine.KindNetImporter, 3, base.Add(cube.Pos{0, 0, 2}), cube.FaceUp, top); left != 2 {
			t.Errorf("%d importers left after placing one, want 2", left)
		}
		if left := place(p, machine.KindCrusher, 3, base.Add(cube.Pos{0, 0, 4}), cube.FaceEast, side); left != 2 {
			t.Errorf("%d crushers left after placing one, want 2", left)
		}
		// the place beside the clicked face is taken: nothing is placed
		if left := place(p, machine.KindCrusher, 3, base.Add(cube.Pos{0, 0, 4}), cube.FaceEast, side); left != 3 {
			t.Errorf("%d crushers left after a click on a taken place, want 3", left)
		}
		p.SetGameMode(world.GameModeCreative)
		if left := place(p, machine.KindMiner, 3, base.Add(cube.Pos{0, 0, 6}), cube.FaceNorth, mgl64.Vec3{0.5, 0.5, 0}); left != 3 {
			t.Errorf("%d miners left after placing one in creative, want 3", left)
		}
		p.SetGameMode(world.GameModeSurvival)
	})
	// The placed machines were set up when the placing transaction ended.
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		if m := machineAt(t, tx, base.Side(cube.FaceEast)); m.Kind != machine.KindMiner || m.State.Facing != cube.FaceWest {
			t.Errorf("the miner placed on an east face is a %s facing %v, want a miner facing west", m.Kind, m.State.Facing)
		}
		imp := machineAt(t, tx, base.Add(cube.Pos{0, 1, 2}))
		if imp.Kind != machine.KindNetImporter || imp.State.Facing != cube.FaceDown {
			t.Errorf("the importer placed on a top face is a %s facing %v, want an importer facing down", imp.Kind, imp.State.Facing)
		}
		if m := machineAt(t, tx, base.Add(cube.Pos{1, 0, 4})); m.Kind != machine.KindCrusher || m.State.Facing != cube.FaceSouth {
			t.Errorf("the crusher is a %s facing %v, want a crusher that did not turn (south)", m.Kind, m.State.Facing)
		}
		if m := machineAt(t, tx, base.Add(cube.Pos{0, 0, 5})); m.State.Facing != cube.FaceSouth {
			t.Errorf("the miner placed on a north face in creative faces %v, want south", m.State.Facing)
		}
	})
	// On the last row a click on the outward face is refused: nothing placed.
	edge := cube.Pos{15, island.IslandY + 2, 4}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		tx.SetBlock(edge, block.Stone{}, nil)
		p.Teleport(edge.Vec3Centre().Add(mgl64.Vec3{-2, 1, 0}))
		if left := place(p, machine.KindMiner, 3, edge, cube.FaceEast, side); left != 3 {
			t.Errorf("%d miners left after a click the side guard refuses, want 3", left)
		}
		if _, ok := tx.Block(edge.Side(cube.FaceEast)).(machine.Machine); ok {
			t.Error("a miner was placed outside the island")
		}
	})
}

// TestMachinePlacedWithoutAClickFacesItsDefault: the face a miner is placed
// by belongs to the click that places it. A click with a miner in hand that
// places nothing (a connector plate takes it; a chest opens) leaves nothing
// behind: a miner placed afterwards without a click of its own, as the
// construction wand places (the handler's HandleBlockPlace), faces down.
func TestMachinePlacedWithoutAClickFacesItsDefault(t *testing.T) {
	s := testSession(t)
	island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
	loadAround(s)
	h := spawnPlayer(s)
	chest := cube.Pos{4, island.IslandY + 2, 4}
	at := cube.Pos{8, island.IslandY + 2, 8}
	n := 0
	placed := func() cube.Face {
		n++
		pos := at.Add(cube.Pos{0, 0, n})
		withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
			s.Handler(p).HandleBlockPlace(player.NewEventContext(tx, p), pos, machine.New(machine.KindMiner))
		})
		var facing cube.Face
		withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) { facing = machineAt(t, tx, pos).State.Facing })
		return facing
	}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		p.Handle(s.Handler(p))
		p.Teleport(chest.Vec3Centre().Add(mgl64.Vec3{2, 1, 0}))
		chestWithBundle(tx, chest, itemnet.Input, 5, 9)
		p.SetHeldItems(item.NewStack(machine.Machine{Kind: machine.KindMiner}, 3), item.Stack{})
		// the connector plate on the chest's east face takes the click
		p.UseItemOnBlock(chest, cube.FaceEast, mgl64.Vec3{1, 0.5, 0.5})
		if held, _ := p.HeldItems(); held.Count() != 3 {
			t.Fatalf("%d miners left after a click on a connector plate, want 3", held.Count())
		}
	})
	if got := placed(); got != cube.FaceDown {
		t.Errorf("a miner placed after a connector plate took a click faces %v, want down", got)
	}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		// the chest opens on its west face: the click passes every
		// subscriber and places nothing
		p.SetHeldItems(item.NewStack(machine.Machine{Kind: machine.KindMiner}, 3), item.Stack{})
		p.UseItemOnBlock(chest, cube.FaceWest, mgl64.Vec3{0, 0.5, 0.5})
		if held, _ := p.HeldItems(); held.Count() != 3 {
			t.Fatalf("%d miners left after a click that opened a chest, want 3", held.Count())
		}
	})
	if got := placed(); got != cube.FaceDown {
		t.Errorf("a miner placed after a click that opened a chest faces %v, want down", got)
	}
}
~~~~

### `content/skygear/pin_use_test.go`

~~~~diff
--- a/content/skygear/pin_use_test.go
+++ b/content/skygear/pin_use_test.go
@@ -301,7 +301,10 @@ func TestPinUse(t *testing.T) {
 	onBlock("outside", holding(stone), outsidePos, cube.FaceUp, centre, held)
 	onBlock("arena", holding(stone), core.ArenaCentre, cube.FaceUp, centre, held)
 	onBlock("outsideCopier", holding(copierItem), outsidePos, cube.FaceUp, centre, held)
-	// 6 machine.noteFace: the face a miner or network block is placed by.
+	// 6 a machine in hand: the click's subscribers note no face. The item
+	// hands the face to its own placement (machine.Machine.UseOnBlock,
+	// machineface_test.go), so a miner placed here without a click of its
+	// own faces down whatever was held.
 	placedFacing := func(sc scene) func() string {
 		return func() string {
 			at := cube.Pos{6, island.IslandY + 2, 6}
@@ -333,7 +336,7 @@ func TestPinUse(t *testing.T) {
 		sc := start(holding(c.held))
 		event("UseOnBlock/"+c.name, sc, 0, false, clickAt(pos, c.face, centre), placedFacing(sc))
 	}
-	// The denied clicks note no face.
+	// A denied click is no different.
 	for _, c := range []struct {
 		name  string
 		setup setupFn
@@ -696,7 +699,7 @@ func TestPinUse(t *testing.T) {
 	pinCheckReached(t, names, lines)
 
 	rows := map[string]string{
-		"UseOnBlock/plain": "dimension.hunting dimension.stone expand.guard screens.menuItem island.guard copier.use wand.use machine.noteFace conduit.plate storage.sneakUse fairy.time handwork.stick handwork.gather farm.composter farm.crop island.guardSide expand.guardSide handwork.stickOther",
+		"UseOnBlock/plain": "dimension.hunting dimension.stone expand.guard screens.menuItem island.guard copier.use wand.use conduit.plate storage.sneakUse fairy.time handwork.stick handwork.gather farm.composter farm.crop island.guardSide expand.guardSide handwork.stickOther",
 		"Use/plain":        "dimension.hunting dimension.stone screens.menuItem tinker.veinToggle copier.clear wand.options autofill.use",
 	}
 	for name, row := range rows {
~~~~

## REF T143

No subscriber stops the others. Prototype patch: `T143.patch`. REF1 "REF T138" holds: its text on what changes mechanically, its proofs and commands (with `S=…/scratchpad/T143`), `core/hook.go` and `core/hook_test.go` whole, and every diff it gives, EXCEPT the three things below. Compared file by file, the added and removed lines of `T143.patch` are those of the old `T138.patch` for 27 of its 29 files; the two that differ are `fairy/subscribe.go` and `order.golden`. `conduit/subscribe.go` has the same change on other context lines (T140 rewrote the comment above it), so the old patch does not apply to it.

1. `fairy/subscribe.go`: `fairy.time` is `On` in `core.Normal` and calls `UseFairyTime` alone (no `TimeItem`, no `island.CanBuild`, no import of `island`). The diff is below.
2. `conduit/subscribe.go`: the same registration change as in REF1, under T140's comment. The diff is below.
3. `order.golden` after the re-record, whole (122 lines; inside a tier sorted by name):

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
UseOnBlock LOW island.guard
UseOnBlock LOW wand.use
UseOnBlock NORMAL conduit.plate
UseOnBlock NORMAL fairy.time
UseOnBlock NORMAL handwork.gather
UseOnBlock NORMAL handwork.stick
UseOnBlock NORMAL storage.sneakUse
UseOnBlock HIGH farm.composter
UseOnBlock HIGH farm.crop
UseOnBlock HIGHEST expand.guardSide
UseOnBlock HIGHEST island.guardSide
UseOnBlock MONITOR handwork.stickOther
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
Join LOW dimension.joined
Join NORMAL climate.showWeather
Join NORMAL core.shapes
Join NORMAL fairy.backfillDust
Join NORMAL island.purgeLost
Join NORMAL progress.sendRecipes
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

The numbers the proofs give on this chain: the sorted dump has 3597 lines and is equal before and after (`cmp`); `git status --short content/skygear/testdata` prints exactly ` M content/skygear/testdata/pin/order.golden`; the check script prints 0 traces changed four times; `TestTierOrder` logs `5 orders, 3597 scenarios each`; `SKYGEAR_HOOK_ORDER=reverse go test -count=1 ./content/skygear/...` passes 28 packages; `go test -count=1 ./...` passes 62.

### `content/skygear/conduit/subscribe.go`

~~~~diff
--- a/content/skygear/conduit/subscribe.go
+++ b/content/skygear/conduit/subscribe.go
@@ -53,11 +53,9 @@ func subscribeBlocks() {
 // behind it is asked (Activate) and before the held item is used: a tier kit
 // or a fairy hourglass on a plate opens the settings.
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

### `content/skygear/fairy/subscribe.go`

~~~~diff
--- a/content/skygear/fairy/subscribe.go
+++ b/content/skygear/fairy/subscribe.go
@@ -30,12 +30,10 @@ func subscribeJoin() {
 // conduit.plate never act on the same click. A visiting fairy gives its gift
 // to a player who touches it, whichever way.
 func subscribeTouch() {
-	core.UseOnBlock.Handle(core.Normal, "fairy.time", func(e core.UseOnBlockEvent) bool {
+	core.UseOnBlock.On(core.Normal, "fairy.time", func(e core.UseOnBlockEvent) {
 		if UseFairyTime(e.S, e.P, e.Tx, e.Pos, e.Held) {
 			e.Cancel()
-			return true
 		}
-		return false
 	})
 	core.UseOnEntity.On(core.Normal, "fairy.touch", touchFairy)
 	core.AttackEntity.On(core.Normal, "fairy.touch", touchFairy)
~~~~

## REF T144

The items used on the air do their work in their own `Use`. Prototype patch: `T144.patch`. Section 3.4 has the design.

Behaviour change C14 (the client entries; unverified on a client). On the server every use does what it did.

What changes:

- `sgitem/airuse.go` (new): the four function variables, `used`, and `Use` of `ConfigCopier`, `Wand`, `InfinityWand`, `Hammer` and `DimensionStone`. `sgitem/dimension.go` loses the stub `Use`.
- `copier/subscribe.go`: `copier.clear` goes; `sgitem.CopierUsed` is set in its place. `wand/subscribe.go`: `wand.options` goes; `sgitem.WandUsed`. `tinker/subscribe.go`: `tinker.veinToggle` goes; `sgitem.HammerUsed` (the held item is read from the player and must be the iron hammer). `dimension/subscribe.go`: `dimension.stone` on `Use` goes; `sgitem.StoneUsed`; `dimension.stone` on `UseOnBlock` and `useStone` stay (the stone must act before the clicked block there).
- `pin_use_test.go`: the `Use/` scenarios go on to the held item's `Use` unless the use was cancelled; the rows of `Use/plain` and `Use/menuItemHunting`; two comments.

Before the change `TestAirItemsAreUsable` fails with `sgitem.ConfigCopier is not usable on the air`, and the same for `sgitem.Wand`, `sgitem.InfinityWand` and `sgitem.Hammer` (twice). `TestItemsUsedOnTheAir` (16 cases through `Player.UseItem`, on the island and in expansion mode) and `TestHunterUsesNothingOnTheAir` (8 cases) pass before and after: the uses and the guard are kept.

Goldens:

1. Re-record `^TestPin(Order|Use)$`. `order.golden` loses `Use LOWEST dimension.stone`, `Use NORMAL copier.clear`, `Use NORMAL tinker.veinToggle`, `Use NORMAL wand.options` (118 lines). The check script prints `handler-use: 123 scenarios, 27 traces changed` and exactly the three `RESULT` lines of section 6 (`Use/stone`, `Use/stoneAfterWindow`, `Use/stoneRepeat`: `cancelled=true` becomes `cancelled=false`, the rest of the line unchanged).
2. Re-record the wire goldens: `SKYGEAR_GOLDEN=update go test -count=1 ./content/skygear/wire`. Before the re-record `TestWire` fails with `golden testdata/items.golden differs: 14 changed, 0 added, 0 removed` and lists `registry/` and `start/` of `skygear:config_copier`, `skygear:diamond_wand`, `skygear:hammer`, `skygear:infinity_wand`, `skygear:iron_wand`, `skygear:stone_hammer`, `skygear:stone_wand`; the by-name golden the same 14. After it `git status --short content/skygear/wire` shows exactly `items.golden` and `items.byname.golden`, and `git diff --stat` 28 lines changed in each (14 removed, 14 added). Any other line or file: restore and report.
3. The proof that only the component was added, for the report: dump the entries before (a worktree of the commit before) and after with `SKYGEAR_GOLDEN_DUMP=<dir> go test -count=1 ./content/skygear/wire` and `diff -r` the two trees: 28 files differ, and every added line is one of `minecraft:use_modifiers: {`, `use_duration: float32(0.05)`, `movement_modifier: float32(1)`, `}` (28 of each on the prototype); nothing is removed.

### `content/skygear/airuse_test.go` (new file)

~~~~go
package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/itemnet"
	"github.com/presentkim/skygear/content/skygear/copier"
	"github.com/presentkim/skygear/content/skygear/dimension"
	"github.com/presentkim/skygear/content/skygear/expand"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/tinker"
	"github.com/presentkim/skygear/platform/account"
)

// The copier, the construction wands, the hammers and the teleport stone do
// their work on the air in their own Use, which Dragonfly calls after the
// handler's subscribers. These tests use the items through Dragonfly
// (Player.UseItem); the pin scenarios call the handler.

// airUse is what one real use of held on the air did: whether the item acted
// at all (it opened its screen or did its work, which arms the session's
// debounce), whether the copier in hand lost its card, and whether diagonal
// vein mining was switched.
type airUse struct{ acted, cleared, toggled bool }

func useOnTheAir(s *session, p *player.Player, held item.Stack, sneak bool) airUse {
	p.SetHeldItems(held, item.Stack{})
	if sneak {
		p.StartSneaking()
	} else {
		p.StopSneaking()
	}
	s.ForgetOpen(p)
	_, hadCard := copier.CardOf(held)
	diagonal := account.On(p, tinker.VeinDiagonalSetting)
	p.UseItem()
	now, _ := p.HeldItems()
	_, hasCard := copier.CardOf(now)
	return airUse{acted: s.Opened(p), cleared: hadCard && !hasCard, toggled: account.On(p, tinker.VeinDiagonalSetting) != diagonal}
}

func filledCopier() item.Stack {
	card := &copier.ConfigCard{Kind: copier.CardSide, Side: 4, Sides: []*copier.ConduitCard{{Mode: int(itemnet.Input), Extract: 5, Insert: 9}}}
	return copier.WithCard(item.NewStack(sgitem.ConfigCopier{}, 1), card, "ko")
}

// airCases are the items used on the air and what each use does for a player
// on the island.
var airCases = []struct {
	name  string
	held  func() item.Stack
	sneak bool
	want  airUse
}{
	{"a filled copier", filledCopier, false, airUse{acted: true, cleared: true}},
	{"a wand, sneaking", func() item.Stack { return item.NewStack(sgitem.NewWand(sgitem.WandStone), 1) }, true, airUse{acted: true}},
	{"a wand, standing", func() item.Stack { return item.NewStack(sgitem.NewWand(sgitem.WandStone), 1) }, false, airUse{}},
	{"the infinity wand, sneaking", func() item.Stack { return item.NewStack(sgitem.NewWand(sgitem.WandInfinity), 1) }, true, airUse{acted: true}},
	{"the iron hammer, sneaking", func() item.Stack { return item.NewStack(sgitem.Hammer{}, 1) }, true, airUse{acted: true, toggled: true}},
	{"the iron hammer, standing", func() item.Stack { return item.NewStack(sgitem.Hammer{}, 1) }, false, airUse{}},
	{"the stone hammer, sneaking", func() item.Stack { return item.NewStack(sgitem.Hammer{Stone: true}, 1) }, true, airUse{}},
	{"the teleport stone", func() item.Stack { return item.NewStack(sgitem.DimensionStone{}, 1) }, false, airUse{acted: true}},
}

// TestItemsUsedOnTheAir: each item does its work when a player on the island
// uses it on the air, and in island expansion mode, where nothing refuses a
// use on the air.
func TestItemsUsedOnTheAir(t *testing.T) {
	for _, mode := range []string{"on the island", "in expansion mode"} {
		for _, c := range airCases {
			s := testSession(t)
			island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
			loadAround(s)
			h := spawnPlayer(s)
			withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
				p.Handle(s.Handler(p))
				if mode == "in expansion mode" {
					expand.Show(s.Session, p)
				}
				if got := useOnTheAir(s, p, c.held(), c.sneak); got != c.want {
					t.Errorf("%s, %s: the use did %+v, want %+v", c.name, mode, got, c.want)
				}
			})
		}
	}
}

// TestHunterUsesNothingOnTheAir: in the second dimension a use on the air is
// refused whatever the hunter holds but the menu item (dimension.hunting
// cancels it), so Dragonfly never calls the item's Use: no item acts.
func TestHunterUsesNothingOnTheAir(t *testing.T) {
	for _, c := range airCases {
		s := testSession(t)
		island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
		loadAround(s)
		h := spawnPlayer(s)
		withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
			p.Handle(s.Handler(p))
			if !dimension.Begin(s.Session, tx, p, sgitem.MobSlime) {
				t.Fatal("the run did not begin")
			}
			if got := useOnTheAir(s, p, c.held(), c.sneak); got != (airUse{}) {
				t.Errorf("%s: a hunter's use did %+v, want nothing", c.name, got)
			}
		})
	}
}

// TestAirItemsAreUsable: the items used on the air are item.Usable, which is
// also what tells a client to send the use at all (clientdef.ItemEntry gives
// such an item minecraft:use_modifiers).
func TestAirItemsAreUsable(t *testing.T) {
	for _, it := range []world.Item{
		sgitem.ConfigCopier{}, sgitem.Wand{Tier: sgitem.WandStone}, sgitem.InfinityWand{},
		sgitem.Hammer{}, sgitem.Hammer{Stone: true}, sgitem.DimensionStone{},
	} {
		if _, ok := it.(item.Usable); !ok {
			t.Errorf("%T is not usable on the air", it)
		}
	}
}
~~~~

### `content/skygear/copier/subscribe.go`

~~~~diff
--- a/content/skygear/copier/subscribe.go
+++ b/content/skygear/copier/subscribe.go
@@ -1,6 +1,9 @@
 package copier
 
 import (
+	"github.com/df-mc/dragonfly/server/item"
+	"github.com/df-mc/dragonfly/server/player"
+	"github.com/df-mc/dragonfly/server/world"
 	"github.com/presentkim/skygear/content/skygear/core"
 	"github.com/presentkim/skygear/content/skygear/island"
 	"github.com/presentkim/skygear/content/skygear/sgitem"
@@ -14,7 +17,10 @@ func subscribe() {
 // The configuration copier copies and pastes settings and does nothing else.
 // Where the island does not allow building it leaves the click alone: the
 // island's guard refuses it. A held tool takes the click away from what the
-// block itself would do with it: the Low tier.
+// block itself would do with it: the Low tier. Used on the air the copier is
+// emptied: no block is asked first there, so that is the item's own Use
+// (sgitem.ConfigCopier), which Dragonfly calls unless a subscriber cancelled
+// the use.
 func subscribeUse() {
 	core.UseOnBlock.On(core.Low, "copier.use", func(e core.UseOnBlockEvent) {
 		if _, ok := e.Held.Item().(sgitem.ConfigCopier); ok && island.CanBuild(e.S, e.Pos) {
@@ -23,9 +29,13 @@ func subscribeUse() {
 			UseCopier(e.S, e.P, e.Tx, e.Pos, e.Face, e.Click, e.Held, off)
 		}
 	})
-	core.Use.On(core.Normal, "copier.clear", func(e core.UseEvent) {
-		if _, ok := e.Held.Item().(sgitem.ConfigCopier); ok {
-			ClearCopier(e.S, e.P, e.Held, e.Off)
+	sgitem.CopierUsed = func(tx *world.Tx, u item.User) {
+		p, ok := u.(*player.Player)
+		s, ok2 := core.Of(tx)
+		if !ok || !ok2 {
+			return
 		}
-	})
+		held, off := p.HeldItems()
+		ClearCopier(s, p, held, off)
+	}
 }
~~~~

### `content/skygear/dimension/subscribe.go`

~~~~diff
--- a/content/skygear/dimension/subscribe.go
+++ b/content/skygear/dimension/subscribe.go
@@ -3,6 +3,7 @@ package dimension
 import (
 	"github.com/df-mc/dragonfly/server/item"
 	"github.com/df-mc/dragonfly/server/player"
+	"github.com/df-mc/dragonfly/server/world"
 	"github.com/presentkim/skygear/content/skygear/core"
 	"github.com/presentkim/skygear/content/skygear/sgitem"
 )
@@ -99,9 +100,16 @@ func subscribeUse() {
 			e.Cancel()
 		}
 	})
-	core.Use.On(core.Lowest, "dimension.stone", func(e core.UseEvent) {
-		useStone(e.Actor, e.Held)
-	})
+	// On the air no block is asked before the item: the stone opens its
+	// window in its own Use (sgitem.DimensionStone), which Dragonfly calls
+	// unless a subscriber cancelled the use (a hunter's: dimension.hunting).
+	sgitem.StoneUsed = func(tx *world.Tx, u item.User) {
+		p, ok := u.(*player.Player)
+		s, ok2 := core.Of(tx)
+		if ok && ok2 && !Hunting(s, p) && !s.Debounced(p) {
+			ShowWindow(s, p, "")
+		}
+	}
 	core.UseOnEntity.On(core.Lowest, "dimension.hunting", func(e core.EntityEvent) {
 		if Hunting(e.S, e.P) {
 			e.Cancel()
@@ -114,8 +122,8 @@ func subscribeUse() {
 	})
 }
 
-// useStone opens the second dimension's window for a held teleport stone,
-// on a block or the air, and cancels the click it took. A hunter's stone
+// useStone opens the second dimension's window for a teleport stone held on a
+// block, and cancels the click it took: the block under it must not open. A hunter's stone
 // opens nothing: the hunter's use is refused (dimension.hunting).
 func useStone(a core.Actor, held item.Stack) {
 	if _, ok := held.Item().(sgitem.DimensionStone); !ok || Hunting(a.S, a.P) {
~~~~

### `content/skygear/pin_use_test.go`

~~~~diff
--- a/content/skygear/pin_use_test.go
+++ b/content/skygear/pin_use_test.go
@@ -538,7 +538,17 @@ func TestPinUse(t *testing.T) {
 		if extra != nil {
 			ex = effects(sc, extra)
 		}
-		event("Use/"+name, sc, prior, forget, func(hd *core.PlayerHandler, ctx *player.Context, p *player.Player) { hd.HandleItemUse(ctx) }, ex)
+		// The game calls the held item's own Use after the handler, unless
+		// the use was cancelled (Player.UseItem): the copier, the wands, the
+		// hammers and the teleport stone do their work on the air there.
+		event("Use/"+name, sc, prior, forget, func(hd *core.PlayerHandler, ctx *player.Context, p *player.Player) {
+			hd.HandleItemUse(ctx)
+			if h, _ := p.HeldItems(); !ctx.Cancelled() {
+				if u, ok := h.Item().(item.Usable); ok {
+					u.Use(p.Tx(), p, &item.UseContext{})
+				}
+			}
+		}, ex)
 	}
 	veinDiagonal := func(s *session, p *player.Player, tx *world.Tx) string {
 		return fmt.Sprintf("diagonal=%t", account.On(p, tinker.VeinDiagonalSetting))
@@ -589,11 +599,11 @@ func TestPinUse(t *testing.T) {
 	// 3 dimension.hunting
 	onUse("hunting", chain(hunting, holding(stone)), 0, false, held)
 	onUse("huntingStone", chain(hunting, holding(dimStone)), 0, false, held)
-	// 4 dimension.stone
+	// 4 the teleport stone: its own Use (sgitem.DimensionStone)
 	onUse("stone", holding(dimStone), 0, false, held)
 	onUse("stoneRepeat", holding(dimStone), 1, false, held)
 	onUse("stoneAfterWindow", holding(dimStone), 1, true, held)
-	// 5 copier.clear, 6 wand.options, 7 tinker.veinToggle
+	// 5 the copier, 6 the wands, 7 the hammers: their own Use (sgitem)
 	card := &copier.ConfigCard{Kind: copier.CardSide, Side: 4, Sides: []*copier.ConduitCard{{Mode: int(itemnet.Input), Extract: 5, Insert: 9}}}
 	filled := copier.WithCard(copierItem, card, "ko")
 	cardIn := func(s *session, p *player.Player, tx *world.Tx) string {
@@ -700,7 +710,7 @@ func TestPinUse(t *testing.T) {
 
 	rows := map[string]string{
 		"UseOnBlock/plain": "dimension.hunting dimension.stone expand.guard screens.menuItem island.guard copier.use wand.use conduit.plate storage.sneakUse fairy.time handwork.stick handwork.gather farm.composter farm.crop island.guardSide expand.guardSide handwork.stickOther",
-		"Use/plain":        "dimension.hunting dimension.stone screens.menuItem tinker.veinToggle copier.clear wand.options autofill.use",
+		"Use/plain":        "dimension.hunting screens.menuItem autofill.use",
 	}
 	for name, row := range rows {
 		if got := lines[name]; hookOrder == "" && !strings.HasPrefix(got, row+" | ") {
@@ -714,7 +724,7 @@ func TestPinUse(t *testing.T) {
 		"UseOnBlock/expanding":    "dimension.hunting dimension.stone expand.guard | cancelled=true",
 		"UseOnBlock/outside":      "dimension.hunting dimension.stone expand.guard screens.menuItem island.guard | cancelled=true",
 		"UseOnBlock/cropSneaking": "farm.crop island.guardSide expand.guardSide",
-		"Use/menuItemHunting":     "dimension.hunting dimension.stone screens.menuItem | cancelled=true",
+		"Use/menuItemHunting":     "dimension.hunting screens.menuItem | cancelled=true",
 		"AttackEntity/monster":    "dimension.hunter | cancelled=true",
 	} {
 		if got := lines[name]; hookOrder == "" && !strings.Contains(got, want) {
~~~~

### `content/skygear/sgitem/airuse.go` (new file)

~~~~go
package sgitem

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

// The items of this package that are used on the air are item.Usable:
// Dragonfly calls their Use after the handler's subscribers, unless one of
// them cancelled the use. What the use does belongs to a feature, and this
// package is a data layer below the features, so the feature sets the
// function of its item from its Setup. An item whose function is not set
// does nothing.
var (
	// CopierUsed is the use of a configuration copier on the air (copier).
	CopierUsed func(tx *world.Tx, user item.User)
	// WandUsed is the use of a construction wand on the air (wand).
	WandUsed func(tx *world.Tx, user item.User)
	// HammerUsed is the use of a hammer on the air (tinker).
	HammerUsed func(tx *world.Tx, user item.User)
	// StoneUsed is the use of a dimension teleport stone on the air
	// (dimension).
	StoneUsed func(tx *world.Tx, user item.User)
)

// used calls f, if it is set, and reports false. A use of these items
// changes the stack in hand or opens a screen by itself; for a true result
// Dragonfly would swing the arm and write the stack it read before the use
// back into the hand.
func used(f func(*world.Tx, item.User), tx *world.Tx, user item.User) bool {
	if f != nil {
		f(tx, user)
	}
	return false
}

// Use implements item.Usable (CopierUsed).
func (ConfigCopier) Use(tx *world.Tx, user item.User, _ *item.UseContext) bool {
	return used(CopierUsed, tx, user)
}

// Use implements item.Usable (WandUsed).
func (Wand) Use(tx *world.Tx, user item.User, _ *item.UseContext) bool {
	return used(WandUsed, tx, user)
}

// Use implements item.Usable (WandUsed).
func (InfinityWand) Use(tx *world.Tx, user item.User, _ *item.UseContext) bool {
	return used(WandUsed, tx, user)
}

// Use implements item.Usable (HammerUsed).
func (Hammer) Use(tx *world.Tx, user item.User, _ *item.UseContext) bool {
	return used(HammerUsed, tx, user)
}

// Use implements item.Usable (StoneUsed).
func (DimensionStone) Use(tx *world.Tx, user item.User, _ *item.UseContext) bool {
	return used(StoneUsed, tx, user)
}
~~~~

### `content/skygear/sgitem/dimension.go`

~~~~diff
--- a/content/skygear/sgitem/dimension.go
+++ b/content/skygear/sgitem/dimension.go
@@ -1,14 +1,12 @@
 package sgitem
 
 import (
-	"github.com/df-mc/dragonfly/server/item"
 	"github.com/df-mc/dragonfly/server/item/category"
-	"github.com/df-mc/dragonfly/server/world"
 	"github.com/presentkim/skygear/common/i18n"
 )
 
 // DimensionStone opens the second dimension's window when used; it is never
-// used up (dimension package).
+// used up (dimension package; its Use is in airuse.go).
 type DimensionStone struct{}
 
 func (DimensionStone) EncodeItem() (string, int16) { return "skygear:dimension_stone", 0 }
@@ -18,7 +16,3 @@ func (DimensionStone) Name() string { return i18n.T(i18n.Default, "name.Dimensio
 func (DimensionStone) Category() category.Category { return category.Equipment() }
 
 func (DimensionStone) MaxCount() int { return 1 }
-
-// Use does nothing itself (the game's handler opens the window first): being
-// usable makes the client send a use on the air at all.
-func (DimensionStone) Use(*world.Tx, item.User, *item.UseContext) bool { return false }
~~~~

### `content/skygear/tinker/subscribe.go`

~~~~diff
--- a/content/skygear/tinker/subscribe.go
+++ b/content/skygear/tinker/subscribe.go
@@ -2,6 +2,7 @@ package tinker
 
 import (
 	"github.com/df-mc/dragonfly/server/item"
+	"github.com/df-mc/dragonfly/server/player"
 	"github.com/df-mc/dragonfly/server/world"
 	"github.com/presentkim/skygear/content/skygear/core"
 	"github.com/presentkim/skygear/content/skygear/look"
@@ -75,13 +76,19 @@ func subscribeTools() {
 }
 
 // The iron hammer used on the air while sneaking switches diagonal vein
-// mining.
+// mining: the item's own Use (sgitem.Hammer), which Dragonfly calls unless a
+// subscriber cancelled the use. The stone hammer does nothing there.
 func subscribeVeinToggle() {
-	core.Use.On(core.Normal, "tinker.veinToggle", func(e core.UseEvent) {
-		if sgitem.IsIronHammer(e.Held) && e.P.Sneaking() && !e.S.Debounced(e.P) {
-			ToggleVeinDiagonal(e.S, e.P)
+	sgitem.HammerUsed = func(tx *world.Tx, u item.User) {
+		p, ok := u.(*player.Player)
+		s, ok2 := core.Of(tx)
+		if !ok || !ok2 {
+			return
 		}
-	})
+		if held, _ := p.HeldItems(); sgitem.IsIronHammer(held) && p.Sneaking() && !s.Debounced(p) {
+			ToggleVeinDiagonal(s, p)
+		}
+	}
 }
 
 // On the fast tick the vein outline follows the block a player looks at.
~~~~

### `content/skygear/wand/subscribe.go`

~~~~diff
--- a/content/skygear/wand/subscribe.go
+++ b/content/skygear/wand/subscribe.go
@@ -1,6 +1,9 @@
 package wand
 
 import (
+	"github.com/df-mc/dragonfly/server/item"
+	"github.com/df-mc/dragonfly/server/player"
+	"github.com/df-mc/dragonfly/server/world"
 	"github.com/presentkim/skygear/content/skygear/core"
 	"github.com/presentkim/skygear/content/skygear/island"
 	"github.com/presentkim/skygear/content/skygear/look"
@@ -13,11 +16,13 @@ func subscribe() {
 	followLook()
 }
 
-// The construction wand builds out the face clicked and does nothing else;
-// used on the air while sneaking it shows its settings. Where the island does
-// not allow building it leaves the click alone: the island's guard refuses it.
-// A held tool takes the click away from what the block itself would do with
-// it: the Low tier.
+// The construction wand builds out the face clicked and does nothing else.
+// Where the island does not allow building it leaves the click alone: the
+// island's guard refuses it. A held tool takes the click away from what the
+// block itself would do with it: the Low tier. Used on the air while sneaking
+// the wand shows its settings: no block is asked first there, so that is the
+// item's own Use (sgitem.Wand, sgitem.InfinityWand), which Dragonfly calls
+// unless a subscriber cancelled the use.
 func subscribeUse() {
 	core.UseOnBlock.On(core.Low, "wand.use", func(e core.UseOnBlockEvent) {
 		if _, ok := sgitem.WandOf(e.Held); ok && island.CanBuild(e.S, e.Pos) {
@@ -25,11 +30,13 @@ func subscribeUse() {
 			Use(e.S, e.P, e.Tx, e.Pos, e.Face, e.Held)
 		}
 	})
-	core.Use.On(core.Normal, "wand.options", func(e core.UseEvent) {
-		if _, ok := sgitem.WandOf(e.Held); ok && e.P.Sneaking() && !e.S.Debounced(e.P) {
-			ShowOptions(e.S, e.P)
+	sgitem.WandUsed = func(tx *world.Tx, u item.User) {
+		p, ok := u.(*player.Player)
+		s, ok2 := core.Of(tx)
+		if ok && ok2 && p.Sneaking() && !s.Debounced(p) {
+			ShowOptions(s, p)
 		}
-	})
+	}
 }
 
 // The preview follows where a player holding a wand looks.
~~~~

## REF T145

A hunter that falls dies for real and respawns at once. Prototype patch: `T145.patch`. Section 3.5 has the design, the order of what the server does and the fixes.

Behaviour changes C15 to C19. C6 (T133) is replaced by C16; T133's change to `island.void` and `walkvoid_test.go` are not touched.

What changes:

- `dimension/subscribe.go`: `dimension.fallen` goes; two comments.
- `dimension/run.go`: `runLocal.results` becomes `fallen` (the result and the return place); `Begin` sends the game rule (`EnableInstantRespawn`); `Run.end` takes it back for a living hunter, loses the "healed without dying" branch and keeps the return place for a dead one; `Respawned` sets the return place, takes the rule back and schedules the landing (`landAfter`) and the result; `Fallen` goes.
- `dimension/monster.go`: the melee hit adds knock-back and its effect only to a target that is alive.
- `platform/instance/death_test.go` (new): the platform keeps the inventory for every death.
- `fellinghit_test.go` goes (it asserted C6); `seconddeath_test.go` (new) has its place. `dimension/dimension_test.go`: the last part of `TestHuntClear`, which called `dimension.Fallen`, goes with the function; `seconddeath_test.go` covers it with a real hit.

Before the change `seconddeath_test.go` fails like this; after it all five pass:

- `TestHunterDiesAndComesBack`: `the hit that kills the hunter did not land`.
- `TestHitThatKillsLeavesNoEffect`: `the monster did not kill the hunter`. With the change but without the guard in `monster.go`: `the dead hunter has 1 effects`.
- `TestDeadHunterWhoLeavesIsBackOnTheIsland`: `the hunter did not die`.
- `TestBlastThatKillsDoesNotThrowTheHunter`: `the blast that felled the hunter threw it: velocity [-0.5658951984586352 0.4 -0.565475573620525]` and `the hunter did not die of the blast`.
- `TestDeathOnTheIslandIsUntouched` passes before and after: a death outside a run is counted as before and the respawn handler leaves the platform's place alone.

A test player has no connection, and Dragonfly closes such a player 1.1 seconds after its death: every check of a test is in the transaction of the hit. `Player.Respawn` does nothing without a connection, so the tests call `HandleRespawn` on the player's handler, as `Player.respawn` does.

Goldens: re-record `^TestPin(Order|Simple)$`. `order.golden` loses `Hurt HIGHEST dimension.fallen` (117 lines). The check script prints `handler-simple: 29 scenarios, 2 traces changed` (`Hurt/fallen`, `Hurt/plain`) and exactly the one `RESULT` line of section 6 (`Hurt/fallen`: `cancelled=true` becomes `cancelled=false`). The dump's line `Hurt/fallen` has the counters `hurt+1` instead of `wave_fail+1`.

### `content/skygear/dimension/dimension_test.go`

~~~~diff
--- a/content/skygear/dimension/dimension_test.go
+++ b/content/skygear/dimension/dimension_test.go
@@ -139,18 +139,6 @@ func TestHuntFails(t *testing.T) {
 			t.Error("the hunter stayed in the ground")
 		}
 
-		// a hit that would kill the hunter ends the run instead: no death,
-		// straight back, healed
-		dimension.Begin(s.Session, tx, p, sgitem.MobSlime)
-		if dimension.Fallen(s.Session, tx, p, 1) {
-			t.Fatal("a light hit felled the hunter")
-		}
-		if !dimension.Fallen(s.Session, tx, p, 1000) {
-			t.Fatal("a deadly hit did not fell the hunter")
-		}
-		if dimension.RunOf(s.Session) != nil || core.InArena(p.Position()) || p.Health() != p.MaxHealth() {
-			t.Errorf("after falling: run %v, at %v, health %v", dimension.RunOf(s.Session) != nil, p.Position(), p.Health())
-		}
 	})
 }
 
~~~~

### `content/skygear/dimension/monster.go`

~~~~diff
--- a/content/skygear/dimension/monster.go
+++ b/content/skygear/dimension/monster.go
@@ -310,7 +310,9 @@ func (m *Monster) Tick(tx *world.Tx, _ int64) {
 		}
 		if v.Fuse == 0 && flat.Len() < monsterReach && math.Abs(to[1]) < 2 && d.cooldown == 0 {
 			d.cooldown = monsterCooldown
-			if _, hurt := target.Hurt(st.damage, entity.AttackDamageSource{Attacker: m}); hurt {
+			// A hit that killed the hunter leaves no effect on it: a death
+			// clears the effects and a respawn would bring this one home.
+			if _, hurt := target.Hurt(st.damage, entity.AttackDamageSource{Attacker: m}); hurt && !target.Dead() {
 				target.KnockBack(m.Data.Pos, 0.4, 0.3)
 				if v.Hit != nil {
 					target.AddEffect(*v.Hit)
~~~~

### `content/skygear/dimension/run.go`

~~~~diff
--- a/content/skygear/dimension/run.go
+++ b/content/skygear/dimension/run.go
@@ -162,12 +162,19 @@ func (r *Run) fightTicks() int { return max(0, r.ticks-countdownTicks) }
 type runLocal struct {
 	// Run is the hunt being fought, nil when none.
 	Run *Run
-	// results wait for hunters that died to show once they are back.
-	results map[*world.EntityHandle]result
+	// fallen are the hunters that died and have not respawned yet: how the
+	// run went, to show once they are back, and where they go back to.
+	fallen map[*world.EntityHandle]fallen
 	// watchers are the players watching the run (window.go).
 	watchers map[*world.EntityHandle]watcher
 }
 
+// fallen is what waits for a hunter that died.
+type fallen struct {
+	res  result
+	back mgl64.Vec3
+}
+
 var runPart = core.NewPart(func() *runLocal { return &runLocal{} })
 
 // RunOf is the hunt being fought in the save of s, nil when none.
@@ -210,6 +217,10 @@ func Begin(s *core.Session, tx *world.Tx, p *player.Player, mob string) bool {
 	// PlaceAt holds the hunter on its spot until the client has the ground.
 	p.Teleport(start)
 	instance.PlaceAt(p, start, start)
+	// A hunter that dies is back at once: its client is told to respawn
+	// without the death screen while the run lasts (the game rule goes to
+	// this player alone; end and Respawned take it back).
+	p.EnableInstantRespawn()
 	achievement.Count(p, "dimension_enter", 1)
 	return true
 }
@@ -396,8 +407,8 @@ type result struct {
 }
 
 // end finishes the run: its monsters and gate go, the hunter (nil when gone)
-// goes back and sees how it went. A dead hunter sees it when it is back
-// (Respawned).
+// goes back and sees how it went. A dead hunter goes back and sees it when
+// it respawns (Respawned).
 func (r *Run) end(s *core.Session, tx *world.Tx, p *player.Player, how string) {
 	l := runPart.Of(s)
 	if l.Run != r {
@@ -450,24 +461,23 @@ func (r *Run) end(s *core.Session, tx *world.Tx, p *player.Player, how string) {
 	} else {
 		achievement.Count(p, "wave_fail", 1)
 	}
-	if how == endDied && !p.Dead() {
-		// fallen (Fallen): no death screen, straight back and healed
-		p.Heal(p.MaxHealth(), entity.FoodHealingSource{})
-	}
 	switch {
 	case how == endDied && p.Dead():
-		// the inventory is kept through death: the rewards go in now, the
-		// result shows once the hunter is back
+		// the inventory is kept through death (the platform keeps it for
+		// every death): the rewards go in now; the hunter goes back and sees
+		// the result when it respawns
 		deliver(s, tx, p, res)
-		if l.results == nil {
-			l.results = map[*world.EntityHandle]result{}
+		if l.fallen == nil {
+			l.fallen = map[*world.EntityHandle]fallen{}
 		}
-		l.results[p.H()] = res
+		l.fallen[p.H()] = fallen{res, r.Return}
 	case how == endQuit:
 		// leaving the save: back on the island before the state is stored
+		p.DisableInstantRespawn()
 		p.Teleport(r.Return)
 		deliver(s, tx, p, res)
 	default:
+		p.DisableInstantRespawn()
 		p.Teleport(r.Return)
 		instance.PlaceAt(p, r.Return, island.Spawn(s))
 		deliver(s, tx, p, res)
@@ -476,33 +486,35 @@ func (r *Run) end(s *core.Session, tx *world.Tx, p *player.Player, how string) {
 }
 
 // Respawned sends p, a hunter that died, back where it came from and shows
-// how the run went. It runs in HandleRespawn.
+// how the run went. It runs in HandleRespawn: pos is where Dragonfly puts the
+// respawned player. Once the player is back in the world it is held on safe
+// ground there until the client has the chunks (instance.PlaceAt), as at the
+// end of a run the hunter survived.
 func Respawned(s *core.Session, p *player.Player, pos *mgl64.Vec3) {
 	l := runPart.Of(s)
-	res, ok := l.results[p.H()]
+	f, ok := l.fallen[p.H()]
 	if !ok {
 		return
 	}
-	delete(l.results, p.H())
-	*pos = island.Spawn(s)
+	delete(l.fallen, p.H())
+	*pos = f.back
+	p.DisableInstantRespawn()
+	p.H().DoAfter(landAfter, func(tx *world.Tx, e world.Entity) {
+		if c, ok := core.Of(tx); ok && c == s {
+			instance.PlaceAt(e.(*player.Player), f.back, island.Spawn(s))
+		}
+	})
 	p.H().DoAfter(500*time.Millisecond, func(tx *world.Tx, e world.Entity) {
 		if c, ok := core.Of(tx); ok && c == s {
-			showResult(e.(*player.Player), res)
+			showResult(e.(*player.Player), f.res)
 		}
 	})
 }
 
-// Fallen ends the run of a hunter a hit of dmg would kill, before it dies:
-// there is no death screen, it goes straight back and sees the result. It
-// reports whether the hit is to be cancelled (HandleHurt).
-func Fallen(s *core.Session, tx *world.Tx, p *player.Player, dmg float64) bool {
-	r := RunOf(s)
-	if r == nil || r.Hunter != p.H() || p.Health()+p.Absorption()-dmg > 1e-6 {
-		return false
-	}
-	r.end(s, tx, p, endDied)
-	return true
-}
+// landAfter is how long after the respawn handler the respawned hunter is
+// put on safe ground: Dragonfly takes the player out of the world and adds it
+// again right after the handler.
+const landAfter = 50 * time.Millisecond
 
 // Frozen reports whether p is a hunter whose hunt waits for its level-up
 // choice: it cannot move or be hurt meanwhile.
@@ -524,7 +536,8 @@ func (r *Run) showTip(s *core.Session, p *player.Player) {
 	p.SendTip(tip)
 }
 
-// Died ends the run of a hunter that died.
+// Died ends the run of a hunter that died: the hunter has what it carried
+// again and its rewards; it goes back when it respawns (Respawned).
 func Died(s *core.Session, tx *world.Tx, p *player.Player) {
 	if r := RunOf(s); r != nil && r.Hunter == p.H() {
 		r.end(s, tx, p, endDied)
@@ -538,7 +551,7 @@ func Leave(s *core.Session, tx *world.Tx, p *player.Player) {
 		r.end(s, tx, p, endQuit)
 	}
 	StopWatching(s, p)
-	delete(runPart.Of(s).results, p.H())
+	delete(runPart.Of(s).fallen, p.H())
 }
 
 // Joined gives a player that was in a run when the server stopped what it
~~~~

### `content/skygear/dimension/subscribe.go`

~~~~diff
--- a/content/skygear/dimension/subscribe.go
+++ b/content/skygear/dimension/subscribe.go
@@ -20,7 +20,9 @@ func subscribe() {
 	registerKinds()
 }
 
-// The second dimension's reaction to a death and a respawn.
+// A hunter that dies loses the run: the death ends it (dimension.died) and
+// the respawn brings the hunter back to where it came from, before the food
+// is put back (dimension.respawned sets the place the other steps read: Low).
 func subscribeLife() {
 	core.Death.On(core.Normal, "dimension.died", func(e core.NoticeEvent) {
 		Died(e.S, e.P.Tx(), e.P)
@@ -30,8 +32,8 @@ func subscribeLife() {
 	})
 }
 
-// The second dimension's reaction to a drop and a hurt. Whether a hit would
-// kill the hunter is asked last, of the damage the other steps left: Highest.
+// The second dimension's reaction to a drop and a hurt. A hit that kills the
+// hunter is not refused: the hunter dies and the run ends in dimension.died.
 func subscribeHurt() {
 	core.ItemDrop.On(core.Lowest, "dimension.hunting", func(e core.ItemEvent) {
 		if Hunting(e.S, e.P) {
@@ -43,11 +45,6 @@ func subscribeHurt() {
 			e.Cancel() // the hunt waits while the hunter chooses an upgrade
 		}
 	})
-	core.Hurt.On(core.Highest, "dimension.fallen", func(e core.HurtEvent) {
-		if Fallen(e.S, e.Tx, e.P, *e.Damage) {
-			e.Cancel() // no hurt for the subscribers after it: the run ended instead
-		}
-	})
 }
 
 // The collectors work and the items of the other dimension are retired every
~~~~

### `content/skygear/fellinghit_test.go`

~~~~diff
--- a/content/skygear/fellinghit_test.go
+++ /dev/null
@@ -1,61 +0,0 @@
-package skygear
-
-import (
-	"testing"
-	"time"
-
-	"github.com/df-mc/dragonfly/server/player"
-	"github.com/df-mc/dragonfly/server/world"
-	"github.com/presentkim/skygear/content/skygear/dimension"
-	"github.com/presentkim/skygear/content/skygear/sgitem"
-	"github.com/presentkim/skygear/platform/account"
-	"github.com/presentkim/skygear/platform/instance"
-)
-
-// TestFellingHitIsNotAHurt: the hit that would kill a hunter is cancelled
-// (the run ends instead and the hunter goes back unharmed), and a cancelled
-// hit is not a hurt: the hurt counter is not called for it. A hit the hunter
-// takes counts.
-func TestFellingHitIsNotAHurt(t *testing.T) {
-	d := bridgeAccount(t)
-	s := testSession(t)
-	loadAround(s)
-	h := spawnPlayer(s)
-	t.Cleanup(func() {
-		s.World.Do(func(tx *world.Tx) {
-			if e, ok := h.Entity(tx); ok {
-				tx.RemoveEntity(e)
-			}
-		})
-	})
-	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
-		acc := instance.AccountID(p)
-		if _, err := d.TouchAccount(acc, "", p.Name()); err != nil {
-			t.Fatal(err)
-		}
-		if !dimension.Begin(s.Session, tx, p, sgitem.MobSlime) {
-			t.Fatal("the hunt did not begin")
-		}
-		hurt := func(dmg float64) *player.Context {
-			ctx := player.NewEventContext(tx, p)
-			var immunity time.Duration
-			s.Handler(p).HandleHurt(ctx, &dmg, false, &immunity, nil)
-			return ctx
-		}
-		if hurt(1).Cancelled() {
-			t.Fatal("a hit the hunter survives was cancelled")
-		}
-		if got := account.Counter(acc, "hurt"); got != 1 {
-			t.Errorf("hurt counted %d for a hit the hunter took, want 1", got)
-		}
-		if !hurt(p.Health() + 100).Cancelled() {
-			t.Fatal("the hit that would kill the hunter was not cancelled")
-		}
-		if dimension.Hunting(s.Session, p) {
-			t.Fatal("the run did not end")
-		}
-		if got := account.Counter(acc, "hurt"); got != 1 {
-			t.Errorf("hurt counted %d after the cancelled felling hit, want 1", got)
-		}
-	})
-}
~~~~

### `content/skygear/seconddeath_test.go` (new file)

~~~~go
package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/dimension"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/platform/account"
	"github.com/presentkim/skygear/platform/instance"
)

// A hunter that takes a hit it does not survive dies for real: the death ends
// the run and the respawn brings the hunter back. These tests kill through
// Dragonfly (Player.Hurt). The respawn itself needs a connected client
// (Player.Respawn does nothing without one), so the tests call the respawn
// handler as Dragonfly calls it.

// keepsInventory is the handler of a player in an instance as far as a death
// goes: the platform keeps the inventory for every death
// (platform/instance.Handler.HandleDeath) before the game's handler runs.
type keepsInventory struct{ player.Handler }

func (h keepsInventory) HandleDeath(p *player.Player, src world.DamageSource, keepInv *bool) {
	*keepInv = true
	h.Handler.HandleDeath(p, src, keepInv)
}

// fallenHunter is a session whose player carries three stone and seven
// levels, is in the second dimension and has the handler of an instance.
func fallenHunter(t *testing.T) (*session, *world.EntityHandle, string, mgl64.Vec3) {
	t.Helper()
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
	var (
		acc  string
		from mgl64.Vec3
	)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		acc = instance.AccountID(p)
		if _, err := d.TouchAccount(acc, "", p.Name()); err != nil {
			t.Fatal(err)
		}
		p.Handle(keepsInventory{s.Handler(p)})
		_ = p.Inventory().SetItem(0, item.NewStack(block.Stone{}, 3))
		p.SetExperienceLevel(7)
		from = p.Position()
		if !dimension.Begin(s.Session, tx, p, sgitem.MobSlime) {
			t.Fatal("the run did not begin")
		}
	})
	return s, h, acc, from
}

// TestHunterDiesAndComesBack: a hit the hunter survives is a hurt; the hit
// that kills it is a hurt and a death. The death ends the run, once: the
// hunter has what it carried and its levels again, nothing lies on the
// ground, the lost run is counted once. The respawn puts the hunter where it
// went in from.
func TestHunterDiesAndComesBack(t *testing.T) {
	s, h, acc, from := fallenHunter(t)
	count := func(key string) int64 { return account.Counter(acc, key) }
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		if _, hurt := p.Hurt(1, entity.AttackDamageSource{}); !hurt || p.Dead() || !dimension.Hunting(s.Session, p) {
			t.Fatal("a light hit did not land, or ended the run")
		}
		if got := count("hurt"); got != 1 {
			t.Errorf("hurt counted %d after a hit the hunter took, want 1", got)
		}
		if _, hurt := p.Hurt(1000, entity.AttackDamageSource{}); !hurt {
			t.Fatal("the hit that kills the hunter did not land")
		}
		if !p.Dead() {
			t.Fatal("the hunter did not die")
		}
		if dimension.Hunting(s.Session, p) || dimension.RunOf(s.Session) != nil {
			t.Fatal("the run did not end with the hunter's death")
		}
		if got, want := [3]int64{count("hurt"), count("death"), count("wave_fail")}, [3]int64{2, 1, 1}; got != want {
			t.Errorf("hurt, death, wave_fail counted %v, want %v", got, want)
		}
		if st, _ := p.Inventory().Item(0); !st.Equal(item.NewStack(block.Stone{}, 3)) {
			t.Errorf("slot 0 holds %v after the death, want the three stone the hunter carried", st)
		}
		for _, st := range p.Armour().Slots() {
			if !st.Empty() {
				t.Errorf("the dead hunter still wears %v of the kit", st)
			}
		}
		if n := len(droppedItems(tx)); n != 0 {
			t.Errorf("%d kinds of items lie on the ground after the death: %v", n, droppedItems(tx))
		}
		if p.ExperienceLevel() != 7 {
			t.Errorf("the hunter has %d levels after the death, want 7", p.ExperienceLevel())
		}
		if n := len(p.Effects()); n != 0 {
			t.Errorf("the dead hunter has %d effects", n)
		}
		if !core.InArena(p.Position()) {
			t.Error("the dead hunter left the second dimension before it respawned")
		}

		// The respawn: Dragonfly asks the handler where the player goes.
		pos, w := mgl64.Vec3{}, tx.World()
		p.Handler().HandleRespawn(p, &pos, &w)
		if pos.Sub(from).Len() > 1e-6 {
			t.Errorf("the hunter respawns at %v, want where it went in from, %v", pos, from)
		}
		// A second respawn is an island's: the handler leaves the place alone.
		island := mgl64.Vec3{1, 2, 3}
		pos = island
		p.Handler().HandleRespawn(p, &pos, &w)
		if pos != island {
			t.Errorf("a second respawn was sent to %v", pos)
		}
		// Nothing ends the run a second time.
		dimension.Died(s.Session, tx, p)
		s.Leave(p)
		if got := count("wave_fail"); got != 1 {
			t.Errorf("wave_fail counted %d for one lost run, want 1", got)
		}
	})
}

// TestHitThatKillsLeavesNoEffect: a monster whose hit slows the hunter leaves
// no slowness on a hunter its hit killed, which a respawn would bring home.
func TestHitThatKillsLeavesNoEffect(t *testing.T) {
	s, h, _, _ := fallenHunter(t)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		r := dimension.RunOf(s.Session)
		p.Armour().Clear()
		p.SetMaxHealth(1)
		m := world.EntitySpawnOpts{Position: p.Position().Add(mgl64.Vec3{1, 0, 0})}.New(dimension.VariantType(sgitem.MobSlime, dimension.VariantElite),
			dimension.MonsterConfig{Mob: sgitem.MobSlime, Variant: dimension.VariantElite})
		r.Alive[m] = true
		r.Spawned++
		tx.AddEntity(m)
		e, _ := m.Entity(tx)
		for range 200 {
			if p.Dead() {
				break
			}
			e.(*dimension.Monster).Tick(tx, 0)
		}
		if !p.Dead() {
			t.Fatal("the monster did not kill the hunter")
		}
		if n := len(p.Effects()); n != 0 {
			t.Errorf("the dead hunter has %d effects: %v", n, p.Effects())
		}
	})
}

// TestDeadHunterWhoLeavesIsBackOnTheIsland: a hunter that disconnects between
// its death and its respawn is respawned by Dragonfly first and then leaves.
// It is sent where it went in from, the lost run is counted once, and it
// holds what it carried: nothing is left for the next join to give back.
func TestDeadHunterWhoLeavesIsBackOnTheIsland(t *testing.T) {
	s, h, acc, from := fallenHunter(t)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		p.Hurt(1000, entity.AttackDamageSource{})
		if !p.Dead() {
			t.Fatal("the hunter did not die")
		}
		pos, w := mgl64.Vec3{}, tx.World()
		p.Handler().HandleRespawn(p, &pos, &w)
		s.Leave(p)
		if pos.Sub(from).Len() > 1e-6 {
			t.Errorf("the leaving hunter is sent to %v, want %v", pos, from)
		}
		if got := account.Counter(acc, "wave_fail"); got != 1 {
			t.Errorf("wave_fail counted %d, want 1", got)
		}
		// The next join finds nothing put away: the inventory stays as it is.
		before := p.Inventory().Slots()
		s.Join(p, false)
		for slot, st := range p.Inventory().Slots() {
			if !st.Equal(before[slot]) {
				t.Errorf("slot %d holds %v after the next join, was %v", slot, st, before[slot])
			}
		}
		if st, _ := p.Inventory().Item(0); !st.Equal(item.NewStack(block.Stone{}, 3)) {
			t.Errorf("slot 0 holds %v, want the three stone the hunter carried", st)
		}
	})
}

// TestDeathOnTheIslandIsUntouched: a player who is not in the second
// dimension dies as before: the death is counted, the respawn handler leaves
// the place to the platform, and no run is involved.
func TestDeathOnTheIslandIsUntouched(t *testing.T) {
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
		p.Handle(keepsInventory{s.Handler(p)})
		p.Hurt(1000, entity.AttackDamageSource{})
		if !p.Dead() {
			t.Fatal("the player did not die")
		}
		if got, want := [3]int64{account.Counter(acc, "hurt"), account.Counter(acc, "death"), account.Counter(acc, "wave_fail")}, [3]int64{1, 1, 0}; got != want {
			t.Errorf("hurt, death, wave_fail counted %v, want %v", got, want)
		}
		spawn := mgl64.Vec3{1, 2, 3}
		pos, w := spawn, tx.World()
		p.Handler().HandleRespawn(p, &pos, &w)
		if pos != spawn {
			t.Errorf("the island death respawns at %v, want the platform's place %v", pos, spawn)
		}
	})
}

// TestBlastThatKillsDoesNotThrowTheHunter: the blast that kills the hunter
// does not throw it: a dead player takes no knock-back, so none follows the
// hunter to where it respawns.
func TestBlastThatKillsDoesNotThrowTheHunter(t *testing.T) {
	s, h, _, _ := fallenHunter(t)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		r := dimension.RunOf(s.Session)
		p.Armour().Clear()
		p.SetMaxHealth(1)
		c := world.EntitySpawnOpts{Position: p.Position().Add(mgl64.Vec3{1, 0, 0})}.New(dimension.VariantType(sgitem.MobCreeper, dimension.VariantBase),
			dimension.MonsterConfig{Mob: sgitem.MobCreeper})
		r.Alive[c] = true
		r.Spawned++
		tx.AddEntity(c)
		e, _ := c.Entity(tx)
		for range 40 {
			if dimension.RunOf(s.Session) == nil {
				break
			}
			e.(*dimension.Monster).Tick(tx, 0)
		}
		if dimension.RunOf(s.Session) != nil {
			t.Fatal("the blast did not end the run")
		}
		if v := p.Velocity(); v.Len() != 0 {
			t.Errorf("the blast that felled the hunter threw it: velocity %v", v)
		}
		if !p.Dead() {
			t.Error("the hunter did not die of the blast")
		}
	})
}
~~~~

### `platform/instance/death_test.go` (new file)

~~~~go
package instance

import "testing"

// TestDeathKeepsTheInventoryEverywhere: the platform's handler keeps the
// inventory for every death, in the lobby and in an instance, before the
// game's handler runs. The game relies on it: a hunter that dies in the
// second dimension gets what it carried back in the game's death step, and
// nothing of it may drop there.
func TestDeathKeepsTheInventoryEverywhere(t *testing.T) {
	keep := false
	(&Handler{}).HandleDeath(nil, nil, &keep)
	if !keep {
		t.Fatal("a death does not keep the inventory")
	}
}
~~~~

## REF T146

The documents. Prototype patch: `T146.patch` (documents only).

By hand: first every replacement of REF1 "REF T139" (the eight of `docs/tech/extension.md` and the four other files), then the two diffs below, which are relative to the result of those replacements, and the one line of `docs/design/17-dimension.md`. `CONTRIBUTING.md`, `content/skygear/README.md` and `docs/design/03-architecture.md` are exactly REF1's.

What the second step does to `docs/tech/extension.md`:

1. A new section "### 훅과 블록·아이템의 메서드" between "### 구독자와 취소" and "### 이벤트의 값": what Dragonfly calls after the handler, the rule "what a method can do is no subscriber", the examples that exist after T145 (the machine, the machine item, the items used on the air with `sgitem/airuse.go` and why their `Use` returns false), the four cases that stay subscribers, what the guards still refuse (C10, a hunter's use on the air), and the test rule.
2. "### 티어 고르기": one line in front (first see whether it is a subscriber's work at all).
3. "### 순서를 지키는 것": no subscriber stands in a tier that is not its role's; the `UseOnBlock` bullet without `fairy.time` in LOW and without `machine.noteFace`, and a second bullet on Dragonfly's order (the plate before the machine and the item); the `Join` bullet with `dimension.joined` (LOW); the `Hurt` bullet without `dimension.fallen` and a new bullet on `Death` and `Respawn`; the condition bullets (`fairy.time` among the NORMAL subscribers, with why it and the plate never meet).
4. The section "### 역할과 다른 티어" is removed (E4 is resolved by T141).
5. "## 구조를 지키는 규칙": the row on conditions loses the placement face and `TestRefusedClickNotesNoFace`; the row on cancelled things loses the felling hit and `TestFellingHitIsNotAHurt`; the last row is replaced by the rows on the stick record, the mediators, the stranded player, the death (six tests) and the items used on the air (three tests).
6. "## 핀": one sentence on what a scenario cannot see.

`docs/TODO.md`: the item on `dimension.joined` is replaced by the open decisions of section 9.2 (the returned menu item's name, C10, the block clean-up, C17, the blast and the immunity window); the item on `screens.board` says LOW; under "실기 확인" a new subsection has the real-client checks of section 9.1, marked as not yet checked. T147 to T150 each shorten the bullet on the block clean-up.

`docs/design/17-dimension.md`: the line on the hit that would kill ("죽을 만큼 맞으면 …") says the hunter dies and respawns at once, that the client half is not yet checked, and that a lost run counts as a death.

Check: `grep -n 'Always\|orderKept\|noteFace\|역할과 다른\|dimension.fallen\|copier.clear' docs/tech/extension.md` prints nothing; no `—` or `–` in the two files.

### `docs/tech/extension.md` (after the replacements of REF1 "REF T139")

~~~~diff
--- a/docs/tech/extension.md
+++ b/docs/tech/extension.md
@@ -117,6 +117,37 @@ MONITOR 구독자는 이벤트를 가공하지도 취소하지도 않는다. MON
 
 같은 티어에서 한 구독자가 취소하고 다른 기본 구독자가 같은 이벤트에 일할 수 있으면, 도는 순서에 따라 뒤 구독자가 불리기도 하고 불리지 않기도 한다. 그런 쌍은 조건이 겹치지 않게 적거나(든 아이템, 블록 종류, 웅크림, `island.CanBuild`) 역할에 맞게 다른 티어에 둔다.
 
+### 훅과 블록·아이템의 메서드
+
+클릭과 캐기는 훅으로만 오지 않는다. Dragonfly 는 핸들러(훅)를 먼저 부르고, 아무도 취소하지 않으면 눌린 블록에 묻고, 그다음 든 아이템에 묻는다.
+
+| 플레이어가 한 일 | 훅 다음에 Dragonfly 가 부르는 것 |
+|---|---|
+| 블록을 누른다 | 눌린 블록의 `Activate`(`block.Activatable`). 웅크리고 아이템을 들었으면 건너뛴다. 블록이 false 를 돌려주면 든 아이템의 `UseOnBlock`(`item.UsableOnBlock`), 그것도 없으면 든 블록을 놓는다 |
+| 공중을 누른다 | 든 아이템의 `Use`(`item.Usable`) |
+| 블록을 친다 | 블록의 `Punch`(`block.Punchable`). 캐기를 막지는 못한다 |
+| 블록을 다 캔다 | 블록을 지운 뒤 `BreakInfo().BreakHandler`. 폭발과 액체가 블록을 없앨 때도 부른다 |
+| 블록이 바뀐다 | 블록의 `BlockChanged`(`world.BlockChangeHook`). 누가 바꿨든 `SetBlock` 마다 부른다 |
+
+이 메서드로 할 수 있는 일은 구독자로 만들지 않는다. 블록이나 아이템이 이 저장소의 타입이고, 눌린 블록보다 먼저 클릭을 써야 할 까닭이 없으면 메서드에 둔다.
+
+- 기계는 자기를 누른 클릭을 `Machine.Activate` 에서 받는다. 창을 열고, 든 등급 키트(`sgitem.TierKit`)와 요정의 모래시계를 쓴다. 웅크린 플레이어의 모래시계는 `sgitem.FairyHourglass` 의 `UseOnBlock` 이 받는다.
+- 기계 아이템은 `Machine.UseOnBlock` 이 놓는다. 누른 면을 놓이는 블록에 실어 `machine.place` 에 넘기므로 자동 자원 채굴기와 저장망의 입력기·출력기가 누른 블록을 향한다. 도관 아이템과 `hopper.FlowHopper` 도 자기 `UseOnBlock` 으로 놓인다.
+- 서랍을 친 일은 `Drawer.Punch` 가, 도관 묶음이 없어질 때의 정리는 `Bundle.BlockChanged` 가 한다.
+- 공중에서 쓰는 아이템은 자기 `Use` 에서 일한다. 요정 판(`fairy.Board`)은 화면을 열고, 복사기는 비워지고, 건축 지팡이는 웅크리면 설정을 보이고, 철 망치는 웅크리면 대각선 광맥 캐기를 바꾸고, 이차원 텔레포트석은 이차원 창을 연다. 뒤의 넷은 타입이 `sgitem` 에 있어서 기능을 부를 수 없다. `sgitem/airuse.go` 가 아이템마다 함수 변수(`CopierUsed`, `WandUsed`, `HammerUsed`, `StoneUsed`)를 두고, 기능이 `Setup` 에서 채운다. 이 `Use` 는 언제나 false 를 돌려준다. true 를 돌려주면 Dragonfly 가 `Use` 전에 읽어 둔 묶음을 손에 다시 써서 `Use` 가 바꾼 손이 되돌아간다.
+- `item.Usable` 인 커스텀 아이템은 클라이언트 항목에 `minecraft:use_modifiers` 를 받는다(`clientdef.ItemEntry`). 클라이언트는 쓸 수 있다고 아는 아이템만 공중 클릭을 보낸다.
+
+구독자로 두는 경우는 넷이다.
+
+- 가드: 어느 블록을 무엇으로 누르든 플레이어가 있는 곳이나 든 모드를 보고 막는 일(`island.guard`, `expand.guard`, `dimension.hunting`).
+- 눌린 블록보다 먼저 클릭을 써야 하는 든 도구: 복사기와 건축 지팡이(LOW), 메뉴 아이템과 이차원 텔레포트석(LOWEST). Dragonfly 는 블록에 먼저 물으므로 아이템의 메서드로는 상자나 기계 위에서 블록이 먼저 열린다. 공중에는 먼저 물을 블록이 없어서 같은 아이템도 공중 클릭은 메서드로 받는다.
+- 메서드를 달 수 없는 Dragonfly 의 블록과 아이템: 막대기, 도끼, 뼛가루, 작물, 퇴비통. 눌린 블록이 그 기능의 블록이 아닌 경우도 같다. 도관 판은 도관 옆 블록의 면에 그려져 있어서, 판을 누르면 기계나 상자가 눌린다(`conduit.plate`).
+- 기록: 종류를 가리지 않고 세는 일(업적, 자동 채움).
+
+메서드는 구독자가 모두 돈 뒤에 불린다. 그래서 가드가 막은 클릭은 메서드에 닿지 않는다. 섬 끝줄에 놓인 기계의 바깥쪽 면은 옆 칸 가드(`island.guardSide`)가 막으므로 거기서는 기계 창이 열리지 않고 모래시계도 쓰이지 않는다. 사냥꾼의 공중 클릭은 `dimension.hunting` 이 취소하므로 사냥꾼이 든 복사기, 지팡이, 망치, 텔레포트석은 아무 일도 하지 않는다. 섬 확장 모드에는 공중 클릭을 막는 가드가 없다.
+
+장면 골든은 핸들러만 부르므로 메서드가 한 일을 보지 못한다. 메서드로 하는 일은 `Player.UseItemOnBlock`, `Player.BreakBlock` 처럼 Dragonfly 를 거쳐 누르고 캐는 테스트로 확인한다(`hourglass_test.go`, `machineface_test.go`, `airuse_test.go`).
+
 ### 이벤트의 값
 
 이벤트는 값으로 건네므로 `Emit` 이 할당하지 않는다. 필드는 두 가지다.
@@ -346,6 +377,7 @@ MONITOR 구독자는 이벤트를 가공하지도 취소하지도 않는다. MON
 ### 티어 고르기
 
 이름은 한 훅 안에서 겹치면 안 된다(훅이 다르면 같아도 된다). 겹치면 `Install` 이 패닉한다. 티어는 겹쳐도 된다.
+먼저 구독자로 할 일인지 본다. 블록이나 아이템의 메서드로 할 수 있으면 구독자를 달지 않는다(위 "훅과 블록·아이템의 메서드").
 다른 구독자의 자리를 보고 고르지 않는다. 내 구독자가 하는 일을 보고 고른다.
 
 1. 이벤트를 가공하지도 취소하지도 않고 결과를 세거나 보여 주기만 한다: MONITOR. 취소된 이벤트도 봐야 하면 `core.HandleCancelled` 를 준다.
@@ -360,27 +392,29 @@ MONITOR 구독자는 이벤트를 가공하지도 취소하지도 않는다. MON
 
 ### 순서를 지키는 것
 
-지금 순서가 필요한 곳과 그것을 지키는 방법이다. 같은 티어 안의 자리로 지키는 곳은 없다.
+지금 순서가 필요한 곳과 그것을 지키는 방법이다. 같은 티어 안의 자리로 지키는 곳은 없고, 역할에 맞지 않는 티어에 둔 구독자도 없다.
 
 티어가 지키는 것:
 
 - `StartBreak`, `BlockBreak`, `BlockPlace`, `UseOnBlock`: `expand.guard`(LOWEST)가 `island.guard`(LOW)보다 먼저다. 확장 모드에서 섬 밖을 누르면 확장 모드가 막고 지도를 다시 보여 준다. 월드를 바꾸는 단계는 모두 NORMAL 이후다. 가드는 조건이 맞으면 취소하고, 취소된 이벤트는 뒤의 기본 구독자에게 가지 않는다.
 - `BlockBreak`: `storage.front`(LOW)가 서랍 앞면을 친 캐기를 취소한다. 취소된 캐기는 세지 않고 도구 경험치도 광맥 캐기도 없다.
-- `UseOnBlock`: 든 도구가 클릭을 쓰는 단계(`copier.use`, `wand.use`, `fairy.time`, LOW)가 블록에 하는 일(NORMAL, HIGH)보다 먼저다. 요정의 모래시계는 기계에 붙은 도관 판 위를 눌러도 기계를 감는다. 든 아이템을 쓰는 단계(`handwork.stick`, NORMAL)가 블록 자체의 동작(`farm.composter`, `farm.crop`, HIGH)보다 먼저다. 옆 칸 가드(`expand.guardSide`, `island.guardSide`, HIGHEST)는 아무도 취소하지 않았을 때 마지막에 막는다. `machine.noteFace` 와 `handwork.stickOther`(MONITOR)는 취소되지 않고 끝까지 간 클릭만 받는다.
+- `UseOnBlock`: 든 도구가 클릭을 쓰는 단계(`copier.use`, `wand.use`, LOW)가 블록에 하는 일(NORMAL, HIGH)보다 먼저다. 든 아이템을 쓰는 단계(`handwork.stick`, `fairy.time`, NORMAL)가 블록 자체의 동작(`farm.composter`, `farm.crop`, HIGH)보다 먼저다. 옆 칸 가드(`expand.guardSide`, `island.guardSide`, HIGHEST)는 아무도 취소하지 않았을 때 마지막에 막는다. `handwork.stickOther`(MONITOR)는 취소되지 않고 끝까지 간 클릭만 받는다.
+- `UseOnBlock` 의 구독자가 모두 돈 뒤에 Dragonfly 가 눌린 블록의 `Activate` 를, 그다음 든 아이템의 `UseOnBlock` 을 부른다. 서서 도관 판을 누르면 `conduit.plate` 가 취소하므로 판 뒤의 기계는 열리지 않고, 든 등급 키트와 요정의 모래시계도 쓰이지 않는다. 웅크리면 판이 클릭을 쓰지 않아 모래시계가 기계를 감는다.
 - `Sneak`: `expand.show`, `elevator.down`(NORMAL), `farm.crouchGrow`(HIGH), `achievement.sneak`(MONITOR) 순서다. 웅크리기는 아무도 취소하지 않으므로 모두 불린다. 엘리베이터가 플레이어를 옮긴 뒤에 `farm.crouchGrow` 가 플레이어 둘레의 묘목을 본다.
-- `Hurt`: `dimension.frozen`(LOWEST)이 먼저 취소하고, `dimension.fallen`(HIGHEST)이 남은 피해로 사냥꾼이 쓰러지는지 보고 취소한다. `Move` 의 `island.void`(HIGHEST)도 마지막에 취소한다. 취소된 피해와 이동은 `achievement.hurt`, `achievement.walk`(MONITOR)가 세지 않는다.
+- `Hurt`: `dimension.frozen`(LOWEST)이 강화를 고르는 사냥꾼의 피해를 취소한다. 사냥꾼을 죽이는 타격은 막지 않는다. 사냥꾼은 실제로 죽고 `achievement.hurt` 가 그 타격을 센다. `Move` 의 `island.void`(HIGHEST)는 마지막에 취소한다. 취소된 피해와 이동은 `achievement.hurt`, `achievement.walk`(MONITOR)가 세지 않는다.
+- `Death`, `Respawn`: 사냥꾼이 죽으면 `dimension.died`(NORMAL)가 판을 실패로 끝낸다. 맡겨 둔 소지품과 보상이 그때 돌아오고 `achievement.death`(MONITOR)가 죽음을 센다. 부활할 때 `dimension.respawned`(LOW)가 부활할 자리를 들어왔던 자리로 고친 뒤에 `tinker.restoreFood`(NORMAL)가 허기를 되돌린다. 인벤토리는 플랫폼의 핸들러가 모든 죽음에서 지킨다(`platform/instance` 의 `HandleDeath`).
 - `ItemDamage`: `tinker.ironHammer`(LOW)가 피해를 0 으로 고친 뒤에 `autofill.broken`(HIGH)이 읽는다.
 - `BlockPlace`: `netstore.place`(HIGH)는 `machine.place`(NORMAL)가 블록 설치를 미뤄 둔 뒤에 망을 버린다.
 - `Leave`: 기능이 퇴장 때 하는 일(`dimension.leave`, `expand.stop`, NORMAL)이 클라이언트에 남은 것을 지우는 정리(`look.clear`, `screens.board`, `progress.recipes`, HIGHEST)보다 먼저다. 확장 모드가 끝나면 사이드바를 다시 그리고, 사냥이 끝나면 레시피를 다시 보낼 수 있다.
-- `Join`: `screens.menuItem`(LOWEST)이 처음 들어온 플레이어의 8번 칸에 메뉴 아이템을 둔 뒤에 아이템을 주는 입장 단계(`fairy.backfillDust`, NORMAL)가 돈다. 쌓아 둔 요정 가루가 여덟 묶음을 넘어도 메뉴 아이템이 가루를 덮어쓰지 않는다.
+- `Join`: `screens.menuItem`(LOWEST)이 처음 들어온 플레이어의 8번 칸에 메뉴 아이템을 둔 뒤에 아이템을 주는 입장 단계(`fairy.backfillDust`, NORMAL)가 돈다. 쌓아 둔 요정 가루가 여덟 묶음을 넘어도 메뉴 아이템이 가루를 덮어쓰지 않는다. `dimension.joined`(LOW)도 그 단계보다 먼저다. 서버가 멈출 때 이차원에 있던 플레이어는 맡겨 둔 소지품을 먼저 돌려받고, 가루는 돌려받은 인벤토리에 들어간다.
 - `Second`: `progress.checkHeld`(LOW)가 플레이어가 든 아이템을 읽어 파트에 둔 뒤에 `dimension.second`(NORMAL)가 `RetireItems` 로 아이템을 정리한다. `progress.collection`(MONITOR)이 그 파트를 읽는다.
 
 구독자의 조건이 지키는 것:
 
 - 사냥 중인 플레이어가 블록을 누르면 무엇을 들었든 `dimension.hunting` 이 취소한다. `screens.menuItem`(`UseOnBlock`)과 `dimension.stone` 은 사냥 중이면 클릭을 쓰지 않는다. 공중에서는 메뉴 아이템이 사냥에서 나가는 길이라 `dimension.hunting`(`Use`)이 메뉴 아이템을 든 클릭을 취소하지 않는다.
 - 메뉴 아이템과 이차원석은 어디서나 자기 화면을 연다. `expand.guard`(`UseOnBlock`)는 그런 아이템(`sgitem.OpensScreen`)을 든 클릭을 취소하지 않는다.
-- `copier.use`, `wand.use`, `fairy.time` 은 섬이 짓기를 허락하는 곳(`island.CanBuild`)에서만 클릭을 쓴다. 섬 밖에서는 `island.guard` 가 취소한다. `storage.front` 도 같은 조건을 단다.
-- `UseOnBlock` 의 NORMAL 구독자는 블록과 든 아이템과 웅크림으로 갈린다. `conduit.plate` 는 서서 누른 도관 판, `storage.sneakUse` 는 웅크리고 누른 서랍, `handwork.stick` 은 막대기로 누른 흙·잔디와 웅크리고 누른 상자, `handwork.gather` 는 도끼로 누른 원목과 괭이로 누른 흙이다.
+- `copier.use`, `wand.use` 는 섬이 짓기를 허락하는 곳(`island.CanBuild`)에서만 클릭을 쓴다. 섬 밖에서는 `island.guard` 가 취소한다. `storage.front` 도 같은 조건을 단다.
+- `UseOnBlock` 의 NORMAL 구독자는 블록과 든 아이템과 웅크림으로 갈린다. `conduit.plate` 는 서서 누른 도관 판, `storage.sneakUse` 는 웅크리고 누른 서랍, `handwork.stick` 은 막대기로 누른 흙·잔디와 웅크리고 누른 상자, `handwork.gather` 는 도끼로 누른 원목과 괭이로 누른 흙, `fairy.time` 은 요정의 뼛가루로 누른 작물·묘목·마법 식물과 웅크리고 뼛가루로 누른 작물이다. 도관 판은 아이템이나 에너지, 액체를 주고받는 블록에만 붙으므로 `conduit.plate` 와 `fairy.time` 이 같은 클릭에 일하는 때는 없다.
 - `Sneak` 의 `expand.show` 는 섬 확장 모드에서만 화면을 다시 보여 준다. `elevator.down` 과 `farm.crouchGrow` 는 다른 모드가 플레이어를 붙잡고 있으면(`core.Takeover`) 일하지 않는다.
 - `handwork.stickOther` 는 막대기로 흙도 잔디도 아닌 블록을 눌렀고 아무도 그 클릭을 취소하지 않았을 때 `stick_other` 를 센다. 서랍 메뉴, 도관 판, 작물, 가득 찬 퇴비통이 쓴 클릭은 세지 않는다.
 
@@ -391,12 +425,6 @@ MONITOR 구독자는 이벤트를 가공하지도 취소하지도 않는다. MON
 - `Move`: 이벤트의 `From` 은 이벤트를 만들 때 플레이어가 서 있던 곳이다. 앞의 구독자가 플레이어를 옮겨도 `achievement.walk` 는 같은 거리를 센다.
 - 커널의 구독자는 `Join` 의 `core.shapes` 하나다(NORMAL). 다른 구독자와 같은 규칙을 따른다.
 
-### 역할과 다른 티어
-
-역할에 맞지 않는 티어에 둔 구독자는 하나다. 예전 동작을 지키려고 둔 것이고, 목록은 늘리지 않는다.
-
-- `Join` 의 `dimension.joined`(HIGH): 서버가 멈출 때 사냥 중이던 플레이어의 소지품을 되돌리는 일이라 역할로는 다른 입장 단계보다 먼저(LOW)다. 지금 순서를 지키려고 뒤에 둔다. 앞의 입장 단계가 그 플레이어의 인벤토리에 넣거나 고친 것(`fairy.backfillDust` 의 가루, `screens.menuItem` 이 고친 메뉴 아이템 이름)은 사냥 장비와 함께 사라진다.
-
 ## 기능의 데이터
 
 기능은 `Session` 에 필드를 더하지 않고 `core` 의 파트를 패키지 변수로 둔다.
@@ -512,9 +540,13 @@ func init() { coretest.Setup(farm.Feature) }
 | 취소된 이벤트는 기본 구독자에게 가지 않고 `core.HandleCancelled` 구독자에게만 간다. 취소는 Dragonfly 의 컨텍스트 하나다 | `core.TestCancelSkipsDefaultSubscribers`, `core.TestAfterCancelOnlyHandleCancelledRuns`, `core.TestCancelIsDragonflysCancel`, `core.TestCancelledTellsTheContext` |
 | MONITOR 구독자는 취소하지도 가공하지도 않는다 | `TestMonitorSubscribersDoNotShape`, `core.TestMonitorCancelPanicsWhileTraced` |
 | `Leave` 의 구독자가 모두 돈 뒤에 플레이어별 파트 항목을 지운다 | `core.TestLeaveDropsThePlayersEntriesLast` |
-| 구독자의 조건이 지키는 것: 사냥꾼의 이차원석, 섬 밖의 복사기와 지팡이, 모드에 붙잡힌 플레이어의 웅크리기, 끝까지 간 클릭만 적는 놓을 면 | `TestHunterStoneOpensNothing`, `TestToolsLeaveTheClickOutsideTheIsland`, `elevator.TestHeldPlayerDoesNotRideDown`, `farm.TestHeldPlayerGrowsNoSapling`, `TestRefusedClickNotesNoFace` |
-| 취소된 일은 일어나지 않은 일이다: 공허로 떨어진 걸음, 사냥꾼을 쓰러뜨린 타격, 서랍 앞면을 친 캐기는 세지 않는다. 웅크리기는 언제나 센다 | `TestMoveIntoTheVoidIsNotWalked`, `TestFellingHitIsNotAHurt`, `TestDrawerFrontHitIsNoBreak`, `TestEverySneakCounts`, `TestSneakRideGrowsSaplingsWhereItArrives` |
-| 든 도구가 블록보다 먼저 클릭을 쓴다. 막대기로 다른 블록을 두드린 기록은 아무도 쓰지 않은 클릭만 센다 | `TestHourglassWindsTheMachineBehindAPlate`, `TestStickOtherCountsTheTapNothingTook` |
+| 구독자의 조건이 지키는 것: 사냥꾼의 이차원석, 섬 밖의 복사기와 지팡이, 모드에 붙잡힌 플레이어의 웅크리기 | `TestHunterStoneOpensNothing`, `TestToolsLeaveTheClickOutsideTheIsland`, `elevator.TestHeldPlayerDoesNotRideDown`, `farm.TestHeldPlayerGrowsNoSapling` |
+| 취소된 일은 일어나지 않은 일이다: 공허로 떨어진 걸음과 서랍 앞면을 친 캐기는 세지 않는다. 웅크리기는 언제나 센다 | `TestMoveIntoTheVoidIsNotWalked`, `TestDrawerFrontHitIsNoBreak`, `TestEverySneakCounts`, `TestSneakRideGrowsSaplingsWhereItArrives` |
+| 사냥꾼을 죽이는 타격은 취소하지 않는다. 사냥꾼은 죽고, 판은 한 번 실패로 끝나고, 소지품과 경험치는 그대로이고, 부활하면 들어왔던 자리로 간다. 죽음과 부활 사이에 나가도 같다. 섬에서 죽는 일은 달라지지 않는다 | `TestHunterDiesAndComesBack`, `TestDeadHunterWhoLeavesIsBackOnTheIsland`, `TestHitThatKillsLeavesNoEffect`, `TestBlastThatKillsDoesNotThrowTheHunter`, `TestDeathOnTheIslandIsUntouched`, `instance.TestDeathKeepsTheInventoryEverywhere` |
+| 공중에서 쓰는 아이템은 `item.Usable` 이고 자기 `Use` 에서 일한다. 사냥꾼의 공중 클릭은 아무 일도 하지 않는다 | `TestAirItemsAreUsable`, `TestItemsUsedOnTheAir`, `TestHunterUsesNothingOnTheAir` |
+| 막대기로 다른 블록을 두드린 기록은 아무도 취소하지 않은 클릭만 센다 | `TestStickOtherCountsTheTapNothingTook` |
+| 블록과 아이템의 메서드로 하는 일은 Dragonfly 를 거친 클릭으로 확인한다. 요정의 모래시계는 기계가 쓰고, 도관 판과 옆 칸 가드가 그보다 먼저다. 기계 아이템은 누른 면을 자기 설치에 넘기고, 클릭 없이 놓인 기계는 기본 방향을 본다 | `TestHourglassThroughTheMachine`, `TestHourglassOnTheOutwardFaceIsRefused`, `machine.TestHourglassIsUsedByTheMachine`, `TestMachineItemPlacesAndFaces`, `TestMachinePlacedWithoutAClickFacesItsDefault` |
+| 서버가 멈출 때 이차원에 있던 플레이어는 입장 때 소지품을 먼저 돌려받고, 그 뒤에 받은 요정 가루를 지닌다 | `TestStrandedPlayerKeepsBackfilledDust` |
 
 핫 경로(이동, 매초, 빠른 틱)를 건드리면 벤치마크로 재 본다. 이벤트 하나의 비용은 `core` 의 `BenchmarkEmit`, `BenchmarkEmitMove` 와 루트의 `BenchmarkHandleMove`, `BenchmarkSecond`, `BenchmarkUseOnBlock`, `BenchmarkUse`, `BenchmarkBlockBreak`, `BenchmarkBlockPlace` 가 잰다. 구독자는 취소되지 않은 이벤트마다 모두 불리므로, 조건은 싼 것부터 본다(든 아이템의 종류, 그다음 맵 조회나 블록 읽기). 패널 하나의 비용은 루트의 `BenchmarkLookMachine`(시선 추적과 패널 보내기), `BenchmarkLookInfoMachine`, `BenchmarkLookInfoStone`(패널 만들기)이 잰다. 다른 기능이 동작을 등록한 기계의 틱 한 번은 `BenchmarkTickHarvester`, `BenchmarkTickDrawerController` 가 잰다.
 
@@ -552,6 +584,7 @@ go test ./content/skygear -run 'TestOldSaveLoads' -count=1   # 옛 세이브 읽
 골든이 달라지면 테스트가 어느 줄(장면)이 어떻게 달라졌는지 `want` 와 `got` 로 보여 준다. 디스패치 순서나 클라이언트 데이터가 바뀌면 골든 차이로 나타난다.
 
 장면 골든의 한 줄은 `<장면>: <거친 구독자 이름> | <결과>` 다. `|` 앞은 등록한 순서로 돌았을 때의 자취이고 뒤가 결과다.
+장면은 핸들러만 부른다. 블록의 `Activate` 나 아이템의 `UseOnBlock` 처럼 Dragonfly 가 핸들러 뒤에 부르는 메서드가 한 일은 장면에 나오지 않는다(위 "훅과 블록·아이템의 메서드").
 `TestTierOrder` 는 루트의 테스트 바이너리를 여섯 번 다시 돌린다. 한 번은 등록한 순서 그대로, 다섯 번은 모든 훅의 구독자를 티어 안에서 뒤집거나 섞어서다(`SKYGEAR_HOOK_ORDER=reverse`, `shuffle:1` 부터 `shuffle:4`). 섞은 실행에서 확인하는 것은 셋이다.
 
 - 루트의 모든 테스트가 통과한다. 장면 골든은 `|` 뒤의 결과만 견준다.
~~~~

### `docs/TODO.md` (after the replacement of REF1 "REF T139")

~~~~diff
--- a/docs/TODO.md
+++ b/docs/TODO.md
@@ -18,14 +18,18 @@
 ### 정리
 - Sky Gear 이름 변경 세이브 변환(`content/skygear/migrate/namespace.go`와 그 테스트)과 그 호출(`cmd/skygear/main.go`)을 모든 서버가 한 번씩 돌린 뒤 지운다. 데이터 폴더의 `skygear-namespace.done` 플래그와 `<데이터 폴더>.before-skygear` 사본도 함께 지운다.
 - `UseOnBlock` 의 두 번째 확장 모드 검사 `expand.guardSide`(HIGHEST)는 일할 때가 없다. 확장 모드의 클릭은 `expand.guard`(LOWEST)나 화면을 여는 아이템의 구독자가 이미 취소한다. 구독자를 지우거나 옆 칸 검사가 의미를 갖게 고친다. 지우면 `order.golden` 과 `handler-use.golden` 에서 그 줄이 빠진다.
-- 훅에 남은 결정([extension.md](tech/extension.md) "구독자와 취소", "역할과 다른 티어"). 정하면 플레이어가 보는 동작이 바뀌므로 테스트를 더하고 장면 골든을 다시 기록한다.
-  - `Join` 의 `dimension.joined`(HIGH): 역할대로 LOW 로 옮긴다. 서버가 멈출 때 사냥 중이던 플레이어가 소급 요정 가루를 잃지 않는다. 메뉴 아이템 이름은 그대로 남는다(`screens.menuItem` 이 LOWEST 에서 먼저 돈다).
-  - 취소가 필요 없는데 취소하는 구독자: 공중에서 연 메뉴 아이템과 이차원석(`Use` 의 `screens.menuItem`, `dimension.stone`), 흙·잔디를 두드린 막대기(`handwork.stick`). 취소하지 않게 바꾸면 장면 골든의 `cancelled=` 가 달라지고, 막대기는 섬 가장자리에서 옆 칸 가드의 "섬 밖" 팁이 함께 뜬다.
+- 훅에 남은 결정([extension.md](tech/extension.md) "구독자와 취소", "훅과 블록·아이템의 메서드"). 정하면 플레이어가 보는 동작이 바뀌므로 테스트를 더하고 장면 골든을 다시 기록한다.
+  - 서버가 멈출 때 이차원에 있던 플레이어가 돌려받는 메뉴 아이템은 옛 이름 그대로다. `screens.menuItem`(LOWEST)이 `dimension.joined`(LOW)보다 먼저 돌아 이름을 고친 뒤에 소지품이 돌아오기 때문이다. 1초 안에 고쳐지는지, 입장 때 고칠지 정한다.
+  - 섬 끝줄에 놓인 기계의 바깥쪽 면에서는 요정의 모래시계가 쓰이지 않는다. 옆 칸 가드(`island.guardSide`)가 그 클릭을 막는다. 그 면에서 기계 창이 열리지 않는 것과 같다. 그대로 둘지 정한다.
+  - 블록이 없어질 때의 정리(`BlockBreak` 의 `machine.break`, `storage.break`, `netstore.break`, `conduit.break`)는 플레이어가 캘 때만 돈다. 폭발로 없어진 기계의 창, 서랍의 표시, 저장망 노드의 설정은 남는다. 블록의 `BreakHandler` 로 옮기면 폭발에도 정리된다. `conduit.break` 는 `Bundle.BlockChanged` 가 같은 일을 이미 하므로 지우면 된다.
+  - 이차원에서 죽으면 `death` 업적도 센다(진 판마다 한 번). 세지 않으려면 `achievement.death` 가 `core.InArena(e.P.Position())` 인 죽음을 건너뛰게 한다.
+  - 폭발 공격은 피해를 주지 못했을 때도(맞은 직후의 무적 시간) 사냥꾼을 밀어낸다. 근접 공격은 피해가 들어갔을 때만 민다. 맞출지 정한다.
+  - 취소가 필요 없는데 취소하는 구독자: 공중에서 연 메뉴 아이템(`Use` 의 `screens.menuItem`), 흙·잔디를 두드린 막대기(`handwork.stick`). 취소하지 않게 바꾸면 장면 골든의 `cancelled=` 가 달라지고, 막대기는 섬 가장자리에서 옆 칸 가드의 "섬 밖" 팁이 함께 뜬다.
   - `stick_other` 는 아무도 취소하지 않은 클릭만 센다. 막대기로 누른 다른 블록을 모두 세려면 `handwork.stickOther` 에 `core.HandleCancelled` 를 주고 `core.Takeover` 조건을 단다(섬 밖, 서랍 메뉴, 도관 판, 작물을 누른 것도 센다).
   - `Jump` 의 `elevator.up` 은 모드에 붙잡힌 플레이어도 태운다. `elevator.down` 처럼 `core.Takeover` 조건을 달지 정한다.
   - 섬 가장자리의 원목을 섬 밖을 향한 면으로 누르면 `handwork.gather`(NORMAL)가 섬유를 떨군 뒤 `island.guardSide`(HIGHEST)가 클릭을 취소해 원목이 벗겨지지 않는다. 되풀이할 수 있다. `handwork.gather` 가 끝까지 간 클릭만 받게 옮길지 정한다.
 - 티어 안 순서 가드(`TestTierOrder`)는 장면의 결과, 업적 카운터, 장면의 플레이어가 든 아이템을 견준다. 채팅·팁·소리·패킷, 장면이 닿지 않는 상태는 보지 못한다. 첫 접속 때 메뉴 아이템이 소급 요정 가루를 덮어쓰던 문제(`TestFirstJoinKeepsBackfilledDust`)는 그런 장면이 없어서 빠져나갔다. 인벤토리를 건드리는 접속·퇴장 구독자마다 장면을 더한다.
-- `Join` 의 `screens.board`(MONITOR)가 이제 `dimension.joined`(HIGH) 뒤에 돈다. 서버가 멈출 때 사냥 중이던 플레이어의 퀘스트 비용 줄이 다음 갱신까지(1초 안) 장비가 아니라 되돌린 소지품을 센다. 그대로 둘지 정한다.
+- `Join` 의 `screens.board`(MONITOR)는 `dimension.joined`(LOW) 뒤에 돈다. 서버가 멈출 때 이차원에 있던 플레이어의 퀘스트 비용 줄이 처음부터 되돌린 소지품을 센다.
 - 같은 MONITOR 안에서는 채팅 순서가 설치 순서를 따른다(`achievement.place`·`conduit.place`·드래곤 알림, `playSeconds` 와 수집 보상). 기록을 MONITOR 로 옮기면서 업적 메시지가 다른 구독자의 메시지 뒤로 간 곳이 있다(`Death`, `BlockPlace`). 순서가 중요한 메시지가 있으면 한 구독자로 묶는다.
 - MONITOR 가드(`TestMonitorSubscribersDoNotShape`)는 구독자 함수 본문만 읽는다. 도우미 안의 취소는 장면 테스트에서 `Emit` 이 잡지만(추적 중일 때 패닉), 도우미 안에서 이벤트의 포인터에 쓰는 경우와 포인터를 변수에 옮겨 쓰는 경우는 보지 못한다. 지금의 MONITOR 구독자는 읽어서 확인했다. 가드를 넓히거나 MONITOR 구독자에 읽기 전용 이벤트를 넘긴다.
 - `TestHuntClear`(`content/skygear/dimension/dimension_test.go`)와 `TestKitAndLevels` 가 `TestHuntFails` 와 같은 시간 의존 방식이라 `-race` 에서 `TestHuntFails` 처럼 불안정해질 수 있다. 같은 방식으로 시간에 기대지 않게 고친다.
@@ -127,6 +131,17 @@
 - 서버를 끄면 접속한 클라이언트에 종료 안내 글이 뜬 채로 끊기는지.
 - 명령이 내부 오류로 실패할 때: 오류 줄이 한 줄 뜨고 그 앞에 빈 줄이 생기지 않는지(클라이언트가 Dragonfly 의 명령 출력 패킷과 오류 패킷을 차례로 받는다).
 
+### 이차원의 죽음과 공중에서 쓰는 아이템 (아직 확인하지 않음)
+서버 쪽은 테스트로 확인했다. 아래는 실제 클라이언트에서만 볼 수 있고, 아직 아무도 보지 않았다.
+- 이차원에서 죽었을 때 죽음 화면 없이 바로 부활하는지. 판이 시작될 때 그 플레이어에게만 `doimmediaterespawn` 게임 규칙을 보내고 판이 끝나거나 부활하면 되돌린다(`Player.EnableInstantRespawn`). 접속 도중에 받은 규칙을 클라이언트가 곧바로 따르는지, 죽고 나서 섬에 서기까지 얼마나 걸리는지 잰다.
+- 클라이언트가 규칙을 따르지 않으면 판에서 질 때마다 죽음 화면이 뜬다. 그때도 판은 이미 끝났고 보상은 들어와 있다. 부활 버튼을 누르면 들어왔던 자리로 오는지 본다.
+- 마지막 타격의 붉은 번쩍임과 맞는 소리, 죽는 모습이 보이는지. 부활해 섬으로 넘어오는 화면이 어색하지 않은지.
+- 부활한 자리에서 청크가 오기 전에 떨어지지 않는지(부활 뒤 `instance.PlaceAt` 이 땅 위에 붙잡는다). 결과 창이 0.5초 뒤에 뜨는지.
+- 판이 끝난 뒤 섬에서 죽으면 예전처럼 죽음 화면이 뜨는지(규칙이 꺼졌는지).
+- 죽음 화면에서 접속을 끊고 다시 들어오면 섬에 있고 소지품이 그대로인지.
+- 복사기, 건축 지팡이, 망치가 `minecraft:use_modifiers`(`use_duration` 0.05, `movement_modifier` 1)를 새로 받는다. 이 아이템을 든 공중 클릭이 서버에 오는지(전에는 오지 않았을 수 있다), 누르고 있을 때 쓰는 모습이나 느려짐이 생기는지, 망치로 캐거나 지팡이로 짓는 느낌이 달라지지 않는지.
+- 공중 클릭으로 복사기가 비워지고, 웅크린 지팡이가 설정을 보이고, 웅크린 철 망치가 대각선 캐기를 바꾸는지. 버튼을 누르고 있어도 한 번만 되는지(0.5초 안의 되풀이는 무시한다).
+
 ### 패치로 남긴 것 (대체할 수 있는지 판단)
 - 섬 확장 지도의 가짜 청크(`ViewFakeChunk`): 지도가 제대로 그려지고 닫으면 실제 월드가 돌아오는지, 그동안 월드 블록 변화가 지도에 끼지 않는지.
 - 블록을 눌러 `Activate` 가 성공한 뒤 주변 블록 재전송: 블록을 든 채로 서랍·기계 같은 블록을 눌러도 손 앞에 유령 블록이 남지 않는지.
~~~~

### `docs/design/17-dimension.md`

~~~~diff
--- a/docs/design/17-dimension.md
+++ b/docs/design/17-dimension.md
@@ -66,7 +66,7 @@
    - **클리어**: 제한 시간 안에 처치 수를 모두 잡는다. 걸린 시간이 기록이 된다.
    - **실패**: 죽거나 시간이 다 된다. 기록은 남지 않지만 잡은 수는 누적 처치에 더한다.
    - 어느 쪽이든 결과 창을 보이고 섬의 입장했던 자리로 돌려보낸다. **아이템은 잃지 않는다.**
-   - 죽을 만큼 맞으면 죽지 않고(부활 화면 없이) 그 자리에서 실패로 끝나 체력을 채우고 바로 돌아온다.
+   - 죽을 만큼 맞으면 실제로 죽는다. 판은 그때 실패로 끝나고, 부활하면 체력이 찬 채로 입장했던 자리에 선다. 판 동안에는 그 플레이어에게만 즉시 부활 규칙을 보내 죽음 화면 없이 바로 부활하게 한다(실제 클라이언트에서 아직 확인하지 않았다). 진 판은 죽음 업적에도 한 번 센다.
    - 사냥하는 동안 화면 아래에 처치 수·남은 시간과 지금까지 얻은 에센스(첫 클리어면 그 보너스)를 보인다. 에센스는 처치할 때마다 바로 정해진다.
    - 판 도중에 나가기(메뉴 → 섬으로)는 실패로 친다.
 
~~~~

## REF T147

A broken bundle is cleaned up by its own `BlockChanged` alone. Prototype patch: `T147.patch`.

No behaviour change. `conduit.break` called `RemoveVisual` just before the block was removed; `Bundle.BlockChanged` calls the same `RemoveVisual` when the block is removed (`conduit/events.go`), for a break by hand and for every other `SetBlock`. The subscriber is deleted.

`blockbroken_test.go` is new and is the file of plan E's tests: its helpers (`brokenSession`, `blowUp`) and `TestBrokenBundleLosesItsDisplay`, which breaks a bundle with `Player.BreakBlock` and blows one up, and reads the display and the plates. It passes before and after the change (the subscriber was redundant). That it guards the mediator is shown by mutation, in a scratch worktree: with the `RemoveVisual` line taken out of `BlockChanged`, the test fails after this task for both cases (`broken by hand: the bundle's display is still there`), and before this task only for `blown up` (the subscriber hid the hand break).

`pin_blocks_test.go`: `conduit.break` leaves the row of `BlockBreak/stone`. `docs/TODO.md`: `conduit.break` leaves the bullet on the block clean-up.

Goldens: re-record `^TestPin(Order|Blocks)$`. `order.golden`: `BlockBreak NORMAL conduit.break` goes. The check script prints `handler-blocks: 33 scenarios, 14 traces changed` and no `RESULT`. The sorted dump equals that of the state before.

### `content/skygear/blockbroken_test.go` (new file)

~~~~go
package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/itemnet"
	"github.com/presentkim/skygear/content/skygear/conduit"
	"github.com/presentkim/skygear/content/skygear/island"
)

// What a block leaves behind when it goes is cleaned up by the block's own
// callbacks, which Dragonfly calls after the handler's subscribers. The pin
// scenarios call the handler only, so these tests remove the block through
// Dragonfly: a player's finished break (Player.BreakBlock) and an explosion.

// brokenSession is a session with a player who stands next to pos and may
// break it.
func brokenSession(t *testing.T, pos cube.Pos) (*session, *world.EntityHandle) {
	t.Helper()
	s := testSession(t)
	island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		p.Handle(s.Handler(p))
		p.Teleport(pos.Vec3Centre().Add(mgl64.Vec3{0, 1, 2}))
	})
	return s, h
}

// blowUp removes the blocks around pos as TNT that goes off beside it does.
func blowUp(tx *world.Tx, pos cube.Pos) {
	block.ExplosionConfig{}.Explode(tx, world.BlockExplosionSource{Pos: pos.Side(cube.FaceUp), ExplosionSize: 4})
}

// TestBrokenBundleLosesItsDisplay: a conduit bundle a player breaks loses its
// display and its connector plates, and so does one an explosion removes.
// No subscriber of the break does it: the bundle's BlockChanged does.
func TestBrokenBundleLosesItsDisplay(t *testing.T) {
	chest := cube.Pos{4, island.IslandY + 2, 4}
	for _, c := range []struct {
		name   string
		remove func(p *player.Player, tx *world.Tx, at cube.Pos)
	}{
		{"broken by hand", func(p *player.Player, _ *world.Tx, at cube.Pos) { p.BreakBlock(at) }},
		{"blown up", func(_ *player.Player, tx *world.Tx, at cube.Pos) { blowUp(tx, at) }},
	} {
		s, h := brokenSession(t, chest.Side(cube.FaceEast))
		withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
			at := chestWithBundle(tx, chest, itemnet.Input, 5, 9)
			l := conduit.LocalOf(s.Session)
			if _, ok := l.Visuals[at]; !ok {
				t.Fatalf("%s: the bundle has no display", c.name)
			}
			if len(l.Plates) == 0 {
				t.Fatalf("%s: the bundle has no connector plate", c.name)
			}
			c.remove(p, tx, at)
			if _, still := tx.Block(at).(conduit.Bundle); still {
				t.Fatalf("%s: the bundle is still there", c.name)
			}
			if _, ok := l.Visuals[at]; ok {
				t.Errorf("%s: the bundle's display is still there", c.name)
			}
			if len(l.Plates) != 0 {
				t.Errorf("%s: %d connector plates are still there", c.name, len(l.Plates))
			}
		})
	}
}
~~~~

### `content/skygear/conduit/subscribe.go`

~~~~diff
--- a/content/skygear/conduit/subscribe.go
+++ b/content/skygear/conduit/subscribe.go
@@ -31,14 +31,10 @@ func subscribeShares() {
 	})
 }
 
-// A conduit bundle breaking loses its visual; a placed one counts (a record:
-// Monitor).
+// A placed conduit bundle counts (a record: Monitor). A bundle that goes
+// needs no subscriber: Bundle.BlockChanged takes it out of the grids and
+// removes its display, whoever removed it.
 func subscribeBlocks() {
-	core.BlockBreak.On(core.Normal, "conduit.break", func(e core.BlockBreakEvent) {
-		if _, ok := e.Block.(Bundle); ok {
-			RemoveVisual(e.S, e.Tx, e.Pos)
-		}
-	})
 	core.BlockPlace.On(core.Monitor, "conduit.place", func(e core.BlockPlaceEvent) {
 		if _, ok := e.Block.(Bundle); ok {
 			achievement.Count(e.P, "conduit_place", 1)
~~~~

### `content/skygear/pin_blocks_test.go`

~~~~diff
--- a/content/skygear/pin_blocks_test.go
+++ b/content/skygear/pin_blocks_test.go
@@ -277,7 +277,7 @@ func TestPinBlocks(t *testing.T) {
 	// reorders the subscribers (TestTierOrder) has other traces.
 	rows := map[string]string{
 		"StartBreak/own":   "expand.guard island.guard tinker.autoTool",
-		"BlockBreak/stone": "expand.guard island.guard storage.front tinker.break machine.break conduit.break storage.break farm.break netstore.break achievement.break",
+		"BlockBreak/stone": "expand.guard island.guard storage.front tinker.break machine.break storage.break farm.break netstore.break achievement.break",
 		"BlockPlace/dirt":  "expand.guard island.guard machine.place storage.place farm.plant autofill.place netstore.place achievement.place progress.dragonEgg conduit.place",
 	}
 	for name, row := range rows {
~~~~

### `docs/TODO.md`

~~~~diff
--- a/docs/TODO.md
+++ b/docs/TODO.md
@@ -21,7 +21,7 @@
 - 훅에 남은 결정([extension.md](tech/extension.md) "구독자와 취소", "훅과 블록·아이템의 메서드"). 정하면 플레이어가 보는 동작이 바뀌므로 테스트를 더하고 장면 골든을 다시 기록한다.
   - 서버가 멈출 때 이차원에 있던 플레이어가 돌려받는 메뉴 아이템은 옛 이름 그대로다. `screens.menuItem`(LOWEST)이 `dimension.joined`(LOW)보다 먼저 돌아 이름을 고친 뒤에 소지품이 돌아오기 때문이다. 1초 안에 고쳐지는지, 입장 때 고칠지 정한다.
   - 섬 끝줄에 놓인 기계의 바깥쪽 면에서는 요정의 모래시계가 쓰이지 않는다. 옆 칸 가드(`island.guardSide`)가 그 클릭을 막는다. 그 면에서 기계 창이 열리지 않는 것과 같다. 그대로 둘지 정한다.
-  - 블록이 없어질 때의 정리(`BlockBreak` 의 `machine.break`, `storage.break`, `netstore.break`, `conduit.break`)는 플레이어가 캘 때만 돈다. 폭발로 없어진 기계의 창, 서랍의 표시, 저장망 노드의 설정은 남는다. 블록의 `BreakHandler` 로 옮기면 폭발에도 정리된다. `conduit.break` 는 `Bundle.BlockChanged` 가 같은 일을 이미 하므로 지우면 된다.
+  - 블록이 없어질 때의 정리(`BlockBreak` 의 `machine.break`, `storage.break`, `netstore.break`)는 플레이어가 캘 때만 돈다. 폭발로 없어진 기계의 창, 서랍의 표시, 저장망 노드의 설정은 남는다. 블록의 `BreakHandler` 로 옮기면 폭발에도 정리된다.
   - 이차원에서 죽으면 `death` 업적도 센다(진 판마다 한 번). 세지 않으려면 `achievement.death` 가 `core.InArena(e.P.Position())` 인 죽음을 건너뛰게 한다.
   - 폭발 공격은 피해를 주지 못했을 때도(맞은 직후의 무적 시간) 사냥꾼을 밀어낸다. 근접 공격은 피해가 들어갔을 때만 민다. 맞출지 정한다.
   - 취소가 필요 없는데 취소하는 구독자: 공중에서 연 메뉴 아이템(`Use` 의 `screens.menuItem`), 흙·잔디를 두드린 막대기(`handwork.stick`). 취소하지 않게 바꾸면 장면 골든의 `cancelled=` 가 달라지고, 막대기는 섬 가장자리에서 옆 칸 가드의 "섬 밖" 팁이 함께 뜬다.
~~~~

## REF T148

A broken machine takes itself down, also after an explosion. Prototype patch: `T148.patch`. Only after decision 1 of section 9.2.

Behaviour change C20. `machine.break` goes; `Machine.BreakInfo()` has a `BreakHandler` that calls `machine.Break` (the monitor forgets the machine, its windows close). Dragonfly calls it when the block was broken and is gone: by a player (after the handler's subscribers, so a refused break never reaches it) and by an explosion.

The handler is a function made once with the machine's state (`State.broken`, set in `New`; `DecodeNBT` goes through `New`), so `BreakInfo` allocates nothing for it. A machine without a state (an item, the transient block of a placement) has no handler.

`TestBrokenMachineIsTakenDown` (added to `blockbroken_test.go`) breaks a machine with `Player.BreakBlock` and blows one up, and reads `machine.Ticking`. Before the change it fails with `blown up: the monitor still lists the machine`; the hand break passes before and after.

`pin_blocks_test.go`: `machine.break` leaves the row. `docs/TODO.md`: the bullet.

Goldens: re-record `^TestPin(Order|Blocks)$`. `order.golden`: `BlockBreak NORMAL machine.break` goes. `handler-blocks: 33 scenarios, 14 traces changed`, no `RESULT`. The sorted dump equals that of the state before. Allocations: run `-bench 'LookMachine$|LookInfoMachine$|BlockBreak$|MachineTick$' -benchmem` before and after; allocs/op equal (47, 23, 1, 8 on the prototype).

### `content/skygear/blockbroken_test.go`

~~~~diff
--- a/content/skygear/blockbroken_test.go
+++ b/content/skygear/blockbroken_test.go
@@ -11,6 +11,7 @@ import (
 	"github.com/presentkim/skygear/common/itemnet"
 	"github.com/presentkim/skygear/content/skygear/conduit"
 	"github.com/presentkim/skygear/content/skygear/island"
+	"github.com/presentkim/skygear/content/skygear/machine"
 )
 
 // What a block leaves behind when it goes is cleaned up by the block's own
@@ -73,3 +74,34 @@ func TestBrokenBundleLosesItsDisplay(t *testing.T) {
 		})
 	}
 }
+
+// TestBrokenMachineIsTakenDown: a machine a player breaks is forgotten by the
+// monitor (and its windows close: the same call, machine.Break), and so is
+// one an explosion removes. The machine's BreakInfo does it, which Dragonfly
+// calls for every break; no subscriber of the player's break does.
+func TestBrokenMachineIsTakenDown(t *testing.T) {
+	pos := cube.Pos{4, island.IslandY + 2, 4}
+	for _, c := range []struct {
+		name   string
+		remove func(p *player.Player, tx *world.Tx)
+	}{
+		{"broken by hand", func(p *player.Player, _ *world.Tx) { p.BreakBlock(pos) }},
+		{"blown up", func(_ *player.Player, tx *world.Tx) { blowUp(tx, pos) }},
+	} {
+		s, h := brokenSession(t, pos)
+		placeTestMachine(s, pos, machine.KindCrusher)
+		tick(s, 2)
+		if _, ok := machine.Ticking(s.Session, machine.KindCrusher)[pos]; !ok {
+			t.Fatalf("%s: the monitor does not know the machine", c.name)
+		}
+		withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
+			c.remove(p, tx)
+			if _, still := tx.Block(pos).(machine.Machine); still {
+				t.Fatalf("%s: the machine is still there", c.name)
+			}
+		})
+		if _, ok := machine.Ticking(s.Session, machine.KindCrusher)[pos]; ok {
+			t.Errorf("%s: the monitor still lists the machine", c.name)
+		}
+	}
+}
~~~~

### `content/skygear/machine/machine.go`

~~~~diff
--- a/content/skygear/machine/machine.go
+++ b/content/skygear/machine/machine.go
@@ -558,6 +558,10 @@ type State struct {
 	// Facing is the side a network attachment acts on (south when it was
 	// placed without a clicked face).
 	Facing cube.Face
+	// broken is the BreakHandler of the machine's BreakInfo: what Dragonfly
+	// calls when the block was broken and is gone. It is made once with the
+	// state (New), so BreakInfo allocates nothing for it.
+	broken func(pos cube.Pos, tx *world.Tx, u item.User)
 	// pos is where the machine stands, set when it ticks or its window is
 	// opened.
 	pos cube.Pos
@@ -738,6 +742,11 @@ func New(kind string) Machine {
 	}
 	st.ShowProgress()
 	st.ShowEnergy()
+	st.broken = func(pos cube.Pos, tx *world.Tx, _ item.User) {
+		if s, ok := core.Of(tx); ok {
+			Break(s, tx, pos, Machine{Kind: kind, State: st})
+		}
+	}
 	return Machine{Kind: kind, State: st}
 }
 
@@ -1279,10 +1288,18 @@ func (Machine) Category() category.Category { return category.Items() }
 
 // BreakInfo drops the machine and its contents.
 func (m Machine) BreakInfo() block.BreakInfo {
+	// A placed machine takes itself down when it was broken (Break): by a
+	// player, or by an explosion. An item has no state and nothing to take
+	// down.
+	var broken func(cube.Pos, *world.Tx, item.User)
+	if m.State != nil {
+		broken = m.State.broken
+	}
 	return block.BreakInfo{
 		Hardness: 1.5, BlastResistance: 6,
-		Harvestable: func(item.Tool) bool { return true },
-		Effective:   blockmodel.WoodEffective(m.Kind == KindSieve),
+		BreakHandler: broken,
+		Harvestable:  func(item.Tool) bool { return true },
+		Effective:    blockmodel.WoodEffective(m.Kind == KindSieve),
 		Drops: func(item.Tool, []item.Enchantment) []item.Stack {
 			drops := []item.Stack{item.NewStack(Machine{Kind: m.Kind}, 1)}
 			if m.State != nil {
@@ -1881,7 +1898,8 @@ func Place(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, b Mach
 	})
 }
 
-// Break closes the windows of a machine being broken.
+// Break takes down a machine that was broken: the monitor forgets it and its
+// windows close. The machine's BreakInfo calls it when the block is gone.
 func Break(s *core.Session, tx *world.Tx, pos cube.Pos, m Machine) {
 	delete(FactoryOf(s).machines, pos)
 	if m.State != nil {
~~~~

### `content/skygear/machine/subscribe.go`

~~~~diff
--- a/content/skygear/machine/subscribe.go
+++ b/content/skygear/machine/subscribe.go
@@ -16,13 +16,9 @@ func subscribeMonitor() {
 	})
 }
 
-// A machine breaking or placed is taken down or set up.
+// A placed machine is set up. A broken one takes itself down: its BreakInfo
+// has the handler (Break), which Dragonfly calls for every break.
 func subscribeBlocks() {
-	core.BlockBreak.On(core.Normal, "machine.break", func(e core.BlockBreakEvent) {
-		if b, ok := e.Block.(Machine); ok {
-			Break(e.S, e.Tx, e.Pos, b)
-		}
-	})
 	core.BlockPlace.On(core.Normal, "machine.place", func(e core.BlockPlaceEvent) {
 		if b, ok := e.Block.(Machine); ok {
 			Place(e.S, e.P, e.Tx, e.Pos, b)
~~~~

### `content/skygear/pin_blocks_test.go`

~~~~diff
--- a/content/skygear/pin_blocks_test.go
+++ b/content/skygear/pin_blocks_test.go
@@ -277,7 +277,7 @@ func TestPinBlocks(t *testing.T) {
 	// reorders the subscribers (TestTierOrder) has other traces.
 	rows := map[string]string{
 		"StartBreak/own":   "expand.guard island.guard tinker.autoTool",
-		"BlockBreak/stone": "expand.guard island.guard storage.front tinker.break machine.break storage.break farm.break netstore.break achievement.break",
+		"BlockBreak/stone": "expand.guard island.guard storage.front tinker.break storage.break farm.break netstore.break achievement.break",
 		"BlockPlace/dirt":  "expand.guard island.guard machine.place storage.place farm.plant autofill.place netstore.place achievement.place progress.dragonEgg conduit.place",
 	}
 	for name, row := range rows {
~~~~

### `docs/TODO.md`

~~~~diff
--- a/docs/TODO.md
+++ b/docs/TODO.md
@@ -21,7 +21,7 @@
 - 훅에 남은 결정([extension.md](tech/extension.md) "구독자와 취소", "훅과 블록·아이템의 메서드"). 정하면 플레이어가 보는 동작이 바뀌므로 테스트를 더하고 장면 골든을 다시 기록한다.
   - 서버가 멈출 때 이차원에 있던 플레이어가 돌려받는 메뉴 아이템은 옛 이름 그대로다. `screens.menuItem`(LOWEST)이 `dimension.joined`(LOW)보다 먼저 돌아 이름을 고친 뒤에 소지품이 돌아오기 때문이다. 1초 안에 고쳐지는지, 입장 때 고칠지 정한다.
   - 섬 끝줄에 놓인 기계의 바깥쪽 면에서는 요정의 모래시계가 쓰이지 않는다. 옆 칸 가드(`island.guardSide`)가 그 클릭을 막는다. 그 면에서 기계 창이 열리지 않는 것과 같다. 그대로 둘지 정한다.
-  - 블록이 없어질 때의 정리(`BlockBreak` 의 `machine.break`, `storage.break`, `netstore.break`)는 플레이어가 캘 때만 돈다. 폭발로 없어진 기계의 창, 서랍의 표시, 저장망 노드의 설정은 남는다. 블록의 `BreakHandler` 로 옮기면 폭발에도 정리된다.
+  - 블록이 없어질 때의 정리(`BlockBreak` 의 `storage.break`, `netstore.break`)는 플레이어가 캘 때만 돈다. 폭발로 없어진 서랍의 표시와 저장망 노드의 설정은 남는다. 블록의 `BreakHandler` 로 옮기면 폭발에도 정리된다. 기계는 자기 `BreakInfo` 의 `BreakHandler` 에서 정리한다.
   - 이차원에서 죽으면 `death` 업적도 센다(진 판마다 한 번). 세지 않으려면 `achievement.death` 가 `core.InArena(e.P.Position())` 인 죽음을 건너뛰게 한다.
   - 폭발 공격은 피해를 주지 못했을 때도(맞은 직후의 무적 시간) 사냥꾼을 밀어낸다. 근접 공격은 피해가 들어갔을 때만 민다. 맞출지 정한다.
   - 취소가 필요 없는데 취소하는 구독자: 공중에서 연 메뉴 아이템(`Use` 의 `screens.menuItem`), 흙·잔디를 두드린 막대기(`handwork.stick`). 취소하지 않게 바꾸면 장면 골든의 `cancelled=` 가 달라지고, 막대기는 섬 가장자리에서 옆 칸 가드의 "섬 밖" 팁이 함께 뜬다.
~~~~

## REF T149

A broken drawer and a broken set machine clean up in the block. Prototype patch: `T149.patch`. Only after decision 1 of section 9.2.

Behaviour change C21. `storage.break` goes.

- `storage/drawer.go`: `Drawer.BreakInfo()` has a `BreakHandler` that calls `RemoveDisplay`. It captures nothing, so `BreakInfo` allocates nothing for it.
- `machine/kinds.go`: `Behaviour.Broken`, what the owner of a kind is told when a machine of the kind was broken and is gone. `machine/machine.go`: `Break` calls it after the machine's own take-down.
- `storage/kinds.go`: the three kinds of a drawer set (controller, interface, trim) register `Broken`: `InvalidateNetworks`.
- `storage/subscribe.go`: the subscriber goes; `storage.front` and `storage.place` are unchanged.

Tests (added to `blockbroken_test.go`): `TestBrokenDrawerLosesItsDisplay` (a hand break and a blast; before: `blown up: the drawer's display is still there`, `blown up: the drawer's label is still shown`) and `TestBrokenTrimCutsTheDrawerSet` (a hand break, and the trim removed as an explosion removes a block, by the helper `explode`: a blast would also take the drawer the test reads; before: `removed as an explosion removes it: the controller still reaches 1 drawers`). The hand breaks pass before and after. By mutation: without `Broken` on the trim both cases of the second test fail.

`pin_blocks_test.go`: `storage.break` leaves the row. `docs/TODO.md`: the bullet.

Goldens: re-record `^TestPin(Order|Blocks)$`. `order.golden`: `BlockBreak NORMAL storage.break` goes. `handler-blocks: 33 scenarios, 14 traces changed`, no `RESULT`. The sorted dump equals that of the state before.

### `content/skygear/blockbroken_test.go`

~~~~diff
--- a/content/skygear/blockbroken_test.go
+++ b/content/skygear/blockbroken_test.go
@@ -5,6 +5,7 @@ import (
 
 	"github.com/df-mc/dragonfly/server/block"
 	"github.com/df-mc/dragonfly/server/block/cube"
+	"github.com/df-mc/dragonfly/server/item"
 	"github.com/df-mc/dragonfly/server/player"
 	"github.com/df-mc/dragonfly/server/world"
 	"github.com/go-gl/mathgl/mgl64"
@@ -12,6 +13,7 @@ import (
 	"github.com/presentkim/skygear/content/skygear/conduit"
 	"github.com/presentkim/skygear/content/skygear/island"
 	"github.com/presentkim/skygear/content/skygear/machine"
+	"github.com/presentkim/skygear/content/skygear/storage"
 )
 
 // What a block leaves behind when it goes is cleaned up by the block's own
@@ -105,3 +107,86 @@ func TestBrokenMachineIsTakenDown(t *testing.T) {
 		}
 	}
 }
+
+// explode removes the one block at pos as an explosion removes a block: it
+// clears the block and then runs the block's break handler without a player
+// (block.ExplosionConfig.Explode). For where a blast would also remove what
+// the test reads afterwards.
+func explode(tx *world.Tx, pos cube.Pos) {
+	b := tx.Block(pos)
+	tx.SetBlock(pos, nil, nil)
+	if h := b.(block.Breakable).BreakInfo().BreakHandler; h != nil {
+		h(pos, tx, nil)
+	}
+}
+
+// TestBrokenDrawerLosesItsDisplay: a drawer a player breaks loses its display
+// and its label, and so does one an explosion removes. The drawer's BreakInfo
+// does it; no subscriber of the player's break does.
+func TestBrokenDrawerLosesItsDisplay(t *testing.T) {
+	pos := cube.Pos{4, island.IslandY + 2, 4}
+	for _, c := range []struct {
+		name   string
+		remove func(p *player.Player, tx *world.Tx)
+	}{
+		{"broken by hand", func(p *player.Player, _ *world.Tx) { p.BreakBlock(pos) }},
+		{"blown up", func(_ *player.Player, tx *world.Tx) { blowUp(tx, pos) }},
+	} {
+		s, h := brokenSession(t, pos)
+		withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
+			tx.SetBlock(pos, storage.Drawer{Facing: cube.South, Item: item.NewStack(block.Dirt{}, 1), Count: 70}, nil)
+		})
+		tick(s, 40)
+		withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
+			if _, ok := storage.LocalOf(s.Session).Displays[pos]; !ok {
+				t.Fatalf("%s: the drawer has no display", c.name)
+			}
+			p.Teleport(pos.Vec3Centre().Add(mgl64.Vec3{0, 1, 2})) // the ticks let the player fall
+			c.remove(p, tx)
+			if _, still := tx.Block(pos).(storage.Drawer); still {
+				t.Fatalf("%s: the drawer is still there", c.name)
+			}
+			if _, ok := storage.LocalOf(s.Session).Displays[pos]; ok {
+				t.Errorf("%s: the drawer's display is still there", c.name)
+			}
+		})
+		if shapeAs(s, storage.DrawerLabelKey(pos)) != nil {
+			t.Errorf("%s: the drawer's label is still shown", c.name)
+		}
+	}
+}
+
+// TestBrokenTrimCutsTheDrawerSet: a controller reaches a drawer through a
+// drawer trim. When the trim is broken, by a player or by an explosion, the
+// controller no longer reaches the drawer at once: the set kinds tell the
+// storage that one of them was broken (machine.Behaviour.Broken), and the
+// networks it remembers for the tick are dropped.
+func TestBrokenTrimCutsTheDrawerSet(t *testing.T) {
+	ctl := cube.Pos{4, island.IslandY + 2, 4}
+	trim, drawer := ctl.Add(cube.Pos{0, 0, 1}), ctl.Add(cube.Pos{0, 0, 2})
+	for _, c := range []struct {
+		name   string
+		remove func(p *player.Player, tx *world.Tx)
+	}{
+		{"broken by hand", func(p *player.Player, _ *world.Tx) { p.BreakBlock(trim) }},
+		{"removed as an explosion removes it", func(_ *player.Player, tx *world.Tx) { explode(tx, trim) }},
+	} {
+		s, h := brokenSession(t, trim)
+		placeTestMachine(s, ctl, machine.KindDrawerController)
+		placeTestMachine(s, trim, machine.KindDrawerTrim)
+		withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
+			tx.SetBlock(drawer, storage.Drawer{Facing: cube.South}, nil)
+			storage.InvalidateNetworks(s.Session)
+			if n := len(storage.Network(tx, ctl)); n != 1 {
+				t.Fatalf("%s: the controller reaches %d drawers through the trim, want 1", c.name, n)
+			}
+			c.remove(p, tx)
+			if _, still := tx.Block(trim).(machine.Machine); still {
+				t.Fatalf("%s: the trim is still there", c.name)
+			}
+			if n := len(storage.Network(tx, ctl)); n != 0 {
+				t.Errorf("%s: the controller still reaches %d drawers", c.name, n)
+			}
+		})
+	}
+}
~~~~

### `content/skygear/machine/kinds.go`

~~~~diff
--- a/content/skygear/machine/kinds.go
+++ b/content/skygear/machine/kinds.go
@@ -43,6 +43,9 @@ type Behaviour struct {
 	// Info are the lines the kind adds to the info icon of its window, after
 	// the lines that explain it (State.InfoItem).
 	Info func(st *State) []wintext.Text
+	// Broken is told when a machine of the kind was broken and is gone, by a
+	// player or by an explosion, after the machine took itself down (Break).
+	Broken func(s *core.Session, tx *world.Tx, pos cube.Pos)
 }
 
 // NoLines is the Lines of a kind whose panel has no lines.
~~~~

### `content/skygear/machine/machine.go`

~~~~diff
--- a/content/skygear/machine/machine.go
+++ b/content/skygear/machine/machine.go
@@ -1898,13 +1898,17 @@ func Place(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, b Mach
 	})
 }
 
-// Break takes down a machine that was broken: the monitor forgets it and its
-// windows close. The machine's BreakInfo calls it when the block is gone.
+// Break takes down a machine that was broken: the monitor forgets it, its
+// windows close and the feature that owns its kind is told (Behaviour.Broken).
+// The machine's BreakInfo calls it when the block is gone.
 func Break(s *core.Session, tx *world.Tx, pos cube.Pos, m Machine) {
 	delete(FactoryOf(s).machines, pos)
 	if m.State != nil {
 		m.State.storage.CloseAll()
 	}
+	if broken := m.Def().hooks.Broken; broken != nil {
+		broken(s, tx, pos)
+	}
 }
 
 func Item(kind string) item.Stack { return item.NewStack(Machine{Kind: kind}, 1) }
~~~~

### `content/skygear/pin_blocks_test.go`

~~~~diff
--- a/content/skygear/pin_blocks_test.go
+++ b/content/skygear/pin_blocks_test.go
@@ -277,7 +277,7 @@ func TestPinBlocks(t *testing.T) {
 	// reorders the subscribers (TestTierOrder) has other traces.
 	rows := map[string]string{
 		"StartBreak/own":   "expand.guard island.guard tinker.autoTool",
-		"BlockBreak/stone": "expand.guard island.guard storage.front tinker.break storage.break farm.break netstore.break achievement.break",
+		"BlockBreak/stone": "expand.guard island.guard storage.front tinker.break farm.break netstore.break achievement.break",
 		"BlockPlace/dirt":  "expand.guard island.guard machine.place storage.place farm.plant autofill.place netstore.place achievement.place progress.dragonEgg conduit.place",
 	}
 	for name, row := range rows {
~~~~

### `content/skygear/storage/drawer.go`

~~~~diff
--- a/content/skygear/storage/drawer.go
+++ b/content/skygear/storage/drawer.go
@@ -118,6 +118,13 @@ func (d Drawer) BreakInfo() block.BreakInfo {
 		Harvestable: func(item.Tool) bool { return true },
 		Effective:   blockmodel.AxeEffective,
 		Drops:       func(item.Tool, []item.Enchantment) []item.Stack { return []item.Stack{d.ToItem()} },
+		// A drawer that was broken, by a player or by an explosion, loses
+		// its display.
+		BreakHandler: func(pos cube.Pos, tx *world.Tx, _ item.User) {
+			if s, ok := core.Of(tx); ok {
+				RemoveDisplay(s, tx, pos)
+			}
+		},
 	}
 }
 
~~~~

### `content/skygear/storage/kinds.go`

~~~~diff
--- a/content/skygear/storage/kinds.go
+++ b/content/skygear/storage/kinds.go
@@ -12,20 +12,24 @@ import (
 // machine kinds. The drawer controller and the drawer interface stand for
 // their drawer set: a click deposits into it, hoppers and conduits put into
 // and take from it, and the panel counts it. The drawer trim only joins
-// drawers: a click does nothing and its panel has no lines.
+// drawers: a click does nothing and its panel has no lines. A machine of a
+// set that was broken invalidates the networks.
 func registerKinds() {
+	broken := func(s *core.Session, _ *world.Tx, _ cube.Pos) { InvalidateNetworks(s) }
 	set := machine.Behaviour{
 		Click: func(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool {
 			UseController(s, p, tx, pos)
 			return true
 		},
 		Insert: ControllerInsert, Peek: ControllerPeek, Extract: ControllerExtract,
-		Lines: Lines,
+		Lines:  Lines,
+		Broken: broken,
 	}
 	machine.Register(machine.KindDrawerController, set)
 	machine.Register(machine.KindDrawerInterface, set)
 	machine.Register(machine.KindDrawerTrim, machine.Behaviour{
-		Click: func(*core.Session, *player.Player, *world.Tx, cube.Pos) bool { return true },
-		Lines: machine.NoLines,
+		Click:  func(*core.Session, *player.Player, *world.Tx, cube.Pos) bool { return true },
+		Lines:  machine.NoLines,
+		Broken: broken,
 	})
 }
~~~~

### `content/skygear/storage/subscribe.go`

~~~~diff
--- a/content/skygear/storage/subscribe.go
+++ b/content/skygear/storage/subscribe.go
@@ -24,8 +24,10 @@ func subscribeDisplays() {
 // Hitting a drawer's front takes items out and never mines it: the break
 // that ends such a hit is refused before any step works on the block (Low),
 // where the island allows building (elsewhere the island's guard refuses it).
-// A drawer that does break loses its display. A set machine breaking or
-// placed invalidates the networks; a placed drawer is set up.
+// A drawer that does break loses its display and a broken set machine
+// invalidates the networks through the block itself (Drawer.BreakInfo, the
+// set kinds' Broken). A placed set machine invalidates the networks; a
+// placed drawer is set up.
 func subscribeBlocks() {
 	core.BlockBreak.On(core.Low, "storage.front", func(e core.BlockBreakEvent) {
 		if _, ok := e.Block.(Drawer); !ok {
@@ -35,16 +37,6 @@ func subscribeBlocks() {
 			e.Cancel()
 		}
 	})
-	core.BlockBreak.On(core.Normal, "storage.break", func(e core.BlockBreakEvent) {
-		switch b := e.Block.(type) {
-		case Drawer:
-			RemoveDisplay(e.S, e.Tx, e.Pos)
-		case machine.Machine:
-			if IsSetMachine(b.Kind) {
-				InvalidateNetworks(e.S)
-			}
-		}
-	})
 	core.BlockPlace.On(core.Normal, "storage.place", func(e core.BlockPlaceEvent) {
 		switch b := e.Block.(type) {
 		case machine.Machine:
~~~~

### `docs/TODO.md`

~~~~diff
--- a/docs/TODO.md
+++ b/docs/TODO.md
@@ -21,7 +21,7 @@
 - 훅에 남은 결정([extension.md](tech/extension.md) "구독자와 취소", "훅과 블록·아이템의 메서드"). 정하면 플레이어가 보는 동작이 바뀌므로 테스트를 더하고 장면 골든을 다시 기록한다.
   - 서버가 멈출 때 이차원에 있던 플레이어가 돌려받는 메뉴 아이템은 옛 이름 그대로다. `screens.menuItem`(LOWEST)이 `dimension.joined`(LOW)보다 먼저 돌아 이름을 고친 뒤에 소지품이 돌아오기 때문이다. 1초 안에 고쳐지는지, 입장 때 고칠지 정한다.
   - 섬 끝줄에 놓인 기계의 바깥쪽 면에서는 요정의 모래시계가 쓰이지 않는다. 옆 칸 가드(`island.guardSide`)가 그 클릭을 막는다. 그 면에서 기계 창이 열리지 않는 것과 같다. 그대로 둘지 정한다.
-  - 블록이 없어질 때의 정리(`BlockBreak` 의 `storage.break`, `netstore.break`)는 플레이어가 캘 때만 돈다. 폭발로 없어진 서랍의 표시와 저장망 노드의 설정은 남는다. 블록의 `BreakHandler` 로 옮기면 폭발에도 정리된다. 기계는 자기 `BreakInfo` 의 `BreakHandler` 에서 정리한다.
+  - 저장망 노드가 없어질 때의 정리(`BlockBreak` 의 `netstore.break`)는 플레이어가 캘 때만 돈다. 폭발로 없어진 노드의 설정은 남는다. 기계와 서랍처럼 블록의 `BreakHandler` 에서 정리하게 옮긴다(`machine.Behaviour` 의 `Broken`).
   - 이차원에서 죽으면 `death` 업적도 센다(진 판마다 한 번). 세지 않으려면 `achievement.death` 가 `core.InArena(e.P.Position())` 인 죽음을 건너뛰게 한다.
   - 폭발 공격은 피해를 주지 못했을 때도(맞은 직후의 무적 시간) 사냥꾼을 밀어낸다. 근접 공격은 피해가 들어갔을 때만 민다. 맞출지 정한다.
   - 취소가 필요 없는데 취소하는 구독자: 공중에서 연 메뉴 아이템(`Use` 의 `screens.menuItem`), 흙·잔디를 두드린 막대기(`handwork.stick`). 취소하지 않게 바꾸면 장면 골든의 `cancelled=` 가 달라지고, 막대기는 섬 가장자리에서 옆 칸 가드의 "섬 밖" 팁이 함께 뜬다.
~~~~

## REF T150

A broken node drops its network and its setting in the block. Prototype patch: `T150.patch`. Only after decision 1 of section 9.2.

Behaviour change C22. `netstore.break` and its deferred function (the mark `netstore.breakDeferred`) go. The eight node kinds (`machine.NetNodes`: controller, cable, disk drive, terminal, importer, exporter, external storage, autocrafter; not the pattern encoder) register `Broken: nodeBroken`, which calls `Invalidate` and `Forget` directly: the block is gone when `Break` runs.

`netstore.place` keeps its `Forget`: a node that went without being broken (another block was set in its place) still leaves its setting. Its comment says so.

`TestBrokenNodeLeavesItsNetwork` (added to `blockbroken_test.go`) breaks one of two joined cables by hand and removes one as an explosion removes a block. Before the change it fails with `removed as an explosion removes it: the cable left behind is not a network of its own` and `the broken node's setting is still there (1 filter entries)`; the hand break passes before and after.

`pin_blocks_test.go`: `netstore.break` leaves the row; two comments. `docs/TODO.md`: the bullet on the block clean-up goes.

Goldens: re-record `^TestPin(Order|Blocks)$`. `order.golden`: `BlockBreak NORMAL netstore.break` goes (113 lines). The check script prints `handler-blocks: 33 scenarios, 14 traces changed` and exactly one `RESULT` line, that of section 6 (`BlockBreak/netNode`: `filter=0` becomes `filter=1`; its trace also loses `netstore.breakDeferred`).

### `content/skygear/blockbroken_test.go`

~~~~diff
--- a/content/skygear/blockbroken_test.go
+++ b/content/skygear/blockbroken_test.go
@@ -13,6 +13,7 @@ import (
 	"github.com/presentkim/skygear/content/skygear/conduit"
 	"github.com/presentkim/skygear/content/skygear/island"
 	"github.com/presentkim/skygear/content/skygear/machine"
+	"github.com/presentkim/skygear/content/skygear/netstore"
 	"github.com/presentkim/skygear/content/skygear/storage"
 )
 
@@ -190,3 +191,42 @@ func TestBrokenTrimCutsTheDrawerSet(t *testing.T) {
 		})
 	}
 }
+
+// TestBrokenNodeLeavesItsNetwork: a storage network node that was broken, by
+// a player or by an explosion, is out of its network at once and its setting
+// is forgotten. The node kinds tell the storage network that one of them was
+// broken (machine.Behaviour.Broken); no subscriber of the player's break
+// does.
+func TestBrokenNodeLeavesItsNetwork(t *testing.T) {
+	pos := cube.Pos{4, island.IslandY + 2, 4}
+	next := pos.Side(cube.FaceEast)
+	for _, c := range []struct {
+		name   string
+		remove func(p *player.Player, tx *world.Tx)
+	}{
+		{"broken by hand", func(p *player.Player, _ *world.Tx) { p.BreakBlock(pos) }},
+		{"removed as an explosion removes it", func(_ *player.Player, tx *world.Tx) { explode(tx, pos) }},
+	} {
+		s, h := brokenSession(t, pos)
+		placeTestMachine(s, pos, machine.KindNetCable)
+		placeTestMachine(s, next, machine.KindNetCable)
+		withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
+			netstore.Config(s.Session, pos).Filter = []string{"kept"}
+			if n := netstore.At(tx, next); n == nil || n.Size() != 2 {
+				t.Fatalf("%s: the two cables are not one network", c.name)
+			}
+			c.remove(p, tx)
+			if _, still := tx.Block(pos).(machine.Machine); still {
+				t.Fatalf("%s: the node is still there", c.name)
+			}
+		})
+		withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
+			if n := netstore.At(tx, next); n == nil || n.Size() != 1 {
+				t.Errorf("%s: the cable left behind is not a network of its own", c.name)
+			}
+			if n := len(netstore.Config(s.Session, pos).Filter); n != 0 {
+				t.Errorf("%s: the broken node's setting is still there (%d filter entries)", c.name, n)
+			}
+		})
+	}
+}
~~~~

### `content/skygear/netstore/kinds.go`

~~~~diff
--- a/content/skygear/netstore/kinds.go
+++ b/content/skygear/netstore/kinds.go
@@ -1,23 +1,40 @@
 package netstore
 
-import "github.com/presentkim/skygear/content/skygear/machine"
+import (
+	"github.com/df-mc/dragonfly/server/block/cube"
+	"github.com/df-mc/dragonfly/server/world"
+	"github.com/presentkim/skygear/content/skygear/core"
+	"github.com/presentkim/skygear/content/skygear/machine"
+)
 
 // registerKinds hands the machine frame the behaviour of the storage
-// network's machine kinds. The cable only joins nodes and has none.
+// network's machine kinds. The cable only joins nodes and has none of its
+// own. Every node that was broken drops its network and forgets its setting
+// (nodeBroken); the pattern encoder is no node.
 func registerKinds() {
 	machine.Register(machine.KindNetController, machine.Behaviour{
 		Click: clickController, Run: runController,
 		Insert: insertAt, Peek: peekAt, Extract: extractAt,
+		Broken: nodeBroken,
 	})
-	machine.Register(machine.KindNetCable, machine.Behaviour{})
-	machine.Register(machine.KindDiskDrive, machine.Behaviour{Accepts: acceptsDisk})
+	machine.Register(machine.KindNetCable, machine.Behaviour{Broken: nodeBroken})
+	machine.Register(machine.KindDiskDrive, machine.Behaviour{Accepts: acceptsDisk, Broken: nodeBroken})
 	machine.Register(machine.KindNetTerminal, machine.Behaviour{
 		Click:  clickTerminal,
 		Insert: insertAt, Peek: peekAt, Extract: extractAt,
+		Broken: nodeBroken,
 	})
-	machine.Register(machine.KindNetImporter, machine.Behaviour{Click: clickFilter, Run: nodeRun, Accepts: acceptsUpgrade})
-	machine.Register(machine.KindNetExporter, machine.Behaviour{Click: clickFilter, Run: nodeRun, Accepts: acceptsUpgrade})
-	machine.Register(machine.KindNetExternal, machine.Behaviour{Click: clickFilter, Run: nodeRun})
+	machine.Register(machine.KindNetImporter, machine.Behaviour{Click: clickFilter, Run: nodeRun, Accepts: acceptsUpgrade, Broken: nodeBroken})
+	machine.Register(machine.KindNetExporter, machine.Behaviour{Click: clickFilter, Run: nodeRun, Accepts: acceptsUpgrade, Broken: nodeBroken})
+	machine.Register(machine.KindNetExternal, machine.Behaviour{Click: clickFilter, Run: nodeRun, Broken: nodeBroken})
 	machine.Register(machine.KindPatternEncoder, machine.Behaviour{Click: clickEncoder, Accepts: encoderAccepts})
-	machine.Register(machine.KindNetAutocrafter, machine.Behaviour{Run: runAutocrafter, Accepts: acceptsPattern})
+	machine.Register(machine.KindNetAutocrafter, machine.Behaviour{Run: runAutocrafter, Accepts: acceptsPattern, Broken: nodeBroken})
+}
+
+// nodeBroken drops the network of the node that was broken at pos, by a
+// player or by an explosion, and forgets the node's setting. The block is
+// gone when it is called.
+func nodeBroken(s *core.Session, tx *world.Tx, pos cube.Pos) {
+	Invalidate(tx, pos)
+	Forget(s, pos)
 }
~~~~

### `content/skygear/netstore/subscribe.go`

~~~~diff
--- a/content/skygear/netstore/subscribe.go
+++ b/content/skygear/netstore/subscribe.go
@@ -24,25 +24,15 @@ func subscribeFlush() {
 	})
 }
 
-// A network node breaking drops its network once the block is gone; a placed
-// node forgets the setting a node broken by an explosion left and drops the
-// network after machine.place's deferred set: netstore.place is in the High
-// tier, after the machine is set up (Normal).
+// A node that was broken drops its network through the block itself (the
+// node kinds' Broken, kinds.go). A placed node forgets the setting a node
+// left that went without being broken (another block was set in its place)
+// and drops the network after machine.place's deferred set: netstore.place
+// is in the High tier, after the machine is set up (Normal).
 func subscribeBlocks() {
-	core.BlockBreak.On(core.Normal, "netstore.break", func(e core.BlockBreakEvent) {
-		if b, ok := e.Block.(machine.Machine); ok && IsNode(b.Kind) {
-			// The block goes after this handler: drop the network once it is gone.
-			s, pos := e.S, e.Pos
-			e.Tx.Defer(func(tx *world.Tx) {
-				core.Mark("netstore.breakDeferred")
-				Invalidate(tx, pos)
-				Forget(s, pos)
-			})
-		}
-	})
 	core.BlockPlace.On(core.High, "netstore.place", func(e core.BlockPlaceEvent) {
 		if b, ok := e.Block.(machine.Machine); ok && IsNode(b.Kind) {
-			Forget(e.S, e.Pos) // a node broken by an explosion left its setting
+			Forget(e.S, e.Pos) // a node that went without being broken left its setting
 			// machine.Place sets the block in a deferred call; this one runs after it.
 			pos := e.Pos
 			e.Tx.Defer(func(tx *world.Tx) {
~~~~

### `content/skygear/pin_blocks_test.go`

~~~~diff
--- a/content/skygear/pin_blocks_test.go
+++ b/content/skygear/pin_blocks_test.go
@@ -32,9 +32,9 @@ import (
 // testdata/pin/handler-blocks.golden.
 //
 // Work the handler hands to Tx.Defer runs after the transaction callback, not
-// where the Defer line is: the netstore steps mark inside their deferred
-// function (netstore.breakDeferred, netstore.placeDeferred), so the golden
-// shows them after every step that ran in the callback. A scenario runs the
+// where the Defer line is: the netstore step marks inside its deferred
+// function (netstore.placeDeferred), so the golden shows it after every step
+// that ran in the callback. A scenario runs the
 // whole transaction inside the traced call, so deferred work is part of the
 // line. Deferred work of the feature functions (machine.Place, PlaceDrawer,
 // RefillHandLater) has no mark; the outcome reads its effect after the
@@ -172,8 +172,9 @@ func TestPinBlocks(t *testing.T) {
 			tx.Block(pos).(storage.Drawer).Punch(pos, face, tx, p)
 		})
 	}
-	// nodeConfig gives the node at pos a setting; the deferred step of a
-	// break or place forgets it.
+	// nodeConfig gives the node at pos a setting; a placed node forgets it.
+	// A broken node forgets it through the block (blockbroken_test.go), which
+	// the handler alone does not reach: the break scenario keeps it.
 	nodeConfig := func(s *session, p *player.Player, tx *world.Tx) {
 		netstore.Config(s.Session, pos).Filter = []string{"pin"}
 	}
@@ -277,7 +278,7 @@ func TestPinBlocks(t *testing.T) {
 	// reorders the subscribers (TestTierOrder) has other traces.
 	rows := map[string]string{
 		"StartBreak/own":   "expand.guard island.guard tinker.autoTool",
-		"BlockBreak/stone": "expand.guard island.guard storage.front tinker.break farm.break netstore.break achievement.break",
+		"BlockBreak/stone": "expand.guard island.guard storage.front tinker.break farm.break achievement.break",
 		"BlockPlace/dirt":  "expand.guard island.guard machine.place storage.place farm.plant autofill.place netstore.place achievement.place progress.dragonEgg conduit.place",
 	}
 	for name, row := range rows {
~~~~

### `docs/TODO.md`

~~~~diff
--- a/docs/TODO.md
+++ b/docs/TODO.md
@@ -21,7 +21,6 @@
 - 훅에 남은 결정([extension.md](tech/extension.md) "구독자와 취소", "훅과 블록·아이템의 메서드"). 정하면 플레이어가 보는 동작이 바뀌므로 테스트를 더하고 장면 골든을 다시 기록한다.
   - 서버가 멈출 때 이차원에 있던 플레이어가 돌려받는 메뉴 아이템은 옛 이름 그대로다. `screens.menuItem`(LOWEST)이 `dimension.joined`(LOW)보다 먼저 돌아 이름을 고친 뒤에 소지품이 돌아오기 때문이다. 1초 안에 고쳐지는지, 입장 때 고칠지 정한다.
   - 섬 끝줄에 놓인 기계의 바깥쪽 면에서는 요정의 모래시계가 쓰이지 않는다. 옆 칸 가드(`island.guardSide`)가 그 클릭을 막는다. 그 면에서 기계 창이 열리지 않는 것과 같다. 그대로 둘지 정한다.
-  - 저장망 노드가 없어질 때의 정리(`BlockBreak` 의 `netstore.break`)는 플레이어가 캘 때만 돈다. 폭발로 없어진 노드의 설정은 남는다. 기계와 서랍처럼 블록의 `BreakHandler` 에서 정리하게 옮긴다(`machine.Behaviour` 의 `Broken`).
   - 이차원에서 죽으면 `death` 업적도 센다(진 판마다 한 번). 세지 않으려면 `achievement.death` 가 `core.InArena(e.P.Position())` 인 죽음을 건너뛰게 한다.
   - 폭발 공격은 피해를 주지 못했을 때도(맞은 직후의 무적 시간) 사냥꾼을 밀어낸다. 근접 공격은 피해가 들어갔을 때만 민다. 맞출지 정한다.
   - 취소가 필요 없는데 취소하는 구독자: 공중에서 연 메뉴 아이템(`Use` 의 `screens.menuItem`), 흙·잔디를 두드린 막대기(`handwork.stick`). 취소하지 않게 바꾸면 장면 골든의 `cancelled=` 가 달라지고, 막대기는 섬 가장자리에서 옆 칸 가드의 "섬 밖" 팁이 함께 뜬다.
~~~~
