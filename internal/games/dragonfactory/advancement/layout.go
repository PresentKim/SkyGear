package advancement

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/common/ui"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/formicon"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/naming"
)

// Chapter is a chapter of the advancement screen: the advancements
// grouped by how their things are got (by hand, the sieve, farming and
// barrels, machines, alloys and fluids, hunting, gear, magic), then the
// projects and the features beside the flow. A parent in another chapter
// is an entrance, so an advancement stays with the way its thing is got.
type Chapter int

const (
	ChapterStart Chapter = iota
	ChapterSieve
	ChapterFarming
	ChapterAutomation
	ChapterLogistics
	ChapterLab
	ChapterAlloy
	ChapterHunting
	ChapterGear
	ChapterMagic
	ChapterFairy
	ChapterProject
	ChapterExtra
	ChapterCount
)

// ChapterKeys name the chapters in the catalog ("advancement.tab.<key>").
var ChapterKeys = [ChapterCount]string{"start", "sieve", "farming", "automation", "logistics", "lab", "alloy", "hunting", "gear", "magic", "fairy", "project", "extra"}

// advChapterIcons are items whose icon marks each chapter.
var advChapterIcons = [ChapterCount]string{
	"crafting_table", "dragonfactory:sieve", "wheat", "dragonfactory:generator", "hopper", "dragonfactory:laboratory", "dragonfactory:alloy_smelter",
	"dragonfactory:mob_simulator", "iron_pickaxe", "dragonfactory:petal_red", "dragonfactory:fairy_dust", "dragon_egg", "lantern",
}

// advChapterLists are the advancements of each chapter in tree order.
var advChapterLists = func() [ChapterCount][]*Advancement {
	var lists [ChapterCount][]*Advancement
	for i := range Tree {
		a := &Tree[i]
		lists[a.Chapter] = append(lists[a.Chapter], a)
	}
	return lists
}()

// View is what the advancement screen shows: the tab.
type View struct{ Tab Chapter }

// Pos is a cell of a chapter's tree grid.
type Pos struct{ X, Y int }

// Layout is the placed tree of a chapter: where each node and each
// entrance is, the parent each node's connector comes from in the tree form,
// the connector lines of every cell's strip, and the size. order lists the
// nodes in tree order and gates the entrances top to bottom.
type Layout struct {
	Pos    map[string]Pos
	Line   map[string]string
	Strips map[Pos]int
	W, H   int
	Order  []*Advancement
	Gates  []*Advancement
	// Vias are the places a line passes in the columns between a parent and
	// a node further than the next column, by advViaKey: kept free, so the
	// quest book's line goes around the nodes between.
	Vias map[string][]string
}

// GateID is the place key of the entrance of an advancement of another
// chapter.
func GateID(id string) string { return "gate:" + id }

// advLayouts are the layouts of the chapters: every node is shown, so they
// do not change.
var advLayouts = func() (out [ChapterCount]Layout) {
	for ch := range ChapterCount {
		out[ch] = layoutChapter(ch)
	}
	return out
}()

