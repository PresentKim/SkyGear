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
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/presentkim/dragonfactory/internal/i18n"
	"github.com/presentkim/dragonfactory/internal/itemio"
	"github.com/presentkim/dragonfactory/internal/itemnet"
)

// Item conduits (docs/design/06-power-and-conduits.md §3.3): they connect to
// the conduits of their kind and to every block that holds items (internal/
// itemio), and move items from the sides that give to the sides that take.
// Using a conduit side with an empty hand while sneaking cycles that side's
// connection mode (both ways, take only, give only, off); the WAILA panel
// lists the modes. A conduit is a block entity: it keeps its modes.

// ItemConduit is an item conduit of a tier (1 to 3); Conn is the set of
// sides it shows an arm to and Modes the connection mode of each side
// (indexed by cube.Face).
type ItemConduit struct {
	Tier, Conn int
	Modes      [6]itemnet.Mode
}

var itemConduitHashes = func() [conduitTiers + 1]uint64 {
	var h [conduitTiers + 1]uint64
	for t := 1; t <= conduitTiers; t++ {
		h[t] = block.NextHash()
	}
	return h
}()

func (c ItemConduit) EncodeItem() (string, int16) {
	return fmt.Sprintf("dragonfactory:item_conduit_%d", c.Tier), 0
}
func (c ItemConduit) EncodeBlock() (string, map[string]any) {
	return fmt.Sprintf("dragonfactory:item_conduit_%d", c.Tier), map[string]any{conduitConnState: conduitConnValue(c.Conn)}
}
func (c ItemConduit) Hash() (uint64, uint64)  { return itemConduitHashes[c.Tier], uint64(c.Conn) }
func (c ItemConduit) Model() world.BlockModel { return conduitModel{c.Conn} }
func (c ItemConduit) Name() string {
	return i18n.T(i18n.Default, fmt.Sprintf("name.ItemConduit.%d", c.Tier))
}
func (ItemConduit) Category() category.Category { return category.Construction() }

func (c ItemConduit) materials() map[string]customblock.Material {
	return map[string]customblock.Material{"*": alphaMaterial(fmt.Sprintf("dragonfactory_conduit_item_%d", c.Tier))}
}

func (c ItemConduit) Properties() customblock.Properties {
	return customblock.Properties{Geometry: conduitGeometry(0), Textures: c.materials(), SelectionBox: conduitCore}
}

func (ItemConduit) States() map[string][]any { return conduitStates() }
func (c ItemConduit) Permutations() []customblock.Permutation {
	return conduitPermutations(c.materials())
}

func (c ItemConduit) BreakInfo() block.BreakInfo {
	return block.BreakInfo{
		Hardness: 1.5, BlastResistance: 3,
		Harvestable: func(item.Tool) bool { return true },
		Effective:   customBlockEffective,
		Drops: func(item.Tool, []item.Enchantment) []item.Stack {
			return []item.Stack{item.NewStack(ItemConduit{Tier: c.Tier}, 1)}
		},
	}
}

// ItemTier and ItemMode implement itemnet.Conduit.
func (c ItemConduit) ItemTier() int                     { return c.Tier }
func (c ItemConduit) ItemMode(f cube.Face) itemnet.Mode { return c.Modes[f] }

// EncodeNBT and DecodeNBT keep the modes (and make the conduit a block entity,
// which is what gets ticked).
func (c ItemConduit) EncodeNBT() map[string]any {
	modes := make([]byte, len(c.Modes))
	for i, m := range c.Modes {
		modes[i] = '0' + byte(m)
	}
	return map[string]any{"id": "DragonFactoryItemConduit", "Modes": string(modes)}
}

func (c ItemConduit) DecodeNBT(data map[string]any) any {
	if modes, ok := data["Modes"].(string); ok && len(modes) == len(c.Modes) {
		for i := range c.Modes {
			c.Modes[i] = itemnet.Mode(modes[i]-'0') % 4
		}
	}
	return c
}

