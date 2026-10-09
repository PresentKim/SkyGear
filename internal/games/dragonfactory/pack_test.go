package dragonfactory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

// packDir is the resource pack that draws the custom blocks and items.
const packDir = "../../../assets/resourcepack/dragonfactory"

func readPackJSON(t *testing.T, path string, v any) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(packDir, path))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
}

func packFileExists(path string) bool {
	_, err := os.Stat(filepath.Join(packDir, path))
	return err == nil
}

type packAtlas struct {
	TextureData map[string]struct {
		Textures string `json:"textures"`
	} `json:"texture_data"`
}

// TestPackHasItemAssets: every custom item has an atlas entry, its PNG, a
// name line matching Name() and a form icon.
func TestPackHasItemAssets(t *testing.T) {
	var atlas packAtlas
	readPackJSON(t, "textures/item_texture.json", &atlas)
	lang := packLang(t)
	items := world.CustomItems()
	if len(items) < 10 {
		t.Fatalf("only %d custom items registered", len(items))
	}
	for _, it := range items {
		id, _ := it.EncodeItem()
		e, ok := atlas.TextureData[id]
		if !ok {
			t.Errorf("%s: not in item_texture.json", id)
		} else if !packFileExists(e.Textures + ".png") {
			t.Errorf("%s: missing %s.png", id, e.Textures)
		}
		if got := lang["item."+id+".name"]; got != it.Name() {
			t.Errorf("%s: texts/en_US.lang name %q, want %q", id, got, it.Name())
		}
		if iconOf(item.NewStack(it, 1)) == "" {
			t.Errorf("%s: no form icon", id)
		}
	}
}

// TestPackHasBlockAssets: every geometry and pack texture a custom block or
// one of its permutations names is in the pack.
func TestPackHasBlockAssets(t *testing.T) {
	var atlas packAtlas
	readPackJSON(t, "textures/terrain_texture.json", &atlas)
	for key, e := range atlas.TextureData {
		if !packFileExists(e.Textures + ".png") {
			t.Errorf("terrain %s: missing %s.png", key, e.Textures)
		}
	}
	geometries := map[string]bool{}
	files, _ := filepath.Glob(filepath.Join(packDir, "models/blocks/*.geo.json"))
	for _, f := range files {
		var geo struct {
			Geometry []struct {
				Description struct {
					Identifier string `json:"identifier"`
				} `json:"description"`
			} `json:"minecraft:geometry"`
		}
		rel, _ := filepath.Rel(packDir, f)
		readPackJSON(t, rel, &geo)
		for _, g := range geo.Geometry {
			geometries[g.Description.Identifier] = true
		}
	}
	lang := packLang(t)
	check := func(id string, props customblock.Properties) {
		if props.Geometry != "" && !strings.HasPrefix(props.Geometry, "minecraft:") && !geometries[props.Geometry] {
			t.Errorf("%s: geometry %s not in models/blocks", id, props.Geometry)
		}
		for target, m := range props.Textures {
			name := m.Encode()["texture"].(string)
			// Other names are vanilla tiles (sieve contents).
			if _, ok := atlas.TextureData[name]; strings.HasPrefix(name, "dragonfactory_") && !ok {
				t.Errorf("%s: material %s texture %s not in terrain_texture.json", id, target, name)
			}
		}
	}
	blocks := world.DefaultBlockRegistry.CustomBlocks()
	if len(blocks) < 5 {
		t.Fatalf("only %d custom blocks registered", len(blocks))
	}
	for id, b := range blocks {
		check(id, b.Properties())
		if p, ok := b.(block.Permutable); ok {
			for _, perm := range p.Permutations() {
				check(id+" "+perm.Condition, perm.Properties)
			}
		}
		if n, ok := b.(interface{ Name() string }); ok && lang["tile."+id+".name"] != n.Name() {
			t.Errorf("%s: texts/en_US.lang tile name %q, want %q", id, lang["tile."+id+".name"], n.Name())
		}
	}
}

func packLang(t *testing.T) map[string]string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(packDir, "texts/en_US.lang"))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, line := range strings.Split(string(b), "\n") {
		if k, v, ok := strings.Cut(line, "="); ok {
			out[k] = v
		}
	}
	return out
}
