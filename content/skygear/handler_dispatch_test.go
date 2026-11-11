package skygear

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"

	"github.com/presentkim/skygear/content/skygear/core"
)

// playerHandlerHooks maps each player handler method to the hook it emits.
var playerHandlerHooks = map[string]string{
	"HandleStartBreak":       "StartBreak",
	"HandleBlockBreak":       "BlockBreak",
	"HandleBlockPlace":       "BlockPlace",
	"HandleItemUseOnBlock":   "UseOnBlock",
	"HandleItemUseOnEntity":  "UseOnEntity",
	"HandleAttackEntity":     "AttackEntity",
	"HandleItemUse":          "Use",
	"HandleMove":             "Move",
	"HandleToggleSneak":      "Sneak",
	"HandleToggleSprint":     "Sprint",
	"HandleJump":             "Jump",
	"HandleItemDamage":       "ItemDamage",
	"HandleItemConsume":      "ItemConsume",
	"HandleItemPickup":       "ItemPickup",
	"HandleItemDrop":         "ItemDrop",
	"HandleDeath":            "Death",
	"HandleRespawn":          "Respawn",
	"HandleHurt":             "Hurt",
	"HandlePunchAir":         "PunchAir",
	"HandleChat":             "Chat",
	"HandleBlockPick":        "BlockPick",
	"HandleHeldSlotChange":   "HeldSlot",
	"HandleCommandExecution": "Command",
}

// TestHandlerMethodsOneWay: each player handler method is served either by
// a legacy body in handler.go or by the subscribers of its hook, never both
// and never neither. A body that stays for a hook with subscribers must only
// forward to the kernel handler's method of the same name.
func TestHandlerMethodsOneWay(t *testing.T) {
	kt := reflect.TypeOf(&core.PlayerHandler{})
	for m := range playerHandlerHooks {
		if _, ok := kt.MethodByName(m); !ok {
			t.Errorf("core.PlayerHandler has no method %s", m)
		}
	}

	subs := map[string]int{}
	for _, s := range core.Subscriptions() {
		subs[s.Hook]++
	}

	f, err := parser.ParseFile(token.NewFileSet(), "handler.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	declared := map[string]*ast.FuncDecl{}
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Recv == nil || !strings.HasPrefix(fd.Name.Name, "Handle") {
			continue
		}
		if star, ok := fd.Recv.List[0].Type.(*ast.StarExpr); !ok || star.X.(*ast.Ident).Name != "handler" {
			continue
		}
		declared[fd.Name.Name] = fd
	}
	for name := range declared {
		if _, ok := playerHandlerHooks[name]; !ok {
			t.Errorf("handler.go declares %s, which is no player handler method of the table", name)
		}
	}

	for method, hook := range playerHandlerHooks {
		n := subs[hook]
		fd, legacy := declared[method]
		switch {
		case legacy && forwards(fd, method):
			if n == 0 {
				t.Errorf("%s forwards to the kernel, but hook %s has no subscribers", method, hook)
			}
		case legacy && n > 0:
			t.Errorf("%s has a legacy body and hook %s has %d subscribers: served twice", method, hook, n)
		case !legacy && n == 0:
			t.Errorf("%s has no legacy body and hook %s has no subscribers: served by nobody", method, hook)
		}
	}
}

// forwards reports whether fd's body is one call of the kernel handler's
// method of the same name.
func forwards(fd *ast.FuncDecl, method string) bool {
	if len(fd.Body.List) != 1 {
		return false
	}
	var call *ast.CallExpr
	switch s := fd.Body.List[0].(type) {
	case *ast.ExprStmt:
		call, _ = s.X.(*ast.CallExpr)
	case *ast.ReturnStmt:
		if len(s.Results) == 1 {
			call, _ = s.Results[0].(*ast.CallExpr)
		}
	}
	if call == nil {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != method {
		return false
	}
	in, ok := sel.X.(*ast.SelectorExpr)
	return ok && in.Sel.Name == "PlayerHandler"
}
