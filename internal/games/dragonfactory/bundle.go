package dragonfactory

import (
	"fmt"
	"github.com/presentkim/dragonfactory/internal/fluidnet"
	"strconv"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/internal/energy"
	"github.com/presentkim/dragonfactory/internal/i18n"
	"github.com/presentkim/dragonfactory/internal/itemio"
	"github.com/presentkim/dragonfactory/internal/itemnet"
)

// fluidRates are the mB a fluid conduit of a tier moves a tick: EnderIO's
// fluid conduits (200, 1000 and 8000; it moves them every 5 ticks in portions of
// five times that).
var fluidRates = [...]int{0, 200, 1000, 8000}

func fluidRate(tier int) int { return fluidRates[max(1, min(tier, conduitTiers))] }

// Conduit bundles (docs/design/06-power-and-conduits.md §3.4): the block every
// conduit is placed as. It holds at most one conduit of each kind (energy, item,
// fluid; each of a tier, 0 for none), so a cable run can carry all, as in
// EnderIO. Placing a conduit item on a bundle that has room for its kind adds
// it; breaking the bundle drops everything in it.
//
// The block is a block entity with one block state and an invisible model: all
// its data (tiers, the sides each kind connects to, the item modes) is stored
// in its NBT, and the display entity of conduitvisual.go draws it. Each kind
// connects to its own neighbours, so a side can have the arm of one kind only.
// Item side modes (itemnet.Mode) belong to the item conduit; using a side
// with an empty hand while sneaking cycles it.

// ConduitBundle is the conduit block. Energy, Item and Fluid are the tiers of
// the conduits in it (0 for none), EConn, IConn and FConn the sides each of
// them connects to (bit i is conduitFaces[i]) and Modes the connection mode of
// each side for the item conduit (indexed by cube.Face).
type ConduitBundle struct {
	Energy, Item, Fluid int
	EConn, IConn, FConn int
	Modes               [6]itemnet.Mode
	// Extract and Insert are the channels (0 to 15) of each side of the item
	// conduit (itemnet.Channels).
	Extract, Insert [6]int
}

var bundleHash = block.NextHash()

func (ConduitBundle) EncodeBlock() (string, map[string]any) {
	return "dragonfactory:conduit_node", nil
}
func (ConduitBundle) Hash() (uint64, uint64)    { return bundleHash, 0 }
func (b ConduitBundle) Model() world.BlockModel { return conduitModel{b.EConn | b.IConn | b.FConn} }
func (ConduitBundle) Name() string              { return i18n.T(i18n.Default, "name.ConduitBundle") }

// bundleSelection is the space the conduits take: the cores of the three lanes
// (shifted 3 pixels either way, 3 pixels across) fit in 9 pixels, 3.5 to
// 12.5. It is what the client lets the player point at and the shape of the
// block (conduit_node.geo.json).
var bundleSelection = cube.Box(3.5/16, 3.5/16, 3.5/16, 12.5/16, 12.5/16, 12.5/16)

func (ConduitBundle) Properties() customblock.Properties {
	return customblock.Properties{
		Geometry:     "geometry.dragonfactory_conduit_node",
		Textures:     map[string]customblock.Material{"*": alphaMaterial("dragonfactory_conduit_node")},
		SelectionBox: bundleSelection,
		CollisionBox: bundleSelection,
	}
}

func (b ConduitBundle) BreakInfo() block.BreakInfo {
	return block.BreakInfo{
		Hardness: 1.5, BlastResistance: 3,
		Harvestable: func(item.Tool) bool { return true },
		Effective:   customBlockEffective,
		Drops: func(item.Tool, []item.Enchantment) []item.Stack {
			var out []item.Stack
			if b.Energy > 0 {
				out = append(out, item.NewStack(EnergyConduit{Tier: b.Energy}, 1))
			}
			if b.Item > 0 {
				out = append(out, item.NewStack(ItemConduit{Tier: b.Item}, 1))
			}
			if b.Fluid > 0 {
				out = append(out, item.NewStack(FluidConduit{Tier: b.Fluid}, 1))
			}
			return out
		},
	}
}

// EnergyTier implements energy.Conduit (0: no energy conduit in the bundle).
func (b ConduitBundle) EnergyTier() int { return b.Energy }

