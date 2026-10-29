package core

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
)

// SolidAt reports whether the block at p stops a monster.
func SolidAt(tx *world.Tx, p cube.Pos) bool {
	b := tx.Block(p)
	if _, liquid := b.(world.Liquid); liquid {
		return false
	}
	return len(b.Model().BBox(p, tx)) > 0
}

// Opt returns a form option whose action runs only while the player is in
// this session's world: session state belongs to that world.
func (s *Session) Opt(label, icon string, run func(p *player.Player, tx *world.Tx)) ui.Option {
	return ui.Option{Label: label, Icon: icon, Run: func(p *player.Player, tx *world.Tx) {
		if tx.World() == s.World {
			run(p, tx)
		}
	}}
}

// Guard makes the options run only in this session's world, as Opt does.
func (s *Session) Guard(opts []ui.Option) []ui.Option {
	out := make([]ui.Option, len(opts))
	for i, o := range opts {
		out[i] = s.Opt(o.Label, o.Icon, o.Run)
	}
	return out
}

// BackOption is the back button to the game menu, drawn left of the close
// button by the theme.
func (s *Session) BackOption(lang string) ui.Option {
	b := ui.Back(i18n.T(lang, "menu.back"), func(p *player.Player, _ *world.Tx) { ShowMenu(s, p) })
	return s.Opt(b.Label, b.Icon, b.Run)
}

// ShowMenu shows the game menu (the game sets it).
var ShowMenu = func(s *Session, p *player.Player) {}
