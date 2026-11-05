package ui

import (
	"strconv"
	"strings"
	"sync"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/google/uuid"
	"github.com/presentkim/skygear/common/invclick"
)

// An inventory screen (ShowInventory) is a DDUI screen of item slots the
// player works like the vanilla inventory: the server holds the stack on the
// player's cursor (Carried), the page sends each click and drag on a slot
// (SlotAct, DragAct) and the game answers by changing its slots and showing
// the screen again. The page draws the parts row by row; a Grow row takes
// the height left and scrolls.

// InvSlot is a slot of an inventory screen.
type InvSlot struct {
	Icon string
	// Count is the text over the slot; "" leaves it empty.
	Count string
	Tip   string
	// Empty is the icon shown while the slot is empty.
	Empty string
	// Dim draws the item faded (a recipe that cannot be made now).
	Dim bool
	// Off draws a slot that is not there (the filler after a list): dark,
	// and it takes no presses.
	Off bool
}

// InvBook is the recipe book panel beside the window's left edge, laid out
// like the vanilla one: tabs over its top, the open tab's label (Title) and
// the filter switch, or on a search tab the search box and the switch, and a
// scrolling grid of recipe slots (area "book"). A Dim slot is a recipe that
// cannot be made now.
type InvBook struct {
	Tabs   []InvTab
	Title  string
	Search *InvSearch
	Filter *Option
	Slots  []InvSlot
	// Note is drawn in place of the slots when there are none.
	Note string
}

// InvTab is a tab of the recipe book; End puts it at the right end.
type InvTab struct {
	Label, Icon string
	On, End     bool
	Run         func(p *player.Player, tx *world.Tx)
}

// StackSlot is the slot of an item stack, with tip its tooltip.
func StackSlot(st item.Stack, tip string) InvSlot {
	if st.Empty() {
		return InvSlot{}
	}
	s := InvSlot{Icon: ItemIcon(st.Item()), Tip: tip}
	if st.Count() > 1 {
		s.Count = strconv.Itoa(st.Count())
	}
	return s
}

// InvPart is a part of a row: a block of slots Cols wide named Area (the
// slots' ids are Area:index), an arrow, or a small button.
type InvPart struct {
	Area  string
	Cols  int
	Slots []InvSlot
	// Big draws the slots like a result slot.
	Big bool
	// Arrow draws a crafting arrow instead of slots.
	Arrow bool
	// Button draws a small button (its label is the tooltip).
	Button *Option
	// Off draws the slots dark and takes no presses: what they would show
	// is not there (a network without power).
	Off bool
}

// InvRow is a row of parts under an optional heading.
type InvRow struct {
	Heading string
	Parts   []InvPart
	// Grow makes the row take the height the others leave and scroll.
	Grow bool
	// Note is a line drawn in place of the parts when they are empty.
	Note string
	// Bar is a column of buttons outside the window's right edge beside the
	// row (TransferBar): level with the row's bottom, or with its top when
	// BarTop is set.
	Bar    []Option
	BarTop bool
}

// TransferBar is the buttons that move items between a container row and
// the inventory below it, as the Java Edition inventory mods lay them out:
// the container's (take all, take the kinds held, sort) beside its bottom
// right, the inventory's (put all, put the kinds the container has, sort)
// beside its top right. A nil run leaves its button out.
type TransferBar struct {
	All, Matching, Sort Option
}

// Options are the bar's buttons with a run, in order.
func (b TransferBar) Options() []Option {
	var out []Option
	for _, o := range []Option{b.All, b.Matching, b.Sort} {
		if o.Run != nil {
			out = append(out, o)
		}
	}
	return out
}

// InvSearch is the search box over an inventory screen.
type InvSearch struct {
	Query, Placeholder string
	Change             func(p *player.Player, tx *world.Tx, query string)
}

// SlotAct is a press on a slot: the left button or the right (a long touch),
// with shift (or the page's quick move switch), or the second left press of
// a double click on the same slot (with the same shift state, the right
// button not pressed between). Split marks a press with a stack held, which
// a DragAct may follow (invclick.StartSplit).
type SlotAct struct {
	Area         string
	Index        int
	Right, Shift bool
	Double       bool
	Split        bool
	// Carried is the cursor stack when the press came.
	Carried item.Stack
}

