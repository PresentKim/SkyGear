// Package netshape is the shape of the storage network's thin blocks, from
// Refined Storage 2's block models (refinedstorage-common models/block
// cable/core, cable/extension, importer, exporter, external_storage at
// d33a173017): a 4 pixel cable core with an arm toward every connected side,
// and for an importer, exporter or external storage a head on the side it
// faces. The pack's geometry (tools/netgeo) and the blocks' collision and
// selection boxes are made from the same boxes.
package netshape

import (
	"github.com/df-mc/dragonfly/server/block/cube"
)

// Tex names the picture a face samples: the cable or the head border.
type Tex int

const (
	Cable Tex = iota
	Border
)

// Face is the part of the 16×16 picture a box face shows: UV as u0, v0, u1, v1.
type Face struct {
	UV  [4]float64
	Tex Tex
}

// Box is a box in block pixels (0 to 16; x east, y up, z south) and the faces
// it draws, by the world side they look at. A side left out is not drawn.
type Box struct {
	From, To [3]float64
	Faces    map[cube.Face]Face
}

// BBox is the box in block units.
func (b Box) BBox() cube.BBox {
	return cube.Box(b.From[0]/16, b.From[1]/16, b.From[2]/16, b.To[0]/16, b.To[1]/16, b.To[2]/16)
}

// Head is the kind of head an attachment has.
type Head int

const (
	Importer Head = iota
	Exporter
	External
)

func faces(uv map[cube.Face][4]float64, tex Tex) map[cube.Face]Face {
	out := map[cube.Face]Face{}
	for f, u := range uv {
		if tex == Border {
			u = rimUV
		}
		out[f] = Face{UV: u, Tex: tex}
	}
	return out
}

// rimUV is the whole head picture: every head face shows all of it, so the
// picture's tinted edge (tools/modtex) outlines each face.
var rimUV = [4]float64{0, 0, 16, 16}

// coreGrow makes the core a little larger than the arms and heads that meet
// it, so their touching faces do not share a plane (the cable picture is
// open in the middle, and faces in one plane flicker through it).
const coreGrow = 0.05

func core() Box {
	uv := map[cube.Face][4]float64{}
	for _, f := range cube.Faces() {
		uv[f] = [4]float64{6, 6, 10, 10}
	}
	return Box{From: [3]float64{6 - coreGrow, 6 - coreGrow, 6 - coreGrow}, To: [3]float64{10 + coreGrow, 10 + coreGrow, 10 + coreGrow}, Faces: faces(uv, Cable)}
}

// northArm is the cable extension drawn toward north.
func northArm() Box {
	return Box{From: [3]float64{6, 6, 0}, To: [3]float64{10, 10, 6}, Faces: faces(map[cube.Face][4]float64{
		cube.FaceNorth: {6, 6, 10, 10}, cube.FaceSouth: {6, 6, 10, 10},
		cube.FaceEast: {10, 6, 16, 10}, cube.FaceWest: {0, 6, 6, 10},
		cube.FaceUp: {6, 0, 10, 6}, cube.FaceDown: {6, 10, 10, 16},
	}, Cable)}
}

// edgeInset keeps a face on the block's edge inside it, far enough from the
// neighbouring block's face that the two do not flicker (0.01 pixels still
// did). Every face is drawn: the blocks use alpha test, which draws back
// faces, so a missing face is a hole through the model.
const edgeInset = 0.1

func inset(b Box) Box {
	for i := range 3 {
		b.From[i] = max(b.From[i], edgeInset)
		b.To[i] = min(b.To[i], 16-edgeInset)
	}
	return b
}

func ring(from, to [3]float64, w, h, d float64) Box {
	return Box{From: from, To: to, Faces: faces(map[cube.Face][4]float64{
		cube.FaceNorth: {0, 0, w, h}, cube.FaceSouth: {0, 0, w, h},
		cube.FaceEast: {0, 0, d, h}, cube.FaceWest: {0, 0, d, h},
		cube.FaceUp: {0, 0, w, d}, cube.FaceDown: {0, 0, w, d},
	}, Border)}
}

