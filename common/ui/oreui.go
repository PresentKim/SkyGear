package ui

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/google/uuid"
	"github.com/presentkim/dragonfactory/common/oreui"
	"github.com/presentkim/dragonfactory/common/slotdeco"
)

// OreUI makes Menu (and Confirm), CardMenu, ItemBoard, SplitMenu,
// SieveBoard, ListSplitMenu, Input, ItemGrid, ItemGridSections and SlotMenu
// show as DDUI screens that the oreui pack draws in the game's Ore UI, like
// the dfui pack's JSON UI forms (packs/oreui/hbui/oui-mc.css): the same tabs,
// control bar, cards, list buttons, slot boards and split screens in one
// window, with the items drawn by the game and the window scrolling with a
// scrollbar. The server must send the oreui pack encrypted and wrap its
// listeners with dfhook. In the lobby the forms stay JSON UI (its frame is
// the dfui pack's). The other forms (Amount, tables, ...) stay JSON UI too;
// one of them closes the DDUI screen first.
var OreUI bool

// PreferJSON reports whether p chose the JSON UI forms over the DDUI screens
// (nil: nobody did). It is called from any goroutine.
var PreferJSON func(p *player.Player) bool

// OreUIFor reports whether forms for p show as DDUI screens.
func OreUIFor(p *player.Player) bool {
	return OreUI && (InLobby == nil || !InLobby(p)) && (PreferJSON == nil || !PreferJSON(p))
}

// oreScreen is the DDUI screen of a player: what its elements run, and gen,
// which counts the screens shown, so a press that shows none closes it (as a
// press answers a form).
type oreScreen struct {
	form *oreui.Form
	runs map[string]func(p *player.Player, tx *world.Tx)
	// input runs when the text of the screen's input box changes (SearchGrid).
	input func(p *player.Player, tx *world.Tx, value string)
	// values are the layout's input boxes' texts by their ids (oreLayout).
	values map[string]string
	gen    int
	// size is the window size of the layout shown.
	size string
}

var oreScreens = struct {
	sync.Mutex
	m map[uuid.UUID]*oreScreen
}{m: map[uuid.UUID]*oreScreen{}}

// showOre shows the layout l built to p: in the screen p has open if its
// window is as large, where the renderer draws again only the nodes that
// differ from the layout shown (a count, a selected slot, a row added), or
// else in a new one. Gameface hit-tests the presses
// after a window changes size at the places from before (a press on the
// codex's second tab after the research screen opened its third), so a
// window of another size is opened anew.
func showOre(p *player.Player, l *oreLayout, layout any) {
	if oreShown != nil {
		oreShown(l, layout)
		return
	}
	oreScreens.Lock()
	defer oreScreens.Unlock()
	st := oreScreens.m[p.UUID()]
	if st == nil {
		st = &oreScreen{}
		oreScreens.m[p.UUID()] = st
	}
	st.gen++
	st.runs, st.input, st.values = l.runs, l.input, l.values
	if st.form != nil && !st.form.Closed() {
		if st.size == l.size {
			_ = st.form.SetLayout(layout)
			return
		}
		_ = st.form.Close()
	}
	st.size = l.size
	h := p.H()
	f, err := oreui.Open(p, layout, func(ev oreui.Event) { oreEvent(h, st, ev) }, func() {
		oreScreens.Lock()
		if oreScreens.m[p.UUID()] == st {
			st.form = nil
		}
		oreScreens.Unlock()
	})
	if err != nil {
		st.form = nil
		return
	}
	st.form = f
}

// withScreen runs f, holding the lock, with the DDUI screen p has open (none:
// f does not run).
func withScreen(p *player.Player, f func(st *oreScreen)) {
	if oreShown != nil {
		return
	}
	oreScreens.Lock()
	defer oreScreens.Unlock()
	if st := oreScreens.m[p.UUID()]; st != nil && st.form != nil {
		f(st)
	}
}

// oreShown, when set (tests), takes the layouts showOre is given instead of
// the players' screens.
var oreShown func(l *oreLayout, layout any)

