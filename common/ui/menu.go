package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/google/uuid"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/invui"
)

// The screens are drawn in the menu window of invui: a screen is the items
// of its slots (menuPage) and the ids of the elements its slots press.

// menuPage is a screen as the slots of the menu window.
type menuPage struct {
	items []item.Stack
	ids   map[int]string
}

func newMenuPage(key string) *menuPage {
	pg := &menuPage{items: make([]item.Stack, invui.Size), ids: map[int]string{}}
	switch key[:2] {
	case invui.KeyMenu, invui.KeyCards, invui.KeyGrid, invui.KeyBoard, invui.KeyPick, invui.KeyResult:
		key += invui.KeyList
	}
	pg.text(invui.SlotKey, key)
	return pg
}

// text puts a paper named s in slot, or empties the slot when s is "".
func (pg *menuPage) text(slot int, s string) {
	if s == "" {
		pg.items[slot] = item.Stack{}
		return
	}
	pg.items[slot] = item.NewStack(item.Paper{}, 1).WithCustomName(s)
}

// button puts the button of an element in slot: its kind and text, pressing
// it presses id.
func (pg *menuPage) button(slot int, text, kind, id string) {
	pg.items[slot] = item.NewStack(item.Paper{}, 1).WithCustomName(kind + "§r" + i18n.Plain(text))
	if id != "" {
		pg.ids[slot] = id
	}
}

// icon puts an icon (oreIcon) in slot, named its face (invui.Face) and kind:
// the item itself for an item icon, a paper for a texture, and a paper of
// kind invui.KindBlank for no icon. The kind is the kind of the element the
// icon belongs to.
func (pg *menuPage) icon(slot int, icon, kind string) {
	pg.items[slot] = item.NewStack(item.Paper{}, 1).WithCustomName(invui.Face("") + invui.KindBlank + kind)
	switch {
	case strings.HasPrefix(icon, "tex:"):
		pg.items[slot] = item.NewStack(item.Paper{}, 1).WithCustomName(invui.Face(strings.TrimPrefix(icon, "tex:")) + kind)
	case strings.HasPrefix(icon, "aux:"):
		aux, err := strconv.ParseInt(strings.TrimPrefix(icon, "aux:"), 10, 64)
		if err != nil {
			return
		}
		it, ok := world.ItemByRuntimeID(int32(aux>>16), 0)
		if !ok {
			return
		}
		pg.items[slot] = item.NewStack(it, 1).WithCustomName(invui.Face("") + kind)
	}
}

// faces names the icon slot of every element with a button after the
// button: the pack draws the element from it (invui.Faces).
func (pg *menuPage) faces() {
	for _, f := range invui.Faces {
		for i := 0; i < f[2]; i++ {
			b := pg.items[f[1]+i]
			if b.Empty() {
				continue
			}
			if pg.items[f[0]+i].Empty() {
				pg.icon(f[0]+i, "", "")
			}
			icon := pg.items[f[0]+i]
			pg.items[f[0]+i] = icon.WithCustomName(invui.FaceName(icon.CustomName(), b.CustomName()))
		}
	}
}

func (pg *menuPage) chrome(c dChrome) {
	pg.text(invui.SlotTitle, "§r"+ink(c.Title))
	for i, t := range c.Tabs {
		if i >= invui.Tabs {
			break
		}
		kind := invui.KindTab
		if t.On {
			kind = invui.KindTabOn
		}
		pg.icon(invui.TabIcon0+i, t.Icon, kind)
		pg.button(invui.TabBtn0+i, t.Tip, kind, t.ID)
	}
	for i, b := range c.Bar {
		if i >= invui.Bars {
			break
		}
		kind := invui.KindBar
		if b.Glyph != "" {
			kind = invui.KindHelp
		}
		pg.icon(invui.BarIcon0+i, b.Icon, kind)
		pg.button(invui.BarBtn0+i, b.Tip, kind, b.ID)
	}
}

