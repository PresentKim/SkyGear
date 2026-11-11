package wire

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item/creative"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/presentkim/skygear/common/sghook/hooktest"
	"github.com/presentkim/skygear/content/skygear"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/platform/db"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// join starts the server the way cmd/skygear does (entity registry, item
// entry edit, the game's content) and returns the client and the packets.
func join(t *testing.T) (*hooktest.Client, *packet.ItemRegistry, *packet.CraftingData, *packet.CreativeContent) {
	t.Helper()
	d, err := db.Open(filepath.Join(t.TempDir(), "platform.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	var conf server.Config
	conf.Entities = entity.DefaultRegistry.Config().New(skygear.EntityTypes())
	c := hooktest.Join(t, conf, func(srv *server.Server) {
		core.EditEntries(srv)
		skygear.New(slog.New(slog.DiscardHandler), d, nil, "test")
	})
	var (
		reg *packet.ItemRegistry
		cd  *packet.CraftingData
		cc  *packet.CreativeContent
	)
	for _, pk := range c.Packets() {
		switch p := pk.(type) {
		case *packet.ItemRegistry:
			reg = p
		case *packet.CraftingData:
			cd = p
		case *packet.CreativeContent:
			cc = p
		}
	}
	return c, reg, cd, cc
}

// TestWire pins what a joining client is sent, from one shared join. The
// golden files change only in a commit meant to change what clients get.
func TestWire(t *testing.T) {
	c, reg, cd, cc := join(t)
	data := c.GameData()

	t.Run("blocks", func(t *testing.T) {
		entries := blockEntries(t, data.CustomBlocks)
		reg := world.CustomBlocks()
		for name := range reg {
			if _, ok := entries[name]; !ok {
				t.Errorf("registered custom block %s is not pinned", name)
			}
		}
		if len(entries) != len(data.CustomBlocks)+1 {
			t.Errorf("pinned %d entries for %d start game blocks", len(entries), len(data.CustomBlocks))
		}
		t.Logf("blocks: %d pinned (%d registered by the game, %d more sent by Dragonfly's own data)", len(data.CustomBlocks), len(reg), len(data.CustomBlocks)-len(reg))
		hooktest.Golden(t, "testdata/blocks.golden", entries)
	})
	// The canonical text keeps the order of the properties list, but no block
	// the game registers has two properties yet (recorded from the patched
	// server: 0 of 205), so the golden cannot see the order. This checks it
	// directly: the client numbers the states by it.
	t.Run("property order", func(t *testing.T) {
		multi := 0
		for _, e := range data.CustomBlocks {
			var names []string
			switch l := e.Properties["properties"].(type) {
			case nil:
			case []map[string]any:
				for _, p := range l {
					names = append(names, p["name"].(string))
				}
			default:
				t.Errorf("block %s: properties are %T", e.Name, l)
			}
			if len(names) > 1 {
				multi++
			}
			if !sort.StringsAreSorted(names) {
				t.Errorf("block %s: properties %v are not in the order of their names", e.Name, names)
			}
		}
		t.Logf("%d of %d blocks have more than one property", multi, len(data.CustomBlocks))
	})
	t.Run("items", func(t *testing.T) {
		entries := itemEntries(t, reg.Items, data.Items)
		custom := 0
		for _, e := range reg.Items {
			if e.ComponentBased {
				custom++
			}
		}
		if len(reg.Items) != len(world.CustomItems()) {
			t.Errorf("registry packet has %d items, the game registered %d custom items", len(reg.Items), len(world.CustomItems()))
		}
		t.Logf("items: %d registry + %d start entries pinned; custom items registered %d, component based in packet %d",
			len(reg.Items), len(data.Items), len(world.CustomItems()), custom)
		hooktest.Golden(t, "testdata/items.golden", entries)
	})
	t.Run("recipes", func(t *testing.T) {
		entries := recipeEntries(t, cd)
		if want := len(recipe.Recipes()) + 1; len(entries) != want {
			t.Errorf("pinned %d recipe entries (with the clear flag), registry has %d", len(entries), want)
		}
		t.Logf("recipes: %d entries pinned; recipe registry %d, dynamic %d", len(entries), len(recipe.Recipes()), len(recipe.DynamicRecipes()))
		hooktest.Golden(t, "testdata/recipes.golden", entries)
	})
	t.Run("creative", func(t *testing.T) {
		entries := creativeEntries(t, cc)
		if len(cc.Groups) != len(creative.Groups()) || len(cc.Items) != len(creative.Items()) {
			t.Errorf("packet has %d groups, %d items; registry has %d, %d", len(cc.Groups), len(cc.Items), len(creative.Groups()), len(creative.Items()))
		}
		t.Logf("creative: %d groups + %d items pinned; registry %d + %d", len(cc.Groups), len(cc.Items), len(creative.Groups()), len(creative.Items()))
		hooktest.Golden(t, "testdata/creative.golden", entries)
	})
	t.Run("sensitivity", func(t *testing.T) { sensitivity(t, data.CustomBlocks, reg.Items) })
}

// blockEntries returns one canonical text per custom block, plus
// "meta/block_ids". The entry order, the permutation order and the
// vanilla_block_data.block_id values follow map order in every process (the
// ids are a permutation of a fixed set), so a text is made of a copy in which
// permutations are sorted by condition and block_id is 0; the set of ids is
// pinned apart as meta/block_ids (sorted). Every other value is kept.
func blockEntries(t testing.TB, blocks []protocol.BlockEntry) map[string]string {
	t.Helper()
	out := make(map[string]string, len(blocks)+1)
	var ids []int
	for _, e := range blocks {
		if _, dup := out[e.Name]; dup {
			t.Errorf("duplicate block %s", e.Name)
		}
		props := cloneMap(e.Properties)
		if vbd, ok := props["vanilla_block_data"].(map[string]any); ok {
			id, ok := vbd["block_id"].(int32)
			if !ok {
				t.Errorf("block %s: block_id is %T", e.Name, vbd["block_id"])
			}
			ids = append(ids, int(id))
			vbd["block_id"] = int32(0)
		} else {
			t.Errorf("block %s: no vanilla_block_data", e.Name)
		}
		if err := sortPermutations(props); err != nil {
			t.Errorf("block %s: %v", e.Name, err)
		}
		out[e.Name] = hooktest.Canon(protocol.BlockEntry{Name: e.Name, Properties: props})
	}
	sort.Ints(ids)
	for i := 1; i < len(ids); i++ {
		if ids[i] == ids[i-1] {
			t.Errorf("block_id %d used twice", ids[i])
		}
	}
	out["meta/block_ids"] = hooktest.Canon(ids)
	return out
}

// sortPermutations orders props["permutations"] by condition.
func sortPermutations(props map[string]any) error {
	p, ok := props["permutations"]
	if !ok {
		return nil
	}
	var cond func(i int) string
	var swap func(i, j int)
	var n int
	switch ps := p.(type) {
	case []map[string]any:
		n, swap = len(ps), func(i, j int) { ps[i], ps[j] = ps[j], ps[i] }
		cond = func(i int) string { s, _ := ps[i]["condition"].(string); return s }
	case []any:
		n, swap = len(ps), func(i, j int) { ps[i], ps[j] = ps[j], ps[i] }
		cond = func(i int) string {
			m, _ := ps[i].(map[string]any)
			s, _ := m["condition"].(string)
			return s
		}
	default:
		return fmt.Errorf("permutations are %T", p)
	}
	for i := range n {
		if cond(i) == "" {
			return fmt.Errorf("permutation %d has no condition", i)
		}
	}
	sort.Sort(sorter{n, func(i, j int) bool { return cond(i) < cond(j) }, swap})
	for i := 1; i < n; i++ {
		if cond(i) == cond(i-1) {
			return fmt.Errorf("condition %q twice", cond(i))
		}
	}
	return nil
}

type sorter struct {
	n    int
	less func(i, j int) bool
	swap func(i, j int)
}

func (s sorter) Len() int           { return s.n }
func (s sorter) Less(i, j int) bool { return s.less(i, j) }
func (s sorter) Swap(i, j int)      { s.swap(i, j) }

// itemEntries returns registry/<name> for the item registry packet and
// start/<name> for the start game data.
func itemEntries(t testing.TB, registry, start []protocol.ItemEntry) map[string]string {
	t.Helper()
	out := make(map[string]string, len(registry)+len(start))
	for prefix, list := range map[string][]protocol.ItemEntry{"registry/": registry, "start/": start} {
		for _, e := range list {
			k := prefix + e.Name
			if _, dup := out[k]; dup {
				t.Errorf("duplicate item %s", k)
			}
			out[k] = hooktest.Canon(e)
		}
	}
	return out
}

// recipeEntries returns <kind>/<RecipeNetworkID> per recipe and <kind>/<index>
// for the recipes without a network id.
func recipeEntries(t testing.TB, cd *packet.CraftingData) map[string]string {
	t.Helper()
	out := map[string]string{}
	add := func(k string, v any) {
		if _, dup := out[k]; dup {
			t.Errorf("duplicate recipe %s", k)
		}
		out[k] = hooktest.Canon(v)
	}
	for _, r := range cd.ShapedRecipes {
		add(fmt.Sprintf("shaped/%d", r.RecipeNetworkID), r)
	}
	for _, r := range cd.ShapelessRecipes {
		add(fmt.Sprintf("shapeless/%d", r.RecipeNetworkID), r)
	}
	for _, r := range cd.MultiRecipes {
		add(fmt.Sprintf("multi/%d", r.RecipeNetworkID), r)
	}
	for _, r := range cd.UserDataShapelessRecipes {
		add(fmt.Sprintf("userdata_shapeless/%d", r.RecipeNetworkID), r)
	}
	for _, r := range cd.ShapelessChemistryRecipes {
		add(fmt.Sprintf("shapeless_chemistry/%d", r.RecipeNetworkID), r)
	}
	for _, r := range cd.ShapedChemistryRecipes {
		add(fmt.Sprintf("shaped_chemistry/%d", r.RecipeNetworkID), r)
	}
	for _, r := range cd.SmithingTransformRecipes {
		add(fmt.Sprintf("smithing_transform/%d", r.RecipeNetworkID), r)
	}
	for _, r := range cd.SmithingTrimRecipes {
		add(fmt.Sprintf("smithing_trim/%d", r.RecipeNetworkID), r)
	}
	for i, r := range cd.PotionRecipes {
		add(fmt.Sprintf("potion/%d", i), r)
	}
	for i, r := range cd.PotionContainerChangeRecipes {
		add(fmt.Sprintf("potion_container/%d", i), r)
	}
	for i, r := range cd.MaterialReducers {
		add(fmt.Sprintf("material_reducer/%d", i), r)
	}
	add("clear", cd.ClearRecipes)
	return out
}

// creativeEntries returns group/<index> and item/<CreativeItemNetworkID>, and
// order/items, the network ids in the order the client shows them.
func creativeEntries(t testing.TB, cc *packet.CreativeContent) map[string]string {
	t.Helper()
	out := map[string]string{}
	for i, g := range cc.Groups {
		out[fmt.Sprintf("group/%d", i)] = hooktest.Canon(g)
	}
	var order []uint32
	for _, it := range cc.Items {
		k := fmt.Sprintf("item/%d", it.CreativeItemNetworkID)
		if _, dup := out[k]; dup {
			t.Errorf("duplicate creative item %s", k)
		}
		out[k] = hooktest.Canon(it)
		order = append(order, it.CreativeItemNetworkID)
	}
	out["order/items"] = hooktest.Canon(order)
	return out
}

// sensitivity shows, without touching production code, that the entries
// are hashed with every component value: it changes one value of one custom
// block (a bone, a collision box, a tool tag, a permutation) and of one
// custom item, and expects exactly that entry to differ. It also shuffles the
// block order, the permutations and the ids, which must change nothing.
func sensitivity(t *testing.T, blocks []protocol.BlockEntry, items []protocol.ItemEntry) {
	base := blockEntries(t, blocks)
	diff := func(a, b map[string]string) []string {
		var d []string
		for k, v := range a {
			if b[k] != v {
				d = append(d, k)
			}
		}
		for k := range b {
			if _, ok := a[k]; !ok {
				d = append(d, k)
			}
		}
		sort.Strings(d)
		return d
	}
	clone := func(i int, mutate func(props map[string]any) bool) (map[string]string, string) {
		cp := slices.Clone(blocks)
		cp[i] = protocol.BlockEntry{Name: blocks[i].Name, Properties: cloneMap(blocks[i].Properties)}
		if !mutate(cp[i].Properties) {
			return nil, ""
		}
		return blockEntries(t, cp), blocks[i].Name
	}
	comp := func(p map[string]any, name string) map[string]any {
		m, _ := p["components"].(map[string]any)
		c, _ := m[name].(map[string]any)
		return c
	}
	cases := []struct {
		what   string
		mutate func(p map[string]any) bool
	}{
		{"bone visibility flipped", func(p map[string]any) bool {
			bv, _ := comp(p, "minecraft:geometry")["bone_visibility"].(map[string]any)
			for k, v := range bv {
				switch x := v.(type) {
				case string:
					bv[k] = x + "!"
				case uint8:
					bv[k] = x ^ 1
				case bool:
					bv[k] = !x
				default:
					continue
				}
				return true
			}
			return false
		}},
		{"collision box coordinate", func(p map[string]any) bool {
			boxes, _ := comp(p, "minecraft:collision_box")["boxes"].([]any)
			if len(boxes) == 0 {
				return false
			}
			b, _ := boxes[0].(map[string]any)
			f, ok := b["maxX"].(float32)
			b["maxX"] = f + 0.5
			return ok
		}},
		{"tool tag added", func(p map[string]any) bool {
			tags, _ := p["blockTags"].([]any)
			if tags == nil {
				return false
			}
			p["blockTags"] = append(tags, "minecraft:is_axe_item_destructible")
			return true
		}},
		{"permutation component value", func(p map[string]any) bool {
			perms, _ := p["permutations"].([]any)
			if len(perms) == 0 {
				return false
			}
			pm, _ := perms[0].(map[string]any)
			tr, _ := pm["components"].(map[string]any)["minecraft:transformation"].(map[string]any)
			v, ok := tr["TX"].(float32)
			if ok {
				tr["TX"] = v + 1
			}
			return ok
		}},
		{"material render method", func(p map[string]any) bool {
			mats, _ := comp(p, "minecraft:material_instances")["materials"].(map[string]any)
			for _, m := range mats {
				if mm, ok := m.(map[string]any); ok {
					mm["render_method"] = "blend"
					return true
				}
			}
			return false
		}},
	}
	for _, tc := range cases {
		found := false
		for i := range blocks {
			got, name := clone(i, tc.mutate)
			if got == nil {
				continue
			}
			found = true
			if d := diff(base, got); !slices.Equal(d, []string{name}) {
				t.Errorf("block %s, %s: changed entries %v, want only that block", name, tc.what, d)
			} else {
				t.Logf("block %s, %s: only its digest changed", name, tc.what)
			}
			break
		}
		if !found {
			t.Errorf("no block to try %q on", tc.what)
		}
	}
	// Order and ids are not content.
	shuf := make([]protocol.BlockEntry, 0, len(blocks))
	for i := len(blocks) - 1; i >= 0; i-- {
		e := protocol.BlockEntry{Name: blocks[i].Name, Properties: cloneMap(blocks[i].Properties)}
		if ps, ok := e.Properties["permutations"].([]any); ok {
			slices.Reverse(ps)
		}
		if ps, ok := e.Properties["permutations"].([]map[string]any); ok {
			slices.Reverse(ps)
		}
		if v, ok := e.Properties["vanilla_block_data"].(map[string]any); ok {
			v["block_id"] = v["block_id"].(int32) + 1 - 1
		}
		shuf = append(shuf, e)
	}
	// Swap the ids of the first two blocks.
	if len(shuf) >= 2 {
		a := shuf[0].Properties["vanilla_block_data"].(map[string]any)
		b := shuf[1].Properties["vanilla_block_data"].(map[string]any)
		a["block_id"], b["block_id"] = b["block_id"], a["block_id"]
	}
	if d := diff(base, blockEntries(t, shuf)); len(d) != 0 {
		t.Errorf("reordering blocks, permutations and ids changed %v", d)
	}

	// Items: one component value, and one added component.
	baseItems := itemEntries(t, items, nil)
	tried := 0
	for _, tc := range []string{"value", "added"} {
		for i, e := range items {
			if !e.ComponentBased {
				continue
			}
			cp := slices.Clone(items)
			cp[i] = protocol.ItemEntry{Name: e.Name, RuntimeID: e.RuntimeID, ComponentBased: true, Version: e.Version, Data: cloneMap(e.Data)}
			comps, _ := cp[i].Data["components"].(map[string]any)
			if comps == nil {
				continue
			}
			if tc == "added" {
				comps["minecraft:digger"] = map[string]any{"use_efficiency": uint8(1)}
			} else if !bumpFirstScalar(comps) {
				continue
			}
			tried++
			if d := diff(baseItems, itemEntries(t, cp, nil)); !slices.Equal(d, []string{"registry/" + e.Name}) {
				t.Errorf("item %s, %s: changed entries %v", e.Name, tc, d)
			} else {
				t.Logf("item %s, component %s: only registry/%s changed", e.Name, tc, e.Name)
			}
			break
		}
	}
	if tried != 2 {
		t.Errorf("tried %d item changes, want 2", tried)
	}
}

// bumpFirstScalar changes the first number or string found under m (by
// sorted key) and reports whether there was one.
func bumpFirstScalar(m map[string]any) bool {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		switch v := m[k].(type) {
		case map[string]any:
			if bumpFirstScalar(v) {
				return true
			}
		case float32:
			m[k] = v + 1
			return true
		case int32:
			m[k] = v + 1
			return true
		case uint8:
			m[k] = v ^ 1
			return true
		case string:
			m[k] = v + "!"
			return true
		}
	}
	return false
}

// cloneMap deep-copies a map of NBT-like values so a test can change a copy
// without touching what the server holds.
func cloneMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = cloneValue(v)
	}
	return out
}

