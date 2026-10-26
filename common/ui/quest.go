package ui

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// Quest states of a QuestNode.
const (
	QuestLocked = "locked"
	QuestOpen   = "open"
	QuestDone   = "done"
)

// Quest shapes of a QuestNode, as FTB Quests draws them.
const (
	ShapeSquare  = "square"
	ShapeDiamond = "diamond"
	ShapeCircle  = "circle"
	// ShapeGate is a small entrance of a quest of another chapter, at the
	// edge of the plane.
	ShapeGate = "gate"
	// ShapeVia is no node but a point a line passes on its way to a quest
	// further than the next column, so it goes around the quests between.
	ShapeVia = "via"
)

// QuestChapter is a chapter in the list left of the quests: its name and
// icon, the count beside it ("3/12") and what choosing it does. On is the
// chapter shown.
type QuestChapter struct {
	Name, Icon, Count string
	On                bool
	Run               func(p *player.Player, tx *world.Tx)
}

// QuestNode is a quest on the plane: its place (1 is a node's width), icon,
// shape, state and tooltip, the quests it follows (lines are drawn from
// them) and what pressing it does.
type QuestNode struct {
	ID          string
	X, Y        float64
	Icon, Shape string
	State       string
	Tracked     bool
	Tip         string
	Deps        []string
	Run         func(p *player.Player, tx *world.Tx)
}

// QuestPanel is an open quest over the plane: the big icon, title and status,
// the text, headed rows of slots (what to bring), the buttons and what its
// close button does (show the book without the panel).
type QuestPanel struct {
	Icon, Title, Status, Body string
	Rows                      []SlotRow
	Actions                   []Option
	Close                     func(p *player.Player, tx *world.Tx)
}

// QuestBook is the content of QuestBookMenu.
type QuestBook struct {
	Chapters []QuestChapter
	// Heading is the chapter's name over the plane (the title when empty),
	// Progress the text at its right.
	Heading, Progress string
	Nodes             []QuestNode
	// Focus is the quest the plane is centred on the first time a chapter
	// is shown; the player's dragging is kept after that.
	Focus string
	// Key tells the planes apart (the chapter), so each keeps where it was
	// dragged to.
	Key   string
	Panel *QuestPanel
}

// questUnit is the width of one place on the plane, in GUI pixels: a node
// is 24.
const questUnit = 36

func rem(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) + "rem" }

// QuestBookMenu shows a quest book in the style of FTB Quests: chapters on
// the left, the chapter's quests on a draggable, zoomable plane with lines
// from their dependencies, and an open quest's panel over it. It is DDUI
// only (OreUIFor).
func QuestBookMenu(p *player.Player, title string, b QuestBook, actions ...Option) {
	l := newOreLayout()
	_, bar, _ := l.chrome(actions)
	var chapters []any
	for _, c := range b.Chapters {
		n := box("mc-chapter", "on", c.On, "ink", "dark", "children", []any{
			img("", c.Icon), box("mc-chapter-text", "children", []any{txt("mc-chapter-name", c.Name), txt("mc-chapter-count", c.Count)}),
		})
		if !c.On {
			l.press(n, c.Run)
		}
		chapters = append(chapters, n)
	}
	at := map[string]QuestNode{}
	for _, q := range b.Nodes {
		at[q.ID] = q
	}
	pos := func(v float64) float64 { return v * questUnit * 0.2 }
	var plane []any
	for _, sg := range questLines(b.Nodes) {
		plane = append(plane, node("line", "class", sg.class, "x1", pos(sg.x1), "y1", pos(sg.y1), "x2", pos(sg.x2), "y2", pos(sg.y2)))
	}
	focusX, focusY := 0.0, 0.0
	for _, q := range b.Nodes {
		if q.ID == b.Focus {
			focusX, focusY = pos(q.X), pos(q.Y)
		}
		if q.Shape == ShapeVia {
			continue
		}
		shape := q.Shape
		if shape == "" {
			shape = ShapeSquare
		}
		class := strings.Join([]string{"mc-quest", shape, q.State}, " ")
		if q.Tracked {
			class += " tracked"
		}
		n := box(class, "tip", q.Tip, "style", map[string]any{"left": rem(pos(q.X)), "top": rem(pos(q.Y))},
			"children", []any{box("mc-quest-shape"), img("", q.Icon)})
		plane = append(plane, l.press(n, q.Run))
	}
	bar = append(bar, box("mc-close", "id", "close", "children", []any{node("img", "icon", "tex:textures/ui/close_button_default_light")}))
	heading := b.Heading
	if heading == "" {
		heading = untoken(title)
	}
	main := []any{
		box("mc-quest-head", "children", []any{txt("", heading), txt("mc-quest-progress", b.Progress)}),
		node("canvas", "key", b.Key, "focusX", focusX, "focusY", focusY, "children", plane),
	}
	if pn := b.Panel; pn != nil {
		head := []any{l.slotNode(ItemSlot{Icon: pn.Icon}, "mc-big"), box("mc-col", "children", []any{txt("mc-quest-title", pn.Title), txt("mc-small-text", pn.Status)})}
		head = append(head, l.press(box("mc-close", "children", []any{node("img", "icon", "tex:textures/ui/close_button_default_light")}), pn.Close))
		body := []any{}
		if pn.Body != "" {
			body = append(body, txt("mc-small-text", pn.Body))
		}
		for _, r := range pn.Rows {
			var slots []any
			for _, sl := range r.Slots {
				slots = append(slots, l.slotNode(sl))
			}
			body = append(body, box("mc-gap-3"))
			if r.Heading != "" {
				body = append(body, txt("mc-head", r.Heading))
			}
			body = append(body, box("mc-grid", "children", slots))
		}
		for _, a := range pn.Actions {
			body = append(body, box("mc-gap-3"), l.actionButton(a))
		}
		main = append(main, box("mc-quest-panel", "children", []any{
			box("mc-quest-panel-head", "children", head),
			scroll(box("mc-col", "children", body)),
		}))
	}
	w, h := windowW, windowH
	l.size = fmt.Sprintf("%dx%d", w, h)
	win := box("mc-quest-window", "ink", "panel", "style", map[string]any{"width": gu(w), "height": gu(h)}, "children", []any{
		box("mc-toolbar", "children", []any{box("mc-toolbar-bg"), box("mc-toolbar-row", "children", bar)}),
		box("mc-chapters", "children", []any{scroll(box("mc-col", "children", chapters))}),
		box("mc-quest-main", "children", main),
	})
	showOre(p, l, box("mc-screen", "children", []any{win}))
}

