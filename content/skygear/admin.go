package skygear

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/perf"
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/content/skygear/conduit"
	"github.com/presentkim/skygear/content/skygear/farm"
	"github.com/presentkim/skygear/content/skygear/hopper"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/netstore"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/screens"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/storage"
	"github.com/presentkim/skygear/content/skygear/tinker"
)

// Operator tools (/sgadmin) for running the server: items, research and
// advancement progress, the load line. Only players for whom the game's
// operator check returns true see and run them (and /sgdebug, debug.go).

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
		o.Error(i18n.T(i18n.Of(p), "skygear.admin.needSave"))
	}
	return p, s, ok
}

// giveItems are the items /sgadmin give hands out: the game's own items
// and every machine under short ids, a water bucket, every other custom
// item by its id without the skygear: prefix, and every vanilla item by its
// full name (minecraft:<name>, minecraft:<name>:<meta> for a variant).
func giveItems() map[string]func() item.Stack {
	out := map[string]func() item.Stack{
		"research_table":   func() item.Stack { return item.NewStack(progress.ResearchTable{}, 1) },
		"drawer":           func() item.Stack { return item.NewStack(storage.Drawer{}, 1) },
		"drawer_key":       func() item.Stack { return item.NewStack(sgitem.DrawerKey{}, 1) },
		"upgrade_capacity": func() item.Stack { return item.NewStack(sgitem.DrawerUpgrade{Kind: sgitem.UpgradeCapacity}, 1) },
		"upgrade_void":     func() item.Stack { return item.NewStack(sgitem.DrawerUpgrade{Kind: sgitem.UpgradeVoid}, 1) },
		"hammer":           func() item.Stack { return item.NewStack(sgitem.Hammer{}, 1) },
		"stone_hammer":     func() item.Stack { return item.NewStack(sgitem.Hammer{Stone: true}, 1) },
		"silkworm":         func() item.Stack { return item.NewStack(farm.Silkworm{}, 1) },
		"cooked_silkworm":  func() item.Stack { return item.NewStack(farm.CookedSilkworm{}, 1) },
		"silkworm_leaves": func() item.Stack {
			return item.NewStack(farm.SilkwormLeaves{Cocoon: farm.CocoonMax, Persistent: true}, 1)
		},
		"string_mesh":  func() item.Stack { return item.NewStack(sgitem.Mesh{Tier: "string"}, 1) },
		"barrel":       func() item.Stack { return item.NewStack(farm.Barrel{Wood: "oak"}, 1) },
		"stone_barrel": func() item.Stack { return item.NewStack(farm.Barrel{Wood: farm.StoneWood}, 1) },
		"wood_hopper":  func() item.Stack { return item.NewStack(hopper.NewFlowHopper("wood_hopper", cube.FaceDown), 1) },
		"wood_hopper_reverse": func() item.Stack {
			return item.NewStack(hopper.NewFlowHopper("wood_hopper_reverse", cube.FaceUp), 1)
		},
		"iron_hopper_reverse": func() item.Stack {
			return item.NewStack(hopper.NewFlowHopper("iron_hopper_reverse", cube.FaceUp), 1)
		},
		"water": func() item.Stack {
			return item.NewStack(item.Bucket{Content: item.LiquidBucketContent(block.Water{Depth: 8, Still: true})}, 1)
		},
	}
	for _, k := range machine.Order {
		out[k] = func() item.Stack { return machine.Item(k) }
	}
	for t := 1; t <= conduit.Tiers; t++ {
		out[fmt.Sprintf("energy_conduit_%d", t)] = func() item.Stack { return item.NewStack(conduit.EnergyConduit{Tier: t}, 1) }
		out[fmt.Sprintf("item_conduit_%d", t)] = func() item.Stack { return item.NewStack(conduit.ItemConduit{Tier: t}, 1) }
		out[fmt.Sprintf("fluid_conduit_%d", t)] = func() item.Stack { return item.NewStack(conduit.FluidConduit{Tier: t}, 1) }
	}
	for t, n := range []string{"1k", "4k", "16k", "64k"} {
		out["storage_disk_"+n] = func() item.Stack { return item.NewStack(netstore.Disk{Tier: t}, 1) }
	}
	out["pattern"] = func() item.Stack { return item.NewStack(netstore.Pattern{}, 1) }
	out["speed_upgrade"] = func() item.Stack { return item.NewStack(netstore.Upgrade{}, 1) }
	out["stack_upgrade"] = func() item.Stack { return item.NewStack(netstore.Upgrade{Stack: true}, 1) }
	out["mycelium_spores"] = func() item.Stack { return item.NewStack(sgitem.MyceliumSpores{}, 1) }
	out["witch_water_bucket"] = func() item.Stack { return item.NewStack(sgitem.WitchWaterBucket{}, 1) }
	for _, k := range sgitem.CrushedKinds {
		out["crushed_"+k] = func() item.Stack { return item.NewStack(sgitem.CrushedBlock{Kind: k}, 1) }
	}
	for _, k := range sgitem.PebbleKinds {
		out["pebble_"+k.Kind] = func() item.Stack { return item.NewStack(sgitem.Pebble{Kind: k.Kind}, 1) }
	}
	for _, k := range sgitem.OreKinds {
		out["ore_chunk_"+k.Ore] = func() item.Stack { return item.NewStack(sgitem.OreChunk{Ore: k.Ore}, 1) }
	}
	for _, m := range sgitem.NewOreMetals {
		out[m+"_ore"] = func() item.Stack { return item.NewStack(sgitem.OreBlock{Metal: m}, 1) }
		out["raw_"+m] = func() item.Stack { return item.NewStack(sgitem.RawOre{Metal: m}, 1) }
	}
	for _, k := range tinker.MadeKinds {
		for _, h := range tinker.HeadMaterials {
			for _, hd := range tinker.HandleMaterials {
				out[fmt.Sprintf("tool_%s_%s_%s", k, h.ID, hd.ID)] = func() item.Stack {
					return tinker.DescribeTool(item.NewStack(tinker.Tool{Kind: k, Head: h.ID, Handle: hd.ID}, 1))
				}
			}
		}
	}
	for _, c := range sgitem.PetalColours {
		out["petal_"+c] = func() item.Stack { return item.NewStack(sgitem.Petal{Colour: c}, 1) }
	}
	for _, sd := range farm.LivePlantSeeds() {
		out["seed_"+sd] = func() item.Stack { return item.NewStack(farm.PlantSeed{Seed: sd}, 1) }
	}
	for _, m := range sgitem.MobKinds {
		out["essence_"+m] = func() item.Stack { return item.NewStack(sgitem.Essence{Mob: m}, 1) }
	}
	out["config_copier"] = func() item.Stack { return item.NewStack(sgitem.ConfigCopier{}, 1) }
	for _, t := range sgitem.CrookTiers {
		out[t.Name+"_crook"] = func() item.Stack { return item.NewStack(sgitem.Crook{Tier: t.Name}, 1) }
	}
	for _, t := range sgitem.WandTiers {
		out[t+"_wand"] = func() item.Stack { return item.NewStack(sgitem.NewWand(t), 1) }
	}
	for _, a := range sgitem.TierAlloys {
		out["alloy_"+a] = func() item.Stack { return item.NewStack(sgitem.TierAlloy{Kind: a}, 1) }
	}
	for l := 1; l <= 4; l++ {
		out[fmt.Sprintf("circuit_%d", l)] = func() item.Stack { return item.NewStack(sgitem.Circuit{Level: l}, 1) }
	}
	for l := 1; l <= machine.MaxTier; l++ {
		out[fmt.Sprintf("kit_%d", l)] = func() item.Stack { return item.NewStack(sgitem.TierKit{Level: l}, 1) }
	}
	for _, k := range sgitem.ResearchEssenceKinds {
		out["research_essence_"+k] = func() item.Stack { return item.NewStack(sgitem.ResearchEssence{Kind: k}, 1) }
	}
	for _, m := range sgitem.DustMetals {
		out["metal_dust_"+m] = func() item.Stack { return item.NewStack(sgitem.MetalDust{Metal: m}, 1) }
	}
	for _, m := range sgitem.IngotMetals {
		out["ingot_"+m] = func() item.Stack { return item.NewStack(sgitem.Ingot{Metal: m}, 1) }
	}
	// the relics at every grade: relic_<kind> is grade 1, relic_<kind>_<grade>
	for _, k := range append(append([]string{}, sgitem.RelicWeapons...), sgitem.RelicCharms...) {
		out["relic_"+k] = func() item.Stack { return sgitem.RelicStack(k, 1) }
		for g := 1; g <= sgitem.RelicMaxGrade; g++ {
			out[fmt.Sprintf("relic_%s_%d", k, g)] = func() item.Stack { return sgitem.RelicStack(k, g) }
		}
	}
	// every other item of the game, by its id without the skygear: prefix,
	// but the screens' icons, the retired items and those listed above
	// under another id
	listed := map[string]bool{}
	for _, f := range out {
		listed[encodedKey(f().Item())] = true
	}
	for _, it := range world.CustomItems() {
		name, meta := it.EncodeItem()
		id, ok := strings.CutPrefix(name, "skygear:")
		if !ok || meta != 0 || out[id] != nil || listed[encodedKey(it)] || slices.ContainsFunc(giveSkip, func(pre string) bool { return strings.HasPrefix(id, pre) }) {
			continue
		}
		out[id] = func() item.Stack { return item.NewStack(it, 1) }
	}
	// every vanilla item by its full name, so the game's own ids above
	// (barrel, water ...) never hide one
	for _, it := range world.Items() {
		name, meta := it.EncodeItem()
		if !strings.HasPrefix(name, "minecraft:") || item.NewStack(it, 1).Empty() {
			continue
		}
		key := name
		if meta != 0 {
			key = fmt.Sprintf("%s:%d", name, meta)
		}
		if out[key] == nil {
			out[key] = func() item.Stack { return item.NewStack(it, 1) }
		}
	}
	return out
}

