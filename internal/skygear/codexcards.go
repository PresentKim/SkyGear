package skygear

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/internal/skygear/formicon"
	"github.com/presentkim/skygear/internal/skygear/machinerecipe"
	"github.com/presentkim/skygear/internal/skygear/naming"
	"github.com/presentkim/skygear/internal/skygear/sgitem"
	"github.com/presentkim/skygear/internal/skygear/sievetable"
	"github.com/presentkim/skygear/internal/skygear/textfmt"
	"github.com/presentkim/skygear/internal/skygear/tinker"
)

// The codex draws each recipe as a card of item slots on a board
// (ui.ItemBoard), JEI style: inputs, an arrow and outputs, grouped under one
// header row per machine. Details are in the slot tooltips.

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
func (s *session) codexCell(lang string, st item.Stack, count string, extra ...string) ui.BoardCell {
	if st.Empty() {
		return ui.Blank()
	}
	st = item.NewStack(st.Item(), 1)
	tip := tipName(lang, st)
	for _, e := range extra {
		tip += "\n§7" + e
	}
	return ui.SlotCell(ui.ItemSlot{Icon: formicon.Of(st), Count: count, Tip: tip, Run: func(p *player.Player, _ *world.Tx) { s.codexGo(p, func(p *player.Player) { s.showCodexItem(p, st) }) }})
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
		// oak first, the most common of a tag's kinds (planks, logs)
		sort.Slice(names, func(i, j int) bool {
			oi, oj := strings.HasPrefix(names[i], "minecraft:oak"), strings.HasPrefix(names[j], "minecraft:oak")
			if oi != oj {
				return oi
			}
			return names[i] < names[j]
		})
		st = item.NewStack(byName[names[0]], 1)
	}
	tagStacks.Store(tag.Tag(), st)
	return st
}

// ingredientCell is the slot of a recipe ingredient.
func (s *session) ingredientCell(lang string, in recipe.Item) ui.BoardCell {
	switch in := in.(type) {
	case item.Stack:
		return s.codexCell(lang, in, countLabel(in.Count()))
	case recipe.ItemTag:
		return s.codexCell(lang, tagStack(in), countLabel(in.Count()), machinerecipe.ReqName(lang, machinerecipe.CrafterReq{Need: in}))
	}
	return ui.Blank()
}

