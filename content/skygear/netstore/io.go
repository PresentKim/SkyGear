package netstore

import (
	"sync"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/itemio"
	"github.com/presentkim/skygear/common/storagenet"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/machine"
)

// Importers, exporters and external storages (spec §4). Each is a job of the
// network's scheduler. The scheduler runs without a world transaction, so a
// job only marks its node ready; the node's own run, which has the
// transaction of its tick, does the work and tells the job whether it did
// any. The rates are Refined Storage 2's (refinedstorage-common at
// d33a173017): importers and exporters work every 9 ticks
// (UpgradeContainer.DEFAULT_WORK_TICK_RATE) and move one item
// (the item transfer strategies' single amount); an external storage looks
// at its block every 5 to 40 ticks, faster while it changes
// (ExternalStorageWorkRate).

const (
	// ioItems is the most items an importer or exporter moves per run.
	ioItems = 1
	// ioRate is the interval of importers and exporters.
	ioRate = 9
	// extMin and extMax are the poll interval of external storages.
	extMin, extMax = 5, 40
)

// ioJob is the state of one node's job.
type ioJob struct {
	// ready is set by the scheduler; worked by the node run, read by the
	// next scheduler run.
	ready, worked bool
}

// ioRT is the io state of one network.
type ioRT struct {
	// tx is the transaction of the work in progress (nil between runs).
	tx   *world.Tx
	jobs map[cube.Pos]*ioJob
	ext  map[cube.Pos]*extSource
}

var (
	rtMu sync.Mutex
	rts  = map[*Network]*ioRT{}
)

func rtOf(n *Network) *ioRT {
	rtMu.Lock()
	defer rtMu.Unlock()
	return rts[n]
}

// cur is the transaction external storages use: the one bound by At or Touch,
// if it is still open. A closed transaction panics on World(), which is how
// a stale one is told from the live one (the world runs one transaction at a
// time, so an open one is the current one). A stale tx is dropped, so no
// external storage ever touches the world through a transaction that ended.
func (rt *ioRT) cur() (tx *world.Tx) {
	tx = rt.tx
	if tx == nil {
		return nil
	}
	defer func() {
		if recover() != nil {
			rt.tx, tx = nil, nil
		}
	}()
	tx.World()
	return tx
}

// bind makes tx the transaction of the external storages of n. At calls it
// for every entry point (terminal, crafting, item I/O of the controller, node
// runs), so a Source works anywhere a network is reached, not only in node
// runs. cur guards against a transaction that ended, and runController
// unbinds when its run ends.
func bind(n *Network, tx *world.Tx) {
	if rt := rtOf(n); rt != nil {
		rt.tx = tx
	}
}

// Touch gives the external storages of n the transaction tx until the
// returned function runs: item I/O on the blocks they face needs one. At
// binds it already; Touch also restores the previous one.
//
//	defer netstore.Touch(n, tx)()
func Touch(n *Network, tx *world.Tx) func() {
	rt := rtOf(n)
	if rt == nil {
		return func() {}
	}
	prev := rt.tx
	rt.tx = tx
	return func() { rt.tx = prev }
}

// front is the block a node acts on and the face of it the items pass
// through: the side the node faces (machine.State.Facing; the autocrafter's
// front texture is on its south side, where it always faces). Every node that
// acts on its front (importer, exporter, external storage, autocrafter push)
// takes it from here.
func front(tx *world.Tx, pos cube.Pos) (cube.Pos, cube.Face) {
	f := cube.FaceSouth
	if st := stateAt(tx, pos); st != nil {
		f = st.Facing
	}
	return pos.Side(f), f.Opposite()
}

func jobID(pos cube.Pos) string { return "io@" + configKey(pos) }

func init() {
	onBuild = append(onBuild, buildIO)
	onDispose = append(onDispose, func(n *Network) {
		rtMu.Lock()
		delete(rts, n)
		rtMu.Unlock()
	})
}

func buildIO(tx *world.Tx, n *Network) {
	rt := &ioRT{tx: tx, jobs: map[cube.Pos]*ioJob{}, ext: map[cube.Pos]*extSource{}}
	rtMu.Lock()
	rts[n] = rt
	rtMu.Unlock()
	defer func() { rt.tx = nil }()
	add := func(pos cube.Pos, lo, hi int64) {
		j := &ioJob{}
		rt.jobs[pos] = j
		n.Sched.Add(jobID(pos), lo, hi, n.Tick, func(int64) bool {
			w := j.worked
			j.worked, j.ready = false, true
			return w
		})
	}
	for _, k := range []string{machine.KindNetImporter, machine.KindNetExporter} {
		for _, pos := range n.Nodes(k) {
			rate := ioRateOf(stateAt(tx, pos))
			add(pos, rate, rate)
		}
	}
	for _, pos := range n.Nodes(machine.KindNetExternal) {
		add(pos, extMin, extMax)
		addExternal(tx, n, rt, pos)
	}
}

