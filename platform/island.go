package platform

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/platform/instance"
)

// The tabs of the island management inside a save: the invite code (the
// host), the members (the host, or an operator) and the island expansion,
// which is the game's screen (instance.ScreenExpand).
const (
	islandInviteTab = iota
	islandMembersTab
)

// islandTabs are the tabs of the island management with on chosen. The
// invite code tab is the host's only.
func (pf *Platform) islandTabs(p *player.Player, on int) []ui.Option {
	lang := i18n.Of(p)
	var tabs []ui.Option
	if pf.hostedSave(p) != nil {
		tabs = append(tabs, ui.Tab(i18n.T(lang, "platform.island.tab.invite"), iconCode, on == islandInviteTab, func(p *player.Player, _ *world.Tx) {
			pf.showIsland(p, islandInviteTab, "")
		}))
	}
	tabs = append(tabs,
		ui.Tab(i18n.T(lang, "platform.island.tab.members"), iconMembers, on == islandMembersTab, func(p *player.Player, _ *world.Tx) {
			pf.showIsland(p, islandMembersTab, "")
		}),
		ui.Tab(i18n.T(lang, "platform.island.tab.expand"), iconExpand, false, func(p *player.Player, _ *world.Tx) {
			if inst := pf.m.Location(instance.AccountID(p)); inst != nil {
				inst.OpenScreen(p, instance.ScreenExpand)
			}
		}),
	)
	return tabs
}

// showIsland shows a tab of the island management of the save p is in, with
// the row sel chosen on the members tab. The invite code tab falls back to
// the members for an operator who is not the host.
func (pf *Platform) showIsland(p *player.Player, tab int, sel string) {
	inst := pf.managedSave(p)
	if inst == nil {
		ui.Notify(p, "%s", i18n.T(i18n.Of(p), "platform.members.hostOnly"))
		return
	}
	if tab == islandInviteTab && pf.hostedSave(p) != nil {
		pf.showInviteCode(p, inst)
		return
	}
	pf.showMembersAt(p, inst.Save.ID, sel)
}

// showInviteCode is the invite code tab: a switch that turns the code of the
// session on and off. Turning it on gives a new code each time; the code is
// gone once the host leaves (the instance closes), so a save never keeps
// one. Close friends enter without it (friends.go).
func (pf *Platform) showInviteCode(p *player.Player, inst *instance.Instance) {
	lang := i18n.Of(p)
	code, on := pf.inviteCode(inst)
	body := i18n.T(lang, "platform.island.code.off")
	if on {
		body = fmt.Sprintf(i18n.T(lang, "platform.island.code.on"), code)
	}
	opts := []ui.Option{ui.Toggle(i18n.T(lang, "platform.island.code.toggle"), on, func(p *player.Player, _ *world.Tx) {
		if cur := pf.hostedSave(p); cur == inst {
			if _, on := pf.inviteCode(inst); on {
				pf.codes.disable(inst.Save.ID)
			} else {
				pf.codes.enable(inst)
			}
		}
		pf.showIsland(p, islandInviteTab, "")
	})}
	opts = append(opts, pf.islandTabs(p, islandInviteTab)...)
	opts = append(opts, ui.Back(i18n.T(lang, "platform.back"), func(p *player.Player, _ *world.Tx) { pf.menuBack(p, instance.ScreenIsland) }))
	ui.Menu(p, i18n.T(lang, "platform.island.title"), body, opts...)
}
