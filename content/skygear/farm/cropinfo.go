package farm

import (
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/climate"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/textfmt"
)

// Crops by hand: using a grown crop
// harvests it and replants it for free; using a growing one does
// nothing, so no block is placed on it. The WAILA panel shows the time
// left (look.go).

// CropNameKeys are the catalog keys of the names of the crop kinds.
var CropNameKeys = map[string]string{
	"wheat": "crop.wheat", "carrot": "crop.carrot", "potato": "crop.potato", "beetroot": "crop.beetroot",
	"melon": "crop.melonStem", "pumpkin": "crop.pumpkinStem",
}

// UseCrop handles a player using the crop at pos: a grown crop is
// harvested. It returns true if pos is a crop (the click is then used up:
// nothing is placed on it).
func UseCrop(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool {
	b := tx.Block(pos)
	kind, _, ok := CropOf(b)
	if !ok {
		return false
	}
	if after, ok := harvestable(b); ok {
		achievement.Count(p, "crop_harvest", 1)
		achievement.Count(p, "crop_"+kind, 1)
		harvestByHand(s, tx, pos, b, after)
		return true
	}
	if _, ok := StateOf(s).Crops[CropKey(pos)]; !ok {
		PlantCrop(s, pos, kind)
	}
	return true
}

// harvestByHand drops a grown crop's products and leaves it replanted.
func harvestByHand(s *core.Session, tx *world.Tx, pos cube.Pos, b, after world.Block) {
	if br, ok := b.(block.Breakable); ok && br.BreakInfo().Drops != nil {
		for _, st := range br.BreakInfo().Drops(item.ToolNone{}, nil) {
			s.DropAt(tx, pos.Vec3Centre(), st)
		}
	}
	tx.SetBlock(pos, after, nil)
	tx.PlaySound(pos.Vec3Centre(), sound.ItemAdd{})
	if kind, _, ok := CropOf(after); ok {
		PlantCrop(s, pos, kind)
	}
}

// CropLines are the time left on a crop (the stage is only how the time
// shows on the plant, not a number for players).
func CropLines(s *core.Session, lang string, r *CropRecord, tx *world.Tx, pos cube.Pos) []string {
	now := climate.GrowNow(s)
	_, stage, _ := CropOf(tx.Block(pos))
	if stage < MaxCropStage {
		left := climate.RealLeft(s, max(0, r.Planted+GrowTime(s, r.Kind)-now))
		return []string{i18n.T(lang, "crop.lines.left") + textfmt.Duration(lang, left.Round(time.Second))}
	}
	if CropKinds[r.Kind].Stem && r.NextFruit > 0 {
		left := climate.RealLeft(s, max(0, r.NextFruit-now))
		return []string{i18n.T(lang, "crop.lines.grown"), i18n.T(lang, "crop.lines.nextFruit") + textfmt.Duration(lang, left.Round(time.Second))}
	}
	return []string{i18n.T(lang, "crop.lines.grown")}
}
