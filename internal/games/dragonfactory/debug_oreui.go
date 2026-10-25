package dragonfactory

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"

	"github.com/presentkim/dragonfactory/common/dfhook"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/common/oreui"
	"github.com/presentkim/dragonfactory/common/ui"
)

// Ore UI probes (/dfdebug oreinput, oreengine, orescreen, oreinv): screens
// that try in the client what an inventory drawn in DDUI needs. What the
// client does is shown on the screen and written to the server log; a part
// that does nothing in the client is a result too (docs/tech/oreui.md).

func oreNode(t string, kv ...any) map[string]any {
	n := map[string]any{"t": t}
	for i := 0; i+1 < len(kv); i += 2 {
		n[kv[i].(string)] = kv[i+1]
	}
	return n
}

func oreRow(children ...any) map[string]any {
	return oreNode("box", "style", map[string]any{"display": "flex", "flexDirection": "row", "alignItems": "center"}, "children", children)
}

func oreCol(children ...any) map[string]any {
	return oreNode("box", "style", map[string]any{"display": "flex", "flexDirection": "column"}, "children", children)
}

// probeScreen is a window of width x height (rem) on the darkened screen; a
// press on the screen around the window is the "outside" event.
func probeScreen(title, width, height string, content ...any) map[string]any {
	children := append([]any{
		oreRow(oreNode("text", "text", "§l"+title), oreNode("space"), oreNode("button", "id", "close", "text", "Close")),
	}, content...)
	win := oreNode("box", "class", "mc-window", "ink", "panel", "children", children,
		"style", map[string]any{"width": width, "height": height, "display": "flex", "flexDirection": "column"})
	return oreNode("box", "class", "mc-screen", "id", "outside", "self", true, "children", []any{win, oreNode("cursor", "bind", "cursor", "size", 32)})
}

func (g *Game) oreLog(p *player.Player, probe string, ev oreui.Event) {
	g.log.Info("ore ui probe", "probe", probe, "player", p.Name(), "id", ev.ID, "kind", ev.Kind, "value", ev.Value)
}

// openProbe opens layout for p and logs its events; onEvent, if not nil, runs
// for the events other than closing.
func (g *Game) openProbe(p *player.Player, probe string, layout any, onEvent func(oreui.Event), onClose func()) (*oreui.Form, error) {
	var f *oreui.Form
	f, err := oreui.Open(p, layout, func(ev oreui.Event) {
		g.oreLog(p, probe, ev)
		if ev.ID == "close" {
			_ = f.Close()
			if onClose != nil {
				onClose()
			}
			return
		}
		if onEvent != nil {
			onEvent(ev)
		}
	}, onClose)
	return f, err
}

type debugOreInput struct {
	adminBase
	Sub cmd.SubCommand `cmd:"oreinput"`
}

// Run opens a screen that lists every input event the client's page gets
// (buttons, double clicks, wheel, keys, touches, how long a press is held)
// and the engine command groups the page can call.
func (c debugOreInput) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p := src.(*player.Player)
	layout := probeScreen("/dfdebug oreinput", "64rem", "40rem",
		oreRow(oreNode("slot", "id", "raw-slot", "raw", true, "size", 32, "class", "mc-slot"), oreNode("button", "id", "button", "text", "Button"), oreNode("input", "id", "input", "placeholder", "keys")),
		oreNode("box", "class", "mc-scroll mc-col", "style", map[string]any{"height": "30rem"}, "children", []any{oreNode("probe")}),
	)
	if _, err := c.g.openProbe(p, "input", layout, nil, nil); err != nil {
		o.Errorf("oreinput: %v", err)
	}
}

// Containers of the vanilla Ore UI (the ContainerName enum of the gameplay
// bundle).
const (
	oreArmour    = 6
	oreLevel     = 7
	oreCombined  = 12
	oreHotbar    = 28
	oreInventory = 29
	oreOffhand   = 34
	oreCursor    = 59
)

