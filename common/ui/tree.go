package ui

import (
	"strings"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// TreeMenu is a form with a tree (docs/tech/advancement-tree.md): the tree is
// laid out by the server on a grid of cells (a column per depth, a row per
// leaf) and the form shows all of it, at most TreeCols×TreeRows cells. Every cell may have a node (a frame with an icon, its
// name and text in the tooltip) and the connector pieces in its top strip.
const (
	// TreeToken starts the title of a TreeMenu.
	TreeToken = "§d§f§z§r"
	// TreeCols and TreeRows are the size of the window onto the tree.
	TreeCols  = 11
	TreeRows  = 21
	TreeCells = TreeCols * TreeRows

	treeCellToken = "§0§v§r"
)

// Strip bits of a TreeCell: where the connector lines of its left strip go
// (a vertical bus in the middle, a line to the left edge and to the node).
const (
	StripUp = 1 << iota
	StripDown
	StripLeft
	StripRight
)

// Frame and state letters of a node (the texture is tree_node_<frame><state>).
const (
	FrameTask      = 't'
	FrameGoal      = 'g'
	FrameChallenge = 'c'
	StateLocked    = 'l'
	StateOpen      = 'o'
	StateDone      = 'd'
)

// TreeCell is one cell of the window. Frame 0 means there is no node (the
// cell may still carry connector lines).
type TreeCell struct {
	Strip        int
	Frame, State byte
	Icon, Tip    string
	Selected     bool
	Run          func(p *player.Player, tx *world.Tx)
}

// Tree is the content of a TreeMenu.
type Tree struct {
	// Tabs are the tab buttons over the tree (Tab).
	Tabs []Option
	// Cells are the grid, row by row, at most TreeCells; the rest of the grid
	// is empty cells (the grid takes the first TreeCells buttons, so every
	// one of them must be a cell, or the buttons after them would show up as
	// broken cells).
	Cells []TreeCell
	// Actions are the title bar buttons (Back).
	Actions []Option
	// Body is the line over the tree.
	Body string
}

// treeNodeLetter is the letter that names a node's frame picture
// (tree_node_<letter>): 'a'+3*frame+state, 'z' for a cell without a node.
func treeNodeLetter(frame, state byte) byte {
	if frame == 0 {
		return 'z'
	}
	f := strings.IndexByte("tgc", frame)
	st := strings.IndexByte("lod", state)
	if f < 0 || st < 0 {
		return 'z'
	}
	return byte('a' + 3*f + st)
}

// treeLabel is the button text of a cell: the marker, then fixed bytes the
// pack reads by position (strip letter 'a'+bits at byte 9, node letter at
// 10, chosen flag 's' at 11), '~' up to byte 20, then the tooltip. Each is a
// single letter so the pack builds a texture name from one piece.
func treeLabel(c TreeCell) string {
	sel := byte('-')
	if c.Selected {
		sel = 's'
	}
	head := string([]byte{byte('a' + c.Strip&15), treeNodeLetter(c.Frame, c.State), sel})
	return treeCellToken + head + strings.Repeat("~", 8) + c.Tip
}

// TreeMenu sends the form. The cells come first among the buttons; any press
// answers the form, so Run must show a form again.
func TreeMenu(p *player.Player, title string, t Tree) {
	nop := func(*player.Player, *world.Tx) {}
	opts := make([]Option, 0, TreeCells+len(t.Actions)+len(t.Tabs))
	for i := range TreeCells {
		var c TreeCell
		if i < len(t.Cells) {
			c = t.Cells[i]
		}
		run := c.Run
		if run == nil {
			run = nop
		}
		opts = append(opts, Option{Label: treeLabel(c), Icon: c.Icon, Run: run})
	}
	opts = append(opts, actionOptions(t.Actions)...)
	opts = append(opts, t.Tabs...)
	Menu(p, TreeToken+title, t.Body, opts...)
}
