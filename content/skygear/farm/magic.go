package farm

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
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/content/skygear/blockmodel"
	"github.com/presentkim/skygear/content/skygear/climate"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/textfmt"
)

// Magic plants (after Botania and Mystical Agriculture) grow on dirt from
// seeds and regrow after each harvest. Flower seeds drop from gate monsters;
// better seeds are crafted from a lower seed, a vanilla plant, petals and a
// material (ore, gem and essence blooms).

func init() {
	for _, c := range sgitem.PetalColours {
		PlantKinds = append(PlantKinds, PlantKind{ID: "flower_" + c, tier: 1, Seed: "flower_" + c, Grow: 90 * time.Second, Yield: []PlantYield{{sgitem.Petal{Colour: c}, 2, 3}}})
	}
	for _, m := range []string{sgitem.OreIron, sgitem.OreGold, sgitem.OreCopper, sgitem.MetalTin, sgitem.MetalLead, sgitem.MetalSilver, sgitem.MetalNickel} {
		PlantKinds = append(PlantKinds, PlantKind{ID: "ore_" + m, tier: 2, Seed: "ore_" + m, Grow: 180 * time.Second, Yield: []PlantYield{{sgitem.OreChunk{Ore: m}, 1, 2}}})
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
		PlantKinds = append(PlantKinds, PlantKind{ID: "gem_" + g.id, tier: 3, Seed: "gem_" + g.id, Grow: 300 * time.Second, Yield: []PlantYield{{g.it, g.min, g.max}}})
	}
	for _, mob := range sgitem.FlowerMobKinds {
		PlantKinds = append(PlantKinds, PlantKind{ID: "ess_" + mob, tier: 4, Seed: "ess_" + mob, Grow: 480 * time.Second, Yield: []PlantYield{{sgitem.Essence{Mob: mob}, 1, 1}}})
	}
	seen := map[string]bool{}
	for _, k := range PlantKinds {
		if !seen[k.Seed] {
			seen[k.Seed] = true
			PlantSeeds = append(PlantSeeds, k.Seed)
		}
	}
}

func init() {
	for _, c := range sgitem.PetalColours {
		world.RegisterItem(sgitem.Petal{Colour: c})
	}
	for _, s := range PlantSeeds {
		world.RegisterItem(PlantSeed{Seed: s})
	}
	for _, k := range PlantKinds {
		world.RegisterBlock(MagicPlant{Kind: k.ID})
		world.RegisterBlock(MagicPlant{Kind: k.ID, Bloom: true})
	}
	PlantRecipes()
}

// PlantYield is what a bloom gives: Min to Max of an item.
type PlantYield struct {
	Item     world.Item
	Min, Max int
}

type PlantKind struct {
	ID   string
	tier int
	// seed is the id of the seed that grows it.
	Seed  string
	Grow  time.Duration
	Yield []PlantYield
}

// PlantKinds are the kinds of plants in a fixed order (the block's states follow it).
var PlantKinds []PlantKind

// PlantSeeds are the ids of the seeds in order.
var PlantSeeds []string

