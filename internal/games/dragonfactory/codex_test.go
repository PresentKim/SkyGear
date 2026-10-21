package dragonfactory

import (
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/i18n"
)

func hasLine(lines []string, part string) bool {
	for _, l := range lines {
		if strings.Contains(l, part) {
			return true
		}
	}
	return false
}

// TestCodexSourcesAndUses: the codex reads where an item comes from and what it
// is for from the tables of the game.
func TestCodexSourcesAndUses(t *testing.T) {
	s := testSession(t)
	lang := i18n.Default
	cases := []struct {
		it            item.Stack
		sources, uses []string
	}{
		{item.NewStack(OreChunk{Ore: OreIron}, 1), []string{"체"}, []string{"연구", "분쇄기"}},
		{item.NewStack(Ingot{Metal: MetalSteel}, 1), []string{"합금로"}, []string{"연구", "장비"}},
		{item.NewStack(Essence{Mob: MobZombie}, 1), []string{"체", "웨이브"}, []string{"몬스터 시뮬레이터"}},
		{item.NewStack(item.Bone{}, 1), []string{"몬스터 시뮬레이터"}, []string{"장비"}},
		{item.NewStack(item.Coal{}, 1), []string{"체"}, []string{"연료"}},
		{item.NewStack(block.Cobblestone{}, 1), []string{"분쇄기"}, []string{"연구", "분쇄기", "화로"}},
		{item.NewStack(MetalDust{Metal: MetalTin}, 1), []string{"광석 세척기"}, []string{"화로"}},
	}
	for _, c := range cases {
		src, use := s.codexSources(lang, c.it), s.codexUses(lang, c.it)
		for _, want := range c.sources {
			if !hasLine(src, want) {
				t.Errorf("%s: no source with %q in %q", itemName(c.it), want, src)
			}
		}
		for _, want := range c.uses {
			if !hasLine(use, want) {
				t.Errorf("%s: no use with %q in %q", itemName(c.it), want, use)
			}
		}
	}
}

// cardsHave reports whether a slot of the cards shows the item with the count
// label (any count when count is "-").
func cardsHave(cards []codexCard, w world.Item, count string) bool {
	icon := iconOf(item.NewStack(w, 1))
	for _, c := range cards {
		for _, row := range c.rows {
			for _, cell := range row {
				if sl, ok := cell.SlotOf(); ok && sl.Icon == icon && (count == "-" || sl.Count == count) {
					return true
				}
			}
		}
	}
	return false
}

// TestCodexPagesAreWritten: the tabs have a card for every table entry.
func TestCodexPagesAreWritten(t *testing.T) {
	s := testSession(t)
	lang := i18n.Default
	for _, pr := range codexProcesses() {
		if cards := pr.cards(s, lang); len(cards) == 0 {
			t.Errorf("process tab: no recipes of %s", itemKey(pr.head.Item()))
		}
	}
	if cards := s.codexMobCards(lang); len(cards) < len(mobKinds)+len(waveLevels) {
		t.Errorf("mobs tab: %d cards", len(cards))
	}
	for _, in := range codexSieveInputs() {
		if cards := s.codexSieveCards(lang, in); len(cards) != len(meshTiers) {
			t.Errorf("%s: %d mesh cards", itemKey(in), len(cards))
		}
	}
	for _, in := range codexSieveInputs() {
		for _, tier := range meshTiers {
			if len(sieveTables[tier.tier][itemKey(in)]) == 0 {
				t.Errorf("no drops of %s in the %s table", itemKey(in), tier.tier)
			}
		}
	}
}

// TestCodexKnowsTheLab: the codex lists where an essence is made, what the
// ingredients are for and the recipes with the process tables.
func TestCodexKnowsTheLab(t *testing.T) {
	s := testSession(t)
	essence := item.NewStack(ResearchEssence{EssenceHunting}, 1)
	if src := strings.Join(s.codexSources(i18n.Default, essence), "\n"); !strings.Contains(src, "연구소") || !strings.Contains(src, "화약 ×2") {
		t.Errorf("sources %q", src)
	}
	if uses := strings.Join(s.codexUses(i18n.Default, item.NewStack(item.SpiderEye{}, 1)), "\n"); !strings.Contains(uses, "사냥의 정수") {
		t.Errorf("uses %q", uses)
	}
	if !cardsHave(s.codexSourceCards(i18n.Default, essence), item.Gunpowder{}, "§f2") {
		t.Error("the lab card of the hunting essence lacks 2 gunpowder")
	}
}

// TestCodexCraftCard: a crafting recipe is drawn as its grid: a 3×1 recipe is
// one row of three ingredients, then the arrow and the result with its count.
func TestCodexCraftCard(t *testing.T) {
	s := testSession(t)
	var c codexCard
	found := false
	for _, cr := range crafterRecipes() {
		if itemKey(cr.out.Item()) == itemKey(block.String{}) {
			if sh, ok := cr.r.(interface{ Input() []recipe.Item }); ok && len(sh.Input()) == 3 {
				c, found = s.craftCard(i18n.Default, cr), true
			}
		}
	}
	if !found {
		t.Fatal("no 3-fibre string recipe")
	}
	if len(c.rows) != 1 {
		t.Fatalf("a 3×1 recipe takes %d rows", len(c.rows))
	}
	for x := range 3 {
		if _, ok := c.rows[0][x].SlotOf(); !ok {
			t.Errorf("cell %d of the grid is not an ingredient", x)
		}
	}
	if !c.rows[0][craftArrow].IsArrow() {
		t.Error("no arrow before the result")
	}
	if sl, ok := c.rows[0][craftArrow+1].SlotOf(); !ok || sl.Count != "§f3" {
		t.Errorf("the result slot is %+v, want 3 string", sl)
	}
}