// oreTraced holds, by connection, the probe screen that shows the packets the
// client sends while it is open.
var oreTraced sync.Map

func init() {
	dfhook.OnClientPacket(func(c *dfhook.Conn, pk packet.Packet) bool {
		v, ok := oreTraced.Load(c)
		if !ok {
			return false
		}
		switch pk.(type) {
		case *packet.PlayerAuthInput, *packet.NetworkStackLatency, *packet.ServerBoundDataStore, *packet.SubChunkRequest:
			return false
		}
		v.(func(packet.Packet))(pk)
		return false
	})
}

type debugOreEngine struct {
	adminBase
	Sub cmd.SubCommand `cmd:"oreengine"`
}

// Run opens a screen that reads the player's slots through the vanilla
// container queries and calls the vanilla container commands, with the
// arguments the vanilla Ore UI slots use, outside a container screen. The
// packets the client sends meanwhile are listed and logged.
func (c debugOreEngine) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p := src.(*player.Player)
	var slots [][2]int
	for i := range 9 {
		slots = append(slots, [2]int{oreHotbar, i})
	}
	for i := range 27 {
		slots = append(slots, [2]int{oreInventory, i})
	}
	for i := range 4 {
		slots = append(slots, [2]int{oreArmour, i})
	}
	slots = append(slots, [2]int{oreOffhand, 0}, [2]int{oreCursor, 0}, [2]int{oreCombined, 0}, [2]int{oreLevel, 0})
	call := func(label, name string, args ...any) map[string]any {
		return map[string]any{"label": label, "cmd": name, "args": args}
	}
	calls := []any{
		call("take all h0", "takeAllItems", oreCursor, 0, oreHotbar, 0),
		call("take half h0", "takeHalfItems", oreCursor, 0, oreHotbar, 0),
		call("place all h1", "placeAllItems", oreCursor, 0, oreHotbar, 1),
		call("place one h2", "placeOneItem", oreCursor, 0, oreHotbar, 2),
		call("auto place h0", "autoPlaceItems", oreHotbar, 0),
		call("coalesce h0", "coalesceOrAutoPlaceItems", oreHotbar, 0),
		call("drop cursor", "dropAllItems", oreCursor, 0),
		call("press L h0", "handleSlotMousePress", 0, oreHotbar, 0, false, false),
		call("release L h0", "handleSlotMouseRelease", 0, oreHotbar, 0),
		call("press R h0", "handleSlotMousePress", 2, oreHotbar, 0, false, false),
		call("release R h0", "handleSlotMouseRelease", 2, oreHotbar, 0),
		call("drag L h1", "handleSlotMouseDrag", 0, oreHotbar, 1),
		call("report", "__report"),
	}
	layout := probeScreen("/dfdebug oreengine", "64rem", "40rem",
		oreNode("text", "bind", "packets", "text", "packets: -"),
		oreNode("box", "class", "mc-scroll mc-col", "style", map[string]any{"height": "30rem"}, "children", []any{
			oreNode("engine", "id", "engine", "slots", slots, "sizes", []int{oreHotbar, oreInventory, oreArmour, oreOffhand, oreCursor, oreCombined, oreLevel}, "calls", calls),
		}),
	)
	conn := dfhook.Of(p)
	var (
		mu   sync.Mutex
		seen []string
	)
	untrace := func() {
		if conn != nil {
			oreTraced.Delete(conn)
		}
	}
	f, err := c.g.openProbe(p, "engine", layout, nil, untrace)
	if err != nil {
		o.Errorf("oreengine: %v", err)
		return
	}
	if conn != nil {
		oreTraced.Store(conn, func(pk packet.Packet) {
			// Another screen replacing this one does not run onClose.
			if f.Closed() {
				untrace()
				return
			}
			name := strings.TrimPrefix(fmt.Sprintf("%T", pk), "*packet.")
			c.g.log.Info("ore ui probe packet", "player", p.Name(), "packet", name, "body", fmt.Sprintf("%+v", pk))
			mu.Lock()
			seen = append(seen, name)
			if len(seen) > 8 {
				seen = seen[len(seen)-8:]
			}
			text := "packets: " + strings.Join(seen, ", ")
			mu.Unlock()
			f.SetVars(map[string]any{"packets": text})
		})
	}
}

