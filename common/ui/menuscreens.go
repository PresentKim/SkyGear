package ui

import (
	"cmp"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/invui"
)

// The screens that have no page of their own in the menu window, drawn with
// the list page's cells and rows.

// board writes cells left to right in rows of cols cells. A row shows while
// its first cell holds something, so a row that is written gets a blank first
// cell when nothing else is there.
type board struct {
	pg   *menuPage
	i    int
	cols int
}

func newBoard(pg *menuPage, cols int) *board { return &board{pg: pg, cols: cols} }

func (b *board) touch() {
	start := b.i - b.i%b.cols
	if start < invui.Cells && b.pg.items[invui.Cell0+start].Empty() {
		b.pg.mark(start, "", invui.KindBlank)
	}
}

func (b *board) slot(s dSlot) {
	b.touch()
	b.pg.cell(b.i, s, invui.KindRow, invui.KindStatic)
	b.i++
}

func (b *board) mini(s dSlot) {
	b.touch()
	b.pg.cell(b.i, s, invui.KindMini, invui.KindMiniView)
	b.i++
}

func (b *board) mark(text, kind string, span int) {
	b.touch()
	b.pg.mark(b.i, text, kind)
	b.i += max(span, 1)
}

func (b *board) skip(n int) {
	b.touch()
	b.i += n
}

// line moves to the start of the next row, unless at the start of one.
func (b *board) line() {
	if b.i%b.cols != 0 {
		b.i += b.cols - b.i%b.cols
	}
}

// at moves to the cell of column c in row r.
func (b *board) at(r, c int) { b.i = r*b.cols + c }

// cells writes a board row (cellsOf); an empty row still takes
// its room.
func (b *board) cells(row []dCell) {
	b.touch()
	for _, c := range row {
		switch c.K {
		case "slot":
			b.slot(*c.Slot)
		case "mini":
			b.mini(*c.Slot)
		case "frame":
			b.mark("", invui.KindFrame, 1)
		case "arrow":
			b.mark("", invui.KindArrow, 1)
		case "text":
			b.mark(ink(c.Text), invui.KindText, c.Span)
		default:
			b.skip(1)
		}
	}
	b.line()
}

// sections writes headed rows of slots.
func (b *board) sections(rows []dSlotRow) {
	for _, r := range rows {
		if r.Heading != "" {
			b.mark(ink(r.Heading), invui.KindHeading, b.cols)
		}
		for _, s := range r.Slots {
			b.slot(s)
		}
		b.line()
	}
}

func listPage(key string, c dChrome, body string) *menuPage {
	pg := newMenuPage(key)
	pg.chrome(c)
	pg.text(invui.SlotBody, "§r"+ink(body))
	return pg
}

func slotMenuPage(v slotMenuData) *menuPage {
	pg := listPage(invui.KeyBoard+invui.KeyRows, v.Chrome, v.Info)
	b := newBoard(pg, invui.BoardColumns)
	b.slot(v.Store)
	b.line()
	for _, s := range v.Left {
		b.slot(s)
	}
	if len(v.Left) > 0 && len(v.Right) > 0 {
		b.skip(1)
	}
	for _, s := range v.Right {
		b.slot(s)
	}
	pg.rows(0, v.Items)
	return pg
}

func choicePage(v choiceData) *menuPage {
	stats := make([]string, 0, len(v.Stats))
	for _, s := range v.Stats {
		stats = append(stats, s.Label+" "+s.Value)
	}
	body := strings.Join(stats, "§r   ")
	if v.Note != "" {
		body += "\n" + v.Note
	}
	pg := listPage(invui.KeyCards+invui.KeyRows, v.Chrome, body)
	i := 0
	put := func(icon, label, kind, id string) {
		if i < invui.Rows {
			pg.icon(invui.RowIcon0+i, icon, kind)
			pg.button(invui.RowBtn0+i, label, kind, id)
		}
		i++
	}
	newRow := func() {
		if i%invui.CardColumns != 0 {
			i += invui.CardColumns - i%invui.CardColumns
		}
	}
	for _, g := range v.Groups {
		newRow()
		put("", ink(g.Title), invui.KindText, "")
		newRow()
		for _, c := range g.Choices {
			kind := invui.KindRow
			switch {
			case c.On:
				kind += invui.KindChosen
			case c.Dim:
				kind = invui.KindStatic
			}
			put(c.Icon, c.Tip, kind, c.ID)
		}
	}
	newRow()
	pg.rows(i, v.Items)
	return pg
}

