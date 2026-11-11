package hooktest

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/df-mc/dragonfly/server"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

var dropTime atomic.Bool

func init() {
	sghook.OnServerPacket(func(_ *sghook.Conn, pk packet.Packet) bool {
		_, ok := pk.(*packet.SetTime)
		return ok && dropTime.Load()
	})
}

func count[T packet.Packet](pks []packet.Packet) (n int) {
	for _, pk := range pks {
		if _, ok := pk.(T); ok {
			n++
		}
	}
	return
}

func TestJoinCapturesStartData(t *testing.T) {
	c := Join(t, server.Config{}, nil)
	if len(c.GameData().Items) == 0 {
		t.Fatal("no items in the start game data")
	}
	pks := c.Packets()
	if n := count[*packet.ItemRegistry](pks); n != 1 {
		t.Errorf("ItemRegistry: %d", n)
	}
	if n := count[*packet.CraftingData](pks); n != 1 {
		t.Errorf("CraftingData: %d", n)
	}
	if n := count[*packet.CreativeContent](pks); n != 1 {
		t.Errorf("CreativeContent: %d", n)
	}
	if c.Handle() == nil || c.Server() == nil {
		t.Error("no handle or server")
	}
	if c.GameData().WorldName != "Dragonfly Server" {
		t.Errorf("world name %q", c.GameData().WorldName)
	}
}

func TestJoinPacketsPassHooks(t *testing.T) {
	// Without the hook the join sends SetTime; with it none is captured.
	c := Join(t, server.Config{}, nil)
	c.Wait(func(pk packet.Packet) bool { _, ok := pk.(*packet.SetTime); return ok })

	dropTime.Store(true)
	defer dropTime.Store(false)
	c = Join(t, server.Config{}, nil)
	if n := count[*packet.SetTime](c.Packets()); n != 0 {
		t.Fatalf("%d SetTime captured despite the hook", n)
	}
}

func TestCanonMapOrder(t *testing.T) {
	v := func() any { return map[string]any{"b": int32(1), "a": []any{uint8(2), "x"}} }
	want := "{\n  a: [\n    uint8(2)\n    \"x\"\n  ]\n  b: int32(1)\n}"
	for i := range 50 {
		if got := Canon(v()); got != want {
			t.Fatalf("run %d:\n%s", i, got)
		}
	}
	m := map[int]string{}
	for i := range 30 {
		m[i] = strconv.Itoa(i)
	}
	first := Canon(m)
	for range 50 {
		if Canon(m) != first {
			t.Fatal("int map text varies")
		}
	}
}

func TestCanonTypes(t *testing.T) {
	if Canon(int32(1)) == Canon(uint8(1)) {
		t.Fatal("int32(1) equals uint8(1)")
	}
	type s struct {
		B int
		a []byte
		P *int
	}
	x := 3
	got := Canon(s{B: 1, a: []byte{0xab, 1}, P: &x})
	if got != "{\n  B: int(1)\n  P: int(3)\n  a: bytes(ab01)\n}" {
		t.Fatalf("struct:\n%s", got)
	}
}

func TestGoldenUpdateAndMismatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "x.golden")
	t.Setenv("SKYGEAR_GOLDEN", "update")
	Golden(t, path, map[string]string{"b": "other", "a": "text"})
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	const sumText = "982d9e3eb996f559e633f4d194def3761d909f5a3b647d1a851fead67c32c9d1" // sha256("text")
	if !strings.HasPrefix(string(raw), "a\t"+sumText+"\nb\t") {
		t.Fatalf("file:\n%s", raw)
	}

	t.Setenv("SKYGEAR_GOLDEN", "")
	Golden(t, path, map[string]string{"b": "other", "a": "text"}) // passes

	f := &fakeTB{TB: t}
	Golden(f, path, map[string]string{"a": "changed", "c": "new"})
	for _, w := range []string{"changed: a", "added: c", "removed: b"} {
		if !strings.Contains(f.msg, w) {
			t.Errorf("message lacks %q:\n%s", w, f.msg)
		}
	}
	if !f.failed {
		t.Error("mismatch did not fail")
	}
}

// TestGoldenDetectsOneFieldChange is the proof the harness can fail: record
// the start game data entries of a real join, change one field of one entry
// and see Golden report exactly that entry.
func TestGoldenDetectsOneFieldChange(t *testing.T) {
	c := Join(t, server.Config{}, nil)
	items := c.GameData().Items
	if len(items) == 0 {
		t.Fatal("no items")
	}
	entries := func(mutate bool) map[string]string {
		m := map[string]string{}
		for i, it := range items {
			if mutate && i == len(items)/2 {
				it.RuntimeID++
			}
			m["item/"+it.Name] = Canon(it)
		}
		return m
	}
	path := filepath.Join(t.TempDir(), "items.golden")
	t.Setenv("SKYGEAR_GOLDEN", "update")
	Golden(t, path, entries(false))
	t.Setenv("SKYGEAR_GOLDEN", "")
	Golden(t, path, entries(false))

	f := &fakeTB{TB: t}
	Golden(f, path, entries(true))
	name := "item/" + items[len(items)/2].Name
	if !f.failed || !strings.Contains(f.msg, "changed: "+name) || !strings.Contains(f.msg, "1 changed, 0 added, 0 removed") {
		t.Fatalf("one-field change not reported as %s:\n%s", name, f.msg)
	}
}

func TestShutdownDisconnects(t *testing.T) {
	c := Join(t, server.Config{}, nil)
	_ = c.Server().Close()
	c.Wait(func(pk packet.Packet) bool { _, ok := pk.(*packet.Disconnect); return ok })
	if n := count[*packet.Disconnect](c.Packets()); n != 1 {
		t.Fatalf("%d Disconnect packets", n)
	}
}

// fakeTB records failures instead of failing the real test.
type fakeTB struct {
	testing.TB
	failed bool
	msg    string
}

func (f *fakeTB) Helper() {}
func (f *fakeTB) Errorf(format string, args ...any) {
	f.failed = true
	f.msg += strings.TrimSpace(sprintf(format, args...)) + "\n"
}

func sprintf(format string, args ...any) string { return fmt.Sprintf(format, args...) }
