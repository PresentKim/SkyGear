package dragonfactory

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/common/ui"
)

// codexBoardRows is how many rows of cells fit the codex's window without
// scrolling.
const codexBoardRows = 7

// Modes of an item's page, like a recipe viewer's R and U.
const (
	codexSourcesMode = iota
	codexUsesMode
)

// showCodexBoard shows the rows of top and under them a page of the cards,
// with the page arrows in the control bar. again shows another page.
func (s *session) showCodexBoard(p *player.Player, title string, top [][]ui.BoardCell, cards []codexCard, page int, again func(p *player.Player, page int), chrome []ui.Option) {
	pages := s.codexBoardPages(cards, codexBoardRows-len(top))
	page = max(0, min(page, len(pages)-1))
	rows := append(append([][]ui.BoardCell(nil), top...), pages[page]...)
	pager := s.guard(ui.BarPager(p, page, len(pages), func(p *player.Player, _ *world.Tx, to int) { again(p, to) }))
	ui.ItemBoard(p, title, "", rows, append(pager, chrome...)...)
}

// showCodexItem shows how an item is made.
func (s *session) showCodexItem(p *player.Player, it item.Stack) {
	s.showCodexItemAt(p, it, codexSourcesMode, 0)
}

// showCodexItemAt shows an item's page: a row with the item, its name and
// the buttons that switch between how it is made and what it is for (and
// make it a goal), then the recipes of that mode as cards.
func (s *session) showCodexItemAt(p *player.Player, it item.Stack, mode, page int) {
	lang := i18n.Of(p)
	it = item.NewStack(it.Item(), 1)
	s.codexSeen(p, it)
	s.codexAt(p, func(p *player.Player) { s.showCodexItemAt(p, it, mode, page) })
	head := blankRow()
	head[0] = ui.SlotCell(ui.ItemSlot{Icon: iconOf(it), Tip: tipName(it), Selected: true})
	head[1] = ui.TextCell("§l" + itemName(it))
	modeSlot := func(m int, icon item.Stack, key string) ui.BoardCell {
		sl := ui.ItemSlot{Icon: iconOf(icon), Tip: "§f" + i18n.T(lang, key), Selected: m == mode}
		if m != mode {
			sl.Run = func(p *player.Player, _ *world.Tx) { s.showCodexItemAt(p, it, m, 0) }
		}
		return ui.SlotCell(sl)
	}
	head[8] = modeSlot(codexSourcesMode, item.NewStack(block.CraftingTable{}, 1), "codex.item.sources")
	head[9] = modeSlot(codexUsesMode, item.NewStack(block.Chest{}, 1), "codex.item.uses")
	if _, ok := codexRecipeOf(it); ok {
		goalMark := ""
		key := "goal.set"
		if s.isGoal(it) {
			key = "goal.unset"
			goalMark = "§e★"
		}
		head[10] = ui.SlotCell(ui.ItemSlot{Icon: iconOf(item.NewStack(item.BookAndQuill{}, 1)), Tip: "§f" + i18n.T(lang, key), Count: goalMark,
			Run: func(p *player.Player, tx *world.Tx) {
				if tx.World() == s.world {
					s.toggleGoal(tx, it)
				}
				s.showCodexItemAt(p, it, mode, page)
			}})
	}
	var cards []codexCard
	if mode == codexUsesMode {
		cards = s.codexUseCards(lang, it)
	} else {
		cards = s.codexSourceCards(lang, it)
		if raw, ok := s.codexRawCard(p, lang, it); ok {
			cards = append(cards, raw)
		}
	}
	if len(cards) == 0 {
		row := blankRow()
		row[0] = ui.TextCell("§7" + i18n.T(lang, "codex.item.nothing"))
		cards = []codexCard{{name: i18n.T(lang, "codex.item."+map[int]string{codexSourcesMode: "sources", codexUsesMode: "uses"}[mode]), rows: [][]ui.BoardCell{row}}}
	}
	chrome := append(s.codexTabs(p, codexNoTab), s.codexBack(p))
	s.showCodexBoard(p, ui.SubTitle(i18n.T(lang, "codex.title"), itemName(it)), [][]ui.BoardCell{head}, cards, page,
		func(p *player.Player, page int) { s.showCodexItemAt(p, it, mode, page) }, chrome)
}