// TestCodexBoardPages: no page is longer than its budget unless one card is,
// and a page starts with the header of its machine.
func TestCodexBoardPages(t *testing.T) {
	s := testSession(t)
	cards := s.codexMobCards(i18n.Default)
	pages := s.codexBoardPages(cards, codexBoardRows)
	if len(pages) < 2 {
		t.Fatalf("%d pages for %d cards", len(pages), len(cards))
	}
	for i, pg := range pages {
		if len(pg) > codexBoardRows {
			t.Errorf("page %d has %d rows", i, len(pg))
		}
		if !pg[0][0].IsHeader() {
			t.Errorf("page %d does not start with a header", i)
		}
	}
}

// TestCodexSearchAndLinks: the index holds the items the codex can explain,
// search finds them by name (initials too), and an item links to what it is
// made from and made into.
func TestCodexSearchAndLinks(t *testing.T) {
	s := testSession(t)
	registerCreativeItems()
	steel := Ingot{Metal: MetalSteel}
	steelName := itemName(item.NewStack(steel, 1))
	found := false
	for _, e := range codexFind(steelName) {
		found = found || itemKey(e.stack.Item()) == itemKey(steel)
	}
	if !found {
		t.Fatalf("searching %q does not find the steel ingot (%d entries in the index)", steelName, len(codexIndex()))
	}
	if n := len(codexFind("")); n < 50 {
		t.Errorf("an empty search lists %d items, want the whole index", n)
	}
	if n := len(codexFind("ㅎㅎㅎㅎㅎㅎㅎㅎㅎㅎㅎ")); n != 0 {
		t.Errorf("a nonsense search found %d", n)
	}
	from := s.codexSourceCards(i18n.Default, item.NewStack(steel, 1))
	if !cardsHave(from, item.IronIngot{}, "-") || !cardsHave(from, item.Coal{}, "§f4") {
		t.Error("steel is made from iron and 4 coal")
	}
	if !cardsHave(s.codexUseCards(i18n.Default, item.NewStack(block.Cobblestone{}, 1)), block.Gravel{}, "-") {
		t.Error("cobblestone is crushed into gravel")
	}
}

// TestCodexRawAndGoals: the raw materials of an item follow the recipes down
// (counting how many crafts a recipe needs), goals are kept to goalMax and the
// quest panel shows their raw needs; the codex tabs have names.
func TestCodexRawAndGoals(t *testing.T) {
	s := testSession(t)
	registerCreativeItems()
	steel := item.NewStack(Ingot{Metal: MetalSteel}, 1)
	counts := map[string]int{}
	for _, l := range codexRaw(steel, 3) {
		counts[itemKey(l.stack.Item())] = l.count
	}
	if counts[itemKey(item.IronIngot{})] != 3 || counts[itemKey(item.Coal{})] != 12 {
		t.Errorf("3 steel need %v, want 3 iron and 12 coal", counts)
	}
	// an item no recipe makes is its own raw material
	if raw := codexRaw(item.NewStack(item.Coal{}, 1), 5); len(raw) != 1 || raw[0].count != 5 {
		t.Errorf("coal is raw: %v", raw)
	}
	s.world.Do(func(tx *world.Tx) {
		for _, w := range []world.Item{Ingot{Metal: MetalSteel}, Ingot{Metal: MetalBronze}, Ingot{Metal: MetalInvar}, Ingot{Metal: MetalElectrum}} {
			s.toggleGoal(tx, item.NewStack(w, 1))
		}
	})
	if n := len(s.state.Goals); n != goalMax {
		t.Fatalf("%d goals kept, want %d", n, goalMax)
	}
	if s.isGoal(steel) {
		t.Error("the oldest goal was not pushed out")
	}
	rows := strings.Join(s.goalRows(), "\n")
	if !strings.Contains(rows, itemName(item.NewStack(Ingot{Metal: MetalBronze}, 1))) {
		t.Errorf("the quest rows lack the goal: %q", rows)
	}
	for tab := codexTab(0); tab < codexTabCount; tab++ {
		if !i18n.Has(codexTabKeys[tab]) {
			t.Errorf("codex tab %d has no name", tab)
		}
	}
}

// TestCodexRawPicksTheUsualRecipe: a diamond pickaxe takes 3 diamonds and
// planks for its sticks, not a block of diamonds turned back into nine
// diamonds or bamboo sticks.
func TestCodexRawPicksTheUsualRecipe(t *testing.T) {
	testSession(t)
	registerCreativeItems()
	registerVanillaRecipes()
	counts := map[string]int{}
	for _, l := range codexRaw(item.NewStack(item.Pickaxe{Tier: item.ToolTierDiamond}, 1), 1) {
		counts[itemKey(l.stack.Item())] = l.count
	}
	if counts[itemKey(item.Diamond{})] != 3 {
		t.Errorf("a diamond pickaxe takes %d diamonds: %v", counts[itemKey(item.Diamond{})], counts)
	}
	for k := range counts {
		if strings.Contains(k, "bamboo") {
			t.Errorf("a diamond pickaxe takes bamboo: %v", counts)
		}
	}
}
