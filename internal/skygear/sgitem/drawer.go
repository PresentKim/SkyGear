package sgitem

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/presentkim/skygear/common/i18n"
)

// MaxCapacityUpgrades limits capacity upgrades per drawer.
const MaxCapacityUpgrades = 3

// DrawerUpgrade is an upgrade item for drawers.
type DrawerUpgrade struct{ Kind int }

const (
	UpgradeCapacity = iota
	UpgradeVoid
)

var UpgradeInfo = []struct{ id, Name string }{
	{"capacity", i18n.T(i18n.Default, "drawer.upgrade.capacity")},
	{"void", i18n.T(i18n.Default, "drawer.upgrade.void")},
}

func (u DrawerUpgrade) EncodeItem() (string, int16) {
	return "skygear:drawer_upgrade_" + UpgradeInfo[u.Kind].id, 0
}

func (u DrawerUpgrade) Name() string { return UpgradeInfo[u.Kind].Name }

func (DrawerUpgrade) Category() category.Category { return category.Items() }

func IsDrawerUpgrade(st item.Stack) bool {
	_, ok := st.Item().(DrawerUpgrade)
	return ok
}

// DrawerKey locks and unlocks drawers. It is never used up.
type DrawerKey struct{}

func (DrawerKey) EncodeItem() (string, int16) { return "skygear:drawer_key", 0 }

func (DrawerKey) Name() string { return i18n.T(i18n.Default, "name.DrawerKey") }

func (DrawerKey) Category() category.Category { return category.Items() }

func (DrawerKey) MaxCount() int { return 1 }

func IsDrawerKey(st item.Stack) bool {
	_, ok := st.Item().(DrawerKey)
	return ok
}
