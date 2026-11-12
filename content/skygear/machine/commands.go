package machine

import (
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/progress"
)

// commands are the machines' commands: the production statistics.
func commands() []cmd.Command {
	return []cmd.Command{
		cmd.New("통계", i18n.T(i18n.Default, "monitor.cmd"), nil, statsCmd{}),
		cmd.New("stats", i18n.T(i18n.Default, "monitor.cmd"), nil, statsCmd{}),
	}
}

// statsCmd is /통계 (and /stats): the production statistics.
type statsCmd struct{}

func (statsCmd) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, s, ok := core.PlayerSession(src, o, tx)
	if !ok {
		return
	}
	progress.Advance(s, tx, advancement.Stats)
	ShowStats(s, p, StatsItems)
}
