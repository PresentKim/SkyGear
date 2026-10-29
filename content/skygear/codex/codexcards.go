package codex

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
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/decor"
	"github.com/presentkim/skygear/content/skygear/farm"
	"github.com/presentkim/skygear/content/skygear/formicon"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/machinerecipe"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/sievetable"
	"github.com/presentkim/skygear/content/skygear/textfmt"
	"github.com/presentkim/skygear/content/skygear/tinker"
	"github.com/presentkim/skygear/content/skygear/wave"
)

// The codex draws each recipe as a card of item slots on a board
// (ui.ItemBoard), JEI style: inputs, an arrow and outputs, grouped under one
// header row per machine. Details are in the slot tooltips.

// Columns of a card: inputs from the left (at most cardInCols to a row), the
// arrow right after them, then the outputs to the end of the row. A crafting
// card's grid is three wide: its arrow is at craftArrow.
const (
	cardInCols = 4
	CraftArrow = 3
)

// Card is one recipe: the machine it is made in (the header) and its
// rows of cells.
type Card struct {
	Head item.Stack
	name string
	Rows [][]ui.BoardCell
}

func BlankRow() []ui.BoardCell { return make([]ui.BoardCell, ui.BoardColumns) }

// countLabel is the count shown on a slot (none for 1).
func countLabel(n int) string {
	if n > 1 {
		return "§f" + ui.CountText(n)
	}
	return ""
}

// codexCell is a slot showing st that opens its codex page. extra lines go
// under the name in the tooltip.
func codexCell(s *core.Session, lang string, st item.Stack, count string, extra ...string) ui.BoardCell {
	if st.Empty() {
		return ui.Blank()
	}
	st = item.NewStack(st.Item(), 1)
	tip := decor.TipName(lang, st)
	for _, e := range extra {
		tip += "\n§7" + e
	}
	return ui.SlotCell(ui.ItemSlot{Icon: formicon.Of(st), Count: count, Tip: tip, Run: func(p *player.Player, _ *world.Tx) { codexGo(s, p, func(p *player.Player) { ShowItem(s, p, st) }) }})
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
func ingredientCell(s *core.Session, lang string, in recipe.Item) ui.BoardCell {
	switch in := in.(type) {
	case item.Stack:
		return codexCell(s, lang, in, countLabel(in.Count()))
	case recipe.ItemTag:
		return codexCell(s, lang, tagStack(in), countLabel(in.Count()), machinerecipe.ReqName(lang, machinerecipe.CrafterReq{Need: in}))
	}
	return ui.Blank()
}

// CraftCard is a crafting table recipe: its grid, the arrow and the result.
func CraftCard(s *core.Session, lang string, cr *machinerecipe.Crafter) Card {
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
		rows[y] = BlankRow()
		for x := range 3 {
			rows[y][x] = ui.Frame()
		}
	}
	for i, in := range ins {
		if in == nil || in.Empty() || i%w >= 3 || i/w >= h {
			continue
		}
		rows[i/w][i%w] = ingredientCell(s, lang, in)
	}
	mid := (h - 1) / 2
	rows[mid][CraftArrow] = ui.Arrow()
	rows[mid][CraftArrow+1] = codexCell(s, lang, cr.Out, countLabel(cr.Out.Count()))
	table := item.NewStack(block.CraftingTable{}, 1)
	return Card{Head: table, name: naming.UIItemName(lang, table), Rows: rows}
}

// machineCard is a machine recipe: inputs (four to a row), the arrow and the
// outputs (six to a row).
func machineCard(lang string, head item.Stack, ins, outs []ui.BoardCell) Card {
	inCols := max(1, min(len(ins), cardInCols))
	arrow, outCol := inCols, inCols+1
	outCols := ui.BoardColumns - outCol
	h := max(1, (len(ins)+inCols-1)/inCols, (len(outs)+outCols-1)/outCols)
	rows := make([][]ui.BoardCell, h)
	for y := range rows {
		rows[y] = BlankRow()
	}
	for i, c := range ins {
		rows[i/inCols][i%inCols] = c
	}
	rows[0][arrow] = ui.Arrow()
	for i, c := range outs {
		rows[i/outCols][outCol+i%outCols] = c
	}
	return Card{Head: head, name: naming.UIItemName(lang, head), Rows: rows}
}

