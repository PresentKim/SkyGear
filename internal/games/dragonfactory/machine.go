package dragonfactory

import (
	"fmt"
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
	"github.com/presentkim/dragonfactory/internal/account"
	"github.com/presentkim/dragonfactory/internal/chestui"
	"github.com/presentkim/dragonfactory/internal/instance"
)

// Machines are the automation blocks of the first era
// (docs/design/02-dragonfactory.md §6). They need no power: each operation
// takes a fixed time. Items go in and out by hand in the machine window
// (an entity chest) or by hoppers: a hopper above or beside a machine
// inserts into its input, a hopper below extracts from its output.

// Machine kinds.
const (
	KindCrusher = "crusher"
	KindSieve   = "sieve"
	// KindAutoSieve sifts on its own; KindSieve only by hand (sieve.go).
	KindAutoSieve = "auto_sieve"
	KindMiner     = "auto_miner"
	KindSellBox   = "sell_box"
	// KindHarvester harvests grown crops around it (harvester.go).
	KindHarvester = "harvester"
)

// machineKind describes a kind of machine. Slots are window slots of a 27
// slot (3 row) window; the window inventory is the machine inventory.
type machineKind struct {
	name string
	// marker in the window title selects the machine's layout in the
	// resource pack (ui/df_machine.json via inventoryui's chest_screen.json).
	marker string
	// interval is the ticks one operation takes. Balancing drafts.
	interval  int
	input     []int
	output    []int
	indicator int
	help      []string
	hash      uint64
}

var machineKinds = map[string]*machineKind{
	KindCrusher: {
		name: "분쇄기", marker: "§d§m§1", interval: 80, input: []int{11}, output: []int{15}, indicator: 13,
		help: []string{"§7돌 → 조약돌 → 자갈 → 모래", "§7개당 4초"},
	},
	// Sieves have no window: the input lies on the mesh and the products
	// drop (or, from the auto sieve, go into containers beside or below).
	KindSieve: {
		name: "체", interval: 60, input: []int{10}, indicator: 12,
		help: []string{"§7자갈·모래·흙을 얹고 여러 번 사용해 거릅니다", "§7부산물은 위로 튀어나옵니다"},
	},
	KindAutoSieve: {
		name: "자동 체", interval: 60, input: []int{10}, indicator: 12,
		help: []string{"§7자갈·모래·흙을 저절로 거릅니다(개당 3초)", "§7부산물은 옆·아래 상자로, 없으면 위로"},
	},
	KindMiner: {
		// One output slot (11th playtest; it had four, like the harvester).
		name: "자동 자원 채굴기", marker: "§d§m§5", interval: 40, output: []int{13}, indicator: 10,
		help: []string{"§7바로 아래 블록을 2초마다 캡니다", "§7결과는 옆·위의 상자·호퍼·서랍·기계로 보냅니다"},
	},
	KindHarvester: {
		name: "자동 농작물 수확기", marker: "§d§m§3", interval: 40, output: []int{13, 14, 15, 16}, indicator: 10,
		help: []string{"§7같은 높이 9×9 안의 다 자란 작물을 2초마다 하나씩 거둡니다", "§7결과는 옆·위의 상자·호퍼·서랍·기계로 보냅니다"},
	},
	KindSellBox: {
		name: "판매함", marker: "§d§m§4", interval: 20, input: []int{9, 10, 11, 12, 13, 14, 15, 16, 17}, indicator: 4,
		help: []string{"§7더 이상 쓰이지 않습니다(상점이 없어졌습니다)", "§7안에 남은 아이템은 꺼낼 수 있습니다"},
	},
}

// machineOrder is the registration and display order.
var machineOrder = []string{KindCrusher, KindSieve, KindAutoSieve, KindMiner, KindSellBox, KindHarvester}

// crushRecipes maps an input item key to its output.
var crushRecipes = map[string]world.Item{
	itemKey(block.Stone{}):       block.Cobblestone{},
	itemKey(block.Cobblestone{}): block.Gravel{},
	itemKey(block.Gravel{}):      block.Sand{},
	// Sand crushes into dust (13th playtest, sky progression).
	itemKey(block.Sand{}): Dust{},
}

