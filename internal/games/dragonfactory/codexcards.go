package dragonfactory

import (
	"fmt"
	"sort"
	"sync"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/common/ui"
)

// The codex draws its tables like a recipe viewer (docs/tech/codex-jei.md):
// each recipe is a card of item slots on a board (ui.ItemBoard), a crafting
// recipe as the crafting table's 3×3 grid with an arrow to the result, a
// machine recipe as its inputs, an arrow and its outputs, a sieve recipe as
// the block and the mesh with every drop and its chance. Cards of the same
// machine sit under one header row with the machine's slot and name. The
// details are in the tooltips of the slots.

// Columns of a card: inputs from the left (at most cardInCols to a row), the
// arrow right after them, then the outputs to the end of the row. A crafting
// card's grid is three wide: its arrow is at craftArrow.
const (
	cardInCols = 4
	craftArrow = 3
)

// codexCard is one recipe: the machine it is made in (the header) and its
// rows of cells.
type codexCard struct {
	head item.Stack
	name string
	rows [][]ui.BoardCell
}

// blankRow is a row of blank cells.
func blankRow() []ui.BoardCell { return make([]ui.BoardCell, ui.BoardColumns) }

// countLabel is the count shown on a slot (none for 1).
func countLabel(n int) string {
	if n > 1 {
		return "§f" + ui.CountText(n)
	}
	return ""
}

// rangeText writes min~max.
func rangeText(lo, hi int) string {
	if hi > lo {
		return fmt.Sprintf("%d~%d", lo, hi)
	}
	return fmt.Sprint(lo)
}

// codexCell is a slot showing st that opens its codex page. extra lines go
// under the name in the tooltip.
func (s *session) codexCell(st item.Stack, count string, extra ...string) ui.BoardCell {
	if st.Empty() {
		return ui.Blank()
	}
	st = item.NewStack(st.Item(), 1)
	tip := tipName(st)
	for _, e := range extra {
		tip += "\n§7" + e
	}
	return ui.SlotCell(ui.ItemSlot{Icon: iconOf(st), Count: count, Tip: tip, Run: func(p *player.Player, _ *world.Tx) { s.codexGo(p, func(p *player.Player) { s.showCodexItem(p, st) }) }})
}

var tagStacks sync.Map

// tagStack is an item with the tag, shown for an ingredient that takes any
// of them.
func tagStack(tag recipe.ItemTag) item.Stack {
	if v, ok := tagStacks.Load(tag.Tag()); ok {
		return v.(item.Stack)
	}
	var names []string
	byName := map[string]world.Item{}
	for _, it := range world.Items() {
		name, _ := it.EncodeItem()
		if tag.Contains(name) {
			if _, seen := byName[name]; !seen {
				names = append(names, name)
			}
			byName[name] = it
		}
	}
	st := item.Stack{}
	if len(names) > 0 {
		sort.Strings(names)
		st = item.NewStack(byName[names[0]], 1)
	}
	tagStacks.Store(tag.Tag(), st)
	return st
}

// ingredientCell is the slot of a recipe ingredient.
func (s *session) ingredientCell(lang string, in recipe.Item) ui.BoardCell {
	switch in := in.(type) {
	case item.Stack:
		return s.codexCell(in, countLabel(in.Count()))
	case recipe.ItemTag:
		return s.codexCell(tagStack(in), countLabel(in.Count()), reqName(lang, crafterReq{need: in}))
	}
	return ui.Blank()
}

