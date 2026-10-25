package dragonfactory

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/common/ui"
)

// The advancement screen (docs/design/02-dragonfactory.md §10,
// docs/tech/advancement-tree.md): category tabs over the whole tree of the tab,
// laid out left to right by the server (a column per depth, a row per leaf). A
// node is done, open (its parent is done) or the next locked one behind an
// open node; the rest is not shown. A node's name and text are its tooltip;
// pressing it opens its detail.

// advTab is a category of the advancement screen.
type advTab int

const (
	advTabBasic advTab = iota
	advTabResource
	advTabFactory
	advTabCombat
	advTabGarden
	advTabCount
)

// advTabKeys name the tabs in the catalog ("advancement.tab.<key>").
var advTabKeys = [advTabCount]string{"basic", "resource", "factory", "combat", "garden"}

// advTabIcons are items whose icon marks each tab.
var advTabIcons = [advTabCount]string{"crafting_table", "dragonfactory:sieve", "dragonfactory:generator", "iron_sword", "oak_sapling"}

// advTabRoots start a category; everything below one that is not itself a
// root belongs to the same category as its parent, the rest to the basic one.
var advTabRoots = map[string]advTab{
	advSift: advTabResource, "iron": advTabResource, "ore_dust": advTabResource,
	"copper": advTabResource, "gold": advTabResource, "mesh_diamond": advTabResource,
	"clay": advTabResource, "dust": advTabResource, "gravel": advTabResource,
	advMachine: advTabFactory, "generator": advTabFactory, "hopper": advTabFactory,
	"drawer": advTabFactory, "fluid_conduit": advTabFactory, "energy_cell": advTabFactory,
	"conduit_upgrade_2": advTabFactory, "conduit_upgrade_3": advTabFactory,
	"tinker_table": advTabFactory, "project_industry": advTabFactory,
	"essence": advTabCombat, "dimension_gate": advTabCombat,
	"seeds": advTabGarden, "magic_seed": advTabGarden, "dye": advTabGarden,
}

// advTabs are the categories of advancementTree by index, and advTabLists
// the advancements of each category in tree order.
var advTabs, advTabLists = func() ([]advTab, [advTabCount][]*Advancement) {
	tabs := make([]advTab, len(advancementTree))
	var lists [advTabCount][]*Advancement
	for i := range advancementTree {
		a := &advancementTree[i]
		tabs[i] = advTabFind(a)
		lists[tabs[i]] = append(lists[tabs[i]], a)
	}
	return tabs, lists
}()

// advTabOf is the category of an advancement.
func advTabOf(a *Advancement) advTab {
	if i, ok := advancementIndex[a.ID]; ok && &advancementTree[i] == a {
		return advTabs[i]
	}
	return advTabFind(a)
}

// advTabFind works out the category of an advancement from its parents.
func advTabFind(a *Advancement) advTab {
	for cur, ok := a, true; ok; cur, ok = advancementByID(cur.Parent) {
		if t, found := advTabRoots[cur.ID]; found {
			return t
		}
	}
	return advTabBasic
}

// advView is what the advancement screen shows: the tab.
type advView struct{ tab advTab }

// advPos is a cell of a tab's tree grid.
type advPos struct{ x, y int }

// advLayout is the placed tree of a tab: where each shown node is, the
// connector lines of every cell's strip, and the size.
type advLayout struct {
	pos    map[string]advPos
	strips map[advPos]int
	w, h   int
	order  []*Advancement
}

// advShown reports whether a node is shown: done, open, or locked right
// behind an open one.
func (s *session) advShown(a *Advancement) bool {
	if s.advanced(a.ID) || s.advancementOpen(a) {
		return true
	}
	p, ok := advancementByID(a.Parent)
	return ok && s.advancementOpen(p)
}