// SieveDrop is a possible product of sifting an item.
type SieveDrop struct {
	Item item.Stack
	// Chance is the percent chance per sifted item.
	Chance int
}

// sieveRecipes maps an input item key to its drops. Balancing drafts.
var sieveRecipes = map[string][]SieveDrop{
	itemKey(block.Gravel{}): {{item.NewStack(item.Flint{}, 1), 25}, {item.NewStack(item.IronNugget{}, 1), 8}, {item.NewStack(item.Coal{}, 1), 5}},
	itemKey(block.Sand{}):   {{item.NewStack(item.ClayBall{}, 1), 20}, {item.NewStack(item.GoldNugget{}, 1), 3}, {item.NewStack(block.Kelp{}, 1), 3}},
	itemKey(block.Dirt{}): {{item.NewStack(block.WheatSeeds{}, 1), 15}, {item.NewStack(block.Sapling{Type: block.OakSapling()}, 1), 5},
		{item.NewStack(block.PumpkinSeeds{}, 1), 4}, {item.NewStack(block.MelonSeeds{}, 1), 4}},
	itemKey(Dust{}): {{item.NewStack(item.BoneMeal{}, 1), 10}, {item.NewStack(item.RedstoneWire{}, 1), 5}, {item.NewStack(item.Gunpowder{}, 1), 3}},
}

// minerTool is what the auto miner mines with.
var minerTool = item.Pickaxe{Tier: item.ToolTierStone}

// Machine is a machine block. st is nil for machine items and is created
// when the machine is placed or loaded.
type Machine struct {
	Kind string
	st   *machineState
	// content and fill are what a sieve shows on its mesh (sieve.go).
	content, fill int
}

// machineState is the inventory and progress of a placed machine. It is
// owned by the machine's world.
type machineState struct {
	kind     *machineKind
	storage  *chestui.Storage
	progress int
	shownPct int
	// owner is the account that placed a sell box and gets its money.
	owner, ownerName string
	// pos is where the machine stands, set when it ticks or its window is
	// opened.
	pos cube.Pos
	// cracking is set while the auto miner shows its target cracking.
	cracking bool
	// lastUse is when a sieve was last sifted by hand.
	lastUse time.Time
	// legacy are saved items of slots the kind no longer has (the auto
	// miner had four output slots); they move into the output as it frees
	// up, and are saved until then.
	legacy []item.Stack
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
	st := &machineState{kind: k, storage: chestui.NewStorage(27, k.marker+k.name), owner: owner, ownerName: ownerName, shownPct: -1}
	inv := st.storage.Inventory()
	for slot := range 27 {
		if st.role(slot) == roleNone {
			_ = inv.SetItem(slot, filler())
		}
	}
	_ = inv.SetItem(infoSlot, st.infoItem())
	inv.Handle(machineHandler{st: st})
	st.showProgress()
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
)

// infoSlot is the window slot of the info icon: its tooltip explains the
// machine. The theme draws it left of the close button.
const infoSlot = 26

// infoItem is the info icon of a machine window: the machine's name, what
// it does and, for the sell box, whose it is.
func (st *machineState) infoItem() item.Stack {
	lines := append([]string{}, st.kind.help...)
	if st.kind == machineKinds[KindSellBox] && st.ownerName != "" {
		lines = append(lines, "§7주인: §f"+st.ownerName)
	}
	return item.NewStack(MachineInfo{}, 1).WithCustomName("§r§f" + st.kind.name).WithLore(lines...)
}

// MachineInfo is the info icon item of machine windows. It only ever sits
// in machine windows.
type MachineInfo struct{}

func (MachineInfo) EncodeItem() (string, int16) { return "dragonfactory:machine_info", 0 }
func (MachineInfo) Name() string                { return "기계 정보" }
func (MachineInfo) Category() category.Category { return category.Items() }