// craftCard is a crafting table recipe: its grid, the arrow and the result.
func (s *session) craftCard(lang string, cr *crafterRecipe) codexCard {
	ins := cr.r.Input()
	w := 3
	if sh, ok := cr.r.(recipe.Shaped); ok {
		w = max(1, sh.Shape().Width())
	} else {
		var packed []recipe.Item
		for _, in := range ins {
			if in != nil && !in.Empty() {
				packed = append(packed, in)
			}
		}
		ins = packed
	}
	h := max(1, (len(ins)+w-1)/w)
	rows := make([][]ui.BoardCell, h)
	for y := range rows {
		rows[y] = blankRow()
		for x := range 3 {
			rows[y][x] = ui.Frame()
		}
	}
	for i, in := range ins {
		if in == nil || in.Empty() || i%w >= 3 || i/w >= h {
			continue
		}
		rows[i/w][i%w] = s.ingredientCell(lang, in)
	}
	mid := (h - 1) / 2
	rows[mid][craftArrow] = ui.Arrow()
	rows[mid][craftArrow+1] = s.codexCell(cr.out, countLabel(cr.out.Count()))
	table := item.NewStack(block.CraftingTable{}, 1)
	return codexCard{head: table, name: itemName(table), rows: rows}
}

// machineCard is a machine recipe: inputs (four to a row), the arrow and the
// outputs (six to a row).
func machineCard(head item.Stack, ins, outs []ui.BoardCell) codexCard {
	inCols := max(1, min(len(ins), cardInCols))
	arrow, outCol := inCols, inCols+1
	outCols := ui.BoardColumns - outCol
	h := max(1, (len(ins)+inCols-1)/inCols, (len(outs)+outCols-1)/outCols)
	rows := make([][]ui.BoardCell, h)
	for y := range rows {
		rows[y] = blankRow()
	}
	for i, c := range ins {
		rows[i/inCols][i%inCols] = c
	}
	rows[0][arrow] = ui.Arrow()
	for i, c := range outs {
		rows[i/outCols][outCol+i%outCols] = c
	}
	return codexCard{head: head, name: itemName(head), rows: rows}
}

// noteCard is a line of text under a machine's header (what has no recipe
// to draw: a fuel, a research cost).
func noteCard(head item.Stack, text string) codexCard {
	row := blankRow()
	row[0] = ui.TextCell(text)
	return codexCard{head: head, name: itemName(head), rows: [][]ui.BoardCell{row}}
}

// codexHeadRow is the header of a group of cards: the machine and its name.
func (s *session) codexHeadRow(c codexCard) []ui.BoardCell {
	row := blankRow()
	if !c.head.Empty() {
		// the machine is a small icon without a slot frame, so it does not read
		// as a slot of the recipes under it
		sl, _ := s.codexCell(c.head, "").SlotOf()
		row[0] = ui.MiniCell(sl)
		row[1] = ui.TextCell(c.name)
	} else {
		row[0] = ui.TextCell(c.name)
	}
	return row
}

// codexBoardPages lays the cards out in pages of at most budget rows. Cards
// of the same machine in a row share one header; a page that goes on with
// the machine of the page before repeats its header. A card is never split.
func (s *session) codexBoardPages(cards []codexCard, budget int) [][][]ui.BoardCell {
	var pages [][][]ui.BoardCell
	var cur [][]ui.BoardCell
	last := ""
	for _, c := range cards {
		need := len(c.rows)
		if c.name != last {
			need++
		}
		if len(cur) > 0 && len(cur)+need > budget {
			pages = append(pages, cur)
			cur, last = nil, ""
		}
		if c.name != last {
			cur = append(cur, s.codexHeadRow(c))
			last = c.name
		}
		cur = append(cur, c.rows...)
	}
	if len(cur) > 0 || len(pages) == 0 {
		pages = append(pages, cur)
	}
	return pages
}

var (
	smeltOnce sync.Once
	smeltFrom map[string][]item.Stack
)