// noteCard is a line of text under a machine's header (what has no recipe
// to draw: a fuel, a research cost).
func noteCard(lang string, head item.Stack, text string) Card {
	row := BlankRow()
	row[0] = ui.TextCell(text)
	return Card{Head: head, name: naming.UIItemName(lang, head), Rows: [][]ui.BoardCell{row}}
}

// codexHeadRow is the header of a group of cards: the machine and its name.
func codexHeadRow(s *core.Session, lang string, c Card) []ui.BoardCell {
	row := BlankRow()
	if !c.Head.Empty() {
		// the machine is a small icon without a slot frame, so it does not read
		// as a slot of the recipes under it
		sl, _ := codexCell(s, lang, c.Head, "").SlotOf()
		row[0] = ui.MiniCell(sl)
		row[1] = ui.TextCell("§e" + c.name)
	} else {
		row[0] = ui.TextCell("§e" + c.name)
	}
	return row
}

// Board lays the cards out the way JEI shows recipes: each machine's
// cards centred in the window, two side by side when two fit, a blank row
// between them. Cards of the same machine share one header, at their left
// edge; a new machine's header comes after a blank row too, so where one
// source ends and the next begins shows.
func Board(s *core.Session, lang string, cards []Card) [][]ui.BoardCell {
	var rows [][]ui.BoardCell
	last := ""
	for _, b := range Blocks(cards) {
		switch {
		case (len(rows) == 0 || b.name != last) && b.name != "":
			if len(rows) > 0 {
				rows = append(rows, BlankRow())
			}
			rows = append(rows, shiftRow(codexHeadRow(s, lang, Card{Head: b.Head, name: b.name}), b.Offset))
		case len(rows) > 0:
			rows = append(rows, BlankRow())
		}
		rows = append(rows, b.Rows...)
		last = b.name
	}
	return rows
}

// codexBlock is a row of one or two cards of the same machine, placed in the
// window's columns.
type codexBlock struct {
	Head   item.Stack
	name   string
	Offset int
	Rows   [][]ui.BoardCell
}

