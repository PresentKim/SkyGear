// Command treetex writes the textures of the advancement tree screen
// (packs/dfui/textures/ui/df), drawn like a Java
// advancements screen: the window and its inset (nine-slice, with .json
// sidecars), the tabs, the node frames tree_node_<letter> (letter = 'a'+
// 3*frame+state: frame task, goal, challenge; state unobtained-locked, open,
// obtained), the hover highlight tree_hl and the connector strips
// tree_strip_<letter> ('a' plus the bits up 1, down 2, left 4, right 8).
// Run it from the repository root: go run ./tools/treetex (only the pictures
// that are missing; -f writes all). The pictures are placeholders to be
// replaced by hand-drawn ones.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

const dir = "packs/dfui/textures/ui/df"

var (
	black = color.NRGBA{16, 16, 16, 255}
	white = color.NRGBA{255, 255, 255, 255}
	clear = color.NRGBA{}
	light = color.NRGBA{198, 198, 198, 255}
	shade = color.NRGBA{110, 110, 110, 255}
)

func fill(img *image.NRGBA, x0, y0, x1, y1 int, c color.NRGBA) {
	for x := x0; x < x1; x++ {
		for y := y0; y < y1; y++ {
			img.SetNRGBA(x, y, c)
		}
	}
}

// bevelBox draws a w×h box with a black edge, a light top left and a dark
// bottom right inside it.
func bevelBox(w, h int, body, hi, lo color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	fill(img, 0, 0, w, h, black)
	fill(img, 1, 1, w-1, h-1, body)
	fill(img, 1, 1, w-1, 2, hi)
	fill(img, 1, 1, 2, h-1, hi)
	fill(img, 1, h-2, w-1, h-1, lo)
	fill(img, w-2, 1, w-1, h-1, lo)
	return img
}

func window() image.Image { return bevelBox(24, 24, light, white, shade) }

func inset() image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	fill(img, 0, 0, 8, 8, white)
	fill(img, 0, 0, 7, 1, color.NRGBA{55, 55, 55, 255})
	fill(img, 0, 0, 1, 7, color.NRGBA{55, 55, 55, 255})
	fill(img, 1, 1, 7, 7, clear)
	return img
}

// tabOff is a tab that is not chosen: 28×32, its bottom 4 rows tucked behind
// the window (transparent).
func tabOff() image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, 28, 32))
	box := bevelBox(28, 28, color.NRGBA{140, 140, 140, 255}, color.NRGBA{190, 190, 190, 255}, color.NRGBA{90, 90, 90, 255})
	for x := range 28 {
		for y := range 28 {
			img.SetNRGBA(x, y+4, box.NRGBAAt(x, y))
		}
	}
	return img
}

// tabOn is the chosen tab: 28×36, open at the bottom so it joins the window.
func tabOn() image.Image {
	img := bevelBox(28, 36, light, white, light)
	fill(img, 1, 30, 27, 36, light)
	fill(img, 0, 33, 1, 36, light)
	fill(img, 27, 33, 28, 36, light)
	fill(img, 1, 1, 27, 2, white)
	return img
}

func hl(w, h int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	fill(img, 0, 0, w, h, color.NRGBA{255, 255, 255, 70})
	return img
}

var (
	// frames: [state] body, hi, lo (obtained is gold).
	bodies = [3][3]color.NRGBA{
		{{60, 60, 60, 255}, {100, 100, 100, 255}, {30, 30, 30, 255}},
		{{150, 150, 150, 255}, {200, 200, 200, 255}, {95, 95, 95, 255}},
		{{222, 172, 42, 255}, {255, 228, 120, 255}, {150, 105, 10, 255}},
	}
)

// node draws a 26×26 frame: a square (task), with rounded corners (goal) or
// with cut corners and studs on the edges (challenge).
func node(frame, state int) image.Image {
	img := bevelBox(26, 26, bodies[state][0], bodies[state][1], bodies[state][2])
	cut := [3]int{0, 2, 3}[frame]
	for i := range cut {
		for j := range cut - i {
			for _, p := range [][2]int{{i, j}, {25 - i, j}, {i, 25 - j}, {25 - i, 25 - j}} {
				img.SetNRGBA(p[0], p[1], clear)
			}
		}
	}
	if frame == 2 {
		for _, p := range [][2]int{{12, 0}, {13, 0}, {12, 25}, {13, 25}, {0, 12}, {0, 13}, {25, 12}, {25, 13}} {
			img.SetNRGBA(p[0], p[1], bodies[state][1])
		}
	}
	return img
}

// strip is the 6×27 connector piece at the left of a cell: a vertical bus
// (column 2) up and/or down from the middle, a line to the left edge and a
// line to the node on the right, white with a black outline (every outline
// is drawn before the lines so the joints are clean).
func strip(mask int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, 6, 27))
	type seg struct{ x0, y0, x1, y1 int }
	var core []seg
	if mask&1 != 0 {
		core = append(core, seg{2, 0, 3, 14})
	}
	if mask&2 != 0 {
		core = append(core, seg{2, 13, 3, 27})
	}
	if mask&4 != 0 {
		core = append(core, seg{0, 13, 3, 14})
	}
	if mask&8 != 0 {
		core = append(core, seg{2, 13, 6, 14})
	}
	for _, c := range core {
		fill(img, max(0, c.x0-1), max(0, c.y0-1), min(6, c.x1+1), min(27, c.y1+1), black)
	}
	for _, c := range core {
		fill(img, c.x0, c.y0, c.x1, c.y1, white)
	}
	return img
}

func main() {
	force := flag.Bool("f", false, "overwrite existing pictures")
	flag.Parse()
	write := func(name string, img image.Image) {
		path := filepath.Join(dir, name+".png")
		if _, err := os.Stat(path); err == nil && !*force {
			return
		}
		f, err := os.Create(path)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		if err := png.Encode(f, img); err != nil {
			panic(err)
		}
		fmt.Println("wrote", path)
	}
	sidecar := func(name, content string) {
		path := filepath.Join(dir, name+".json")
		if _, err := os.Stat(path); err == nil && !*force {
			return
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			panic(err)
		}
	}
	write("tree_window", window())
	sidecar("tree_window", "{\n  \"base_size\": [ 24, 24 ],\n  \"nineslice_size\": 3\n}\n")
	write("tree_inset", inset())
	sidecar("tree_inset", "{\n  \"base_size\": [ 8, 8 ],\n  \"nineslice_size\": 1\n}\n")
	write("tree_tab_off", tabOff())
	write("tree_tab_on", tabOn())
	write("tree_hl", hl(26, 26))
	for frame := range 3 {
		for state := range 3 {
			write(fmt.Sprintf("tree_node_%c", 'a'+3*frame+state), node(frame, state))
		}
	}
	for m := 1; m < 16; m++ {
		write(fmt.Sprintf("tree_strip_%c", 'a'+m), strip(m))
	}
}