// joinsItem reports whether a conduit connects to the block b at at, on its
// side f: another item conduit, or a block that holds items, unless that side
// is off.
func (c ItemConduit) joins(tx *world.Tx, f cube.Face, at cube.Pos, b world.Block) bool {
	if _, ok := b.(itemnet.Conduit); ok {
		return true
	}
	if c.Modes[f] == itemnet.Off {
		return false
	}
	in, ex := itemio.IO(tx, at)
	return in != nil || ex != nil
}

// refresh sets the block to show the sides it connects to.
func (c ItemConduit) refresh(pos cube.Pos, tx *world.Tx) {
	conn := connectionsOf(tx, pos, func(f cube.Face, at cube.Pos, b world.Block) bool { return c.joins(tx, f, at, b) })
	if conn != c.Conn {
		c.Conn = conn
		tx.SetBlock(pos, c, nil)
	}
}

// NeighbourUpdateTick keeps the arms up to date.
func (c ItemConduit) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) { c.refresh(pos, tx) }

// Tick registers the conduit with the item grid every few ticks.
func (c ItemConduit) Tick(currentTick int64, pos cube.Pos, tx *world.Tx) {
	if currentTick%4 != 0 {
		return
	}
	s, ok := activeGame.sessionOf(tx.World())
	if !ok {
		return
	}
	c.refresh(pos, tx)
	s.itemGrid().AddConduit(pos)
}

// Activate cycles the mode of the clicked side (empty hand, sneaking).
func (c ItemConduit) Activate(pos cube.Pos, face cube.Face, tx *world.Tx, u item.User, _ *item.UseContext) bool {
	p, ok := u.(*player.Player)
	if !ok || !p.Sneaking() {
		return false
	}
	if held, _ := p.HeldItems(); !held.Empty() {
		return false
	}
	at := pos.Side(face)
	if _, isConduit := tx.Block(at).(itemnet.Conduit); isConduit {
		return true
	}
	if in, ex := itemio.IO(tx, at); in == nil && ex == nil && c.Modes[face] == itemnet.Both {
		return true
	}
	c.Modes[face] = c.Modes[face].Next()
	c.refresh(pos, tx)
	if now, ok := tx.Block(pos).(ItemConduit); ok {
		c.Conn = now.Conn
	}
	tx.SetBlock(pos, c, nil)
	tx.PlaySound(pos.Vec3Centre(), sound.Click{})
	return true
}

// itemConduitLines are the lines of an item conduit's WAILA panel: the mode
// of every side that has a block with items.
func itemConduitLines(tx *world.Tx, pos cube.Pos, c ItemConduit) []string {
	names := map[cube.Face]string{
		cube.FaceUp: "up", cube.FaceDown: "down", cube.FaceNorth: "north", cube.FaceSouth: "south", cube.FaceWest: "west", cube.FaceEast: "east",
	}
	var lines []string
	for _, f := range conduitFaces {
		at := pos.Side(f)
		if _, isConduit := tx.Block(at).(itemnet.Conduit); isConduit {
			continue
		}
		in, ex := itemio.IO(tx, at)
		if in == nil && ex == nil && c.Modes[f] == itemnet.Both {
			continue
		}
		lines = append(lines, i18n.T(i18n.Default, "itemconduit.side."+names[f])+" §f"+i18n.T(i18n.Default, fmt.Sprintf("itemconduit.mode.%d", c.Modes[f])))
	}
	if len(lines) == 0 {
		lines = []string{i18n.T(i18n.Default, "itemconduit.lines.none")}
	}
	lines = append(lines, fmt.Sprintf(i18n.T(i18n.Default, "itemconduit.lines.rate"), itemnet.RateOf(c.Tier), float64(itemnet.StepTicks)/20))
	return append(lines, i18n.T(i18n.Default, "itemconduit.lines.hint"))
}

// itemGrid is the item grid of the island, made when first used.
func (s *session) itemGrid() *itemnet.Grid {
	if s.items == nil {
		s.items = itemnet.NewGrid()
	}
	return s.items
}

func init() {
	for t := 1; t <= conduitTiers; t++ {
		for conn := range 64 {
			world.RegisterBlock(ItemConduit{Tier: t, Conn: conn})
		}
		world.RegisterItem(ItemConduit{Tier: t})
	}
}
