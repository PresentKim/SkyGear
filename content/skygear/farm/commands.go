package farm

import (
	"time"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/core"
)

// commands are the farm's commands: the sapling.
func commands() []cmd.Command {
	return []cmd.Command{
		cmd.New("묘목", i18n.T(i18n.Default, "cmd.sapling"), nil, saplingCmd{}),
		cmd.New("sapling", i18n.T(i18n.Default, "cmd.sapling"), nil, saplingCmd{}),
	}
}

// saplingCmd is /묘목 (and /sapling).
type saplingCmd struct{}

func (saplingCmd) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, ok := src.(*player.Player)
	if !ok {
		return
	}
	s, ok := core.Of(tx)
	if !ok {
		o.Error(i18n.T(i18n.Of(p), "sapling.needIsland"))
		return
	}
	if err := GiveSapling(s, p, tx, time.Now()); err != nil {
		o.Errorf(i18n.T(i18n.Of(p), "sapling.cooldown"), err.Error())
		return
	}
	o.Printf("%s", i18n.T(i18n.Of(p), "sapling.given"))
}