// oreEvent runs what the pressed element does in the player's world. Esc
// and the close button close the screen.
func oreEvent(h *world.EntityHandle, st *oreScreen, ev oreui.Event) {
	if ev.ID == "close" {
		oreScreens.Lock()
		if st.form != nil {
			_ = st.form.Close()
			st.form = nil
		}
		oreScreens.Unlock()
		return
	}
	oreScreens.Lock()
	for id, v := range ev.Inputs {
		if _, ok := st.values[id]; ok {
			st.values[id] = v
		}
	}
	oreScreens.Unlock()
	if ev.Kind == "change" {
		oreScreens.Lock()
		in, gen := st.input, st.gen
		if v, ok := ev.Value.(string); ok {
			if _, kept := st.values[ev.ID]; kept {
				st.values[ev.ID] = v
			}
		}
		oreScreens.Unlock()
		value, _ := ev.Value.(string)
		if in == nil {
			return
		}
		h.Do(func(tx *world.Tx, e world.Entity) {
			oreScreens.Lock()
			same := st.gen == gen
			oreScreens.Unlock()
			if same {
				p := e.(*player.Player)
				defer recoverForm(p)
				in(p, tx, value)
			}
		})
		return
	}
	if ev.Kind != "click" {
		return
	}
	oreScreens.Lock()
	run, gen := st.runs[ev.ID], st.gen
	oreScreens.Unlock()
	if run == nil {
		return
	}
	h.Do(func(tx *world.Tx, e world.Entity) {
		p := e.(*player.Player)
		func() {
			defer recoverForm(p)
			run(p, tx)
		}()
		oreScreens.Lock()
		if st.gen == gen && st.form != nil {
			_ = st.form.Close()
			st.form = nil
		}
		oreScreens.Unlock()
	})
}

// closeOre closes the DDUI screen of p, before a JSON UI form is sent.
func closeOre(p *player.Player) {
	oreScreens.Lock()
	defer oreScreens.Unlock()
	if st := oreScreens.m[p.UUID()]; st != nil && st.form != nil {
		_ = st.form.Close()
		st.form = nil
	}
}

// oreLayout builds a layout: the nodes of the renderer (packs/oreui/hbui/
// oui-ddui.js) and the runs of the elements that can be pressed.
type oreLayout struct {
	runs  map[string]func(p *player.Player, tx *world.Tx)
	input func(p *player.Player, tx *world.Tx, value string)
	// values keep the texts of the input boxes made by field, guarded by
	// oreScreens.
	values map[string]string
	// size is the window's size, set by window.
	size string
	// prefix starts the ids of the elements, so the parts drawn again into a
	// region do not reuse the ids of the screen around them.
	prefix string
}

func newOreLayout() *oreLayout {
	return &oreLayout{runs: map[string]func(p *player.Player, tx *world.Tx){}, values: map[string]string{}, prefix: "o"}
}

// field is an input box with the text def whose text the presses of the
// screen read with value. enter is the id of the button Enter presses ("":
// none).
func (l *oreLayout) field(id, def, placeholder, enter string) map[string]any {
	l.values[id] = def
	n := node("input", "id", id, "value", def, "class", "mc-input")
	if placeholder != "" {
		n["placeholder"] = placeholder
	}
	if enter != "" {
		n["enter"] = enter
	}
	return n
}

// value is the text of the input box id as the player left it.
func (l *oreLayout) value(id string) string {
	oreScreens.Lock()
	defer oreScreens.Unlock()
	return l.values[id]
}

// press gives a node an id that runs run when pressed (nil: not pressable).
func (l *oreLayout) press(n map[string]any, run func(p *player.Player, tx *world.Tx)) map[string]any {
	if run == nil {
		return n
	}
	id := l.prefix + strconv.Itoa(len(l.runs))
	l.runs[id] = run
	n["id"] = id
	return n
}

func node(t string, kv ...any) map[string]any {
	n := map[string]any{"t": t}
	for i := 0; i+1 < len(kv); i += 2 {
		n[kv[i].(string)] = kv[i+1]
	}
	return n
}

func box(class string, kv ...any) map[string]any {
	return node("box", append([]any{"class", class}, kv...)...)
}

