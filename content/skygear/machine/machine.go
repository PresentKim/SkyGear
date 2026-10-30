package machine

import (
	"fmt"
	"strings"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/presentkim/skygear/common/chestui"
	"github.com/presentkim/skygear/common/fluidnet"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/slotdeco"
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/content/skygear/blockmodel"
	"github.com/presentkim/skygear/content/skygear/climate"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/machinerecipe"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/network"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/sievetable"
	"github.com/presentkim/skygear/content/skygear/tinker"
	"github.com/presentkim/skygear/content/skygear/wintext"
	"github.com/presentkim/skygear/platform/account"
	"github.com/presentkim/skygear/platform/instance"
)

func init() {
	for i := range ProgressSteps {
		world.RegisterItem(ProgressArrow{Step: i})
		world.RegisterItem(CycleGauge{Step: i})
		world.RegisterItem(EnergyGauge{Step: i})
	}
	world.RegisterItem(Info{})
	for _, k := range Order {
		Kinds[k].Hash = block.NextHash()
		world.RegisterBlock(Machine{Kind: k})
		world.RegisterItem(Machine{Kind: k})
	}
	for _, k := range Order {
		for t := 0; t <= MaxTier; t++ {
			for _, on := range []bool{false, true} {
				if (t == 0 && !on) || (t > 0 && !TierLook(k)) || (on && !AnimatedKinds[k]) {
					continue
				}
				world.RegisterBlock(Machine{Kind: k, Active: on, Tier: t})
			}
		}
	}
	for _, k := range []string{KindSieve, KindAutoSieve, KindMagicSieve} {
		for _, l := range SieveLooks() {
			world.RegisterBlock(Machine{Kind: k, MeshLook: l[0], Content: l[1], Fill: l[2]})
		}
	}
}

// HourglassTicks is how long one fairy hourglass boosts a machine, and
// hourglassMaxTicks how much boost a machine holds.
const (
	HourglassTicks    = 10 * 60 * 20
	HourglassMaxTicks = 60 * 60 * 20
)

// hourglassable reports whether the hourglass speeds the machine: one that
// works on energy, like the snow (no generator or energy cell).
func (st *State) hourglassable() bool {
	return st.Def.Use > 0 && st.Def.Generates == 0 && !st.Def.Cell
}

// windHourglass adds an hourglass's time to the machine and reports whether
// it took it (not past hourglassMaxTicks).
func (st *State) WindHourglass() bool {
	if !st.hourglassable() || st.Boost+HourglassTicks > HourglassMaxTicks {
		return false
	}
	st.Boost += HourglassTicks
	return true
}

// hourglassReps is how many runs a machine that runs reps times a tick makes
// on top of them while an hourglass runs: as many again.
func (st *State) HourglassReps(reps int) int {
	if st.Boost <= 0 {
		return 0
	}
	return reps
}

// harvestRadius is how far the harvester reaches in x and z.
const harvestRadius = 4

// BlockedLines say why a machine that finished does not go on: its output has
// no room for the product.
func BlockedLines(lang string, st *State) []string {
	if len(st.Def.Output) > 0 && st.Progress >= st.CycleLen() {
		return []string{i18n.T(lang, "look.blocked")}
	}
	return nil
}

// CrafterMissing lists the ingredients an auto crafter lacks for its recipe
// ("화약 ×2, 막대기 ×1").
func CrafterMissing(lang string, st *State) string {
	var missing []string
	for _, req := range st.Recipe.Reqs {
		have := 0
		for _, slot := range st.Def.Input {
			if it, _ := st.Inv().Item(slot); machinerecipe.MatchReq(it, req) {
				have += it.Count()
			}
		}
		if have < req.Count {
			missing = append(missing, fmt.Sprintf("%s ×%d", machinerecipe.ReqName(lang, req), req.Count-have))
		}
	}
	return strings.Join(missing, ", ")
}

// Machine kinds.
const (
	KindCrusher = "crusher"
	KindSieve   = "sieve"
	// KindAutoSieve sifts on its own; KindSieve only by hand (sieve.go).
	KindAutoSieve = "auto_sieve"
	// KindMagicSieve is the auto sieve with an essence slot (magicsieve.go).
	KindMagicSieve = "magic_sieve"
	KindMiner      = "auto_miner"
	KindSellBox    = "sell_box"
	// KindHarvester harvests grown crops around it (harvester.go).
	KindHarvester = "harvester"
	// KindGenerator burns fuel into RE (generator.go).
	KindGenerator = "generator"
	// KindAlloy makes alloys of two ingredients (alloy.go); KindEnergyCell stores
	// energy (generator.go).
	KindAlloy = "alloy_smelter"
	// The fluid machines (fluidmachine.go).
	KindTank   = "fluid_tank"
	KindWasher = "ore_washer"
	// KindStorageController and KindStorageLink make the drawers one storage
	// (storage.go).
	KindStorageController = "storage_controller"
	KindStorageLink       = "storage_link"
	// KindTinker is the tinker table (tinkerform.go).
	KindTinker = "tinker_table"
	// KindGate is the dimension gate (wave.go).
	KindGate          = "dimension_gate"
	KindCrucible      = "crucible"
	KindLavaGenerator = "lava_generator"
	// KindSimulator turns an essence into the drops of a mob (essence.go).
	KindSimulator  = "mob_simulator"
	KindEnergyCell = "energy_cell"
	// The automation equipment (autoequip.go, autocraft.go).
	KindFurnace = "electric_furnace"
	KindVacuum  = "item_vacuum"
	KindFeller  = "tree_feller"
	KindCrafter = "auto_crafter"
	KindLab     = "laboratory"
)

type ID int

const (
	idCrusher ID = iota + 1
	idSieve
	idAutoSieve
	idMagicSieve
	idMiner
	idSellBox
	idHarvester
	idGenerator
	idAlloy
	idTank
	idWasher
	idStorageController
	idStorageLink
	idTinker
	idGate
	idCrucible
	idLavaGenerator
	idSimulator
	idEnergyCell
	idFurnace
	idVacuum
	idFeller
	idCrafter
	idLab
)

