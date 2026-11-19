package ui

import (
	"strconv"
	"strings"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/presentkim/skygear/common/slotdeco"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

// LobbyHeader, when set, is the game's name and logo (a form icon) that the
// screens of the lobby draw at the top middle. The platform sets it.
var LobbyHeader func(p *player.Player) (title, logo string)

// touchDevice reports whether p plays on a phone or tablet (Android, iOS,
// Fire OS), whose screen edges may hold a camera cutout or a system handle
// that takes touches.
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

// ShowingKey is the key of the screen p has open (InventoryScreen.Key), ""
// when none is open or it has no key. A screen that redraws itself while open
// (a machine's) checks it so it never replaces another.
func ShowingKey(p *player.Player) string {
	key, _ := formKey(p)
	return key
}

// oreLayout builds a screen: the runs of the elements that can be pressed
// and what the screen does on input and on closing.
type oreLayout struct {
	runs  map[string]func(p *player.Player, tx *world.Tx)
	input func(p *player.Player, tx *world.Tx, value string)
	// inputs run instead of input for the input boxes they are kept by.
	inputs map[string]func(p *player.Player, tx *world.Tx, value string)
	// values keep the texts of the screen's input boxes, guarded by
	// formScreens.
	values map[string]string
	// size is the window's size (showScreen sets it).
	size string
	// screenKey is the screen's key (ShowingKey).
	screenKey string
	// prefix starts the ids of the elements, so the parts drawn again into a
	// region do not reuse the ids of the screen around them.
	prefix string
	// close, if set, runs when the player closes the screen (Esc, the close
	// button) instead of Idle.
	close func(p *player.Player, tx *world.Tx)
	// keys are the ids the screen's keys press, by Key (oreLayout.key).
	keys map[string]string
	// loading marks a loading screen (LoadingScreen), which CloseLoading
	// closes.
	loading bool
	// inventory marks an inventory screen, which keeps the cursor stack
	// (Carried); any other screen gives it back.
	inventory bool
	slot      func(p *player.Player, tx *world.Tx, a SlotAct)
	drag      func(p *player.Player, tx *world.Tx, d DragAct)
	drop      func(p *player.Player, tx *world.Tx, one bool)
	hold      func(p *player.Player, tx *world.Tx, on bool)
}

// Key is a key of the keyboard a screen can answer: the arrow keys and WASD
// for the directions, Enter and the space bar for KeyOK.
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
	return &oreLayout{runs: map[string]func(p *player.Player, tx *world.Tx){}, inputs: map[string]func(p *player.Player, tx *world.Tx, value string){}, values: map[string]string{}, prefix: "o"}
}

// value is the text of the input box id as the player left it.
func (l *oreLayout) value(id string) string {
	formScreens.Lock()
	defer formScreens.Unlock()
	return l.values[id]
}

// oreIcon is the icon of an Option or slot icon: an ItemIcon (runtime id
// times 65536) is drawn as the item itself, a texture path from the packs.
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

// slotDeco reads the rarity mark of a slot's tooltip ("§k§r§a" and the like,
// slotdeco): it returns the tooltip without the hidden part and the rarity.
func slotDeco(tip string) (string, slotdeco.Rarity) {
	for r := slotdeco.Uncommon; r <= slotdeco.Legendary; r++ {
		if m := r.Mark(); strings.HasPrefix(tip, m) {
			return tip[len(m)-len("§a"):], r
		}
	}
	return tip, slotdeco.Common
}

// windowW by windowH is the size of every window but the quest book's, in
// GUI pixels: the vanilla inventory's, the largest that fits a phone.
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
	// Prompt asks for the query in an input form, for the menu window.
	Prompt dButton `json:"-"`
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
		prompt := l.buttonOf(Option{Label: label + "§7  " + query, Icon: "textures/ui/magnifyingGlass", Run: func(p *player.Player, _ *world.Tx) {
			Input(p, title, label, query, func(p *player.Player, _ *world.Tx, v string) {
				turn++
				show(p, v, "r"+strconv.Itoa(turn)+"_")
			}, func(p *player.Player, _ *world.Tx) {
				turn++
				show(p, query, "r"+strconv.Itoa(turn)+"_")
			})
		}})
		showScreen(p, l, "search", searchData{
			Chrome: chrome, Field: "search", Query: query, Placeholder: label, Info: r.Info,
			Sections: l.rowsOf(sections), Items: l.itemsOf(rest), Prompt: prompt,
		})
	}
	show(p, query, "o")
}
