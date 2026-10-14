package dragonfactory

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/internal/i18n"
	"github.com/presentkim/dragonfactory/internal/itemnet"
)

// Conduits placed in earlier versions are blocks of the types below (the same
// block names and states as then): they turn into a ConduitBundle the first
// time they tick or a neighbour changes, and are invisible until then. Their
// definitions stay registered so the world can still read them (and the
// client breaks them at the speed the server does).
//
//   - energy_conduit_N and item_conduit_N: one conduit per block, the sides it
//     connects to as a state (conduitConnState);
//   - conduit_bundle: a bundle with its tiers and sides as one state.

var legacyHashes = func() (h [3][conduitTiers + 1]uint64) {
	for k := range h {
		for t := range h[k] {
			h[k][t] = block.NextHash()
		}
	}
	return
}()

// legacyConduit is a conduit block of the first kind: kind 0 is an energy
// conduit, 1 an item conduit.
type legacyConduit struct {
	Kind, Tier, Conn int
	Modes            [6]itemnet.Mode
}

func (c legacyConduit) id() string {
	return fmt.Sprintf("dragonfactory:%s_conduit_%d", [2]string{"energy", "item"}[c.Kind], c.Tier)
}
func (c legacyConduit) EncodeItem() (string, int16) { return c.id(), 0 }
func (c legacyConduit) EncodeBlock() (string, map[string]any) {
	return c.id(), map[string]any{conduitConnState: conduitConnValue(c.Conn)}
}
func (c legacyConduit) Hash() (uint64, uint64)  { return legacyHashes[c.Kind][c.Tier], uint64(c.Conn) }
func (c legacyConduit) Model() world.BlockModel { return conduitModel{c.Conn} }
func (c legacyConduit) Name() string {
	key := [2]string{"name.EnergyConduit.%d", "name.ItemConduit.%d"}[c.Kind]
	return i18n.T(i18n.Default, fmt.Sprintf(key, c.Tier))
}
func (c legacyConduit) Properties() customblock.Properties { return invisibleProperties() }
func (legacyConduit) Category() category.Category          { return category.Construction() }
func (legacyConduit) States() map[string][]any {
	var values []any
	for conn := range 64 {
		values = append(values, conduitConnValue(conn))
	}
	return map[string][]any{conduitConnState: values}
}
func (c legacyConduit) Permutations() []customblock.Permutation { return nil }
func (c legacyConduit) BreakInfo() block.BreakInfo              { return c.bundle().BreakInfo() }

func (c legacyConduit) bundle() ConduitBundle {
	if c.Kind == 0 {
		return ConduitBundle{Energy: c.Tier}
	}
	return ConduitBundle{Item: c.Tier, Modes: c.Modes}
}

func (c legacyConduit) upgrade(pos cube.Pos, tx *world.Tx) {
	b := c.bundle()
	tx.SetBlock(pos, b, nil)
	b.refresh(pos, tx)
}

func (c legacyConduit) Tick(_ int64, pos cube.Pos, tx *world.Tx)          { c.upgrade(pos, tx) }
func (c legacyConduit) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) { c.upgrade(pos, tx) }
func (c legacyConduit) EncodeNBT() map[string]any {
	if c.Kind == 1 {
		return c.bundle().EncodeNBT()
	}
	return map[string]any{"id": "DragonFactoryConduit"}
}
func (c legacyConduit) DecodeNBT(data map[string]any) any {
	if c.Kind == 1 {
		c.Modes = c.bundle().DecodeNBT(data).(ConduitBundle).Modes
	}
	return c
}

// legacyBundle is the bundle block of the second kind: Value is its one state
// (the tier of the energy conduit, of the item conduit and the sides, "TTCC").
type legacyBundle struct {
	Energy, Item, Conn int
	Modes              [6]itemnet.Mode
}

