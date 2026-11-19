// Package termlayout is the slots of the storage network terminal's window,
// which the server (netstore) fills and the pack (tools/termuigen) draws.
//
// The insert slots come first: a shift click moves a stack to the first
// empty slot, so what the player puts in flies to them, in the middle of the
// window, and the server puts it into the network. A take slot holds a real
// stack of a kind (as many as one stack, at most what the network holds),
// moved by the client as in any chest; the server takes what left it out of
// the network and fills it again. Its show slot holds the kind's picture
// named with its count, which the pack draws over it with the tooltip.
package termlayout

const (
	Insert0 = 0
	Inserts = 27
	Take0   = Insert0 + Inserts
	Kinds   = 9 * 10
	Show0   = Take0 + Kinds
	Button0 = Show0 + Kinds
	Buttons = 9
	Size    = Button0 + Buttons
)

// The buttons, in the order of their column.
const (
	Prev = Button0 + iota
	Search
	Sort
	Info
	Grid
	Tasks
	TakeMatching
	PutAll
	Next
)
