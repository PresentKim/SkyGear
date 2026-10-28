package platform

import (
	"github.com/df-mc/dragonfly/server/player"

	"github.com/presentkim/skygear/common/oreui"
)

// keepClientSettings sets the client settings the server holds for p while
// connected (docs/tech/oreui.md, client settings): the classic UI profile,
// whose screens and windows the server's UI is laid out for.
func (pf *Platform) keepClientSettings(p *player.Player) {
	s := []oreui.Setting{{ID: "video.ui_profile", Kind: "option", Value: 0}}
	oreui.SetClientSettings(p, s)
}
