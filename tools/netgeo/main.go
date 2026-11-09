// Command netgeo writes the block geometries of the storage network's thin
// blocks (content/skygear/netshape): one per kind (cable, importer, exporter
// and external storage) with a bone per part (machine.NetParts: the core, an
// arm per side, a head per facing); each shape's permutation shows its parts
// with bone_visibility.
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
	"math"
	"os"

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
		"origin": round(8-b.To[0], b.From[1], b.From[2]-8),
		"size":   round(b.To[0]-b.From[0], b.To[1]-b.From[1], b.To[2]-b.From[2]),
		"uv":     uv,
	}
}

// round drops the float noise of the sums (4.1000000000000005): a ten
// thousandth of a pixel is far below what the client can show.
func round(v ...float64) []float64 {
	for i := range v {
		v[i] = math.Round(v[i]*1e4) / 1e4
	}
	return v
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: netgeo out.geo.json")
		os.Exit(2)
	}
	var geos []any
	for _, k := range machine.NetKinds {
		var bones []any
		for _, p := range machine.NetParts(k) {
			var cubes []any
			for _, b := range p.Boxes {
				cubes = append(cubes, cubeOf(b))
			}
			bones = append(bones, map[string]any{"name": p.Name, "pivot": []float64{0, 0, 0}, "cubes": cubes})
		}
		geos = append(geos, map[string]any{
			"description": map[string]any{
				"identifier": machine.NetGeometry(k), "texture_width": 16, "texture_height": 16,
				"visible_bounds_width": 2, "visible_bounds_height": 2, "visible_bounds_offset": []float64{0, 0.5, 0},
			},
			"bones": bones,
		})
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
