package dimension

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/wintext"
)

// registerKinds hands the machine frame the behaviour of the dimension's
// machine kinds: the essence collector's window has a button to pick the
// monster it receives, which its info icon names; a click opens the relic
// table, whose panel has no lines.
func registerKinds() {
	machine.Register(machine.KindCollector, machine.Behaviour{
		Info:       collectorInfo,
		ButtonItem: collectorButton,
		Button:     showCollectorMobs,
	})
	machine.Register(machine.KindRelicTable, machine.Behaviour{
		Click: func(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool {
			if !s.Debounced(p) {
				ShowRelicTable(s, p, "")
			}
			return true
		},
		Lines: machine.NoLines,
	})
}

// collectorInfo is the line the collector's info icon adds: the monster it
// receives, or that none is chosen.
func collectorInfo(st *machine.State) []wintext.Text {
	if st.Mob != "" {
		return []wintext.Text{{wintext.Key("machine.collector.mob", wintext.Key("mob."+st.Mob))}}
	}
	return []wintext.Text{{wintext.Key("machine.collector.none")}}
}