// ioRateOf is the ticks between an importer's or exporter's operations: 9,
// less 2 a speed upgrade (UpgradeContainer.updateTicker).
func ioRateOf(st *machine.State) int64 {
	speed, _ := upgradesOf(st)
	return max(1, ioRate-2*int64(speed))
}

// ioAmountOf is how many items an importer or exporter moves an operation:
// a stack with the stack upgrade (the transfer quota providers).
func ioAmountOf(st *machine.State) int {
	if _, stack := upgradesOf(st); stack > 0 {
		return ioItems * 64
	}
	return ioItems
}

// mirrored reports whether an external storage of the network faces target:
// an importer or exporter there would only move its items out of the
// network and back in.
func (rt *ioRT) mirrored(target cube.Pos) bool {
	for _, e := range rt.ext {
		if e.target == target {
			return true
		}
	}
	return false
}

// nodeRun is the run of an importer, exporter or external storage.
func nodeRun(_ *core.Session, _ *machine.State, pos cube.Pos, tx *world.Tx) {
	n := At(tx, pos)
	if n == nil {
		return
	}
	rt := rtOf(n)
	if rt == nil {
		return
	}
	j := rt.jobs[pos]
	if j == nil || !j.ready || !n.Online() {
		return
	}
	j.ready = false
	defer Touch(n, tx)()
	var did bool
	switch nodeAt(tx, pos) {
	case machine.KindNetImporter:
		did = importWork(tx, n, rt, pos)
	case machine.KindNetExporter:
		did = exportWork(tx, n, rt, pos)
	case machine.KindNetExternal:
		did = pollExternal(n, rt, pos)
	}
	j.worked = did
}

// importWork moves up to ioAmountOf items from the block in front into the
// network; it reports whether any moved.
func importWork(tx *world.Tx, n *Network, rt *ioRT, pos cube.Pos) bool {
	target, face := front(tx, pos)
	if nodeAt(tx, target) != "" || rt.mirrored(target) {
		return false
	}
	amount := ioAmountOf(stateAt(tx, pos))
	f := Config(n.Sess, pos).filterOf()
	var refused []item.Stack
	for range 4 {
		peek := itemio.Peek(tx, target, face, func(st item.Stack) bool {
			for _, r := range refused {
				if st.Comparable(r) {
					return false
				}
			}
			return f == nil || f(string(KeyOf(st)))
		})
		if peek.Empty() {
			return false
		}
		got := itemio.Extract(tx, target, face, peek, min(amount, peek.Count()))
		if got.Empty() {
			return false
		}
		took := int(netInsert(n, got))
		if rest := got.Count() - took; rest > 0 {
			back := got.Grow(rest - got.Count())
			if left := rest - itemio.Insert(tx, target, face, back); left > 0 {
				n.Sess.DropAt(tx, target.Vec3Centre(), back.Grow(left-back.Count()))
			}
		}
		if took > 0 {
			return true
		}
		refused = append(refused, peek)
	}
	return false
}

// exportWork moves up to ioAmountOf items of the first kind of the filter, in
// its order, that the network holds and the block in front takes, as
// Refined Storage's exporter (its filter is the list to export, whatever the
// mode). An empty filter exports nothing.
func exportWork(tx *world.Tx, n *Network, rt *ioRT, pos cube.Pos) bool {
	target, face := front(tx, pos)
	if nodeAt(tx, target) != "" || rt.mirrored(target) {
		return false
	}
	amount := ioAmountOf(stateAt(tx, pos))
	for _, f := range Config(n.Sess, pos).Filter {
		k := storagenet.Key(f)
		proto := Proto(n.Sess, k)
		if proto.Empty() {
			continue
		}
		want := int64(min(amount, proto.MaxCount()))
		if n.Root.Extract(k, want, true) == 0 {
			continue
		}
		got := int(n.Root.Extract(k, want, false))
		if got == 0 {
			continue
		}
		took := itemio.Insert(tx, target, face, proto.Grow(got-1))
		if took < got {
			n.Root.Insert(k, int64(got-took), false)
		}
		if took > 0 {
			return true
		}
	}
	return false
}
