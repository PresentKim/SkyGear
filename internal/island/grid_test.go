package island

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
)

func TestIndexAtRoundTrip(t *testing.T) {
	g := Grid{Spacing: 512, Radius: 100, Height: 64}
	for _, i := range []int{0, 1, 999, 1000, 1234, 54321} {
		c := g.Centre(i)
		for _, off := range [][2]int{{0, 0}, {100, 100}, {-100, -100}, {100, -100}} {
			p := cube.Pos{c.X() + off[0], 64, c.Z() + off[1]}
			got, ok := g.IndexAt(p)
			if !ok || got != i {
				t.Fatalf("IndexAt(%v) = %d,%v; want %d", p, got, ok, i)
			}
		}
		if _, ok := g.IndexAt(cube.Pos{c.X() + 101, 64, c.Z()}); ok {
			t.Fatalf("position outside radius of island %d is protected", i)
		}
	}
	if _, ok := g.IndexAt(cube.Pos{0, 64, 0}); ok {
		t.Fatal("lobby must not belong to an island")
	}
}
