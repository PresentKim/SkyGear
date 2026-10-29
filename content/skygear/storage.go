package skygear

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
	"github.com/presentkim/skygear/content/skygear/naming"
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
	pos cube.Pos
	d   Drawer
}

// storageNetwork finds the drawers of the controller at pos: the drawers and
// links connected to it by touching sides, within controllerReach, nearest first.
func storageNetwork(tx *world.Tx, pos cube.Pos) []storageDrawer {
	seen := map[cube.Pos]bool{pos: true}
	queue := []cube.Pos{pos}
	var out []storageDrawer
	for len(queue) > 0 {
		at := queue[0]
		queue = queue[1:]
		for _, f := range cube.Faces() {
			next := at.Side(f)
			if seen[next] || abs(next.X()-pos.X()) > controllerReach || abs(next.Y()-pos.Y()) > controllerReach || abs(next.Z()-pos.Z()) > controllerReach {
				continue
			}
			switch b := tx.Block(next).(type) {
			case Drawer:
				seen[next] = true
				out = append(out, storageDrawer{next, b})
				queue = append(queue, next)
			case Machine:
				if b.Kind == KindStorageLink {
					seen[next] = true
					queue = append(queue, next)
				}
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return distSq(out[i].pos, pos) < distSq(out[j].pos, pos) })
	return out
}

func distSq(a, b cube.Pos) int {
	dx, dy, dz := a.X()-b.X(), a.Y()-b.Y(), a.Z()-b.Z()
	return dx*dx + dy*dy + dz*dz
}

// storeDrawer saves a changed drawer.
func storeDrawer(tx *world.Tx, pos cube.Pos, d Drawer) {
	if s := activeGameSession(tx); s != nil {
		s.setDrawer(tx, pos, d)
	} else {
		tx.SetBlockEntity(pos, d)
	}
}

// controllerInsert puts st into the network: first into drawers that already hold
// that item, then into empty ones that take it; it returns how many it took.
func controllerInsert(tx *world.Tx, pos cube.Pos, st item.Stack) int {
	return controllerInsertPasses(tx, pos, st, 2)
}

// controllerInsertHeld is controllerInsert into the drawers that already hold
// the kind of item only: it never fills an empty drawer with a new kind.
func controllerInsertHeld(tx *world.Tx, pos cube.Pos, st item.Stack) int {
	return controllerInsertPasses(tx, pos, st, 1)
}

func controllerInsertPasses(tx *world.Tx, pos cube.Pos, st item.Stack, passes int) int {
	net := storageNetwork(tx, pos)
	taken := 0
	for pass := 0; pass < passes && taken < st.Count(); pass++ {
		for _, sd := range net {
			if taken >= st.Count() {
				break
			}
			holds := !sd.d.Item.Empty()
			if (pass == 0) != holds || !sd.d.takesAutomated(st) {
				continue
			}
			d := sd.d
			if n := d.insert(st, st.Count()-taken); n > 0 {
				taken += n
				storeDrawer(tx, sd.pos, d)
			}
		}
	}
	return taken
}

// controllerPeek returns the first stack in the network that want accepts.
func controllerPeek(tx *world.Tx, pos cube.Pos, want func(item.Stack) bool) item.Stack {
	for _, sd := range storageNetwork(tx, pos) {
		if sd.d.Count > 0 && (want == nil || want(sd.d.Item)) {
			return sd.d.Item.Grow(int(min(sd.d.Count, int64(sd.d.Item.MaxCount()))) - sd.d.Item.Count())
		}
	}
	return item.Stack{}
}

// controllerExtract takes up to n items of the kind of st out of the network.
func controllerExtract(tx *world.Tx, pos cube.Pos, st item.Stack, n int) item.Stack {
	var out item.Stack
	got := 0
	for _, sd := range storageNetwork(tx, pos) {
		if got >= n {
			break
		}
		if sd.d.Count == 0 || !sd.d.Item.Comparable(st) {
			continue
		}
		d := sd.d
		part := d.extract(min(n-got, int(d.Count)))
		if part.Empty() {
			continue
		}
		storeDrawer(tx, sd.pos, d)
		got += part.Count()
		out = part.Grow(got - part.Count())
	}
	return out
}

// storageEntry is an item the network holds with its count.
type storageEntry struct {
	item  item.Stack
	count int64
}

func storageContents(tx *world.Tx, pos cube.Pos) ([]storageEntry, int) {
	net := storageNetwork(tx, pos)
	var list []storageEntry
	for _, sd := range net {
		if sd.d.Count == 0 {
			continue
		}
		merged := false
		for i := range list {
			if list[i].item.Comparable(sd.d.Item) {
				list[i].count += sd.d.Count
				merged = true
			}
		}
		if !merged {
			list = append(list, storageEntry{sd.d.Item, sd.d.Count})
		}
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].count > list[j].count })
	return list, len(net)
}

