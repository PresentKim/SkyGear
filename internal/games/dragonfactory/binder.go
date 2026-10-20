package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/i18n"
)

// The conduit binder (EnderIO's conduit binder, docs/design/06-power-and-conduits.md
// §5.1): sand and gravel in a checkerboard make the binder mix, which a
// furnace bakes into the binder, the base every conduit is made of.

// ConduitBinderMix is baked into a ConduitBinder.
type ConduitBinderMix struct{}

func (ConduitBinderMix) EncodeItem() (string, int16) { return "dragonfactory:conduit_binder_mix", 0 }
func (ConduitBinderMix) Name() string                { return i18n.T(i18n.Default, "name.ConduitBinderMix") }
func (ConduitBinderMix) Category() category.Category { return category.Items() }

// SmeltInfo implements item.Smeltable.
func (ConduitBinderMix) SmeltInfo() item.SmeltInfo {
	return item.SmeltInfo{Product: item.NewStack(ConduitBinder{}, 1), Experience: 0.1}
}

// ConduitBinder is the base of the conduits.
type ConduitBinder struct{}

func (ConduitBinder) EncodeItem() (string, int16) { return "dragonfactory:conduit_binder", 0 }
func (ConduitBinder) Name() string                { return i18n.T(i18n.Default, "name.ConduitBinder") }
func (ConduitBinder) Category() category.Category { return category.Items() }

func init() {
	world.RegisterItem(ConduitBinderMix{})
	world.RegisterItem(ConduitBinder{})
}
