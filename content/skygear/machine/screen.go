package machine

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/wintext"
)

// The machine window is the machine's own chest (State.storage), which the
// skygear pack's sg_machine.json draws by the title marker: the client moves
// the items itself and machineHandler keeps each slot to its role.

// rangeSlot is the window slot of the range button of a machine with a
// range, left of the info icon in the window's control bar. Taking it opens
// the range screen.
const rangeSlot = 25

// ShowMachine shows p the window of the machine at pos.
func ShowMachine(_ *core.Session, p *player.Player, _ *world.Tx, pos cube.Pos, st *State) {
	st.pos = pos
	st.OpenWindow(p)
}

// ShowPreview shows p the screen of a machine of kind with state st that is
// not in the world, as if it stood at pos (/sgdebug ui).
func ShowPreview(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, kind string, st *State) {
	if kind == KindLab {
		showLab(s, p, tx, pos, st)
		return
	}
	ShowMachine(s, p, tx, pos, st)
}

// showRangeButton puts the range button in the window of a machine with a
// range: its name and the size of the area.
func (st *State) showRangeButton() {
	kind, ok := st.rangeKind()
	if !ok {
		return
	}
	a := st.Area(kind)
	name := wintext.Text{wintext.Raw("§r"), wintext.Key("machine.range.title")}
	size := wintext.Text{wintext.Raw("§7"), wintext.Key("machine.range.size", 2*a.X+1, a.Down+a.Up+1, 2*a.Z+1)}
	_ = st.Inv().SetItem(rangeSlot, wintext.With(item.NewStack(item.Spyglass{}, 1), name, size))
}

// openRange shows p the range screen of the machine, whose back button opens
// the machine's window again.
func (st *State) openRange(p *player.Player) {
	kind, ok := st.rangeKind()
	if !ok {
		return
	}
	st.showRangeScreen(p, kind, func(p *player.Player, _ *world.Tx) { st.openFromMenu(p) })
}

// openFromMenu opens the machine's window from a button of the menu window,
// once the menu window has handled the press and closed: the window closing
// cancels a window still waiting to open.
func (st *State) openFromMenu(p *player.Player) {
	p.H().Do(func(_ *world.Tx, e world.Entity) { st.OpenWindow(e.(*player.Player)) })
}
