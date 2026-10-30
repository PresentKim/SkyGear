package codex

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/decor"
	"github.com/presentkim/skygear/content/skygear/farm"
	"github.com/presentkim/skygear/content/skygear/formicon"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/progress"
)

// Modes of an item's page, like a recipe viewer's R and U.
const (
	codexSourcesMode = iota
	codexUsesMode
)

// showCodexBoard shows the rows of top, a blank row and under them all the
// cards; the window scrolls when they do not fit.
func showCodexBoard(s *core.Session, p *player.Player, title string, top [][]ui.BoardCell, cards []Card, chrome []ui.Option) {
	var rows [][]ui.BoardCell
	if len(top) > 0 {
		rows = append(append(rows, top...), BlankRow())
	}
	rows = append(rows, Board(s, i18n.Of(p), cards)...)
	ui.ItemBoard(p, title, "", rows, chrome...)
}

// ShowItem shows how an item is made.
func ShowItem(s *core.Session, p *player.Player, it item.Stack) {
	showCodexItemAt(s, p, it, codexSourcesMode)
}

// showCodexItemAt shows an item's page: a row with the item, its name and
// the buttons that switch between how it is made and what it is for (and
// make it a goal), then the recipes of that mode as cards.
func showCodexItemAt(s *core.Session, p *player.Player, it item.Stack, mode int) {
	lang := i18n.Of(p)
	it = item.NewStack(it.Item(), 1)
	codexSeen(s, p, it)
	codexAt(s, p, func(p *player.Player) { showCodexItemAt(s, p, it, mode) })
	head := BlankRow()
	head[0] = ui.SlotCell(ui.ItemSlot{Icon: formicon.Of(it), Tip: decor.TipName(lang, it), Selected: true})
	head[1] = ui.TextCell("§l" + naming.UIItemName(lang, it))
	modeSlot := func(m int, icon item.Stack, key string) ui.BoardCell {
		sl := ui.ItemSlot{Icon: formicon.Of(icon), Tip: "§f" + i18n.T(lang, key), Selected: m == mode}
		if m != mode {
			sl.Run = func(p *player.Player, _ *world.Tx) { showCodexItemAt(s, p, it, m) }
		}
		return ui.SlotCell(sl)
	}
	head[8] = modeSlot(codexSourcesMode, item.NewStack(block.CraftingTable{}, 1), "codex.item.sources")
	head[9] = modeSlot(codexUsesMode, item.NewStack(block.Chest{}, 1), "codex.item.uses")
	favMark, favKey := "", "codex.fav.add"
	if IsFavourite(s, p, it) {
		favMark, favKey = "§e★", "codex.fav.remove"
	}
	head[10] = ui.SlotCell(ui.ItemSlot{Icon: formicon.Of(item.NewStack(item.NetherStar{}, 1)), Tip: "§f" + i18n.T(lang, favKey), Count: favMark,
		Run: func(p *player.Player, tx *world.Tx) {
			if tx.World() == s.World {
				ToggleFavourite(s, p, it)
				if IsFavourite(s, p, it) {
					progress.Advance(s, tx, advancement.Favourite)
				}
			}
			showCodexItemAt(s, p, it, mode)
		}})
	var cards []Card
	if mode == codexUsesMode {
		cards = UseCards(s, lang, it)
	} else {
		cards = SourceCards(s, lang, it)
		if raw, ok := codexRawCard(s, p, lang, it); ok {
			cards = append(cards, raw)
		}
	}
	if len(cards) == 0 {
		row := BlankRow()
		row[0] = ui.TextCell("§7" + i18n.T(lang, "codex.item.nothing"))
		cards = []Card{{name: i18n.T(lang, "codex.item."+map[int]string{codexSourcesMode: "sources", codexUsesMode: "uses"}[mode]), Rows: [][]ui.BoardCell{row}}}
	}
	chrome := append(codexTabs(s, p, codexNoTab), codexBack(s, p))
	showCodexBoard(s, p, ui.SubTitle(i18n.T(lang, "codex.title"), naming.UIItemName(lang, it)), [][]ui.BoardCell{head}, cards, chrome)
}

// codexRawCard is what an item takes in raw materials, down the recipe
// chain: each with how many it takes on the slot and how many p holds in the
// tooltip.
func codexRawCard(s *core.Session, p *player.Player, lang string, it item.Stack) (Card, bool) {
	if _, ok := codexRecipeOf(it); !ok {
		return Card{}, false
	}
	var rows [][]ui.BoardCell
	for i, l := range Raw(it, 1) {
		if i%ui.BoardColumns == 0 {
			rows = append(rows, BlankRow())
		}
		have := core.CountPlain(p.Inventory(), l.Stack.Item())
		colour := "§c"
		if have >= max(1, l.Count) {
			colour = "§a"
		}
		rows[len(rows)-1][i%ui.BoardColumns] = codexCell(s, lang, l.Stack, countLabel(l.Count), colour+fmt.Sprintf(i18n.T(lang, "codex.card.have"), have))
	}
	return Card{name: i18n.T(lang, "codex.card.raw"), Rows: rows}, len(rows) > 0
}

