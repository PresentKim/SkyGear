package skygear

import (
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/presentkim/skygear/content/skygear/achievement"

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
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/content/skygear/blockmodel"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/textfmt"
)

// Magic plants (after Botania and Mystical Agriculture) grow on dirt from
// seeds and regrow after each harvest. Flower seeds drop from gate monsters;
// better seeds are crafted from a lower seed, a vanilla plant, petals and a
// material (ore, gem and essence blooms).

// plantYield is what a bloom gives: Min to Max of an item.
type plantYield struct {
	item     world.Item
	min, max int
}

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
	for _, c := range sgitem.PetalColours {
		plantKinds = append(plantKinds, plantKind{id: "flower_" + c, tier: 1, seed: "flower_" + c, grow: 90 * time.Second, yield: []plantYield{{sgitem.Petal{Colour: c}, 2, 3}}})
	}
	for _, m := range []string{sgitem.OreIron, sgitem.OreGold, sgitem.OreCopper, sgitem.MetalTin, sgitem.MetalLead, sgitem.MetalSilver, sgitem.MetalNickel} {
		plantKinds = append(plantKinds, plantKind{id: "ore_" + m, tier: 2, seed: "ore_" + m, grow: 180 * time.Second, yield: []plantYield{{sgitem.OreChunk{Ore: m}, 1, 2}}})
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
	for _, mob := range sgitem.FlowerMobKinds {
		plantKinds = append(plantKinds, plantKind{id: "ess_" + mob, tier: 4, seed: "ess_" + mob, grow: 480 * time.Second, yield: []plantYield{{sgitem.Essence{Mob: mob}, 1, 1}}})
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

func (s PlantSeed) EncodeItem() (string, int16) { return "skygear:seed_" + s.Seed, 0 }
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
	now := sess.growNow()
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

const plantState = "skygear:plant"

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
	return "skygear:magic_plant", map[string]any{plantState: plantValue(b.Kind, b.Bloom)}
}

// index is the number of the state of a plant.
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
		Geometry:     "geometry.skygear_magic_plant",
		Textures:     map[string]customblock.Material{"*": blockmodel.AlphaMaterial("skygear_plant_" + plantKinds[0].id + "_0")},
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
					Geometry: "geometry.skygear_magic_plant",
					Textures: map[string]customblock.Material{"*": blockmodel.AlphaMaterial(fmt.Sprintf("skygear_plant_%s_%d", k.id, stage))},
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
			s.DropAt(tx, pos.Vec3Centre(), d)
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
	drops, ok := s.plantDrops(b)
	if !ok {
		return
	}
	for _, st := range drops {
		s.DropAt(tx, pos.Vec3Centre().Add(mgl64.Vec3{0, 0.4, 0}), st)
	}
	s.resetPlant(tx, pos, b)
}

// plantDrops rolls what harvesting a bloom gives: its yield, and a seed now
// and then.
func (s *session) plantDrops(b MagicPlant) ([]item.Stack, bool) {
	i := plantKindIndex(b.Kind)
	if i < 0 {
		return nil, false
	}
	k := plantKinds[i]
	out := s.rollPlantYield(k)
	if s.Rand.IntN(100) < 6 {
		out = append(out, k.seedStack())
	}
	return out, true
}

