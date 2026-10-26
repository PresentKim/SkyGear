package dragonfactory

import (
	"fmt"
	"github.com/df-mc/dragonfly/server/block/cube"
	"maps"
	"slices"
	"strings"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/common/perf"
)

// Operator tools (/dfadmin) for running the server: items, research and
// advancement progress, the load line. Only players for whom the game's
// operator check returns true see and run them (and /dfdebug, debug.go).

type adminBase struct{ g *Game }

func (a adminBase) Allow(src cmd.Source) bool {
	p, ok := src.(*player.Player)
	return ok && a.g.isOperator != nil && a.g.isOperator(p)
}

// session returns the session of the operator's world.
func (a adminBase) session(src cmd.Source, o *cmd.Output, tx *world.Tx) (*player.Player, *session, bool) {
	p := src.(*player.Player)
	s, ok := a.g.sessionOf(tx.World())
	if !ok {
		o.Error(i18n.T(i18n.Of(p), "dragonfactory.admin.needSave"))
	}
	return p, s, ok
}

type adminItem struct {
	adminBase
	Sub   cmd.SubCommand    `cmd:"item"`
	Name  string            `cmd:"name"`
	Count cmd.Optional[int] `cmd:"count"`
}

// Run gives any registered item by name, e.g. minecraft:iron_ore.
func (c adminItem) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, s, ok := c.session(src, o, tx)
	if !ok {
		return
	}
	name := c.Name
	if !strings.Contains(name, ":") {
		name = "minecraft:" + name
	}
	it, ok := world.ItemByName(name, 0)
	if !ok {
		o.Errorf(i18n.T(i18n.Default, "dragonfactory.admin.unknownItem"), name)
		return
	}
	st := item.NewStack(it, max(1, c.Count.LoadOr(1)))
	if added, _ := p.Inventory().AddItem(st); added < st.Count() {
		s.dropAt(tx, p.Position(), st.Grow(-added))
	}
	o.Printf("%s x%d", itemName(i18n.Of(p), st), st.Count())
}

