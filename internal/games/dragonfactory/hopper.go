package dragonfactory

import (
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/internal/itemio"
)

// The game's own hoppers (docs/design/02-dragonfactory.md §3.8):
//   - the wood hopper: one slot and very slow (an item every four seconds),
//     for the first bulk work before iron;
//   - reverse hoppers (wood and iron): the same hoppers turned upside down,
//     which take from the block below and give to the block above, only
//     that way. Crafting a hopper alone gives its reverse and the reverse
//     alone gives the hopper back.
// The iron hopper is Dragonfly's; it moves items like these (stepHopper).
// All of them move items through internal/itemio.

// hopperKind describes a kind of hopper of the game.
type hopperKind struct {
	id, name string
	// slots is the inventory size, interval the ticks between two moves.
	slots, interval int
	// reverse hoppers take from below and give upwards.
	reverse bool
	// texture is the pack texture (textures/blocks/<texture>.png), wood
	// says which tool mines it.
	texture string
	wood    bool
	hash    uint64
}

var hopperKinds = map[string]*hopperKind{
	"wood_hopper":         {id: "wood_hopper", name: "나무 호퍼", slots: 1, interval: 80, texture: "wood_hopper", wood: true},
	"wood_hopper_reverse": {id: "wood_hopper_reverse", name: "나무 역방향 호퍼", slots: 1, interval: 80, reverse: true, texture: "wood_hopper", wood: true},
	"iron_hopper_reverse": {id: "iron_hopper_reverse", name: "역방향 호퍼", slots: 5, interval: 8, reverse: true, texture: "iron_hopper"},
}

// hopperOrder is the registration and display order.
var hopperOrder = []string{"wood_hopper", "wood_hopper_reverse", "iron_hopper_reverse"}

// hopperFacings are the sides a forward hopper can face, in state order.
var hopperFacings = []cube.Face{cube.FaceDown, cube.FaceNorth, cube.FaceEast, cube.FaceSouth, cube.FaceWest}

var hopperFacingNames = map[cube.Face]string{cube.FaceDown: "down", cube.FaceNorth: "north", cube.FaceEast: "east", cube.FaceSouth: "south", cube.FaceWest: "west", cube.FaceUp: "up"}

// hopperFacingState is the one block state of a forward hopper.
const hopperFacingState = "dragonfactory:hopper_facing"

// FlowHopper is a hopper of the game. Facing is the side a forward hopper
// gives to (Down or a horizontal side); reverse hoppers always face Up.
type FlowHopper struct {
	Kind   string
	Facing cube.Face
	inv    *inventory.Inventory
}

func newFlowHopper(kind string, facing cube.Face) FlowHopper {
	k := hopperKinds[kind]
	if k.reverse {
		facing = cube.FaceUp
	}
	return FlowHopper{Kind: kind, Facing: facing, inv: inventory.New(k.slots, nil)}
}

func (h FlowHopper) kind() *hopperKind { return hopperKinds[h.Kind] }

func (h FlowHopper) EncodeItem() (string, int16) { return "dragonfactory:" + h.Kind, 0 }
func (h FlowHopper) EncodeBlock() (string, map[string]any) {
	if h.kind().reverse {
		return "dragonfactory:" + h.Kind, nil
	}
	return "dragonfactory:" + h.Kind, map[string]any{hopperFacingState: hopperFacingNames[h.Facing]}
}
func (h FlowHopper) Hash() (uint64, uint64) {
	for i, f := range hopperFacings {
		if f == h.Facing {
			return h.kind().hash, uint64(i)
		}
	}
	return h.kind().hash, 0
}
func (FlowHopper) Model() world.BlockModel { return model.Solid{} }

// LightDiffusionLevel lets light through like the vanilla hopper. A chest
// opens only when the block above it diffuses light (Chest.Activate), so a
// hopper above a chest no longer blocks it.
func (FlowHopper) LightDiffusionLevel() uint8 { return 0 }
func (h FlowHopper) Name() string             { return h.kind().name }
func (FlowHopper) Category() category.Category {
	return category.Construction()
}

