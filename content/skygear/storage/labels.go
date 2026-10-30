package storage

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player/debug"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/textfmt"
)

// DrawerLabelKey, drawerLockKey and drawerUpgradeKey key the count, lock and
// upgrade texts of a drawer.
type (
	DrawerLabelKey   cube.Pos
	DrawerLockKey    cube.Pos
	DrawerUpgradeKey cube.Pos
)

// faceTextYaw is the yaw in degrees that lays a text flat on a face
// looking to each direction, readable from in front of it. The client reads
// the rotation in degrees ([pitch, yaw, roll]); a yaw of 0 reads from the
// north. Entity yaw (S 0, N 180) mirrors south and north texts.
var faceTextYaw = map[cube.Direction]float64{cube.North: 0, cube.East: -90, cube.South: 180, cube.West: 90}

// FaceTextRotation is the shape rotation of a text on a face looking to
// facing.
func FaceTextRotation(facing cube.Direction) mgl64.Vec3 {
	return mgl64.Vec3{0, faceTextYaw[facing], 0}
}

// faceTextDepth is how far from the block centre the front-face texts
// sit: just in front of the face (0.5), so they lie on it without
// z-fighting.
const faceTextDepth = 0.505

// faceText returns a text fixed flat on a drawer's front face, without a
// background, at (right, up) from the face centre as the viewer sees it.
func faceText(pos cube.Pos, d Drawer, right, up, scale float64, text string) *debug.Text {
	front := cube.Pos{}.Side(d.Facing.Face()).Vec3()
	// Looking at the front, the viewer's right is the facing turned left.
	side := cube.Pos{}.Side(d.Facing.RotateLeft().Face()).Vec3()
	at := pos.Vec3Centre().Add(front.Mul(faceTextDepth)).Add(side.Mul(right)).Add(mgl64.Vec3{0, up, 0})
	return &debug.Text{
		Position: at, Text: text, Scale: scale, HideBackground: true,
		LockRotation: true, Rotation: FaceTextRotation(d.Facing),
	}
}

// drawerLabel returns the count text of a drawer, at the bottom of its front.
func drawerLabel(pos cube.Pos, d Drawer) *debug.Text {
	return faceText(pos, d, 0, -0.34, 0.4, d.label())
}

// drawerMarkUp is the height of the lock and upgrade marks above the
// centre of a drawer front: in the top frame, above the handle.
const drawerMarkUp = 0.43

// drawerLock returns the small lock mark of a locked drawer, at the top
// centre of its front, level with the upgrade marks.
func drawerLock(pos cube.Pos, d Drawer) *debug.Text {
	return faceText(pos, d, 0, drawerMarkUp, 0.16, textfmt.LockGlyph)
}

// drawerUpgrades returns the upgrade marks of a drawer (capacity multiplier,
// void), in the top right corner of its front.
func drawerUpgrades(pos cube.Pos, d Drawer) *debug.Text {
	return faceText(pos, d, 0.36, drawerMarkUp, 0.2, d.UpgradeMarks())
}
