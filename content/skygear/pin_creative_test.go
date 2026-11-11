package skygear

import (
	"fmt"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/item/creative"
	"github.com/presentkim/skygear/content/skygear/catalog"
)

// TestPinCreative pins the game's part of the creative inventory as text: one
// line per item, in the order the client lists them, with its group. The
// client-data golden of the wire package pins the same order as hashes; this
// one shows which item moved. It changes only when the creative list is
// changed on purpose.
func TestPinCreative(t *testing.T) {
	catalog.RegisterCreativeItems()
	lines := map[string]string{}
	n := 0
	for _, c := range creative.Items() {
		if !strings.HasPrefix(c.Group, "anon_sg_") {
			continue
		}
		name, meta := c.Stack.Item().EncodeItem()
		lines[fmt.Sprintf("%04d", n)] = fmt.Sprintf("%s %s:%d", c.Group, name, meta)
		n++
	}
	if n == 0 {
		t.Fatal("the game added no creative item")
	}
	pinGolden(t, "testdata/pin/creative.golden", lines)
}