// codexSmeltInputs are the items that smelt into it, from every registered
// item (built once: the registry is complete when a form is shown).
func codexSmeltInputs(it item.Stack) []item.Stack {
	smeltOnce.Do(func() {
		smeltFrom = map[string][]item.Stack{}
		for _, w := range world.Items() {
			st := item.NewStack(w, 1)
			if got, ok := smeltOf(st); ok {
				k := itemKey(got.Item())
				smeltFrom[k] = append(smeltFrom[k], st)
			}
		}
		for _, list := range smeltFrom {
			sort.Slice(list, func(i, j int) bool { return itemKey(list[i].Item()) < itemKey(list[j].Item()) })
		}
	})
	return smeltFrom[itemKey(it.Item())]
}

// codexSourceCards are the recipes that make it, as cards.
func (s *session) codexSourceCards(lang string, it item.Stack) []codexCard {
	k := itemKey(it.Item())
	var cards []codexCard
	for _, cr := range crafterRecipes() {
		if itemKey(cr.out.Item()) == k {
			cards = append(cards, s.craftCard(lang, cr))
		}
	}
	furnace := item.NewStack(block.Furnace{}, 1)
	for _, in := range codexSmeltInputs(it) {
		cards = append(cards, machineCard(furnace, []ui.BoardCell{s.codexCell(in, "")}, []ui.BoardCell{s.codexCell(it, "")}))
	}
	for _, r := range alloyRecipes {
		if itemKey(r.Out) == k {
			cards = append(cards, s.alloyCard(r))
		}
	}
	for i := range labRecipes {
		if itemKey(labRecipes[i].Out) == k {
			cards = append(cards, s.labCard(&labRecipes[i]))
		}
	}
	for _, in := range crushInputs() {
		if got, ok := crusherOutput(item.NewStack(in, 1)); ok && itemKey(got.Item()) == k {
			cards = append(cards, s.crushCard(in, got))
		}
	}
	if _, ok := it.Item().(MetalDust); ok {
		cards = append(cards, noteCard(machineItem(KindWasher), i18n.T(lang, "codex.src.washer")))
	}
	sieve := machineItem(KindSieve)
	for _, in := range codexSieveInputs() {
		for _, t := range meshTiers {
			if d, ok := sieveDropOf(t.tier, in, k); ok {
				cards = append(cards, machineCard(sieve,
					[]ui.BoardCell{s.codexCell(item.NewStack(in, 1), ""), s.codexCell(item.NewStack(Mesh{Tier: t.tier}, 1), "")},
					[]ui.BoardCell{s.chanceCell(lang, d.Item, d.Permille)}))
				break
			}
		}
	}
	for _, mob := range mobKinds {
		for _, d := range simDrops[mob] {
			if itemKey(d.Item) == k {
				cards = append(cards, s.simCard(lang, mob, []SimDrop{d}))
			}
		}
	}
	for _, mob := range mobKinds {
		for _, in := range codexSieveInputs() {
			for _, d := range magicThemes[mob].Adds[itemKey(in)] {
				if itemKey(d.Item.Item()) == k {
					cards = append(cards, machineCard(machineItem(KindMagicSieve),
						[]ui.BoardCell{s.codexCell(item.NewStack(Essence{Mob: mob}, 1), ""), s.codexCell(item.NewStack(in, 1), "")},
						[]ui.BoardCell{s.chanceCell(lang, d.Item, d.Permille)}))
				}
			}
		}
	}
	if _, ok := it.Item().(Essence); ok {
		cards = append(cards, noteCard(machineItem(KindGate), i18n.T(lang, "codex.src.waveDrop")))
	}
	for i := range waveLevels {
		for _, r := range waveLevels[i].Reward {
			if itemKey(r.Item()) == k {
				cards = append(cards, s.waveCard(lang, i)...)
				break
			}
		}
	}
	for _, pk := range plantKinds {
		for _, y := range pk.yield {
			if itemKey(y.item) == k {
				cards = append(cards, s.plantCard(pk))
			}
		}
	}
	return cards
}

