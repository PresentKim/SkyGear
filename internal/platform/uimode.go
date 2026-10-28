package platform

import (
	"sync"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/google/uuid"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/internal/instance"
)

// uiModeSetting is the account setting (db account_settings) that makes a
// player's menus JSON UI forms instead of DDUI screens ("json"). The default
// is the same for every device.
const uiModeSetting = "ui_mode"

// jsonUI holds the players online who chose the JSON UI forms, by UUID.
var jsonUI sync.Map

func init() {
	ui.PreferJSON = func(p *player.Player) bool {
		_, ok := jsonUI.Load(p.UUID())
		return ok
	}
}

// loadUIMode reads p's menu setting.
func (pf *Platform) loadUIMode(p *player.Player) {
	v, err := pf.db.Setting(instance.AccountID(p), uiModeSetting)
	if err != nil {
		pf.log.Error("read setting", "player", p.Name(), "err", err)
	}
	setJSONUI(p.UUID(), v == "json")
}

func setJSONUI(id uuid.UUID, on bool) {
	if on {
		jsonUI.Store(id, true)
	} else {
		jsonUI.Delete(id)
	}
}

// toggleUIMode switches p's menus between DDUI screens and JSON UI forms,
// stores it and tells p which it is now.
func (pf *Platform) toggleUIMode(p *player.Player) {
	_, was := jsonUI.Load(p.UUID())
	if !pf.setUIMode(p, !was) {
		return
	}
	key := "platform.uimode.json"
	if was {
		key = "platform.uimode.ddui"
	}
	p.Message(i18n.T(i18n.Of(p), key))
}

// setUIMode makes p's menus JSON UI forms (json) or DDUI screens and stores
// it; it reports whether it could.
func (pf *Platform) setUIMode(p *player.Player, json bool) bool {
	value := ""
	if json {
		value = "json"
	}
	if err := pf.db.SetSetting(instance.AccountID(p), uiModeSetting, value); err != nil {
		pf.fail(p, "store setting", err)
		return false
	}
	setJSONUI(p.UUID(), json)
	return true
}

type uiModeCmd struct {
	playerOnly
	pf *Platform
}

func (c uiModeCmd) Run(src cmd.Source, _ *cmd.Output, _ *world.Tx) {
	c.pf.toggleUIMode(src.(*player.Player))
}

// uiDebugCmd shows or hides the debug line of the player's own Ore UI screens
// (common ui.SetOreDebug): the renderer's state and the fonts it found,
// for a screenshot from a device that draws the screens wrong.
type uiDebugCmd struct {
	playerOnly
}

func (uiDebugCmd) Run(src cmd.Source, _ *cmd.Output, _ *world.Tx) {
	p := src.(*player.Player)
	on := !ui.OreDebug(p)
	ui.SetOreDebug(p, on)
	key := "platform.uidebug.off"
	if on {
		key = "platform.uidebug.on"
	}
	p.Message(i18n.T(i18n.Of(p), key))
}
