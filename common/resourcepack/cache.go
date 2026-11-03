package resourcepack

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sandertv/gophertunnel/minecraft/resource"
)

// builderVersion is part of every cache key: raise it when the way a pack is
// built (minify, encryption, zip layout) changes.
var builderVersion = 1

// cacheKey identifies a built pack: the content hash, the options, a hash of
// the secret and the builder version.
func cacheKey(sum [32]byte, opt Options) string {
	h := sha256.New()
	fmt.Fprintf(h, "%x\x00%t\x00%t\x00%d\x00", sum, opt.Minify, opt.Encrypt, builderVersion)
	s := sha256.Sum256(opt.Secret)
	h.Write(s[:])
	return hex.EncodeToString(h.Sum(nil))[:24]
}

// cachePath is the cache file of the pack called name.
func cachePath(dir, name, key string) string {
	var b strings.Builder
	for _, r := range name {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return filepath.Join(dir, b.String()+"-"+key+".mcpack")
}

// readCache loads a cached pack. Any problem (missing, corrupt) is a miss.
func readCache(path string) (*resource.Pack, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	now := time.Now()
	os.Chtimes(path, now, now) // mark as used for pruneCache
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return nil, false
	}
	pack, err := resource.Read(bytes.NewReader(b))
	if err != nil {
		return nil, false
	}
	if key := zr.Comment; key != "" {
		pack = pack.WithContentKey(key)
	}
	return pack, true
}

// writeCache stores zipped under path atomically; failures are ignored since
// the cache is only an optimisation.
func writeCache(path string, zipped []byte) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return
	}
	_, werr := tmp.Write(zipped)
	if cerr := tmp.Close(); werr != nil || cerr != nil || os.Rename(tmp.Name(), path) != nil {
		os.Remove(tmp.Name())
		return
	}
	pruneCache(filepath.Dir(path), path)
}

// zipDir zips dir in memory, in path order with a fixed time so the bytes
// (and so the pack checksum) only depend on the content. Images and
// encrypted files are stored, the rest deflated. comment is the zip comment.
func zipDir(dir string, encrypted bool, comment string) ([]byte, error) {
	var paths []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			paths = append(paths, p)
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	if err := zw.SetComment(comment); err != nil {
		return nil, err
	}
	for _, p := range paths {
		rel, _ := filepath.Rel(dir, p)
		rel = filepath.ToSlash(rel)
		method := zip.Deflate
		if strings.HasSuffix(rel, ".png") || strings.HasSuffix(rel, ".jpg") || (encrypted && rel != "manifest.json") {
			method = zip.Store
		}
		w, err := zw.CreateHeader(&zip.FileHeader{Name: rel, Method: method})
		if err != nil {
			return nil, err
		}
		f, err := os.Open(p)
		if err != nil {
			return nil, err
		}
		_, err = io.Copy(w, f)
		f.Close()
		if err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// pruneAge is how long a cache file may go unused before it is removed.
const pruneAge = 30 * 24 * time.Hour

// pruneCache removes cache files not used for pruneAge and abandoned temp
// files, so keys of old pack versions do not pile up. keep is never removed.
func pruneCache(dir, keep string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	now := time.Now()
	for _, e := range entries {
		name := e.Name()
		p := filepath.Join(dir, name)
		if p == keep || e.IsDir() {
			continue
		}
		stale := strings.HasSuffix(name, ".mcpack")
		tmp := strings.HasPrefix(name, ".tmp-")
		if !stale && !tmp {
			continue
		}
		limit := pruneAge
		if tmp {
			limit = time.Hour
		}
		if info, err := e.Info(); err == nil && now.Sub(info.ModTime()) > limit {
			os.Remove(p)
		}
	}
}
