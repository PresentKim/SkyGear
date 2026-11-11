package wire

import (
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/presentkim/skygear/common/sghook/hooktest"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// A custom item's numeric id is its place in the order the items are
// registered in, which is the order the packages are initialised in. The
// goldens of the items, the recipes and the creative list pin those ids, so a
// change of that order alone (an import turned around, a registration moved)
// changes hundreds of their entries and nothing a player can tell.
//
// The goldens by name (testdata/*.byname.golden) hold the same entries with
// every item id written as the item's name, and the set of ids the custom
// items have. A change of the numbering alone leaves them byte for byte as
// they are: that is the proof that nothing but the numbering changed, and the
// lines that did change in the numbered goldens are then ids only.

// itemIDFields are the struct fields that hold a numeric item id.
var itemIDFields = map[reflect.Type]string{
	reflect.TypeFor[protocol.ItemEntry]():             "RuntimeID",
	reflect.TypeFor[protocol.ItemType]():              "NetworkID",
	reflect.TypeFor[protocol.DefaultItemDescriptor](): "NetworkID",
}

// itemNames returns the name of every numeric item id a client is told: the
// ids of the item registry packet and of the start game data. An id with two
// names or a name with two ids fails t.
func itemNames(t testing.TB, registry, start []protocol.ItemEntry) map[int64]string {
	t.Helper()
	names, ids := map[int64]string{}, map[string]int64{}
	for _, list := range [][]protocol.ItemEntry{registry, start} {
		for _, e := range list {
			id := int64(e.RuntimeID)
			if n, ok := names[id]; ok && n != e.Name {
				t.Errorf("item id %d is %s and %s", id, n, e.Name)
			}
			if i, ok := ids[e.Name]; ok && i != id {
				t.Errorf("item %s has the ids %d and %d", e.Name, i, id)
			}
			names[id], ids[e.Name] = e.Name, id
		}
	}
	return names
}

// byName returns hooktest.Canon with every item id of itemIDFields written as
// the item's name. An id names does not hold (0, the empty stack) stays a
// number.
func byName(names map[int64]string) func(any) string {
	return func(v any) string {
		return hooktest.CanonWith(v, func(owner reflect.Type, field string, f reflect.Value) (string, bool) {
			if itemIDFields[owner] != field {
				return "", false
			}
			name, ok := names[f.Int()]
			return "item " + strconv.Quote(name), ok
		})
	}
}

// byNameGoldens pins the items, the recipes and the creative list by name,
// and with the items the sorted ids of the custom items (meta/custom_ids).
func byNameGoldens(t *testing.T, registry, start []protocol.ItemEntry, cd *packet.CraftingData, cc *packet.CreativeContent) {
	canon := byName(itemNames(t, registry, start))
	items := itemEntries(t, canon, registry, start)
	ids := make([]int, 0, len(registry))
	for _, e := range registry {
		ids = append(ids, int(e.RuntimeID))
	}
	sort.Ints(ids)
	items["meta/custom_ids"] = hooktest.Canon(ids)
	hooktest.Golden(t, "testdata/items.byname.golden", items)
	hooktest.Golden(t, "testdata/recipes.byname.golden", recipeEntries(t, canon, cd))
	hooktest.Golden(t, "testdata/creative.byname.golden", creativeEntries(t, canon, cc))
	byNameSensitivity(t, registry, cd)
}

// byNameSensitivity shows what the texts by name leave out and what they
// keep: two custom items that trade their ids change their two entries of
// the numbered texts and no text by name; one changed component value changes
// exactly that item's text by name; a recipe whose result becomes another
// item has another text by name.
func byNameSensitivity(t *testing.T, registry []protocol.ItemEntry, cd *packet.CraftingData) {
	if len(registry) < 2 {
		t.Fatal("fewer than two custom items")
	}
	canon := byName(itemNames(t, registry, nil))
	base := itemEntries(t, canon, registry, nil)

	traded := slices.Clone(registry)
	traded[0].RuntimeID, traded[1].RuntimeID = registry[1].RuntimeID, registry[0].RuntimeID
	want := []string{"registry/" + registry[0].Name, "registry/" + registry[1].Name}
	sort.Strings(want)
	if d := diffEntries(itemEntries(t, hooktest.Canon, registry, nil), itemEntries(t, hooktest.Canon, traded, nil)); !slices.Equal(d, want) {
		t.Errorf("traded ids changed the numbered entries %v, want %v", d, want)
	}
	if d := diffEntries(base, itemEntries(t, byName(itemNames(t, traded, nil)), traded, nil)); len(d) != 0 {
		t.Errorf("traded ids changed the entries by name %v", d)
	}

	tried := false
	for i, e := range registry {
		cp := slices.Clone(registry)
		cp[i].Data = cloneMap(e.Data)
		comps, _ := cp[i].Data["components"].(map[string]any)
		if comps == nil || !bumpFirstScalar(comps) {
			continue
		}
		tried = true
		if d := diffEntries(base, itemEntries(t, canon, cp, nil)); !slices.Equal(d, []string{"registry/" + e.Name}) {
			t.Errorf("item %s, component value: changed entries by name %v", e.Name, d)
		}
		break
	}
	if !tried {
		t.Error("no item with a component to change")
	}

	tried = false
	for _, r := range cd.ShapedRecipes {
		if len(r.Output) == 0 || int64(r.Output[0].NetworkID) == int64(registry[0].RuntimeID) {
			continue
		}
		other := r
		other.Output = slices.Clone(r.Output)
		other.Output[0].NetworkID = int32(registry[0].RuntimeID)
		tried = true
		if got := canon(other); got == canon(r) || !strings.Contains(got, "NetworkID: item "+strconv.Quote(registry[0].Name)) {
			t.Errorf("a recipe that gives %s instead has the text by name:\n%s", registry[0].Name, got)
		}
		break
	}
	if !tried {
		t.Error("no shaped recipe to change")
	}
}
