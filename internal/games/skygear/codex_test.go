package skygear

import (
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/internal/games/skygear/formicon"
	"github.com/presentkim/skygear/internal/games/skygear/machinerecipe"
	"github.com/presentkim/skygear/internal/games/skygear/naming"
	"github.com/presentkim/skygear/internal/games/skygear/sgitem"
	"github.com/presentkim/skygear/internal/games/skygear/sievetable"
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
		{item.NewStack(sgitem.OreChunk{Ore: sgitem.OreIron}, 1), []string{"체"}, []string{"연구", "조합"}},
		{item.NewStack(sgitem.OreBlock{Metal: sgitem.MetalTin}, 1), []string{"조합"}, []string{"분쇄기", "화로"}},
		{item.NewStack(sgitem.RawOre{Metal: sgitem.MetalTin}, 1), []string{"분쇄기"}, []string{"화로"}},
		{item.NewStack(sgitem.Ingot{Metal: sgitem.MetalSteel}, 1), []string{"합금로"}, []string{"연구", "장비"}},
		{item.NewStack(sgitem.Essence{Mob: sgitem.MobZombie}, 1), []string{"체", "웨이브"}, []string{"몬스터 시뮬레이터"}},
		{item.NewStack(item.Bone{}, 1), []string{"몬스터 시뮬레이터"}, []string{"장비"}},
		{item.NewStack(item.Coal{}, 1), []string{"체"}, []string{"연료"}},
		{item.NewStack(block.Cobblestone{}, 1), []string{"분쇄기"}, []string{"연구", "분쇄기", "화로"}},
		{item.NewStack(sgitem.MetalDust{Metal: sgitem.MetalTin}, 1), []string{"광석 세척기"}, []string{"화로"}},
	}
	for _, c := range cases {
		src, use := s.codexSources(lang, c.it), s.codexUses(lang, c.it)
		for _, want := range c.sources {
			if !hasLine(src, want) {
				t.Errorf("%s: no source with %q in %q", naming.ItemName(i18n.Default, c.it), want, src)
			}
		}
		for _, want := range c.uses {
			if !hasLine(use, want) {
				t.Errorf("%s: no use with %q in %q", naming.ItemName(i18n.Default, c.it), want, use)
			}
		}
	}
}

// cardsHave reports whether a slot of the cards shows the item with the count
// label (any count when count is "-").
func cardsHave(cards []codexCard, w world.Item, count string) bool {
	icon := formicon.Of(item.NewStack(w, 1))
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
			t.Errorf("process tab: no recipes of %s", naming.ItemKey(pr.head.Item()))
		}
	}
	if cards := s.codexMobCards(lang); len(cards) < len(sgitem.MobKinds)+len(waveLevels) {
		t.Errorf("mobs tab: %d cards", len(cards))
	}
	for _, in := range codexSieveInputs() {
		if cards := s.codexSieveCards(lang, in); len(cards) != len(sgitem.MeshTiers) {
			t.Errorf("%s: %d mesh cards", naming.ItemKey(in), len(cards))
		}
	}
	for _, in := range codexSieveInputs() {
		for _, tier := range sgitem.MeshTiers {
			if len(sievetable.Tables[tier.Tier][naming.ItemKey(in)]) == 0 {
				t.Errorf("no drops of %s in the %s table", naming.ItemKey(in), tier.Tier)
			}
		}
	}
}

