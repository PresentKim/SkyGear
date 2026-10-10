package dragonfactory

import (
	"fmt"
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
	if it, ok := b.(world.Item); ok {
		info.Item = it
		info.Name = itemName(item.NewStack(it, 1))
	}
	info.Container = containerOf(tx, pos, b)
	switch b := b.(type) {
	case Machine:
		info.Lines = machineLines(b)
		if b.Kind == KindMiner && b.st != nil {
			// The auto miner's one output slot shows like a container's.
			it, _ := b.st.inv().Item(b.st.kind.output[0])
			info.Container = &waila.Container{Layout: waila.LayoutChest, Items: []item.Stack{it}}
		}
	case Drawer:
		// The drawer stays the icon; the held item is a line with its own
		// small icon, and the lock and upgrades are small marks after the
		// name (11th playtest).
		info.Marks = waila.Marks{Lock: b.Locked, Void: b.Void}
		if b.Capacity > 0 {
			info.Marks.Multiplier = 1 << b.Capacity
		}
		if b.Item.Empty() {
			info.Lines = []string{"§7비어있음"}
		} else {
			info.Line = &waila.ItemLine{Item: b.Item, Name: itemName(b.Item),
				Text: fmt.Sprintf("%s/%s", formatInt(b.Count), formatInt(b.capacity()))}
		}
	case Core:
		info.Lines, info.Line = s.coreLines(tx, pos, b)
	case DeliveryBox:
		info.Lines = []string{s.missionLine()}
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

// machineLines are what a machine holds and its progress, as a ring gauge
// (11th playtest: the percent moved in uneven steps). Machines show no
// slots; an empty one says so in grey.
func machineLines(m Machine) []string {
	st := m.st
	if st == nil {
		return nil
	}
	gauge := gaugeGlyph(st.progress * 10 / st.kind.interval)
	var lines []string
	switch {
	case st.kind == machineKinds[KindSellBox]:
		lines = append(lines, "판매 주기 §f"+gauge)
	case st.kind == machineKinds[KindHarvester]:
		lines = append(lines, "수확 주기 §f"+gauge)
	case st.kind == machineKinds[KindMiner]:
		lines = append(lines, "채굴 §f"+gauge)
	default:
		if _, it, ok := st.firstInput(); ok {
			lines = append(lines, stackLine(it)+" §f"+gauge)
		} else {
			lines = append(lines, "§7비어있음")
		}
	}
	if st.kind == machineKinds[KindMiner] {
		return lines // its output shows as a slot
	}
	for _, slot := range st.kind.output {
		if it, _ := st.inv().Item(slot); !it.Empty() {
			lines = append(lines, "§7결과 "+stackLine(it))
		}
	}
	return lines
}

// stackLine is an item's name with its count when more than one.
func stackLine(it item.Stack) string {
	if it.Count() > 1 {
		return fmt.Sprintf("§f%s §7×%d", itemName(it), it.Count())
	}
	return "§f" + itemName(it)
}

// coreLines are a core's lines and its item line: what it makes, with its
// icon and a ring gauge ("생성물: 조약돌 [icon] ◔"; the tier is in the name).
func (s *session) coreLines(tx *world.Tx, pos cube.Pos, c Core) ([]string, *waila.ItemLine) {
	ct := c.ctype()
	if ct == nil {
		return nil, nil
	}
	r, need := matchRecipe(tx, pos, ct.Tier)
	switch {
	case r != nil:
		line := &waila.ItemLine{Name: "생성물: §f" + blockName(r.Output)}
		if it, ok := r.Output.(world.Item); ok {
			line.Item = item.NewStack(it, 1)
		}
		if _, air := tx.Block(pos.Side(cube.FaceUp)).(block.Air); !air {
			return []string{"§c위 칸이 막혀 멈춤"}, line
		}
		line.Gauge, line.ShowGauge = c.Progress*10/max(s.coreInterval(ct.Tier), 1), true
		return nil, line
	case need > 0:
		return []string{fmt.Sprintf("§c%d티어 이상 코어가 필요", need)}, nil
	default:
		return []string{"§7이 코어 기반으로는 만들 것이 없음"}, nil
	}
}
