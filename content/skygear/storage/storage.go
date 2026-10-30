package storage

import (
	"cmp"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/textsearch"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/decor"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/textfmt"
)

// The storage controller (after FunctionalStorage's): it
// makes the drawers that touch it, directly or through each other and the storage
// links (blocks that only connect), up to controllerReach blocks away, one
// storage: hoppers and conduits put items into it and take them out through the
// controller, a click deposits the held stack and a sneaking click opens a
// list to take from or deposit into.

// controllerReach is how far (in blocks, each axis) from the controller a block
// still belongs to its network.
const controllerReach = 16

type storageDrawer struct {
	Pos cube.Pos
	D   Drawer
}

// Network finds the drawers of the controller at pos: the drawers and
// links connected to it by touching sides, within controllerReach, nearest first.
func Network(tx *world.Tx, pos cube.Pos) []storageDrawer {
	seen := map[cube.Pos]bool{pos: true}
	queue := []cube.Pos{pos}
	var out []storageDrawer
	for len(queue) > 0 {
		at := queue[0]
		queue = queue[1:]
		for _, f := range cube.Faces() {
			next := at.Side(f)
			if seen[next] || core.Abs(next.X()-pos.X()) > controllerReach || core.Abs(next.Y()-pos.Y()) > controllerReach || core.Abs(next.Z()-pos.Z()) > controllerReach {
				continue
			}
			switch b := tx.Block(next).(type) {
			case Drawer:
				seen[next] = true
				out = append(out, storageDrawer{next, b})
				queue = append(queue, next)
			case machine.Machine:
				if b.Kind == machine.KindStorageLink {
					seen[next] = true
					queue = append(queue, next)
				}
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return distSq(out[i].Pos, pos) < distSq(out[j].Pos, pos) })
	return out
}

func distSq(a, b cube.Pos) int {
	dx, dy, dz := a.X()-b.X(), a.Y()-b.Y(), a.Z()-b.Z()
	return dx*dx + dy*dy + dz*dz
}

// storeDrawer saves a changed drawer.
func storeDrawer(tx *world.Tx, pos cube.Pos, d Drawer) {
	if s := core.At(tx); s != nil {
		SetDrawer(s, tx, pos, d)
	} else {
		tx.SetBlockEntity(pos, d)
	}
}

// ControllerInsert puts st into the network: first into drawers that already hold
// that item, then into empty ones that take it; it returns how many it took.
func ControllerInsert(tx *world.Tx, pos cube.Pos, st item.Stack) int {
	return controllerInsertPasses(tx, pos, st, 2)
}

// ControllerInsertHeld is controllerInsert into the drawers that already hold
// the kind of item only: it never fills an empty drawer with a new kind.
func ControllerInsertHeld(tx *world.Tx, pos cube.Pos, st item.Stack) int {
	return controllerInsertPasses(tx, pos, st, 1)
}

func controllerInsertPasses(tx *world.Tx, pos cube.Pos, st item.Stack, passes int) int {
	net := Network(tx, pos)
	taken := 0
	for pass := 0; pass < passes && taken < st.Count(); pass++ {
		for _, sd := range net {
			if taken >= st.Count() {
				break
			}
			holds := !sd.D.Item.Empty()
			if (pass == 0) != holds || !sd.D.TakesAutomated(st) {
				continue
			}
			d := sd.D
			if n := d.Insert(st, st.Count()-taken); n > 0 {
				taken += n
				storeDrawer(tx, sd.Pos, d)
			}
		}
	}
	return taken
}

// ControllerPeek returns the first stack in the network that want accepts.
func ControllerPeek(tx *world.Tx, pos cube.Pos, want func(item.Stack) bool) item.Stack {
	for _, sd := range Network(tx, pos) {
		if sd.D.Count > 0 && (want == nil || want(sd.D.Item)) {
			return sd.D.Item.Grow(int(min(sd.D.Count, int64(sd.D.Item.MaxCount()))) - sd.D.Item.Count())
		}
	}
	return item.Stack{}
}

// ControllerExtract takes up to n items of the kind of st out of the network.
func ControllerExtract(tx *world.Tx, pos cube.Pos, st item.Stack, n int) item.Stack {
	var out item.Stack
	got := 0
	for _, sd := range Network(tx, pos) {
		if got >= n {
			break
		}
		if sd.D.Count == 0 || !sd.D.Item.Comparable(st) {
			continue
		}
		d := sd.D
		part := d.Extract(min(n-got, int(d.Count)))
		if part.Empty() {
			continue
		}
		storeDrawer(tx, sd.Pos, d)
		got += part.Count()
		out = part.Grow(got - part.Count())
	}
	return out
}

// Entry is an item the network holds with its count.
type Entry struct {
	Item  item.Stack
	Count int64
}

func Contents(tx *world.Tx, pos cube.Pos) ([]Entry, int) {
	net := Network(tx, pos)
	var list []Entry
	for _, sd := range net {
		if sd.D.Count == 0 {
			continue
		}
		merged := false
		for i := range list {
			if list[i].Item.Comparable(sd.D.Item) {
				list[i].Count += sd.D.Count
				merged = true
			}
		}
		if !merged {
			list = append(list, Entry{sd.D.Item, sd.D.Count})
		}
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].Count > list[j].Count })
	return list, len(net)
}

