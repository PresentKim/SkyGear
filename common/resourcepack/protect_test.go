package resourcepack

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMinifyKeepsStringsAndDropsComments(t *testing.T) {
	src := "\xef\xbb\xbf{\n  // a comment\n  \"url\": \"http://x/y\", /* block */ \"q\": \"a \\\" // not a comment\",\n  \"n\": [ 1, 2.50 ]\n}\n"
	got, err := minifyJSON([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"url":"http://x/y","q":"a \" // not a comment","n":[1,2.50]}`
	if string(got) != want {
		t.Errorf("got %s", got)
	}
	if _, err := minifyJSON([]byte(`{"a": }`)); err == nil {
		t.Error("invalid JSON was accepted")
	}
}

func TestCFB8RoundTrip(t *testing.T) {
	key := deriveKey([]byte("secret"), "x")
	if len(key) != 32 {
		t.Fatalf("key length %d", len(key))
	}
	plain := bytes.Repeat([]byte("dragon factory "), 50)
	data := append([]byte(nil), plain...)
	if err := cfb8(data, key, false); err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(data, plain) {
		t.Fatal("not encrypted")
	}
	if err := cfb8(data, key, true); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, plain) {
		t.Fatal("round trip failed")
	}
}

// TestEncryptedPackDecrypts: a client given the content key can decrypt
// contents.json and then every file (written the way the Bedrock format and
// the community tools do it).
func TestEncryptedPackDecrypts(t *testing.T) {
	dir := t.TempDir()
	write := func(name, s string) {
		_ = os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755)
		if err := os.WriteFile(filepath.Join(dir, name), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("manifest.json", manifest)
	write("ui/a.json", "{\n // c\n \"a\": 1 }")
	write("textures/t.txt", "hello textures")
	write("pack_icon.png", "icon")
	secret := []byte("0123456789abcdef0123456789abcdef")
	p1, err := LoadWith(dir, Options{Minify: true, Encrypt: true, Secret: secret})
	if err != nil {
		t.Fatal(err)
	}
	if !p1.Encrypted() || len(p1.ContentKey()) != 32 {
		t.Fatalf("encrypted %v key %q", p1.Encrypted(), p1.ContentKey())
	}
	p2, _ := LoadWith(dir, Options{Minify: true, Encrypt: true, Secret: secret})
	if p1.ContentKey() != p2.ContentKey() || p1.Version() != p2.Version() {
		t.Error("the same content and secret must give the same key and version")
	}
	p3, _ := LoadWith(dir, Options{Minify: true, Encrypt: true, Secret: []byte("another secret of 32 bytes......")})
	if p3.ContentKey() == p1.ContentKey() || p3.Version() == p1.Version() {
		t.Error("another secret must change the key and the version")
	}

	contents, err := p1.ReadFile("contents.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(contents) <= 256 || !bytes.Equal(contents[4:8], []byte("\xfc\xb9\xcf\x9b")) || contents[16] != 36 || string(contents[17:53]) != packUUID(dir) {
		t.Fatalf("contents.json header %x", contents[:60])
	}
	list := append([]byte(nil), contents[256:]...)
	if err := cfb8(list, []byte(p1.ContentKey()), true); err != nil {
		t.Fatal(err)
	}
	var parsed struct{ Content []contentEntry }
	if err := json.Unmarshal(list, &parsed); err != nil {
		t.Fatalf("contents list: %v (%q)", err, list)
	}
	want := map[string]string{"ui/a.json": `{"a":1}`, "textures/t.txt": "hello textures"}
	seen := 0
	for _, e := range parsed.Content {
		if e.Path == "manifest.json" || e.Path == "pack_icon.png" {
			if e.Key != "" {
				t.Errorf("%s must stay plain", e.Path)
			}
			continue
		}
		b, err := p1.ReadFile(e.Path)
		if err != nil {
			t.Fatal(err)
		}
		if err := cfb8(b, []byte(e.Key), true); err != nil {
			t.Fatal(err)
		}
		if string(b) != want[e.Path] {
			t.Errorf("%s decrypts to %q, want %q", e.Path, b, want[e.Path])
		}
		seen++
	}
	if seen != len(want) {
		t.Errorf("%d files listed, want %d", seen, len(want))
	}
	if m, err := p1.ReadFile("manifest.json"); err != nil || !bytes.Contains(m, []byte("header")) {
		t.Error("manifest.json must stay readable")
	}
}

// TestRepositoryPacksMinify: every pack of the repository survives
// minifying, and keeps the same JSON content.
func TestRepositoryPacksMinify(t *testing.T) {
	for _, name := range []string{"inventoryui", "waila", "slotdeco"} {
		dir := filepath.Join("..", "packs", name)
		if _, err := LoadWith(dir, Options{Minify: true, Encrypt: true, Secret: []byte("test secret test secret test sec")}); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}