// questSeg is a straight piece of a line on the plane, in places.
type questSeg struct {
	x1, y1, x2, y2 float64
	class          string
	from, to       string
}

// questLines are the lines from the quests each quest follows. A line runs
// right to its own lane in the gap before the quest's column, along it and
// into the quest, so it never crosses a node or overlaps another line; lanes
// are ordered for the fewest crossings.
func questLines(nodes []QuestNode) []questSeg {
	at := map[string]QuestNode{}
	var xs []float64
	for _, q := range nodes {
		at[q.ID] = q
		xs = append(xs, q.X)
	}
	slices.Sort(xs)
	xs = slices.Compact(xs)
	// prev is the column before x, or a node's width before it.
	prev := func(x float64) float64 {
		i, _ := slices.BinarySearch(xs, x)
		if i == 0 {
			return x - 1
		}
		return xs[i-1]
	}
	type edge struct{ from, to string }
	// entry is the height a line comes into its quest at.
	entry := map[edge]float64{}
	for _, q := range nodes {
		var deps []QuestNode
		for _, d := range q.Deps {
			if from, ok := at[d]; ok {
				deps = append(deps, from)
			}
		}
		if len(deps) < 2 {
			continue
		}
		slices.SortStableFunc(deps, func(a, b QuestNode) int { return cmp.Compare(a.Y, b.Y) })
		// A line from the same row stays straight; the others are spread
		// around it, a node is 2/3 of a place high.
		mid := float64(len(deps)-1) / 2
		if i := slices.IndexFunc(deps, func(n QuestNode) bool { return n.Y == q.Y }); i >= 0 {
			mid = float64(i)
		}
		reach := max(mid, float64(len(deps)-1)-mid)
		step := min(0.15, 0.25/reach)
		for i, from := range deps {
			entry[edge{from.ID, q.ID}] = (float64(i) - mid) * step
		}
	}
	type lane struct {
		from QuestNode
		ins  []float64 // heights its lines come into their quests at
	}
	lanes := map[float64][]lane{}
	for _, q := range nodes {
		for _, d := range q.Deps {
			from, ok := at[d]
			if !ok || from.Y == q.Y {
				continue
			}
			ls := lanes[q.X]
			i := slices.IndexFunc(ls, func(l lane) bool { return l.from.ID == from.ID })
			if i < 0 {
				ls, i = append(ls, lane{from: from}), len(ls)
			}
			ls[i].ins = append(ls[i].ins, q.Y+entry[edge{from.ID, q.ID}])
			lanes[q.X] = ls
		}
	}
	for x, ls := range lanes {
		lanes[x] = questLaneOrder(ls, func(l lane) (float64, []float64) { return l.from.Y, l.ins })
	}
	var out []questSeg
	for _, q := range nodes {
		for _, d := range q.Deps {
			from, ok := at[d]
			if !ok {
				continue
			}
			class := "oui-line-seg"
			switch {
			case from.State == QuestDone && q.State == QuestDone:
				class += " done"
			case q.State == QuestLocked:
				class += " locked"
			}
			in := q.Y + entry[edge{from.ID, q.ID}]
			if from.Y == q.Y {
				out = append(out, questSeg{from.X, from.Y, q.X, in, class, from.ID, q.ID})
				continue
			}
			ls := lanes[q.X]
			i := slices.IndexFunc(ls, func(l lane) bool { return l.from.ID == from.ID })
			// The gap between the nodes (a node is 2/3 of a place wide),
			// shared by its lanes.
			p := max(prev(q.X), from.X)
			room := q.X - p - 2.0/3
			step := min(0.15, room*0.8/float64(len(ls)))
			lx := (p+q.X)/2 + (float64(i)-float64(len(ls)-1)/2)*step
			// The upright piece reaches half a line's width past each turn,
			// so the corners are closed.
			e := 0.2 / (questUnit * 0.2)
			if in < from.Y {
				e = -e
			}
			out = append(out,
				questSeg{from.X, from.Y, lx, from.Y, class, from.ID, q.ID},
				questSeg{lx, from.Y - e, lx, in + e, class, from.ID, q.ID},
				questSeg{lx, in, q.X, in, class, from.ID, q.ID})
		}
	}
	// Lines of a quest followed share their first piece: the done ones are
	// drawn last, over the open and locked ones, and the locked ones first.
	rank := func(class string) int {
		switch {
		case strings.HasSuffix(class, " done"):
			return 2
		case strings.HasSuffix(class, " locked"):
			return 0
		}
		return 1
	}
	slices.SortStableFunc(out, func(a, b questSeg) int { return cmp.Compare(rank(a.class), rank(b.class)) })
	return out
}