// dropBar takes the control bar button that presses id off the bar.
func (pg *menuPage) dropBar(id string) {
	for i := 0; i < invui.Bars; i++ {
		if pg.ids[invui.BarBtn0+i] == id {
			pg.items[invui.BarIcon0+i], pg.items[invui.BarBtn0+i] = item.Stack{}, item.Stack{}
			delete(pg.ids, invui.BarBtn0+i)
		}
	}
}

// rows puts list rows in the rows from first.
func (pg *menuPage) rows(first int, items []dItem) {
	for n, it := range items {
		i := first + n
		if i >= invui.Rows {
			break
		}
		kind := invui.KindRow + invui.FamRow
		text := it.Label
		switch {
		case it.Kind == "heading":
			kind, text = invui.KindHeading, ink(it.Label)
		case it.Kind == "divider":
			kind, text = invui.KindDivider, ""
		case it.Toggle && it.On:
			kind = invui.KindToggleOn + invui.FamRow
		case it.Toggle:
			kind = invui.KindToggleOff + invui.FamRow
		case it.ID == "":
			kind = invui.KindStatic + invui.FamRow
		}
		pg.icon(invui.RowIcon0+i, it.Icon, kind)
		pg.button(invui.RowBtn0+i, text, kind, it.ID)
	}
}

// buttons puts buttons with icons in the rows from first, with kind.
func (pg *menuPage) buttons(first int, kind string, bs []dButton) {
	for n, b := range bs {
		i := first + n
		if i >= invui.Rows {
			break
		}
		k := kind
		if b.ID == "" {
			k = invui.KindStatic
		}
		pg.icon(invui.RowIcon0+i, b.Icon, k)
		pg.button(invui.RowBtn0+i, b.Label, k, b.ID)
	}
}

// cell puts an item slot in cell i: the item itself, or a paper for a
// texture icon, named its count text and texture path (invui.CellFace), its
// kind and its tooltip. press and view are the kinds of a slot that can and
// cannot be pressed.
func (pg *menuPage) cell(i int, s dSlot, press, view string) {
	if i < 0 || i >= invui.Cells {
		return
	}
	slot := invui.Cell0 + i
	kind := press
	switch {
	case s.Selected:
		kind = invui.KindChosen
	case s.ID == "":
		kind = view
	}
	pressable := kind == press
	if press == invui.KindMini {
		kind += invui.FamMini
	} else {
		kind += invui.FamSlot
	}
	var it world.Item = item.Paper{}
	path := ""
	switch {
	case strings.HasPrefix(s.Icon, "aux:"):
		aux, err := strconv.ParseInt(strings.TrimPrefix(s.Icon, "aux:"), 10, 64)
		if err != nil {
			return
		}
		var ok bool
		if it, ok = world.ItemByRuntimeID(int32(aux>>16), 0); !ok {
			return
		}
	case strings.HasPrefix(s.Icon, "tex:"):
		path = strings.TrimPrefix(s.Icon, "tex:")
	}
	count, text := stackCount(it, s.Count)
	pg.items[slot] = item.NewStack(it, count).WithCustomName(invui.CellFace(text, path) + kind + "§r" + i18n.Plain(s.Tip))
	if pressable && s.ID != "" {
		pg.ids[slot] = s.ID
	}
}

// stackCount is the stack size and the count text (invui.CountMark) of a
// cell of it with count: a whole number the item stacks to is the stack's
// size, which the pack draws as the client does; any other count is text.
func stackCount(it world.Item, count string) (int, string) {
	n, err := strconv.Atoi(count)
	if err != nil || n < 2 {
		return 1, count
	}
	limit := 64
	if m, ok := it.(item.MaxCounter); ok {
		limit = m.MaxCount()
	}
	if n > limit {
		return 1, count
	}
	return n, ""
}

// mark puts a cell that draws only its kind (a frame, an arrow) or a text.
func (pg *menuPage) mark(i int, text, kind string) {
	if i < 0 || i >= invui.Cells {
		return
	}
	pg.items[invui.Cell0+i] = item.NewStack(item.Paper{}, 1).WithCustomName("§r" + text + kind)
}