// craftCard is a crafting table recipe: its grid, the arrow and the result.
func (s *session) craftCard(lang string, cr *machinerecipe.Crafter) codexCard {
	ins := cr.Recipe.Input()
	w := 3
	if sh, ok := cr.Recipe.(recipe.Shaped); ok {
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
	rows[mid][craftArrow+1] = s.codexCell(lang, cr.Out, countLabel(cr.Out.Count()))
	table := item.NewStack(block.CraftingTable{}, 1)
	return codexCard{head: table, name: naming.UIItemName(lang, table), rows: rows}
}

// machineCard is a machine recipe: inputs (four to a row), the arrow and the
// outputs (six to a row).
func machineCard(lang string, head item.Stack, ins, outs []ui.BoardCell) codexCard {
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
	return codexCard{head: head, name: naming.UIItemName(lang, head), rows: rows}
}

// noteCard is a line of text under a machine's header (what has no recipe
// to draw: a fuel, a research cost).
func noteCard(lang string, head item.Stack, text string) codexCard {
	row := blankRow()
	row[0] = ui.TextCell(text)
	return codexCard{head: head, name: naming.UIItemName(lang, head), rows: [][]ui.BoardCell{row}}
}

// codexHeadRow is the header of a group of cards: the machine and its name.
func (s *session) codexHeadRow(lang string, c codexCard) []ui.BoardCell {
	row := blankRow()
	if !c.head.Empty() {
		// the machine is a small icon without a slot frame, so it does not read
		// as a slot of the recipes under it
		sl, _ := s.codexCell(lang, c.head, "").SlotOf()
		row[0] = ui.MiniCell(sl)
		row[1] = ui.TextCell("§e" + c.name)
	} else {
		row[0] = ui.TextCell("§e" + c.name)
	}
	return row
}

// codexBoard lays the cards out the way JEI shows recipes: each machine's
// cards centred in the window, two side by side when two fit, a blank row
// between them. Cards of the same machine share one header, at their left
// edge; a new machine's header comes after a blank row too, so where one
// source ends and the next begins shows.
func (s *session) codexBoard(lang string, cards []codexCard) [][]ui.BoardCell {
	var rows [][]ui.BoardCell
	last := ""
	for _, b := range codexBlocks(cards) {
		switch {
		case (len(rows) == 0 || b.name != last) && b.name != "":
			if len(rows) > 0 {
				rows = append(rows, blankRow())
			}
			rows = append(rows, shiftRow(s.codexHeadRow(lang, codexCard{head: b.head, name: b.name}), b.offset))
		case len(rows) > 0:
			rows = append(rows, blankRow())
		}
		rows = append(rows, b.rows...)
		last = b.name
	}
	return rows
}

// codexBlock is a row of one or two cards of the same machine, placed in the
// window's columns.
type codexBlock struct {
	head   item.Stack
	name   string
	offset int
	rows   [][]ui.BoardCell
}

// codexBlocks places the cards: the cards of a machine share a width (the
// widest one's), so their arrows line up, and are centred; two go side by
// side, with a gap of up to three cells, when both fit a row.
func codexBlocks(cards []codexCard) []codexBlock {
	var out []codexBlock
	for i := 0; i < len(cards); {
		j := i
		w := 0
		for j < len(cards) && cards[j].name == cards[i].name {
			w = max(w, cardWidth(cards[j]))
			j++
		}
		perRow, gap := 1, 0
		if 2*w+1 <= ui.BoardColumns {
			perRow, gap = 2, min(3, ui.BoardColumns-2*w)
		}
		offset := (ui.BoardColumns - (perRow*w + (perRow-1)*gap)) / 2
		for k := i; k < j; k += perRow {
			b := codexBlock{head: cards[k].head, name: cards[k].name, offset: offset}
			for n, c := range cards[k:min(k+perRow, j)] {
				for r, row := range c.rows {
					if r == len(b.rows) {
						b.rows = append(b.rows, blankRow())
					}
					at := offset + n*(w+gap)
					copy(b.rows[r][at:], row[:min(len(row), w)])
				}
			}
			out = append(out, b)
		}
		i = j
	}
	return out
}

// cardWidth is how many columns a card's cells take; a line of text takes
// the whole row, as it runs over the blank cells after it.
func cardWidth(c codexCard) int {
	w := 1
	for _, row := range c.rows {
		for i, cell := range row {
			if cell.IsText() {
				return ui.BoardColumns
			}
			if !cell.IsBlank() {
				w = max(w, i+1)
			}
		}
	}
	return min(w, ui.BoardColumns)
}

// shiftRow moves a row's cells right by n columns, dropping what falls off.
func shiftRow(row []ui.BoardCell, n int) []ui.BoardCell {
	out := blankRow()
	for i, c := range row {
		if i+n < len(out) {
			out[i+n] = c
		}
	}
	return out
}

// centredRow is a row of the cells centred in the window.
func centredRow(cells []ui.BoardCell) []ui.BoardCell {
	return shiftRow(cells, (ui.BoardColumns-len(cells))/2)
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
				k := naming.ItemKey(got.Item())
				smeltFrom[k] = append(smeltFrom[k], st)
			}
		}
		for _, list := range smeltFrom {
			sort.Slice(list, func(i, j int) bool { return naming.ItemKey(list[i].Item()) < naming.ItemKey(list[j].Item()) })
		}
	})
	return smeltFrom[naming.ItemKey(it.Item())]
}