// advancementLayout places the shown nodes of a tab: a node's column is its
// depth below the root of the tab, leaves take the rows one after the other
// and a parent sits beside the middle of its children (the tidy tree of a
// family tree). Connectors run in the left strip of a child's column: the
// parent's row has a line to the strip, the bus between the rows is vertical
// and the child's row has a line into the node.
func (s *session) advancementLayout(tab advTab) advLayout {
	l := advLayout{pos: map[string]advPos{}, strips: map[advPos]int{}}
	kids := map[string][]*Advancement{}
	var roots []*Advancement
	for _, a := range advancementsIn(tab) {
		if !s.advShown(a) {
			continue
		}
		l.order = append(l.order, a)
		if par, ok := advancementByID(a.Parent); ok && advTabOf(par) == tab {
			kids[par.ID] = append(kids[par.ID], a)
		} else {
			roots = append(roots, a)
		}
	}
	row := 0
	var place func(a *Advancement, depth int) int
	place = func(a *Advancement, depth int) int {
		var y int
		if cs := kids[a.ID]; len(cs) == 0 {
			y = row
			row++
		} else {
			first, last := 0, 0
			for i, c := range cs {
				cy := place(c, depth+1)
				if i == 0 {
					first = cy
				}
				last = cy
			}
			y = (first + last) / 2
		}
		l.pos[a.ID] = advPos{depth, y}
		l.w, l.h = max(l.w, depth+1), max(l.h, y+1)
		return y
	}
	for _, r := range roots {
		place(r, 0)
	}
	for _, a := range l.order {
		par, ok := advancementByID(a.Parent)
		if !ok || advTabOf(par) != advTab(tab) {
			continue
		}
		c, p := l.pos[a.ID], l.pos[par.ID]
		l.strips[advPos{c.x, p.y}] |= ui.StripLeft
		l.strips[c] |= ui.StripRight
		switch {
		case c.y > p.y:
			l.strips[advPos{c.x, p.y}] |= ui.StripDown
			l.strips[c] |= ui.StripUp
		case c.y < p.y:
			l.strips[advPos{c.x, p.y}] |= ui.StripUp
			l.strips[c] |= ui.StripDown
		}
		for y := min(c.y, p.y) + 1; y < max(c.y, p.y); y++ {
			l.strips[advPos{c.x, y}] |= ui.StripUp | ui.StripDown
		}
	}
	return l
}

// advChallenges are the big milestones, drawn with the challenge frame; the
// advancements of a research get the goal frame, the rest the task frame.
var advChallenges = map[string]bool{
	advDragon: true, advWave8: true, "chunks_16": true,
	advTool5: true, "project_dragon": true, "ess_harvest": true, "gem_harvest": true,
}

// advFrame is the frame letter of a node.
func advFrame(a *Advancement) byte {
	switch {
	case advChallenges[a.ID]:
		return ui.FrameChallenge
	case a.Research != "":
		return ui.FrameGoal
	}
	return ui.FrameTask
}

// showAdvancements opens the advancement screen on the tab of the advancement
// the quest panel follows.
func (s *session) showAdvancements(p *player.Player) {
	count(p, "adv_view", 1)
	var v advView
	if a, ok := s.nextAdvancement(); ok {
		v.tab = advTabOf(a)
	}
	s.showAdvancementView(p, v)
}

// advancementTree builds the form's tree of a view: every row of the tab's
// layout.
func (s *session) advancementTree(p *player.Player, v advView) ui.Tree {
	lang := i18n.Of(p)
	l := s.advancementLayout(v.tab)
	var t ui.Tree
	byPos := map[advPos]*Advancement{}
	for _, a := range l.order {
		byPos[l.pos[a.ID]] = a
	}
	for y := range min(l.h, ui.TreeRows) {
		for x := range ui.TreeCols {
			ps := advPos{x, y}
			c := ui.TreeCell{Strip: l.strips[ps]}
			if a := byPos[ps]; a != nil {
				c.Frame = advFrame(a)
				switch {
				case s.advanced(a.ID):
					c.State, c.Icon, c.Tip = ui.StateDone, advancementIcon(a), "§a"+a.Title(lang)+"\n§7"+flatHint(lang, a)
				case s.advancementOpen(a):
					c.State, c.Icon, c.Tip = ui.StateOpen, advancementIcon(a), "§e"+a.Title(lang)+"\n§7"+flatHint(lang, a)
				default:
					c.State, c.Icon, c.Tip = ui.StateLocked, iconLocked, "§8???"
				}
				c.Run = func(p *player.Player, tx *world.Tx) {
					if tx.World() == s.world {
						s.showAdvancementNode(p, a)
					}
				}
			}
			t.Cells = append(t.Cells, c)
		}
	}
	for tab := advTab(0); tab < advTabCount; tab++ {
		o := ui.Tab(i18n.T(lang, "advancement.tab."+advTabKeys[tab]), advTabIcon(tab), tab == v.tab, nil)
		t.Tabs = append(t.Tabs, s.opt(o.Label, o.Icon, func(p *player.Player, _ *world.Tx) {
			s.showAdvancementView(p, advView{tab: tab})
		}))
	}
	done := 0
	list := advancementsIn(v.tab)
	for _, a := range list {
		if s.advanced(a.ID) {
			done++
		}
	}
	t.Body = fmt.Sprintf(i18n.T(lang, "advancement.progress"), i18n.T(lang, "advancement.tab."+advTabKeys[v.tab]), done, len(list))
	t.Actions = []ui.Option{s.backOption()}
	return t
}

