package dragonfactory

import (
	"slices"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"

	"github.com/presentkim/dragonfactory/common/ui"
)

// windowKindEnum names a machine kind that has a window (/dfdebug ui).
type windowKindEnum string

func (windowKindEnum) Type() string                { return "df_window" }
func (windowKindEnum) Options(cmd.Source) []string { return windowKinds() }

// windowKinds returns the machine kinds whose use opens a machine window.
func windowKinds() []string {
	var out []string
	for _, k := range machineOrder {
		// the laboratory's use opens its screen (labui.go), not the window
		if machineKinds[k].marker != "" && k != KindSieve && k != KindLab {
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
		opts = append(opts, ui.Option{Label: itemName(machineItem(k)) + "\n§8" + k, Icon: iconOf(machineItem(k)), Run: func(p *player.Player, _ *world.Tx) {
			previewLater(p, k)
		}})
	}
	ui.Menu(p, "/dfdebug ui", "", opts...)
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
	m := newMachine(kind, "", "")
	st := m.st
	inv := st.inv()
	for i, s := range st.kind.input {
		_ = inv.SetItem(s, item.NewStack(block.Cobblestone{}, 16+i))
	}
	for _, s := range st.kind.charge {
		_ = inv.SetItem(s, item.NewStack(Essence{Mob: mobKinds[0]}, 4))
	}
	for i, s := range st.kind.output {
		_ = inv.SetItem(s, item.NewStack(block.Gravel{}, 8+i))
	}
	st.progress = st.cycleLen() / 2
	st.showProgress()
	if st.kind.energyCap > 0 {
		st.energy = st.energyCap() / 2
		st.showEnergy()
	}
	st.storage.Open(p, nil)
}