// menuPageOf is the screen name with data as the slots of the menu window.
// A screen without its own layout shows its body and a row per element that
// can be pressed (pressRows).
func menuPageOf(name string, data any) *menuPage {
	switch v := data.(type) {
	case menuData:
		pg := newMenuPage(invui.KeyMenu + invui.KeyRows)
		pg.chrome(v.Chrome)
		pg.text(invui.SlotBody, "§r"+ink(v.Body))
		pg.rows(0, v.Items)
		return pg
	case cardsData:
		pg := newMenuPage(invui.KeyCards)
		pg.chrome(v.Chrome)
		pg.text(invui.SlotBody, "§r"+ink(v.Body))
		pg.buttons(0, invui.KindRow, v.Items)
		return pg
	case gridData:
		pg := newMenuPage(invui.KeyGrid + invui.KeyRows)
		pg.chrome(v.Chrome)
		pg.text(invui.SlotBody, "§r"+ink(v.Body))
		i := 0
		for _, sec := range v.Sections {
			if sec.Heading != "" {
				pg.mark(i, ink(sec.Heading), invui.KindHeading)
				i += invui.GridColumns
			}
			for _, s := range sec.Slots {
				pg.cell(i, s, invui.KindRow, invui.KindStatic)
				i++
			}
			i = (i + invui.GridColumns - 1) / invui.GridColumns * invui.GridColumns
		}
		pg.rows(0, v.Items)
		return pg
	case boardData:
		pg := newMenuPage(invui.KeyBoard + invui.KeyRows)
		pg.chrome(v.Chrome)
		pg.text(invui.SlotBody, "§r"+ink(v.Body))
		b := newBoard(pg, invui.BoardColumns)
		for _, row := range v.Rows {
			b.cells(row)
		}
		pg.rows(0, v.Items)
		return pg
	case splitData:
		pg := newMenuPage(invui.KeySplit)
		pg.chrome(v.Chrome)
		pg.text(invui.SlotBody, "§r"+ink(v.Detail.Body))
		pg.text(invui.SlotInfo, "§r"+ink(v.Detail.Status))
		for i := 0; i < invui.SplitCells; i++ {
			if i < len(v.Slots) {
				pg.cell(i, v.Slots[i], invui.KindRow, invui.KindStatic)
			} else {
				pg.mark(i, "", invui.KindFrame)
			}
		}
		pager := func(slot int, id string) {
			kind := invui.KindRow
			if id == "" {
				kind = invui.KindStatic
			}
			pg.button(slot, "", kind, id)
		}
		pager(invui.SlotPrev, v.Prev)
		pager(invui.SlotNext, v.Next)
		pg.icon(invui.RowIcon0, v.Detail.Icon, invui.KindStatic)
		pg.button(invui.RowBtn0, ink(v.Detail.Name), invui.KindStatic, "")
		pg.buttons(1, invui.KindRow, v.Actions[:min(len(v.Actions), invui.SplitActions)])
		i := invui.SplitCells
		for _, r := range v.Rows {
			if r.Heading != "" {
				pg.mark(i, ink(r.Heading), invui.KindHeading)
				i += invui.RightColumns
			}
			for _, s := range r.Slots {
				pg.cell(i, s, invui.KindRow, invui.KindStatic)
				i++
			}
			i = invui.SplitCells + (i-invui.SplitCells+invui.RightColumns-1)/invui.RightColumns*invui.RightColumns
		}
		return pg
	case listSplitData:
		pg := newMenuPage(invui.KeyListSplit)
		pg.chrome(v.Chrome)
		pg.text(invui.SlotBody, "§r"+ink(v.Body))
		for n, r := range v.Rows {
			if n >= invui.ListRows {
				break
			}
			kind := invui.KindRow
			if r.Selected {
				kind = invui.KindChosen
			}
			pg.icon(invui.RowIcon0+n, r.Icon, kind)
			pg.button(invui.RowBtn0+n, r.Name, kind, r.ID)
			if r.Info != "" {
				pg.mark(n, r.Info, invui.KindStatic)
			}
		}
		pg.buttons(invui.ListRows, invui.KindRow, v.Actions[:min(len(v.Actions), invui.ListActions)])
		return pg
	case slotMenuData:
		return slotMenuPage(v)
	case choiceData:
		return choicePage(v)
	case sieveData:
		return sievePage(v)
	case sidesData:
		return sidesPage(v)
	case pickData:
		return pickPage(v)
	case resultData:
		return resultPage(v)
	case equipData:
		return equipPage(v)
	case consoleData:
		return consolePage(v)
	case searchData:
		return searchPage(v)
	case loadingData:
		return loadingPage(v)
	case radialData:
		return radialPage(v)
	case planeData:
		return planePage(v)
	case questData:
		return questPage(v)
	case rangeData:
		return rangePage(v)
	case homeData:
		return homePage(v)
	}
	pg := newMenuPage(invui.KeyMenu + invui.KeyRows)
	chrome, body, rows := pressRows(data)
	pg.chrome(chrome)
	pg.text(invui.SlotBody, "§r"+ink(body))
	pg.rows(0, rows)
	return pg
}