func PlantKindIndex(id string) int {
	for i, k := range PlantKinds {
		if k.ID == id {
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
	sess, ok := core.Of(tx)
	if !ok {
		return false
	}
	id := s.Seed
	if PlantKindIndex(id) < 0 {
		return false
	}
	tx.SetBlock(at, MagicPlant{Kind: id}, nil)
	now := climate.GrowNow(sess)
	if StateOf(sess).Plants == nil {
		StateOf(sess).Plants = map[string]*PlantRecord{}
	}
	StateOf(sess).Plants[CropKey(at)] = &PlantRecord{Kind: id, ReadyAt: now + PlantKinds[PlantKindIndex(id)].Grow}
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
	i := PlantKindIndex(b.Kind)
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

func (MagicPlant) Model() world.BlockModel { return model.Empty{} }

func (MagicPlant) Name() string { return i18n.T(i18n.Default, "name.MagicPlant") }

func (MagicPlant) Properties() customblock.Properties {
	return customblock.Properties{
		Geometry:     "geometry.skygear_magic_plant",
		Textures:     map[string]customblock.Material{"*": blockmodel.AlphaMaterial("skygear_plant_" + PlantKinds[0].ID + "_0")},
		CollisionBox: cube.Box(0.45, 0, 0.45, 0.55, 0.01, 0.55),
		SelectionBox: cube.Box(0.15, 0, 0.15, 0.85, 0.7, 0.85),
	}
}

// States implements block.Permutable: one state with the kind and stage.
func (MagicPlant) States() map[string][]any {
	var values []any
	for _, k := range PlantKinds {
		values = append(values, plantValue(k.ID, false), plantValue(k.ID, true))
	}
	return map[string][]any{plantState: values}
}

// Permutations implements block.Permutable: one texture per kind and stage.
func (MagicPlant) Permutations() []customblock.Permutation {
	var out []customblock.Permutation
	for _, k := range PlantKinds {
		for _, bloom := range []bool{false, true} {
			stage := 0
			if bloom {
				stage = 1
			}
			out = append(out, customblock.Permutation{
				Condition: fmt.Sprintf("q.block_state('%s') == '%s'", plantState, plantValue(k.ID, bloom)),
				Properties: customblock.Properties{
					Geometry: "geometry.skygear_magic_plant",
					Textures: map[string]customblock.Material{"*": blockmodel.AlphaMaterial(fmt.Sprintf("skygear_plant_%s_%d", k.ID, stage))},
				},
			})
		}
	}
	return out
}

// seedStack returns the seed that grows kind.
func (k PlantKind) SeedStack() item.Stack { return item.NewStack(PlantSeed{Seed: k.Seed}, 1) }

func (b MagicPlant) BreakInfo() block.BreakInfo {
	return block.BreakInfo{
		Hardness: 0, BlastResistance: 0,
		Harvestable: func(item.Tool) bool { return true },
		Effective:   func(item.Tool) bool { return false },
		Drops: func(item.Tool, []item.Enchantment) []item.Stack {
			i := PlantKindIndex(b.Kind)
			if i < 0 {
				return nil
			}
			out := []item.Stack{PlantKinds[i].SeedStack()}
			if b.Bloom {
				out = append(out, rollYield(PlantKinds[i])...)
			}
			return out
		},
	}
}

// rollYield rolls what a bloom of kind gives.
func rollYield(k PlantKind) []item.Stack {
	var out []item.Stack
	for _, y := range k.Yield {
		n := y.Min
		if y.Max > y.Min {
			n += rollN(y.Max - y.Min + 1)
		}
		out = append(out, item.NewStack(y.Item, n))
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
	if s, ok := core.Of(tx); ok {
		delete(StateOf(s).Plants, CropKey(pos))
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
	s, ok := core.Of(tx)
	if !ok {
		return false
	}
	harvestPlant(s, tx, pos, b)
	return true
}

// harvestPlant gives a bloom's yield and sets the plant back to a bud.
func harvestPlant(s *core.Session, tx *world.Tx, pos cube.Pos, b MagicPlant) {
	drops, ok := plantDrops(s, b)
	if !ok {
		return
	}
	for _, st := range drops {
		s.DropAt(tx, pos.Vec3Centre().Add(mgl64.Vec3{0, 0.4, 0}), st)
	}
	resetPlant(s, tx, pos, b)
}

// plantDrops rolls what harvesting a bloom gives: its yield, and a seed now
// and then.
func plantDrops(s *core.Session, b MagicPlant) ([]item.Stack, bool) {
	i := PlantKindIndex(b.Kind)
	if i < 0 {
		return nil, false
	}
	k := PlantKinds[i]
	out := rollPlantYield(s, k)
	if s.Rand.IntN(100) < 6 {
		out = append(out, k.SeedStack())
	}
	return out, true
}

// resetPlant sets a harvested plant back to a bud, starts its next growth
// and reports the harvest to the advancements.
func resetPlant(s *core.Session, tx *world.Tx, pos cube.Pos, b MagicPlant) {
	k := PlantKinds[PlantKindIndex(b.Kind)]
	progress.Advance(s, tx, advancement.Bloom)
	achievement.CountAll(tx, "bloom_total", 1)
	switch k.tier {
	case 2:
		progress.Advance(s, tx, advancement.OreHarvest)
		achievement.CountAll(tx, "bloom_ore", 1)
	case 3:
		progress.Advance(s, tx, advancement.GemHarvest)
		achievement.CountAll(tx, "bloom_gem", 1)
	case 4:
		progress.Advance(s, tx, advancement.EssHarvest)
		achievement.CountAll(tx, "bloom_ess", 1)
	}
	tx.SetBlock(pos, MagicPlant{Kind: b.Kind}, nil)
	now := climate.GrowNow(s)
	if r := StateOf(s).Plants[CropKey(pos)]; r != nil {
		r.ReadyAt = now + k.Grow
	} else {
		if StateOf(s).Plants == nil {
			StateOf(s).Plants = map[string]*PlantRecord{}
		}
		StateOf(s).Plants[CropKey(pos)] = &PlantRecord{Kind: b.Kind, ReadyAt: now + k.Grow}
	}
	tx.PlaySound(pos.Vec3Centre(), sound.BlockPlace{Block: block.Grass{}})
}

// rollPlantYield rolls the yield with the session's random source.
func rollPlantYield(s *core.Session, k PlantKind) []item.Stack {
	var out []item.Stack
	for _, y := range k.Yield {
		n := y.Min
		if y.Max > y.Min {
			n += s.Rand.IntN(y.Max - y.Min + 1)
		}
		out = append(out, item.NewStack(y.Item, n))
	}
	return out
}

// TickPlants blooms the plants whose time has come (every second).
func TickPlants(s *core.Session, tx *world.Tx) {
	if len(StateOf(s).Plants) == 0 {
		return
	}
	now := climate.GrowNow(s)
	for key, r := range StateOf(s).Plants {
		pos, ok := parseCropKey(key)
		if !ok {
			delete(StateOf(s).Plants, key)
			continue
		}
		b, isPlant := tx.Block(pos).(MagicPlant)
		if !isPlant {
			delete(StateOf(s).Plants, key)
			continue
		}
		if !b.Bloom && now >= r.ReadyAt {
			tx.SetBlock(pos, MagicPlant{Kind: b.Kind, Bloom: true}, nil)
		}
	}
}

// PlantRecipes registers the recipes of the better seeds.
func PlantRecipes() {
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

// PlantLines are the WAILA lines of the plant b of kind k at pos: the time
// left until it blooms, or that it has and what harvesting it gives.
func PlantLines(s *core.Session, lang string, pos cube.Pos, b MagicPlant, k PlantKind) []string {
	if !b.Bloom {
		r := StateOf(s).Plants[CropKey(pos)]
		if r == nil {
			return nil
		}
		now := climate.GrowNow(s)
		return []string{i18n.T(lang, "crop.lines.left") + textfmt.Duration(lang, climate.RealLeft(s, max(0, r.ReadyAt-now)).Round(time.Second))}
	}
	lines := []string{i18n.T(lang, "plant.waila.bloom")}
	for _, y := range k.Yield {
		lines = append(lines, fmt.Sprintf("§f%s §7×%s", naming.ItemName(lang, item.NewStack(y.Item, 1)), textfmt.Range(y.Min, y.Max)))
	}
	return lines
}
