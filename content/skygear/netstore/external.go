package netstore

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/itemio"
	"github.com/presentkim/skygear/common/storagenet"
)

// extSource is the block in front of an external storage as a
// storagenet.Source. Its calls need the transaction rt.cur gives (bound by At
// or Touch); without a live one it lists nothing and takes and gives none.
type extSource struct {
	n      *Network
	rt     *ioRT
	target cube.Pos
	face   cube.Face
	ext    *storagenet.External
}

func extID(pos cube.Pos) string { return "ext@" + configKey(pos) }

// addExternal puts the external storage at pos in n.Root.
func addExternal(tx *world.Tx, n *Network, rt *ioRT, pos cube.Pos) {
	target, face := front(tx, pos)
	if nodeAt(tx, target) != "" {
		return // a network node in front: nothing to mirror
	}
	src := &extSource{n: n, rt: rt, target: target, face: face}
	src.ext = storagenet.NewExternal(src)
	rt.ext[pos] = src
	cfg := Config(n.Sess, pos)
	var filter func(storagenet.Key) bool
	if f := cfg.filterOf(); f != nil {
		filter = func(k storagenet.Key) bool { return f(string(k)) }
	}
	n.Root.Add(extID(pos), src.ext, cfg.Priority, filter)
}

// pollExternal rereads the block in front of the external storage at pos.
func pollExternal(n *Network, rt *ioRT, pos cube.Pos) bool {
	src := rt.ext[pos]
	if src == nil || !src.ext.Poll() {
		return false
	}
	n.Root.Resync(extID(pos))
	return true
}

func (e *extSource) List() map[storagenet.Key]int64 {
	out := map[storagenet.Key]int64{}
	tx := e.rt.cur()
	if tx == nil {
		return out
	}
	for _, st := range itemio.List(tx, e.target, e.face) {
		out[KeyOf(st)] += int64(st.Count())
	}
	delete(out, "")
	return out
}

func (e *extSource) Insert(k storagenet.Key, n int64, sim bool) int64 {
	tx := e.rt.cur()
	if tx == nil || n <= 0 {
		return 0
	}
	if in, _ := itemio.IO(tx, e.target); in == nil {
		return 0
	}
	if sim {
		return n // the block cannot say how much it would take
	}
	proto := Proto(e.n.Sess, k)
	if proto.Empty() {
		return 0
	}
	var done int64
	for done < n {
		chunk := int(min(n-done, int64(proto.MaxCount())))
		took := itemio.Insert(tx, e.target, e.face, proto.Grow(chunk-1))
		done += int64(took)
		if took < chunk {
			break
		}
	}
	return done
}

func (e *extSource) Extract(k storagenet.Key, n int64, sim bool) int64 {
	tx := e.rt.cur()
	if tx == nil || n <= 0 {
		return 0
	}
	proto := Proto(e.n.Sess, k)
	if proto.Empty() {
		return 0
	}
	if sim {
		for _, st := range itemio.List(tx, e.target, e.face) {
			if st.Comparable(proto) {
				return min(n, int64(st.Count()))
			}
		}
		return 0
	}
	var done int64
	for range 64 {
		if done >= n {
			break
		}
		got := itemio.Extract(tx, e.target, e.face, proto, int(min(n-done, int64(proto.MaxCount()))))
		if got.Empty() {
			break
		}
		done += int64(got.Count())
	}
	return done
}
