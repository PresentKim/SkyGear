package netstore

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/conduit"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/machine"
)

// showController lists the nodes of the controller's network by kind and its
// energy: what the network takes each tick and what the controller holds.
func showController(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) {
	lang := i18n.Of(p)
	n := At(tx, pos)
	if n == nil {
		return
	}
	body := i18n.T(lang, "netstore.controller.on")
	switch {
	case n.Controllers > 1:
		body = i18n.T(lang, "netstore.controller.many")
	case len(n.members) > NodeCap:
		body = fmt.Sprintf(i18n.T(lang, "netstore.controller.cap"), NodeCap)
	case !n.Online():
		body = i18n.T(lang, "netstore.terminal.offline")
	}
	body += "\n" + fmt.Sprintf(i18n.T(lang, "netstore.controller.use"), n.Use())
	if st := stateAt(tx, pos); st != nil {
		if st.Unlimited {
			body += "\n" + i18n.T(lang, "netstore.controller.unlimited")
		} else {
			body += "\n" + fmt.Sprintf(i18n.T(lang, "netstore.controller.energy"), st.Energy, st.EnergyCap())
		}
	}
	body += "\n" + fmt.Sprintf(i18n.T(lang, "netstore.controller.nodes"), len(n.members), NodeCap)
	kinds := make([]string, 0, len(n.byKind))
	for k := range n.byKind {
		kinds = append(kinds, k)
	}
	slices.SortFunc(kinds, func(a, b string) int { return slices.Index(machine.Order, a) - slices.Index(machine.Order, b) })
	slots := make([]ui.ItemSlot, 0, len(kinds))
	for _, k := range kinds {
		count := len(n.byKind[k])
		use := nodeUse(k) * count
		if k == machine.KindNetController {
			use = controllerUse
		}
		icon, title := kindLook(lang, k)
		slots = append(slots, ui.ItemSlot{
			Icon:  icon,
			Count: strconv.Itoa(count),
			Tip:   title + "\n" + fmt.Sprintf(i18n.T(lang, "netstore.controller.kindUse"), count, use),
		})
	}
	ui.ItemGrid(p, machine.Kinds[machine.KindNetController].Title(lang), body, slots)
}

// kindLook is the icon and name in lang of the nodes of kind on the
// controller's list: a machine's, or the network conduit's for the conduit
// bundles (KindConduit, which is no machine kind).
func kindLook(lang, kind string) (icon, title string) {
	if kind == KindConduit {
		return ui.ItemIcon(conduit.NetConduit{}), i18n.T(lang, "name.NetConduit")
	}
	return ui.ItemIcon(machine.Machine{Kind: kind}), machine.Kinds[kind].Title(lang)
}
