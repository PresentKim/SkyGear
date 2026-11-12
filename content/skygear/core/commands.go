package core

import (
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
)

// Commands returns the commands of the installed features: each feature's
// Commands in installation order. The game registers them when it starts.
func Commands() []cmd.Command {
	var out []cmd.Command
	for _, f := range Features() {
		if f.Commands != nil {
			out = append(out, f.Commands()...)
		}
	}
	return out
}

// PlayerSession is the player running a command and the session of its
// world. Without a player it returns false quietly; without a session it
// tells the player to save first.
func PlayerSession(src cmd.Source, o *cmd.Output, tx *world.Tx) (*player.Player, *Session, bool) {
	p, ok := src.(*player.Player)
	if !ok || tx == nil {
		return nil, nil, false
	}
	s, ok := Of(tx)
	if !ok {
		o.Error(i18n.T(i18n.Of(p), "menu.needSave"))
		return nil, nil, false
	}
	return p, s, true
}