// questLaneOrder orders the lanes of a gap, left to right, for the fewest
// crossings and overlaps. Up to seven lanes every order is tried; more are
// improved by swapping neighbours. Ties keep the lower quest followed left,
// as lines mostly run down.
func questLaneOrder[L any](ls []L, at func(L) (float64, []float64)) []L {
	slices.SortStableFunc(ls, func(a, b L) int {
		ya, _ := at(a)
		yb, _ := at(b)
		return cmp.Compare(yb, ya)
	})
	n := len(ls)
	if n < 2 {
		return ls
	}
	type span struct {
		src, lo, hi float64
		ins         []float64
	}
	sp := make([]span, n)
	for i, l := range ls {
		y, ins := at(l)
		lo, hi := y, y
		for _, v := range ins {
			lo, hi = min(lo, v), max(hi, v)
		}
		sp[i] = span{y, lo, hi, ins}
	}
	const eps = 1e-6
	reaches := func(s span, y float64) bool { return y > s.lo+eps && y < s.hi-eps }
	// cost[a][b] is the crossings of a left of b.
	cost := make([][]int, n)
	for a := range cost {
		cost[a] = make([]int, n)
		for b := range cost[a] {
			if a == b {
				continue
			}
			if reaches(sp[a], sp[b].src) {
				cost[a][b]++
			}
			for _, v := range sp[a].ins {
				switch {
				case reaches(sp[b], v):
					cost[a][b]++
				case math.Abs(v-sp[b].src) < eps:
					// b's line comes in on a's line out: they would lie
					// on each other.
					cost[a][b] += 10
				}
			}
		}
	}
	total := func(order []int) int {
		c := 0
		for i := range order {
			for j := i + 1; j < len(order); j++ {
				c += cost[order[i]][order[j]]
			}
		}
		return c
	}
	best := make([]int, n)
	for i := range best {
		best[i] = i
	}
	bestCost := total(best)
	if n <= 7 {
		var try func(order []int, used []bool)
		try = func(order []int, used []bool) {
			if len(order) == n {
				if c := total(order); c < bestCost {
					best, bestCost = slices.Clone(order), c
				}
				return
			}
			for i := 0; i < n; i++ {
				if !used[i] {
					used[i] = true
					try(append(order, i), used)
					used[i] = false
				}
			}
		}
		try(nil, make([]bool, n))
	} else {
		for better := true; better; {
			better = false
			for i := 0; i+1 < n; i++ {
				a, b := best[i], best[i+1]
				if cost[b][a] < cost[a][b] {
					best[i], best[i+1] = b, a
					better = true
				}
			}
		}
	}
	out := make([]L, n)
	for i, j := range best {
		out[i] = ls[j]
	}
	return out
}