func txt(class, text string) map[string]any { return node("text", "class", class, "text", text) }

func img(class, icon string) map[string]any {
	return node("img", "class", class, "icon", oreIcon(icon))
}

// gu is n GUI pixels in Ore UI units: the html font size is 5 x the GUI
// scale, so a JSON UI pixel is 0.2rem.
func gu(n int) string { return strconv.FormatFloat(float64(n)*0.2, 'f', -1, 64) + "rem" }

// oreIcon is the renderer's icon of a form icon: an ItemIcon (runtime id
// times 65536) is drawn by the game's item renderer, a texture path from the
// packs.
func oreIcon(icon string) string {
	switch {
	case icon == "":
		return ""
	case strings.HasPrefix(icon, "textures/"):
		return "tex:" + icon
	}
	if _, err := strconv.ParseInt(icon, 10, 64); err == nil {
		return "aux:" + icon
	}
	return icon
}

// tokens are the form and button markers of the JSON UI theme, which say
// where a thing goes; the DDUI screens take them off.
var tokens = []string{UntitledToken, LobbyToken, InputBackToken, CardToken, ShopToken, TableToken, SlotToken,
	DetailToken, WrapToken, ColumnsToken, SplitToken, CreateToken, TabToken, TabOnToken, BackToken, ActionToken,
	CellToken, CellNumToken, TotalToken, StoreToken, UpgradeToken, InfoToken, UtilToken, HelpToken, barPagerToken}

func untoken(s string) string {
	for again := true; again; {
		again = false
		for _, t := range tokens {
			if rest, ok := strings.CutPrefix(s, t); ok {
				s, again = rest, true
			}
		}
	}
	return s
}

// slotNode is an item slot: cell_image, the rarity behind it (the slotdeco
// marks a tooltip starts with), the chosen frame, the icon and the count.
func (l *oreLayout) slotNode(sl ItemSlot, extra ...string) map[string]any {
	tip, deco := slotDeco(sl.Tip)
	var ch []any
	if deco != "" {
		ch = append(ch, node("img", "class", "mc-deco", "icon", deco))
	}
	// slot_chosen has an opaque middle, so like in JSON UI it goes under the icon.
	ch = append(ch, node("img", "class", "mc-chosen", "icon", "tex:textures/ui/df/slot_chosen"))
	if sl.Icon != "" {
		ch = append(ch, img("mc-icon", sl.Icon))
	}
	if sl.Count != "" {
		ch = append(ch, node("text", "class", "mc-count", "text", sl.Count, "ink", "dark"))
	}
	class := strings.Join(append([]string{"mc-slot"}, extra...), " ")
	if sl.Selected {
		class += " on"
	}
	n := box(class, "children", ch, "tip", tip)
	if sl.Run != nil && !sl.Selected {
		n["class"] = class + " pressable"
		l.press(n, sl.Run)
	}
	return n
}

// slotDeco reads the rarity marks of a slot's tooltip ("§k§r§a" and the like,
// slotdeco): it returns the tooltip without the hidden part and the texture
// of the rarity.
func slotDeco(tip string) (string, string) {
	for _, d := range rarityDecos {
		if strings.HasPrefix(tip, d.mark) {
			return tip[len(d.mark)-len("§a"):], d.texture
		}
	}
	return tip, ""
}

type rarityDeco struct{ mark, texture string }

var rarityDecos = func() (decos []rarityDeco) {
	for r := slotdeco.Uncommon; r <= slotdeco.Legendary; r++ {
		decos = append(decos, rarityDeco{r.Mark(), fmt.Sprintf("tex:textures/ui/df/deco_%d_0", r)})
	}
	return decos
}()

func (l *oreLayout) button(class string, o Option) map[string]any {
	ch := []any{}
	if o.Icon != "" {
		ch = append(ch, img("", o.Icon))
	}
	ch = append(ch, txt("mc-btn-text", untoken(o.Label)))
	return l.press(box("mc-btn "+class, "ink", "dark", "children", ch), o.Run)
}

// listButton is df_form's list_button: 30u high, a 20u icon and the text.
func (l *oreLayout) listButton(o Option) map[string]any { return l.button("mc-list-row", o) }

