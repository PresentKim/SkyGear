package nsmigrate

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/df-mc/dragonfly/server/item"
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
	nb, ok := convertBytes(b)
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

	nb, err := rewriteSubChunk(b)
	if err != nil {
		t.Fatal(err)
	}
	if hasOld(nb) || !bytes.Contains(nb, []byte("skygear:conduit_item_1")) {
		t.Fatalf("not converted: %q", nb)
	}
	if !bytes.Equal(nb[:4+len(words)], b[:4+len(words)]) {
		t.Fatal("indices changed")
	}
	if _, err := rewriteSubChunk(nb); err != nil {
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
	nb, ok := convertBytes(in)
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
	if err := Run(slog.Default(), data, filepath.Join(data, "backups")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(data, FlagFile)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(data + ".before-skygear"); err == nil {
		t.Fatal("copied the data of a new server")
	}
}