// Lines are the WAILA lines of a controller.
func Lines(lang string, tx *world.Tx, pos cube.Pos) []string {
	list, drawers := Contents(tx, pos)
	var total int64
	for _, e := range list {
		total += e.Count
	}
	return []string{fmt.Sprintf(i18n.T(lang, "storage.lines"), drawers, len(list), total)}
}

func IsStorageController(b world.Block) bool {
	m, ok := b.(machine.Machine)
	return ok && m.Kind == machine.KindStorageController
}

// UseController handles a click on a controller: a sneaking click opens the
// list, any other deposits the held stack (all of it), and a second click
// within 500ms of a click also deposits the inventory (depositAll).
func UseController(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) {
	if p.Sneaking() {
		if !s.Debounced(p) {
			Show(s, p, tx, pos, "")
		}
		return
	}
	last := lastUse.Of(s)[p.H()]
	double := last.Pos == pos && time.Since(last.at) < 500*time.Millisecond
	lastUse.Of(s)[p.H()] = drawerUse{Pos: pos, at: time.Now()}
	held, off := p.HeldItems()
	total := 0
	if n := ControllerInsert(tx, pos, held); n > 0 {
		p.SetHeldItems(held.Grow(-n), off)
		total += n
	}
	if double {
		total += depositAll(tx, pos, p)
	}
	if total > 0 {
		tx.PlaySound(pos.Vec3Centre(), sound.ItemAdd{})
	}
}

// depositAll puts everything of p's inventory that the network already holds
// a kind of into it, and returns how many items went in.
func depositAll(tx *world.Tx, pos cube.Pos, p *player.Player) int {
	inv := p.Inventory()
	total := 0
	for slot, st := range inv.Slots() {
		if st.Empty() {
			continue
		}
		if n := ControllerInsertHeld(tx, pos, st); n > 0 {
			_ = inv.SetItem(slot, st.Grow(-n))
			total += n
		}
	}
	return total
}

// Filter keeps the entries whose name matches query.
func Filter(lang string, list []Entry, query string) []Entry {
	if query == "" {
		return list
	}
	var out []Entry
	for _, e := range list {
		if textsearch.Match(query, naming.ItemName(lang, e.Item)) {
			out = append(out, e)
		}
	}
	return out
}

// Show lists all the items of the network whose name matches query
// (all when it is empty) in one scrolling grid under a search box in the
// window, drawn again as the player types; pressing one takes a stack.
func Show(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, query string) {
	if _, drawers := Contents(tx, pos); drawers == 0 {
		return
	}
	lang := i18n.Of(p)
	find := func(p *player.Player, query string) ui.SearchResult {
		query = strings.TrimSpace(query)
		all, drawers := Contents(p.Tx(), pos)
		list := Sort(lang, Filter(lang, all, query), Orders.Of(s)[p.H()])
		again := func(p *player.Player, tx *world.Tx) { Show(s, p, tx, pos, query) }
		return ui.SearchResult{Info: Info(lang, all, drawers, query, len(list)), Sections: []ui.GridSection{{Slots: Slots(s, lang, pos, list, again)}}}
	}
	again := func(p *player.Player, tx *world.Tx) { Show(s, p, tx, pos, query) }
	ui.SearchGrid(p, i18n.T(lang, "storage.title"), i18n.T(lang, "storage.search.label"), query, find,
		DepositAllOption(s, lang, pos, again), OrderOption(s, p, again))
}

