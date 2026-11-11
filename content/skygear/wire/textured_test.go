package wire

import (
	"fmt"
	"os"
	"testing"

	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/world"
)

// untextured lists the registered items with a Name and a Category that are
// not a world.CustomItem. Dragonfly leaves such an item out of the client's
// item registry, so the list must be empty: every one of them needs Texture.
func untextured() (bad []string, checked int) {
	type named interface {
		Name() string
		Category() category.Category
	}
	for _, it := range world.Items() {
		if _, ok := it.(named); !ok {
			continue
		}
		checked++
		if _, ok := it.(world.CustomItem); !ok {
			bad = append(bad, fmt.Sprintf("%T", it))
		}
	}
	return bad, checked
}

// TestMain runs the guard after the tests: the item registry is filled once
// per process by the join of TestWire, and a second join would register the
// recipes and the creative list twice. Skipped when no test joined.
func TestMain(m *testing.M) {
	code := m.Run()
	if bad, checked := untextured(); checked > 0 && len(bad) > 0 {
		fmt.Fprintf(os.Stderr, "FAIL TestCustomItemsTextured: no Texture method on %v\n", bad)
		code = 1
	}
	os.Exit(code)
}
