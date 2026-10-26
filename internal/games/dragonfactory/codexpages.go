package dragonfactory

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/common/ui"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/advancement"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/formicon"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/naming"
)

// Modes of an item's page, like a recipe viewer's R and U.
const (
	codexSourcesMode = iota
	codexUsesMode
)

// showCodexBoard shows the rows of top, a blank row and under them all the
// cards; the window scrolls when they do not fit.
func (s *session) showCodexBoard(p *player.Player, title string, top [][]ui.BoardCell, cards []codexCard, chrome []ui.Option) {
	var rows [][]ui.BoardCell
	if len(top) > 0 {
		rows = append(append(rows, top...), blankRow())
	}
	rows = append(rows, s.codexBoard(i18n.Of(p), cards)...)
	ui.ItemBoard(p, title, "", rows, chrome...)
}

// showCodexItem shows how an item is made.
func (s *session) showCodexItem(p *player.Player, it item.Stack) {
	s.showCodexItemAt(p, it, codexSourcesMode)
}

// showCodexItemAt shows an item's page: a row with the item, its name and
// the buttons that switch between how it is made and what it is for (and
// make it a goal), then the recipes of that mode as cards.
func (s *session) showCodexItemAt(p *player.Player, it item.Stack, mode int) {
	lang := i18n.Of(p)
	it = item.NewStack(it.Item(), 1)
	s.codexSeen(p, it)
	s.codexAt(p, func(p *player.Player) { s.showCodexItemAt(p, it, mode) })
	head := blankRow()
	head[0] = ui.SlotCell(ui.ItemSlot{Icon: formicon.Of(it), Tip: tipName(lang, it), Selected: true})
	head[1] = ui.TextCell("§l" + naming.ItemName(lang, it))
	modeSlot := func(m int, icon item.Stack, key string) ui.BoardCell {
		sl := ui.ItemSlot{Icon: formicon.Of(icon), Tip: "§f" + i18n.T(lang, key), Selected: m == mode}
		if m != mode {
			sl.Run = func(p *player.Player, _ *world.Tx) { s.showCodexItemAt(p, it, m) }
		}
		return ui.SlotCell(sl)
	}
	head[8] = modeSlot(codexSourcesMode, item.NewStack(block.CraftingTable{}, 1), "codex.item.sources")
	head[9] = modeSlot(codexUsesMode, item.NewStack(block.Chest{}, 1), "codex.item.uses")
	favMark, favKey := "", "codex.fav.add"
	if s.isFavourite(p, it) {
		favMark, favKey = "§e★", "codex.fav.remove"
	}
	head[10] = ui.SlotCell(ui.ItemSlot{Icon: formicon.Of(item.NewStack(item.NetherStar{}, 1)), Tip: "§f" + i18n.T(lang, favKey), Count: favMark,
		Run: func(p *player.Player, tx *world.Tx) {
			if tx.World() == s.world {
				s.toggleFavourite(p, it)
				if s.isFavourite(p, it) {
					s.advance(tx, advancement.Favourite)
				}
			}
			s.showCodexItemAt(p, it, mode)
		}})
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
	s.showCodexBoard(p, ui.SubTitle(i18n.T(lang, "codex.title"), naming.ItemName(lang, it)), [][]ui.BoardCell{head}, cards, chrome)
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
		rows[len(rows)-1][i%ui.BoardColumns] = s.codexCell(lang, l.stack, countLabel(l.count), colour+fmt.Sprintf(i18n.T(lang, "codex.card.have"), have))
	}
	return codexCard{name: i18n.T(lang, "codex.card.raw"), rows: rows}, len(rows) > 0
}

// showCodexSieve is the sieve tab: the first block's drops.
func (s *session) showCodexSieve(p *player.Player) { s.showCodexSieveAt(p, 0) }

// showCodexSieveAt shows a row of the blocks a sieve takes, the chosen one
// lit, and under it the drops of every mesh for that block.
func (s *session) showCodexSieveAt(p *player.Player, which int) {
	lang := i18n.Of(p)
	ins := codexSieveInputs()
	which = max(0, min(which, len(ins)-1))
	s.codexAt(p, func(p *player.Player) { s.showCodexSieveAt(p, which) })
	var top []ui.BoardCell
	for i, in := range ins {
		st := item.NewStack(in, 1)
		sl := ui.ItemSlot{Icon: formicon.Of(st), Tip: tipName(lang, st), Selected: i == which}
		if i != which {
			sl.Run = func(p *player.Player, _ *world.Tx) { s.showCodexSieveAt(p, i) }
		}
		top = append(top, ui.SlotCell(sl))
	}
	cards := s.codexSieveCards(lang, ins[which])
	for i := range cards {
		cards[i].name = ""
	}
	chrome := append(s.codexTabs(p, codexSieveTab), s.codexBack(p))
	if ui.OreUIFor(p) {
		in := item.NewStack(ins[which], 1)
		ui.SieveBoard(p, codexPickTitle(lang, "codex.sieve", in), [][]ui.BoardCell{centredRow(top)},
			s.sieveGroups(lang, in, cards), chrome...)
		return
	}
	s.showCodexBoard(p, codexPickTitle(lang, "codex.sieve", item.NewStack(ins[which], 1)), [][]ui.BoardCell{centredRow(top)},
		cards, chrome)
}