// codexSourceCards are the recipes that make it, as cards.
func (s *session) codexSourceCards(lang string, it item.Stack) []codexCard {
	k := naming.ItemKey(it.Item())
	var cards []codexCard
	for _, cr := range machinerecipe.Crafters() {
		if cr.OutKey == k {
			cards = append(cards, s.craftCard(lang, cr))
		}
	}
	furnace := item.NewStack(block.Furnace{}, 1)
	for _, in := range codexSmeltInputs(it) {
		cards = append(cards, machineCard(lang, furnace, []ui.BoardCell{s.codexCell(lang, in, "")}, []ui.BoardCell{s.codexCell(lang, it, "")}))
	}
	for _, r := range machinerecipe.Alloys {
		if naming.ItemKey(r.Out) == k {
			cards = append(cards, s.alloyCard(lang, r))
		}
	}
	for i := range machinerecipe.Labs {
		if naming.ItemKey(machinerecipe.Labs[i].Out) == k {
			cards = append(cards, s.labCard(lang, &machinerecipe.Labs[i]))
		}
	}
	for _, in := range crushInputs() {
		if got, ok := crusherOutput(item.NewStack(in, 1)); ok && naming.ItemKey(got.Item()) == k {
			cards = append(cards, s.crushCard(lang, in, got))
		}
	}
	if _, ok := it.Item().(sgitem.MetalDust); ok {
		cards = append(cards, noteCard(lang, machineItem(KindWasher), i18n.T(lang, "codex.src.washer")))
	}
	sieve := machineItem(KindSieve)
	for _, in := range codexSieveInputs() {
		for _, t := range sgitem.MeshTiers {
			if d, ok := sieveDropOf(t.Tier, in, k); ok {
				cards = append(cards, machineCard(lang, sieve,
					[]ui.BoardCell{s.codexCell(lang, item.NewStack(in, 1), ""), s.codexCell(lang, item.NewStack(sgitem.Mesh{Tier: t.Tier}, 1), "")},
					[]ui.BoardCell{s.chanceCell(lang, d.Item, d.Permille)}))
				break
			}
		}
	}
	for _, mob := range sgitem.MobKinds {
		for _, d := range sgitem.SimDrops[mob] {
			if naming.ItemKey(d.Item) == k {
				cards = append(cards, s.simCard(lang, mob, []sgitem.SimDrop{d}))
			}
		}
	}
	for _, mob := range sgitem.MobKinds {
		for _, in := range codexSieveInputs() {
			for _, d := range sievetable.MagicThemes[mob].Adds[naming.ItemKey(in)] {
				if naming.ItemKey(d.Item.Item()) == k {
					cards = append(cards, machineCard(lang, machineItem(KindMagicSieve),
						[]ui.BoardCell{s.codexCell(lang, item.NewStack(sgitem.Essence{Mob: mob}, 1), ""), s.codexCell(lang, item.NewStack(in, 1), "")},
						[]ui.BoardCell{s.chanceCell(lang, d.Item, d.Permille)}))
				}
			}
		}
	}
	if _, ok := it.Item().(sgitem.Essence); ok {
		cards = append(cards, noteCard(lang, machineItem(KindGate), i18n.T(lang, "codex.src.waveDrop")))
	}
	for i := range waveLevels {
		for _, r := range waveLevels[i].Reward {
			if naming.ItemKey(r.Item()) == k {
				cards = append(cards, s.waveCard(lang, i)...)
				break
			}
		}
	}
	for _, pk := range plantKinds {
		for _, y := range pk.yield {
			if naming.ItemKey(y.item) == k {
				cards = append(cards, s.plantCard(lang, pk))
			}
		}
	}
	return append(cards, s.codexWorkFor(lang, it, true)...)
}