// showCodexSieve is the sieve tab: the first block's drops.
func showCodexSieve(s *core.Session, p *player.Player) { showCodexSieveAt(s, p, 0) }

// showCodexSieveAt shows a row of the blocks a sieve takes, the chosen one
// lit, and under it the drops of every mesh for that block.
func showCodexSieveAt(s *core.Session, p *player.Player, which int) {
	lang := i18n.Of(p)
	ins := SieveInputs()
	which = max(0, min(which, len(ins)-1))
	codexAt(s, p, func(p *player.Player) { showCodexSieveAt(s, p, which) })
	var top []ui.BoardCell
	for i, in := range ins {
		st := item.NewStack(in, 1)
		sl := ui.ItemSlot{Icon: formicon.Of(st), Tip: decor.TipName(lang, st), Selected: i == which}
		if i != which {
			sl.Run = func(p *player.Player, _ *world.Tx) { showCodexSieveAt(s, p, i) }
		}
		top = append(top, ui.SlotCell(sl))
	}
	cards := SieveCards(s, lang, ins[which])
	for i := range cards {
		cards[i].name = ""
	}
	chrome := append(codexTabs(s, p, codexSieveTab), codexBack(s, p))
	in := item.NewStack(ins[which], 1)
	ui.SieveBoard(p, codexPickTitle(lang, "codex.sieve", in), [][]ui.BoardCell{centredRow(top)},
		sieveGroups(s, lang, in, cards), chrome...)
}

// sieveGroups are the sieve cards of in as recipes (ui.SieveBoard), each
// sieve's under it: a card's first slot is its mesh, the other slots are the
// drops.
func sieveGroups(s *core.Session, lang string, in item.Stack, cards []Card) []ui.SieveGroup {
	var groups []ui.SieveGroup
	lastHead := ""
	for _, c := range cards {
		if k := naming.ItemKey(c.Head.Item()); len(groups) == 0 || k != lastHead {
			lastHead = k
			groups = append(groups, ui.SieveGroup{Head: ui.ItemSlot{Icon: formicon.Of(c.Head), Tip: decor.TipName(lang, c.Head)}, Name: naming.UIItemName(lang, c.Head)})
		}
		r := ui.SieveRecipe{Input: ui.ItemSlot{Icon: formicon.Of(in), Tip: decor.TipName(lang, in)}}
		first := true
		for _, row := range c.Rows {
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
	return ui.SubTitle(ui.SubTitle(i18n.T(lang, "codex.title"), i18n.T(lang, tab)), naming.UIItemName(lang, pick))
}

// showCodexProcess is the process tab: the first machine's recipes.
func showCodexProcess(s *core.Session, p *player.Player) { showCodexProcessAt(s, p, 0) }

// showCodexProcessAt shows a row of the machines that process items, the
// chosen one lit, and under it that machine's recipes.
func showCodexProcessAt(s *core.Session, p *player.Player, which int) {
	lang := i18n.Of(p)
	procs := Processes()
	which = max(0, min(which, len(procs)-1))
	codexAt(s, p, func(p *player.Player) { showCodexProcessAt(s, p, which) })
	var top []ui.BoardCell
	for i, pr := range procs {
		sl := ui.ItemSlot{Icon: formicon.Of(pr.Head), Tip: decor.TipName(lang, pr.Head), Selected: i == which}
		if i != which {
			sl.Run = func(p *player.Player, _ *world.Tx) { showCodexProcessAt(s, p, i) }
		}
		top = append(top, ui.SlotCell(sl))
	}
	cards := procs[which].Cards(s, lang)
	for i := range cards {
		// the machine is the one chosen above: no header
		cards[i].name = ""
	}
	chrome := append(codexTabs(s, p, codexProcessTab), codexBack(s, p))
	showCodexBoard(s, p, codexPickTitle(lang, "codex.process", procs[which].Head), [][]ui.BoardCell{centredRow(top)},
		cards, chrome)
}

// showCodexMobs is the mobs tab: the simulator, the magic sieve's themes and
// the gate's levels.
func showCodexMobs(s *core.Session, p *player.Player) {
	lang := i18n.Of(p)
	codexAt(s, p, func(p *player.Player) { showCodexMobs(s, p) })
	chrome := append(codexTabs(s, p, codexMobsTab), codexBack(s, p))
	showCodexBoard(s, p, ui.SubTitle(i18n.T(lang, "codex.title"), i18n.T(lang, "codex.mobs")), nil,
		MobCards(s, lang), chrome)
}

// showCodexPlants is the plants tab: each seed and what it yields.
func showCodexPlants(s *core.Session, p *player.Player) {
	lang := i18n.Of(p)
	codexAt(s, p, func(p *player.Player) { showCodexPlants(s, p) })
	var cards []Card
	for _, pk := range farm.PlantKinds {
		cards = append(cards, plantCard(s, lang, pk))
	}
	chrome := append(codexTabs(s, p, codexPlantsTab), codexBack(s, p))
	showCodexBoard(s, p, ui.SubTitle(i18n.T(lang, "codex.title"), i18n.T(lang, "codex.plants")), nil, cards, chrome)
}
