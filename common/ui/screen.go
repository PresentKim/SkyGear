package ui

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// The DDUI screens are drawn by the React page (web/oreui), which owns their
// layout: the server sends a screen name and its data
// (docs/plans/2026-10-30-react-screens.contract.md). The d* types below are
// that data, with the JSON names of web/oreui/src/protocol.ts; a slice in
// them is never nil, so it is sent as [] and not null.

// dChrome is the title, the tabs over the window and the control bar.
type dChrome struct {
	Title string       `json:"title"`
	Tabs  []dTab       `json:"tabs"`
	Bar   []dBarButton `json:"bar"`
}

// MarshalJSON sends the empty lists of c as [].
func (c dChrome) MarshalJSON() ([]byte, error) {
	if c.Tabs == nil {
		c.Tabs = []dTab{}
	}
	if c.Bar == nil {
		c.Bar = []dBarButton{}
	}
	type plain dChrome
	return json.Marshal(plain(c))
}

// dTab is a tab over the window; On is the selected one, which is not
// pressable.
type dTab struct {
	ID   string `json:"id"`
	Tip  string `json:"tip"`
	Icon string `json:"icon"`
	On   bool   `json:"on"`
}

// dBarButton is a button of the control bar: an icon or a glyph.
type dBarButton struct {
	ID    string `json:"id"`
	Tip   string `json:"tip"`
	Icon  string `json:"icon"`
	Glyph string `json:"glyph"`
	Quick bool   `json:"quick"`
}

// dButton is a button with a label and an icon.
type dButton struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Icon  string `json:"icon"`
}

// dItem is a list row; Toggle makes it a row with a switch, On its state.
// Kind "heading" or "divider" is a row that is no button.
type dItem struct {
	dButton
	Toggle bool   `json:"toggle,omitempty"`
	On     bool   `json:"on,omitempty"`
	Kind   string `json:"kind,omitempty"`
}

// dSlot is an item slot.
type dSlot struct {
	ID       string `json:"id"`
	Alt      string `json:"alt,omitempty"`
	Icon     string `json:"icon"`
	Count    string `json:"count"`
	Tip      string `json:"tip"`
	Deco     string `json:"deco"`
	Selected bool   `json:"selected"`
}

// dSlotRow is a heading with a row of slots.
type dSlotRow struct {
	Heading string  `json:"heading"`
	Slots   []dSlot `json:"slots"`
}

// MarshalJSON sends the empty list of r as [].
func (r dSlotRow) MarshalJSON() ([]byte, error) {
	if r.Slots == nil {
		r.Slots = []dSlot{}
	}
	type plain dSlotRow
	return json.Marshal(plain(r))
}

// dCell is a cell of a board row; the cells a text cell covers are left out
// and counted in its Span.
type dCell struct {
	K    string `json:"k"`
	Slot *dSlot `json:"slot,omitempty"`
	Text string `json:"text,omitempty"`
	Span int    `json:"span,omitempty"`
}

// dLobby is what the page draws over a screen in the lobby.
type dLobby struct {
	Title string `json:"title"`
	Logo  string `json:"logo"`
	Touch bool   `json:"touch"`
}

// id gives run an id that the page sends when the element is pressed (nil:
// "", not pressable). Ids of one layout never repeat.
func (l *oreLayout) id(run func(p *player.Player, tx *world.Tx)) string {
	if run == nil {
		return ""
	}
	id := l.prefix + strconv.Itoa(len(l.runs))
	l.runs[id] = run
	return id
}

// chromeOf sorts the options of a screen as chrome does: tab tokens to the
// tabs, the bar buttons (pagers, help, back, in that order) to the bar and
// the rest returned.
func (l *oreLayout) chromeOf(title string, opts []Option) (dChrome, []Option) {
	c := dChrome{Title: untoken(title), Tabs: []dTab{}, Bar: []dBarButton{}}
	var pagers, help, back []dBarButton
	var rest []Option
	for _, o := range opts {
		switch {
		case strings.HasPrefix(o.Label, TabToken), strings.HasPrefix(o.Label, TabOnToken):
			on := strings.HasPrefix(o.Label, TabOnToken)
			t := dTab{Tip: untoken(o.Label), Icon: oreIcon(o.Icon), On: on}
			if !on {
				t.ID = l.id(o.Run)
			}
			c.Tabs = append(c.Tabs, t)
		case strings.HasPrefix(o.Label, barPagerToken):
			pagers = append(pagers, dBarButton{ID: l.id(o.Run), Tip: untoken(o.Label), Icon: oreIcon(o.Icon), Quick: true})
		case strings.HasPrefix(o.Label, HelpToken):
			help = append(help, dBarButton{ID: l.id(o.Run), Tip: untoken(o.Label), Glyph: "?"})
		case strings.HasPrefix(o.Label, BackToken):
			back = append(back, dBarButton{ID: l.id(o.Run), Tip: untoken(o.Label), Icon: oreIcon("textures/ui/sg/chevron_left"), Quick: true})
		default:
			rest = append(rest, o)
		}
	}
	c.Bar = append(append(append(c.Bar, pagers...), help...), back...)
	return c, rest
}