// TestCodexKnowsTheLab: the codex lists where an essence is made, what the
// ingredients are for and the recipes with the process tables.
func TestCodexKnowsTheLab(t *testing.T) {
	s := testSession(t)
	essence := item.NewStack(sgitem.ResearchEssence{Kind: sgitem.EssenceHunting}, 1)
	if src := strings.Join(s.codexSources(i18n.Default, essence), "\n"); !strings.Contains(src, "정수 추출기") || !strings.Contains(src, "화약 ×2") {
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
	for _, cr := range machinerecipe.Crafters() {
		if naming.ItemKey(cr.Out.Item()) == naming.ItemKey(block.String{}) {
			if sh, ok := cr.Recipe.(interface{ Input() []recipe.Item }); ok && len(sh.Input()) == 3 {
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

// TestCodexBoard: every card is on one board that starts with the header of
// its machine.
func TestCodexBoard(t *testing.T) {
	s := testSession(t)
	cards := s.codexMobCards(i18n.Default)
	rows := s.codexBoard(i18n.Default, cards)
	cardRows := 0
	for _, b := range codexBlocks(cards) {
		cardRows += len(b.rows)
	}
	if len(rows) < cardRows {
		t.Fatalf("%d rows for %d card rows", len(rows), cardRows)
	}
	first := rows[0][0]
	for _, c := range rows[0] {
		if !c.IsBlank() {
			first = c
			break
		}
	}
	if !first.IsHeader() {
		t.Error("board does not start with a header")
	}
}

// TestCodexBlocks: narrow cards go two to a row, centred, the same columns
// for every row of a machine; a wide card is alone.
func TestCodexBlocks(t *testing.T) {
	narrow := func() codexCard {
		row := blankRow()
		row[0], row[1], row[2] = ui.Frame(), ui.Arrow(), ui.Frame()
		return codexCard{rows: [][]ui.BoardCell{row}}
	}
	blocks := codexBlocks([]codexCard{narrow(), narrow(), narrow()})
	if len(blocks) != 2 {
		t.Fatalf("%d blocks for three narrow cards, want 2", len(blocks))
	}
	// 3 + 3 gap + 3 = 9 of 11 columns: one blank column on each side
	for _, at := range []int{1, 7} {
		if !blocks[0].rows[0][at+1].IsArrow() {
			t.Errorf("no arrow at column %d of %v", at+1, blocks[0].rows[0])
		}
	}
	if !blocks[1].rows[0][2].IsArrow() || !blocks[1].rows[0][8].IsBlank() {
		t.Errorf("the last card is not in the left place alone: %v", blocks[1].rows[0])
	}
	wide := blankRow()
	wide[0], wide[7] = ui.Frame(), ui.Frame()
	if b := codexBlocks([]codexCard{{rows: [][]ui.BoardCell{wide}}}); len(b) != 1 || b[0].offset != 1 {
		t.Errorf("a card 8 wide: %+v, want one block at column 1", b)
	}
}

// TestCodexSearchAndLinks: the index holds the items the codex can explain,
// search finds them by name (initials too), and an item links to what it is
// made from and made into.
func TestCodexSearchAndLinks(t *testing.T) {
	s := testSession(t)
	registerCreativeItems()
	steel := sgitem.Ingot{Metal: sgitem.MetalSteel}
	steelName := naming.ItemName(i18n.Default, item.NewStack(steel, 1))
	found := false
	for _, e := range codexFind(i18n.Default, steelName) {
		found = found || naming.ItemKey(e.stack.Item()) == naming.ItemKey(steel)
	}
	if !found {
		t.Fatalf("searching %q does not find the steel ingot (%d entries in the index)", steelName, len(codexIndex(i18n.Default)))
	}
	if n := len(codexFind(i18n.Default, "")); n < 50 {
		t.Errorf("an empty search lists %d items, want the whole index", n)
	}
	// only what the game uses: no potions the game never asks for, but its
	// own items, what research opens and what those are made of
	in := codexRelevant()
	// nothing the island cannot get: other woods, deepslate ores
	for _, it := range []world.Item{item.Potion{}, item.LingeringPotion{}, item.SplashPotion{},
		block.Log{Wood: block.SpruceWood()}, block.Log{Wood: block.MangroveWood()}, block.Log{Wood: block.BirchWood(), Stripped: true}} {
		if in[naming.ItemKey(it)] {
			t.Errorf("%s is in the codex", naming.ItemName(i18n.Default, item.NewStack(it, 1)))
		}
	}
	sieve, _ := world.ItemByName("skygear:sieve", 0)
	for _, it := range []world.Item{steel, sieve, item.IronIngot{}, block.Furnace{}, item.Stick{}, block.Log{Wood: block.OakWood()}} {
		if !in[naming.ItemKey(it)] {
			t.Errorf("%s is not in the codex", naming.ItemName(i18n.Default, item.NewStack(it, 1)))
		}
	}
	t.Logf("%d items play a part in the game", len(in))
	if n := len(codexFind(i18n.Default, "ㅎㅎㅎㅎㅎㅎㅎㅎㅎㅎㅎ")); n != 0 {
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

// TestCodexRawAndFavourites: the raw materials of an item follow the recipes
// down (counting how many crafts a recipe needs), a player's favourites are
// kept in order and starring again takes the star off; the codex tabs have
// names.
func TestCodexRawAndFavourites(t *testing.T) {
	s := testSession(t)
	registerCreativeItems()
	steel := item.NewStack(sgitem.Ingot{Metal: sgitem.MetalSteel}, 1)
	counts := map[string]int{}
	for _, l := range codexRaw(steel, 3) {
		counts[naming.ItemKey(l.stack.Item())] = l.count
	}
	if counts[naming.ItemKey(item.IronIngot{})] != 3 || counts[naming.ItemKey(item.Coal{})] != 12 {
		t.Errorf("3 steel need %v, want 3 iron and 12 coal", counts)
	}
	// an item no recipe makes is its own raw material
	if raw := codexRaw(item.NewStack(item.Coal{}, 1), 5); len(raw) != 1 || raw[0].count != 5 {
		t.Errorf("coal is raw: %v", raw)
	}
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		bronze := item.NewStack(sgitem.Ingot{Metal: sgitem.MetalBronze}, 1)
		s.toggleFavourite(p, steel)
		s.toggleFavourite(p, bronze)
		if favs := s.favourites(p); len(favs) != 2 || naming.ItemKey(favs[0].Item()) != naming.ItemKey(steel.Item()) {
			t.Fatalf("favourites %v", favs)
		}
		s.toggleFavourite(p, steel)
		if s.isFavourite(p, steel) || !s.isFavourite(p, bronze) {
			t.Error("starring again did not take the star off")
		}
	})
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
		counts[naming.ItemKey(l.stack.Item())] = l.count
	}
	if counts[naming.ItemKey(item.Diamond{})] != 3 {
		t.Errorf("a diamond pickaxe takes %d diamonds: %v", counts[naming.ItemKey(item.Diamond{})], counts)
	}
	for k := range counts {
		if strings.Contains(k, "bamboo") {
			t.Errorf("a diamond pickaxe takes bamboo: %v", counts)
		}
	}
}

// TestCrushInputsCoverRecipes: the codex lists every block the crusher takes.
func TestCrushInputsCoverRecipes(t *testing.T) {
	listed := map[string]bool{}
	for _, in := range crushInputs() {
		listed[naming.ItemKey(in)] = true
	}
	for k := range sgitem.CrushRecipes {
		if !listed[k] {
			t.Errorf("crusher input %s missing from the codex", k)
		}
	}
}

// TestCodexHandWork: an item page lists the hand work and the barrels that
// make it, and a tool's page what it is used on.
func TestCodexHandWork(t *testing.T) {
	s := testSession(t)
	lang := i18n.Default
	src := func(w world.Item) []codexCard { return s.codexSourceCards(lang, item.NewStack(w, 1)) }
	use := func(w world.Item) []codexCard { return s.codexUseCards(lang, item.NewStack(w, 1)) }
	if !cardsHave(src(sgitem.Pebble{Kind: sgitem.PebbleStone}), block.Grass{}, "-") || !cardsHave(src(sgitem.Pebble{Kind: sgitem.PebbleStone}), block.Dirt{}, "-") {
		t.Error("a stick on grass or dirt gives pebbles")
	}
	if !cardsHave(use(item.Stick{}), block.Grass{}, "-") {
		t.Error("the stick's page does not show it used on grass")
	}
	ore := sgitem.OreBlocks()[0]
	metal, _ := sgitem.OreBlockMetal(ore)
	if !cardsHave(src(sgitem.MetalDust{Metal: metal}), ore, "-") {
		t.Error("the hammer breaks an ore block into dust")
	}
	if !cardsHave(use(sgitem.Hammer{Stone: true}), block.Stone{}, "-") {
		t.Error("the hammer's page does not show what it breaks")
	}
	if !cardsHave(src(sgitem.BarkFiber{}), block.Log{Wood: block.OakWood()}, "-") {
		t.Error("stripping a log gives bark fibers")
	}
	if !cardsHave(src(sgitem.GrassSeeds{}), block.Dirt{}, "-") {
		t.Error("tilling dirt gives grass seeds")
	}
	if !cardsHave(src(block.Dirt{}), block.Leaves{Type: block.OakLeaves()}, "-") {
		t.Error("the composter makes dirt")
	}
	if !cardsHave(src(block.Clay{}), sgitem.Dust{}, "-") || !cardsHave(src(sgitem.WitchWaterBucket{}), block.Mycelium{}, "-") {
		t.Error("the barrel's clay and witch water are not on their pages")
	}
	if cardsHave(src(sgitem.Pebble{Kind: sgitem.PebbleDeepslate}), block.Grass{}, "-") {
		t.Error("a stick gives a pebble kind it never drops")
	}
	idx := map[string]bool{}
	for _, st := range codexIndex(lang) {
		idx[naming.ItemKey(st.stack.Item())] = true
	}
	if !idx[naming.ItemKey(sgitem.GrassSeeds{})] {
		t.Error("grass seeds are not in the codex search")
	}
}
