package platform

import (
	"github.com/df-mc/dragonfly/server/player"

	"github.com/presentkim/skygear/common/oreui"
)

// keepClientSettings sets the client settings the server holds for p while
// connected (docs/tech/oreui.md, client settings): the classic UI profile,
// whose screens and windows the server's UI is laid out for, and, for
// players who are not operators, the content log off.
func (pf *Platform) keepClientSettings(p *player.Player) {
	s := []oreui.Setting{{ID: "video.ui_profile", Kind: "option", Value: 0}}
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
