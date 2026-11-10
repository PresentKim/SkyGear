// Command infestedtex writes the block textures of the leaves silkworms eat
// (content/skygear/farm/infest.go): infesting leaves at each stage, the
// silk spreading over them from a few spots, and the infested leaves, all
// silk and cobweb. The leaves are a cluster drawn here (leaf), not a picture
// of another game; Ex Nihilo Fabrico tints the leaves' own texture on the
// client instead and has no picture of them.
//
//	go run ./tools/infestedtex assets/resourcepack/skygear
package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"

	"github.com/presentkim/skygear/tools/internal/imgio"
)

// leaf is the cluster of leaves: 'd' dark, 'm' mid, 'l' light, '.' a gap
// between the leaves (see-through, as leaves are).
var leaf = []string{
	"mld.mmld.dmlm.dm",
	"lmmd.lmmddlmmdlm",
	"dlmm.dllm.dmlld.",
	".dmd.ddmd..dmd.d",
	"md..mlld.mld..ml",
	"lmd.dlmmdlmmd.dm",
	"mlmd.dmd.dllmdml",
	"dmd...d.mld.dmmd",
	".d.mlmd.dmd..dd.",
	"mdlmmlmd..d.mlmd",
	"lmdmlmmmd.mdlmld",
	"dlm.dmld.dlmmdmm",
	".dd..dd.mlld.dmd",
	"mlm.mlmd.dmmd.d.",
	"dllmdlmmd.dlmmdl",
	"mdmd.dmd..mdd.dm",
}

// green are the leaf colours by 'd', 'm', 'l': about vanilla oak leaves
// with the plains foliage tint, so the block is no brighter than its
// neighbours.
var green = map[byte]color.RGBA{
	'd': {34, 78, 22, 255},
	'm': {50, 106, 32, 255},
	'l': {76, 136, 46, 255},
}

// silk are the silk colours by 'd', 'm', 'l': off-white with a cool shade.
var silk = map[byte]color.RGBA{
	'd': {160, 162, 168, 255},
	'm': {196, 196, 202, 255},
	'l': {226, 225, 220, 255},
}

// seeds are where the silk starts, so it spreads from patches.
var seeds = [][2]float64{{4, 4}, {11, 11}, {12, 3}, {3, 12}}

// hash is a fixed pseudo-random value in [0, 1) for x, y.
func hash(x, y, salt int) float64 {
	h := uint32(x*73856093) ^ uint32(y*19349663) ^ uint32(salt*83492791)
	h ^= h >> 13
	h *= 0x5bd1e995
	h ^= h >> 15
	return float64(h%1000) / 1000
}

// reach is how late the silk reaches x, y: 0 at a seed, near 1 at the
// farthest pixel, a little rough.
func reach(x, y int) float64 {
	d := math.MaxFloat64
	for _, s := range seeds {
		// The texture tiles: measure across the edges too.
		dx := math.Abs(float64(x) - s[0])
		dy := math.Abs(float64(y) - s[1])
		dx, dy = min(dx, 16-dx), min(dy, 16-dy)
		d = min(d, math.Hypot(dx, dy))
	}
	return min(d/6.5+hash(x, y, 1)*0.25, 1)
}

// web reports whether a strand of cobweb crosses the gap at x, y: two
// diagonals and a ring around each seed.
func web(x, y int) bool {
	if (x+y)%5 == 0 || (x-y+16)%7 == 0 {
		return true
	}
	for _, s := range seeds {
		r := math.Hypot(float64(x)-s[0], float64(y)-s[1])
		if math.Abs(r-3) < 0.5 {
			return true
		}
	}
	return false
}

// draw draws the leaves with the silk over cover of them (0 none, 1 all);
// webbed gaps get strands of cobweb from cover 0.5 on.
func draw(cover float64) *image.NRGBA {
	im := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	for y, row := range leaf {
		for x := 0; x < 16; x++ {
			c := row[x]
			silked := reach(x, y) < cover
			if c == '.' {
				if cover >= 0.5 && silked && web(x, y) {
					a := uint8(150 + 80*cover)
					im.SetNRGBA(x, y, color.NRGBA{232, 232, 236, a})
				}
				continue
			}
			if silked {
				im.SetNRGBA(x, y, color.NRGBA(silk[c]))
			} else {
				im.SetNRGBA(x, y, color.NRGBA(green[c]))
			}
		}
	}
	return im
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: infestedtex <pack dir>")
		os.Exit(2)
	}
	dir := filepath.Join(os.Args[1], "textures", "blocks")
	// The four stages of infesting leaves, then the infested leaves.
	for stage, cover := range []float64{0.15, 0.35, 0.55, 0.78} {
		imgio.Save(filepath.Join(dir, fmt.Sprintf("infesting_leaves_%d.png", stage)), draw(cover))
	}
	imgio.Save(filepath.Join(dir, "infested_leaves.png"), draw(1.01))
}