// sieveGroups are the sieve cards of in as recipes (ui.SieveBoard), each
// sieve's under it: a card's first slot is its mesh, the other slots are the
// drops.
func (s *session) sieveGroups(lang string, in item.Stack, cards []codexCard) []ui.SieveGroup {
	var groups []ui.SieveGroup
	lastHead := ""
	for _, c := range cards {
		if k := naming.ItemKey(c.head.Item()); len(groups) == 0 || k != lastHead {
			lastHead = k
			groups = append(groups, ui.SieveGroup{Head: ui.ItemSlot{Icon: formicon.Of(c.head), Tip: tipName(lang, c.head)}, Name: naming.ItemName(lang, c.head)})
		}
		r := ui.SieveRecipe{Input: ui.ItemSlot{Icon: formicon.Of(in), Tip: tipName(lang, in)}}
		first := true
		for _, row := range c.rows {
			for _, cell := range row {
				sl, ok := cell.SlotOf()
				switch {
				case !ok:
				case first:
					r.Mesh, first = sl, false
				default:
					r.Drops = append(r.Drops, sl)
				}
			}
		}
		g := &groups[len(groups)-1]
		g.Recipes = append(g.Recipes, r)
	}
	return groups
}

// codexPickTitle is the title of a tab whose top row picks a block or a
// machine: the tab and the pick, as JEI names the recipe category shown.
func codexPickTitle(lang, tab string, pick item.Stack) string {
	return ui.SubTitle(ui.SubTitle(i18n.T(lang, "codex.title"), i18n.T(lang, tab)), naming.ItemName(lang, pick))
}

// showCodexProcess is the process tab: the first machine's recipes.
func (s *session) showCodexProcess(p *player.Player) { s.showCodexProcessAt(p, 0) }

// showCodexProcessAt shows a row of the machines that process items, the
// chosen one lit, and under it that machine's recipes.
func (s *session) showCodexProcessAt(p *player.Player, which int) {
	lang := i18n.Of(p)
	procs := codexProcesses()
	which = max(0, min(which, len(procs)-1))
	s.codexAt(p, func(p *player.Player) { s.showCodexProcessAt(p, which) })
	var top []ui.BoardCell
	for i, pr := range procs {
		sl := ui.ItemSlot{Icon: formicon.Of(pr.head), Tip: tipName(lang, pr.head), Selected: i == which}
		if i != which {
			sl.Run = func(p *player.Player, _ *world.Tx) { s.showCodexProcessAt(p, i) }
		}
		top = append(top, ui.SlotCell(sl))
	}
	cards := procs[which].cards(s, lang)
	for i := range cards {
		// the machine is the one chosen above: no header
		cards[i].name = ""
	}
	chrome := append(s.codexTabs(p, codexProcessTab), s.codexBack(p))
	s.showCodexBoard(p, codexPickTitle(lang, "codex.process", procs[which].head), [][]ui.BoardCell{centredRow(top)},
		cards, chrome)
}

// showCodexMobs is the mobs tab: the simulator, the magic sieve's themes and
// the gate's levels.
func (s *session) showCodexMobs(p *player.Player) {
	lang := i18n.Of(p)
	s.codexAt(p, func(p *player.Player) { s.showCodexMobs(p) })
	chrome := append(s.codexTabs(p, codexMobsTab), s.codexBack(p))
	s.showCodexBoard(p, ui.SubTitle(i18n.T(lang, "codex.title"), i18n.T(lang, "codex.mobs")), nil,
		s.codexMobCards(lang), chrome)
}

// showCodexPlants is the plants tab: each seed and what it yields.
func (s *session) showCodexPlants(p *player.Player) {
	lang := i18n.Of(p)
	s.codexAt(p, func(p *player.Player) { s.showCodexPlants(p) })
	var cards []codexCard
	for _, pk := range plantKinds {
		cards = append(cards, s.plantCard(lang, pk))
	}
	chrome := append(s.codexTabs(p, codexPlantsTab), s.codexBack(p))
	s.showCodexBoard(p, ui.SubTitle(i18n.T(lang, "codex.title"), i18n.T(lang, "codex.plants")), nil, cards, chrome)
}
