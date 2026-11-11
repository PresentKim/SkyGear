# Phase 3 part 3-3 (type switches become capabilities): reference

The machine plans `2026-11-10-cleanup-restructure-p3f.md`, `-p3g.md` and `-p3h.md` (tasks T59 to T74) cite this file as REF. It holds what a task needs word for word: the inventory, the decisions, every new name with its signature, the creative slots, the measured numbers and the code.

Everything here was measured at 1ec765f. The code blocks were built and run together in a scratch worktree: with all of them applied `go build ./...`, `go vet ./...` and `go test ./...` pass (51 packages), the six goldens of `content/skygear/testdata/pin/` and the four of `content/skygear/wire/testdata` are byte for byte unchanged, and the two goldens T59 records from the code of 1ec765f (`look.golden`, 143 lines; `creative.golden`, 243 lines) stay unchanged. A task copies its blocks as they are. Where a block and the code a developer finds disagree, the developer stops and reports (`NEEDS_CONTEXT`); it does not choose.

## 1. Inventory

Every place outside a package that switches or asserts on a type of another package of `content/skygear`, or lists another package's items or blocks by hand. An AST scan of all non-test files of `content/`, `common/`, `platform/` and `cmd/` found 16 type switches and 54 type assertions on such types; `common/`, `platform/` and `cmd/` have none (they cannot import `content/`). `screens` has none. The value switch of `look.MachineLines` and the hand lists are not type switches and were read by hand.

