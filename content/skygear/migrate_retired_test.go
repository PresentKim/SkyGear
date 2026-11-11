package skygear

import (
	"bytes"
	"database/sql"
	"log/slog"
	"path/filepath"
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
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/migrate"
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
		for from, to := range p.Rename {
			if !strings.HasPrefix(to, "minecraft:") && !strings.HasPrefix(to, "skygear:") {
				continue
			}
			_, isBlock := world.DefaultBlockRegistry.DefaultProperties(to)
			_, isItem := world.ItemByName(to, 0)
			if !isBlock && !isItem {
				t.Errorf("26.11.9: %s -> %s: not a registered block or item", from, to)
			}
		}
		for from := range p.Rename {
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
			if _, ok := world.DefaultBlockRegistry.DefaultProperties(b.Name); !ok {
				t.Errorf("26.11.9: %s -> %s: not a registered block", from, b.Name)
			}
		}
	}
}
