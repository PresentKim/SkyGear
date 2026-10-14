package dragonfactory

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/account"
	"github.com/presentkim/dragonfactory/internal/i18n"
	"github.com/presentkim/dragonfactory/internal/ui"
)

// Research (docs/design/02-dragonfactory.md §3.4): every crafting recipe is
// locked but oak planks, the crafting table and the research table. A
// research opens the recipes making its results for everyone on the
// island at once (recipes.go). Researches form a tree: each needs the ones
// before it. A research is paid with items, delivered from the inventory
// at the research table (in parts if need be). Trees and costs are drafts.

// Research is one node of the research tree.
type Research struct {
	ID string
	// Needs are the researches that must be done first.
	Needs []string
	// Cost are the items to deliver.
	Cost []Requirement
	// Unlocks are the item names (without "minecraft:") whose recipes the
	// research opens; every variant of a name opens together.
	Unlocks []string
}

var (
	oakPlanks = block.Planks{Wood: block.OakWood()}
	stick     = item.Stick{}
)

// Title is the name of the research in lang (the catalog, i18n).
func (r *Research) Title(lang string) string { return i18n.T(lang, "research."+r.ID+".name") }

// langOf is the catalog language of p (the default for a nil p).
func langOf(p *player.Player) string {
	if p == nil {
		return i18n.Default
	}
	return i18n.For(p)
}

