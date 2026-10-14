// Package resourcepack loads the hand-made resource packs kept in the
// repository (assets/resourcepack/...).
package resourcepack

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/sandertv/gophertunnel/minecraft/resource"
)

// Load reads the resource pack folder at dir. The header and module versions
// in its manifest are replaced by a version derived from the pack contents.
//
// Clients cache packs by UUID and version and are not sent a pack they
// already have, so a pack that changed without a version bump would keep
// showing the old content. Deriving the version from the content makes every
// change reach clients without editing manifest.json by hand.
func Load(dir string) (*resource.Pack, error) { return LoadWith(dir, Options{}) }

// LoadWith is Load with the pack minified and/or encrypted (Options).
func LoadWith(dir string, opt Options) (*resource.Pack, error) {
	if opt.Encrypt && len(opt.Secret) == 0 {
		return nil, fmt.Errorf("resource pack %s: encryption needs a secret", dir)
	}
	sum, err := hashDir(dir)
	if err != nil {
		return nil, err
	}
	if opt.Encrypt {
		sum = sha256.Sum256(append(sum[:], opt.Secret...))
	}
	version := Version(sum)

	tmp, err := os.MkdirTemp("", "df-resourcepack-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)
	if err := os.CopyFS(tmp, os.DirFS(dir)); err != nil {
		return nil, fmt.Errorf("copy resource pack %s: %w", dir, err)
	}
	if err := setManifestVersion(filepath.Join(tmp, "manifest.json"), version); err != nil {
		return nil, fmt.Errorf("resource pack %s: %w", dir, err)
	}
	if opt.Minify {
		if err := minifyDir(tmp); err != nil {
			return nil, fmt.Errorf("resource pack %s: %w", dir, err)
		}
	}
	var key string
	if opt.Encrypt {
		if key, err = encryptDir(tmp, packUUID(tmp), opt.Secret); err != nil {
			return nil, fmt.Errorf("encrypt resource pack %s: %w", dir, err)
		}
	}
	pack, err := resource.ReadPath(tmp)
	if err != nil {
		return nil, err
	}
	if key != "" {
		pack = pack.WithContentKey(key)
	}
	return pack, nil
}

// packUUID reads the header UUID of the manifest in dir.
func packUUID(dir string) string {
	b, _ := os.ReadFile(filepath.Join(dir, "manifest.json"))
	var m struct {
		Header struct {
			UUID string `json:"uuid"`
		} `json:"header"`
	}
	_ = json.Unmarshal(b, &m)
	return m.Header.UUID
}

// Version maps a content hash to a manifest version [1, a, b].
func Version(sum [32]byte) [3]int {
	return [3]int{1, int(binary.BigEndian.Uint16(sum[0:2])), int(binary.BigEndian.Uint16(sum[2:4]))}
}

func hashDir(dir string) ([32]byte, error) {
	var paths []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			paths = append(paths, p)
		}
		return nil
	})
	if err != nil {
		return [32]byte{}, err
	}
	sort.Strings(paths)
	h := sha256.New()
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return [32]byte{}, err
		}
		rel, _ := filepath.Rel(dir, p)
		fmt.Fprintf(h, "%s\x00%d\x00", filepath.ToSlash(rel), len(b))
		h.Write(b)
	}
	var sum [32]byte
	copy(sum[:], h.Sum(nil))
	return sum, nil
}

func setManifestVersion(path string, v [3]int) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return fmt.Errorf("decode manifest: %w", err)
	}
	header, ok := m["header"].(map[string]any)
	if !ok {
		return fmt.Errorf("manifest has no header")
	}
	header["version"] = v
	if modules, ok := m["modules"].([]any); ok {
		for _, mod := range modules {
			if mm, ok := mod.(map[string]any); ok {
				mm["version"] = v
			}
		}
	}
	if b, err = json.MarshalIndent(m, "", "  "); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
