package core

import (
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/player"
)

// cleanTakeovers empties the takeovers for one test and again when it ends.
func cleanTakeovers(t *testing.T) {
	t.Helper()
	resetTakeovers()
	t.Cleanup(resetTakeovers)
}

func TestTakeoverNames(t *testing.T) {
	cleanTakeovers(t)
	withFix(t, nil, func(x *fix) {
		holding := map[string]bool{}
		holds := func(name string) func(*Session, *player.Player) bool {
			return func(s *Session, p *player.Player) bool { return s == x.s && p == x.p && holding[name] }
		}
		if got := Takeover(x.s, x.p); got != "" {
			t.Fatalf("no takeover registered: Takeover = %q", got)
		}
		RegisterTakeover("a", holds("a"))
		RegisterTakeover("b.mode", holds("b.mode"))
		if got := Takeover(x.s, x.p); got != "" {
			t.Fatalf("none holds: Takeover = %q", got)
		}
		holding["b.mode"] = true
		if got := Takeover(x.s, x.p); got != "b.mode" {
			t.Fatalf("b.mode holds: Takeover = %q", got)
		}
		holding["a"] = true
		if got := Takeover(x.s, x.p); got != "a" {
			t.Fatalf("both hold: Takeover = %q, want the first registered", got)
		}
	})
}

func TestRegisterTakeoverPanics(t *testing.T) {
	cleanTakeovers(t)
	never := func(*Session, *player.Player) bool { return false }
	RegisterTakeover("a", never)
	for _, c := range []struct {
		name  string
		holds func(*Session, *player.Player) bool
		want  string
	}{
		{"", never, "empty"},
		{"a", never, "twice"},
		{"b", nil, "nil"},
	} {
		func() {
			defer func() {
				r, _ := recover().(string)
				if !strings.Contains(r, c.want) {
					t.Errorf("RegisterTakeover(%q): panic %q, want one that says %q", c.name, r, c.want)
				}
			}()
			RegisterTakeover(c.name, c.holds)
		}()
	}
}
