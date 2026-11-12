package sgitem

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

// The items of this package that are used on the air are item.Usable:
// Dragonfly calls their Use after the handler's subscribers, unless one of
// them cancelled the use. What the use does belongs to a feature, and this
// package is a data layer below the features, so the feature sets the
// function of its item from its Setup. An item whose function is not set
// does nothing.
var (
	// CopierUsed is the use of a configuration copier on the air (copier).
	CopierUsed func(tx *world.Tx, user item.User)
	// WandUsed is the use of a construction wand on the air (wand).
	WandUsed func(tx *world.Tx, user item.User)
	// HammerUsed is the use of a hammer on the air (tinker).
	HammerUsed func(tx *world.Tx, user item.User)
	// StoneUsed is the use of a dimension teleport stone on the air
	// (dimension).
	StoneUsed func(tx *world.Tx, user item.User)
)

// used calls f, if it is set, and reports false. A use of these items
// changes the stack in hand or opens a screen by itself; for a true result
// Dragonfly would swing the arm and write the stack it read before the use
// back into the hand.
func used(f func(*world.Tx, item.User), tx *world.Tx, user item.User) bool {
	if f != nil {
		f(tx, user)
	}
	return false
}

// Use implements item.Usable (CopierUsed).
func (ConfigCopier) Use(tx *world.Tx, user item.User, _ *item.UseContext) bool {
	return used(CopierUsed, tx, user)
}

// Use implements item.Usable (WandUsed).
func (Wand) Use(tx *world.Tx, user item.User, _ *item.UseContext) bool {
	return used(WandUsed, tx, user)
}

// Use implements item.Usable (WandUsed).
func (InfinityWand) Use(tx *world.Tx, user item.User, _ *item.UseContext) bool {
	return used(WandUsed, tx, user)
}

// Use implements item.Usable (HammerUsed).
func (Hammer) Use(tx *world.Tx, user item.User, _ *item.UseContext) bool {
	return used(HammerUsed, tx, user)
}

// Use implements item.Usable (StoneUsed).
func (DimensionStone) Use(tx *world.Tx, user item.User, _ *item.UseContext) bool {
	return used(StoneUsed, tx, user)
}
