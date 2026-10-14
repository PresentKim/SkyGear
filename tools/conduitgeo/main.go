// conduitgeo writes assets/resourcepack/dragonfactory/models/blocks/conduit.geo.json:
// one geometry per set of connected sides of a conduit (64 of them), a 4×4×4
// core with a 4×4 arm towards every connected side. Bit i of the number is
// side i in the order down, up, north, south, west, east (the game's
// conduitFaces, conduit.go). East and west are drawn on the opposite sides:
// the client's geometry x axis is mirrored against the world's.
//
//	go run ./tools/conduitgeo assets/resourcepack/dragonfactory/models/blocks/conduit.geo.json
package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type box struct{ origin, size [3]float64 }

// arms are the boxes of the six sides (model space, pixels: x -8..8 and z
// -8..8 around the block's centre, y 0..16).
var arms = [6]box{
	{[3]float64{-2, 0, -2}, [3]float64{4, 6, 4}},  // down
	{[3]float64{-2, 10, -2}, [3]float64{4, 6, 4}}, // up
	{[3]float64{-2, 6, -8}, [3]float64{4, 4, 6}},  // north
	{[3]float64{-2, 6, 2}, [3]float64{4, 4, 6}},   // south
	{[3]float64{2, 6, -2}, [3]float64{6, 4, 4}},   // west (drawn at +x: mirrored)
	{[3]float64{-8, 6, -2}, [3]float64{6, 4, 4}},  // east (drawn at -x)
}

// The textures are EnderIO's 64×64 conduit sheets (Unlicense): the core face
// is the 24×24 square at (40, 40), the arm side the 16×52 strip at (0, 0).
var (
	coreUV = map[string]any{"uv": []float64{40, 40}, "uv_size": []float64{24, 24}}
	armUV  = map[string]any{"uv": []float64{0, 0}, "uv_size": []float64{16, 52}}
)

func cube(b box, arm bool) map[string]any {
	side := coreUV
	if arm {
		side = armUV
	}
	return map[string]any{
		"origin": b.origin[:], "size": b.size[:],
		"uv": map[string]any{
			"down": coreUV, "up": coreUV,
			"north": side, "south": side,
			"east": side, "west": side,
		},
	}
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: conduitgeo out.geo.json")
		os.Exit(2)
	}
	var geos []any
	for mask := range 64 {
		cubes := []any{cube(box{[3]float64{-2, 6, -2}, [3]float64{4, 4, 4}}, false)}
		for i, a := range arms {
			if mask&(1<<i) != 0 {
				cubes = append(cubes, cube(a, true))
			}
		}
		geos = append(geos, map[string]any{
			"description": map[string]any{
				"identifier":            fmt.Sprintf("geometry.dragonfactory_conduit_%02d", mask),
				"texture_width":         64,
				"texture_height":        64,
				"visible_bounds_width":  2,
				"visible_bounds_height": 2,
				"visible_bounds_offset": []float64{0, 0.5, 0},
			},
			"bones": []any{map[string]any{"name": "root", "pivot": []float64{0, 0, 0}, "cubes": cubes}},
		})
	}
	data, _ := json.MarshalIndent(map[string]any{"format_version": "1.12.0", "minecraft:geometry": geos}, "", "  ")
	if err := os.WriteFile(os.Args[1], append(data, '\n'), 0o644); err != nil {
		panic(err)
	}
}
