package skygear

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/presentkim/skygear/content/skygear/core"
)

// TestMonitorSubscribersDoNotShape: a Monitor subscriber handles the result
// of an event and neither shapes nor cancels it. The kernel refuses Handle in
// the Monitor tier (core.TestHandleAtMonitorPanics); this test reads the
// function literal of every subscriber registered with core.Monitor in a
// feature package and fails when it calls Cancel or assigns through one of the
// event's pointers (*e.Drops, *e.XP, *e.Damage, *e.Pos, *e.Pick). It sees the
// literal only, not what the literal calls.
func TestMonitorSubscribersDoNotShape(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("*", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) != 3 {
				return true
			}
			if tier, ok := call.Args[0].(*ast.SelectorExpr); !ok || tier.Sel.Name != "Monitor" {
				return true
			}
			name := strings.Trim(call.Args[1].(*ast.BasicLit).Value, `"`)
			found++
			lit, ok := call.Args[2].(*ast.FuncLit)
			if !ok {
				t.Errorf("%s: %s: a Monitor subscriber is a function literal, so this test can read it", fset.Position(call.Pos()), name)
				return true
			}
			ast.Inspect(lit.Body, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.CallExpr:
					if sel, ok := n.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Cancel" {
						t.Errorf("%s: %s cancels the event; a Monitor subscriber does not (use an earlier tier)", fset.Position(n.Pos()), name)
					}
				case *ast.AssignStmt:
					for _, lhs := range n.Lhs {
						if star, ok := lhs.(*ast.StarExpr); ok {
							if sel, ok := star.X.(*ast.SelectorExpr); ok {
								t.Errorf("%s: %s writes *%s.%s; a Monitor subscriber does not shape the event (use an earlier tier)", fset.Position(n.Pos()), name, exprName(sel.X), sel.Sel.Name)
							}
						}
					}
				}
				return true
			})
			return true
		})
	}
	want := 0
	for _, s := range core.Subscriptions() {
		if s.Tier == core.Monitor {
			want++
		}
	}
	if found != want {
		t.Errorf("read %d Monitor subscribers in the feature packages, the kernel lists %d", found, want)
	}
}

func exprName(e ast.Expr) string {
	if id, ok := e.(*ast.Ident); ok {
		return id.Name
	}
	return "?"
}
