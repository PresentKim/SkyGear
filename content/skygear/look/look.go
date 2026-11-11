// Package look is the game's WAILA panel (common/waila): it shows the block
// a player looks at, with its name, icon, tools and the game's own lines
// (machine progress, drawer contents, crop time, barrel contents); drawers
// and containers also show their item or contents in the slots.
package look

import (
	"fmt"
	"math"
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
	"github.com/presentkim/skygear/content/skygear/conduit"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/dimension"
	"github.com/presentkim/skygear/content/skygear/expand"
	"github.com/presentkim/skygear/content/skygear/fairy"
	"github.com/presentkim/skygear/content/skygear/farm"
	"github.com/presentkim/skygear/content/skygear/hopper"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/storage"
	"github.com/presentkim/skygear/content/skygear/textfmt"
	"github.com/presentkim/skygear/content/skygear/tinker"
	"github.com/presentkim/skygear/content/skygear/wand"
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
		if _, stage, crop := farm.CropOf(b); crop {
			h := float64(stage+1) / float64(farm.MaxCropStage+1)
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

// Update updates p's panel for a player at pos with rotation rot (the
// move handler passes where the player is moving to), unless it was
// updated less than lookInterval ago (force: always).
func Update(s *core.Session, p *player.Player, tx *world.Tx, pos mgl64.Vec3, rot cube.Rotation, force bool) {
	now := time.Now()
	if !force && now.Sub(lastLook.Of(s)[p.H()]) < lookInterval {
		return
	}
	lastLook.Of(s)[p.H()] = now
	if _, expanding := expand.Of(s, p); expanding {
		waila.Clear(p)
		return
	}
	at, b, ok := LookedAt(p, tx, pos, rot)
	showLook(s, p, tx, at, b, ok)
	wand.ShowPreview(s, p, tx)
}

// showLook shows p the panel of block b at pos it looks at (ok), or clears
// it.
func showLook(s *core.Session, p *player.Player, tx *world.Tx, at cube.Pos, b world.Block, ok bool) {
	if !ok {
		waila.Clear(p)
		return
	}
	if p.Sneaking() {
		machine.PeekRange(tx, at, b)
	}
	waila.Show(p, Info(s, p, tx, at, b))
}

// updateLookAt is updateLook with force for block b at pos p looks at (ok).
func updateLookAt(s *core.Session, p *player.Player, tx *world.Tx, at cube.Pos, b world.Block, ok bool) {
	lastLook.Of(s)[p.H()] = time.Now()
	if _, expanding := expand.Of(s, p); expanding {
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
		if ok {
			tinker.ShowVeinOutline(s, p, tx, at)
		} else {
			tinker.HideVeinOutline(s, p)
		}
		wand.ShowPreview(s, p, tx)
	}
}

// Info is the panel of block b at pos.
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
	switch b := b.(type) {
	case machine.Machine:
		info.Lines, info.Line = MachineLines(lang, b)
		if l := fairy.HourglassLine(lang, b.State); l != "" {
			info.Lines = append(info.Lines, l)
		}
		if b.State != nil && b.State.Def.Controller {
			info.Lines = storage.Lines(lang, tx, pos)
		}
		if c := outputContainer(b.State); c != nil {
			info.Container = c
		}
		if p.Sneaking() && b.State != nil {
			info.Lines = append(info.Lines, detailLines(lang, b.State)...)
		}
	case storage.Drawer:
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
		info.Lines = farm.ComposterLines(s, lang, pos, b)
	case conduit.Bundle:
		info.Lines = conduit.BundleLines(lang, tx, pos, b)
	case farm.SilkwormLeaves:
		info.Lines = []string{fmt.Sprintf(i18n.T(lang, "look.cocoon"), b.CocoonPercent())}
	case farm.Barrel:
		info.Lines = farm.BarrelLines(s, lang, tx, pos)
		info.Line = farm.BarrelLiquid(s, lang, pos)
	case hopper.FlowHopper:
		info.Lines = hopper.HopperLines(lang, b)
		if b.Inv != nil {
			// What it holds shows like a chest's, slots with counts under them.
			info.Container = &waila.Container{Layout: waila.LayoutChest, Items: b.Inv.Items()}
		}
	case farm.MagicPlant:
		if i := farm.PlantKindIndex(b.Kind); i >= 0 {
			k := farm.PlantKinds[i]
			seed := k.SeedStack()
			info.Item, info.NameKey = seed.Item(), ""
			info.Name = i18n.T(lang, "pseed."+k.Seed)
			info.Tools = []waila.ToolIcon{{Tool: waila.Crops, Mark: waila.MarkNone}}
			info.Lines = farm.PlantLines(s, lang, pos, b, k)
		}
	}
	if kind, _, ok := farm.CropOf(b); ok {
		info.Name = i18n.T(lang, farm.CropNameKeys[kind])
		info.Tools = []waila.ToolIcon{{Tool: waila.Crops, Mark: waila.MarkNone}}
		if r, ok := farm.StateOf(s).Crops[farm.CropKey(pos)]; ok {
			info.Lines = farm.CropLines(s, lang, r, tx, pos)
		}
	}
	return info
}

// outputContainer is the output slots of the machine st as a chest's slots,
// each item with its count under it, or nil for a machine with no output or
// a drawer controller (whose lines count the drawers instead), and nil while
// the output is empty.
func outputContainer(st *machine.State) *waila.Container {
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
	return &waila.Container{Layout: layout, Items: inv.Slots(), Progress: FurnaceArrow(b)}
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
	return item.NewStack(machine.ProgressArrow{Step: step}, 1)
}

// MachineLines are a machine's lines and item line: what it holds, with
// its icon and progress as a ring gauge ("흙 [icon] ×3 ◔"), or grey
// "비어 있음" when empty. What waits in its output shows as slots
// (outputContainer).
func MachineLines(lang string, m machine.Machine) ([]string, *waila.ItemLine) {
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
	case st.Def == machine.Kinds[machine.KindLab]:
		// The research it makes shows with its icon on the item line.
		lines := machine.LabLines(lang, st)
		if st.Lab == nil {
			return lines, nil
		}
		return lines[1:], itemLine(lang, item.NewStack(st.Lab.Out, st.Lab.OutCount))
	case st.Def == machine.Kinds[machine.KindSellBox]:
		lines = append(lines, i18n.T(lang, "look.machine.sellCycle")+gauge)
	case st.Def == machine.Kinds[machine.KindHarvester]:
		lines = append(lines, i18n.T(lang, "look.machine.harvestCycle")+gauge)
	case st.Def == machine.Kinds[machine.KindGenerator]:
		return machine.GeneratorLines(lang, st), nil
	case machine.IsNatureGenerator(m.Kind):
		return machine.NatureLines(lang, st), nil
	case m.Kind == machine.KindGate, m.Kind == machine.KindDrawerTrim, m.Kind == machine.KindTinker, m.Kind == machine.KindRelicTable:
		return nil, nil
	case st.Def.Controller:
		return nil, nil
	case st.Def.FluidCap > 0:
		lines := machine.FluidLines(lang, st)
		if _, in, ok := st.FirstInput(); ok {
			line = itemLine(lang, in)
		}
		if st.Def.Generates > 0 {
			lines = append(lines, fmt.Sprintf(i18n.T(lang, "generator.lines.energy"), st.Energy, st.EnergyCap()))
		} else {
			lines = append(lines, machine.ConsumerLines(lang, st)...)
		}
		return lines, line
	case st.Def.Cell:
		return []string{fmt.Sprintf(i18n.T(lang, "generator.lines.energy"), st.Energy, st.EnergyCap())}, nil
	case st.Def == machine.Kinds[machine.KindMiner]:
		// Its output shows as a slot.
		return append(append([]string{i18n.T(lang, "look.machine.mining") + gauge}, machine.ConsumerLines(lang, st)...), machine.BlockedLines(lang, st)...), nil
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
		lines = append(lines, fmt.Sprintf(i18n.T(lang, "magicsieve.lines.charged"), dimension.MobName(lang, st.Mob), st.Charges))
	}
	lines = append(lines, machine.ConsumerLines(lang, st)...)
	lines = append(lines, machine.BlockedLines(lang, st)...)
	return lines, line
}

