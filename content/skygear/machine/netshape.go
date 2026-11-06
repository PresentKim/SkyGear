package machine

import (
	"fmt"
	"slices"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/blockmodel"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/netshape"
)

// NetShapeState is the block state of the storage network's thin blocks
// (netshape): the sides with an arm, and for an attachment its facing.
const NetShapeState = "skygear:net_shape"

// NetNodes are the machine kinds that belong to a storage network.
var NetNodes = map[string]bool{
	KindNetController: true, KindNetCable: true, KindDiskDrive: true,
	KindNetTerminal: true, KindNetImporter: true, KindNetExporter: true,
	KindNetExternal: true, KindNetAutocrafter: true,
}

// netHeads are the attachments: a cable with a head on the side it faces.
var netHeads = map[string]netshape.Head{KindNetImporter: netshape.Importer, KindNetExporter: netshape.Exporter, KindNetExternal: netshape.External}

// NetShaped reports whether kind is drawn as a thin network block.
func NetShaped(kind string) bool {
	_, head := netHeads[kind]
	return head || kind == KindNetCable
}

// Faces the network attachments act on: an attachment faces the block
// clicked to place it (NotePlaceFace); without one it faces south.
var placeFaces = core.NewPlayers[cube.Face]()

// NotePlaceFace keeps the face of the block p clicked with a machine, which
// the attachment placed by that click faces.
func NotePlaceFace(s *core.Session, p *player.Player, face cube.Face) {
	placeFaces.Of(s)[p.H()] = face
}

func takePlaceFace(s *core.Session, p *player.Player) (cube.Face, bool) {
	f, ok := placeFaces.Of(s)[p.H()]
	delete(placeFaces.Of(s), p.H())
	return f, ok
}

// netRaw is a shape as the arm mask, plus 64 times the facing for an
// attachment.
func netRaw(kind string, facing cube.Face, mask int) int {
	if _, head := netHeads[kind]; head {
		return int(facing)*64 + mask&^(1<<facing)
	}
	return mask
}

// netItemRaw is the look of the item: a cable running east and west; an
// attachment's bare head facing south, which the inventory shows to the
// viewer, so the three heads tell apart.
func netItemRaw(kind string) int {
	if _, head := netHeads[kind]; head {
		return netRaw(kind, cube.FaceSouth, 0)
	}
	return netRaw(kind, cube.FaceNorth, 1<<cube.FaceEast|1<<cube.FaceWest)
}

// netShapeValue is the state number of a shape. The client draws a block's
// item with its first state, so the item look is 0 (the raw shape XOR the
// item's).
func netShapeValue(kind string, facing cube.Face, mask int) int {
	return netRaw(kind, facing, mask) ^ netItemRaw(kind)
}

// NetMask is the arms a thin network block draws (bit n: cube.Face(n)).
func NetMask(m Machine) int { return (m.Shape ^ netItemRaw(m.Kind)) % 64 }

func netShapes(kind string) []int {
	var out []int
	for _, f := range cube.Faces() {
		for mask := range 64 {
			if v := netShapeValue(kind, f, mask); !slices.Contains(out, v) {
				out = append(out, v)
			}
		}
	}
	slices.Sort(out)
	return out
}

func netBoxes(kind string, shape int) []netshape.Box {
	h, head := netHeads[kind]
	raw := shape ^ netItemRaw(kind)
	return netshape.Of(head, h, cube.Face(raw/64), raw%64)
}

// NetGeometry is the geometry of a shape (tools/netgeo writes them).
func NetGeometry(kind string, shape int) string {
	return fmt.Sprintf("geometry.skygear_%s_%d", kind, shape)
}

// NetShapes lists the shapes of every thin network kind, for tools/netgeo.
func NetShapes() map[string][]int {
	out := map[string][]int{}
	for _, k := range []string{KindNetCable, KindNetImporter, KindNetExporter, KindNetExternal} {
		out[k] = netShapes(k)
	}
	return out
}

