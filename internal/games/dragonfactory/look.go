package dragonfactory

import (
	"fmt"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/cube/trace"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/internal/waila"
)

// The WAILA panel (internal/waila, docs/tech/waila.md) shows the block a
// player looks at: its name, icon, tools and the game's own lines (machine
// progress, drawer contents, crop time, what a core makes, the mission).

// lookReach is how far a player looks for a block, in blocks.
const lookReach = 6

// lookInterval limits how often moving or turning updates the panel.
const lookInterval = 100 * time.Millisecond

// lookedAt returns the block a player at pos with rotation rot looks at.
// Blocks without a collision box (crops, flowers) count when the view
// passes through them; air and liquids do not.
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
		if len(b.Model().BBox(pos, tx)) > 0 && !trace.BlockIntersects(pos, tx, b, eye, end) {
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

// updateLooks updates every player's panel (every second, for changing
// lines such as progress and time left).
func (s *session) updateLooks(tx *world.Tx) {
	for e := range tx.Players() {
		p := e.(*player.Player)
		s.updateLook(p, tx, p.Position(), p.Rotation(), true)
	}
}

// lookInfo is the panel of block b at pos.
func (s *session) lookInfo(p *player.Player, tx *world.Tx, pos cube.Pos, b world.Block) waila.Info {
	held, _ := p.HeldItems()
	info := waila.Info{Name: blockName(b), Source: "Minecraft", Tools: waila.ToolsFor(b, held)}
	if it, ok := b.(world.Item); ok {
		info.Item = it
		info.Name = itemName(item.NewStack(it, 1))
	}
	if _, custom := b.(world.CustomBlock); custom {
		info.Source = "드래곤팩토리"
	}
	switch b := b.(type) {
	case Machine:
		info.Lines = machineLines(b)
	case Drawer:
		info.Lines = drawerLines(b)
	case Core:
		info.Lines = s.coreLines(tx, pos, b)
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

// machineLines are a machine's progress and what it works on.
func machineLines(m Machine) []string {
	st := m.st
	if st == nil {
		return nil
	}
	pct := st.progress * 100 / st.kind.interval
	var lines []string
	switch st.kind {
	case machineKinds[KindSellBox]:
		lines = append(lines, fmt.Sprintf("판매 주기 §f%d%%", pct))
	case machineKinds[KindHarvester]:
		lines = append(lines, fmt.Sprintf("수확 주기 §f%d%%", pct))
	default:
		if _, in, ok := st.firstInput(); ok {
			lines = append(lines, fmt.Sprintf("진행 §f%d%%", pct), fmt.Sprintf("넣은 것 §f%s ×%d", itemName(in), in.Count()))
		} else if st.kind == machineKinds[KindMiner] {
			lines = append(lines, fmt.Sprintf("채굴 §f%d%%", pct))
		} else {
			lines = append(lines, "대기 §7(넣을 것 없음)")
		}
	}
	return lines
}

// drawerLines are a drawer's contents, lock and upgrades.
func drawerLines(d Drawer) []string {
	if d.Item.Empty() {
		return []string{"비어 있음"}
	}
	lines := []string{fmt.Sprintf("%s §f%s§7/%s", itemName(d.Item), formatInt(d.Count), formatInt(d.capacity()))}
	if d.Locked {
		lines = append(lines, "§e잠김")
	}
	if d.Capacity > 0 {
		lines = append(lines, fmt.Sprintf("용량 업그레이드 §f×%d", 1<<d.Capacity))
	}
	if d.Void {
		lines = append(lines, "§d공허 업그레이드")
	}
	return lines
}

// coreLines are what a core makes and when.
func (s *session) coreLines(tx *world.Tx, pos cube.Pos, c Core) []string {
	ct := c.ctype()
	if ct == nil {
		return nil
	}
	lines := []string{fmt.Sprintf("%d티어", ct.Tier)}
	r, need := matchRecipe(tx, pos, ct.Tier)
	switch {
	case r != nil:
		lines = append(lines, "만드는 것 §f"+blockName(r.Output))
		if _, air := tx.Block(pos.Side(cube.FaceUp)).(block.Air); !air {
			lines = append(lines, "§c위 칸이 막혀 멈춤")
		} else {
			left := time.Duration(s.coreInterval(ct.Tier)-c.Progress) * time.Second / 20
			lines = append(lines, fmt.Sprintf("다음까지 §f%.1f초", left.Seconds()))
		}
	case need > 0:
		lines = append(lines, fmt.Sprintf("§c%d티어 이상 코어가 필요", need))
	default:
		lines = append(lines, "§7이 코어 기반으로는 만들 것이 없음")
	}
	return lines
}
