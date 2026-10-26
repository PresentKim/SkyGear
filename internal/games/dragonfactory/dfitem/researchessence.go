package dfitem

import (
	"image/color"

	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/presentkim/dragonfactory/common/i18n"
)

// Research essences (docs/design/11-research-essences.md): a research of a
// high enough level costs essences next to its materials. Each kind is one
// era of the factory, made in the laboratory (lab.go) from that era's
// materials. They are not the essences of mobs (essence.go).
const (
	EssenceEngineering = "engineering"
	EssenceAlchemy     = "alchemy"
	EssenceFarming     = "farming"
	EssenceHunting     = "hunting"
	EssenceMagic       = "magic"
)

// ResearchEssenceKinds are the kinds in the order they come in the game.
var ResearchEssenceKinds = []string{EssenceEngineering, EssenceAlchemy, EssenceFarming, EssenceHunting, EssenceMagic}

// ResearchEssence is the research essence of a kind.
type ResearchEssence struct{ Kind string }

func (e ResearchEssence) EncodeItem() (string, int16) {
	return "dragonfactory:research_essence_" + e.Kind, 0
}

func (e ResearchEssence) Name() string {
	return i18n.T(i18n.Default, "name.ResearchEssence."+e.Kind)
}

func (ResearchEssence) Category() category.Category { return category.Items() }

// EssenceColours are the colours of the essences, as splashed when a
// research is paid with them.
var EssenceColours = map[string]color.RGBA{
	EssenceEngineering: {214, 52, 52, 255}, EssenceAlchemy: {236, 140, 36, 255}, EssenceFarming: {232, 202, 48, 255},
	EssenceHunting: {64, 172, 72, 255}, EssenceMagic: {156, 76, 214, 255},
}
