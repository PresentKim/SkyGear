package netstore

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/storagenet"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/machine"
)

// Hooks for the parts of the storage network that other files add. They are
// plain package variables, appended to from init.
//
//	onBuild runs when a network has just been built and is structurally
//	valid (one controller, within NodeCap): add storages to n.Root, jobs to
//	n.Sched, and read nodes with n.Nodes(kind).
//	onDispose runs when a network is dropped (a member changed): undo what
//	onBuild did that is not owned by n.
var (
	onBuild   []func(tx *world.Tx, n *Network)
	onDispose []func(n *Network)
)

const (
	// checkEvery is how often (in controller runs) a network confirms that its
	// nodes are still where it thinks.
	checkEvery = 200
	// driveEvery is how often drive slots are compared with the Root.
	driveEvery = 10
	// flushEvery is the most often (in controller runs) changed disks are
	// copied to the save.
	flushEvery = 20
)

// Network is the storage network around one controller.
type Network struct {
	// Root is the combined storage of the network; Sched wakes the importers,
	// exporters and external storages. Sched runs on Tick, the number of
	// times the controller has run.
	Root  *storagenet.Root
	Sched *storagenet.Scheduler
	Tick  int64
	// Controller is the controller, valid when Controllers is 1.
	anchor      cube.Pos
	Controller  cube.Pos
	Controllers int
	// Sess is the session of the network.
	Sess *core.Session

	members map[cube.Pos]string
	byKind  map[string][]cube.Pos
	// use is the RE per tick the network takes; powered is whether the
	// controller paid it at its last run.
	use     int
	powered bool
	// attached are the disks put in Root by drive slot (Root id → what).
	attached map[string]attachment
	dirty    bool
	unlisten func()
}

type attachment struct {
	disk string
	prio int
}

// Online reports whether the network works: one controller, within NodeCap,
// and the controller paid its RE use at its last run. An offline network
// keeps its items but takes and gives none.
func (n *Network) Online() bool { return n.alive() && n.valid() && n.powered }

// alive reports whether n is still the network of its nodes: a network that was
// dropped (Invalidate, a failed check) may still be held by an open screen, and
// must neither move items nor touch tasks, as its successor owns them now.
func (n *Network) alive() bool { return localOf(n.Sess).of[n.anchor] == n }

func (n *Network) valid() bool { return n.Controllers == 1 && len(n.members) <= NodeCap }

// Nodes are the positions of the nodes of a kind, in x, y, z order.
func (n *Network) Nodes(kind string) []cube.Pos { return n.byKind[kind] }

// Size is the number of nodes in the network.
func (n *Network) Size() int { return len(n.members) }

// Use is the RE per tick the network takes.
func (n *Network) Use() int { return n.use }

// local is the networks of a session, by every member position.
type local struct{ of map[cube.Pos]*Network }

var localPart = core.NewPart[local](nil)

func localOf(s *core.Session) *local {
	l := localPart.Of(s)
	if l.of == nil {
		l.of = map[cube.Pos]*Network{}
	}
	return l
}

// At is the network the node at pos belongs to, built on first use and kept
// until Invalidate drops it; nil if pos is no node. It also binds tx to the
// network's external storages (see bind), so they work from every entry
// point. The binding cannot go stale: ioRT.cur uses it only while tx is open
// and runController drops it at the end of its run.
func At(tx *world.Tx, pos cube.Pos) *Network {
	s, ok := core.Of(tx)
	if !ok {
		return nil
	}
	l := localOf(s)
	n := l.of[pos]
	if n == nil {
		if nodeAt(tx, pos) == "" {
			return nil
		}
		n = build(tx, s, l, pos)
	}
	if n != nil {
		bind(n, tx)
	}
	return n
}

func build(tx *world.Tx, s *core.Session, l *local, pos cube.Pos) *Network {
	members := flood(tx, pos)
	n := &Network{
		anchor: pos,
		Root:   storagenet.NewRoot(), Sched: storagenet.NewScheduler(), Sess: s,
		members: members, byKind: map[string][]cube.Pos{}, attached: map[string]attachment{},
	}
	for p, k := range members {
		n.byKind[k] = append(n.byKind[k], p)
		n.use += nodeUse(k)
		l.of[p] = n
	}
	for _, ps := range n.byKind {
		sortPos(ps)
	}
	n.Controllers = len(n.byKind[machine.KindNetController])
	if n.Controllers > 0 {
		n.Controller = n.byKind[machine.KindNetController][0]
	}
	if n.Controllers == 1 {
		n.use += controllerUse
		if st := stateAt(tx, n.Controller); st != nil {
			n.powered = st.Unlimited || st.Energy >= n.use
		}
	}
	n.unlisten = n.Root.Listen(func(storagenet.Key, int64) { n.dirty = true })
	if n.valid() {
		n.syncDrives(tx)
		for _, f := range onBuild {
			f(tx, n)
		}
	}
	return n
}