// actionButton is the split screen's 17u action button.
func (l *oreLayout) actionButton(o Option) map[string]any { return l.button("mc-action", o) }

// card is df_form's card_button: 86x46 in a 90x50 cell.
func (l *oreLayout) card(o Option) map[string]any {
	return box("mc-card-cell", "children", []any{l.button("mc-card", o)})
}

// chrome sorts the buttons of a form: tabs over the window, the control bar
// (page arrows, help, back) and the rest.
func (l *oreLayout) chrome(opts []Option, wideTabs bool) (tabs, bar []any, rest []Option) {
	var pagers, help, back []any
	for _, o := range opts {
		switch {
		case strings.HasPrefix(o.Label, TabToken), strings.HasPrefix(o.Label, TabOnToken):
			tabs = append(tabs, l.tab(o, wideTabs))
		case strings.HasPrefix(o.Label, barPagerToken):
			pagers = append(pagers, quick(l.iconButton(o, o.Icon, "")))
		case strings.HasPrefix(o.Label, HelpToken):
			help = append(help, l.iconButton(o, "", "?"))
		case strings.HasPrefix(o.Label, BackToken):
			back = append(back, quick(l.iconButton(o, "textures/ui/df/chevron_left", "")))
		default:
			rest = append(rest, o)
		}
	}
	bar = append(append(pagers, help...), back...)
	return tabs, bar, rest
}

// quick makes a tap on n press it at once on a touch screen: its tooltip is
// only its name. A tap on another node with a tooltip shows the tooltip
// first and presses at the second tap (oui-ddui.js onPress), so the help
// text, an item's tooltip and the like can be read.
func quick(n map[string]any) map[string]any {
	n["quick"] = true
	return n
}

// iconButton is a 15x15 button of the control bar with an icon or a glyph;
// the label is its tooltip.
func (l *oreLayout) iconButton(o Option, icon, glyph string) map[string]any {
	var ch []any
	if icon != "" {
		ch = append(ch, img("", icon))
	} else {
		ch = append(ch, node("text", "text", glyph, "ink", "dark"))
	}
	n := box("mc-icon-button", "children", ch)
	if tip := untoken(o.Label); tip != "" {
		n["tip"] = tip
	}
	return l.press(n, o.Run)
}

// tab is a creative inventory tab (24x22) whose name is its tooltip; a wide
// tab (96x22) shows the name, like the split screen's.
func (l *oreLayout) tab(o Option, wide bool) map[string]any {
	on := strings.HasPrefix(o.Label, TabOnToken)
	ch := []any{box("mc-tab-bg"), box("mc-tab-front"), img("", o.Icon)}
	class := "mc-tab"
	n := box(class, "on", on)
	if wide {
		n["class"] = class + " mc-tab-wide"
		ch = append(ch, node("text", "class", "mc-tab-label", "text", untoken(o.Label), "ink", "dark"))
	} else {
		n["tip"] = untoken(o.Label)
		quick(n)
	}
	n["children"] = ch
	if !on {
		l.press(n, o.Run)
	}
	return n
}

// windowW by windowH is the size of every DDUI window in GUI pixels (but the
// quest book's): the vanilla inventory's, the largest that fits a phone
// (docs/tech/json-ui.md). One size for all lets a press go from any screen
// to any other in the screen open, where the renderer draws again only what
// differs; a window of another size has to be opened anew (showOre).
const windowW, windowH = 326, 166

// window is df_form's window: the tabs over its top left, the control bar
// over its top right (with the close button), the title and the content.
func (l *oreLayout) window(title string, tabs, bar []any, content any) any {
	w, h := windowW, windowH
	l.size = fmt.Sprintf("%dx%d", w, h)
	bar = append(bar, box("mc-close", "id", "close", "children", []any{node("img", "icon", "tex:textures/ui/close_button_default_light")}))
	var children []any
	if len(tabs) > 0 {
		children = append(children, box("mc-tabs", "children", tabs))
	}
	children = append(children,
		box("mc-toolbar", "children", []any{box("mc-toolbar-bg"), box("mc-toolbar-row", "children", bar)}),
		txt("mc-title", untoken(title)),
		box("mc-content", "children", []any{content}),
	)
	win := box("mc-window", "ink", "panel", "style", map[string]any{"width": gu(w), "height": gu(h)}, "children", children)
	return box("mc-screen", "children", []any{win})
}

