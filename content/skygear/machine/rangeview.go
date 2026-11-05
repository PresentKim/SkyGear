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
// block to past its last.
func (st *State) rangeBox(kind string, pos cube.Pos) (lo, hi mgl64.Vec3) {
	r := st.Reach(kind)
	down, up := rangeHeight(kind)
	lo = pos.Vec3().Sub(mgl64.Vec3{float64(r), float64(down), float64(r)})
	hi = pos.Vec3().Add(mgl64.Vec3{float64(r + 1), float64(up + 1), float64(r + 1)})
	return lo, hi
}

// rangeShown reports whether the outline of the machine's area is on.
func (st *State) rangeShown() bool { return st.rangeUntil > 0 && st.curTick <= st.rangeUntil }

// boxVariant is the box from lo to hi as the range box entity's variant:
// width, height and depth in blocks, 7 bits each, then how many blocks the
// box's bottom is under the machine (entity/range_box.entity.json reads them
// back). The entity stands just above the machine, in air: the client lights
// an entity by the block it stands in, and one in a solid block draws black.
func boxVariant(lo, hi mgl64.Vec3, down int) int32 {
	size := hi.Sub(lo)
	return int32(size[0]) | int32(size[1])<<7 | int32(size[2])<<14 | int32(down)<<21
}

// showRange keeps the range box over the machine's area while its outline
// is on, and takes it away when it goes off.
func (st *State) showRange(tx *world.Tx, pos cube.Pos) {
	kind, ok := st.rangeKind()
	if !ok || (st.rangeEntity == nil && !st.rangeShown()) {
		return
	}
	lo, hi := st.rangeBox(kind, pos)
	down, _ := rangeHeight(kind)
	size := boxVariant(lo, hi, down)
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

// rangeTools are the screen's buttons of a machine with a range: narrower,
// wider, and the outline on or off.
func (st *State) rangeTools(lang string, again func(p *player.Player, tx *world.Tx)) []ui.Option {
	kind, ok := st.rangeKind()
	if !ok {
		return nil
	}
	reach, most := st.Reach(kind), st.MaxReach(kind)
	set := func(r int) func(p *player.Player, tx *world.Tx) {
		return func(p *player.Player, tx *world.Tx) {
			st.Range = max(1, min(r, most))
			if st.Range == most {
				st.Range = 0
			}
			again(p, tx)
		}
	}
	size := fmt.Sprintf(i18n.T(lang, "machine.range.size"), 2*reach+1, 2*reach+1, reach, most)
	viewKey := "machine.range.show"
	if st.rangeShown() {
		viewKey = "machine.range.hide"
	}
	return []ui.Option{
		{Label: i18n.T(lang, "machine.range.wider") + "\n" + size, Icon: "textures/ui/sg/arrow_right", Run: set(reach + 1)},
		{Label: i18n.T(lang, "machine.range.narrower") + "\n" + size, Icon: "textures/ui/sg/arrow_left", Run: set(reach - 1)},
		{Label: i18n.T(lang, viewKey), Icon: "textures/ui/sg/icon_cube", Run: func(p *player.Player, tx *world.Tx) {
			if st.rangeShown() {
				st.rangeUntil = 0
			} else {
				st.rangeUntil = st.curTick + rangeShowTicks
			}
			again(p, tx)
		}},
	}
}
