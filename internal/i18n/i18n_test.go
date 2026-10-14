package i18n

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

	"golang.org/x/text/language"
)

var verbs = regexp.MustCompile(`%[-+# 0-9.]*[a-zA-Z%]`)

// TestCatalogs: another language only has keys the default has, with the same
// format verbs.
func TestCatalogs(t *testing.T) {
	for _, l := range Languages() {
		for key, text := range catalogs[l] {
			if strings.HasPrefix(key, "vanilla.") {
				continue // generated from the official texts; may lack a language's name
			}
			def, ok := catalogs[Default][key]
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

func TestLookup(t *testing.T) {
	catalogs["xx"] = map[string]string{"only.xx": "x %d"}
	catalogs[Default]["only.default"] = "d"
	defer delete(catalogs, "xx")
	defer delete(catalogs[Default], "only.default")
	if got := T("xx", "only.default"); got != "d" {
		t.Errorf("a missing key must fall back to the default: %q", got)
	}
	if got := T("xx", "only.xx", 3); got != "x 3" {
		t.Errorf("format: %q", got)
	}
	if got := T("xx", "nope"); got != "nope" {
		t.Errorf("unknown key: %q", got)
	}
	if Lang(language.Korean) != "ko" || Lang(language.French) != Default || Lang(language.AmericanEnglish) != "en" {
		t.Errorf("languages: %s %s", Lang(language.Korean), Lang(language.French))
	}
}

// repoFiles parses the non-test Go files of internal/ (not this package).
func repoFiles(t *testing.T, visit func(path string, f *ast.File, fset *token.FileSet)) {
	t.Helper()
	root := filepath.Join("..")
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") || strings.Contains(filepath.ToSlash(p), "/i18n/") {
			return err
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, p, nil, 0)
		if err != nil {
			return err
		}
		visit(p, f, fset)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestKeysInCodeExist: every i18n.T call with a literal key names a key of
// the default catalog (a typo would show the key to players).
func TestKeysInCodeExist(t *testing.T) {
	n := 0
	repoFiles(t, func(path string, f *ast.File, fset *token.FileSet) {
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
				if _, ok := catalogs[Default][key]; !ok {
					t.Errorf("%s: unknown text key %s", fset.Position(lit.Pos()), key)
				}
			}
			return true
		})
	})
	if n < 100 {
		t.Errorf("only %d literal keys found; is the scan broken?", n)
	}
}

// TestNoKoreanInCode: player-facing Korean lives in the catalog, not in Go
// code. What stays: command names (a command has one name), and the marker
// text stored in time-limited items (timed.go).
func TestNoKoreanInCode(t *testing.T) {
	hangul := regexp.MustCompile(`[가-힣]`)
	repoFiles(t, func(path string, f *ast.File, fset *token.FileSet) {
		ast.Inspect(f, func(node ast.Node) bool {
			lit, ok := node.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING || !hangul.MatchString(lit.Value) {
				return true
			}
			slash := filepath.ToSlash(path)
			if strings.HasSuffix(slash, "timed/timed.go") || strings.HasSuffix(slash, "dragonfactory/menu.go") {
				return true
			}
			t.Errorf("%s: Korean text %s in code; move it to internal/i18n/locales/ko.json", fset.Position(lit.Pos()), lit.Value)
			return true
		})
	})
}
