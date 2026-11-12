package netstore

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/machine"
)

// registerKinds hands the machine frame the behaviour of the storage
// network's machine kinds. The cable only joins nodes and has none of its
// own. Every node that was broken drops its network and forgets its setting
// (nodeBroken); the pattern encoder is no node.
func registerKinds() {
	machine.Register(machine.KindNetController, machine.Behaviour{
		Click: clickController, Run: runController,
		Insert: insertAt, Peek: peekAt, Extract: extractAt,
		Broken: nodeBroken,
	})
	machine.Register(machine.KindNetCable, machine.Behaviour{Broken: nodeBroken})
	machine.Register(machine.KindDiskDrive, machine.Behaviour{Accepts: acceptsDisk, Broken: nodeBroken})
	machine.Register(machine.KindNetTerminal, machine.Behaviour{
		Click:  clickTerminal,
		Insert: insertAt, Peek: peekAt, Extract: extractAt,
		Broken: nodeBroken,
	})
	machine.Register(machine.KindNetImporter, machine.Behaviour{Click: clickFilter, Run: nodeRun, Accepts: acceptsUpgrade, Broken: nodeBroken})
	machine.Register(machine.KindNetExporter, machine.Behaviour{Click: clickFilter, Run: nodeRun, Accepts: acceptsUpgrade, Broken: nodeBroken})
	machine.Register(machine.KindNetExternal, machine.Behaviour{Click: clickFilter, Run: nodeRun, Broken: nodeBroken})
	machine.Register(machine.KindPatternEncoder, machine.Behaviour{Click: clickEncoder, Accepts: encoderAccepts})
	machine.Register(machine.KindNetAutocrafter, machine.Behaviour{Run: runAutocrafter, Accepts: acceptsPattern, Broken: nodeBroken})
}

// nodeBroken drops the network of the node that was broken at pos, by a
// player or by an explosion, and forgets the node's setting. The block is
// gone when it is called.
func nodeBroken(s *core.Session, tx *world.Tx, pos cube.Pos) {
	Invalidate(tx, pos)
	Forget(s, pos)
}
