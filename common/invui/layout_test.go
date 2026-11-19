package invui

import "testing"

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
