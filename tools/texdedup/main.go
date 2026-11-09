// Command texdedup points the terrain atlas entries whose block textures are
// the same picture at one file and deletes the copies. The machine texture
// tools (machinetex, tierblocks) make many faces that come out pixel-equal
// (the down face of most machines, every face of a plain cube); each copy is
// one more file in the pack for the client to load.
//
// Only textures/blocks pictures named by terrain_texture.json alone are
// merged: a file another pack file names (flipbook_textures.json, an
// entity) is kept, and it is the one the others point at. The kept file of a
// group is otherwise the first by name. To redraw one face of a group by
// hand, give its entry a file of its own again.
//
// The texture tools write the copies again when they run (a missing file is
// made, an entry reset); run this after them. TestPackBlockTexturesUnique
// fails while copies are left.
//
//	go run ./tools/texdedup assets/resourcepack/skygear
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	_ "image/png"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: texdedup <pack dir>")
		os.Exit(2)
	}
	pack := os.Args[1]
	atlasPath := filepath.Join(pack, "textures/terrain_texture.json")
	raw, err := os.ReadFile(atlasPath)
	if err != nil {
		panic(err)
	}
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

	elsewhere := otherReferences(pack)
	// groups are the atlas pictures of the pack by their pixels.
	groups := map[string][]string{}
	seen := map[string]bool{}
	for _, e := range doc.TextureData {
		p := e["textures"]
		if seen[p] || !strings.HasPrefix(p, "textures/blocks/") {
			continue
		}
		seen[p] = true
		key, ok := pixels(filepath.Join(pack, p+".png"))
		if ok {
			groups[key] = append(groups[key], p)
		}
	}

	moved := map[string]string{}
	for _, g := range groups {
		if len(g) < 2 {
			continue
		}
		sort.Slice(g, func(i, j int) bool {
			if elsewhere[g[i]] != elsewhere[g[j]] {
				return elsewhere[g[i]]
			}
			return g[i] < g[j]
		})
		for _, p := range g[1:] {
			if !elsewhere[p] {
				moved[p] = g[0]
			}
		}
	}
	for _, e := range doc.TextureData {
		if to, ok := moved[e["textures"]]; ok {
			e["textures"] = to
		}
	}
	for p := range moved {
		if err := os.Remove(filepath.Join(pack, p+".png")); err != nil {
			panic(err)
		}
	}
	enc, _ := json.MarshalIndent(doc, "", "  ")
	if err := os.WriteFile(atlasPath, append(enc, '\n'), 0o644); err != nil {
		panic(err)
	}
	fmt.Println(len(moved), "copies merged")
}

// pixels is the size and RGBA pixels of a PNG, "" if it cannot be read.
func pixels(path string) (string, bool) {
	f, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return "", false
	}
	b := img.Bounds()
	rgba := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(rgba, rgba.Bounds(), img, b.Min, draw.Src)
	return fmt.Sprintf("%dx%d:", b.Dx(), b.Dy()) + string(rgba.Pix), true
}

// otherReferences are the textures/blocks paths that pack files other than
// the terrain atlas name.
func otherReferences(pack string) map[string]bool {
	out := map[string]bool{}
	_ = filepath.WalkDir(pack, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".json") || strings.HasSuffix(p, "terrain_texture.json") {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		for {
			i := bytes.Index(b, []byte("textures/blocks/"))
			if i < 0 {
				break
			}
			j := i
			for j < len(b) && b[j] != '"' && b[j] != '.' {
				j++
			}
			out[string(b[i:j])] = true
			b = b[j:]
		}
		return nil
	})
	return out
}
