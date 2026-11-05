package netstore

import (
	"sort"
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
// any, so an idle node backs off (4 to 64 ticks, 20 to 200 for an external
// storage) and an exporter wakes when a key it exports arrives.

const (
	// ioItems is the most items an importer or exporter moves per run.
	ioItems = 8
	// ioMin and ioMax are the interval of importers and exporters.
	ioMin, ioMax = 4, 64
	// extMin and extMax are the poll interval of external storages.
	extMin, extMax = 20, 200
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
	tx *world.Tx
	// busy is set while the nodes work: items they put back do not wake
	// exporters.
	busy bool
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
// through. Machines have no facing of their own and blockmodel.FrontCubeProperties
// puts the front texture on the "south" material (+Z), so the front is the
// south side and the items pass through the north face of the block there.
// Every node that acts on its front (importer, exporter, external storage,
// autocrafter push) takes it from here.
func front(pos cube.Pos) (cube.Pos, cube.Face) {
	return pos.Side(cube.FaceSouth), cube.FaceNorth
}

func jobID(pos cube.Pos) string { return "io@" + configKey(pos) }

func init() {
	onBuild = append(onBuild, buildIO)
	onDispose = append(onDispose, func(n *Network) {
		rtMu.Lock()
		delete(rts, n)
		rtMu.Unlock()
	})
	for _, k := range []string{machine.KindNetImporter, machine.KindNetExporter, machine.KindNetExternal} {
		machine.Kinds[k].Run = nodeRun
	}
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
	for _, pos := range n.Nodes(machine.KindNetImporter) {
		add(pos, ioMin, ioMax)
	}
	exporters := map[cube.Pos]func(string) bool{}
	for _, pos := range n.Nodes(machine.KindNetExporter) {
		add(pos, ioMin, ioMax)
		if f := Config(n.Sess, pos).filterOf(); f != nil {
			exporters[pos] = f
		}
	}
	for _, pos := range n.Nodes(machine.KindNetExternal) {
		add(pos, extMin, extMax)
		addExternal(tx, n, rt, pos)
	}
	if len(exporters) > 0 {
		n.Root.Listen(func(k storagenet.Key, delta int64) {
			if delta <= 0 || rt.busy {
				return
			}
			for pos, f := range exporters {
				if f(string(k)) {
					n.Sched.Wake(jobID(pos), n.Tick)
				}
			}
		})
	}
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
	rt.busy = true
	defer func() { rt.busy = false }()
	var did bool
	switch nodeAt(tx, pos) {
	case machine.KindNetImporter:
		did = importWork(tx, n, pos)
	case machine.KindNetExporter:
		did = exportWork(tx, n, pos)
	case machine.KindNetExternal:
		did = pollExternal(n, rt, pos)
	}
	j.worked = did
}

// importWork moves up to ioItems items from the block in front into the
// network; it reports whether any moved.
func importWork(tx *world.Tx, n *Network, pos cube.Pos) bool {
	target, face := front(pos)
	if nodeAt(tx, target) != "" {
		return false
	}
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
		got := itemio.Extract(tx, target, face, peek, min(ioItems, peek.Count()))
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

// exportWork moves up to ioItems items of the filtered kinds from the
// network into the block in front. An empty filter exports nothing.
func exportWork(tx *world.Tx, n *Network, pos cube.Pos) bool {
	cfg := Config(n.Sess, pos)
	f := cfg.filterOf()
	if f == nil {
		return false
	}
	target, face := front(pos)
	if nodeAt(tx, target) != "" {
		return false
	}
	var keys []storagenet.Key
	if cfg.Allow {
		for _, k := range cfg.Filter {
			keys = append(keys, storagenet.Key(k))
		}
	} else {
		for k := range n.Root.Totals() {
			if f(string(k)) {
				keys = append(keys, k)
			}
		}
		sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	}
	budget, moved := ioItems, 0
	for _, k := range keys {
		if budget <= 0 {
			break
		}
		proto := Proto(n.Sess, k)
		if proto.Empty() {
			continue
		}
		got := int(n.Root.Extract(k, int64(min(budget, proto.MaxCount())), false))
		if got == 0 {
			continue
		}
		took := itemio.Insert(tx, target, face, proto.Grow(got-1))
		if took < got {
			n.Root.Insert(k, int64(got-took), false)
		}
		budget -= took
		moved += took
	}
	return moved > 0
}
