package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/ui"
)

// probeCmd is /formprobe: a form for checking in a real client which binding
// forms make a form layout build its buttons (docs/tech/json-ui.md, "폼 성능").
type probeCmd struct{}

func (probeCmd) Run(src cmd.Source, _ *cmd.Output, tx *world.Tx) {
	p, ok := src.(*player.Player)
	if !ok || tx == nil {
		return
	}
	nop := func(*player.Player, *world.Tx) {}
	var opts []ui.Option
	for _, l := range []string{"1", "2", "3", "4", "5"} {
		opts = append(opts, ui.Option{Label: "b" + l, Run: nop})
	}
	ui.Menu(p, "§d§f§x§r probe", "", opts...)
}
