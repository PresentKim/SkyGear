package dragonfactory

import (
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/internal/i18n"
)

// Magic plants (docs/design/08-storage-gear-magic.md §4): plants that grow on dirt
// from seeds and give things when they bloom, after Botania and Mystical
// Agriculture. Only the plants are here; mana comes later.
//   - Magic flowers: a seed, which monsters of the gate sometimes drop, grows into
//     a flower of its own colour (a seed for each of the 16) that gives petals.
//   - Better plants are made by crafting: a seed of a plant below + a vanilla plant
//     + petals + a material give the seed of the next kind (ore blooms give ore
//     chunks, gem blooms gems and redstone, essence blooms mob essences).
// A bloom regrows when it is harvested.

// petalColours are the colours of the petals, named like the dyes.
var petalColours = []string{"white", "orange", "magenta", "light_blue", "yellow", "lime", "pink", "gray", "light_gray", "cyan", "purple", "blue", "brown", "green", "red", "black"}

// Petal is a petal of a magic flower in a colour.
type Petal struct{ Colour string }

func (p Petal) EncodeItem() (string, int16) { return "dragonfactory:petal_" + p.Colour, 0 }
func (p Petal) Name() string {
	return i18n.T(i18n.Default, "pcolour."+p.Colour) + i18n.T(i18n.Default, "name.Petal.suffix")
}
func (Petal) Category() category.Category { return category.Items() }

// plantYield is what a bloom gives: Min to Max of an item.
type plantYield struct {
	item     world.Item
	min, max int
}

// plantKind is a kind of magic plant.
type plantKind struct {
	id   string
	tier int
	// seed is the id of the seed that grows it.
	seed  string
	grow  time.Duration
	yield []plantYield
}

// plantKinds are the kinds of plants in a fixed order (the block's states follow it).
var plantKinds []plantKind

// plantSeeds are the ids of the seeds in order.
var plantSeeds []string

func init() {
	for _, c := range petalColours {
		plantKinds = append(plantKinds, plantKind{id: "flower_" + c, tier: 1, seed: "flower_" + c, grow: 90 * time.Second, yield: []plantYield{{Petal{Colour: c}, 2, 3}}})
	}
	for _, m := range []string{OreIron, OreGold, OreCopper, MetalTin, MetalLead, MetalSilver, MetalNickel} {
		plantKinds = append(plantKinds, plantKind{id: "ore_" + m, tier: 2, seed: "ore_" + m, grow: 180 * time.Second, yield: []plantYield{{OreChunk{Ore: m}, 1, 2}}})
	}
	gems := []struct {
		id       string
		it       world.Item
		min, max int
	}{
		{"lapis", item.LapisLazuli{}, 2, 3}, {"redstone", item.RedstoneWire{}, 3, 4}, {"quartz", item.NetherQuartz{}, 1, 2},
		{"coal", item.Coal{}, 2, 3}, {"diamond", item.Diamond{}, 1, 1}, {"emerald", item.Emerald{}, 1, 1},
	}
	for _, g := range gems {
		plantKinds = append(plantKinds, plantKind{id: "gem_" + g.id, tier: 3, seed: "gem_" + g.id, grow: 300 * time.Second, yield: []plantYield{{g.it, g.min, g.max}}})
	}
	for _, mob := range mobKinds {
		plantKinds = append(plantKinds, plantKind{id: "ess_" + mob, tier: 4, seed: "ess_" + mob, grow: 480 * time.Second, yield: []plantYield{{Essence{Mob: mob}, 1, 1}}})
	}
	seen := map[string]bool{}
	for _, k := range plantKinds {
		if !seen[k.seed] {
			seen[k.seed] = true
			plantSeeds = append(plantSeeds, k.seed)
		}
	}
}

func plantKindIndex(id string) int {
	for i, k := range plantKinds {
		if k.id == id {
			return i
		}
	}
	return -1
}

// PlantSeed is the seed of a plant (each flower colour has its own seed).
type PlantSeed struct{ Seed string }

func (s PlantSeed) EncodeItem() (string, int16) { return "dragonfactory:seed_" + s.Seed, 0 }
func (s PlantSeed) Name() string {
	return i18n.T(i18n.Default, "pseed."+s.Seed) + i18n.T(i18n.Default, "name.PlantSeed.suffix")
}
func (PlantSeed) Category() category.Category { return category.Items() }

