package dfitem

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/presentkim/dragonfactory/common/i18n"
)

// TierAlloy is an alloy of the machine tiers: infused, reinforced, atomic.
type TierAlloy struct{ Kind string }

// TierAlloys are the alloys, weakest first.
var TierAlloys = []string{"infused", "reinforced", "atomic"}

func (a TierAlloy) EncodeItem() (string, int16) { return "dragonfactory:alloy_" + a.Kind, 0 }

func (a TierAlloy) Name() string { return i18n.T(i18n.Default, "tier.alloy."+a.Kind) }

func (TierAlloy) Category() category.Category { return category.Items() }

// Circuit is a control circuit of level 1 to 4.
type Circuit struct{ Level int }

func (c Circuit) EncodeItem() (string, int16) {
	return fmt.Sprintf("dragonfactory:circuit_%d", c.Level), 0
}

func (c Circuit) Name() string { return i18n.T(i18n.Default, fmt.Sprintf("tier.circuit.%d", c.Level)) }

func (Circuit) Category() category.Category { return category.Items() }

// TierKit is a kit that raises a machine to its level (1 to maxTier).
type TierKit struct{ Level int }

func (k TierKit) EncodeItem() (string, int16) { return fmt.Sprintf("dragonfactory:kit_%d", k.Level), 0 }

func (k TierKit) Name() string { return i18n.T(i18n.Default, fmt.Sprintf("tier.kit.%d", k.Level)) }

func (TierKit) Category() category.Category { return category.Items() }

func (TierKit) MaxCount() int { return 16 }