// showAdvancementView shows the advancement screen on a view.
func (s *session) showAdvancementView(p *player.Player, v advView) {
	if ui.OreUIFor(p) {
		s.showQuestBook(p, v.tab, nil)
		return
	}
	t := s.advancementTree(p, v)
	ui.TreeMenu(p, i18n.T(i18n.Of(p), "dragonfactory.advancement.showAdvancements.1"), t)
}

// showAdvancementNode shows one advancement with its buttons (track, codex,
// back to its tab); a hidden one shows only what comes before it.
func (s *session) showAdvancementNode(p *player.Player, a *Advancement) {
	lang := i18n.Of(p)
	title, body := a.Title(lang), ""
	switch {
	case s.advanced(a.ID):
		body = i18n.T(lang, "dragonfactory.advancement.showAdvancementNode.1") + flatHint(lang, a)
	case s.advancementOpen(a):
		body = i18n.T(lang, "dragonfactory.advancement.showAdvancementNode.2") + flatHint(lang, a)
	default:
		title, body = "???", i18n.T(lang, "dragonfactory.advancement.showAdvancementNode.3")
	}
	var opts []ui.Option
	if s.advancementOpen(a) {
		label := i18n.T(lang, "advancement.node.track")
		if s.state.Tracked == a.ID {
			label = i18n.T(lang, "advancement.node.untrack")
		}
		opts = append(opts, s.opt(label, iconMission, func(p *player.Player, tx *world.Tx) {
			if s.state.Tracked != a.ID {
				count(p, "adv_track", 1)
			}
			s.track(tx, a)
			s.showAdvancementNode(p, a)
		}))
	}
	if it, ok := advancementStack(a); ok && (s.advanced(a.ID) || s.advancementOpen(a)) {
		opts = append(opts, s.opt(i18n.T(lang, "advancement.node.codex"), iconOf(it), func(p *player.Player, _ *world.Tx) {
			s.codexFrom(p, func(p *player.Player) { s.showAdvancementNode(p, a) }, it)
		}))
	}
	back := ui.Back(i18n.T(lang, "codex.back"), func(p *player.Player, _ *world.Tx) { s.showAdvancementView(p, advView{tab: advTabOf(a)}) })
	opts = append(opts, s.opt(back.Label, back.Icon, back.Run))
	ui.Menu(p, i18n.T(lang, "dragonfactory.advancement.showAdvancementNode.4")+title, body, opts...)
}

// advShapes are the quest shapes of the frames in the quest book.
var advShapes = map[byte]string{ui.FrameTask: ui.ShapeDiamond, ui.FrameGoal: ui.ShapeSquare, ui.FrameChallenge: ui.ShapeCircle}

