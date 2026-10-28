package account

import (
	"github.com/df-mc/dragonfly/server/player"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/internal/instance"
)

// Setting is an on/off choice of a player, kept with the account (db
// account_settings, "1" on and "0" off): the platform's settings screen lists
// every registered one, and the code it is about reads it with On.
type Setting struct {
	// Key is the account_settings key.
	Key string
	// NameKey is the catalog key of the name the settings screen shows.
	NameKey string
	// Default is the choice of a player who never changed it.
	Default bool
}

// Name is the setting's name in lang.
func (s Setting) Name(lang string) string { return i18n.T(lang, s.NameKey) }

// The settings of the platform: how an achievement earned is told (a toast,
// chat lines, the sound).
const (
	SettingAchievementToast = "notify_achievement_toast"
	SettingAchievementChat  = "notify_achievement_chat"
	SettingAchievementSound = "notify_achievement_sound"
)

var (
	settings []Setting
	// chosen caches the settings of online accounts that read one, by
	// account and key (missing: the default).
	chosen = map[string]map[string]bool{}
)

func init() {
	RegisterSetting(
		Setting{Key: SettingAchievementToast, NameKey: "setting.achievement.toast", Default: true},
		Setting{Key: SettingAchievementChat, NameKey: "setting.achievement.chat", Default: true},
		Setting{Key: SettingAchievementSound, NameKey: "setting.achievement.sound", Default: true},
	)
}

// RegisterSetting adds settings, shown in registration order. Keys must be
// unique across games.
func RegisterSetting(s ...Setting) {
	mu.Lock()
	defer mu.Unlock()
	settings = append(settings, s...)
}

// Settings returns every registered setting in registration order.
func Settings() []Setting {
	mu.Lock()
	defer mu.Unlock()
	return append([]Setting(nil), settings...)
}

// On reports whether p has the setting key on. The settings of an account
// are read once, the first time one is asked. It may be called from any
// world transaction.
func On(p *player.Player, key string) bool {
	acc := instance.AccountID(p)
	mu.Lock()
	set, ok := chosen[acc]
	all := settings
	d := store
	mu.Unlock()
	if !ok {
		set = map[string]bool{}
		for _, s := range all {
			if d == nil {
				break
			}
			v, err := d.Setting(acc, s.Key)
			if err != nil {
				log.Error("read setting", "account", acc, "key", s.Key, "err", err)
				continue
			}
			if v != "" {
				set[s.Key] = v == "1"
			}
		}
		mu.Lock()
		chosen[acc] = set
		mu.Unlock()
	}
	mu.Lock()
	defer mu.Unlock()
	if v, ok := chosen[acc][key]; ok {
		return v
	}
	for _, s := range settings {
		if s.Key == key {
			return s.Default
		}
	}
	return false
}

// Toggle turns the setting key of p over, stores it and returns the new
// choice.
func Toggle(p *player.Player, key string) (bool, error) {
	on := !On(p, key)
	value := "0"
	if on {
		value = "1"
	}
	acc := instance.AccountID(p)
	mu.Lock()
	d := store
	mu.Unlock()
	if d != nil {
		if err := d.SetSetting(acc, key, value); err != nil {
			return !on, err
		}
	}
	mu.Lock()
	if chosen[acc] == nil {
		chosen[acc] = map[string]bool{}
	}
	chosen[acc][key] = on
	mu.Unlock()
	return on, nil
}
