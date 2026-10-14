package dragonfactory

import (
	"fmt"
	"sort"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/presentkim/dragonfactory/internal/i18n"
	"github.com/presentkim/dragonfactory/internal/ui"
)

// The storage controller (FunctionalStorage's controller, docs/design/08): it
// makes the drawers that touch it, directly or through each other and the storage
// links (blocks that only connect), up to controllerReach blocks away, one
// storage: hoppers and conduits put items into it and take them out through the
// controller, and a click opens a list to take from or deposit into.

// controllerReach is how far (in blocks, each axis) from the controller a block
// still belongs to its network.
const controllerReach = 16

// storageDrawer is a drawer of a network.
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
			if seen[next] || absInt(next.X()-pos.X()) > controllerReach || absInt(next.Y()-pos.Y()) > controllerReach || absInt(next.Z()-pos.Z()) > controllerReach {
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

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
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
	net := storageNetwork(tx, pos)
	taken := 0
	for pass := 0; pass < 2 && taken < st.Count(); pass++ {
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

// storageTotals lists what the network holds, by item, biggest count first.
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
func storageLines(tx *world.Tx, pos cube.Pos) []string {
	list, drawers := storageContents(tx, pos)
	var total int64
	for _, e := range list {
		total += e.count
	}
	return []string{fmt.Sprintf(i18n.T(i18n.Default, "storage.lines"), drawers, len(list), total)}
}

// useController handles a click on a controller: an item in hand is deposited
// (all of it), an empty hand opens the list.
func (s *session) useController(p *player.Player, tx *world.Tx, pos cube.Pos, held item.Stack) {
	if !held.Empty() {
		if n := controllerInsert(tx, pos, held); n > 0 {
			_, off := p.HeldItems()
			p.SetHeldItems(held.Grow(-n), off)
			tx.PlaySound(pos.Vec3Centre(), sound.ItemAdd{})
		}
		return
	}
	s.showStorage(p, tx, pos, 0)
}

// storagePage is how many entries a page of the list shows.
const storagePage = 44

// showStorage lists the items of the network; pressing one takes a stack.
func (s *session) showStorage(p *player.Player, tx *world.Tx, pos cube.Pos, page int) {
	lang := i18n.Of(p)
	list, drawers := storageContents(tx, pos)
	if drawers == 0 {
		return
	}
	pages := max(1, (len(list)+storagePage-1)/storagePage)
	page = max(0, min(page, pages-1))
	var slots []ui.ItemSlot
	for _, e := range list[page*storagePage : min(len(list), (page+1)*storagePage)] {
		e := e
		slots = append(slots, ui.ItemSlot{
			Icon:  ui.ItemIcon(e.item.Item()),
			Count: "§f" + ui.CountText(int(e.count)),
			Tip:   "§f" + itemName(e.item) + "\n§7" + fmt.Sprintf(i18n.T(lang, "storage.tip"), e.count),
			Run: func(p *player.Player, tx *world.Tx) {
				got := controllerExtract(tx, pos, e.item, e.item.MaxCount())
				if !got.Empty() {
					s.giveItem(p, tx, p.Position(), got)
				}
				s.showStorage(p, tx, pos, page)
			},
		})
	}
	var actions []ui.Option
	if page+1 < pages {
		actions = append(actions, ui.Option{Label: i18n.T(lang, "storage.next"), Run: func(p *player.Player, tx *world.Tx) { s.showStorage(p, tx, pos, page+1) }})
	}
	if page > 0 {
		actions = append(actions, ui.Option{Label: i18n.T(lang, "storage.prev"), Run: func(p *player.Player, tx *world.Tx) { s.showStorage(p, tx, pos, page-1) }})
	}
	ui.ItemGrid(p, i18n.T(lang, "storage.title"), fmt.Sprintf(i18n.T(lang, "storage.lines"), drawers, len(list), totalOf(list)), slots, actions...)
}

func totalOf(list []storageEntry) int64 {
	var t int64
	for _, e := range list {
		t += e.count
	}
	return t
}