// storageLines are the WAILA lines of a controller.
func storageLines(lang string, tx *world.Tx, pos cube.Pos) []string {
	list, drawers := storageContents(tx, pos)
	var total int64
	for _, e := range list {
		total += e.count
	}
	return []string{fmt.Sprintf(i18n.T(lang, "storage.lines"), drawers, len(list), total)}
}

func isStorageController(b world.Block) bool {
	m, ok := b.(Machine)
	return ok && m.Kind == KindStorageController
}

// useController handles a click on a controller: a sneaking click opens the
// list, any other deposits the held stack (all of it), and a second click
// within 500ms of a click also deposits the inventory (depositAll).
func (s *session) useController(p *player.Player, tx *world.Tx, pos cube.Pos) {
	if p.Sneaking() {
		if !s.debounced(p) {
			s.showStorage(p, tx, pos, "")
		}
		return
	}
	last := s.lastUse[p.H()]
	double := last.pos == pos && time.Since(last.at) < 500*time.Millisecond
	s.lastUse[p.H()] = drawerUse{pos: pos, at: time.Now()}
	held, off := p.HeldItems()
	total := 0
	if n := controllerInsert(tx, pos, held); n > 0 {
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
		if n := controllerInsertHeld(tx, pos, st); n > 0 {
			_ = inv.SetItem(slot, st.Grow(-n))
			total += n
		}
	}
	return total
}

// filterStorage keeps the entries whose name matches query.
func filterStorage(lang string, list []storageEntry, query string) []storageEntry {
	if query == "" {
		return list
	}
	var out []storageEntry
	for _, e := range list {
		if textsearch.Match(query, naming.ItemName(lang, e.item)) {
			out = append(out, e)
		}
	}
	return out
}

// showStorage lists all the items of the network whose name matches query
// (all when it is empty) in one scrolling grid under a search box in the
// window, drawn again as the player types; pressing one takes a stack.
func (s *session) showStorage(p *player.Player, tx *world.Tx, pos cube.Pos, query string) {
	if _, drawers := storageContents(tx, pos); drawers == 0 {
		return
	}
	lang := i18n.Of(p)
	find := func(p *player.Player, query string) ui.SearchResult {
		query = strings.TrimSpace(query)
		all, drawers := storageContents(p.Tx(), pos)
		list := sortStorage(lang, filterStorage(lang, all, query), s.storageOrders[p.H()])
		again := func(p *player.Player, tx *world.Tx) { s.showStorage(p, tx, pos, query) }
		return ui.SearchResult{Info: storageInfo(lang, all, drawers, query, len(list)), Sections: []ui.GridSection{{Slots: s.storageSlots(lang, pos, list, again)}}}
	}
	again := func(p *player.Player, tx *world.Tx) { s.showStorage(p, tx, pos, query) }
	ui.SearchGrid(p, i18n.T(lang, "storage.title"), i18n.T(lang, "storage.search.label"), query, find,
		s.depositAllOption(lang, pos, again), s.storageOrderOption(p, again))
}

