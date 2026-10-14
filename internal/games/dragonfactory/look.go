package dragonfactory

import (
	"fmt"
	"github.com/presentkim/dragonfactory/internal/i18n"
	"github.com/presentkim/dragonfactory/internal/itemnet"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/cube/trace"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/internal/instance"
	"github.com/presentkim/dragonfactory/internal/waila"
)

// The WAILA panel (internal/waila, docs/tech/waila.md) shows the block a
// player looks at: its name, icon, tools and the game's own lines (machine
// progress, drawer contents, crop time, what a core makes, the mission);
// drawers and cores also show their item or output block in the slots.

// lookReach is how far a player looks for a block, in blocks.
const lookReach = 6

// lookInterval limits how often moving or turning updates the panel.
const lookInterval = 100 * time.Millisecond

// lookedAt returns the block a player at pos with rotation rot looks at.
// Crops count where the view crosses their height (it grows with the
// stage; the client targets what is behind the empty space above a low
// crop, 11th playtest), other blocks without a collision box (flowers)
// anywhere in their cell; air and liquids do not count.
func lookedAt(p *player.Player, tx *world.Tx, pos mgl64.Vec3, rot cube.Rotation) (cube.Pos, world.Block, bool) {
	eye := pos.Add(mgl64.Vec3{0, p.EyeHeight(), 0})
	end := eye.Add(rot.Vec3().Mul(lookReach))
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
		if _, stage, crop := cropOf(b); crop {
			h := float64(stage+1) / float64(maxCropStage+1)
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

// updateLook updates p's panel for a player at pos with rotation rot (the
// move handler passes where the player is moving to), unless it was
// updated less than lookInterval ago (force: always).
func (s *session) updateLook(p *player.Player, tx *world.Tx, pos mgl64.Vec3, rot cube.Rotation, force bool) {
	if s.lastLook == nil {
		s.lastLook = map[*world.EntityHandle]time.Time{}
	}
	now := time.Now()
	if !force && now.Sub(s.lastLook[p.H()]) < lookInterval {
		return
	}
	s.lastLook[p.H()] = now
	if _, expanding := s.expansionOf(p); expanding {
		waila.Clear(p)
		return
	}
	at, b, ok := lookedAt(p, tx, pos, rot)
	if !ok {
		waila.Clear(p)
		return
	}
	waila.Show(p, s.lookInfo(p, tx, at, b))
}

// Fast updates every player's panel (instance.FastTicker, every 0.2
// seconds: gauges and time left change while a player looks; Show sends only
// what changed).
func (s *session) Fast(tx *world.Tx) {
	// Energy moves every FastInterval (four ticks) over the conduits.
	s.energyGrid().Step(tx, 4)
	// Items move every itemnet.StepTicks ticks (two Fast calls).
	if s.fastCount++; s.fastCount%(itemnet.StepTicks/4) == 0 {
		s.itemGrid().Step(tx)
	}
	s.updateLooks(tx)
}

var _ instance.FastTicker = (*session)(nil)

// updateLooks updates every player's panel.
func (s *session) updateLooks(tx *world.Tx) {
	for e := range tx.Players() {
		p := e.(*player.Player)
		s.updateLook(p, tx, p.Position(), p.Rotation(), true)
	}
}

// lookInfo is the panel of block b at pos.
func (s *session) lookInfo(p *player.Player, tx *world.Tx, pos cube.Pos, b world.Block) waila.Info {
	held, _ := p.HeldItems()
	info := waila.Info{Name: blockName(b), Tools: waila.ToolsFor(b, held)}
	if len(info.Tools) == 0 {
		info.Tools = customTool(b, held)
	}
	if it, ok := b.(world.Item); ok {
		info.Item = it
		info.Name = itemName(item.NewStack(it, 1))
	}
	info.NameKey = blockNameKey(b)
	info.Container = containerOf(tx, pos, b)
	switch b := b.(type) {
	case Machine:
		info.Lines, info.Line = machineLines(b)
		if b.Kind == KindMiner && b.st != nil {
			// The auto miner's one output slot shows like a container's.
			it, _ := b.st.inv().Item(b.st.kind.output[0])
			info.Container = &waila.Container{Layout: waila.LayoutChest, Items: []item.Stack{it}}
		}
	case Drawer:
		// The drawer stays the icon; the held item is a line with its own
		// small icon. Its count, lock and upgrades show on the drawer's
		// front, so the panel leaves them out (11th playtest).
		if b.Item.Empty() {
			info.Lines = []string{i18n.T(i18n.Of(p), "dragonfactory.look.lookInfo.1")}
		} else {
			info.Line = &waila.ItemLine{Item: b.Item, Name: itemName(b.Item)}
		}
	case Core, DeliveryBox:
		info.Lines = []string{i18n.T(i18n.Of(p), "dragonfactory.look.lookInfo.2")}
	case block.Composter:
		info.Lines = s.composterLines(pos, b)
	case ItemConduit:
		info.Lines = itemConduitLines(tx, pos, b)
	case Barrel:
		info.Lines = s.barrelLines(tx, pos)
	case FlowHopper:
		info.Lines = hopperLines(b)
	}
	if kind, _, ok := cropOf(b); ok {
		info.Name = cropNames[kind]
		info.Tools = []waila.ToolIcon{{Tool: waila.Crops, Mark: waila.MarkNone}}
		if r, ok := s.state.Crops[cropKey(pos)]; ok {
			info.Lines = s.cropLines(r, tx, pos)
		}
	}
	return info
}

// customTool is the tool icon of our blocks. No tool mines them faster
// (customBlockEffective), so ToolsFor finds none; the panel still shows the
// tool that fits them, marked when it is held (11th playtest).
func customTool(b world.Block, held item.Stack) []waila.ToolIcon {
	var (
		icon waila.Tool
		typ  item.ToolType
	)
	switch b.(type) {
	case Drawer, DeliveryBox, ResearchTable, Barrel:
		icon, typ = waila.AxeWood, item.TypeAxe
	case FlowHopper:
		if h := b.(FlowHopper); h.kind().wood {
			icon, typ = waila.AxeWood, item.TypeAxe
		} else {
			icon, typ = waila.PickaxeWood, item.TypePickaxe
		}
	case Machine, Core:
		icon, typ = waila.PickaxeWood, item.TypePickaxe
	default:
		return nil
	}
	mark := waila.MarkNone
	if t, ok := held.Item().(item.Tool); ok && t.ToolType() == typ {
		mark = waila.MarkRight
	}
	return []waila.ToolIcon{{Tool: icon, Mark: mark}}
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
	return &waila.Container{Layout: layout, Items: inv.Slots(), Progress: furnaceArrow(b)}
}

// furnaceArrow is the progress arrow of a furnace, blast furnace or smoker:
// filled to how far the item is cooked while it burns, empty when it does
// not (11th playtest).
func furnaceArrow(b world.Block) item.Stack {
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

// machineLines are a machine's lines and item line: what it holds, with
// its icon and progress as a ring gauge ("흙 [icon] ×3 ◔"; 11th playtest),
// grey "비어있음" when empty, and what waits in its output.
func machineLines(m Machine) ([]string, *waila.ItemLine) {
	st := m.st
	if st == nil {
		return nil, nil
	}
	step := st.progress * 10 / st.kind.interval
	gauge := gaugeGlyph(step)
	var (
		lines []string
		line  *waila.ItemLine
	)
	switch {
	case st.kind == machineKinds[KindSellBox]:
		lines = append(lines, i18n.T(i18n.Default, "dragonfactory.look.machineLines.1")+gauge)
	case st.kind == machineKinds[KindHarvester]:
		lines = append(lines, i18n.T(i18n.Default, "dragonfactory.look.machineLines.2")+gauge)
	case st.kind == machineKinds[KindGenerator]:
		return generatorLines(st), nil
	case st.kind == machineKinds[KindMiner]:
		// Its output shows as a slot.
		return append([]string{i18n.T(i18n.Default, "dragonfactory.look.machineLines.3") + gauge}, consumerLines(st)...), nil
	default:
		if _, it, ok := st.firstInput(); ok {
			line = &waila.ItemLine{Item: it, Name: itemName(it), Gauge: step, ShowGauge: true}
			if it.Count() > 1 {
				line.Text = fmt.Sprintf("§f×%d", it.Count())
			}
		} else {
			lines = append(lines, i18n.T(i18n.Default, "dragonfactory.look.machineLines.4"))
		}
	}
	lines = append(lines, consumerLines(st)...)
	for _, slot := range st.kind.output {
		if it, _ := st.inv().Item(slot); !it.Empty() {
			lines = append(lines, i18n.T(i18n.Default, "dragonfactory.look.machineLines.5")+stackLine(it))
		}
	}
	return lines, line
}

// stackLine is an item's name with its count when more than one.
func stackLine(it item.Stack) string {
	if it.Count() > 1 {
		return fmt.Sprintf("§f%s §7×%d", itemName(it), it.Count())
	}
	return "§f" + itemName(it)
}
