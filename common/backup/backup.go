// Package backup copies the platform database and the save worlds so that a
// broken disk or a bad update does not lose them (docs/tech/backup.md).
package backup

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const stamp = "20060102-150405"

// Source is a database that can write a consistent copy of itself.
type Source interface {
	Backup(path string) error
}

// Database writes a copy of d to dir as db-<time>.sqlite and removes all but
// the newest keep copies.
func Database(d Source, dir string, keep int, now time.Time) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "db-"+now.Format(stamp)+".sqlite")
	if err := d.Backup(path); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	prune(dir, "db-", keep)
	return path, nil
}

// Directory writes src as <prefix><time>.tar.gz into dir and removes all but
// the newest keep archives with that prefix. src must not change meanwhile.
func Directory(src, dir, prefix string, keep int, now time.Time) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, prefix+now.Format(stamp)+".tar.gz")
	tmp := path + ".part"
	if err := archive(src, tmp); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	if err := os.Rename(tmp, path); err != nil {
		return "", err
	}
	prune(dir, prefix, keep)
	return path, nil
}

func archive(src, dst string) (err error) {
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}()
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	err = filepath.WalkDir(src, func(path string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil || rel == "." {
			return err
		}
		info, err := e.Info()
		if err != nil {
			return err
		}
		h, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		h.Name = filepath.ToSlash(rel)
		if err := tw.WriteHeader(h); err != nil || !info.Mode().IsRegular() {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		_, err = io.Copy(tw, in)
		return err
	})
	if err != nil {
		return err
	}
	if err := tw.Close(); err != nil {
		return err
	}
	return gz.Close()
}

// prune removes the oldest files of dir that start with prefix until keep
// remain. The names carry the time, so name order is age order.
func prune(dir, prefix string, keep int) {
	if keep <= 0 {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	var names []string
	for _, e := range entries {
		if n := e.Name(); strings.HasPrefix(n, prefix) && !strings.HasSuffix(n, ".part") {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	for len(names) > keep {
		_ = os.Remove(filepath.Join(dir, names[0]))
		names = names[1:]
	}
}

// RunDatabase backs d up every interval until stop is closed.
func RunDatabase(log *slog.Logger, d Source, dir string, interval time.Duration, keep int, stop <-chan struct{}) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case now := <-t.C:
			if p, err := Database(d, dir, keep, now); err != nil {
				log.Error("database backup", "err", err)
			} else {
				log.Info("database backup", "file", p)
			}
		}
	}
}

// WorldPrefix is the archive name prefix of a save's world.
func WorldPrefix(saveID int64) string { return fmt.Sprintf("world-%d-", saveID) }