// homePage is the lobby's main screen in the home slots (invui.Home*).
func homePage(v homeData) *menuPage {
	key := invui.KeyHome
	if v.Overlay != nil {
		key += invui.KeyOverlay
	}
	pg := newMenuPage(key)
	cell := func(i int, s string) {
		if s != "" {
			pg.items[invui.Cell0+i] = item.NewStack(item.Paper{}, 1).WithCustomName("§r" + i18n.Plain(s))
		}
	}
	row := func(i int, icon, label, kind, id string) {
		pg.icon(invui.RowIcon0+i, icon, kind)
		pg.button(invui.RowBtn0+i, label, kind, id)
	}
	pr := v.Profile
	row(invui.HomeProfileRow, pr.Icon, pr.Name, invui.KindRow, pr.ID)
	cell(invui.HomeTitleCell, pr.Title)
	cell(invui.HomeLineCell, pr.Line)
	pg.items[invui.Cell0+invui.HomeFillCell] = item.NewStack(item.Paper{}, 1).WithCustomName(fmt.Sprintf("w%02d", int(pr.Fill*invui.HomeFillSteps+0.5)))
	for i, b := range v.Corner {
		if i >= invui.HomeCorners {
			break
		}
		row(invui.HomeCornerRow0+i, b.Icon, b.Label, invui.KindRow, b.ID)
		if b.Badge > 0 {
			cell(invui.HomeBadgeCell0+i, strconv.Itoa(b.Badge))
		}
	}
	for i, b := range v.Side {
		if i >= invui.HomeSides {
			break
		}
		row(invui.HomeSideRow0+i, b.Icon, b.Label, invui.KindRow, b.ID)
		if b.Badge > 0 {
			cell(invui.HomeSideBadge0+i, strconv.Itoa(b.Badge))
		}
	}
	for i, line := range v.Feed {
		if i >= invui.HomeFeeds {
			break
		}
		cell(invui.HomeFeedCell0+i, line)
	}
	s := v.Save
	row(invui.HomeSaveRow, s.Icon, s.Name, invui.KindStatic, "")
	cell(invui.HomeHeadCell, s.Head)
	for i, line := range s.Lines {
		if i >= invui.HomeLines {
			break
		}
		cell(invui.HomeLineCell0+i, line)
	}
	if s.Arrows {
		for slot, id := range map[int]string{invui.SlotPrev: s.Prev, invui.SlotNext: s.Next} {
			kind := invui.KindRow
			if id == "" {
				kind = invui.KindStatic
			}
			pg.button(slot, "", kind, id)
		}
	}
	row(invui.HomePlayRow, "", v.Play.Label, invui.KindRow, v.Play.ID)
	cell(invui.HomeSubCell, v.Play.Sub)
	if m := v.Manage; m != nil {
		row(invui.HomeManageRow, "", m.Label, invui.KindRow, m.ID)
	}
	if o := v.Overlay; o != nil {
		cell(invui.HomeOverTitle, o.Title)
		cell(invui.HomeOverBody, o.Body)
		for i, b := range o.Buttons {
			if i >= invui.HomeOvers {
				break
			}
			row(invui.HomeOverRow0+i, "", b.Label, invui.KindRow, b.ID)
		}
	}
	return pg
}

