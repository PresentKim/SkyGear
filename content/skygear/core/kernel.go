package core

// KernelFeature is the kernel's own feature (the shapes and parts steps of
// the lifecycle); the game lists it first.
var KernelFeature = &Feature{Name: "core", Setup: subscribeKernel}

// Shapes are sent when a player joins and cleared, client-side, before the
// player leaves the instance world; the core session drops the player's
// per-player parts.
func subscribeKernel() {
	Join.On(40, "core.shapes", func(e Presence) {
		e.S.SendShapes(e.P)
	})
	Leave.On(20, "core.parts", func(e Presence) {
		e.S.Leave(e.P)
	})
	Leave.On(50, "core.shapes", func(e Presence) {
		e.P.RemoveAllDebugShapes()
	})
}