func scroll(children ...any) map[string]any { return box("mc-scroll mc-col", "children", children) }

// oreMenu is a plain menu (df_form action_form): the body text and a
// list button per option.
func oreMenu(p *player.Player, title, body string, opts []Option) {
	l := newOreLayout()
	tabs, bar, rest := l.chrome(opts, false)
	content := []any{}
	if body != "" {
		content = append(content, txt("mc-body-text", body), box("mc-gap-6"))
	}
	for _, o := range rest {
		content = append(content, l.listButton(o))
	}
	showOre(p, l, l.window(title, tabs, bar, scroll(content...)))
}

// oreCardMenu is a card menu (df_form card_form): the body text and
// three cards to a row.
func oreCardMenu(p *player.Player, title, body string, opts []Option) {
	l := newOreLayout()
	tabs, bar, rest := l.chrome(opts, false)
	var cards []any
	for _, o := range rest {
		cards = append(cards, l.card(o))
	}
	content := []any{}
	if body != "" {
		content = append(content, txt("mc-body-text", body), box("mc-gap-6"))
	}
	content = append(content, box("mc-cards", "children", cards))
	if title == UntitledToken {
		title = ""
	}
	showOre(p, l, l.window(title, tabs, bar, scroll(content...)))
}

// boardRow is a row of board cells (BoardColumns to a row, 18u each).
func (l *oreLayout) boardRow(row []BoardCell) any {
	var cells []any
	at := func(i int) BoardCell {
		if i < len(row) {
			return row[i]
		}
		return BoardCell{}
	}
	for i := 0; i < BoardColumns; i++ {
		c := at(i)
		switch c.kind {
		case cellSlot:
			cells = append(cells, l.slotNode(c.slot))
		case cellMini:
			cells = append(cells, l.slotNode(c.slot, "mc-mini"))
		case cellFrame:
			cells = append(cells, box("mc-frame"))
		case cellArrow:
			cells = append(cells, box("mc-arrow", "children", []any{box("mc-arrow-img")}))
		case cellText:
			// The text runs over the blank cells after it, as in JSON UI, so
			// the cells after those stay in their columns.
			span := 1
			for i+span < BoardColumns && at(i+span).kind == cellBlank {
				span++
			}
			n := txt("mc-cell-text", c.text)
			n["style"] = map[string]any{"width": gu(18 * span), "flexShrink": "0"}
			cells = append(cells, n)
			i += span - 1
		default:
			cells = append(cells, box("mc-gap"))
		}
	}
	return box("mc-board-row", "children", cells)
}

// oreBoard is an ItemBoard (df_form's board): the body text, the
// rows and list buttons for the actions that are not tabs or bar buttons.
func oreBoard(p *player.Player, title, body string, rows [][]BoardCell, actions []Option) {
	l := newOreLayout()
	tabs, bar, rest := l.chrome(actions, false)
	var board []any
	if body != "" {
		board = append(board, txt("mc-body-text", body), box("mc-gap-3"))
	}
	for _, r := range rows {
		board = append(board, l.boardRow(r))
	}
	content := []any{box("mc-board", "children", board)}
	for _, o := range rest {
		content = append(content, box("mc-gap-3"), l.listButton(o))
	}
	showOre(p, l, l.window(title, tabs, bar, scroll(content...)))
}

// SieveRecipe is one recipe of a sieve: the block that goes in, the mesh and
// the drops.
type SieveRecipe struct {
	Input, Mesh ItemSlot
	Drops       []ItemSlot
}

// SieveGroup is the recipes of one sieve, under a heading with the sieve.
type SieveGroup struct {
	Head    ItemSlot
	Name    string
	Recipes []SieveRecipe
}

