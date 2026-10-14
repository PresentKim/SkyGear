package dragonfactory

import (
	"fmt"
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

// researchTree is every research in menu order.
var researchTree = []Research{
	{ID: "stick", Name: "막대기", Cost: []Requirement{{oakPlanks, 16}}, Unlocks: []string{"stick"}},
	{ID: "wood_tools", Name: "나무 도구", Needs: []string{"stick"}, Cost: []Requirement{{oakPlanks, 32}, {stick, 16}},
		Unlocks: []string{"wooden_pickaxe", "wooden_axe", "wooden_shovel", "wooden_hoe", "wooden_sword"}},
	{ID: "chest", Name: "상자", Needs: []string{"stick"}, Cost: []Requirement{{oakPlanks, 64}}, Unlocks: []string{"chest"}},
	{ID: "furnace", Name: "화로", Needs: []string{"stick"}, Cost: []Requirement{{block.Cobblestone{}, 64}}, Unlocks: []string{"furnace"}},
	{ID: "stone_tools", Name: "돌 도구", Needs: []string{"wood_tools"}, Cost: []Requirement{{block.Cobblestone{}, 64}, {stick, 32}},
		Unlocks: []string{"stone_pickaxe", "stone_axe", "stone_shovel", "stone_hoe", "stone_sword"}},
	{ID: "torch", Name: "횃불", Needs: []string{"furnace"}, Cost: []Requirement{{item.Charcoal{}, 16}, {stick, 16}}, Unlocks: []string{"torch"}},
	{ID: "woodwork", Name: "목공", Needs: []string{"chest"}, Cost: []Requirement{{oakPlanks, 128}, {stick, 32}},
		Unlocks: []string{"oak_slab", "oak_stairs", "oak_fence", "fence_gate", "wooden_door", "trapdoor", "ladder", "bowl"}},
	{ID: "masonry", Name: "석공", Needs: []string{"furnace"}, Cost: []Requirement{{block.Stone{}, 64}, {block.Cobblestone{}, 64}},
		Unlocks: []string{"cobblestone_slab", "stone_stairs", "cobblestone_wall", "normal_stone_slab", "normal_stone_stairs", "stone_bricks"}},
	{ID: "glasswork", Name: "유리 공예", Needs: []string{"furnace"}, Cost: []Requirement{{block.Glass{}, 32}}, Unlocks: []string{"glass_pane", "glass_bottle"}},
	{ID: "iron", Name: "철 가공", Needs: []string{"furnace"}, Cost: []Requirement{{item.IronNugget{}, 36}}, Unlocks: []string{"iron_ingot", "iron_block", "iron_nugget"}},
	{ID: "hopper", Name: "호퍼", Needs: []string{"iron", "chest"}, Cost: []Requirement{{item.IronIngot{}, 10}, {block.NewChest(), 4}}, Unlocks: []string{"hopper"}},
	{ID: "iron_tools", Name: "철 도구", Needs: []string{"iron", "stone_tools"}, Cost: []Requirement{{item.IronIngot{}, 48}},
		Unlocks: []string{"iron_pickaxe", "iron_axe", "iron_shovel", "iron_hoe", "iron_sword", "bucket", "shears"}},
	{ID: "farming", Name: "농사", Needs: []string{"wood_tools"}, Cost: []Requirement{{item.Wheat{}, 32}}, Unlocks: []string{"bread", "hay_block"}},
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

// researchPaid is how many of an item of a research's cost are delivered.
func (s *session) researchPaid(r *Research, it world.Item) int {
	return s.state.ResearchPaid[r.ID][itemKey(it)]
}

// payResearch delivers what p has of the research's cost and completes it
// once everything is paid. It returns how many items were taken.
func (s *session) payResearch(p *player.Player, tx *world.Tx, r *Research) int {
	if !s.researchOpen(r) {
		return 0
	}
	taken := 0
	inv := p.Inventory()
	for _, c := range r.Cost {
		key := itemKey(c.Item)
		for slot, st := range inv.Slots() {
			need := c.Count - s.researchPaid(r, c.Item)
			if need <= 0 {
				break
			}
			if st.Empty() || !plain(st) || itemKey(st.Item()) != key {
				continue
			}
			n := min(need, st.Count())
			_ = inv.SetItem(slot, st.Grow(-n))
			if s.state.ResearchPaid == nil {
				s.state.ResearchPaid = map[string]map[string]int{}
			}
			if s.state.ResearchPaid[r.ID] == nil {
				s.state.ResearchPaid[r.ID] = map[string]int{}
			}
			s.state.ResearchPaid[r.ID][key] += n
			taken += n
		}
	}
	if taken > 0 {
		s.storeState()
		s.updateBoards(tx)
	}
	for _, c := range r.Cost {
		if s.researchPaid(r, c.Item) < c.Count {
			return taken
		}
	}
	s.completeResearch(tx, r)
	account.Grant(p, achResearch)
	return taken
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
		e.(*player.Player).Messagef("§d연구 완료: §f%s §7— 이제 %s 을(를) 만들 수 있습니다.", r.Name, s.unlockNames(r))
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

// showResearch shows the research tree.
func (s *session) showResearch(p *player.Player) {
	body, opts := s.researchMenu()
	ui.Menu(p, "연구", body, opts...)
}

func (s *session) researchMenu() (string, []ui.Option) {
	done := 0
	var opts []ui.Option
	for i := range researchTree {
		r := &researchTree[i]
		label := "§8" + r.Name
		switch {
		case s.researched(r.ID):
			done++
			label = "§a" + r.Name + " §7(완료)"
		case s.researchOpen(r):
			label = "§e" + r.Name
		}
		opts = append(opts, s.opt(label, researchIcon(r), func(p *player.Player, _ *world.Tx) { s.showResearchNode(p, r) }))
	}
	body := fmt.Sprintf("§7조합법은 연구로 하나씩 열립니다. 노란 연구를 할 수 있고, 회색은 먼저 해야 할 연구가 있습니다.\n§f연구 완료 §d%d/%d", done, len(researchTree))
	return body, append(opts, s.backOption())
}

// showResearchNode shows one research: what it opens, what it needs and
// its cost, with the delivery button.
func (s *session) showResearchNode(p *player.Player, r *Research) {
	body, opts := s.researchNode(r)
	ui.Menu(p, "연구 · "+r.Name, body, opts...)
}

func (s *session) researchNode(r *Research) (string, []ui.Option) {
	var b strings.Builder
	fmt.Fprintf(&b, "§f여는 조합법: §e%s\n", s.unlockNames(r))
	if len(r.Needs) > 0 {
		var needs []string
		for _, id := range r.Needs {
			n, _ := researchByID(id)
			mark := "§c"
			if s.researched(id) {
				mark = "§a"
			}
			needs = append(needs, mark+n.Name)
		}
		fmt.Fprintf(&b, "§f먼저 할 연구: %s\n", strings.Join(needs, "§7, "))
	}
	b.WriteString("\n§f비용(아이템 납품)\n")
	for _, c := range r.Cost {
		paid := c.Count
		if !s.researched(r.ID) {
			paid = s.researchPaid(r, c.Item)
		}
		mark := "§7"
		if paid >= c.Count {
			mark = "§a"
		}
		fmt.Fprintf(&b, "%s%s  %d/%d\n", mark, itemName(item.NewStack(c.Item, 1)), paid, c.Count)
	}
	var opts []ui.Option
	switch {
	case s.researched(r.ID):
		b.WriteString("\n§a연구 완료")
	case s.researchOpen(r):
		b.WriteString("\n§7납품하기를 누르면 인벤토리에서 필요한 만큼 냅니다(나눠 내도 됩니다).")
		opts = append(opts, s.opt("§a납품하기", iconMission, func(p *player.Player, tx *world.Tx) {
			if n := s.payResearch(p, tx, r); n > 0 {
				ui.Flash(p, "§a%d개를 납품했습니다.", n)
			} else {
				ui.Notify(p, "§7납품할 아이템이 없습니다.")
			}
			if s.researched(r.ID) {
				s.showResearch(p)
				return
			}
			s.showResearchNode(p, r)
		}))
	default:
		b.WriteString("\n§c먼저 할 연구를 마쳐야 합니다.")
	}
	research := ui.Back("연구 목록", func(p *player.Player, _ *world.Tx) { s.showResearch(p) })
	return b.String(), append(opts, s.opt(research.Label, research.Icon, research.Run))
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
}