// encodedKey is an item's saved id and meta, to tell whether an item is
// already listed under another id.
func encodedKey(it world.Item) string {
	name, meta := it.EncodeItem()
	return fmt.Sprintf("%s:%d", name, meta)
}

// giveSkip are the id prefixes of the custom items /sgadmin give leaves out:
// the screens' icons and the retired items. Items listed under another id
// are left out by their encoding instead.
var giveSkip = []string{
	"cycle_", "energy_gauge_", "progress_", "liquid_", "machine_info", "fairy_board", // screen icons
	"dimension_gate", "seed_ess_", "tool_sword_", // retired (old saves still load them)
}

type giveEnum string

func (giveEnum) Type() string { return "sg_item" }
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
		o.Errorf(i18n.T(i18n.Default, "skygear.admin.unknownItem"), c.Item)
		return
	}
	n := max(1, c.Count.LoadOr(1))
	for range n {
		it := f()
		if added, _ := p.Inventory().AddItem(it); added < it.Count() {
			s.DropAt(tx, p.Position(), it.Grow(-added))
		}
	}
	o.Printf("%s x%d", naming.ItemName(i18n.Of(p), f()), n)
}

func registerAdmin(g *Game) {
	b := adminBase{g: g}
	cmd.Register(cmd.New("sgadmin", i18n.T(i18n.Default, "skygear.admin.cmd"), nil,
		adminGive{adminBase: b}, adminTPS{adminBase: b}, adminResearch{adminBase: b}, adminAdvance{adminBase: b}))
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
		screens.UpdateBoard(s.Session, p)
	}
	if on {
		o.Printf("%s", i18n.T(i18n.Of(p), "skygear.admin.perf.on"))
	} else {
		o.Printf("%s", i18n.T(i18n.Of(p), "skygear.admin.perf.off"))
	}
}

