package dragonfactory

import (
	"image"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
)

// Hammer crushes blocks by hand: using it on a crushable block hits it, and
// after hammerHits hits the block turns into what the crusher makes of it
// (stone → cobblestone → gravel → sand). Holding the use button keeps
// hitting. Each crushed block costs one durability.
type Hammer struct{}

// hammerHits is the number of hits that crush a block, and hammerCooldown
// the least time between counted hits (clients repeat the use packet).
const (
	hammerHits     = 4
	hammerCooldown = 150 * time.Millisecond
)

// hammering is the block a player is hitting.
type hammering struct {
	pos  cube.Pos
	hits int
	last time.Time
}

func (Hammer) EncodeItem() (string, int16) { return "dragonfactory:hammer", 0 }
func (Hammer) Name() string                { return "망치" }
func (Hammer) Texture() image.Image        { return itemTexture("hammer") }
func (Hammer) Category() category.Category { return category.Equipment() }
func (Hammer) MaxCount() int               { return 1 }
func (Hammer) HandEquipped() bool          { return true }
func (Hammer) DurabilityInfo() item.DurabilityInfo {
	return item.DurabilityInfo{MaxDurability: 131, BrokenItem: func() item.Stack { return item.Stack{} }}
}

// UseOnBlock hits the block if it can be crushed.
func (Hammer) UseOnBlock(pos cube.Pos, _ cube.Face, _ mgl64.Vec3, tx *world.Tx, u item.User, ctx *item.UseContext) bool {
	p, ok := u.(*player.Player)
	if !ok {
		return false
	}
	s, ok := activeGame.sessionOf(tx.World())
	if !ok {
		return false
	}
	return s.hammer(p, tx, pos, ctx)
}

// crushed returns the block b turns into when crushed.
func crushed(b world.Block) (world.Block, bool) {
	out, ok := crushRecipes[itemKey(b.(world.Item))]
	if !ok {
		return nil, false
	}
	nb, ok := out.(world.Block)
	return nb, ok
}

func (s *session) hammer(p *player.Player, tx *world.Tx, pos cube.Pos, ctx *item.UseContext) bool {
	b := tx.Block(pos)
	if _, isItem := b.(world.Item); !isItem {
		return false
	}
	into, ok := crushed(b)
	if !ok {
		p.SendTip("§7망치로 부술 수 없는 블록입니다. (돌·조약돌·자갈)")
		return false
	}
	if _, expanding := s.expansionOf(p); expanding || !s.canBuild(pos) {
		return false
	}
	if s.hammering == nil {
		s.hammering = map[*world.EntityHandle]*hammering{}
	}
	h := s.hammering[p.H()]
	if h == nil || h.pos != pos {
		h = &hammering{pos: pos}
		s.hammering[p.H()] = h
	}
	if time.Since(h.last) < hammerCooldown {
		return true
	}
	h.last = time.Now()
	h.hits++
	p.SwingArm()
	if h.hits < hammerHits {
		tx.AddParticle(pos.Vec3Centre(), particle.PunchBlock{Block: b, Face: cube.FaceUp})
		tx.PlaySound(pos.Vec3Centre(), sound.BlockBreaking{Block: b})
		p.SendTip(hammerTip(h.hits))
		return true
	}
	delete(s.hammering, p.H())
	tx.SetBlock(pos, into, nil)
	tx.AddParticle(pos.Vec3Centre(), particle.BlockBreak{Block: b})
	tx.PlaySound(pos.Vec3Centre(), sound.BlockBreaking{Block: into})
	ctx.DamageItem(1)
	return true
}

// hammerTip shows the hits so far, e.g. "■■□□".
func hammerTip(hits int) string {
	out := "§e"
	for i := range hammerHits {
		if i == hits {
			out += "§8"
		}
		out += "■"
	}
	return out
}

func init() {
	world.RegisterItem(Hammer{})
}
