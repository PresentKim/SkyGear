// Command fabricotex builds the sieve and mesh visuals of the pack from Ex
// Nihilo Fabrico (https://github.com/SirLyle/ExNihiloFabrico, MIT, see
// third_party/licenses/ExNihiloFabrico-MIT.txt):
//
//   - models/blocks/{sieve,auto_sieve,magic_sieve}.geo.json: Fabrico's sieve
//     and mesh models plus a content cube per fill height, as
//     geometry.dragonfactory_<kind>_<fill> with material instances "*",
//     "mesh" and "content".
//   - textures/blocks/sieve_mesh*.png and textures/items/mesh_*.png: Fabrico
//     tints white wool per mesh, so its rods are drawn top-down as a 16×16
//     picture in each mesh's colour.
//
// The sieve frame keeps the vanilla planks textures.
//
//	git clone --depth 1 -b 1.14.4-Fabric https://github.com/SirLyle/ExNihiloFabrico <dir>
//	go run ./tools/fabricotex <dir> assets/resourcepack/dragonfactory
package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"

	"github.com/presentkim/dragonfactory/tools/internal/imgio"
)

const assets = "src/main/resources/assets/exnihilofabrico"

// meshes are the mesh tiers (dragonfactory/dfitem) with Fabrico's colour for
// each; netherite is ours.
var meshes = []struct {
	tier string
	col  color.RGBA
}{
	{"string", rgb(0xFFFFFF)},    // Color.WHITE
	{"flint", rgb(0x555555)},     // Color.DARK_GRAY
	{"iron", rgb(0x777777)},      // Color("777777")
	{"gold", rgb(0xFFAA00)},      // Color.GOLD
	{"diamond", rgb(0x00AAAA)},   // Color.DARK_AQUA
	{"netherite", rgb(0x524248)}, // not in Fabrico
}

func rgb(v uint32) color.RGBA { return color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 255} }

// kinds are the sieves sharing the model.
var kinds = []string{"sieve", "auto_sieve", "magic_sieve"}

// sieveStages is the number of content heights (sieve.go).
const sieveStages = 8

type element struct {
	From  [3]float64     `json:"from"`
	To    [3]float64     `json:"to"`
	Faces map[string]any `json:"faces"`
}

