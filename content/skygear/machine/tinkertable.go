package machine

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/tinker"
)

// The tinker table is the tinker's block, but the tinker feature cannot
// register its kind (Register): it would have to import this package, and
// then the machines would be initialised, and their items registered, before
// the tinker's tools. The item ids the client receives follow the
// registration order. So this package, which imports tinker, sets the kind's
// behaviour itself: a click opens the tinker table, and the panel has no
// lines. This file is the only one of the package that imports a feature.
func init() {
	Kinds[KindTinker].Behaviour = Behaviour{
		Click: func(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool {
			if !s.Debounced(p) {
				tinker.ShowTable(s, p, tx, pos)
			}
			return true
		},
		Lines: NoLines,
	}
}