// codexRawCard is what an item takes in raw materials, down the recipe
// chain: each with how many it takes on the slot and how many p holds in the
// tooltip.
func (s *session) codexRawCard(p *player.Player, lang string, it item.Stack) (codexCard, bool) {
	if _, ok := codexRecipeOf(it); !ok {
		return codexCard{}, false
	}
	var rows [][]ui.BoardCell
	for i, l := range codexRaw(it, 1) {
		if i%ui.BoardColumns == 0 {
			rows = append(rows, blankRow())
		}
		have := countPlain(p.Inventory(), l.stack.Item())
		colour := "§c"
		if have >= max(1, l.count) {
			colour = "§a"
		}
		rows[len(rows)-1][i%ui.BoardColumns] = s.codexCell(l.stack, countLabel(l.count), colour+fmt.Sprintf(i18n.T(lang, "codex.card.have"), have))
	}
	return codexCard{name: i18n.T(lang, "codex.card.raw"), rows: rows}, len(rows) > 0
}

// showCodexSieve is the sieve tab: the first block's drops.
func (s *session) showCodexSieve(p *player.Player) { s.showCodexSieveAt(p, 0, 0) }

// showCodexSieveAt shows a row of the blocks a sieve takes, the chosen one
// lit, and under it the drops of every mesh for that block.
func (s *session) showCodexSieveAt(p *player.Player, which, page int) {
	lang := i18n.Of(p)
	ins := codexSieveInputs()
	which = max(0, min(which, len(ins)-1))
	s.codexAt(p, func(p *player.Player) { s.showCodexSieveAt(p, which, page) })
	top := blankRow()
	for i, in := range ins {
		st := item.NewStack(in, 1)
		sl := ui.ItemSlot{Icon: iconOf(st), Tip: tipName(st), Selected: i == which}
		if i != which {
			sl.Run = func(p *player.Player, _ *world.Tx) { s.showCodexSieveAt(p, i, 0) }
		}
		top[i] = ui.SlotCell(sl)
	}
	chrome := append(s.codexTabs(p, codexSieveTab), s.codexBack(p))
	s.showCodexBoard(p, ui.SubTitle(i18n.T(lang, "codex.title"), i18n.T(lang, "codex.sieve")), [][]ui.BoardCell{top},
		s.codexSieveCards(lang, ins[which]), page, func(p *player.Player, page int) { s.showCodexSieveAt(p, which, page) }, chrome)
}

// showCodexProcess is the process tab: the crusher, washer, alloys, lab and
// the tinker table's materials.
func (s *session) showCodexProcess(p *player.Player, page int) {
	lang := i18n.Of(p)
	s.codexAt(p, func(p *player.Player) { s.showCodexProcess(p, page) })
	chrome := append(s.codexTabs(p, codexProcessTab), s.codexBack(p))
	s.showCodexBoard(p, ui.SubTitle(i18n.T(lang, "codex.title"), i18n.T(lang, "codex.process")), nil,
		s.codexProcessCards(lang), page, s.showCodexProcess, chrome)
}

// showCodexMobs is the mobs tab: the simulator, the magic sieve's themes and
// the gate's levels.
func (s *session) showCodexMobs(p *player.Player, page int) {
	lang := i18n.Of(p)
	s.codexAt(p, func(p *player.Player) { s.showCodexMobs(p, page) })
	chrome := append(s.codexTabs(p, codexMobsTab), s.codexBack(p))
	s.showCodexBoard(p, ui.SubTitle(i18n.T(lang, "codex.title"), i18n.T(lang, "codex.mobs")), nil,
		s.codexMobCards(lang), page, s.showCodexMobs, chrome)
}

// showCodexPlants is the plants tab: each seed and what it yields.
func (s *session) showCodexPlants(p *player.Player, page int) {
	lang := i18n.Of(p)
	s.codexAt(p, func(p *player.Player) { s.showCodexPlants(p, page) })
	var cards []codexCard
	for _, pk := range plantKinds {
		cards = append(cards, s.plantCard(pk))
	}
	chrome := append(s.codexTabs(p, codexPlantsTab), s.codexBack(p))
	s.showCodexBoard(p, ui.SubTitle(i18n.T(lang, "codex.title"), i18n.T(lang, "codex.plants")), nil, cards, page, s.showCodexPlants, chrome)
}