func cloneValue(v any) any {
	switch x := v.(type) {
	case map[string]any:
		return cloneMap(x)
	case []any:
		out := make([]any, len(x))
		for i, e := range x {
			out[i] = cloneValue(e)
		}
		return out
	case []map[string]any:
		out := make([]map[string]any, len(x))
		for i, e := range x {
			out[i] = cloneMap(e)
		}
		return out
	case nil:
		return nil
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice:
		cp := reflect.MakeSlice(rv.Type(), rv.Len(), rv.Len())
		reflect.Copy(cp, rv)
		return cp.Interface()
	case reflect.Map, reflect.Pointer, reflect.Struct:
		panic(fmt.Sprintf("cloneValue: unsupported %T in %s", v, strings.TrimSpace(fmt.Sprint(rv.Kind()))))
	}
	return v
}

// idleListener is a listener that never accepts a connection.
type idleListener struct{ done chan struct{} }

func (l idleListener) Accept() (session.Conn, error) {
	<-l.done
	return nil, errors.New("listener closed")
}
func (idleListener) Disconnect(c session.Conn, reason string) error { return c.Close() }
func (l idleListener) Close() error {
	select {
	case <-l.done:
	default:
		close(l.done)
	}
	return nil
}

// TestCheckEntries: the edits of the game's item and block entries, the ones
// cmd/skygear checks right after Listen, run on every entry without a panic, so
// a bad block or item declaration fails here and not at startup. The server
// has no skygear.New and no client, so it is not a second join.
func TestCheckEntries(t *testing.T) {
	var conf server.Config
	conf.Log = slog.New(slog.DiscardHandler)
	conf.DisableResourceBuilding = true
	conf.Entities = entity.DefaultRegistry.Config().New(skygear.EntityTypes())
	conf.Generator = func(world.Dimension) world.Generator { return world.NopGenerator{} }
	conf.Listeners = []func(server.Config) (server.Listener, error){
		func(server.Config) (server.Listener, error) { return idleListener{done: make(chan struct{})}, nil },
	}
	sghook.Wrap(&conf)
	srv := conf.New()
	core.EditEntries(srv)
	srv.Listen()
	t.Cleanup(func() { _ = srv.Close() })
	if err := core.CheckEntries(); err != nil {
		t.Fatal(err)
	}
	if n := len(world.CustomBlocks()); n == 0 {
		t.Fatal("no custom block is registered: the check covers nothing")
	}
}
