package dragonfactory

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/common/ui"
)

// The advancement screen (docs/design/02-dragonfactory.md §10,
// docs/tech/advancement-tree.md): chapters over the whole tree of the
// chapter, laid out left to right by the server (layoutChapter). Every node
// is shown: done, open (its parents are done) or locked (drawn darker). A
// parent in another chapter is an entrance at the left edge that leads
// there. A node's name and text are its tooltip; pressing it opens its
// detail.

// advChapter is a chapter of the advancement screen: the advancements
// grouped by how their things are got (by hand, the sieve, farming and
// barrels, machines, alloys and fluids, hunting, gear, magic), then the
// projects and the features beside the flow. A parent in another chapter
// is an entrance, so an advancement stays with the way its thing is got.
type advChapter int

const (
	advChapterStart advChapter = iota
	advChapterSieve
	advChapterFarming
	advChapterAutomation
	advChapterLogistics
	advChapterLab
	advChapterAlloy
	advChapterHunting
	advChapterGear
	advChapterMagic
	advChapterProject
	advChapterExtra
	advChapterCount
)

// advChapterKeys name the chapters in the catalog ("advancement.tab.<key>").
var advChapterKeys = [advChapterCount]string{"start", "sieve", "farming", "automation", "logistics", "lab", "alloy", "hunting", "gear", "magic", "project", "extra"}

// advChapterIcons are items whose icon marks each chapter.
var advChapterIcons = [advChapterCount]string{
	"crafting_table", "dragonfactory:sieve", "wheat", "dragonfactory:generator", "hopper", "dragonfactory:laboratory", "dragonfactory:alloy_smelter",
	"dragonfactory:mob_simulator", "iron_pickaxe", "dragonfactory:petal_red", "dragon_egg", "lantern",
}

// advChapterLists are the advancements of each chapter in tree order.
var advChapterLists = func() [advChapterCount][]*Advancement {
	var lists [advChapterCount][]*Advancement
	for i := range advancementTree {
		a := &advancementTree[i]
		lists[a.Chapter] = append(lists[a.Chapter], a)
	}
	return lists
}()

// advView is what the advancement screen shows: the tab.
type advView struct{ tab advChapter }

// advPos is a cell of a chapter's tree grid.
type advPos struct{ x, y int }

// advLayout is the placed tree of a chapter: where each node and each
// entrance is, the parent each node's connector comes from in the tree form,
// the connector lines of every cell's strip, and the size. order lists the
// nodes in tree order and gates the entrances top to bottom.
type advLayout struct {
	pos    map[string]advPos
	line   map[string]string
	strips map[advPos]int
	w, h   int
	order  []*Advancement
	gates  []*Advancement
	// vias are the places a line passes in the columns between a parent and
	// a node further than the next column, by advViaKey: kept free, so the
	// quest book's line goes around the nodes between.
	vias map[string][]string
}

// advGateID is the place key of the entrance of an advancement of another
// chapter.
func advGateID(id string) string { return "gate:" + id }

// advLayouts are the layouts of the chapters: every node is shown, so they
// do not change.
var advLayouts = func() (out [advChapterCount]advLayout) {
	for ch := range advChapterCount {
		out[ch] = layoutChapter(ch)
	}
	return out
}()

// advancementLayout is the layout of a chapter.
func advancementLayout(ch advChapter) advLayout { return advLayouts[ch] }