// hopperGeometry is the geometry of a facing (reverse hoppers are Up).
//
// The client's block geometry has its x axis mirrored against the world's
// (the east and west models showed the spout on the wrong side in game), so
// east uses the model drawn at x < 0 and west the one at x > 0.
func hopperGeometry(f cube.Face) string {
	switch f {
	case cube.FaceEast:
		f = cube.FaceWest
	case cube.FaceWest:
		f = cube.FaceEast
	}
	return "geometry.dragonfactory_flow_hopper_" + hopperFacingNames[f]
}

func (h FlowHopper) materials() map[string]customblock.Material {
	return map[string]customblock.Material{"*": alphaMaterial("dragonfactory_" + h.kind().texture)}
}

func (h FlowHopper) Properties() customblock.Properties {
	return customblock.Properties{Geometry: hopperGeometry(h.Facing), Textures: h.materials()}
}

// States implements block.Permutable: forward hoppers have their facing.
func (h FlowHopper) States() map[string][]any {
	if h.kind().reverse {
		return nil
	}
	var values []any
	for _, f := range hopperFacings {
		values = append(values, hopperFacingNames[f])
	}
	return map[string][]any{hopperFacingState: values}
}

// Permutations implements block.Permutable: one geometry per facing.
func (h FlowHopper) Permutations() []customblock.Permutation {
	if h.kind().reverse {
		return nil
	}
	var out []customblock.Permutation
	for _, f := range hopperFacings[1:] {
		out = append(out, customblock.Permutation{
			Condition:  fmt.Sprintf("q.block_state('%s') == '%s'", hopperFacingState, hopperFacingNames[f]),
			Properties: customblock.Properties{Geometry: hopperGeometry(f), Textures: h.materials()},
		})
	}
	return out
}

func (h FlowHopper) BreakInfo() block.BreakInfo {
	return block.BreakInfo{
		Hardness: 2, BlastResistance: 4,
		Harvestable: func(item.Tool) bool { return true },
		Effective:   customBlockEffective,
		Drops: func(item.Tool, []item.Enchantment) []item.Stack {
			drops := []item.Stack{item.NewStack(newFlowHopper(h.Kind, cube.FaceDown), 1)}
			if h.inv != nil {
				for _, st := range h.inv.Items() {
					if !st.Empty() {
						drops = append(drops, st)
					}
				}
			}
			return drops
		},
	}
}

func (h FlowHopper) EncodeNBT() map[string]any {
	data := map[string]any{"id": "DragonFactoryFlowHopper"}
	var items []any
	if h.inv != nil {
		for slot, st := range h.inv.Slots() {
			if !st.Empty() {
				nbt := item.WriteNBT(st, true)
				nbt["Slot"] = byte(slot)
				items = append(items, nbt)
			}
		}
	}
	data["Items"] = items
	return data
}

func (h FlowHopper) DecodeNBT(data map[string]any) any {
	n := newFlowHopper(h.Kind, h.Facing)
	items, _ := data["Items"].([]any)
	for _, v := range items {
		if nbt, ok := v.(map[string]any); ok {
			slot, _ := nbt["Slot"].(byte)
			if int(slot) < n.inv.Size() {
				_ = n.inv.SetItem(int(slot), item.ReadNBT(nbt, nil))
			}
		}
	}
	return n
}

// sides are where a hopper gives to and takes from.
func (h FlowHopper) sides() (out, in cube.Face) {
	if h.kind().reverse {
		return cube.FaceUp, cube.FaceDown
	}
	return h.Facing, cube.FaceUp
}

// Tick moves an item every interval ticks.
func (h FlowHopper) Tick(currentTick int64, pos cube.Pos, tx *world.Tx) {
	if h.inv == nil {
		// Loaded without its items: start empty.
		tx.SetBlock(pos, newFlowHopper(h.Kind, h.Facing), nil)
		return
	}
	if currentTick%int64(h.kind().interval) != 0 {
		return
	}
	out, in := h.sides()
	stepHopper(tx, pos, h.inv, out, in)
}

// InsertItems implements itemio.Inserter: any face fills the hopper.
func (h FlowHopper) InsertItems(_ *world.Tx, _ cube.Pos, _ cube.Face, st item.Stack) int {
	if h.inv == nil {
		return 0
	}
	n, _ := h.inv.AddItem(st)
	return n
}