// resetPlant sets a harvested plant back to a bud, starts its next growth
// and reports the harvest to the advancements.
func (s *session) resetPlant(tx *world.Tx, pos cube.Pos, b MagicPlant) {
	k := plantKinds[plantKindIndex(b.Kind)]
	s.advance(tx, advancement.Bloom)
	achievement.CountAll(tx, "bloom_total", 1)
	switch k.tier {
	case 2:
		s.advance(tx, advancement.OreHarvest)
		achievement.CountAll(tx, "bloom_ore", 1)
	case 3:
		s.advance(tx, advancement.GemHarvest)
		achievement.CountAll(tx, "bloom_gem", 1)
	case 4:
		s.advance(tx, advancement.EssHarvest)
		achievement.CountAll(tx, "bloom_ess", 1)
	}
	tx.SetBlock(pos, MagicPlant{Kind: b.Kind}, nil)
	now := s.growNow()
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
			n += s.Rand.IntN(y.max - y.min + 1)
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
	now := s.growNow()
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
	for _, c := range sgitem.PetalColours {
		recipe.Register(recipe.NewShapeless(st(sgitem.Petal{Colour: c}, 1), item.NewStack(item.Dye{Colour: sgitem.DyeColour(c)}, 2), "crafting_table"))
	}
	// ore blooms: a flower seed, a vanilla flower, petals and four dust of the metal
	ore := []struct {
		metal  string
		flower block.FlowerType
		p1, p2 string
	}{
		{sgitem.OreIron, block.Poppy(), "gray", "red"}, {sgitem.OreGold, block.Dandelion(), "yellow", "orange"}, {sgitem.OreCopper, block.OrangeTulip(), "orange", "brown"},
		{sgitem.MetalTin, block.WhiteTulip(), "light_gray", "white"}, {sgitem.MetalLead, block.BlueOrchid(), "purple", "blue"},
		{sgitem.MetalSilver, block.OxeyeDaisy(), "white", "light_blue"}, {sgitem.MetalNickel, block.Allium(), "green", "lime"},
	}
	for _, o := range ore {
		reg("ore_"+o.metal, st(PlantSeed{Seed: "flower_" + o.p1}, 1), st(block.Flower{Type: o.flower}, 1), st(sgitem.Petal{Colour: o.p1}, 2), st(sgitem.Petal{Colour: o.p2}, 1), st(sgitem.MetalDust{Metal: o.metal}, 4))
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
		{"coal", "ore_" + sgitem.OreCopper, item.Wheat{}, "black", 2, item.Coal{}, 4},
		{"lapis", "ore_" + sgitem.MetalTin, item.Beetroot{}, "blue", 3, item.LapisLazuli{}, 4},
		{"redstone", "ore_" + sgitem.OreIron, block.Carrot{}, "red", 3, item.RedstoneWire{}, 4},
		{"quartz", "ore_" + sgitem.MetalSilver, block.Potato{}, "white", 3, item.NetherQuartz{}, 4},
		{"diamond", "ore_" + sgitem.OreGold, block.MelonSeeds{}, "light_blue", 4, item.Diamond{}, 2},
		{"emerald", "ore_" + sgitem.MetalNickel, block.PumpkinSeeds{}, "green", 4, item.Emerald{}, 2},
	}
	for _, g := range gem {
		reg("gem_"+g.id, st(PlantSeed{Seed: g.from}, 1), st(g.plant, 1), st(sgitem.Petal{Colour: g.colour}, g.pn), st(g.mat, g.mn))
	}
	// essence blooms: a redstone bloom seed, three essences, petals and a mob item
	mob := map[string]struct {
		it     world.Item
		colour string
	}{
		sgitem.MobZombie: {item.RottenFlesh{}, "green"}, sgitem.MobSkeleton: {item.Bone{}, "white"}, sgitem.MobCreeper: {item.Gunpowder{}, "lime"},
		sgitem.MobSpider: {item.SpiderEye{}, "red"}, sgitem.MobSlime: {item.Slimeball{}, "lime"}, sgitem.MobEnder: {item.EnderPearl{}, "purple"}, sgitem.MobBlaze: {item.BlazePowder{}, "orange"},
	}
	for _, m := range sgitem.FlowerMobKinds {
		d := mob[m]
		reg("ess_"+m, st(PlantSeed{Seed: "gem_redstone"}, 1), st(sgitem.Essence{Mob: m}, 3), st(sgitem.Petal{Colour: d.colour}, 2), st(d.it, 1))
	}
}

func init() {
	for _, c := range sgitem.PetalColours {
		world.RegisterItem(sgitem.Petal{Colour: c})
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

// plantLines are the WAILA lines of the plant b of kind k at pos: the time
// left until it blooms, or that it has and what harvesting it gives.
func (s *session) plantLines(lang string, pos cube.Pos, b MagicPlant, k plantKind) []string {
	if !b.Bloom {
		r := s.state.Plants[cropKey(pos)]
		if r == nil {
			return nil
		}
		now := s.growNow()
		return []string{i18n.T(lang, "crop.lines.left") + textfmt.Duration(lang, s.realLeft(max(0, r.ReadyAt-now)).Round(time.Second))}
	}
	lines := []string{i18n.T(lang, "plant.waila.bloom")}
	for _, y := range k.yield {
		lines = append(lines, fmt.Sprintf("§f%s §7×%s", naming.ItemName(lang, item.NewStack(y.item, 1)), rangeText(y.min, y.max)))
	}
	return lines
}
