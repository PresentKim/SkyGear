package invui

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestFaceNameHasFixedLength(t *testing.T) {
	for _, label := range []string{"", "§r짧은 §f글", "§r" + strings.Repeat("긴 글", 100)} {
		got := FaceName(Face("textures/x")+KindRow, KindRow+FamRow+label)
		if len(got) != FaceBytes+FaceKindBytes+FaceTextBytes || !utf8.ValidString(got) || strings.Contains(got, "§f") {
			t.Errorf("FaceName(%q) is %d bytes", label, len(got))
		}
		if !strings.Contains(got[FaceBytes:FaceBytes+FaceKindBytes], KindRow+KindRow+FamRow) {
			t.Errorf("FaceName(%q) kinds %q", label, got[FaceBytes:FaceBytes+FaceKindBytes])
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
