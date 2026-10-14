// conduitgeo writes two files of the pack:
//   - models/entity/conduit_visual.geo.json, the models of the conduit display
//     entity (conduitvisual.go), one geometry for every kind of conduit;
//   - models/blocks/conduit_icon.geo.json, the core of every kind alone, a little
//     bigger (9 pixels, centred) inside an invisible full 16 pixel cube, so the
//     client, which scales an item block to fill its frame, draws it 9/16 as big as
//     a full block: the model of the conduit item blocks that show a conduit in
//     the inventory and the hand;
//   - the geometry "conduit_plate" in the entity file: the connector plate at the
//     end of the arms where conduits meet a chest or a machine, one set of bones
//     for each side (conduitplate.go).
//
// Like EnderIO's conduit model, a conduit is a 3×3×3
// core with a thin 2×2 arm towards each side; every part is a bone of its own
// (core, down, up, north, south, west, east) that the render controller
// (render_controllers/conduit_visual.render_controllers.json) shows or hides
// by the entity's variant, so no combination of conduits needs a model of its
// own.
//
// Each kind of conduit has a lane of its own, the same in every block, so
// conduits of one kind meet end to end and the kinds never overlap: energy is
// shifted 3 pixels towards -x, -y and -z, items sit in the middle and fluids are 3
// pixels towards +x, +y and +z. The lanes differ on every
// axis, so whichever way arms run, the arms of the kinds are side by side seen
// from every direction (a lane that differs on one axis only hides behind
// another from one side).
//
// Entity models are drawn mirrored in z against the world (the model's front
// is north, a yaw of 0 faces south, and x is the same): a model's z is the
// world's, negated (flipZ); a real client showed arms to the wrong sides when x
// was negated as well.
//
// The texture of a kind and tier is one 64×64 image (tools/conduittex): the
// arm strip (16×52, lengthways) at the left, the core face (16×16) beside it,
// the strip turned on its side (32×10) below.
//
//	go run ./tools/conduitgeo assets/resourcepack/dragonfactory/models/entity/conduit_visual.geo.json assets/resourcepack/dragonfactory/models/blocks/conduit_icon.geo.json
package main

import (
	"encoding/json"
	"fmt"
	"os"
)

const (
	coreHalf = 1.5 // the core is 3 pixels
	armHalf  = 1.0 // an arm is 2 pixels
	iconHalf = 4.5 // the core of the item block is 9 pixels
	flipZ    = true
)

type bounds struct{ min, max [3]float64 }

// lanes are the shifts of the kinds in pixels (x, y, z).
var kinds = []struct {
	name  string
	shift [3]float64
}{
	{"energy", [3]float64{-3, -3, -3}},
	{"item", [3]float64{0, 0, 0}},
	{"fluid", [3]float64{3, 3, 3}},
}

// directions of the arms in the order of the game's bit set (down, up,
// north, south, west, east): the axis and whether it is the low end.
var directions = []struct {
	name string
	axis int
	low  bool
}{
	{"down", 1, true}, {"up", 1, false},
	{"north", 2, true}, {"south", 2, false},
	{"west", 0, true}, {"east", 0, false},
}

func uv(x, y, w, h float64) map[string]any {
	return map[string]any{"uv": []float64{x, y}, "uv_size": []float64{w, h}}
}

// cube makes a model cube from a box in world pixels relative to the block
// (x and z -8..8 around its centre, y 0..16). faces are the uvs by face.
func cube(b bounds, faces map[string]any, flip bool) map[string]any {
	min, max := b.min, b.max
	if flip && flipZ {
		min[2], max[2] = -b.max[2], -b.min[2]
	}
	return map[string]any{
		"origin": []float64{min[0], min[1], min[2]},
		"size":   []float64{max[0] - min[0], max[1] - min[1], max[2] - min[2]},
		"uv":     faces,
	}
}

func coreFaces() map[string]any {
	f := uv(16, 0, 16, 16)
	return map[string]any{"down": f, "up": f, "north": f, "south": f, "east": f, "west": f}
}

// armFaces: the strip runs lengthways on every long face, the tall strip where
// a face's v runs along the arm (every side of an arm along y, the top and
// bottom of an arm along z), the turned strip where its u does (an arm along
// x, the sides of an arm along z); the ends show the core face.
func armFaces(axis int) map[string]any {
	tall, flat, end := uv(0, 0, 16, 52), uv(0, 52, 32, 10), uv(16, 0, 16, 16)
	switch axis {
	case 0:
		return map[string]any{"north": flat, "south": flat, "up": flat, "down": flat, "east": end, "west": end}
	case 1:
		return map[string]any{"north": tall, "south": tall, "east": tall, "west": tall, "up": end, "down": end}
	}
	return map[string]any{"east": flat, "west": flat, "up": tall, "down": tall, "north": end, "south": end}
}