// slotOf is the data of an item slot: its rarity and tooltip from slotDeco,
// and no id when it has no Run or is selected.
func (l *oreLayout) slotOf(sl ItemSlot) dSlot {
	tip, deco := slotDeco(sl.Tip)
	s := dSlot{Icon: oreIcon(sl.Icon), Count: sl.Count, Tip: tip, Deco: deco, Selected: sl.Selected}
	if !sl.Selected {
		s.ID = l.id(sl.Run)
	}
	return s
}

// buttonOf is the data of a button.
func (l *oreLayout) buttonOf(o Option) dButton {
	return dButton{ID: l.id(o.Run), Label: untoken(o.Label), Icon: oreIcon(o.Icon)}
}

// itemOf is the data of a list row.
func (l *oreLayout) itemOf(o Option) dItem {
	return dItem{dButton: l.buttonOf(o), Toggle: o.toggle, On: o.toggle && o.on, Kind: o.row}
}

// buttonsOf is the data of buttons, [] when there are none.
func (l *oreLayout) buttonsOf(opts []Option) []dButton {
	out := make([]dButton, 0, len(opts))
	for _, o := range opts {
		out = append(out, l.buttonOf(o))
	}
	return out
}

// itemsOf is the data of list rows, [] when there are none.
func (l *oreLayout) itemsOf(opts []Option) []dItem {
	out := make([]dItem, 0, len(opts))
	for _, o := range opts {
		out = append(out, l.itemOf(o))
	}
	return out
}

// rowsOf is the data of headed rows of slots, [] when there are none.
func (l *oreLayout) rowsOf(rows []SlotRow) []dSlotRow {
	out := make([]dSlotRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, dSlotRow{Heading: r.Heading, Slots: l.slotsOf(r.Slots)})
	}
	return out
}

// slotsOf is the data of slots, [] when there are none.
func (l *oreLayout) slotsOf(slots []ItemSlot) []dSlot {
	out := make([]dSlot, 0, len(slots))
	for _, s := range slots {
		out = append(out, l.slotOf(s))
	}
	return out
}

// cellsOf is the data of a board row: BoardColumns columns, a text cell
// covering the blank cells after it in its span.
func (l *oreLayout) cellsOf(row []BoardCell) []dCell {
	at := func(i int) BoardCell {
		if i < len(row) {
			return row[i]
		}
		return BoardCell{}
	}
	out := make([]dCell, 0, BoardColumns)
	for i := 0; i < BoardColumns; i++ {
		c := at(i)
		switch c.kind {
		case cellSlot, cellMini:
			s := l.slotOf(c.slot)
			k := "slot"
			if c.kind == cellMini {
				k = "mini"
			}
			out = append(out, dCell{K: k, Slot: &s})
		case cellFrame:
			out = append(out, dCell{K: "frame"})
		case cellArrow:
			out = append(out, dCell{K: "arrow"})
		case cellText:
			span := 1
			for i+span < BoardColumns && at(i+span).kind == cellBlank {
				span++
			}
			out = append(out, dCell{K: "text", Text: c.text, Span: span})
			i += span - 1
		default:
			out = append(out, dCell{K: "blank"})
		}
	}
	return out
}

// rootOf builds the root the page gets: the screen, its data and what
// goes around it.
func rootOf(name string, data any, keys map[string]string, lobby *dLobby, debug bool) map[string]any {
	root := map[string]any{"screen": name, "data": data}
	if len(keys) > 0 {
		root["keys"] = keys
	}
	if lobby != nil {
		root["lobby"] = lobby
	}
	if debug {
		root["debug"] = true
	}
	return root
}

// showScreen shows the screen name with data to p: like showOre, it reuses
// the open screen when the window has the same size.
func showScreen(p *player.Player, l *oreLayout, name string, data any) {
	if l.size == "" {
		l.size = fmt.Sprintf("%dx%d", windowW, windowH)
	}
	var lobby *dLobby
	if InLobby != nil && InLobby(p) {
		lobby = &dLobby{Touch: touchDevice(p)}
		if LobbyHeader != nil {
			title, logo := LobbyHeader(p)
			lobby.Title, lobby.Logo = title, oreIcon(logo)
		}
	}
	openOre(p, l, rootOf(name, data, l.keys, lobby, p != nil && OreDebug(p)))
}