// ItemTier and ItemMode implement itemnet.Conduit (tier 0: no item conduit).
func (b ConduitBundle) ItemTier() int                     { return b.Item }
func (b ConduitBundle) ItemMode(f cube.Face) itemnet.Mode { return b.Modes[f] }

// ItemChannels implements itemnet.Conduit.
func (b ConduitBundle) ItemChannels(f cube.Face) (extract, insert int) {
	return b.Extract[f], b.Insert[f]
}

// EncodeNBT and DecodeNBT keep everything the bundle holds (and make it a
// block entity, which is what gets ticked).
func (b ConduitBundle) EncodeNBT() map[string]any {
	modes := make([]byte, len(b.Modes))
	for i, m := range b.Modes {
		modes[i] = '0' + byte(m)
	}
	channels := func(c [6]int) string {
		out := make([]byte, len(c))
		for i, v := range c {
			out[i] = "0123456789abcdef"[v&15]
		}
		return string(out)
	}
	return map[string]any{
		"Extract": channels(b.Extract), "Insert": channels(b.Insert),
		"id": "DragonFactoryConduitBundle", "Energy": int32(b.Energy), "Item": int32(b.Item), "Fluid": int32(b.Fluid),
		"EConn": int32(b.EConn), "IConn": int32(b.IConn), "FConn": int32(b.FConn), "Modes": string(modes),
	}
}

func (b ConduitBundle) DecodeNBT(data map[string]any) any {
	num := func(key string) int {
		n, _ := data[key].(int32)
		return int(n)
	}
	b.Energy, b.Item, b.Fluid = min(num("Energy"), conduitTiers), min(num("Item"), conduitTiers), min(num("Fluid"), conduitTiers)
	b.EConn, b.IConn, b.FConn = num("EConn")&63, num("IConn")&63, num("FConn")&63
	for key, dst := range map[string]*[6]int{"Extract": &b.Extract, "Insert": &b.Insert} {
		if v, ok := data[key].(string); ok && len(v) == 6 {
			for i := range dst {
				if n, err := strconv.ParseUint(v[i:i+1], 16, 8); err == nil {
					dst[i] = int(n)
				}
			}
		}
	}
	if modes, ok := data["Modes"].(string); ok && len(modes) == len(b.Modes) {
		for i := range b.Modes {
			b.Modes[i] = itemnet.Mode(modes[i]-'0') % 4
		}
	}
	return b
}

// joinsEnergy reports whether an energy conduit connects to b: another energy
// conduit or a block that gives or takes energy.
func joinsEnergy(b world.Block) bool {
	if _, ok := energy.IsConduit(b); ok {
		return true
	}
	src, snk := energy.RolesOf(b)
	return src != nil || snk != nil
}

// joinsItem reports whether the item conduit connects to the block other at
// at, on its side f: another item conduit, or a block that holds items, unless
// that side is off.
func (b ConduitBundle) joinsItem(tx *world.Tx, f cube.Face, at cube.Pos, other world.Block) bool {
	if _, ok := itemnet.IsConduit(other); ok {
		return true
	}
	if b.Modes[f] == itemnet.Off {
		return false
	}
	in, ex := itemio.IO(tx, at)
	return in != nil || ex != nil
}

// joinsFluid reports whether the fluid conduit connects to b: another fluid
// conduit, or a block that holds fluid (so far the barrel).
func joinsFluid(b world.Block) bool {
	if o, ok := b.(ConduitBundle); ok {
		return o.Fluid > 0
	}
	if h, ok := b.(fluidnet.Holder); ok {
		src, snk := h.FluidRoles()
		return src || snk
	}
	return false
}

// connections returns the sides the energy, item and fluid conduit connect to.
func (b ConduitBundle) connections(tx *world.Tx, pos cube.Pos) (e, i, f int) {
	if b.Energy > 0 {
		e = connectionsOf(tx, pos, func(_ cube.Face, _ cube.Pos, other world.Block) bool { return joinsEnergy(other) })
	}
	if b.Item > 0 {
		i = connectionsOf(tx, pos, func(f cube.Face, at cube.Pos, other world.Block) bool { return b.joinsItem(tx, f, at, other) })
	}
	if b.Fluid > 0 {
		f = connectionsOf(tx, pos, func(_ cube.Face, _ cube.Pos, other world.Block) bool { return joinsFluid(other) })
	}
	return e, i, f
}

