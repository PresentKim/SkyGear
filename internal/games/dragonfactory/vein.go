package dragonfactory

import (
	"slices"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/presentkim/dragonfactory/common/dfhook"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/internal/account"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/dfitem"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// Vein mining (like the VeinMiner mod): the iron hammer, mining a crushable
// block while sneaking, also breaks the blocks of the same kind connected to
// it face to face, up to veinLimit and what the hammer's durability (one a
// block) and the player's food (veinFoodCost a block, from the food bar;
// fractions carry over to the next vein in session.veinFood) allow. Only blocks the hammer crushes (hammerDrop) count.
// While a player sneaks with the iron hammer and looks at such a block, what
// it would break shows tinted to that player (the vein preview, below).

// veinLimit is the most blocks one vein mining breaks, the mined one
// included. Each block past the mined one also costs veinFoodCost food
// points: a full food bar (20) pays for veinLimit blocks.
const (
	veinLimit    = 256
	veinFoodCost = 20.0 / veinLimit
)

// veinNeighbours are the offsets vein mining follows: the six faces, and
// with diagonal also the blocks that touch only at an edge or a corner.
func veinNeighbours(diagonal bool) []cube.Pos {
	var out []cube.Pos
	for x := -1; x <= 1; x++ {
		for y := -1; y <= 1; y++ {
			for z := -1; z <= 1; z++ {
				if n := abs(x) + abs(y) + abs(z); n == 1 || (diagonal && n > 1) {
					out = append(out, cube.Pos{x, y, z})
				}
			}
		}
	}
	return out
}

// veinBlocks are the blocks of the same kind as the one at start connected
// to it face to face (with diagonal also edge to edge and corner to corner),
// start first, up to limit; none when the hammer does not crush it. canBuild
// filters the blocks a player may break.
func veinBlocks(tx *world.Tx, start cube.Pos, limit int, diagonal bool, canBuild func(cube.Pos) bool) []cube.Pos {
	b := tx.Block(start)
	it, ok := b.(world.Item)
	if _, crushable := dfitem.HammerDrop(b); !ok || !crushable {
		return nil
	}
	name, meta := it.EncodeItem()
	same := func(pos cube.Pos) bool {
		other, ok := tx.Block(pos).(world.Item)
		if !ok {
			return false
		}
		otherName, otherMeta := other.EncodeItem()
		return otherName == name && otherMeta == meta && canBuild(pos)
	}
	out := []cube.Pos{start}
	seen := map[cube.Pos]bool{start: true}
	neighbours := veinNeighbours(diagonal)
	for i := 0; i < len(out) && len(out) < limit; i++ {
		for _, d := range neighbours {
			n := out[i].Add(d)
			if seen[n] || n.OutOfBounds(tx.Range()) {
				continue
			}
			seen[n] = true
			if same(n) {
				if out = append(out, n); len(out) >= limit {
					break
				}
			}
		}
	}
	return out
}

// veinDiagonalSetting is the player setting (account.Setting) that makes a
// player's vein mining follow diagonal neighbours.
const veinDiagonalSetting = "vein_diagonal"

// toggleVeinDiagonal switches p's diagonal vein mining, stores it and tells p
// which it is now: nothing on screen shows it otherwise.
func (s *session) toggleVeinDiagonal(p *player.Player) {
	on, err := account.Toggle(p, veinDiagonalSetting)
	if err != nil {
		s.g.log.Error("store setting", "player", p.Name(), "err", err)
	}
	key := "hammer.diagonal.off"
	if on {
		key = "hammer.diagonal.on"
	}
	s.hideVeinOutline(p)
	p.SendTip(i18n.T(i18n.Of(p), key))
}

// veinBudget is how many blocks past the mined one p can break with held:
// what the durability (one is left for the mined block) and the food allow.
func (s *session) veinBudget(p *player.Player, held item.Stack) int {
	return max(0, min(s.veinToolBudget(held), s.veinFoodBudget(p)))
}

// veinToolBudget is how many blocks past the mined one held can break
// whatever the food: veinLimit (the iron hammer does not wear).
func (s *session) veinToolBudget(held item.Stack) int {
	return veinLimit - 1
}

// veinFoodBudget is how many blocks past the mined one p's food pays for.
func (s *session) veinFoodBudget(p *player.Player) int {
	food := float64(p.Food()) - s.veinFood[p.H()]
	return max(0, int(food/veinFoodCost+1e-9))
}

// payVeinFood takes the food of n vein mined blocks from p.
func (s *session) payVeinFood(p *player.Player, n int) {
	if s.veinFood == nil {
		s.veinFood = map[*world.EntityHandle]float64{}
	}
	owed := s.veinFood[p.H()] + float64(n)*veinFoodCost
	whole := int(owed + 1e-9)
	s.veinFood[p.H()] = max(0, owed-float64(whole))
	if whole > 0 {
		p.SetFood(p.Food() - whole)
	}
}

// veinMine breaks the blocks connected to the one at pos p mines with an
// iron hammer while sneaking (pos itself is broken by the game): each costs
// veinFoodCost food, and what the hammer makes of them
// drops together at pos.
func (s *session) veinMine(p *player.Player, tx *world.Tx, pos cube.Pos) {
	held, _ := p.HeldItems()
	if !dfitem.IsIronHammer(held) || !p.Sneaking() {
		return
	}
	// the preview goes first, or its glass would stay where blocks were
	s.hideVeinOutline(p)
	left := s.veinBudget(p, held)
	diagonal := account.On(p, veinDiagonalSetting)
	if tool := s.veinToolBudget(held); s.veinFoodBudget(p) < tool {
		// The food stops the vein short of what the hammer would break: the
		// player sees fewer blocks go and nothing else says why.
		if len(veinBlocks(tx, pos, tool+1, diagonal, s.canBuild))-1 > left {
			p.SendTip(i18n.T(i18n.Of(p), "hammer.vein.hungry"))
		}
	}
	blocks := veinBlocks(tx, pos, left+1, diagonal, s.canBuild)
	if len(blocks) < 2 || left <= 0 {
		return
	}
	broken := 0
	var drops []item.Stack
	for _, at := range blocks[1:] {
		if broken >= left {
			break
		}
		b := tx.Block(at)
		drop, ok := dfitem.HammerDrop(b)
		if !ok {
			continue
		}
		countBreak(p, b, true)
		tx.SetBlock(at, nil, nil)
		tx.AddParticle(at.Vec3Centre(), particle.BlockBreak{Block: b})
		drops = addDrop(drops, drop)
		broken++
	}
	for _, drop := range drops {
		for n := drop.Count(); n > 0; n -= drop.MaxCount() {
			s.dropAt(tx, pos.Vec3Centre(), drop.Grow(min(n, drop.MaxCount())-drop.Count()))
		}
	}
	if broken > 0 {
		s.payVeinFood(p, broken)
	}
}

// addDrop adds drop to drops, onto a stack it stacks with when there is one.
func addDrop(drops []item.Stack, drop item.Stack) []item.Stack {
	if i := slices.IndexFunc(drops, drop.Comparable); i >= 0 {
		drops[i] = drops[i].Grow(drop.Count())
		return drops
	}
	return append(drops, drop)
}

// The vein preview (like presentkim-pm/selection-visualize-utils'
// BlockPreview): every block the iron hammer would break gets a light blue
// stained glass on its liquid layer (layer 1), sent only to the player, so
// the block looks tinted however close the player stands (an outline drawn
// on the edges went off screen up close). The real layer 1 (air, or the
// water of a waterlogged block) is sent again to remove it. The preview
// stays while the player keeps looking at the same vein, changes with it,
// and is removed before the blocks break (veinMine, HandleBlockBreak) so
// no glass is left floating where a block was.

// veinPreviewBlock is the block the preview puts on the liquid layer.
var veinPreviewBlock world.Block = block.StainedGlass{Colour: item.ColourLightBlue()}

// veinOutline is the preview a player is shown: the vein it is of (its
// first block and size) and the positions tinted.
type veinOutline struct {
	start  cube.Pos
	size   int
	blocks []cube.Pos
}

// showVeinOutline shows p, sneaking with the iron hammer, the blocks it
// would break mining the block at pos (hiding them otherwise).
func (s *session) showVeinOutline(p *player.Player, tx *world.Tx, pos cube.Pos) {
	if held, _ := p.HeldItems(); !dfitem.IsIronHammer(held) || !p.Sneaking() || !s.canBuild(pos) {
		s.hideVeinOutline(p)
		return
	}
	held, _ := p.HeldItems()
	blocks := veinBlocks(tx, pos, s.veinBudget(p, held)+1, account.On(p, veinDiagonalSetting), s.canBuild)
	if len(blocks) == 0 {
		s.hideVeinOutline(p)
		return
	}
	if cur, ok := s.veinOutlines[p.H()]; ok && cur.size == len(blocks) && slices.Contains(blocks, cur.start) {
		return
	}
	old := s.veinOutlines[p.H()].blocks
	tinted := map[cube.Pos]bool{}
	for _, b := range blocks {
		tinted[b] = true
	}
	// restore what the new preview does not cover, tint what it adds
	var stale []cube.Pos
	had := map[cube.Pos]bool{}
	for _, b := range old {
		had[b] = true
		if !tinted[b] {
			stale = append(stale, b)
		}
	}
	restoreLiquidLayer(p, tx, stale)
	for _, b := range blocks {
		if !had[b] {
			sendLiquidLayer(p, b, veinPreviewBlock)
		}
	}
	if s.veinOutlines == nil {
		s.veinOutlines = map[*world.EntityHandle]veinOutline{}
	}
	s.veinOutlines[p.H()] = veinOutline{start: pos, size: len(blocks), blocks: blocks}
}

// hideVeinOutline removes the preview p is shown, if any.
func (s *session) hideVeinOutline(p *player.Player) {
	o, ok := s.veinOutlines[p.H()]
	if !ok {
		return
	}
	delete(s.veinOutlines, p.H())
	if tx := p.Tx(); tx != nil {
		restoreLiquidLayer(p, tx, o.blocks)
	}
}

// sendLiquidLayer shows p b on the liquid layer at pos.
func sendLiquidLayer(p *player.Player, pos cube.Pos, b world.Block) {
	dfhook.WritePacket(dfhook.Session(p), &packet.UpdateBlock{
		Position:          protocol.BlockPos{int32(pos[0]), int32(pos[1]), int32(pos[2])},
		NewBlockRuntimeID: world.BlockRuntimeID(b),
		Flags:             packet.BlockUpdateNetwork,
		Layer:             1,
	})
}

// restoreLiquidLayer shows p the real liquid layer at each of the positions.
func restoreLiquidLayer(p *player.Player, tx *world.Tx, at []cube.Pos) {
	for _, pos := range at {
		var b world.Block = block.Air{}
		if l, ok := tx.Liquid(pos); ok {
			b = l
		}
		sendLiquidLayer(p, pos, b)
	}
}