// lobby draws the lobby's logo and name over the screen of pg.
func (pg *menuPage) lobby(l *dLobby) {
	pg.items[invui.SlotKey] = pg.items[invui.SlotKey].WithCustomName(pg.items[invui.SlotKey].CustomName() + invui.KeyLobby)
	pg.icon(invui.SlotLogo, l.Logo, invui.KindIcon)
	pg.text(invui.SlotGame, "§r"+i18n.Plain(l.Title))
}

// pressRows reads a screen's data: its chrome, its first texts and a list row
// per element that can be pressed outside the chrome, in the order of the
// data.
func pressRows(data any) (dChrome, string, []dItem) {
	raw, err := json.Marshal(data)
	if err != nil {
		return dChrome{}, "", nil
	}
	var top struct {
		Chrome dChrome `json:"chrome"`
	}
	_ = json.Unmarshal(raw, &top)
	chromeIDs := map[string]bool{}
	for _, t := range top.Chrome.Tabs {
		chromeIDs[t.ID] = true
	}
	for _, b := range top.Chrome.Bar {
		chromeIDs[b.ID] = true
	}
	var body []string
	var rows []dItem
	title := top.Chrome.Title
	walkJSON(json.NewDecoder(bytes.NewReader(raw)), func(obj map[string]any) {
		if t, ok := obj["title"].(string); ok && title == "" {
			title = t
		}
		for _, k := range []string{"body", "info", "line", "note", "headline"} {
			if s, ok := obj[k].(string); ok && s != "" && len(body) < 3 {
				body = append(body, s)
			}
		}
		id, _ := obj["id"].(string)
		if id == "" || chromeIDs[id] {
			return
		}
		label := id
		for _, k := range []string{"label", "name", "title", "tip"} {
			if s, ok := obj[k].(string); ok && s != "" {
				label = s
				break
			}
		}
		icon, _ := obj["icon"].(string)
		rows = append(rows, dItem{dButton: dButton{ID: id, Label: label, Icon: icon}})
	})
	top.Chrome.Title = title
	return top.Chrome, strings.Join(body, "\n"), rows
}

// walkJSON calls f with every object of the JSON value dec holds, the outer
// ones after the ones inside them, in their order in the text.
func walkJSON(dec *json.Decoder, f func(map[string]any)) {
	var walk func() any
	walk = func() any {
		tok, err := dec.Token()
		if err != nil {
			return nil
		}
		switch tok {
		case json.Delim('{'):
			obj := map[string]any{}
			for dec.More() {
				kt, err := dec.Token()
				if err != nil {
					return obj
				}
				k, _ := kt.(string)
				obj[k] = walk()
			}
			_, _ = dec.Token()
			f(obj)
			return obj
		case json.Delim('['):
			var arr []any
			for dec.More() {
				arr = append(arr, walk())
			}
			_, _ = dec.Token()
			return arr
		}
		return tok
	}
	walk()
}

// arrowOf is the step of an arrow slot, false for any other slot.
func arrowOf(slot int) (dx, dy int, ok bool) {
	switch slot {
	case invui.SlotUp:
		return 0, -1, true
	case invui.SlotDown:
		return 0, 1, true
	case invui.SlotLeft:
		return -1, 0, true
	case invui.SlotRight:
		return 1, 0, true
	}
	return 0, 0, false
}

