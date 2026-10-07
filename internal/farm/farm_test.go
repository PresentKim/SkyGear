package farm

import (
	"testing"

	_ "github.com/df-mc/dragonfly/server/block"
	_ "github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

func TestLevel(t *testing.T) {
	cases := map[int]int{0: 1, 49: 1, 50: 2, 149: 2, 150: 3, 1 << 30: MaxLevel}
	for xp, want := range cases {
		if got := Level(xp); got != want {
			t.Errorf("Level(%d) = %d, want %d", xp, got, want)
		}
	}
}

func TestPricesAreKnownItems(t *testing.T) {
	for name := range Prices {
		if _, ok := world.ItemByName(name, 0); !ok {
			t.Errorf("unknown item %q in Prices", name)
		}
	}
}