// oreScreenIDs are the DDUI screens the game knows: the roots of the vanilla
// pack and the layouts in the gameplay bundle.
var oreScreenIDs = []string{
	"minecraft:inventory_screen", "minecraft:workbench_screen", "minecraft:furnace_screen", "minecraft:blast_furnace_screen",
	"minecraft:smoker_screen", "minecraft:anvil_screen", "minecraft:grindstone_screen", "minecraft:brewing_stand_screen",
	"minecraft:hopper_screen", "minecraft:dispenser_screen", "minecraft:dropper_screen", "minecraft:chest_screen",
	"minecraft:barrel_screen", "minecraft:copper_chest_screen", "minecraft:default_chest_screen", "minecraft:ender_chest_screen",
	"minecraft:shulker_box_screen", "minecraft:base_layout", "minecraft:custom_form", "minecraft:message_box",
}

type oreScreenEnum string

func (oreScreenEnum) Type() string { return "df_orescreen" }
func (oreScreenEnum) Options(cmd.Source) []string {
	return append([]string{"close"}, oreScreenIDs...)
}

type debugOreScreen struct {
	adminBase
	Sub    cmd.SubCommand     `cmd:"orescreen"`
	Screen oreScreenEnum      `cmd:"screen"`
	Fill   cmd.Optional[bool] `cmd:"fill"`
}

// Run opens a DDUI screen of the game by its ID, drawn by the game itself.
// With fill, data store minecraft / container first gets the player's
// inventory in the shape the bundle's container layouts read; "close" closes
// every DDUI screen.
func (c debugOreScreen) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p := src.(*player.Player)
	if c.Screen == "close" {
		if err := oreui.CloseAll(p); err != nil {
			o.Errorf("orescreen: %v", err)
		}
		return
	}
	if fill, _ := c.Fill.Load(); fill {
		if err := oreui.SetStore(p, "container", oreContainerData(p)); err != nil {
			o.Errorf("orescreen: %v", err)
			return
		}
	}
	id, err := oreui.ShowScreen(p, string(c.Screen))
	if err != nil {
		o.Errorf("orescreen: %v", err)
		return
	}
	c.g.log.Info("ore ui probe screen", "player", p.Name(), "screen", c.Screen, "form", id)
	o.Printf("opened %s (form %d)", c.Screen, id)
}

// oreContainerData is the player's inventory as data store minecraft /
// container: slots by the bundle's collection names, an item as {id, name,
// amount, details}.
func oreContainerData(p *player.Player) map[string]any {
	lang := i18n.Of(p)
	entry := func(s item.Stack) any {
		if s.Empty() {
			return map[string]any{"id": "", "name": "", "amount": 0, "details": ""}
		}
		name, _ := s.Item().EncodeItem()
		return map[string]any{"id": strings.TrimPrefix(name, "minecraft:"), "name": itemName(lang, s), "amount": s.Count(), "details": ""}
	}
	list := func(stacks []item.Stack) []any {
		out := make([]any, len(stacks))
		for i, s := range stacks {
			out[i] = entry(s)
		}
		return out
	}
	inv := p.Inventory().Slots()
	_, off := p.HeldItems()
	return map[string]any{
		"name":      "/dfdebug orescreen",
		"colorMode": "dark",
		"slots": map[string]any{
			"hotbarContainer":                     list(inv[:9]),
			"inventoryContainer":                  list(inv[9:]),
			"combinedHotbarAndInventoryContainer": list(inv),
			"armorContainer":                      list(p.Armour().Slots()),
			"offhandContainer":                    list([]item.Stack{off}),
			"levelEntityContainer":                list(inv[9:]),
			"craftingInputContainer":              list(make([]item.Stack, 4)),
			"craftingOutputPreviewContainer":      list(make([]item.Stack, 1)),
			"cursorContainer":                     list(make([]item.Stack, 1)),
		},
	}
}