// UseOnBlock plants the seed on dirt or grass.
func (s PlantSeed) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, _ item.User, ctx *item.UseContext) bool {
	if face != cube.FaceUp {
		return false
	}
	switch tx.Block(pos).(type) {
	case block.Grass, block.Dirt, block.Farmland, block.Podzol:
	default:
		return false
	}
	at := pos.Side(cube.FaceUp)
	if _, air := tx.Block(at).(block.Air); !air {
		return false
	}
	sess, ok := activeGame.sessionOf(tx.World())
	if !ok {
		return false
	}
	id := s.Seed
	if plantKindIndex(id) < 0 {
		return false
	}
	tx.SetBlock(at, MagicPlant{Kind: id}, nil)
	now, _, _ := sess.clock.Now()
	if sess.state.Plants == nil {
		sess.state.Plants = map[string]*PlantRecord{}
	}
	sess.state.Plants[cropKey(at)] = &PlantRecord{Kind: id, ReadyAt: now + plantKinds[plantKindIndex(id)].grow}
	tx.PlaySound(at.Vec3Centre(), sound.BlockPlace{Block: block.Grass{}})
	ctx.SubtractFromCount(1)
	return true
}

// PlantRecord is a plant of the save: what it is and when it blooms.
type PlantRecord struct {
	Kind    string        `json:"kind"`
	ReadyAt time.Duration `json:"ready_at"`
}

const plantState = "dragonfactory:plant"

var plantHash = block.NextHash()

// MagicPlant is the block of a magic plant: its kind and whether it has bloomed.
type MagicPlant struct {
	Kind  string
	Bloom bool
}

func plantValue(kind string, bloom bool) string {
	if bloom {
		return kind + "_1"
	}
	return kind + "_0"
}

func (b MagicPlant) EncodeBlock() (string, map[string]any) {
	return "dragonfactory:magic_plant", map[string]any{plantState: plantValue(b.Kind, b.Bloom)}
}

// plantIndex is the number of the state of a plant.
func (b MagicPlant) index() uint64 {
	i := plantKindIndex(b.Kind)
	if i < 0 {
		i = 0
	}
	n := uint64(i * 2)
	if b.Bloom {
		n++
	}
	return n
}

func (b MagicPlant) Hash() (uint64, uint64) { return plantHash, b.index() }
func (MagicPlant) Model() world.BlockModel  { return model.Empty{} }
func (MagicPlant) Name() string             { return i18n.T(i18n.Default, "name.MagicPlant") }
func (MagicPlant) Properties() customblock.Properties {
	return customblock.Properties{
		Geometry:     "geometry.dragonfactory_magic_plant",
		Textures:     map[string]customblock.Material{"*": alphaMaterial("dragonfactory_plant_" + plantKinds[0].id + "_0")},
		CollisionBox: cube.Box(0.45, 0, 0.45, 0.55, 0.01, 0.55),
		SelectionBox: cube.Box(0.15, 0, 0.15, 0.85, 0.7, 0.85),
	}
}

// States implements block.Permutable: one state with the kind and stage.
func (MagicPlant) States() map[string][]any {
	var values []any
	for _, k := range plantKinds {
		values = append(values, plantValue(k.id, false), plantValue(k.id, true))
	}
	return map[string][]any{plantState: values}
}

// Permutations implements block.Permutable: one texture per kind and stage.
func (MagicPlant) Permutations() []customblock.Permutation {
	var out []customblock.Permutation
	for _, k := range plantKinds {
		for _, bloom := range []bool{false, true} {
			stage := 0
			if bloom {
				stage = 1
			}
			out = append(out, customblock.Permutation{
				Condition: fmt.Sprintf("q.block_state('%s') == '%s'", plantState, plantValue(k.id, bloom)),
				Properties: customblock.Properties{
					Geometry: "geometry.dragonfactory_magic_plant",
					Textures: map[string]customblock.Material{"*": alphaMaterial(fmt.Sprintf("dragonfactory_plant_%s_%d", k.id, stage))},
				},
			})
		}
	}
	return out
}

// seedStack returns the seed that grows kind.
func (k plantKind) seedStack() item.Stack { return item.NewStack(PlantSeed{Seed: k.seed}, 1) }

func (b MagicPlant) BreakInfo() block.BreakInfo {
	return block.BreakInfo{
		Hardness: 0, BlastResistance: 0,
		Harvestable: func(item.Tool) bool { return true },
		Effective:   func(item.Tool) bool { return false },
		Drops: func(item.Tool, []item.Enchantment) []item.Stack {
			i := plantKindIndex(b.Kind)
			if i < 0 {
				return nil
			}
			out := []item.Stack{plantKinds[i].seedStack()}
			if b.Bloom {
				out = append(out, rollYield(plantKinds[i])...)
			}
			return out
		},
	}
}

