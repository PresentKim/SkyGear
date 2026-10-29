package ui

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/google/uuid"
	"github.com/presentkim/skygear/common/oreui"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/presentkim/skygear/common/slotdeco"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

// The menus are DDUI screens drawn by the oreui pack. The server must send
// the oreui pack encrypted (the game takes the Ore UI page from an encrypted
// pack only) and wrap its listeners with sghook.

// LobbyHeader, when set, is the game's name and logo (a form icon) that the
// DDUI screens of the lobby draw at the top middle. The platform sets it.
var LobbyHeader func(p *player.Player) (title, logo string)

// lobbySkin restyles a layout for p in the lobby (.mc-lobby): the lobby
// island seen clear (through a shade behind a window), the game's name at the top middle, navy windows
// with light text. The main screen (HomeScreen) and the screens it opens have
// the same shade and name, so going between them changes only what is over
// the island.
func lobbySkin(p *player.Player, layout any) any {
	darkInk(layout)
	root, ok := layout.(map[string]any)
	if !ok {
		return layout
	}
	root["class"] = fmt.Sprint(root["class"]) + " mc-lobby"
	if touchDevice(p) {
		root["class"] = fmt.Sprint(root["class"]) + " mc-touch"
	}
	skin := []any{
		box("mc-lobby-preload mc-lobby-preload-hover"),
		box("mc-lobby-preload mc-lobby-preload-pressed"),
		box("mc-lobby-shade"),
	}
	if LobbyHeader != nil {
		title, logo := LobbyHeader(p)
		skin = append(skin, box("mc-lobby-head", "children", []any{img("mc-lobby-logo", logo), txt("mc-lobby-title", title)}))
	}
	kids, _ := root["children"].([]any)
	root["children"] = append(skin, kids...)
	return layout
}

// touchDevice reports whether p plays on a phone or tablet (Android, iOS,
// Fire OS), whose screen edges may hold a camera cutout or a system handle
// that takes touches: the lobby keeps its buttons larger and away from the
// edges there (.mc-touch).
func touchDevice(p *player.Player) bool {
	if p == nil {
		return false
	}
	c := sghook.Of(p)
	if c == nil {
		return false
	}
	switch c.ClientData().DeviceOS {
	case protocol.DeviceAndroid, protocol.DeviceIOS, protocol.DeviceFireOS:
		return true
	}
	return false
}

// darkInk turns the panel ink of a layout's nodes into the dark one.
func darkInk(n any) {
	switch v := n.(type) {
	case map[string]any:
		if v["ink"] == "panel" {
			v["ink"] = "dark"
		}
		for _, c := range v {
			darkInk(c)
		}
	case []any:
		for _, c := range v {
			darkInk(c)
		}
	}
}

// oreOpen reports whether p has a DDUI screen open.
func oreOpen(p *player.Player) bool {
	oreScreens.Lock()
	defer oreScreens.Unlock()
	st := oreScreens.m[p.UUID()]
	return st != nil && st.form != nil && !st.form.Closed()
}

// oreClosed runs what the closing of st does for the player of h: its close
// run, or Idle.
func oreClosed(h *world.EntityHandle, st *oreScreen) {
	oreScreens.Lock()
	run := st.close
	oreScreens.Unlock()
	if run == nil {
		oreIdle(h, true)
		return
	}
	h.Do(func(tx *world.Tx, e world.Entity) {
		p := e.(*player.Player)
		defer recoverForm(p)
		run(p, tx)
	})
}

// oreIdle runs Idle for the player of h, whose DDUI screen closed (closed:
// without a press), when no form or screen followed.
func oreIdle(h *world.EntityHandle, closed bool) {
	if Idle == nil {
		return
	}
	h.Do(func(tx *world.Tx, e world.Entity) {
		if p := e.(*player.Player); !FormOpen(p) {
			Idle(p, tx, "", closed)
		}
	})
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
	// close is the layout's run on closing (oreLayout.close).
	close func(p *player.Player, tx *world.Tx)
	// probe is the layout's run for the lines its test nodes report
	// (oreLayout.probe).
	probe func(p *player.Player, tx *world.Tx, id, line string)
	// loading is when the loading screen shown came up (zero: the screen is
	// not a loading screen, LoadingScreen).
	loading time.Time
}

