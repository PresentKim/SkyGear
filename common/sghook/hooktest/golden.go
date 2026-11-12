package hooktest

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const maxReported = 20

// Golden compares entries (name to canonical text, see Canon) with the file
// at path. The file holds one line per entry, "name", a tab, the sha256 of
// the text in hex, sorted by name. A mismatch fails t and names up to 20
// added, removed and changed entries. Names hold no tab or newline.
//
// SKYGEAR_GOLDEN=update rewrites the file from entries. SKYGEAR_GOLDEN_DUMP=<dir>
// also writes each text to <dir>/<file base>/<name>.txt (name path-escaped),
// so the output of two trees can be compared with diff -r.
func Golden(t testing.TB, path string, entries map[string]string) {
	t.Helper()
	got := make(map[string]string, len(entries))
	for name, text := range entries {
		if strings.ContainsAny(name, "\t\n") {
			t.Errorf("golden %s: entry name %q holds a tab or newline", path, name)
			return
		}
		sum := sha256.Sum256([]byte(text))
		got[name] = hex.EncodeToString(sum[:])
	}
	if dir := os.Getenv("SKYGEAR_GOLDEN_DUMP"); dir != "" {
		if err := dump(filepath.Join(dir, filepath.Base(path)), entries); err != nil {
			t.Errorf("golden %s: dump: %v", path, err)
			return
		}
	}
	if os.Getenv("SKYGEAR_GOLDEN") == "update" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Errorf("golden %s: %v", path, err)
			return
		}
		if err := os.WriteFile(path, []byte(format(got)), 0o644); err != nil {
			t.Errorf("golden %s: %v", path, err)
		}
		return
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("golden %s: %v\nrecord it with SKYGEAR_GOLDEN=update", path, err)
		return
	}
	want := map[string]string{}
	for i, line := range strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n") {
		if line == "" {
			continue
		}
		name, sum, ok := strings.Cut(line, "\t")
		if !ok {
			t.Errorf("golden %s: line %d has no tab", path, i+1)
			return
		}
		want[name] = sum
	}
	var added, removed, changed []string
	for n, s := range got {
		w, ok := want[n]
		switch {
		case !ok:
			added = append(added, n)
		case w != s:
			changed = append(changed, n)
		}
	}
	for n := range want {
		if _, ok := got[n]; !ok {
			removed = append(removed, n)
		}
	}
	if len(added)+len(removed)+len(changed) == 0 {
		return
	}
	var b strings.Builder
	fmt.Fprintf(&b, "golden %s differs: %d changed, %d added, %d removed", path, len(changed), len(added), len(removed))
	list(&b, "changed", changed)
	list(&b, "added", added)
	list(&b, "removed", removed)
	b.WriteString("\nif intended, re-record with SKYGEAR_GOLDEN=update; to see the text, dump both trees with SKYGEAR_GOLDEN_DUMP=<dir> and diff -r")
	t.Errorf("%s", b.String())
}

func list(b *strings.Builder, kind string, names []string) {
	sort.Strings(names)
	for i, n := range names {
		if i == maxReported {
			fmt.Fprintf(b, "\n  %s: ... and %d more", kind, len(names)-maxReported)
			break
		}
		fmt.Fprintf(b, "\n  %s: %s", kind, n)
	}
}

func format(sums map[string]string) string {
	names := make([]string, 0, len(sums))
	for n := range sums {
		names = append(names, n)
	}
	sort.Strings(names)
	var b strings.Builder
	for _, n := range names {
		b.WriteString(n + "\t" + sums[n] + "\n")
	}
	return b.String()
}

func dump(dir string, entries map[string]string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for n, text := range entries {
		if err := os.WriteFile(filepath.Join(dir, url.PathEscape(n)+".txt"), []byte(text), 0o644); err != nil {
			return err
		}
	}
	return nil
}
