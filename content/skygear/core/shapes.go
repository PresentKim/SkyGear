package core

import (
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/debug"
	"github.com/df-mc/dragonfly/server/world"
)

// Shapes and text drawn in the air with the client's debug shape packet
// (PrimitiveShapes, Dragonfly's player/debug). They are client-side only and
// per player: shapes for everyone in the save are kept here and sent to each
// player who joins (SendShapes), and removed from players who leave.

var shapes = NewPart(func() *map[any]debug.Shape {
	m := map[any]debug.Shape{}
	return &m
})

// ShowShape shows a shape to every player in the save, replacing the shape
// stored under key.
func (s *Session) ShowShape(tx *world.Tx, key any, sh debug.Shape) {
	m := *shapes.Of(s)
	if old, ok := m[key]; ok && old != sh {
		forPlayers(tx, func(p *player.Player) { p.RemoveDebugShape(old) })
	}
	m[key] = sh
	forPlayers(tx, func(p *player.Player) { p.AddDebugShape(sh) })
}

// HideShape removes the shape stored under key from every player.
func (s *Session) HideShape(tx *world.Tx, key any) {
	m := *shapes.Of(s)
	sh, ok := m[key]
	if !ok {
		return
	}
	delete(m, key)
	forPlayers(tx, func(p *player.Player) { p.RemoveDebugShape(sh) })
}

// Shape is the shape stored under key, if any.
func (s *Session) Shape(key any) (debug.Shape, bool) {
	sh, ok := (*shapes.Of(s))[key]
	return sh, ok
}

// SendShapes shows the save's shapes to a player who joined.
func (s *Session) SendShapes(p *player.Player) {
	for _, sh := range *shapes.Of(s) {
		p.AddDebugShape(sh)
	}
}

func forPlayers(tx *world.Tx, f func(p *player.Player)) {
	if tx == nil {
		return
	}
	for e := range tx.Players() {
		f(e.(*player.Player))
	}
}