// DragAct is the drag that follows a Split press, sent each time it reaches
// another slot: every distinct slot reached so far, the pressed one first,
// with the left button (spread evenly) or the right (one each).
type DragAct struct {
	Right bool
	Slots []SlotRef
}

// SlotRef is a slot of an inventory screen.
type SlotRef struct {
	Area  string
	Index int
}

// InventoryScreen is what ShowInventory draws.
type InventoryScreen struct {
	Title  string
	Search *InvSearch
	Info   string
	Rows   []InvRow
	// Side are the buttons in a column outside the window's right edge.
	Side []Option
	// Tools are square buttons in a column outside the window's left edge
	// (Refined Storage's side buttons: sort direction, sort kind, ...).
	Tools []Option
	// Book, if set, is the recipe book panel left of the window.
	Book *InvBook
	// Quick is the tooltip of the page's quick move switch under them,
	// which makes every click a shift click (for touch); "" leaves it out.
	Quick string
	// Options are the window's tabs and control bar buttons.
	Options []Option
	Slot    func(p *player.Player, tx *world.Tx, a SlotAct)
	Drag    func(p *player.Player, tx *world.Tx, d DragAct)
	// Hold, if set, hears when the player starts holding a mouse button, a
	// finger or shift on the screen (on) and when every one is let go.
	Hold func(p *player.Player, tx *world.Tx, on bool)
	// Again draws the screen again after a press outside the window dropped
	// the cursor stack (DropCarried).
	Again func(p *player.Player, tx *world.Tx)
	// Close runs when the player closes the screen, after the cursor stack
	// went back into the inventory.
	Close func(p *player.Player, tx *world.Tx)
	// Size names the window size; screens of one size switch in place.
	Size string
}

type dInvSlot struct {
	ID    string `json:"id"`
	Icon  string `json:"icon"`
	Count string `json:"count"`
	Tip   string `json:"tip"`
	Rare  int    `json:"rare"`
	Empty string `json:"empty"`
	Dim   bool   `json:"dim"`
	Off   bool   `json:"off"`
}

type dInvBook struct {
	Tabs        []dInvTab   `json:"tabs"`
	Title       string      `json:"title"`
	Field       string      `json:"field"`
	Query       string      `json:"query"`
	Placeholder string      `json:"placeholder"`
	Filter      *dBarButton `json:"filter"`
	Slots       []dInvSlot  `json:"slots"`
	Note        string      `json:"note"`
}

type dInvTab struct {
	ID   string `json:"id"`
	Tip  string `json:"tip"`
	Icon string `json:"icon"`
	On   bool   `json:"on"`
	End  bool   `json:"end"`
}

type dInvPart struct {
	Area   string      `json:"area"`
	Cols   int         `json:"cols"`
	Slots  []dInvSlot  `json:"slots"`
	Big    bool        `json:"big"`
	Arrow  bool        `json:"arrow"`
	Button *dBarButton `json:"button"`
	Off    bool        `json:"off"`
}

type dInvRow struct {
	Heading string       `json:"heading"`
	Parts   []dInvPart   `json:"parts"`
	Grow    bool         `json:"grow"`
	Note    string       `json:"note"`
	Bar     []dBarButton `json:"bar"`
	BarTop  bool         `json:"barTop"`
}

type inventoryData struct {
	Chrome      dChrome      `json:"chrome"`
	Field       string       `json:"field"`
	Query       string       `json:"query"`
	Placeholder string       `json:"placeholder"`
	Info        string       `json:"info"`
	Rows        []dInvRow    `json:"rows"`
	Side        []dBarButton `json:"side"`
	Tools       []dBarButton `json:"tools"`
	Quick       string       `json:"quick"`
	Book        *dInvBook    `json:"book"`
	Cursor      *dInvSlot    `json:"cursor"`
	Hold        bool         `json:"hold"`
}