// showQuestBook shows the advancement screen as a quest book (DDUI): the
// categories as chapters, the tab's tree on a plane (the same places as the
// tree form) and, when open is set, its quest panel over it.
func (s *session) showQuestBook(p *player.Player, tab advTab, open *Advancement) {
	lang := i18n.Of(p)
	b := ui.QuestBook{Key: "adv_" + advTabKeys[tab]}
	for t := advTab(0); t < advTabCount; t++ {
		done, list := 0, advancementsIn(t)
		for _, a := range list {
			if s.advanced(a.ID) {
				done++
			}
		}
		b.Chapters = append(b.Chapters, ui.QuestChapter{
			Name: i18n.T(lang, "advancement.tab."+advTabKeys[t]), Icon: advTabIcon(t), Count: fmt.Sprintf("%d/%d", done, len(list)), On: t == tab,
			Run: func(p *player.Player, tx *world.Tx) {
				if tx.World() == s.world {
					s.showQuestBook(p, t, nil)
				}
			},
		})
		if t == tab {
			b.Heading = fmt.Sprintf(i18n.T(lang, "advancement.progress"), i18n.T(lang, "advancement.tab."+advTabKeys[t]), done, len(list))
		}
	}
	l := s.advancementLayout(tab)
	if next, ok := s.nextAdvancement(); ok && advTabOf(next) == tab {
		b.Focus = next.ID
	} else if len(l.order) > 0 {
		b.Focus = l.order[0].ID
	}
	for _, a := range l.order {
		pos := l.pos[a.ID]
		q := ui.QuestNode{ID: a.ID, X: float64(pos.x), Y: float64(pos.y), Shape: advShapes[advFrame(a)], Tracked: s.state.Tracked == a.ID}
		switch {
		case s.advanced(a.ID):
			q.State, q.Icon, q.Tip = ui.QuestDone, advancementIcon(a), "§a"+a.Title(lang)+"\n§7"+flatHint(lang, a)
		case s.advancementOpen(a):
			q.State, q.Icon, q.Tip = ui.QuestOpen, advancementIcon(a), "§e"+a.Title(lang)+"\n§7"+flatHint(lang, a)
		default:
			q.State, q.Icon, q.Tip = ui.QuestLocked, iconLocked, "§8???"
		}
		if par, ok := advancementByID(a.Parent); ok && advTabOf(par) == tab {
			q.Deps = []string{par.ID}
		}
		q.Run = func(p *player.Player, tx *world.Tx) {
			if tx.World() == s.world {
				s.showQuestBook(p, tab, a)
			}
		}
		b.Nodes = append(b.Nodes, q)
	}
	if open != nil {
		b.Panel = s.questPanel(p, tab, open)
	}
	ui.QuestBookMenu(p, i18n.T(lang, "dragonfactory.advancement.showAdvancements.1"), b, s.backOption())
}

// questPanel is an advancement's quest panel: its state and hint, the item
// it asks for (pressing it opens the codex) and the track and codex buttons,
// like showAdvancementNode.
func (s *session) questPanel(p *player.Player, tab advTab, a *Advancement) *ui.QuestPanel {
	lang := i18n.Of(p)
	back := func(p *player.Player, tx *world.Tx) {
		if tx.World() == s.world {
			s.showQuestBook(p, tab, nil)
		}
	}
	again := func(p *player.Player) { s.showQuestBook(p, tab, a) }
	pn := &ui.QuestPanel{Icon: advancementIcon(a), Title: a.Title(lang), Close: back}
	switch {
	case s.advanced(a.ID):
		pn.Status = "§2" + i18n.T(lang, "dragonfactory.advancement.showAdvancementNode.1")
		pn.Body = flatHint(lang, a)
	case s.advancementOpen(a):
		pn.Status = "§1" + i18n.T(lang, "dragonfactory.advancement.showAdvancementNode.2")
		pn.Body = flatHint(lang, a)
	default:
		pn.Icon, pn.Title, pn.Body = iconLocked, "???", i18n.T(lang, "dragonfactory.advancement.showAdvancementNode.3")
		return pn
	}
	if it, ok := advancementStack(a); ok {
		pn.Rows = []ui.SlotRow{{Slots: []ui.ItemSlot{{
			Icon: iconOf(it), Tip: tipName(lang, it),
			Run: func(p *player.Player, _ *world.Tx) { s.codexFrom(p, again, it) },
		}}}}
	}
	if s.advancementOpen(a) {
		label := i18n.T(lang, "advancement.node.track")
		if s.state.Tracked == a.ID {
			label = i18n.T(lang, "advancement.node.untrack")
		}
		pn.Actions = append(pn.Actions, s.opt(label, iconMission, func(p *player.Player, tx *world.Tx) {
			if s.state.Tracked != a.ID {
				count(p, "adv_track", 1)
			}
			s.track(tx, a)
			again(p)
		}))
	}
	return pn
}

// advancementsIn lists a category in tree order. The list is shared: callers
// only read it.
func advancementsIn(tab advTab) []*Advancement {
	return advTabLists[tab]
}

// advTabIcon is the icon of a tab.
func advTabIcon(t advTab) string {
	if it, ok := world.ItemByName(unlockName(advTabIcons[t]), 0); ok {
		return iconOf(item.NewStack(it, 1))
	}
	return iconMission
}