type debugOreInv struct {
	adminBase
	Sub cmd.SubCommand `cmd:"oreinv"`
}

// Run opens the player's inventory drawn by the renderer, played by the
// server like the vanilla one: the item on the cursor, left and right button,
// shift, double click, splitting by dragging, number keys, Q, dropping
// outside the window.
func (c debugOreInv) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p := src.(*player.Player)
	st := &oreInv{g: c.g}
	h := p.H()
	f, err := c.g.openProbe(p, "inventory", st.layout(p), func(ev oreui.Event) {
		h.Do(func(_ *world.Tx, e world.Entity) { st.event(e.(*player.Player), ev) })
	}, func() {
		h.Do(func(_ *world.Tx, e world.Entity) { st.giveBack(e.(*player.Player)) })
	})
	if err != nil {
		o.Errorf("oreinv: %v", err)
		return
	}
	st.form = f
	st.refresh(p)
	st.watch(h)
}

// watch gives the cursor item back once the screen is gone: another screen
// replacing it does not run onClose.
func (st *oreInv) watch(h *world.EntityHandle) {
	h.DoAfter(time.Second, func(_ *world.Tx, e world.Entity) {
		if st.form.Closed() {
			st.giveBack(e.(*player.Player))
			return
		}
		st.watch(h)
	})
}

// oreInv is the state of an /dfdebug oreinv screen, used in the player's
// transactions only.
type oreInv struct {
	g      *Game
	form   *oreui.Form
	cursor item.Stack
	// placing is a press made with an item on the cursor: the item goes down
	// when the button is let go, on the slots it passed over.
	placing bool
	info    string
	sent    map[string]string
	shown   string
}

// oreSlot is a slot of the screen: h0-h8 the hotbar, i0-i26 the rest of the
// inventory, a0-a3 the armour, o0 the off hand.
type oreSlot struct {
	kind byte
	i    int
}

func (s oreSlot) id() string { return fmt.Sprintf("%c%d", s.kind, s.i) }

func parseOreSlot(id string) (oreSlot, bool) {
	var s oreSlot
	if len(id) < 2 {
		return s, false
	}
	s.kind = id[0]
	if _, err := fmt.Sscan(id[1:], &s.i); err != nil {
		return s, false
	}
	limit := map[byte]int{'h': 9, 'i': 27, 'a': 4, 'o': 1}[s.kind]
	return s, s.i >= 0 && s.i < limit
}

func oreSlots() []oreSlot {
	var out []oreSlot
	for i := range 4 {
		out = append(out, oreSlot{'a', i})
	}
	out = append(out, oreSlot{'o', 0})
	for i := range 27 {
		out = append(out, oreSlot{'i', i})
	}
	for i := range 9 {
		out = append(out, oreSlot{'h', i})
	}
	return out
}

func (s oreSlot) get(p *player.Player) item.Stack {
	var st item.Stack
	switch s.kind {
	case 'h':
		st, _ = p.Inventory().Item(s.i)
	case 'i':
		st, _ = p.Inventory().Item(9 + s.i)
	case 'a':
		st, _ = p.Armour().Inventory().Item(s.i)
	case 'o':
		_, st = p.HeldItems()
	}
	return st
}

func (s oreSlot) set(p *player.Player, st item.Stack) bool {
	if st.Count() <= 0 {
		st = item.Stack{}
	}
	switch s.kind {
	case 'h':
		return p.Inventory().SetItem(s.i, st) == nil
	case 'i':
		return p.Inventory().SetItem(9+s.i, st) == nil
	case 'a':
		return p.Armour().Inventory().SetItem(s.i, st) == nil
	case 'o':
		main, _ := p.HeldItems()
		p.SetHeldItems(main, st)
		return true
	}
	return false
}

