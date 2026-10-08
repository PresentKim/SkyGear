package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/debug"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// Shapes and text drawn in the air with the client's debug shape packet
// (PrimitiveShapes, Dragonfly's player/debug). They are client-side only and
// per player: shapes for everyone in the instance are kept here and sent to
// each player who joins, and removed from players who leave.

// showShape shows a shape to every player in the instance, replacing the
// shape stored under key.
func (s *session) showShape(tx *world.Tx, key any, sh debug.Shape) {
	if s.shapes == nil {
		s.shapes = map[any]debug.Shape{}
	}
	if old, ok := s.shapes[key]; ok && old != sh {
		s.forPlayers(tx, func(p *player.Player) { p.RemoveDebugShape(old) })
	}
	s.shapes[key] = sh
	s.forPlayers(tx, func(p *player.Player) { p.AddDebugShape(sh) })
}

// hideShape removes the shape stored under key from every player.
func (s *session) hideShape(tx *world.Tx, key any) {
	sh, ok := s.shapes[key]
	if !ok {
		return
	}
	delete(s.shapes, key)
	s.forPlayers(tx, func(p *player.Player) { p.RemoveDebugShape(sh) })
}

// sendShapes shows the instance shapes to a player who joined.
func (s *session) sendShapes(p *player.Player) {
	for _, sh := range s.shapes {
		p.AddDebugShape(sh)
	}
}

func (s *session) forPlayers(tx *world.Tx, f func(p *player.Player)) {
	if tx == nil {
		return
	}
	for e := range tx.Players() {
		f(e.(*player.Player))
	}
}

// drawerLabelKey, drawerLockKey and drawerUpgradeKey key the count, lock and
// upgrade texts of a drawer.
type (
	drawerLabelKey   cube.Pos
	drawerLockKey    cube.Pos
	drawerUpgradeKey cube.Pos
)

// textYawOffset turns front-face texts so they read from the front. It is
// added to the entity yaw looking out of the face; flip it by 180 if texts
// show mirrored on real clients.
const textYawOffset = 0

// faceText returns a text fixed flat on a drawer's front face, without a
// background, at (right, up) from the face centre as the viewer sees it.
// It sits slightly in front of the face so it does not z-fight with it.
func faceText(pos cube.Pos, d Drawer, right, up, scale float64, text string) *debug.Text {
	front := cube.Pos{}.Side(d.Facing.Face()).Vec3()
	// Looking at the front, the viewer's right is the facing turned left.
	side := cube.Pos{}.Side(d.Facing.RotateLeft().Face()).Vec3()
	at := pos.Vec3Centre().Add(front.Mul(0.53)).Add(side.Mul(right)).Add(mgl64.Vec3{0, up, 0})
	return &debug.Text{
		Position: at, Text: text, Scale: scale, HideBackground: true,
		LockRotation: true, Rotation: mgl64.Vec3{0, yawOf(d.Facing) + textYawOffset, 0},
	}
}

// drawerLabel returns the count text of a drawer, at the bottom of its front.
func drawerLabel(pos cube.Pos, d Drawer) *debug.Text {
	return faceText(pos, d, 0, -0.34, 0.4, d.label())
}

// drawerLock returns the small lock mark of a locked drawer, on the handle
// at the top centre of its front.
func drawerLock(pos cube.Pos, d Drawer) *debug.Text {
	return faceText(pos, d, 0, 0.34, 0.22, lockGlyph)
}

// drawerUpgrades returns the upgrade marks of a drawer (capacity multiplier,
// void), in the top right corner of its front.
func drawerUpgrades(pos cube.Pos, d Drawer) *debug.Text {
	return faceText(pos, d, 0.36, 0.4, 0.2, d.upgradeMarks())
}