// Def describes a kind of machine. Slots are window slots of a 27
// slot (3 row) window; the window inventory is the machine inventory.
type Def struct {
	ID                     ID
	Animated, CycleOnReset bool
	// nameKey is the catalog key of the name (title).
	nameKey string
	// marker in the window title selects the machine's layout in the
	// resource pack (ui/sg_machine.json via inventoryui's chest_screen.json).
	Marker string
	// interval is the ticks one operation takes. Balancing drafts.
	Interval  int
	Input     []int
	Output    []int
	Indicator int
	// helpKeys are the catalog keys of the lines that explain it.
	helpKeys []string
	Hash     uint64
	// generates is the RE per tick a generator makes while it burns; energyCap
	// the RE its buffer holds.
	Generates, EnergyCap int
	// fluidCap is the mB a tank of the machine holds (0: no tank); fluidSource and
	// fluidSink say whether fluid conduits take fluid from it and put it in;
	// fluidOnly restricts it to one fluid (fluidnet.None: any).
	FluidCap               int
	fluidSource, fluidSink bool
	fluidOnly              fluidnet.Kind
	// controller marks the storage controller (storage.go).
	Controller bool
	// gate marks the dimension gate (wave.go): using it opens its form.
	Gate bool
	// cell makes the machine an energy store: it takes energy from the network and
	// gives it back, without working (energyCap is its capacity).
	Cell bool
	// use is the RE per tick a machine needs while it works (0: it needs
	// none).
	Use int
	// crafter marks the auto crafter, whose window has a recipe slot.
	Crafter bool
	// charge are the slots that take the essence of a magic sieve or the bone
	// meal of the tree feller (takesCharge); autoSieve
	// marks the sieves that sift on their own (sieve.go).
	Charge    []int
	autoSieve bool

	// The kinds whose behaviour is another package's set these (Register):
	// click is what a click does instead of opening the window, true if
	// it took the click; run is the work of one run; insert, peek and extract
	// replace the item I/O of the block.
	Click   func(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool
	Run     func(s *core.Session, st *State, pos cube.Pos, tx *world.Tx)
	Insert  func(tx *world.Tx, pos cube.Pos, st item.Stack) int
	Peek    func(tx *world.Tx, pos cube.Pos, want func(item.Stack) bool) item.Stack
	Extract func(tx *world.Tx, pos cube.Pos, st item.Stack, n int) item.Stack
}

var Kinds = map[string]*Def{
	KindCrusher: {
		ID: idCrusher, nameKey: "machine.crusher.name", Marker: "§d§m§1", Interval: 80, Input: []int{11}, Output: []int{15}, Indicator: 13,
		Use: 20, EnergyCap: ConsumerBuffer,
		helpKeys: []string{"machine.crusher.help.1", "machine.crusher.help.2"},
	},
	// The hand sieve has no window: the input lies on the mesh and the
	// products drop on top.
	KindSieve: {
		ID: idSieve, nameKey: "machine.sieve.name", Interval: 60, Input: []int{10}, Indicator: 12,
		helpKeys: []string{"machine.sieve.help.1", "machine.sieve.help.2"},
	},
	KindAutoSieve: {
		// The window shows the input (fed by hand, a hopper or a conduit), the ring
		// and the four slots the products go into (ui/sg_machine.json sieve_slots).
		ID: idAutoSieve, nameKey: "machine.autosieve.name", Marker: "§d§m§2", Interval: 60, Input: []int{10}, Output: []int{13, 14, 15, 16}, Indicator: 12,
		Use: 10, EnergyCap: ConsumerBuffer, autoSieve: true,
		helpKeys: []string{"machine.autosieve.help.1", "machine.autosieve.help.2"},
	},
	KindMagicSieve: {
		// Like the auto sieve, with the essence slot between the input and the ring
		// (ui/sg_machine.json magicsieve_slots).
		ID: idMagicSieve, nameKey: "machine.magicsieve.name", Marker: "§d§m§9", Interval: 90, Input: []int{10}, Charge: []int{11}, Output: []int{13, 14, 15, 16}, Indicator: 12,
		Use: 40, EnergyCap: ConsumerBuffer, autoSieve: true,
		helpKeys: []string{"machine.magicsieve.help.1", "machine.magicsieve.help.2", "machine.magicsieve.help.3"},
	},
	KindMiner: {
		ID: idMiner, nameKey: "machine.miner.name", Marker: "§d§m§5", Interval: 40, Output: []int{13}, Indicator: 10,
		Use: 20, EnergyCap: ConsumerBuffer,
		helpKeys: []string{"machine.miner.help.1"},
	},
	KindHarvester: {
		ID: idHarvester, nameKey: "machine.harvester.name", Marker: "§d§m§3", Interval: 40, Output: []int{13, 14, 15, 16}, Indicator: 10,
		Use: 10, EnergyCap: ConsumerBuffer,
		helpKeys: []string{"machine.harvester.help.1"},
	},
	KindGenerator: {
		ID: idGenerator, nameKey: "generator.name", Marker: "§d§m§5", Interval: 100, Input: []int{13}, Indicator: 10,
		Generates: GeneratorOutput, EnergyCap: generatorBuffer,
		helpKeys: []string{"generator.help.1", "generator.help.2"},
	},
	KindAlloy: {
		ID: idAlloy, nameKey: "machine.alloy.name", Marker: "§d§m§6", Interval: 100, Input: []int{10, 11}, Output: []int{15}, Indicator: 13,
		Use: 20, EnergyCap: ConsumerBuffer,
		helpKeys: []string{"machine.alloy.help.1", "machine.alloy.help.2"},
	},
	KindStorageController: {
		ID: idStorageController, nameKey: "machine.controller.name", Interval: 1, Indicator: 10, Controller: true,
		helpKeys: []string{"machine.controller.help.1", "machine.controller.help.2"},
	},
	KindStorageLink: {
		ID: idStorageLink, nameKey: "machine.link.name", Interval: 1, Indicator: 10,
		helpKeys: []string{"machine.link.help.1"},
	},
	KindTinker: {
		ID: idTinker, nameKey: "machine.tinker.name", Interval: 1, Indicator: 10,
		helpKeys: []string{"machine.tinker.help.1", "machine.tinker.help.2"},
	},
	KindGate: {
		ID: idGate, nameKey: "machine.gate.name", Interval: 1, Indicator: 10, Gate: true,
		helpKeys: []string{"machine.gate.help.1", "machine.gate.help.2"},
	},
	KindWasher: {
		ID: idWasher, nameKey: "machine.washer.name", Marker: "§d§m§2", Interval: 60, Input: []int{10}, Output: []int{13, 14, 15, 16}, Indicator: 12,
		Use: 40, EnergyCap: ConsumerBuffer,
		FluidCap: 4000, fluidSink: true, fluidOnly: fluidnet.Water,
		helpKeys: []string{"machine.washer.help.1", "machine.washer.help.2"},
	},
	KindTank: {
		ID: idTank, nameKey: "machine.tank.name", Interval: 1, Indicator: 10,
		FluidCap: tankCapacity, fluidSource: true, fluidSink: true,
		helpKeys: []string{"machine.tank.help.1", "machine.tank.help.2"},
	},
	KindCrucible: {
		ID: idCrucible, nameKey: "machine.crucible.name", Interval: 20, Input: []int{10}, Indicator: 12,
		Use: 30, EnergyCap: ConsumerBuffer,
		FluidCap: 8000, fluidSource: true, fluidOnly: fluidnet.Lava,
		helpKeys: []string{"machine.crucible.help.1", "machine.crucible.help.2"},
	},
	KindLavaGenerator: {
		ID: idLavaGenerator, nameKey: "machine.lavagen.name", Interval: 1, Indicator: 10,
		Generates: LavaGenOutput, EnergyCap: generatorBuffer,
		FluidCap: 4000, fluidSink: true, fluidOnly: fluidnet.Lava,
		helpKeys: []string{"machine.lavagen.help.1", "machine.lavagen.help.2"},
	},
	KindSimulator: {
		ID: idSimulator, nameKey: "machine.simulator.name", Marker: "§d§m§2", Interval: 200, Input: []int{10}, Output: []int{13, 14, 15, 16}, Indicator: 12,
		Use: 40, EnergyCap: ConsumerBuffer,
		helpKeys: []string{"machine.simulator.help.1", "machine.simulator.help.2"},
	},
	KindEnergyCell: {
		ID: idEnergyCell, nameKey: "machine.cell.name", Interval: 1, Indicator: 10,
		Cell: true, EnergyCap: CellCapacity,
		helpKeys: []string{"machine.cell.help.1", "machine.cell.help.2"},
	},
	KindFurnace: {
		ID: idFurnace, nameKey: "machine.furnace.name", Marker: "§d§m§1", Interval: 100, Input: []int{11}, Output: []int{15}, Indicator: 13,
		Use: 30, EnergyCap: ConsumerBuffer,
		helpKeys: []string{"machine.furnace.help.1", "machine.furnace.help.2"},
	},
	KindVacuum: {
		ID: idVacuum, nameKey: "machine.vacuum.name", Marker: "§d§m§3", Interval: 10, Output: []int{13, 14, 15, 16}, Indicator: 10,
		Use: 5, EnergyCap: ConsumerBuffer,
		helpKeys: []string{"machine.vacuum.help.1", "machine.vacuum.help.2"},
	},
	KindFeller: {
		ID: idFeller, nameKey: "machine.feller.name", Marker: "§d§m§9", Interval: 60, Input: []int{10}, Charge: []int{11}, Output: []int{13, 14, 15, 16}, Indicator: 12,
		Use: 15, EnergyCap: ConsumerBuffer,
		helpKeys: []string{"machine.feller.help.1", "machine.feller.help.2", "machine.feller.help.3"},
	},
	KindCrafter: {
		// The store is the 3×3 on the left, the recipe slot above the ring.
		ID: idCrafter, nameKey: "machine.crafter.name", Marker: "§d§m§7", Interval: 40, Input: []int{0, 1, 2, 9, 10, 11, 18, 19, 20}, Output: []int{6, 7, 15, 16}, Indicator: 13,
		Use: 25, EnergyCap: ConsumerBuffer, Crafter: true,
		helpKeys: []string{"machine.crafter.help.1", "machine.crafter.help.2", "machine.crafter.help.3"},
	},
	KindLab: {
		// The materials are the 3×3 on the left, the essences go to the four
		// slots on the right.
		ID: idLab, nameKey: "machine.lab.name", Marker: "§d§m§8", Interval: 320, Input: []int{0, 1, 2, 9, 10, 11, 18, 19, 20}, Output: []int{6, 7, 15, 16}, Indicator: 13,
		Use: 40, EnergyCap: ConsumerBuffer,
		helpKeys: []string{"machine.lab.help.1", "machine.lab.help.2", "machine.lab.help.3"},
	},
	KindSellBox: {
		ID: idSellBox, nameKey: "machine.sellbox.name", Marker: "§d§m§4", Interval: 20, Input: []int{9, 10, 11, 12, 13, 14, 15, 16, 17}, Indicator: 4,
		helpKeys: []string{"machine.sellbox.help.1", "machine.sellbox.help.2"},
	},
}

// Order is the registration and display order.
var Order = []string{KindCrusher, KindSieve, KindAutoSieve, KindMiner, KindSellBox, KindHarvester, KindGenerator, KindAlloy, KindEnergyCell, KindSimulator, KindTank, KindCrucible, KindLavaGenerator, KindWasher, KindGate, KindStorageController, KindStorageLink, KindTinker, KindFurnace, KindVacuum, KindFeller, KindCrafter, KindLab, KindMagicSieve}

// CrusherOutput is what the crusher always makes of an item: the blocks of
// the hammer chain one step down, an ore block sgitem.CrusherRaw raw ores
// (and crusherBonusOf more by chance).
func CrusherOutput(in item.Stack) (item.Stack, bool) {
	if metal, ok := sgitem.OreBlockMetal(in.Item()); ok {
		return item.NewStack(sgitem.RawOf(metal), sgitem.CrusherRaw), true
	}
	if out, ok := sgitem.CrushRecipes[naming.ItemKey(in.Item())]; ok {
		return item.NewStack(out, 1), true
	}
	return item.Stack{}, false
}

// CrusherBonusOf is the raw ore the crusher may make more of an ore block:
// sgitem.CrusherBonus of them at sgitem.CrusherBonusChance percent each.
func CrusherBonusOf(in item.Stack) (item.Stack, bool) {
	metal, ok := sgitem.OreBlockMetal(in.Item())
	if !ok {
		return item.Stack{}, false
	}
	return item.NewStack(sgitem.RawOf(metal), 1), true
}

// minerTool is what the auto miner mines with.
var minerTool = item.Pickaxe{Tier: item.ToolTierStone}

// Machine is a machine block. st is nil for machine items and is created
// when the machine is placed or loaded.
type Machine struct {
	Kind  string
	State *State
	// content and fill are what a sieve shows on its mesh (sieve.go).
	Content, Fill, MeshLook int
	// active is the working look of an animated machine (animate.go).
	Active bool
	// tier is the tier look of the block (tierlook.go).
	Tier int
}

// State is the inventory and progress of a placed machine. It is
// owned by the machine's world.
type State struct {
	Def      *Def
	storage  *chestui.Storage
	Progress int
	ShownPct int
	// shownEnergy is the buffer the energy gauge shows now (-1: nothing yet).
	shownEnergy int
	// owner is the account that placed a sell box and gets its money.
	Owner, ownerName string
	// pos is where the machine stands, set when it ticks or its window is
	// opened.
	pos cube.Pos
	// cracking is set while the auto miner shows its target cracking.
	cracking bool
	// lastUse is when a sieve was last sifted by hand.
	LastUse time.Time
	// mesh is the tier of the mesh fitted in a sieve ("" none, mesh.go).
	Mesh string
	// unlimited makes the machine run without energy (the operator's load
	// test factories); stalled is set while a machine that works has no
	// energy for a tick.
	Unlimited, Stalled bool
	// free marks a run the snow adds (snowReps): it needs the energy of a
	// run but spends none.
	free bool
	// boost is the ticks of fairy hourglass left: the machine runs twice
	// as fast, the extra runs free (fairytime.go).
	Boost int
	// energy is the RE in the buffer; burn the ticks the fuel in a generator
	// burns still, burnMax the ticks it burned for in all (generator.go).
	Energy, Burn, burnMax int
	// legacy are saved items of slots outside the kind's layout (saves of a
	// four-output auto miner); they move into the output as it frees up, and
	// are saved until then.
	Legacy []item.Stack
	// fluid and fluidAmt are what the tank holds (fluidmachine.go).
	Fluid    fluidnet.Kind
	FluidAmt int
	// mob and charges are what the mob simulator is charged with (essence.go).
	Mob     string
	Charges int
	// recipe is the recipe an auto crafter makes (recipeID is how it is saved);
	// shownRecipe is the one the recipe slot shows; sample is an item put on
	// the recipe slot to choose by, dropRecipe a request to clear it.
	Recipe      *machinerecipe.Crafter
	RecipeID    string
	shownRecipe string
	Sample      item.Stack
	dropRecipe  bool
	// pending are the products of an auto sieve that did not fit its output slots
	// yet: it waits until they do.
	pending []item.Stack
	// done counts the work cycles finished and made the RE generated since
	// they were last counted for the players' achievements (counter.go).
	Done, made int
	// tally are the counts to pass to the players' counters at the next
	// tick (counter.go); lab is the recipe the laboratory works on, period
	// the length of the cycle it is on (0: the kind's interval).
	tallyCounts map[string]int64
	Lab         *machinerecipe.Lab
	period      int
	// sess is the session the machine ticks in; stuck why the last tick
	// could not work, didWork whether a tick worked since the last second,
	// mon what the stall alerts know of it (monitor.go).
	Sess  *core.Session
	Stuck stuckKind
	// shownStuck is whether the info icon of the window shows the warning
	// badge (a stalled machine).
	ShownStuck bool
	didWork    bool
	Mon        monitor
	// tier is the machine's tier (tier.go); worked counts the ticks of work
	// done, to tell whether a run did anything.
	Tier, worked int
	// stallTicks counts the ticks in a row the machine had no energy.
	stallTicks int
	// seenWorked and lastWorkTick tell when the machine last worked (animate.go).
	seenWorked   int
	lastWorkTick int64
}

// cycleLen is the ticks of the cycle the machine is on.
func (st *State) CycleLen() int {
	if st.period > 0 {
		return st.period
	}
	return st.Def.Interval
}

// legacySlot is the first saved slot number of legacy items (not a window
// slot).
const legacySlot = 100

// drainLegacy moves a legacy item into the output when it fits.
func (st *State) DrainLegacy() {
	if len(st.Legacy) == 0 || len(st.Def.Output) == 0 {
		return
	}
	if it := st.Legacy[0]; Fits(st.Inv(), st.Def.Output, []item.Stack{it}) {
		put(st.Inv(), st.Def.Output, it)
		st.Legacy = st.Legacy[1:]
	}
}

func New(kind, owner, ownerName string) Machine {
	k := Kinds[kind]
	st := &State{Def: k, storage: chestui.NewStorage(27, k.windowTitle(i18n.Default)), Owner: owner, ownerName: ownerName, ShownPct: -1, shownEnergy: -1}
	inv := st.storage.Inventory()
	for slot := range 27 {
		if st.Role(slot) == RoleNone {
			_ = inv.SetItem(slot, Filler())
		}
	}
	_ = inv.SetItem(infoSlot, st.InfoItem())
	inv.Handle(machineHandler{State: st})
	st.shownRecipe = "\x00"
	if k.Crafter {
		st.showRecipe()
	}
	st.ShowProgress()
	st.ShowEnergy()
	return Machine{Kind: kind, State: st}
}

func Filler() item.Stack {
	return item.NewStack(block.StainedGlassPane{Colour: item.ColourGrey()}, 1).WithCustomName("§r ")
}

type slotRole int

const (
	RoleNone slotRole = iota
	roleInput
	roleOutput
	// roleRecipe is the recipe slot of the auto crafter: it takes nothing.
	roleRecipe
	// roleCharge is the essence slot of a magic sieve and the bone meal slot
	// of the tree feller.
	roleCharge
)

// infoSlot is the window slot of the info icon: its tooltip explains the
// machine. The theme draws it left of the close button.
const infoSlot = 26

func (k *Def) Title(lang string) string { return i18n.T(lang, k.nameKey) }

// windowTitle is the title of the kind's window in lang. The marker's last
// code is a colour (§1 dark blue ...): §r gives the name the window's own
// title colour back.
func (k *Def) windowTitle(lang string) string { return k.Marker + "§r" + k.Title(lang) }

func (st *State) OpenWindow(p *player.Player) {
	st.storage.SetTitle(st.Def.windowTitle(i18n.Of(p)))
	st.storage.Open(p, nil)
}

// infoItem is the info icon of a machine window: the machine's name, what
// it does and, for the sell box, whose it is.
func (st *State) InfoItem() item.Stack {
	var lore []wintext.Text
	for _, k := range st.Def.helpKeys {
		lore = append(lore, wintext.Text{wintext.Key(k)})
	}
	if st.Def.ID == idSellBox && st.ownerName != "" {
		lore = append(lore, wintext.Text{wintext.Key("machine.info.owner"), wintext.Raw(st.ownerName)})
	}
	if st.Tier > 0 {
		spec := st.spec()
		lore = append(lore, wintext.Text{wintext.Key("tier.info", wintext.Key(tierNameKey(st.Tier)), spec.speed, float64(spec.use10)/10)})
	}
	badge := 0
	if st.Stuck != stuckNone {
		badge = slotdeco.BadgeWarn
	}
	st.ShownStuck = badge != 0
	name := wintext.Text{wintext.Raw(slotdeco.Common.Mark() + "§r§f"), wintext.Key(st.Def.nameKey)}
	if st.Tier > 0 {
		name = append(name, wintext.Raw(" ["), wintext.Key(tierNameKey(st.Tier)), wintext.Raw("]"))
	}
	name = append(name, wintext.Raw(slotdeco.TierMark(badge)))
	return wintext.With(item.NewStack(Info{}, 1), name, lore...)
}

// Info is the info icon item of machine windows. It only ever sits
// in machine windows.
type Info struct{}

func (Info) EncodeItem() (string, int16) { return "skygear:machine_info", 0 }

func (Info) Name() string { return i18n.T(i18n.Default, "name.MachineInfo") }

func (Info) Category() category.Category { return category.Items() }

func (st *State) Role(slot int) slotRole {
	if st.Def.Crafter && slot == CrafterRecipeSlot {
		return roleRecipe
	}
	for _, s := range st.Def.Input {
		if s == slot {
			return roleInput
		}
	}
	for _, s := range st.Def.Output {
		if s == slot {
			return roleOutput
		}
	}
	for _, s := range st.Def.Charge {
		if s == slot {
			return roleCharge
		}
	}
	return RoleNone
}

func (st *State) Inv() *inventory.Inventory { return st.storage.Inventory() }

// accepts reports whether it may go into the input of the machine.
// Time-limited items are never processed or sold.
func (st *State) Accepts(it item.Stack) bool {
	if it.Empty() || !core.Plain(it) {
		return false
	}
	switch st.Def.ID {
	case idCrusher:
		_, ok := CrusherOutput(it)
		return ok
	case idGenerator:
		return FuelDuration(it) > 0
	case idAlloy:
		return machinerecipe.AlloyAccepts(it)
	case idSimulator:
		return sgitem.AcceptsEssence(it)
	case idCrucible:
		return naming.ItemKey(it.Item()) == naming.ItemKey(block.Cobblestone{})
	case idWasher:
		_, ok := sgitem.WasherOutput(it)
		return ok
	case idFurnace:
		_, ok := SmeltOf(it)
		return ok
	case idFeller:
		_, ok := it.Item().(block.Sapling)
		return ok || st.takesCharge(it)
	case idCrafter:
		return st.Recipe != nil && st.Recipe.Wants(it)
	case idLab:
		return st.labAccepts(it)
	case idSieve, idAutoSieve, idMagicSieve:
		// A sieve sifts only with a mesh fitted; the magic sieve also takes essences
		// (into its essence slot).
		_, ok := sievetable.Tables[st.Mesh][naming.ItemKey(it.Item())]
		return ok || st.takesCharge(it)
	}
	// The sell box takes nothing.
	return false
}

// takesCharge reports whether it goes into the charge slots of the machine:
// the essence of a magic sieve, the bone meal of the tree feller.
func (st *State) takesCharge(it item.Stack) bool {
	switch {
	case len(st.Def.Charge) == 0:
		return false
	case st.Def.ID == idFeller:
		_, ok := it.Item().(item.BoneMeal)
		return ok
	}
	return sgitem.AcceptsEssence(it)
}

// slotAccepts reports whether the item may be put on a slot of the machine:
// an input slot takes what the machine works on, except what goes into its
// charge slots.
func (st *State) SlotAccepts(slot int, it item.Stack) bool {
	switch st.Role(slot) {
	case roleInput:
		return st.Accepts(it) && !st.takesCharge(it)
	case roleCharge:
		return st.takesCharge(it)
	}
	return false
}

// machineHandler limits moves in the machine window to its input and
// output slots.
type machineHandler struct{ State *State }

func (h machineHandler) HandleTake(ctx *inventory.Context, slot int, _ item.Stack) {
	if h.State.Role(slot) == roleRecipe {
		// Taking the recipe off the slot clears it.
		ctx.Cancel()
		h.State.dropRecipe = true
		return
	}
	if h.State.Role(slot) == RoleNone {
		ctx.Cancel()
	}
}

func (h machineHandler) HandlePlace(ctx *inventory.Context, slot int, it item.Stack) {
	if h.State.Role(slot) == roleRecipe {
		// An item put on the recipe slot chooses the recipe that makes it
		// and goes back to the player.
		ctx.Cancel()
		h.State.Sample = it
		return
	}
	if h.State.SlotAccepts(slot, it) {
		return
	}
	ctx.Cancel()
	// The client's quick move (shift click, the take all and put all bar of
	// the inventoryui pack) puts an item into the first free slot, which may
	// be an output: what the machine takes goes into its input slots instead,
	// after this move is answered, so the client's view is fixed first.
	p, ok := ctx.Val().(*player.Player)
	if !ok || !h.State.Accepts(it) {
		return
	}
	p.H().Do(func(_ *world.Tx, e world.Entity) { h.State.putFrom(e.(*player.Player), it) })
}

func (h machineHandler) HandleDrop(ctx *inventory.Context, slot int, _ item.Stack) {
	if h.State.Role(slot) == RoleNone || h.State.Role(slot) == roleRecipe {
		ctx.Cancel()
	}
}

// putFrom moves up to it.Count() of it from p's inventory into the input
// slots: onto a stack of the same kind first; a kind that no input slot holds
// yet goes into one empty slot. What does not fit stays with p.
func (st *State) putFrom(p *player.Player, it item.Stack) {
	have := 0
	for _, cur := range p.Inventory().Slots() {
		if !cur.Empty() && cur.Comparable(it) {
			have += cur.Count()
		}
	}
	want := min(it.Count(), have)
	if want == 0 {
		return
	}
	if n := st.Fill(it.Grow(want - it.Count())); n > 0 {
		_ = p.Inventory().RemoveItem(it.Grow(n - it.Count()))
	}
}

// fill puts as much of it as fits into the input slots that take it and
// returns how many it put: onto stacks of the same kind first, and only
// when none holds the kind into one empty slot.
func (st *State) Fill(it item.Stack) int {
	inv := st.Inv()
	slots := append(append([]int{}, st.Def.Input...), st.Def.Charge...)
	left, held := it.Count(), false
	for _, s := range slots {
		cur, _ := inv.Item(s)
		if cur.Empty() || !cur.Comparable(it) || !st.SlotAccepts(s, it) {
			continue
		}
		held = true
		if n := min(left, cur.MaxCount()-cur.Count()); n > 0 {
			_ = inv.SetItem(s, cur.Grow(n))
			left -= n
		}
	}
	for _, s := range slots {
		if left == 0 || held {
			break
		}
		if cur, _ := inv.Item(s); cur.Empty() && st.SlotAccepts(s, it) {
			n := min(left, it.MaxCount())
			_ = inv.SetItem(s, it.Grow(n-it.Count()))
			left -= n
			break
		}
	}
	return it.Count() - left
}

// Fits reports whether every stack fits into slots together.
func Fits(inv *inventory.Inventory, slots []int, stacks []item.Stack) bool {
	room := make([]item.Stack, len(slots))
	for i, s := range slots {
		room[i], _ = inv.Item(s)
	}
	for _, it := range stacks {
		left := it.Count()
		for i := range room {
			switch {
			case room[i].Empty():
				n := min(left, it.MaxCount())
				room[i] = it.Grow(n - it.Count())
				left -= n
			case room[i].Comparable(it):
				n := min(left, room[i].MaxCount()-room[i].Count())
				room[i] = room[i].Grow(n)
				left -= n
			}
			if left == 0 {
				break
			}
		}
		if left > 0 {
			return false
		}
	}
	return true
}

// put adds it to slots. The caller checked that it fits.
func put(inv *inventory.Inventory, slots []int, it item.Stack) {
	left := it.Count()
	for _, s := range slots {
		cur, _ := inv.Item(s)
		switch {
		case cur.Empty():
			n := min(left, it.MaxCount())
			_ = inv.SetItem(s, it.Grow(n-it.Count()))
			left -= n
		case cur.Comparable(it):
			n := min(left, cur.MaxCount()-cur.Count())
			if n > 0 {
				_ = inv.SetItem(s, cur.Grow(n))
				left -= n
			}
		}
		if left == 0 {
			return
		}
	}
}

// firstInput returns the first non-empty input slot.
func (st *State) FirstInput() (int, item.Stack, bool) {
	for _, s := range st.Def.Input {
		if it, _ := st.Inv().Item(s); !it.Empty() {
			return s, it, true
		}
	}
	return 0, item.Stack{}, false
}

// insert puts one of it into the input, reporting success.
func (st *State) Insert(it item.Stack) bool {
	one := it.Grow(1 - it.Count())
	slots := st.Def.Input
	if st.takesCharge(one) {
		slots = st.Def.Charge
	}
	if !st.Accepts(one) || !Fits(st.Inv(), slots, []item.Stack{one}) {
		return false
	}
	if st.Def.ID == idLab && st.LabHave()[naming.ItemKey(one.Item())] >= st.LabFeedCap(one) {
		return false
	}
	put(st.Inv(), slots, one)
	return true
}

// OutputCount counts the items of a kind in the output slots.
func OutputCount(st *State, it world.Item) int {
	n := 0
	for _, slot := range st.Def.Output {
		if got, _ := st.Inv().Item(slot); !got.Empty() && naming.ItemKey(got.Item()) == naming.ItemKey(it) {
			n += got.Count()
		}
	}
	return n
}

// showProgress updates the indicator item when the shown percent changes.
// The indicator is a progress arrow item (ProgressArrow), so the window
// shows a filling arrow between input and output.
func (st *State) ShowProgress() {
	pct := min(100, st.Progress*100/st.CycleLen())
	pct -= pct % 10
	if pct == st.ShownPct {
		return
	}
	st.ShownPct = pct
	name := wintext.Key("machine.progress.percent", pct)
	it := item.NewStack(ProgressArrow{Step: pct / 10}, 1)
	switch st.Def.ID {
	case idSellBox:
		// Machines that work in a repeating cycle, the auto miner and the generator
		// show a ring that fills.
		name, it = wintext.Key("machine.progress.sellCycle"), item.NewStack(CycleGauge{Step: pct / 10}, 1)
	case idMiner, idAutoSieve, idMagicSieve, idSimulator, idWasher, idVacuum, idFeller, idCrafter, idLab:
		name, it = wintext.Key("machine.progress.mining", pct), item.NewStack(CycleGauge{Step: pct / 10}, 1)
	case idGenerator:
		// The ring shows the fuel left in the one burning.
		name, it = wintext.Key("machine.progress.fuel", pct), item.NewStack(CycleGauge{Step: pct / 10}, 1)
	case idHarvester:
		name, it = wintext.Key("machine.progress.harvestCycle"), item.NewStack(CycleGauge{Step: pct / 10}, 1)
	}
	_ = st.Inv().SetItem(st.Def.Indicator, wintext.With(it, wintext.Text{wintext.Raw("§r"), name}))
}

// EnergyGaugeSlot is the window slot of the energy gauge, in the column right
// of the machine's slots (ui/sg_machine.json).
const EnergyGaugeSlot = 17

// showEnergy updates the energy gauge of a machine with a window that holds
// energy: a battery filled in tenths, whose name is the energy it holds and the
// most it can (shown when the player hovers it). A machine that runs without
// energy (the operator's test factory) shows a full one.
func (st *State) ShowEnergy() {
	if st.Def.Marker == "" || st.Def.EnergyCap == 0 {
		return
	}
	capacity := st.EnergyCap()
	have := st.Energy
	if st.Unlimited {
		have = capacity
	}
	if have == st.shownEnergy {
		return
	}
	st.shownEnergy = have
	it := item.NewStack(EnergyGauge{Step: min(10, have*10/capacity)}, 1)
	_ = st.Inv().SetItem(EnergyGaugeSlot, wintext.With(it, wintext.Text{wintext.Key("machine.energy.gauge", have, capacity)}))
}

// EnergyGauge is the energy indicator of machine windows: a battery filled in
// Step tenths. It only ever sits in machine windows.
type EnergyGauge struct{ Step int }

func (g EnergyGauge) EncodeItem() (string, int16) {
	return fmt.Sprintf("skygear:energy_gauge_%d", g.Step), 0
}

func (g EnergyGauge) Name() string { return i18n.T(i18n.Default, "name.EnergyGauge") }

func (EnergyGauge) Category() category.Category { return category.Items() }

// ProgressArrow is the indicator item of machine windows: an arrow filled
// in Step tenths. It only ever sits in machine windows.
type ProgressArrow struct{ Step int }

// ProgressSteps is the number of ProgressArrow steps (0 to 10).
const ProgressSteps = 11

func (a ProgressArrow) EncodeItem() (string, int16) {
	return fmt.Sprintf("skygear:progress_%d", a.Step), 0
}

func (a ProgressArrow) Name() string { return i18n.T(i18n.Default, "name.ProgressArrow") }

func (ProgressArrow) Category() category.Category { return category.Items() }

// CycleGauge is the indicator of machines that work in a repeating cycle
// (auto harvester, sell box): a hollow ring filled in Step tenths, which
// turns over and over like a loading circle.
type CycleGauge struct{ Step int }

func (g CycleGauge) EncodeItem() (string, int16) {
	return fmt.Sprintf("skygear:cycle_%d", g.Step), 0
}

func (g CycleGauge) Name() string { return i18n.T(i18n.Default, "name.CycleGauge") }

func (CycleGauge) Category() category.Category { return category.Items() }

func (m Machine) Def() *Def { return Kinds[m.Kind] }

func (m Machine) EncodeItem() (string, int16) { return "skygear:" + m.Kind, 0 }

func (m Machine) EncodeBlock() (string, map[string]any) {
	if IsSieve(m.Kind) {
		return "skygear:" + m.Kind, map[string]any{SieveLookState: LookValue(m.MeshLook, m.Content, m.Fill)}
	}
	if TierLook(m.Kind) {
		return "skygear:" + m.Kind, map[string]any{TierState: TierValue(m.Kind, m.Tier, m.Active)}
	}
	if AnimatedKinds[m.Kind] {
		return "skygear:" + m.Kind, map[string]any{activeState: activeValue(m.Active)}
	}
	return "skygear:" + m.Kind, nil
}

func (m Machine) Hash() (uint64, uint64) {
	if m.Active || m.Tier > 0 {
		return m.Def().Hash, uint64(tierLookIndex(m.Kind, m.Tier, m.Active))
	}
	return m.Def().Hash, uint64(lookIndex(m.MeshLook, m.Content, m.Fill))
}

func (Machine) Model() world.BlockModel { return model.Solid{} }

func (m Machine) Name() string {
	if m.State != nil {
		return tierTitle(i18n.Default, m.Def().Title(i18n.Default), m.State.Tier)
	}
	return m.Def().Title(i18n.Default)
}

func (Machine) Category() category.Category { return category.Items() }

// cubeMachines are the machines drawn as plain cubes with one side texture;
// frontMachines have a front (north), back and side textures. The sieves
// have models.
var (
	cubeMachines  = map[string]bool{KindMiner: true, KindHarvester: true, KindGenerator: true, KindEnergyCell: true, KindVacuum: true, KindFeller: true, KindTank: true, KindLavaGenerator: true, KindGate: true, KindStorageController: true, KindStorageLink: true, KindTinker: true}
	frontMachines = map[string]bool{KindCrusher: true, KindWasher: true, KindSellBox: true, KindAlloy: true, KindSimulator: true, KindCrucible: true, KindFurnace: true, KindCrafter: true, KindLab: true}
)

// States implements block.Permutable: only the sieve has states.
func (m Machine) States() map[string][]any {
	if IsSieve(m.Kind) {
		return SieveStates()
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
	if IsSieve(m.Kind) {
		return sievePermutations(m.Kind)
	}
	if TierLook(m.Kind) {
		return tierPermutations(m.Kind)
	}
	if AnimatedKinds[m.Kind] {
		return activePermutations(m.Kind)
	}
	return nil
}

func (m Machine) Properties() customblock.Properties {
	if cubeMachines[m.Kind] {
		return blockmodel.CubeProperties(m.Kind)
	}
	if frontMachines[m.Kind] {
		return blockmodel.FrontCubeProperties(m.Kind)
	}
	return sieveProperties(m.Kind)
}

// BreakInfo drops the machine and its contents.
func (m Machine) BreakInfo() block.BreakInfo {
	return block.BreakInfo{
		Hardness: 1.5, BlastResistance: 6,
		Harvestable: func(item.Tool) bool { return true },
		Effective:   blockmodel.CustomBlockEffective,
		Drops: func(item.Tool, []item.Enchantment) []item.Stack {
			drops := []item.Stack{item.NewStack(Machine{Kind: m.Kind}, 1)}
			if m.State != nil {
				drops = append(drops, m.State.pending...)
			}
			if m.State != nil && m.State.Mesh != "" {
				drops = append(drops, item.NewStack(sgitem.Mesh{Tier: m.State.Mesh}, 1))
			}
			if m.State != nil {
				// the kits it was raised with come back
				for l := 1; l <= m.State.Tier; l++ {
					drops = append(drops, item.NewStack(sgitem.TierKit{Level: l}, 1))
				}
			}
			if m.State != nil {
				for _, slots := range [][]int{m.State.Def.Input, m.State.Def.Output, m.State.Def.Charge} {
					for _, s := range slots {
						if it, _ := m.State.Inv().Item(s); !it.Empty() {
							drops = append(drops, it)
						}
					}
				}
			}
			return drops
		},
	}
}

func (m Machine) EncodeNBT() map[string]any {
	data := map[string]any{"id": "SkyGearMachine"}
	if m.State == nil {
		return data
	}
	var items []any
	for _, slots := range [][]int{m.State.Def.Input, m.State.Def.Output, m.State.Def.Charge} {
		for _, s := range slots {
			if it, _ := m.State.Inv().Item(s); !it.Empty() {
				nbt := item.WriteNBT(it, true)
				nbt["Slot"] = byte(s)
				items = append(items, nbt)
			}
		}
	}
	for i, it := range m.State.Legacy {
		nbt := item.WriteNBT(it, true)
		nbt["Slot"] = byte(legacySlot + i)
		items = append(items, nbt)
	}
	data["Items"] = items
	data["Progress"] = int32(m.State.Progress)
	data["Energy"] = int32(m.State.Energy)
	data["Unlimited"] = m.State.Unlimited
	data["Burn"] = int32(m.State.Burn)
	data["BurnMax"] = int32(m.State.burnMax)
	data["Mesh"] = m.State.Mesh
	data["Tier"] = int32(m.State.Tier)
	data["Mob"] = m.State.Mob
	data["Fluid"] = int32(m.State.Fluid)
	data["FluidAmt"] = int32(m.State.FluidAmt)
	data["Charges"] = int32(m.State.Charges)
	data["Boost"] = int32(m.State.Boost)
	data["Recipe"] = m.State.RecipeID
	data["Owner"] = m.State.Owner
	data["OwnerName"] = m.State.ownerName
	return data
}

func (m Machine) DecodeNBT(data map[string]any) any {
	owner, _ := data["Owner"].(string)
	ownerName, _ := data["OwnerName"].(string)
	n := New(m.Kind, owner, ownerName)
	if f, ok := data["Fluid"].(int32); ok {
		n.State.Fluid = fluidnet.Kind(f)
	}
	if a, ok := data["FluidAmt"].(int32); ok {
		n.State.FluidAmt = int(a)
	}
	if mob, ok := data["Mob"].(string); ok {
		n.State.Mob = mob
	}
	if c, ok := data["Charges"].(int32); ok {
		n.State.Charges = int(c)
	}
	if id, ok := data["Recipe"].(string); ok {
		n.State.RecipeID = id
		if m.Kind == KindLab {
			n.State.Lab = machinerecipe.LabByKind(id)
		}
	}
	if mesh, ok := data["Mesh"].(string); ok {
		n.State.Mesh = mesh
	}
	if t, ok := data["Tier"].(int32); ok && tierable(m.Kind) {
		n.State.Tier = max(0, min(int(t), MaxTier))
		n.Tier = n.State.Tier
	}
	if p, ok := data["Progress"].(int32); ok {
		n.State.Progress = min(int(p), n.State.Def.Interval)
	}
	if v, ok := data["Unlimited"].(bool); ok {
		n.State.Unlimited = v
	}
	for key, dst := range map[string]*int{"Energy": &n.State.Energy, "Burn": &n.State.Burn, "BurnMax": &n.State.burnMax, "Boost": &n.State.Boost} {
		if v, ok := data[key].(int32); ok {
			*dst = int(v)
		}
	}
	items, _ := data["Items"].([]any)
	for _, v := range items {
		if nbt, ok := v.(map[string]any); ok {
			slot, _ := nbt["Slot"].(byte)
			it := item.ReadNBT(nbt, nil)
			switch {
			case n.State.Role(int(slot)) != RoleNone:
				_ = n.State.Inv().SetItem(int(slot), it)
			case !it.Empty():
				n.State.Legacy = append(n.State.Legacy, it)
			}
		}
	}
	if m.Kind == KindLab && n.State.Lab == nil {
		// saved before the essence was chosen on the screen: it brews what
		// its ingredients are for
		if r, _ := n.State.labClosest(); r != nil {
			n.State.SetLab(r)
		}
	}
	n.State.ShowProgress()
	return n
}

// Activate opens the machine window, or uses a sieve.
func (m Machine) Activate(pos cube.Pos, _ cube.Face, tx *world.Tx, u item.User, ctx *item.UseContext) bool {
	p, ok := u.(*player.Player)
	s, ok2 := core.Of(tx)
	if !ok || !ok2 || !island.CanBuild(s, pos) || m.State == nil {
		return false
	}
	m.State.pos = pos
	if held, off := p.HeldItems(); !held.Empty() {
		if kit, ok := held.Item().(sgitem.TierKit); ok {
			UseKit(s, p, tx, pos, m, kit, held, off)
			return true
		}
	}
	if IsSieve(m.Kind) {
		useSieve(s, p, tx, pos, m.State)
		return true
	}
	if c := m.State.Def.Click; c != nil && c(s, p, tx, pos) {
		return true
	}
	if m.Kind == KindStorageLink {
		return true
	}
	if m.Kind == KindTinker {
		if !s.Debounced(p) {
			tinker.ShowTable(s, p, tx, pos)
		}
		return true
	}
	if m.Kind == KindLab {
		if !s.Debounced(p) {
			showLab(s, p, tx, pos, m.State)
		}
		return true
	}
	if m.State.Def.Crafter && p.Sneaking() {
		if !s.Debounced(p) {
			showCrafterRecipes(s, p, tx, pos, m.State, 0)
		}
		return true
	}
	if m.State.Def.FluidCap > 0 {
		held, _ := p.HeldItems()
		if UseFluid(s, p, tx, pos, m, held, ctx) || m.State.Def.Marker == "" {
			return true
		}
	}
	if m.State.Def.Cell {
		return false
	}
	m.State.OpenWindow(p)
	return true
}

// InsertItems implements itemio.Inserter: a machine takes inputs through any
// face but the bottom, which is its output. The hand sieve holds one block
// (it needs its mesh), so it takes one only while it is empty, and its look
// follows.
func (m Machine) InsertItems(tx *world.Tx, pos cube.Pos, face cube.Face, st item.Stack) int {
	if m.State != nil && m.State.Def.Insert != nil {
		return m.State.Def.Insert(tx, pos, st)
	}
	if m.State == nil || len(m.State.Def.Input) == 0 || face == cube.FaceDown {
		return 0
	}
	if m.Kind == KindSieve {
		if _, _, filled := m.State.FirstInput(); filled || !m.State.Accepts(st) || !m.State.Insert(st) {
			return 0
		}
		showSieve(tx, pos, m.State)
		return 1
	}
	n := 0
	for n < st.Count() && m.State.Insert(st) {
		n++
	}
	return n
}

// PeekItems implements itemio.Extractor: the output leaves through any face
// (a hopper under the machine, a conduit anywhere); machines do not push it
// into their neighbours.
func (m Machine) PeekItems(tx *world.Tx, pos cube.Pos, _ cube.Face, want func(item.Stack) bool) item.Stack {
	if m.State != nil && m.State.Def.Peek != nil {
		return m.State.Def.Peek(tx, pos, want)
	}
	if m.State == nil {
		return item.Stack{}
	}
	for _, s := range m.State.Def.Output {
		if it, _ := m.State.Inv().Item(s); !it.Empty() && (want == nil || want(it)) {
			return it
		}
	}
	return item.Stack{}
}

// ExtractItems implements itemio.Extractor.
func (m Machine) ExtractItems(tx *world.Tx, pos cube.Pos, _ cube.Face, st item.Stack, n int) item.Stack {
	if m.State != nil && m.State.Def.Extract != nil {
		return m.State.Def.Extract(tx, pos, st, n)
	}
	if m.State == nil {
		return item.Stack{}
	}
	got := 0
	var out item.Stack
	for _, s := range m.State.Def.Output {
		it, _ := m.State.Inv().Item(s)
		if got >= n || it.Empty() || !it.Comparable(st) {
			continue
		}
		take := min(n-got, it.Count())
		_ = m.State.Inv().SetItem(s, it.Grow(-take))
		got += take
		out = it.Grow(got - it.Count())
	}
	return out
}

// Tick runs the machine.
func (m Machine) Tick(currentTick int64, pos cube.Pos, tx *world.Tx) {
	s, ok := core.Of(tx)
	if !ok {
		return
	}
	if m.State == nil {
		// Placed without going through the place handler.
		tx.SetBlock(pos, New(m.Kind, "", ""), nil)
		return
	}
	st := m.State
	st.Sess, st.pos, st.Stuck = s, pos, stuckNone
	st.Mon.seen, FactoryOf(s).lastTick = currentTick, currentTick
	FactoryOf(s).machines[pos] = st
	st.DrainLegacy()
	reps := st.Reps()
	if st.hourglassable() {
		reps += climate.SnowReps(s, reps, currentTick) + st.HourglassReps(reps)
	}
	if st.Boost > 0 {
		st.Boost--
	}
	for i := range reps {
		st.free = i >= st.Reps()
		full := st.Def.Interval > 1 && st.Progress >= st.Def.Interval
		before, at := st.worked, st.Progress
		runMachine(s, st, pos, tx)
		if full && st.Progress == 0 && st.Def.CycleOnReset {
			st.Done++
		}
		// a machine that is stuck or did nothing has nothing to repeat
		if st.Stuck != stuckNone || (st.worked == before && st.Progress == at) {
			break
		}
	}
	st.free = false
	st.countWork(tx, m.Kind, currentTick)
	st.pos = pos
	if st.Def.Animated {
		showActive(tx, pos, st, currentTick)
	}
	switch st.Def.ID {
	case idEnergyCell, idTank, idCrucible, idLavaGenerator, idGate, idStorageController, idStorageLink, idTinker:
		if st.Def.ID == idEnergyCell && st.Energy > 0 {
			network.Energy(s).AddSource(pos)
		}
		return
	}
	if currentTick%10 == 0 {
		st.ShowEnergy()
		if (st.Stuck != stuckNone) != st.ShownStuck {
			_ = st.Inv().SetItem(infoSlot, st.InfoItem())
		}
	}
	if IsSieve(m.Kind) {
		// The sieve shows its content and progress on the model.
		showSieve(tx, pos, st)
		if st.Def.autoSieve {
			st.ShowProgress()
		}
		return
	}
	st.ShowProgress()
}

// runMachine runs the work of a machine once.
func runMachine(s *core.Session, st *State, pos cube.Pos, tx *world.Tx) {
	switch st.Def.ID {
	case idCrusher:
		st.process(func(in item.Stack) []item.Stack {
			out, _ := CrusherOutput(in)
			if bonus, ok := CrusherBonusOf(in); ok {
				out = out.Grow(bonus.Count() * sgitem.CrusherBonus)
			}
			return []item.Stack{out}
		}, func(in item.Stack) []item.Stack {
			out, _ := CrusherOutput(in)
			if bonus, ok := CrusherBonusOf(in); ok {
				for range sgitem.CrusherBonus {
					if s.Rand.IntN(100) < sgitem.CrusherBonusChance {
						out = out.Grow(bonus.Count())
					}
				}
			}
			return []item.Stack{out}
		})
	case idGenerator:
		runGenerator(s, st, pos, tx)
	case idAlloy:
		st.runAlloy()
	case idSimulator:
		runSimulator(s, st)
	case idCrucible:
		st.runCrucible()
	case idWasher:
		st.runWasher()
	case idLavaGenerator:
		runLavaGenerator(s, st, pos)
	case idAutoSieve, idMagicSieve:
		runAutoSieve(s, st, pos, tx)
	case idMiner:
		runMiner(s, st, pos, tx)
	case idFurnace:
		st.runFurnace()
	case idVacuum:
		runVacuum(s, st, pos, tx)
	case idFeller:
		runFeller(s, st, pos, tx)
	case idCrafter:
		runCrafter(s, st)
	case idLab:
		st.runLab()
	default:
		if st.Def.Run != nil {
			st.Def.Run(s, st, pos, tx)
		}
	}
}

// process advances an input → output operation on accepted inputs.
// possible returns every product one input item may give; a completed
// operation waits until they all fit. roll, if not nil, picks the actual
// products (otherwise all possible ones are given).
func (st *State) process(possible, roll func(in item.Stack) []item.Stack) {
	slot, in, ok := st.FirstInput()
	if !ok || !st.Accepts(in) {
		st.Progress = 0
		return
	}
	if st.Progress < st.Def.Interval {
		st.TickWork()
		return
	}
	out := possible(in)
	if !st.RoomFor(out) {
		return
	}
	if roll != nil {
		out = roll(in)
	}
	_ = st.Inv().SetItem(slot, in.Grow(-1))
	for _, it := range out {
		st.Emit(it)
	}
	st.Progress = 0
}

// Mineable reports whether the auto miner may mine b: natural blocks only,
// never machines, cores, drawers or other block entities.
func Mineable(b world.Block) (block.BreakInfo, bool) {
	if _, ok := b.(block.Air); ok || b == nil {
		return block.BreakInfo{}, false
	}
	if _, ok := b.(world.NBTer); ok {
		return block.BreakInfo{}, false
	}
	if _, ok := b.(world.Liquid); ok {
		return block.BreakInfo{}, false
	}
	br, ok := b.(block.Breakable)
	if !ok {
		return block.BreakInfo{}, false
	}
	info := br.BreakInfo()
	if info.Hardness < 0 || info.Harvestable == nil || !info.Harvestable(minerTool) {
		return block.BreakInfo{}, false
	}
	return info, true
}

// runMiner mines the block below every interval while the drops fit.
func runMiner(s *core.Session, st *State, pos cube.Pos, tx *world.Tx) {
	below := pos.Side(cube.FaceDown)
	info, ok := Mineable(tx.Block(below))
	if !ok {
		if st.cracking {
			viewCrack(tx, below, block.StopCrackAction{})
			st.cracking = false
		}
		st.Progress = 0
		return
	}
	if !st.cracking {
		// Players nearby see the block crack as if a player mined it.
		viewCrack(tx, below, block.StartCrackAction{BreakTime: time.Duration(st.Def.Interval-st.Progress) * time.Second / 20})
		st.cracking = true
	}
	if st.Progress < st.Def.Interval {
		st.TickWork()
		return
	}
	var drops []item.Stack
	if info.Drops != nil {
		drops = info.Drops(minerTool, nil)
	}
	if !st.RoomFor(drops) {
		return
	}
	viewCrack(tx, below, block.StopCrackAction{})
	st.cracking = false
	mined := tx.Block(below)
	tx.SetBlock(below, nil, nil)
	tx.AddParticle(below.Vec3Centre(), particle.BlockBreak{Block: mined})
	for _, it := range drops {
		st.Emit(it)
	}
	st.Progress = 0
	st.Done++
}

// viewCrack shows a block crack action to the players viewing pos.
func viewCrack(tx *world.Tx, pos cube.Pos, a world.BlockAction) {
	for _, v := range tx.Viewers(pos.Vec3Centre()) {
		v.ViewBlockAction(pos, a)
	}
}

// Place sets up a placed machine with a fresh state.
func Place(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, kind string) {
	m := New(kind, instance.AccountID(p), p.Name())
	account.Grant(p, achievement.FirstMachine)
	achievement.Count(p, "machine_place", 1)
	achievement.Count(p, "place_"+kind, 1)
	if m.Kind != KindSieve {
		progress.Advance(s, tx, advancement.Machine)
	}
	tx.Defer(func(tx *world.Tx) { tx.SetBlock(pos, m, nil) })
}

// Break closes the windows of a machine being broken.
func Break(s *core.Session, tx *world.Tx, pos cube.Pos, m Machine) {
	delete(FactoryOf(s).machines, pos)
	if m.State != nil {
		m.State.storage.CloseAll()
	}
}

func Item(kind string) item.Stack { return item.NewStack(Machine{Kind: kind}, 1) }
