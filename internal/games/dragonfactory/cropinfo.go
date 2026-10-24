package dragonfactory

import (
	"github.com/presentkim/dragonfactory/common/i18n"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
)

// Crops by hand (docs/design/02-dragonfactory.md §12): using a grown crop
// harvests it and replants it (free for now); using a growing one does
// nothing, so no block is placed on it. The WAILA panel shows the time
// left (look.go).

// cropNameKeys are the catalog keys of the names of the crop kinds.
var cropNameKeys = map[string]string{
	"wheat": "dragonfactory.cropinfo.init.1", "carrot": "dragonfactory.cropinfo.init.2", "potato": "dragonfactory.cropinfo.init.3", "beetroot": "dragonfactory.cropinfo.init.4",
	"melon": "dragonfactory.cropinfo.init.5", "pumpkin": "dragonfactory.cropinfo.init.6",
}

// useCrop handles a player using the crop at pos: a grown crop is
// harvested. It returns true if pos is a crop (the click is then used up:
// nothing is placed on it).
func (s *session) useCrop(p *player.Player, tx *world.Tx, pos cube.Pos) bool {
	b := tx.Block(pos)
	kind, _, ok := cropOf(b)
	if !ok {
		return false
	}
	if after, ok := harvestable(b); ok {
		count(p, "crop_harvest", 1)
		count(p, "crop_"+kind, 1)
		s.harvestByHand(tx, pos, b, after)
		return true
	}
	if _, ok := s.state.Crops[cropKey(pos)]; !ok {
		s.plantCrop(pos, kind)
	}
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

// cropLines are the time left on a crop (the stage is only how the time
// shows on the plant, not a number for players; 11th playtest).
func (s *session) cropLines(lang string, r *CropRecord, tx *world.Tx, pos cube.Pos) []string {
	now, _, _ := s.clock.Now()
	_, stage, _ := cropOf(tx.Block(pos))
	if stage < maxCropStage {
		left := max(0, r.Planted+s.growTime(r.Kind)-now)
		return []string{i18n.T(lang, "dragonfactory.cropinfo.cropLines.1") + formatDuration(lang, left.Round(time.Second))}
	}
	if cropKinds[r.Kind].stem && r.NextFruit > 0 {
		left := max(0, r.NextFruit-now)
		return []string{i18n.T(lang, "dragonfactory.cropinfo.cropLines.2"), i18n.T(lang, "dragonfactory.cropinfo.cropLines.3") + formatDuration(lang, left.Round(time.Second))}
	}
	return []string{i18n.T(lang, "dragonfactory.cropinfo.cropLines.4")}
}
