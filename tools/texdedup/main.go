// Command texdedup points the terrain atlas entries whose block textures are
// the same picture at one file and deletes the copies. The machine texture
// tool (tierblocks) makes many faces that come out pixel-equal
// (the down face of most machines, every face of a plain cube); each copy is
// one more file in the pack for the client to load.
//
// Only textures/blocks pictures named by terrain_texture.json alone are
// merged: a file named anywhere else in the repository (another pack file
// such as flipbook_textures.json or an entity, a style sheet, Go code) is
// kept, and it is the one the others point at. The kept file of a
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
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	_ "image/png"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
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

// blockPath is a textures/blocks path as files name it.
var blockPath = regexp.MustCompile(`textures/blocks/[A-Za-z0-9_/-]+`)

// otherReferences are the textures/blocks paths named by the text files of
// the repository the pack is in (every pack, the Go code, the Ore UI
// sources) other than the terrain atlas itself.
func otherReferences(pack string) map[string]bool {
	root, err := filepath.Abs(pack)
	if err != nil {
		panic(err)
	}
	for !exists(filepath.Join(root, "go.mod")) {
		up := filepath.Dir(root)
		if up == root {
			panic("texdedup: the pack is not in a repository (no go.mod above it)")
		}
		root = up
	}
	atlas, _ := filepath.Abs(filepath.Join(pack, "textures/terrain_texture.json"))
	out := map[string]bool{}
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "third_party":
				return filepath.SkipDir
			}
			return nil
		}
		switch filepath.Ext(p) {
		case ".json", ".css", ".js", ".html", ".go", ".ts", ".tsx":
		default:
			return nil
		}
		if p == atlas {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		for _, m := range blockPath.FindAll(b, -1) {
			out[string(m)] = true
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	return out
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }
