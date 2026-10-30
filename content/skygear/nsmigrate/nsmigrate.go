// Package nsmigrate converts the saves of the server from before its rename
// from DragonFactory to Sky Gear: identifiers, item value and message keys
// move from dragonfactory: (_ and .) to skygear:, the time limit key from df-server:expires to skygear:expires,
// and the df_ achievement and item value keys to sg_.
// Run does it once at startup and leaves a flag file in the data folder; the
// package goes away once every server has run it.
package nsmigrate

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/pelletier/go-toml"
)

// FlagFile is the file in the data folder that marks the conversion done.
const FlagFile = "skygear-namespace.done"

// The old names: identifiers (dragonfactory:), item value keys
// such as the menu item's (dragonfactory_menu), message keys (dragonfactory.)
// and the time limit key.
var oldNames = [][]byte{
	[]byte("dragonfactory:"), []byte("dragonfactory_"), []byte("dragonfactory."),
	[]byte("df-server:expires"),
}

var replacer = strings.NewReplacer(
	"df-server:expires", "skygear:expires",
	"dragonfactory:", "skygear:", "dragonfactory_", "skygear_", "dragonfactory.", "skygear.",
)

// dfKey is the old df_ prefix of achievement, counter and item value keys.
var dfKey = regexp.MustCompile(`(^|[^A-Za-z0-9])df_`)

func hasDFKey(b []byte) bool { return dfKey.Match(b) }

// replaceKey also renames df_ keys. It is used where keys are kept: the
// database and item stack values.
func replaceKey(s string) string {
	s = replaceOld(s)
	if strings.Contains(s, "df_") {
		s = dfKey.ReplaceAllString(s, "${1}sg_")
	}
	return s
}

func replaceOld(s string) string {
	if !hasOld([]byte(s)) {
		return s
	}
	return replacer.Replace(s)
}

func hasOld(b []byte) bool {
	for _, o := range oldNames {
		if bytes.Contains(b, o) {
			return true
		}
	}
	return false
}

// Run converts the database in dataDir, the save worlds under it and the
// lobby world, unless the flag file says it was done. Before it changes
// anything it copies them to <dataDir>.before-skygear (backupDir, the
// platform's backup folder, is left out of the copy).
func Run(log *slog.Logger, dataDir, backupDir string) error {
	flag := filepath.Join(dataDir, FlagFile)
	if _, err := os.Stat(flag); err == nil {
		return nil
	}
	dbPath := filepath.Join(dataDir, "platform.db")
	worlds := worldDirs(dataDir, backupDir)
	if lobby := lobbyFolder(); lobby != "" {
		worlds = append(worlds, lobby)
	}
	_, dbErr := os.Stat(dbPath)
	if errors.Is(dbErr, fs.ErrNotExist) && len(worlds) == 0 {
		// A new server: nothing to convert.
		return os.WriteFile(flag, []byte(time.Now().Format(time.RFC3339)+" new\n"), 0o644)
	}

	copyDir := filepath.Clean(dataDir) + ".before-skygear"
	if _, err := os.Stat(copyDir); errors.Is(err, fs.ErrNotExist) {
		log.Info("namespace conversion: copying the data before converting", "to", copyDir)
		if err := copyTree(dataDir, filepath.Join(copyDir, "data"), backupDir); err != nil {
			return fmt.Errorf("namespace conversion: copy data: %w", err)
		}
		for _, w := range worlds {
			if !inside(w, dataDir) {
				if err := copyTree(w, filepath.Join(copyDir, filepath.Base(w)), ""); err != nil {
					return fmt.Errorf("namespace conversion: copy %s: %w", w, err)
				}
			}
		}
	}

	start := time.Now()
	total := 0
	if dbErr == nil {
		n, err := convertDB(log, dbPath)
		if err != nil {
			return fmt.Errorf("namespace conversion: database: %w", err)
		}
		total += n
	}
	for _, w := range worlds {
		n, err := convertWorld(log, w)
		if err != nil {
			return fmt.Errorf("namespace conversion: world: %w", err)
		}
		total += n
	}
	log.Info("namespace conversion done", "values", total, "worlds", len(worlds), "took", time.Since(start).Round(time.Millisecond))
	return os.WriteFile(flag, []byte(fmt.Sprintf("%s %d values in %d worlds\n", time.Now().Format(time.RFC3339), total, len(worlds))), 0o644)
}

// worldDirs are the worlds under dataDir (the saves are
// <dataDir>/saves/<id>/world): the folders holding a LevelDB db folder.
// skip, the backup folder, is left out.
func worldDirs(dataDir, skip string) []string {
	skipAbs, _ := filepath.Abs(skip)
	var dirs []string
	_ = filepath.WalkDir(dataDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		if abs, _ := filepath.Abs(p); skip != "" && abs == skipAbs {
			return filepath.SkipDir
		}
		if d.Name() == "db" {
			if _, err := os.Stat(filepath.Join(p, "CURRENT")); err == nil {
				dirs = append(dirs, filepath.Dir(p))
			}
			return filepath.SkipDir
		}
		return nil
	})
	return dirs
}

// lobbyFolder is the world folder of config.toml, if it holds a world.
func lobbyFolder() string {
	folder := "world"
	if data, err := os.ReadFile("config.toml"); err == nil {
		var c struct{ World struct{ Folder string } }
		if toml.Unmarshal(data, &c) == nil && c.World.Folder != "" {
			folder = c.World.Folder
		}
	}
	if st, err := os.Stat(filepath.Join(folder, "db")); err == nil && st.IsDir() {
		return folder
	}
	return ""
}

func inside(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	return err == nil && !strings.HasPrefix(rel, "..")
}

// copyTree copies the files of src to dst, leaving out skip and the
// LevelDB lock files.
func copyTree(src, dst, skip string) error {
	skipAbs, _ := filepath.Abs(skip)
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if abs, _ := filepath.Abs(p); skip != "" && abs == skipAbs {
			return filepath.SkipDir
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if d.Name() == "LOCK" {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}
