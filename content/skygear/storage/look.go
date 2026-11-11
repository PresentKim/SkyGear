package storage

import (
	"fmt"
	"math"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/waila"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/textfmt"
)

// Waila fills in a drawer's panel (waila.Provider). The drawer stays the
// icon; the held item shows like a chest's, a slot with its count under it;
// the lock and upgrades show on the drawer's front and, while the viewer
// sneaks, in the panel.
func (d Drawer) Waila(v waila.View, info waila.Info) waila.Info {
	if d.Item.Empty() {
		info.Lines = []string{i18n.T(v.Lang, "look.empty")}
	} else {
		info.Container = &waila.Container{Layout: waila.LayoutChest, Items: []item.Stack{d.Item.Grow(int(min(d.Count, math.MaxInt32)) - d.Item.Count())}}
	}
	if v.Sneaking {
		info.Lines = append(info.Lines, drawerDetail(v.Lang, d)...)
	}
	return info
}

// drawerDetail are the lines a sneaking viewer gets of a drawer: its upgrades
// and lock.
func drawerDetail(lang string, d Drawer) []string {
	lines := []string{fmt.Sprintf(i18n.T(lang, "look.drawer.capacity"), d.Capacity, sgitem.MaxCapacityUpgrades, textfmt.Int(d.MaxItems()))}
	if d.Void {
		lines = append(lines, i18n.T(lang, "look.drawer.void"))
	}
	if d.Locked {
		lines = append(lines, i18n.T(lang, "look.drawer.locked"))
	}
	return lines
}
