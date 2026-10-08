package dragonfactory

import (
	"fmt"
	"image"
	"image/color"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/debug"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/go-gl/mathgl/mgl64"
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
	KindMiner   = "auto_miner"
	KindSellBox = "sell_box"
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
	texture   image.Image
	hash      uint64
}

var machineKinds = map[string]*machineKind{
	KindCrusher: {
		name: "분쇄기", marker: "§d§m§1", interval: 80, input: []int{11}, output: []int{15}, indicator: 13,
		help:    []string{"§7돌 → 조약돌 → 자갈 → 모래", "§7개당 4초"},
		texture: machineTexture(rgb(90, 90, 95), rgb(200, 120, 40)),
	},
	KindSieve: {
		name: "체", marker: "§d§m§2", interval: 60, input: []int{10}, output: []int{14, 15, 16}, indicator: 12,
		help:    []string{"§7자갈·모래·흙을 걸러 부산물을 얻습니다", "§7개당 3초 · 결과는 확률"},
		texture: machineTexture(rgb(120, 85, 50), rgb(230, 220, 170)),
	},
	KindMiner: {
		name: "자동 채굴기", marker: "§d§m§3", interval: 40, output: []int{13, 14, 15, 16}, indicator: 10,
		help:    []string{"§7바로 아래 블록을 2초마다 캡니다", "§7결과는 옆·위의 상자·호퍼·서랍·기계로 보냅니다"},
		texture: machineTexture(rgb(50, 50, 60), rgb(80, 200, 220)),
	},
	KindSellBox: {
		name: "판매함", marker: "§d§m§4", interval: 20, input: []int{9, 10, 11, 12, 13, 14, 15, 16, 17}, indicator: 4,
		help:    []string{"§7들어온 아이템을 상점 매입 규칙대로 팝니다", "§7돈은 설치한 플레이어에게 들어갑니다"},
		texture: machineTexture(rgb(70, 45, 20), rgb(240, 200, 60)),
	},
}

// machineOrder is the registration and display order.
var machineOrder = []string{KindCrusher, KindSieve, KindMiner, KindSellBox}

func machineTexture(frame, core color.RGBA) image.Image {
	return frameTexture(frame, shade(frame, 1.3), core, 8)
}

// crushRecipes maps an input item key to its output.
var crushRecipes = map[string]world.Item{
	itemKey(block.Stone{}):       block.Cobblestone{},
	itemKey(block.Cobblestone{}): block.Gravel{},
	itemKey(block.Gravel{}):      block.Sand{},
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
	itemKey(block.Sand{}):   {{item.NewStack(item.ClayBall{}, 1), 20}, {item.NewStack(item.GoldNugget{}, 1), 3}},
	itemKey(block.Dirt{}):   {{item.NewStack(block.WheatSeeds{}, 1), 15}, {item.NewStack(block.Sapling{Type: block.OakSapling()}, 1), 5}},
}

// minerTool is what the auto miner mines with.
var minerTool = item.Pickaxe{Tier: item.ToolTierStone}

