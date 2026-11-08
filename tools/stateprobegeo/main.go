// Command stateprobegeo writes the block geometries of the state count probe
// (SKYGEAR_STATE_PROBE, content/skygear/stateprobe.go): a conduit that is a
// 3×3×3 pixel core with a 2×2 pixel arm to each side, the proportions of
// tools/conduitgeo and of EnderIO's, drawn as a block.
//
//   - state_probe_shape.geo.json (mode perm): 64 geometries,
//     geometry.skygear.state_probe_shape_<faces>, one for each set of arms
//     (bit i of faces is the arm of directions[i]: down, up, north, south,
//     west, east, the order of the game's conduit faces).
//   - state_probe_bones.geo.json (mode bones): one geometry,
//     geometry.skygear.state_probe_bones, with the core and all six arms as
//     bones for bone_visibility to switch.
//
// Both use the 64×64 conduit sheets of tools/conduittex (arm strip left, core
// face beside it, turned strip below). The files ship in the pack only in the
// mode that needs them (assets.Pack).
//
//	go run ./tools/stateprobegeo assets/resourcepack/skygear/models/blocks
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	coreHalf = 1.5 // the core is 3 pixels
	armHalf  = 1.0 // an arm is 2 pixels
)

// directions of the arms in the order of the game's bit set.
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

func coreFaces() map[string]any {
	f := uv(16, 0, 16, 16)
	return map[string]any{"down": f, "up": f, "north": f, "south": f, "east": f, "west": f}
}

// armFaces: as in tools/conduitgeo, the strip runs lengthways on every long
// face and the ends show the core face.
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

// box makes a cube from its corners in pixels (x and z around the middle of
// the block, y from its bottom).
func box(min, max [3]float64, faces map[string]any) map[string]any {
	return map[string]any{
		"origin": []float64{min[0], min[1], min[2]},
		"size":   []float64{max[0] - min[0], max[1] - min[1], max[2] - min[2]},
		"uv":     faces,
	}
}

func bone(name string, cube map[string]any) map[string]any {
	return map[string]any{"name": name, "pivot": []float64{0, 0, 0}, "cubes": []any{cube}}
}

func coreBone() map[string]any {
	return bone("core", box([3]float64{-coreHalf, 8 - coreHalf, -coreHalf}, [3]float64{coreHalf, 8 + coreHalf, coreHalf}, coreFaces()))
}

// armBone is the arm of direction d: from the core out to the edge of the block.
func armBone(d int) map[string]any {
	dir := directions[d]
	min := [3]float64{-armHalf, 8 - armHalf, -armHalf}
	max := [3]float64{armHalf, 8 + armHalf, armHalf}
	lo, hi := -8.0, 8.0
	if dir.axis == 1 {
		lo, hi = 0, 16
	}
	// the core spans -coreHalf..coreHalf around x and z, 8-coreHalf..8+coreHalf in y
	mid := 0.0
	if dir.axis == 1 {
		mid = 8
	}
	if dir.low {
		min[dir.axis], max[dir.axis] = lo, mid-coreHalf
	} else {
		min[dir.axis], max[dir.axis] = mid+coreHalf, hi
	}
	return bone(dir.name, box(min, max, armFaces(dir.axis)))
}

func geometry(identifier string, bones []any) map[string]any {
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

// shape is the geometry of a set of arms (bit i of faces is directions[i]).
func shape(faces int) map[string]any {
	bones := []any{coreBone()}
	for d := range directions {
		if faces&(1<<d) != 0 {
			bones = append(bones, armBone(d))
		}
	}
	return geometry(fmt.Sprintf("geometry.skygear.state_probe_shape_%d", faces), bones)
}

// bones is the geometry with every arm, for bone_visibility.
func bones() map[string]any {
	b := []any{coreBone()}
	for d := range directions {
		b = append(b, armBone(d))
	}
	return geometry("geometry.skygear.state_probe_bones", b)
}

// files are the files written, by name: compact JSON, as the shape file has
// 64 geometries.
func files() map[string][]byte {
	var shapes []any
	for f := range 64 {
		shapes = append(shapes, shape(f))
	}
	out := map[string][]byte{}
	for name, geos := range map[string][]any{
		"state_probe_shape.geo.json": shapes,
		"state_probe_bones.geo.json": {bones()},
	} {
		data, err := json.Marshal(map[string]any{"format_version": "1.12.0", "minecraft:geometry": geos})
		if err != nil {
			panic(err)
		}
		out[name] = append(data, '\n')
	}
	return out
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: stateprobegeo models/blocks-dir")
		os.Exit(2)
	}
	for name, data := range files() {
		if err := os.WriteFile(filepath.Join(os.Args[1], name), data, 0o644); err != nil {
			panic(err)
		}
	}
}
