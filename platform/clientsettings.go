package platform

import (
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/oreui"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/platform/instance"
)

// keepClientSettings sets the client settings the server holds for p while
// connected (docs/tech/oreui.md, client settings): the classic UI profile,
// whose screens and windows the server's UI is laid out for, when classic is
// true, and, for players who are not operators, the content log off.
func (pf *Platform) keepClientSettings(p *player.Player, classic bool) {
	var s []oreui.Setting
	if classic {
		s = append(s, oreui.Setting{ID: "video.ui_profile", Kind: "option", Value: 0})
	}
	if pf.isOperator == nil || !pf.isOperator(p) {
		s = append(s,
			oreui.Setting{ID: "creator.contentLog.gui.enable", Kind: "bool", Value: false},
			oreui.Setting{ID: "creator.contentLog.gui.showErrors", Kind: "bool", Value: false},
			oreui.Setting{ID: "creator.contentLog.file.enable", Kind: "bool", Value: false},
			oreui.Setting{ID: "creator.scriptDiagnostic.enableClient", Kind: "bool", Value: false},
		)
	}
	oreui.SetClientSettings(p, s)
}

// usesClassicUI reports whether p's client joined with the classic UI
// profile (login client data UIProfile 0; 1 is pocket).
func usesClassicUI(p *player.Player) bool {
	c := sghook.Of(p)
	return c == nil || c.ClientData().UIProfile == 0
}

// askClassicUI recommends the classic UI to p: agreeing sets the profile,
// which stays in the client's options after p leaves. Either answer goes on
// to the guide when guide is true, else to the lobby's main screen.
func (pf *Platform) askClassicUI(p *player.Player, guide bool) {
	lang := i18n.Of(p)
	next := func(p *player.Player) {
		if guide {
			pf.showGuide(p, true)
			return
		}
		pf.showLobbyMenu(p)
	}
	ui.Menu(p, i18n.T(lang, "platform.classicUI.title"), i18n.T(lang, "platform.classicUI.body"),
		ui.Option{Label: i18n.T(lang, "platform.classicUI.agree"), Icon: "textures/ui/confirm", Run: func(p *player.Player, _ *world.Tx) {
			pf.keepClientSettings(p, true)
			next(p)
		}},
		ui.Option{Label: i18n.T(lang, "platform.classicUI.refuse"), Icon: "textures/ui/cancel", Run: func(p *player.Player, _ *world.Tx) {
			next(p)
		}},
	)
}

// holdLobbyMenu keeps the lobby's main screen from opening when p arrives
// from joining, as askClassicUI opens first.
func (pf *Platform) holdLobbyMenu(p *player.Player) {
	pf.classicAsk.Store(instance.AccountID(p), true)
}

func (pf *Platform) lobbyMenuHeld(p *player.Player) bool {
	_, held := pf.classicAsk.LoadAndDelete(instance.AccountID(p))
	return held
}