// Machine is a machine block. st is nil for machine items and is created
// when the machine is placed or loaded.
type Machine struct {
	Kind string
	st   *machineState
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
	// shownText is the text shown above the machine.
	shownText string
	// cracking is set while the auto miner shows its target cracking.
	cracking bool
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
	_ = inv.SetItem(pickupSlot, item.NewStack(block.Barrel{}, 1).WithCustomName("§r§c기계 회수").WithLore("§7기계와 안의 아이템을 인벤토리로 가져옵니다."))
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

// pickupSlot is the window slot of the "pick up" button: it removes the
// machine and gives it, with its contents, to the player.
const pickupSlot = 26

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
func (st *machineState) accepts(it item.Stack) bool {
	if it.Empty() {
		return false
	}
	k := itemKey(it.Item())
	switch st.kind {
	case machineKinds[KindCrusher]:
		_, ok := crushRecipes[k]
		return ok
	case machineKinds[KindSieve]:
		_, ok := sieveRecipes[k]
		return ok
	case machineKinds[KindSellBox]:
		_, ok := buybackFor(it.Item())
		return ok
	}
	return false
}

// machineHandler limits moves in the machine window to its input and
// output slots.
type machineHandler struct{ st *machineState }

func (h machineHandler) HandleTake(ctx *inventory.Context, slot int, _ item.Stack) {
	if h.st.role(slot) != roleNone {
		return
	}
	ctx.Cancel()
	if p, ok := ctx.Val().(*player.Player); ok && slot == pickupSlot {
		st := h.st
		p.H().Do(func(tx *world.Tx, e world.Entity) { pickUpMachine(e.(*player.Player), tx, st) })
	}
}

// pickUpMachine removes the machine whose window p used and gives it and
// its contents to p (custom blocks can also be broken normally).
func pickUpMachine(p *player.Player, tx *world.Tx, st *machineState) {
	s, ok := activeGame.sessionOf(tx.World())
	m, isMachine := tx.Block(st.pos).(Machine)
	if !ok || !isMachine || m.st != st || !s.canBuild(st.pos) {
		return
	}
	drops := m.BreakInfo().Drops(nil, nil)
	s.breakMachine(tx, st.pos, m)
	tx.SetBlock(st.pos, nil, nil)
	for _, it := range drops {
		if n, _ := p.Inventory().AddItem(it); n < it.Count() {
			s.dropAt(tx, p.Position(), it.Grow(n-it.Count()))
		}
	}
	p.Messagef("§a%s을(를) 회수했습니다.", m.kind().name)
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
	lines := append([]string{}, st.kind.help...)
	name := fmt.Sprintf("§e진행 %d%%", pct)
	it := item.NewStack(ProgressArrow{Step: pct / 10}, 1)
	switch st.kind {
	case machineKinds[KindSellBox]:
		name, it = "§6판매함", item.NewStack(item.GoldIngot{}, 1)
		if st.ownerName != "" {
			lines = append(lines, "§7주인: §f"+st.ownerName)
		}
	case machineKinds[KindMiner]:
		name = fmt.Sprintf("§b채굴 %d%%", pct)
	}
	_ = st.inv().SetItem(st.kind.indicator, it.WithCustomName("§r"+name).WithLore(lines...))
}

// ProgressArrow is the indicator item of machine windows: an arrow filled
// in Step tenths. It only ever sits in machine windows.
type ProgressArrow struct{ Step int }

var progressTextures = func() [11]image.Image {
	var out [11]image.Image
	for i := range out {
		out[i] = progressArrowTexture(i)
	}
	return out
}()

func (a ProgressArrow) EncodeItem() (string, int16) {
	return fmt.Sprintf("dragonfactory:progress_%d", a.Step), 0
}
func (a ProgressArrow) Name() string              { return "진행도" }
func (a ProgressArrow) Texture() image.Image      { return progressTextures[a.Step] }
func (ProgressArrow) Category() category.Category { return category.Items() }

func (m Machine) kind() *machineKind { return machineKinds[m.Kind] }

func (m Machine) EncodeItem() (string, int16)           { return "dragonfactory:" + m.Kind, 0 }
func (m Machine) EncodeBlock() (string, map[string]any) { return "dragonfactory:" + m.Kind, nil }
func (m Machine) Hash() (uint64, uint64)                { return m.kind().hash, 0 }
func (Machine) Model() world.BlockModel                 { return model.Solid{} }
func (m Machine) Name() string                          { return m.kind().name }
func (Machine) Category() category.Category             { return category.Items() }
func (Machine) Geometry() []byte                        { return nil }
func (m Machine) Texture() image.Image                  { return m.kind().texture }
func (m Machine) Textures() map[string]image.Image {
	return map[string]image.Image{"dragonfactory_" + m.Kind: m.kind().texture}
}
func (m Machine) Properties() customblock.Properties {
	return customblock.Properties{Cube: true, Textures: map[string]customblock.Material{
		"*": customblock.NewMaterial("dragonfactory_"+m.Kind, customblock.OpaqueRenderMethod()),
	}}
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
			if n.st.role(int(slot)) != roleNone {
				_ = n.st.inv().SetItem(int(slot), item.ReadNBT(nbt, nil))
			}
		}
	}
	n.st.showProgress()
	return n
}

// Activate opens the machine window.
func (m Machine) Activate(pos cube.Pos, _ cube.Face, tx *world.Tx, u item.User, _ *item.UseContext) bool {
	p, ok := u.(*player.Player)
	s, ok2 := activeGame.sessionOf(tx.World())
	if !ok || !ok2 || !s.canBuild(pos) || m.st == nil {
		return false
	}
	m.st.pos = pos
	m.st.storage.Open(p, nil)
	return true
}

