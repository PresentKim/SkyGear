package instance

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopySaveCopiesTheWorldFolder(t *testing.T) {
	f := setup(t)
	for _, id := range []string{"owner", "admin"} {
		if _, err := f.d.TouchAccount(id, "", id); err != nil {
			t.Fatal(err)
		}
	}
	src, err := f.d.CreateSave("owner", "test", 1, "island")
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(f.m.worldDir(src.ID), "db", "CURRENT")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("MANIFEST-1"), 0o644); err != nil {
		t.Fatal(err)
	}
	cp, err := f.m.CopySave(src.ID, "admin", 2, "copy")
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(f.m.worldDir(cp.ID), "db", "CURRENT"))
	if err != nil || string(got) != "MANIFEST-1" {
		t.Fatalf("copied world file = %q, %v", got, err)
	}
	if _, err := os.Stat(file); err != nil {
		t.Errorf("source world file gone: %v", err)
	}
}
