package migrate

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/goleveldb/leveldb"
	"github.com/presentkim/skygear/common/nbtrewrite"
	"github.com/presentkim/skygear/common/playerstate"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

func encodeNBT(t *testing.T, v any) []byte {
	t.Helper()
	b, err := nbt.MarshalEncoding(v, nbt.LittleEndian)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestPlayerState: item names and stack values (the time limit key) in a
// player state are renamed, and the state still decodes.
func TestPlayerState(t *testing.T) {
	st := item.NewStack(item.Stick{}, 1).WithValue("df-server:expires", int64(123)).WithValue("dragonfactory_menu", true).WithValue("df_card", "dragonfactory:drawer").WithValue("kept", "dragonfactory:x")
	m := item.WriteNBT(st, true)
	m["Slot"] = byte(0)
	m["Extra"] = map[string]any{"name": "dragonfactory:drawer"}
	b, err := playerstate.Marshal(playerstate.State{Version: playerstate.Version, Pos: []float64{0, 0, 0}, Inventory: []map[string]any{m}})
	if err != nil {
		t.Fatal(err)
	}
	nb, ok := rewriteBytes(b, dbRenamer())
	if !ok || hasOld(nb) || hasDFKey(nb) {
		t.Fatalf("not converted: %v %q", ok, nb)
	}
	s, err := playerstate.Unmarshal(nb)
	if err != nil {
		t.Fatal(err)
	}
	got := item.ReadNBT(s.Inventory[0], nil)
	if v, _ := got.Value("skygear:expires"); v != int64(123) {
		t.Fatalf("expires = %v", v)
	}
	if v, _ := got.Value("skygear_menu"); v != true {
		t.Fatalf("menu = %v", v)
	}
	if v, _ := got.Value("sg_card"); v != "skygear:drawer" {
		t.Fatalf("card = %v", v)
	}
	if v, _ := got.Value("kept"); v != "skygear:x" {
		t.Fatalf("kept = %v", v)
	}
	if name := s.Inventory[0]["Extra"].(map[string]any)["name"]; name != "skygear:drawer" {
		t.Fatalf("block name = %v", name)
	}
}

// TestSubChunk: the palettes of a disk sub chunk are renamed and the indices
// are copied.
func TestSubChunk(t *testing.T) {
	b := []byte{9, 2, 0xfc}
	b = append(b, 1<<1)
	words := make([]byte, 128*4)
	for i := range words {
		words[i] = byte(i)
	}
	b = append(b, words...)
	b = binary.LittleEndian.AppendUint32(b, 2)
	b = append(b, encodeNBT(t, map[string]any{"name": "minecraft:air"})...)
	b = append(b, encodeNBT(t, map[string]any{"name": "dragonfactory:drawer", "states": map[string]any{"dragonfactory:facing": "north"}})...)
	b = append(b, 0)
	b = append(b, encodeNBT(t, map[string]any{"name": "dragonfactory:conduit_item_1"})...)

	nb, err := nbtrewrite.SubChunk(b, replaceOld)
	if err != nil {
		t.Fatal(err)
	}
	if hasOld(nb) || !bytes.Contains(nb, []byte("skygear:conduit_item_1")) {
		t.Fatalf("not converted: %q", nb)
	}
	if !bytes.Equal(nb[:4+len(words)], b[:4+len(words)]) {
		t.Fatal("indices changed")
	}
	if _, err := nbtrewrite.SubChunk(nb, replaceOld); err != nil {
		t.Fatalf("converted sub chunk does not parse: %v", err)
	}
}

// TestJSON: the game state JSON and the item NBT kept in it as base64.
func TestJSON(t *testing.T) {
	itemNBT := encodeNBT(t, map[string]any{"Name": "dragonfactory:fairy_board"})
	in, _ := json.Marshal(map[string]any{
		"machines": []any{map[string]any{"id": "dragonfactory:crusher", "energy": 12345678901}},
		"titles":   []any{"df_tinker"},
		"lost":     []any{map[string]any{"Item": itemNBT}},
	})
	nb, ok := rewriteBytes(in, dbRenamer())
	if !ok || hasOld(nb) {
		t.Fatalf("not converted: %s", nb)
	}
	var out struct {
		Machines []struct {
			ID     string
			Energy json.Number
		}
		Lost   []struct{ Item string }
		Titles []string
	}
	if err := json.Unmarshal(nb, &out); err != nil {
		t.Fatal(err)
	}
	if out.Machines[0].ID != "skygear:crusher" || out.Machines[0].Energy != "12345678901" {
		t.Fatalf("machines = %+v", out.Machines)
	}
	if out.Titles[0] != "sg_tinker" {
		t.Fatalf("titles = %v", out.Titles)
	}
	raw, _ := base64.StdEncoding.DecodeString(out.Lost[0].Item)
	var m map[string]any
	if err := nbt.UnmarshalEncoding(raw, &m, nbt.LittleEndian); err != nil || m["Name"] != "skygear:fairy_board" {
		t.Fatalf("lost item = %v %v", m, err)
	}
}

// TestRunNewServer: with nothing to convert the flag is written and nothing
// is copied.
func TestRunNewServer(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	data := filepath.Join(dir, "data")
	if err := os.MkdirAll(data, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Namespace(slog.Default(), data, filepath.Join(data, "backups")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(data, NamespaceFlagFile)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(data + ".before-skygear"); err == nil {
		t.Fatal("copied the data of a new server")
	}
}

// TestNamespaceWorldStackKeys: in a world value, item stack value keys go
// through replaceKey (df_ -> sg_) while tag names and strings only lose the
// dragonfactory names.
func TestNamespaceWorldStackKeys(t *testing.T) {
	dir := t.TempDir()
	m := item.WriteNBT(item.NewStack(item.Stick{}, 1).WithValue("df_card", "dragonfactory:x"), true)
	m["df_tag"] = int32(1)
	val := encodeNBT(t, m)
	ldb, err := leveldb.OpenFile(filepath.Join(dir, "db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := ldb.Put([]byte("plain"), val, nil); err != nil {
		t.Fatal(err)
	}
	ldb.Close()
	if n, err := convertWorld(slog.Default(), dir); err != nil || n != 1 {
		t.Fatalf("convertWorld = %d, %v", n, err)
	}
	ldb, err = leveldb.OpenFile(filepath.Join(dir, "db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ldb.Close()
	got, err := ldb.Get([]byte("plain"), nil)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := nbt.UnmarshalEncoding(got, &out, nbt.LittleEndian); err != nil {
		t.Fatal(err)
	}
	if _, ok := out["df_tag"]; !ok {
		t.Fatalf("tag name changed: %v", out)
	}
	s := item.ReadNBT(out, nil)
	if v, _ := s.Value("sg_card"); v != "skygear:x" {
		t.Fatalf("card = %v", v)
	}
	if _, ok := s.Value("df_card"); ok {
		t.Fatal("df_card kept")
	}
}

// TestNamespaceLostItemNBT: the item NBT kept as base64 in JSON is renamed
// like the world's: only when it holds a dragonfactory name, and then tag
// names and strings lose that name but keep df_ (stack value keys excepted).
func TestNamespaceLostItemNBT(t *testing.T) {
	only := encodeNBT(t, map[string]any{"df_tag": "df_val"})
	in, _ := json.Marshal(map[string]any{"Item": only})
	if nb, ok := rewriteBytes(in, dbRenamer()); ok {
		t.Fatalf("NBT with only df_ rewritten: %s", nb)
	}

	m := item.WriteNBT(item.NewStack(item.Stick{}, 1).WithValue("df_card", "dragonfactory:x"), true)
	m["df_tag"] = "df_val"
	m["Extra"] = "dragonfactory:y"
	in, _ = json.Marshal(map[string]any{"Item": encodeNBT(t, m)})
	nb, ok := rewriteBytes(in, dbRenamer())
	if !ok {
		t.Fatal("not converted")
	}
	var out struct{ Item []byte }
	if err := json.Unmarshal(nb, &out); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := nbt.UnmarshalEncoding(out.Item, &got, nbt.LittleEndian); err != nil {
		t.Fatal(err)
	}
	if got["df_tag"] != "df_val" || got["Extra"] != "skygear:y" {
		t.Fatalf("lost item = %v", got)
	}
	s := item.ReadNBT(got, nil)
	if v, _ := s.Value("sg_card"); v != "skygear:x" {
		t.Fatalf("card = %v", v)
	}
}

// TestNamespaceWarnsUnrewritten: the conversion reports a value that holds a
// name it could not rewrite (a blob that is no NBT and no UTF-8).
func TestNamespaceWarnsUnrewritten(t *testing.T) {
	path := filepath.Join(t.TempDir(), "platform.db")
	d, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	bad := append([]byte{0xff, 0xfe}, "dragonfactory:x"...)
	if _, err := d.Exec(`CREATE TABLE save_state (key TEXT); CREATE TABLE t (b BLOB)`); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Exec(`INSERT INTO t VALUES (?)`, bad); err != nil {
		t.Fatal(err)
	}
	d.Close()
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))
	if n, err := convertDB(log, path); err != nil || n != 0 {
		t.Fatalf("convertDB = %d, %v", n, err)
	}
	if !strings.Contains(buf.String(), "value left as it is") {
		t.Fatalf("no warning: %q", buf.String())
	}
}
