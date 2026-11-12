package migrate

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"testing"

	"github.com/presentkim/skygear/common/nbtcanon"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

// diskKey is the key of a disk for the item NBT m, as netstore makes it.
func diskKey(t *testing.T, m map[string]any) string {
	t.Helper()
	b, err := nbt.MarshalEncoding(m, nbt.LittleEndian)
	if err != nil {
		t.Fatal(err)
	}
	return base64.RawStdEncoding.EncodeToString(nbtcanon.Encode(b))
}

func diskStack(name string, extra map[string]any) map[string]any {
	m := map[string]any{"Name": name, "Count": byte(1), "Damage": int16(0)}
	for k, v := range extra {
		m[k] = v
	}
	return m
}

func stateDoc(t *testing.T, disks map[string]map[string]any) []byte {
	t.Helper()
	b, err := json.Marshal(map[string]any{"net_disks": disks, "net_disk_next": 3, "other": []int{1, 2}})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func diskOf(t *testing.T, doc []byte, id string) map[string]int64 {
	t.Helper()
	var st struct {
		Disks map[string]map[string]int64 `json:"net_disks"`
	}
	if err := json.Unmarshal(doc, &st); err != nil {
		t.Fatal(err)
	}
	return st.Disks[id]
}

// TestDiskKeysCollapseAndAdd: the keys of a disk follow the renames, and keys
// that become one add their counts, whether both were renamed or one was there
// already.
func TestDiskKeysCollapseAndAdd(t *testing.T) {
	ps := []Patch{{Version: "26.11.1", Name: "p", Plain: map[string]string{
		"a:sword_x": "a:ingot", "a:sword_y": "a:ingot", "a:note": "a:paper",
	}}}
	named := map[string]any{"tag": map[string]any{"display": map[string]any{"Name": "Excalibur", "Lore": []any{"old"}}}}
	swordX := diskKey(t, diskStack("a:sword_x", named))
	swordY := diskKey(t, diskStack("a:sword_y", map[string]any{"Damage": int16(9)}))
	ingot := diskKey(t, diskStack("a:ingot", nil))
	other := diskKey(t, diskStack("a:other", named))
	note := diskKey(t, diskStack("a:note", named))
	doc := stateDoc(t, map[string]map[string]any{
		"d1": {swordX: 3, swordY: 4, ingot: 10, other: 7, note: 6},
		"d2": {other: 2},
	})
	got, changed := rewriteBytes(doc, newRenamer(ps))
	if !changed {
		t.Fatal("the disk keys were not rewritten")
	}
	d1 := diskOf(t, got, "d1")
	want := map[string]int64{ingot: 17, other: 7, diskKey(t, diskStack("a:paper", nil)): 6}
	if fmt.Sprint(d1) != fmt.Sprint(want) {
		t.Errorf("d1 = %v, want %v", d1, want)
	}
	if d2 := diskOf(t, got, "d2"); fmt.Sprint(d2) != fmt.Sprint(map[string]int64{other: 2}) {
		t.Errorf("d2 = %v", d2)
	}
	if !bytes.Contains(got, []byte(`"net_disk_next":3`)) || !bytes.Contains(got, []byte(`"other":[1,2]`)) {
		t.Errorf("the rest of the document changed: %s", got)
	}
	// A second pass changes nothing.
	if again, ch := rewriteBytes(got, newRenamer(ps)); ch || !bytes.Equal(again, got) {
		t.Error("the second pass changed the document")
	}
}

// TestDiskKeysOverflowKeepsTheDisk: counts that cannot be added leave the
// disk as it was: nothing is lost.
func TestDiskKeysOverflowKeepsTheDisk(t *testing.T) {
	ps := []Patch{{Version: "26.11.1", Name: "p", Rename: map[string]string{"a:x": "a:y"}}}
	x, y := diskKey(t, diskStack("a:x", nil)), diskKey(t, diskStack("a:y", nil))
	doc := stateDoc(t, map[string]map[string]any{"d1": {x: math.MaxInt64, y: 5}})
	got, _ := rewriteBytes(doc, newRenamer(ps))
	d1 := diskOf(t, got, "d1")
	if d1[x] != math.MaxInt64 || d1[y] != 5 || len(d1) != 2 {
		t.Errorf("d1 = %v: the disk should stay as it was", d1)
	}
}

// TestDiskKeysNoOpWithoutNames: a document with no patched name in any key
// comes back byte for byte, whatever its formatting.
func TestDiskKeysNoOpWithoutNames(t *testing.T) {
	ps := []Patch{{Version: "26.11.1", Name: "p", Plain: map[string]string{"a:gone": "a:here"}}}
	k := diskKey(t, diskStack("a:kept", map[string]any{"tag": map[string]any{"display": map[string]any{"Name": "n"}}}))
	doc := []byte(`{ "net_disks" : {"d1": {"` + k + `": 4, "!!notbase64": 2}}, "z": 1,  "a": 2 }`)
	got, changed := rewriteBytes(doc, newRenamer(ps))
	if changed || !bytes.Equal(got, doc) {
		t.Errorf("the document changed: %s", got)
	}
}

// TestDiskKeysAreNoOpForReleasedPatches: the released patches leave a disk
// without their names byte for byte.
func TestDiskKeysAreNoOpForReleasedPatches(t *testing.T) {
	var released []Patch
	for _, p := range Patches {
		if Compare(p.Version, "26.11.9") < 0 {
			released = append(released, p)
		}
	}
	k := diskKey(t, diskStack("minecraft:stone", map[string]any{"tag": map[string]any{"display": map[string]any{"Name": "n"}}}))
	doc := []byte(`{"net_disks":{"d1":{"` + k + `":4}},"net_disk_next":1}`)
	got, changed := rewriteBytes(doc, newRenamer(released))
	if changed || !bytes.Equal(got, doc) {
		t.Errorf("the document changed: %s", got)
	}
}
