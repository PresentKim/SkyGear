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
	if _, ok := s.state.Crops[cropKey(pos)]; !ok {
		s.plantCrop(pos, kind)
	}
	s.showCropTag(p, tx, pos)
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

// cropTag is the crop time tag shown to one player.
type cropTag struct {
	text  *debug.Text
	pos   cube.Pos
	until time.Time
}

// showCropTag shows the time left on the crop at pos above it to p only,
// replacing the player's previous crop tag. refreshCropTags counts it down
// and removes it after cropTagTime.
func (s *session) showCropTag(p *player.Player, tx *world.Tx, pos cube.Pos) {
	if s.cropTags == nil {
		s.cropTags = map[*world.EntityHandle]*cropTag{}
	}
	if old, ok := s.cropTags[p.H()]; ok {
		p.RemoveDebugShape(old.text)
	}
	tag := &cropTag{pos: pos, until: time.Now().Add(cropTagTime)}
	s.cropTags[p.H()] = tag
	s.drawCropTag(p, tx, tag)
}

// drawCropTag shows a tag's current text. Shown shapes are read by the
// session later, so a changed text is a new shape replacing the old one.
func (s *session) drawCropTag(p *player.Player, tx *world.Tx, tag *cropTag) {
	r, ok := s.state.Crops[cropKey(tag.pos)]
	if !ok {
		return
	}
	text := s.cropText(r, tx, tag.pos)
	if tag.text != nil {
		if tag.text.Text == text {
			return
		}
		p.RemoveDebugShape(tag.text)
	}
	tag.text = &debug.Text{Position: tag.pos.Vec3Centre().Add(mgl64.Vec3{0, 0.6, 0}), Text: text, Scale: 0.6}
	p.AddDebugShape(tag.text)
}

// refreshCropTags counts the shown crop tags down every second and removes
// them when their time is up or the crop is gone.
func (s *session) refreshCropTags(tx *world.Tx) {
	if len(s.cropTags) == 0 {
		return
	}
	now := time.Now()
	for e := range tx.Players() {
		p := e.(*player.Player)
		tag, ok := s.cropTags[p.H()]
		if !ok {
			continue
		}
		if _, crop := s.state.Crops[cropKey(tag.pos)]; !crop || now.After(tag.until) {
			if tag.text != nil {
				p.RemoveDebugShape(tag.text)
			}
			delete(s.cropTags, p.H())
			continue
		}
		s.drawCropTag(p, tx, tag)
	}
}
