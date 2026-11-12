package hopper

import "github.com/presentkim/skygear/common/waila"

// Waila adds a hopper's lines to its panel (waila.Provider); what it holds
// shows like a chest's, slots with counts under them.
func (h FlowHopper) Waila(v waila.View, info waila.Info) waila.Info {
	info.Lines = HopperLines(v.Lang, h)
	if h.Inv != nil {
		info.Container = &waila.Container{Layout: waila.LayoutChest, Items: h.Inv.Items()}
	}
	return info
}