// codexUseCards are the recipes it goes into, as cards.
func (s *session) codexUseCards(lang string, it item.Stack) []codexCard {
	k := naming.ItemKey(it.Item())
	var cards []codexCard
	for _, cr := range machinerecipe.Crafters() {
		if cr.Wants(it) {
			cards = append(cards, s.craftCard(lang, cr))
		}
	}
	if sm, ok := smeltOf(it); ok {
		cards = append(cards, machineCard(lang, item.NewStack(block.Furnace{}, 1), []ui.BoardCell{s.codexCell(lang, it, "")}, []ui.BoardCell{s.codexCell(lang, sm, countLabel(sm.Count()))}))
	}
	for _, r := range machinerecipe.Alloys {
		if naming.ItemKey(r.A) == k || naming.ItemKey(r.B) == k {
			cards = append(cards, s.alloyCard(lang, r))
		}
	}
	for i := range machinerecipe.Labs {
		for _, in := range machinerecipe.Labs[i].In {
			if naming.ItemKey(in.Item) == k {
				cards = append(cards, s.labCard(lang, &machinerecipe.Labs[i]))
				break
			}
		}
	}
	if got, ok := crusherOutput(it); ok {
		cards = append(cards, s.crushCard(lang, it.Item(), got))
	}
	if e, ok := it.Item().(sgitem.Essence); ok {
		cards = append(cards, s.simCard(lang, e.Mob, sgitem.SimDrops[e.Mob]))
		cards = append(cards, s.themeCards(lang, e.Mob)...)
	}
	research := item.NewStack(ResearchTable{}, 1)
	for i := range researchTree {
		r := &researchTree[i]
		for _, c := range r.Cost {
			if naming.ItemKey(c.Item) == k {
				cards = append(cards, noteCard(lang, research, fmt.Sprintf(i18n.T(lang, "codex.use.research"), r.Title(lang), c.Count)))
			}
		}
	}
	if fuelDuration(it) > 0 {
		cards = append(cards, noteCard(lang, machineItem(KindGenerator), i18n.T(lang, "codex.use.fuel")))
	}
	if plantMatter(it) {
		cards = append(cards, noteCard(lang, item.NewStack(block.Composter{}, 1), i18n.T(lang, "codex.use.compost")))
	}
	tinkerTable := machineItem(KindTinker)
	for _, h := range tinker.HeadMaterials {
		if naming.ItemKey(h.Item) == k {
			cards = append(cards, noteCard(lang, tinkerTable, fmt.Sprintf(i18n.T(lang, "codex.use.tool"), i18n.T(lang, "tmat."+h.ID), i18n.T(lang, "codex.use.head"))))
		}
	}
	for _, h := range tinker.HandleMaterials {
		if naming.ItemKey(h.Item) == k {
			cards = append(cards, noteCard(lang, tinkerTable, fmt.Sprintf(i18n.T(lang, "codex.use.tool"), i18n.T(lang, "thandle."+h.ID), i18n.T(lang, "codex.use.handle"))))
		}
	}
	for _, m := range tinker.ToolModifiers {
		if naming.ItemKey(m.Item) == k {
			cards = append(cards, noteCard(lang, tinkerTable, fmt.Sprintf(i18n.T(lang, "codex.use.tool"), i18n.T(lang, "tmod."+m.ID), fmt.Sprintf("×%d", m.Cost))))
		}
	}
	return append(cards, s.codexWorkFor(lang, it, false)...)
}

// crushInputs are the blocks the crusher takes.
func crushInputs() []world.Item {
	return append([]world.Item{
		block.Stone{}, block.Cobblestone{}, block.Gravel{}, block.Sand{}, block.Sand{Red: true},
		block.Deepslate{}, block.Deepslate{Type: block.CobbledDeepslate()}, block.Netherrack{},
		sgitem.CrushedBlock{Kind: sgitem.CrushedNetherrack}, block.Blackstone{}, block.EndStone{},
	}, sgitem.OreBlocks()...)
}

// sieveDropOf is the drop with the key in a mesh's table for a block.
func sieveDropOf(tier string, in world.Item, key string) (sievetable.Drop, bool) {
	for _, d := range sievetable.Tables[tier][naming.ItemKey(in)] {
		if naming.ItemKey(d.Item.Item()) == key {
			return d, true
		}
	}
	return sievetable.Drop{}, false
}

// chanceCell is a drop with its chance in per mille on the slot. A stack of
// several is rolled once for each (sieveDrops), which the tip tells.
func (s *session) chanceCell(lang string, st item.Stack, permille int) ui.BoardCell {
	tip := fmt.Sprintf(i18n.T(lang, "codex.sieve.chance"), textfmt.Permille(permille))
	if st.Count() > 1 {
		tip = fmt.Sprintf(i18n.T(lang, "codex.sieve.rolls"), st.Count(), textfmt.Permille(permille))
	}
	return s.codexCell(lang, st, "§f"+textfmt.Permille(permille)+"%", tip)
}

func (s *session) alloyCard(lang string, r machinerecipe.Alloy) codexCard {
	return machineCard(lang, machineItem(KindAlloy),
		[]ui.BoardCell{s.codexCell(lang, item.NewStack(r.A, 1), countLabel(r.CountA)), s.codexCell(lang, item.NewStack(r.B, 1), countLabel(r.CountB))},
		[]ui.BoardCell{s.codexCell(lang, item.NewStack(r.Out, 1), countLabel(r.OutCount))})
}