| Row | Where (HEAD) | What it lists | Cases | Order matters | Pinned today by | Choice |
|---|---|---|---|---|---|---|
| R1 | `look/look.go:156` `Info` | `machine.Machine`, `storage.Drawer`, `block.Composter`, `conduit.Bundle`, `farm.SilkwormLeaves`, `farm.Barrel`, `hopper.FlowHopper`, `farm.MagicPlant`; then the crops (`farm.CropOf`, 6 vanilla blocks) at 208 | 8 clauses and the crop check | no (one clause per block; the crop check after the switch) | `look_test.go` (crusher, drawer with an item), `magic_test.go` (plant), `cropinfo_test.go` (wheat); nothing for composter, bundle, leaves, barrel, hopper | (a) `waila.Provider` on the block, `waila.Adapter` for vanilla blocks; T59 pins all first |
| R2 | `look/look.go:318` `MachineLines` | machine definitions and kinds | 10 clauses and a default | yes (first match) | `autoequip_test.go` (3 lines) | moves whole into `machine` (the switch is then over the package's own kinds); 3-4 turns clauses into definition fields |
| R3 | `look/look.go` calls | `expand.Of` (90, 115), `machine.PeekRange` (107), `tinker.ShowVeinOutline`/`HideVeinOutline` (134, 136), `wand.ShowPreview` (96, 138), `farm.CropOf` in the trace (67), `machine.ProgressArrow` (300), `fairy.HourglassLine` (159), `storage.Lines` (163), `dimension.MobName` (368) | 11 calls | yes (inside one subscriber) | pins of `look.update`, `look.updateAll` (reached only) | seams, section 2 |
| R4 | `decor/decor.go:21` `ItemDecor` | 12 `sgitem` types in 9 clauses, then the fairy board by its item name | 10 | no | `TestPackHasItemAssets` (every registered custom item against the pack's names), `pack_test.go:520` | (a) `slotdeco.Decorated` on the item |
| R5 | `catalog/creative.go` | 243 creative items in 16 runs by owner: 10 runs of 8 features, 6 runs of `sgitem` and `hopper` | 16 runs | yes (the whole order) | `wire/testdata/creative.golden` (hashes) | (b) `core.Feature.Creative` with slots; `sgitem` and `hopper` runs stay in `catalog` |
| R6 | root `commands.go` `registerCommands` | 12 registrations of 6 commands of `screens` (menu, menu item, help, codex), `farm` (sapling), `machine` (statistics) | 12 | no (the registry is a map) | none | (b) `core.Feature.Commands` |
| R7 | root `admin.go:56` `giveItems` | short ids for items of `progress`, `storage`, `farm`, `hopper`, `machine`, `conduit`, `netstore`, `tinker`, `sgitem` | about 60 statements | no (a map) | `admingive_test.go`, `names_test.go` | (c) leave: an operator tool; every custom item is already reached by its id through `world.CustomItems()`, the hand entries are the operator's aliases |
| R8 | root `names.go`, `features.go:62` | `timed.Name` set in `init`; `localItemName`; `itemsFeature` | 3 | the edit runs after every feature's | `TestWire` items, `names_test.go` | `timed.Name` and `LocalItemName` move to `naming`; `itemsFeature` stays (c): it edits every item of the game, no feature owns that |
| R9 | root `hopper.go` | `init` calls `hopper.Register()` | 1 | yes | `TestWire` | (c) leave: measured, moving the call into `hopper`'s own `init` changes 698 entries of `items.golden`, 327 of `recipes.golden` and 243 of `creative.golden` (item registration order is in the goldens) |
| R10 | root `debug.go:147`, `debug_ui.go:32`, `debug_factory.go` | `storage.ItemHolder`, `storage.HolderData`, machine kinds | 2 assertions and lists | no | their own tests | (c) leave: operator and debug tools know everything |
| R11 | `copier/copier.go:163` and 8 assertions | `conduit.Bundle`, `machine.Machine`, `storage.Drawer` | 3 | no | `copier_test.go` | (c) leave: the copier is the top of these three and nothing above it lists them; an interface would move its card format into three features |
| R12 | `netstore/nodes.go:57` and 6 assertions | `machine.Machine`, `conduit.Bundle` | 2 | no | `netstore` tests | (c) leave: downward use of allowed siblings; its kinds are a 3-4 matter |
| R13 | `storage/storage.go:146,214`, `storage/subscribe.go:27,41` and 2 assertions | `machine.Machine` | 4 switches | no | `drawer_test.go` | defer to 3-4: storage's own kinds live in `machine` |
| R14 | `fairy/fairytime.go:58,92,70`; `dimension/collector.go:134` | `sgitem` items, `farm.MagicPlant`, `machine.Machine` | 3 and 1 | no | their tests | (c) leave: downward use of allowed siblings |
| R15 | `codex/codexwork.go:78-79` and 7 assertions on `sgitem` | `farm.Barrel`, `machine.Machine`, shared items | 9 | no | `codex_test.go` | (c) leave: the codex is design data across features (review plan, out of scope) |
| R16 | assertions on `sgitem` types in `farm` (7), `machine` (7), `dimension` (8), `progress` (3), `storage` (2), `copier` (2), `handwork` (1), `wand` (1) | shared item types | 31 | no | their tests | (c) leave: `sgitem` is the data layer below them |
| R17 | `core/refill.go`, `core/kernel.go` `AutofillFeature` | a feature inside the kernel | 1 | settings order | `settings.golden`, `order.golden` | moves to its own package `content/skygear/autofill` |
| R18 | `core/arena.go` (`InArena`), `core/screens.go` (`ShowMenu`, `UpdateBoard`), `Session.Debounced` | dimension and screen knowledge in the kernel | 3 | no | | (c) leave: `island` (a lower layer) and `fairy` need `InArena` without importing `dimension`; the function variables are the documented way a lower layer calls a higher one; the debounce knows no feature |

By consumer package, type switches and assertions on another content package's types: look 1 and 0, decor 1 and 0, copier 1 and 10, netstore 1 and 6, storage 4 and 4, fairy 2 and 1, farm 3 and 4, machine 1 and 6, handwork 1 and 0, wand 1 and 0, dimension 0 and 9, progress 0 and 3, codex 0 and 9, root 0 and 2.

## 2. Decisions

One line of reason each. They bind.

1. The panel capability lives in `common/waila` (`View`, `Provider`, `Adapter`, `RegisterAdapter`, `Describe`), not in `look` or `core`: `hopper` is below `core` in the layer list and may import only `common/` and the data layer.
2. A provider takes and returns `waila.Info` by value. A pointer to the panel passed through an interface escapes to the heap; by value the benchmarks keep their allocation counts.
3. A provider gets no session. A block that needs one calls `core.At(v.Tx)` (the session registered for the transaction's world) and adds nothing when it is nil, as `farm`'s item adapter already does.
4. `look.Info` keeps its signature (`s` is no longer used inside it): the pin test and three other tests call it and must pass unchanged.
5. `MachineLines`, `detailLines`, `crafterLines`, `outputContainer`, `itemLine` and `FurnaceArrow` move into `machine` as they are. `dimension.MobName(lang, st.Mob)` becomes `i18n.T(lang, "mob."+st.Mob)`, which is its whole body.
6. `fairy.HourglassLine` becomes `machine.BoostLine`: it reads only `State.Boost`, a field of `machine`; a hook for one caller is not needed.
7. The drawer controller's lines come through a new definition field `machine.Def.Lines`, filled by `storage` in the `init` that already fills `Click`, `Insert`, `Peek`, `Extract` of the same two kinds.
8. `machine.PeekRange` is called by `Machine.Waila` when the viewer sneaks. It only sets a field of the state, so the order against the panel's packets cannot change.
9. The vanilla furnaces' progress arrow is set by an adapter of `machine` (it owns the arrow item), after `look` built the container.
10. `look` moves from the screens layer to the core layer of `layers_test.go`, because the features that hide the panel or follow it (`expand`, `tinker`, `wand`) import it. The layer list then forbids `look` to import any feature.
11. `look.Hidden` is a function variable set by `expand` from its `Setup`; `look.Follow` is a named list filled by `tinker` and `wand` from their `Setup`, in feature-list order (tinker before wand). No new hook and no new subscriber: `order.golden` and `lifecycle.golden` may not change in this part.
12. The trace's crop height is `look`'s own (`look.CropHeight`, six vanilla blocks): it is the shape of vanilla blocks, like the vanilla container switch that stays. `TestLookCropsMatchFarm` keeps it in step with `farm.CropOf`.
13. The slot decoration capability is `slotdeco.Decorated` in `common/slotdeco`; the twelve `sgitem` types implement it in one file `sgitem/decor.go`, `fairy.Board` in `fairy`. `decor` keeps `tinker.RarityColour` (lowering it is part of 3-4).
14. The creative list is declared with slots: `core.CreativeItem{Slot, Category, Item}`; a feature returns its items from `Feature.Creative`; `catalog` keeps the runs whose item types are declared in `sgitem` or `hopper` (they are no feature) and the three groups. An item belongs to the package that declares its Go type. A list derived from `Category()` cannot reproduce today's: 45 of the 243 entries are in another tab than their `Category()`, and 106 of the 349 registered custom items are not listed.
15. `Feature.Creative` and `Feature.Commands` are functions called when the game starts, not values built at package initialisation: the texts and every item are registered by then.
16. A command finds its session with `core.PlayerSession` (the root's `playerSession` with `core.Of`); the codex command lives in `screens` (it opens a screen; `codex` is not a feature).
17. `Install` does nothing new: `Creative` and `Commands` are read later by `core.CreativeItems` and `core.Commands`, in feature-list order.

## 3. New names

```go
// common/waila
type View struct { Lang string; Sneaking bool; Tx *world.Tx; Pos cube.Pos }
type Provider interface { Waila(v View, info Info) Info }
type Adapter func(v View, b world.Block, info Info) Info
func RegisterAdapter(a Adapter)
func Describe(v View, b world.Block, info Info) Info

// implemented by
func (h hopper.FlowHopper) Waila(v waila.View, info waila.Info) waila.Info
func (b conduit.Bundle) Waila(v waila.View, info waila.Info) waila.Info
func (d storage.Drawer) Waila(v waila.View, info waila.Info) waila.Info
func (l farm.SilkwormLeaves) Waila(v waila.View, info waila.Info) waila.Info
func (b farm.Barrel) Waila(v waila.View, info waila.Info) waila.Info
func (b farm.MagicPlant) Waila(v waila.View, info waila.Info) waila.Info
func (m machine.Machine) Waila(v waila.View, info waila.Info) waila.Info

// machine
func MachineLines(lang string, m Machine) ([]string, *waila.ItemLine)
func BoostLine(lang string, st *State) string
func FurnaceArrow(b world.Block) item.Stack
Def.Lines func(lang string, tx *world.Tx, pos cube.Pos) []string

// look
func CropHeight(b world.Block) (float64, bool)
var Hidden = func(s *core.Session, p *player.Player) bool { return false }
type Target struct { S *core.Session; P *player.Player; Tx *world.Tx; Pos cube.Pos; Ok bool; Tick bool }
func Follow(name string, f func(Target))
func Followers() []string

// common/slotdeco
type Decorated interface { Decor() (Rarity, int) }

// core
type CreativeItem struct { Slot int; Category creative.Category; Item world.Item }
Feature.Creative func() []CreativeItem
func CreativeItems(own []CreativeItem) []CreativeItem
Feature.Commands func() []cmd.Command
func Commands() []cmd.Command
func PlayerSession(src cmd.Source, o *cmd.Output, tx *world.Tx) (*player.Player, *Session, bool)

// naming
func LocalItemName(e protocol.ItemEntry) protocol.ItemEntry

// autofill (was core)
const Setting = "autofill"                                  // core.AutofillSetting
func On(p *player.Player) bool                              // core.Autofill
func RefillHand(p *player.Player, proto item.Stack) bool    // core.RefillHand
func RefillHandLater(p *player.Player, tx *world.Tx, proto item.Stack) // core.RefillHandLater
var Feature *core.Feature                                   // core.AutofillFeature
```

Removed: `look.MachineLines`, `look.FurnaceArrow`, `fairy.HourglassLine`, the root's `registerCommands`, `playerSession`, `openCodex`, `localItemName`, the files `content/skygear/commands.go`, `content/skygear/names.go`, `content/skygear/core/refill.go`.

## 4. Imports of `content/skygear` packages, before and after

| Package | Before | After |
|---|---|---|
| `look` | conduit, core, dimension, expand, fairy, farm, hopper, machine, naming, sgitem, storage, textfmt, tinker, wand (14) | core, naming |
| `decor` | naming, sgitem, tinker | naming, tinker |
| `catalog` | conduit, elevator, fairy, farm, hopper, machine, netstore, progress, sgitem, storage | the same without fairy, plus core (`creative.go` itself: core, hopper, machine, sgitem; the others come from `recipes.go` and `research.go`, which stay) |
| root | every feature (it lists them); 12 non-test files, 2,115 lines | every feature plus autofill; 10 non-test files, 1,927 lines (`commands.go` and `names.go` are gone) |
| new edges | | `expand`, `tinker`, `wand`, `screens` import `look`; `machine`, `wand` import `autofill`; `hopper`, `conduit`, `storage`, `machine` import `common/waila`; `sgitem`, `fairy` import `common/slotdeco`; `naming` imports `common/timed` |

Layer list after T73: the core group is `"core", "network", "achievement", "preview", "look", "autofill"`; the screens group is `"codex", "screens"`. The feature list keeps its 23 names in the same order (`TestFeatureNames` is not edited).

## 5. Measured numbers

Benchmarks at 1ec765f in this container (`go test -run xxx -bench <name> -benchmem -count 5 ./content/skygear`), mean of the runs; the same with every block of this file applied:

| Benchmark | ns/op before | B/op and allocs/op before | after |
|---|---|---|---|
| `BenchmarkHandleMove` | about 190 | 24 B, 0 | 24 B, 0; ns within noise |
| `BenchmarkSecond` | about 27,000 | 10,557 B, 63 | not touched |
| `BenchmarkLookMachine` (new, T59) | about 6,000 | 4,232 B, 47 | 4,232 B, 47; ns within noise |
| `BenchmarkLookInfoMachine` (new, T59) | about 2,300 | 1,600 B, 23 | 1,600 B, 23; ns within noise |
| `BenchmarkLookInfoStone` (new, T59) | about 770 | 488 B, 10 | 488 B, 10; about 30 ns more (the provider check and two adapters; 3 percent of `look.Info` in a profile) |

The rule for T60 to T65: B/op and allocs/op of the three look benchmarks and of `BenchmarkHandleMove` equal the numbers T59 reported; ns/op (mean of 5) within 15 percent of T59's.

Other counts: 123 subscribers in `order.golden` (unchanged by this part); 36 machine kinds; 349 registered custom items; 243 creative items of the game (88 construction, 24 equipment, 131 items).

## 6. Creative slots

The runs of today's list in order. A run is the statements of one owner that follow each other.

| Slot | Owner | Tab | Items, in order |
|---|---|---|---|
| 10 | progress | construction | `ResearchTable{}` |
| 20 | storage | construction | `Drawer{}` |
| 30 | farm | construction | `Barrel{Wood: w}` for `BarrelWoods` |
| 40 | conduit | construction | for tier 1 to `Tiers`: `EnergyConduit`, `ItemConduit`, `FluidConduit`; then `NetConduit{}` |
| 50 | machine | construction | `Machine{Kind: k}` for `Order` |
| 60 | catalog | construction | `sgitem.Dust{}` |
| 70 | elevator | construction | `Elevator{Colour: c}` for `item.Colours()` |
| 80 | catalog | construction, then equipment | crushed blocks, compact blocks, the hoppers of `hopper.HopperOrder`; stone hammer, hammer, crooks |
| 90 | farm | items | `Silkworm{}`, `CookedSilkworm{}` |
| 100 | catalog | mixed | wands (equipment), wooden bucket, water bucket, drawer key (equipment), the two drawer upgrades, bark fiber, fairy dust |
| 110 | netstore | items | `Disk{Tier: t}` for `DiskCaps`, `Pattern{}`, `Upgrade{}`, `Upgrade{Stack: true}` |
| 120 | catalog | items, then equipment | fairy emerald, dimension stone, the relics |
| 130 | fairy | equipment | `Board{}` |
| 140 | catalog | mixed | fairy bone meal, hourglass, grass seeds, mycelium spores, witch water bucket, binder mix, binder, meshes, ore chunks, ore blocks (construction) with raw ores, pebbles, petals |
| 150 | farm | items | `PlantSeed{Seed: sd}` for `PlantSeeds` |
| 160 | catalog | items, then equipment | essences, metal dusts, ingots, tier alloys, circuits 1 to 4, tier kits 1 to `machine.MaxTier`, config copier (equipment) |

A new feature takes a free number between its neighbours, as with hook priorities. `content/skygear/testdata/pin/creative.golden` shows the resulting order as text.

## 7. Code by task

### T59

`content/skygear/pin_look_test.go`

```go
package skygear

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/fluidnet"
	"github.com/presentkim/skygear/common/waila"
	"github.com/presentkim/skygear/content/skygear/conduit"
	"github.com/presentkim/skygear/content/skygear/farm"
	"github.com/presentkim/skygear/content/skygear/hopper"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/look"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/machinerecipe"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/storage"
)

// pinStack is a stack as a pin line shows it: "-" when empty, else
// "<id>:<meta>x<count>".
func pinStack(st item.Stack) string {
	if st.Empty() {
		return "-"
	}
	name, meta := st.Item().EncodeItem()
	return fmt.Sprintf("%s:%dx%d", name, meta, st.Count())
}

// pinInfo is a panel as one pin line: every field of waila.Info.
func pinInfo(info waila.Info) string {
	icon := "-"
	if info.Item != nil {
		name, meta := info.Item.EncodeItem()
		icon = fmt.Sprintf("%s:%d", name, meta)
	}
	line := "-"
	if l := info.Line; l != nil {
		line = fmt.Sprintf("{%s name=%q text=%q gauge=%d show=%t}", pinStack(l.Item), l.Name, l.Text, l.Gauge, l.ShowGauge)
	}
	cont := "-"
	if c := info.Container; c != nil {
		var items []string
		for _, it := range c.Items {
			items = append(items, pinStack(it))
		}
		cont = fmt.Sprintf("{%s [%s] progress=%s}", c.Layout, strings.Join(items, " "), pinStack(c.Progress))
	}
	var tools []string
	for _, t := range info.Tools {
		tools = append(tools, string(t.Tool)+string(rune(t.Mark)))
	}
	return fmt.Sprintf("name=%q key=%q icon=%s lines=%q line=%s container=%s tools=%v marks=%+v source=%q",
		info.Name, info.NameKey, icon, info.Lines, line, cont, tools, info.Marks, info.Source)
}

// TestPinLook pins the WAILA panel (look.Info) of every kind of block the
// game adds lines to: each machine kind idle, to a standing and to a sneaking
// viewer, the machine states that change the lines, the drawer, the composter,
// the conduit bundle, the silkworm leaves, the barrels, the hoppers, the magic
// plant, the crops and the vanilla containers. The golden was recorded from
// the type switch of look.Info; it changes only when a panel is changed on
// purpose.
func TestPinLook(t *testing.T) {
	// The panel names blocks by the pack's names, as the running server does.
	saved := naming.PackNames
	t.Cleanup(func() { naming.PackNames = saved })
	if err := naming.LoadNames(os.DirFS(packDir)); err != nil {
		t.Fatal(err)
	}
	s := testSession(t)
	progress.LocalOf(s.Session).RecipesFree = true
	loadAround(s)
	h := spawnPlayer(s)
	lines := map[string]string{}
	y := island.IslandY + 6
	next := 0
	// free is a new position above the island, two blocks from the last.
	free := func() cube.Pos {
		next++
		return cube.Pos{2 + (next%6)*2, y + (next/36)*3, 2 + (next/6%6)*2}
	}
	fs := farm.StateOf(s.Session)
	if fs.Barrels == nil {
		fs.Barrels = map[string]*farm.BarrelRecord{}
	}
	if fs.Plants == nil {
		fs.Plants = map[string]*farm.PlantRecord{}
	}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		rec := func(name string, pos cube.Pos, b world.Block) {
			t.Helper()
			if _, dup := lines[name]; dup {
				t.Fatalf("pin scenario %q recorded twice", name)
			}
			lines[name] = pinInfo(look.Info(s.Session, p, tx, pos, b))
		}
		// both records name plain and name/sneak.
		both := func(name string, pos cube.Pos, b world.Block) {
			t.Helper()
			rec(name, pos, b)
			p.StartSneaking()
			rec(name+"/sneak", pos, b)
			p.StopSneaking()
		}
		// placed sets b at a free position and returns the position.
		placed := func(b world.Block) cube.Pos {
			pos := free()
			tx.SetBlock(pos, b, nil)
			return pos
		}

		// Every machine kind, idle.
		for _, k := range machine.Order {
			pos := free()
			both("machine/"+k, pos, machine.New(k))
		}

		// The machine states that change the lines.
		state := func(kind string) (machine.Machine, *machine.State) {
			m := machine.New(kind)
			return m, m.State
		}
		m, st := state(machine.KindCrusher)
		st.Insert(item.NewStack(block.Cobblestone{}, 5))
		st.Progress = st.Def.Interval / 2
		both("state/crusher/working", free(), m)
		m, st = state(machine.KindCrusher)
		_ = st.Inv().SetItem(st.Def.Output[0], item.NewStack(block.Gravel{}, 12))
		rec("state/crusher/output", free(), m)
		m, st = state(machine.KindCrusher)
		st.Progress = st.Def.Interval
		rec("state/crusher/blocked", free(), m)
		m, st = state(machine.KindCrusher)
		st.Boost = 400
		rec("state/crusher/boost", free(), m)
		m, st = state(machine.KindCrusher)
		st.Stalled = true
		rec("state/crusher/stalled", free(), m)
		m, st = state(machine.KindCrusher)
		st.Unlimited = true
		rec("state/crusher/unlimited", free(), m)
		m, st = state(machine.KindCrafter)
		st.SetRecipe(machine.CrafterCandidates(s.Session, machine.Item(machine.KindCrusher))[0])
		both("state/crafter/missing", free(), m)
		m, st = state(machine.KindLab)
		r := machinerecipe.Labs[0]
		st.SetLab(machinerecipe.LabByKind(r.Out.Kind))
		rec("state/lab/missing", free(), m)
		fillLab(st, r, 2)
		rec("state/lab/filled", free(), m)
		m, st = state(machine.KindGenerator)
		st.Energy, st.Burn = 300, 40
		rec("state/generator/burning", free(), m)
		m, st = state(machine.KindEnergyCell)
		st.Energy = 1234
		rec("state/energy_cell/charged", free(), m)
		m, st = state(machine.KindTank)
		st.Fluid, st.FluidAmt = fluidnet.Water, 3000
		rec("state/tank/water", free(), m)
		m, st = state(machine.KindCrucible)
		st.Insert(item.NewStack(block.Cobblestone{}, 3))
		st.Fluid, st.FluidAmt = fluidnet.Lava, 500
		rec("state/crucible/melting", free(), m)
		m, st = state(machine.KindLavaGenerator)
		st.Fluid, st.FluidAmt, st.Energy = fluidnet.Lava, 2000, 77
		rec("state/lava_generator/fuelled", free(), m)
		m, st = state(machine.KindMagicSieve)
		st.Insert(item.NewStack(block.Gravel{}, 4))
		st.SetMob("zombie")
		st.Charges = 3
		rec("state/magic_sieve/charged", free(), m)
		m, st = state(machine.KindMiner)
		st.Progress = st.Def.Interval / 2
		rec("state/miner/working", free(), m)
		m, st = state(machine.KindHarvester)
		st.Progress = st.Def.Interval / 2
		rec("state/harvester/working", free(), m)

		// A drawer controller counts the drawers of its set.
		ctrl := free()
		tx.SetBlock(ctrl, machine.New(machine.KindDrawerController), nil)
		tx.SetBlock(ctrl.Side(cube.FaceEast), storage.Drawer{Facing: cube.South, Item: item.NewStack(block.Dirt{}, 1), Count: 70}, nil)
		next++ // the drawer took the next column
		rec("state/drawer_controller/set", ctrl, tx.Block(ctrl))

		// The drawer.
		both("drawer/empty", free(), storage.Drawer{Facing: cube.South})
		both("drawer/dirt", free(), storage.Drawer{Facing: cube.South, Item: item.NewStack(block.Dirt{}, 1), Count: 70})
		both("drawer/upgraded", free(), storage.Drawer{Facing: cube.South, Item: item.NewStack(block.Dirt{}, 1), Count: 5000, Capacity: 2, Void: true, Locked: true})

		// The composter.
		pos := placed(block.Composter{})
		rec("composter/empty", pos, tx.Block(pos))
		pos = placed(block.Composter{})
		farm.SetComposterCount(s.Session, pos, 5)
		rec("composter/plants", pos, tx.Block(pos))
		pos = placed(block.Composter{Level: farm.ComposterLevels})
		rec("composter/level", pos, tx.Block(pos))

		// The conduit bundle.
		chest := free()
		next++ // the bundle stands east of the chest
		pos = chestWithBundle(tx, chest, 0, 2, 3)
		rec("bundle/item", pos, tx.Block(pos))
		pos = placed(conduit.Bundle{Energy: 1, Fluid: 2, Net: 1})
		both("bundle/all", pos, tx.Block(pos))

		// The silkworm leaves.
		rec("leaves/none", free(), farm.SilkwormLeaves{})
		rec("leaves/full", free(), farm.SilkwormLeaves{Cocoon: farm.CocoonMax})

		// The barrels.
		pos = placed(farm.Barrel{Wood: "oak"})
		rec("barrel/empty", pos, tx.Block(pos))
		pos = placed(farm.Barrel{Wood: "oak"})
		fs.Barrels[farm.CropKey(pos)] = &farm.BarrelRecord{Count: 3}
		rec("barrel/plants", pos, tx.Block(pos))
		pos = placed(farm.Barrel{Wood: "oak"})
		fs.Barrels[farm.CropKey(pos)] = &farm.BarrelRecord{Count: farm.MatterCapacity, Water: true}
		rec("barrel/water", pos, tx.Block(pos))
		pos = placed(farm.Barrel{Wood: "oak"})
		fs.Barrels[farm.CropKey(pos)] = &farm.BarrelRecord{Count: farm.MatterCapacity, Water: true, Clay: 2}
		rec("barrel/clay", pos, tx.Block(pos))
		pos = placed(farm.Barrel{Wood: "oak"})
		fs.Barrels[farm.CropKey(pos)] = &farm.BarrelRecord{Count: farm.MatterCapacity, Water: true, Witch: true}
		rec("barrel/witch", pos, tx.Block(pos))
		pos = placed(farm.Barrel{Wood: farm.StoneWood})
		rec("barrel/stone", pos, tx.Block(pos))
		pos = placed(farm.Barrel{Wood: farm.StoneWood})
		fs.Barrels[farm.CropKey(pos)] = &farm.BarrelRecord{LavaMB: 400}
		rec("barrel/stoneLava", pos, tx.Block(pos))

		// The hoppers.
		for _, id := range hopper.HopperOrder {
			rec("hopper/"+id+"/empty", free(), hopper.NewFlowHopper(id, cube.FaceDown))
		}
		fh := hopper.NewFlowHopper("wood_hopper", cube.FaceDown)
		_, _ = fh.Inv.AddItem(item.NewStack(block.Dirt{}, 7))
		rec("hopper/wood_hopper/dirt", free(), fh)

		// The magic plant: a bud with its time, a bud nobody recorded, a bloom.
		pos = placed(farm.MagicPlant{Kind: "ore_iron"})
		fs.Plants[farm.CropKey(pos)] = &farm.PlantRecord{Kind: "ore_iron", ReadyAt: 90_000_000_000}
		rec("plant/bud", pos, tx.Block(pos))
		rec("plant/unrecorded", free(), farm.MagicPlant{Kind: "ore_iron"})
		rec("plant/bloom", free(), farm.MagicPlant{Kind: "ore_iron", Bloom: true})
		rec("plant/unknown", free(), farm.MagicPlant{Kind: "no_such_plant"})

		// The crops: one planted (its time), one grown, one nobody recorded.
		for _, c := range []struct {
			name string
			b    world.Block
		}{
			{"wheat", block.WheatSeeds{}}, {"carrot", block.Carrot{}}, {"potato", block.Potato{}},
			{"beetroot", block.BeetrootSeeds{}}, {"melon", block.MelonSeeds{}}, {"pumpkin", block.PumpkinSeeds{}},
		} {
			pos = free()
			tx.SetBlock(pos.Side(cube.FaceDown), block.Farmland{Hydration: 7}, nil)
			tx.SetBlock(pos, c.b, nil)
			farm.PlantCrop(s.Session, pos, c.name)
			rec("crop/"+c.name+"/young", pos, tx.Block(pos))
			tx.SetBlock(pos, farm.WithGrowth(c.b, farm.MaxCropStage), nil)
			rec("crop/"+c.name+"/grown", pos, tx.Block(pos))
		}
		rec("crop/wheat/unrecorded", free(), block.WheatSeeds{})

		// Blocks the game adds no lines to, and the vanilla containers.
		rec("vanilla/stone", free(), block.Stone{})
		rec("vanilla/fire", free(), block.Fire{})
		pos = placed(block.NewChest())
		_, _ = tx.Block(pos).(block.Chest).Inventory(tx, pos).AddItem(item.NewStack(block.Dirt{}, 20))
		rec("vanilla/chest", pos, tx.Block(pos))
		pos = placed(block.NewFurnace(cube.North))
		rec("vanilla/furnace", pos, tx.Block(pos))
		pos = placed(block.NewBlastFurnace(cube.North))
		rec("vanilla/blast_furnace", pos, tx.Block(pos))
		pos = placed(block.NewSmoker(cube.North))
		rec("vanilla/smoker", pos, tx.Block(pos))
		pos = placed(block.NewHopper())
		rec("vanilla/hopper", pos, tx.Block(pos))
		pos = placed(block.NewBrewingStand())
		rec("vanilla/brewing_stand", pos, tx.Block(pos))
		pos = placed(block.NewBarrel())
		rec("vanilla/barrel", pos, tx.Block(pos))
	})
	pinGolden(t, "testdata/pin/look.golden", lines)
}
```


`content/skygear/pin_creative_test.go`

```go
package skygear

import (
	"fmt"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/item/creative"
	"github.com/presentkim/skygear/content/skygear/catalog"
)

// TestPinCreative pins the game's part of the creative inventory as text: one
// line per item, in the order the client lists them, with its group. The
// client-data golden of the wire package pins the same order as hashes; this
// one shows which item moved. It changes only when the creative list is
// changed on purpose.
func TestPinCreative(t *testing.T) {
	catalog.RegisterCreativeItems()
	lines := map[string]string{}
	n := 0
	for _, c := range creative.Items() {
		if !strings.HasPrefix(c.Group, "anon_sg_") {
			continue
		}
		name, meta := c.Stack.Item().EncodeItem()
		lines[fmt.Sprintf("%04d", n)] = fmt.Sprintf("%s %s:%d", c.Group, name, meta)
		n++
	}
	if n == 0 {
		t.Fatal("the game added no creative item")
	}
	pinGolden(t, "testdata/pin/creative.golden", lines)
}
```


Appended to `content/skygear/pin_bench_test.go` (add `look` and `machine` to its imports):

```go
// BenchmarkLookMachine updates the WAILA panel of a player who looks at a
// working crusher: the trace to the block, the panel's lines and the title.
func BenchmarkLookMachine(b *testing.B) {
	s := benchSession(b)
	h := spawnPlayer(s)
	target := cube.Pos{4, island.IslandY + 3, 8}
	m := machine.New(machine.KindCrusher)
	m.State.Insert(item.NewStack(block.Cobblestone{}, 5))
	m.State.Progress = m.State.Def.Interval / 2
	b.ReportAllocs()
	withPlayer(b, s, h, func(p *player.Player, tx *world.Tx) {
		tx.SetBlock(target, m, nil)
		pos, rot := eyeAt(p, target, 3)
		if at, _, ok := look.LookedAt(p, tx, pos, rot); !ok || at != target {
			b.Fatalf("the player looks at %v, want the crusher at %v", at, target)
		}
		b.ResetTimer()
		for range b.N {
			look.Update(s.Session, p, tx, pos, rot, true)
		}
	})
}

// BenchmarkLookInfoMachine builds the panel of a working crusher (look.Info).
func BenchmarkLookInfoMachine(b *testing.B) {
	s := benchSession(b)
	h := spawnPlayer(s)
	target := cube.Pos{4, island.IslandY + 3, 8}
	m := machine.New(machine.KindCrusher)
	m.State.Insert(item.NewStack(block.Cobblestone{}, 5))
	m.State.Progress = m.State.Def.Interval / 2
	b.ReportAllocs()
	withPlayer(b, s, h, func(p *player.Player, tx *world.Tx) {
		b.ResetTimer()
		for range b.N {
			look.Info(s.Session, p, tx, target, m)
		}
	})
}

// BenchmarkLookInfoStone builds the panel of a block the game adds nothing to.
func BenchmarkLookInfoStone(b *testing.B) {
	s := benchSession(b)
	h := spawnPlayer(s)
	target := cube.Pos{4, island.IslandY + 3, 8}
	b.ReportAllocs()
	withPlayer(b, s, h, func(p *player.Player, tx *world.Tx) {
		b.ResetTimer()
		for range b.N {
			look.Info(s.Session, p, tx, target, block.Stone{})
		}
	})
}
```


Record both goldens once, from the unchanged code: `SKYGEAR_GOLDEN=update go test ./content/skygear/ -run 'TestPinLook$|TestPinCreative$' -count=1`. Then run both without the variable in 5 separate processes, once with `-race`, and once inside the whole package (`go test ./content/skygear/ -count=1`): every run passes. The files must have 143 and 243 lines.

### T60

`common/waila/provider.go`

```go
package waila

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

// View is who looks at a block and where the block is.
type View struct {
	// Lang is the viewer's language.
	Lang string
	// Sneaking is whether the viewer sneaks: blocks add their details then.
	Sneaking bool
	// Tx and Pos are the transaction and the position of the block.
	Tx  *world.Tx
	Pos cube.Pos
}

// Provider is a block that fills in its own panel.
type Provider interface {
	// Waila returns info, which holds the block's name, icon, tools and the
	// contents of a vanilla container, with what the block adds to it.
	Waila(v View, info Info) Info
}

// Adapter fills in the panel of blocks that have no Waila method of their
// own, such as vanilla blocks; it returns info unchanged for any other block.
type Adapter func(v View, b world.Block, info Info) Info

var adapters []Adapter

// RegisterAdapter adds an adapter. Adapters run in the order they were added,
// after the block's own Waila.
func RegisterAdapter(a Adapter) { adapters = append(adapters, a) }

// Describe returns info with what block b adds to its panel: its Waila if it
// is a Provider, then every adapter.
func Describe(v View, b world.Block, info Info) Info {
	if p, ok := b.(Provider); ok {
		info = p.Waila(v, info)
	}
	for _, a := range adapters {
		info = a(v, b, info)
	}
	return info
}
```


`common/waila/provider_test.go`

```go
package waila

import (
	"slices"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/world"
)

// described is a block with a panel of its own.
type described struct{ block.Stone }

func (described) Waila(v View, info Info) Info {
	info.Lines = append(info.Lines, "own:"+v.Lang)
	return info
}

// withAdapters runs f with only the given adapters registered.
func withAdapters(t *testing.T, as []Adapter, f func()) {
	t.Helper()
	saved := adapters
	t.Cleanup(func() { adapters = saved })
	adapters = nil
	for _, a := range as {
		RegisterAdapter(a)
	}
	f()
}

func appendLine(line string) Adapter {
	return func(_ View, _ world.Block, info Info) Info {
		info.Lines = append(info.Lines, line)
		return info
	}
}

// TestDescribeProviderThenAdapters: a block's own Waila runs first, then the
// adapters in the order they were registered.
func TestDescribeProviderThenAdapters(t *testing.T) {
	withAdapters(t, []Adapter{appendLine("a1"), appendLine("a2")}, func() {
		got := Describe(View{Lang: "ko"}, described{}, Info{Name: "n"})
		if want := []string{"own:ko", "a1", "a2"}; !slices.Equal(got.Lines, want) || got.Name != "n" {
			t.Errorf("lines %q name %q, want %q and n", got.Lines, got.Name, want)
		}
	})
}

// TestDescribePlainBlock: a block that is no Provider gets only the adapters,
// and with none its panel is unchanged.
func TestDescribePlainBlock(t *testing.T) {
	withAdapters(t, nil, func() {
		got := Describe(View{}, block.Stone{}, Info{Name: "n", Lines: []string{"x"}})
		if got.Name != "n" || !slices.Equal(got.Lines, []string{"x"}) {
			t.Errorf("panel %+v changed", got)
		}
	})
	withAdapters(t, []Adapter{appendLine("a1")}, func() {
		if got := Describe(View{}, block.Stone{}, Info{}); !slices.Equal(got.Lines, []string{"a1"}) {
			t.Errorf("lines %q, want the adapter's", got.Lines)
		}
	})
}

// TestDescribeAllocatesNothing: asking a block and the adapters allocates
// nothing by itself.
func TestDescribeAllocatesNothing(t *testing.T) {
	keep := func(_ View, _ world.Block, info Info) Info { return info }
	withAdapters(t, []Adapter{keep, keep}, func() {
		var b world.Block = block.Stone{}
		info := Info{Name: "n"}
		if n := testing.AllocsPerRun(200, func() { info = Describe(View{}, b, info) }); n != 0 {
			t.Errorf("Describe allocates %v times, want 0", n)
		}
	})
}
```


`content/skygear/look/look.go`, in `Info`, two edits and nothing else:

- after the line `info.Container = containerOf(tx, pos, b)` add `view := waila.View{Lang: lang, Sneaking: p.Sneaking(), Tx: tx, Pos: pos}`;
- at the end of the `switch b := b.(type)` (after the `case farm.MagicPlant:` clause) add the clause `default:` with the one statement `info = waila.Describe(view, b, info)`.

### T61

`content/skygear/hopper/look.go`

```go
package hopper

import "github.com/presentkim/skygear/common/waila"

// Waila adds a hopper's lines to its panel (waila.Provider); what it holds
// shows like a chest's, slots with counts under them.
func (h FlowHopper) Waila(v waila.View, info waila.Info) waila.Info {
	info.Lines = HopperLines(v.Lang, h)
	if h.Inv != nil {
		info.Container = &waila.Container{Layout: waila.LayoutChest, Items: h.Inv.Items()}
	}
	return info
}
```


`content/skygear/conduit/look.go`

```go
package conduit

import "github.com/presentkim/skygear/common/waila"

// Waila adds a bundle's lines to its panel (waila.Provider).
func (b Bundle) Waila(v waila.View, info waila.Info) waila.Info {
	info.Lines = BundleLines(v.Lang, v.Tx, v.Pos, b)
	return info
}
```


`look.go`: delete the clauses `case conduit.Bundle:` and `case hopper.FlowHopper:` and the imports of `conduit` and `hopper`.

### T62

`content/skygear/storage/look.go`

```go
package storage

import (
	"fmt"
	"math"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/waila"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/textfmt"
)

// Waila fills in a drawer's panel (waila.Provider). The drawer stays the
// icon; the held item shows like a chest's, a slot with its count under it;
// the lock and upgrades show on the drawer's front and, while the viewer
// sneaks, in the panel.
func (d Drawer) Waila(v waila.View, info waila.Info) waila.Info {
	if d.Item.Empty() {
		info.Lines = []string{i18n.T(v.Lang, "look.empty")}
	} else {
		info.Container = &waila.Container{Layout: waila.LayoutChest, Items: []item.Stack{d.Item.Grow(int(min(d.Count, math.MaxInt32)) - d.Item.Count())}}
	}
	if v.Sneaking {
		info.Lines = append(info.Lines, drawerDetail(v.Lang, d)...)
	}
	return info
}

// drawerDetail are the lines a sneaking viewer gets of a drawer: its upgrades
// and lock.
func drawerDetail(lang string, d Drawer) []string {
	lines := []string{fmt.Sprintf(i18n.T(lang, "look.drawer.capacity"), d.Capacity, sgitem.MaxCapacityUpgrades, textfmt.Int(d.MaxItems()))}
	if d.Void {
		lines = append(lines, i18n.T(lang, "look.drawer.void"))
	}
	if d.Locked {
		lines = append(lines, i18n.T(lang, "look.drawer.locked"))
	}
	return lines
}
```


`look.go`: delete the clause `case storage.Drawer:`, the function `drawerDetail`, and the imports `math` and `sgitem`. The import of `storage` stays (the machine clause uses `storage.Lines`).

### T63

`content/skygear/farm/look.go`

```go
package farm

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/waila"
	"github.com/presentkim/skygear/content/skygear/core"
)

// Waila adds how full of cocoons the leaves are to their panel
// (waila.Provider).
func (l SilkwormLeaves) Waila(v waila.View, info waila.Info) waila.Info {
	info.Lines = []string{fmt.Sprintf(i18n.T(v.Lang, "look.cocoon"), l.CocoonPercent())}
	return info
}

// Waila adds a barrel's contents to its panel (waila.Provider): its lines and
// its liquid on the icon line.
func (b Barrel) Waila(v waila.View, info waila.Info) waila.Info {
	s := core.At(v.Tx)
	if s == nil {
		return info
	}
	info.Lines = BarrelLines(s, v.Lang, v.Tx, v.Pos)
	info.Line = BarrelLiquid(s, v.Lang, v.Pos)
	return info
}

// Waila fills in a magic plant's panel (waila.Provider): its seed as the
// name and the icon, and its time or yield.
func (b MagicPlant) Waila(v waila.View, info waila.Info) waila.Info {
	i := PlantKindIndex(b.Kind)
	if i < 0 {
		return info
	}
	k := PlantKinds[i]
	seed := k.SeedStack()
	info.Item, info.NameKey = seed.Item(), ""
	info.Name = i18n.T(v.Lang, "pseed."+k.Seed)
	info.Tools = []waila.ToolIcon{{Tool: waila.Crops, Mark: waila.MarkNone}}
	if s := core.At(v.Tx); s != nil {
		info.Lines = PlantLines(s, v.Lang, v.Pos, b, k)
	}
	return info
}

// registerLook fills in the panels of the vanilla blocks the farm runs: the
// composter's contents, and a crop's name and time.
func registerLook() {
	waila.RegisterAdapter(func(v waila.View, b world.Block, info waila.Info) waila.Info {
		if c, ok := b.(block.Composter); ok {
			if s := core.At(v.Tx); s != nil {
				info.Lines = ComposterLines(s, v.Lang, v.Pos, c)
			}
		}
		if kind, _, ok := CropOf(b); ok {
			info.Name = i18n.T(v.Lang, CropNameKeys[kind])
			info.Tools = []waila.ToolIcon{{Tool: waila.Crops, Mark: waila.MarkNone}}
			if s := core.At(v.Tx); s != nil {
				if r, ok := StateOf(s).Crops[CropKey(v.Pos)]; ok {
					info.Lines = CropLines(s, v.Lang, r, v.Tx, v.Pos)
				}
			}
		}
		return info
	})
}
```


`farm/subscribe.go`: `subscribe()` calls `registerLook()` as its last statement, after `registerComposterIO()`.

`look.go`: delete the clauses `case block.Composter:`, `case farm.SilkwormLeaves:`, `case farm.Barrel:`, `case farm.MagicPlant:` and the whole `if kind, _, ok := farm.CropOf(b); ok { ... }` block after the switch; in `LookedAt` replace the two lines `if _, stage, crop := farm.CropOf(b); crop {` and `h := float64(stage+1) / float64(farm.MaxCropStage+1)` by the one line `if h, crop := CropHeight(b); crop {`; add, above `Update`:

```go
// maxCropGrowth is the last growth stage of a vanilla crop.
const maxCropGrowth = 7

// CropHeight is how high the model of a vanilla crop stands in its cell at
// its growth stage (a fraction of the cell), and whether b is such a crop.
func CropHeight(b world.Block) (float64, bool) {
	var growth int
	switch c := b.(type) {
	case block.WheatSeeds:
		growth = c.Growth
	case block.Carrot:
		growth = c.Growth
	case block.Potato:
		growth = c.Growth
	case block.BeetrootSeeds:
		growth = c.Growth
	case block.MelonSeeds:
		growth = c.Growth
	case block.PumpkinSeeds:
		growth = c.Growth
	default:
		return 0, false
	}
	return float64(growth+1) / float64(maxCropGrowth+1), true
}
```


Delete the import of `farm`. Appended to `content/skygear/look_test.go`:

```go
// TestLookCropsMatchFarm: the look trace knows the height of exactly the
// crops the farm grows, from an eighth of the cell at stage 0 to all of it
// at the last stage.
func TestLookCropsMatchFarm(t *testing.T) {
	for _, b := range []world.Block{
		block.WheatSeeds{}, block.Carrot{}, block.Potato{}, block.BeetrootSeeds{}, block.MelonSeeds{}, block.PumpkinSeeds{},
		block.Stone{}, block.Farmland{}, block.Sapling{Type: block.OakSapling()}, farm.MagicPlant{Kind: "ore_iron"},
	} {
		_, _, crop := farm.CropOf(b)
		if _, ok := look.CropHeight(b); ok != crop {
			t.Errorf("%T: look.CropHeight says %t, farm.CropOf says %t", b, ok, crop)
		}
	}
	for stage, want := range map[int]float64{0: 0.125, 3: 0.5, farm.MaxCropStage: 1} {
		if h, _ := look.CropHeight(farm.WithGrowth(block.WheatSeeds{}, stage)); h != want {
			t.Errorf("wheat at stage %d is %v high, want %v", stage, h, want)
		}
	}
}
```


### T64

The whole new file. Its functions from `FurnaceArrow` down are the functions of `look.go` moved as they are, with the `machine.` qualifier dropped and `dimension.MobName(lang, st.Mob)` replaced by `i18n.T(lang, "mob."+st.Mob)`.

`content/skygear/machine/look.go`

```go
package machine

import (
	"fmt"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/waila"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/textfmt"
)

// Waila fills in a machine's panel (waila.Provider): its lines and item
// line, the time boost left, the lines of a kind another package describes
// (Def.Lines), what waits in its output as slots, and the cycle figures while
// the viewer sneaks. A sneaking look also shows the machine's range for a
// moment (PeekRange).
func (m Machine) Waila(v waila.View, info waila.Info) waila.Info {
	if v.Sneaking {
		PeekRange(v.Tx, v.Pos, m)
	}
	info.Lines, info.Line = MachineLines(v.Lang, m)
	if l := BoostLine(v.Lang, m.State); l != "" {
		info.Lines = append(info.Lines, l)
	}
	if m.State != nil && m.State.Def.Lines != nil {
		info.Lines = m.State.Def.Lines(v.Lang, v.Tx, v.Pos)
	}
	if c := outputContainer(m.State); c != nil {
		info.Container = c
	}
	if v.Sneaking && m.State != nil {
		info.Lines = append(info.Lines, detailLines(v.Lang, m.State)...)
	}
	return info
}

// BoostLine is the panel line of the time boost a machine has left, or "".
func BoostLine(lang string, st *State) string {
	if st == nil || st.Boost <= 0 {
		return ""
	}
	left := time.Duration(st.Boost) * time.Second / 20
	return i18n.T(lang, "fairy.hourglass.left") + textfmt.Duration(lang, left.Round(time.Second))
}

// registerLook gives the vanilla furnaces' panels the machines' progress
// arrow.
func registerLook() {
	waila.RegisterAdapter(func(_ waila.View, b world.Block, info waila.Info) waila.Info {
		switch b.(type) {
		case block.Furnace, block.BlastFurnace, block.Smoker:
			if info.Container != nil {
				info.Container.Progress = FurnaceArrow(b)
			}
		}
		return info
	})
}

// FurnaceArrow is the progress arrow of a furnace, blast furnace or smoker:
// filled to how far the item is cooked while it burns, empty when it does
// not.
func FurnaceArrow(b world.Block) item.Stack {
	var (
		lit         bool
		cook        time.Duration
		requirement = 5 * time.Second
	)
	switch f := b.(type) {
	case block.Furnace:
		lit, requirement = f.Lit, 10*time.Second
		_, _, cook = f.Durations()
	case block.BlastFurnace:
		lit = f.Lit
		_, _, cook = f.Durations()
	case block.Smoker:
		lit = f.Lit
		_, _, cook = f.Durations()
	default:
		return item.Stack{}
	}
	step := 0
	if lit {
		step = min(int(cook*10/requirement), 10)
	}
	return item.NewStack(ProgressArrow{Step: step}, 1)
}

// MachineLines are a machine's lines and item line: what it holds, with
// its icon and progress as a ring gauge ("흙 [icon] ×3 ◔"), or grey
// "비어 있음" when empty. What waits in its output shows as slots
// (outputContainer).
func MachineLines(lang string, m Machine) ([]string, *waila.ItemLine) {
	st := m.State
	if st == nil {
		return nil, nil
	}
	step := min(10, st.Progress*10/st.CycleLen())
	gauge := textfmt.GaugeGlyph(step)
	var (
		lines []string
		line  *waila.ItemLine
	)
	switch {
	case st.Def.Crafter:
		if st.Recipe == nil {
			return crafterLines(lang, st), nil
		}
		return crafterLines(lang, st)[1:], itemLine(lang, st.Recipe.Out)
	case st.Def == Kinds[KindLab]:
		// The research it makes shows with its icon on the item line.
		lines := LabLines(lang, st)
		if st.Lab == nil {
			return lines, nil
		}
		return lines[1:], itemLine(lang, item.NewStack(st.Lab.Out, st.Lab.OutCount))
	case st.Def == Kinds[KindHarvester]:
		lines = append(lines, i18n.T(lang, "look.machine.harvestCycle")+gauge)
	case st.Def == Kinds[KindGenerator]:
		return GeneratorLines(lang, st), nil
	case IsNatureGenerator(m.Kind):
		return NatureLines(lang, st), nil
	case m.Kind == KindDrawerTrim, m.Kind == KindTinker, m.Kind == KindRelicTable:
		return nil, nil
	case st.Def.Controller:
		return nil, nil
	case st.Def.FluidCap > 0:
		lines := FluidLines(lang, st)
		if _, in, ok := st.FirstInput(); ok {
			line = itemLine(lang, in)
		}
		if st.Def.Generates > 0 {
			lines = append(lines, fmt.Sprintf(i18n.T(lang, "generator.lines.energy"), st.Energy, st.EnergyCap()))
		} else {
			lines = append(lines, ConsumerLines(lang, st)...)
		}
		return lines, line
	case st.Def.Cell:
		return []string{fmt.Sprintf(i18n.T(lang, "generator.lines.energy"), st.Energy, st.EnergyCap())}, nil
	case st.Def == Kinds[KindMiner]:
		// Its output shows as a slot.
		return append(append([]string{i18n.T(lang, "look.machine.mining") + gauge}, ConsumerLines(lang, st)...), BlockedLines(lang, st)...), nil
	default:
		if _, it, ok := st.FirstInput(); ok {
			line = &waila.ItemLine{Item: it, Name: naming.ItemName(lang, it), Gauge: step, ShowGauge: true}
			if it.Count() > 1 {
				line.Text = fmt.Sprintf("§f×%d", it.Count())
			}
		} else {
			lines = append(lines, i18n.T(lang, "look.empty"))
		}
	}
	if len(st.Def.Charge) > 0 && st.Charges > 0 {
		lines = append(lines, fmt.Sprintf(i18n.T(lang, "magicsieve.lines.charged"), i18n.T(lang, "mob."+st.Mob), st.Charges))
	}
	lines = append(lines, ConsumerLines(lang, st)...)
	lines = append(lines, BlockedLines(lang, st)...)
	return lines, line
}

// detailLines are the figures the panel adds while its viewer sneaks: the
// length of a work cycle with the ticks left (the energy is in the lines).
func detailLines(lang string, st *State) []string {
	var lines []string
	if cycle := st.CycleLen(); cycle > 1 && len(st.Def.Input)+len(st.Def.Output) > 0 && !st.Def.Crafter {
		if st.Progress > 0 {
			lines = append(lines, fmt.Sprintf(i18n.T(lang, "look.detail.cycle"), cycle, max(0, cycle-st.Progress)))
		} else {
			lines = append(lines, fmt.Sprintf(i18n.T(lang, "look.detail.cycleOnly"), cycle))
		}
	}
	return lines
}

// crafterLines say what the auto crafter does: its recipe (the first line),
// what it lacks.
func crafterLines(lang string, st *State) []string {
	if st.Recipe == nil {
		return []string{i18n.T(lang, "crafter.status.none")}
	}
	lines := []string{"§f" + naming.ItemName(lang, st.Recipe.Out)}
	if _, ok := st.CrafterPlan(st.Recipe); !ok {
		lines = append(lines, fmt.Sprintf(i18n.T(lang, "crafter.status.missing"), CrafterMissing(lang, st)))
	} else if !Fits(st.Inv(), st.Def.Output, []item.Stack{st.Recipe.Out}) {
		lines = append(lines, i18n.T(lang, "look.blocked"))
	} else {
		lines = append(lines, i18n.T(lang, "crafter.status.working"))
	}
	return append(lines, ConsumerLines(lang, st)...)
}

// outputContainer is the output slots of the machine st as a chest's slots,
// each item with its count under it, or nil for a machine with no output or
// a drawer controller (whose lines count the drawers instead), and nil while
// the output is empty.
func outputContainer(st *State) *waila.Container {
	if st == nil || st.Def.Controller || len(st.Def.Output) == 0 {
		return nil
	}
	var items []item.Stack
	for _, slot := range st.Def.Output {
		if it, _ := st.Inv().Item(slot); !it.Empty() {
			items = append(items, it)
		}
	}
	if len(items) == 0 {
		return nil
	}
	return &waila.Container{Layout: waila.LayoutChest, Items: items}
}

// itemLine is the line of item it with its icon, its count after it.
func itemLine(lang string, it item.Stack) *waila.ItemLine {
	line := &waila.ItemLine{Item: it, Name: naming.ItemName(lang, it)}
	if it.Count() > 1 {
		line.Text = fmt.Sprintf("§f×%d", it.Count())
	}
	return line
}
```


Other edits of T64:

- `machine/machine.go`, at the end of `type Def struct`, after `Accepts`: the field `Lines func(lang string, tx *world.Tx, pos cube.Pos) []string` with the comment `// Lines, if set, are the kind's panel lines in place of the machine's own (Machine.Waila).`
- `machine/subscribe.go`: `subscribe()` calls `registerLook()` as its last statement.
- `storage/storage.go`, in the `init` loop over the two controller kinds, after the line that sets `def.Insert, def.Peek, def.Extract`: `def.Lines = Lines`.
- `fairy/fairytime.go`: delete `HourglassLine` and the imports it alone used (`common/i18n`, `textfmt`).
- `look.go`: delete the line `view := waila.View{...}` of T60, and replace the whole `switch b := b.(type) { ... }` and the `return info` after it by `return waila.Describe(waila.View{Lang: lang, Sneaking: p.Sneaking(), Tx: tx, Pos: pos}, b, info)`; delete `outputContainer`, `itemLine`, `FurnaceArrow`, `MachineLines`, `detailLines`, `crafterLines`; in `containerOf` the last line becomes `return &waila.Container{Layout: layout, Items: inv.Slots()}`; in `showLook` delete the three lines `if p.Sneaking() { machine.PeekRange(tx, at, b) }`; delete the imports `fmt`, `dimension`, `fairy`, `machine`, `storage`, `textfmt`; the doc comment of `Info` becomes the four lines it has in the file of T65 below.
- Call sites in tests, nothing else in them: `autoequip_test.go` `look.MachineLines` becomes `machine.MachineLines` (three times; drop the `look` import there); `look_test.go` `look.FurnaceArrow` becomes `machine.FurnaceArrow` (twice); `fairytime_test.go` `fairy.HourglassLine` becomes `machine.BoostLine`.

### T65

`look.go` as it is after T65, whole. T65 itself changes only: the imports (`expand`, `tinker`, `wand` go), the new `Hidden`, `Target`, `follower`, `followers`, `Follow`, `Followers`, `follow`, and the bodies of `Update`, `updateLookAt` and `UpdateAll`. If anything else differs from the file T64 left, stop and report.

`content/skygear/look/look.go`

```go
// Package look is the game's WAILA panel (common/waila): it shows the block
// a player looks at, with its name, icon, tools and the game's own lines
// (machine progress, drawer contents, crop time, barrel contents); drawers
// and containers also show their item or contents in the slots.
package look

import (
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/cube/trace"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/waila"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/naming"
)

// Reach is how far a player looks for a block, in blocks.
const Reach = 6

// lookInterval limits how often moving or turning updates the panel.
const lookInterval = 100 * time.Millisecond

// updateAllSkip is how recent a panel update makes UpdateAll leave a player be.
const updateAllSkip = 50 * time.Millisecond

// LookedAt returns the block a player at pos with rotation rot looks at.
// Crops count where the view crosses their height (it grows with the
// stage; the client targets what is behind the empty space above a low
// crop), other blocks without a collision box (flowers)
// anywhere in their cell; air and liquids do not count.
func LookedAt(p *player.Player, tx *world.Tx, pos mgl64.Vec3, rot cube.Rotation) (cube.Pos, world.Block, bool) {
	eye := pos.Add(mgl64.Vec3{0, p.EyeHeight(), 0})
	end := eye.Add(rot.Vec3().Mul(Reach))
	var (
		hit   cube.Pos
		found world.Block
	)
	trace.TraverseBlocks(eye, end, func(pos cube.Pos) bool {
		b := tx.Block(pos)
		if _, air := b.(block.Air); air {
			return true
		}
		if _, liquid := b.(world.Liquid); liquid {
			return true
		}
		if h, crop := CropHeight(b); crop {
			if !trace.BBoxIntersects(cube.Box(0, 0, 0, 1, h, 1).Translate(pos.Vec3()), eye, end) {
				return true
			}
		} else if len(b.Model().BBox(pos, tx)) > 0 && !trace.BlockIntersects(pos, tx, b, eye, end) {
			return true
		}
		hit, found = pos, b
		return false
	})
	return hit, found, found != nil
}

// maxCropGrowth is the last growth stage of a vanilla crop.
const maxCropGrowth = 7

// CropHeight is how high the model of a vanilla crop stands in its cell at
// its growth stage (a fraction of the cell), and whether b is such a crop.
func CropHeight(b world.Block) (float64, bool) {
	var growth int
	switch c := b.(type) {
	case block.WheatSeeds:
		growth = c.Growth
	case block.Carrot:
		growth = c.Growth
	case block.Potato:
		growth = c.Growth
	case block.BeetrootSeeds:
		growth = c.Growth
	case block.MelonSeeds:
		growth = c.Growth
	case block.PumpkinSeeds:
		growth = c.Growth
	default:
		return 0, false
	}
	return float64(growth+1) / float64(maxCropGrowth+1), true
}

// Hidden reports whether p's panel is hidden: the panel is cleared and not
// updated while it is. The feature that hides the panel sets it.
var Hidden = func(s *core.Session, p *player.Player) bool { return false }

// Target is what a player looks at after its panel was updated.
type Target struct {
	S  *core.Session
	P  *player.Player
	Tx *world.Tx
	// Pos is the block looked at, if Ok.
	Pos cube.Pos
	Ok  bool
	// Tick is true on the update of every player on the fast tick and false
	// on the update a player's own movement makes.
	Tick bool
}

type follower struct {
	name string
	f    func(Target)
}

var followers []follower

// Follow adds f, which runs after every panel update, in the order added.
// name is "<feature>.<step>".
func Follow(name string, f func(Target)) {
	followers = append(followers, follower{name, f})
}

// Followers are the names of the followers in the order they run.
func Followers() []string {
	out := make([]string, len(followers))
	for i, fl := range followers {
		out[i] = fl.name
	}
	return out
}

func follow(t Target) {
	for i := range followers {
		followers[i].f(t)
	}
}

// Update updates p's panel for a player at pos with rotation rot (the
// move handler passes where the player is moving to), unless it was
// updated less than lookInterval ago (force: always).
func Update(s *core.Session, p *player.Player, tx *world.Tx, pos mgl64.Vec3, rot cube.Rotation, force bool) {
	now := time.Now()
	if !force && now.Sub(lastLook.Of(s)[p.H()]) < lookInterval {
		return
	}
	lastLook.Of(s)[p.H()] = now
	if Hidden(s, p) {
		waila.Clear(p)
		return
	}
	at, b, ok := LookedAt(p, tx, pos, rot)
	showLook(s, p, tx, at, b, ok)
	follow(Target{S: s, P: p, Tx: tx, Pos: at, Ok: ok})
}

// showLook shows p the panel of block b at pos it looks at (ok), or clears
// it.
func showLook(s *core.Session, p *player.Player, tx *world.Tx, at cube.Pos, b world.Block, ok bool) {
	if !ok {
		waila.Clear(p)
		return
	}
	waila.Show(p, Info(s, p, tx, at, b))
}

// updateLookAt is updateLook with force for block b at pos p looks at (ok).
func updateLookAt(s *core.Session, p *player.Player, tx *world.Tx, at cube.Pos, b world.Block, ok bool) {
	lastLook.Of(s)[p.H()] = time.Now()
	if Hidden(s, p) {
		waila.Clear(p)
		return
	}
	showLook(s, p, tx, at, b, ok)
}

// UpdateAll updates every player's panel.
func UpdateAll(s *core.Session, tx *world.Tx) {
	now := time.Now()
	for e := range tx.Players() {
		p := e.(*player.Player)
		// a move updated the panel just now
		if now.Sub(lastLook.Of(s)[p.H()]) < updateAllSkip {
			continue
		}
		at, b, ok := LookedAt(p, tx, p.Position(), p.Rotation())
		updateLookAt(s, p, tx, at, b, ok)
		follow(Target{S: s, P: p, Tx: tx, Pos: at, Ok: ok, Tick: true})
	}
}

// Info is the panel of block b at pos: its name, icon, tools and the contents
// of a vanilla container, then what the block itself and the adapters add
// (waila.Describe). s is the session of tx's world; a block that needs it
// finds it through the transaction.
func Info(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, b world.Block) waila.Info {
	lang := i18n.Of(p)
	held, _ := p.HeldItems()
	info := waila.Info{Name: naming.BlockName(lang, b), Tools: waila.ToolsFor(b, held)}
	if it, ok := b.(world.Item); ok {
		info.Item = it
		info.Name = naming.ItemName(lang, item.NewStack(it, 1))
	}
	if _, fire := b.(block.Fire); fire {
		info.Item = item.FlintAndSteel{}
	}
	info.NameKey = naming.BlockNameKey(b)
	info.Container = containerOf(tx, pos, b)
	return waila.Describe(waila.View{Lang: lang, Sneaking: p.Sneaking(), Tx: tx, Pos: pos}, b, info)
}

// containerOf is the contents of a vanilla container block for the panel.
func containerOf(tx *world.Tx, pos cube.Pos, b world.Block) *waila.Container {
	c, ok := b.(interface {
		Inventory(*world.Tx, cube.Pos) *inventory.Inventory
	})
	if !ok {
		return nil
	}
	layout := waila.LayoutChest
	switch b.(type) {
	case block.Furnace, block.BlastFurnace, block.Smoker:
		layout = waila.LayoutFurnace
	case block.Hopper:
		layout = waila.LayoutHopper
	case block.BrewingStand:
		layout = waila.LayoutBrewing
	case block.Chest, block.Barrel, block.ShulkerBox:
	default:
		return nil
	}
	inv := c.Inventory(tx, pos)
	if inv == nil {
		return nil
	}
	return &waila.Container{Layout: layout, Items: inv.Slots()}
}

// lastLook is when each player's WAILA panel was last updated.
var lastLook = core.NewPlayers[time.Time]()
```


`expand/subscribe.go`: import `look`; `subscribe()` calls `hideLook()` as its last statement:

```go
// A player in island expansion mode sees no WAILA panel.
func hideLook() {
	look.Hidden = func(s *core.Session, p *player.Player) bool {
		_, expanding := Of(s, p)
		return expanding
	}
}
```

`tinker/subscribe.go`: import `look`; `subscribe()` calls `followLook()` as its last statement:

```go
// On the fast tick the vein outline follows the block a player looks at.
func followLook() {
	look.Follow("tinker.vein", func(t look.Target) {
		if !t.Tick {
			return
		}
		if t.Ok {
			ShowVeinOutline(t.S, t.P, t.Tx, t.Pos)
		} else {
			HideVeinOutline(t.S, t.P)
		}
	})
}
```

`wand/subscribe.go`: import `look`; `subscribe()` calls `followLook()` as its last statement:

```go
// The preview follows where a player holding a wand looks.
func followLook() {
	look.Follow("wand.preview", func(t look.Target) {
		ShowPreview(t.S, t.P, t.Tx)
	})
}
```

`content/skygear/layers_test.go`: in `layers` the core group becomes `"core", "network", "achievement", "preview", "look",` and the screens group `"codex", "screens",`; import `look`; append (T65 writes the `look` row only; T66 adds the `decor` row):

```go
// onlyImports are the packages of the game that may import nothing of the
// game but the listed packages: the look panel and the slot decoration ask
// blocks and items through an interface (waila.Provider, slotdeco.Decorated)
// and so know no feature and no item type.
var onlyImports = map[string][]string{
	"look":  {"core", "naming"},
	"decor": {"naming", "tinker"},
}

// TestCapabilityImports: the packages of onlyImports import only what their
// row lists.
func TestCapabilityImports(t *testing.T) {
	const prefix = "github.com/presentkim/skygear/content/skygear/"
	for p, allowed := range onlyImports {
		pkg, err := build.Default.ImportDir(p, 0)
		if err != nil {
			t.Errorf("%s: %v", p, err)
			continue
		}
		for _, imp := range pkg.Imports {
			if dep, ok := strings.CutPrefix(imp, prefix); ok && !slices.Contains(allowed, dep) {
				t.Errorf("%s imports %s: a block or an item describes itself through an interface, %s does not import its package", p, dep, p)
			}
		}
	}
}

// TestLookFollowers: what follows a panel update, in the order it runs.
func TestLookFollowers(t *testing.T) {
	if got, want := look.Followers(), []string{"tinker.vein", "wand.preview"}; !slices.Equal(got, want) {
		t.Errorf("look followers %v, want %v", got, want)
	}
}
```


### T66

Appended to `common/slotdeco/slotdeco.go`:

```go
// Decorated is an item that has a rarity or a tier badge of its own.
type Decorated interface {
	// Decor returns the item's rarity and its tier (0: no pips).
	Decor() (Rarity, int)
}
```

`content/skygear/sgitem/decor.go`

```go
package sgitem

import "github.com/presentkim/skygear/common/slotdeco"

// The slot decoration of the shared items (slotdeco.Decorated): the rarity
// grows with how far into the game the item is, and the tier shows as pips.

func (a TierAlloy) Decor() (slotdeco.Rarity, int) {
	for i, k := range TierAlloys {
		if k == a.Kind {
			return slotdeco.Rarity(i + 1), i + 1
		}
	}
	return slotdeco.Common, 0
}

func (k TierKit) Decor() (slotdeco.Rarity, int) {
	return slotdeco.Rarity(min(k.Level, int(slotdeco.Epic))), k.Level
}

func (c Circuit) Decor() (slotdeco.Rarity, int) {
	return slotdeco.Rarity(min(c.Level, int(slotdeco.Legendary))), c.Level
}

func (m Mesh) Decor() (slotdeco.Rarity, int) {
	switch m.Tier {
	case "flint", "iron":
		return slotdeco.Uncommon, 0
	case "gold":
		return slotdeco.Rare, 0
	case "diamond":
		return slotdeco.Epic, 0
	case "netherite":
		return slotdeco.Legendary, 0
	}
	return slotdeco.Common, 0
}

func (i Ingot) Decor() (slotdeco.Rarity, int) {
	switch i.Metal {
	case MetalBronze, MetalSteel:
		return slotdeco.Uncommon, 0
	case MetalElectrum, MetalInvar:
		return slotdeco.Rare, 0
	}
	return slotdeco.Common, 0
}

// The level shows as pips; the higher levels in rarer colours.
func (c CompactBlock) Decor() (slotdeco.Rarity, int) {
	return slotdeco.Rarity(c.Level - 1), c.Level
}

func (ResearchEssence) Decor() (slotdeco.Rarity, int) { return slotdeco.Uncommon, 0 }

func (e Essence) Decor() (slotdeco.Rarity, int) {
	if MobDifficulty[e.Mob] >= 4 {
		return slotdeco.Epic, 0
	}
	return slotdeco.Rare, 0
}

func (Relic) Decor() (slotdeco.Rarity, int) { return slotdeco.Epic, 0 }

func (FairyEmerald) Decor() (slotdeco.Rarity, int) { return slotdeco.Rare, 0 }

func (FairyHourglass) Decor() (slotdeco.Rarity, int) { return slotdeco.Rare, 0 }

func (DimensionStone) Decor() (slotdeco.Rarity, int) { return slotdeco.Rare, 0 }
```


`fairy/fairymagic.go`, after `func (Board) MaxCount() int { return 1 }` (import `common/slotdeco`):

```go
// Decor is the board's slot decoration (slotdeco.Decorated).
func (Board) Decor() (slotdeco.Rarity, int) { return slotdeco.Rare, 0 }
```

`decor/decor.go`: the body of `ItemDecor` becomes the four lines below, its doc comment stays, the import of `sgitem` goes:

```go
func ItemDecor(it world.Item) (slotdeco.Rarity, int) {
	if d, ok := it.(slotdeco.Decorated); ok {
		return d.Decor()
	}
	return slotdeco.Common, 0
}
```

The test, written first and green before the change:

`content/skygear/decor_test.go`

```go
package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/slotdeco"
	"github.com/presentkim/skygear/content/skygear/decor"
	"github.com/presentkim/skygear/content/skygear/fairy"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// TestItemDecor: an item says its own slot decoration (slotdeco.Decorated);
// an item that says nothing is common and has no tier.
func TestItemDecor(t *testing.T) {
	for _, c := range []struct {
		it   world.Item
		r    slotdeco.Rarity
		tier int
	}{
		{item.Stick{}, slotdeco.Common, 0},
		{sgitem.Hammer{}, slotdeco.Common, 0},
		{sgitem.Mesh{Tier: "string"}, slotdeco.Common, 0},
		{sgitem.Mesh{Tier: "gold"}, slotdeco.Rare, 0},
		{sgitem.TierKit{Level: 2}, slotdeco.Rare, 2},
		{sgitem.Circuit{Level: 4}, slotdeco.Legendary, 4},
		{sgitem.TierAlloy{Kind: sgitem.TierAlloys[0]}, slotdeco.Uncommon, 1},
		{sgitem.Ingot{Metal: sgitem.MetalSteel}, slotdeco.Uncommon, 0},
		{sgitem.ResearchEssence{Kind: sgitem.EssenceEngineering}, slotdeco.Uncommon, 0},
		{sgitem.Relic{Kind: sgitem.RelicWeapons[0]}, slotdeco.Epic, 0},
		{sgitem.FairyEmerald{}, slotdeco.Rare, 0},
		{sgitem.DimensionStone{}, slotdeco.Rare, 0},
		{fairy.Board{}, slotdeco.Rare, 0},
	} {
		if r, tier := decor.ItemDecor(c.it); r != c.r || tier != c.tier {
			t.Errorf("%T%+v: rarity %d tier %d, want %d and %d", c.it, c.it, r, tier, c.r, c.tier)
		}
	}
}
```


### T67

`content/skygear/core/creative.go`

```go
package core

import (
	"fmt"
	"slices"

	"github.com/df-mc/dragonfly/server/item/creative"
	"github.com/df-mc/dragonfly/server/world"
)

// CreativeItem is one item of the game's part of the creative inventory.
type CreativeItem struct {
	// Slot is the item's place in the list: items sort by Slot, and the items
	// of one Slot keep the order their owner gave them in. A Slot belongs to
	// one owner.
	Slot int
	// Category is the creative tab the item is in.
	Category creative.Category
	// Item is the item, listed as a stack of one.
	Item world.Item
}

// CreativeItems returns own, the items of no feature, and the Creative items
// of every installed feature in one list ordered by Slot. It panics when two
// owners use the same Slot.
func CreativeItems(own []CreativeItem) []CreativeItem {
	owner := map[int]string{}
	var out []CreativeItem
	add := func(name string, items []CreativeItem) {
		mine := map[int]bool{}
		for _, it := range items {
			if o, taken := owner[it.Slot]; taken && !mine[it.Slot] {
				panic(fmt.Sprintf("core: creative slot %d is used by %s and by %s", it.Slot, o, name))
			}
			owner[it.Slot], mine[it.Slot] = name, true
		}
		out = append(out, items...)
	}
	add("the content list", own)
	for _, f := range Features() {
		if f.Creative != nil {
			add("feature "+f.Name, f.Creative())
		}
	}
	slices.SortStableFunc(out, func(a, b CreativeItem) int { return a.Slot - b.Slot })
	return out
}
```


`content/skygear/core/creative_test.go`

```go
package core

import (
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/creative"
	"github.com/df-mc/dragonfly/server/world"
)

func creativeOf(slot int, it world.Item) CreativeItem {
	return CreativeItem{Slot: slot, Category: creative.ItemsCategory(), Item: it}
}

// TestCreativeItemsSortBySlot: the list is ordered by slot whoever owns the
// slot, and the items of one slot keep their owner's order.
func TestCreativeItemsSortBySlot(t *testing.T) {
	ResetHooks()
	Install(
		&Feature{Name: "a", Creative: func() []CreativeItem {
			return []CreativeItem{creativeOf(30, item.Stick{}), creativeOf(10, item.Diamond{}), creativeOf(30, item.Apple{})}
		}},
		&Feature{Name: "b"},
		&Feature{Name: "c", Creative: func() []CreativeItem { return []CreativeItem{creativeOf(20, item.Coal{})} }},
	)
	got := CreativeItems([]CreativeItem{creativeOf(40, block.Stone{}), creativeOf(15, block.Dirt{})})
	want := []world.Item{item.Diamond{}, block.Dirt{}, item.Coal{}, item.Stick{}, item.Apple{}, block.Stone{}}
	if len(got) != len(want) {
		t.Fatalf("%d items, want %d", len(got), len(want))
	}
	for i, it := range got {
		if it.Item != want[i] {
			t.Errorf("item %d is %T, want %T", i, it.Item, want[i])
		}
	}
}

// TestCreativeSlotOfTwoOwnersPanics: a slot belongs to one owner; the panic
// names both.
func TestCreativeSlotOfTwoOwnersPanics(t *testing.T) {
	for name, own := range map[string][]CreativeItem{
		"two features":         nil,
		"feature and the list": {creativeOf(20, block.Stone{})},
	} {
		t.Run(name, func(t *testing.T) {
			ResetHooks()
			fs := []*Feature{{Name: "a", Creative: func() []CreativeItem { return []CreativeItem{creativeOf(20, item.Stick{})} }}}
			if own == nil {
				fs = append(fs, &Feature{Name: "b", Creative: func() []CreativeItem { return []CreativeItem{creativeOf(20, item.Coal{})} }})
			}
			Install(fs...)
			defer func() {
				r, _ := recover().(string)
				if !strings.Contains(r, "creative slot 20") || !strings.Contains(r, "feature a") {
					t.Errorf("panic %q, want one naming slot 20 and feature a", r)
				}
			}()
			CreativeItems(own)
			t.Error("no panic")
		})
	}
}
```


`core/feature.go`, in `type Feature struct` before `Setup`:

```go
	// Creative, if set, returns the items the feature adds to the creative
	// inventory, each with its Slot in the list (CreativeItems). It is called
	// when the list is built, after every item is registered.
	Creative func() []CreativeItem
```

The list after T67: every run is still here, each with its slot.

`content/skygear/catalog/creative.go`

```go
package catalog

import (
	"sync"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/creative"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/conduit"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/elevator"
	"github.com/presentkim/skygear/content/skygear/fairy"
	"github.com/presentkim/skygear/content/skygear/farm"
	"github.com/presentkim/skygear/content/skygear/hopper"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/netstore"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/storage"
)

// Creative inventory entries of the game's own items and blocks. The client
// builds its recipe book from the creative inventory: a recipe whose result
// is not in it is never listed, so every item and block the game makes is
// registered here, in anonymous groups of the fitting tab. Slots order the
// list (core.CreativeItems): a run of items has one slot.
var registerCreativeOnce sync.Once

// RegisterCreativeItems adds the game's items to the creative inventory. It
// runs when the game starts, after every item is registered.
func RegisterCreativeItems() {
	registerCreativeOnce.Do(func() {
		construction, equipment, items := creative.ConstructionCategory(), creative.EquipmentCategory(), creative.ItemsCategory()
		groupName := func(c creative.Category) string {
			switch c {
			case construction:
				return "anon_sg_construction"
			case equipment:
				return "anon_sg_equipment"
			}
			return "anon_sg_items"
		}
		for _, c := range []creative.Category{construction, equipment, items} {
			// No icon makes the group anonymous: its items are not grouped.
			creative.RegisterGroup(creative.Group{Category: c, Name: groupName(c)})
		}
		for _, it := range core.CreativeItems(ownCreative()) {
			creative.RegisterItem(creative.Item{Stack: item.NewStack(it.Item, 1), Group: groupName(it.Category)})
		}
	})
}

// ownCreative are the creative items this package lists, each run with its
// slot.
func ownCreative() []core.CreativeItem {
	construction, equipment, items := creative.ConstructionCategory(), creative.EquipmentCategory(), creative.ItemsCategory()
	var out []core.CreativeItem
	slot := 0
	add := func(c creative.Category, it world.Item) {
		out = append(out, core.CreativeItem{Slot: slot, Category: c, Item: it})
	}

	slot = 10
	add(construction, progress.ResearchTable{})

	slot = 20
	add(construction, storage.Drawer{})

	slot = 30
	for _, w := range farm.BarrelWoods {
		add(construction, farm.Barrel{Wood: w})
	}

	slot = 40
	for t := 1; t <= conduit.Tiers; t++ {
		add(construction, conduit.EnergyConduit{Tier: t})
		add(construction, conduit.ItemConduit{Tier: t})
		add(construction, conduit.FluidConduit{Tier: t})
	}
	add(construction, conduit.NetConduit{})

	slot = 50
	for _, k := range machine.Order {
		add(construction, machine.Machine{Kind: k})
	}

	slot = 60
	add(construction, sgitem.Dust{})

	slot = 70
	for _, c := range item.Colours() {
		add(construction, elevator.Elevator{Colour: c})
	}

	slot = 80
	for _, k := range sgitem.CrushedKinds {
		add(construction, sgitem.CrushedBlock{Kind: k})
	}
	for _, c := range sgitem.CompactBlocks() {
		add(construction, c)
	}
	for _, id := range hopper.HopperOrder {
		add(construction, hopper.NewFlowHopper(id, cube.FaceDown))
	}
	add(equipment, sgitem.Hammer{Stone: true})
	add(equipment, sgitem.Hammer{})
	for _, t := range sgitem.CrookTiers {
		add(equipment, sgitem.Crook{Tier: t.Name})
	}

	slot = 90
	add(items, farm.Silkworm{})
	add(items, farm.CookedSilkworm{})

	slot = 100
	for _, t := range sgitem.WandTiers {
		add(equipment, sgitem.NewWand(t))
	}
	add(items, sgitem.WoodenBucket{})
	add(items, sgitem.WoodenBucket{Water: true})
	add(equipment, sgitem.DrawerKey{})
	add(items, sgitem.DrawerUpgrade{Kind: sgitem.UpgradeCapacity})
	add(items, sgitem.DrawerUpgrade{Kind: sgitem.UpgradeVoid})
	add(items, sgitem.BarkFiber{})
	add(items, sgitem.FairyDust{})

	slot = 110
	for t := range netstore.DiskCaps {
		add(items, netstore.Disk{Tier: t})
	}
	add(items, netstore.Pattern{})
	add(items, netstore.Upgrade{})
	add(items, netstore.Upgrade{Stack: true})

	slot = 120
	add(items, sgitem.FairyEmerald{})
	add(items, sgitem.DimensionStone{})
	for _, kind := range append(append([]string(nil), sgitem.RelicWeapons...), sgitem.RelicCharms...) {
		add(equipment, sgitem.Relic{Kind: kind})
	}

	slot = 130
	add(equipment, fairy.Board{})

	slot = 140
	add(items, sgitem.FairyBoneMeal{})
	add(items, sgitem.FairyHourglass{})
	add(items, sgitem.GrassSeeds{})
	add(items, sgitem.MyceliumSpores{})
	add(items, sgitem.WitchWaterBucket{})
	add(items, sgitem.ConduitBinderMix{})
	add(items, sgitem.ConduitBinder{})
	for _, t := range sgitem.MeshTiers {
		add(items, sgitem.Mesh{Tier: t.Tier})
	}
	for _, k := range sgitem.OreKinds {
		add(items, sgitem.OreChunk{Ore: k.Ore})
	}
	for _, m := range sgitem.NewOreMetals {
		add(construction, sgitem.OreBlock{Metal: m})
		add(items, sgitem.RawOre{Metal: m})
	}
	for _, k := range sgitem.PebbleKinds {
		add(items, sgitem.Pebble{Kind: k.Kind})
	}
	for _, c := range sgitem.PetalColours {
		add(items, sgitem.Petal{Colour: c})
	}

	slot = 150
	for _, sd := range farm.PlantSeeds {
		add(items, farm.PlantSeed{Seed: sd})
	}

	slot = 160
	for _, m := range sgitem.MobKinds {
		add(items, sgitem.Essence{Mob: m})
	}
	for _, m := range sgitem.DustMetals {
		add(items, sgitem.MetalDust{Metal: m})
	}
	for _, m := range sgitem.IngotMetals {
		add(items, sgitem.Ingot{Metal: m})
	}
	for _, a := range sgitem.TierAlloys {
		add(items, sgitem.TierAlloy{Kind: a})
	}
	for l := 1; l <= 4; l++ {
		add(items, sgitem.Circuit{Level: l})
	}
	for l := 1; l <= machine.MaxTier; l++ {
		add(items, sgitem.TierKit{Level: l})
	}
	add(equipment, sgitem.ConfigCopier{})
	return out
}
```


### T68 and T69

Each feature gets `creative.go` below and, in its `feature.go`, the field `Creative: creativeItems` before `Setup: subscribe`; the same task deletes the feature's run (the `slot = N` line and its statements) from `ownCreative` in `catalog/creative.go` and the import that only it used. T68: progress (10), storage (20), farm (30, 90, 150), conduit (40). T69: machine (50), elevator (70), netstore (110), fairy (130).

`content/skygear/progress/creative.go`

```go
package progress

import (
	"github.com/df-mc/dragonfly/server/item/creative"
	"github.com/presentkim/skygear/content/skygear/core"
)

// creativeItems are the feature's items in the creative inventory.
func creativeItems() []core.CreativeItem {
	return []core.CreativeItem{{Slot: 10, Category: creative.ConstructionCategory(), Item: ResearchTable{}}}
}
```


`content/skygear/storage/creative.go`

```go
package storage

import (
	"github.com/df-mc/dragonfly/server/item/creative"
	"github.com/presentkim/skygear/content/skygear/core"
)

// creativeItems are the feature's items in the creative inventory.
func creativeItems() []core.CreativeItem {
	return []core.CreativeItem{{Slot: 20, Category: creative.ConstructionCategory(), Item: Drawer{}}}
}
```


`content/skygear/farm/creative.go`

```go
package farm

import (
	"github.com/df-mc/dragonfly/server/item/creative"
	"github.com/presentkim/skygear/content/skygear/core"
)

// creativeItems are the feature's items in the creative inventory: the
// barrels, the silkworms and the seeds of the magic plants.
func creativeItems() []core.CreativeItem {
	var out []core.CreativeItem
	for _, w := range BarrelWoods {
		out = append(out, core.CreativeItem{Slot: 30, Category: creative.ConstructionCategory(), Item: Barrel{Wood: w}})
	}
	out = append(out,
		core.CreativeItem{Slot: 90, Category: creative.ItemsCategory(), Item: Silkworm{}},
		core.CreativeItem{Slot: 90, Category: creative.ItemsCategory(), Item: CookedSilkworm{}},
	)
	for _, sd := range PlantSeeds {
		out = append(out, core.CreativeItem{Slot: 150, Category: creative.ItemsCategory(), Item: PlantSeed{Seed: sd}})
	}
	return out
}
```


`content/skygear/conduit/creative.go`

```go
package conduit

import (
	"github.com/df-mc/dragonfly/server/item/creative"
	"github.com/presentkim/skygear/content/skygear/core"
)

// creativeItems are the feature's items in the creative inventory: the
// conduits of every tier and the network conduit.
func creativeItems() []core.CreativeItem {
	var out []core.CreativeItem
	construction := creative.ConstructionCategory()
	for t := 1; t <= Tiers; t++ {
		out = append(out,
			core.CreativeItem{Slot: 40, Category: construction, Item: EnergyConduit{Tier: t}},
			core.CreativeItem{Slot: 40, Category: construction, Item: ItemConduit{Tier: t}},
			core.CreativeItem{Slot: 40, Category: construction, Item: FluidConduit{Tier: t}},
		)
	}
	return append(out, core.CreativeItem{Slot: 40, Category: construction, Item: NetConduit{}})
}
```


`content/skygear/machine/creative.go`

```go
package machine

import (
	"github.com/df-mc/dragonfly/server/item/creative"
	"github.com/presentkim/skygear/content/skygear/core"
)

// creativeItems are the feature's items in the creative inventory: every
// machine kind in Order.
func creativeItems() []core.CreativeItem {
	var out []core.CreativeItem
	for _, k := range Order {
		out = append(out, core.CreativeItem{Slot: 50, Category: creative.ConstructionCategory(), Item: Machine{Kind: k}})
	}
	return out
}
```


`content/skygear/elevator/creative.go`

```go
package elevator

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/creative"
	"github.com/presentkim/skygear/content/skygear/core"
)

// creativeItems are the feature's items in the creative inventory: the
// elevator in every colour.
func creativeItems() []core.CreativeItem {
	var out []core.CreativeItem
	for _, c := range item.Colours() {
		out = append(out, core.CreativeItem{Slot: 70, Category: creative.ConstructionCategory(), Item: Elevator{Colour: c}})
	}
	return out
}
```


`content/skygear/netstore/creative.go`

```go
package netstore

import (
	"github.com/df-mc/dragonfly/server/item/creative"
	"github.com/presentkim/skygear/content/skygear/core"
)

// creativeItems are the feature's items in the creative inventory: the
// disks, the pattern and the upgrades.
func creativeItems() []core.CreativeItem {
	var out []core.CreativeItem
	items := creative.ItemsCategory()
	for t := range DiskCaps {
		out = append(out, core.CreativeItem{Slot: 110, Category: items, Item: Disk{Tier: t}})
	}
	return append(out,
		core.CreativeItem{Slot: 110, Category: items, Item: Pattern{}},
		core.CreativeItem{Slot: 110, Category: items, Item: Upgrade{}},
		core.CreativeItem{Slot: 110, Category: items, Item: Upgrade{Stack: true}},
	)
}
```


`content/skygear/fairy/creative.go`

```go
package fairy

import (
	"github.com/df-mc/dragonfly/server/item/creative"
	"github.com/presentkim/skygear/content/skygear/core"
)

// creativeItems are the feature's items in the creative inventory.
func creativeItems() []core.CreativeItem {
	return []core.CreativeItem{{Slot: 130, Category: creative.EquipmentCategory(), Item: Board{}}}
}
```


`catalog/creative.go` after T69, from its first comment on; its imports are then `sync`, `cube`, `item`, `creative`, `world`, `core`, `hopper`, `machine`, `sgitem`:

```go
// Creative inventory entries of the game's own items and blocks. The client
// builds its recipe book from the creative inventory: a recipe whose result
// is not in it is never listed, so every item and block the game makes is
// registered here, in anonymous groups of the fitting tab. A feature lists
// its own items in its descriptor (core.Feature.Creative); the items of the
// shared item packages are listed below (ownCreative). Slots order the whole
// list (core.CreativeItems).
var registerCreativeOnce sync.Once

// RegisterCreativeItems adds the game's items to the creative inventory. It
// runs when the game starts, after every item is registered.
func RegisterCreativeItems() {
	registerCreativeOnce.Do(func() {
		construction, equipment, items := creative.ConstructionCategory(), creative.EquipmentCategory(), creative.ItemsCategory()
		groupName := func(c creative.Category) string {
			switch c {
			case construction:
				return "anon_sg_construction"
			case equipment:
				return "anon_sg_equipment"
			}
			return "anon_sg_items"
		}
		for _, c := range []creative.Category{construction, equipment, items} {
			// No icon makes the group anonymous: its items are not grouped.
			creative.RegisterGroup(creative.Group{Category: c, Name: groupName(c)})
		}
		for _, it := range core.CreativeItems(ownCreative()) {
			creative.RegisterItem(creative.Item{Stack: item.NewStack(it.Item, 1), Group: groupName(it.Category)})
		}
	})
}

// ownCreative are the creative items of the packages that are no feature:
// the shared items (sgitem) and the hoppers.
func ownCreative() []core.CreativeItem {
	construction, equipment, items := creative.ConstructionCategory(), creative.EquipmentCategory(), creative.ItemsCategory()
	var out []core.CreativeItem
	slot := 0
	add := func(c creative.Category, it world.Item) {
		out = append(out, core.CreativeItem{Slot: slot, Category: c, Item: it})
	}

	slot = 60
	add(construction, sgitem.Dust{})

	slot = 80
	for _, k := range sgitem.CrushedKinds {
		add(construction, sgitem.CrushedBlock{Kind: k})
	}
	for _, c := range sgitem.CompactBlocks() {
		add(construction, c)
	}
	for _, id := range hopper.HopperOrder {
		add(construction, hopper.NewFlowHopper(id, cube.FaceDown))
	}
	add(equipment, sgitem.Hammer{Stone: true})
	add(equipment, sgitem.Hammer{})
	for _, t := range sgitem.CrookTiers {
		add(equipment, sgitem.Crook{Tier: t.Name})
	}

	slot = 100
	for _, t := range sgitem.WandTiers {
		add(equipment, sgitem.NewWand(t))
	}
	add(items, sgitem.WoodenBucket{})
	add(items, sgitem.WoodenBucket{Water: true})
	add(equipment, sgitem.DrawerKey{})
	add(items, sgitem.DrawerUpgrade{Kind: sgitem.UpgradeCapacity})
	add(items, sgitem.DrawerUpgrade{Kind: sgitem.UpgradeVoid})
	add(items, sgitem.BarkFiber{})
	add(items, sgitem.FairyDust{})

	slot = 120
	add(items, sgitem.FairyEmerald{})
	add(items, sgitem.DimensionStone{})
	for _, kind := range append(append([]string(nil), sgitem.RelicWeapons...), sgitem.RelicCharms...) {
		add(equipment, sgitem.Relic{Kind: kind})
	}

	slot = 140
	add(items, sgitem.FairyBoneMeal{})
	add(items, sgitem.FairyHourglass{})
	add(items, sgitem.GrassSeeds{})
	add(items, sgitem.MyceliumSpores{})
	add(items, sgitem.WitchWaterBucket{})
	add(items, sgitem.ConduitBinderMix{})
	add(items, sgitem.ConduitBinder{})
	for _, t := range sgitem.MeshTiers {
		add(items, sgitem.Mesh{Tier: t.Tier})
	}
	for _, k := range sgitem.OreKinds {
		add(items, sgitem.OreChunk{Ore: k.Ore})
	}
	for _, m := range sgitem.NewOreMetals {
		add(construction, sgitem.OreBlock{Metal: m})
		add(items, sgitem.RawOre{Metal: m})
	}
	for _, k := range sgitem.PebbleKinds {
		add(items, sgitem.Pebble{Kind: k.Kind})
	}
	for _, c := range sgitem.PetalColours {
		add(items, sgitem.Petal{Colour: c})
	}

	slot = 160
	for _, m := range sgitem.MobKinds {
		add(items, sgitem.Essence{Mob: m})
	}
	for _, m := range sgitem.DustMetals {
		add(items, sgitem.MetalDust{Metal: m})
	}
	for _, m := range sgitem.IngotMetals {
		add(items, sgitem.Ingot{Metal: m})
	}
	for _, a := range sgitem.TierAlloys {
		add(items, sgitem.TierAlloy{Kind: a})
	}
	for l := 1; l <= 4; l++ {
		add(items, sgitem.Circuit{Level: l})
	}
	for l := 1; l <= machine.MaxTier; l++ {
		add(items, sgitem.TierKit{Level: l})
	}
	add(equipment, sgitem.ConfigCopier{})
	return out
}
```


T69 appends this test to `content/skygear/creative_test.go` (imports to add: `go/parser`, `go/token`, `maps`, `slices`, `strconv`, `core`):

```go
// TestCreativeListNamesNoFeature: the content list's creative file lists the
// items of the shared item packages only; a feature lists its own items in
// its descriptor (core.Feature.Creative).
func TestCreativeListNamesNoFeature(t *testing.T) {
	const prefix = "github.com/presentkim/skygear/content/skygear/"
	f, err := parser.ParseFile(token.NewFileSet(), "catalog/creative.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	allowed := []string{"core", "hopper", "machine", "sgitem"}
	for _, imp := range f.Imports {
		path, _ := strconv.Unquote(imp.Path.Value)
		if dep, ok := strings.CutPrefix(path, prefix); ok && !slices.Contains(allowed, dep) {
			t.Errorf("catalog/creative.go imports %s: the feature lists its creative items itself", dep)
		}
	}
	slots := map[int]string{}
	for _, f := range core.Features() {
		if f.Creative == nil {
			continue
		}
		for _, it := range f.Creative() {
			if o, taken := slots[it.Slot]; taken && o != f.Name {
				t.Errorf("creative slot %d is used by %s and %s", it.Slot, o, f.Name)
			}
			slots[it.Slot] = f.Name
		}
	}
	want := map[int]string{10: "progress", 20: "storage", 30: "farm", 40: "conduit", 50: "machine", 70: "elevator", 90: "farm", 110: "netstore", 130: "fairy", 150: "farm"}
	if !maps.Equal(slots, want) {
		t.Errorf("creative slots of the features %v, want %v", slots, want)
	}
}
```


### T70

`content/skygear/core/commands.go`

```go
package core

import (
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
)

// Commands returns the commands of the installed features: each feature's
// Commands in installation order. The game registers them when it starts.
func Commands() []cmd.Command {
	var out []cmd.Command
	for _, f := range Features() {
		if f.Commands != nil {
			out = append(out, f.Commands()...)
		}
	}
	return out
}

// PlayerSession is the player running a command and the session of its
// world. Without a player it returns false quietly; without a session it
// tells the player to save first.
func PlayerSession(src cmd.Source, o *cmd.Output, tx *world.Tx) (*player.Player, *Session, bool) {
	p, ok := src.(*player.Player)
	if !ok || tx == nil {
		return nil, nil, false
	}
	s, ok := Of(tx)
	if !ok {
		o.Error(i18n.T(i18n.Of(p), "menu.needSave"))
		return nil, nil, false
	}
	return p, s, true
}
```


`content/skygear/core/commands_test.go`

```go
package core

import (
	"slices"
	"testing"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/world"
)

type nopCmd struct{}

func (nopCmd) Run(cmd.Source, *cmd.Output, *world.Tx) {}

// TestCommandsInListOrder: the commands are those of the features, in the
// order of the features.
func TestCommandsInListOrder(t *testing.T) {
	ResetHooks()
	Install(
		&Feature{Name: "a", Commands: func() []cmd.Command {
			return []cmd.Command{cmd.New("a1", "", nil, nopCmd{}), cmd.New("a2", "", nil, nopCmd{})}
		}},
		&Feature{Name: "b"},
		&Feature{Name: "c", Commands: func() []cmd.Command { return []cmd.Command{cmd.New("c1", "", nil, nopCmd{})} }},
	)
	var names []string
	for _, c := range Commands() {
		names = append(names, c.Name())
	}
	if want := []string{"a1", "a2", "c1"}; !slices.Equal(names, want) {
		t.Errorf("commands %v, want %v", names, want)
	}
}

// TestPlayerSessionWithoutPlayer: a source that is no player gets nothing,
// quietly.
func TestPlayerSessionWithoutPlayer(t *testing.T) {
	o := &cmd.Output{}
	if p, s, ok := PlayerSession(nil, o, nil); ok || p != nil || s != nil || o.ErrorCount() != 0 {
		t.Errorf("PlayerSession without a player: %v %v %v, %d errors", p, s, ok, o.ErrorCount())
	}
}
```


`core/feature.go`, in `type Feature struct` before `Setup` (import `github.com/df-mc/dragonfly/server/cmd`):

```go
	// Commands, if set, returns the commands the feature adds (Commands). It
	// is called when the game starts, after the catalogs are loaded.
	Commands func() []cmd.Command
```

`content/skygear/farm/commands.go`

```go
package farm

import (
	"time"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/core"
)

// commands are the farm's commands: the sapling.
func commands() []cmd.Command {
	return []cmd.Command{
		cmd.New("묘목", i18n.T(i18n.Default, "cmd.sapling"), nil, saplingCmd{}),
		cmd.New("sapling", i18n.T(i18n.Default, "cmd.sapling"), nil, saplingCmd{}),
	}
}

// saplingCmd is /묘목 (and /sapling).
type saplingCmd struct{}

func (saplingCmd) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, ok := src.(*player.Player)
	if !ok {
		return
	}
	s, ok := core.Of(tx)
	if !ok {
		o.Error(i18n.T(i18n.Of(p), "sapling.needIsland"))
		return
	}
	if err := GiveSapling(s, p, tx, time.Now()); err != nil {
		o.Errorf(i18n.T(i18n.Of(p), "sapling.cooldown"), err.Error())
		return
	}
	o.Printf("%s", i18n.T(i18n.Of(p), "sapling.given"))
}
```


`content/skygear/machine/commands.go`

```go
package machine

import (
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/progress"
)

// commands are the machines' commands: the production statistics.
func commands() []cmd.Command {
	return []cmd.Command{
		cmd.New("통계", i18n.T(i18n.Default, "monitor.cmd"), nil, statsCmd{}),
		cmd.New("stats", i18n.T(i18n.Default, "monitor.cmd"), nil, statsCmd{}),
	}
}

// statsCmd is /통계 (and /stats): the production statistics.
type statsCmd struct{}

func (statsCmd) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, s, ok := core.PlayerSession(src, o, tx)
	if !ok {
		return
	}
	progress.Advance(s, tx, advancement.Stats)
	ShowStats(s, p, StatsItems)
}
```


`farm/feature.go` and `machine/feature.go`: the field `Commands: commands` before `Setup: subscribe`. `content/skygear/game.go`, in `New`, directly after `registerCommands(g)`:

```go
	for _, c := range core.Commands() {
		cmd.Register(c)
	}
```

`content/skygear/commands.go`: delete the four registrations of `"묘목"`, `"통계"`, `"stats"`, `"sapling"`, the types `statsCmd` and `saplingCmd` with their `Run`, and the imports they alone used. `content/skygear/locales/locales_test.go`: the first call of `TestNoKoreanInCode` becomes `i18ntest.NoKoreanInCode(t, "..", nil, "../commands.go", "../farm/commands.go", "../machine/commands.go")`.

The root test (T70 writes it with the first four names of `want`; T71 makes it the twelve below):

`content/skygear/commands_test.go`

```go
package skygear

import (
	"slices"
	"testing"

	"github.com/presentkim/skygear/content/skygear/core"
)

// TestFeatureCommands: the commands the features add, by feature order: the
// machines' statistics, the farm's sapling, then the screens' commands.
func TestFeatureCommands(t *testing.T) {
	var names []string
	for _, c := range core.Commands() {
		names = append(names, c.Name())
	}
	want := []string{"통계", "stats", "묘목", "sapling", "menu", "sg", "메뉴발급", "menuitem", "도감", "codex", "도움말", "sghelp"}
	if !slices.Equal(names, want) {
		t.Errorf("feature commands %v, want %v", names, want)
	}
}
```


### T71

`content/skygear/screens/commands.go`

```go
package screens

import (
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/codex"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/look"
	"github.com/presentkim/skygear/content/skygear/naming"
)

// commands are the commands that open the game's screens: the menu, a new
// menu item, the codex and the help.
func commands() []cmd.Command {
	return []cmd.Command{
		cmd.New("menu", i18n.T(i18n.Default, "cmd.menu"), nil, menuCmd{}),
		// A separate command rather than an alias: clients check typed commands
		// against the command list, where aliases may not be accepted.
		cmd.New("sg", i18n.T(i18n.Default, "cmd.menu"), nil, menuCmd{}),
		cmd.New("메뉴발급", i18n.T(i18n.Default, "cmd.menuItem"), nil, menuItemCmd{}),
		cmd.New("menuitem", i18n.T(i18n.Default, "cmd.menuItem"), nil, menuItemCmd{}),
		cmd.New("도감", i18n.T(i18n.Default, "cmd.codex"), nil, codexCmd{}),
		cmd.New("codex", i18n.T(i18n.Default, "cmd.codex"), nil, codexCmd{}),
		cmd.New("도움말", i18n.T(i18n.Default, "help.cmd"), nil, helpCmd{}),
		cmd.New("sghelp", i18n.T(i18n.Default, "help.cmd"), nil, helpCmd{}),
	}
}

// menuCmd is /menu (and /sg): the game menu.
type menuCmd struct{}

func (menuCmd) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, s, ok := core.PlayerSession(src, o, tx)
	if !ok {
		return
	}
	OpenMenu(s, p)
}

// helpCmd is /도움말 (and /sghelp): the help window.
type helpCmd struct{}

func (helpCmd) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, s, ok := core.PlayerSession(src, o, tx)
	if !ok {
		return
	}
	ShowHelp(s, p)
}

// menuItemCmd is /메뉴발급 (and /menuitem): a new menu item for a player who
// lost theirs.
type menuItemCmd struct{}

func (menuItemCmd) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, _, ok := core.PlayerSession(src, o, tx)
	if !ok {
		return
	}
	if p.Inventory().ContainsItemFunc(1, IsMenuItem) {
		o.Error(i18n.T(i18n.Of(p), "menu.item.alreadyHave"))
		return
	}
	if _, err := p.Inventory().AddItem(MenuItem(i18n.Of(p))); err != nil {
		o.Error(i18n.T(i18n.Of(p), "menu.item.inventoryFull"))
		return
	}
	o.Printf("%s", i18n.T(i18n.Of(p), "menu.item.given"))
}

// openCodex opens the codex: the page of the item p holds, else the page of
// the block p looks at (the one the WAILA panel shows), else the first page.
func openCodex(s *core.Session, p *player.Player) {
	achievement.Count(p, "codex_open", 1)
	codex.Start(s, p)
	if held, _ := p.HeldItems(); !held.Empty() && !IsMenuItem(held) {
		codex.ShowItem(s, p, held)
		return
	}
	if tx := p.Tx(); tx != nil {
		if _, b, ok := look.LookedAt(p, tx, p.Position(), p.Rotation()); ok {
			if it, isItem := b.(world.Item); isItem {
				if e, found := codex.ItemByKey(naming.ItemKey(it)); found {
					codex.ShowItem(s, p, e)
					return
				}
			}
		}
	}
	codex.Show(s, p)
}

// codexCmd is /도감 (and /codex).
type codexCmd struct{}

func (codexCmd) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, s, ok := core.PlayerSession(src, o, tx)
	if !ok {
		return
	}
	openCodex(s, p)
}
```


`screens/feature.go`: `Commands: commands` before `Setup: subscribe`. `content/skygear/commands.go` is deleted (what is left of it: `registerCommands`, `playerSession`, `menuCmd`, `helpCmd`, `menuItemCmd`, `openCodex`, `codexCmd`). `game.go`: delete the line `registerCommands(g)`. `locales_test.go`: the allow list becomes `"../screens/commands.go", "../farm/commands.go", "../machine/commands.go"`, and the test's comment says the command names are in the features' `commands.go`.

### T72

`content/skygear/names.go` is deleted; its two parts go here:

`content/skygear/naming/entry.go`

```go
package naming

import (
	"maps"

	"github.com/presentkim/skygear/common/timed"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

// A timed item is named as any other item of the game.
func init() {
	timed.Name = ItemName
}

// LocalItemName makes the client name an item by its translation key
// ("item.<id>.name", from the pack's texts/*.lang) instead of the Korean name
// Dragonfly puts in minecraft:display_name, so each client shows its own
// language.
func LocalItemName(e protocol.ItemEntry) protocol.ItemEntry {
	if !e.ComponentBased {
		return e
	}
	comps, ok := e.Data["components"].(map[string]any)
	if !ok {
		return e
	}
	if _, ok := comps["minecraft:display_name"]; !ok {
		return e
	}
	comps = maps.Clone(comps)
	comps["minecraft:display_name"] = map[string]any{"value": "item." + e.Name + ".name"}
	e.Data = maps.Clone(e.Data)
	e.Data["components"] = comps
	return e
}
```


`content/skygear/features.go`: `sgitem.ClipWoodenBucket(localItemName(e))` becomes `sgitem.ClipWoodenBucket(naming.LocalItemName(e))` (import `naming`). `content/skygear/names_test.go`: `localItemName(` becomes `naming.LocalItemName(` (twice).

### T73

`content/skygear/core/refill.go` moves to `content/skygear/autofill/refill.go` with these changes and no other: the package clause becomes `// Package autofill refills a hand whose stack ran out.` and `package autofill`; `AutofillSetting` becomes `Setting`; `Autofill` becomes `On`; inside the file `!Autofill(p)` becomes `!On(p)`. `sameKind`, `refillSlot`, `RefillHand`, `RefillHandLater` keep their names and bodies.

`content/skygear/core/kernel.go` loses `AutofillFeature`, `subscribeAutofill`, `sameItemName` and its import block; they become:

`content/skygear/autofill/feature.go`

```go
package autofill

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/platform/account"
)

// Feature is the autofill feature, listed in the game's feature list after
// the features whose steps it follows.
var Feature = &core.Feature{
	Name: "autofill",
	Settings: []account.Setting{
		{Key: Setting, NameKey: "setting.autofill", Default: true, Group: "setting.group.play"},
	},
	Setup: subscribe,
}

// A broken tool and a consumed item are replaced from the inventory, the last
// block of a stack placed is replaced, and a thrown snowball, egg or pearl may
// be the last of the stack.
func subscribe() {
	core.ItemDamage.On(20, "autofill.broken", func(e core.ItemDamageEvent) {
		// a held tool that breaks is replaced by another of its kind
		if held, _ := e.P.HeldItems(); *e.Damage > 0 && e.Item.Durability() >= 0 && *e.Damage >= e.Item.Durability() && sameItemName(held, e.Item) {
			RefillHandLater(e.P, e.Tx, e.Item)
		}
	})
	core.ItemConsume.On(20, "autofill.consume", func(e core.ItemEvent) {
		RefillHandLater(e.P, e.Tx, e.Item)
	})
	core.BlockPlace.On(40, "autofill.place", func(e core.BlockPlaceEvent) {
		if e.Held.Count() == 1 {
			RefillHandLater(e.P, e.Tx, e.Held)
		}
	})
	core.Use.On(10, "autofill.use", func(e core.UseEvent) {
		if e.Held.Count() == 1 {
			RefillHandLater(e.P, e.Tx, e.Held)
		}
	})
}

// sameItemName reports whether a and b are the same item, however worn.
func sameItemName(a, b item.Stack) bool {
	n1, m1 := a.Item().EncodeItem()
	n2, m2 := b.Item().EncodeItem()
	return !a.Empty() && !b.Empty() && n1 == n2 && m1 == m2
}
```


Call sites: `machine/sieve.go` and `wand/use.go` `core.RefillHand(` becomes `autofill.RefillHand(`; `wand_test.go` the same three times and `core.AutofillSetting` becomes `autofill.Setting`; `features.go` `core.AutofillFeature` becomes `autofill.Feature`; each file imports `autofill` and drops `core` only if nothing else uses it. `layers_test.go`: the core group becomes `"core", "network", "achievement", "preview", "look", "autofill",`. The comment in `pin_blocks_test.go` that names `RefillHandLater` needs no change.
