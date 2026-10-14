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

// TestPackHasItemAssets: every custom item that is no block has an atlas
// entry and its PNG (block items show as their block and have none), and
// every item has a name line matching Name() and a form icon.
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
		if _, block := it.(world.Block); block {
			// Block items are drawn as their block; an icon is unused.
			if ok {
				t.Errorf("%s: block item has an unused icon in item_texture.json", id)
			}
		} else if !ok {
			t.Errorf("%s: not in item_texture.json", id)
		} else if !packFileExists(e.Textures + ".png") {
			t.Errorf("%s: missing %s.png", id, e.Textures)
		}
		if got := lang["item."+id+".name"]; got != it.Name() {
			t.Errorf("%s: texts/ko_KR.lang name %q, want %q", id, got, it.Name())
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
			t.Errorf("%s: texts/ko_KR.lang tile name %q, want %q", id, lang["tile."+id+".name"], n.Name())
		}
	}
}

// packLang reads the Korean names of the pack (the default language, what
// Name() says); packLangFile reads any language file of the pack.
func packLang(t *testing.T) map[string]string { return packLangFile(t, "ko_KR") }

func packLangFile(t *testing.T, name string) map[string]string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(packDir, "texts", name+".lang"))
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

// TestPackLanguagesAgree: the English fallback (en_US, what a client in any
// other language shows) names exactly what the Korean file does, and both
// are listed in languages.json.
func TestPackLanguagesAgree(t *testing.T) {
	ko, en := packLangFile(t, "ko_KR"), packLangFile(t, "en_US")
	for k := range ko {
		if en[k] == "" {
			t.Errorf("en_US.lang lacks %s", k)
		}
	}
	for k := range en {
		if _, ok := ko[k]; !ok {
			t.Errorf("ko_KR.lang lacks %s", k)
		}
	}
	var langs []string
	readPackJSON(t, "texts/languages.json", &langs)
	if len(langs) != 2 {
		t.Errorf("languages.json lists %v", langs)
	}
}

// TestCustomBlockStatesMatchTheDefinition: the client numbers the states of a
// custom block from its definition (every combination of the values of its
// states, in the order of the values), and the server numbers them by
// registration. A missing or reordered state shifts the ids of every block
// after it and breaks the textures of other blocks, so each block name must
// have exactly the states its States() lists, in that order.
func TestCustomBlockStatesMatchTheDefinition(t *testing.T) {
	registered := map[string][]map[string]any{}
	for _, b := range world.Blocks() {
		if _, ok := b.(world.CustomBlock); !ok {
			continue
		}
		name, props := b.EncodeBlock()
		registered[name] = append(registered[name], props)
	}
	for name, cb := range world.DefaultBlockRegistry.CustomBlocks() {
		p, ok := cb.(block.Permutable)
		if !ok || len(p.States()) == 0 {
			if n := len(registered[name]); n > 1 {
				t.Errorf("%s has no states but %d registered", name, n)
			}
			continue
		}
		states := p.States()
		if len(states) != 1 {
			t.Errorf("%s has %d state properties: only one keeps the order of the states certain", name, len(states))
			continue
		}
		for prop, values := range states {
			got := registered[name]
			if len(got) != len(values) {
				t.Errorf("%s: %d states registered, the definition has %d", name, len(got), len(values))
				continue
			}
			for i, v := range values {
				if got[i][prop] != v {
					t.Errorf("%s: state %d is %v, the definition says %v", name, i, got[i][prop], v)
					break
				}
			}
		}
	}
}

// TestConduitVisualPack: the pieces of the conduit display entity agree: the
// client entity names geometries and textures that exist, the render
// controllers it names exist and show only bones its geometry has.
func TestConduitVisualPack(t *testing.T) {
	var ent struct {
		Entity struct {
			Description struct {
				Identifier        string            `json:"identifier"`
				Textures          map[string]string `json:"textures"`
				Geometry          map[string]string `json:"geometry"`
				RenderControllers []string          `json:"render_controllers"`
			} `json:"description"`
		} `json:"minecraft:client_entity"`
	}
	readPackJSON(t, "entity/conduit_visual.entity.json", &ent)
	d := ent.Entity.Description
	if d.Identifier != "dragonfactory:conduit_visual" || d.Identifier != mustEncode(VisualType) {
		t.Errorf("entity identifier %s", d.Identifier)
	}
	var geo struct {
		Geometries []struct {
			Description struct {
				Identifier string `json:"identifier"`
			} `json:"description"`
			Bones []struct {
				Name string `json:"name"`
			} `json:"bones"`
		} `json:"minecraft:geometry"`
	}
	readPackJSON(t, "models/entity/conduit_visual.geo.json", &geo)
	bones := map[string]map[string]bool{}
	for _, g := range geo.Geometries {
		bones[g.Description.Identifier] = map[string]bool{}
		for _, b := range g.Bones {
			bones[g.Description.Identifier][b.Name] = true
		}
	}
	for key, id := range d.Geometry {
		if bones[id] == nil {
			t.Errorf("geometry %s (%s) is not in models/entity", key, id)
		}
	}
	for key, path := range d.Textures {
		if _, err := os.Stat(filepath.Join(packDir, path+".png")); err != nil {
			t.Errorf("texture %s: %v", key, err)
		}
	}
	var rc struct {
		Controllers map[string]struct {
			Geometry string              `json:"geometry"`
			Parts    []map[string]string `json:"part_visibility"`
			Textures []string            `json:"textures"`
		} `json:"render_controllers"`
	}
	readPackJSON(t, "render_controllers/conduit_visual.render_controllers.json", &rc)
	for _, name := range d.RenderControllers {
		c, ok := rc.Controllers[name]
		if !ok {
			t.Errorf("render controller %s is missing", name)
			continue
		}
		ref := strings.TrimPrefix(c.Geometry, "Geometry.")
		bs := bones[d.Geometry[ref]]
		if bs == nil {
			t.Errorf("%s: geometry %s is not defined", name, c.Geometry)
		}
		for _, part := range c.Parts {
			for bone := range part {
				if bone != "*" && !bs[bone] {
					t.Errorf("%s shows bone %s, which the geometry lacks", name, bone)
				}
			}
		}
		for _, bone := range []string{"core", "down", "up", "north", "south", "west", "east"} {
			if !bs[bone] {
				t.Errorf("%s: geometry has no bone %s", name, bone)
			}
		}
	}
}