func loadModel(path string) []element {
	raw, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	var m struct {
		Elements []element `json:"elements"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		panic(fmt.Errorf("%s: %w", path, err))
	}
	return m.Elements
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: fabricotex fabrico-clone pack-dir")
		os.Exit(2)
	}
	src, pack := os.Args[1], os.Args[2]
	frame := loadModel(filepath.Join(src, assets, "models/block/sieve.json"))
	rods := loadModel(filepath.Join(src, assets, "models/item/mesh.json"))
	writeTextures(pack, rods)
	for _, k := range kinds {
		writeGeometry(pack, k, frame, rods)
	}
}

// along tells a rod along z (true) from one along x.
func along(e element) bool { return e.To[2]-e.From[2] > e.To[0]-e.From[0] }

// rodPixel is the pixel column (rod along z) or row (along x) of a rod.
func rodPixel(e element) int {
	if along(e) {
		return int(e.From[0])
	}
	return int(e.From[2])
}

// rodSpan is the pixel range a rod covers lengthwise, rounded inward.
func rodSpan(e element) (int, int) {
	i := 0
	if along(e) {
		i = 2
	}
	return int(math.Ceil(e.From[i])), int(math.Floor(e.To[i])) - 1
}

// meshBase is the rods seen from above, white; the rods along z are a little
// darker so the crossings read as woven.
func meshBase(rods []element) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for pass, z := range []bool{true, false} {
		lum := uint8(216 + 39*pass)
		for _, r := range rods {
			if along(r) != z {
				continue
			}
			p := rodPixel(r)
			lo, hi := rodSpan(r)
			for i := lo; i <= hi; i++ {
				x, y := p, i
				if !z {
					x, y = i, p
				}
				img.SetRGBA(x, y, color.RGBA{lum, lum, lum, 255})
			}
		}
	}
	return img
}

func tint(base *image.RGBA, c color.RGBA) *image.RGBA {
	out := image.NewRGBA(base.Bounds())
	for i := 0; i < len(base.Pix); i += 4 {
		if base.Pix[i+3] == 0 {
			continue
		}
		l := int(base.Pix[i])
		out.Pix[i] = uint8(l * int(c.R) / 255)
		out.Pix[i+1] = uint8(l * int(c.G) / 255)
		out.Pix[i+2] = uint8(l * int(c.B) / 255)
		out.Pix[i+3] = 255
	}
	return out
}

func writeTextures(pack string, rods []element) {
	base := meshBase(rods)
	tex := filepath.Join(pack, "textures")
	imgio.Save(filepath.Join(tex, "blocks/sieve_mesh.png"), base)
	imgio.Save(filepath.Join(tex, "blocks/sieve_nomesh.png"), image.NewRGBA(base.Bounds()))
	for _, m := range meshes {
		img := tint(base, m.col)
		imgio.Save(filepath.Join(tex, "blocks/sieve_mesh_"+m.tier+".png"), img)
		imgio.Save(filepath.Join(tex, "items/mesh_"+m.tier+".png"), img)
	}
}

type face struct {
	UV       [2]float64 `json:"uv"`
	Size     [2]float64 `json:"uv_size"`
	Material string     `json:"material_instance,omitempty"`
}

type cubeJSON struct {
	Origin [3]float64      `json:"origin"`
	Size   [3]float64      `json:"size"`
	UV     map[string]face `json:"uv"`
}

func round(v float64) float64 { return math.Round(v*1000) / 1000 }

// cube converts a Java element (0..16, x east, z south) to a Bedrock cube
// (origin -8..8). The model is symmetric in x, so the mirrored x axis of
// Bedrock needs no flip. faces lists the faces drawn; uv gives each its UV.
func cube(from, to [3]float64, faces []string, uv func(f string) face) cubeJSON {
	c := cubeJSON{
		Origin: [3]float64{round(from[0] - 8), round(from[1]), round(from[2] - 8)},
		Size:   [3]float64{round(to[0] - from[0]), round(to[1] - from[1]), round(to[2] - from[2])},
		UV:     map[string]face{},
	}
	for _, f := range faces {
		c.UV[f] = uv(f)
	}
	return c
}

// autoUV is the UV Java derives from the element position when a face has
// none: the face's own rectangle of the 16×16 texture.
func autoUV(from, to [3]float64) func(string) face {
	return func(f string) face {
		var u, v, w, h float64
		switch f {
		case "up", "down":
			u, v, w, h = from[0], from[2], to[0]-from[0], to[2]-from[2]
		case "north", "south":
			u, v, w, h = from[0], 16-to[1], to[0]-from[0], to[1]-from[1]
		default:
			u, v, w, h = from[2], 16-to[1], to[2]-from[2], to[1]-from[1]
		}
		return face{UV: [2]float64{round(u), round(v)}, Size: [2]float64{round(w), round(h)}}
	}
}

func faceNames(e element) []string {
	var out []string
	for _, f := range []string{"down", "east", "north", "south", "up", "west"} {
		if _, ok := e.Faces[f]; ok {
			out = append(out, f)
		}
	}
	return out
}

// rodCube draws a mesh rod with the mesh texture: the top and bottom sample
// the rod's own line of the top-down picture, the one-pixel sides a pixel of
// it away from the crossings. The rods along x are 0.01 pixel thicker so no
// faces at the crossings are coplanar.
func rodCube(r element) cubeJSON {
	from, to := r.From, r.To
	p := float64(rodPixel(r))
	lo, hi := rodSpan(r)
	n := float64(hi - lo + 1)
	line := face{UV: [2]float64{p, float64(lo)}, Size: [2]float64{1, n}, Material: "mesh"}
	side := face{UV: [2]float64{p, float64(lo)}, Size: [2]float64{1, 1}, Material: "mesh"}
	if !along(r) {
		from[1] -= 0.01
		to[1] += 0.01
		line = face{UV: [2]float64{float64(lo), p}, Size: [2]float64{n, 1}, Material: "mesh"}
		side = face{UV: [2]float64{float64(lo), p}, Size: [2]float64{1, 1}, Material: "mesh"}
	}
	return cube(from, to, faceNames(r), func(f string) face {
		if f == "up" || f == "down" {
			return line
		}
		return side
	})
}

// contentCube is the block on the mesh at a fill (1 nearly sifted, 8 fresh):
// Fabrico draws it from the mesh (y 10) up 6 pixels when fresh down to 1, 14
// pixels wide; it is inset 0.1 pixel from the frame rails against z-fighting.
func contentCube(fill int) cubeJSON {
	h := 1 + 5*float64(fill-1)/float64(sieveStages-1)
	from := [3]float64{1.1, 10, 1.1}
	to := [3]float64{14.9, 10 + h, 14.9}
	c := cube(from, to, []string{"down", "east", "north", "south", "up", "west"}, autoUV(from, to))
	for k, f := range c.UV {
		f.Material = "content"
		c.UV[k] = f
	}
	return c
}

func writeGeometry(pack, kind string, frame, rods []element) {
	var geos []any
	for fill := 0; fill <= sieveStages; fill++ {
		var cubes []cubeJSON
		for _, e := range frame {
			cubes = append(cubes, cube(e.From, e.To, faceNames(e), autoUV(e.From, e.To)))
		}
		for _, r := range rods {
			cubes = append(cubes, rodCube(r))
		}
		if fill > 0 {
			cubes = append(cubes, contentCube(fill))
		}
		geos = append(geos, map[string]any{
			"description": map[string]any{
				"identifier":            fmt.Sprintf("geometry.dragonfactory_%s_%d", kind, fill),
				"texture_width":         16,
				"texture_height":        16,
				"visible_bounds_width":  2,
				"visible_bounds_height": 2,
				"visible_bounds_offset": []float64{0, 0.5, 0},
			},
			"bones": []any{map[string]any{"name": "root", "pivot": []float64{0, 0, 0}, "cubes": cubes}},
		})
	}
	doc := map[string]any{"format_version": "1.12.0", "minecraft:geometry": geos}
	enc, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(pack, "models/blocks", kind+".geo.json"), append(enc, '\n'), 0o644); err != nil {
		panic(err)
	}
}