func sievePage(v sieveData) *menuPage {
	pg := listPage(invui.KeyBoard+invui.KeyRows, v.Chrome, "")
	b := newBoard(pg, invui.BoardColumns)
	for _, row := range v.Top {
		b.cells(row)
	}
	drops := b.cols - 3
	for _, g := range v.Groups {
		b.slot(g.Head)
		b.mark(ink(g.Name), invui.KindText, b.cols-1)
		for _, r := range g.Recipes {
			b.slot(r.Input)
			b.slot(r.Mesh)
			b.mark("", invui.KindArrow, 1)
			for n, d := range r.Drops {
				if n > 0 && n%drops == 0 {
					b.line()
					b.skip(3)
				}
				b.slot(d)
			}
			b.line()
		}
	}
	pg.rows(0, v.Items)
	return pg
}

func sidesPage(v sidesData) *menuPage {
	pg := listPage(invui.KeyBoard, v.Chrome, v.Guide)
	b := newBoard(pg, invui.BoardColumns)
	for _, c := range v.Cards {
		if c.Icon != "" {
			b.slot(dSlot{Icon: c.Icon, Tip: c.Tip})
		} else {
			b.mark("", invui.KindFrame, 1)
		}
		name := c.Name
		if c.Focus {
			name = "§l" + name
		}
		if c.None != "" {
			name += "§r  §7" + c.None
		}
		b.mark(ink(name), invui.KindText, 4)
		open := false
		for _, port := range c.Ports {
			if port.Swatch != nil {
				b.slot(dSlot{ID: port.Swatch.ID, Icon: port.Swatch.Icon, Tip: port.Label + ": " + port.Swatch.Tip})
			} else {
				b.mark("", invui.KindFrame, 1)
			}
			b.mini(dSlot{ID: port.Plus.ID, Icon: "tex:textures/ui/plus"})
			open = open || port.Plus.On
		}
		b.line()
		if open && v.Picker != nil {
			b.skip(5)
			for _, ch := range v.Picker.Choices {
				b.slot(dSlot{ID: ch.ID, Icon: ch.Icon, Tip: ch.Tip, Selected: ch.On})
			}
			b.line()
		}
	}
	return pg
}

func pickPage(v pickData) *menuPage {
	pg := listPage(invui.KeyPick+invui.KeyRows, v.Chrome, v.Body)
	for i, c := range v.Cards {
		if i >= invui.PickCards {
			break
		}
		kind := invui.KindRow
		switch c.Rarity {
		case "rare":
			kind += invui.KindRare
		case "epic":
			kind += invui.KindEpic
		}
		pg.icon(invui.RowIcon0+i, c.Icon, kind)
		pg.button(invui.RowBtn0+i, c.Title, kind, c.ID)
		pg.text(invui.Cell0+2*i, "§r"+i18n.Plain(c.Tag))
		pg.text(invui.Cell0+2*i+1, "§r"+i18n.Plain(c.Text))
	}
	pg.rows(invui.PickCards, v.Items)
	return pg
}

func resultPage(v resultData) *menuPage {
	head := "§7" + v.Headline
	if v.Win {
		head = "§6" + v.Headline
	}
	pg := listPage(invui.KeyResult+invui.KeyRows, v.Chrome, "")
	pg.text(invui.SlotInfo, "§r"+i18n.Plain(head))
	for i, st := range v.Stars {
		if i >= invui.ResultStars {
			break
		}
		star := "tex:textures/ui/emptyStar"
		if st.On {
			star = "tex:textures/ui/filledStar"
		}
		pg.cell(invui.ResultStar0+i, dSlot{Icon: star, Selected: st.New}, invui.KindRow, invui.KindStatic)
	}
	for i, ln := range v.Lines {
		if i >= invui.ResultLines {
			break
		}
		pg.mark(invui.ResultLine0+2*i, ink(ln.Label), invui.KindText)
		pg.mark(invui.ResultLine0+2*i+1, ink(ln.Value), invui.KindText)
	}
	for i, s := range v.Rewards {
		if i >= invui.ResultRewards {
			break
		}
		pg.cell(invui.ResultReward0+i, s, invui.KindRow, invui.KindStatic)
	}
	pg.rows(0, []dItem{{dButton: v.OK}})
	return pg
}