// researchTree is every research in menu order; a research's needs come
// before it.
var researchTree = []Research{
	{ID: "stick", Cost: []Requirement{{oakPlanks, 4}}, Unlocks: []string{"stick"}},
	{ID: "pebbles", Needs: []string{"stick"}, Cost: []Requirement{{Pebble{Kind: PebbleStone}, 4}},
		Unlocks: []string{"cobblestone", "andesite", "diorite", "granite"}},
	{ID: "wood_tools", Needs: []string{"stick"}, Cost: []Requirement{{oakPlanks, 8}, {stick, 4}},
		Unlocks: []string{"wooden_pickaxe", "wooden_axe", "wooden_shovel", "wooden_hoe", "wooden_sword"}},
	{ID: "chest", Needs: []string{"stick"}, Cost: []Requirement{{oakPlanks, 64}}, Unlocks: []string{"chest"}},
	{ID: "wood_hopper", Needs: []string{"chest"}, Cost: []Requirement{{oakPlanks, 48}},
		Unlocks: []string{"dragonfactory:wood_hopper", "dragonfactory:wood_hopper_reverse"}},
	{ID: "slab", Needs: []string{"wood_tools"}, Cost: []Requirement{{oakPlanks, 16}}, Unlocks: []string{"oak_slab"}},
	{ID: "compost", Needs: []string{"slab"}, Cost: []Requirement{{oakPlanks, 32}}, Unlocks: []string{"composter"}},
	{ID: "barrel", Needs: []string{"compost"}, Cost: []Requirement{{oakPlanks, 32}, {stick, 4}}, Unlocks: []string{"dragonfactory:barrel_oak"}},
	{ID: "furnace", Needs: []string{"pebbles"}, Cost: []Requirement{{block.Cobblestone{}, 32}}, Unlocks: []string{"furnace"}},
	{ID: "stone_tools", Needs: []string{"wood_tools", "pebbles"}, Cost: []Requirement{{block.Cobblestone{}, 32}, {stick, 16}},
		Unlocks: []string{"stone_pickaxe", "stone_axe", "stone_shovel", "stone_hoe", "stone_sword"}},
	{ID: "string", Needs: []string{"stone_tools"}, Cost: []Requirement{{BarkFiber{}, 8}}, Unlocks: []string{"string"}},
	{ID: "hammer", Needs: []string{"pebbles"}, Cost: []Requirement{{block.Cobblestone{}, 32}, {stick, 8}}, Unlocks: []string{"dragonfactory:hammer"}},
	{ID: "sieve", Needs: []string{"string", "hammer"}, Cost: []Requirement{{block.String{}, 8}, {oakPlanks, 16}}, Unlocks: []string{"dragonfactory:sieve", "dragonfactory:mesh_string", "dragonfactory:mesh_flint"}},
	{ID: "torch", Needs: []string{"furnace"}, Cost: []Requirement{{item.Charcoal{}, 16}, {stick, 16}}, Unlocks: []string{"torch"}},
	{ID: "woodwork", Needs: []string{"chest"}, Cost: []Requirement{{oakPlanks, 128}, {stick, 32}},
		Unlocks: []string{"oak_stairs", "oak_fence", "fence_gate", "wooden_door", "trapdoor", "ladder", "bowl"}},
	{ID: "masonry", Needs: []string{"furnace"}, Cost: []Requirement{{block.Stone{}, 64}, {block.Cobblestone{}, 64}},
		Unlocks: []string{"cobblestone_slab", "stone_stairs", "cobblestone_wall", "normal_stone_slab", "normal_stone_stairs", "stone_bricks"}},
	{ID: "glasswork", Needs: []string{"furnace"}, Cost: []Requirement{{block.Glass{}, 32}}, Unlocks: []string{"glass_pane", "glass_bottle"}},
	{ID: "iron", Needs: []string{"furnace", "sieve"}, Cost: []Requirement{{OreChunk{Ore: OreIron}, 16}}, Unlocks: []string{"iron_ore", "iron_ingot", "iron_block", "iron_nugget", "dragonfactory:mesh_iron"}},
	{ID: "hopper", Needs: []string{"iron", "chest"}, Cost: []Requirement{{item.IronIngot{}, 10}, {block.NewChest(), 4}}, Unlocks: []string{"hopper", "dragonfactory:iron_hopper_reverse", "dragonfactory:wood_hopper_reverse"}},
	{ID: "crusher", Needs: []string{"hammer", "iron"}, Cost: []Requirement{{block.Cobblestone{}, 128}, {item.IronIngot{}, 8}}, Unlocks: []string{"dragonfactory:crusher"}},
	{ID: "auto_sieve", Needs: []string{"sieve", "hopper"}, Cost: []Requirement{{item.IronIngot{}, 16}, {block.String{}, 16}}, Unlocks: []string{"dragonfactory:auto_sieve"}},
	{ID: "iron_tools", Needs: []string{"iron", "stone_tools"}, Cost: []Requirement{{item.IronIngot{}, 48}},
		Unlocks: []string{"iron_pickaxe", "iron_axe", "iron_shovel", "iron_hoe", "iron_sword", "bucket", "shears"}},
	{ID: "auto_miner", Needs: []string{"iron_tools", "hopper"}, Cost: []Requirement{{item.IronIngot{}, 32}, {block.NewHopper(), 2}}, Unlocks: []string{"dragonfactory:auto_miner"}},
	{ID: "drawer", Needs: []string{"chest"}, Cost: []Requirement{{oakPlanks, 64}, {block.NewChest(), 2}}, Unlocks: []string{"dragonfactory:drawer", "dragonfactory:drawer_key"}},
	{ID: "drawer_upgrades", Needs: []string{"drawer", "iron"}, Cost: []Requirement{{item.IronIngot{}, 16}},
		Unlocks: []string{"dragonfactory:drawer_upgrade_capacity", "dragonfactory:drawer_upgrade_void"}},
	{ID: "farming", Needs: []string{"wood_tools"}, Cost: []Requirement{{item.Wheat{}, 32}}, Unlocks: []string{"bread", "hay_block"}},
	{ID: "harvester", Needs: []string{"farming", "hopper"}, Cost: []Requirement{{item.IronIngot{}, 32}}, Unlocks: []string{"dragonfactory:harvester"}},
	{ID: "clay", Needs: []string{"furnace"}, Cost: []Requirement{{item.ClayBall{}, 32}}, Unlocks: []string{"clay", "brick_block"}},
	{ID: "gold", Needs: []string{"iron"}, Cost: []Requirement{{OreChunk{Ore: OreGold}, 16}}, Unlocks: []string{"gold_ore", "gold_ingot", "gold_block", "gold_nugget", "dragonfactory:mesh_gold"}},
	{ID: "mesh_diamond", Needs: []string{"gold"}, Cost: []Requirement{{item.Diamond{}, 4}}, Unlocks: []string{"dragonfactory:mesh_diamond"}},
	{ID: "mesh_netherite", Needs: []string{"mesh_diamond"}, Cost: []Requirement{{item.NetheriteScrap{}, 4}}, Unlocks: []string{"dragonfactory:mesh_netherite", "netherite_ingot"}},
	{ID: "copper", Needs: []string{"furnace", "sieve"}, Cost: []Requirement{{OreChunk{Ore: OreCopper}, 16}}, Unlocks: []string{"copper_ore"}},
	{ID: "harvest", Needs: []string{"farming"}, Cost: []Requirement{{item.MelonSlice{}, 32}, {block.Pumpkin{}, 8}},
		Unlocks: []string{"melon_block", "pumpkin_pie", "melon_seeds", "pumpkin_seeds"}},
	{ID: "kelp", Needs: []string{"furnace"}, Cost: []Requirement{{item.DriedKelp{}, 32}}, Unlocks: []string{"dried_kelp_block"}},
}