// LayoutOf is the layout of a chapter.
func LayoutOf(ch Chapter) Layout { return advLayouts[ch] }

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
func layoutChapter(ch Chapter) Layout {
	l := Layout{Pos: map[string]Pos{}, Line: map[string]string{}, Strips: map[Pos]int{}, Vias: map[string][]string{}}
	l.Order = In(ch)
	g := advGraph{col: map[string]int{}, parents: map[string][]string{}, children: map[string][]string{}}
	var keys []string
	gateOf := map[string]bool{}
	for _, a := range l.Order {
		c := 0
		for _, id := range a.Parents {
			par, ok := ByID(id)
			if !ok {
				continue
			}
			key := id
			if par.Chapter != ch {
				key = GateID(id)
				if !gateOf[key] {
					gateOf[key] = true
					l.Gates = append(l.Gates, par)
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
	for _, gt := range l.Gates {
		key := GateID(gt.ID)
		c := -1
		for _, k := range g.children[key] {
			if c < 0 || g.col[k]-1 < c {
				c = g.col[k] - 1
			}
		}
		g.col[key] = c
	}
	real := g.parents
	g, keys = g.withVias(keys, l.Vias)
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
			l.Pos[k] = Pos{x0 + p.X, y0 + p.Y}
		}
		band = max(band, w)
		l.W, l.H = max(l.W, x0+w), max(l.H, y0+h)
		y0 += h
	}
	for _, a := range l.Order {
		c := l.Pos[a.ID]
		var from string
		for _, k := range real[a.ID] {
			if l.Pos[k].X == c.X-1 && (from == "" || abs(l.Pos[k].Y-c.Y) < abs(l.Pos[from].Y-c.Y)) {
				from = k
			}
		}
		if from == "" {
			continue
		}
		l.Line[a.ID] = from
		p := l.Pos[from]
		l.Strips[Pos{c.X, p.Y}] |= ui.StripLeft
		l.Strips[c] |= ui.StripRight
		switch {
		case c.Y > p.Y:
			l.Strips[Pos{c.X, p.Y}] |= ui.StripDown
			l.Strips[c] |= ui.StripUp
		case c.Y < p.Y:
			l.Strips[Pos{c.X, p.Y}] |= ui.StripUp
			l.Strips[c] |= ui.StripDown
		}
		for y := min(c.Y, p.Y) + 1; y < max(c.Y, p.Y); y++ {
			l.Strips[Pos{c.X, y}] |= ui.StripUp | ui.StripDown
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
				v := fmt.Sprintf("via:%s:%d", ViaKey(p, k), c)
				e.col[v] = c
				e.parents[v] = []string{from}
				e.children[from] = append(e.children[from], v)
				vias[ViaKey(p, k)] = append(vias[ViaKey(p, k)], v)
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

// ViaKey names the line from the place from to the node to.
func ViaKey(from, to string) string { return from + ">" + to }

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
func (g advGraph) layoutGroup(keys []string) (pos map[string]Pos, w, h int) {
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
	pos = map[string]Pos{}
	for x, c := range cols {
		next := 0
		for _, k := range c {
			y := next
			if ps := g.parents[k]; len(ps) > 0 {
				sum := 0
				for _, p := range ps {
					sum += pos[p].Y
				}
				y = max(next, (sum+len(ps)/2)/len(ps))
			}
			pos[k] = Pos{x, y}
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
				sum += pos[ch].Y
			}
			y := sum / len(ks)
			if i+1 < len(c) {
				y = min(y, pos[c[i+1]].Y-1)
			}
			if y > pos[k].Y {
				pos[k] = Pos{x, y}
				h = max(h, y+1)
			}
		}
	}
	return pos, w, h
}

// advChallenges are the big milestones, drawn with the challenge frame; the
// advancements of a research get the goal frame, the rest the task frame.
var advChallenges = map[string]bool{
	Dragon: true, Wave8: true, "chunks_16": true,
	Tool5: true, "project_dragon": true, "ess_harvest": true, "gem_harvest": true,
}

// Frame is the frame letter of a node.
func Frame(a *Advancement) byte {
	switch {
	case advChallenges[a.ID]:
		return ui.FrameChallenge
	case a.Research != "":
		return ui.FrameGoal
	}
	return ui.FrameTask
}

// GateTip is the tooltip of the entrance of an advancement of another
// chapter: the chapter and the advancement.
func GateTip(lang string, a *Advancement) string {
	return "§7" + i18n.T(lang, "advancement.tab."+ChapterKeys[a.Chapter]) + " · §f" + a.Title(lang)
}

// Parents names every parent of an advancement (the tree form
// draws the line of one).
func Parents(lang string, a *Advancement) string {
	var names []string
	for _, id := range a.Parents {
		if p, ok := ByID(id); ok {
			names = append(names, p.Title(lang))
		}
	}
	return i18n.T(lang, "advancement.node.parents") + strings.Join(names, ", ")
}

// Shapes are the quest shapes of the frames in the quest book.
var Shapes = map[byte]string{ui.FrameTask: ui.ShapeDiamond, ui.FrameGoal: ui.ShapeSquare, ui.FrameChallenge: ui.ShapeCircle}

// QuestX is the place on the quest book's plane of a column: wider apart
// than the rows, so the lines between columns have room to turn.
func QuestX(x int) float64 { return float64(x) * 1.5 }

// In lists a category in tree order. The list is shared: callers
// only read it.
func In(tab Chapter) []*Advancement {
	return advChapterLists[tab]
}

// ChapterIcon is the icon of a tab.
func ChapterIcon(t Chapter) string {
	if it, ok := world.ItemByName(naming.UnlockName(advChapterIcons[t]), 0); ok {
		return formicon.Of(item.NewStack(it, 1))
	}
	return formicon.Mission
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
