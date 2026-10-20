package ui

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// TestIconsInTheme: the icons this package puts on its own buttons (page
// arrows, back) are pictures of the dfui theme.
func TestIconsInTheme(t *testing.T) {
	b, err := os.ReadFile("ui.go")
	if err != nil {
		t.Fatal(err)
	}
	icons := regexp.MustCompile(`"(textures/ui/df/\w+)"`).FindAllStringSubmatch(string(b), -1)
	if len(icons) == 0 {
		t.Fatal("no icons found")
	}
	for _, m := range icons {
		if _, err := os.Stat(filepath.Join("../packs/dfui", m[1]+".png")); err != nil {
			t.Error(err)
		}
	}
}
