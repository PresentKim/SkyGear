package skygear

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/presentkim/skygear/content/skygear/core"
)

// TestPinOrder compares the kernel's subscriptions with
// testdata/pin/order.golden: one line "<Hook> <TIER> <name>" per subscriber,
// with " handleCancelled" after the name of a subscriber that also receives
// a cancelled event (core.HandleCancelled), hooks in declaration order, each
// hook's tiers in order. The subscribers of one tier have no order: their
// lines are sorted by name, in the file and in the comparison. It names no
// feature, so it does not change when a step moves to its feature.
//
// Re-record it (SKYGEAR_GOLDEN=update) only when a step is added, removed or
// moved to another tier on purpose.
func TestPinOrder(t *testing.T) {
	var lines []string
	for _, s := range core.Subscriptions() {
		ln := fmt.Sprintf("%s %s %s", s.Hook, s.Tier, s.Name)
		if s.HandleCancelled {
			ln += " handleCancelled"
		}
		lines = append(lines, ln)
	}
	sortTiers(lines)
	const file = "testdata/pin/order.golden"
	if os.Getenv("SKYGEAR_GOLDEN") == "update" {
		if err := os.WriteFile(file, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	wl := strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n")
	for i := 0; i < len(wl) || i < len(lines); i++ {
		var w, g string
		if i < len(wl) {
			w = wl[i]
		}
		if i < len(lines) {
			g = lines[i]
		}
		if w != g {
			t.Fatalf("%s line %d:\nwant: %q\ngot:  %q", file, i+1, w, g)
		}
	}
}

// sortTiers sorts each run of lines that share "<Hook> <TIER>".
func sortTiers(lines []string) {
	tier := func(ln string) string {
		f := strings.SplitN(ln, " ", 3)
		return f[0] + " " + f[1]
	}
	for from := 0; from < len(lines); {
		to := from
		for to < len(lines) && tier(lines[to]) == tier(lines[from]) {
			to++
		}
		slices.Sort(lines[from:to])
		from = to
	}
}

// The tests below keep the root package a feature list. Adding a feature
// touches, in the production code, its own package and one line (plus its
// import) in features.go: the audit counted 8 central files for adding
// `elevator` before the kernel (handler.go, second.go, game.go, settings.go,
// the entity list, the item edit, the lost/found and conduit-load glue). The
// test side has tables a new package also fills, on purpose: its row in
// layers and featureLayer and its name in TestFeatureNames' want list
// (layers_test.go) and, once it subscribes, its lines in
// testdata/pin/order.golden and the scenario goldens that reach its hook.
// core/newfeature_test.go proves that a feature
// with a subscriber receives its event with no other central change.

const modulePrefix = "github.com/presentkim/skygear/"

// rootSources parses the non-test Go files of the root package.
func rootSources(t *testing.T) map[string]*ast.File {
	t.Helper()
	files, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]*ast.File{}
	for _, de := range files {
		name := de.Name()
		if de.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		out[name] = f
	}
	return out
}

// TestRootSubscribesToNothing: no non-test file of the root package calls a
// hook's On, calls core.Mark, declares a Handle* method or is named
// legacy_*.go. A step belongs in the Setup of the feature that owns it.
func TestRootSubscribesToNothing(t *testing.T) {
	const hint = "a step belongs in the Setup of its feature package (content/skygear/<feature>/feature.go); the root package only lists features and holds platform glue"
	for name, f := range rootSources(t) {
		if strings.HasPrefix(name, "legacy_") {
			t.Errorf("%s: the root package holds no legacy_*.go file: %s", name, hint)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.FuncDecl:
				if n.Recv != nil && strings.HasPrefix(n.Name.Name, "Handle") {
					t.Errorf("%s: method %s looks like a Dragonfly handler method; the kernel's handler serves events: %s", name, n.Name.Name, hint)
				}
			case *ast.CallExpr:
				sel, ok := n.Fun.(*ast.SelectorExpr)
				if !ok {
					break
				}
				if id, ok := sel.X.(*ast.Ident); ok && id.Name == "core" && sel.Sel.Name == "Mark" {
					t.Errorf("%s: core.Mark call: %s", name, hint)
				}
				if sel.Sel.Name == "On" && (len(n.Args) == 3 || len(n.Args) == 4) {
					t.Errorf("%s: On(tier, name, f) subscribes to a hook: %s", name, hint)
				}
			}
			return true
		})
	}
}