// codexUseCards are the recipes it goes into, as cards.
func (s *session) codexUseCards(lang string, it item.Stack) []codexCard {
	k := itemKey(it.Item())
	var cards []codexCard
	for _, cr := range crafterRecipes() {
		if cr.wants(it) {
			cards = append(cards, s.craftCard(lang, cr))
		}
	}
	if sm, ok := smeltOf(it); ok {
		cards = append(cards, machineCard(item.NewStack(block.Furnace{}, 1), []ui.BoardCell{s.codexCell(it, "")}, []ui.BoardCell{s.codexCell(sm, countLabel(sm.Count()))}))
	}
	for _, r := range alloyRecipes {
		if itemKey(r.A) == k || itemKey(r.B) == k {
			cards = append(cards, s.alloyCard(r))
		}
	}
	for i := range labRecipes {
		for _, in := range labRecipes[i].In {
			if itemKey(in.Item) == k {
				cards = append(cards, s.labCard(&labRecipes[i]))
				break
			}
		}
	}
	if got, ok := crusherOutput(it); ok {
		cards = append(cards, s.crushCard(it.Item(), got))
	}
	if e, ok := it.Item().(Essence); ok {
		cards = append(cards, s.simCard(lang, e.Mob, simDrops[e.Mob]))
		cards = append(cards, s.themeCards(lang, e.Mob)...)
	}
	research := item.NewStack(ResearchTable{}, 1)
	for i := range researchTree {
		r := &researchTree[i]
		for _, c := range r.Cost {
			if itemKey(c.Item) == k {
				cards = append(cards, noteCard(research, fmt.Sprintf(i18n.T(lang, "codex.use.research"), r.Title(lang), c.Count)))
			}
		}
	}
	if fuelDuration(it) > 0 {
		cards = append(cards, noteCard(machineItem(KindGenerator), i18n.T(lang, "codex.use.fuel")))
	}
	if plantMatter(it) {
		cards = append(cards, noteCard(item.NewStack(block.Composter{}, 1), i18n.T(lang, "codex.use.compost")))
	}
	tinker := machineItem(KindTinker)
	for _, h := range headMaterials {
		if itemKey(h.item) == k {
			cards = append(cards, noteCard(tinker, fmt.Sprintf(i18n.T(lang, "codex.use.tool"), i18n.T(lang, "tmat."+h.id), i18n.T(lang, "codex.use.head"))))
		}
	}
	for _, h := range handleMaterials {
		if itemKey(h.item) == k {
			cards = append(cards, noteCard(tinker, fmt.Sprintf(i18n.T(lang, "codex.use.tool"), i18n.T(lang, "thandle."+h.id), i18n.T(lang, "codex.use.handle"))))
		}
	}
	for _, m := range toolModifiers {
		if itemKey(m.item) == k {
			cards = append(cards, noteCard(tinker, fmt.Sprintf(i18n.T(lang, "codex.use.tool"), i18n.T(lang, "tmod."+m.id), fmt.Sprintf("×%d", m.cost))))
		}
	}
	return cards
}

// crushInputs are the blocks and chunks the crusher takes.
func crushInputs() []world.Item {
	return append([]world.Item{block.Stone{}, block.Cobblestone{}, block.Gravel{}, block.Sand{}}, chunkItems()...)
}

// sieveDropOf is the drop with the key in a mesh's table for a block.
func sieveDropOf(tier string, in world.Item, key string) (SieveDrop, bool) {
	for _, d := range sieveTables[tier][itemKey(in)] {
		if itemKey(d.Item.Item()) == key {
			return d, true
		}
	}
	return SieveDrop{}, false
}

// chanceCell is a drop with its chance in per mille on the slot.
func (s *session) chanceCell(lang string, st item.Stack, permille int) ui.BoardCell {
	return s.codexCell(st, "§f"+formatPermille(permille)+"%", fmt.Sprintf(i18n.T(lang, "codex.sieve.chance"), formatPermille(permille)))
}