// geometry makes the model of a kind at its shift: as the display entity (with
// all six arms as bones of their own) or, if icon, as a block (the core and the
// arms along x only).
func geometry(name string, shift [3]float64, icon bool) map[string]any {
	entity := !icon
	c := [3]float64{shift[0], 8 + shift[1], shift[2]}
	half := coreHalf
	if icon {
		half = iconHalf
	}
	core := bounds{
		[3]float64{c[0] - half, c[1] - half, c[2] - half},
		[3]float64{c[0] + half, c[1] + half, c[2] + half},
	}
	var bones []any
	if icon {
		// an invisible frame: one transparent texel of the sheet on a full cube
		clear := uv(60, 60, 1, 1)
		bones = append(bones, map[string]any{"name": "frame", "pivot": []float64{0, 0, 0}, "cubes": []any{map[string]any{
			"origin": []float64{-8, 0, -8}, "size": []float64{16, 16, 16},
			"uv": map[string]any{"north": clear, "south": clear, "east": clear, "west": clear, "up": clear, "down": clear},
		}}})
	}
	bones = append(bones, map[string]any{
		"name": "core", "pivot": []float64{0, 0, 0},
		"cubes": []any{cube(core, coreFaces(), entity)},
	})
	for _, d := range directions {
		if icon {
			continue
		}
		arm := bounds{
			[3]float64{c[0] - armHalf, c[1] - armHalf, c[2] - armHalf},
			[3]float64{c[0] + armHalf, c[1] + armHalf, c[2] + armHalf},
		}
		lo, hi := -8.0, 8.0
		if d.axis == 1 {
			lo, hi = 0, 16
		}
		if d.low {
			arm.min[d.axis], arm.max[d.axis] = lo, core.min[d.axis]
		} else {
			arm.min[d.axis], arm.max[d.axis] = core.max[d.axis], hi
		}
		bones = append(bones, map[string]any{
			"name": d.name, "pivot": []float64{0, 0, 0},
			"cubes": []any{cube(arm, armFaces(d.axis), entity)},
		})
	}
	identifier := "geometry.dragonfactory.conduit_" + name
	if icon {
		identifier = "geometry.dragonfactory.conduit_icon_" + name
	}
	return map[string]any{
		"description": map[string]any{
			"identifier":            identifier,
			"texture_width":         64,
			"texture_height":        64,
			"visible_bounds_width":  2,
			"visible_bounds_height": 2,
			"visible_bounds_offset": []float64{0, 0.5, 0},
		},
		"bones": bones,
	}
}

// plateGeometry makes the model of the connector entity (conduitplate.go),
// drawn like EnderIO's connector: where the conduits meet a chest or a machine,
// behind the end of the arms a 3×3 collar for each kind of conduit in the
// kind's lane, two pixels deep (the item collar is 4×4 and three deep, to carry
// the arrows), and on the item collar the extract arrow (taking from the block)
// and the insert arrow (giving to it), flat on its sides. There is no plate at
// the block's edge: it would share its plane with the face of the block there
// and flicker. Each part is a bone per side (collar_energy_<side>,
// collar_item_<side>, collar_fluid_<side>, in_<side>, out_<side>) that the
// render controllers show.
//
// An arrow is the 24×24 arrow of the texture at (0, 0), or the same turned on
// its side at (32, 0) where it has to run along the u axis of a face (an arm
// along x or z). The "in" arrows sit on the +x/+z (and for an arm along y, both)
// sides of the collar, the "out" arrows on the opposite ones.
func plateGeometry() map[string]any {
	var bones []any
	// the two cross axes (u, v) of a side by its axis: where u and v point
	cross := map[int][2]int{0: {2, 1}, 1: {0, 2}, 2: {0, 1}}
	for _, d := range directions {
		sign := 1.0
		if d.low {
			sign = -1
		}
		axes := cross[d.axis]
		// box makes a world box whose depth runs from t0 to t1 (pixels from the
		// middle of the block towards the side) and which covers u0..u1, v0..v1.
		box := func(t0, t1, u0, u1, v0, v1 float64) bounds {
			var b bounds
			lo, hi := t0*sign, t1*sign
			if lo > hi {
				lo, hi = hi, lo
			}
			b.min[d.axis], b.max[d.axis] = lo, hi
			b.min[axes[0]], b.max[axes[0]] = u0, u1
			b.min[axes[1]], b.max[axes[1]] = v0, v1
			b.min[1] += 8
			b.max[1] += 8
			return b
		}
		for _, k := range kinds {
			lu, lv := k.shift[axes[0]], k.shift[axes[1]]
			half, t0, t1 := 1.5, 5.0, 7.0
			if k.name == "item" {
				half, t0, t1 = 2.0, 4.5, 7.5
			}
			bones = append(bones, map[string]any{
				"name": "collar_" + k.name + "_" + d.name, "pivot": []float64{0, 0, 0},
				"cubes": []any{cube(box(t0, t1, lu-half, lu+half, lv-half, lv+half), coreFaces(), true)},
			})
		}
		// the arrows: a flat quad on a lateral face of the item collar. The model is
		// z mirrored, so a world +z face is the model's "north" one.
		var in, out []any
		for _, lat := range lateralFaces(d.axis) {
			cubes := arrowQuad(d.axis, sign, lat, box)
			if lat.positive {
				in = append(in, cubes)
			} else {
				out = append(out, cubes)
			}
		}
		for _, a := range []struct {
			name  string
			cubes []any
		}{{"in_", in}, {"out_", out}} {
			bones = append(bones, map[string]any{"name": a.name + d.name, "pivot": []float64{0, 0, 0}, "cubes": a.cubes})
		}
	}
	return map[string]any{
		"description": map[string]any{
			"identifier": "geometry.dragonfactory.conduit_plate", "texture_width": 64, "texture_height": 64,
			"visible_bounds_width": 2, "visible_bounds_height": 2, "visible_bounds_offset": []float64{0, 0.5, 0},
		},
		"bones": bones,
	}
}

