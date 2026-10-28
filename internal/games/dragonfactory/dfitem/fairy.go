package dfitem

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/i18n"
)

// ResearchNote is a retired fairy gift. It stays registered so old
// inventories load; using it only says so.
type ResearchNote struct{}

func (ResearchNote) EncodeItem() (string, int16) { return "dragonfactory:research_note", 0 }

func (ResearchNote) Name() string { return i18n.T(i18n.Default, "name.ResearchNote") }

func (ResearchNote) Category() category.Category { return category.Items() }

func (ResearchNote) Use(_ *world.Tx, u item.User, _ *item.UseContext) bool {
	if p, ok := u.(*player.Player); ok {
		p.Message(i18n.T(i18n.Of(p), "fairy.researchNote.unused"))
	}
	return false
}

// FairyDust is what the fairy leaves: the stuff of the fairy magic.
type FairyDust struct{}

func (FairyDust) EncodeItem() (string, int16) { return "dragonfactory:fairy_dust", 0 }

func (FairyDust) Name() string { return i18n.T(i18n.Default, "name.FairyDust") }

func (FairyDust) Category() category.Category { return category.Items() }

// FairyEmerald is spent by every spell of the fairy board.
type FairyEmerald struct{}

func (FairyEmerald) EncodeItem() (string, int16) { return "dragonfactory:fairy_emerald", 0 }

func (FairyEmerald) Name() string { return i18n.T(i18n.Default, "name.FairyEmerald") }

func (FairyEmerald) Category() category.Category { return category.Items() }

// FairyBoneMeal grows the timed crops and magic plants ahead, which plain
// bone meal does not (fairytime.go).
type FairyBoneMeal struct{}

func (FairyBoneMeal) EncodeItem() (string, int16) { return "dragonfactory:fairy_bone_meal", 0 }

func (FairyBoneMeal) Name() string { return i18n.T(i18n.Default, "name.FairyBoneMeal") }

func (FairyBoneMeal) Category() category.Category { return category.Items() }

// FairyHourglass runs an energy machine faster for a while (fairytime.go).
type FairyHourglass struct{}

func (FairyHourglass) EncodeItem() (string, int16) { return "dragonfactory:fairy_hourglass", 0 }

func (FairyHourglass) Name() string { return i18n.T(i18n.Default, "name.FairyHourglass") }

func (FairyHourglass) Category() category.Category { return category.Items() }