// refresh sets the block to show the sides it connects to, and shows it.
func (b ConduitBundle) refresh(pos cube.Pos, tx *world.Tx) {
	e, i, f := b.connections(tx, pos)
	if e != b.EConn || i != b.IConn || f != b.FConn {
		b.EConn, b.IConn, b.FConn = e, i, f
		tx.SetBlock(pos, b, nil)
	}
	if s, ok := activeGame.sessionOf(tx.World()); ok {
		s.showConduit(tx, pos, b)
	}
}

// NeighbourUpdateTick keeps the arms up to date.
func (b ConduitBundle) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) { b.refresh(pos, tx) }

// Tick registers the bundle with the grids of its conduits every few ticks
// (and keeps its display entity there).
func (b ConduitBundle) Tick(currentTick int64, pos cube.Pos, tx *world.Tx) {
	if currentTick%4 != 0 {
		return
	}
	s, ok := activeGame.sessionOf(tx.World())
	if !ok {
		return
	}
	b.refresh(pos, tx)
	if b.Energy > 0 {
		s.energyGrid().AddConduit(pos)
	}
	if b.Item > 0 {
		s.itemGrid().AddConduit(pos)
	}
	if b.Fluid > 0 {
		s.fluidGrid().AddConduit(pos)
	}
}

// FluidTier implements fluidnet.Conduit.
func (b ConduitBundle) FluidTier() int { return b.Fluid }

var _ fluidnet.Conduit = ConduitBundle{}

// Activate opens the settings of the clicked side (sneaking, empty hand: with
// something in hand the handler does it) if that side has a connector plate
// (conduitplate.go).
func (b ConduitBundle) Activate(pos cube.Pos, face cube.Face, tx *world.Tx, u item.User, _ *item.UseContext) bool {
	p, ok := u.(*player.Player)
	if !ok || !p.Sneaking() {
		return false
	}
	if held, _ := p.HeldItems(); !held.Empty() {
		return false
	}
	s, ok := activeGame.sessionOf(tx.World())
	if !ok {
		return false
	}
	side := faceIndex(face)
	if len(plateKinds(tx, pos, b, side)) == 0 {
		return false
	}
	if !s.debounced(p) {
		s.showPlateForm(p, tx, pos, side)
	}
	return true
}

// bundleLines are the lines of a bundle's WAILA panel: the conduits in it
// and the mode of every side that has a block with items.
func bundleLines(tx *world.Tx, pos cube.Pos, b ConduitBundle) []string {
	var lines []string
	if b.Energy > 0 {
		lines = append(lines, i18n.T(i18n.Default, fmt.Sprintf("name.EnergyConduit.%d", b.Energy))+
			fmt.Sprintf(" §7%d RE/t", energy.RateOf(b.Energy)))
	}
	if b.Item > 0 {
		lines = append(lines, i18n.T(i18n.Default, fmt.Sprintf("name.ItemConduit.%d", b.Item))+
			" §7"+fmt.Sprintf(i18n.T(i18n.Default, "itemconduit.lines.rate"), formatRate(itemsPerTick(b.Item))))
	}
	if b.Fluid > 0 {
		lines = append(lines, i18n.T(i18n.Default, fmt.Sprintf("name.FluidConduit.%d", b.Fluid))+fmt.Sprintf(" §7%d mB/t", fluidRate(b.Fluid)))
	}
	return lines
}

// formatRate writes a rate without trailing zeros.
func formatRate(r float64) string { return strconv.FormatFloat(r, 'f', -1, 64) }

// itemsPerTick is how many items an item conduit of a tier moves a tick.
func itemsPerTick(tier int) float64 {
	return float64(itemnet.RateOf(tier)) / itemnet.StepTicks
}

// energyGrid and itemGrid are the grids of the island, made when first used.
func (s *session) energyGrid() *energy.Grid {
	if s.grid == nil {
		s.grid = energy.NewGrid()
	}
	return s.grid
}

