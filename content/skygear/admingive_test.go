package skygear

import (
	"slices"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

// TestGiveListsEveryItem: /sgadmin give reaches every registered item, the
// vanilla ones by their full name, the custom ones by a short id, and only
// the screens' icons and the retired items are left out.
func TestGiveListsEveryItem(t *testing.T) {
	g := giveItems()
	reach := map[string]bool{}
	for _, f := range g {
		reach[encodedKey(f().Item())] = true
	}
	for _, it := range world.Items() {
		if item.NewStack(it, 1).Empty() {
			continue // air
		}
		name, _ := it.EncodeItem()
		id, custom := strings.CutPrefix(name, "skygear:")
		if custom && slices.ContainsFunc(giveSkip, func(pre string) bool { return strings.HasPrefix(id, pre) }) {
			continue
		}
		if !reach[encodedKey(it)] {
			t.Errorf("%s is not in /sgadmin give", encodedKey(it))
		}
	}
	for _, id := range []string{"minecraft:diamond", "minecraft:barrel", "barrel", "conduit_binder", "conduit_binder_mix"} {
		if g[id] == nil {
			t.Errorf("/sgadmin give has no %s", id)
		}
	}
	if a, b := encodedKey(g["barrel"]().Item()), encodedKey(g["minecraft:barrel"]().Item()); a == b {
		t.Errorf("barrel and minecraft:barrel are the same item %s", a)
	}
}

// TestResolveGive: a name without its namespace is taken when only one
// namespace has it, and a name both have asks for the namespace.
func TestResolveGive(t *testing.T) {
	g := giveItems()
	for in, want := range map[string]string{
		"diamond":           "minecraft:diamond",
		"Minecraft:Diamond": "minecraft:diamond",
		"minecraft:barrel":  "minecraft:barrel",
		"skygear:barrel":    "barrel",
		"conduit_binder":    "conduit_binder",
		"skygear:hammer":    "hammer",
	} {
		if got, err := resolveGive(g, in); err != nil || got != want {
			t.Errorf("resolveGive(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"barrel", "no_such_item", "skygear:diamond", "minecraft:hammer"} {
		if got, err := resolveGive(g, in); err == nil {
			t.Errorf("resolveGive(%q) = %q, want an error", in, got)
		}
	}
	for _, id := range (giveEnum("")).Options(nil) {
		if !strings.HasPrefix(id, "skygear:") && !strings.HasPrefix(id, "minecraft:") {
			t.Errorf("completion %q has no namespace", id)
		}
	}
}
