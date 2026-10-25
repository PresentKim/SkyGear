package ui

import (
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
	// Heading is the chapter's name over the plane, Progress the text at its
	// right.
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

// QuestBookMenu shows a quest book the way FTB Quests does in ATM10 To The
// Sky: the chapters on the left, the chapter's quests on a dark plane the
// player drags around, with lines from the quests each follows, and the
// panel of an open quest over it. It is a DDUI screen only (OreUIFor); the
// window takes most of the screen. Actions (Back, Help) go to the control
// bar.
func QuestBookMenu(p *player.Player, title string, b QuestBook, actions ...Option) {
	l := newOreLayout()
	_, bar, _ := l.chrome(actions, false)
	var chapters []any
	for _, c := range b.Chapters {
		n := box("mc-chapter", "on", c.On, "ink", "dark", "children", []any{
			img("", c.Icon), txt("mc-chapter-name", c.Name), txt("mc-chapter-count", c.Count),
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
	for _, q := range b.Nodes {
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
			plane = append(plane, node("line", "class", class, "x1", pos(from.X), "y1", pos(from.Y), "x2", pos(q.X), "y2", pos(q.Y)))
		}
	}
	focusX, focusY := 0.0, 0.0
	for _, q := range b.Nodes {
		if q.ID == b.Focus {
			focusX, focusY = pos(q.X), pos(q.Y)
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
	main := []any{
		box("mc-quest-head", "children", []any{txt("", b.Heading), txt("mc-quest-progress", b.Progress)}),
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
	bar = append(bar, box("mc-close", "id", "close", "children", []any{node("img", "icon", "tex:textures/ui/close_button_default_light")}))
	win := box("mc-quest-window", "ink", "panel", "children", []any{
		box("mc-toolbar", "children", []any{box("mc-toolbar-bg"), box("mc-toolbar-row", "children", bar)}),
		box("mc-chapters", "children", []any{txt("mc-quest-head", untoken(title)), scroll(box("mc-col", "children", chapters))}),
		box("mc-quest-main", "children", main),
	})
	l.size = "quest"
	showOre(p, l, box("mc-screen", "children", []any{win}))
}