// Blocks places the cards: the cards of a machine share a width (the
// widest one's), so their arrows line up, and are centred; two go side by
// side, with a gap of up to three cells, when both fit a row.
func Blocks(cards []Card) []codexBlock {
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
			b := codexBlock{Head: cards[k].Head, name: cards[k].name, Offset: offset}
			for n, c := range cards[k:min(k+perRow, j)] {
				for r, row := range c.Rows {
					if r == len(b.Rows) {
						b.Rows = append(b.Rows, BlankRow())
					}
					at := offset + n*(w+gap)
					copy(b.Rows[r][at:], row[:min(len(row), w)])
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
func cardWidth(c Card) int {
	w := 1
	for _, row := range c.Rows {
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
	out := BlankRow()
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
			if got, ok := machine.SmeltOf(st); ok {
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

// SourceCards are the recipes that make it, as cards.
func SourceCards(s *core.Session, lang string, it item.Stack) []Card {
	k := naming.ItemKey(it.Item())
	var cards []Card
	for _, cr := range machinerecipe.Crafters() {
		if cr.OutKey == k {
			cards = append(cards, CraftCard(s, lang, cr))
		}
	}
	furnace := item.NewStack(block.Furnace{}, 1)
	for _, in := range codexSmeltInputs(it) {
		cards = append(cards, machineCard(lang, furnace, []ui.BoardCell{codexCell(s, lang, in, "")}, []ui.BoardCell{codexCell(s, lang, it, "")}))
	}
	for _, r := range machinerecipe.Alloys {
		if naming.ItemKey(r.Out) == k {
			cards = append(cards, alloyCard(s, lang, r))
		}
	}
	for i := range machinerecipe.Labs {
		if naming.ItemKey(machinerecipe.Labs[i].Out) == k {
			cards = append(cards, labCard(s, lang, &machinerecipe.Labs[i]))
		}
	}
	for _, in := range CrushInputs() {
		if got, ok := machine.CrusherOutput(item.NewStack(in, 1)); ok && naming.ItemKey(got.Item()) == k {
			cards = append(cards, crushCard(s, lang, in, got))
		}
	}
	if _, ok := it.Item().(sgitem.MetalDust); ok {
		cards = append(cards, noteCard(lang, machine.Item(machine.KindWasher), i18n.T(lang, "codex.src.washer")))
	}
	sieve := machine.Item(machine.KindSieve)
	for _, in := range SieveInputs() {
		for _, t := range sgitem.MeshTiers {
			if d, ok := sieveDropOf(t.Tier, in, k); ok {
				cards = append(cards, machineCard(lang, sieve,
					[]ui.BoardCell{codexCell(s, lang, item.NewStack(in, 1), ""), codexCell(s, lang, item.NewStack(sgitem.Mesh{Tier: t.Tier}, 1), "")},
					[]ui.BoardCell{chanceCell(s, lang, d.Item, d.Permille)}))
				break
			}
		}
	}
	for _, mob := range sgitem.MobKinds {
		for _, d := range sgitem.SimDrops[mob] {
			if naming.ItemKey(d.Item) == k {
				cards = append(cards, simCard(s, lang, mob, []sgitem.SimDrop{d}))
			}
		}
	}
	for _, mob := range sgitem.MobKinds {
		for _, in := range SieveInputs() {
			for _, d := range sievetable.MagicThemes[mob].Adds[naming.ItemKey(in)] {
				if naming.ItemKey(d.Item.Item()) == k {
					cards = append(cards, machineCard(lang, machine.Item(machine.KindMagicSieve),
						[]ui.BoardCell{codexCell(s, lang, item.NewStack(sgitem.Essence{Mob: mob}, 1), ""), codexCell(s, lang, item.NewStack(in, 1), "")},
						[]ui.BoardCell{chanceCell(s, lang, d.Item, d.Permille)}))
				}
			}
		}
	}
	if _, ok := it.Item().(sgitem.Essence); ok {
		cards = append(cards, noteCard(lang, machine.Item(machine.KindGate), i18n.T(lang, "codex.src.waveDrop")))
	}
	for i := range wave.Levels {
		for _, r := range wave.Levels[i].Reward {
			if naming.ItemKey(r.Item()) == k {
				cards = append(cards, waveCard(s, lang, i)...)
				break
			}
		}
	}
	for _, pk := range farm.PlantKinds {
		for _, y := range pk.Yield {
			if naming.ItemKey(y.Item) == k {
				cards = append(cards, plantCard(s, lang, pk))
			}
		}
	}
	return append(cards, codexWorkFor(s, lang, it, true)...)
}

// UseCards are the recipes it goes into, as cards.
func UseCards(s *core.Session, lang string, it item.Stack) []Card {
	k := naming.ItemKey(it.Item())
	var cards []Card
	for _, cr := range machinerecipe.Crafters() {
		if cr.Wants(it) {
			cards = append(cards, CraftCard(s, lang, cr))
		}
	}
	if sm, ok := machine.SmeltOf(it); ok {
		cards = append(cards, machineCard(lang, item.NewStack(block.Furnace{}, 1), []ui.BoardCell{codexCell(s, lang, it, "")}, []ui.BoardCell{codexCell(s, lang, sm, countLabel(sm.Count()))}))
	}
	for _, r := range machinerecipe.Alloys {
		if naming.ItemKey(r.A) == k || naming.ItemKey(r.B) == k {
			cards = append(cards, alloyCard(s, lang, r))
		}
	}
	for i := range machinerecipe.Labs {
		for _, in := range machinerecipe.Labs[i].In {
			if naming.ItemKey(in.Item) == k {
				cards = append(cards, labCard(s, lang, &machinerecipe.Labs[i]))
				break
			}
		}
	}
	if got, ok := machine.CrusherOutput(it); ok {
		cards = append(cards, crushCard(s, lang, it.Item(), got))
	}
	if e, ok := it.Item().(sgitem.Essence); ok {
		cards = append(cards, simCard(s, lang, e.Mob, sgitem.SimDrops[e.Mob]))
		cards = append(cards, themeCards(s, lang, e.Mob)...)
	}
	research := item.NewStack(progress.ResearchTable{}, 1)
	for i := range progress.Tree {
		r := &progress.Tree[i]
		for _, c := range r.Cost {
			if naming.ItemKey(c.Item) == k {
				cards = append(cards, noteCard(lang, research, fmt.Sprintf(i18n.T(lang, "codex.use.research"), r.Title(lang), c.Count)))
			}
		}
	}
	if machine.FuelDuration(it) > 0 {
		cards = append(cards, noteCard(lang, machine.Item(machine.KindGenerator), i18n.T(lang, "codex.use.fuel")))
	}
	if farm.PlantMatter(it) {
		cards = append(cards, noteCard(lang, item.NewStack(block.Composter{}, 1), i18n.T(lang, "codex.use.compost")))
	}
	tinkerTable := machine.Item(machine.KindTinker)
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
	return append(cards, codexWorkFor(s, lang, it, false)...)
}

// CrushInputs are the blocks the crusher takes.
func CrushInputs() []world.Item {
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
func chanceCell(s *core.Session, lang string, st item.Stack, permille int) ui.BoardCell {
	tip := fmt.Sprintf(i18n.T(lang, "codex.sieve.chance"), textfmt.Permille(permille))
	if st.Count() > 1 {
		tip = fmt.Sprintf(i18n.T(lang, "codex.sieve.rolls"), st.Count(), textfmt.Permille(permille))
	}
	return codexCell(s, lang, st, "§f"+textfmt.Permille(permille)+"%", tip)
}

func alloyCard(s *core.Session, lang string, r machinerecipe.Alloy) Card {
	return machineCard(lang, machine.Item(machine.KindAlloy),
		[]ui.BoardCell{codexCell(s, lang, item.NewStack(r.A, 1), countLabel(r.CountA)), codexCell(s, lang, item.NewStack(r.B, 1), countLabel(r.CountB))},
		[]ui.BoardCell{codexCell(s, lang, item.NewStack(r.Out, 1), countLabel(r.OutCount))})
}

func labCard(s *core.Session, lang string, r *machinerecipe.Lab) Card {
	var ins []ui.BoardCell
	for _, in := range r.In {
		ins = append(ins, codexCell(s, lang, item.NewStack(in.Item, 1), countLabel(in.Count)))
	}
	return machineCard(lang, machine.Item(machine.KindLab), ins, []ui.BoardCell{codexCell(s, lang, item.NewStack(r.Out, 1), countLabel(r.OutCount))})
}

func crushCard(s *core.Session, lang string, in world.Item, out item.Stack) Card {
	outs := []ui.BoardCell{codexCell(s, lang, out, countLabel(out.Count()))}
	if bonus, ok := machine.CrusherBonusOf(item.NewStack(in, 1)); ok {
		for range sgitem.CrusherBonus {
			outs = append(outs, codexCell(s, lang, bonus, fmt.Sprintf("§f%d%%", sgitem.CrusherBonusChance), fmt.Sprintf(i18n.T(lang, "codex.card.chance"), sgitem.CrusherBonusChance)))
		}
	}
	return machineCard(lang, machine.Item(machine.KindCrusher), []ui.BoardCell{codexCell(s, lang, item.NewStack(in, 1), "")}, outs)
}

// simCard is the drops of a mob's essence in the simulator.
func simCard(s *core.Session, lang string, mob string, drops []sgitem.SimDrop) Card {
	var outs []ui.BoardCell
	for _, d := range drops {
		outs = append(outs, codexCell(s, lang, item.NewStack(d.Item, 1), fmt.Sprintf("§f%d%%", d.Chance),
			fmt.Sprintf(i18n.T(lang, "codex.card.simDrop"), textfmt.Range(d.Min, d.Max), d.Chance)))
	}
	return machineCard(lang, machine.Item(machine.KindSimulator), []ui.BoardCell{codexCell(s, lang, item.NewStack(sgitem.Essence{Mob: mob}, 1), "")}, outs)
}

// themeCards are what a mob's essence adds to each block in the magic sieve.
func themeCards(s *core.Session, lang string, mob string) []Card {
	th := sievetable.MagicThemes[mob]
	var cards []Card
	for _, in := range SieveInputs() {
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
			outs = append(outs, codexCell(s, lang, d.Item, "§f"+textfmt.Permille(d.Permille)+"%", extra...))
		}
		block := codexCell(s, lang, item.NewStack(in, 1), "")
		if th.Replace[naming.ItemKey(in)] {
			block = codexCell(s, lang, item.NewStack(in, 1), "", i18n.T(lang, "codex.card.replace"))
		}
		cards = append(cards, machineCard(lang, machine.Item(machine.KindMagicSieve), []ui.BoardCell{codexCell(s, lang, item.NewStack(sgitem.Essence{Mob: mob}, 1), ""), block}, outs))
	}
	return cards
}

// waveCard is a gate level: its number and time, the arrow and the first
// clear rewards, and under them the mobs that come.
func waveCard(s *core.Session, lang string, i int) []Card {
	w := wave.Levels[i]
	var outs []ui.BoardCell
	for _, r := range w.Reward {
		outs = append(outs, codexCell(s, lang, r, countLabel(r.Count())))
	}
	c := machineCard(lang, machine.Item(machine.KindGate), []ui.BoardCell{ui.TextCell(fmt.Sprintf(i18n.T(lang, "codex.card.wave"), i+1, w.Seconds))}, outs)
	var groups string
	for j, g := range w.Groups {
		if j > 0 {
			groups += ", "
		}
		groups += fmt.Sprintf("%s ×%d", wave.MobName(lang, g.Mob), g.N)
	}
	row := BlankRow()
	row[0] = ui.TextCell("§8" + groups)
	c.Rows = append(c.Rows, row)
	return []Card{c}
}

// plantCard is a magic plant: its seed, the arrow and what it yields.
func plantCard(s *core.Session, lang string, pk farm.PlantKind) Card {
	seed := item.NewStack(farm.PlantSeed{Seed: pk.Seed}, 1)
	var outs []ui.BoardCell
	for _, y := range pk.Yield {
		outs = append(outs, codexCell(s, lang, item.NewStack(y.Item, 1), "§f"+textfmt.Range(y.Min, y.Max)))
	}
	head := seed
	c := machineCard(lang, head, []ui.BoardCell{codexCell(s, lang, seed, "")}, outs)
	c.Head, c.name = item.Stack{}, i18n.T(lang, "codex.plants")
	return c
}

// codexProcess is one machine of the process tab: its block and its recipes.
type codexProcess struct {
	Head  item.Stack
	Cards func(s *core.Session, lang string) []Card
}

// Processes are the machines of the process tab, in the order of the
// row the player picks from.
func Processes() []codexProcess {
	water := item.NewStack(item.Bucket{Content: item.LiquidBucketContent(block.Water{Depth: 8, Still: true})}, 1)
	lava := item.NewStack(item.Bucket{Content: item.LiquidBucketContent(block.Lava{Depth: 8, Still: true})}, 1)
	return []codexProcess{
		{machine.Item(machine.KindCrusher), func(s *core.Session, lang string) []Card {
			var cards []Card
			for _, in := range CrushInputs() {
				if got, ok := machine.CrusherOutput(item.NewStack(in, 1)); ok {
					cards = append(cards, crushCard(s, lang, in, got))
				}
			}
			return cards
		}},
		{machine.Item(machine.KindWasher), func(s *core.Session, lang string) []Card {
			var cards []Card
			for _, k := range sgitem.OreKinds {
				in := item.NewStack(sgitem.RawOf(k.Ore), 1)
				out, _ := sgitem.WasherOutput(in)
				cards = append(cards, machineCard(lang, machine.Item(machine.KindWasher), []ui.BoardCell{codexCell(s, lang, in, ""), codexCell(s, lang, water, "", i18n.T(lang, "codex.card.water"))}, []ui.BoardCell{codexCell(s, lang, out, countLabel(out.Count()))}))
			}
			return cards
		}},
		{machine.Item(machine.KindFurnace), func(s *core.Session, lang string) []Card {
			var cards []Card
			// from every registered item (the index is built once the
			// registry is complete; this may run before)
			for _, in := range smeltables() {
				got, _ := machine.SmeltOf(in)
				cards = append(cards, machineCard(lang, machine.Item(machine.KindFurnace), []ui.BoardCell{codexCell(s, lang, in, "")}, []ui.BoardCell{codexCell(s, lang, got, countLabel(got.Count()))}))
			}
			return cards
		}},
		{machine.Item(machine.KindAlloy), func(s *core.Session, lang string) []Card {
			var cards []Card
			for _, r := range machinerecipe.Alloys {
				cards = append(cards, alloyCard(s, lang, r))
			}
			return cards
		}},
		{machine.Item(machine.KindLab), func(s *core.Session, lang string) []Card {
			var cards []Card
			for i := range machinerecipe.Labs {
				cards = append(cards, labCard(s, lang, &machinerecipe.Labs[i]))
			}
			return cards
		}},
		{machine.Item(machine.KindCrucible), func(s *core.Session, lang string) []Card {
			return []Card{machineCard(lang, machine.Item(machine.KindCrucible), []ui.BoardCell{codexCell(s, lang, item.NewStack(block.Cobblestone{}, 1), "")},
				[]ui.BoardCell{codexCell(s, lang, lava, "", fmt.Sprintf(i18n.T(lang, "codex.card.mb"), machine.CrucibleLava))})}
		}},
		{item.NewStack(farm.Barrel{Wood: "oak"}, 1), func(s *core.Session, lang string) []Card {
			barrel := item.NewStack(farm.Barrel{Wood: "oak"}, 1)
			plants := codexCell(s, lang, item.NewStack(block.Leaves{Type: block.OakLeaves()}, 1), countLabel(farm.BarrelCapacity("oak")), i18n.T(lang, "codex.card.anyPlant"))
			return append([]Card{
				machineCard(lang, barrel, []ui.BoardCell{plants}, []ui.BoardCell{codexCell(s, lang, water, "")}),
				machineCard(lang, barrel, []ui.BoardCell{codexCell(s, lang, water, ""), codexCell(s, lang, item.NewStack(sgitem.Dust{}, 1), "")}, []ui.BoardCell{codexCell(s, lang, item.NewStack(block.Clay{}, 1), "")}),
				machineCard(lang, barrel, []ui.BoardCell{codexCell(s, lang, water, ""), codexCell(s, lang, item.NewStack(block.Mycelium{}, 1), "", i18n.T(lang, "codex.card.under"))}, []ui.BoardCell{codexCell(s, lang, item.NewStack(sgitem.WitchWaterBucket{}, 1), "")}),
			}, barrelMixCards(s, lang, barrel, false)...)
		}},
		{item.NewStack(farm.Barrel{Wood: farm.StoneWood}, 1), func(s *core.Session, lang string) []Card {
			barrel := item.NewStack(farm.Barrel{Wood: farm.StoneWood}, 1)
			heat := codexCell(s, lang, item.NewStack(block.Torch{}, 1), "", i18n.T(lang, "codex.card.heat"))
			var cards []Card
			for _, m := range farm.StoneMelts {
				cards = append(cards, machineCard(lang, barrel,
					[]ui.BoardCell{codexCell(s, lang, item.NewStack(m, 1), ""), heat},
					[]ui.BoardCell{codexCell(s, lang, lava, "", fmt.Sprintf(i18n.T(lang, "codex.card.mb"), farm.StoneMeltMB))}))
			}
			return append(cards, barrelMixCards(s, lang, barrel, true)...)
		}},
		{machine.Item(machine.KindTinker), func(s *core.Session, lang string) []Card { return codexGearCards(s, lang) }},
	}
}

// codexGearCards are the tinker table's head and handle materials and
// modifiers: the material's slot with what it gives.
func codexGearCards(s *core.Session, lang string) []Card {
	tinkerTable := machine.Item(machine.KindTinker)
	var cards []Card
	gear := func(it world.Item, text string) {
		row := BlankRow()
		row[0] = codexCell(s, lang, item.NewStack(it, 1), "", text)
		row[1] = ui.TextCell(text)
		cards = append(cards, Card{Head: tinkerTable, name: naming.UIItemName(lang, tinkerTable), Rows: [][]ui.BoardCell{row}})
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

// MobCards are the mobs tab: the simulator, the magic sieve's themes
// and the gate's levels.
func MobCards(s *core.Session, lang string) []Card {
	var cards []Card
	for _, mob := range sgitem.MobKinds {
		cards = append(cards, simCard(s, lang, mob, sgitem.SimDrops[mob]))
	}
	for _, mob := range sgitem.MobKinds {
		cards = append(cards, themeCards(s, lang, mob)...)
	}
	for i := range wave.Levels {
		cards = append(cards, waveCard(s, lang, i)...)
	}
	return cards
}

// SieveCards are the drops of every mesh for a block: the mesh, the
// arrow and the drops (the block is the one chosen above them).
func SieveCards(s *core.Session, lang string, in world.Item) []Card {
	var cards []Card
	for _, t := range sgitem.MeshTiers {
		drops := append([]sievetable.Drop(nil), sievetable.Tables[t.Tier][naming.ItemKey(in)]...)
		sort.SliceStable(drops, func(i, j int) bool { return drops[i].Permille > drops[j].Permille })
		var outs []ui.BoardCell
		for _, d := range drops {
			outs = append(outs, chanceCell(s, lang, d.Item, d.Permille))
		}
		cards = append(cards, machineCard(lang, machine.Item(machine.KindSieve),
			[]ui.BoardCell{codexCell(s, lang, item.NewStack(sgitem.Mesh{Tier: t.Tier}, 1), "")}, outs))
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
		if _, ok := machine.SmeltOf(st); ok && !seen[k] {
			seen[k] = true
			out = append(out, st)
		}
	}
	sort.Slice(out, func(i, j int) bool { return naming.ItemKey(out[i].Item()) < naming.ItemKey(out[j].Item()) })
	return out
}

// barrelMixCards are the codex cards of the mixings of a barrel of witch
// water (lava).
func barrelMixCards(s *core.Session, lang string, barrel item.Stack, lava bool) []Card {
	liquid := item.NewStack(sgitem.WitchWaterBucket{}, 1)
	if lava {
		liquid = item.NewStack(farm.LavaBucket, 1)
	}
	var cards []Card
	for _, m := range farm.BarrelMixes() {
		if m.Lava == lava {
			cards = append(cards, machineCard(lang, barrel,
				[]ui.BoardCell{codexCell(s, lang, liquid, ""), codexCell(s, lang, item.NewStack(m.In, 1), "")},
				[]ui.BoardCell{codexCell(s, lang, item.NewStack(m.Out, 1), "")}))
		}
	}
	return cards
}
