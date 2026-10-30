package skygear

import (
	"go/build"
	"os"
	"strings"
	"testing"
)

// layers are the game's packages from the bottom up (README.md): a package
// imports only the packages before it, so the dependencies run one way.
var layers = []string{
	// data
	"locales", "naming", "textfmt", "formicon", "blockmodel", "sgitem", "sievetable", "wintext", "machinerecipe",
	"advancement", "hopper", "nsmigrate",
	// core
	"core", "network", "achievement",
	// progress
	"progress", "island", "climate",
	// features
	"tinker", "decor", "machine", "conduit", "storage", "farm", "fairy", "wave", "handwork", "expand", "copier",
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