func (s *session) labCard(lang string, r *machinerecipe.Lab) codexCard {
	var ins []ui.BoardCell
	for _, in := range r.In {
		ins = append(ins, s.codexCell(lang, item.NewStack(in.Item, 1), countLabel(in.Count)))
	}
	return machineCard(lang, machineItem(KindLab), ins, []ui.BoardCell{s.codexCell(lang, item.NewStack(r.Out, 1), countLabel(r.OutCount))})
}

func (s *session) crushCard(lang string, in world.Item, out item.Stack) codexCard {
	outs := []ui.BoardCell{s.codexCell(lang, out, countLabel(out.Count()))}
	if bonus, ok := crusherBonusOf(item.NewStack(in, 1)); ok {
		for range sgitem.CrusherBonus {
			outs = append(outs, s.codexCell(lang, bonus, fmt.Sprintf("§f%d%%", sgitem.CrusherBonusChance), fmt.Sprintf(i18n.T(lang, "codex.card.chance"), sgitem.CrusherBonusChance)))
		}
	}
	return machineCard(lang, machineItem(KindCrusher), []ui.BoardCell{s.codexCell(lang, item.NewStack(in, 1), "")}, outs)
}

// simCard is the drops of a mob's essence in the simulator.
func (s *session) simCard(lang string, mob string, drops []sgitem.SimDrop) codexCard {
	var outs []ui.BoardCell
	for _, d := range drops {
		outs = append(outs, s.codexCell(lang, item.NewStack(d.Item, 1), fmt.Sprintf("§f%d%%", d.Chance),
			fmt.Sprintf(i18n.T(lang, "codex.card.simDrop"), rangeText(d.Min, d.Max), d.Chance)))
	}
	return machineCard(lang, machineItem(KindSimulator), []ui.BoardCell{s.codexCell(lang, item.NewStack(sgitem.Essence{Mob: mob}, 1), "")}, outs)
}

// themeCards are what a mob's essence adds to each block in the magic sieve.
func (s *session) themeCards(lang string, mob string) []codexCard {
	th := sievetable.MagicThemes[mob]
	var cards []codexCard
	for _, in := range codexSieveInputs() {
		drops := th.Adds[naming.ItemKey(in)]
		if len(drops) == 0 {
			continue
		}
		var outs []ui.BoardCell
		for _, d := range drops {
			extra := []string{fmt.Sprintf(i18n.T(lang, "codex.sieve.chance"), textfmt.Permille(d.Permille))}
			if d.Rare {
				extra = append(extra, i18n.T(lang, "codex.card.rare"))
			}
			outs = append(outs, s.codexCell(lang, d.Item, "§f"+textfmt.Permille(d.Permille)+"%", extra...))
		}
		block := s.codexCell(lang, item.NewStack(in, 1), "")
		if th.Replace[naming.ItemKey(in)] {
			block = s.codexCell(lang, item.NewStack(in, 1), "", i18n.T(lang, "codex.card.replace"))
		}
		cards = append(cards, machineCard(lang, machineItem(KindMagicSieve), []ui.BoardCell{s.codexCell(lang, item.NewStack(sgitem.Essence{Mob: mob}, 1), ""), block}, outs))
	}
	return cards
}

// waveCard is a gate level: its number and time, the arrow and the first
// clear rewards, and under them the mobs that come.
func (s *session) waveCard(lang string, i int) []codexCard {
	w := waveLevels[i]
	var outs []ui.BoardCell
	for _, r := range w.Reward {
		outs = append(outs, s.codexCell(lang, r, countLabel(r.Count())))
	}
	c := machineCard(lang, machineItem(KindGate), []ui.BoardCell{ui.TextCell(fmt.Sprintf(i18n.T(lang, "codex.card.wave"), i+1, w.Seconds))}, outs)
	var groups string
	for j, g := range w.Groups {
		if j > 0 {
			groups += ", "
		}
		groups += fmt.Sprintf("%s ×%d", mobName(lang, g.Mob), g.N)
	}
	row := blankRow()
	row[0] = ui.TextCell("§8" + groups)
	c.rows = append(c.rows, row)
	return []codexCard{c}
}

