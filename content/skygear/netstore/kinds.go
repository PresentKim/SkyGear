package netstore

import "github.com/presentkim/skygear/content/skygear/machine"

// registerKinds hands the machine frame the behaviour of the storage
// network's machine kinds. The cable only joins nodes and has none.
func registerKinds() {
	machine.Register(machine.KindNetController, machine.Behaviour{
		Click: clickController, Run: runController,
		Insert: insertAt, Peek: peekAt, Extract: extractAt,
	})
	machine.Register(machine.KindNetCable, machine.Behaviour{})
	machine.Register(machine.KindDiskDrive, machine.Behaviour{Accepts: acceptsDisk})
	machine.Register(machine.KindNetTerminal, machine.Behaviour{
		Click:  clickTerminal,
		Insert: insertAt, Peek: peekAt, Extract: extractAt,
	})
	machine.Register(machine.KindNetImporter, machine.Behaviour{Click: clickFilter, Run: nodeRun, Accepts: acceptsUpgrade})
	machine.Register(machine.KindNetExporter, machine.Behaviour{Click: clickFilter, Run: nodeRun, Accepts: acceptsUpgrade})
	machine.Register(machine.KindNetExternal, machine.Behaviour{Click: clickFilter, Run: nodeRun})
	machine.Register(machine.KindPatternEncoder, machine.Behaviour{Click: clickEncoder, Accepts: encoderAccepts})
	machine.Register(machine.KindNetAutocrafter, machine.Behaviour{Run: runAutocrafter, Accepts: acceptsPattern})
}