func (st *machineState) role(slot int) slotRole {
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
		_, ok := crushRecipes[k]
		return ok
	case machineKinds[KindSieve], machineKinds[KindAutoSieve]:
		_, ok := sieveRecipes[k]
		return ok
	}
	// The sell box (no more shop, 13th playtest) takes nothing.
	return false
}

// machineHandler limits moves in the machine window to its input and
// output slots.
type machineHandler struct{ st *machineState }

func (h machineHandler) HandleTake(ctx *inventory.Context, slot int, _ item.Stack) {
	if h.st.role(slot) == roleNone {
		ctx.Cancel()
	}
}

func (h machineHandler) HandlePlace(ctx *inventory.Context, slot int, it item.Stack) {
	if h.st.role(slot) != roleInput || !h.st.accepts(it) {
		ctx.Cancel()
	}
}

func (h machineHandler) HandleDrop(ctx *inventory.Context, slot int, _ item.Stack) {
	if h.st.role(slot) == roleNone {
		ctx.Cancel()
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
	if !st.accepts(one) || !fits(st.inv(), st.kind.input, []item.Stack{one}) {
		return false
	}
	put(st.inv(), st.kind.input, one)
	return true
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
	pct := st.progress * 100 / st.kind.interval
	pct -= pct % 10
	if pct == st.shownPct {
		return
	}
	st.shownPct = pct
	name := fmt.Sprintf("§e진행 %d%%", pct)
	it := item.NewStack(ProgressArrow{Step: pct / 10}, 1)
	switch st.kind {
	case machineKinds[KindSellBox]:
		// Machines that work in a repeating cycle show a ring that fills.
		name, it = "§6판매 주기", item.NewStack(CycleGauge{Step: pct / 10}, 1)
	case machineKinds[KindMiner]:
		name = fmt.Sprintf("§b채굴 %d%%", pct)
	case machineKinds[KindHarvester]:
		name, it = "§a수확 주기", item.NewStack(CycleGauge{Step: pct / 10}, 1)
	}
	_ = st.inv().SetItem(st.kind.indicator, it.WithCustomName("§r"+name))
}

// ProgressArrow is the indicator item of machine windows: an arrow filled
// in Step tenths. It only ever sits in machine windows.
type ProgressArrow struct{ Step int }

// progressSteps is the number of ProgressArrow steps (0 to 10).
const progressSteps = 11

func (a ProgressArrow) EncodeItem() (string, int16) {
	return fmt.Sprintf("dragonfactory:progress_%d", a.Step), 0
}
func (a ProgressArrow) Name() string              { return "진행도" }
func (ProgressArrow) Category() category.Category { return category.Items() }

// CycleGauge is the indicator of machines that work in a repeating cycle
// (auto harvester, sell box): a hollow ring filled in Step tenths, which
// turns over and over like a loading circle.
type CycleGauge struct{ Step int }

func (g CycleGauge) EncodeItem() (string, int16) {
	return fmt.Sprintf("dragonfactory:cycle_%d", g.Step), 0
}
func (g CycleGauge) Name() string              { return "주기" }
func (CycleGauge) Category() category.Category { return category.Items() }

func (m Machine) kind() *machineKind { return machineKinds[m.Kind] }

func (m Machine) EncodeItem() (string, int16) { return "dragonfactory:" + m.Kind, 0 }
func (m Machine) EncodeBlock() (string, map[string]any) {
	if isSieve(m.Kind) {
		return "dragonfactory:" + m.Kind, map[string]any{sieveLookState: lookValue(m.content, m.fill)}
	}
	return "dragonfactory:" + m.Kind, nil
}
func (m Machine) Hash() (uint64, uint64) {
	return m.kind().hash, uint64(lookIndex(m.content, m.fill))
}
func (Machine) Model() world.BlockModel     { return model.Solid{} }
func (m Machine) Name() string              { return m.kind().name }
func (Machine) Category() category.Category { return category.Items() }

// cubeMachines are the machines drawn as plain cubes with one side texture;
// frontMachines have a front (north), back and side textures. The sieves
// have models.
var (
	cubeMachines  = map[string]bool{KindMiner: true, KindHarvester: true}
	frontMachines = map[string]bool{KindCrusher: true, KindSellBox: true}
)

// States implements block.Permutable: only the sieve has states.
func (m Machine) States() map[string][]any {
	if isSieve(m.Kind) {
		return sieveStates()
	}
	return nil
}

// Permutations implements block.Permutable.
func (m Machine) Permutations() []customblock.Permutation {
	if isSieve(m.Kind) {
		return sievePermutations(m.Kind)
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
				for _, slots := range [][]int{m.st.kind.input, m.st.kind.output} {
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
	for _, slots := range [][]int{m.st.kind.input, m.st.kind.output} {
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
	data["Owner"] = m.st.owner
	data["OwnerName"] = m.st.ownerName
	return data
}

func (m Machine) DecodeNBT(data map[string]any) any {
	owner, _ := data["Owner"].(string)
	ownerName, _ := data["OwnerName"].(string)
	n := newMachine(m.Kind, owner, ownerName)
	if p, ok := data["Progress"].(int32); ok {
		n.st.progress = min(int(p), n.st.kind.interval)
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
	n.st.showProgress()
	return n
}

// Activate opens the machine window, or uses a sieve.
func (m Machine) Activate(pos cube.Pos, _ cube.Face, tx *world.Tx, u item.User, _ *item.UseContext) bool {
	p, ok := u.(*player.Player)
	s, ok2 := activeGame.sessionOf(tx.World())
	if !ok || !ok2 || !s.canBuild(pos) || m.st == nil {
		return false
	}
	m.st.pos = pos
	if isSieve(m.Kind) {
		s.useSieve(p, tx, pos, m.st)
		return true
	}
	m.st.storage.Open(p, nil)
	return true
}

// InsertItem implements block.HopperInsertable: one item per transfer.
func (m Machine) InsertItem(h block.Hopper, pos cube.Pos, tx *world.Tx) bool {
	if m.st == nil || len(m.st.kind.input) == 0 || m.Kind == KindSieve {
		// The hand sieve is filled by hand only.
		return false
	}
	inv := h.Inventory(tx, pos.Side(h.Facing.Opposite()))
	for slot, it := range inv.Slots() {
		if !it.Empty() && m.st.insert(it) {
			_ = inv.SetItem(slot, it.Grow(-1))
			return true
		}
	}
	return false
}

// ExtractItem implements block.HopperExtractable (hopper below).
func (m Machine) ExtractItem(h block.Hopper, pos cube.Pos, tx *world.Tx) bool {
	if m.st == nil {
		return false
	}
	for _, s := range m.st.kind.output {
		it, _ := m.st.inv().Item(s)
		if it.Empty() {
			continue
		}
		if _, err := h.Inventory(tx, pos.Side(cube.FaceDown)).AddItem(it.Grow(1 - it.Count())); err != nil {
			return false
		}
		_ = m.st.inv().SetItem(s, it.Grow(-1))
		return true
	}
	return false
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
	st.drainLegacy()
	switch m.Kind {
	case KindCrusher:
		st.process(func(in item.Stack) []item.Stack {
			return []item.Stack{item.NewStack(crushRecipes[itemKey(in.Item())], 1)}
		}, nil)
	case KindAutoSieve:
		s.runAutoSieve(st, pos, tx)
	case KindMiner:
		s.runMiner(st, pos, tx)
		if currentTick%4 == 0 {
			s.pushOutput(st, pos, tx)
		}
	case KindHarvester:
		s.runHarvester(st, pos, tx)
		if currentTick%4 == 0 {
			s.pushOutput(st, pos, tx)
		}
	}
	st.pos = pos
	if isSieve(m.Kind) {
		// The sieve shows its content and progress on the model.
		showSieve(tx, pos, st)
		return
	}
	st.showProgress()
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
		st.progress++
		return
	}
	out := possible(in)
	if !fits(st.inv(), st.kind.output, out) {
		return
	}
	if roll != nil {
		out = roll(in)
	}
	_ = st.inv().SetItem(slot, in.Grow(-1))
	for _, it := range out {
		put(st.inv(), st.kind.output, it)
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
	switch b.(type) {
	case Core, DeliveryBox:
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
		st.progress++
		return
	}
	var drops []item.Stack
	if info.Drops != nil {
		drops = info.Drops(minerTool, nil)
	}
	if !fits(st.inv(), st.kind.output, drops) {
		return
	}
	viewCrack(tx, below, block.StopCrackAction{})
	st.cracking = false
	mined := tx.Block(below)
	tx.SetBlock(below, nil, nil)
	tx.AddParticle(below.Vec3Centre(), particle.BlockBreak{Block: mined})
	for _, it := range drops {
		put(st.inv(), st.kind.output, it)
	}
	st.progress = 0
}

// viewCrack shows a block crack action to the players viewing pos.
func viewCrack(tx *world.Tx, pos cube.Pos, a world.BlockAction) {
	for _, v := range tx.Viewers(pos.Vec3Centre()) {
		v.ViewBlockAction(pos, a)
	}
}

// pushTargets are the sides the auto miner pushes into, in turn.
var pushTargets = []cube.Face{cube.FaceUp, cube.FaceNorth, cube.FaceEast, cube.FaceSouth, cube.FaceWest}

// pushOutput moves one output item into a neighbouring block that takes
// it. An item no neighbour takes does not hold up the others, so mixed
// output can be split, e.g. a crusher on one side and a sell box on another.
func (s *session) pushOutput(st *machineState, pos cube.Pos, tx *world.Tx) {
	for _, slot := range st.kind.output {
		it, _ := st.inv().Item(slot)
		if it.Empty() {
			continue
		}
		for _, f := range pushTargets {
			if s.insertInto(tx, pos.Side(f), it.Grow(1-it.Count())) {
				_ = st.inv().SetItem(slot, it.Grow(-1))
				return
			}
		}
	}
}

// insertInto puts one item into the block at pos if it takes it.
func (s *session) insertInto(tx *world.Tx, pos cube.Pos, one item.Stack) bool {
	switch b := tx.Block(pos).(type) {
	case Machine:
		return b.st != nil && b.st.insert(one)
	case Drawer:
		if b.takesAutomated(one) && b.insert(one, 1) == 1 {
			s.setDrawer(tx, pos, b)
			return true
		}
		return false
	case block.Container:
		_, err := b.Inventory(tx, pos).AddItem(one)
		return err == nil
	}
	return false
}

// placeMachine sets up a placed machine with a fresh state.
func (s *session) placeMachine(p *player.Player, tx *world.Tx, pos cube.Pos, kind string) {
	m := newMachine(kind, instance.AccountID(p), p.Name())
	account.Grant(p, achMachine)
	tx.Defer(func(tx *world.Tx) { tx.SetBlock(pos, m, nil) })
}

// breakMachine closes the windows of a machine being broken.
func (s *session) breakMachine(tx *world.Tx, pos cube.Pos, m Machine) {
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
	}
	world.RegisterItem(MachineInfo{})
	for _, k := range machineOrder {
		machineKinds[k].hash = block.NextHash()
		world.RegisterBlock(Machine{Kind: k})
		world.RegisterItem(Machine{Kind: k})
	}
	for _, k := range []string{KindSieve, KindAutoSieve} {
		for c := range sieveContents {
			for fill := range sieveStages + 1 {
				if c != 0 || fill != 0 {
					world.RegisterBlock(Machine{Kind: k, content: c, fill: fill})
				}
			}
		}
	}
}

// customBlockEffective makes no tool faster on custom blocks. The client
// takes its break time from the block definition, which only carries the
// bare-hand time; if the server used a faster tool time, the crack would
// finish while the client still mines and the block would never break.
func customBlockEffective(item.Tool) bool { return false }