// northHead is the head drawn toward north.
func northHead(h Head) []Box {
	var out []Box
	switch h {
	case Importer:
		out = []Box{
			ring([3]float64{6, 6, 4}, [3]float64{10, 10, 6}, 4, 4, 2),
			ring([3]float64{5, 5, 2}, [3]float64{11, 11, 4}, 6, 6, 2),
			ring([3]float64{3, 3, 0}, [3]float64{13, 13, 2}, 10, 10, 2),
		}
	case Exporter:
		out = []Box{
			ring([3]float64{6, 6, 0}, [3]float64{10, 10, 2}, 4, 4, 2),
			ring([3]float64{5, 5, 2}, [3]float64{11, 11, 4}, 6, 6, 2),
			ring([3]float64{3, 3, 4}, [3]float64{13, 13, 6}, 10, 10, 2),
		}
	case External:
		stem := Box{From: [3]float64{7, 7, 2}, To: [3]float64{9, 9, 6}, Faces: faces(map[cube.Face][4]float64{
			cube.FaceSouth: {0, 0, 4, 4},
			cube.FaceEast:  {0, 0, 4, 2}, cube.FaceWest: {0, 0, 4, 2},
			cube.FaceUp: {0, 0, 2, 4}, cube.FaceDown: {0, 0, 2, 4},
		}, Border)}
		out = []Box{stem, ring([3]float64{3, 3, 0}, [3]float64{13, 13, 2}, 10, 10, 2)}
	}
	return out
}

// turn maps a point of a box drawn toward north to the box drawn toward f.
func turn(f cube.Face, p [3]float64) [3]float64 {
	x, y, z := p[0], p[1], p[2]
	switch f {
	case cube.FaceSouth:
		return [3]float64{16 - x, y, 16 - z}
	case cube.FaceEast:
		return [3]float64{16 - z, y, x}
	case cube.FaceWest:
		return [3]float64{z, y, 16 - x}
	case cube.FaceUp:
		return [3]float64{x, 16 - z, y}
	case cube.FaceDown:
		return [3]float64{x, z, 16 - y}
	}
	return p
}

var unit = map[cube.Face][3]float64{
	cube.FaceDown: {0, -1, 0}, cube.FaceUp: {0, 1, 0},
	cube.FaceNorth: {0, 0, -1}, cube.FaceSouth: {0, 0, 1},
	cube.FaceWest: {-1, 0, 0}, cube.FaceEast: {1, 0, 0},
}

func turnFace(f, side cube.Face) cube.Face {
	o, d := turn(f, [3]float64{0, 0, 0}), turn(f, unit[side])
	v := [3]float64{d[0] - o[0], d[1] - o[1], d[2] - o[2]}
	for s, u := range unit {
		if u == v {
			return s
		}
	}
	panic("netshape: no face")
}

func turnBox(f cube.Face, b Box) Box {
	p, q := turn(f, b.From), turn(f, b.To)
	out := Box{Faces: map[cube.Face]Face{}}
	for i := range 3 {
		out.From[i], out.To[i] = min(p[i], q[i]), max(p[i], q[i])
	}
	for side, face := range b.Faces {
		out.Faces[turnFace(f, side)] = face
	}
	return out
}

// Of is the shape of a cable (head false) or an attachment with head h
// facing f, with arms toward the sides in mask (bit n: cube.Face(n)). An
// attachment draws no arm toward its facing.
func Of(head bool, h Head, f cube.Face, mask int) []Box {
	if head {
		mask &^= 1 << f
	}
	out := []Box{core()}
	for _, side := range cube.Faces() {
		if mask&(1<<side) != 0 {
			out = append(out, inset(turnBox(side, northArm())))
		}
	}
	if head {
		for _, b := range northHead(h) {
			out = append(out, inset(turnBox(f, b)))
		}
	}
	return out
}

// Bounds is the box around every box of a shape, in block units.
func Bounds(boxes []Box) cube.BBox {
	lo, hi := boxes[0].From, boxes[0].To
	for _, b := range boxes[1:] {
		for i := range 3 {
			lo[i], hi[i] = min(lo[i], b.From[i]), max(hi[i], b.To[i])
		}
	}
	return Box{From: lo, To: hi}.BBox()
}