func mustEncode(t world.EntityType) string { return t.EncodeEntity() }

// TestConduitLanesDifferOnEveryAxis: the cores of the kinds sit apart on all
// three axes, so arms running along any axis are side by side seen from both
// directions across it (a kind that differs on one axis only hides behind the
// other from one side).
func TestConduitLanesDifferOnEveryAxis(t *testing.T) {
	var geo struct {
		Geometries []struct {
			Description struct {
				Identifier string `json:"identifier"`
			} `json:"description"`
			Bones []struct {
				Name  string `json:"name"`
				Cubes []struct {
					Origin []float64 `json:"origin"`
				} `json:"cubes"`
			} `json:"bones"`
		} `json:"minecraft:geometry"`
	}
	readPackJSON(t, "models/entity/conduit_visual.geo.json", &geo)
	cores := map[string][]float64{}
	for _, g := range geo.Geometries {
		for _, b := range g.Bones {
			if b.Name == "core" {
				cores[g.Description.Identifier] = b.Cubes[0].Origin
			}
		}
	}
	names := []string{"energy", "item", "fluid"}
	for x := range names {
		for y := x + 1; y < len(names); y++ {
			a, b := cores["geometry.dragonfactory.conduit_"+names[x]], cores["geometry.dragonfactory.conduit_"+names[y]]
			if a == nil || b == nil {
				t.Fatalf("cores %v", cores)
			}
			for axis := range 3 {
				if d := a[axis] - b[axis]; d > -3 && d < 3 {
					t.Errorf("%s and %s: axis %d: the cores are only %v pixels apart", names[x], names[y], axis, d)
				}
			}
		}
	}
}

// TestConduitPlatePack: the connector plate entity names an existing geometry
// and textures, its render controllers show only bones the geometry has, and
// every bone of the geometry is shown by one of them.
func TestConduitPlatePack(t *testing.T) {
	var ent struct {
		Entity struct {
			Description struct {
				Identifier        string            `json:"identifier"`
				Textures          map[string]string `json:"textures"`
				Geometry          map[string]string `json:"geometry"`
				RenderControllers []string          `json:"render_controllers"`
			} `json:"description"`
		} `json:"minecraft:client_entity"`
	}
	readPackJSON(t, "entity/conduit_plate.entity.json", &ent)
	d := ent.Entity.Description
	if d.Identifier != PlateType.EncodeEntity() {
		t.Errorf("entity identifier %s", d.Identifier)
	}
	for key, path := range d.Textures {
		if _, err := os.Stat(filepath.Join(packDir, path+".png")); err != nil {
			t.Errorf("texture %s: %v", key, err)
		}
	}
	var geo struct {
		Geometries []struct {
			Description struct {
				Identifier string `json:"identifier"`
			} `json:"description"`
			Bones []struct {
				Name string `json:"name"`
			} `json:"bones"`
		} `json:"minecraft:geometry"`
	}
	readPackJSON(t, "models/entity/conduit_visual.geo.json", &geo)
	bones := map[string]bool{}
	for _, g := range geo.Geometries {
		if g.Description.Identifier == d.Geometry["default"] {
			for _, b := range g.Bones {
				bones[b.Name] = true
			}
		}
	}
	if len(bones) != 30 {
		t.Errorf("%d bones, want 6 for each of 6 sides", len(bones))
	}
	var rc struct {
		Controllers map[string]struct {
			Parts    []map[string]string `json:"part_visibility"`
			Textures []string            `json:"textures"`
		} `json:"render_controllers"`
	}
	readPackJSON(t, "render_controllers/conduit_visual.render_controllers.json", &rc)
	shown := map[string]bool{}
	for _, name := range d.RenderControllers {
		c, ok := rc.Controllers[name]
		if !ok {
			t.Fatalf("render controller %s is missing", name)
		}
		for _, part := range c.Parts {
			for bone := range part {
				if bone != "*" && !bones[bone] {
					t.Errorf("%s shows bone %s, which the geometry lacks", name, bone)
				}
				shown[bone] = true
			}
		}
		for _, ref := range c.Textures {
			if !strings.Contains(ref, "Array.") && !strings.HasPrefix(ref, "Texture.") {
				t.Errorf("%s: texture %s", name, ref)
			}
		}
	}
	for bone := range bones {
		if !shown[bone] {
			t.Errorf("no render controller shows bone %s", bone)
		}
	}
}
