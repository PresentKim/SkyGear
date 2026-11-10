package machine

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
)

// The machines that work on an area around them (the harvester, the item
// vacuum, the tree feller) take a range on their screen, up to what their
// tier gives, and show that area in the world for a while as a see-through
// box (RangeBox).

// rangeShowTicks is how long the box shows after it is turned on.
const rangeShowTicks = 60 * 20

// rangeKind is the kind of the machine st when it works on an area, and
// false otherwise.
func (st *State) rangeKind() (string, bool) {
	for kind := range tierRanges {
		if Kinds[kind].ID == st.Def.ID {
			return kind, true
		}
	}
	return "", false
}

// rangeHeight is how far below and above the machine its area reaches.
func rangeHeight(kind string) (down, up int) {
	switch kind {
	case KindVacuum:
		return vacuumDown, vacuumUp
	case KindFeller:
		return fellerDown, fellerUp
	}
	return 0, 0
}

// rangeBox is the area of the machine of kind st at pos, from its first
// block to past its last, moved by its offset.
func (st *State) rangeBox(kind string, pos cube.Pos) (lo, hi mgl64.Vec3) {
	a, c := st.Area(kind), st.Centre(kind, pos)
	lo = c.Vec3().Sub(mgl64.Vec3{float64(a.X), float64(a.Down), float64(a.Z)})
	hi = c.Vec3().Add(mgl64.Vec3{float64(a.X + 1), float64(a.Up + 1), float64(a.Z + 1)})
	return lo, hi
}

// rangeToggled reports whether the box is turned on by its button.
func (st *State) rangeToggled() bool { return st.rangeUntil > 0 && st.curTick <= st.rangeUntil }

// rangeShown reports whether the box over the machine's area shows: turned
// on by its button, or a player sneaks and looks at the machine.
func (st *State) rangeShown() bool { return st.rangeToggled() || st.curTick <= st.peekUntil }

// peekTicks is how long the box stays after a sneaking player looked at the
// machine; looking again keeps it.
const peekTicks = 30

// PeekRange shows the box over the area of the machine at pos for a moment
// to a player who sneaks and looks at it (look.go).
func PeekRange(tx *world.Tx, pos cube.Pos, b world.Block) {
	if m, ok := b.(Machine); ok && m.State != nil {
		if _, ok := m.State.rangeKind(); ok {
			m.State.peekUntil = m.State.curTick + peekTicks
		}
	}
}

// boxVariant is the box from lo to hi as the range box entity's variant:
// width, height and depth in blocks, 7 bits each, then how many blocks the
// box's bottom is under the machine, plus dropBias so a box moved above the
// machine fits too (entity/range_box.entity.json reads them back). The entity
// stands just above the machine, in air: the client lights an entity by the
// block it stands in, and one in a solid block draws black.
func boxVariant(lo, hi mgl64.Vec3, down int) int32 {
	size := hi.Sub(lo)
	return int32(size[0]) | int32(size[1])<<7 | int32(size[2])<<14 | int32(down+dropBias)<<21
}

// dropBias is added to the drop of the box's bottom in its variant, which has
// ten bits left for it.
const dropBias = 512

// showRange keeps the range box over the machine's area while its outline
// is on, and takes it away when it goes off.
func (st *State) showRange(tx *world.Tx, pos cube.Pos) {
	kind, ok := st.rangeKind()
	if !ok || (st.rangeEntity == nil && !st.rangeShown()) {
		return
	}
	lo, hi := st.rangeBox(kind, pos)
	size := boxVariant(lo, hi, pos[1]-int(lo[1]))
	if st.rangeShown() && st.rangeEntity != nil && st.rangeSize == size {
		return
	}
	if st.rangeEntity != nil {
		if e, ok := st.rangeEntity.Entity(tx); ok {
			_ = e.(*RangeBox).Close()
		}
		st.rangeEntity = nil
	}
	if !st.rangeShown() {
		return
	}
	at := mgl64.Vec3{(lo[0] + hi[0]) / 2, float64(pos[1]) + 1.02, (lo[2] + hi[2]) / 2}
	st.rangeEntity = world.EntitySpawnOpts{Position: at}.New(RangeBoxType, rangeBoxConfig{Size: size, Machine: pos})
	st.rangeSize = size
	tx.AddEntity(st.rangeEntity)
}