// giveItems are the items /dfadmin give hands out: the game's own items
// and every machine, plus a water bucket.
func giveItems() map[string]func() item.Stack {
	out := map[string]func() item.Stack{
		"research_table":   func() item.Stack { return item.NewStack(ResearchTable{}, 1) },
		"drawer":           func() item.Stack { return item.NewStack(Drawer{}, 1) },
		"drawer_key":       func() item.Stack { return item.NewStack(DrawerKey{}, 1) },
		"upgrade_capacity": func() item.Stack { return item.NewStack(DrawerUpgrade{Kind: UpgradeCapacity}, 1) },
		"upgrade_void":     func() item.Stack { return item.NewStack(DrawerUpgrade{Kind: UpgradeVoid}, 1) },
		"hammer":           func() item.Stack { return item.NewStack(Hammer{}, 1) },
		"stone_hammer":     func() item.Stack { return item.NewStack(Hammer{Stone: true}, 1) },
		"string_mesh":      func() item.Stack { return item.NewStack(Mesh{Tier: "string"}, 1) },
		"barrel":           func() item.Stack { return item.NewStack(Barrel{Wood: "oak"}, 1) },
		"stone_barrel":     func() item.Stack { return item.NewStack(Barrel{Wood: stoneWood}, 1) },
		"wood_hopper":      func() item.Stack { return item.NewStack(newFlowHopper("wood_hopper", cube.FaceDown), 1) },
		"wood_hopper_reverse": func() item.Stack {
			return item.NewStack(newFlowHopper("wood_hopper_reverse", cube.FaceUp), 1)
		},
		"iron_hopper_reverse": func() item.Stack {
			return item.NewStack(newFlowHopper("iron_hopper_reverse", cube.FaceUp), 1)
		},
		"water": func() item.Stack {
			return item.NewStack(item.Bucket{Content: item.LiquidBucketContent(block.Water{Depth: 8, Still: true})}, 1)
		},
	}
	for _, k := range machineOrder {
		out[k] = func() item.Stack { return machineItem(k) }
	}
	for t := 1; t <= conduitTiers; t++ {
		out[fmt.Sprintf("energy_conduit_%d", t)] = func() item.Stack { return item.NewStack(EnergyConduit{Tier: t}, 1) }
		out[fmt.Sprintf("item_conduit_%d", t)] = func() item.Stack { return item.NewStack(ItemConduit{Tier: t}, 1) }
		out[fmt.Sprintf("fluid_conduit_%d", t)] = func() item.Stack { return item.NewStack(FluidConduit{Tier: t}, 1) }
	}
	out["mycelium_spores"] = func() item.Stack { return item.NewStack(MyceliumSpores{}, 1) }
	out["witch_water_bucket"] = func() item.Stack { return item.NewStack(WitchWaterBucket{}, 1) }
	for _, k := range crushedKinds {
		out["crushed_"+k] = func() item.Stack { return item.NewStack(CrushedBlock{Kind: k}, 1) }
	}
	for _, k := range pebbleKinds {
		out["pebble_"+k.kind] = func() item.Stack { return item.NewStack(Pebble{Kind: k.kind}, 1) }
	}
	for _, k := range oreKinds {
		out["ore_chunk_"+k.ore] = func() item.Stack { return item.NewStack(OreChunk{Ore: k.ore}, 1) }
	}
	for _, m := range newOreMetals {
		out[m+"_ore"] = func() item.Stack { return item.NewStack(OreBlock{Metal: m}, 1) }
		out["raw_"+m] = func() item.Stack { return item.NewStack(RawOre{Metal: m}, 1) }
	}
	for _, k := range toolKinds {
		for _, h := range headMaterials {
			for _, hd := range handleMaterials {
				out[fmt.Sprintf("tool_%s_%s_%s", k, h.id, hd.id)] = func() item.Stack {
					return describeTool(item.NewStack(TinkerTool{Kind: k, Head: h.id, Handle: hd.id}, 1))
				}
			}
		}
	}
	for _, c := range petalColours {
		out["petal_"+c] = func() item.Stack { return item.NewStack(Petal{Colour: c}, 1) }
	}
	for _, sd := range plantSeeds {
		out["seed_"+sd] = func() item.Stack { return item.NewStack(PlantSeed{Seed: sd}, 1) }
	}
	for _, m := range mobKinds {
		out["essence_"+m] = func() item.Stack { return item.NewStack(Essence{Mob: m}, 1) }
	}
	out["config_copier"] = func() item.Stack { return item.NewStack(ConfigCopier{}, 1) }
	for _, a := range tierAlloys {
		out["alloy_"+a] = func() item.Stack { return item.NewStack(TierAlloy{Kind: a}, 1) }
	}
	for l := 1; l <= 4; l++ {
		out[fmt.Sprintf("circuit_%d", l)] = func() item.Stack { return item.NewStack(Circuit{Level: l}, 1) }
	}
	for l := 1; l <= maxTier; l++ {
		out[fmt.Sprintf("kit_%d", l)] = func() item.Stack { return item.NewStack(TierKit{Level: l}, 1) }
	}
	for _, k := range researchEssenceKinds {
		out["research_essence_"+k] = func() item.Stack { return item.NewStack(ResearchEssence{Kind: k}, 1) }
	}
	for _, m := range dustMetals {
		out["metal_dust_"+m] = func() item.Stack { return item.NewStack(MetalDust{Metal: m}, 1) }
	}
	for _, m := range ingotMetals {
		out["ingot_"+m] = func() item.Stack { return item.NewStack(Ingot{Metal: m}, 1) }
	}
	return out
}

type giveEnum string

func (giveEnum) Type() string { return "df_item" }
func (giveEnum) Options(cmd.Source) []string {
	ids := slices.Sorted(maps.Keys(giveItems()))
	return ids
}

type adminGive struct {
	adminBase
	Sub   cmd.SubCommand    `cmd:"give"`
	Item  giveEnum          `cmd:"item"`
	Count cmd.Optional[int] `cmd:"count"`
}

func (c adminGive) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, s, ok := c.session(src, o, tx)
	if !ok {
		return
	}
	f, ok := giveItems()[string(c.Item)]
	if !ok {
		o.Errorf(i18n.T(i18n.Default, "dragonfactory.admin.unknownItem"), c.Item)
		return
	}
	n := max(1, c.Count.LoadOr(1))
	for range n {
		it := f()
		if added, _ := p.Inventory().AddItem(it); added < it.Count() {
			s.dropAt(tx, p.Position(), it.Grow(-added))
		}
	}
	o.Printf("%s x%d", itemName(i18n.Of(p), f()), n)
}

func registerAdmin(g *Game) {
	b := adminBase{g: g}
	cmd.Register(cmd.New("dfadmin", i18n.T(i18n.Default, "dragonfactory.admin.cmd"), nil,
		adminItem{adminBase: b}, adminGive{adminBase: b}, adminTPS{adminBase: b}, adminResearch{adminBase: b}, adminAdvance{adminBase: b}))
}

// adminTPS turns the server load line at the top left of the operator's
// HUD on or off (perf.Toggle), in the lobby too. It is kept until the
// server restarts.
type adminTPS struct {
	adminBase
	Sub cmd.SubCommand `cmd:"tps"`
}

