package slottext

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/world"
)

func TestPad(t *testing.T) {
	for _, c := range []struct {
		s    string
		n    int
		want string
	}{
		{"", 4, "~~~~"},
		{"ab", 4, "ab~~"},
		{"a~b", 4, "ab~~"},
		{"§f가나다라", 10, "§f가나~"},
		{"abcdef", 4, "abcd"},
	} {
		if got := Pad(c.s, c.n); got != c.want {
			t.Errorf("Pad(%q, %d) = %q, want %q", c.s, c.n, got, c.want)
		}
	}
}

func TestUnique(t *testing.T) {
	seen := map[string]bool{}
	var got []string
	for _, s := range []string{"a", "a", "b", "a", "a§r"} {
		got = append(got, Unique(seen, s))
	}
	want := []string{"a", "a§r", "b", "a§r§r", "a§r§r§r"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Unique %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestItemAux(t *testing.T) {
	rid, _, _ := world.ItemRuntimeID(block.Stone{})
	if aux, ok := ItemAux(block.Stone{}); !ok || aux != int64(rid)*65536 {
		t.Errorf("ItemAux(stone) = %d, %v", aux, ok)
	}
}
