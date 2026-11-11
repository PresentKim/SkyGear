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
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/textfmt"
)

// The drawer controller (after Storage Drawers'): it makes the drawers that
// touch it, directly or through each other, the drawer trims (blocks that
// only connect) and the drawer interfaces, up to controllerReach blocks
// away, one storage: hoppers and conduits put items into it and take them
// out through the controller, a click deposits the held stack and a second
// click the inventory. A drawer interface in the set does what the
// controller does. A set holds one controller: when two touch it, neither
// works.

// controllerReach is how far (in blocks, each axis) from the controller a block
// still belongs to its network.
const controllerReach = 16

type storageDrawer struct {
	Pos cube.Pos
	D   Drawer
}

// network is what a controller reaches: its drawers, nearest first, and
// its drawer interfaces.
type network struct {
	drawers []storageDrawer
	ifaces  []cube.Pos
}

// Network finds the drawers of the controller at pos, or of the controller
// of the drawer interface at pos.
func Network(tx *world.Tx, pos cube.Pos) []storageDrawer {
	if isInterface(tx.Block(pos)) {
		ctl, ok := interfaceController(tx, pos)
		if !ok {
			return nil
		}
		pos = ctl
	}
	return controllerNetwork(tx, pos).drawers
}

func controllerNetwork(tx *world.Tx, pos cube.Pos) network {
	s := core.At(tx)
	if s == nil {
		return findNetwork(tx, pos)
	}
	l := LocalOf(s)
	now := networkTick()
	if c, ok := l.networks[pos]; ok && c.tick == now {
		if ds, ok := drawersAt(tx, c.pos); ok {
			return network{ds, c.ifaces}
		}
	}
	n := findNetwork(tx, pos)
	ps := make([]cube.Pos, len(n.drawers))
	for i, sd := range n.drawers {
		ps[i] = sd.Pos
	}
	if l.networks == nil {
		l.networks = map[cube.Pos]networkCache{}
	}
	l.networks[pos] = networkCache{now, ps, n.ifaces}
	return n
}

// drawersAt reads the drawers at ps; ok is false when one is gone.
func drawersAt(tx *world.Tx, ps []cube.Pos) ([]storageDrawer, bool) {
	out := make([]storageDrawer, 0, len(ps))
	for _, p := range ps {
		d, ok := tx.Block(p).(Drawer)
		if !ok {
			return nil, false
		}
		out = append(out, storageDrawer{p, d})
	}
	return out, true
}

// networkCache is the drawer and interface positions of a controller found
// at a tick. The drawer values are always read again.
type networkCache struct {
	tick   int64
	pos    []cube.Pos
	ifaces []cube.Pos
}

// interfaceCache is the controller of a drawer interface found at a tick.
type interfaceCache struct {
	tick int64
	ctl  cube.Pos
	ok   bool
}

// networkTick is the tick (50 ms step) the network lists are kept for.
var networkTick = func() int64 { return time.Now().UnixMilli() / 50 }

// invalidateNetworks forgets the cached network lists (a drawer, trim,
// interface or controller was placed or broken).
func invalidateNetworks(s *core.Session) {
	clear(LocalOf(s).networks)
	clear(LocalOf(s).interfaces)
}

// InvalidateNetworks is invalidateNetworks for a machine of a drawer set
// placed or broken.
func InvalidateNetworks(s *core.Session) { invalidateNetworks(s) }

// IsSetMachine reports whether kind is a machine of a drawer set: the
// controller, the trim or the interface.
func IsSetMachine(kind string) bool {
	return kind == machine.KindDrawerController || kind == machine.KindDrawerTrim || kind == machine.KindDrawerInterface
}