// RangeBox is the see-through box over the area of a machine whose outline
// is on. It closes itself once the machine at its place no longer holds it.
type RangeBox struct {
	tx     *world.Tx
	Handle *world.EntityHandle
	Data   *world.EntityData
}

type rangeBoxData struct {
	Size    int32
	Machine cube.Pos
}

type rangeBoxConfig rangeBoxData

func (c rangeBoxConfig) Apply(data *world.EntityData) {
	d := rangeBoxData(c)
	data.Data = &d
}

func (b *RangeBox) H() *world.EntityHandle { return b.Handle }

func (b *RangeBox) Position() mgl64.Vec3 { return b.Data.Pos }

func (b *RangeBox) Rotation() cube.Rotation { return b.Data.Rot }

func (b *RangeBox) Variant() int32 { return b.Data.Data.(*rangeBoxData).Size }

func (b *RangeBox) Tick(tx *world.Tx, _ int64) {
	m, ok := tx.Block(b.Data.Data.(*rangeBoxData).Machine).(Machine)
	if !ok || m.State == nil || m.State.rangeEntity != b.Handle {
		_ = b.Close()
	}
}

// Close takes the box out of the world before closing its handle.
func (b *RangeBox) Close() error {
	b.tx.RemoveEntity(b)
	return b.Handle.Close()
}

// RangeBoxType is the world.EntityType of RangeBox.
var RangeBoxType rangeBoxType

type rangeBoxType struct{}

func (rangeBoxType) Open(tx *world.Tx, handle *world.EntityHandle, data *world.EntityData) world.Entity {
	if data.Data == nil {
		data.Data = &rangeBoxData{}
	}
	return &RangeBox{tx: tx, Handle: handle, Data: data}
}

func (rangeBoxType) EncodeEntity() string { return "skygear:range_box" }

func (rangeBoxType) Transient() bool { return true }

func (rangeBoxType) BBox(world.Entity) cube.BBox { return cube.BBox{} }

func (rangeBoxType) DecodeNBT(_ map[string]any, d *world.EntityData) { d.Data = &rangeBoxData{} }

func (rangeBoxType) EncodeNBT(*world.EntityData) map[string]any { return nil }

// offsetLine is how far the machine's area is moved: "기준 (+2, 0, -1)".
func (st *State) offsetLine(lang, kind string) string {
	o := st.Offset(kind)
	return fmt.Sprintf(i18n.T(lang, "machine.range.offset"), signed(o[0]), signed(o[1]), signed(o[2]))
}

// signed is v with its sign, "+2", "0" or "-1".
func signed(v int) string {
	if v > 0 {
		return fmt.Sprintf("+%d", v)
	}
	return fmt.Sprint(v)
}

// sizeLine is the size of the machine's area, as JE Item Collectors shows
// it: "범위 (11×5×7)".
func (st *State) sizeLine(lang, kind string) string {
	a := st.Area(kind)
	return fmt.Sprintf(i18n.T(lang, "machine.range.size"), 2*a.X+1, a.Down+a.Up+1, 2*a.Z+1)
}

// toggleRange turns the box over the machine's area on for a while, or off.
func (st *State) toggleRange() {
	if st.rangeToggled() {
		st.rangeUntil = 0
	} else {
		st.rangeUntil = st.curTick + rangeShowTicks
	}
}

// rangeViewOption is the button that turns the box on or off, then runs
// again.
func (st *State) rangeViewOption(lang string, again func(p *player.Player, tx *world.Tx)) ui.Option {
	key := "machine.range.show"
	if st.rangeToggled() {
		key = "machine.range.hide"
	}
	return ui.Option{Label: i18n.T(lang, key), Icon: "textures/ui/sg/icon_cube", Run: func(p *player.Player, tx *world.Tx) {
		st.toggleRange()
		again(p, tx)
	}}
}

// rangeTools are the machine screen's buttons of a machine with a range:
// the range screen, and the box on or off.
func (st *State) rangeTools(lang string, again func(p *player.Player, tx *world.Tx)) []ui.Option {
	kind, ok := st.rangeKind()
	if !ok {
		return nil
	}
	return []ui.Option{
		{Label: i18n.T(lang, "machine.range.title") + "\n§7" + st.sizeLine(lang, kind), Icon: "textures/ui/sg/menu_stats", Run: func(p *player.Player, _ *world.Tx) {
			st.showRangeScreen(p, kind, again)
		}},
		st.rangeViewOption(lang, again),
	}
}

