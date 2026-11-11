package migrate

import (
	"bytes"
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"

	"github.com/df-mc/goleveldb/leveldb"
	"github.com/presentkim/skygear/common/nbtrewrite"
	"github.com/presentkim/skygear/platform/db"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

var testPatch = Patch{Version: "26.11.3", Name: "test", Rename: map[string]string{
	"skygear:old_block":    "skygear:new_block",
	"place_old_block":      "place_new_block",
	"sg_place_old_block_1": "sg_place_new_block_1",
}}

func withPatches(t *testing.T, ps ...Patch) {
	old := Patches
	Patches = ps
	t.Cleanup(func() { Patches = old })
}

func TestPatchesAreOrdered(t *testing.T) {
	for i, p := range Patches {
		if err := valid(p.Version); err != nil {
			t.Fatal(err)
		}
		if i > 0 && Compare(Patches[i-1].Version, p.Version) >= 0 {
			t.Errorf("patch %s is not newer than %s", p.Version, Patches[i-1].Version)
		}
	}
}

func TestCompare(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want int
	}{{"26.11.3", "26.11.2", 1}, {"26.9.1", "26.10.1", -1}, {"", "26.1.1", -1}, {"27.1.1", "27.1.1", 0}} {
		if got := Compare(c.a, c.b); got != c.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestStampTakesTheNewestPatch(t *testing.T) {
	withPatches(t, testPatch)
	if got := Stamp("26.1.1"); got != "26.11.3" {
		t.Errorf("Stamp of an older release = %q, want 26.11.3", got)
	}
	if got := Stamp("99.1.1"); got != "99.1.1" {
		t.Errorf("Stamp of a newer release = %q", got)
	}
}

func TestRenameKeepsTheMeta(t *testing.T) {
	r := newRenamer([]Patch{testPatch})
	for in, want := range map[string]string{
		"skygear:old_block":    "skygear:new_block",
		"skygear:old_block:0":  "skygear:new_block:0",
		"place_old_block":      "place_new_block",
		"skygear:old_blocks":   "skygear:old_blocks",
		"old_block":            "old_block",
		"a skygear:old_block.": "a skygear:old_block.",
	} {
		if got := r.rep(in); got != want {
			t.Errorf("rep(%q) = %q, want %q", in, got, want)
		}
	}
}

func itemNBT(t *testing.T, name string) []byte {
	b, err := nbt.MarshalEncoding(map[string]any{"Items": []any{map[string]any{"Name": name, "Count": byte(1)}}}, nbt.LittleEndian)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func itemName(t *testing.T, b []byte) string {
	var v struct {
		Items []struct {
			Name  string
			Count byte
		}
	}
	if err := nbt.UnmarshalEncoding(b, &v, nbt.LittleEndian); err != nil {
		t.Fatal(err)
	}
	return v.Items[0].Name
}

func newDB(t *testing.T, dir string) (string, *sql.DB) {
	path := filepath.Join(dir, "platform.db")
	pdb, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	pdb.Close()
	d, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	return path, d
}

func exec(t *testing.T, d *sql.DB, q string, args ...any) {
	if _, err := d.Exec(q, args...); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
}

func TestRunMigratesOldSaves(t *testing.T) {
	withPatches(t, testPatch)
	dir := t.TempDir()
	dbPath, d := newDB(t, dir)
	exec(t, d, `INSERT INTO accounts (id, name, created_at, last_seen) VALUES ('a', 'A', 0, 0)`)
	exec(t, d, `INSERT INTO saves (id, owner, slot, name, created_at, last_played) VALUES (1, 'a', 0, 'old', 0, 0), (2, 'a', 1, 'new', 0, 0)`)
	exec(t, d, `INSERT INTO counters (account_id, key, value) VALUES ('a', 'place_old_block', 3)`)
	exec(t, d, `INSERT INTO achievements (account_id, id, earned_at) VALUES ('a', 'sg_place_old_block_1', 0)`)
	exec(t, d, `INSERT INTO save_state (save_id, key, value) VALUES (1, 'skygear.state', '{"keys":["skygear:old_block:0"],"research":["old_block"]}'),
		(2, 'skygear.state', '{"keys":["skygear:old_block:0"]}'), (2, 'skygear.version', '26.11.3')`)
	exec(t, d, `INSERT INTO save_players (save_id, account_id, state) VALUES (1, 'a', ?)`, itemNBT(t, "skygear:old_block"))
	world := filepath.Join(dir, "saves", "1", "world")
	ldb, err := leveldb.OpenFile(filepath.Join(world, "db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := ldb.Put([]byte("chest"), itemNBT(t, "skygear:old_block"), nil); err != nil {
		t.Fatal(err)
	}
	ldb.Close()

	backups := filepath.Join(dir, "backup")
	if err := Run(slog.Default(), dbPath, filepath.Join(dir, "saves"), backups, "26.11.2"); err != nil {
		t.Fatal(err)
	}

	var counter, achievement, state1, state2, version1 string
	var player []byte
	for _, q := range []struct {
		sql string
		to  any
	}{
		{`SELECT key FROM counters`, &counter},
		{`SELECT id FROM achievements`, &achievement},
		{`SELECT value FROM save_state WHERE save_id = 1 AND key = 'skygear.state'`, &state1},
		{`SELECT value FROM save_state WHERE save_id = 2 AND key = 'skygear.state'`, &state2},
		{`SELECT value FROM save_state WHERE save_id = 1 AND key = 'skygear.version'`, &version1},
		{`SELECT state FROM save_players`, &player},
	} {
		if err := d.QueryRow(q.sql).Scan(q.to); err != nil {
			t.Fatalf("%s: %v", q.sql, err)
		}
	}
	if counter != "place_new_block" || achievement != "sg_place_new_block_1" {
		t.Errorf("account data: counter %q, achievement %q", counter, achievement)
	}
	if state1 != `{"keys":["skygear:new_block:0"],"research":["old_block"]}` {
		t.Errorf("old save state %s", state1)
	}
	if state2 != `{"keys":["skygear:old_block:0"]}` {
		t.Errorf("a save at the newest version changed: %s", state2)
	}
	if version1 != "26.11.3" {
		t.Errorf("old save version %q, want 26.11.3", version1)
	}
	if got := itemName(t, player); got != "skygear:new_block" {
		t.Errorf("player item %q", got)
	}
	ldb, err = leveldb.OpenFile(filepath.Join(world, "db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	v, err := ldb.Get([]byte("chest"), nil)
	ldb.Close()
	if err != nil {
		t.Fatal(err)
	}
	if got := itemName(t, v); got != "skygear:new_block" {
		t.Errorf("world item %q", got)
	}
	files, _ := os.ReadDir(backups)
	if len(files) != 2 {
		t.Errorf("%d backups, want the database and the world", len(files))
	}

	if err := Run(slog.Default(), dbPath, filepath.Join(dir, "saves"), backups, "26.11.2"); err != nil {
		t.Fatal(err)
	}
	if files, _ := os.ReadDir(backups); len(files) != 2 {
		t.Errorf("a second run made backups: %d files", len(files))
	}
}

func TestRunOnANewServerMakesNoBackup(t *testing.T) {
	withPatches(t, testPatch)
	dir := t.TempDir()
	dbPath, d := newDB(t, dir)
	backups := filepath.Join(dir, "backup")
	if err := Run(slog.Default(), dbPath, filepath.Join(dir, "saves"), backups, "26.11.2"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(backups); !os.IsNotExist(err) {
		t.Errorf("backup folder: %v", err)
	}
	var v string
	if err := d.QueryRow(`SELECT version FROM data_versions WHERE scope = 'accounts'`).Scan(&v); err != nil || v != "26.11.3" {
		t.Errorf("account data version %q, %v", v, err)
	}
}

var blocksPatch = Patch{Version: "26.11.9", Name: "blocks", Rename: map[string]string{"a:both": "a:renamed"},
	Blocks: map[string]Block{
		"a:old":  {Name: "a:new", States: map[string]any{"s": "10"}},
		"a:both": {Name: "a:blocknew", States: map[string]any{"s": "1"}},
	}}

func palette(t *testing.T, names ...string) []byte {
	words := make([]byte, 128*4)
	b := append([]byte{9, 2, 0}, 1<<1)
	for range 2 { // two layers with the same palette
		b = append(b[:len(b):len(b)], words...)
		b = append(b, byte(len(names)), 0, 0, 0)
		for _, n := range names {
			e, err := nbt.MarshalEncoding(map[string]any{"name": n, "states": map[string]any{"x": "y"}, "version": int32(7)}, nbt.LittleEndian)
			if err != nil {
				t.Fatal(err)
			}
			b = append(b, e...)
		}
		b = append(b, 1<<1)
	}
	return b[:len(b)-1]
}

var subKey = []byte{1, 0, 0, 0, 2, 0, 0, 0, '/', 0}

func entries(t *testing.T, b []byte) (names []string, states []any) {
	b = b[3:]
	for range 2 {
		b = b[1+128*4+4:]
		for range 3 {
			var m map[string]any
			if err := nbt.NewDecoderWithEncoding(bytes.NewReader(b), nbt.LittleEndian).Decode(&m); err != nil {
				t.Fatal(err)
			}
			n, _ := nbt.MarshalEncoding(m, nbt.LittleEndian)
			b = b[len(n):]
			names = append(names, m["name"].(string))
			states = append(states, m["states"])
		}
	}
	return
}

func blocksSave(t *testing.T, dir string, id int64, d *sql.DB, version string) string {
	exec(t, d, `INSERT INTO saves (id, owner, slot, name, created_at, last_played) VALUES (?, 'a', ?, 'x', 0, 0)`, id, id)
	if version != "" {
		exec(t, d, `INSERT INTO save_state (save_id, key, value) VALUES (?, 'skygear.version', ?)`, id, version)
	}
	world := filepath.Join(dir, "saves", strconv.FormatInt(id, 10), "world")
	ldb, err := leveldb.OpenFile(filepath.Join(world, "db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ldb.Close()
	if err := ldb.Put(subKey, palette(t, "a:old", "a:both", "a:keep"), nil); err != nil {
		t.Fatal(err)
	}
	// Not a sub chunk: the same name in other NBT is only renamed.
	if err := ldb.Put([]byte("chest"), itemNBT(t, "a:both"), nil); err != nil {
		t.Fatal(err)
	}
	return world
}

func readSub(t *testing.T, world string) []byte {
	ldb, err := leveldb.OpenFile(filepath.Join(world, "db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ldb.Close()
	v, err := ldb.Get(subKey, nil)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestRunAppliesBlocks(t *testing.T) {
	withPatches(t, blocksPatch)
	dir := t.TempDir()
	dbPath, d := newDB(t, dir)
	exec(t, d, `INSERT INTO accounts (id, name, created_at, last_seen) VALUES ('a', 'A', 0, 0)`)
	oldWorld := blocksSave(t, dir, 1, d, "26.11.8")
	newWorld := blocksSave(t, dir, 2, d, "26.11.9")
	untouched := readSub(t, newWorld)
	backups := filepath.Join(dir, "backup")
	for range 2 { // the second run is a no-op
		if err := Run(slog.Default(), dbPath, filepath.Join(dir, "saves"), backups, "26.11.8"); err != nil {
			t.Fatal(err)
		}
	}
	names, states := entries(t, readSub(t, oldWorld))
	want := []string{"a:new", "a:blocknew", "a:keep"}
	for i, n := range names {
		if n != want[i%3] {
			t.Errorf("entry %d is %q, want %q", i, n, want[i%3])
		}
		wantStates := map[string]any{"x": "y"}
		if i%3 == 0 {
			wantStates = map[string]any{"s": "10"}
		} else if i%3 == 1 {
			wantStates = map[string]any{"s": "1"}
		}
		if !reflect.DeepEqual(states[i], wantStates) {
			t.Errorf("entry %d states %v, want %v", i, states[i], wantStates)
		}
	}
	if !bytes.Equal(readSub(t, newWorld), untouched) {
		t.Error("a save at the patch version changed")
	}
	if files, _ := os.ReadDir(backups); len(files) != 2 {
		t.Errorf("%d backups, want the database and the old world", len(files))
	}
}

// TestBlocksBeforeRename: a name in both gets Blocks in the palette and
// Rename in item NBT.
func TestBlocksBeforeRename(t *testing.T) {
	r := newRenamer([]Patch{blocksPatch})
	if n, s, ok := r.blocks("a:both"); !ok || n != "a:blocknew" || s["s"] != "1" {
		t.Errorf("blocks(a:both) = %q, %v, %v", n, s, ok)
	}
	if _, _, ok := r.blocks("a:keep"); ok {
		t.Error("a name not in Blocks was replaced")
	}
	if got := r.rep("a:both"); got != "a:renamed" {
		t.Errorf("rep(a:both) = %q", got)
	}
	if b, ok := blocksPatch.block("a:old"); !ok || b.Name != "a:new" {
		t.Errorf("block(a:old) = %v, %v", b, ok)
	}
	if _, ok := blocksPatch.block("minecraft:stone"); ok {
		t.Error("block(minecraft:stone) found")
	}
	world := t.TempDir()
	ldb, err := leveldb.OpenFile(filepath.Join(world, "db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	ldb.Put([]byte("chest"), itemNBT(t, "a:both"), nil)
	ldb.Close()
	if _, err := rewriteWorld(slog.Default(), world, r); err != nil {
		t.Fatal(err)
	}
	ldb, _ = leveldb.OpenFile(filepath.Join(world, "db"), nil)
	defer ldb.Close()
	v, _ := ldb.Get([]byte("chest"), nil)
	if got := itemName(t, v); got != "a:renamed" {
		t.Errorf("item NBT %q", got)
	}
}

// TestBlocksAreNoOpForReleasedPatches: no released patch has Blocks, so
// the renamer has no hook and sub chunks are rewritten as before.
func TestBlocksAreNoOpForReleasedPatches(t *testing.T) {
	for _, p := range Patches {
		if len(p.Blocks) > 0 && Compare(p.Version, "26.11.9") < 0 {
			t.Errorf("released patch %s has Blocks", p.Version)
		}
	}
	var released []Patch
	for _, p := range Patches {
		if Compare(p.Version, "26.11.9") < 0 {
			released = append(released, p)
		}
	}
	r := newRenamer(released)
	if r.blocks != nil {
		t.Fatal("released patches have a block hook")
	}
	sub := palette(t, "skygear:storage_controller", "a:keep", "minecraft:air")
	want, err := nbtrewrite.SubChunk(sub, r.rep)
	if err != nil {
		t.Fatal(err)
	}
	got, err := nbtrewrite.SubChunkBlocks(sub, r.rep, r.blocks)
	if err != nil || !bytes.Equal(got, want) {
		t.Errorf("sub chunk differs: %v", err)
	}
}
