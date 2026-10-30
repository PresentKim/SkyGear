package skygear

import (
	"github.com/presentkim/skygear/content/skygear/tinker"
	"github.com/presentkim/skygear/platform/account"
)

// The game's own settings, after the advancement settings (progress).
func init() {
	account.RegisterSetting(
		account.Setting{Key: tinker.VeinDiagonalSetting, NameKey: "setting.vein.diagonal", Group: "setting.group.play"},
		account.Setting{Key: tinker.AutoToolSetting, NameKey: "setting.autotool", Group: "setting.group.play"},
	)
}