// baseUnlocks are the results open without research.
var baseUnlocks = []string{"oak_planks", "crafting_table", "dragonfactory:research_table"}

// researchByID returns a research of the tree.
func researchByID(id string) (*Research, bool) {
	for i := range researchTree {
		if researchTree[i].ID == id {
			return &researchTree[i], true
		}
	}
	return nil, false
}

// unlockName is the item name of an unlock as items encode it.
func unlockName(n string) string {
	if strings.Contains(n, ":") {
		return n
	}
	return "minecraft:" + n
}

// researched reports whether the save has done a research.
func (s *session) researched(id string) bool { return s.state.Researched[id] }

// researchOpen reports whether a research can be done: not done yet and
// everything it needs is done.
func (s *session) researchOpen(r *Research) bool {
	if s.researched(r.ID) {
		return false
	}
	for _, n := range r.Needs {
		if !s.researched(n) {
			return false
		}
	}
	return true
}

// researchPaid is how many of an item of a research's cost an older save
// had already delivered in parts (partial delivery is gone: a research is
// paid whole, at once; the count only keeps what older saves gave).
func (s *session) researchPaid(r *Research, it world.Item) int {
	return s.state.ResearchPaid[r.ID][itemKey(it)]
}

// researchNeed is how many of an item of a research's cost are still to be
// spent.
func (s *session) researchNeed(r *Research, c Requirement) int {
	return max(0, c.Count-s.researchPaid(r, c.Item))
}

// researchMissing lists what p lacks to do a research now: the cost items
// with how many more they need.
func (s *session) researchMissing(p *player.Player, r *Research) []Requirement {
	var missing []Requirement
	for _, c := range r.Cost {
		if short := s.researchNeed(r, c) - countPlain(p.Inventory(), c.Item); short > 0 {
			missing = append(missing, Requirement{c.Item, short})
		}
	}
	return missing
}

// doResearch spends the whole cost from p's inventory and completes the
// research. A research is never paid in parts: when p lacks anything it
// takes nothing and returns what is missing.
func (s *session) doResearch(p *player.Player, tx *world.Tx, r *Research) (missing []Requirement, done bool) {
	if !s.researchOpen(r) {
		return nil, false
	}
	if missing = s.researchMissing(p, r); len(missing) > 0 {
		return missing, false
	}
	for _, c := range r.Cost {
		takePlain(p.Inventory(), c.Item, s.researchNeed(r, c))
	}
	s.completeResearch(tx, r)
	account.Grant(p, achResearch)
	return nil, true
}

// completeResearch marks a research done and opens its recipes for
// everyone on the island.
func (s *session) completeResearch(tx *world.Tx, r *Research) {
	if s.state.Researched == nil {
		s.state.Researched = map[string]bool{}
	}
	s.state.Researched[r.ID] = true
	delete(s.state.ResearchPaid, r.ID)
	s.storeState()
	s.sendRecipes(tx)
	s.updateBoards(tx)
	for e := range tx.Players() {
		p := e.(*player.Player)
		after := researchText(langOf(p), r.ID).After
		p.Messagef(i18n.T(i18n.Of(p), "dragonfactory.research.completeResearch.1"), r.Title(langOf(p)), s.unlockNames(r))
		if after != "" {
			p.Messagef(i18n.T(i18n.Of(p), "dragonfactory.research.completeResearch.2"), after)
		}
	}
}

// unlockNames lists a research's results in Korean.
func (s *session) unlockNames(r *Research) string {
	var names []string
	for _, n := range r.Unlocks {
		if it, ok := world.ItemByName(unlockName(n), 0); ok {
			names = append(names, itemName(item.NewStack(it, 1)))
		} else {
			names = append(names, n)
		}
	}
	return strings.Join(names, ", ")
}