// TestEverySubscriberHasItsFeature: every subscriber was registered during the
// Setup of the feature its name starts with, and that feature is in Features.
// The kernel's own feature (core) is in the list like the rest.
// The count is that of order.golden.
func TestEverySubscriberHasItsFeature(t *testing.T) {
	listed := map[string]bool{}
	for _, f := range Features {
		listed[f.Name] = true
	}
	subs := core.Subscriptions()
	for _, s := range subs {
		prefix, _, _ := strings.Cut(s.Name, ".")
		switch {
		case s.Feature == "":
			t.Errorf("%s %s: registered outside any feature's Setup; subscribe from the Setup of a feature listed in features.go", s.Hook, s.Name)
		case s.Feature != prefix:
			t.Errorf("%s %s: registered by feature %q, named for %q", s.Hook, s.Name, s.Feature, prefix)
		case !listed[s.Feature]:
			t.Errorf("%s %s: feature %q is not in Features; add it to the list in features.go", s.Hook, s.Name, s.Feature)
		}
	}
	raw, err := os.ReadFile("testdata/pin/order.golden")
	if err != nil {
		t.Fatal(err)
	}
	if want := strings.Count(string(raw), "\n"); len(subs) != want {
		t.Errorf("%d subscribers, order.golden has %d", len(subs), want)
	}
}

// featureDirs returns the sub-directories of the root package, core excepted,
// that hold a feature.go, with the Name each declares.
func featureDirs(t *testing.T) map[string]string {
	t.Helper()
	dirs, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, d := range dirs {
		if !d.IsDir() || d.Name() == "core" { // core/feature.go declares the Feature type
			continue
		}
		path := filepath.Join(d.Name(), "feature.go")
		if _, err := os.Stat(path); err != nil {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		name := ""
		ast.Inspect(f, func(n ast.Node) bool {
			if kv, ok := n.(*ast.KeyValueExpr); ok {
				if k, ok := kv.Key.(*ast.Ident); ok && k.Name == "Name" {
					if lit, ok := kv.Value.(*ast.BasicLit); ok && name == "" {
						name, _ = strconv.Unquote(lit.Value)
					}
				}
			}
			return true
		})
		out[d.Name()] = name
	}
	return out
}

// TestEveryFeatureListed: the list in features.go is the one place features
// are named. Each package with a feature.go (core excepted: its feature.go
// declares the Feature type) declares a Feature named for the package and is
// in Features exactly once; each listed feature is such a
// package, except the kernel's (core) and the root's own (items); the
// list is the one core.Features reports. To add a feature, add it to Features
// in features.go.
func TestEveryFeatureListed(t *testing.T) {
	count := map[string]int{}
	for _, f := range Features {
		count[f.Name]++
	}
	dirs := featureDirs(t)
	for dir, name := range dirs {
		if name != dir {
			t.Errorf("%s/feature.go declares the feature %q; name it %q", dir, name, dir)
		}
		if count[dir] != 1 {
			t.Errorf("package %s has a feature.go but is in Features %d times; add %s.Feature to the list in features.go (once)", dir, count[dir], dir)
		}
	}
	notPackage := map[string]bool{"core": true, "items": true}
	for name, n := range count {
		if n > 1 {
			t.Errorf("feature %q is listed %d times", name, n)
		}
		if _, ok := dirs[name]; !ok && !notPackage[name] {
			t.Errorf("Features lists %q, which has no package with a feature.go", name)
		}
	}
	if got := core.Features(); !slices.Equal(got, Features) {
		t.Errorf("core.Features() is not the list in features.go")
	}
}

// importClosure returns every package of the module that the package in dir
// imports, directly or not, as module-relative paths. dir is relative to the
// root package.
func importClosure(t *testing.T, dir string) map[string]bool {
	t.Helper()
	seen := map[string]bool{}
	var walk func(dir string)
	walk = func(dir string) {
		pkg, err := build.Default.ImportDir(dir, 0)
		if err != nil {
			t.Errorf("%s: %v", dir, err)
			return
		}
		for _, imp := range pkg.Imports {
			rel, ok := strings.CutPrefix(imp, modulePrefix)
			if !ok || seen[rel] {
				continue
			}
			seen[rel] = true
			walk(filepath.Join("..", "..", filepath.FromSlash(rel)))
		}
	}
	walk(dir)
	return seen
}

// TestFeaturesLiftable: a feature package can be lifted out of the game. No
// feature package depends, directly or not, on the root package; core depends
// on no feature package; common/* depends on nothing under content/ or
// platform/.
func TestFeaturesLiftable(t *testing.T) {
	const root = "content/skygear"
	dirs := featureDirs(t)
	for dir := range dirs {
		if dir == "core" {
			continue
		}
		if importClosure(t, dir)[root] {
			t.Errorf("feature package %s imports the root package (%s); a feature reaches the game only through core", dir, root)
		}
	}
	for dep := range importClosure(t, "core") {
		if dep == root {
			t.Errorf("core imports the root package")
		}
		if sub, ok := strings.CutPrefix(dep, root+"/"); ok {
			if _, isFeature := dirs[strings.SplitN(sub, "/", 2)[0]]; isFeature {
				t.Errorf("core imports the feature package %s; the kernel knows no feature", dep)
			}
		}
	}
	err := filepath.WalkDir(filepath.Join("..", "..", "common"), func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return err
		}
		if _, err := build.Default.ImportDir(p, 0); err != nil {
			return nil // no Go files here
		}
		for dep := range importClosure(t, p) {
			if strings.HasPrefix(dep, "content/") || strings.HasPrefix(dep, "platform/") {
				t.Errorf("%s imports %s; common/* imports nothing under content/ or platform/", p, dep)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