// rollYield rolls what a bloom of kind gives.
func rollYield(k plantKind) []item.Stack {
	var out []item.Stack
	for _, y := range k.yield {
		n := y.min
		if y.max > y.min {
			n += rollN(y.max - y.min + 1)
		}
		out = append(out, item.NewStack(y.item, n))
	}
	return out
}

func rollN(n int) int { return rand.IntN(n) }

// NeighbourUpdateTick drops a plant whose ground is gone.
func (b MagicPlant) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	switch tx.Block(pos.Side(cube.FaceDown)).(type) {
	case block.Grass, block.Dirt, block.Farmland, block.Podzol:
		return
	}
	tx.SetBlock(pos, nil, nil)
	if s, ok := activeGame.sessionOf(tx.World()); ok {
		delete(s.state.Plants, cropKey(pos))
		for _, d := range b.BreakInfo().Drops(nil, nil) {
			s.dropAt(tx, pos.Vec3Centre(), d)
		}
	}
}

// Activate harvests a bloom by hand: it gives its yield and starts to regrow.
func (b MagicPlant) Activate(pos cube.Pos, _ cube.Face, tx *world.Tx, _ item.User, _ *item.UseContext) bool {
	if !b.Bloom {
		return true
	}
	s, ok := activeGame.sessionOf(tx.World())
	if !ok {
		return false
	}
	s.harvestPlant(tx, pos, b)
	return true
}

// harvestPlant gives a bloom's yield and sets the plant back to a bud.
func (s *session) harvestPlant(tx *world.Tx, pos cube.Pos, b MagicPlant) {
	i := plantKindIndex(b.Kind)
	if i < 0 {
		return
	}
	k := plantKinds[i]
	for _, st := range s.rollPlantYield(k) {
		s.dropAt(tx, pos.Vec3Centre().Add(mgl64.Vec3{0, 0.4, 0}), st)
	}
	// a seed now and then
	if s.rand.IntN(100) < 6 {
		s.dropAt(tx, pos.Vec3Centre().Add(mgl64.Vec3{0, 0.4, 0}), k.seedStack())
	}
	s.advance(tx, advBloom)
	switch k.tier {
	case 2:
		s.advance(tx, advOreHarvest)
	case 3:
		s.advance(tx, advGemHarvest)
	case 4:
		s.advance(tx, advEssHarvest)
	}
	tx.SetBlock(pos, MagicPlant{Kind: b.Kind}, nil)
	now, _, _ := s.clock.Now()
	if r := s.state.Plants[cropKey(pos)]; r != nil {
		r.ReadyAt = now + k.grow
	} else {
		if s.state.Plants == nil {
			s.state.Plants = map[string]*PlantRecord{}
		}
		s.state.Plants[cropKey(pos)] = &PlantRecord{Kind: b.Kind, ReadyAt: now + k.grow}
	}
	tx.PlaySound(pos.Vec3Centre(), sound.BlockPlace{Block: block.Grass{}})
}

// rollPlantYield rolls the yield with the session's random source.
func (s *session) rollPlantYield(k plantKind) []item.Stack {
	var out []item.Stack
	for _, y := range k.yield {
		n := y.min
		if y.max > y.min {
			n += s.rand.IntN(y.max - y.min + 1)
		}
		out = append(out, item.NewStack(y.item, n))
	}
	return out
}

// tickPlants blooms the plants whose time has come (every second).
func (s *session) tickPlants(tx *world.Tx) {
	if len(s.state.Plants) == 0 {
		return
	}
	now, _, _ := s.clock.Now()
	for key, r := range s.state.Plants {
		pos, ok := parseCropKey(key)
		if !ok {
			delete(s.state.Plants, key)
			continue
		}
		b, isPlant := tx.Block(pos).(MagicPlant)
		if !isPlant {
			delete(s.state.Plants, key)
			continue
		}
		if !b.Bloom && now >= r.ReadyAt {
			tx.SetBlock(pos, MagicPlant{Kind: b.Kind, Bloom: true}, nil)
		}
	}
}

