package dragonfactory

import (
	"fmt"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/common/slotdeco"
	"github.com/presentkim/dragonfactory/internal/fluidnet"
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
	"github.com/presentkim/dragonfactory/common/chestui"
	"github.com/presentkim/dragonfactory/internal/account"
	"github.com/presentkim/dragonfactory/internal/instance"
)

// Machines are the automation blocks
// (docs/design/02-dragonfactory.md §11). Most run on energy (RE) and stop
// without it; each operation takes a fixed time. Items go in and out by hand in the machine window
// (an entity chest) or by hoppers: a hopper above or beside a machine
// inserts into its input, a hopper below extracts from its output.

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

// machineKind describes a kind of machine. Slots are window slots of a 27
// slot (3 row) window; the window inventory is the machine inventory.
type machineKind struct {
	// nameKey is the catalog key of the name (title).
	nameKey string
	// marker in the window title selects the machine's layout in the
	// resource pack (ui/df_machine.json via inventoryui's chest_screen.json).
	marker string
	// interval is the ticks one operation takes. Balancing drafts.
	interval  int
	input     []int
	output    []int
	indicator int
	// helpKeys are the catalog keys of the lines that explain it.
	helpKeys []string
	hash     uint64
	// generates is the RE per tick a generator makes while it burns; energyCap
	// the RE its buffer holds (docs/design/06-power-and-conduits.md).
	generates, energyCap int
	// fluidCap is the mB a tank of the machine holds (0: no tank); fluidSource and
	// fluidSink say whether fluid conduits take fluid from it and put it in;
	// fluidOnly restricts it to one fluid (fluidnet.None: any).
	fluidCap               int
	fluidSource, fluidSink bool
	fluidOnly              fluidnet.Kind
	// controller marks the storage controller (storage.go).
	controller bool
	// gate marks the dimension gate (wave.go): using it opens its form.
	gate bool
	// cell makes the machine an energy store: it takes energy from the network and
	// gives it back, without working (energyCap is its capacity).
	cell bool
	// use is the RE per tick a machine needs while it works (0: it needs
	// none).
	use int
	// crafter marks the auto crafter, whose window has a recipe slot.
	crafter bool
	// charge are the slots that take the essence of a magic sieve or the bone
	// meal of the tree feller (takesCharge); autoSieve
	// marks the sieves that sift on their own (sieve.go).
	charge    []int
	autoSieve bool
}

var machineKinds = map[string]*machineKind{
	KindCrusher: {
		nameKey: "dragonfactory.machine.init.1", marker: "§d§m§1", interval: 80, input: []int{11}, output: []int{15}, indicator: 13,
		use: 20, energyCap: consumerBuffer,
		helpKeys: []string{"dragonfactory.machine.init.2", "dragonfactory.machine.init.3"},
	},
	// Sieves have no window: the input lies on the mesh and the products
	// drop (or, from the auto sieve, go into containers beside or below).
	KindSieve: {
		nameKey: "dragonfactory.machine.init.4", interval: 60, input: []int{10}, indicator: 12,
		helpKeys: []string{"dragonfactory.machine.init.5", "dragonfactory.machine.init.6"},
	},
	KindAutoSieve: {
		// The window shows the input (fed by hand, a hopper or a conduit), the ring
		// and the four slots the products go into (ui/df_machine.json sieve_slots).
		nameKey: "dragonfactory.machine.init.7", marker: "§d§m§2", interval: 60, input: []int{10}, output: []int{13, 14, 15, 16}, indicator: 12,
		use: 10, energyCap: consumerBuffer, autoSieve: true,
		helpKeys: []string{"dragonfactory.machine.init.8", "dragonfactory.machine.init.9"},
	},
	KindMagicSieve: {
		// Like the auto sieve, with the essence slot between the input and the ring
		// (ui/df_machine.json magicsieve_slots).
		nameKey: "machine.magicsieve.name", marker: "§d§m§9", interval: 90, input: []int{10}, charge: []int{11}, output: []int{13, 14, 15, 16}, indicator: 12,
		use: 40, energyCap: consumerBuffer, autoSieve: true,
		helpKeys: []string{"machine.magicsieve.help.1", "machine.magicsieve.help.2", "machine.magicsieve.help.3"},
	},
	KindMiner: {
		// One output slot (11th playtest; it had four, like the harvester).
		nameKey: "dragonfactory.machine.init.10", marker: "§d§m§5", interval: 40, output: []int{13}, indicator: 10,
		use: 20, energyCap: consumerBuffer,
		helpKeys: []string{"dragonfactory.machine.init.11"},
	},
	KindHarvester: {
		nameKey: "dragonfactory.machine.init.13", marker: "§d§m§3", interval: 40, output: []int{13, 14, 15, 16}, indicator: 10,
		use: 10, energyCap: consumerBuffer,
		helpKeys: []string{"dragonfactory.machine.init.14"},
	},
	KindGenerator: {
		nameKey: "generator.name", marker: "§d§m§5", interval: 100, input: []int{13}, indicator: 10,
		generates: generatorOutput, energyCap: generatorBuffer,
		helpKeys: []string{"generator.help.1", "generator.help.2"},
	},
	KindAlloy: {
		nameKey: "machine.alloy.name", marker: "§d§m§6", interval: 100, input: []int{10, 11}, output: []int{15}, indicator: 13,
		use: 20, energyCap: consumerBuffer,
		helpKeys: []string{"machine.alloy.help.1", "machine.alloy.help.2"},
	},
	KindStorageController: {
		nameKey: "machine.controller.name", interval: 1, indicator: 10, controller: true,
		helpKeys: []string{"machine.controller.help.1", "machine.controller.help.2"},
	},
	KindStorageLink: {
		nameKey: "machine.link.name", interval: 1, indicator: 10,
		helpKeys: []string{"machine.link.help.1"},
	},
	KindTinker: {
		nameKey: "machine.tinker.name", interval: 1, indicator: 10,
		helpKeys: []string{"machine.tinker.help.1", "machine.tinker.help.2"},
	},
	KindGate: {
		nameKey: "machine.gate.name", interval: 1, indicator: 10, gate: true,
		helpKeys: []string{"machine.gate.help.1", "machine.gate.help.2"},
	},
	KindWasher: {
		nameKey: "machine.washer.name", marker: "§d§m§2", interval: 60, input: []int{10}, output: []int{13, 14, 15, 16}, indicator: 12,
		use: 40, energyCap: consumerBuffer,
		fluidCap: 4000, fluidSink: true, fluidOnly: fluidnet.Water,
		helpKeys: []string{"machine.washer.help.1", "machine.washer.help.2"},
	},
	KindTank: {
		nameKey: "machine.tank.name", interval: 1, indicator: 10,
		fluidCap: tankCapacity, fluidSource: true, fluidSink: true,
		helpKeys: []string{"machine.tank.help.1", "machine.tank.help.2"},
	},
	KindCrucible: {
		nameKey: "machine.crucible.name", interval: 20, input: []int{10}, indicator: 12,
		use: 30, energyCap: consumerBuffer,
		fluidCap: 8000, fluidSource: true, fluidOnly: fluidnet.Lava,
		helpKeys: []string{"machine.crucible.help.1", "machine.crucible.help.2"},
	},
	KindLavaGenerator: {
		nameKey: "machine.lavagen.name", interval: 1, indicator: 10,
		generates: lavaGenOutput, energyCap: generatorBuffer,
		fluidCap: 4000, fluidSink: true, fluidOnly: fluidnet.Lava,
		helpKeys: []string{"machine.lavagen.help.1", "machine.lavagen.help.2"},
	},
	KindSimulator: {
		nameKey: "machine.simulator.name", marker: "§d§m§2", interval: 200, input: []int{10}, output: []int{13, 14, 15, 16}, indicator: 12,
		use: 40, energyCap: consumerBuffer,
		helpKeys: []string{"machine.simulator.help.1", "machine.simulator.help.2"},
	},
	KindEnergyCell: {
		nameKey: "machine.cell.name", interval: 1, indicator: 10,
		cell: true, energyCap: cellCapacity,
		helpKeys: []string{"machine.cell.help.1", "machine.cell.help.2"},
	},
	KindFurnace: {
		nameKey: "machine.furnace.name", marker: "§d§m§1", interval: 100, input: []int{11}, output: []int{15}, indicator: 13,
		use: 30, energyCap: consumerBuffer,
		helpKeys: []string{"machine.furnace.help.1", "machine.furnace.help.2"},
	},
	KindVacuum: {
		nameKey: "machine.vacuum.name", marker: "§d§m§3", interval: 10, output: []int{13, 14, 15, 16}, indicator: 10,
		use: 5, energyCap: consumerBuffer,
		helpKeys: []string{"machine.vacuum.help.1", "machine.vacuum.help.2"},
	},
	KindFeller: {
		nameKey: "machine.feller.name", marker: "§d§m§9", interval: 60, input: []int{10}, charge: []int{11}, output: []int{13, 14, 15, 16}, indicator: 12,
		use: 15, energyCap: consumerBuffer,
		helpKeys: []string{"machine.feller.help.1", "machine.feller.help.2", "machine.feller.help.3"},
	},
	KindCrafter: {
		// The store is the 3×3 on the left, the recipe slot above the ring.
		nameKey: "machine.crafter.name", marker: "§d§m§7", interval: 40, input: []int{0, 1, 2, 9, 10, 11, 18, 19, 20}, output: []int{6, 7, 15, 16}, indicator: 13,
		use: 25, energyCap: consumerBuffer, crafter: true,
		helpKeys: []string{"machine.crafter.help.1", "machine.crafter.help.2", "machine.crafter.help.3"},
	},
	KindLab: {
		// The materials are the 3×3 on the left, the essences go to the four
		// slots on the right.
		nameKey: "machine.lab.name", marker: "§d§m§8", interval: 320, input: []int{0, 1, 2, 9, 10, 11, 18, 19, 20}, output: []int{6, 7, 15, 16}, indicator: 13,
		use: 40, energyCap: consumerBuffer,
		helpKeys: []string{"machine.lab.help.1", "machine.lab.help.2", "machine.lab.help.3"},
	},
	KindSellBox: {
		nameKey: "dragonfactory.machine.init.16", marker: "§d§m§4", interval: 20, input: []int{9, 10, 11, 12, 13, 14, 15, 16, 17}, indicator: 4,
		helpKeys: []string{"dragonfactory.machine.init.17", "dragonfactory.machine.init.18"},
	},
}

