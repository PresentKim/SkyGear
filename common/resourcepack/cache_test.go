package resourcepack

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"
)

func cacheFS() fstest.MapFS {
	return fstest.MapFS{
		"manifest.json":    {Data: []byte(manifest)},
		"textures/a.png":   {Data: []byte("\x89PNG not really")},
		"ui/main.json":     {Data: []byte("// c\n{\"a\": 1}")},
		"texts/en_US.lang": {Data: []byte("a=b\n")},
	}
}

func cacheFiles(t *testing.T, dir string) []string {
	t.Helper()
	m, _ := filepath.Glob(filepath.Join(dir, "*.mcpack"))
	return m
}

func TestCachedPackSameChecksum(t *testing.T) {
	for _, enc := range []bool{false, true} {
		cdir := t.TempDir()
		opt := Options{Minify: true, Encrypt: enc, Secret: []byte("0123456789abcdef0123456789abcdef"), CacheDir: cdir}
		fresh, err := LoadFS(cacheFS(), "t", opt)
		if err != nil {
			t.Fatal(err)
		}
		if len(cacheFiles(t, cdir)) != 1 {
			t.Fatalf("enc=%v: cache not written", enc)
		}
		hit, err := LoadFS(cacheFS(), "t", opt)
		if err != nil {
			t.Fatal(err)
		}
		if fresh.UUID() != hit.UUID() || fresh.Version() != hit.Version() || fresh.Checksum() != hit.Checksum() || fresh.ContentKey() != hit.ContentKey() {
			t.Fatalf("enc=%v: cached pack differs from fresh", enc)
		}
		nocache := opt
		nocache.CacheDir = ""
		plain, _ := LoadFS(cacheFS(), "t", nocache)
		if plain.Checksum() != fresh.Checksum() {
			t.Fatal("cache changes the pack")
		}
	}
}

func TestCacheKeyChanges(t *testing.T) {
	cdir := t.TempDir()
	sec := []byte("0123456789abcdef0123456789abcdef")
	load := func(fsys fstest.MapFS, opt Options) {
		opt.CacheDir = cdir
		if _, err := LoadFS(fsys, "t", opt); err != nil {
			t.Fatal(err)
		}
	}
	load(cacheFS(), Options{Encrypt: true, Secret: sec})
	n := 1
	check := func(what string) {
		if got := len(cacheFiles(t, cdir)); got != n+1 {
			t.Fatalf("%s: want miss (%d files), got %d", what, n+1, got)
		}
		n++
	}
	load(cacheFS(), Options{Encrypt: true, Secret: sec})
	if len(cacheFiles(t, cdir)) != 1 {
		t.Fatal("same inputs missed")
	}
	load(cacheFS(), Options{Encrypt: true, Secret: sec, Minify: true})
	check("options")
	load(cacheFS(), Options{Encrypt: true, Secret: []byte("ffffffffffffffffffffffffffffffff")})
	check("secret")
	old := builderVersion
	builderVersion = old + 1
	defer func() { builderVersion = old }()
	load(cacheFS(), Options{Encrypt: true, Secret: sec})
	check("builder version")
}

func TestCorruptCacheFallsBack(t *testing.T) {
	cdir := t.TempDir()
	opt := Options{CacheDir: cdir}
	want, err := LoadFS(cacheFS(), "t", opt)
	if err != nil {
		t.Fatal(err)
	}
	f := cacheFiles(t, cdir)[0]
	if err := os.WriteFile(f, []byte("garbage"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadFS(cacheFS(), "t", opt)
	if err != nil || got.Checksum() != want.Checksum() {
		t.Fatalf("fallback failed: %v", err)
	}
	if b, _ := os.ReadFile(f); string(b) == "garbage" {
		t.Fatal("corrupt cache not rewritten")
	}
}

func TestCfb8RoundTripLarge(t *testing.T) {
	key := deriveKey([]byte("s"), "k")
	data := make([]byte, 10000)
	for i := range data {
		data[i] = byte(i * 7)
	}
	orig := append([]byte(nil), data...)
	if err := cfb8(data, key, false); err != nil {
		t.Fatal(err)
	}
	if err := cfb8(data, key, true); err != nil {
		t.Fatal(err)
	}
	if string(data) != string(orig) {
		t.Fatal("round trip differs")
	}
}

func TestPackBuildTiming(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dir := "../../assets/resourcepack/inventoryui"
	if _, err := os.Stat(dir); err != nil {
		t.Skip()
	}
	opt := Options{Minify: true, Encrypt: true, Secret: []byte("0123456789abcdef0123456789abcdef"), CacheDir: t.TempDir()}
	t0 := time.Now()
	if _, err := LoadFS(os.DirFS(dir), "inventoryui", opt); err != nil {
		t.Fatal(err)
	}
	full := time.Since(t0)
	t0 = time.Now()
	if _, err := LoadFS(os.DirFS(dir), "inventoryui", opt); err != nil {
		t.Fatal(err)
	}
	t.Logf("full build %v, cache hit %v", full, time.Since(t0))
}

func TestPruneCacheRemovesOnlyStale(t *testing.T) {
	dir := t.TempDir()
	old := time.Now().Add(-40 * 24 * time.Hour)
	mk := func(name string, mt time.Time) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		os.Chtimes(p, mt, mt)
		return p
	}
	stale := mk("a-1.mcpack", old)
	fresh := mk("a-2.mcpack", time.Now())
	keep := mk("a-3.mcpack", old)
	tmp := mk(".tmp-1", time.Now().Add(-2*time.Hour))
	other := mk("notes.txt", old)
	pruneCache(dir, keep)
	for p, want := range map[string]bool{stale: false, fresh: true, keep: true, tmp: false, other: true} {
		if _, err := os.Stat(p); (err == nil) != want {
			t.Errorf("%s exists=%v, want %v", filepath.Base(p), err == nil, want)
		}
	}
}
