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
	// root is the root last sent, which SetOreDebug sends again.
	root any
}

var oreScreens = struct {
	sync.Mutex
	m map[uuid.UUID]*oreScreen
}{m: map[uuid.UUID]*oreScreen{}}

// openOre sends root (a screen's root) to p, reusing p's open screen when its
// window is the same size: it opens the screen or replaces the one open, and
// takes the layout's runs. Gameface hit-tests presses at the old places after
// a window resizes, so a window of another size is opened anew.
func openOre(p *player.Player, l *oreLayout, root any) {
	if oreShown != nil {
		oreShown(l, root)
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
	st.root = root
	switch {
	case !l.loading:
		st.loading = time.Time{}
	case st.loading.IsZero() || st.form == nil || st.form.Closed():
		st.loading = time.Now()
	}
	if st.form != nil && !st.form.Closed() {
		if st.size == l.size {
			_ = st.form.SetLayout(root)
			return
		}
		_ = st.form.Close()
	}
	st.size = l.size
	h := p.H()
	f, err := oreui.Open(p, root, func(ev oreui.Event) { oreEvent(h, st, ev) }, func() {
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
}

// oreDebug holds the players whose DDUI screens show the renderer's debug
// line, by UUID.
var oreDebug sync.Map

// SetOreDebug shows or hides the debug line of p's DDUI screens: what the
// page knows of the screen and the fonts, over the top of the screen, for a
// screenshot from a device. It applies to the screen open at once and to the
// screens shown after.
func SetOreDebug(p *player.Player, on bool) {
	if on {
		oreDebug.Store(p.UUID(), true)
	} else {
		oreDebug.Delete(p.UUID())
	}
	if oreShown != nil {
		return
	}
	oreScreens.Lock()
	defer oreScreens.Unlock()
	if st := oreScreens.m[p.UUID()]; st != nil && st.form != nil && !st.form.Closed() && st.root != nil {
		st.root = rootDebug(st.root, on)
		_ = st.form.SetLayout(st.root)
	}
}

// rootDebug is root with its debug line on or off.
func rootDebug(root any, on bool) any {
	m, ok := root.(map[string]any)
	if !ok {
		return root
	}
	out := make(map[string]any, len(m)+1)
	for k, v := range m {
		out[k] = v
	}
	if on {
		out["debug"] = true
	} else {
		delete(out, "debug")
	}
	return out
}

// OreDebug reports whether p's DDUI screens show the debug line.
func OreDebug(p *player.Player) bool {
	_, ok := oreDebug.Load(p.UUID())
	return ok
}

// oreShown, when set (tests), takes the roots showScreen is given instead of
// the players' screens.
var oreShown func(l *oreLayout, root any)

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
	if ev.Kind == "seen" {
		// A look at an element (QuestNode.Seen): its run records it and
		// shows no screen.
		oreScreens.Lock()
		run, gen := st.runs[ev.ID], st.gen
		oreScreens.Unlock()
		if run == nil {
			return
		}
		h.Do(func(tx *world.Tx, e world.Entity) {
			oreScreens.Lock()
			same := st.gen == gen
			oreScreens.Unlock()
			if same {
				p := e.(*player.Player)
				defer recoverForm(p)
				run(p, tx)
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

// oreLayout builds a screen: the runs of the elements that can be pressed
// and what the screen does on input and on closing.
type oreLayout struct {
	runs  map[string]func(p *player.Player, tx *world.Tx)
	input func(p *player.Player, tx *world.Tx, value string)
	// values keep the texts of the screen's input boxes, guarded by
	// oreScreens.
	values map[string]string
	// size is the window's size (showScreen sets it).
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
// (the page's keyboard handling).
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

// value is the text of the input box id as the player left it.
func (l *oreLayout) value(id string) string {
	oreScreens.Lock()
	defer oreScreens.Unlock()
	return l.values[id]
}

// node is a raw probe node of type t with the fields kv (ProbeNode).
func node(t string, kv ...any) map[string]any {
	n := map[string]any{"t": t}
	for i := 0; i+1 < len(kv); i += 2 {
		n[kv[i].(string)] = kv[i+1]
	}
	return n
}

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

// windowW by windowH is the size of every DDUI window but the quest book's,
// in GUI pixels: the vanilla inventory's, the largest that fits a phone. One
// size lets showOre switch screens in place.
const windowW, windowH = 326, 166

// menuData is the data of the menu screen: the body text and a row per option.
type menuData struct {
	Chrome dChrome `json:"chrome"`
	Body   string  `json:"body"`
	Items  []dItem `json:"items"`
}

// oreMenu is a plain menu: the body text and a list button per option.
func oreMenu(p *player.Player, title, body string, opts []Option) {
	l := newOreLayout()
	chrome, rest := l.chromeOf(title, opts)
	showScreen(p, l, "menu", menuData{Chrome: chrome, Body: body, Items: l.itemsOf(rest)})
}

// cardsData is the data of the cards screen: the body text and four cards to
// a row.
type cardsData struct {
	Chrome dChrome   `json:"chrome"`
	Body   string    `json:"body"`
	Items  []dButton `json:"items"`
}

// oreCardMenu is a card menu: the body text and four cards to a row.
func oreCardMenu(p *player.Player, title, body string, opts []Option) {
	l := newOreLayout()
	chrome, rest := l.chromeOf(title, opts)
	showScreen(p, l, "cards", cardsData{Chrome: chrome, Body: body, Items: l.buttonsOf(rest)})
}

// boardData is the data of the board screen: the body text, the rows of
// cells and the list rows for the actions.
type boardData struct {
	Chrome dChrome   `json:"chrome"`
	Body   string    `json:"body"`
	Rows   [][]dCell `json:"rows"`
	Items  []dItem   `json:"items"`
}

// rowsOfCells is the data of board rows, [] when there are none.
func (l *oreLayout) rowsOfCells(rows [][]BoardCell) [][]dCell {
	out := make([][]dCell, 0, len(rows))
	for _, r := range rows {
		out = append(out, l.cellsOf(r))
	}
	return out
}

// oreBoard is an ItemBoard: the body text, the rows and list buttons for the actions that are not tabs or bar buttons.
func oreBoard(p *player.Player, title, body string, rows [][]BoardCell, actions []Option) {
	l := newOreLayout()
	chrome, rest := l.chromeOf(title, actions)
	showScreen(p, l, "board", boardData{Chrome: chrome, Body: body, Rows: l.rowsOfCells(rows), Items: l.itemsOf(rest)})
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

// sieveRecipeData is a recipe of the sieve screen.
type sieveRecipeData struct {
	Input dSlot   `json:"input"`
	Mesh  dSlot   `json:"mesh"`
	Drops []dSlot `json:"drops"`
}

// sieveGroupData is the recipes of one sieve under its heading.
type sieveGroupData struct {
	Head    dSlot             `json:"head"`
	Name    string            `json:"name"`
	Recipes []sieveRecipeData `json:"recipes"`
}

// sieveData is the data of the sieve screen; Lines is the number of rows of
// nine drops every recipe is drawn with.
type sieveData struct {
	Chrome dChrome          `json:"chrome"`
	Top    [][]dCell        `json:"top"`
	Groups []sieveGroupData `json:"groups"`
	Lines  int              `json:"lines"`
	Items  []dItem          `json:"items"`
}

// SieveBoard shows rows of top (the blocks to pick) and under them the
// recipes of groups laid out like Ex Deorum's JEI sieve category: input and
// mesh on top, drops in rows of 9 (counts in the tooltips).
func SieveBoard(p *player.Player, title string, top [][]BoardCell, groups []SieveGroup, actions ...Option) {
	l := newOreLayout()
	chrome, rest := l.chromeOf(title, actions)
	d := sieveData{Chrome: chrome, Top: l.rowsOfCells(top), Groups: []sieveGroupData{}, Lines: 1}
	for _, g := range groups {
		for _, r := range g.Recipes {
			d.Lines = max(d.Lines, (len(r.Drops)+8)/9)
		}
	}
	for _, g := range groups {
		gd := sieveGroupData{Head: l.slotOf(g.Head), Name: "§l" + g.Name, Recipes: []sieveRecipeData{}}
		for _, r := range g.Recipes {
			drops := append([]ItemSlot(nil), r.Drops...)
			for i := range drops {
				drops[i].Count = ""
			}
			gd.Recipes = append(gd.Recipes, sieveRecipeData{Input: l.slotOf(r.Input), Mesh: l.slotOf(r.Mesh), Drops: l.slotsOf(drops)})
		}
		d.Groups = append(d.Groups, gd)
	}
	d.Items = l.itemsOf(rest)
	showScreen(p, l, "sieve", d)
}

// splitDetail is the chosen slot's card of the split screen.
type splitDetail struct {
	Icon   string `json:"icon"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Body   string `json:"body"`
}

// splitData is the data of the split screen: the grid of slots with the
// ids of the pager arrows ("": dimmed), and the detail of the chosen slot.
type splitData struct {
	Chrome  dChrome     `json:"chrome"`
	Slots   []dSlot     `json:"slots"`
	Prev    string      `json:"prev"`
	Next    string      `json:"next"`
	Detail  splitDetail `json:"detail"`
	Actions []dButton   `json:"actions"`
	Rows    []dSlotRow  `json:"rows"`
}

// oreSplit is a SplitMenu: icon tabs over the
// window, the grid of SplitCells slots with the page arrows under it on the
// left, and the chosen slot's detail on the right: the big slot, the name and
// status with the action buttons under them, the body text and the headed
// rows of slots.
func oreSplit(p *player.Player, title string, sp Split) {
	l := newOreLayout()
	chrome, actions := l.chromeOf(title, append(append([]Option(nil), sp.Tabs...), sp.Actions...))
	pages := max(1, sp.Pages)
	page := min(max(sp.Page, 0), pages-1)
	pager := func(on bool, to int) string {
		if !on || sp.Turn == nil {
			return ""
		}
		return l.id(func(p *player.Player, tx *world.Tx) { sp.Turn(p, tx, to) })
	}
	d := splitData{
		Chrome:  chrome,
		Slots:   l.slotsOf(sp.Slots[:min(len(sp.Slots), SplitCells)]),
		Prev:    pager(page > 0, page-1),
		Next:    pager(page < pages-1, page+1),
		Detail:  splitDetail{Icon: oreIcon(sp.Detail.Icon), Name: sp.Detail.Name, Status: sp.Detail.Status, Body: sp.Detail.Body},
		Actions: l.buttonsOf(actions),
		Rows:    l.rowsOf(sp.Rows),
	}
	showScreen(p, l, "split", d)
}

// SearchResult is what a search shows for a query: a line about it and the
// headed sections of slots.
type SearchResult struct {
	Info     string
	Sections []GridSection
}

// searchData is the data of the search screen: the input box with the
// query, a line about the results and the sections of slots.
type searchData struct {
	Chrome      dChrome    `json:"chrome"`
	Field       string     `json:"field"`
	Query       string     `json:"query"`
	Placeholder string     `json:"placeholder"`
	Info        string     `json:"info"`
	Sections    []dSlotRow `json:"sections"`
	Items       []dItem    `json:"items"`
}

// SearchGrid shows an input box over the results of find, shown again as
// the player types (a new layout of the same screen, so the slots of the new
// results press). find runs in the player's world.
func SearchGrid(p *player.Player, title, label, query string, find func(p *player.Player, query string) SearchResult, actions ...Option) {
	turn := 0
	var show func(p *player.Player, query, prefix string)
	show = func(p *player.Player, query, prefix string) {
		l := newOreLayout()
		l.prefix = prefix
		l.input = func(p *player.Player, _ *world.Tx, value string) {
			turn++
			show(p, value, "r"+strconv.Itoa(turn)+"_")
		}
		chrome, rest := l.chromeOf(title, actions)
		r := find(p, query)
		sections := make([]SlotRow, 0, len(r.Sections))
		for _, sec := range r.Sections {
			sections = append(sections, SlotRow{Heading: sec.Heading, Slots: sec.Slots})
		}
		showScreen(p, l, "search", searchData{
			Chrome: chrome, Field: "search", Query: query, Placeholder: label, Info: r.Info,
			Sections: l.rowsOf(sections), Items: l.itemsOf(rest),
		})
	}
	show(p, query, "o")
}