// NetBoxes is the shape's boxes, for tools/netgeo.
func NetBoxes(kind string, shape int) []netshape.Box { return netBoxes(kind, shape) }

func netShapeName(shape int) string { return fmt.Sprintf("%03d", shape) }

// netMaterials are the cable picture and the kind's head colour, with
// ambient occlusion on, as Java Edition shades them: alpha test leaves it
// off, and the dark faces then showed no edges.
func netMaterials(kind string) map[string]customblock.Material {
	out := map[string]customblock.Material{"*": blockmodel.AlphaMaterial("skygear_net_cable").WithAmbientOcclusion()}
	if _, head := netHeads[kind]; head {
		out["border"] = blockmodel.AlphaMaterial("skygear_" + kind + "_border").WithAmbientOcclusion()
	}
	return out
}

// clientBox is a box as the client's block components take it: like the
// block geometry (tools/netgeo), their x axis is mirrored against the world's.
func clientBox(b cube.BBox) cube.BBox {
	lo, hi := b.Min(), b.Max()
	return cube.Box(1-hi.X(), lo.Y(), lo.Z(), 1-lo.X(), hi.Y(), hi.Z())
}

func netProperties(kind string, shape int) customblock.Properties {
	boxes := netBoxes(kind, shape)
	coll := make([]cube.BBox, len(boxes))
	for i, b := range boxes {
		coll[i] = clientBox(b.BBox())
	}
	return customblock.Properties{Geometry: NetGeometry(kind, shape), Textures: netMaterials(kind), SelectionBox: clientBox(netshape.Bounds(boxes)), CollisionBoxes: coll}
}

func netStates(kind string) map[string][]any {
	var values []any
	for _, s := range netShapes(kind) {
		values = append(values, netShapeName(s))
	}
	return map[string][]any{NetShapeState: values}
}

func netPermutations(kind string) []customblock.Permutation {
	var out []customblock.Permutation
	for _, s := range netShapes(kind) {
		out = append(out, customblock.Permutation{
			Condition:  fmt.Sprintf("q.block_state('%s') == '%s'", NetShapeState, netShapeName(s)),
			Properties: netProperties(kind, s),
		})
	}
	return out
}

// netModel is the server's model of a thin network block.
type netModel struct{ boxes []netshape.Box }

func (n netModel) BBox(cube.Pos, world.BlockSource) []cube.BBox {
	out := make([]cube.BBox, len(n.boxes))
	for i, b := range n.boxes {
		out[i] = b.BBox()
	}
	return out
}

func (netModel) FaceSolid(cube.Pos, cube.Face, world.BlockSource) bool { return false }

// netConnects reports whether the node at pos draws an arm toward its side
// face: there is a network node there whose head does not face it.
func netConnects(tx *world.Tx, pos cube.Pos, face cube.Face) bool {
	n, ok := tx.Block(pos.Side(face)).(Machine)
	if !ok || !NetNodes[n.Kind] {
		return false
	}
	_, head := netHeads[n.Kind]
	return !head || n.State == nil || n.State.Facing != face.Opposite()
}

func netShapeAt(tx *world.Tx, pos cube.Pos, m Machine) int {
	facing := cube.FaceSouth
	if m.State != nil {
		facing = m.State.Facing
	}
	mask := 0
	for _, f := range cube.Faces() {
		if netConnects(tx, pos, f) {
			mask |= 1 << f
		}
	}
	return netShapeValue(m.Kind, facing, mask)
}

// showNetShape sets the look of the thin network block at pos to its
// facing and its connections.
func showNetShape(tx *world.Tx, pos cube.Pos, m Machine) {
	if s := netShapeAt(tx, pos, m); s != m.Shape {
		m.Shape = s
		tx.SetBlock(pos, m, nil)
	}
}

// NeighbourUpdateTick redraws a thin network block's arms when a side
// changes.
func (m Machine) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	if NetShaped(m.Kind) && m.State != nil {
		showNetShape(tx, pos, m)
	}
}
