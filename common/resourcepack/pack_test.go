package resourcepack

import (
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
	p, err := Load("../packs/inventoryui")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("inventoryui pack version %s", p.Version())
}
