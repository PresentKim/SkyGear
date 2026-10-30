package resourcepack

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

const manifest = `{"format_version": 2,
 "header": {"name": "t", "description": "", "uuid": "6b1f5c0e-3d0a-4c4e-9a43-8f0d2a7c1e01", "version": [0, 0, 1], "min_engine_version": [1, 21, 0]},
 "modules": [{"type": "resources", "uuid": "0c6e2b7a-5f41-4d7b-b0c2-2f3e9a6d4b02", "version": [0, 0, 1]}]}`

func TestVersionFollowsContent(t *testing.T) {
	dir := t.TempDir()
	write := func(name, s string) {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("manifest.json", manifest)
	write("models/entity/a.geo.json", `{"rotation": [0, 0, 0]}`)

	p1, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	p1again, _ := Load(dir)
	if p1.Version() != p1again.Version() {
		t.Fatalf("same content, different versions: %s %s", p1.Version(), p1again.Version())
	}
	if p1.Version() == "0.0.1" {
		t.Fatal("manifest version was not replaced")
	}

	write("models/entity/a.geo.json", `{"rotation": [90, 0, 0]}`)
	p2, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if p1.Version() == p2.Version() {
		t.Fatalf("changed content kept version %s", p1.Version())
	}
	if p1.UUID() != p2.UUID() {
		t.Fatal("UUID must stay stable")
	}
	b, _ := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if string(b) != manifest {
		t.Fatal("source manifest.json was modified")
	}
}

func TestRepositoryPack(t *testing.T) {
	p, err := Load("../../assets/resourcepack/inventoryui")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("inventoryui pack version %s", p.Version())
}

// TestContentsZip: an unencrypted pack carries an empty contents.zip (the
// game then takes an Ore UI page from it); an encrypted one does not.
func TestContentsZip(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, opt := range []Options{{}, {Encrypt: true, Secret: []byte("test secret test secret test sec")}} {
		p, err := LoadWith(dir, opt)
		if err != nil {
			t.Fatal(err)
		}
		b := make([]byte, p.Len())
		if _, err := p.ReadAt(b, 0); err != nil {
			t.Fatal(err)
		}
		z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
		if err != nil {
			t.Fatal(err)
		}
		var found *zip.File
		for _, f := range z.File {
			if f.Name == "contents.zip" {
				found = f
			}
		}
		switch {
		case opt.Encrypt && found != nil:
			t.Error("encrypted pack has a contents.zip")
		case !opt.Encrypt && found == nil:
			t.Error("unencrypted pack has no contents.zip")
		case !opt.Encrypt && found.UncompressedSize64 != 0:
			t.Errorf("contents.zip has %d bytes", found.UncompressedSize64)
		}
	}
}