// researchIcon is a research's form icon: its first result.
func researchIcon(r *Research) string {
	if it, ok := world.ItemByName(unlockName(r.Unlocks[0]), 0); ok {
		return iconOf(item.NewStack(it, 1))
	}
	return iconMission
}

// showResearch shows the research list on the tab of researches still to do.
func (s *session) showResearch(p *player.Player) { s.showResearchTab(p, false) }

// showResearchTab shows the research list as a grid of item slots (the
// representative item of each research) under two tabs, the researches
// still to do and the done ones.
func (s *session) showResearchTab(p *player.Player, done bool) {
	body, slots, actions := s.researchList(p, done)
	ui.ItemGrid(p, i18n.T(i18n.Of(p), "dragonfactory.research.showResearchTab.1"), body, slots, actions...)
}

// researchTip is the hover text of a research in the list: its name in
// colour and the short description.
func researchTip(lang, colour string, r *Research) string {
	tip := colour + r.Title(lang)
	if short := researchText(lang, r.ID).Short; short != "" {
		tip += "\n§7" + short
	}
	return tip
}

// researchList builds the research list of a tab. A research is hidden until
// everything before it is done, so the tree is found step by step. The
// tooltip of a research to do is its name in yellow, in green when p has
// all it spends and can do it now. p may be nil.
func (s *session) researchList(p *player.Player, done bool) (string, []ui.ItemSlot, []ui.Option) {
	var slots []ui.ItemSlot
	todoN, doneN := 0, 0
	for i := range researchTree {
		r := &researchTree[i]
		show := func(p *player.Player, _ *world.Tx) { s.showResearchNode(p, r) }
		switch {
		case s.researched(r.ID):
			doneN++
			if done {
				slots = append(slots, ui.ItemSlot{Icon: researchIcon(r), Count: "§a✔", Tip: researchTip(langOf(p), "§7", r), Run: show})
			}
		case s.researchOpen(r):
			todoN++
			if !done {
				colour := "§e"
				if p != nil && len(s.researchMissing(p, r)) == 0 {
					colour = "§a"
				}
				slots = append(slots, ui.ItemSlot{Icon: researchIcon(r), Tip: researchTip(langOf(p), colour, r), Run: show})
			}
		}
	}
	body := i18n.T(i18n.Of(p), "dragonfactory.research.researchList.1")
	if len(slots) == 0 {
		if done {
			body += i18n.T(i18n.Of(p), "dragonfactory.research.researchList.2")
		} else {
			body += i18n.T(i18n.Of(p), "dragonfactory.research.researchList.3")
		}
	}
	tabs := []ui.Option{
		ui.Tab(fmt.Sprintf(i18n.T(i18n.Of(p), "dragonfactory.research.researchList.4"), todoN), "", !done, func(p *player.Player, tx *world.Tx) {
			if tx.World() == s.world {
				s.showResearchTab(p, false)
			}
		}),
		ui.Tab(fmt.Sprintf(i18n.T(i18n.Of(p), "dragonfactory.research.researchList.5"), doneN), "", done, func(p *player.Player, tx *world.Tx) {
			if tx.World() == s.world {
				s.showResearchTab(p, true)
			}
		}),
		s.backOption(),
	}
	return body, slots, tabs
}

// showResearchNode shows one research as an item detail form
// (ui.ItemDetail): on the left its big item, name, the research button (or
// "연구 완료") and what it is for; on the right the materials it spends and
// the recipes it opens as item slots (the count at the bottom right, the
// name and owned count in the tooltip).
func (s *session) showResearchNode(p *player.Player, r *Research) {
	d, rows, opts := s.researchNode(p, r)
	ui.ItemDetail(p, i18n.T(i18n.Of(p), "dragonfactory.research.showResearchNode.1")+r.Title(langOf(p)), d, rows, opts...)
}