// machineOrder is the registration and display order.
var machineOrder = []string{KindCrusher, KindSieve, KindAutoSieve, KindMiner, KindSellBox, KindHarvester, KindGenerator, KindAlloy, KindEnergyCell, KindSimulator, KindTank, KindCrucible, KindLavaGenerator, KindWasher, KindGate, KindStorageController, KindStorageLink, KindTinker, KindFurnace, KindVacuum, KindFeller, KindCrafter, KindLab, KindMagicSieve}

// crushRecipes maps an input item key to its output.
var crushRecipes = map[string]world.Item{
	itemKey(block.Stone{}):       block.Cobblestone{},
	itemKey(block.Cobblestone{}): block.Gravel{},
	itemKey(block.Gravel{}):      block.Sand{},
	// Sand crushes into dust (13th playtest, sky progression), red sand too.
	itemKey(block.Sand{}):          Dust{},
	itemKey(block.Sand{Red: true}): Dust{},
	// The stones of the deep, the Nether and the End (Ex Deorum, ATM10: To
	// the Sky; docs/design/16-ore-balance.md §2.1).
	itemKey(block.Deepslate{}):                               CrushedBlock{Kind: CrushedDeepslate},
	itemKey(block.Deepslate{Type: block.CobbledDeepslate()}): CrushedBlock{Kind: CrushedDeepslate},
	itemKey(block.Netherrack{}):                              CrushedBlock{Kind: CrushedNetherrack},
	itemKey(CrushedBlock{Kind: CrushedNetherrack}):           block.Sand{Red: true},
	itemKey(block.Blackstone{}):                              CrushedBlock{Kind: CrushedBlackstone},
	itemKey(block.EndStone{}):                                CrushedBlock{Kind: CrushedEndStone},
}

// crusherOutput is what the crusher always makes of an item: the blocks of
// the hammer chain one step down, an ore block crusherRaw raw ores (and
// crusherBonusOf more by chance).
func crusherOutput(in item.Stack) (item.Stack, bool) {
	if metal, ok := oreBlockMetal(in.Item()); ok {
		return item.NewStack(rawOf(metal), crusherRaw), true
	}
	if out, ok := crushRecipes[itemKey(in.Item())]; ok {
		return item.NewStack(out, 1), true
	}
	return item.Stack{}, false
}

// crusherBonusOf is the raw ore the crusher may make more of an ore block:
// crusherBonus of them at crusherBonusChance percent each.
func crusherBonusOf(in item.Stack) (item.Stack, bool) {
	metal, ok := oreBlockMetal(in.Item())
	if !ok {
		return item.Stack{}, false
	}
	return item.NewStack(rawOf(metal), 1), true
}

// minerTool is what the auto miner mines with.
var minerTool = item.Pickaxe{Tier: item.ToolTierStone}

// Machine is a machine block. st is nil for machine items and is created
// when the machine is placed or loaded.
type Machine struct {
	Kind string
	st   *machineState
	// content and fill are what a sieve shows on its mesh (sieve.go).
	content, fill, meshLook int
	// active is the working look of an animated machine (animate.go).
	active bool
	// tier is the tier look of the block (tierlook.go).
	tier int
}