// equipPage draws a group of equipment slots a row, its heading at the left,
// and the bag under them.
func equipPage(v equipData) *menuPage {
	pg := listPage(invui.KeyBoard+invui.KeyRows, v.Chrome, v.Note)
	b := newBoard(pg, invui.BoardColumns)
	const head = 3
	for _, g := range v.Groups {
		b.mark(ink(g.Heading), invui.KindText, head)
		for _, s := range g.Slots {
			if s.Icon == "" {
				s.Icon = s.Empty
			}
			b.slot(s.dSlot)
		}
		b.line()
	}
	b.sections([]dSlotRow{{Heading: v.BagHeading, Slots: v.Bag}})
	pg.rows(0, v.Items)
	return pg
}

// consoleLines is how many of the newest console lines the menu window shows.
const consoleLines = 12

func consolePage(v consoleData) *menuPage {
	lines := v.Lines[max(0, len(v.Lines)-consoleLines):]
	pg := listPage(invui.KeyMenu+invui.KeyRows, v.Chrome, strings.Join(lines, "\n§r"))
	pg.rows(0, []dItem{{dButton: v.Prompt}})
	return pg
}

func searchPage(v searchData) *menuPage {
	pg := listPage(invui.KeyGrid+invui.KeyRows, v.Chrome, v.Info)
	newBoard(pg, invui.GridColumns).sections(v.Sections)
	text := v.Query
	if text == "" {
		text = "§7" + v.Placeholder
	}
	pg.icon(invui.RowIcon0+invui.RowField, v.Prompt.Icon, invui.KindField)
	pg.button(invui.RowBtn0+invui.RowField, text, invui.KindField, v.Prompt.ID)
	pg.rows(0, v.Items)
	return pg
}

func loadingPage(v loadingData) *menuPage {
	pg := newMenuPage(invui.KeyLoading)
	pg.icon(invui.RowIcon0, v.Icon, invui.KindStatic)
	pg.text(invui.SlotTitle, "§r"+i18n.Plain(v.Title))
	pg.text(invui.SlotInfo, "§r"+i18n.Plain(v.Sub))
	pg.text(invui.SlotBody, "§r"+i18n.Plain(v.Line)+"...")
	pg.text(invui.Cell0, "§r"+i18n.Plain(v.Tip))
	return pg
}

func radialPage(v radialData) *menuPage {
	n := min(max(len(v.Items), 1), invui.RadialItems)
	pg := newMenuPage(invui.KeyRadial + strconv.Itoa(n))
	pg.text(invui.SlotTitle, "§r"+i18n.Plain(v.Title))
	pg.text(invui.SlotBody, "§r"+i18n.Plain(v.Line))
	pg.text(invui.SlotInfo, "§r"+i18n.Plain(v.Info))
	for i, it := range v.Items[:min(len(v.Items), n)] {
		kind, label := invui.KindRow, it.Label
		switch {
		case it.ID == "":
			kind, label = invui.KindStatic, "§7"+label
		case it.On:
			label = "§a" + label
		}
		pg.icon(invui.RowIcon0+i, it.Icon, kind)
		pg.button(invui.RowBtn0+i, label, kind, it.ID)
		pg.text(invui.Cell0+i, "§r"+i18n.Plain(it.Value))
	}
	back, icon := v.Back, "textures/ui/sg/chevron_left"
	if back == "close" {
		back, icon = "", "textures/ui/cancel"
	}
	pg.icon(invui.RowIcon0+invui.RadialItems, oreIcon(icon), invui.KindRow)
	pg.button(invui.RowBtn0+invui.RadialItems, v.BackTip, invui.KindRow, back)
	return pg
}

