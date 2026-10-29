// Command tierblocks writes the per-tier textures of the upgradable machines
// (content/skygear/tierlook.go): for every machine face it
// copies the base picture to "<name>_<face>_t1" .. "_t3" with a trim in the
// colour of the tier along the edges (iron, gold, diamond), the same for the
// working strips ("_on", every frame), and adds the atlas and flipbook
// entries. Files that already exist are left alone, so a texture that was
// redrawn by hand is kept; delete a file to have it made again.
//
//	go run ./tools/tierblocks assets/resourcepack/skygear
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

// machine is one upgradable machine: its texture name, the faces it has and
// its working face ("front_on" or "side_on", "" for none).
type machine struct {
	name  string
	faces []string
	on    string
}

var cube = []string{"side", "up", "down"}
var front = []string{"side", "up", "down", "front", "back"}

var machines = []machine{
	{"crusher", front, "front_on"}, {"electric_furnace", front, "front_on"},
	{"ore_washer", front, "front_on"}, {"crucible", front, "front_on"},
	{"alloy_smelter", front, "front_on"}, {"mob_simulator", front, "front_on"},
	{"laboratory", front, "front_on"}, {"auto_crafter", front, "front_on"},
	{"auto_miner", cube, "side_on"}, {"harvester", cube, "side_on"},
	{"item_vacuum", cube, "side_on"}, {"tree_feller", cube, "side_on"},
	{"generator", cube, "side_on"}, {"lava_generator", cube, "side_on"},
	{"energy_cell", cube, ""},
}

var trims = [3]color.RGBA{
	{205, 210, 220, 255}, // tier 1: iron
	{250, 200, 60, 255},  // tier 2: gold
	{90, 230, 230, 255},  // tier 3: diamond
}

func exists(path string) bool { _, err := os.Stat(path); return err == nil }

// trimmed copies base and paints the tier trim: the outer pixel ring in the
// tier colour, and the corners a shade brighter.
func trimmed(base image.Image, tier int) *image.RGBA {
	b := base.Bounds()
	out := image.NewRGBA(b)
	c := trims[tier-1]
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := base.At(x, y).RGBA()
			px := color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(bl >> 8), uint8(a >> 8)}
			fy := (y - b.Min.Y) % 16
			fx := x - b.Min.X
			if fx == 0 || fx == 15 || fy == 0 || fy == 15 {
				px = c
				if (fx == 0 || fx == 15) && (fy == 0 || fy == 15) {
					px = color.RGBA{uint8(min(int(c.R)+30, 255)), uint8(min(int(c.G)+30, 255)), uint8(min(int(c.B)+30, 255)), 255}
				}
			}
			out.Set(x, y, px)
		}
	}
	return out
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: tierblocks pack-dir")
		os.Exit(2)
	}
	pack := os.Args[1]
	dir := filepath.Join(pack, "textures/blocks")
	atlas := map[string]string{}
	flips := map[string]map[string]any{}
	for _, m := range machines {
		faces := append([]string{}, m.faces...)
		if m.on != "" {
			faces = append(faces, m.on)
		}
		for _, face := range faces {
			base := imgio.Load(filepath.Join(dir, m.name+"_"+face+".png"))
			for tier := 1; tier <= 3; tier++ {
				name := fmt.Sprintf("%s_%s_t%d", m.name, face, tier)
				tile := "skygear_" + name
				atlas[tile] = "textures/blocks/" + name
				if face == m.on {
					flips[tile] = map[string]any{"flipbook_texture": "textures/blocks/" + name, "atlas_tile": tile, "ticks_per_frame": 4, "blend_frames": false}
				}
				path := filepath.Join(dir, name+".png")
				if exists(path) {
					continue
				}
				imgio.Save(path, trimmed(base, tier))
			}
		}
	}

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
	for k, v := range atlas {
		doc.TextureData[k] = map[string]string{"textures": v}
	}
	enc, _ := json.MarshalIndent(doc, "", "  ")
	_ = os.WriteFile(atlasPath, append(enc, '\n'), 0o644)

	flipPath := filepath.Join(pack, "textures/flipbook_textures.json")
	var list []map[string]any
	raw, _ = os.ReadFile(flipPath)
	if err := json.Unmarshal(raw, &list); err != nil {
		panic(err)
	}
	kept := list[:0]
	for _, e := range list {
		if _, again := flips[fmt.Sprint(e["atlas_tile"])]; !again {
			kept = append(kept, e)
		}
	}
	var tiles []string
	for t := range flips {
		tiles = append(tiles, t)
	}
	sort.Strings(tiles)
	for _, t := range tiles {
		kept = append(kept, flips[t])
	}
	fl, _ := json.MarshalIndent(kept, "", "  ")
	_ = os.WriteFile(flipPath, append(fl, '\n'), 0o644)
}