// machineState is the inventory and progress of a placed machine. It is
// owned by the machine's world.
type machineState struct {
	kind     *machineKind
	storage  *chestui.Storage
	progress int
	shownPct int
	// shownEnergy is the buffer the energy gauge shows now (-1: nothing yet).
	shownEnergy int
	// owner is the account that placed a sell box and gets its money.
	owner, ownerName string
	// pos is where the machine stands, set when it ticks or its window is
	// opened.
	pos cube.Pos
	// cracking is set while the auto miner shows its target cracking.
	cracking bool
	// lastUse is when a sieve was last sifted by hand.
	lastUse time.Time
	// mesh is the tier of the mesh fitted in a sieve ("" none, mesh.go).
	mesh string
	// unlimited makes the machine run without energy (the operator's load
	// test factories); stalled is set while a machine that works has no
	// energy for a tick.
	unlimited, stalled bool
	// energy is the RE in the buffer; burn the ticks the fuel in a generator
	// burns still, burnMax the ticks it burned for in all (generator.go).
	energy, burn, burnMax int
	// legacy are saved items of slots the kind no longer has (the auto
	// miner had four output slots); they move into the output as it frees
	// up, and are saved until then.
	legacy []item.Stack
	// fluid and fluidAmt are what the tank holds (fluidmachine.go).
	fluid    fluidnet.Kind
	fluidAmt int
	// mob and charges are what the mob simulator is charged with (essence.go).
	mob     string
	charges int
	// recipe is the recipe an auto crafter makes (recipeID is how it is saved);
	// shownRecipe is the one the recipe slot shows; sample is an item put on
	// the recipe slot to choose by, dropRecipe a request to clear it.
	recipe      *crafterRecipe
	recipeID    string
	shownRecipe string
	sample      item.Stack
	dropRecipe  bool
	// pending are the products of an auto sieve that did not fit its output slots
	// yet: it waits until they do.
	pending []item.Stack
	// done counts the work cycles finished and made the RE generated since
	// they were last counted for the players' achievements (counter.go).
	done, made int
	// tally are the counts to pass to the players' counters at the next
	// tick (counter.go); lab is the recipe the laboratory works on, period
	// the length of the cycle it is on (0: the kind's interval).
	tallyCounts map[string]int64
	lab         *LabRecipe
	period      int
	// sess is the session the machine ticks in; stuck why the last tick
	// could not work, didWork whether a tick worked since the last second,
	// mon what the stall alerts know of it (monitor.go).
	sess  *session
	stuck stuckKind
	// shownStuck is whether the info icon of the window shows the warning
	// badge (a stalled machine).
	shownStuck bool
	didWork    bool
	mon        monitor
	// tier is the machine's tier (tier.go); worked counts the ticks of work
	// done, to tell whether a run did anything.
	tier, worked int
	// stallTicks counts the ticks in a row the machine had no energy.
	stallTicks int
	// seenWorked and lastWorkTick tell when the machine last worked (animate.go).
	seenWorked   int
	lastWorkTick int64
}

// cycleLen is the ticks of the cycle the machine is on.
func (st *machineState) cycleLen() int {
	if st.period > 0 {
		return st.period
	}
	return st.kind.interval
}

// legacySlot is the first saved slot number of legacy items (not a window
// slot).
const legacySlot = 100

// drainLegacy moves a legacy item into the output when it fits.
func (st *machineState) drainLegacy() {
	if len(st.legacy) == 0 || len(st.kind.output) == 0 {
		return
	}
	if it := st.legacy[0]; fits(st.inv(), st.kind.output, []item.Stack{it}) {
		put(st.inv(), st.kind.output, it)
		st.legacy = st.legacy[1:]
	}
}

func newMachine(kind, owner, ownerName string) Machine {
	k := machineKinds[kind]
	st := &machineState{kind: k, storage: chestui.NewStorage(27, k.windowTitle(i18n.Default)), owner: owner, ownerName: ownerName, shownPct: -1, shownEnergy: -1}
	inv := st.storage.Inventory()
	for slot := range 27 {
		if st.role(slot) == roleNone {
			_ = inv.SetItem(slot, filler())
		}
	}
	_ = inv.SetItem(infoSlot, st.infoItem())
	// A button is only there when the machine has the slots it moves.
	_ = inv.SetItem(takeAllSlot, item.Stack{})
	_ = inv.SetItem(putAllSlot, item.Stack{})
	if len(k.output) > 0 {
		_ = inv.SetItem(takeAllSlot, withWindowText(item.NewStack(MachineTakeAll{}, 1), wtext{wraw("§r§f"), wkey("machine.takeAll")}))
	}
	if len(k.input)+len(k.charge) > 0 {
		_ = inv.SetItem(putAllSlot, withWindowText(item.NewStack(MachinePutAll{}, 1), wtext{wraw("§r§f"), wkey("machine.putAll")}))
	}
	inv.Handle(machineHandler{st: st})
	st.shownRecipe = "\x00"
	if k.crafter {
		st.showRecipe()
	}
	st.showProgress()
	st.showEnergy()
	return Machine{Kind: kind, st: st}
}

func filler() item.Stack {
	return item.NewStack(block.StainedGlassPane{Colour: item.ColourGrey()}, 1).WithCustomName("§r ")
}

type slotRole int

const (
	roleNone slotRole = iota
	roleInput
	roleOutput
	// roleRecipe is the recipe slot of the auto crafter: it takes nothing.
	roleRecipe
	// roleCharge is the essence slot of a magic sieve and the bone meal slot
	// of the tree feller.
	roleCharge
)

// takeAllSlot and putAllSlot are the window slots of the two buttons left of
// the info icon: "take everything out" moves the output slots into the
// player's inventory, "put everything in" fills the input slots from it.
// Pressing one only runs the move; the slot never changes hands.
const (
	takeAllSlot = 24
	putAllSlot  = 25
)

// infoSlot is the window slot of the info icon: its tooltip explains the
// machine. The theme draws it left of the close button.
const infoSlot = 26

// title is the name of the kind in lang.
func (k *machineKind) title(lang string) string { return i18n.T(lang, k.nameKey) }

// windowTitle is the title of the kind's window in lang. The marker's last
// code is a colour (§1 dark blue ...): §r gives the name the window's own
// title colour back.
func (k *machineKind) windowTitle(lang string) string { return k.marker + "§r" + k.title(lang) }

// openWindow opens the machine's window for p, titled in p's language.
func (st *machineState) openWindow(p *player.Player) {
	st.storage.SetTitle(st.kind.windowTitle(i18n.Of(p)))
	st.storage.Open(p, nil)
}