// SieveBoard shows rows of top (the blocks to pick) and under them the
// recipes of groups the way Ex Deorum's JEI sieve category does
// (compat/jei/SieveCategory): each recipe 162u wide, the input and the mesh
// slot at x 58 and 86 on top, the drops from y 28 in rows of 9 slots, as
// many rows for every recipe, empty slots drawn. The drops' counts are left
// off (their tooltips tell the chance). Without DDUI it is an ItemBoard of
// the mesh, an arrow and the drops.
func SieveBoard(p *player.Player, title string, top [][]BoardCell, groups []SieveGroup, actions ...Option) {
	if !OreUIFor(p) {
		rows := append([][]BoardCell(nil), top...)
		for _, g := range groups {
			for _, r := range g.Recipes {
				row := []BoardCell{SlotCell(r.Mesh), Arrow()}
				for _, d := range r.Drops {
					if len(row) == BoardColumns {
						rows, row = append(rows, row), []BoardCell{Blank(), Blank()}
					}
					row = append(row, SlotCell(d))
				}
				rows = append(rows, row)
			}
		}
		ItemBoard(p, title, "", rows, actions...)
		return
	}
	l := newOreLayout()
	tabs, bar, rest := l.chrome(actions, false)
	var board []any
	for _, r := range top {
		board = append(board, l.boardRow(r))
	}
	lines := 1
	for _, g := range groups {
		for _, r := range g.Recipes {
			lines = max(lines, (len(r.Drops)+8)/9)
		}
	}
	for _, g := range groups {
		board = append(board, box("mc-gap-6"), box("mc-board-row", "children", []any{l.slotNode(g.Head), txt("mc-cell-text", "§l"+g.Name)}), box("mc-gap-3"))
		for _, r := range g.Recipes {
			var cells []any
			for _, d := range r.Drops {
				d.Count = ""
				cells = append(cells, l.slotNode(d))
			}
			for len(cells) < lines*9 {
				cells = append(cells, box("mc-frame"))
			}
			board = append(board, box("mc-jei-recipe", "children", []any{
				box("mc-jei-top", "children", []any{l.slotNode(r.Input), l.slotNode(r.Mesh, "mc-jei-mesh")}),
				box("mc-jei-drops", "children", cells),
			}), box("mc-gap-6"))
		}
	}
	content := []any{box("mc-board", "children", board)}
	for _, o := range rest {
		content = append(content, box("mc-gap-3"), l.listButton(o))
	}
	showOre(p, l, l.window(title, tabs, bar, scroll(content...)))
}

// oreSplit is a SplitMenu (df_form split_form): wide tabs over the
// window, the grid of SplitCells slots with the page arrows under it on the
// left, and the chosen slot's detail on the right: the big slot, the name and
// status with the action buttons under them, the body text and the headed
// rows of slots.
func oreSplit(p *player.Player, title string, sp Split) {
	l := newOreLayout()
	tabs, bar, actions := l.chrome(append(append([]Option(nil), sp.Tabs...), sp.Actions...), true)
	var cells []any
	for i := range SplitCells {
		if i < len(sp.Slots) {
			cells = append(cells, l.slotNode(sp.Slots[i]))
		} else {
			cells = append(cells, box("mc-frame"))
		}
	}
	pages := max(1, sp.Pages)
	page := min(max(sp.Page, 0), pages-1)
	pager := func(on bool, icon string, to int) any {
		n := box("mc-btn", "children", []any{img("", icon)})
		if !on || sp.Turn == nil {
			n["class"] = "mc-btn off"
			return n
		}
		return l.press(n, func(p *player.Player, tx *world.Tx) { sp.Turn(p, tx, to) })
	}
	left := box("mc-panel mc-split-left", "children", []any{box("mc-col", "children", []any{
		box("mc-split-grid", "children", cells),
		box("mc-pager", "children", []any{
			pager(page > 0, "textures/ui/df/arrow_left", page-1),
			pager(page < pages-1, "textures/ui/df/arrow_right", page+1),
		}),
	})})
	head := []any{}
	for i, line := range strings.Split(sp.Detail.Name, "\n") {
		if i > 0 {
			head = append(head, box("mc-gap-3"))
		}
		head = append(head, txt("mc-small-text", line))
	}
	if sp.Detail.Status != "" {
		head = append(head, box("mc-gap-3"), txt("mc-small-text", sp.Detail.Status))
	}
	for _, a := range actions {
		head = append(head, box("mc-gap-3"), l.actionButton(a))
	}
	detail := []any{box("mc-row", "children", []any{
		l.slotNode(ItemSlot{Icon: sp.Detail.Icon}, "mc-big"),
		box("mc-col", "style", map[string]any{"marginLeft": gu(4), "flexGrow": "1"}, "children", head),
	})}
	if sp.Detail.Body != "" {
		detail = append(detail, box("mc-gap-3"), txt("mc-small-text", sp.Detail.Body))
	}
	for _, r := range sp.Rows {
		var slots []any
		for _, sl := range r.Slots {
			slots = append(slots, l.slotNode(sl))
		}
		detail = append(detail, box("mc-gap-3"))
		if r.Heading != "" {
			detail = append(detail, txt("mc-head", r.Heading))
		}
		detail = append(detail, box("mc-grid", "children", slots))
	}
	right := box("mc-panel mc-split-right", "children", []any{scroll(box("mc-col", "children", detail))})
	content := box("mc-row", "style", map[string]any{"height": "100%"}, "children", []any{left, right})
	showOre(p, l, l.window(title, tabs, bar, content))
}