func (c adminTPS) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p := src.(*player.Player)
	on := perf.Toggle(p.Name())
	// In the lobby, platform.ShowLoad shows or clears it within a second.
	if s, ok := c.g.sessionOf(tx.World()); ok {
		s.updateBoard(p)
	}
	if on {
		o.Printf("%s", i18n.T(i18n.Of(p), "dragonfactory.admin.perf.on"))
	} else {
		o.Printf("%s", i18n.T(i18n.Of(p), "dragonfactory.admin.perf.off"))
	}
}

// researchEnum names a research of the tree or all, reset, free.
type researchEnum string

func (researchEnum) Type() string { return "df_research" }
func (researchEnum) Options(cmd.Source) []string {
	opts := []string{"all", "reset", "free"}
	for _, r := range researchTree {
		opts = append(opts, r.ID)
	}
	return opts
}

// adminResearch completes a research (or all), forgets every research
// (reset), or opens every recipe until the instance closes (free; again to
// lock).
type adminResearch struct {
	adminBase
	Sub      cmd.SubCommand `cmd:"research"`
	Research researchEnum   `cmd:"research"`
}

func (c adminResearch) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	_, s, ok := c.session(src, o, tx)
	if !ok {
		return
	}
	switch c.Research {
	case "all":
		for i := range researchTree {
			if !s.researched(researchTree[i].ID) {
				s.completeResearch(tx, &researchTree[i])
			}
		}
		o.Printf("%s", i18n.T(i18n.Default, "dragonfactory.admin.research.all"))
	case "reset":
		s.state.Researched, s.state.ResearchPaid = nil, nil
		s.storeState()
		s.sendRecipes(tx)
		o.Printf("%s", i18n.T(i18n.Default, "dragonfactory.admin.research.cleared"))
	case "free":
		s.recipesFree = !s.recipesFree
		s.sendRecipes(tx)
		if s.recipesFree {
			o.Printf("%s", i18n.T(i18n.Default, "dragonfactory.admin.recipes.unlocked"))
		} else {
			o.Printf("%s", i18n.T(i18n.Default, "dragonfactory.admin.recipes.locked"))
		}
	default:
		r, ok := researchByID(string(c.Research))
		if !ok {
			o.Errorf(i18n.T(i18n.Default, "dragonfactory.admin.research.unknown"), c.Research)
			return
		}
		if !s.researched(r.ID) {
			s.completeResearch(tx, r)
		}
		o.Printf(i18n.T(i18n.Default, "dragonfactory.admin.research.done"), r.Title(i18n.Default))
	}
}

// advanceEnum names an advancement of the tree or all, reset.
type advanceEnum string

func (advanceEnum) Type() string { return "df_advancement" }
func (advanceEnum) Options(cmd.Source) []string {
	opts := []string{"all", "reset"}
	for _, a := range advancementTree {
		opts = append(opts, a.ID)
	}
	return opts
}

// adminAdvance completes an advancement (with the ones before it, or all)
// or forgets every advancement (reset).
type adminAdvance struct {
	adminBase
	Sub         cmd.SubCommand `cmd:"advance"`
	Advancement advanceEnum    `cmd:"advancement"`
}

func (c adminAdvance) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	_, s, ok := c.session(src, o, tx)
	if !ok {
		return
	}
	switch c.Advancement {
	case "all":
		var as []*Advancement
		for i := range advancementTree {
			if !s.advanced(advancementTree[i].ID) {
				as = append(as, &advancementTree[i])
			}
		}
		if len(as) > 0 {
			s.completeAdvancements(tx, as)
		}
		o.Printf("%s", i18n.T(i18n.Default, "dragonfactory.admin.advancement.all"))
	case "reset":
		s.state.Advancements = nil
		s.storeState()
		s.updateBoards(tx)
		o.Printf("%s", i18n.T(i18n.Default, "dragonfactory.admin.advancement.cleared"))
	default:
		a, ok := advancementByID(string(c.Advancement))
		if !ok {
			o.Errorf(i18n.T(i18n.Default, "dragonfactory.admin.advancement.unknown"), c.Advancement)
			return
		}
		var as []*Advancement
		for _, b := range advancementWithAncestors(a) {
			if !s.advanced(b.ID) {
				as = append(as, b)
			}
		}
		if len(as) > 0 {
			s.completeAdvancements(tx, as)
		}
		o.Printf(i18n.T(i18n.Default, "dragonfactory.admin.advancement.done"), a.Title(i18n.Default))
	}
}