func (s *session) fluidGrid() *fluidnet.Grid {
	if s.fluids == nil {
		s.fluids = fluidnet.NewGrid()
	}
	return s.fluids
}

func (s *session) itemGrid() *itemnet.Grid {
	if s.items == nil {
		s.items = itemnet.NewGrid()
	}
	return s.items
}

// EnergyConduit, ItemConduit and FluidConduit are the conduit items, of a tier
// (1 to 3). They are blocks only to look like one in the inventory and the hand
// (a short length of pipe, models/blocks/conduit_icon.geo.json): using one never
// places the block but puts the conduit into a bundle (see placeConduit).
type EnergyConduit struct{ Tier int }
type ItemConduit struct{ Tier int }
type FluidConduit struct{ Tier int }

// conduitKinds are the kinds in the order of conduitItemHashes.
var conduitKinds = [3]string{"energy", "item", "fluid"}

var conduitItemHashes = func() (h [3][conduitTiers + 1]uint64) {
	for k := range h {
		for t := 1; t <= conduitTiers; t++ {
			h[k][t] = block.NextHash()
		}
	}
	return
}()

func conduitItemID(kind string, tier int) string {
	return fmt.Sprintf("dragonfactory:conduit_%s_%d", kind, tier)
}

func conduitIconProperties(kind string, tier int) customblock.Properties {
	return customblock.Properties{
		Geometry:     "geometry.dragonfactory.conduit_icon_" + kind,
		Textures:     map[string]customblock.Material{"*": alphaMaterial(fmt.Sprintf("dragonfactory_conduit_%s_%d", kind, tier))},
		SelectionBox: conduitCore,
	}
}

func (c EnergyConduit) EncodeItem() (string, int16) { return conduitItemID("energy", c.Tier), 0 }
func (c EnergyConduit) EncodeBlock() (string, map[string]any) {
	return conduitItemID("energy", c.Tier), nil
}
func (c EnergyConduit) Hash() (uint64, uint64) { return conduitItemHashes[0][c.Tier], 0 }
func (EnergyConduit) Model() world.BlockModel  { return conduitModel{} }
func (c EnergyConduit) Name() string {
	return i18n.T(i18n.Default, fmt.Sprintf("name.EnergyConduit.%d", c.Tier))
}
func (EnergyConduit) Category() category.Category { return category.Construction() }
func (c EnergyConduit) Properties() customblock.Properties {
	return conduitIconProperties("energy", c.Tier)
}

func (c ItemConduit) EncodeItem() (string, int16) { return conduitItemID("item", c.Tier), 0 }
func (c ItemConduit) EncodeBlock() (string, map[string]any) {
	return conduitItemID("item", c.Tier), nil
}
func (c ItemConduit) Hash() (uint64, uint64) { return conduitItemHashes[1][c.Tier], 0 }
func (ItemConduit) Model() world.BlockModel  { return conduitModel{} }
func (c ItemConduit) Name() string {
	return i18n.T(i18n.Default, fmt.Sprintf("name.ItemConduit.%d", c.Tier))
}
func (ItemConduit) Category() category.Category { return category.Construction() }
func (c ItemConduit) Properties() customblock.Properties {
	return conduitIconProperties("item", c.Tier)
}

func (c FluidConduit) EncodeItem() (string, int16) { return conduitItemID("fluid", c.Tier), 0 }
func (c FluidConduit) EncodeBlock() (string, map[string]any) {
	return conduitItemID("fluid", c.Tier), nil
}
func (c FluidConduit) Hash() (uint64, uint64) { return conduitItemHashes[2][c.Tier], 0 }
func (FluidConduit) Model() world.BlockModel  { return conduitModel{} }
func (c FluidConduit) Name() string {
	return i18n.T(i18n.Default, fmt.Sprintf("name.FluidConduit.%d", c.Tier))
}
func (FluidConduit) Category() category.Category { return category.Construction() }
func (c FluidConduit) Properties() customblock.Properties {
	return conduitIconProperties("fluid", c.Tier)
}

func (c EnergyConduit) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) bool {
	return placeConduit(pos, face, tx, user, ctx, func(b *ConduitBundle) bool {
		if b.Energy != 0 {
			return false
		}
		b.Energy = c.Tier
		return true
	})
}