// infoItem is the info icon of a machine window: the machine's name, what
// it does and, for the sell box, whose it is.
func (st *machineState) infoItem() item.Stack {
	var lore []wtext
	for _, k := range st.kind.helpKeys {
		lore = append(lore, wtext{wkey(k)})
	}
	if st.kind == machineKinds[KindSellBox] && st.ownerName != "" {
		lore = append(lore, wtext{wkey("dragonfactory.machine.infoItem.1"), wraw(st.ownerName)})
	}
	if st.tier > 0 {
		spec := st.spec()
		lore = append(lore, wtext{wkey("tier.info", wkey(tierNameKey(st.tier)), spec.speed, float64(spec.use10)/10)})
	}
	badge := 0
	if st.stuck != stuckNone {
		badge = slotdeco.BadgeWarn
	}
	st.shownStuck = badge != 0
	name := wtext{wraw(slotdeco.Common.Mark() + "§r§f"), wkey(st.kind.nameKey)}
	if st.tier > 0 {
		name = append(name, wraw(" ["), wkey(tierNameKey(st.tier)), wraw("]"))
	}
	name = append(name, wraw(slotdeco.TierMark(badge)))
	return withWindowText(item.NewStack(MachineInfo{}, 1), name, lore...)
}

// MachineTakeAll and MachinePutAll are the button items of machine windows
// (takeAllSlot, putAllSlot). They only ever sit in machine windows.
type MachineTakeAll struct{}

func (MachineTakeAll) EncodeItem() (string, int16) { return "dragonfactory:machine_take_all", 0 }
func (MachineTakeAll) Name() string                { return i18n.T(i18n.Default, "name.MachineTakeAll") }
func (MachineTakeAll) Category() category.Category { return category.Items() }

type MachinePutAll struct{}

func (MachinePutAll) EncodeItem() (string, int16) { return "dragonfactory:machine_put_all", 0 }
func (MachinePutAll) Name() string                { return i18n.T(i18n.Default, "name.MachinePutAll") }
func (MachinePutAll) Category() category.Category { return category.Items() }

// MachineInfo is the info icon item of machine windows. It only ever sits
// in machine windows.
type MachineInfo struct{}

func (MachineInfo) EncodeItem() (string, int16) { return "dragonfactory:machine_info", 0 }
func (MachineInfo) Name() string                { return i18n.T(i18n.Default, "name.MachineInfo") }
func (MachineInfo) Category() category.Category { return category.Items() }

func (st *machineState) role(slot int) slotRole {
	if st.kind.crafter && slot == crafterRecipeSlot {
		return roleRecipe
	}
	for _, s := range st.kind.input {
		if s == slot {
			return roleInput
		}
	}
	for _, s := range st.kind.output {
		if s == slot {
			return roleOutput
		}
	}
	for _, s := range st.kind.charge {
		if s == slot {
			return roleCharge
		}
	}
	return roleNone
}

func (st *machineState) inv() *inventory.Inventory { return st.storage.Inventory() }

// accepts reports whether it may go into the input of the machine.
// Time-limited items are never processed or sold.
func (st *machineState) accepts(it item.Stack) bool {
	if it.Empty() || !plain(it) {
		return false
	}
	k := itemKey(it.Item())
	switch st.kind {
	case machineKinds[KindCrusher]:
		_, ok := crusherOutput(it)
		return ok
	case machineKinds[KindGenerator]:
		return fuelDuration(it) > 0
	case machineKinds[KindAlloy]:
		return alloyAccepts(it)
	case machineKinds[KindSimulator]:
		return acceptsEssence(it)
	case machineKinds[KindCrucible]:
		return k == itemKey(block.Cobblestone{})
	case machineKinds[KindWasher]:
		_, ok := washerOutput(it)
		return ok
	case machineKinds[KindFurnace]:
		_, ok := smeltOf(it)
		return ok
	case machineKinds[KindFeller]:
		_, ok := it.Item().(block.Sapling)
		return ok || st.takesCharge(it)
	case machineKinds[KindCrafter]:
		return st.recipe != nil && st.recipe.wants(it)
	case machineKinds[KindLab]:
		return st.labAccepts(it)
	case machineKinds[KindSieve], machineKinds[KindAutoSieve], machineKinds[KindMagicSieve]:
		// A sieve sifts only with a mesh fitted; the magic sieve also takes essences
		// (into its essence slot).
		_, ok := sieveTables[st.mesh][k]
		return ok || st.takesCharge(it)
	}
	// The sell box (no more shop, 13th playtest) takes nothing.
	return false
}

// takesCharge reports whether it goes into the charge slots of the machine:
// the essence of a magic sieve, the bone meal of the tree feller.
func (st *machineState) takesCharge(it item.Stack) bool {
	switch {
	case len(st.kind.charge) == 0:
		return false
	case st.kind == machineKinds[KindFeller]:
		_, ok := it.Item().(item.BoneMeal)
		return ok
	}
	return acceptsEssence(it)
}

// slotAccepts reports whether the item may be put on a slot of the machine:
// an input slot takes what the machine works on, except what goes into its
// charge slots.
func (st *machineState) slotAccepts(slot int, it item.Stack) bool {
	switch st.role(slot) {
	case roleInput:
		return st.accepts(it) && !st.takesCharge(it)
	case roleCharge:
		return st.takesCharge(it)
	}
	return false
}

// machineHandler limits moves in the machine window to its input and
// output slots.
type machineHandler struct{ st *machineState }

func (h machineHandler) HandleTake(ctx *inventory.Context, slot int, _ item.Stack) {
	if slot == takeAllSlot || slot == putAllSlot {
		// A button: nothing moves by hand. The move runs after this click
		// is answered, so the client's view is fixed first and the changes
		// follow it.
		ctx.Cancel()
		if p, ok := ctx.Val().(*player.Player); ok {
			put := slot == putAllSlot
			p.H().Do(func(_ *world.Tx, e world.Entity) {
				if put {
					h.st.putAll(e.(*player.Player))
				} else {
					h.st.takeAll(e.(*player.Player))
				}
			})
		}
		return
	}
	if h.st.role(slot) == roleRecipe {
		// Taking the recipe off the slot clears it.
		ctx.Cancel()
		h.st.dropRecipe = true
		return
	}
	if h.st.role(slot) == roleNone {
		ctx.Cancel()
	}
}

func (h machineHandler) HandlePlace(ctx *inventory.Context, slot int, it item.Stack) {
	if h.st.role(slot) == roleRecipe {
		// An item put on the recipe slot chooses the recipe that makes it
		// and goes back to the player.
		ctx.Cancel()
		h.st.sample = it
		return
	}
	if !h.st.slotAccepts(slot, it) {
		ctx.Cancel()
	}
}

func (h machineHandler) HandleDrop(ctx *inventory.Context, slot int, _ item.Stack) {
	if h.st.role(slot) == roleNone || h.st.role(slot) == roleRecipe {
		ctx.Cancel()
	}
}

