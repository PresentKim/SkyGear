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
	ID, Name string
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

// researchTree is every research in menu order; a research's needs come
// before it.
var researchTree = []Research{
	{ID: "stick", Name: "막대기", Cost: []Requirement{{oakPlanks, 4}}, Unlocks: []string{"stick"}},
	{ID: "pebbles", Name: "돌 다지기", Needs: []string{"stick"}, Cost: []Requirement{{Pebble{Kind: PebbleStone}, 4}},
		Unlocks: []string{"cobblestone", "andesite", "diorite", "granite"}},
	{ID: "wood_tools", Name: "나무 도구", Needs: []string{"stick"}, Cost: []Requirement{{oakPlanks, 8}, {stick, 4}},
		Unlocks: []string{"wooden_pickaxe", "wooden_axe", "wooden_shovel", "wooden_hoe", "wooden_sword"}},
	{ID: "chest", Name: "상자", Needs: []string{"stick"}, Cost: []Requirement{{oakPlanks, 64}}, Unlocks: []string{"chest"}},
	{ID: "compost", Name: "퇴비통", Needs: []string{"wood_tools"}, Cost: []Requirement{{oakPlanks, 32}}, Unlocks: []string{"composter", "oak_slab"}},
	{ID: "barrel", Name: "배럴", Needs: []string{"compost"}, Cost: []Requirement{{oakPlanks, 32}, {stick, 4}}, Unlocks: []string{"dragonfactory:barrel_oak"}},
	{ID: "furnace", Name: "화로", Needs: []string{"pebbles"}, Cost: []Requirement{{block.Cobblestone{}, 32}}, Unlocks: []string{"furnace"}},
	{ID: "stone_tools", Name: "돌 도구", Needs: []string{"wood_tools", "pebbles"}, Cost: []Requirement{{block.Cobblestone{}, 32}, {stick, 16}},
		Unlocks: []string{"stone_pickaxe", "stone_axe", "stone_shovel", "stone_hoe", "stone_sword"}},
	{ID: "string", Name: "실", Needs: []string{"stone_tools"}, Cost: []Requirement{{BarkFiber{}, 8}}, Unlocks: []string{"string"}},
	{ID: "hammer", Name: "망치", Needs: []string{"pebbles"}, Cost: []Requirement{{block.Cobblestone{}, 32}, {stick, 8}}, Unlocks: []string{"dragonfactory:hammer"}},
	{ID: "sieve", Name: "체", Needs: []string{"string", "hammer"}, Cost: []Requirement{{block.String{}, 8}, {oakPlanks, 16}}, Unlocks: []string{"dragonfactory:sieve"}},
	{ID: "torch", Name: "횃불", Needs: []string{"furnace"}, Cost: []Requirement{{item.Charcoal{}, 16}, {stick, 16}}, Unlocks: []string{"torch"}},
	{ID: "woodwork", Name: "목공", Needs: []string{"chest"}, Cost: []Requirement{{oakPlanks, 128}, {stick, 32}},
		Unlocks: []string{"oak_slab", "oak_stairs", "oak_fence", "fence_gate", "wooden_door", "trapdoor", "ladder", "bowl"}},
	{ID: "masonry", Name: "석공", Needs: []string{"furnace"}, Cost: []Requirement{{block.Stone{}, 64}, {block.Cobblestone{}, 64}},
		Unlocks: []string{"cobblestone_slab", "stone_stairs", "cobblestone_wall", "normal_stone_slab", "normal_stone_stairs", "stone_bricks"}},
	{ID: "glasswork", Name: "유리 공예", Needs: []string{"furnace"}, Cost: []Requirement{{block.Glass{}, 32}}, Unlocks: []string{"glass_pane", "glass_bottle"}},
	{ID: "iron", Name: "철 가공", Needs: []string{"furnace", "sieve"}, Cost: []Requirement{{item.IronNugget{}, 36}}, Unlocks: []string{"iron_ingot", "iron_block", "iron_nugget"}},
	{ID: "hopper", Name: "호퍼", Needs: []string{"iron", "chest"}, Cost: []Requirement{{item.IronIngot{}, 10}, {block.NewChest(), 4}}, Unlocks: []string{"hopper"}},
	{ID: "crusher", Name: "분쇄기", Needs: []string{"hammer", "iron"}, Cost: []Requirement{{block.Cobblestone{}, 128}, {item.IronIngot{}, 8}}, Unlocks: []string{"dragonfactory:crusher"}},
	{ID: "auto_sieve", Name: "자동 체", Needs: []string{"sieve", "hopper"}, Cost: []Requirement{{item.IronIngot{}, 16}, {block.String{}, 16}}, Unlocks: []string{"dragonfactory:auto_sieve"}},
	{ID: "iron_tools", Name: "철 도구", Needs: []string{"iron", "stone_tools"}, Cost: []Requirement{{item.IronIngot{}, 48}},
		Unlocks: []string{"iron_pickaxe", "iron_axe", "iron_shovel", "iron_hoe", "iron_sword", "bucket", "shears"}},
	{ID: "auto_miner", Name: "자동 자원 채굴기", Needs: []string{"iron_tools", "hopper"}, Cost: []Requirement{{item.IronIngot{}, 32}, {block.NewHopper(), 2}}, Unlocks: []string{"dragonfactory:auto_miner"}},
	{ID: "drawer", Name: "서랍", Needs: []string{"chest"}, Cost: []Requirement{{oakPlanks, 64}, {block.NewChest(), 2}}, Unlocks: []string{"dragonfactory:drawer", "dragonfactory:drawer_key"}},
	{ID: "drawer_upgrades", Name: "서랍 업그레이드", Needs: []string{"drawer", "iron"}, Cost: []Requirement{{item.IronIngot{}, 16}},
		Unlocks: []string{"dragonfactory:drawer_upgrade_capacity", "dragonfactory:drawer_upgrade_void"}},
	{ID: "farming", Name: "농사", Needs: []string{"wood_tools"}, Cost: []Requirement{{item.Wheat{}, 32}}, Unlocks: []string{"bread", "hay_block"}},
	{ID: "harvester", Name: "자동 농작물 수확기", Needs: []string{"farming", "hopper"}, Cost: []Requirement{{item.IronIngot{}, 32}}, Unlocks: []string{"dragonfactory:harvester"}},
	{ID: "clay", Name: "점토", Needs: []string{"furnace"}, Cost: []Requirement{{item.ClayBall{}, 32}}, Unlocks: []string{"clay", "brick_block"}},
	{ID: "gold", Name: "금 가공", Needs: []string{"iron"}, Cost: []Requirement{{item.GoldNugget{}, 36}}, Unlocks: []string{"gold_ingot", "gold_block", "gold_nugget"}},
	{ID: "harvest", Name: "수확물", Needs: []string{"farming"}, Cost: []Requirement{{item.MelonSlice{}, 32}, {block.Pumpkin{}, 8}},
		Unlocks: []string{"melon_block", "pumpkin_pie", "melon_seeds", "pumpkin_seeds"}},
	{ID: "kelp", Name: "해초", Needs: []string{"furnace"}, Cost: []Requirement{{item.DriedKelp{}, 32}}, Unlocks: []string{"dried_kelp_block"}},
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
	after := researchText(r.ID).After
	for e := range tx.Players() {
		p := e.(*player.Player)
		p.Messagef("§d연구 완료: §f%s §7— 이제 %s 을(를) 만들 수 있습니다.", r.Name, s.unlockNames(r))
		if after != "" {
			p.Messagef("§e힌트: §f%s", after)
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
	ui.ItemGrid(p, "연구", body, slots, actions...)
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
				slots = append(slots, ui.ItemSlot{Icon: researchIcon(r), Count: "§a✔", Tip: "§7" + r.Name + "\n§a연구 완료\n§7누르면 자세히 봅니다", Run: show})
			}
		case s.researchOpen(r):
			todoN++
			if !done {
				title, line := "§e"+r.Name, "§7재료가 모자랍니다"
				if p != nil && len(s.researchMissing(p, r)) == 0 {
					title, line = "§a"+r.Name, "§a재료가 모두 있습니다: 지금 연구할 수 있습니다"
				}
				slots = append(slots, ui.ItemSlot{Icon: researchIcon(r), Tip: title + "\n" + line + "\n§7누르면 자세히 봅니다", Run: show})
			}
		}
	}
	body := "§7조합법은 연구로 하나씩 열립니다. 연구를 마치면 다음 연구가 나타납니다. 아이템에 마우스를 올리면 이름이 보입니다(노랑: 할 연구, 초록: 지금 할 수 있는 연구)."
	if len(slots) == 0 {
		if done {
			body += "\n\n§7아직 마친 연구가 없습니다."
		} else {
			body += "\n\n§a할 수 있는 연구를 모두 마쳤습니다."
		}
	}
	tabs := []ui.Option{
		ui.Tab(fmt.Sprintf("연구 필요 %d", todoN), "", !done, func(p *player.Player, tx *world.Tx) {
			if tx.World() == s.world {
				s.showResearchTab(p, false)
			}
		}),
		ui.Tab(fmt.Sprintf("연구 완료 %d", doneN), "", done, func(p *player.Player, tx *world.Tx) {
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
	ui.ItemDetail(p, "연구 · "+r.Name, d, rows, opts...)
}

// researchNode builds the screen of a research. p may be nil (the owned
// counts are then left out).
func (s *session) researchNode(p *player.Player, r *Research) (ui.Detail, []ui.SlotRow, []ui.Option) {
	txt := researchText(r.ID)
	done := s.researched(r.ID)
	var b strings.Builder
	if txt.About != "" {
		fmt.Fprintf(&b, "§f%s\n", txt.About)
	}
	d := ui.Detail{Icon: researchIcon(r), Name: "§6" + r.Name}
	var opts []ui.Option
	switch {
	case done:
		d.Status = "§a✔ 연구 완료"
		if txt.After != "" {
			fmt.Fprintf(&b, "\n§e다음 할 일\n§f%s", txt.After)
		}
	case s.researchOpen(r):
		if txt.Hint != "" {
			fmt.Fprintf(&b, "\n§e힌트\n§f%s\n", txt.Hint)
		}
		opts = append(opts, s.opt("§a연구 진행", iconMission, func(p *player.Player, tx *world.Tx) {
			missing, ok := s.doResearch(p, tx, r)
			switch {
			case ok:
				ui.Flash(p, "§a연구를 마쳤습니다.")
				s.showResearch(p)
				return
			case len(missing) > 0:
				var names []string
				for _, m := range missing {
					names = append(names, costText(m))
				}
				ui.Notify(p, "§c재료가 모자랍니다: %s 더 필요", strings.Join(names, ", "))
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
		mark, tip := "§f", fmt.Sprintf("§f%s\n§7필요 %d개", itemName(st), need)
		switch {
		case done:
			mark, tip = "§a", fmt.Sprintf("§f%s\n§a%d개 사용", itemName(st), c.Count)
			need = c.Count
		case p != nil:
			have := countPlain(p.Inventory(), c.Item)
			if have >= need {
				mark = "§a"
			}
			tip += fmt.Sprintf("\n%s보유 %d개", mark, have)
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
		opens = append(opens, ui.ItemSlot{Icon: iconOf(st), Tip: "§f" + itemName(st) + "\n§7조합법이 열립니다", Run: again})
	}
	rows := []ui.SlotRow{{Heading: "필요한 재료", Slots: cost}, {Heading: "보상 레시피", Slots: opens}}
	back := ui.Back("연구 목록", func(p *player.Player, _ *world.Tx) { s.showResearch(p) })
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
func (ResearchTable) Name() string                       { return "연구대" }
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
	cobble, planks, iron, str := block.Cobblestone{}, oakPlanks, item.IronIngot{}, block.String{}
	add(st(Hammer{}), n(cobble, 3), n(stick, 2))
	add(machineItem(KindSieve), n(planks, 3), n(stick, 2), n(str, 4))
	add(machineItem(KindCrusher), n(cobble, 6), n(iron, 3))
	add(machineItem(KindAutoSieve), n(iron, 4), n(str, 4), n(block.NewHopper(), 1))
	add(machineItem(KindMiner), n(iron, 6), n(block.NewHopper(), 1))
	add(machineItem(KindHarvester), n(iron, 5), n(block.NewHopper(), 1), n(block.NewChest(), 1))
	add(st(Drawer{}), n(planks, 8), n(block.NewChest(), 1))
	add(st(Barrel{Wood: "oak"}), n(planks, 6), n(stick, 2))
	add(st(DrawerKey{}), n(item.IronNugget{}, 4), n(stick, 1))
	add(st(DrawerUpgrade{Kind: UpgradeCapacity}), n(iron, 4), n(block.NewChest(), 1))
	add(st(DrawerUpgrade{Kind: UpgradeVoid}), n(item.Flint{}, 4), n(iron, 1))
}