// plantCard is a magic plant: its seed, the arrow and what it yields.
func (s *session) plantCard(lang string, pk plantKind) codexCard {
	seed := item.NewStack(PlantSeed{Seed: pk.seed}, 1)
	var outs []ui.BoardCell
	for _, y := range pk.yield {
		outs = append(outs, s.codexCell(lang, item.NewStack(y.item, 1), "§f"+rangeText(y.min, y.max)))
	}
	head := seed
	c := machineCard(lang, head, []ui.BoardCell{s.codexCell(lang, seed, "")}, outs)
	c.head, c.name = item.Stack{}, i18n.T(lang, "codex.plants")
	return c
}

// codexProcess is one machine of the process tab: its block and its recipes.
type codexProcess struct {
	head  item.Stack
	cards func(s *session, lang string) []codexCard
}

// codexProcesses are the machines of the process tab, in the order of the
// row the player picks from.
func codexProcesses() []codexProcess {
	water := item.NewStack(item.Bucket{Content: item.LiquidBucketContent(block.Water{Depth: 8, Still: true})}, 1)
	lava := item.NewStack(item.Bucket{Content: item.LiquidBucketContent(block.Lava{Depth: 8, Still: true})}, 1)
	return []codexProcess{
		{machineItem(KindCrusher), func(s *session, lang string) []codexCard {
			var cards []codexCard
			for _, in := range crushInputs() {
				if got, ok := crusherOutput(item.NewStack(in, 1)); ok {
					cards = append(cards, s.crushCard(lang, in, got))
				}
			}
			return cards
		}},
		{machineItem(KindWasher), func(s *session, lang string) []codexCard {
			var cards []codexCard
			for _, k := range sgitem.OreKinds {
				in := item.NewStack(sgitem.RawOf(k.Ore), 1)
				out, _ := sgitem.WasherOutput(in)
				cards = append(cards, machineCard(lang, machineItem(KindWasher), []ui.BoardCell{s.codexCell(lang, in, ""), s.codexCell(lang, water, "", i18n.T(lang, "codex.card.water"))}, []ui.BoardCell{s.codexCell(lang, out, countLabel(out.Count()))}))
			}
			return cards
		}},
		{machineItem(KindFurnace), func(s *session, lang string) []codexCard {
			var cards []codexCard
			// from every registered item (the index is built once the
			// registry is complete; this may run before)
			for _, in := range smeltables() {
				got, _ := smeltOf(in)
				cards = append(cards, machineCard(lang, machineItem(KindFurnace), []ui.BoardCell{s.codexCell(lang, in, "")}, []ui.BoardCell{s.codexCell(lang, got, countLabel(got.Count()))}))
			}
			return cards
		}},
		{machineItem(KindAlloy), func(s *session, lang string) []codexCard {
			var cards []codexCard
			for _, r := range machinerecipe.Alloys {
				cards = append(cards, s.alloyCard(lang, r))
			}
			return cards
		}},
		{machineItem(KindLab), func(s *session, lang string) []codexCard {
			var cards []codexCard
			for i := range machinerecipe.Labs {
				cards = append(cards, s.labCard(lang, &machinerecipe.Labs[i]))
			}
			return cards
		}},
		{machineItem(KindCrucible), func(s *session, lang string) []codexCard {
			return []codexCard{machineCard(lang, machineItem(KindCrucible), []ui.BoardCell{s.codexCell(lang, item.NewStack(block.Cobblestone{}, 1), "")},
				[]ui.BoardCell{s.codexCell(lang, lava, "", fmt.Sprintf(i18n.T(lang, "codex.card.mb"), crucibleLava))})}
		}},
		{item.NewStack(Barrel{Wood: "oak"}, 1), func(s *session, lang string) []codexCard {
			barrel := item.NewStack(Barrel{Wood: "oak"}, 1)
			plants := s.codexCell(lang, item.NewStack(block.Leaves{Type: block.OakLeaves()}, 1), countLabel(barrelCapacity("oak")), i18n.T(lang, "codex.card.anyPlant"))
			return append([]codexCard{
				machineCard(lang, barrel, []ui.BoardCell{plants}, []ui.BoardCell{s.codexCell(lang, water, "")}),
				machineCard(lang, barrel, []ui.BoardCell{s.codexCell(lang, water, ""), s.codexCell(lang, item.NewStack(sgitem.Dust{}, 1), "")}, []ui.BoardCell{s.codexCell(lang, item.NewStack(block.Clay{}, 1), "")}),
				machineCard(lang, barrel, []ui.BoardCell{s.codexCell(lang, water, ""), s.codexCell(lang, item.NewStack(block.Mycelium{}, 1), "", i18n.T(lang, "codex.card.under"))}, []ui.BoardCell{s.codexCell(lang, item.NewStack(sgitem.WitchWaterBucket{}, 1), "")}),
			}, s.barrelMixCards(lang, barrel, false)...)
		}},
		{item.NewStack(Barrel{Wood: stoneWood}, 1), func(s *session, lang string) []codexCard {
			barrel := item.NewStack(Barrel{Wood: stoneWood}, 1)
			heat := s.codexCell(lang, item.NewStack(block.Torch{}, 1), "", i18n.T(lang, "codex.card.heat"))
			var cards []codexCard
			for _, m := range stoneMelts {
				cards = append(cards, machineCard(lang, barrel,
					[]ui.BoardCell{s.codexCell(lang, item.NewStack(m, 1), ""), heat},
					[]ui.BoardCell{s.codexCell(lang, lava, "", fmt.Sprintf(i18n.T(lang, "codex.card.mb"), stoneMeltMB))}))
			}
			return append(cards, s.barrelMixCards(lang, barrel, true)...)
		}},
		{machineItem(KindTinker), func(s *session, lang string) []codexCard { return s.codexGearCards(lang) }},
	}
}