// takeAll moves the output slots into p's inventory as far as it fits; what
// does not fit stays in the machine.
func (st *machineState) takeAll(p *player.Player) {
	inv := st.inv()
	for _, slot := range st.kind.output {
		it, _ := inv.Item(slot)
		if it.Empty() {
			continue
		}
		if n, _ := p.Inventory().AddItem(it); n > 0 {
			_ = inv.SetItem(slot, it.Grow(-n))
		}
	}
}

// putAll moves what the machine takes from p's inventory into its input
// slots: onto a stack of the same kind first; a kind that no input slot holds
// yet goes into one empty slot. Items the machine does not take stay.
func (st *machineState) putAll(p *player.Player) {
	inv := st.inv()
	slots := append(append([]int{}, st.kind.input...), st.kind.charge...)
	for pslot, it := range p.Inventory().Slots() {
		if it.Empty() {
			continue
		}
		left, held := it.Count(), false
		for _, s := range slots {
			cur, _ := inv.Item(s)
			if cur.Empty() || !cur.Comparable(it) || !st.slotAccepts(s, it) {
				continue
			}
			held = true
			n := min(left, cur.MaxCount()-cur.Count())
			if n > 0 {
				_ = inv.SetItem(s, cur.Grow(n))
				left -= n
			}
		}
		for _, s := range slots {
			if left == 0 || held {
				break
			}
			if cur, _ := inv.Item(s); cur.Empty() && st.slotAccepts(s, it) {
				n := min(left, it.MaxCount())
				_ = inv.SetItem(s, it.Grow(n-it.Count()))
				left -= n
				break
			}
		}
		if left < it.Count() {
			_ = p.Inventory().SetItem(pslot, it.Grow(left-it.Count()))
		}
	}
}

