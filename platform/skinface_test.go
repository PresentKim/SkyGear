package platform

import (
	"testing"

	"github.com/df-mc/dragonfly/server/player/skin"
)

// paint fills the w x h rectangle at (x, y) of s with c.
func paint(s skin.Skin, x, y, w, h int, c [4]uint8) {
	for py := y; py < y+h; py++ {
		for px := x; px < x+w; px++ {
			copy(s.Pix[(py*s.Bounds().Dx()+px)*4:], c[:])
		}
	}
}

// TestSkinFace: the face is read where the standard skin has it, scaled for
// a 128px skin, with the hat over it; a model's head bone moves it; a skin
// with no face gives none.
func TestSkinFace(t *testing.T) {
	s := skin.New(64, 64)
	paint(s, 8, 8, 8, 8, [4]uint8{0x11, 0x22, 0x33, 255})
	paint(s, 40, 8, 1, 1, [4]uint8{0xff, 0, 0, 255})
	f := skinFace(s)
	if len(f) != 64 || f[0] != "ff0000" || f[1] != "112233" || f[63] != "112233" {
		t.Fatalf("64px face %v", f)
	}

	big := skin.New(128, 128)
	paint(big, 16, 16, 16, 16, [4]uint8{0xaa, 0xbb, 0xcc, 255})
	if f := skinFace(big); len(f) != 64 || f[10] != "aabbcc" {
		t.Fatalf("128px face %v", f)
	}

	model := skin.New(64, 64)
	model.Model = []byte(`{"format_version":"1.12.0","minecraft:geometry":[{"description":{"texture_width":64,"texture_height":64},
		"bones":[{"name":"head","cubes":[{"size":[8,8,8],"uv":[32,32]}]}]}]}`)
	paint(model, 40, 40, 8, 8, [4]uint8{1, 2, 3, 255})
	if f := skinFace(model); len(f) != 64 || f[5] != "010203" {
		t.Fatalf("model face %v", f)
	}

	if f := skinFace(skin.New(64, 64)); f != nil {
		t.Fatalf("clear skin: %v", f)
	}
	if f := skinFace(skin.Skin{}); f != nil {
		t.Fatalf("no skin: %v", f)
	}
}

// TestTextLine: a translation key with parameters is a client span with the
// parameters after it; other text stays.
func TestTextLine(t *testing.T) {
	if got := textLine("commands.generic.unknown", []string{"x"}, true); got == "commands.generic.unknown" || got[len(got)-1] != 'x' {
		t.Errorf("key line %q", got)
	}
	if got := textLine("§aDone", nil, true); got != "§aDone" {
		t.Errorf("text line %q", got)
	}
}
