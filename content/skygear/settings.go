package skygear

import "github.com/presentkim/skygear/platform/account"

// The game's own settings, after the advancement settings (progress).
func init() {
	account.RegisterSetting(
		account.Setting{Key: veinDiagonalSetting, NameKey: "setting.vein.diagonal"},
		account.Setting{Key: autoToolSetting, NameKey: "setting.autotool"},
	)
}