func (b legacyBundle) EncodeBlock() (string, map[string]any) {
	return "dragonfactory:conduit_bundle", map[string]any{"dragonfactory:bundle": fmt.Sprintf("%d%d%s", b.Energy, b.Item, conduitConnValue(b.Conn))}
}
func (b legacyBundle) Hash() (uint64, uint64) {
	return legacyHashes[2][0], uint64(b.Energy) | uint64(b.Item)<<2 | uint64(b.Conn)<<4
}
func (b legacyBundle) Model() world.BlockModel          { return conduitModel{b.Conn} }
func (legacyBundle) Name() string                       { return i18n.T(i18n.Default, "name.ConduitBundle") }
func (legacyBundle) Properties() customblock.Properties { return invisibleProperties() }
func (legacyBundle) States() map[string][]any {
	var values []any
	legacyBundleCombos(func(e, i, conn int) { values = append(values, fmt.Sprintf("%d%d%s", e, i, conduitConnValue(conn))) })
	return map[string][]any{"dragonfactory:bundle": values}
}
func (legacyBundle) Permutations() []customblock.Permutation { return nil }
func (b legacyBundle) bundle() ConduitBundle {
	return ConduitBundle{Energy: b.Energy, Item: b.Item, Modes: b.Modes}
}
func (b legacyBundle) BreakInfo() block.BreakInfo { return b.bundle().BreakInfo() }
func (b legacyBundle) upgrade(pos cube.Pos, tx *world.Tx) {
	n := b.bundle()
	tx.SetBlock(pos, n, nil)
	n.refresh(pos, tx)
}
func (b legacyBundle) Tick(_ int64, pos cube.Pos, tx *world.Tx)          { b.upgrade(pos, tx) }
func (b legacyBundle) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) { b.upgrade(pos, tx) }
func (b legacyBundle) EncodeNBT() map[string]any                         { return b.bundle().EncodeNBT() }
func (b legacyBundle) DecodeNBT(data map[string]any) any {
	b.Modes = b.bundle().DecodeNBT(data).(ConduitBundle).Modes
	return b
}

// legacyBundleCombos calls f for every state of the second kind, in the order
// the client numbers them.
func legacyBundleCombos(f func(e, i, conn int)) {
	for e := 0; e <= conduitTiers; e++ {
		for i := 0; i <= conduitTiers; i++ {
			for conn := range 64 {
				f(e, i, conn)
			}
		}
	}
}

// invisibleProperties is the look of the conduit block: a speck of a model with
// a transparent texture, the display entity draws the conduits.
func invisibleProperties() customblock.Properties {
	return customblock.Properties{
		Geometry:     "geometry.dragonfactory_conduit_node",
		Textures:     map[string]customblock.Material{"*": alphaMaterial("dragonfactory_conduit_node")},
		SelectionBox: bundleSelection,
	}
}

// UseOnBlock lets an old conduit item (still in some inventories) place its
// conduit into a bundle like the new one.
func (c legacyConduit) UseOnBlock(pos cube.Pos, face cube.Face, clickPos mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) bool {
	if c.Kind == 0 {
		return EnergyConduit{Tier: c.Tier}.UseOnBlock(pos, face, clickPos, tx, user, ctx)
	}
	return ItemConduit{Tier: c.Tier}.UseOnBlock(pos, face, clickPos, tx, user, ctx)
}

// migrateConduitItems turns the old conduit items in a player's inventory into
// the new ones (the old ones are blocks that draw nothing).
func migrateConduitItems(p *player.Player) {
	inv := p.Inventory()
	for slot, st := range inv.Slots() {
		old, ok := st.Item().(legacyConduit)
		if !ok {
			continue
		}
		var now world.Item = EnergyConduit{Tier: old.Tier}
		if old.Kind == 1 {
			now = ItemConduit{Tier: old.Tier}
		}
		_ = inv.SetItem(slot, item.NewStack(now, st.Count()))
	}
}

func init() {
	for t := 1; t <= conduitTiers; t++ {
		for conn := range 64 {
			world.RegisterBlock(legacyConduit{Kind: 0, Tier: t, Conn: conn})
			world.RegisterBlock(legacyConduit{Kind: 1, Tier: t, Conn: conn})
		}
	}
	legacyBundleCombos(func(e, i, conn int) { world.RegisterBlock(legacyBundle{Energy: e, Item: i, Conn: conn}) })
	// the old conduit items are the old blocks (still in some inventories)
	for t := 1; t <= conduitTiers; t++ {
		world.RegisterItem(legacyConduit{Kind: 0, Tier: t})
		world.RegisterItem(legacyConduit{Kind: 1, Tier: t})
	}
}
