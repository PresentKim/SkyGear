package tinker

import (
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/platform/account"
)

// Feature is the tinker feature, listed in the game's feature list.
var Feature = &core.Feature{
	Name: "tinker",
	Settings: []account.Setting{
		{Key: VeinDiagonalSetting, NameKey: "setting.vein.diagonal", Group: "setting.group.play"},
		{Key: AutoToolSetting, NameKey: "setting.autotool", Group: "setting.group.play"},
	},
	Setup: subscribe,
}
