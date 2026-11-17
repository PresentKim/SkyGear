package ui

import (
	"bytes"
	"encoding/json"
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
	case invui.KeyMenu, invui.KeyCards, invui.KeyGrid, invui.KeyBoard:
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

// button puts the button of an element in slot: its text and kind, pressing
// it presses id.
func (pg *menuPage) button(slot int, text, kind, id string) {
	pg.items[slot] = item.NewStack(item.Paper{}, 1).WithCustomName("§r" + i18n.Plain(text) + kind)
	if id != "" {
		pg.ids[slot] = id
	}
}

// icon puts an Ore UI icon in slot: the item itself for an item icon, a
// paper named the texture path for a texture. The kind is the kind of the
// element the icon belongs to.
func (pg *menuPage) icon(slot int, icon, kind string) {
	switch {
	case strings.HasPrefix(icon, "tex:"):
		pg.items[slot] = item.NewStack(item.Paper{}, 1).WithCustomName(strings.TrimPrefix(icon, "tex:") + kind)
	case strings.HasPrefix(icon, "aux:"):
		aux, err := strconv.ParseInt(strings.TrimPrefix(icon, "aux:"), 10, 64)
		if err != nil {
			return
		}
		it, ok := world.ItemByRuntimeID(int32(aux>>16), 0)
		if !ok {
			return
		}
		pg.items[slot] = item.NewStack(it, 1).WithCustomName(kind)
	}
}

func (pg *menuPage) chrome(c dChrome) {
	pg.text(invui.SlotTitle, "§r"+i18n.Plain(c.Title))
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

// cell puts an item slot in cell i: the item itself named with the
// tooltip, its count as the stack's when it is a number up to 64, and a
// texture icon as a paper named with the path (which has no tooltip). press
// and view are the kinds of a slot that can and cannot be pressed.
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
	count := 1
	if n, err := strconv.Atoi(s.Count); err == nil && n > 1 && n <= 64 {
		count = n
	}
	switch {
	case strings.HasPrefix(s.Icon, "aux:"):
		aux, err := strconv.ParseInt(strings.TrimPrefix(s.Icon, "aux:"), 10, 64)
		if err != nil {
			return
		}
		it, ok := world.ItemByRuntimeID(int32(aux>>16), 0)
		if !ok {
			return
		}
		pg.items[slot] = item.NewStack(it, count).WithCustomName("§r" + i18n.Plain(s.Tip) + kind)
	case strings.HasPrefix(s.Icon, "tex:"):
		pg.items[slot] = item.NewStack(item.Paper{}, count).WithCustomName(strings.TrimPrefix(s.Icon, "tex:") + kind)
	default:
		pg.items[slot] = item.NewStack(item.Paper{}, count).WithCustomName("§r" + i18n.Plain(s.Tip) + kind)
	}
	if pressable && s.ID != "" {
		pg.ids[slot] = s.ID
	}
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
		for r, row := range v.Rows {
			i := r * invui.BoardColumns
			pg.mark(i, "", invui.KindBlank)
			for _, c := range row {
				switch c.K {
				case "slot":
					pg.cell(i, *c.Slot, invui.KindRow, invui.KindStatic)
				case "mini":
					pg.cell(i, *c.Slot, invui.KindMini, invui.KindMiniView)
				case "frame":
					pg.mark(i, "", invui.KindFrame)
				case "arrow":
					pg.mark(i, "", invui.KindArrow)
				case "text":
					pg.mark(i, ink(c.Text), invui.KindText)
					i += max(c.Span, 1) - 1
				}
				i++
			}
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
		pg.buttons(1, invui.KindRow, v.Actions)
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
		pg.buttons(invui.ListRows, invui.KindRow, v.Actions)
		return pg
	}
	pg := newMenuPage(invui.KeyMenu + invui.KeyRows)
	chrome, body, rows := pressRows(data)
	pg.chrome(chrome)
	pg.text(invui.SlotBody, "§r"+ink(body))
	pg.rows(0, rows)
	return pg
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
	if !l.inventory {
		ReturnCarried(p)
	}
	closeOreScreen(p)
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
	invui.Show(p, pg.items, func(p *player.Player, tx *world.Tx, slot int) {
		run := l.runs[pg.ids[slot]]
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