func findNetwork(tx *world.Tx, pos cube.Pos) network {
	seen := map[cube.Pos]bool{pos: true}
	queue := []cube.Pos{pos}
	var n network
	for len(queue) > 0 {
		at := queue[0]
		queue = queue[1:]
		for _, f := range cube.Faces() {
			next := at.Side(f)
			if seen[next] || !within(next, pos, controllerReach) {
				continue
			}
			switch b := tx.Block(next).(type) {
			case Drawer:
				seen[next] = true
				n.drawers = append(n.drawers, storageDrawer{next, b})
				queue = append(queue, next)
			case machine.Machine:
				switch b.Kind {
				case machine.KindDrawerController:
					return network{}
				case machine.KindDrawerTrim, machine.KindDrawerInterface:
					seen[next] = true
					queue = append(queue, next)
					if b.Kind == machine.KindDrawerInterface {
						n.ifaces = append(n.ifaces, next)
					}
				}
			}
		}
	}
	sort.SliceStable(n.drawers, func(i, j int) bool { return distSq(n.drawers[i].Pos, pos) < distSq(n.drawers[j].Pos, pos) })
	return n
}

func within(a, b cube.Pos, reach int) bool {
	return core.Abs(a.X()-b.X()) <= reach && core.Abs(a.Y()-b.Y()) <= reach && core.Abs(a.Z()-b.Z()) <= reach
}

func isInterface(b world.Block) bool {
	m, ok := b.(machine.Machine)
	return ok && m.Kind == machine.KindDrawerInterface
}

// interfaceController finds the controller whose network holds the drawer
// interface at pos.
func interfaceController(tx *world.Tx, pos cube.Pos) (cube.Pos, bool) {
	s := core.At(tx)
	if s == nil {
		return findInterfaceController(tx, pos)
	}
	l := LocalOf(s)
	now := networkTick()
	if c, ok := l.interfaces[pos]; ok && c.tick == now {
		return c.ctl, c.ok
	}
	ctl, ok := findInterfaceController(tx, pos)
	if l.interfaces == nil {
		l.interfaces = map[cube.Pos]interfaceCache{}
	}
	l.interfaces[pos] = interfaceCache{now, ctl, ok}
	return ctl, ok
}

// findInterfaceController walks the set from the interface at pos to the
// controllers it touches, as far as a controller could reach past it, and
// returns the one whose network holds the interface.
func findInterfaceController(tx *world.Tx, pos cube.Pos) (cube.Pos, bool) {
	seen := map[cube.Pos]bool{pos: true}
	queue := []cube.Pos{pos}
	var ctls []cube.Pos
	for len(queue) > 0 {
		at := queue[0]
		queue = queue[1:]
		for _, f := range cube.Faces() {
			next := at.Side(f)
			if seen[next] || !within(next, pos, 2*controllerReach) {
				continue
			}
			seen[next] = true
			switch b := tx.Block(next).(type) {
			case Drawer:
				queue = append(queue, next)
			case machine.Machine:
				if b.Kind == machine.KindDrawerController {
					ctls = append(ctls, next)
				} else if b.Kind == machine.KindDrawerTrim || b.Kind == machine.KindDrawerInterface {
					queue = append(queue, next)
				}
			}
		}
	}
	for _, c := range ctls {
		if within(c, pos, controllerReach) && slices.Contains(controllerNetwork(tx, c).ifaces, pos) {
			return c, true
		}
	}
	return cube.Pos{}, false
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

// Lines are the WAILA lines of a controller or drawer interface.
func Lines(lang string, tx *world.Tx, pos cube.Pos) []string {
	list, drawers := Contents(tx, pos)
	var total int64
	for _, e := range list {
		total += e.Count
	}
	return []string{fmt.Sprintf(i18n.T(lang, "storage.lines"), drawers, len(list), total)}
}

// IsDrawerController reports whether b is a drawer controller or a drawer
// interface, the blocks that stand for a drawer set.
func IsDrawerController(b world.Block) bool {
	m, ok := b.(machine.Machine)
	return ok && (m.Kind == machine.KindDrawerController || m.Kind == machine.KindDrawerInterface)
}

// UseController handles a click on a controller or drawer interface: it
// deposits the held stack (all of it), and a second click within 500ms of
// a click also deposits the inventory (depositAll).
func UseController(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) {
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

// storageOrder is how a storage list is sorted; the zero value is the most
// items first.
type storageOrder int

const (
	OrderCountDesc storageOrder = iota
	OrderCountAsc
	OrderNameAsc
	OrderNameDesc
)

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