// researchNode builds the screen of a research. p may be nil (the owned
// counts are then left out).
func (s *session) researchNode(p *player.Player, r *Research) (ui.Detail, []ui.SlotRow, []ui.Option) {
	txt := researchText(langOf(p), r.ID)
	done := s.researched(r.ID)
	var b strings.Builder
	if txt.About != "" {
		fmt.Fprintf(&b, "§f%s\n", txt.About)
	}
	d := ui.Detail{Icon: researchIcon(r), Name: "§6" + r.Title(langOf(p))}
	var opts []ui.Option
	switch {
	case done:
		d.Status = i18n.T(i18n.Of(p), "dragonfactory.research.researchNode.1")
		if txt.After != "" {
			fmt.Fprintf(&b, i18n.T(i18n.Of(p), "dragonfactory.research.researchNode.2"), txt.After)
		}
	case s.researchOpen(r):
		if txt.Hint != "" {
			fmt.Fprintf(&b, i18n.T(i18n.Of(p), "dragonfactory.research.researchNode.3"), txt.Hint)
		}
		opts = append(opts, s.opt(i18n.T(i18n.Of(p), "dragonfactory.research.researchNode.4"), iconMission, func(p *player.Player, tx *world.Tx) {
			missing, ok := s.doResearch(p, tx, r)
			switch {
			case ok:
				ui.Flash(p, "%s", i18n.T(i18n.Of(p), "dragonfactory.research.researchNode.5"))
				s.showResearch(p)
				return
			case len(missing) > 0:
				var names []string
				for _, m := range missing {
					names = append(names, costText(m))
				}
				ui.Notify(p, i18n.T(i18n.Of(p), "dragonfactory.research.researchNode.6"), strings.Join(names, ", "))
			}
			s.showResearchNode(p, r)
		}))
	}
	d.Body = b.String()
	again := func(p *player.Player, _ *world.Tx) { s.showResearchNode(p, r) }
	// The materials: the count shown is what the research spends (green
	// once p has enough, or the research is done); the tooltip says what
	// p has.
	var cost []ui.ItemSlot
	for _, c := range r.Cost {
		need := s.researchNeed(r, c)
		st := item.NewStack(c.Item, 1)
		mark, tip := "§f", fmt.Sprintf(i18n.T(i18n.Of(p), "dragonfactory.research.researchNode.7"), itemName(st), need)
		switch {
		case done:
			mark, tip = "§a", fmt.Sprintf(i18n.T(i18n.Of(p), "dragonfactory.research.researchNode.8"), itemName(st), c.Count)
			need = c.Count
		case p != nil:
			have := countPlain(p.Inventory(), c.Item)
			if have >= need {
				mark = "§a"
			}
			tip += fmt.Sprintf(i18n.T(i18n.Of(p), "dragonfactory.research.researchNode.9"), mark, have)
		}
		cost = append(cost, ui.ItemSlot{Icon: iconOf(st), Count: mark + strconv.Itoa(need), Tip: tip, Run: again})
	}
	// The recipes it opens.
	var opens []ui.ItemSlot
	for _, n := range r.Unlocks {
		it, ok := world.ItemByName(unlockName(n), 0)
		if !ok {
			continue
		}
		st := item.NewStack(it, 1)
		opens = append(opens, ui.ItemSlot{Icon: iconOf(st), Tip: "§f" + itemName(st) + i18n.T(i18n.Of(p), "dragonfactory.research.researchNode.10"), Run: again})
	}
	rows := []ui.SlotRow{{Heading: i18n.T(i18n.Of(p), "dragonfactory.research.researchNode.11"), Slots: cost}, {Heading: i18n.T(i18n.Of(p), "dragonfactory.research.researchNode.12"), Slots: opens}}
	back := ui.Back(i18n.T(i18n.Of(p), "dragonfactory.research.researchNode.13"), func(p *player.Player, _ *world.Tx) { s.showResearch(p) })
	return d, rows, append(opts, s.opt(back.Label, back.Icon, back.Run))
}

// ResearchTable is the block where research is done. It is crafted from two
// crafting tables (open from the start).
type ResearchTable struct{}

var researchTableHash = block.NextHash()

func (ResearchTable) EncodeItem() (string, int16) { return "dragonfactory:research_table", 0 }
func (ResearchTable) EncodeBlock() (string, map[string]any) {
	return "dragonfactory:research_table", nil
}
func (ResearchTable) Hash() (uint64, uint64)             { return researchTableHash, 0 }
func (ResearchTable) Model() world.BlockModel            { return model.Solid{} }
func (ResearchTable) Name() string                       { return i18n.T(i18n.Default, "name.ResearchTable") }
func (ResearchTable) Category() category.Category        { return category.Items() }
func (ResearchTable) Properties() customblock.Properties { return cubeProperties("research_table") }
func (t ResearchTable) BreakInfo() block.BreakInfo {
	return block.BreakInfo{
		Hardness: 2.5, BlastResistance: 2.5,
		Harvestable: func(item.Tool) bool { return true },
		Effective:   customBlockEffective,
		Drops:       func(item.Tool, []item.Enchantment) []item.Stack { return []item.Stack{item.NewStack(t, 1)} },
	}
}