var oreScreens = struct {
	sync.Mutex
	m map[uuid.UUID]*oreScreen
}{m: map[uuid.UUID]*oreScreen{}}

// showOre shows the layout l built to p, reusing p's open screen (the
// renderer redraws only changed nodes) when its window is the same size.
// Gameface hit-tests presses at the old places after a window resizes, so a
// window of another size is opened anew.
func showOre(p *player.Player, l *oreLayout, layout any) {
	if InLobby != nil && InLobby(p) {
		layout = lobbySkin(p, layout)
	}
	if root, ok := layout.(map[string]any); ok && len(l.keys) > 0 {
		root["keys"] = l.keys
	}
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
	st.runs, st.input, st.values, st.close, st.probe = l.runs, l.input, l.values, l.close, l.probe
	switch {
	case !l.loading:
		st.loading = time.Time{}
	case st.loading.IsZero() || st.form == nil || st.form.Closed():
		st.loading = time.Now()
	}
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
		oreClosed(h, st)
	})
	if err != nil {
		st.form = nil
		return
	}
	st.form = f
	if OreDebug(p) {
		f.SetVars(map[string]any{"__debug": true})
	}
}

// oreDebug holds the players whose DDUI screens show the renderer's debug
// line, by UUID.
var oreDebug sync.Map

// SetOreDebug shows or hides the debug line of p's DDUI screens: what the
// renderer knows of the screen and the fonts (packs/oreui/hbui/oui-ddui.js
// drawStatus), over the top of the screen, for a screenshot from a device.
func SetOreDebug(p *player.Player, on bool) {
	if on {
		oreDebug.Store(p.UUID(), true)
	} else {
		oreDebug.Delete(p.UUID())
	}
	withScreen(p, func(st *oreScreen) { st.form.SetVars(map[string]any{"__debug": on}) })
}

