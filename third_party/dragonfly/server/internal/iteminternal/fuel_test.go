package iteminternal

import (
	"image"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
)

// testFuel is a custom item that burns for five seconds.
type testFuel struct{}

func (testFuel) EncodeItem() (string, int16) { return "test:fuel", 0 }
func (testFuel) Name() string                { return "Fuel" }
func (testFuel) Category() category.Category { return category.Items() }
func (testFuel) Texture() image.Image        { return nil }
func (testFuel) FuelInfo() item.FuelInfo     { return item.FuelInfo{Duration: 5 * time.Second} }

// TestFuelComponent: a custom fuel sends minecraft:fuel with its burn time in
// seconds (skygear patch), so clients let it into a furnace's fuel slot.
func TestFuelComponent(t *testing.T) {
	c := Components(testFuel{})["components"].(map[string]any)
	f, ok := c["minecraft:fuel"].(map[string]any)
	if !ok || f["duration"] != float32(5) {
		t.Fatalf("fuel component %#v", c["minecraft:fuel"])
	}
}
