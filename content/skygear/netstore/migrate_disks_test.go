package netstore

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"path/filepath"
	"slices"
	"strconv"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/presentkim/skygear/common/nbtcanon"
	"github.com/presentkim/skygear/common/storagenet"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/migrate"
	"github.com/presentkim/skygear/platform/db"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

const stateKey = "skygear.state"

func oldKey(t *testing.T, m map[string]any) storagenet.Key {
	t.Helper()
	b, err := nbt.MarshalEncoding(m, nbt.LittleEndian)
	if err != nil {
		t.Fatal(err)
	}
	return storagenet.Key(base64.RawStdEncoding.EncodeToString(nbtcanon.Encode(b)))
}

func named() map[string]any {
	return map[string]any{"display": map[string]any{"Name": "Excalibur", "Lore": []any{"a sword"}}, "Damage": int32(11)}
}

func loadDisks(t *testing.T, d *sql.DB) *core.Session {
	t.Helper()
	var doc []byte
	if err := d.QueryRow(`SELECT value FROM save_state WHERE save_id = 1 AND key = ?`, stateKey).Scan(&doc); err != nil {
		t.Fatal(err)
	}
	return core.New(core.Config{Log: slog.Default(), State: doc})
}

// TestMigrateRewritesDiskKeys: a save of 26.11.8 whose disks hold retired
// items comes out of migrate.Run with the keys of the items that replace them,
// the counts of keys that became one added, every key one the running server
// makes itself, and nothing lost.
func TestMigrateRewritesDiskKeys(t *testing.T) {
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

	swordA := oldKey(t, map[string]any{"Name": "skygear:tool_sword_iron_wood", "Count": byte(1), "Damage": int16(37), "tag": named()})
	swordB := oldKey(t, map[string]any{"Name": "skygear:tool_sword_iron_bone", "Count": byte(1), "Damage": int16(2)})
	gate := oldKey(t, map[string]any{"Name": "skygear:dimension_gate", "Count": byte(1), "Damage": int16(0),
		"Block": map[string]any{"name": "skygear:dimension_gate", "states": map[string]any{}, "version": int32(18153475)}, "tag": named()})
	note := oldKey(t, map[string]any{"Name": "skygear:research_note", "Count": byte(1), "Damage": int16(0), "tag": named()})
	sellBox := oldKey(t, map[string]any{"Name": "skygear:sell_box", "Count": byte(1), "Damage": int16(0),
		"Block": map[string]any{"name": "skygear:sell_box", "states": map[string]any{}, "version": int32(18153475)}})
	ingot := KeyOf(item.NewStack(item.IronIngot{}, 1))
	obsidian := KeyOf(item.NewStack(block.Obsidian{}, 1))
	cobble := KeyOf(item.NewStack(block.Cobblestone{}, 1))
	pick := KeyOf(item.NewStack(item.Pickaxe{Tier: item.ToolTierIron}, 1).WithCustomName("Mine"))
	if slowKeyOf(item.NewStack(item.IronIngot{}, 1)) != ingot || pick == slowKeyOf(item.NewStack(item.Pickaxe{Tier: item.ToolTierIron}, 1)) {
		t.Fatal("the test keys are not of both forms")
	}

	before := map[string]map[storagenet.Key]int64{
		"d1": {swordA: 3, swordB: 4, ingot: 10, cobble: 7, pick: 2, gate: 5, obsidian: 20, note: 6, sellBox: 1},
		"d2": {swordA: 1},
		"d3": {cobble: 9},
	}
	total := func(m map[storagenet.Key]int64) (n int64) {
		for _, v := range m {
			n += v
		}
		return
	}
	doc, err := json.Marshal(map[string]any{"net_disks": before, "net_disk_next": 3})
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`INSERT INTO accounts (id, name, created_at, last_seen) VALUES ('a', 'A', 0, 0)`,
		`INSERT INTO saves (id, owner, slot, name, created_at, last_played) VALUES (1, 'a', 0, 'old', 0, 0)`,
		`INSERT INTO save_state (save_id, key, value) VALUES (1, 'skygear.version', '26.11.8')`,
	} {
		if _, err := d.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := d.Exec(`INSERT INTO save_state (save_id, key, value) VALUES (1, ?, ?)`, stateKey, doc); err != nil {
		t.Fatal(err)
	}
	runs := 0
	run := func() {
		t.Helper()
		runs++
		if err := migrate.Run(slog.Default(), dbPath, filepath.Join(dir, "saves"), filepath.Join(dir, "backup", strconv.Itoa(runs)), "26.11.9"); err != nil {
			t.Fatal(err)
		}
	}
	run()

	s := loadDisks(t, d)
	d1 := Disks(s).Open("d1", 1<<20).Contents()
	if got, want := total(d1), total(before["d1"]); got != want {
		t.Errorf("d1 holds %d items, want %d", got, want)
	}
	paper, chest := KeyOf(item.NewStack(item.Paper{}, 1)), KeyOf(item.NewStack(block.NewChest(), 1))
	for _, c := range []struct {
		name string
		key  storagenet.Key
		n    int64
	}{
		{"iron ingots (swords and ingots)", ingot, 17}, {"paper (research notes)", paper, 6},
		{"obsidian (gates and obsidian)", obsidian, 25}, {"chest (sell box)", chest, 1},
		{"cobblestone", cobble, 7}, {"named pickaxe", pick, 2},
	} {
		if d1[c.key] != c.n {
			t.Errorf("%s = %d, want %d", c.name, d1[c.key], c.n)
		}
	}
	for _, old := range []storagenet.Key{swordA, swordB, gate, note, sellBox} {
		if _, ok := d1[old]; ok {
			t.Errorf("an old key is still on the disk")
		}
	}
	if len(d1) != 6 {
		t.Errorf("d1 has %d keys, want 6: %v", len(d1), d1)
	}
	for _, id := range []string{"d1", "d2", "d3"} {
		for k := range Disks(s).Open(id, 1<<20).Contents() {
			if p := Proto(s, k); p.Empty() || KeyOf(p) != k {
				t.Errorf("%s: key %q does not round trip through Proto and KeyOf", id, k)
			}
		}
	}
	if d2 := Disks(s).Open("d2", 100).Contents(); d2[ingot] != 1 || len(d2) != 1 {
		t.Errorf("d2 = %v", d2)
	}
	if d3 := Disks(s).Open("d3", 100).Contents(); d3[cobble] != 9 || len(d3) != 1 {
		t.Errorf("d3 = %v", d3)
	}
	if _, ok := Proto(s, chest).Item().(block.Chest); !ok {
		t.Errorf("the sell box key reads as %T", Proto(s, chest).Item())
	}

	// A second run over the same document changes nothing.
	var first []byte
	if err := d.QueryRow(`SELECT value FROM save_state WHERE save_id = 1 AND key = ?`, stateKey).Scan(&first); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Exec(`UPDATE save_state SET value = '26.11.8' WHERE key = 'skygear.version'`); err != nil {
		t.Fatal(err)
	}
	run()
	var second []byte
	if err := d.QueryRow(`SELECT value FROM save_state WHERE save_id = 1 AND key = ?`, stateKey).Scan(&second); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(first, second) {
		t.Errorf("a second run changed the document:\n%s\n%s", first, second)
	}
}
