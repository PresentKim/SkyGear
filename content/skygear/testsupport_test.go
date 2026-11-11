package skygear

import (
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
)

// testSupport are the packages that exist for tests (module-relative paths):
// they give a package's tests a session, a player and the features they need
// without the root package.
var testSupport = []string{"content/skygear/core/coretest", "content/skygear/machine/machinetest"}

// TestSupportPackagesAreTestOnly: no non-test file of the module imports a
// package of testSupport, so the server never links one.
func TestSupportPackagesAreTestOnly(t *testing.T) {
	err := filepath.WalkDir(filepath.Join("..", ".."), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "third_party":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(token.NewFileSet(), p, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, im := range f.Imports {
			path, _ := strconv.Unquote(im.Path.Value)
			if rel, ok := strings.CutPrefix(path, modulePrefix); ok && slices.Contains(testSupport, rel) {
				t.Errorf("%s imports %s: only test files import a test support package", p, rel)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestPackageTestsStayBelow: the tests of a package run without the composed
// game. No package's tests import the root package (wire excepted: it tests
// what the composed game sends a client), and the tests of a package below
// the content lists import nothing from the content lists up (catalog, codex,
// screens): a test that needs the game's recipes, research or screens belongs
// in the root package. A package's tests set features up through
// coretest.Setup, which remembers what it set up: outside core, no test file
// calls core.Install or a feature's Setup itself.
func TestPackageTestsStayBelow(t *testing.T) {
	const root = modulePrefix + "content/skygear"
	top := slices.Index(layers, "catalog")
	if top < 0 {
		t.Fatal("layers has no catalog")
	}
	for i, p := range layers {
		pkg, err := build.Default.ImportDir(p, 0)
		if err != nil {
			t.Errorf("%s: %v", p, err)
			continue
		}
		for _, imp := range slices.Concat(pkg.TestImports, pkg.XTestImports) {
			if imp == root && p != "wire" {
				t.Errorf("the tests of %s import the root package: a test of the composed game belongs in the root package", p)
			}
			if dep, ok := strings.CutPrefix(imp, root+"/"); ok && i < top && slices.Contains(layers[top:], dep) {
				t.Errorf("the tests of %s import %s: a test that needs the content lists or the screens belongs in the root package", p, dep)
			}
		}
		if p == "core" {
			continue
		}
		for _, name := range slices.Concat(pkg.TestGoFiles, pkg.XTestGoFiles) {
			src, err := os.ReadFile(filepath.Join(p, name))
			if err != nil {
				t.Fatal(err)
			}
			for _, call := range []string{"core.Install(", "Feature.Setup()"} {
				if strings.Contains(string(src), call) {
					t.Errorf("%s/%s calls %s: a package's tests set features up with coretest.Setup", p, name, call)
				}
			}
		}
	}
}
