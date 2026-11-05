// Command netgeo writes the block geometries of the storage network's thin
// blocks (content/skygear/netshape): one per shape of the cable, importer,
// exporter and external storage (machine.NetShapes).
//
// The client's block geometry has its x axis mirrored against the world's
// (hopper.HopperGeometry), so a box east of the centre is drawn at x < 0 and
// its east face is the geometry's west face. North is -z.
//
//	go run ./tools/netgeo assets/resourcepack/skygear/models/blocks/net.geo.json
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/netshape"
)

var faceNames = map[cube.Face]string{
	cube.FaceDown: "down", cube.FaceUp: "up", cube.FaceNorth: "north",
	cube.FaceSouth: "south", cube.FaceWest: "east", cube.FaceEast: "west",
}

func cubeOf(b netshape.Box) map[string]any {
	uv := map[string]any{}
	for f, face := range b.Faces {
		u := map[string]any{"uv": []float64{face.UV[0], face.UV[1]}, "uv_size": []float64{face.UV[2] - face.UV[0], face.UV[3] - face.UV[1]}}
		if face.Tex == netshape.Border {
			u["material_instance"] = "border"
		}
		uv[faceNames[f]] = u
	}
	return map[string]any{
		"origin": []float64{8 - b.To[0], b.From[1], b.From[2] - 8},
		"size":   []float64{b.To[0] - b.From[0], b.To[1] - b.From[1], b.To[2] - b.From[2]},
		"uv":     uv,
	}
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: netgeo out.geo.json")
		os.Exit(2)
	}
	shapes := machine.NetShapes()
	kinds := make([]string, 0, len(shapes))
	for k := range shapes {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	var geos []any
	for _, k := range kinds {
		for _, s := range shapes[k] {
			var cubes []any
			for _, b := range machine.NetBoxes(k, s) {
				cubes = append(cubes, cubeOf(b))
			}
			geos = append(geos, map[string]any{
				"description": map[string]any{
					"identifier": machine.NetGeometry(k, s), "texture_width": 16, "texture_height": 16,
					"visible_bounds_width": 2, "visible_bounds_height": 2, "visible_bounds_offset": []float64{0, 0.5, 0},
				},
				"bones": []any{map[string]any{"name": "root", "pivot": []float64{0, 0, 0}, "cubes": cubes}},
			})
		}
	}
	out, err := json.Marshal(map[string]any{"format_version": "1.12.0", "minecraft:geometry": geos})
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(os.Args[1], out, 0o644); err != nil {
		panic(err)
	}
	fmt.Println(len(geos), "geometries written")
}
