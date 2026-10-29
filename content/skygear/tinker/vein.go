package tinker

import (
	"slices"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/platform/account"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// Vein mining (like the VeinMiner mod): mining a block the iron hammer
// crushes while sneaking also breaks the connected blocks of the same kind,
// up to veinLimit and what the player's food pays for (veinFoodCost a
// block; fractions carry over in veinFood).

// The vein preview tints every block the hammer would break with stained
// glass on its liquid layer, sent only to that player (an edge outline goes
// off screen up close). It is removed before the blocks break, or glass
// would be left floating where they were.

// veinNeighbours are the offsets vein mining follows: the six faces, and
// with diagonal also the blocks that touch only at an edge or a corner.
func veinNeighbours(diagonal bool) []cube.Pos {
	var out []cube.Pos
	for x := -1; x <= 1; x++ {
		for y := -1; y <= 1; y++ {
			for z := -1; z <= 1; z++ {
				if n := core.Abs(x) + core.Abs(y) + core.Abs(z); n == 1 || (diagonal && n > 1) {
					out = append(out, cube.Pos{x, y, z})
				}
			}
		}
	}
	return out
}

// VeinDiagonalSetting is the player setting (account.Setting) that makes a
// player's vein mining follow diagonal neighbours.
const VeinDiagonalSetting = "vein_diagonal"

// ToggleVeinDiagonal switches p's diagonal vein mining, stores it and tells p
// which it is now: nothing on screen shows it otherwise.
func ToggleVeinDiagonal(s *core.Session, p *player.Player) {
	on, err := account.Toggle(p, VeinDiagonalSetting)
	if err != nil {
		s.Log.Error("store setting", "player", p.Name(), "err", err)
	}
	key := "hammer.diagonal.off"
	if on {
		key = "hammer.diagonal.on"
	}
	HideVeinOutline(s, p)
	p.SendTip(i18n.T(i18n.Of(p), key))
}

// veinToolBudget is how many blocks past the mined one held can break
// whatever the food: veinLimit (the iron hammer does not wear).
func veinToolBudget(s *core.Session, held item.Stack) int {
	return VeinLimit - 1
}

// veinFoodBudget is how many blocks past the mined one p's food pays for.
func veinFoodBudget(s *core.Session, p *player.Player) int {
	food := float64(p.Food()) - VeinFood.Of(s)[p.H()]
	return max(0, int(food/veinFoodCost+1e-9))
}

// VeinMine breaks the blocks connected to the one at pos p mines with an
// iron hammer while sneaking (pos itself is broken by the game): each costs
// veinFoodCost food, and what the hammer makes of them
// drops together at pos.
func VeinMine(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) {
	held, _ := p.HeldItems()
	if !sgitem.IsIronHammer(held) || !p.Sneaking() {
		return
	}
	// the preview goes first, or its glass would stay where blocks were
	HideVeinOutline(s, p)
	left := VeinBudget(s, p, held)
	diagonal := account.On(p, VeinDiagonalSetting)
	if tool := veinToolBudget(s, held); veinFoodBudget(s, p) < tool {
		// The food stops the vein short of what the hammer would break: the
		// player sees fewer blocks go and nothing else says why.
		if len(VeinBlocks(tx, pos, tool+1, diagonal, func(p cube.Pos) bool { return island.CanBuild(s, p) }))-1 > left {
			p.SendTip(i18n.T(i18n.Of(p), "hammer.vein.hungry"))
		}
	}
	blocks := VeinBlocks(tx, pos, left+1, diagonal, func(p cube.Pos) bool { return island.CanBuild(s, p) })
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
		drop, ok := sgitem.HammerDrop(b)
		if !ok {
			continue
		}
		achievement.CountBreak(p, b, true)
		tx.SetBlock(at, nil, nil)
		tx.AddParticle(at.Vec3Centre(), particle.BlockBreak{Block: b})
		drops = addDrop(drops, drop)
		broken++
	}
	for _, drop := range drops {
		for n := drop.Count(); n > 0; n -= drop.MaxCount() {
			s.DropAt(tx, pos.Vec3Centre(), drop.Grow(min(n, drop.MaxCount())-drop.Count()))
		}
	}
	if broken > 0 {
		PayVeinFood(s, p, broken)
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

// veinPreviewBlock is the block the preview puts on the liquid layer.
var veinPreviewBlock world.Block = block.StainedGlass{Colour: item.ColourLightBlue()}

// veinOutline is the preview a player is shown: the vein it is of (its
// first block and size) and the positions tinted.
type veinOutline struct {
	start  cube.Pos
	size   int
	blocks []cube.Pos
}

// ShowVeinOutline shows p, sneaking with the iron hammer, the blocks it
// would break mining the block at pos (hiding them otherwise).
func ShowVeinOutline(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) {
	if held, _ := p.HeldItems(); !sgitem.IsIronHammer(held) || !p.Sneaking() || !island.CanBuild(s, pos) {
		HideVeinOutline(s, p)
		return
	}
	held, _ := p.HeldItems()
	blocks := VeinBlocks(tx, pos, VeinBudget(s, p, held)+1, account.On(p, VeinDiagonalSetting), func(p cube.Pos) bool { return island.CanBuild(s, p) })
	if len(blocks) == 0 {
		HideVeinOutline(s, p)
		return
	}
	if cur, ok := veinOutlines.Of(s)[p.H()]; ok && cur.size == len(blocks) && slices.Contains(blocks, cur.start) {
		return
	}
	old := veinOutlines.Of(s)[p.H()].blocks
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
	veinOutlines.Of(s)[p.H()] = veinOutline{start: pos, size: len(blocks), blocks: blocks}
}

// HideVeinOutline removes the preview p is shown, if any.
func HideVeinOutline(s *core.Session, p *player.Player) {
	o, ok := veinOutlines.Of(s)[p.H()]
	if !ok {
		return
	}
	delete(veinOutlines.Of(s), p.H())
	if tx := p.Tx(); tx != nil {
		restoreLiquidLayer(p, tx, o.blocks)
	}
}

// sendLiquidLayer shows p b on the liquid layer at pos.
func sendLiquidLayer(p *player.Player, pos cube.Pos, b world.Block) {
	sghook.WritePacket(sghook.Session(p), &packet.UpdateBlock{
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

// veinOutlines are the vein outlines players are shown.
var veinOutlines = core.NewPlayers[veinOutline]()