// SearchResult is what a search shows for a query: a line about it and the
// headed sections of slots.
type SearchResult struct {
	Info     string
	Sections []GridSection
}

// SearchGrid shows a search window (df_form's board): an input box
// over the results, which are drawn again as the player types (in the
// screen, without opening the window again: the box keeps its text and
// focus). find makes the results of a query; it runs in the player's world
// and may record where the player is (the query to come back to).
// Without DDUI it shows the results of query as ItemGridSections, with a
// button that asks for another query in an input form (ask).
func SearchGrid(p *player.Player, title, label, query string, find func(p *player.Player, query string) SearchResult,
	ask func(p *player.Player, tx *world.Tx), actions ...Option) {
	if !OreUIFor(p) {
		r := find(p, query)
		ItemGridSections(p, title, r.Info, r.Sections, append([]Option{{Label: label, Run: ask}}, actions...)...)
		return
	}
	l := newOreLayout()
	tabs, bar, rest := l.chrome(actions, false)
	results := func(p *player.Player, query string, prefix string) (any, string, map[string]func(p *player.Player, tx *world.Tx)) {
		sub := newOreLayout()
		sub.prefix = prefix
		r := find(p, query)
		var parts []any
		for _, sec := range r.Sections {
			var slots []any
			for _, sl := range sec.Slots {
				slots = append(slots, sub.slotNode(sl))
			}
			if sec.Heading != "" {
				parts = append(parts, txt("mc-head", sec.Heading))
			}
			parts = append(parts, box("mc-grid", "children", slots), box("mc-gap-3"))
		}
		return box("mc-col", "children", parts), r.Info, sub.runs
	}
	first, info, runs := results(p, query, "r0_")
	for k, v := range runs {
		l.runs[k] = v
	}
	turn := 0
	l.input = func(p *player.Player, _ *world.Tx, value string) {
		turn++
		n, info, runs := results(p, value, "r"+strconv.Itoa(turn)+"_")
		withScreen(p, func(st *oreScreen) {
			for k, v := range runs {
				st.runs[k] = v
			}
			st.form.SetVars(map[string]any{"@results": n, "searchInfo": info})
		})
	}
	content := []any{
		box("mc-search", "children", []any{
			node("img", "icon", "tex:textures/ui/magnifyingGlass"),
			node("input", "id", "search", "value", query, "placeholder", label, "delay", 250, "class", "mc-input"),
		}),
		node("text", "class", "mc-body-text", "bind", "searchInfo"),
		box("mc-gap-3"),
		node("region", "name", "results"),
	}
	for _, o := range rest {
		content = append(content, box("mc-gap-3"), l.listButton(o))
	}
	showOre(p, l, l.window(title, tabs, bar, scroll(box("mc-board", "children", content))))
	withScreen(p, func(st *oreScreen) { st.form.SetVars(map[string]any{"@results": first, "searchInfo": info}) })
}
