package blockinternal

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
)

// skygear patch: Properties.NoCollision and NoSelection send the boxes
// disabled, over any box set.
func TestNoBoxes(t *testing.T) {
	c := componentsFromProperties(customblock.Properties{Cube: true, NoCollision: true, NoSelection: true, CollisionBox: cube.Box(0, 0, 0, 1, 1, 1)})
	for _, name := range []string{"minecraft:collision_box", "minecraft:selection_box"} {
		m, ok := c[name].(map[string]any)
		if !ok || m["enabled"] != false {
			t.Errorf("%s = %v, want enabled false", name, c[name])
		}
	}
	c = componentsFromProperties(customblock.Properties{Cube: true})
	if _, ok := c["minecraft:collision_box"]; ok {
		t.Errorf("collision box set by default: %v", c)
	}
	if _, ok := c["minecraft:selection_box"]; ok {
		t.Errorf("selection box set by default: %v", c)
	}
}
