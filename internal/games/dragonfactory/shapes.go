package dragonfactory

import (
	"fmt"
	"image/color"
	"time"

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

// Chunk borders: shown to one player while the chunk map is used, and for
// a while after, so they can see where the island ends in the world.

const chunkBorderTime = 30 * time.Second

var (
	ownedColour   = color.RGBA{G: 255, A: 255}
	buyableColour = color.RGBA{R: 255, G: 255, A: 255}
)

type chunkBorders struct {
	shapes []debug.Shape
	shown  time.Time
}

// showChunkBorders draws the owned (green) and buyable (yellow) chunks
// around the island surface for p, replacing earlier borders, and hides
// them after chunkBorderTime.
func (s *session) showChunkBorders(p *player.Player) {
	if s.borders == nil {
		s.borders = map[*world.EntityHandle]*chunkBorders{}
	}
	s.hideChunkBorders(p)
	b := &chunkBorders{shown: time.Now()}
	add := func(sh debug.Shape) {
		b.shapes = append(b.shapes, sh)
		p.AddDebugShape(sh)
	}
	// Two boxes a little apart make the thin debug lines read as a band.
	box := func(c ChunkPos, col color.RGBA) {
		for _, inset := range []float64{0, 0.15} {
			add(&debug.Box{
				Colour:   col,
				Position: mgl64.Vec3{float64(c[0])*16 + inset, IslandY - 1, float64(c[1])*16 + inset},
				Bounds:   mgl64.Vec3{16 - 2*inset, 8, 16 - 2*inset},
			})
		}
	}
	for _, c := range s.state.Chunks {
		box(c, ownedColour)
	}
	price := formatMoney(chunkPrice(len(s.state.Chunks)))
	for _, c := range s.purchasableChunks() {
		box(c, buyableColour)
		add(&debug.Text{
			Position: mgl64.Vec3{float64(c[0])*16 + 8, IslandY + 2, float64(c[1])*16 + 8},
			Text:     fmt.Sprintf("구매 가능 (%d, %d)\n%s", c[0], c[1], price),
			Colour:   buyableColour,
		})
	}
	s.borders[p.H()] = b
	shown := b.shown
	p.H().DoAfter(chunkBorderTime, func(tx *world.Tx, e world.Entity) {
		if tx.World() != s.world {
			return
		}
		if cur, ok := s.borders[e.H()]; ok && cur.shown == shown {
			s.hideChunkBorders(e.(*player.Player))
		}
	})
}

func (s *session) hideChunkBorders(p *player.Player) {
	b, ok := s.borders[p.H()]
	if !ok {
		return
	}
	delete(s.borders, p.H())
	for _, sh := range b.shapes {
		p.RemoveDebugShape(sh)
	}
}

// drawerLabelKey and drawerLockKey key the count and lock texts of a drawer.
type (
	drawerLabelKey cube.Pos
	drawerLockKey  cube.Pos
)

// textYawOffset turns front-face texts so they read from the front. It is
// added to the entity yaw looking out of the face; flip it by 180 if texts
// show mirrored on real clients.
const textYawOffset = 0

// faceText returns a text fixed flat on a drawer's front face, at height dy
// from the face centre.
func faceText(pos cube.Pos, d Drawer, dy float64, text string) *debug.Text {
	at := pos.Vec3Centre().Add(cube.Pos{}.Side(d.Facing.Face()).Vec3().Mul(0.51)).Add(mgl64.Vec3{0, dy, 0})
	return &debug.Text{
		Position: at, Text: text, Scale: 0.5,
		LockRotation: true, Rotation: mgl64.Vec3{0, yawOf(d.Facing) + textYawOffset, 0},
	}
}

// drawerLabel returns the count text of a drawer, at the bottom of its front.
func drawerLabel(pos cube.Pos, d Drawer) *debug.Text {
	return faceText(pos, d, -0.32, d.label())
}

// drawerLock returns the lock mark of a locked drawer, at the top of its front.
func drawerLock(pos cube.Pos, d Drawer) *debug.Text {
	return faceText(pos, d, 0.34, lockGlyph)
}