// PeekItems implements itemio.Extractor: any face gives any item.
func (h FlowHopper) PeekItems(_ *world.Tx, _ cube.Pos, _ cube.Face, want func(item.Stack) bool) item.Stack {
	if h.inv == nil {
		return item.Stack{}
	}
	for _, st := range h.inv.Slots() {
		if !st.Empty() && (want == nil || want(st)) {
			return st
		}
	}
	return item.Stack{}
}

// ExtractItems implements itemio.Extractor.
func (h FlowHopper) ExtractItems(_ *world.Tx, _ cube.Pos, _ cube.Face, st item.Stack, n int) item.Stack {
	if h.inv == nil {
		return item.Stack{}
	}
	got := 0
	var out item.Stack
	for slot, cur := range h.inv.Slots() {
		if got >= n || cur.Empty() || !cur.Comparable(st) {
			continue
		}
		take := min(n-got, cur.Count())
		_ = h.inv.SetItem(slot, cur.Grow(-take))
		got += take
		out = cur.Grow(got - cur.Count())
	}
	return out
}

// UseOnBlock places the hopper like the vanilla one: facing the clicked
// block (down when placed on top of one); reverse hoppers always face up.
func (h FlowHopper) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) bool {
	target := pos
	if !hopperReplaceable(tx, target, h) {
		target = pos.Side(face)
		if !hopperReplaceable(tx, target, h) {
			return false
		}
	}
	facing := cube.FaceDown
	if face != cube.FaceUp && face != cube.FaceDown {
		facing = face.Opposite()
	}
	placed := newFlowHopper(h.Kind, facing)
	if placer, ok := user.(block.Placer); ok {
		placer.PlaceBlock(target, placed, ctx)
	} else {
		tx.SetBlock(target, placed, nil)
		ctx.SubtractFromCount(1)
	}
	return ctx.CountSub > 0
}

// hopperReplaceable reports whether a hopper can be placed at pos.
func hopperReplaceable(tx *world.Tx, pos cube.Pos, with world.Block) bool {
	if pos.OutOfBounds(tx.Range()) {
		return false
	}
	b := tx.Block(pos)
	if r, ok := b.(block.Replaceable); ok && r.ReplaceableBy(with) {
		return true
	}
	_, air := b.(block.Air)
	return air
}

// Activate shows what the hopper holds (it has no window).
func (h FlowHopper) Activate(_ cube.Pos, _ cube.Face, _ *world.Tx, u item.User, _ *item.UseContext) bool {
	p, ok := u.(*player.Player)
	if !ok {
		return false
	}
	p.SendTip(h.contentsText())
	return true
}

// contentsText lists what the hopper holds.
func (h FlowHopper) contentsText() string {
	var parts []string
	if h.inv != nil {
		for _, st := range h.inv.Items() {
			if !st.Empty() {
				parts = append(parts, fmt.Sprintf("%s ×%d", itemName(st), st.Count()))
			}
		}
	}
	if len(parts) == 0 {
		return "§7" + h.Name() + ": 비어있음"
	}
	return "§f" + h.Name() + ": §e" + strings.Join(parts, "§7, §e")
}

// hopperLines are the lines of a hopper's WAILA panel: what it holds, and
// how fast it moves.
func hopperLines(h FlowHopper) []string {
	var lines []string
	if h.inv != nil {
		for _, st := range h.inv.Items() {
			if !st.Empty() {
				lines = append(lines, fmt.Sprintf("§f%s §e×%d", itemName(st), st.Count()))
			}
		}
	}
	if len(lines) == 0 {
		lines = []string{"§7비어있음"}
	}
	return append(lines, fmt.Sprintf("§7%.1f초에 1개", float64(h.kind().interval)/20))
}

var _ itemio.Inserter = FlowHopper{}
var _ itemio.Extractor = FlowHopper{}

func init() {
	for _, id := range hopperOrder {
		k := hopperKinds[id]
		k.hash = block.NextHash()
		if k.reverse {
			world.RegisterBlock(FlowHopper{Kind: id, Facing: cube.FaceUp})
		} else {
			for _, f := range hopperFacings {
				world.RegisterBlock(FlowHopper{Kind: id, Facing: f})
			}
		}
		world.RegisterItem(FlowHopper{Kind: id, Facing: cube.FaceDown})
	}
}
