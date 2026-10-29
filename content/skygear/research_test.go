package skygear

import (
	"github.com/presentkim/skygear/common/i18n"

	"slices"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/naming"
)

// TestResearchTreeIsValid: IDs are unique, needs come earlier in the tree,
// every result is a known item and every name is Korean.
func TestResearchTreeIsValid(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range researchTree {
		if seen[r.ID] || len(r.Cost) == 0 || (len(r.Unlocks) == 0 && !strings.HasPrefix(r.ID, "project_")) {
			t.Errorf("research %q is invalid", r.ID)
		}
		for _, n := range r.Needs {
			if !seen[n] {
				t.Errorf("research %q needs %q, which is not before it", r.ID, n)
			}
		}
		seen[r.ID] = true
		for _, n := range r.Unlocks {
			it, ok := world.ItemByName(naming.UnlockName(n), 0)
			if !ok {
				t.Errorf("research %q opens unknown item %q", r.ID, n)
				continue
			}
			if got := recipeKey(it); got != naming.UnlockName(n) {
				t.Errorf("research %q: item %q encodes as %q", r.ID, n, got)
			}
			if name := naming.ItemName(i18n.Default, item.NewStack(it, 1)); name == "" || name[0] < 0x80 {
				t.Errorf("research %q: no Korean name for %q (%q)", r.ID, n, name)
			}
		}
		for _, c := range r.Cost {
			if name := naming.ItemName(i18n.Default, item.NewStack(c.Item, 1)); name == "" || name[0] < 0x80 {
				t.Errorf("research %q: no Korean name for cost %v (%q)", r.ID, c.Item, name)
			}
		}
	}
}

// recipeFor returns a crafting recipe whose result is it. (Vanilla
// recipes are registered with the server config, not in tests.)
func recipeFor(it world.Item) recipe.Recipe {
	return recipe.NewShapeless([]recipe.Item{item.NewStack(block.Planks{Wood: block.OakWood()}, 1)}, item.NewStack(it, 1), "crafting_table")
}

// TestResearchOpensRecipes: only the base recipes are open at first; a
// research paid in parts opens its results when the last part is paid,
// and needs the researches before it.
func TestResearchOpensRecipes(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		open := s.recipeFilter()
		for _, it := range []world.Item{oakPlanks, block.CraftingTable{}, ResearchTable{}} {
			if !open(recipeFor(it)) {
				t.Errorf("base recipe of %v is locked", it)
			}
		}
		if open(recipeFor(stick)) || open(recipeFor(block.NewChest())) {
			t.Fatal("a recipe is open before research")
		}
		chest, _ := researchByID("chest")
		if _, done := s.doResearch(p, tx, chest); done {
			t.Fatal("did a research whose needs are not done")
		}
		sticks, _ := researchByID("stick")
		_ = p.Inventory().SetItem(0, item.NewStack(oakPlanks, 3))
		if missing, done := s.doResearch(p, tx, sticks); done || len(missing) != 1 || missing[0].Count != 1 {
			t.Fatalf("with 3 of 4 planks: done %v, missing %v", done, missing)
		}
		if st, _ := p.Inventory().Item(0); st.Count() != 3 {
			t.Fatalf("a research that was not done took planks: %d left", st.Count())
		}
		_ = p.Inventory().SetItem(0, item.NewStack(oakPlanks, 6))
		if _, done := s.doResearch(p, tx, sticks); !done || !s.researched("stick") {
			t.Fatal("stick research not done with enough planks")
		}
		if st, _ := p.Inventory().Item(0); st.Count() != 2 {
			t.Fatalf("%d planks left, want 2", st.Count())
		}
		// The chest research costs more and is also paid whole, at once.
		_ = p.Inventory().SetItem(0, item.NewStack(oakPlanks, 31))
		if _, done := s.doResearch(p, tx, chest); done {
			t.Fatal("chest research done with 31 of 32 planks")
		}
		_ = p.Inventory().SetItem(0, item.NewStack(oakPlanks, 32))
		if missing, done := s.doResearch(p, tx, chest); !done || len(missing) != 0 || !s.researched("chest") {
			t.Fatalf("chest research: done %v missing %v", done, missing)
		}
		if p.Inventory().ContainsItem(item.NewStack(oakPlanks, 1)) {
			t.Fatal("planks left, want none")
		}
		if !p.Inventory().ContainsItem(item.NewStack(block.NewChest(), 2)) {
			t.Fatal("the chest research gave no chests")
		}
		if !s.recipeFilter()(recipeFor(stick)) || !s.recipeFilter()(recipeFor(block.NewChest())) {
			t.Fatal("researches did not open sticks and the chest")
		}
	})
}

// TestResearchTableRecipe: the research table is crafted from two crafting
// tables.
func TestResearchTableRecipe(t *testing.T) {
	for _, r := range recipe.Recipes() {
		for _, out := range r.Output() {
			if _, ok := out.Item().(ResearchTable); ok && len(r.Input()) == 2 {
				return
			}
		}
	}
	t.Fatal("no research table recipe")
}

// TestResearchRewards: every reward is something its research opens, and
// researching gives it to the researcher.
func TestResearchRewards(t *testing.T) {
	for id, counts := range researchRewards {
		r, ok := researchByID(id)
		if !ok {
			t.Errorf("reward for unknown research %s", id)
			continue
		}
		for n, c := range counts {
			if !slices.Contains(r.Unlocks, n) || c <= 0 {
				t.Errorf("%s: reward %s ×%d is not one of its unlocks", id, n, c)
			}
		}
		if len(researchReward(r)) != len(counts) {
			t.Errorf("%s: reward items %v, want %d kinds", id, researchReward(r), len(counts))
		}
	}
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		r, _ := researchByID("generator")
		s.giveResearchReward(p, tx, r)
		if !p.Inventory().ContainsItem(machineItem(KindGenerator)) {
			t.Fatal("no generator given")
		}
	})
}

func TestResearchRewardDropsWhenInventoryFull(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		for slot := range p.Inventory().Size() {
			_ = p.Inventory().SetItem(slot, item.NewStack(block.Dirt{}, 64))
		}
		r, _ := researchByID("generator")
		for _, n := range r.Needs {
			need, _ := researchByID(n)
			s.completeResearch(tx, need)
		}
		for i, c := range r.Cost {
			_ = p.Inventory().SetItem(i, item.NewStack(c.Item, c.Count+1))
		}
		if missing, ok := s.doResearch(p, tx, r); !ok {
			t.Fatalf("research not done, missing %v", missing)
		}
		if p.Inventory().ContainsItem(machineItem(KindGenerator)) {
			t.Fatal("went into the inventory")
		}
		if droppedItems(tx)[naming.ItemKey(machineItem(KindGenerator).Item())] != 1 {
			t.Fatalf("dropped %v, want the generator", droppedItems(tx))
		}
	})
}