// accepts reports whether st may go in s: an armour slot takes its kind of
// armour only.
func (s oreSlot) accepts(st item.Stack) bool {
	if s.kind != 'a' || st.Empty() {
		return true
	}
	switch it := st.Item().(type) {
	case item.HelmetType:
		return s.i == 0 && it.Helmet()
	case item.ChestplateType:
		return s.i == 1 && it.Chestplate()
	case item.LeggingsType:
		return s.i == 2 && it.Leggings()
	case item.BootsType:
		return s.i == 3 && it.Boots()
	}
	return false
}

func withCount(s item.Stack, n int) item.Stack {
	if n <= 0 || s.Empty() {
		return item.Stack{}
	}
	return s.Grow(n - s.Count())
}

func oreNum(v any, key string) int {
	m, _ := v.(map[string]any)
	n, _ := m[key].(float64)
	return int(n)
}

func oreFlag(v any, key string) bool {
	m, _ := v.(map[string]any)
	b, _ := m[key].(bool)
	return b
}

func (st *oreInv) event(p *player.Player, ev oreui.Event) {
	if st.form == nil || st.form.Closed() {
		return
	}
	b := oreNum(ev.Value, "b")
	switch {
	case ev.ID == "outside":
		st.dropCursor(p, b)
	case ev.Kind == "slot":
		if s, ok := parseOreSlot(ev.ID); ok {
			st.press(p, s, b, oreFlag(ev.Value, "shift"), oreFlag(ev.Value, "dbl"))
		}
	case ev.Kind == "slotup":
		st.release(p, ev.Value, b)
	case ev.Kind == "slotkey":
		if s, ok := parseOreSlot(ev.ID); ok {
			st.key(p, s, oreNum(ev.Value, "k"), oreFlag(ev.Value, "ctrl"))
		}
	default:
		return
	}
	st.refresh(p)
}

func (st *oreInv) press(p *player.Player, s oreSlot, b int, shift, dbl bool) {
	st.placing = false
	switch {
	case dbl && b == 0 && !st.cursor.Empty():
		st.collect(p)
		st.info = fmt.Sprintf("%s double click: collect", s.id())
	case shift && b == 0:
		st.quickMove(p, s)
		st.info = fmt.Sprintf("%s shift click: quick move", s.id())
	case st.cursor.Empty():
		have := s.get(p)
		if have.Empty() {
			st.info = fmt.Sprintf("%s button %d: empty", s.id(), b)
			return
		}
		take := have.Count()
		if b == 2 {
			take = (take + 1) / 2
		}
		st.cursor = withCount(have, take)
		s.set(p, withCount(have, have.Count()-take))
		st.info = fmt.Sprintf("%s button %d: take %d", s.id(), b, take)
	default:
		st.placing = true
		st.info = fmt.Sprintf("%s button %d: holding", s.id(), b)
	}
}

func (st *oreInv) release(p *player.Player, v any, b int) {
	if !st.placing {
		return
	}
	st.placing = false
	m, _ := v.(map[string]any)
	raw, _ := m["slots"].([]any)
	var slots []oreSlot
	for _, r := range raw {
		id, _ := r.(string)
		if s, ok := parseOreSlot(id); ok {
			slots = append(slots, s)
		}
	}
	if len(slots) == 0 {
		return
	}
	if len(slots) == 1 {
		if b == 2 {
			st.placeOne(p, slots[0])
			st.info = slots[0].id() + " right release: place one"
		} else {
			st.placeAll(p, slots[0])
			st.info = slots[0].id() + " left release: place, merge or swap"
		}
		return
	}
	st.split(p, slots, b)
	st.info = fmt.Sprintf("drag over %d slots with button %d: split", len(slots), b)
}

func (st *oreInv) room(p *player.Player, s oreSlot) int {
	if !s.accepts(st.cursor) {
		return 0
	}
	have := s.get(p)
	if have.Empty() {
		return st.cursor.MaxCount()
	}
	if !have.Comparable(st.cursor) {
		return 0
	}
	return have.MaxCount() - have.Count()
}

