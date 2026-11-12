package tinker

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/machine"
)

// registerKinds hands the machine frame the behaviour of the tinker's machine
// kind: a click opens the tinker table, whose panel has no lines.
func registerKinds() {
	machine.Register(machine.KindTinker, machine.Behaviour{
		Click: func(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool {
			if !s.Debounced(p) {
				ShowTable(s, p, tx, pos)
			}
			return true
		},
		Lines: machine.NoLines,
	})
}
