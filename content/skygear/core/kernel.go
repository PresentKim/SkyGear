package core

// KernelFeature is the kernel's own feature; the game lists it first, so its
// subscriber is the first of its tier. It is a subscriber like any other: it
// does not depend on its place inside the tier.
var KernelFeature = &Feature{Name: "core", Setup: subscribeKernel}

// Shapes are sent when a player joins. What the kernel does when a player
// leaves (clearing the shapes client-side, dropping the player's per-player
// parts) is not a subscriber: Lifecycle.Leave does it when every subscriber
// has run.
func subscribeKernel() {
	Join.On(Normal, "core.shapes", func(e Presence) {
		e.S.SendShapes(e.P)
	})
}