// fits reports whether every stack fits into slots together.
func fits(inv *inventory.Inventory, slots []int, stacks []item.Stack) bool {
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
func (st *machineState) firstInput() (int, item.Stack, bool) {
	for _, s := range st.kind.input {
		if it, _ := st.inv().Item(s); !it.Empty() {
			return s, it, true
		}
	}
	return 0, item.Stack{}, false
}

// insert puts one of it into the input, reporting success.
func (st *machineState) insert(it item.Stack) bool {
	one := it.Grow(1 - it.Count())
	slots := st.kind.input
	if st.takesCharge(one) {
		slots = st.kind.charge
	}
	if !st.accepts(one) || !fits(st.inv(), slots, []item.Stack{one}) {
		return false
	}
	if st.kind == machineKinds[KindLab] && st.labHave()[itemKey(one.Item())] >= st.labFeedCap(one) {
		return false
	}
	put(st.inv(), slots, one)
	return true
}

// outputCount counts the items of a kind in the output slots.
func outputCount(st *machineState, it world.Item) int {
	n := 0
	for _, slot := range st.kind.output {
		if got, _ := st.inv().Item(slot); !got.Empty() && itemKey(got.Item()) == itemKey(it) {
			n += got.Count()
		}
	}
	return n
}

// takeOutput removes one item from the output.
func (st *machineState) takeOutput() (item.Stack, bool) {
	for _, s := range st.kind.output {
		if it, _ := st.inv().Item(s); !it.Empty() {
			_ = st.inv().SetItem(s, it.Grow(-1))
			return it.Grow(1 - it.Count()), true
		}
	}
	return item.Stack{}, false
}

// showProgress updates the indicator item when the shown percent changes.
// The indicator is a progress arrow item (ProgressArrow), so the window
// shows a filling arrow between input and output.
func (st *machineState) showProgress() {
	pct := min(100, st.progress*100/st.cycleLen())
	pct -= pct % 10
	if pct == st.shownPct {
		return
	}
	st.shownPct = pct
	name := wkey("dragonfactory.machine.showProgress.1", pct)
	it := item.NewStack(ProgressArrow{Step: pct / 10}, 1)
	switch st.kind {
	case machineKinds[KindSellBox]:
		// Machines that work in a repeating cycle, the auto miner and the generator
		// show a ring that fills.
		name, it = wkey("dragonfactory.machine.showProgress.2"), item.NewStack(CycleGauge{Step: pct / 10}, 1)
	case machineKinds[KindMiner], machineKinds[KindAutoSieve], machineKinds[KindMagicSieve], machineKinds[KindSimulator], machineKinds[KindWasher], machineKinds[KindVacuum], machineKinds[KindFeller], machineKinds[KindCrafter], machineKinds[KindLab]:
		name, it = wkey("dragonfactory.machine.showProgress.3", pct), item.NewStack(CycleGauge{Step: pct / 10}, 1)
	case machineKinds[KindGenerator]:
		// The ring shows the fuel left in the one burning.
		name, it = wkey("dragonfactory.machine.showProgress.5", pct), item.NewStack(CycleGauge{Step: pct / 10}, 1)
	case machineKinds[KindHarvester]:
		name, it = wkey("dragonfactory.machine.showProgress.4"), item.NewStack(CycleGauge{Step: pct / 10}, 1)
	}
	_ = st.inv().SetItem(st.kind.indicator, withWindowText(it, wtext{wraw("§r"), name}))
}

// energyGaugeSlot is the window slot of the energy gauge, in the column right
// of the machine's slots (ui/df_machine.json).
const energyGaugeSlot = 17

// showEnergy updates the energy gauge of a machine with a window that holds
// energy: a battery filled in tenths, whose name is the energy it holds and the
// most it can (shown when the player hovers it). A machine that runs without
// energy (the operator's test factory) shows a full one.
func (st *machineState) showEnergy() {
	if st.kind.marker == "" || st.kind.energyCap == 0 {
		return
	}
	capacity := st.energyCap()
	have := st.energy
	if st.unlimited {
		have = capacity
	}
	if have == st.shownEnergy {
		return
	}
	st.shownEnergy = have
	it := item.NewStack(EnergyGauge{Step: min(10, have*10/capacity)}, 1)
	_ = st.inv().SetItem(energyGaugeSlot, withWindowText(it, wtext{wkey("machine.energy.gauge", have, capacity)}))
}

// EnergyGauge is the energy indicator of machine windows: a battery filled in
// Step tenths. It only ever sits in machine windows.
type EnergyGauge struct{ Step int }

func (g EnergyGauge) EncodeItem() (string, int16) {
	return fmt.Sprintf("dragonfactory:energy_gauge_%d", g.Step), 0
}
func (g EnergyGauge) Name() string              { return i18n.T(i18n.Default, "name.EnergyGauge") }
func (EnergyGauge) Category() category.Category { return category.Items() }

// ProgressArrow is the indicator item of machine windows: an arrow filled
// in Step tenths. It only ever sits in machine windows.
type ProgressArrow struct{ Step int }

// progressSteps is the number of ProgressArrow steps (0 to 10).
const progressSteps = 11

func (a ProgressArrow) EncodeItem() (string, int16) {
	return fmt.Sprintf("dragonfactory:progress_%d", a.Step), 0
}
func (a ProgressArrow) Name() string              { return i18n.T(i18n.Default, "name.ProgressArrow") }
func (ProgressArrow) Category() category.Category { return category.Items() }

// CycleGauge is the indicator of machines that work in a repeating cycle
// (auto harvester, sell box): a hollow ring filled in Step tenths, which
// turns over and over like a loading circle.
type CycleGauge struct{ Step int }

func (g CycleGauge) EncodeItem() (string, int16) {
	return fmt.Sprintf("dragonfactory:cycle_%d", g.Step), 0
}
func (g CycleGauge) Name() string              { return i18n.T(i18n.Default, "name.CycleGauge") }
func (CycleGauge) Category() category.Category { return category.Items() }

func (m Machine) kind() *machineKind { return machineKinds[m.Kind] }

func (m Machine) EncodeItem() (string, int16) { return "dragonfactory:" + m.Kind, 0 }
func (m Machine) EncodeBlock() (string, map[string]any) {
	if isSieve(m.Kind) {
		return "dragonfactory:" + m.Kind, map[string]any{sieveLookState: lookValue(m.meshLook, m.content, m.fill)}
	}
	if tierLook(m.Kind) {
		return "dragonfactory:" + m.Kind, map[string]any{tierState: tierValue(m.Kind, m.tier, m.active)}
	}
	if animatedMachines[m.Kind] {
		return "dragonfactory:" + m.Kind, map[string]any{activeState: activeValue(m.active)}
	}
	return "dragonfactory:" + m.Kind, nil
}
func (m Machine) Hash() (uint64, uint64) {
	if m.active || m.tier > 0 {
		return m.kind().hash, uint64(tierLookIndex(m.Kind, m.tier, m.active))
	}
	return m.kind().hash, uint64(lookIndex(m.meshLook, m.content, m.fill))
}
func (Machine) Model() world.BlockModel { return model.Solid{} }
func (m Machine) Name() string {
	if m.st != nil {
		return tierTitle(i18n.Default, m.kind().title(i18n.Default), m.st.tier)
	}
	return m.kind().title(i18n.Default)
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
	if isSieve(m.Kind) {
		return sieveStates()
	}
	if tierLook(m.Kind) {
		return tierStates(m.Kind)
	}
	if animatedMachines[m.Kind] {
		return activeStates()
	}
	return nil
}

// Permutations implements block.Permutable.
func (m Machine) Permutations() []customblock.Permutation {
	if isSieve(m.Kind) {
		return sievePermutations(m.Kind)
	}
	if tierLook(m.Kind) {
		return tierPermutations(m.Kind)
	}
	if animatedMachines[m.Kind] {
		return activePermutations(m.Kind)
	}
	return nil
}

func (m Machine) Properties() customblock.Properties {
	if cubeMachines[m.Kind] {
		return cubeProperties(m.Kind)
	}
	if frontMachines[m.Kind] {
		return frontCubeProperties(m.Kind)
	}
	return sieveProperties(m.Kind)
}

// BreakInfo drops the machine and its contents.
func (m Machine) BreakInfo() block.BreakInfo {
	return block.BreakInfo{
		Hardness: 1.5, BlastResistance: 6,
		Harvestable: func(item.Tool) bool { return true },
		Effective:   customBlockEffective,
		Drops: func(item.Tool, []item.Enchantment) []item.Stack {
			drops := []item.Stack{item.NewStack(Machine{Kind: m.Kind}, 1)}
			if m.st != nil {
				drops = append(drops, m.st.pending...)
			}
			if m.st != nil && m.st.mesh != "" {
				drops = append(drops, item.NewStack(Mesh{Tier: m.st.mesh}, 1))
			}
			if m.st != nil {
				// the kits it was raised with come back
				for l := 1; l <= m.st.tier; l++ {
					drops = append(drops, item.NewStack(TierKit{Level: l}, 1))
				}
			}
			if m.st != nil {
				for _, slots := range [][]int{m.st.kind.input, m.st.kind.output, m.st.kind.charge} {
					for _, s := range slots {
						if it, _ := m.st.inv().Item(s); !it.Empty() {
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
	data := map[string]any{"id": "DragonFactoryMachine"}
	if m.st == nil {
		return data
	}
	var items []any
	for _, slots := range [][]int{m.st.kind.input, m.st.kind.output, m.st.kind.charge} {
		for _, s := range slots {
			if it, _ := m.st.inv().Item(s); !it.Empty() {
				nbt := item.WriteNBT(it, true)
				nbt["Slot"] = byte(s)
				items = append(items, nbt)
			}
		}
	}
	for i, it := range m.st.legacy {
		nbt := item.WriteNBT(it, true)
		nbt["Slot"] = byte(legacySlot + i)
		items = append(items, nbt)
	}
	data["Items"] = items
	data["Progress"] = int32(m.st.progress)
	data["Energy"] = int32(m.st.energy)
	data["Unlimited"] = m.st.unlimited
	data["Burn"] = int32(m.st.burn)
	data["BurnMax"] = int32(m.st.burnMax)
	data["Mesh"] = m.st.mesh
	data["Tier"] = int32(m.st.tier)
	data["Mob"] = m.st.mob
	data["Fluid"] = int32(m.st.fluid)
	data["FluidAmt"] = int32(m.st.fluidAmt)
	data["Charges"] = int32(m.st.charges)
	data["Recipe"] = m.st.recipeID
	data["Owner"] = m.st.owner
	data["OwnerName"] = m.st.ownerName
	return data
}

func (m Machine) DecodeNBT(data map[string]any) any {
	owner, _ := data["Owner"].(string)
	ownerName, _ := data["OwnerName"].(string)
	n := newMachine(m.Kind, owner, ownerName)
	if f, ok := data["Fluid"].(int32); ok {
		n.st.fluid = fluidnet.Kind(f)
	}
	if a, ok := data["FluidAmt"].(int32); ok {
		n.st.fluidAmt = int(a)
	}
	if mob, ok := data["Mob"].(string); ok {
		n.st.mob = mob
	}
	if c, ok := data["Charges"].(int32); ok {
		n.st.charges = int(c)
	}
	if id, ok := data["Recipe"].(string); ok {
		n.st.recipeID = id
		if m.Kind == KindLab {
			n.st.lab = labByKind(id)
		}
	}
	if mesh, ok := data["Mesh"].(string); ok {
		n.st.mesh = mesh
	}
	if t, ok := data["Tier"].(int32); ok && tierable(m.Kind) {
		n.st.tier = max(0, min(int(t), maxTier))
		n.tier = n.st.tier
	}
	if p, ok := data["Progress"].(int32); ok {
		n.st.progress = min(int(p), n.st.kind.interval)
	}
	if v, ok := data["Unlimited"].(bool); ok {
		n.st.unlimited = v
	}
	for key, dst := range map[string]*int{"Energy": &n.st.energy, "Burn": &n.st.burn, "BurnMax": &n.st.burnMax} {
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
			case n.st.role(int(slot)) != roleNone:
				_ = n.st.inv().SetItem(int(slot), it)
			case !it.Empty():
				n.st.legacy = append(n.st.legacy, it)
			}
		}
	}
	if m.Kind == KindLab && n.st.lab == nil {
		// saved before the essence was chosen on the screen: it brews what
		// its ingredients are for
		if r, _ := n.st.labClosest(); r != nil {
			n.st.setLab(r)
		}
	}
	n.st.showProgress()
	return n
}

// Activate opens the machine window, or uses a sieve.
func (m Machine) Activate(pos cube.Pos, _ cube.Face, tx *world.Tx, u item.User, ctx *item.UseContext) bool {
	p, ok := u.(*player.Player)
	s, ok2 := activeGame.sessionOf(tx.World())
	if !ok || !ok2 || !s.canBuild(pos) || m.st == nil {
		return false
	}
	m.st.pos = pos
	if held, off := p.HeldItems(); !held.Empty() {
		if kit, ok := held.Item().(TierKit); ok {
			s.useKit(p, tx, pos, m, kit, held, off)
			return true
		}
	}
	if isSieve(m.Kind) {
		s.useSieve(p, tx, pos, m.st)
		return true
	}
	if m.st.kind.controller {
		held, _ := p.HeldItems()
		if !s.debounced(p) {
			s.useController(p, tx, pos, held)
		}
		return true
	}
	if m.Kind == KindStorageLink {
		return true
	}
	if m.Kind == KindTinker {
		if !s.debounced(p) {
			s.showTinker(p, tx, pos)
		}
		return true
	}
	if m.st.kind.gate {
		if !s.debounced(p) {
			s.showGateForm(p, tx, pos)
		}
		return true
	}
	if m.Kind == KindLab {
		if !s.debounced(p) {
			s.showLab(p, tx, pos, m.st)
		}
		return true
	}
	if m.st.kind.crafter && p.Sneaking() {
		if !s.debounced(p) {
			s.showCrafterRecipes(p, tx, pos, m.st, 0)
		}
		return true
	}
	if m.st.kind.fluidCap > 0 {
		held, _ := p.HeldItems()
		if s.useFluidMachine(p, tx, pos, m, held, ctx) || m.st.kind.marker == "" {
			return true
		}
	}
	if m.st.kind.cell {
		return false
	}
	m.st.openWindow(p)
	return true
}

// InsertItems implements itemio.Inserter: a machine takes inputs through any
// face but the bottom, which is its output. The hand sieve holds one block
// (it needs its mesh), so it takes one only while it is empty, and its look
// follows.
func (m Machine) InsertItems(tx *world.Tx, pos cube.Pos, face cube.Face, st item.Stack) int {
	if m.st != nil && m.st.kind.controller {
		return controllerInsert(tx, pos, st)
	}
	if m.st == nil || len(m.st.kind.input) == 0 || face == cube.FaceDown {
		return 0
	}
	if m.Kind == KindSieve {
		if _, _, filled := m.st.firstInput(); filled || !m.st.accepts(st) || !m.st.insert(st) {
			return 0
		}
		showSieve(tx, pos, m.st)
		return 1
	}
	n := 0
	for n < st.Count() && m.st.insert(st) {
		n++
	}
	return n
}

// PeekItems implements itemio.Extractor: the output leaves through any face
// (a hopper under the machine, a conduit anywhere, since machines no longer
// push their output into their neighbours).
func (m Machine) PeekItems(tx *world.Tx, pos cube.Pos, _ cube.Face, want func(item.Stack) bool) item.Stack {
	if m.st != nil && m.st.kind.controller {
		return controllerPeek(tx, pos, want)
	}
	if m.st == nil {
		return item.Stack{}
	}
	for _, s := range m.st.kind.output {
		if it, _ := m.st.inv().Item(s); !it.Empty() && (want == nil || want(it)) {
			return it
		}
	}
	return item.Stack{}
}

// ExtractItems implements itemio.Extractor.
func (m Machine) ExtractItems(tx *world.Tx, pos cube.Pos, _ cube.Face, st item.Stack, n int) item.Stack {
	if m.st != nil && m.st.kind.controller {
		return controllerExtract(tx, pos, st, n)
	}
	if m.st == nil {
		return item.Stack{}
	}
	got := 0
	var out item.Stack
	for _, s := range m.st.kind.output {
		it, _ := m.st.inv().Item(s)
		if got >= n || it.Empty() || !it.Comparable(st) {
			continue
		}
		take := min(n-got, it.Count())
		_ = m.st.inv().SetItem(s, it.Grow(-take))
		got += take
		out = it.Grow(got - it.Count())
	}
	return out
}

// Tick runs the machine.
func (m Machine) Tick(currentTick int64, pos cube.Pos, tx *world.Tx) {
	s, ok := activeGame.sessionOf(tx.World())
	if !ok {
		return
	}
	if m.st == nil {
		// Placed without going through the place handler.
		tx.SetBlock(pos, newMachine(m.Kind, "", ""), nil)
		return
	}
	st := m.st
	s.initMonitor()
	st.sess, st.pos, st.stuck = s, pos, stuckNone
	st.mon.seen, s.lastTick = currentTick, currentTick
	s.machines[pos] = st
	st.drainLegacy()
	for range st.reps() {
		full := st.kind.interval > 1 && st.progress >= st.kind.interval
		before, at := st.worked, st.progress
		s.runMachine(m.Kind, st, pos, tx)
		if full && st.progress == 0 && cycleOnReset[m.Kind] {
			st.done++
		}
		// a machine that is stuck or did nothing has nothing to repeat
		if st.stuck != stuckNone || (st.worked == before && st.progress == at) {
			break
		}
	}
	st.countWork(tx, m.Kind, currentTick)
	st.pos = pos
	if animatedMachines[m.Kind] {
		showActive(tx, pos, st, currentTick)
	}
	if m.Kind == KindEnergyCell || m.Kind == KindTank || m.Kind == KindCrucible || m.Kind == KindLavaGenerator || m.Kind == KindGate || m.Kind == KindStorageController || m.Kind == KindStorageLink || m.Kind == KindTinker {
		if m.Kind == KindEnergyCell && st.energy > 0 {
			s.energyGrid().AddSource(pos)
		}
		return
	}
	if currentTick%10 == 0 {
		st.showEnergy()
		if (st.stuck != stuckNone) != st.shownStuck {
			_ = st.inv().SetItem(infoSlot, st.infoItem())
		}
	}
	if isSieve(m.Kind) {
		// The sieve shows its content and progress on the model.
		showSieve(tx, pos, st)
		if st.kind.autoSieve {
			st.showProgress()
		}
		return
	}
	st.showProgress()
}

// runMachine runs the work of a machine of a kind once.
func (s *session) runMachine(kind string, st *machineState, pos cube.Pos, tx *world.Tx) {
	switch kind {
	case KindCrusher:
		st.process(func(in item.Stack) []item.Stack {
			out, _ := crusherOutput(in)
			if bonus, ok := crusherBonusOf(in); ok {
				out = out.Grow(bonus.Count() * crusherBonus)
			}
			return []item.Stack{out}
		}, func(in item.Stack) []item.Stack {
			out, _ := crusherOutput(in)
			if bonus, ok := crusherBonusOf(in); ok {
				for range crusherBonus {
					if s.rand.IntN(100) < crusherBonusChance {
						out = out.Grow(bonus.Count())
					}
				}
			}
			return []item.Stack{out}
		})
	case KindGenerator:
		s.runGenerator(st, pos, tx)
	case KindAlloy:
		st.runAlloy()
	case KindSimulator:
		s.runSimulator(st)
	case KindCrucible:
		st.runCrucible()
	case KindWasher:
		st.runWasher()
	case KindLavaGenerator:
		s.runLavaGenerator(st, pos)
	case KindAutoSieve, KindMagicSieve:
		s.runAutoSieve(st, pos, tx)
	case KindMiner:
		s.runMiner(st, pos, tx)
	case KindHarvester:
		s.runHarvester(st, pos, tx)
	case KindFurnace:
		st.runFurnace()
	case KindVacuum:
		s.runVacuum(st, pos, tx)
	case KindFeller:
		s.runFeller(st, pos, tx)
	case KindCrafter:
		s.runCrafter(st)
	case KindLab:
		st.runLab()
	}
}

// process advances an input → output operation on accepted inputs.
// possible returns every product one input item may give; a completed
// operation waits until they all fit. roll, if not nil, picks the actual
// products (otherwise all possible ones are given).
func (st *machineState) process(possible, roll func(in item.Stack) []item.Stack) {
	slot, in, ok := st.firstInput()
	if !ok || !st.accepts(in) {
		st.progress = 0
		return
	}
	if st.progress < st.kind.interval {
		st.tickWork()
		return
	}
	out := possible(in)
	if !st.roomFor(out) {
		return
	}
	if roll != nil {
		out = roll(in)
	}
	_ = st.inv().SetItem(slot, in.Grow(-1))
	for _, it := range out {
		st.emit(it)
	}
	st.progress = 0
}

// mineable reports whether the auto miner may mine b: natural blocks only,
// never machines, cores, drawers or other block entities.
func mineable(b world.Block) (block.BreakInfo, bool) {
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
func (s *session) runMiner(st *machineState, pos cube.Pos, tx *world.Tx) {
	below := pos.Side(cube.FaceDown)
	info, ok := mineable(tx.Block(below))
	if !ok {
		if st.cracking {
			viewCrack(tx, below, block.StopCrackAction{})
			st.cracking = false
		}
		st.progress = 0
		return
	}
	if !st.cracking {
		// Players nearby see the block crack as if a player mined it.
		viewCrack(tx, below, block.StartCrackAction{BreakTime: time.Duration(st.kind.interval-st.progress) * time.Second / 20})
		st.cracking = true
	}
	if st.progress < st.kind.interval {
		st.tickWork()
		return
	}
	var drops []item.Stack
	if info.Drops != nil {
		drops = info.Drops(minerTool, nil)
	}
	if !st.roomFor(drops) {
		return
	}
	viewCrack(tx, below, block.StopCrackAction{})
	st.cracking = false
	mined := tx.Block(below)
	tx.SetBlock(below, nil, nil)
	tx.AddParticle(below.Vec3Centre(), particle.BlockBreak{Block: mined})
	for _, it := range drops {
		st.emit(it)
	}
	st.progress = 0
	st.done++
}

// viewCrack shows a block crack action to the players viewing pos.
func viewCrack(tx *world.Tx, pos cube.Pos, a world.BlockAction) {
	for _, v := range tx.Viewers(pos.Vec3Centre()) {
		v.ViewBlockAction(pos, a)
	}
}

// placeMachine sets up a placed machine with a fresh state.
func (s *session) placeMachine(p *player.Player, tx *world.Tx, pos cube.Pos, kind string) {
	m := newMachine(kind, instance.AccountID(p), p.Name())
	account.Grant(p, achMachine)
	count(p, "machine_place", 1)
	count(p, "place_"+kind, 1)
	if m.Kind != KindSieve {
		s.advance(tx, advMachine)
	}
	tx.Defer(func(tx *world.Tx) { tx.SetBlock(pos, m, nil) })
}

// breakMachine closes the windows of a machine being broken.
func (s *session) breakMachine(tx *world.Tx, pos cube.Pos, m Machine) {
	delete(s.machines, pos)
	if m.st != nil {
		m.st.storage.CloseAll()
	}
}

// machineItem returns a machine item stack.
func machineItem(kind string) item.Stack { return item.NewStack(Machine{Kind: kind}, 1) }

func init() {
	for i := range progressSteps {
		world.RegisterItem(ProgressArrow{Step: i})
		world.RegisterItem(CycleGauge{Step: i})
		world.RegisterItem(EnergyGauge{Step: i})
	}
	world.RegisterItem(MachineInfo{})
	world.RegisterItem(MachineTakeAll{})
	world.RegisterItem(MachinePutAll{})
	for _, k := range machineOrder {
		machineKinds[k].hash = block.NextHash()
		world.RegisterBlock(Machine{Kind: k})
		world.RegisterItem(Machine{Kind: k})
	}
	for _, k := range machineOrder {
		for t := 0; t <= maxTier; t++ {
			for _, on := range []bool{false, true} {
				if (t == 0 && !on) || (t > 0 && !tierLook(k)) || (on && !animatedMachines[k]) {
					continue
				}
				world.RegisterBlock(Machine{Kind: k, active: on, tier: t})
			}
		}
	}
	for _, k := range []string{KindSieve, KindAutoSieve, KindMagicSieve} {
		for _, l := range sieveLooks() {
			world.RegisterBlock(Machine{Kind: k, meshLook: l[0], content: l[1], fill: l[2]})
		}
	}
}

// customBlockEffective makes no tool faster on custom blocks. The client
// takes its break time from the block definition, which only carries the
// bare-hand time; if the server used a faster tool time, the crack would
// finish while the client still mines and the block would never break.
func customBlockEffective(item.Tool) bool { return false }
