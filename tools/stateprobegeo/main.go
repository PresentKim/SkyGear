// Command stateprobegeo writes the block geometries of the state count probe
// (SKYGEAR_STATE_PROBE=pair2, content/skygear/stateprobe.go): a conduit that is
// a 3×3×3 pixel core with a 2×2 pixel arm to each side, the proportions of
// tools/conduitgeo and of EnderIO's, drawn as a block. Like the display
// entity's geometries, each kind has its lane (energy -3, item 0, fluid +3
// pixels on every axis, network +6, -6, +6) built into the geometry, the arms
// running from the shifted core to the edge of the block (so an arm of a
// shifted lane is longer or shorter by the shift, as in the entity), not as a
// block translation, which would move the arms past the edge of the block.
//
// state_probe_four.geo.json has the two geometries of pair2, one for each of
// its blocks: geometry.skygear.state_probe_two_energy_item and
// geometry.skygear.state_probe_two_fluid_network. Each holds the bones of its
// two kinds (<kind>_core, <kind>_down ... <kind>_east, for bone_visibility to
// switch) and every face names the material instance of its kind, so the
// block maps each kind to the texture of its tier (see laneBones).
//
// The arms and core use the 64×64 conduit sheets of tools/conduittex (arm
// strip left, core face beside it, turned strip below). The file ships in the
// pack only with the probe on (assets.Pack).
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

// kinds are the lanes: the shift of each kind in pixels on the x, y and z
// axes, as in tools/conduitgeo for the first three. The fourth, network (the
// logistics network conduit to come), has the lane (+6, -6, +6). Two lanes' arms along different axes cross unless the lanes
// are 2 pixels or more apart on the third axis (arms are 2 wide), so a lane
// must differ from every other on every axis: the three lanes take -3, 0 and
// +3, leaving -6 or +6 (3 apart, the core of 3 inside the block). (+3, -3, 0)
// would cross, e.g. energy's south arm and network's west arm at (-3, -3, 0).
// TestLanesDoNotMeet checks every pair of bones.
var kinds = []struct {
	name  string
	shift [3]float64
}{{"energy", [3]float64{-3, -3, -3}}, {"item", [3]float64{0, 0, 0}}, {"fluid", [3]float64{3, 3, 3}}, {"network", [3]float64{6, -6, 6}}}

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

// centre is the middle of the core of a lane in geometry pixels.
func centre(shift [3]float64) [3]float64 { return [3]float64{shift[0], 8 + shift[1], shift[2]} }

func coreBone(shift [3]float64) map[string]any {
	c := centre(shift)
	return bone("core", box([3]float64{c[0] - coreHalf, c[1] - coreHalf, c[2] - coreHalf}, [3]float64{c[0] + coreHalf, c[1] + coreHalf, c[2] + coreHalf}, coreFaces()))
}

// armBone is the arm of direction d of the lane at shift: from the core out to
// the edge of the block.
func armBone(d int, shift [3]float64) map[string]any {
	dir := directions[d]
	centre := centre(shift)
	min := [3]float64{centre[0] - armHalf, centre[1] - armHalf, centre[2] - armHalf}
	max := [3]float64{centre[0] + armHalf, centre[1] + armHalf, centre[2] + armHalf}
	lo, hi := -8.0, 8.0
	if dir.axis == 1 {
		lo, hi = 0, 16
	}
	if dir.low {
		min[dir.axis], max[dir.axis] = lo, centre[dir.axis]-coreHalf
	} else {
		min[dir.axis], max[dir.axis] = centre[dir.axis]+coreHalf, hi
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

// laneBones are the core and the six arms of a kind of a kind:
// bones named <kind>_core, <kind>_down ... and every face drawn with the
// material instance named after the kind, so one geometry can show two kinds
// with their own textures (the block's minecraft:material_instances maps the
// kind's name to the texture of its tier).
func laneBones(kind int) []any {
	shift := kinds[kind].shift
	b := []any{coreBone(shift)}
	for d := range directions {
		b = append(b, armBone(d, shift))
	}
	for _, x := range b {
		bn := x.(map[string]any)
		bn["name"] = kinds[kind].name + "_" + bn["name"].(string)
		for _, c := range bn["cubes"].([]any) {
			faces := c.(map[string]any)["uv"].(map[string]any)
			for side, f := range faces {
				g := map[string]any{"material_instance": kinds[kind].name}
				for k, v := range f.(map[string]any) {
					g[k] = v
				}
				faces[side] = g
			}
		}
	}
	return b
}

// pairGeometries are the geometries of pair2, in state_probe_four.geo.json:
// geometry.skygear.state_probe_two_<a>_<b> with the bones of the kinds a and b,
// for the pairs of kinds energy + item and fluid + network.
func pairGeometries() []any {
	var out []any
	for _, p := range [][2]int{{0, 1}, {2, 3}} {
		out = append(out, geometry(fmt.Sprintf("geometry.skygear.state_probe_two_%s_%s", kinds[p[0]].name, kinds[p[1]].name), append(laneBones(p[0]), laneBones(p[1])...)))
	}
	return out
}

// files are the files written, by name: compact JSON.
func files() map[string][]byte {
	out := map[string][]byte{}
	for name, geos := range map[string][]any{
		"state_probe_four.geo.json": pairGeometries(),
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