func (st *oreInv) put(p *player.Player, s oreSlot, n int) {
	have := s.get(p)
	if have.Empty() {
		if s.set(p, withCount(st.cursor, n)) {
			st.cursor = withCount(st.cursor, st.cursor.Count()-n)
		}
		return
	}
	if s.set(p, withCount(have, have.Count()+n)) {
		st.cursor = withCount(st.cursor, st.cursor.Count()-n)
	}
}

func (st *oreInv) placeAll(p *player.Player, s oreSlot) {
	have := s.get(p)
	if !have.Empty() && !have.Comparable(st.cursor) {
		if s.accepts(st.cursor) && s.set(p, st.cursor) {
			st.cursor = have
		}
		return
	}
	if n := min(st.room(p, s), st.cursor.Count()); n > 0 {
		st.put(p, s, n)
	}
}

func (st *oreInv) placeOne(p *player.Player, s oreSlot) {
	if st.room(p, s) > 0 {
		st.put(p, s, 1)
	}
}

func (st *oreInv) split(p *player.Player, slots []oreSlot, b int) {
	var open []oreSlot
	for _, s := range slots {
		if st.room(p, s) > 0 {
			open = append(open, s)
		}
	}
	if len(open) == 0 {
		return
	}
	each := 1
	if b == 0 {
		each = st.cursor.Count() / len(open)
		if each == 0 {
			each = 1
		}
	}
	for _, s := range open {
		if st.cursor.Empty() {
			return
		}
		st.put(p, s, min(each, st.room(p, s), st.cursor.Count()))
	}
}

// quickMove moves the stack of s to the other part of the inventory, as with
// shift: the hotbar and the rest of the inventory trade, armour goes to its
// slot first, armour and the off hand go back to the inventory.
func (st *oreInv) quickMove(p *player.Player, s oreSlot) {
	have := s.get(p)
	if have.Empty() {
		return
	}
	var to []oreSlot
	if s.kind == 'h' || s.kind == 'i' {
		for i := range 4 {
			to = append(to, oreSlot{'a', i})
		}
	}
	row := func(kind byte, n int) {
		for i := range n {
			to = append(to, oreSlot{kind, i})
		}
	}
	switch s.kind {
	case 'h':
		row('i', 27)
	case 'i':
		row('h', 9)
	default:
		row('i', 27)
		row('h', 9)
	}
	held := st.cursor
	st.cursor = have
	for _, pass := range []bool{true, false} {
		for _, t := range to {
			if st.cursor.Empty() {
				break
			}
			if pass == t.get(p).Empty() {
				continue
			}
			if n := min(st.room(p, t), st.cursor.Count()); n > 0 {
				st.put(p, t, n)
			}
		}
	}
	s.set(p, st.cursor)
	st.cursor = held
}

// collect fills the cursor from the stacks of its item, as with a double
// click.
func (st *oreInv) collect(p *player.Player) {
	for _, s := range oreSlots() {
		room := st.cursor.MaxCount() - st.cursor.Count()
		if room <= 0 {
			return
		}
		have := s.get(p)
		if have.Empty() || !have.Comparable(st.cursor) {
			continue
		}
		n := min(room, have.Count())
		s.set(p, withCount(have, have.Count()-n))
		st.cursor = withCount(st.cursor, st.cursor.Count()+n)
	}
}

const (
	oreKey1 = 49
	oreKey9 = 57
	oreKeyQ = 81
)