// rangeAxis is one axis of the range screen: its name, the value set, the
// least and most it may be, and how a value is kept.
type rangeAxis struct {
	name           string
	have, low, top int
	show           string
	keep           func(v int)
}

// showRangeScreen shows p the range screen of the machine of kind, as JE
// Item Collectors lays it out: the size, an up arrow, "x: 5" and a down
// arrow for each axis, and the box button. A side button turns the screen
// to the offset of the area, the same arrows moving its centre. back goes
// to the machine.
func (st *State) showRangeScreen(p *player.Player, kind string, back func(p *player.Player, tx *world.Tx)) {
	lang := i18n.Of(p)
	again := func(p *player.Player, _ *world.Tx) { st.showRangeScreen(p, kind, back) }
	axes, line := st.reachAxes(kind), st.sizeLine(lang, kind)
	pageKey := "machine.range.to_offset"
	if st.offsetPage {
		axes, line = st.offsetAxes(kind), st.offsetLine(lang, kind)+"  §7"+line
		pageKey = "machine.range.to_reach"
	}
	var steppers []ui.Stepper
	for _, ax := range axes {
		set := func(v int) func(p *player.Player, tx *world.Tx) {
			if v < ax.low || v > ax.top {
				return nil
			}
			return func(p *player.Player, tx *world.Tx) {
				ax.keep(v)
				again(p, tx)
			}
		}
		up := fmt.Sprintf(i18n.T(lang, "machine.range.wider"), ax.top)
		down := i18n.T(lang, "machine.range.narrower")
		if st.offsetPage {
			up = fmt.Sprintf(i18n.T(lang, "machine.range.shift_up"), ax.top)
			down = fmt.Sprintf(i18n.T(lang, "machine.range.shift_down"), ax.low)
		}
		steppers = append(steppers, ui.Stepper{
			Label: ax.name, Value: ax.show,
			Up:   ui.Option{Label: up, Run: set(ax.have + 1)},
			Down: ui.Option{Label: down, Run: set(ax.have - 1)},
		})
	}
	page := ui.Option{Label: i18n.T(lang, pageKey), Icon: "textures/ui/sg/menu_expand", Run: func(p *player.Player, tx *world.Tx) {
		st.offsetPage = !st.offsetPage
		again(p, tx)
	}}
	ui.ShowSteppers(p, ui.StepperScreen{
		Title:    st.Def.Title(lang),
		Line:     line,
		Steppers: steppers,
		Side:     []ui.Option{st.rangeViewOption(lang, again), page},
		Options:  []ui.Option{ui.Back(i18n.T(lang, "machine.range.back"), back)},
	})
}

// reachAxes are the range screen's axes of the reach of the machine of kind:
// 1 up to what its tier gives, which is kept as 0 (all).
func (st *State) reachAxes(kind string) []rangeAxis {
	a, most := st.Area(kind), st.MaxArea(kind)
	axis := func(name string, have, top int, set *int) rangeAxis {
		return rangeAxis{name: name, have: have, low: 1, top: top, show: fmt.Sprint(have), keep: func(v int) {
			*set = v
			if v == top {
				*set = 0
			}
		}}
	}
	axes := []rangeAxis{axis("x", a.X, most.X, &st.RangeX)}
	if top := max(most.Down, most.Up); top > 0 {
		axes = append(axes, axis("y", max(a.Down, a.Up), top, &st.RangeY))
	}
	return append(axes, axis("z", a.Z, most.Z, &st.RangeZ))
}

// offsetAxes are the range screen's axes of the offset of the area of the
// machine of kind, each from -MaxOffset to +MaxOffset.
func (st *State) offsetAxes(kind string) []rangeAxis {
	o, most := st.Offset(kind), st.MaxOffset(kind)
	axis := func(name string, i int, set *int) rangeAxis {
		return rangeAxis{name: name, have: o[i], low: -most[i], top: most[i], show: signed(o[i]), keep: func(v int) { *set = v }}
	}
	return []rangeAxis{axis("x", 0, &st.OffsetX), axis("y", 1, &st.OffsetY), axis("z", 2, &st.OffsetZ)}
}
