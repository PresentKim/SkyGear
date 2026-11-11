package migrate

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/df-mc/goleveldb/leveldb"
	"github.com/df-mc/goleveldb/leveldb/opt"
	"github.com/pelletier/go-toml"
	"github.com/presentkim/skygear/common/nbtrewrite"
)

// The conversion of the saves of the server from before its rename from
// DragonFactory to Sky Gear: identifiers, item value and message keys move
// from dragonfactory: (_ and .) to skygear:, the time limit key from
// df-server:expires to skygear:expires, and the df_ achievement and item value
// keys to sg_. Namespace does it once at startup and leaves a flag file in the
// data folder; it goes away once every server has run it.

// NamespaceFlagFile is the file in the data folder that marks the conversion
// done.
const NamespaceFlagFile = "skygear-namespace.done"

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

// worldRenamer renames the NBT of the worlds: tag names and strings lose the
// dragonfactory names, item stack value keys also get df_ -> sg_.
func worldRenamer() renamer {
	return renamer{rep: replaceOld, key: replaceKey, mentions: func(b []byte) bool { return hasOld(b) || hasDFKey(b) }}
}

// dbRenamer renames the database: names and keys alike. The item NBT that
// JSON keeps as base64 (the lost items) is the world's, and only read when it
// holds a dragonfactory name.
func dbRenamer() renamer {
	r := worldRenamer()
	r.rep = replaceKey
	r.warn = true
	r.lost = &renamer{rep: replaceOld, key: replaceKey, mentions: hasOld}
	return r
}

// Namespace converts the database in dataDir, the save worlds under it and the
// lobby world, unless the flag file says it was done. Before it changes
// anything it copies them to <dataDir>.before-skygear (backupDir, the
// platform's backup folder, is left out of the copy).
func Namespace(log *slog.Logger, dataDir, backupDir string) error {
	flag := filepath.Join(dataDir, NamespaceFlagFile)
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

// convertDB rewrites the old names in every text and blob column of the
// platform database: item names in player states, the game's JSON state
// (with the NBT of its lost items), counters and settings.
func convertDB(log *slog.Logger, path string) (int, error) {
	d, err := sql.Open("sqlite", "file:"+path+"?_pragma=busy_timeout(5000)")
	if err != nil {
		return 0, err
	}
	defer d.Close()
	tx, err := d.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	changed := 0
	for _, q := range []string{
		`UPDATE save_state SET key = 'skygear.state' WHERE key = 'dragonfactory.state'`,
	} {
		res, err := tx.Exec(q)
		if err != nil {
			return 0, fmt.Errorf("%s: %w", q, err)
		}
		n, _ := res.RowsAffected()
		changed += int(n)
	}
	n, err := rewriteTables(log, tx, dbRenamer(), func(table) bool { return true }, "", 0)
	if err != nil {
		return 0, err
	}
	return changed + n, tx.Commit()
}

// convertWorld rewrites the old names in an mcdb world folder: block palettes
// of the sub chunks, block entities, entities and any other NBT value that
// holds them. It returns the number of values changed.
func convertWorld(log *slog.Logger, dir string) (int, error) {
	ldb, err := leveldb.OpenFile(filepath.Join(dir, "db"), &opt.Options{BlockSize: 16 * opt.KiB})
	if err != nil {
		return 0, fmt.Errorf("open %s: %w", dir, err)
	}
	defer ldb.Close()

	r := worldRenamer()
	batch := new(leveldb.Batch)
	changed := 0
	it := ldb.NewIterator(nil, nil)
	for it.Next() {
		k, v := it.Key(), it.Value()
		if !r.mentions(v) {
			continue
		}
		var nv []byte
		if nbtrewrite.IsSubChunkKey(k) {
			nv, err = nbtrewrite.SubChunk(v, r.rep)
		} else {
			nv, err = nbtrewrite.NBTKeys(v, r.rep, r.key)
		}
		if err != nil {
			if !hasOld(v) {
				continue // df_ was only bytes of a value that is no NBT
			}
			log.Warn("namespace conversion: value left as it is", "world", dir, "key", fmt.Sprintf("%x", k), "err", err)
			continue
		}
		if !bytes.Equal(nv, v) {
			batch.Put(slices.Clone(k), nv)
			changed++
		}
	}
	it.Release()
	if err := it.Error(); err != nil {
		return 0, err
	}
	if err := ldb.Write(batch, nil); err != nil {
		return 0, err
	}
	return changed, nil
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
