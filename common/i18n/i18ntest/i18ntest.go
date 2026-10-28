// Package i18ntest has the catalog checks every server runs over its own
// code: format verbs agree between languages, literal keys exist, and no
// player-facing Korean is written in Go code.
package i18ntest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/presentkim/skygear/common/i18n"
)

var verbs = regexp.MustCompile(`%[-+# 0-9.]*[a-zA-Z%]`)

// Catalogs checks that every language has exactly the keys the default has
// (a missing key falls back to the default language), with the same format
// verbs. Keys starting with "vanilla." are generated from the official texts
// and may lack a language's name, so they are skipped.
func Catalogs(t *testing.T) {
	t.Helper()
	for _, l := range i18n.Languages() {
		for _, key := range i18n.Keys(i18n.Default) {
			if _, ok := i18n.Lookup(l, key); !ok && !strings.HasPrefix(key, "vanilla.") {
				t.Errorf("%s: key %s is missing", l, key)
			}
		}
		for _, key := range i18n.Keys(l) {
			if strings.HasPrefix(key, "vanilla.") {
				continue
			}
			text, _ := i18n.Lookup(l, key)
			def, ok := i18n.Lookup(i18n.Default, key)
			if !ok {
				t.Errorf("%s: key %s is not in the default catalog", l, key)
				continue
			}
			if len(verbs.FindAllString(text, -1)) != len(verbs.FindAllString(def, -1)) {
				t.Errorf("%s: %s has other format verbs than the default", l, key)
			}
		}
	}
}

// skipDirs are folders never scanned: copies of other code and assets.
var skipDirs = map[string]bool{"third_party": true, "testdata": true, ".git": true, "vendor": true}

// walk parses the non-test Go files under root, skipping skipDirs and any
// folder for which skip returns true.
func walk(t *testing.T, root string, skip func(slashPath string) bool, visit func(path string, f *ast.File, fset *token.FileSet)) {
	t.Helper()
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		slash := filepath.ToSlash(p)
		if d.IsDir() {
			if p != root && (skipDirs[d.Name()] || (skip != nil && skip(slash))) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, p, nil, 0)
		if err != nil {
			return err
		}
		visit(slash, f, fset)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// KeysInCode checks that every i18n.T call with a literal key under root
// names a key of the default catalog (a typo would show the key to
// players). min is a floor on the keys found, so a broken scan fails.
func KeysInCode(t *testing.T, root string, min int, skip func(slashPath string) bool) {
	t.Helper()
	n := 0
	walk(t, root, skip, func(path string, f *ast.File, fset *token.FileSet) {
		ast.Inspect(f, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) < 2 {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if id, isID := sel.X.(*ast.Ident); !isID || id.Name != "i18n" || sel.Sel.Name != "T" {
				return true
			}
			if lit, ok := call.Args[1].(*ast.BasicLit); ok && lit.Kind == token.STRING {
				n++
				key, _ := strconv.Unquote(lit.Value)
				if !i18n.Has(key) {
					t.Errorf("%s: unknown text key %s", fset.Position(lit.Pos()), key)
				}
			}
			return true
		})
	})
	if n < min {
		t.Errorf("only %d literal keys found; is the scan broken?", n)
	}
}

var hangul = regexp.MustCompile(`[가-힣]`)

// NoKoreanInCode checks that player-facing Korean lives in the catalog, not
// in Go code under root. Files whose slash path ends with one of allow are
// skipped (command names, markers stored in items).
func NoKoreanInCode(t *testing.T, root string, skip func(slashPath string) bool, allow ...string) {
	t.Helper()
	walk(t, root, skip, func(path string, f *ast.File, fset *token.FileSet) {
		for _, a := range allow {
			if strings.HasSuffix(path, a) {
				return
			}
		}
		ast.Inspect(f, func(node ast.Node) bool {
			lit, ok := node.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING || !hangul.MatchString(lit.Value) {
				return true
			}
			t.Errorf("%s: Korean text %s in code; move it to the catalog", fset.Position(lit.Pos()), lit.Value)
			return true
		})
	})
}
