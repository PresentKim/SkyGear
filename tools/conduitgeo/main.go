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

// plateGeometry makes the model of the connector plate entity, drawn like
// EnderIO's connector: on the side of the block that faces a chest or a machine
// a black plate 10×10 pixels and 1 thick at the edge; behind it a 3×3 collar
// for each kind of conduit in the kind's lane, two pixels deep; and, on the
// plate's inner face, the extract arrow (taking from the block) and the insert
// arrow (giving to it) in the corners the lanes leave free. Each part is a bone
// per side (plate_<side>, collar_energy_<side>, collar_item_<side>,
// collar_fluid_<side>, in_<side>, out_<side>) that the render controllers show.
func plateGeometry() map[string]any {
	var bones []any
	// the two cross axes (u, v) of a side by its axis: where u and v point
	cross := map[int][2]int{0: {2, 1}, 1: {0, 2}, 2: {0, 1}}
	arrowUV := uv(0, 0, 24, 24)
	arrowFaces := map[string]any{"north": arrowUV, "south": arrowUV, "east": arrowUV, "west": arrowUV, "up": arrowUV, "down": arrowUV}
	plateUV := uv(0, 0, 16, 16)
	plateFaces := map[string]any{"north": plateUV, "south": plateUV, "east": plateUV, "west": plateUV, "up": plateUV, "down": plateUV}
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
		add := func(name string, b bounds, faces map[string]any) {
			bones = append(bones, map[string]any{
				"name": name + "_" + d.name, "pivot": []float64{0, 0, 0},
				"cubes": []any{cube(b, faces, true)},
			})
		}
		add("plate", box(7, 8, -5, 5, -5, 5), plateFaces)
		for _, k := range kinds {
			lu, lv := k.shift[axes[0]], k.shift[axes[1]]
			add("collar_"+k.name, box(5, 7, lu-1.5, lu+1.5, lv-1.5, lv+1.5), coreFaces())
		}
		add("in", box(6.9, 7, -5, -2, 2, 5), arrowFaces)
		add("out", box(6.9, 7, 2, 5, -5, -2), arrowFaces)
	}
	return map[string]any{
		"description": map[string]any{
			"identifier": "geometry.dragonfactory.conduit_plate", "texture_width": 64, "texture_height": 64,
			"visible_bounds_width": 2, "visible_bounds_height": 2, "visible_bounds_offset": []float64{0, 0.5, 0},
		},
		"bones": bones,
	}
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