// Activate opens the research tree.
func (ResearchTable) Activate(_ cube.Pos, _ cube.Face, tx *world.Tx, u item.User, _ *item.UseContext) bool {
	p, ok := u.(*player.Player)
	if !ok {
		return false
	}
	s, ok := activeGame.sessionOf(tx.World())
	if !ok {
		return false
	}
	if !s.debounced(p) {
		s.showResearch(p)
	}
	return true
}

func init() {
	world.RegisterBlock(ResearchTable{})
	world.RegisterItem(ResearchTable{})
	table := item.NewStack(block.CraftingTable{}, 1)
	recipe.Register(recipe.NewShapeless([]recipe.Item{table, table}, item.NewStack(ResearchTable{}, 1), "crafting_table"))
	gameRecipes()
}

// gameRecipes registers the crafting recipes of the game's own items, open
// once researched. Drafts.
func gameRecipes() {
	st := func(it world.Item) item.Stack { return item.NewStack(it, 1) }
	n := func(it world.Item, count int) []recipe.Item {
		out := make([]recipe.Item, count)
		for i := range out {
			out[i] = st(it)
		}
		return out
	}
	add := func(out item.Stack, in ...[]recipe.Item) {
		recipe.Register(recipe.NewShapeless(slices.Concat(in...), out, "crafting_table"))
	}
	// shaped adds a recipe that crafts only as arranged (cells row by row,
	// nil is an empty cell), not mirrored either.
	shaped := func(out item.Stack, w, h int, cells ...world.Item) {
		in := make([]recipe.Item, len(cells))
		for i, c := range cells {
			in[i] = item.Stack{}
			if c != nil {
				in[i] = st(c)
			}
		}
		recipe.Register(recipe.NewShaped(in, out, recipe.NewShape(w, h), "crafting_table").WithoutMirror())
	}
	cobble, planks, iron := block.Cobblestone{}, oakPlanks, item.IronIngot{}
	// The hammer (and its later tiers, with another stone): two stones and
	// two sticks in a diagonal pattern.
	shaped(st(Hammer{}), 3, 3,
		nil, cobble, nil,
		nil, stick, cobble,
		stick, nil, nil)
	// The sieve: a row of planks on two stick legs. The mesh is made and
	// fitted separately (sieve.go).
	shaped(machineItem(KindSieve), 3, 2,
		planks, planks, planks,
		stick, nil, stick)
	add(machineItem(KindCrusher), n(cobble, 6), n(iron, 3))
	add(machineItem(KindAutoSieve), n(iron, 4), n(Machine{Kind: KindSieve}, 1), n(block.NewHopper(), 1))
	add(machineItem(KindMiner), n(iron, 6), n(block.NewHopper(), 1))
	add(machineItem(KindHarvester), n(iron, 5), n(block.NewHopper(), 1), n(block.NewChest(), 1))
	// The drawer: a ring of planks round a chest.
	shaped(st(Drawer{}), 3, 3,
		planks, planks, planks,
		planks, block.NewChest(), planks,
		planks, planks, planks)
	add(st(Barrel{Wood: "oak"}), n(planks, 6), n(stick, 2))
	// Hoppers: the wood hopper from planks and a chest; each hopper and its
	// reverse turn into one another when crafted alone.
	woodHopper, woodReverse := newFlowHopper("wood_hopper", cube.FaceDown), newFlowHopper("wood_hopper_reverse", cube.FaceUp)
	ironReverse := newFlowHopper("iron_hopper_reverse", cube.FaceUp)
	add(st(woodHopper), n(planks, 5), n(block.NewChest(), 1))
	add(st(woodReverse), n(woodHopper, 1))
	add(st(woodHopper), n(woodReverse, 1))
	add(st(ironReverse), n(block.NewHopper(), 1))
	add(st(block.NewHopper()), n(ironReverse, 1))
	add(st(DrawerKey{}), n(item.IronNugget{}, 4), n(stick, 1))
	add(st(DrawerUpgrade{Kind: UpgradeCapacity}), n(iron, 4), n(block.NewChest(), 1))
	add(st(DrawerUpgrade{Kind: UpgradeVoid}), n(item.Flint{}, 4), n(iron, 1))
}