func (c ItemConduit) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) bool {
	return placeConduit(pos, face, tx, user, ctx, func(b *ConduitBundle) bool {
		if b.Item != 0 {
			return false
		}
		b.Item = c.Tier
		// a new item conduit moves nothing until a side is switched on in its form
		for i := range b.Modes {
			b.Modes[i] = itemnet.Off
		}
		return true
	})
}

func (c FluidConduit) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) bool {
	return placeConduit(pos, face, tx, user, ctx, func(b *ConduitBundle) bool {
		if b.Fluid != 0 {
			return false
		}
		b.Fluid = c.Tier
		return true
	})
}

// placeConduit puts a conduit into a bundle or makes a new one. add sets the
// conduit in a bundle and reports whether there was room (the kind is not in
// it yet). The conduit goes
//   - into the bundle that was clicked, if it has room;
//   - otherwise, with the player sneaking, as if that bundle were the ground:
//     into the bundle on the clicked side if it has room, or a new one there;
//   - and when a block behind a thin conduit was clicked and the conduit stands
//     on the clicked side, into that bundle if it has room.
func placeConduit(pos cube.Pos, face cube.Face, tx *world.Tx, user item.User, ctx *item.UseContext, add func(*ConduitBundle) bool) bool {
	sneaking := false
	if p, ok := user.(*player.Player); ok {
		sneaking = p.Sneaking()
	}
	var target cube.Pos
	var b ConduitBundle
	found := false
	try := func(at cube.Pos) bool {
		if at.OutOfBounds(tx.Range()) {
			return false
		}
		if cur, ok := tx.Block(at).(ConduitBundle); ok {
			if !add(&cur) {
				return false
			}
			target, b, found = at, cur, true
			return true
		}
		if !replaceableByConduit(tx.Block(at)) {
			return false
		}
		var fresh ConduitBundle
		add(&fresh)
		target, b, found = at, fresh, true
		return true
	}
	_, clickedBundle := tx.Block(pos).(ConduitBundle)
	switch {
	case clickedBundle && !sneaking:
		try(pos)
	case clickedBundle:
		try(pos.Side(face))
	default:
		try(pos.Side(face))
	}
	if !found {
		// The client predicted a block: show what is there.
		if p, ok := user.(*player.Player); ok {
			side := pos.Side(face)
			p.ViewFakeBlock(side, tx.Block(side))
		}
		return false
	}
	b.EConn, b.IConn, b.FConn = b.connections(tx, target)
	if placer, ok := user.(block.Placer); ok {
		placer.PlaceBlock(target, b, ctx)
	} else {
		tx.SetBlock(target, b, nil)
		ctx.SubtractFromCount(1)
	}
	if got, ok := tx.Block(target).(ConduitBundle); ok {
		if s, ok := activeGame.sessionOf(tx.World()); ok {
			s.showConduit(tx, target, got)
		}
	}
	if side := pos.Side(face); target != side {
		// The client may have predicted a block on the clicked side.
		if p, ok := user.(*player.Player); ok {
			p.ViewFakeBlock(side, tx.Block(side))
		}
	}
	return ctx.CountSub > 0
}

func replaceableByConduit(b world.Block) bool {
	if _, air := b.(block.Air); air {
		return true
	}
	r, ok := b.(block.Replaceable)
	return ok && r.ReplaceableBy(ConduitBundle{Energy: 1})
}

func init() {
	world.RegisterBlock(ConduitBundle{})
	for t := 1; t <= conduitTiers; t++ {
		world.RegisterBlock(EnergyConduit{Tier: t})
		world.RegisterBlock(ItemConduit{Tier: t})
		world.RegisterBlock(FluidConduit{Tier: t})
		world.RegisterItem(EnergyConduit{Tier: t})
		world.RegisterItem(ItemConduit{Tier: t})
		world.RegisterItem(FluidConduit{Tier: t})
	}
}

// LightDiffusionLevel makes a bundle transparent to light: a chest under it
// opens only if the block above diffuses little light (block.Chest.Activate).
func (ConduitBundle) LightDiffusionLevel() uint8 { return 0 }