// ShowInventory shows p an inventory screen. The cursor stack is drawn from
// Carried(p).
func ShowInventory(p *player.Player, s InventoryScreen) {
	l := newOreLayout()
	l.size = s.Size
	l.inventory = true
	l.close = func(p *player.Player, tx *world.Tx) {
		ReturnCarried(p)
		if s.Close != nil {
			s.Close(p, tx)
		}
	}
	l.slot = func(p *player.Player, tx *world.Tx, a SlotAct) {
		if s.Slot != nil {
			a.Carried = Carried(p)
			s.Slot(p, tx, a)
		}
	}
	l.drag = func(p *player.Player, tx *world.Tx, d DragAct) {
		if s.Drag != nil {
			s.Drag(p, tx, d)
		}
	}
	l.drop = func(p *player.Player, tx *world.Tx, one bool) {
		DropCarried(p, one)
		if s.Again != nil {
			s.Again(p, tx)
		}
	}
	l.hold = s.Hold
	chrome, _ := l.chromeOf(s.Title, s.Options)
	d := inventoryData{Chrome: chrome, Info: s.Info, Rows: []dInvRow{}, Side: []dBarButton{}, Tools: []dBarButton{}, Quick: untoken(s.Quick), Hold: s.Hold != nil}
	button := func(o Option) dBarButton {
		return dBarButton{ID: l.id(o.Run), Tip: untoken(o.Label), Icon: oreIcon(o.Icon), Quick: true}
	}
	for _, o := range s.Tools {
		d.Tools = append(d.Tools, button(o))
	}
	if s.Search != nil {
		d.Field, d.Query, d.Placeholder = "search", s.Search.Query, s.Search.Placeholder
		if s.Search.Change != nil {
			l.input = s.Search.Change
		}
	}
	for _, r := range s.Rows {
		dr := dInvRow{Heading: r.Heading, Parts: []dInvPart{}, Grow: r.Grow, Note: r.Note, Bar: []dBarButton{}, BarTop: r.BarTop}
		for _, o := range r.Bar {
			dr.Bar = append(dr.Bar, button(o))
		}
		for _, part := range r.Parts {
			dp := dInvPart{Area: part.Area, Cols: part.Cols, Slots: []dInvSlot{}, Big: part.Big, Arrow: part.Arrow, Off: part.Off}
			if part.Button != nil {
				b := dBarButton{ID: l.id(part.Button.Run), Tip: untoken(part.Button.Label), Icon: oreIcon(part.Button.Icon), Quick: true}
				dp.Button = &b
			}
			for i, sl := range part.Slots {
				dp.Slots = append(dp.Slots, invSlotData(part.Area+":"+strconv.Itoa(i), sl))
			}
			dr.Parts = append(dr.Parts, dp)
		}
		d.Rows = append(d.Rows, dr)
	}
	for _, o := range s.Side {
		d.Side = append(d.Side, dBarButton{ID: l.id(o.Run), Tip: untoken(o.Label), Icon: oreIcon(o.Icon), Quick: true})
	}
	if b := s.Book; b != nil {
		db := dInvBook{Tabs: []dInvTab{}, Title: untoken(b.Title), Slots: []dInvSlot{}, Note: b.Note}
		for _, t := range b.Tabs {
			tab := dInvTab{Tip: untoken(t.Label), Icon: oreIcon(t.Icon), On: t.On, End: t.End}
			if !t.On && t.Run != nil {
				tab.ID = l.id(t.Run)
			}
			db.Tabs = append(db.Tabs, tab)
		}
		if b.Search != nil {
			db.Field, db.Query, db.Placeholder = bookField, b.Search.Query, b.Search.Placeholder
			if b.Search.Change != nil {
				l.inputs[bookField] = b.Search.Change
			}
		}
		if b.Filter != nil {
			db.Filter = &dBarButton{ID: l.id(b.Filter.Run), Tip: untoken(b.Filter.Label), Icon: oreIcon(b.Filter.Icon), Quick: true}
		}
		for i, sl := range b.Slots {
			db.Slots = append(db.Slots, invSlotData("book:"+strconv.Itoa(i), sl))
		}
		d.Book = &db
	}
	if p == nil {
		showScreen(p, l, "inventory", d)
		return
	}
	if c := Carried(p); !c.Empty() {
		cs := invSlotData("", StackSlot(c, ""))
		d.Cursor = &cs
	}
	showScreen(p, l, "inventory", d)
}

