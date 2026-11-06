package netstore

import (
	"fmt"
	"sort"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/machine"
)

// nodeKinds are the machine kinds that belong to a storage network: each
// conducts to every side-adjacent node. The pattern encoder is not one; it
// writes patterns and needs no network.
var nodeKinds = machine.NetNodes

// IsNode reports whether kind is a storage network node.
func IsNode(kind string) bool { return nodeKinds[kind] }

// nodeUse is the RE per tick a node adds to the network's use: 1 for a
// device, 2 for a disk drive, 4 for an autocrafter, none for a cable or the
// controller (whose base use is controllerUse).
func nodeUse(kind string) int {
	switch kind {
	case machine.KindNetCable, machine.KindNetController:
		return 0
	case machine.KindDiskDrive:
		return 2
	case machine.KindNetAutocrafter:
		return 4
	}
	return 1
}

const (
	// controllerUse is the RE per tick of the controller itself.
	controllerUse = 8
	// NodeCap is the most nodes (controller and cables included) a network
	// has before it goes offline.
	NodeCap = 512
)

// nodeAt is the kind of the network node at pos ("" for anything else).
func nodeAt(tx *world.Tx, pos cube.Pos) string {
	if m, ok := tx.Block(pos).(machine.Machine); ok && nodeKinds[m.Kind] {
		return m.Kind
	}
	return ""
}

// stateAt is the machine state of the node at pos.
func stateAt(tx *world.Tx, pos cube.Pos) *machine.State {
	if m, ok := tx.Block(pos).(machine.Machine); ok {
		return m.State
	}
	return nil
}

// flood is the nodes joined to pos, which must be a node: every node
// reached over side-adjacent nodes, by kind. It reads the world only.
func flood(tx *world.Tx, pos cube.Pos) map[cube.Pos]string {
	out := map[cube.Pos]string{pos: nodeAt(tx, pos)}
	queue := []cube.Pos{pos}
	for len(queue) > 0 {
		at := queue[0]
		queue = queue[1:]
		for _, f := range cube.Faces() {
			next := at.Side(f)
			if _, seen := out[next]; seen {
				continue
			}
			if k := nodeAt(tx, next); k != "" {
				out[next] = k
				queue = append(queue, next)
			}
		}
	}
	return out
}

func sortPos(ps []cube.Pos) {
	sort.Slice(ps, func(i, j int) bool {
		a, b := ps[i], ps[j]
		if a[0] != b[0] {
			return a[0] < b[0]
		}
		if a[1] != b[1] {
			return a[1] < b[1]
		}
		return a[2] < b[2]
	})
}

// NodeConfig is the setting of one node. Priority orders storages (drives,
// external storages); Filter lists keys, which Allow turns from a block list
// into an allow list.
type NodeConfig struct {
	Priority int      `json:"p,omitempty"`
	Filter   []string `json:"f,omitempty"`
	Allow    bool     `json:"a,omitempty"`
}

// configState is the node settings in the save, by "x,y,z".
type configState struct {
	Nodes map[string]*NodeConfig `json:"net_nodes,omitempty"`
}

var configSaved = core.NewSaved[configState](nil)

func configKey(pos cube.Pos) string { return fmt.Sprintf("%d,%d,%d", pos[0], pos[1], pos[2]) }

// Config is the setting of the node at pos, made with defaults on first use.
// The caller may change it; MarkDirty on the session stores the change.
func Config(s *core.Session, pos cube.Pos) *NodeConfig {
	st := configSaved.Of(s)
	if st.Nodes == nil {
		st.Nodes = map[string]*NodeConfig{}
	}
	c := st.Nodes[configKey(pos)]
	if c == nil {
		c = &NodeConfig{}
		st.Nodes[configKey(pos)] = c
		s.MarkDirty()
	}
	return c
}

// Forget drops the setting of the node at pos (the block was broken).
func Forget(s *core.Session, pos cube.Pos) {
	st := configSaved.Of(s)
	if _, ok := st.Nodes[configKey(pos)]; ok {
		delete(st.Nodes, configKey(pos))
		s.MarkDirty()
	}
}

// filterOf is the key filter of a setting (nil: everything passes).
func (c *NodeConfig) filterOf() func(k string) bool {
	if len(c.Filter) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(c.Filter))
	for _, f := range c.Filter {
		set[f] = struct{}{}
	}
	return func(k string) bool {
		_, in := set[k]
		return in == c.Allow
	}
}
