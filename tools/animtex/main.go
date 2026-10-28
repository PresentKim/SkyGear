// Command animtex writes the animated "working" textures of the machines
// (internal/games/skygear/animate.go): a vertical strip of frames made
// from a machine's own texture, with a bright band of an accent colour that
// moves across it (the generators flicker instead), and the atlas and flipbook
// entries that go with them. They are temporary pictures.
//
//	go run ./tools/animtex assets/resourcepack/skygear
package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"sort"

	"github.com/presentkim/skygear/tools/internal/imgio"
)

// spec is one machine: the texture that animates (its front or its side), the
// kind of motion and the accent colour.
type spec struct {
	name   string // machine texture name, e.g. "crusher"
	face   string // "front" or "side"
	accent color.RGBA
	flick  bool // flicker instead of a moving band
}

var specs = []spec{
	{"crusher", "front", color.RGBA{230, 90, 70, 255}, false},
	{"ore_washer", "front", color.RGBA{90, 160, 240, 255}, false},
	{"alloy_smelter", "front", color.RGBA{255, 150, 50, 255}, false},
	{"mob_simulator", "front", color.RGBA{170, 90, 230, 255}, false},
	{"crucible", "front", color.RGBA{255, 120, 30, 255}, true},
	{"auto_crafter", "front", color.RGBA{240, 210, 80, 255}, false},
	{"laboratory", "front", color.RGBA{100, 230, 140, 255}, false},
	{"auto_miner", "side", color.RGBA{240, 200, 90, 255}, false},
	{"harvester", "side", color.RGBA{140, 230, 90, 255}, false},
	{"generator", "side", color.RGBA{255, 140, 40, 255}, true},
	{"lava_generator", "side", color.RGBA{255, 100, 20, 255}, true},
	{"item_vacuum", "side", color.RGBA{120, 200, 240, 255}, false},
	{"tree_feller", "side", color.RGBA{200, 160, 90, 255}, false},
}

const frames = 4

func mix(c, a color.RGBA, t float64) color.RGBA {
	m := func(x, y uint8) uint8 { return uint8(float64(x)*(1-t) + float64(y)*t) }
	return color.RGBA{m(c.R, a.R), m(c.G, a.G), m(c.B, a.B), c.A}
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: animtex pack-dir")
		os.Exit(2)
	}
	pack := os.Args[1]
	dir := filepath.Join(pack, "textures/blocks")
	var flips []map[string]any
	atlas := map[string]string{}
	for _, sp := range specs {
		base := imgio.Load(filepath.Join(dir, sp.name+"_"+sp.face+".png"))
		strip := image.NewRGBA(image.Rect(0, 0, 16, 16*frames))
		for f := 0; f < frames; f++ {
			for y := 0; y < 16; y++ {
				for x := 0; x < 16; x++ {
					r, g, b, a := base.At(x, y).RGBA()
					c := color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
					if sp.flick {
						// light pixels glow, a different amount each frame
						lum := (int(c.R) + int(c.G) + int(c.B)) / 3
						t := [frames]float64{0.15, 0.4, 0.25, 0.5}[(f+(x*3+y*5)%2)%frames]
						if lum > 90 {
							c = mix(c, sp.accent, t)
						}
					} else {
						// a diagonal band moves across the picture
						d := (x + y + 32 - f*8) % 32
						switch {
						case d < 3:
							c = mix(c, sp.accent, 0.55)
						case d < 5:
							c = mix(c, sp.accent, 0.25)
						}
					}
					strip.Set(x, 16*f+y, c)
				}
			}
		}
		out := sp.name + "_" + sp.face + "_on"
		imgio.Save(filepath.Join(dir, out+".png"), strip)
		tile := "skygear_" + out
		atlas[tile] = "textures/blocks/" + out
		flips = append(flips, map[string]any{"flipbook_texture": "textures/blocks/" + out, "atlas_tile": tile, "ticks_per_frame": 4, "blend_frames": false})
	}
	// the electric furnace's own strip (a turning gear) is kept as it is
	flips = append(flips, map[string]any{"flipbook_texture": "textures/blocks/electric_furnace_front_on", "atlas_tile": "skygear_electric_furnace_front_on", "ticks_per_frame": 4, "blend_frames": false})

	// atlas: add the new tiles
	atlasPath := filepath.Join(pack, "textures/terrain_texture.json")
	raw, _ := os.ReadFile(atlasPath)
	var doc struct {
		NumMipLevels     int                          `json:"num_mip_levels"`
		TextureName      string                       `json:"texture_name"`
		Padding          int                          `json:"padding"`
		ResourcePackName string                       `json:"resource_pack_name"`
		TextureData      map[string]map[string]string `json:"texture_data"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		panic(err)
	}
	var keys []string
	for k := range atlas {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		doc.TextureData[k] = map[string]string{"textures": atlas[k]}
	}
	enc, _ := json.MarshalIndent(doc, "", "  ")
	_ = os.WriteFile(atlasPath, append(enc, '\n'), 0o644)
	fl, _ := json.MarshalIndent(flips, "", "  ")
	_ = os.WriteFile(filepath.Join(pack, "textures/flipbook_textures.json"), append(fl, '\n'), 0o644)
}
