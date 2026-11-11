package skygear

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
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

// TestHandlerMethodsOneWay: the player handler is the kernel's. Every player
// handler method the game uses is a method of core.PlayerHandler and has at
// least one subscriber on its hook, the root package declares no player
// handler method of its own, and the session's handler is the kernel's.
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
	for method, hook := range playerHandlerHooks {
		if subs[hook] == 0 {
			t.Errorf("%s: hook %s has no subscribers", method, hook)
		}
	}

	files, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, de := range files {
		name := de.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range f.Decls {
			if fd, ok := d.(*ast.FuncDecl); ok && fd.Recv != nil {
				if _, isHandler := playerHandlerHooks[fd.Name.Name]; isHandler {
					t.Errorf("%s declares %s: the kernel's handler serves it", name, fd.Name.Name)
				}
			}
		}
	}

	s := testSession(t)
	if _, ok := s.Handler(nil).(*core.PlayerHandler); !ok {
		t.Errorf("session.Handler is %T, want *core.PlayerHandler", s.Handler(nil))
	}
}
