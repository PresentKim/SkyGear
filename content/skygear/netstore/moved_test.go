package netstore

import (
	"testing"

	"github.com/df-mc/dragonfly/server/item"
)

func TestMovedCounts(t *testing.T) {
	flint := func(n int) item.Stack { return item.NewStack(item.Flint{}, n) }
	for _, c := range []struct {
		name          string
		before, after item.Stack
		taken, put    int
	}{
		{"all taken", flint(64), item.Stack{}, 64, 0},
		{"half taken", flint(64), flint(32), 32, 0},
		{"put in empty", item.Stack{}, flint(5), 0, 5},
		{"added to", flint(10), flint(15), 0, 5},
	} {
		taken, put := moved(c.before, c.after)
		if taken.Count() != c.taken || put.Count() != c.put {
			t.Errorf("%s: taken %d put %d", c.name, taken.Count(), put.Count())
		}
	}
}
