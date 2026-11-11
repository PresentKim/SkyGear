package skygear

import (
	"go/build"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/conduit"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/dimension"
	"github.com/presentkim/skygear/content/skygear/fairy"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/storage"
)

// layers are the game's packages from the bottom up (README.md): a package
// imports only the packages before it, so the dependencies run one way.
var layers = []string{
	// data
	"locales", "naming", "textfmt", "formicon", "blockmodel", "netshape", "sgitem", "sievetable", "wintext", "machinerecipe",
	"advancement", "hopper", "migrate", "wire",
	// core
	"core", "network", "achievement", "preview",
	// progress
	"progress", "island", "climate",
	// features
	"tinker", "decor", "machine", "conduit", "storage", "farm", "fairy", "dimension", "handwork", "expand", "copier", "wand",
	"netstore", "elevator",
	// content lists
	"catalog",
	// screens
	"codex", "look", "screens",
}

// TestLayers: every package of the game is in layers, and imports only the
// packages before it.
func TestLayers(t *testing.T) {
	const prefix = "github.com/presentkim/skygear/content/skygear/"
	at := map[string]int{}
	for i, p := range layers {
		at[p] = i
	}
	for i, p := range layers {
		pkg, err := build.Default.ImportDir(p, 0)
		if err != nil {
			t.Errorf("%s: %v", p, err)
			continue
		}
		for _, imp := range pkg.Imports {
			dep, ok := strings.CutPrefix(imp, prefix)
			if !ok {
				continue
			}
			if j, ok := at[dep]; !ok || j >= i {
				t.Errorf("%s imports %s, which is not below it", p, dep)
			}
		}
	}
	dirs, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range dirs {
		if _, listed := at[d.Name()]; d.IsDir() && !listed {
			if _, err := build.Default.ImportDir(d.Name(), 0); err == nil {
				t.Errorf("package %s is not in layers", d.Name())
			}
		}
	}
}

// siblings are the imports among the feature layer: a package of that layer
// imports another one of it only if its row lists it, and every listed import
// exists. A feature without a row imports no sibling.
var siblings = map[string][]string{
	"decor":     {"tinker"},
	"machine":   {"decor", "tinker"},
	"conduit":   {"machine"},
	"storage":   {"machine"},
	"farm":      {"machine"},
	"fairy":     {"farm", "machine"},
	"dimension": {"machine"},
	"copier":    {"conduit", "machine", "storage"},
	"netstore":  {"conduit", "decor", "machine", "storage"},
}

// featureLayer is the "features" group of layers.
var featureLayer = []string{
	"tinker", "decor", "machine", "conduit", "storage", "farm", "fairy", "dimension", "handwork", "expand", "copier", "wand",
	"netstore", "elevator",
}

// TestFeatureSiblings: a package of the feature layer imports a sibling only
// if siblings says so, and siblings lists no import that does not exist.
func TestFeatureSiblings(t *testing.T) {
	const prefix = "github.com/presentkim/skygear/content/skygear/"
	in := map[string]bool{}
	for _, p := range featureLayer {
		in[p] = true
	}
	for p, row := range siblings {
		if !in[p] {
			t.Errorf("siblings row %s is not a package of the feature layer", p)
		}
		for _, dep := range row {
			if !in[dep] {
				t.Errorf("siblings row %s lists %s, which is not a package of the feature layer", p, dep)
			}
		}
	}
	for _, p := range featureLayer {
		pkg, err := build.Default.ImportDir(p, 0)
		if err != nil {
			t.Errorf("%s: %v", p, err)
			continue
		}
		used := map[string]bool{}
		for _, imp := range pkg.Imports {
			if dep, ok := strings.CutPrefix(imp, prefix); ok && in[dep] {
				used[dep] = true
				if !slices.Contains(siblings[p], dep) {
					t.Errorf("%s imports the sibling %s, which siblings does not list", p, dep)
				}
			}
		}
		for _, dep := range siblings[p] {
			if !used[dep] {
				t.Errorf("siblings lists %s -> %s, which %s does not import", p, dep, p)
			}
		}
	}
}

// TestEntityTypesAsBefore: EntityTypes holds the types the game registered
// before the features were listed (the default ones, then each of the game's
// own), the same set by name. The registry is a map by name, so the order of
// the list is not observable.
func TestEntityTypesAsBefore(t *testing.T) {
	want := append(append(entity.DefaultRegistry.Types(), storage.HolderType, fairy.EntityType, conduit.VisualType, conduit.PlateType, dimension.GatePortalType, dimension.FxDemoType, dimension.GemType, dimension.PopupType, machine.RangeBoxType), dimension.MonsterTypes()...)
	got := EntityTypes()
	if len(got) != len(want) {
		t.Fatalf("%d entity types, want %d", len(got), len(want))
	}
	byName := map[string]world.EntityType{}
	for _, e := range want {
		byName[e.EncodeEntity()] = e
	}
	if len(byName) != len(want) {
		t.Fatal("the expected types repeat a name")
	}
	for _, e := range got {
		w, ok := byName[e.EncodeEntity()]
		if !ok || !reflect.DeepEqual(w, e) {
			t.Errorf("entity type %s differs from the expected one", e.EncodeEntity())
		}
		delete(byName, e.EncodeEntity())
	}
	for name := range byName {
		t.Errorf("entity type %s is missing", name)
	}
	entity.DefaultRegistry.Config().New(got) // a repeated name panics
}

// TestFeatureNames: the features are listed in the order of the plan, once
// each.
func TestFeatureNames(t *testing.T) {
	var names []string
	for _, f := range Features {
		names = append(names, f.Name)
	}
	want := []string{"core", "achievement", "progress", "island", "climate", "network", "tinker", "autofill", "machine", "conduit", "storage", "farm", "fairy", "dimension", "handwork", "expand", "copier", "wand", "netstore", "elevator", "look", "screens", "items"}
	if !slices.Equal(names, want) {
		t.Errorf("features %v, want %v", names, want)
	}
	if got := core.Features(); !slices.Equal(got, Features) {
		t.Errorf("core.Features() is not the root list")
	}
}