// detailLines are the figures the panel adds while its viewer sneaks: the
// length of a work cycle with the ticks left (the energy is in the lines).
func detailLines(lang string, st *machine.State) []string {
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
func crafterLines(lang string, st *machine.State) []string {
	if st.Recipe == nil {
		return []string{i18n.T(lang, "crafter.status.none")}
	}
	lines := []string{"§f" + naming.ItemName(lang, st.Recipe.Out)}
	if _, ok := st.CrafterPlan(st.Recipe); !ok {
		lines = append(lines, fmt.Sprintf(i18n.T(lang, "crafter.status.missing"), machine.CrafterMissing(lang, st)))
	} else if !machine.Fits(st.Inv(), st.Def.Output, []item.Stack{st.Recipe.Out}) {
		lines = append(lines, i18n.T(lang, "look.blocked"))
	} else {
		lines = append(lines, i18n.T(lang, "crafter.status.working"))
	}
	return append(lines, machine.ConsumerLines(lang, st)...)
}

// drawerDetail are the lines a sneaking viewer gets of a drawer: its upgrades
// and lock.
func drawerDetail(lang string, d storage.Drawer) []string {
	lines := []string{fmt.Sprintf(i18n.T(lang, "look.drawer.capacity"), d.Capacity, sgitem.MaxCapacityUpgrades, textfmt.Int(d.MaxItems()))}
	if d.Void {
		lines = append(lines, i18n.T(lang, "look.drawer.void"))
	}
	if d.Locked {
		lines = append(lines, i18n.T(lang, "look.drawer.locked"))
	}
	return lines
}

// lastLook is when each player's WAILA panel was last updated.
var lastLook = core.NewPlayers[time.Time]()