// plantRecipes registers the recipes of the better seeds.
func plantRecipes() {
	st := func(it world.Item, n int) []recipe.Item {
		out := make([]recipe.Item, n)
		for i := range out {
			out[i] = item.NewStack(it, 1)
		}
		return out
	}
	join := func(parts ...[]recipe.Item) []recipe.Item {
		var out []recipe.Item
		for _, p := range parts {
			out = append(out, p...)
		}
		return out
	}
	reg := func(seed string, in ...[]recipe.Item) {
		recipe.Register(recipe.NewShapeless(join(in...), item.NewStack(PlantSeed{Seed: seed}, 1), "crafting_table"))
	}
	// dyes: a petal gives two dyes of its colour
	for _, c := range petalColours {
		recipe.Register(recipe.NewShapeless(st(Petal{Colour: c}, 1), item.NewStack(item.Dye{Colour: dyeColour(c)}, 2), "crafting_table"))
	}
	// ore blooms: a flower seed, a vanilla flower, petals and four dust of the metal
	ore := []struct {
		metal  string
		flower block.FlowerType
		p1, p2 string
	}{
		{OreIron, block.Poppy(), "gray", "red"}, {OreGold, block.Dandelion(), "yellow", "orange"}, {OreCopper, block.OrangeTulip(), "orange", "brown"},
		{MetalTin, block.WhiteTulip(), "light_gray", "white"}, {MetalLead, block.BlueOrchid(), "purple", "blue"},
		{MetalSilver, block.OxeyeDaisy(), "white", "light_blue"}, {MetalNickel, block.Allium(), "green", "lime"},
	}
	for _, o := range ore {
		reg("ore_"+o.metal, st(PlantSeed{Seed: "flower_" + o.p1}, 1), st(block.Flower{Type: o.flower}, 1), st(Petal{Colour: o.p1}, 2), st(Petal{Colour: o.p2}, 1), st(MetalDust{Metal: o.metal}, 4))
	}
	// gem blooms: the seed of an ore bloom, a vanilla plant, petals and the material
	gem := []struct {
		id, from string
		plant    world.Item
		colour   string
		pn       int
		mat      world.Item
		mn       int
	}{
		{"coal", "ore_" + OreCopper, item.Wheat{}, "black", 2, item.Coal{}, 8},
		{"lapis", "ore_" + MetalTin, item.Beetroot{}, "blue", 3, item.LapisLazuli{}, 4},
		{"redstone", "ore_" + OreIron, block.Carrot{}, "red", 3, item.RedstoneWire{}, 8},
		{"quartz", "ore_" + MetalSilver, block.Potato{}, "white", 3, item.NetherQuartz{}, 4},
		{"diamond", "ore_" + OreGold, block.MelonSeeds{}, "light_blue", 4, item.Diamond{}, 2},
		{"emerald", "ore_" + MetalNickel, block.PumpkinSeeds{}, "green", 4, item.Emerald{}, 2},
	}
	for _, g := range gem {
		reg("gem_"+g.id, st(PlantSeed{Seed: g.from}, 1), st(g.plant, 1), st(Petal{Colour: g.colour}, g.pn), st(g.mat, g.mn))
	}
	// essence blooms: a redstone bloom seed, three essences, petals and a mob item
	mob := map[string]struct {
		it     world.Item
		colour string
	}{
		MobZombie: {item.RottenFlesh{}, "green"}, MobSkeleton: {item.Bone{}, "white"}, MobCreeper: {item.Gunpowder{}, "lime"},
		MobSpider: {item.SpiderEye{}, "red"}, MobSlime: {item.Slimeball{}, "lime"}, MobEnder: {item.EnderPearl{}, "purple"}, MobBlaze: {item.BlazePowder{}, "orange"},
	}
	for _, m := range mobKinds {
		d := mob[m]
		reg("ess_"+m, st(PlantSeed{Seed: "gem_redstone"}, 1), st(Essence{Mob: m}, 3), st(Petal{Colour: d.colour}, 2), st(d.it, 1))
	}
}

func init() {
	for _, c := range petalColours {
		world.RegisterItem(Petal{Colour: c})
	}
	for _, s := range plantSeeds {
		world.RegisterItem(PlantSeed{Seed: s})
	}
	for _, k := range plantKinds {
		world.RegisterBlock(MagicPlant{Kind: k.id})
		world.RegisterBlock(MagicPlant{Kind: k.id, Bloom: true})
	}
	plantRecipes()
}

// dyeColour is the dye colour of a petal colour.
func dyeColour(c string) item.Colour {
	return map[string]item.Colour{
		"white": item.ColourWhite(), "orange": item.ColourOrange(), "magenta": item.ColourMagenta(), "light_blue": item.ColourLightBlue(),
		"yellow": item.ColourYellow(), "lime": item.ColourLime(), "pink": item.ColourPink(), "gray": item.ColourGrey(),
		"light_gray": item.ColourLightGrey(), "cyan": item.ColourCyan(), "purple": item.ColourPurple(), "blue": item.ColourBlue(),
		"brown": item.ColourBrown(), "green": item.ColourGreen(), "red": item.ColourRed(), "black": item.ColourBlack(),
	}[c]
}
