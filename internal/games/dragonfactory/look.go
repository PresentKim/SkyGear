package dragonfactory

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/internal/fluidnet"
	"github.com/presentkim/dragonfactory/internal/itemnet"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/cube/trace"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/common/waila"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/dfitem"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/hopper"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/machinerecipe"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/naming"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/textfmt"
	"github.com/presentkim/dragonfactory/internal/instance"
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
	s.showLook(p, tx, at, b, ok)
}

// showLook shows p the panel of block b at pos it looks at (ok), or clears
// it.
func (s *session) showLook(p *player.Player, tx *world.Tx, at cube.Pos, b world.Block, ok bool) {
	if !ok {
		waila.Clear(p)
		return
	}
	waila.Show(p, s.lookInfo(p, tx, at, b))
}

// updateLookAt is updateLook with force for block b at pos p looks at (ok).
func (s *session) updateLookAt(p *player.Player, tx *world.Tx, at cube.Pos, b world.Block, ok bool) {
	if s.lastLook == nil {
		s.lastLook = map[*world.EntityHandle]time.Time{}
	}
	s.lastLook[p.H()] = time.Now()
	if _, expanding := s.expansionOf(p); expanding {
		waila.Clear(p)
		return
	}
	s.showLook(p, tx, at, b, ok)
}

// Fast updates every player's panel (instance.FastTicker, every 0.2
// seconds: gauges and time left change while a player looks; Show sends only
// what changed).
func (s *session) Fast(tx *world.Tx) {
	// Energy moves every FastInterval (four ticks) over the conduits.
	s.energyGrid().Step(tx, 4)
	// Items move every itemnet.StepTicks ticks and fluids every
	// fluidnet.StepTicks ticks (a Fast call is four ticks).
	s.fastCount++
	if s.fastCount%(itemnet.StepTicks/4) == 0 {
		s.itemGrid().Step(tx)
	}
	if s.fastCount%(fluidnet.StepTicks/4) == 0 {
		s.fluidGrid().Step(tx)
	}
	s.emergeMonsters(tx)
	s.updateLooks(tx)
}

var _ instance.FastTicker = (*session)(nil)

// updateLooks updates every player's panel.
func (s *session) updateLooks(tx *world.Tx) {
	for e := range tx.Players() {
		p := e.(*player.Player)
		at, b, ok := lookedAt(p, tx, p.Position(), p.Rotation())
		s.updateLookAt(p, tx, at, b, ok)
		if ok {
			s.showVeinOutline(p, tx, at)
		} else {
			s.hideVeinOutline(p)
		}
	}
}

