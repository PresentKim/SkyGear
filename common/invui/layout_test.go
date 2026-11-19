package invui

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestFaceNameHasFixedLength(t *testing.T) {
	for _, label := range []string{"", "§r짧은 글", "§r" + strings.Repeat("긴 글", 100)} {
		got := FaceName(Face("textures/x")+KindRow, label)
		if len(got) != FaceBytes+FaceTextBytes || !utf8.ValidString(got) {
			t.Errorf("FaceName(%q) is %d bytes", label, len(got))
		}
	}
}

func TestCountMark(t *testing.T) {
	for count, want := range map[string]string{
		"":       "§ § § § § § § ",
		"1":      "§ § § § § § § ",
		"438":    "§ § § § §4§3§8",
		"12k":    "§ § § § §1§2§k",
		"123456": "§ § §2§3§4§5§6",
	} {
		if got := CountMark(count); got != want {
			t.Errorf("CountMark(%q) = %q, want %q", count, got, want)
		}
		if n := len(CountMark(count)); n != 3*CountChars {
			t.Errorf("CountMark(%q) is %d bytes", count, n)
		}
	}
}