// bookField is the id of the recipe book's search box.
const bookField = "booksearch"

func invSlotData(id string, sl InvSlot) dInvSlot {
	tip, rare := slotDeco(sl.Tip)
	return dInvSlot{ID: id, Icon: oreIcon(sl.Icon), Count: sl.Count, Tip: tip, Rare: int(rare), Empty: oreIcon(sl.Empty), Dim: sl.Dim, Off: sl.Off}
}

// parseSlotRef reads an Area:index slot id.
func parseSlotRef(id string) (SlotRef, bool) {
	i := strings.LastIndexByte(id, ':')
	if i <= 0 {
		return SlotRef{}, false
	}
	n, err := strconv.Atoi(id[i+1:])
	if err != nil || n < 0 {
		return SlotRef{}, false
	}
	return SlotRef{id[:i], n}, true
}

// slotEvent reads the value of a "slot" event: {b: 0|1, shift, dbl}.
func slotEvent(id string, v any) (SlotAct, bool) {
	ref, ok := parseSlotRef(id)
	if !ok {
		return SlotAct{}, false
	}
	m, _ := v.(map[string]any)
	b, _ := m["b"].(float64)
	shift, _ := m["shift"].(bool)
	dbl, _ := m["dbl"].(bool)
	split, _ := m["split"].(bool)
	return SlotAct{Area: ref.Area, Index: ref.Index, Right: b == 1, Shift: shift, Double: dbl, Split: split}, true
}

// dropEvent reads the value of a "drop" event: {b: 0|1}, right is one item.
func dropEvent(v any) bool {
	m, _ := v.(map[string]any)
	b, _ := m["b"].(float64)
	return b == 1
}

// DropCarried drops the cursor stack of p at its feet, or one item of it
// (a press outside the window with a stack held).
func DropCarried(p *player.Player, one bool) {
	st := Carried(p)
	if st.Empty() {
		return
	}
	n := st.Count()
	if one {
		n = 1
	}
	SetCarried(p, st.Grow(-n))
	p.Drop(st.Grow(n - st.Count()))
}

// dragEvent reads the value of a "drag" event: {b: 0|1, ids}. Repeated
// slots count once.
func dragEvent(v any) (DragAct, bool) {
	m, _ := v.(map[string]any)
	b, _ := m["b"].(float64)
	ids, _ := m["ids"].([]any)
	d := DragAct{Right: b == 1}
	seen := map[SlotRef]bool{}
	for _, raw := range ids {
		id, _ := raw.(string)
		if ref, ok := parseSlotRef(id); ok && !seen[ref] {
			seen[ref] = true
			d.Slots = append(d.Slots, ref)
		}
	}
	return d, len(d.Slots) > 0
}

var carried = struct {
	sync.Mutex
	m map[uuid.UUID]item.Stack
}{m: map[uuid.UUID]item.Stack{}}

// Carried is the stack on p's cursor in an inventory screen.
func Carried(p *player.Player) item.Stack {
	carried.Lock()
	defer carried.Unlock()
	return carried.m[p.UUID()]
}

// SetCarried puts st on p's cursor.
func SetCarried(p *player.Player, st item.Stack) {
	carried.Lock()
	defer carried.Unlock()
	if st.Empty() {
		delete(carried.m, p.UUID())
		return
	}
	carried.m[p.UUID()] = st
}

// ReturnCarried puts the stack on p's cursor back into its inventory and
// drops what does not fit at its feet. It must run in p's transaction.
func ReturnCarried(p *player.Player) {
	st := Carried(p)
	if st.Empty() {
		return
	}
	SetCarried(p, item.Stack{})
	inv := p.Inventory()
	if rest := invclick.Move(st, invclick.Hotbar(inv), invclick.Main(inv)); !rest.Empty() {
		p.Drop(rest)
	}
}
