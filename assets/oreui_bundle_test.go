package assets

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestOreUIBundleFresh checks that oui-app.js was built from the current web/oreui sources: the hash on
// its first line covers every file under src/ and package-lock.json.
func TestOreUIBundleFresh(t *testing.T) {
	web := filepath.Join("..", "web", "oreui")
	var files []string
	err := filepath.WalkDir(filepath.Join(web, "src"), func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			rel, _ := filepath.Rel(web, p)
			files = append(files, filepath.ToSlash(rel))
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	files = append(files, "package-lock.json")
	sort.Strings(files)
	h := sha256.New()
	for _, rel := range files {
		b, err := os.ReadFile(filepath.Join(web, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		h.Write([]byte(rel + "\n"))
		h.Write(b)
		h.Write([]byte("\n"))
	}
	want := "/* oui-app src " + hex.EncodeToString(h.Sum(nil)) + " */"

	js, err := os.ReadFile(filepath.Join("resourcepack", "oreui", "hbui", "oui-app.js"))
	if err != nil {
		t.Fatal(err)
	}
	first, _, _ := strings.Cut(string(js), "\n")
	if strings.TrimSpace(first) != want {
		t.Fatalf("oui-app.js is stale: run npm run build in web/oreui\n got %s\nwant %s", first, want)
	}
}

// TestGameplayKeepsVanillaBundle checks that gameplay.html still loads the vanilla bundle and the oreui files.
func TestGameplayKeepsVanillaBundle(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("resourcepack", "oreui", "hbui", "gameplay.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"/hbui/gameplay-6de8e340eddced55490b.js", "gameplay-2778064b6874c712adbb.css", "oui-ddui.css", "oui-mc.css", "oui-app.css", "oui-app.js", `<div id="root">`} {
		if !strings.Contains(string(b), s) {
			t.Errorf("gameplay.html lacks %s", s)
		}
	}
	if strings.Contains(string(b), "oui-ddui.js") {
		t.Error("gameplay.html still loads oui-ddui.js")
	}
}