// lateral is a side face of a collar that carries an arrow: the axis its normal
// runs along (0 x, 1 y, 2 z) and whether it points the positive way, which
// is the face of the "in" arrow.
type lateral struct {
	axis     int
	positive bool
}

// lateralFaces are the faces of the collar of an arm along axis that carry
// arrows: the two sides perpendicular to the other horizontal axis (arms along x
// and z), or every side face (arms along y), positive ones first.
func lateralFaces(axis int) []lateral {
	switch axis {
	case 0:
		return []lateral{{2, true}, {2, false}}
	case 2:
		return []lateral{{0, true}, {0, false}}
	}
	return []lateral{{0, true}, {2, true}, {0, false}, {2, false}}
}

// faceName is the model face name of the face of a world axis: the model is
// mirrored in z, so the world +z face is the model's north.
func faceName(axis int, positive bool) string {
	switch {
	case axis == 0 && positive:
		return "east"
	case axis == 0:
		return "west"
	case axis == 2 && positive:
		return "north"
	}
	return "south"
}

// arrowQuad makes the cube of an arrow on face lat of the item collar of an arm
// along axis towards sign; box is the collar's world box maker. The in arrows
// point towards the block, the out arrows away from it.
func arrowQuad(axis int, sign float64, lat lateral, box func(t0, t1, u0, u1, v0, v1 float64) bounds) map[string]any {
	// the collar's box and the flat quad 0.05 outside the face of its centre
	c := box(4.5, 7.5, -2, 2, -2, 2)
	q := c
	mid := (c.min[axis] + c.max[axis]) / 2
	q.min[axis], q.max[axis] = mid-1.5, mid+1.5
	// the quad covers 3 along the arm; across it (the other lateral axis) 3 too
	for a := range 3 {
		if a != axis && a != lat.axis {
			m := (c.min[a] + c.max[a]) / 2
			q.min[a], q.max[a] = m-1.5, m+1.5
		}
	}
	if lat.positive {
		q.min[lat.axis], q.max[lat.axis] = c.max[lat.axis], c.max[lat.axis]+0.05
	} else {
		q.min[lat.axis], q.max[lat.axis] = c.min[lat.axis]-0.05, c.min[lat.axis]
	}
	// which way the arrow has to point along the arm: in towards the block
	want := sign
	if !lat.positive {
		want = -sign
	}
	var face map[string]any
	if axis == 1 {
		// the texture's top is up; the in arrow points down, the out arrow up
		point := -1.0
		if !lat.positive {
			point = 1
		}
		face = arrowUV(false, point != want)
	} else {
		// the turned arrow points along u; u runs along -x on a model north face, +x
		// on a south face, +z (world) on an east face and -z on a west face
		uDir := 1.0
		switch faceName(lat.axis, lat.positive) {
		case "north", "west":
			uDir = -1
		}
		point := uDir // the turned in arrow points along +u, the out arrow along -u
		if !lat.positive {
			point = -uDir
		}
		face = arrowUV(true, point != want)
	}
	return cube(q, map[string]any{faceName(lat.axis, lat.positive): face}, true)
}

// arrowUV is the uv of an arrow: at (0, 0), or the turned one at (32, 0), mirrored
// along its length if flip.
func arrowUV(turned, flip bool) map[string]any {
	x, y, w, h := 0.0, 0.0, 24.0, 24.0
	if turned {
		x = 32
		if flip {
			x, w = x+24, -24
		}
	} else if flip {
		y, h = 24, -24
	}
	return map[string]any{"uv": []float64{x, y}, "uv_size": []float64{w, h}}
}

func write(path string, icon bool) {
	var geos []any
	for _, k := range kinds {
		shift := k.shift
		if icon {
			shift = [3]float64{}
		}
		geos = append(geos, geometry(k.name, shift, icon))
	}
	if !icon {
		geos = append(geos, plateGeometry())
	}
	data, _ := json.MarshalIndent(map[string]any{"format_version": "1.12.0", "minecraft:geometry": geos}, "", "  ")
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		panic(err)
	}
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: conduitgeo entity.geo.json icon.geo.json")
		os.Exit(2)
	}
	write(os.Args[1], false)
	write(os.Args[2], true)
}