// storageOrder is how a controller list is sorted; the zero value is the
// most items first.
type storageOrder int

const (
	orderCountDesc storageOrder = iota
	orderCountAsc
	orderNameAsc
	orderNameDesc
	storageOrderCount
)

var storageOrderKeys = [storageOrderCount]string{"storage.order.countDesc", "storage.order.countAsc", "storage.order.nameAsc", "storage.order.nameDesc"}

// sortStorage returns list sorted by o, names in lang.
func sortStorage(lang string, list []storageEntry, o storageOrder) []storageEntry {
	type named struct {
		storageEntry
		name string
	}
	out := make([]named, len(list))
	for i, e := range list {
		out[i] = named{e, plainName(naming.ItemName(lang, e.item))}
	}
	slices.SortStableFunc(out, func(a, b named) int {
		switch o {
		case orderCountAsc:
			return cmp.Compare(a.count, b.count)
		case orderNameAsc:
			return strings.Compare(a.name, b.name)
		case orderNameDesc:
			return strings.Compare(b.name, a.name)
		}
		return cmp.Compare(b.count, a.count)
	})
	sorted := make([]storageEntry, len(out))
	for i, n := range out {
		sorted[i] = n.storageEntry
	}
	return sorted
}

// storageOrderOption is the control bar button that names p's order and
// switches to the next one.
func (s *session) storageOrderOption(p *player.Player, again func(p *player.Player, tx *world.Tx)) ui.Option {
	o := s.storageOrders[p.H()]
	return ui.BarButton(i18n.T(i18n.Of(p), storageOrderKeys[o]), "textures/ui/store_filter_icon", func(p *player.Player, tx *world.Tx) {
		if s.storageOrders == nil {
			s.storageOrders = map[*world.EntityHandle]storageOrder{}
		}
		s.storageOrders[p.H()] = (o + 1) % storageOrderCount
		again(p, tx)
	})
}

// storageSlots are the slots of list; pressing one takes a stack of it and
// runs again.
func (s *session) storageSlots(lang string, pos cube.Pos, list []storageEntry, again func(p *player.Player, tx *world.Tx)) []ui.ItemSlot {
	var slots []ui.ItemSlot
	for _, e := range list {
		slots = append(slots, ui.ItemSlot{
			Icon:  ui.ItemIcon(e.item.Item()),
			Count: "§f" + ui.CountText(int(e.count)),
			Tip:   tipName(lang, e.item) + "\n§7" + fmt.Sprintf(i18n.T(lang, "storage.tip"), e.count),
			Run: func(p *player.Player, tx *world.Tx) {
				got := controllerExtract(tx, pos, e.item, e.item.MaxCount())
				if !got.Empty() {
					s.giveItem(p, tx, p.Position(), got)
				}
				again(p, tx)
			},
		})
	}
	return slots
}

func (s *session) depositAllOption(lang string, pos cube.Pos, again func(p *player.Player, tx *world.Tx)) ui.Option {
	return ui.BarButton(i18n.T(lang, "storage.depositAll"), "textures/ui/sg/arrow_down", func(p *player.Player, tx *world.Tx) {
		if depositAll(tx, pos, p) > 0 {
			tx.PlaySound(pos.Vec3Centre(), sound.ItemAdd{})
		}
		again(p, tx)
	})
}

// storageInfo is the line about the network over the list, and about the
// search when there is a query.
func storageInfo(lang string, all []storageEntry, drawers int, query string, found int) string {
	info := fmt.Sprintf(i18n.T(lang, "storage.lines"), drawers, len(all), totalOf(all))
	if query != "" {
		info += "\n" + fmt.Sprintf(i18n.T(lang, "storage.search.found"), query, found)
	}
	return info
}

func totalOf(list []storageEntry) int64 {
	var t int64
	for _, e := range list {
		t += e.count
	}
	return t
}