func (st *oreInv) key(p *player.Player, s oreSlot, k int, ctrl bool) {
	switch {
	case k >= oreKey1 && k <= oreKey9:
		hb := oreSlot{'h', k - oreKey1}
		a, b := s.get(p), hb.get(p)
		if hb == s || !s.accepts(b) {
			return
		}
		s.set(p, b)
		hb.set(p, a)
		st.info = fmt.Sprintf("%s key %d: swap with %s", s.id(), k-oreKey1+1, hb.id())
	case k == oreKeyQ:
		have := s.get(p)
		if have.Empty() {
			return
		}
		n := 1
		if ctrl {
			n = have.Count()
		}
		dropped := p.Drop(withCount(have, n))
		s.set(p, withCount(have, have.Count()-dropped))
		st.info = fmt.Sprintf("%s Q: drop %d", s.id(), dropped)
	default:
		st.info = fmt.Sprintf("%s key %d", s.id(), k)
	}
}

func (st *oreInv) dropCursor(p *player.Player, b int) {
	if st.cursor.Empty() {
		return
	}
	n := st.cursor.Count()
	if b == 2 {
		n = 1
	}
	dropped := p.Drop(withCount(st.cursor, n))
	st.cursor = withCount(st.cursor, st.cursor.Count()-dropped)
	st.info = fmt.Sprintf("outside button %d: drop %d", b, dropped)
}

// giveBack puts the item on the cursor back in the inventory when the screen
// closes, dropping what does not fit.
func (st *oreInv) giveBack(p *player.Player) {
	if st.cursor.Empty() {
		return
	}
	n, _ := p.Inventory().AddItem(st.cursor)
	if rest := st.cursor.Count() - n; rest > 0 {
		p.Drop(withCount(st.cursor, rest))
	}
	st.cursor = item.Stack{}
}

func oreSlotValue(s item.Stack) map[string]any {
	if s.Empty() {
		return map[string]any{"id": "", "n": 0}
	}
	return map[string]any{"id": "aux:" + ui.ItemIcon(s.Item()), "n": s.Count()}
}

func (st *oreInv) slotNode(p *player.Player, s oreSlot) map[string]any {
	n := oreNode("slot", "id", s.id(), "bind", "s:"+s.id(), "raw", true, "size", 32, "class", "mc-slot")
	if have := s.get(p); !have.Empty() {
		n["tip"] = fmt.Sprintf("%s\n§7x%d §8%s", itemName(i18n.Of(p), have), have.Count(), s.id())
	}
	return n
}

func (st *oreInv) layout(p *player.Player) map[string]any {
	row := func(kind byte, from, n int) map[string]any {
		var ch []any
		for i := from; i < from+n; i++ {
			ch = append(ch, st.slotNode(p, oreSlot{kind, i}))
		}
		return oreRow(ch...)
	}
	var armour []any
	for i := range 4 {
		armour = append(armour, st.slotNode(p, oreSlot{'a', i}))
	}
	gap := oreNode("box", "style", map[string]any{"height": "0.8rem"})
	return probeScreen("/dfdebug oreinv", "36rem", "38rem",
		oreNode("text", "bind", "info", "text", "-"),
		oreRow(oreCol(armour...), oreNode("box", "style", map[string]any{"width": "1.6rem"}), st.slotNode(p, oreSlot{'o', 0})),
		gap,
		row('i', 0, 9), row('i', 9, 9), row('i', 18, 9),
		gap,
		row('h', 0, 9),
	)
}

// refresh sends what changed: the slots and the cursor as variables, the
// layout when a tooltip changed.
func (st *oreInv) refresh(p *player.Player) {
	if st.form == nil || st.form.Closed() {
		return
	}
	if st.sent == nil {
		st.sent = map[string]string{}
	}
	vars := map[string]any{}
	note := func(key string, v any) {
		b, _ := json.Marshal(v)
		if st.sent[key] != string(b) {
			st.sent[key] = string(b)
			vars[key] = v
		}
	}
	for _, s := range oreSlots() {
		note("s:"+s.id(), oreSlotValue(s.get(p)))
	}
	note("cursor", oreSlotValue(st.cursor))
	note("info", st.info)
	if len(vars) > 0 {
		st.form.SetVars(vars)
	}
	layout := st.layout(p)
	if b, _ := json.Marshal(layout); string(b) != st.shown {
		st.shown = string(b)
		_ = st.form.SetLayout(layout)
	}
}