// codexGearCards are the tinker table's head and handle materials and
// modifiers: the material's slot with what it gives.
func (s *session) codexGearCards(lang string) []codexCard {
	tinkerTable := machineItem(KindTinker)
	var cards []codexCard
	gear := func(it world.Item, text string) {
		row := blankRow()
		row[0] = s.codexCell(lang, item.NewStack(it, 1), "", text)
		row[1] = ui.TextCell(text)
		cards = append(cards, codexCard{head: tinkerTable, name: naming.UIItemName(lang, tinkerTable), rows: [][]ui.BoardCell{row}})
	}
	for _, h := range tinker.HeadMaterials {
		gear(h.Item, fmt.Sprintf(i18n.T(lang, "codex.card.head"), i18n.T(lang, "tmat."+h.ID), h.Dur, h.Eff, h.Attack))
	}
	for _, h := range tinker.HandleMaterials {
		gear(h.Item, fmt.Sprintf(i18n.T(lang, "codex.card.handle"), i18n.T(lang, "thandle."+h.ID)))
	}
	for _, m := range tinker.ToolModifiers {
		gear(m.Item, fmt.Sprintf(i18n.T(lang, "codex.card.mod"), i18n.T(lang, "tmod."+m.ID), m.Cost))
	}
	return cards
}

// codexMobCards are the mobs tab: the simulator, the magic sieve's themes
// and the gate's levels.
func (s *session) codexMobCards(lang string) []codexCard {
	var cards []codexCard
	for _, mob := range sgitem.MobKinds {
		cards = append(cards, s.simCard(lang, mob, sgitem.SimDrops[mob]))
	}
	for _, mob := range sgitem.MobKinds {
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
	for _, t := range sgitem.MeshTiers {
		drops := append([]sievetable.Drop(nil), sievetable.Tables[t.Tier][naming.ItemKey(in)]...)
		sort.SliceStable(drops, func(i, j int) bool { return drops[i].Permille > drops[j].Permille })
		var outs []ui.BoardCell
		for _, d := range drops {
			outs = append(outs, s.chanceCell(lang, d.Item, d.Permille))
		}
		cards = append(cards, machineCard(lang, machineItem(KindSieve),
			[]ui.BoardCell{s.codexCell(lang, item.NewStack(sgitem.Mesh{Tier: t.Tier}, 1), "")}, outs))
	}
	return cards
}

// smeltables are the registered items a furnace smelts, by key.
func smeltables() []item.Stack {
	var out []item.Stack
	seen := map[string]bool{}
	for _, w := range world.Items() {
		st := item.NewStack(w, 1)
		k := naming.ItemKey(w)
		if _, ok := smeltOf(st); ok && !seen[k] {
			seen[k] = true
			out = append(out, st)
		}
	}
	sort.Slice(out, func(i, j int) bool { return naming.ItemKey(out[i].Item()) < naming.ItemKey(out[j].Item()) })
	return out
}
