package packs

import (
	"encoding/json"
	"io/fs"
	"regexp"
	"strings"
	"testing"
)

func stripComments(s string) string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(l), "//") {
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n")
}

func readJSON(t *testing.T, dir fs.FS, path string, v any) string {
	t.Helper()
	b, err := fs.ReadFile(dir, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(stripComments(string(b))), v); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return string(b)
}

func sub(t *testing.T, name string) fs.FS {
	t.Helper()
	dir, err := Sub(name)
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestPacks: every pack is embedded with its own UUIDs, and the UI files its
// _ui_defs.json lists exist.
func TestPacks(t *testing.T) {
	seen := map[string]string{}
	for _, name := range Names {
		dir := sub(t, name)
		var m struct {
			Header struct {
				UUID string `json:"uuid"`
			} `json:"header"`
			Modules []struct {
				UUID string `json:"uuid"`
			} `json:"modules"`
		}
		readJSON(t, dir, "manifest.json", &m)
		ids := []string{m.Header.UUID}
		for _, mod := range m.Modules {
			ids = append(ids, mod.UUID)
		}
		for _, id := range ids {
			if id == "" {
				t.Errorf("%s: a UUID is empty", name)
			} else if other, ok := seen[id]; ok {
				t.Errorf("%s: UUID %s is also %s's", name, id, other)
			}
			seen[id] = name
		}
		var defs struct {
			UIDefs []string `json:"ui_defs"`
		}
		if _, err := fs.Stat(dir, "ui/_ui_defs.json"); err != nil {
			continue
		}
		readJSON(t, dir, "ui/_ui_defs.json", &defs)
		for _, f := range defs.UIDefs {
			if _, err := fs.Stat(dir, f); err != nil {
				t.Errorf("%s: _ui_defs.json lists %s: %v", name, f, err)
			}
		}
	}
}

var (
	namespaceRe = regexp.MustCompile(`"namespace"\s*:\s*"(\w+)"`)
	// A reference to an element of a theme namespace: "@ns.name" in a key
	// or "ns.name" as a value.
	refRe     = regexp.MustCompile(`["@]((?:sg|sgui)_\w+)\.\w+`)
	textureRe = regexp.MustCompile(`textures/ui/sg/(\w+)`)
)

// TestThemeStandsAlone: the sgui theme refers only to its own namespaces
// (not to the game namespaces of a server pack) and to pictures it has. Pictures picked by a binding (a name ending in "_") are
// checked by tree pictures here and by the slotdeco test.
func TestThemeStandsAlone(t *testing.T) {
	dir := sub(t, "sgui")
	files, err := fs.Glob(dir, "ui/*.json")
	if err != nil {
		t.Fatal(err)
	}
	own := map[string]bool{}
	texts := map[string]string{}
	for _, f := range files {
		var v any
		text := readJSON(t, dir, f, &v)
		texts[f] = stripComments(text)
		if m := namespaceRe.FindStringSubmatch(text); m != nil {
			own[m[1]] = true
		}
	}
	for _, ns := range []string{"sg_form"} {
		if !own[ns] {
			t.Errorf("no file has namespace %s", ns)
		}
	}
	for f, text := range texts {
		for _, m := range refRe.FindAllStringSubmatch(text, -1) {
			if !own[m[1]] {
				t.Errorf("%s refers to %s, not a namespace of the theme", f, m[0][1:])
			}
		}
		for _, m := range textureRe.FindAllStringSubmatch(text, -1) {
			if strings.HasSuffix(m[1], "_") {
				continue
			}
			if _, err := fs.Stat(dir, "textures/ui/sg/"+m[1]+".png"); err != nil {
				t.Errorf("%s: %v", f, err)
			}
		}
	}
	for c := 'a'; c <= 'i'; c++ {
		if _, err := fs.Stat(dir, "textures/ui/sg/tree_node_"+string(c)+".png"); err != nil {
			t.Error(err)
		}
	}
	for c := 'b'; c <= 'p'; c++ {
		if _, err := fs.Stat(dir, "textures/ui/sg/tree_strip_"+string(c)+".png"); err != nil {
			t.Error(err)
		}
	}
}

// TestThemeTexts: every language of the theme names the same keys, and the
// keys the theme's labels show are among them.
func TestThemeTexts(t *testing.T) {
	dir := sub(t, "sgui")
	var langs []string
	readJSON(t, dir, "texts/languages.json", &langs)
	keys := map[string]map[string]bool{}
	for _, lang := range langs {
		b, err := fs.ReadFile(dir, "texts/"+lang+".lang")
		if err != nil {
			t.Fatal(err)
		}
		keys[lang] = map[string]bool{}
		for _, l := range strings.Split(string(b), "\n") {
			if k, _, ok := strings.Cut(l, "="); ok {
				keys[lang][strings.TrimSpace(k)] = true
			}
		}
	}
	b, err := fs.ReadFile(dir, "ui/sg_form.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range regexp.MustCompile(`"text"\s*:\s*"(sgui\.[\w.]+)"`).FindAllStringSubmatch(string(b), -1) {
		for _, lang := range langs {
			if !keys[lang][m[1]] {
				t.Errorf("%s has no %s", lang, m[1])
			}
		}
	}
	for _, lang := range langs {
		for k := range keys[langs[0]] {
			if !keys[lang][k] {
				t.Errorf("%s has no %s", lang, k)
			}
		}
		if len(keys[lang]) != len(keys[langs[0]]) {
			t.Errorf("%s has %d keys, %s %d", lang, len(keys[lang]), langs[0], len(keys[langs[0]]))
		}
	}
}
