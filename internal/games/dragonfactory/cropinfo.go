package dragonfactory

import (
	"fmt"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/debug"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
)

// Crops by hand (docs/design/02-dragonfactory.md §9): using a crop shows its
// time to the next growth for a few seconds above it, to that player only
// and one crop at a time; using a grown crop harvests it and replants it
// (free for now).

// cropTagTime is how long the growth time stays above a crop.
const cropTagTime = 4 * time.Second

// cropNames are the Korean names of the crop kinds.
var cropNames = map[string]string{
	"wheat": "밀", "carrot": "당근", "potato": "감자", "beetroot": "비트",
	"melon": "수박 줄기", "pumpkin": "호박 줄기",
}

// useCrop handles a player using the crop at pos: a grown crop is
// harvested, a growing one shows its time. It returns true if pos is a crop
// (the click is then used up: nothing is placed on it).
func (s *session) useCrop(p *player.Player, tx *world.Tx, pos cube.Pos) bool {
	b := tx.Block(pos)
	kind, _, ok := cropOf(b)
	if !ok {
		return false
	}
	if after, ok := harvestable(b); ok {
		s.harvestByHand(tx, pos, b, after)
		return true
	}
	r, ok := s.state.Crops[cropKey(pos)]
	if !ok {
		s.plantCrop(pos, kind)
		r = s.state.Crops[cropKey(pos)]
	}
	s.showCropTag(p, pos, s.cropText(r, tx, pos))
	return true
}

// harvestByHand drops a grown crop's products and leaves it replanted.
func (s *session) harvestByHand(tx *world.Tx, pos cube.Pos, b, after world.Block) {
	if br, ok := b.(block.Breakable); ok && br.BreakInfo().Drops != nil {
		for _, st := range br.BreakInfo().Drops(item.ToolNone{}, nil) {
			s.dropAt(tx, pos.Vec3Centre(), st)
		}
	}
	tx.SetBlock(pos, after, nil)
	tx.PlaySound(pos.Vec3Centre(), sound.ItemAdd{})
	if kind, _, ok := cropOf(after); ok {
		s.plantCrop(pos, kind)
	}
}

// cropText describes the time left on a crop.
func (s *session) cropText(r *CropRecord, tx *world.Tx, pos cube.Pos) string {
	now, _, _ := s.clock.Now()
	_, stage, _ := cropOf(tx.Block(pos))
	name := cropNames[r.Kind]
	if stage < maxCropStage {
		left := max(0, r.Planted+s.growTime(r.Kind)-now)
		return fmt.Sprintf("§a%s §f%d/%d단계\n§7다 자라기까지 §f%s", name, stage, maxCropStage, formatDuration(left.Round(time.Second)))
	}
	if cropKinds[r.Kind].stem && r.NextFruit > 0 {
		left := max(0, r.NextFruit-now)
		return fmt.Sprintf("§a%s §f다 자람\n§7다음 열매까지 §f%s", name, formatDuration(left.Round(time.Second)))
	}
	return fmt.Sprintf("§a%s §f다 자람", name)
}

// showCropTag shows text above the crop at pos to p only, replacing the
// player's previous crop tag, and removes it after cropTagTime.
func (s *session) showCropTag(p *player.Player, pos cube.Pos, text string) {
	if s.cropTags == nil {
		s.cropTags = map[*world.EntityHandle]*debug.Text{}
	}
	if old, ok := s.cropTags[p.H()]; ok {
		p.RemoveDebugShape(old)
	}
	tag := &debug.Text{Position: pos.Vec3Centre().Add(mgl64.Vec3{0, 0.6, 0}), Text: text, Scale: 0.6}
	s.cropTags[p.H()] = tag
	p.AddDebugShape(tag)
	p.H().DoAfter(cropTagTime, func(tx *world.Tx, e world.Entity) {
		if tx.World() != s.world || s.cropTags[p.H()] != tag {
			return
		}
		delete(s.cropTags, p.H())
		e.(*player.Player).RemoveDebugShape(tag)
	})
}