func (s *session) alloyCard(r AlloyRecipe) codexCard {
	return machineCard(machineItem(KindAlloy),
		[]ui.BoardCell{s.codexCell(item.NewStack(r.A, 1), countLabel(r.CountA)), s.codexCell(item.NewStack(r.B, 1), countLabel(r.CountB))},
		[]ui.BoardCell{s.codexCell(item.NewStack(r.Out, 1), countLabel(r.OutCount))})
}

func (s *session) labCard(r *LabRecipe) codexCard {
	var ins []ui.BoardCell
	for _, in := range r.In {
		ins = append(ins, s.codexCell(item.NewStack(in.Item, 1), countLabel(in.Count)))
	}
	return machineCard(machineItem(KindLab), ins, []ui.BoardCell{s.codexCell(item.NewStack(r.Out, 1), countLabel(r.OutCount))})
}

func (s *session) crushCard(in world.Item, out item.Stack) codexCard {
	return machineCard(machineItem(KindCrusher), []ui.BoardCell{s.codexCell(item.NewStack(in, 1), "")}, []ui.BoardCell{s.codexCell(out, countLabel(out.Count()))})
}

// simCard is the drops of a mob's essence in the simulator.
func (s *session) simCard(lang string, mob string, drops []SimDrop) codexCard {
	var outs []ui.BoardCell
	for _, d := range drops {
		outs = append(outs, s.codexCell(item.NewStack(d.Item, 1), fmt.Sprintf("§f%d%%", d.Chance),
			fmt.Sprintf(i18n.T(lang, "codex.card.simDrop"), rangeText(d.Min, d.Max), d.Chance)))
	}
	return machineCard(machineItem(KindSimulator), []ui.BoardCell{s.codexCell(item.NewStack(Essence{Mob: mob}, 1), "")}, outs)
}

// themeCards are what a mob's essence adds to each block in the magic sieve.
func (s *session) themeCards(lang string, mob string) []codexCard {
	th := magicThemes[mob]
	var cards []codexCard
	for _, in := range codexSieveInputs() {
		drops := th.Adds[itemKey(in)]
		if len(drops) == 0 {
			continue
		}
		var outs []ui.BoardCell
		for _, d := range drops {
			extra := []string{fmt.Sprintf(i18n.T(lang, "codex.sieve.chance"), formatPermille(d.Permille))}
			if d.Rare {
				extra = append(extra, i18n.T(lang, "codex.card.rare"))
			}
			outs = append(outs, s.codexCell(d.Item, "§f"+formatPermille(d.Permille)+"%", extra...))
		}
		block := s.codexCell(item.NewStack(in, 1), "")
		if th.Replace[itemKey(in)] {
			block = s.codexCell(item.NewStack(in, 1), "", i18n.T(lang, "codex.card.replace"))
		}
		cards = append(cards, machineCard(machineItem(KindMagicSieve), []ui.BoardCell{s.codexCell(item.NewStack(Essence{Mob: mob}, 1), ""), block}, outs))
	}
	return cards
}

// waveCard is a gate level: its number and time, the arrow and the first
// clear rewards, and under them the mobs that come.
func (s *session) waveCard(lang string, i int) []codexCard {
	w := waveLevels[i]
	var outs []ui.BoardCell
	for _, r := range w.Reward {
		outs = append(outs, s.codexCell(r, countLabel(r.Count())))
	}
	c := machineCard(machineItem(KindGate), []ui.BoardCell{ui.TextCell(fmt.Sprintf(i18n.T(lang, "codex.card.wave"), i+1, w.Seconds))}, outs)
	var groups string
	for j, g := range w.Groups {
		if j > 0 {
			groups += ", "
		}
		groups += fmt.Sprintf("%s ×%d", mobName(g.Mob), g.N)
	}
	row := blankRow()
	row[0] = ui.TextCell("§8" + groups)
	c.rows = append(c.rows, row)
	return []codexCard{c}
}

