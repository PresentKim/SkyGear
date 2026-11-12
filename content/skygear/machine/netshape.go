package machine

import (
	"fmt"
	"slices"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/clientdef"
	"github.com/presentkim/skygear/content/skygear/blockmodel"
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

// netVisible are the bones (netshape.Parts) a shape draws.
func netVisible(kind string, shape int) []string {
	_, head := netHeads[kind]
	raw := shape ^ netItemRaw(kind)
	return netshape.Visible(head, cube.Face(raw/64), raw%64)
}

// NetGeometry is the geometry of a kind (tools/netgeo writes them): the
// bones of every part (NetParts), of which a shape shows some
// (bone_visibility).
func NetGeometry(kind string) string { return "geometry.skygear_" + kind }

// NetKinds are the thin network kinds, for tools/netgeo.
var NetKinds = []string{KindNetCable, KindNetImporter, KindNetExporter, KindNetExternal}

// NetShapes lists the shapes of every thin network kind.
func NetShapes() map[string][]int {
	out := map[string][]int{}
	for _, k := range NetKinds {
		out[k] = netShapes(k)
	}
	return out
}

// NetParts are the bones of a kind's geometry, for tools/netgeo.
func NetParts(kind string) []netshape.Part {
	h, head := netHeads[kind]
	return netshape.Parts(head, h)
}

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

// netExtras are the collision boxes and bone visibility of a kind's shape.
func netExtras(kind string, shape int) clientdef.Extras {
	boxes := netBoxes(kind, shape)
	coll := make([]cube.BBox, len(boxes))
	for i, b := range boxes {
		coll[i] = clientBox(b.BBox())
	}
	visible := map[string]string{}
	for _, p := range NetParts(kind) {
		visible[p.Name] = "false"
	}
	for _, name := range netVisible(kind, shape) {
		visible[name] = "true"
	}
	return clientdef.Extras{CollisionBoxes: coll, BoneVisibility: visible}
}

func netProperties(kind string, shape int) customblock.Properties {
	return customblock.Properties{Geometry: NetGeometry(kind), Textures: netMaterials(kind), SelectionBox: clientBox(netshape.Bounds(netBoxes(kind, shape)))}
}

func netCondition(shape int) string {
	return fmt.Sprintf("q.block_state('%s') == '%s'", NetShapeState, netShapeName(shape))
}

// netAllExtras are the extras of the base definition (shape 0) and of each
// shape's permutation.
func netAllExtras(kind string) (clientdef.Extras, map[string]clientdef.Extras) {
	perms := map[string]clientdef.Extras{}
	for _, s := range netShapes(kind) {
		perms[netCondition(s)] = netExtras(kind, s)
	}
	return netExtras(kind, 0), perms
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
			Condition:  netCondition(s),
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

// NetLinker is a block that is not a network machine but carries the storage
// network: a conduit bundle with a network conduit.
type NetLinker interface{ LinksNet() bool }

// LinksNet reports whether b, on side face of a part of the storage network,
// joins it there: a network node whose head does not face back, or a block
// that carries the network (NetLinker).
func LinksNet(b world.Block, face cube.Face) bool {
	if l, ok := b.(NetLinker); ok {
		return l.LinksNet()
	}
	n, ok := b.(Machine)
	if !ok || !NetNodes[n.Kind] {
		return false
	}
	_, head := netHeads[n.Kind]
	return !head || n.State == nil || n.State.Facing != face.Opposite()
}

// netConnects reports whether the node at pos draws an arm toward its side
// face (LinksNet).
func netConnects(tx *world.Tx, pos cube.Pos, face cube.Face) bool {
	return LinksNet(tx.Block(pos.Side(face)), face)
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
