package skygear

import (
	"go/build"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/conduit"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/dimension"
	"github.com/presentkim/skygear/content/skygear/expand"
	"github.com/presentkim/skygear/content/skygear/fairy"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/look"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/preview"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/storage"
)

// layers are the game's packages from the bottom up (README.md): a package
// imports only the packages before it, so the dependencies run one way.
var layers = []string{
	// data
	"locales", "naming", "textfmt", "formicon", "blockmodel", "netshape", "sgitem", "sievetable", "wintext", "machinerecipe",
	"advancement", "hopper", "migrate", "wire",
	// core
	"core", "network", "achievement", "preview", "look", "autofill", "decor",
	// progress
	"progress", "island", "climate",
	// features
	"tinker", "machine", "conduit", "storage", "farm", "fairy", "dimension", "handwork", "expand", "copier", "wand",
	"netstore", "elevator",
	// content lists
	"catalog",
	// screens
	"codex", "screens",
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
	"machine":   {"tinker"},
	"conduit":   {"machine"},
	"storage":   {"machine"},
	"farm":      {"machine"},
	"fairy":     {"farm", "machine"},
	"dimension": {"machine"},
	"copier":    {"conduit", "machine", "storage"},
	"netstore":  {"conduit", "machine", "storage"},
}

// featureLayer is the "features" group of layers.
var featureLayer = []string{
	"tinker", "machine", "conduit", "storage", "farm", "fairy", "dimension", "handwork", "expand", "copier", "wand",
	"netstore", "elevator",
}

// TestFeatureSiblings: a package of the feature layer imports a sibling only
// if siblings says so, and siblings lists no import that does not exist.
func TestFeatureSiblings(t *testing.T) {
	const prefix = "github.com/presentkim/skygear/content/skygear/"
	// featureLayer is the run of layers from the first feature up to the
	// content lists (catalog): a feature package added to one list is in the
	// other.
	if i := slices.Index(layers, featureLayer[0]); i < 0 || i+len(featureLayer) >= len(layers) ||
		!slices.Equal(layers[i:i+len(featureLayer)], featureLayer) || layers[i+len(featureLayer)] != "catalog" {
		t.Errorf("featureLayer %v is not the features group of layers (from %s up to catalog); list a feature package in both", featureLayer, featureLayer[0])
	}
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

// machineFeatureImports are the packages of the feature layer machine may
// import. The features that own a machine kind import machine and register
// the kind's behaviour (machine.Register); machine does not import them back.
// tinker is the one exception, and only in machine/tinkertable.go: the tinker
// feature cannot import machine, because the tinker's tools must register
// before the machines (the client goldens pin the item ids), so machine sets
// the tinker table's behaviour itself.
var machineFeatureImports = map[string]string{"tinker": "tinkertable.go"}

// TestMachineImportsNoFeature: machine imports no package of the feature
// layer but those of machineFeatureImports, each only in its one file, and
// its siblings row lists exactly those.
func TestMachineImportsNoFeature(t *testing.T) {
	const prefix = "github.com/presentkim/skygear/content/skygear/"
	if row := siblings["machine"]; len(row) != len(machineFeatureImports) {
		t.Errorf("the siblings row of machine is %v, want the %d of machineFeatureImports", row, len(machineFeatureImports))
	}
	for _, dep := range siblings["machine"] {
		if _, ok := machineFeatureImports[dep]; !ok {
			t.Errorf("siblings lists machine -> %s: the feature registers its kind's behaviour with machine.Register instead", dep)
		}
	}
	files, err := os.ReadDir("machine")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".go") || strings.HasSuffix(f.Name(), "_test.go") {
			continue
		}
		src, err := os.ReadFile("machine/" + f.Name())
		if err != nil {
			t.Fatal(err)
		}
		for _, dep := range featureLayer {
			if !strings.Contains(string(src), `"`+prefix+dep+`"`) {
				continue
			}
			if machineFeatureImports[dep] != f.Name() {
				t.Errorf("machine/%s imports the feature %s: the feature registers its kind's behaviour with machine.Register instead", f.Name(), dep)
			}
		}
	}
}

