package skygear

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"maps"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/goleveldb/leveldb"
	"github.com/presentkim/skygear/common/nbtrewrite"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/farm"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/migrate"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/platform/db"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

// sellBoxSubChunk is a sub chunk of 26.11.8 whose only block is a sell box:
// a chunk of chests, its palette entries rewritten to the sell box (no
// block states).
func sellBoxSubChunk(t *testing.T) []byte {
	t.Helper()
	c := chunk.New(world.DefaultBlockRegistry, cube.Range{0, 15})
	rid := world.DefaultBlockRegistry.BlockRuntimeID(block.NewChest())
	for x := range uint8(16) {
		for z := range uint8(16) {
			for y := range int16(16) {
				c.SetBlock(x, y, z, 0, rid)
			}
		}
	}
	sub := chunk.EncodeSubChunk(c, chunk.DiskEncoding, 0)
	old, err := nbtrewrite.SubChunkBlocks(sub, func(s string) string { return s }, func(name string) (string, map[string]any, bool) {
		return "skygear:sell_box", map[string]any{}, name == "minecraft:chest"
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(old, []byte("skygear:sell_box")) || bytes.Contains(old, []byte("minecraft:chest")) {
		t.Fatal("the old sub chunk was not built")
	}
	return old
}

// oldSellBoxEntity is the block entity of a sell box as 26.11.8 saved it: the
// machine's NBT, with its owner, and 5 cobblestone in slot 9.
func oldSellBoxEntity(t *testing.T) []byte {
	t.Helper()
	m := machine.Machine{Kind: "crusher"} // any machine: the layout is the same
	data := m.DecodeNBT(map[string]any{"id": "SkyGearMachine"}).(machine.Machine).EncodeNBT()
	data["Items"] = []any{map[string]any{"Name": "minecraft:cobblestone", "Count": byte(5), "Damage": int16(0), "Slot": byte(9)}}
	data["Owner"], data["OwnerName"] = "acct", "Alex"
	data["Progress"] = int32(7)
	data["x"], data["y"], data["z"] = int32(1), int32(2), int32(3)
	b, err := nbt.MarshalEncoding(data, nbt.LittleEndian)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestRetiredSellBox(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "platform.db")
	pdb, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	pdb.Close()
	d, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	for _, q := range []string{
		`INSERT INTO accounts (id, name, created_at, last_seen) VALUES ('a', 'A', 0, 0)`,
		`INSERT INTO saves (id, owner, slot, name, created_at, last_played) VALUES (1, 'a', 0, 'old', 0, 0)`,
		`INSERT INTO save_state (save_id, key, value) VALUES (1, 'skygear.version', '26.11.8')`,
	} {
		if _, err := d.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	held, err := nbt.MarshalEncoding(map[string]any{"Items": []any{map[string]any{"Name": "skygear:sell_box", "Count": byte(1), "Damage": int16(0)}}}, nbt.LittleEndian)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Exec(`INSERT INTO save_players (save_id, account_id, state) VALUES (1, 'a', ?)`, held); err != nil {
		t.Fatal(err)
	}
	world1 := filepath.Join(dir, "saves", "1", "world")
	subKey := []byte{0, 0, 0, 0, 0, 0, 0, 0, '/', 0}
	entityKey := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0x31}
	ldb, err := leveldb.OpenFile(filepath.Join(world1, "db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := ldb.Put(subKey, sellBoxSubChunk(t), nil); err != nil {
		t.Fatal(err)
	}
	if err := ldb.Put(entityKey, oldSellBoxEntity(t), nil); err != nil {
		t.Fatal(err)
	}
	ldb.Close()

	if err := migrate.Run(slog.Default(), dbPath, filepath.Join(dir, "saves"), filepath.Join(dir, "backup"), "26.11.9"); err != nil {
		t.Fatal(err)
	}

	ldb, err = leveldb.OpenFile(filepath.Join(world1, "db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ldb.Close()
	sub, err := ldb.Get(subKey, nil)
	if err != nil {
		t.Fatal(err)
	}
	ent, err := ldb.Get(entityKey, nil)
	if err != nil {
		t.Fatal(err)
	}

	// The block: the real chunk decoder resolves the entry to a chest.
	c, err := chunk.DiskDecode(world.DefaultBlockRegistry, chunk.SerialisedData{SubChunks: [][]byte{sub}}, cube.Range{0, 15})
	if err != nil {
		t.Fatalf("the migrated sub chunk does not load: %v", err)
	}
	b, ok := world.DefaultBlockRegistry.BlockByRuntimeID(c.Block(1, 2, 3, 0))
	if !ok {
		t.Fatal("no block at the sell box")
	}
	ch, ok := b.(block.Chest)
	if !ok {
		t.Fatalf("the sell box loads as %T", b)
	}

	// The block entity: decoded by the chest now at its position, as the
	// world does, it keeps its stacks by slot.
	var data map[string]any
	if err := nbt.UnmarshalEncoding(ent, &data, nbt.LittleEndian); err != nil {
		t.Fatal(err)
	}
	got := ch.DecodeNBT(data).(block.Chest)
	if it, _ := got.Inventory(nil, cube.Pos{}).Item(9); it.Count() != 5 {
		t.Fatalf("slot 9 = %v, want 5 cobblestone", it)
	} else if _, ok := it.Item().(block.Cobblestone); !ok {
		t.Fatalf("slot 9 holds %T", it.Item())
	}

	// What the chest writes back has no machine keys, and loads again.
	enc := got.EncodeNBT()
	for _, k := range []string{"Progress", "Owner", "OwnerName", "Recipe"} {
		if _, ok := enc[k]; ok {
			t.Errorf("the chest saves the stale key %s", k)
		}
	}
	if id, _ := enc["id"].(string); id == "SkyGearMachine" {
		t.Error("the chest saves the machine id")
	}
	again := ch.DecodeNBT(enc).(block.Chest)
	for slot := range 27 {
		a, _ := got.Inventory(nil, cube.Pos{}).Item(slot)
		b, _ := again.Inventory(nil, cube.Pos{}).Item(slot)
		if a.Count() != b.Count() {
			t.Errorf("second load: slot %d = %v, want %v", slot, b, a)
		}
	}

	// The item in a player's state reads as a chest.
	var state []byte
	if err := d.QueryRow(`SELECT state FROM save_players WHERE save_id = 1`).Scan(&state); err != nil {
		t.Fatal(err)
	}
	var held2 struct {
		Items []map[string]any
	}
	if err := nbt.UnmarshalEncoding(state, &held2, nbt.LittleEndian); err != nil {
		t.Fatal(err)
	}
	st := item.ReadNBT(held2.Items[0], nil)
	if st.Empty() {
		t.Fatalf("the sell box item reads as empty: %v", held2.Items[0])
	}
	if _, ok := st.Item().(block.Chest); !ok {
		t.Fatalf("the sell box item reads as %T", st.Item())
	}
}

func TestMachineNBTIgnoresOwner(t *testing.T) {
	old := map[string]any{"id": "SkyGearMachine", "Owner": "acct", "OwnerName": "Alex", "Progress": int32(40),
		"Items": []any{map[string]any{"Name": "minecraft:cobblestone", "Count": byte(7), "Damage": int16(0), "Slot": byte(11)}}}
	m := machine.Machine{Kind: machine.KindCrusher}.DecodeNBT(old).(machine.Machine)
	if m.State.Progress != 40 {
		t.Fatalf("progress = %d", m.State.Progress)
	}
	if in, _ := m.State.Inv().Item(11); in.Count() != 7 {
		t.Fatalf("input = %v", in)
	}
	enc := m.EncodeNBT()
	for _, k := range []string{"Owner", "OwnerName"} {
		if _, ok := enc[k]; ok {
			t.Errorf("EncodeNBT still has %s", k)
		}
	}
	// A blank machine, as an item, still saves only its id and so stacks.
	for _, k := range machine.Order {
		if enc := machine.New(k).EncodeNBT(); len(enc) != 1 {
			t.Errorf("%s: a blank machine saves %v, want only its id", k, enc)
		}
	}
	a, b := machine.New(machine.KindCrusher), machine.Machine{Kind: machine.KindCrusher}.DecodeNBT(map[string]any{"id": "SkyGearMachine", "Owner": "x"})
	if !slices.Equal(mapKeys(a.EncodeNBT()), mapKeys(b.(machine.Machine).EncodeNBT())) {
		t.Error("a machine decoded from old NBT with an Owner differs from a new one")
	}
}

func mapKeys(m map[string]any) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	slices.Sort(ks)
	return ks
}

func TestPatchTargetsExist(t *testing.T) {
	for _, p := range migrate.Patches {
		if p.Version != "26.11.9" {
			continue
		}
		renames := maps.Clone(p.Rename)
		maps.Copy(renames, p.Plain)
		for from, to := range renames {
			if !strings.HasPrefix(to, "minecraft:") && !strings.HasPrefix(to, "skygear:") {
				continue
			}
			_, isBlock := world.DefaultBlockRegistry.DefaultProperties(to)
			_, isItem := world.ItemByName(to, 0)
			if !isBlock && !isItem {
				t.Errorf("26.11.9: %s -> %s: not a registered block or item", from, to)
			}
		}
		for from := range renames {
			if !strings.HasPrefix(from, "skygear:") {
				continue
			}
			if _, ok := world.DefaultBlockRegistry.DefaultProperties(from); ok {
				t.Errorf("26.11.9: %s is renamed but still registered as a block", from)
			}
			if _, ok := world.ItemByName(from, 0); ok {
				t.Errorf("26.11.9: %s is renamed but still registered as an item", from)
			}
		}
		for from, b := range p.Blocks {
			if _, ok := world.DefaultBlockRegistry.DefaultProperties(from); ok {
				t.Errorf("26.11.9: %s is replaced but still registered as a block", from)
			}
			if _, ok := world.DefaultBlockRegistry.DefaultProperties(b.Name); !ok {
				t.Errorf("26.11.9: %s -> %s: not a registered block", from, b.Name)
			}
		}
	}
}

// oldStack is an item stack as 26.11.8 saved it: the NBT of a stack of base
// with the values, name and lore a tinker tool kept, under the old name.
func oldStack(t *testing.T, name string, count int, base world.Item, slot int) map[string]any {
	t.Helper()
	st := item.NewStack(base, count).WithValue("tinker_xp", 90).WithValue("tinker_mods", "reinforced").
		WithCustomName("§r§bOld Name §7Lv.3").WithLore("Level 3", "§bReinforced I")
	m := item.WriteNBT(st, true)
	m["Name"] = name
	m["Damage"] = int16(0)
	m["tag"].(map[string]any)["Damage"] = int32(12) // the tool's durability
	if slot >= 0 {
		m["Slot"] = byte(slot)
	}
	return m
}

// retiredSubChunk is a sub chunk with stone at (1, 2, 3) renamed to name and a
// chest at (4, 5, 6).
func retiredSubChunk(t *testing.T, name string) []byte {
	t.Helper()
	c := chunk.New(world.DefaultBlockRegistry, cube.Range{0, 15})
	c.SetBlock(1, 2, 3, 0, world.DefaultBlockRegistry.BlockRuntimeID(block.Stone{}))
	c.SetBlock(4, 5, 6, 0, world.DefaultBlockRegistry.BlockRuntimeID(block.NewChest()))
	old, err := nbtrewrite.SubChunkBlocks(chunk.EncodeSubChunk(c, chunk.DiskEncoding, 0), func(s string) string { return s }, func(n string) (string, map[string]any, bool) {
		return name, map[string]any{}, n == "minecraft:stone"
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(old, []byte(name)) || bytes.Contains(old, []byte("minecraft:stone")) {
		t.Fatal("the old sub chunk was not built")
	}
	return old
}

// blockEntity is the NBT of a block entity at (x, y, z).
func blockEntity(t *testing.T, id string, x, y, z int32, items ...map[string]any) []byte {
	t.Helper()
	m := map[string]any{"id": id, "x": x, "y": y, "z": z}
	if len(items) > 0 {
		var l []any
		for _, it := range items {
			l = append(l, it)
		}
		m["Items"] = l
	}
	b, err := nbt.MarshalEncoding(m, nbt.LittleEndian)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// newRetiredSave makes a 26.11.8 save with one player; it returns the
// database, its path and the saves folder.
func newRetiredSave(t *testing.T, player []byte) (d *sql.DB, dbPath, saves string) {
	t.Helper()
	dir := t.TempDir()
	dbPath = filepath.Join(dir, "platform.db")
	pdb, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	pdb.Close()
	d, err = sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	for _, q := range []string{
		`INSERT INTO accounts (id, name, created_at, last_seen) VALUES ('a', 'A', 0, 0)`,
		`INSERT INTO saves (id, owner, slot, name, created_at, last_played) VALUES (1, 'a', 0, 'old', 0, 0)`,
		`INSERT INTO save_state (save_id, key, value) VALUES (1, 'skygear.version', '26.11.8')`,
	} {
		if _, err := d.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := d.Exec(`INSERT INTO save_players (save_id, account_id, state) VALUES (1, 'a', ?)`, player); err != nil {
		t.Fatal(err)
	}
	return d, dbPath, filepath.Join(dir, "saves")
}

// readItem reads a saved stack the way the world does.
func readItem(t *testing.T, m map[string]any) item.Stack {
	t.Helper()
	st := item.ReadNBT(m, nil)
	if st.Empty() {
		t.Fatalf("the stack reads as empty: %v", m["Name"])
	}
	return st
}

// checkPlain checks that a stack that was a retired item is now a plain stack
// of the replacement: stack-equal to a new one, with nothing left of the old
// name, lore, durability or values, and also after saving it again.
func checkPlain(t *testing.T, st item.Stack, want world.Item, count int) {
	t.Helper()
	fresh := item.NewStack(want, count)
	if !st.Equal(fresh) || st.Count() != count {
		t.Errorf("stack = %d x %T %q %v %v, want a plain %d x %T", st.Count(), st.Item(), st.CustomName(), st.Lore(), st.Values(), count, want)
	}
	back := item.ReadNBT(item.WriteNBT(st, true), nil)
	if !back.Equal(fresh) {
		t.Errorf("a second round trip gives %v", back)
	}
}

func TestRetiredGateSwordNote(t *testing.T) {
	steel := sgitem.Ingot{Metal: sgitem.MetalSteel}
	bronze := sgitem.Ingot{Metal: sgitem.MetalBronze}
	gate := oldStack(t, "skygear:dimension_gate", 2, block.Obsidian{}, -1)
	gate["Block"] = map[string]any{"name": "skygear:dimension_gate", "states": map[string]any{}, "version": int32(chunk.CurrentBlockVersion)}
	held, err := nbt.MarshalEncoding(map[string]any{"Items": []any{
		gate,
		oldStack(t, "skygear:tool_sword_steel_blaze", 1, item.Paper{}, -1),
		oldStack(t, "skygear:tool_sword_flint_wood", 1, item.Paper{}, -1),
		oldStack(t, "skygear:tool_sword_diamond_bone", 1, item.Paper{}, -1),
		oldStack(t, "skygear:research_note", 3, item.Stick{}, -1),
		map[string]any{"Name": "minecraft:cobblestone", "Count": byte(5), "Damage": int16(0)},
	}}, nbt.LittleEndian)
	if err != nil {
		t.Fatal(err)
	}
	d, dbPath, saves := newRetiredSave(t, held)

	// The world: a gate at (1, 2, 3) with its block entity, a chest at
	// (4, 5, 6) that holds a sword and a note, and a block entity in another
	// chunk that is none of that.
	world1 := filepath.Join(saves, "1", "world")
	subKey := []byte{0, 0, 0, 0, 0, 0, 0, 0, '/', 0}
	entityKey := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0x31}
	farSubKey := []byte{1, 0, 0, 0, 0, 0, 0, 0, '/', 0}
	farEntityKey := []byte{1, 0, 0, 0, 0, 0, 0, 0, 0x31}
	ldb, err := leveldb.OpenFile(filepath.Join(world1, "db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	entities := append(blockEntity(t, "SkyGearMachine", 1, 2, 3),
		blockEntity(t, "Chest", 4, 5, 6,
			oldStack(t, "skygear:tool_sword_bronze_wood", 1, item.Paper{}, 0),
			oldStack(t, "skygear:research_note", 2, item.Stick{}, 4))...)
	for k, v := range map[string][]byte{
		string(subKey):       retiredSubChunk(t, "skygear:dimension_gate"),
		string(entityKey):    entities,
		string(farSubKey):    retiredSubChunk(t, "skygear:dimension_gate"),
		string(farEntityKey): blockEntity(t, "SkyGearMachine", 16+1, 2, 3),
	} {
		if err := ldb.Put([]byte(k), v, nil); err != nil {
			t.Fatal(err)
		}
	}
	// A chunk without a gate keeps its machine block entity.
	plainKey := []byte{2, 0, 0, 0, 0, 0, 0, 0, 0x31}
	plain := blockEntity(t, "SkyGearMachine", 32+1, 2, 3)
	if err := ldb.Put(plainKey, plain, nil); err != nil {
		t.Fatal(err)
	}
	ldb.Close()

	if err := migrate.Run(slog.Default(), dbPath, saves, filepath.Join(filepath.Dir(saves), "backup"), "26.11.9"); err != nil {
		t.Fatal(err)
	}

	ldb, err = leveldb.OpenFile(filepath.Join(world1, "db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ldb.Close()

	// The block: the real decoder loads the chunk, with obsidian where the
	// gate was and the chest where it was.
	for _, k := range [][]byte{subKey, farSubKey} {
		sub, err := ldb.Get(k, nil)
		if err != nil {
			t.Fatal(err)
		}
		c, err := chunk.DiskDecode(world.DefaultBlockRegistry, chunk.SerialisedData{SubChunks: [][]byte{sub}}, cube.Range{0, 15})
		if err != nil {
			t.Fatalf("the migrated sub chunk does not load: %v", err)
		}
		if b, _ := world.DefaultBlockRegistry.BlockByRuntimeID(c.Block(1, 2, 3, 0)); !reflect.DeepEqual(b, block.Obsidian{}) {
			t.Errorf("the gate loads as %T", b)
		}
		if b, _ := world.DefaultBlockRegistry.BlockByRuntimeID(c.Block(4, 5, 6, 0)); !isChest(b) {
			t.Errorf("the chest loads as %T", b)
		}
	}

	// The block entities: the world decodes each by the block at its
	// position and logs an error if that block has no NBT, so none may be
	// left at the obsidian, and the chest's stays.
	ent, err := ldb.Get(entityKey, nil)
	if err != nil {
		t.Fatal(err)
	}
	rd := bytes.NewReader(ent)
	dec := nbt.NewDecoderWithEncoding(rd, nbt.LittleEndian)
	var left []map[string]any
	for rd.Len() > 0 {
		var m map[string]any
		if err := dec.Decode(&m); err != nil {
			t.Fatal(err)
		}
		left = append(left, m)
	}
	if len(left) != 1 || left[0]["id"] != "Chest" || left[0]["x"] != int32(4) {
		t.Fatalf("block entities left in the chunk: %v", left)
	}
	if _, ok := (world.Block)(block.Obsidian{}).(world.NBTer); ok {
		t.Fatal("obsidian has NBT: a block entity at it would be fine")
	}
	got := block.NewChest().DecodeNBT(left[0]).(block.Chest)
	sword, _ := got.Inventory(nil, cube.Pos{}).Item(0)
	checkPlain(t, sword, bronze, 1)
	note, _ := got.Inventory(nil, cube.Pos{}).Item(4)
	checkPlain(t, note, item.Paper{}, 2)

	// Another chunk's lone gate entity goes, with its key; a chunk without
	// a gate keeps its entity as it was.
	if _, err := ldb.Get(farEntityKey, nil); err == nil {
		t.Error("the block entity of the gate in the other chunk is kept")
	}
	if v, err := ldb.Get(plainKey, nil); err != nil || !bytes.Equal(v, plain) {
		t.Errorf("a chunk without a gate changed: %v", err)
	}

	// The items of the player.
	var state []byte
	if err := d.QueryRow(`SELECT state FROM save_players WHERE save_id = 1`).Scan(&state); err != nil {
		t.Fatal(err)
	}
	var items struct{ Items []map[string]any }
	if err := nbt.UnmarshalEncoding(state, &items, nbt.LittleEndian); err != nil {
		t.Fatal(err)
	}
	if len(items.Items) != 6 {
		t.Fatalf("%d stacks, want 6", len(items.Items))
	}
	checkPlain(t, readItem(t, items.Items[0]), block.Obsidian{}, 2)
	checkPlain(t, readItem(t, items.Items[1]), steel, 1)
	checkPlain(t, readItem(t, items.Items[2]), item.Flint{}, 1)
	checkPlain(t, readItem(t, items.Items[3]), item.Diamond{}, 1)
	checkPlain(t, readItem(t, items.Items[4]), item.Paper{}, 3)
	if st := readItem(t, items.Items[5]); st.Count() != 5 || !reflect.DeepEqual(st.Item(), block.Cobblestone{}) {
		t.Errorf("an unrelated stack changed: %v", st)
	}
}

func isChest(b world.Block) bool {
	_, ok := b.(block.Chest)
	return ok
}

// TestRetiredStacksArePlainEverywhere: a retired item with a custom name,
// lore, durability and values becomes a plain stack of its replacement in an
// item entity, in the lost-and-found box and inside another stack, and a
// stack that is not retired keeps all of its own.
func TestRetiredStacksArePlainEverywhere(t *testing.T) {
	player, err := nbt.MarshalEncoding(map[string]any{"Items": []any{}}, nbt.LittleEndian)
	if err != nil {
		t.Fatal(err)
	}
	d, dbPath, saves := newRetiredSave(t, player)

	// A container stack (one that holds other stacks in its tag, as a
	// bundle would) and a bystander with a name that must stay.
	inner := oldStack(t, "skygear:tool_sword_diamond_blaze", 1, item.Paper{}, 3)
	outer := item.WriteNBT(item.NewStack(item.Stick{}, 7).WithCustomName("bundle"), true)
	outer["tag"].(map[string]any)["Items"] = []any{inner}
	retiredHolder := oldStack(t, "skygear:research_note", 2, item.Stick{}, -1)
	retiredHolder["tag"].(map[string]any)["Items"] = []any{oldStack(t, "skygear:research_note", 1, item.Stick{}, 0)}
	bystander := item.WriteNBT(item.NewStack(item.Stick{}, 4).WithCustomName("keep me").WithLore("lore").WithValue("v", 1), true)
	entity := map[string]any{"identifier": "minecraft:item", "Item": oldStack(t, "skygear:tool_sword_iron_bone", 1, item.Paper{}, -1)}
	entityB, err := nbt.MarshalEncoding(entity, nbt.LittleEndian)
	if err != nil {
		t.Fatal(err)
	}
	holder, err := nbt.MarshalEncoding(map[string]any{"Items": []any{outer, bystander, retiredHolder}}, nbt.LittleEndian)
	if err != nil {
		t.Fatal(err)
	}
	world1 := filepath.Join(saves, "1", "world")
	ldb, err := leveldb.OpenFile(filepath.Join(world1, "db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range map[string][]byte{"entity": entityB, "holder": holder} {
		if err := ldb.Put([]byte(k), v, nil); err != nil {
			t.Fatal(err)
		}
	}
	ldb.Close()

	// A lost-and-found box: the JSON holds the stack NBT as base64.
	lostItem := oldStack(t, "skygear:dimension_gate", 1, block.Obsidian{}, -1)
	lostB, err := nbt.MarshalEncoding(lostItem, nbt.LittleEndian)
	if err != nil {
		t.Fatal(err)
	}
	state := `{"lost":[{"item":"` + base64.StdEncoding.EncodeToString(lostB) + `","count":5}]}`
	if _, err := d.Exec(`INSERT INTO save_state (save_id, key, value) VALUES (1, 'skygear.state', ?)`, state); err != nil {
		t.Fatal(err)
	}

	if err := migrate.Run(slog.Default(), dbPath, saves, filepath.Join(filepath.Dir(saves), "backup"), "26.11.9"); err != nil {
		t.Fatal(err)
	}

	ldb, err = leveldb.OpenFile(filepath.Join(world1, "db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ldb.Close()
	decode := func(key string) map[string]any {
		v, err := ldb.Get([]byte(key), nil)
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		if err := nbt.UnmarshalEncoding(v, &m, nbt.LittleEndian); err != nil {
			t.Fatal(err)
		}
		return m
	}
	checkPlain(t, item.MapNBT(decode("entity"), "Item"), item.IronIngot{}, 1)

	h := decode("holder")["Items"].([]any)
	o := h[0].(map[string]any)
	if st := readItem(t, o); st.CustomName() != "bundle" || st.Count() != 7 {
		t.Errorf("the container changed: %v", st)
	}
	in := o["tag"].(map[string]any)["Items"].([]any)[0].(map[string]any)
	checkPlain(t, readItem(t, in), item.Diamond{}, 1)
	if in["Slot"] != byte(3) {
		t.Errorf("the slot of a nested stack = %v", in["Slot"])
	}
	// A retired stack that holds stacks becomes plain and holds nothing.
	rh := h[2].(map[string]any)
	checkPlain(t, readItem(t, rh), item.Paper{}, 2)
	if _, ok := rh["tag"]; ok || rh["Damage"] != int16(0) {
		t.Errorf("the retired container keeps %v", rh)
	}
	if st := readItem(t, h[1].(map[string]any)); !st.Equal(item.NewStack(item.Stick{}, 4).WithCustomName("keep me").WithLore("lore").WithValue("v", 1)) {
		t.Errorf("a stack that is not retired changed: %v", st)
	}

	var saved []byte
	if err := d.QueryRow(`SELECT value FROM save_state WHERE save_id = 1 AND key = 'skygear.state'`).Scan(&saved); err != nil {
		t.Fatal(err)
	}
	var lost struct {
		Lost []struct {
			Item  []byte `json:"item"`
			Count int    `json:"count"`
		} `json:"lost"`
	}
	if err := json.Unmarshal(saved, &lost); err != nil || len(lost.Lost) != 1 || lost.Lost[0].Count != 5 {
		t.Fatalf("the lost box = %s (%v)", saved, err)
	}
	checkPlain(t, core.DecodeItem(lost.Lost[0].Item), block.Obsidian{}, 1)
}

// bloomsSubChunk is a sub chunk of 26.11.8 with an essence bloom at
// (1, 2, 3), an essence bud at (2, 2, 3), an old infested leaves block at
// (3, 2, 3) and a coal bloom at (4, 2, 3), which stays: the current blocks of
// other kinds, their palette states rewritten to the old ones.
func bloomsSubChunk(t *testing.T) []byte {
	t.Helper()
	c := chunk.New(world.DefaultBlockRegistry, cube.Range{0, 15})
	rid := func(b world.Block) uint32 { return world.DefaultBlockRegistry.BlockRuntimeID(b) }
	c.SetBlock(1, 2, 3, 0, rid(farm.MagicPlant{Kind: "gem_lapis", Bloom: true}))
	c.SetBlock(2, 2, 3, 0, rid(farm.MagicPlant{Kind: "gem_quartz"}))
	c.SetBlock(3, 2, 3, 0, rid(block.Stone{}))
	c.SetBlock(4, 2, 3, 0, rid(farm.MagicPlant{Kind: "gem_coal", Bloom: true}))
	// A second kind that collapses into the same palette entry as (1, 2, 3).
	c.SetBlock(5, 2, 3, 0, rid(farm.MagicPlant{Kind: "gem_quartz", Bloom: true}))
	rep := func(s string) string {
		switch s {
		case "gem_lapis_1":
			return "ess_zombie_1"
		case "gem_quartz_1":
			return "ess_blaze_1"
		case "gem_quartz_0":
			return "ess_slime_0"
		}
		return s
	}
	old, err := nbtrewrite.SubChunkBlocks(chunk.EncodeSubChunk(c, chunk.DiskEncoding, 0), rep, func(name string) (string, map[string]any, bool) {
		return "skygear:infested_leaves", map[string]any{}, name == "minecraft:stone"
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"ess_zombie_1", "ess_slime_0", "skygear:infested_leaves", "gem_coal_1"} {
		if !bytes.Contains(old, []byte(s)) {
			t.Fatalf("the old sub chunk has no %s", s)
		}
	}
	return old
}

// seedStack is a saved stack of the seed with the id, with a name and lore.
func seedStack(t *testing.T, id string, count int, slot int) map[string]any {
	t.Helper()
	m := oldStack(t, "skygear:seed_"+id, count, farm.PlantSeed{Seed: "gem_redstone"}, slot)
	return m
}

// TestRetiredBloomsAndLeaves: a save of 26.11.8 with essence blooms and old
// infested leaves comes out of the patch loading as redstone blooms (same
// stage), full silkworm leaves that are not persistent, and the other blooms
// and the plant records as they were; the seeds are redstone bloom seeds
// wherever they were, a coverage that includes everything the game once did
// at run time for a player's inventory.
func TestRetiredBloomsAndLeaves(t *testing.T) {
	held, err := nbt.MarshalEncoding(map[string]any{"Items": []any{
		seedStack(t, "ess_slime", 3, 0),
		seedStack(t, "ess_blaze", 2, 1),
		map[string]any{"Name": "skygear:seed_gem_redstone", "Count": byte(4), "Damage": int16(0), "Slot": byte(2)},
		map[string]any{"Name": "skygear:seed_gem_coal", "Count": byte(1), "Damage": int16(0), "Slot": byte(3)},
		oldStack(t, "skygear:infested_leaves", 6, block.Obsidian{}, 4),
	}}, nbt.LittleEndian)
	if err != nil {
		t.Fatal(err)
	}
	d, dbPath, saves := newRetiredSave(t, held)

	world1 := filepath.Join(saves, "1", "world")
	subKey := []byte{0, 0, 0, 0, 0, 0, 0, 0, '/', 0}
	chestKey := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0x31}
	ldb, err := leveldb.OpenFile(filepath.Join(world1, "db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := ldb.Put(subKey, bloomsSubChunk(t), nil); err != nil {
		t.Fatal(err)
	}
	if err := ldb.Put(chestKey, blockEntity(t, "Chest", 4, 5, 6, seedStack(t, "ess_zombie", 5, 7)), nil); err != nil {
		t.Fatal(err)
	}
	ldb.Close()

	at := func(x, y, z int) string { return farm.CropKey(cube.Pos{x, y, z}) }
	state := `{"plants":{"` + at(1, 2, 3) + `":{"kind":"ess_blaze","ready_at":7},"` + at(2, 2, 3) + `":{"kind":"ess_slime","ready_at":9},"` + at(4, 2, 3) + `":{"kind":"gem_coal","ready_at":11}}}`
	if _, err := d.Exec(`INSERT INTO save_state (save_id, key, value) VALUES (1, 'skygear.state', ?)`, state); err != nil {
		t.Fatal(err)
	}

	if err := migrate.Run(slog.Default(), dbPath, saves, filepath.Join(filepath.Dir(saves), "backup"), "26.11.9"); err != nil {
		t.Fatal(err)
	}

	ldb, err = leveldb.OpenFile(filepath.Join(world1, "db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ldb.Close()
	sub, err := ldb.Get(subKey, nil)
	if err != nil {
		t.Fatal(err)
	}
	c, err := chunk.DiskDecode(world.DefaultBlockRegistry, chunk.SerialisedData{SubChunks: [][]byte{sub}}, cube.Range{0, 15})
	if err != nil {
		t.Fatalf("the migrated sub chunk does not load: %v", err)
	}
	blockAt := func(x, y, z uint8) world.Block {
		b, ok := world.DefaultBlockRegistry.BlockByRuntimeID(c.Block(x, int16(y), z, 0))
		if !ok {
			t.Fatalf("no block at %d %d %d", x, y, z)
		}
		return b
	}
	if b := blockAt(1, 2, 3); b != (farm.MagicPlant{Kind: "gem_redstone", Bloom: true}) {
		t.Errorf("the essence bloom loads as %#v", b)
	}
	if b := blockAt(5, 2, 3); b != (farm.MagicPlant{Kind: "gem_redstone", Bloom: true}) {
		t.Errorf("the second essence bloom loads as %#v", b)
	}
	// The next save of the chunk (palette with a doubled entry) reads again.
	if again, err := chunk.DiskDecode(world.DefaultBlockRegistry, chunk.SerialisedData{SubChunks: [][]byte{chunk.EncodeSubChunk(c, chunk.DiskEncoding, 0)}}, cube.Range{0, 15}); err != nil || again.Block(5, 2, 3, 0) != c.Block(5, 2, 3, 0) {
		t.Errorf("the chunk does not survive a save: %v", err)
	}
	if b := blockAt(2, 2, 3); b != (farm.MagicPlant{Kind: "gem_redstone"}) {
		t.Errorf("the essence bud loads as %#v", b)
	}
	if b := blockAt(3, 2, 3); b != (farm.SilkwormLeaves{Cocoon: farm.CocoonMax}) {
		t.Errorf("the infested leaves load as %#v", b)
	}
	if b := blockAt(4, 2, 3); b != (farm.MagicPlant{Kind: "gem_coal", Bloom: true}) {
		t.Errorf("the coal bloom changed to %#v", b)
	}

	// The plant records of the save.
	var doc []byte
	if err := d.QueryRow(`SELECT value FROM save_state WHERE save_id = 1 AND key = 'skygear.state'`).Scan(&doc); err != nil {
		t.Fatal(err)
	}
	var st struct {
		Plants map[string]*farm.PlantRecord `json:"plants"`
	}
	if err := json.Unmarshal(doc, &st); err != nil {
		t.Fatal(err)
	}
	for _, w := range []struct {
		pos     cube.Pos
		kind    string
		readyAt int64
	}{{cube.Pos{1, 2, 3}, "gem_redstone", 7}, {cube.Pos{2, 2, 3}, "gem_redstone", 9}, {cube.Pos{4, 2, 3}, "gem_coal", 11}} {
		r := st.Plants[farm.CropKey(w.pos)]
		if r == nil || r.Kind != w.kind || int64(r.ReadyAt) != w.readyAt {
			t.Errorf("record at %v = %+v, want %s ready at %d", w.pos, r, w.kind, w.readyAt)
		}
	}

	// The seeds: in a player's state, a chest and the leaves item.
	var pstate []byte
	if err := d.QueryRow(`SELECT state FROM save_players WHERE save_id = 1`).Scan(&pstate); err != nil {
		t.Fatal(err)
	}
	var items struct{ Items []map[string]any }
	if err := nbt.UnmarshalEncoding(pstate, &items, nbt.LittleEndian); err != nil {
		t.Fatal(err)
	}
	redstone := farm.PlantSeed{Seed: "gem_redstone"}
	checkPlain(t, readItem(t, items.Items[0]), redstone, 3)
	checkPlain(t, readItem(t, items.Items[1]), redstone, 2)
	checkPlain(t, readItem(t, items.Items[2]), redstone, 4)
	if st := readItem(t, items.Items[3]); !st.Equal(item.NewStack(farm.PlantSeed{Seed: "gem_coal"}, 1)) {
		t.Errorf("a coal bloom seed changed: %v", st)
	}
	checkPlain(t, readItem(t, items.Items[4]), farm.SilkwormLeaves{Cocoon: farm.CocoonMax, Persistent: true}, 6)
	if items.Items[0]["Slot"] != byte(0) || items.Items[1]["Slot"] != byte(1) {
		t.Error("a slot moved")
	}

	ent, err := ldb.Get(chestKey, nil)
	if err != nil {
		t.Fatal(err)
	}
	var data map[string]any
	if err := nbt.UnmarshalEncoding(ent, &data, nbt.LittleEndian); err != nil {
		t.Fatal(err)
	}
	got := block.NewChest().DecodeNBT(data).(block.Chest)
	seeds, _ := got.Inventory(nil, cube.Pos{}).Item(7)
	checkPlain(t, seeds, redstone, 5)
}