// OreDebug reports whether p's DDUI screens show the debug line.
func OreDebug(p *player.Player) bool {
	_, ok := oreDebug.Load(p.UUID())
	return ok
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
// and the close button run the layout's close run, or Idle, and close the
// screen when that showed no other screen.
func oreEvent(h *world.EntityHandle, st *oreScreen, ev oreui.Event) {
	if ev.ID == "close" {
		oreScreens.Lock()
		open, run, gen := st.form != nil, st.close, st.gen
		oreScreens.Unlock()
		if !open {
			return
		}
		h.Do(func(tx *world.Tx, e world.Entity) {
			p := e.(*player.Player)
			if run == nil {
				oreEnd(p, tx, st, gen, true)
				return
			}
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
	if ev.Kind == "probe" {
		oreScreens.Lock()
		run := st.probe
		oreScreens.Unlock()
		line, _ := ev.Value.(string)
		if run != nil {
			h.Do(func(tx *world.Tx, e world.Entity) { run(e.(*player.Player), tx, ev.ID, line) })
		}
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
		oreEnd(p, tx, st, gen, false)
	})
}

// oreEnd ends a press (closed: Esc) on the screen st of p that showed no other
// screen or form (st.gen is still gen): Idle runs while the screen is still
// open, so a screen it shows (the lobby's main screen) takes its place
// without closing it, and the screen closes when Idle showed none either.
func oreEnd(p *player.Player, tx *world.Tx, st *oreScreen, gen int, closed bool) {
	oreScreens.Lock()
	ended := st.gen == gen && st.form != nil
	oreScreens.Unlock()
	if !ended {
		return
	}
	if Idle != nil {
		func() {
			defer recoverForm(p)
			Idle(p, tx, "", closed)
		}()
	}
	oreScreens.Lock()
	if st.gen == gen && st.form != nil {
		_ = st.form.Close()
		st.form = nil
	}
	oreScreens.Unlock()
}

// oreLayout builds a layout: the renderer's nodes and the runs of the
// elements that can be pressed.
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
	// close, if set, runs when the player closes the screen (Esc, the close
	// button) instead of Idle.
	close func(p *player.Player, tx *world.Tx)
	// keys are the ids the screen's keys press, by Key (oreLayout.key).
	keys map[string]string
	// probe, if set, takes the lines the renderer's test nodes report
	// (ProbeScreen).
	probe func(p *player.Player, tx *world.Tx, id, line string)
	// loading marks a loading screen (LoadingScreen), which CloseLoading
	// closes.
	loading bool
}

// Key is a key of the keyboard a DDUI screen can answer: the arrow keys and
// WASD for the directions, Enter and the space bar for KeyOK. A key the
// screen does not answer moves and presses the keyboard selection
// (oui-ddui.js navKey).
type Key string

const (
	KeyUp    Key = "up"
	KeyDown  Key = "down"
	KeyLeft  Key = "left"
	KeyRight Key = "right"
	KeyOK    Key = "ok"
)

// keyOrder is the order the keys of a layout get their ids in.
var keyOrder = []Key{KeyUp, KeyDown, KeyLeft, KeyRight, KeyOK}

// key makes key k run run on the screen (nil: k is not answered).
func (l *oreLayout) key(k Key, run func(p *player.Player, tx *world.Tx)) {
	if run == nil {
		return
	}
	id := l.prefix + strconv.Itoa(len(l.runs))
	l.runs[id] = run
	if l.keys == nil {
		l.keys = map[string]string{}
	}
	l.keys[string(k)] = id
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

// alt gives a pressable node an id that runs run when it is pressed with the
// right mouse button (nil: the right button does nothing).
func (l *oreLayout) alt(n map[string]any, run func(p *player.Player, tx *world.Tx)) map[string]any {
	if run == nil {
		return n
	}
	id := l.prefix + strconv.Itoa(len(l.runs))
	l.runs[id] = run
	n["alt"] = id
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
// scale, so a GUI pixel is 0.2rem.
func gu(n int) string { return strconv.FormatFloat(float64(n)*0.2, 'f', -1, 64) + "rem" }

// oreIcon is the renderer's icon of an Option or slot icon: an ItemIcon (runtime id
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

// tokens are the title and button markers that say where a thing goes
// (chrome, CreateToken); the screens take them off the text.
var tokens = []string{CreateToken, TabToken, TabOnToken, BackToken, HelpToken, barPagerToken}

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
	// slot_chosen has an opaque middle, so it goes under the icon.
	ch = append(ch, node("img", "class", "mc-chosen", "icon", "tex:textures/ui/sg/slot_chosen"))
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
		decos = append(decos, rarityDeco{r.Mark(), fmt.Sprintf("tex:textures/ui/sg/deco_%d_0", r)})
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

// listButton is a list button: 30u high, a 20u icon and the text.
func (l *oreLayout) listButton(o Option) map[string]any { return l.button("mc-list-row", o) }

// actionButton is the split screen's 17u action button.
func (l *oreLayout) actionButton(o Option) map[string]any { return l.button("mc-action", o) }

// card is a card button: 72x46 in a 76x50 cell.
func (l *oreLayout) card(o Option) map[string]any {
	return box("mc-card-cell", "children", []any{l.button("mc-card", o)})
}

// chrome sorts the buttons of a screen: tabs over the window, the control bar
// (page arrows, help, back) and the rest.
func (l *oreLayout) chrome(opts []Option) (tabs, bar []any, rest []Option) {
	var pagers, help, back []any
	for _, o := range opts {
		switch {
		case strings.HasPrefix(o.Label, TabToken), strings.HasPrefix(o.Label, TabOnToken):
			tabs = append(tabs, l.tab(o))
		case strings.HasPrefix(o.Label, barPagerToken):
			pagers = append(pagers, quick(l.iconButton(o, o.Icon, "")))
		case strings.HasPrefix(o.Label, HelpToken):
			help = append(help, l.iconButton(o, "", "?"))
		case strings.HasPrefix(o.Label, BackToken):
			back = append(back, quick(l.iconButton(o, "textures/ui/sg/chevron_left", "")))
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

// tab is a creative inventory tab (24x22) whose name is its tooltip.
func (l *oreLayout) tab(o Option) map[string]any {
	on := strings.HasPrefix(o.Label, TabOnToken)
	n := box("mc-tab", "on", on, "tip", untoken(o.Label), "children", []any{box("mc-tab-bg"), box("mc-tab-front"), img("", o.Icon)})
	quick(n)
	if !on {
		l.press(n, o.Run)
	}
	return n
}

// windowW by windowH is the size of every DDUI window but the quest book's,
// in GUI pixels: the vanilla inventory's, the largest that fits a phone. One
// size lets showOre switch screens in place.
const windowW, windowH = 326, 166

// window is the window of a screen: the tabs over its top left, the control bar
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

// oreMenu is a plain menu: the body text and a list button per option.
func oreMenu(p *player.Player, title, body string, opts []Option) {
	l := newOreLayout()
	tabs, bar, rest := l.chrome(opts)
	content := []any{}
	if body != "" {
		content = append(content, txt("mc-body-text", body), box("mc-gap-6"))
	}
	for _, o := range rest {
		content = append(content, l.listButton(o))
	}
	showOre(p, l, l.window(title, tabs, bar, scroll(content...)))
}

// oreCardMenu is a card menu: the body text and four cards to a row.
func oreCardMenu(p *player.Player, title, body string, opts []Option) {
	l := newOreLayout()
	tabs, bar, rest := l.chrome(opts)
	var cards []any
	for _, o := range rest {
		cards = append(cards, l.card(o))
	}
	content := []any{}
	if body != "" {
		content = append(content, txt("mc-body-text", body), box("mc-gap-6"))
	}
	content = append(content, box("mc-cards", "children", cards))
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
			// The text runs over the blank cells after it, so the cells after
			// those stay in their columns.
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

// oreBoard is an ItemBoard: the body text, the rows and list buttons for the actions that are not tabs or bar buttons.
func oreBoard(p *player.Player, title, body string, rows [][]BoardCell, actions []Option) {
	l := newOreLayout()
	tabs, bar, rest := l.chrome(actions)
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
// recipes of groups laid out like Ex Deorum's JEI sieve category: input and
// mesh on top, drops in rows of 9 (counts in the tooltips).
func SieveBoard(p *player.Player, title string, top [][]BoardCell, groups []SieveGroup, actions ...Option) {
	l := newOreLayout()
	tabs, bar, rest := l.chrome(actions)
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

// oreSplit is a SplitMenu: icon tabs over the
// window, the grid of SplitCells slots with the page arrows under it on the
// left, and the chosen slot's detail on the right: the big slot, the name and
// status with the action buttons under them, the body text and the headed
// rows of slots.
func oreSplit(p *player.Player, title string, sp Split) {
	l := newOreLayout()
	tabs, bar, actions := l.chrome(append(append([]Option(nil), sp.Tabs...), sp.Actions...))
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
			pager(page > 0, "textures/ui/sg/arrow_left", page-1),
			pager(page < pages-1, "textures/ui/sg/arrow_right", page+1),
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

// SearchGrid shows an input box over the results of find, redrawn in place
// as the player types. find runs in the player's world.
func SearchGrid(p *player.Player, title, label, query string, find func(p *player.Player, query string) SearchResult, actions ...Option) {
	l := newOreLayout()
	tabs, bar, rest := l.chrome(actions)
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