// dispose forgets the network.
func (n *Network) dispose(l *local) {
	for p := range n.members {
		if l.of[p] == n {
			delete(l.of, p)
		}
	}
	for _, f := range onDispose {
		f(n)
	}
	n.unlisten()
	n.flush()
}

// flush copies the disks to the save if anything changed.
func (n *Network) flush() {
	if n.dirty {
		n.dirty = false
		Disks(n.Sess).Flush()
	}
}

// Invalidate drops the networks of pos and of the nodes next to it; call it
// when a block at pos was placed or broken. Networks are rebuilt by the next
// At.
func Invalidate(tx *world.Tx, pos cube.Pos) {
	s, ok := core.Of(tx)
	if !ok {
		return
	}
	l := localOf(s)
	for _, p := range append([]cube.Pos{pos}, neighbours(pos)...) {
		if n := l.of[p]; n != nil {
			n.dispose(l)
		}
	}
}

func neighbours(pos cube.Pos) []cube.Pos {
	out := make([]cube.Pos, 0, 6)
	for _, f := range cube.Faces() {
		out = append(out, pos.Side(f))
	}
	return out
}

// verify reports whether every member is still a node of its kind.
func (n *Network) verify(tx *world.Tx) bool {
	for p, k := range n.members {
		if nodeAt(tx, p) != k {
			return false
		}
	}
	return true
}

// syncDrives puts the disks in the drive slots in Root and takes out those
// that left. A disk without an id gets one here.
func (n *Network) syncDrives(tx *world.Tx) {
	ds := Disks(n.Sess)
	seen := map[string]bool{}
	used := map[string]string{} // disk id → Root id
	for _, pos := range n.byKind[machine.KindDiskDrive] {
		st := stateAt(tx, pos)
		if st == nil {
			continue
		}
		cfg := Config(n.Sess, pos)
		for slot := range 8 {
			rid := driveID(pos, slot)
			stack, _ := st.Inv().Item(slot)
			d, isDisk := stack.Item().(Disk)
			if !isDisk {
				continue
			}
			id := DiskID(stack)
			if id == "" {
				id = ds.NewID()
				_ = st.Inv().SetItem(slot, WithDiskID(stack, id))
			}
			if other, dup := used[id]; dup && other != rid {
				continue // the same disk twice: only the first counts
			}
			used[id] = rid
			seen[rid] = true
			at := attachment{disk: id, prio: cfg.Priority}
			if n.attached[rid] == at {
				continue
			}
			var filter func(storagenet.Key) bool
			if f := cfg.filterOf(); f != nil {
				filter = func(k storagenet.Key) bool { return f(string(k)) }
			}
			n.Root.Add(rid, ds.Open(id, DiskCaps[max(0, min(d.Tier, len(DiskCaps)-1))]), cfg.Priority, filter)
			n.attached[rid] = at
		}
	}
	for rid := range n.attached {
		if !seen[rid] {
			n.Root.Remove(rid)
			delete(n.attached, rid)
		}
	}
}

func driveID(pos cube.Pos, slot int) string {
	return "drive@" + configKey(pos) + "#" + string(rune('0'+slot))
}

// runController is the work of a controller each tick: it pays the RE use of
// its network, and while paid runs the drive check, the scheduler and the
// periodic checks.
func runController(s *core.Session, st *machine.State, pos cube.Pos, tx *world.Tx) {
	n := At(tx, pos)
	if n == nil {
		return
	}
	defer func() {
		if rt := rtOf(n); rt != nil {
			rt.tx = nil
		}
	}()
	n.Tick++
	if n.Controllers != 1 || n.Controller != pos {
		n.powered, st.Stalled = false, true
		return
	}
	if n.Tick%checkEvery == 0 && !n.verify(tx) {
		n.dispose(localOf(s))
		return
	}
	switch {
	case st.Unlimited:
		n.powered = true
	case st.Energy >= n.use:
		st.Energy -= n.use
		n.powered = true
	default:
		n.powered = false
	}
	st.Stalled = !n.powered
	if !n.valid() || !n.powered {
		return
	}
	if n.Tick%driveEvery == 0 {
		n.syncDrives(tx)
	}
	n.Sched.Run(n.Tick)
	if n.Tick%flushEvery == 0 {
		n.flush()
	}
}