// InsertItem implements block.HopperInsertable: one item per transfer.
func (m Machine) InsertItem(h block.Hopper, pos cube.Pos, tx *world.Tx) bool {
	if m.st == nil || len(m.st.kind.input) == 0 {
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
	switch m.Kind {
	case KindCrusher:
		st.process(func(in item.Stack) []item.Stack {
			return []item.Stack{item.NewStack(crushRecipes[itemKey(in.Item())], 1)}
		}, nil)
	case KindSieve:
		drops := func(in item.Stack) []SieveDrop { return sieveRecipes[itemKey(in.Item())] }
		st.process(func(in item.Stack) []item.Stack {
			var all []item.Stack
			for _, d := range drops(in) {
				all = append(all, d.Item)
			}
			return all
		}, func(in item.Stack) []item.Stack {
			var out []item.Stack
			for _, d := range drops(in) {
				if s.rand.IntN(100) < d.Chance {
					out = append(out, d.Item)
				}
			}
			return out
		})
	case KindMiner:
		s.runMiner(st, pos, tx)
		if currentTick%4 == 0 {
			s.pushOutput(st, pos, tx)
		}
	case KindSellBox:
		s.runSellBox(st, tx)
	}
	st.pos = pos
	st.showProgress()
	s.showMachineText(tx, st)
}

// machineTextKey keys the text shown above a machine.
type machineTextKey cube.Pos

// text returns the text shown above the machine: its name and progress.
func (st *machineState) text() string {
	switch {
	case st.kind == machineKinds[KindSellBox]:
		return "§6" + st.kind.name
	case st.progress == 0:
		return "§f" + st.kind.name + " §7대기"
	}
	return fmt.Sprintf("§f%s §e%d%%", st.kind.name, st.shownPct)
}

// showMachineText shows the machine's text above it when it changed.
func (s *session) showMachineText(tx *world.Tx, st *machineState) {
	text := st.text()
	if text == st.shownText {
		return
	}
	st.shownText = text
	s.showShape(tx, machineTextKey(st.pos), &debug.Text{
		Position: st.pos.Vec3Centre().Add(mgl64.Vec3{0, 0.95, 0}),
		Text:     text,
		Scale:    0.6,
	})
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
		if b.insert(one, 1) == 1 {
			s.setDrawer(tx, pos, b)
			return true
		}
		return false
	case DeliveryBox:
		return s.deliver(tx, nil, one.Item(), 1) == 1
	case block.Container:
		_, err := b.Inventory(tx, pos).AddItem(one)
		return err == nil
	}
	return false
}

// runSellBox sells the input every interval for the owner.
func (s *session) runSellBox(st *machineState, tx *world.Tx) {
	if st.progress < st.kind.interval {
		st.progress++
		return
	}
	st.progress = 0
	owner := st.owner
	if owner == "" {
		owner = s.hostID
	}
	for _, slot := range st.kind.input {
		it, _ := st.inv().Item(slot)
		if it.Empty() {
			continue
		}
		if n, earned := s.sellFor(owner, it); n > 0 {
			_ = st.inv().SetItem(slot, it.Grow(-n))
			s.addMoneyTo(tx, owner, earned)
		}
	}
}

// addMoneyTo adds to the wallet of an account, online or not.
func (s *session) addMoneyTo(tx *world.Tx, acc string, delta int64) {
	if _, ok := s.money[acc]; ok {
		s.money[acc] += delta
		for e := range tx.Players() {
			if p := e.(*player.Player); instance.AccountID(p) == acc {
				s.updateBoard(p)
			}
		}
		return
	}
	if err := s.g.db.AddMoney(s.saveID, acc, delta); err != nil {
		s.g.log.Error("add money", "save", s.saveID, "account", acc, "err", err)
	}
}

// placeMachine sets up a placed machine with a fresh state.
func (s *session) placeMachine(p *player.Player, tx *world.Tx, pos cube.Pos, kind string) {
	m := newMachine(kind, instance.AccountID(p), p.Name())
	tx.Defer(func(tx *world.Tx) { tx.SetBlock(pos, m, nil) })
}

// breakMachine closes the windows of a machine being broken and removes
// its text.
func (s *session) breakMachine(tx *world.Tx, pos cube.Pos, m Machine) {
	s.hideShape(tx, machineTextKey(pos))
	if m.st != nil {
		m.st.storage.CloseAll()
	}
}

// machineItem returns a machine item stack.
func machineItem(kind string) item.Stack { return item.NewStack(Machine{Kind: kind}, 1) }

func init() {
	for i := range progressTextures {
		world.RegisterItem(ProgressArrow{Step: i})
	}
	for _, k := range machineOrder {
		machineKinds[k].hash = block.NextHash()
		world.RegisterBlock(Machine{Kind: k})
		world.RegisterItem(Machine{Kind: k})
	}
}

// customBlockEffective makes no tool faster on custom blocks. The client
// takes its break time from the block definition, which only carries the
// bare-hand time; if the server used a faster tool time, the crack would
// finish while the client still mines and the block would never break.
func customBlockEffective(item.Tool) bool { return false }