// plantCard is a magic plant: its seed, the arrow and what it yields.
func (s *session) plantCard(pk plantKind) codexCard {
	seed := item.NewStack(PlantSeed{Seed: pk.seed}, 1)
	var outs []ui.BoardCell
	for _, y := range pk.yield {
		outs = append(outs, s.codexCell(item.NewStack(y.item, 1), "§f"+rangeText(y.min, y.max)))
	}
	head := seed
	c := machineCard(head, []ui.BoardCell{s.codexCell(seed, "")}, outs)
	c.head, c.name = item.Stack{}, i18n.T(i18n.Default, "codex.plants")
	return c
}

// codexProcessCards are the process tab: the crusher, the alloys, the lab
// and the tinker table's materials.
func (s *session) codexProcessCards(lang string) []codexCard {
	var cards []codexCard
	for _, in := range crushInputs() {
		if got, ok := crusherOutput(item.NewStack(in, 1)); ok {
			cards = append(cards, s.crushCard(in, got))
		}
	}
	cards = append(cards, noteCard(machineItem(KindWasher), i18n.T(lang, "codex.src.washer")))
	for _, r := range alloyRecipes {
		cards = append(cards, s.alloyCard(r))
	}
	for i := range labRecipes {
		cards = append(cards, s.labCard(&labRecipes[i]))
	}
	return append(cards, s.codexGearCards(lang)...)
}

// codexGearCards are the tinker table's head and handle materials and
// modifiers: the material's slot with what it gives.
func (s *session) codexGearCards(lang string) []codexCard {
	tinker := machineItem(KindTinker)
	var cards []codexCard
	gear := func(it world.Item, text string) {
		row := blankRow()
		row[0] = s.codexCell(item.NewStack(it, 1), "", text)
		row[1] = ui.TextCell(text)
		cards = append(cards, codexCard{head: tinker, name: itemName(tinker), rows: [][]ui.BoardCell{row}})
	}
	for _, h := range headMaterials {
		gear(h.item, fmt.Sprintf(i18n.T(lang, "codex.card.head"), i18n.T(lang, "tmat."+h.id), h.dur, h.eff, h.attack))
	}
	for _, h := range handleMaterials {
		gear(h.item, fmt.Sprintf(i18n.T(lang, "codex.card.handle"), i18n.T(lang, "thandle."+h.id)))
	}
	for _, m := range toolModifiers {
		gear(m.item, fmt.Sprintf(i18n.T(lang, "codex.card.mod"), i18n.T(lang, "tmod."+m.id), m.cost))
	}
	return cards
}

// codexMobCards are the mobs tab: the simulator, the magic sieve's themes
// and the gate's levels.
func (s *session) codexMobCards(lang string) []codexCard {
	var cards []codexCard
	for _, mob := range mobKinds {
		cards = append(cards, s.simCard(lang, mob, simDrops[mob]))
	}
	for _, mob := range mobKinds {
		cards = append(cards, s.themeCards(lang, mob)...)
	}
	for i := range waveLevels {
		cards = append(cards, s.waveCard(lang, i)...)
	}
	return cards
}

// codexSieveCards are the drops of every mesh for a block: the mesh, the
// arrow and the drops (the block is the one chosen above them).
func (s *session) codexSieveCards(lang string, in world.Item) []codexCard {
	var cards []codexCard
	for _, t := range meshTiers {
		drops := append([]SieveDrop(nil), sieveTables[t.tier][itemKey(in)]...)
		sort.SliceStable(drops, func(i, j int) bool { return drops[i].Permille > drops[j].Permille })
		var outs []ui.BoardCell
		for _, d := range drops {
			outs = append(outs, s.chanceCell(lang, d.Item, d.Permille))
		}
		cards = append(cards, machineCard(machineItem(KindSieve),
			[]ui.BoardCell{s.codexCell(item.NewStack(Mesh{Tier: t.tier}, 1), "")}, outs))
	}
	return cards
}