// planePage draws the spots as cells on a map, a cell for a spot of the
// smallest size.
func planePage(v planeData) *menuPage {
	body := v.Info
	if v.Corner != "" {
		body = v.Corner + "\n" + body
	}
	pg := listPage(invui.KeyBoard+invui.KeyRows, dChrome{}, body)
	b := newBoard(pg, invui.BoardColumns)
	if len(v.Spots) > 0 {
		unit, minX, minZ := math.Inf(1), math.Inf(1), math.Inf(1)
		for _, s := range v.Spots {
			unit, minX, minZ = min(unit, s.At[2]), min(minX, s.At[0]), min(minZ, s.At[1])
		}
		ground := oreIcon(ItemIcon(block.Grass{}))
		for _, s := range v.Spots {
			c, r := int((s.At[0]-minX)/unit), int((s.At[1]-minZ)/unit)
			if c >= b.cols {
				continue
			}
			b.at(r, c)
			b.slot(dSlot{ID: s.ID, Icon: ground, Tip: s.Tip, Selected: s.On})
		}
	}
	items := make([]dItem, 0, len(v.Actions))
	for _, a := range v.Actions {
		items = append(items, dItem{dButton: a})
	}
	pg.rows(0, items)
	return pg
}

// questPage draws the open quest's panel, or the chapters as tabs over the
// quests as cells at their places.
func questPage(v questData) *menuPage {
	c := dChrome{Title: v.Heading, Bar: v.Bar}
	if pn := v.Panel; pn != nil {
		c.Title = pn.Title
		c.Bar = append([]dBarButton{{ID: pn.Close, Icon: oreIcon("textures/ui/sg/chevron_left")}}, c.Bar...)
		pg := listPage(invui.KeyGrid+invui.KeyRows, c, pn.Status+"\n"+pn.Body)
		newBoard(pg, invui.GridColumns).sections(pn.Rows)
		items := make([]dItem, 0, len(pn.Actions))
		for _, a := range pn.Actions {
			items = append(items, dItem{dButton: a})
		}
		pg.rows(0, items)
		return pg
	}
	if pg := questCanvas(v, c); pg != nil {
		return pg
	}
	for _, ch := range v.Chapters {
		c.Tabs = append(c.Tabs, dTab{ID: ch.ID, Tip: ch.Name + "§r  " + ch.Count, Icon: ch.Icon, On: ch.On})
	}
	pg := listPage(invui.KeyMenu+invui.KeyRows, c, v.Progress)
	items := make([]dItem, 0, len(v.Nodes))
	for _, n := range v.Nodes {
		if n.Shape == ShapeVia {
			continue
		}
		items = append(items, dItem{dButton: dButton{ID: n.ID, Label: strings.ReplaceAll(n.Tip, "\n", "  "), Icon: n.Icon}})
	}
	pg.rows(0, items)
	return pg
}