// researchEnum names a research of the tree or all, reset, free.
type researchEnum string

func (researchEnum) Type() string { return "sg_research" }
func (researchEnum) Options(cmd.Source) []string {
	opts := []string{"all", "reset", "free"}
	for _, r := range progress.Tree {
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
		for i := range progress.Tree {
			if !progress.Researched(s.Session, progress.Tree[i].ID) {
				progress.Complete(s.Session, tx, &progress.Tree[i])
			}
		}
		o.Printf("%s", i18n.T(i18n.Default, "skygear.admin.research.all"))
	case "reset":
		progress.StateOf(s.Session).Researched, progress.StateOf(s.Session).ResearchPaid = nil, nil
		s.MarkDirty()
		progress.SendRecipes(s.Session, tx)
		o.Printf("%s", i18n.T(i18n.Default, "skygear.admin.research.cleared"))
	case "free":
		progress.LocalOf(s.Session).RecipesFree = !progress.LocalOf(s.Session).RecipesFree
		progress.SendRecipes(s.Session, tx)
		if progress.LocalOf(s.Session).RecipesFree {
			o.Printf("%s", i18n.T(i18n.Default, "skygear.admin.recipes.unlocked"))
		} else {
			o.Printf("%s", i18n.T(i18n.Default, "skygear.admin.recipes.locked"))
		}
	default:
		r, ok := progress.ByID(string(c.Research))
		if !ok {
			o.Errorf(i18n.T(i18n.Default, "skygear.admin.research.unknown"), c.Research)
			return
		}
		if !progress.Researched(s.Session, r.ID) {
			progress.Complete(s.Session, tx, r)
		}
		o.Printf(i18n.T(i18n.Default, "skygear.admin.research.done"), r.Title(i18n.Default))
	}
}

// advanceEnum names an advancement of the tree or all, reset.
type advanceEnum string

func (advanceEnum) Type() string { return "sg_advancement" }
func (advanceEnum) Options(cmd.Source) []string {
	opts := []string{"all", "reset"}
	for _, a := range advancement.Tree {
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
		var as []*advancement.Advancement
		for i := range advancement.Tree {
			if !progress.Advanced(s.Session, advancement.Tree[i].ID) {
				as = append(as, &advancement.Tree[i])
			}
		}
		if len(as) > 0 {
			progress.CompleteAdvancements(s.Session, tx, as)
		}
		o.Printf("%s", i18n.T(i18n.Default, "skygear.admin.advancement.all"))
	case "reset":
		progress.StateOf(s.Session).Advancements = nil
		progress.StateOf(s.Session).Unseen = nil
		s.MarkDirty()
		screens.UpdateBoards(s.Session, tx)
		o.Printf("%s", i18n.T(i18n.Default, "skygear.admin.advancement.cleared"))
	default:
		a, ok := advancement.ByID(string(c.Advancement))
		if !ok {
			o.Errorf(i18n.T(i18n.Default, "skygear.admin.advancement.unknown"), c.Advancement)
			return
		}
		var as []*advancement.Advancement
		for _, b := range advancement.WithAncestors(a) {
			if !progress.Advanced(s.Session, b.ID) {
				as = append(as, b)
			}
		}
		if len(as) > 0 {
			progress.CompleteAdvancements(s.Session, tx, as)
		}
		o.Printf(i18n.T(i18n.Default, "skygear.admin.advancement.done"), a.Title(i18n.Default))
	}
}