// lookInfo is the panel of block b at pos.
func (s *session) lookInfo(p *player.Player, tx *world.Tx, pos cube.Pos, b world.Block) waila.Info {
	lang := i18n.Of(p)
	held, _ := p.HeldItems()
	info := waila.Info{Name: naming.BlockName(lang, b), Tools: waila.ToolsFor(b, held)}
	if len(info.Tools) == 0 {
		info.Tools = customTool(b, held)
	}
	if it, ok := b.(world.Item); ok {
		info.Item = it
		info.Name = naming.ItemName(lang, item.NewStack(it, 1))
	}
	info.NameKey = naming.BlockNameKey(b)
	info.Container = containerOf(tx, pos, b)
	switch b := b.(type) {
	case Machine:
		info.Lines, info.Line = machineLines(lang, b)
		if b.st != nil && b.st.kind.controller {
			info.Lines = storageLines(lang, tx, pos)
		}
		if b.Kind == KindMiner && b.st != nil {
			// The auto miner's one output slot shows like a container's.
			it, _ := b.st.inv().Item(b.st.kind.output[0])
			info.Container = &waila.Container{Layout: waila.LayoutChest, Items: []item.Stack{it}}
		}
		if p.Sneaking() && b.st != nil {
			info.Lines = append(info.Lines, detailLines(lang, b.st)...)
		}
	case Drawer:
		// The drawer stays the icon; the held item shows like a chest's, a
		// slot with its count under it; the lock and upgrades show on the
		// drawer's front and, while sneaking, in the panel.
		if b.Item.Empty() {
			info.Lines = []string{i18n.T(lang, "look.empty")}
		} else {
			info.Container = &waila.Container{Layout: waila.LayoutChest, Items: []item.Stack{b.Item.Grow(int(min(b.Count, math.MaxInt32)) - b.Item.Count())}}
		}
		if p.Sneaking() {
			info.Lines = append(info.Lines, drawerDetail(lang, b)...)
		}
	case block.Composter:
		info.Lines = s.composterLines(lang, pos, b)
	case ConduitBundle:
		info.Lines = bundleLines(lang, tx, pos, b)
	case Barrel:
		info.Lines = s.barrelLines(lang, tx, pos)
		info.Line = s.barrelLiquid(lang, pos)
	case hopper.FlowHopper:
		info.Lines = hopper.HopperLines(lang, b)
	case MagicPlant:
		if i := plantKindIndex(b.Kind); i >= 0 {
			k := plantKinds[i]
			seed := k.seedStack()
			info.Item, info.NameKey = seed.Item(), ""
			info.Name = i18n.T(lang, "pseed."+k.seed)
			info.Tools = []waila.ToolIcon{{Tool: waila.Crops, Mark: waila.MarkNone}}
			info.Lines = s.plantLines(lang, pos, b, k)
		}
	}
	if kind, _, ok := cropOf(b); ok {
		info.Name = i18n.T(lang, cropNameKeys[kind])
		info.Tools = []waila.ToolIcon{{Tool: waila.Crops, Mark: waila.MarkNone}}
		if r, ok := s.state.Crops[cropKey(pos)]; ok {
			info.Lines = s.cropLines(lang, r, tx, pos)
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
	case Drawer, ResearchTable, Barrel:
		icon, typ = waila.AxeWood, item.TypeAxe
	case hopper.FlowHopper:
		if h := b.(hopper.FlowHopper); h.Info().Wood {
			icon, typ = waila.AxeWood, item.TypeAxe
		} else {
			icon, typ = waila.PickaxeWood, item.TypePickaxe
		}
	case Machine:
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
func machineLines(lang string, m Machine) ([]string, *waila.ItemLine) {
	st := m.st
	if st == nil {
		return nil, nil
	}
	step := min(10, st.progress*10/st.cycleLen())
	gauge := textfmt.GaugeGlyph(step)
	var (
		lines []string
		line  *waila.ItemLine
	)
	switch {
	case st.kind.crafter:
		return crafterLines(lang, st), nil
	case st.kind == machineKinds[KindLab]:
		return labLines(lang, st), nil
	case st.kind == machineKinds[KindSellBox]:
		lines = append(lines, i18n.T(lang, "look.machine.sellCycle")+gauge)
	case st.kind == machineKinds[KindHarvester]:
		lines = append(lines, i18n.T(lang, "look.machine.harvestCycle")+gauge)
	case st.kind == machineKinds[KindGenerator]:
		return generatorLines(lang, st), nil
	case st.kind.gate, m.Kind == KindStorageLink, m.Kind == KindTinker:
		return nil, nil
	case st.kind.controller:
		return nil, nil
	case st.kind.fluidCap > 0:
		lines := fluidLines(lang, st)
		if _, in, ok := st.firstInput(); ok {
			lines = append(lines, stackLine(lang, in))
		}
		if st.kind.generates > 0 {
			lines = append(lines, fmt.Sprintf(i18n.T(lang, "generator.lines.energy"), st.energy, st.energyCap()))
		} else {
			lines = append(lines, consumerLines(lang, st)...)
		}
		return lines, nil
	case st.kind.cell:
		return []string{fmt.Sprintf(i18n.T(lang, "generator.lines.energy"), st.energy, st.energyCap())}, nil
	case st.kind == machineKinds[KindMiner]:
		// Its output shows as a slot.
		return append(append([]string{i18n.T(lang, "look.machine.mining") + gauge}, consumerLines(lang, st)...), blockedLines(lang, st)...), nil
	default:
		if _, it, ok := st.firstInput(); ok {
			line = &waila.ItemLine{Item: it, Name: naming.ItemName(lang, it), Gauge: step, ShowGauge: true}
			if it.Count() > 1 {
				line.Text = fmt.Sprintf("§f×%d", it.Count())
			}
		} else {
			lines = append(lines, i18n.T(lang, "look.empty"))
		}
	}
	if len(st.kind.charge) > 0 && st.charges > 0 {
		lines = append(lines, fmt.Sprintf(i18n.T(lang, "magicsieve.lines.charged"), mobName(lang, st.mob), st.charges))
	}
	lines = append(lines, consumerLines(lang, st)...)
	lines = append(lines, blockedLines(lang, st)...)
	for _, slot := range st.kind.output {
		if it, _ := st.inv().Item(slot); !it.Empty() {
			lines = append(lines, i18n.T(lang, "look.machine.output")+stackLine(lang, it))
		}
	}
	return lines, line
}

// detailLines are the figures the panel adds while its viewer sneaks: the
// length of a work cycle with the ticks left (the energy is in the lines).
func detailLines(lang string, st *machineState) []string {
	var lines []string
	if cycle := st.cycleLen(); cycle > 1 && len(st.kind.input)+len(st.kind.output) > 0 && !st.kind.crafter {
		if st.progress > 0 {
			lines = append(lines, fmt.Sprintf(i18n.T(lang, "look.detail.cycle"), cycle, max(0, cycle-st.progress)))
		} else {
			lines = append(lines, fmt.Sprintf(i18n.T(lang, "look.detail.cycleOnly"), cycle))
		}
	}
	return lines
}

// stackLine is an item's name in lang with its count when more than one.
func stackLine(lang string, it item.Stack) string {
	if it.Count() > 1 {
		return fmt.Sprintf("§f%s §7×%d", naming.ItemName(lang, it), it.Count())
	}
	return "§f" + naming.ItemName(lang, it)
}

// blockedLines say why a machine that finished does not go on: its output has
// no room for the product.
func blockedLines(lang string, st *machineState) []string {
	if len(st.kind.output) > 0 && st.progress >= st.cycleLen() {
		return []string{i18n.T(lang, "look.blocked")}
	}
	return nil
}

// crafterLines say what the auto crafter does: its recipe, what it lacks.
func crafterLines(lang string, st *machineState) []string {
	if st.recipe == nil {
		return []string{i18n.T(lang, "crafter.status.none")}
	}
	lines := []string{"§f" + naming.ItemName(lang, st.recipe.Out)}
	if _, ok := st.crafterPlan(st.recipe); !ok {
		lines = append(lines, fmt.Sprintf(i18n.T(lang, "crafter.status.missing"), crafterMissing(lang, st)))
	} else if !fits(st.inv(), st.kind.output, []item.Stack{st.recipe.Out}) {
		lines = append(lines, i18n.T(lang, "look.blocked"))
	} else {
		lines = append(lines, i18n.T(lang, "crafter.status.working"))
	}
	return append(lines, consumerLines(lang, st)...)
}

// crafterMissing lists the ingredients an auto crafter lacks for its recipe
// ("화약 ×2, 막대기 ×1").
func crafterMissing(lang string, st *machineState) string {
	var missing []string
	for _, req := range st.recipe.Reqs {
		have := 0
		for _, slot := range st.kind.input {
			if it, _ := st.inv().Item(slot); machinerecipe.MatchReq(it, req) {
				have += it.Count()
			}
		}
		if have < req.Count {
			missing = append(missing, fmt.Sprintf("%s ×%d", machinerecipe.ReqName(lang, req), req.Count-have))
		}
	}
	return strings.Join(missing, ", ")
}

// drawerDetail are the lines a sneaking viewer gets of a drawer: its upgrades
// and lock.
func drawerDetail(lang string, d Drawer) []string {
	lines := []string{fmt.Sprintf(i18n.T(lang, "look.drawer.capacity"), d.Capacity, dfitem.MaxCapacityUpgrades, textfmt.Int(d.capacity()))}
	if d.Void {
		lines = append(lines, i18n.T(lang, "look.drawer.void"))
	}
	if d.Locked {
		lines = append(lines, i18n.T(lang, "look.drawer.locked"))
	}
	return lines
}