// questCanvas draws the quest book on its own page: the chapters down the
// left and the chapter's quests on a dark canvas, a quest a cell, with the
// lines between them through the gaps right of the quests. It is nil when
// the chapter does not fit any canvas (invui.QuestGrids).
func questCanvas(v questData, c dChrome) *menuPage {
	var xs, ys []float64
	for _, n := range v.Nodes {
		xs, ys = append(xs, n.X), append(ys, n.Y)
	}
	slices.Sort(xs)
	slices.Sort(ys)
	xs, ys = slices.Compact(xs), slices.Compact(ys)
	grid := -1
	for i, g := range invui.QuestGrids {
		if len(xs) <= g[0] && len(ys) <= g[1] {
			grid = i
			break
		}
	}
	if grid < 0 || len(v.Chapters) > invui.QuestChapters {
		return nil
	}
	cols := invui.QuestGrids[grid][0]
	left := (cols - len(xs)) / 2

	pg := newMenuPage(invui.KeyQuest + strconv.Itoa(grid))
	pg.chrome(dChrome{Title: v.Title, Bar: c.Bar})
	heading := v.Heading
	if v.Progress != "" {
		heading += "  " + v.Progress
	}
	pg.text(invui.SlotInfo, "§r"+ink(heading))
	for i, ch := range v.Chapters {
		kind := invui.KindRow
		if ch.On {
			kind = invui.KindChosen
		}
		if ch.New {
			kind += invui.KindNew
		}
		pg.icon(invui.RowIcon0+i, ch.Icon, kind)
		pg.button(invui.RowBtn0+i, ch.Name, kind, ch.ID)
		pg.button(invui.RowBtn0+invui.QuestChapters+i, ch.Count, invui.KindText, "")
	}

	// A place on the canvas a line passes: 2c is the middle of column c, 2c+1
	// the lane in the gap right of it.
	place := func(vs []float64, v float64) int {
		i, found := slices.BinarySearchFunc(vs, v, func(a, b float64) int {
			if math.Abs(a-b) < 0.01 {
				return 0
			}
			return cmp.Compare(a, b)
		})
		if found {
			return 2 * i
		}
		return 2*i - 1
	}
	row := func(y float64) int {
		best := 0
		for i, r := range ys {
			if math.Abs(r-y) < math.Abs(ys[best]-y) {
				best = i
			}
		}
		return best
	}
	type at struct{ c, r int }
	kinds := map[at]string{}
	rank := map[at]int{}
	mark := func(c, r int, kind string, k int) {
		if c < 0 || c >= len(xs) {
			return
		}
		p := at{c, r}
		if !strings.Contains(kinds[p], kind) {
			kinds[p] += kind
		}
		if old, ok := rank[p]; !ok || k > old {
			rank[p] = k
		}
	}
	for _, ln := range v.Lines {
		k := 1
		switch {
		case strings.HasSuffix(ln.Cls, " done"):
			k = 2
		case strings.HasSuffix(ln.Cls, " locked"):
			k = 0
		}
		x1, x2 := place(xs, ln.X1), place(xs, ln.X2)
		if x1 < 0 || x2 < 0 {
			continue
		}
		if x1 != x2 {
			r := row(ln.Y1)
			for x := min(x1, x2); x < max(x1, x2); x++ {
				if x%2 == 0 {
					mark(x/2, r, invui.KindLineE+invui.KindGapW, k)
				} else {
					mark(x/2, r, invui.KindGapE, k)
					mark(x/2+1, r, invui.KindLineW, k)
				}
			}
			continue
		}
		up, down := invui.KindLineN, invui.KindLineS
		if x1%2 == 1 {
			up, down = invui.KindGapN, invui.KindGapS
		}
		r1, r2 := row(ln.Y1), row(ln.Y2)
		for r := min(r1, r2); r < max(r1, r2); r++ {
			mark(x1/2, r, down, k)
			mark(x1/2, r+1, up, k)
		}
	}
	state := map[int]string{0: invui.KindLocked, 2: invui.KindDone}
	for p, k := range kinds {
		pg.mark(p.r*cols+left+p.c, "", k+state[rank[p]]+invui.KindCanvas)
	}
	for r := range ys {
		for c := -left; c < cols-left; c++ {
			if _, ok := kinds[at{c, r}]; !ok {
				pg.mark(r*cols+left+c, "", invui.KindCanvas)
			}
		}
	}
	for _, n := range v.Nodes {
		if n.Shape == ShapeVia {
			continue
		}
		p := at{place(xs, n.X) / 2, row(n.Y)}
		i := p.r*cols + left + p.c
		pg.cell(i, dSlot{ID: n.ID, Icon: n.Icon, Tip: n.Tip, Selected: n.Tracked}, invui.KindRow, invui.KindStatic)
		extra := kinds[p] + invui.KindCanvas
		switch n.Shape {
		case ShapeDiamond:
			extra += invui.KindDiamond
		case ShapeCircle:
			extra += invui.KindCircle
		case ShapeGate:
			extra += invui.KindGate
		}
		switch n.State {
		case QuestDone:
			extra += invui.KindDone
		case QuestLocked:
			extra += invui.KindLocked
		}
		if n.New {
			extra += invui.KindNew
		}
		slot := invui.Cell0 + i
		if !pg.items[slot].Empty() {
			cm := invui.CountMark("")
			pg.items[slot] = pg.items[slot].WithCustomName(cm + extra + strings.TrimPrefix(pg.items[slot].CustomName(), cm))
		}
	}
	return pg
}