// storageOrder is how a controller list is sorted; the zero value is the
// most items first.
type storageOrder int

const (
	OrderCountDesc storageOrder = iota
	OrderCountAsc
	OrderNameAsc
	OrderNameDesc
	StorageOrderCount
)

var storageOrderKeys = [StorageOrderCount]string{"storage.order.countDesc", "storage.order.countAsc", "storage.order.nameAsc", "storage.order.nameDesc"}

// Sort returns list sorted by o, names in lang.
func Sort(lang string, list []Entry, o storageOrder) []Entry {
	type named struct {
		Entry
		name string
	}
	out := make([]named, len(list))
	for i, e := range list {
		out[i] = named{e, textfmt.PlainName(naming.ItemName(lang, e.Item))}
	}
	slices.SortStableFunc(out, func(a, b named) int {
		switch o {
		case OrderCountAsc:
			return cmp.Compare(a.Count, b.Count)
		case OrderNameAsc:
			return strings.Compare(a.name, b.name)
		case OrderNameDesc:
			return strings.Compare(b.name, a.name)
		}
		return cmp.Compare(b.Count, a.Count)
	})
	sorted := make([]Entry, len(out))
	for i, n := range out {
		sorted[i] = n.Entry
	}
	return sorted
}

// OrderOption is the control bar button that names p's order and
// switches to the next one.
func OrderOption(s *core.Session, p *player.Player, again func(p *player.Player, tx *world.Tx)) ui.Option {
	o := Orders.Of(s)[p.H()]
	return ui.BarButton(i18n.T(i18n.Of(p), storageOrderKeys[o]), "textures/ui/store_filter_icon", func(p *player.Player, tx *world.Tx) {
		Orders.Of(s)[p.H()] = (o + 1) % StorageOrderCount
		again(p, tx)
	})
}

// Slots are the slots of list; pressing one takes a stack of it and
// runs again.
func Slots(s *core.Session, lang string, pos cube.Pos, list []Entry, again func(p *player.Player, tx *world.Tx)) []ui.ItemSlot {
	var slots []ui.ItemSlot
	for _, e := range list {
		slots = append(slots, ui.ItemSlot{
			Icon:  ui.ItemIcon(e.Item.Item()),
			Count: "§f" + ui.CountText(int(e.Count)),
			Tip:   decor.TipName(lang, e.Item) + "\n§7" + fmt.Sprintf(i18n.T(lang, "storage.tip"), e.Count),
			Run: func(p *player.Player, tx *world.Tx) {
				got := ControllerExtract(tx, pos, e.Item, e.Item.MaxCount())
				if !got.Empty() {
					s.GiveItem(p, tx, p.Position(), got)
				}
				again(p, tx)
			},
		})
	}
	return slots
}

func DepositAllOption(s *core.Session, lang string, pos cube.Pos, again func(p *player.Player, tx *world.Tx)) ui.Option {
	return ui.BarButton(i18n.T(lang, "storage.depositAll"), "textures/ui/sg/arrow_down", func(p *player.Player, tx *world.Tx) {
		if depositAll(tx, pos, p) > 0 {
			tx.PlaySound(pos.Vec3Centre(), sound.ItemAdd{})
		}
		again(p, tx)
	})
}

// Info is the line about the network over the list, and about the
// search when there is a query.
func Info(lang string, all []Entry, drawers int, query string, found int) string {
	info := fmt.Sprintf(i18n.T(lang, "storage.lines"), drawers, len(all), totalOf(all))
	if query != "" {
		info += "\n" + fmt.Sprintf(i18n.T(lang, "storage.search.found"), query, found)
	}
	return info
}

func totalOf(list []Entry) int64 {
	var t int64
	for _, e := range list {
		t += e.Count
	}
	return t
}

// Orders are how each player sorts storage controller lists.
var Orders = core.NewPlayers[storageOrder]()

// The storage controller is a machine kind whose clicks and item I/O are the
// storage's.
func init() {
	ctl := machine.Kinds[machine.KindStorageController]
	ctl.Click = func(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool {
		UseController(s, p, tx, pos)
		return true
	}
	ctl.Insert, ctl.Peek, ctl.Extract = ControllerInsert, ControllerPeek, ControllerExtract
}