// layoutChapter places the nodes of a chapter (a layered drawing): a node's
// column is one more than its latest parent's, so it sits right of every
// parent; a parent in another chapter is an entrance right before its first
// child.
// Each group of nodes joined by lines is drawn apart (layoutGroup); the
// groups go one under another in tree order, and one that would pass the
// bottom of the tree form's grid starts a new band right of the others.
// Connectors of the tree form run in the left strip of a child's column
// from the parent in the column before it: the parent's row has a line to
// the strip, the bus between the rows is vertical and the child's row has a
// line into the node.
func layoutChapter(ch advChapter) advLayout {
	l := advLayout{pos: map[string]advPos{}, line: map[string]string{}, strips: map[advPos]int{}, vias: map[string][]string{}}
	l.order = advancementsIn(ch)
	g := advGraph{col: map[string]int{}, parents: map[string][]string{}, children: map[string][]string{}}
	var keys []string
	gateOf := map[string]bool{}
	for _, a := range l.order {
		c := 0
		for _, id := range a.Parents {
			par, ok := advancementByID(id)
			if !ok {
				continue
			}
			key := id
			if par.Chapter != ch {
				key = advGateID(id)
				if !gateOf[key] {
					gateOf[key] = true
					l.gates = append(l.gates, par)
					keys = append(keys, key)
				}
			}
			g.parents[a.ID] = append(g.parents[a.ID], key)
			g.children[key] = append(g.children[key], a.ID)
			c = max(c, g.col[key]+1)
		}
		g.col[a.ID] = c
		keys = append(keys, a.ID)
	}
	// An entrance stands right before its first child, so its lines are
	// short.
	for _, gt := range l.gates {
		key := advGateID(gt.ID)
		c := -1
		for _, k := range g.children[key] {
			if c < 0 || g.col[k]-1 < c {
				c = g.col[k] - 1
			}
		}
		g.col[key] = c
	}
	real := g.parents
	g, keys = g.withVias(keys, l.vias)
	group := map[string]string{}
	var find func(k string) string
	find = func(k string) string {
		if r, ok := group[k]; ok && r != k {
			group[k] = find(r)
			return group[k]
		}
		return k
	}
	for _, k := range keys {
		for _, p := range g.parents[k] {
			if a, b := find(k), find(p); a != b {
				group[a] = b
			}
		}
	}
	var roots []string
	members := map[string][]string{}
	for _, k := range keys {
		r := find(k)
		if _, ok := members[r]; !ok {
			roots = append(roots, r)
		}
		members[r] = append(members[r], k)
	}
	x0, y0, band := 0, 0, 0
	for _, r := range roots {
		pos, w, h := g.layoutGroup(members[r])
		if y0 > 0 && y0+h > ui.TreeRows {
			x0, y0, band = x0+band, 0, 0
		}
		for k, p := range pos {
			l.pos[k] = advPos{x0 + p.x, y0 + p.y}
		}
		band = max(band, w)
		l.w, l.h = max(l.w, x0+w), max(l.h, y0+h)
		y0 += h
	}
	for _, a := range l.order {
		c := l.pos[a.ID]
		var from string
		for _, k := range real[a.ID] {
			if l.pos[k].x == c.x-1 && (from == "" || abs(l.pos[k].y-c.y) < abs(l.pos[from].y-c.y)) {
				from = k
			}
		}
		if from == "" {
			continue
		}
		l.line[a.ID] = from
		p := l.pos[from]
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

// withVias is the graph with a via point in every column a line passes
// between a parent and a node further than the next column, the keys in tree
// order with the vias of a node's lines before it; vias gets the vias of
// each line (advViaKey).
func (g advGraph) withVias(keys []string, vias map[string][]string) (advGraph, []string) {
	e := advGraph{col: map[string]int{}, parents: map[string][]string{}, children: map[string][]string{}}
	var out []string
	for _, k := range keys {
		e.col[k] = g.col[k]
		for _, p := range g.parents[k] {
			from := p
			for c := g.col[p] + 1; c < g.col[k]; c++ {
				v := fmt.Sprintf("via:%s:%d", advViaKey(p, k), c)
				e.col[v] = c
				e.parents[v] = []string{from}
				e.children[from] = append(e.children[from], v)
				vias[advViaKey(p, k)] = append(vias[advViaKey(p, k)], v)
				out = append(out, v)
				from = v
			}
			e.parents[k] = append(e.parents[k], from)
			e.children[from] = append(e.children[from], k)
		}
		out = append(out, k)
	}
	return e, out
}

// advViaKey names the line from the place from to the node to.
func advViaKey(from, to string) string { return from + ">" + to }

// advGraph is the nodes and entrances of a chapter: the column of each, and
// the places it follows and that follow it.
type advGraph struct {
	col               map[string]int
	parents, children map[string][]string
}

// layoutGroup places a group of places joined by lines, given in tree
// order: the rows of a column are sorted by the middle of the parents'
// rows, then of the children's, back and forth twice, and neighbours swap
// while that crosses fewer lines, so few lines cross; a place then takes
// the row of the middle of its parents, pushed down past the one above it,
// and one without parents the row of the middle of its children.
func (g advGraph) layoutGroup(keys []string) (pos map[string]advPos, w, h int) {
	first := g.col[keys[0]]
	for _, k := range keys {
		first = min(first, g.col[k])
	}
	var cols [][]string
	for _, k := range keys {
		c := g.col[k] - first
		for len(cols) <= c {
			cols = append(cols, nil)
		}
		cols[c] = append(cols[c], k)
	}
	idx := map[string]int{}
	index := func(c []string) {
		for i, k := range c {
			idx[k] = i
		}
	}
	for _, c := range cols {
		index(c)
	}
	sortBy := func(c []string, near map[string][]string) {
		mid := map[string]float64{}
		for _, k := range c {
			ns := near[k]
			if len(ns) == 0 {
				mid[k] = float64(idx[k])
				continue
			}
			sum := 0.0
			for _, n := range ns {
				sum += float64(idx[n])
			}
			mid[k] = sum / float64(len(ns))
		}
		slices.SortStableFunc(c, func(a, b string) int { return cmp.Compare(mid[a], mid[b]) })
		index(c)
	}
	for range 2 {
		for i := 1; i < len(cols); i++ {
			sortBy(cols[i], g.parents)
		}
		for i := len(cols) - 2; i >= 0; i-- {
			sortBy(cols[i], g.children)
		}
	}
	for i := 1; i < len(cols); i++ {
		sortBy(cols[i], g.parents)
	}
	// Then two neighbours in a column swap while that crosses fewer lines to
	// the columns beside it.
	crossings := func(a, b string) int {
		n := 0
		for _, near := range []map[string][]string{g.parents, g.children} {
			for _, na := range near[a] {
				for _, nb := range near[b] {
					if idx[na] > idx[nb] {
						n++
					}
				}
			}
		}
		return n
	}
	for better, pass := true, 0; better && pass < 8; pass++ {
		better = false
		for _, c := range cols {
			for i := 0; i+1 < len(c); i++ {
				if crossings(c[i+1], c[i]) < crossings(c[i], c[i+1]) {
					c[i], c[i+1] = c[i+1], c[i]
					index(c)
					better = true
				}
			}
		}
	}
	pos = map[string]advPos{}
	for x, c := range cols {
		next := 0
		for _, k := range c {
			y := next
			if ps := g.parents[k]; len(ps) > 0 {
				sum := 0
				for _, p := range ps {
					sum += pos[p].y
				}
				y = max(next, (sum+len(ps)/2)/len(ps))
			}
			pos[k] = advPos{x, y}
			next = y + 1
			w, h = max(w, x+1), max(h, y+1)
		}
	}
	// A place without parents (an entrance, mostly) moves down to the
	// middle of its children's rows, short of the place below it, so its
	// lines do not run past the column.
	for x := len(cols) - 1; x >= 0; x-- {
		c := cols[x]
		for i := len(c) - 1; i >= 0; i-- {
			k := c[i]
			ks := g.children[k]
			if len(g.parents[k]) > 0 || len(ks) == 0 {
				continue
			}
			sum := 0
			for _, ch := range ks {
				sum += pos[ch].y
			}
			y := sum / len(ks)
			if i+1 < len(c) {
				y = min(y, pos[c[i+1]].y-1)
			}
			if y > pos[k].y {
				pos[k] = advPos{x, y}
				h = max(h, y+1)
			}
		}
	}
	return pos, w, h
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
		v.tab = a.Chapter
	}
	s.showAdvancementView(p, v)
}

// advancementTree builds the form's tree of a view: every row of the tab's
// layout.
func (s *session) advancementTree(p *player.Player, v advView) ui.Tree {
	lang := i18n.Of(p)
	l := advancementLayout(v.tab)
	var t ui.Tree
	byPos := map[advPos]*Advancement{}
	gateAt := map[advPos]*Advancement{}
	for _, a := range l.order {
		byPos[l.pos[a.ID]] = a
	}
	for _, g := range l.gates {
		gateAt[l.pos[advGateID(g.ID)]] = g
	}
	for y := range min(l.h, ui.TreeRows) {
		for x := range ui.TreeCols {
			ps := advPos{x, y}
			c := ui.TreeCell{Strip: l.strips[ps]}
			if a := byPos[ps]; a != nil {
				c.Frame = advFrame(a)
				c.Icon, c.Tip = advancementIcon(a), s.advancementTip(lang, a)
				if len(a.Parents) > 1 && (s.advanced(a.ID) || s.advancementOpen(a)) {
					c.Tip += "\n" + advancementParents(lang, a)
				}
				c.State = s.advTreeState(a)
				c.Run = func(p *player.Player, tx *world.Tx) {
					if tx.World() == s.world {
						s.showAdvancementNode(p, a)
					}
				}
			} else if g := gateAt[ps]; g != nil {
				c.Frame, c.State = ui.FrameTask, s.advTreeState(g)
				c.Icon, c.Tip = advancementIcon(g), advGateTip(lang, g)
				c.Run = func(p *player.Player, tx *world.Tx) {
					if tx.World() == s.world {
						s.showAdvancementView(p, advView{tab: g.Chapter})
					}
				}
			}
			t.Cells = append(t.Cells, c)
		}
	}
	for tab := advChapter(0); tab < advChapterCount; tab++ {
		o := ui.Tab(i18n.T(lang, "advancement.tab."+advChapterKeys[tab]), advChapterIcon(tab), tab == v.tab, nil)
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
	t.Body = fmt.Sprintf(i18n.T(lang, "advancement.progress"), i18n.T(lang, "advancement.tab."+advChapterKeys[v.tab]), done, len(list))
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
// back to its chapter); a locked one also names the parents left.
func (s *session) showAdvancementNode(p *player.Player, a *Advancement) {
	lang := i18n.Of(p)
	title, body := a.Title(lang), ""
	switch {
	case s.advanced(a.ID):
		body = i18n.T(lang, "dragonfactory.advancement.showAdvancementNode.1") + flatHint(lang, a)
	case s.advancementOpen(a):
		body = i18n.T(lang, "dragonfactory.advancement.showAdvancementNode.2") + flatHint(lang, a)
	default:
		body = i18n.T(lang, "dragonfactory.advancement.showAdvancementNode.2") + flatHint(lang, a) + "\n\n" + s.advancementBefore(lang, a)
	}
	var opts []ui.Option
	if !s.advanced(a.ID) {
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
	if it, ok := advancementStack(a); ok {
		opts = append(opts, s.opt(i18n.T(lang, "advancement.node.codex"), iconOf(it), func(p *player.Player, _ *world.Tx) {
			s.codexFrom(p, func(p *player.Player) { s.showAdvancementNode(p, a) }, it)
		}))
	}
	back := ui.Back(i18n.T(lang, "codex.back"), func(p *player.Player, _ *world.Tx) { s.showAdvancementView(p, advView{tab: a.Chapter}) })
	opts = append(opts, s.opt(back.Label, back.Icon, back.Run))
	ui.Menu(p, i18n.T(lang, "dragonfactory.advancement.showAdvancementNode.4")+title, body, opts...)
}

// advancementTip is a node's tooltip: its name, coloured by its state, and
// its hint; a locked one also names the parents left.
func (s *session) advancementTip(lang string, a *Advancement) string {
	switch {
	case s.advanced(a.ID):
		return "§a" + a.Title(lang) + "\n§7" + flatHint(lang, a)
	case s.advancementOpen(a):
		return "§e" + a.Title(lang) + "\n§7" + flatHint(lang, a)
	}
	return "§7" + a.Title(lang) + "\n§8" + flatHint(lang, a) + "\n" + s.advancementBefore(lang, a)
}

// advancementBefore names the parents of an advancement not done yet.
func (s *session) advancementBefore(lang string, a *Advancement) string {
	var names []string
	for _, id := range a.Parents {
		if p, ok := advancementByID(id); ok && !s.advanced(id) {
			names = append(names, p.Title(lang))
		}
	}
	return i18n.T(lang, "advancement.node.before") + strings.Join(names, ", ")
}

// advTreeState is the state letter of a node in the tree form.
func (s *session) advTreeState(a *Advancement) byte {
	switch {
	case s.advanced(a.ID):
		return ui.StateDone
	case s.advancementOpen(a):
		return ui.StateOpen
	}
	return ui.StateLocked
}

// advQuestState is the state of a node in the quest book.
func (s *session) advQuestState(a *Advancement) string {
	switch {
	case s.advanced(a.ID):
		return ui.QuestDone
	case s.advancementOpen(a):
		return ui.QuestOpen
	}
	return ui.QuestLocked
}

// advGateTip is the tooltip of the entrance of an advancement of another
// chapter: the chapter and the advancement.
func advGateTip(lang string, a *Advancement) string {
	return "§7" + i18n.T(lang, "advancement.tab."+advChapterKeys[a.Chapter]) + " · §f" + a.Title(lang)
}

// advancementParents names every parent of an advancement (the tree form
// draws the line of one).
func advancementParents(lang string, a *Advancement) string {
	var names []string
	for _, id := range a.Parents {
		if p, ok := advancementByID(id); ok {
			names = append(names, p.Title(lang))
		}
	}
	return i18n.T(lang, "advancement.node.parents") + strings.Join(names, ", ")
}

// advShapes are the quest shapes of the frames in the quest book.
var advShapes = map[byte]string{ui.FrameTask: ui.ShapeDiamond, ui.FrameGoal: ui.ShapeSquare, ui.FrameChallenge: ui.ShapeCircle}

// showQuestBook shows the advancement screen as a quest book (DDUI): the
// categories as chapters, the tab's tree on a plane (the same places as the
// tree form) and, when open is set, its quest panel over it.
func (s *session) showQuestBook(p *player.Player, tab advChapter, open *Advancement) {
	lang := i18n.Of(p)
	b := ui.QuestBook{Key: "adv_" + advChapterKeys[tab]}
	for t := advChapter(0); t < advChapterCount; t++ {
		done, list := 0, advancementsIn(t)
		for _, a := range list {
			if s.advanced(a.ID) {
				done++
			}
		}
		b.Chapters = append(b.Chapters, ui.QuestChapter{
			Name: i18n.T(lang, "advancement.tab."+advChapterKeys[t]), Icon: advChapterIcon(t), Count: fmt.Sprintf("%d/%d", done, len(list)), On: t == tab,
			Run: func(p *player.Player, tx *world.Tx) {
				if tx.World() == s.world {
					s.showQuestBook(p, t, nil)
				}
			},
		})
		if t == tab {
			b.Heading = fmt.Sprintf(i18n.T(lang, "advancement.progress"), i18n.T(lang, "advancement.tab."+advChapterKeys[t]), done, len(list))
		}
	}
	l := advancementLayout(tab)
	if next, ok := s.nextAdvancement(); ok && next.Chapter == tab {
		b.Focus = next.ID
	} else if len(l.order) > 0 {
		b.Focus = l.order[0].ID
	}
	for _, a := range l.order {
		pos := l.pos[a.ID]
		q := ui.QuestNode{ID: a.ID, X: advQuestX(pos.x), Y: float64(pos.y), Shape: advShapes[advFrame(a)], Tracked: s.state.Tracked == a.ID}
		q.Icon, q.Tip, q.State = advancementIcon(a), s.advancementTip(lang, a), s.advQuestState(a)
		for _, id := range a.Parents {
			par, ok := advancementByID(id)
			if !ok {
				continue
			}
			from := id
			if par.Chapter != tab {
				from = advGateID(id)
			}
			// A line past the next column goes through its vias, which take
			// the state of the whole line.
			viaState := ui.QuestOpen
			switch pst := s.advQuestState(par); {
			case pst == ui.QuestDone && q.State == ui.QuestDone:
				viaState = ui.QuestDone
			case q.State == ui.QuestLocked:
				viaState = ui.QuestLocked
			}
			for _, v := range l.vias[advViaKey(from, a.ID)] {
				vp := l.pos[v]
				b.Nodes = append(b.Nodes, ui.QuestNode{ID: v, X: advQuestX(vp.x), Y: float64(vp.y), Shape: ui.ShapeVia, State: viaState, Deps: []string{from}})
				from = v
			}
			q.Deps = append(q.Deps, from)
		}
		q.Run = func(p *player.Player, tx *world.Tx) {
			if tx.World() == s.world {
				s.showQuestBook(p, tab, a)
			}
		}
		b.Nodes = append(b.Nodes, q)
	}
	for _, g := range l.gates {
		pos := l.pos[advGateID(g.ID)]
		b.Nodes = append(b.Nodes, ui.QuestNode{
			ID: advGateID(g.ID), X: advQuestX(pos.x), Y: float64(pos.y), Shape: ui.ShapeGate,
			Icon: advancementIcon(g), Tip: advGateTip(lang, g), State: s.advQuestState(g),
			Run: func(p *player.Player, tx *world.Tx) {
				if tx.World() == s.world {
					s.showQuestBook(p, g.Chapter, g)
				}
			},
		})
	}
	if open != nil {
		b.Panel = s.questPanel(p, tab, open)
	}
	ui.QuestBookMenu(p, i18n.T(lang, "dragonfactory.advancement.showAdvancements.1"), b, s.backOption())
}

// advQuestX is the place on the quest book's plane of a column: wider apart
// than the rows, so the lines between columns have room to turn.
func advQuestX(x int) float64 { return float64(x) * 1.5 }

// questPanel is an advancement's quest panel: its state and hint, the item
// it asks for (pressing it opens the codex) and the track and codex buttons,
// like showAdvancementNode.
func (s *session) questPanel(p *player.Player, tab advChapter, a *Advancement) *ui.QuestPanel {
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
		pn.Status = s.advancementBefore(lang, a)
		pn.Body = flatHint(lang, a)
	}
	if it, ok := advancementStack(a); ok {
		pn.Rows = []ui.SlotRow{{Slots: []ui.ItemSlot{{
			Icon: iconOf(it), Tip: tipName(lang, it),
			Run: func(p *player.Player, _ *world.Tx) { s.codexFrom(p, again, it) },
		}}}}
	}
	if !s.advanced(a.ID) {
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
func advancementsIn(tab advChapter) []*Advancement {
	return advChapterLists[tab]
}

// advChapterIcon is the icon of a tab.
func advChapterIcon(t advChapter) string {
	if it, ok := world.ItemByName(unlockName(advChapterIcons[t]), 0); ok {
		return iconOf(item.NewStack(it, 1))
	}
	return iconMission
}
