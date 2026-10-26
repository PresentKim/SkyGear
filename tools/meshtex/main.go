// meshtex writes the mesh textures of the sieves, one per mesh tier
// (internal/games/dragonfactory/sieve.go), from textures/blocks/sieve_mesh.png:
// the net keeps its shading and takes the colour of the tier. It also adds the
// atlas tiles. They are temporary pictures.
//
//	go run ./tools/meshtex assets/resourcepack/dragonfactory
package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"

	"github.com/presentkim/dragonfactory/tools/internal/imgio"
)

var tiers = []struct {
	name string
	col  color.RGBA
}{
	{"string", color.RGBA{225, 220, 201, 255}},
	{"flint", color.RGBA{96, 96, 104, 255}},
	{"iron", color.RGBA{206, 208, 212, 255}},
	{"gold", color.RGBA{244, 204, 64, 255}},
	{"diamond", color.RGBA{90, 232, 222, 255}},
	{"netherite", color.RGBA{82, 66, 72, 255}},
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: meshtex pack-dir")
		os.Exit(2)
	}
	pack := os.Args[1]
	dir := filepath.Join(pack, "textures/blocks")
	base := imgio.Load(filepath.Join(dir, "sieve_mesh.png"))
	// the average brightness of the net, to keep its shading
	var sum, n float64
	b := base.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := base.At(x, y).RGBA()
			if a > 0 {
				sum += float64(r+g+bl) / 3 / 257
				n++
			}
		}
	}
	avg := sum / n
	atlasPath := filepath.Join(pack, "textures/terrain_texture.json")
	raw, _ := os.ReadFile(atlasPath)
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		panic(err)
	}
	data := doc["texture_data"].(map[string]any)
	for _, t := range tiers {
		out := image.NewRGBA(b)
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				r, g, bl, a := base.At(x, y).RGBA()
				if a == 0 {
					continue
				}
				k := float64(r+g+bl) / 3 / 257 / avg
				cl := func(v float64) uint8 { return uint8(min(255, max(0, v))) }
				out.Set(x, y, color.RGBA{cl(float64(t.col.R) * k), cl(float64(t.col.G) * k), cl(float64(t.col.B) * k), 255})
			}
		}
		name := "sieve_mesh_" + t.name
		imgio.Save(filepath.Join(dir, name+".png"), out)
		data["dragonfactory_"+name] = map[string]any{"textures": "textures/blocks/" + name}
	}
	enc, _ := json.MarshalIndent(doc, "", "  ")
	_ = os.WriteFile(atlasPath, append(enc, '\n'), 0o644)
}
