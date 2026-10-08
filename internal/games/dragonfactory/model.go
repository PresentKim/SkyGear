package dragonfactory

import (
	"encoding/json"
)

// Block models (Bedrock geometry) built in code. A model is a list of boxes
// in block pixels: x and z from -8 to 8 around the block centre, y from 0 at
// the bottom. Each box face takes the texture region from (0, 0) the size of
// the face, so 16×16 textures tile without per-face UV work.

type box struct {
	from, size [3]float64
	// mat is the material instance of the faces ("" means "*").
	mat string
}

func faceUV(w, h float64, mat string) map[string]any {
	uv := map[string]any{"uv": []float64{0, 0}, "uv_size": []float64{w, h}}
	if mat != "" {
		uv["material_instance"] = mat
	}
	return uv
}

// geometry returns one geometry entry with the boxes.
func geometry(id string, boxes []box) map[string]any {
	cubes := make([]any, 0, len(boxes))
	for _, b := range boxes {
		x, y, z := b.size[0], b.size[1], b.size[2]
		cubes = append(cubes, map[string]any{
			"origin": b.from,
			"size":   b.size,
			"uv": map[string]any{
				"north": faceUV(x, y, b.mat), "south": faceUV(x, y, b.mat),
				"east": faceUV(z, y, b.mat), "west": faceUV(z, y, b.mat),
				"up": faceUV(x, z, b.mat), "down": faceUV(x, z, b.mat),
			},
		})
	}
	return map[string]any{
		"description": map[string]any{
			"identifier":            id,
			"texture_width":         16,
			"texture_height":        16,
			"visible_bounds_width":  2,
			"visible_bounds_height": 2,
			"visible_bounds_offset": []float64{0, 0.5, 0},
		},
		"bones": []any{map[string]any{"name": "root", "pivot": []float64{0, 0, 0}, "cubes": cubes}},
	}
}

// geoFile returns a geometry file holding the geometries.
func geoFile(geos ...map[string]any) []byte {
	b, err := json.Marshal(map[string]any{"format_version": "1.12.0", "minecraft:geometry": geos})
	if err != nil {
		panic(err)
	}
	return b
}
