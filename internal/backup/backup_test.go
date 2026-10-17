package backup

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type fakeDB struct{}

func (fakeDB) Backup(path string) error { return os.WriteFile(path, []byte("x"), 0o644) }

func TestDatabaseKeepsNewest(t *testing.T) {
	dir := t.TempDir()
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := range 5 {
		if _, err := Database(fakeDB{}, dir, 3, base.Add(time.Duration(i)*time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	m, _ := filepath.Glob(filepath.Join(dir, "db-*"))
	if len(m) != 3 || filepath.Base(m[0]) != "db-20260101-020000.sqlite" {
		t.Fatalf("kept %v", m)
	}
}

func TestDirectoryArchivesFiles(t *testing.T) {
	src, dir := t.TempDir(), t.TempDir()
	os.MkdirAll(filepath.Join(src, "db"), 0o755)
	os.WriteFile(filepath.Join(src, "db", "000001.ldb"), []byte("chunk"), 0o644)
	p, err := Directory(src, dir, WorldPrefix(7), 2, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	f, _ := os.Open(p)
	defer f.Close()
	gz, err := gzipReader(f)
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(gz)
	var names []string
	for {
		h, err := tr.Next()
		if err != nil {
			break
		}
		names = append(names, h.Name)
	}
	if len(names) != 2 || names[1] != "db/000001.ldb" {
		t.Fatalf("archive holds %v", names)
	}
}

func gzipReader(f *os.File) (*gzip.Reader, error) { return gzip.NewReader(f) }
