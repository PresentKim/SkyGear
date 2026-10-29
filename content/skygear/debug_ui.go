package skygear

import (
	"slices"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/formicon"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// windowKindEnum names a machine kind that has a window (/sgdebug ui).
type windowKindEnum string

func (windowKindEnum) Type() string                { return "sg_window" }
func (windowKindEnum) Options(cmd.Source) []string { return windowKinds() }

// windowKinds returns the machine kinds whose use opens a machine window.
func windowKinds() []string {
	var out []string
	for _, k := range machine.Order {
		// the laboratory's use opens its screen (labui.go), not the window
		if machine.Kinds[k].Marker != "" && k != machine.KindSieve && k != machine.KindLab {
			out = append(out, k)
		}
	}
	return out
}

// debugUI opens a machine window without placing the machine, filled with
// sample items, a half-done progress and a half-full energy gauge, so every
// window layout can be checked in the client in a row.
type debugUI struct {
	adminBase
	Sub  cmd.SubCommand               `cmd:"ui"`
	Kind cmd.Optional[windowKindEnum] `cmd:"kind"`
}

func (c debugUI) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, _, ok := c.session(src, o, tx)
	if !ok {
		return
	}
	if k, ok := c.Kind.Load(); ok {
		if !slices.Contains(windowKinds(), string(k)) {
			o.Errorf("no window: %s", k)
			return
		}
		previewLater(p, string(k))
		return
	}
	showWindowPicker(p)
}

// showWindowPicker lists every machine window; a button opens its preview.
func showWindowPicker(p *player.Player) {
	var opts []ui.Option
	for _, k := range windowKinds() {
		opts = append(opts, ui.Option{Label: naming.ItemName(i18n.Of(p), machine.Item(k)) + "\n§8" + k, Icon: formicon.Of(machine.Item(k)), Run: func(p *player.Player, _ *world.Tx) {
			previewLater(p, k)
		}})
	}
	ui.Menu(p, "/sgdebug ui", "", opts...)
}

// previewDelay is how long the preview waits before opening: the client
// drops a container opened while the chat screen or a form is still closing.
const previewDelay = 500 * time.Millisecond

// previewLater opens the preview of kind once the screen p used to ask for it
// is gone.
func previewLater(p *player.Player, kind string) {
	p.H().DoAfter(previewDelay, func(_ *world.Tx, e world.Entity) {
		previewWindow(e.(*player.Player), kind)
	})
}

// previewWindow opens a sample window of the machine kind.
func previewWindow(p *player.Player, kind string) {
	m := machine.New(kind, "", "")
	st := m.State
	inv := st.Inv()
	for i, s := range st.Def.Input {
		_ = inv.SetItem(s, item.NewStack(block.Cobblestone{}, 16+i))
	}
	for _, s := range st.Def.Charge {
		_ = inv.SetItem(s, item.NewStack(sgitem.Essence{Mob: sgitem.MobKinds[0]}, 4))
	}
	for i, s := range st.Def.Output {
		_ = inv.SetItem(s, item.NewStack(block.Gravel{}, 8+i))
	}
	st.Progress = st.CycleLen() / 2
	st.ShowProgress()
	if st.Def.EnergyCap > 0 {
		st.Energy = st.EnergyCap() / 2
		st.ShowEnergy()
	}
	st.OpenWindow(p)
}
