package skygear

import (
	"slices"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/formicon"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// windowKindEnum names a machine kind that has a window (/sgdebug ui).
type windowKindEnum string

func (windowKindEnum) Type() string                { return "sg_window" }
func (windowKindEnum) Options(cmd.Source) []string { return windowKinds() }

// windowKinds returns the machine kinds whose use opens a machine screen.
func windowKinds() []string {
	var out []string
	for _, k := range machine.Order {
		if d := machine.Kinds[k]; d.Marker != "" && d.Click == nil && k != machine.KindSieve && k != machine.KindTinker {
			out = append(out, k)
		}
	}
	return out
}

// debugUI opens a machine screen without placing the machine, filled with
// sample items, a half-done progress and a half-full energy gauge, so every
// screen layout can be checked in the client in a row.
type debugUI struct {
	adminBase
	Sub  cmd.SubCommand               `cmd:"ui"`
	Kind cmd.Optional[windowKindEnum] `cmd:"kind"`
}

func (c debugUI) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, s, ok := c.session(src, o, tx)
	if !ok {
		return
	}
	if k, ok := c.Kind.Load(); ok {
		if !slices.Contains(windowKinds(), string(k)) {
			o.Errorf("no window: %s", k)
			return
		}
		previewLater(s.Session, p, string(k))
		return
	}
	showWindowPicker(s.Session, p)
}

// showWindowPicker lists every machine screen; a button opens its preview.
func showWindowPicker(s *core.Session, p *player.Player) {
	var opts []ui.Option
	for _, k := range windowKinds() {
		opts = append(opts, ui.Option{Label: naming.ItemName(i18n.Of(p), machine.Item(k)) + "\n§8" + k, Icon: formicon.Of(machine.Item(k)), Run: func(p *player.Player, _ *world.Tx) {
			previewLater(s, p, k)
		}})
	}
	ui.Menu(p, "/sgdebug ui", "", opts...)
}

// previewDelay is how long the preview waits before opening: the client
// drops a screen opened while the chat screen or a form is still closing.
const previewDelay = 500 * time.Millisecond

// previewLater opens the preview of kind once the screen p used to ask for it
// is gone.
func previewLater(s *core.Session, p *player.Player, kind string) {
	p.H().DoAfter(previewDelay, func(tx *world.Tx, e world.Entity) {
		previewWindow(s, tx, e.(*player.Player), kind)
	})
}

// previewWindow opens a sample screen of the machine kind, as if it stood
// under the world.
func previewWindow(s *core.Session, tx *world.Tx, p *player.Player, kind string) {
	st := machine.New(kind, "", "").State
	inv := st.Inv()
	for i, slot := range st.Def.Input {
		_ = inv.SetItem(slot, item.NewStack(block.Cobblestone{}, 16+i))
	}
	for _, slot := range st.Def.Charge {
		_ = inv.SetItem(slot, item.NewStack(sgitem.Essence{Mob: sgitem.MobKinds[0]}, 4))
	}
	for i, slot := range st.Def.Output {
		_ = inv.SetItem(slot, item.NewStack(block.Gravel{}, 8+i))
	}
	st.Progress = st.CycleLen() / 2
	if st.Def.EnergyCap > 0 {
		st.Energy = st.EnergyCap() / 2
	}
	st.FluidAmt = st.Def.FluidCap / 2
	pos := cube.PosFromVec3(p.Position()).Sub(cube.Pos{0, 512, 0})
	machine.ShowPreview(s, p, tx, pos, kind, st)
}
