package skygear

import (
	"time"

	"github.com/presentkim/skygear/content/skygear/climate"

	"github.com/presentkim/skygear/content/skygear/machine"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/textfmt"
)

// Fairy time: fairy dust makes things that cut waiting short. Fairy bone
// meal grows a timed crop, the next fruit of a grown stem or a magic plant
// fairyBoneMealTime ahead on the growth clock, and a sapling into its tree;
// plain bone meal does nothing on the timed crops. The fairy hourglass runs
// an energy machine twice as fast for hourglassTicks, the extra runs free
// like the snow's, adding up to hourglassMaxTicks. Values are drafts.
const (
	fairyBoneMealTime = 2 * time.Minute
	fairyBoneMealMade = 4
)

// fairyTimeRecipes registers fairy bone meal (bone meal in the four corners,
// dust in the middle, four made) and the fairy hourglass (glass above and
// below sand, dust either side).
func fairyTimeRecipes() {
	var e item.Stack
	d := item.NewStack(sgitem.FairyDust{}, 1)
	b := item.NewStack(item.BoneMeal{}, 1)
	g := item.NewStack(block.Glass{}, 1)
	s := item.NewStack(block.Sand{}, 1)
	recipe.Register(recipe.NewShaped([]recipe.Item{b, e, b, e, d, e, b, e, b}, item.NewStack(sgitem.FairyBoneMeal{}, fairyBoneMealMade), recipe.NewShape(3, 3), "crafting_table"))
	recipe.Register(recipe.NewShaped([]recipe.Item{e, g, e, d, s, d, e, g, e}, item.NewStack(sgitem.FairyHourglass{}, 1), recipe.NewShape(3, 3), "crafting_table"))
}

func init() {
	world.RegisterItem(sgitem.FairyBoneMeal{})
	world.RegisterItem(sgitem.FairyHourglass{})
	fairyTimeRecipes()
}

// useFairyTime uses a held fairy bone meal or hourglass, or keeps plain
// bone meal off a timed crop (a click without sneaking is useCrop's). It
// reports whether the click was taken: an item that would change nothing
// is not spent and leaves the click to the block.
func (s *session) useFairyTime(p *player.Player, tx *world.Tx, pos cube.Pos, held item.Stack) bool {
	switch held.Item().(type) {
	case item.BoneMeal:
		_, _, crop := cropOf(tx.Block(pos))
		return crop && p.Sneaking()
	case sgitem.FairyBoneMeal:
		if !s.fairyBoneMeal(tx, pos) {
			return false
		}
		spendHeld(p, held)
		tx.AddParticle(pos.Vec3(), particle.BoneMeal{})
		return true
	case sgitem.FairyHourglass:
		m, ok := tx.Block(pos).(machine.Machine)
		if !ok || m.State == nil || !m.State.WindHourglass() {
			return false
		}
		spendHeld(p, held)
		tx.PlaySound(pos.Vec3Centre(), sound.ItemAdd{})
		return true
	}
	return false
}

// spendHeld takes one of the held stack held.
func spendHeld(p *player.Player, held item.Stack) {
	if p.GameMode().CreativeInventory() {
		return
	}
	_, off := p.HeldItems()
	p.SetHeldItems(held.Grow(-1), off)
}

// fairyBoneMeal grows the plant at pos ahead and reports whether it changed.
func (s *session) fairyBoneMeal(tx *world.Tx, pos cube.Pos) bool {
	switch b := tx.Block(pos).(type) {
	case MagicPlant:
		r := s.state.Plants[cropKey(pos)]
		if b.Bloom || r == nil {
			return false
		}
		r.ReadyAt -= fairyBoneMealTime
		s.tickPlants(tx)
		return true
	case block.Sapling:
		// Bone meal ages a sapling and then grows its tree, each by chance;
		// fairy bone meal goes on until the tree stands.
		for range 64 {
			b.BoneMeal(pos, tx)
			next, still := tx.Block(pos).(block.Sapling)
			if !still {
				return true
			}
			b = next
		}
		return false
	}
	kind, stage, ok := cropOf(tx.Block(pos))
	if !ok {
		return false
	}
	now := climate.GrowNow(s.Session)
	r := s.state.Crops[cropKey(pos)]
	if r == nil {
		s.plantCrop(pos, kind)
		r = s.state.Crops[cropKey(pos)]
		r.Planted = now - time.Duration(stage)*s.growTime(kind)/maxCropStage
	}
	switch {
	case stage < maxCropStage:
		r.Planted -= fairyBoneMealTime
	case cropKinds[kind].stem && r.NextFruit > 0:
		r.NextFruit -= fairyBoneMealTime
	default:
		return false
	}
	s.growCrop(tx, pos, r, now)
	return true
}

// hourglassLine is the WAILA line of the hourglass time left, or "".
func hourglassLine(lang string, st *machine.State) string {
	if st == nil || st.Boost <= 0 {
		return ""
	}
	left := time.Duration(st.Boost) * time.Second / 20
	return i18n.T(lang, "fairy.hourglass.left") + textfmt.Duration(lang, left.Round(time.Second))
}