// arrow is the id an arrow key presses: the cell of the screen's grid next
// to the chosen one (the first that can be pressed when none is chosen),
// or a split screen's page arrow past the grid's left or right edge.
func (pg *menuPage) arrow(dx, dy int) string {
	cols, n := 0, invui.Cells
	switch invui.LayoutOf(pg.items[invui.SlotKey].CustomName()) {
	case invui.KeySplit:
		cols, n = invui.SplitColumns, invui.SplitCells
	case invui.KeyGrid:
		cols = invui.GridColumns
	case invui.KeyBoard:
		cols = invui.BoardColumns
	default:
		return ""
	}
	at := -1
	for i := 0; i < n; i++ {
		if strings.Contains(pg.items[invui.Cell0+i].CustomName(), invui.KindChosen) {
			at = i
			break
		}
	}
	if at < 0 {
		for i := 0; i < n; i++ {
			if pg.ids[invui.Cell0+i] != "" {
				return pg.ids[invui.Cell0+i]
			}
		}
		return ""
	}
	x, y := at%cols+dx, at/cols+dy
	switch {
	case x < 0 && n == invui.SplitCells:
		return pg.ids[invui.SlotPrev]
	case x >= cols && n == invui.SplitCells:
		return pg.ids[invui.SlotNext]
	case x < 0 || x >= cols || y < 0 || y*cols+x >= n:
		return ""
	}
	return pg.ids[invui.Cell0+y*cols+x]
}

// closeID is the id of a button that closes the screen as Esc does.
const closeID = "\x00close"

// closeMenuOf closes the menu window of the player id in its world.
func closeMenuOf(id uuid.UUID) {
	formScreens.Lock()
	st := formScreens.m[id]
	var h *world.EntityHandle
	if st != nil {
		h = st.h
	}
	formScreens.Unlock()
	if h != nil {
		h.Do(func(_ *world.Tx, e world.Entity) { invui.Close(e.(*player.Player)) })
	}
}

// openMenu shows the screen of l in p's menu window. A press runs the
// element's run, then Idle when that showed no other screen, and the window
// closes when Idle showed none either; closing the window runs the layout's
// close run, or Idle.
func openMenu(p *player.Player, l *oreLayout, pg *menuPage) {
	closeForm(p)
	formScreens.Lock()
	st := formScreens.m[p.UUID()]
	if st == nil {
		st = &formScreen{}
		formScreens.m[p.UUID()] = st
	}
	st.gen++
	gen := st.gen
	st.key = l.screenKey
	st.menu = true
	st.h = p.H()
	st.loading = time.Time{}
	if l.loading {
		st.loading = time.Now()
	}
	formScreens.Unlock()

	current := func() bool {
		formScreens.Lock()
		defer formScreens.Unlock()
		return st.gen == gen
	}
	closeRun := l.close
	for _, s := range []int{invui.SlotEsc, invui.SlotUp, invui.SlotDown, invui.SlotLeft, invui.SlotRight} {
		pg.text(s, "§r")
	}
	pg.faces()
	invui.Show(p, pg.items, func(p *player.Player, tx *world.Tx, slot int) {
		if slot == invui.SlotEsc || pg.ids[slot] == closeID {
			if !current() {
				return
			}
			func() {
				defer recoverForm(p)
				if closeRun != nil {
					closeRun(p, tx)
				} else if Idle != nil {
					Idle(p, tx, "", true)
				}
			}()
			if current() {
				formScreens.Lock()
				st.menu = false
				formScreens.Unlock()
				invui.Close(p)
			}
			return
		}
		id := pg.ids[slot]
		if dx, dy, ok := arrowOf(slot); ok {
			id = pg.arrow(dx, dy)
		}
		run := l.runs[id]
		if !current() || run == nil {
			return
		}
		func() {
			defer recoverForm(p)
			run(p, tx)
		}()
		if !current() {
			return
		}
		if Idle != nil {
			func() {
				defer recoverForm(p)
				Idle(p, tx, "", false)
			}()
		}
		if current() {
			formScreens.Lock()
			st.menu = false
			formScreens.Unlock()
			invui.Close(p)
		}
	}, func(p *player.Player, tx *world.Tx) {
		if !current() {
			return
		}
		formScreens.Lock()
		st.menu = false
		formScreens.Unlock()
		defer recoverForm(p)
		if closeRun != nil {
			closeRun(p, tx)
			return
		}
		if Idle != nil {
			Idle(p, tx, "", true)
		}
	})
}