// TestLookExpandingWithTool: a player in island expansion mode who holds a
// wand, or sneaks with the iron hammer, gets no preview from the panel update
// its movement makes (the hidden panel returns first), but gets it from the
// update of the fast tick (the previews follow it whatever the panel did).
func TestLookExpandingWithTool(t *testing.T) {
	cases := []struct {
		name, owner string
		setup       func(p *player.Player, tx *world.Tx, at cube.Pos)
	}{
		{"wand", "wand", func(p *player.Player, tx *world.Tx, at cube.Pos) {
			slab := block.Slab{Block: block.Planks{Wood: block.OakWood()}}
			for x := -1; x <= 1; x++ {
				for z := -1; z <= 1; z++ {
					tx.SetBlock(at.Add(cube.Pos{x, 0, z}), slab, nil)
				}
			}
			p.SetHeldItems(item.NewStack(sgitem.NewWand(sgitem.WandStone), 1), item.Stack{})
			_, _ = p.Inventory().AddItem(item.NewStack(slab, 20))
		}},
		{"ironHammer", "vein", func(p *player.Player, tx *world.Tx, at cube.Pos) {
			tx.SetBlock(at, block.Stone{}, nil)
			p.SetHeldItems(item.NewStack(sgitem.Hammer{}, 1), item.Stack{})
			p.StartSneaking()
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := testSession(t)
			island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
			loadAround(s)
			h := spawnPlayer(s)
			at := cube.Pos{4, island.IslandY + 5, 4}
			withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
				c.setup(p, tx, at)
				expand.Show(s.Session, p)
				// looking straight down at the block (the map view moved the
				// body above the map)
				p.Teleport(at.Vec3Centre().Add(mgl64.Vec3{0, 2, 0}))
				p.Move(mgl64.Vec3{}, 0, 90-p.Rotation().Pitch())
				look.Update(s.Session, p, tx, p.Position(), p.Rotation(), true)
				if preview.Shown(s.Session, p, c.owner) {
					t.Fatalf("the movement's update showed the %s preview to an expanding player", c.owner)
				}
			})
			time.Sleep(60 * time.Millisecond) // the fast tick skips a panel updated within 50 ms
			withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
				look.UpdateAll(s.Session, tx)
				if !preview.Shown(s.Session, p, c.owner) {
					t.Fatalf("the fast tick's update did not show the %s preview to an expanding player", c.owner)
				}
			})
		})
	}
}

// TestEntityTypesAsBefore:EntityTypes holds the types the game registered
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

// onlyImports are the packages of the game that may import nothing of the
// game but the listed packages: the look panel and the slot decoration ask
// blocks and items through an interface (waila.Provider, slotdeco.Decorated)
// and so know no feature and no item type.
var onlyImports = map[string][]string{
	"look":  {"core", "naming"},
	"decor": {"naming"},
}

// TestCapabilityImports: the packages of onlyImports import only what their
// row lists.
func TestCapabilityImports(t *testing.T) {
	const prefix = "github.com/presentkim/skygear/content/skygear/"
	for p, allowed := range onlyImports {
		pkg, err := build.Default.ImportDir(p, 0)
		if err != nil {
			t.Errorf("%s: %v", p, err)
			continue
		}
		for _, imp := range pkg.Imports {
			if dep, ok := strings.CutPrefix(imp, prefix); ok && !slices.Contains(allowed, dep) {
				t.Errorf("%s imports %s: a block or an item describes itself through an interface, %s does not import its package", p, dep, p)
			}
		}
	}
}

// TestLookFollowers: what follows a panel update, in the order it runs.
func TestLookFollowers(t *testing.T) {
	if got, want := look.Followers(), []string{"tinker.vein", "wand.preview"}; !slices.Equal(got, want) {
		t.Errorf("look followers %v, want %v", got, want)
	}
}
