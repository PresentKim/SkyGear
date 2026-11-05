package netstore

import (
	"testing"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
)

func TestBuildTestNet(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		term, ok := BuildTestNet(tx, p(0), "", "", 30)
		n := At(tx, term)
		if !ok || n == nil || !n.Online() {
			t.Fatal("network offline")
		}
		if got := len(entriesOf(n)); got != 30 {
			t.Fatalf("kinds %d", got)
		}
		if _, ok := BuildTestNet(tx, p(1), "", "", 30); ok {
			t.Fatal("built over blocks")
		}
	})
}
