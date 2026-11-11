package migrate

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/df-mc/goleveldb/leveldb"
	"github.com/df-mc/goleveldb/leveldb/opt"
	"github.com/presentkim/skygear/common/backup"
	"github.com/presentkim/skygear/common/nbtrewrite"

	_ "modernc.org/sqlite"
)

// VersionKey is the save_state key holding the release a save's data is in.
const VersionKey = "skygear.version"

// accountsScope is the data_versions scope of the data outside saves.
const accountsScope = "accounts"

// keepBackups is how many copies of each before-migration backup are kept.
const keepBackups = 3

type saveVersion struct {
	id      int64
	version string
}

// Run brings the database at dbPath (its schema already migrated) and the
// save worlds under savesDir (<savesDir>/<id>/world) up to release, copying
// what it changes to backupDir first. The worlds must not be open.
func Run(log *slog.Logger, dbPath, savesDir, backupDir, release string) error {
	for _, p := range Patches {
		if err := valid(p.Version); err != nil {
			return fmt.Errorf("migrate: patch %s: %w", p.Name, err)
		}
	}
	d, err := sql.Open("sqlite", "file:"+dbPath+"?_pragma=busy_timeout(5000)")
	if err != nil {
		return err
	}
	defer d.Close()
	stamp := Stamp(release)
	accounts, err := scopeVersion(d, accountsScope)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	saves, err := saveVersions(d)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	if !needed(accounts, saves) {
		return nil
	}
	now := time.Now()
	if used, err := hasData(d); err != nil {
		return fmt.Errorf("migrate: %w", err)
	} else if used {
		if err := backupDB(d, backupDir, now); err != nil {
			return fmt.Errorf("migrate: back up the database: %w", err)
		}
	}
	if ps := pending(accounts); len(ps) > 0 {
		n, err := rewriteRows(log, d, newRenamer(ps), "", 0)
		if err != nil {
			return fmt.Errorf("migrate: account data: %w", err)
		}
		if _, err := d.Exec(`INSERT INTO data_versions (scope, version) VALUES (?, ?)
			ON CONFLICT(scope) DO UPDATE SET version = excluded.version`, accountsScope, stamp); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
		log.Info("migrated the account data", "to", stamp, "patches", names(ps), "values", n)
	}
	for _, s := range saves {
		ps := pending(s.version)
		if len(ps) == 0 {
			continue
		}
		if err := migrateSave(log, d, s.id, filepath.Join(savesDir, strconv.FormatInt(s.id, 10), "world"), backupDir, ps, stamp, now); err != nil {
			return fmt.Errorf("migrate: save %d: %w", s.id, err)
		}
	}
	return nil
}

func needed(accounts string, saves []saveVersion) bool {
	if len(pending(accounts)) > 0 {
		return true
	}
	for _, s := range saves {
		if len(pending(s.version)) > 0 {
			return true
		}
	}
	return false
}

func migrateSave(log *slog.Logger, d *sql.DB, id int64, world, backupDir string, ps []Patch, stamp string, now time.Time) error {
	r := newRenamer(ps)
	changed := 0
	if st, err := os.Stat(filepath.Join(world, "db")); err == nil && st.IsDir() {
		if _, err := backup.Directory(world, backupDir, fmt.Sprintf("migrate-save%d-", id), keepBackups, now); err != nil {
			return fmt.Errorf("back up the world: %w", err)
		}
		n, err := rewriteWorld(log, world, r)
		if err != nil {
			return err
		}
		changed += n
	} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	n, err := rewriteRows(log, d, r, "save_id", id)
	if err != nil {
		return err
	}
	changed += n
	if err := SetSaveVersion(d, id, stamp); err != nil {
		return err
	}
	log.Info("migrated a save", "save", id, "to", stamp, "patches", names(ps), "values", changed)
	return nil
}

// SetSaveVersion records version as the release of the data of save id.
func SetSaveVersion(d interface {
	Exec(string, ...any) (sql.Result, error)
}, id int64, version string) error {
	_, err := d.Exec(`INSERT INTO save_state (save_id, key, value) VALUES (?, ?, ?)
		ON CONFLICT(save_id, key) DO UPDATE SET value = excluded.value`, id, VersionKey, []byte(version))
	return err
}

func scopeVersion(d *sql.DB, scope string) (string, error) {
	var v string
	err := d.QueryRow(`SELECT version FROM data_versions WHERE scope = ?`, scope).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return v, err
}

func saveVersions(d *sql.DB) ([]saveVersion, error) {
	rows, err := d.Query(`SELECT s.id, COALESCE(v.value, '') FROM saves s
		LEFT JOIN save_state v ON v.save_id = s.id AND v.key = ? ORDER BY s.id`, VersionKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []saveVersion
	for rows.Next() {
		var s saveVersion
		var v []byte
		if err := rows.Scan(&s.id, &v); err != nil {
			return nil, err
		}
		s.version = string(v)
		out = append(out, s)
	}
	return out, rows.Err()
}

func hasData(d *sql.DB) (bool, error) {
	var n int
	err := d.QueryRow(`SELECT (SELECT COUNT(*) FROM accounts) + (SELECT COUNT(*) FROM saves)`).Scan(&n)
	return n > 0, err
}

func backupDB(d *sql.DB, dir string, now time.Time) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, "migrate-db-"+now.Format("20060102-150405")+".sqlite")
	_, err := d.Exec(`VACUUM INTO ?`, path)
	return err
}

func names(ps []Patch) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.Version + " " + p.Name
	}
	return out
}

// rewriteWorld rewrites the values of an mcdb world folder that hold a
// renamed string: block palettes of the sub chunks, block entities,
// entities and other NBT. It returns the number of values changed.
func rewriteWorld(log *slog.Logger, dir string, r renamer) (int, error) {
	ldb, err := leveldb.OpenFile(filepath.Join(dir, "db"), &opt.Options{BlockSize: 16 * opt.KiB})
	if err != nil {
		return 0, fmt.Errorf("open %s: %w", dir, err)
	}
	defer ldb.Close()
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
			nv, err = nbtrewrite.NBT(v, r.rep)
		}
		if err != nil {
			log.Warn("migrate: world value left as it is", "world", dir, "key", fmt.Sprintf("%x", k), "err", err)
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
	return changed, ldb.Write(batch, nil)
}

// rewriteRows rewrites the text and blob columns of the database: with
// scopeCol "", the tables without a save_id column (data_versions aside);
// otherwise the rows of the tables with a save_id column whose save_id is
// id.
func rewriteRows(log *slog.Logger, d *sql.DB, r renamer, scopeCol string, id int64) (int, error) {
	tx, err := d.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	changed, err := rewriteTables(log, tx, r, func(t table) bool {
		return t.name != "data_versions" && slices.Contains(t.cols, "save_id") == (scopeCol != "")
	}, scopeCol, id)
	if err != nil {
		return 0, err
	}
	return changed, tx.Commit()
}

// rewriteTables rewrites the tables of tx that pick selects (scopeCol and id
// as in rewriteRows) and returns the number of values changed.
func rewriteTables(log *slog.Logger, tx *sql.Tx, r renamer, pick func(table) bool, scopeCol string, id int64) (int, error) {
	tables, err := tableColumns(tx)
	if err != nil {
		return 0, err
	}
	changed := 0
	for _, t := range tables {
		if !pick(t) {
			continue
		}
		n, err := rewriteTable(log, tx, t, r, scopeCol, id)
		if err != nil {
			return 0, fmt.Errorf("table %s: %w", t.name, err)
		}
		changed += n
	}
	return changed, nil
}

type table struct {
	name string
	cols []string
}

func tableColumns(tx *sql.Tx) ([]table, error) {
	rows, err := tx.Query(`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		return nil, err
	}
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			rows.Close()
			return nil, err
		}
		names = append(names, n)
	}
	rows.Close()
	var out []table
	for _, n := range names {
		rows, err := tx.Query(fmt.Sprintf(`SELECT * FROM "%s" LIMIT 0`, n))
		if err != nil {
			return nil, err
		}
		cols, err := rows.Columns()
		rows.Close()
		if err != nil {
			return nil, err
		}
		out = append(out, table{n, cols})
	}
	return out, nil
}

func rewriteTable(log *slog.Logger, tx *sql.Tx, t table, r renamer, scopeCol string, id int64) (int, error) {
	type update struct {
		rowid int64
		col   string
		val   any
	}
	q := `SELECT rowid, "` + strings.Join(t.cols, `", "`) + `" FROM "` + t.name + `"`
	var args []any
	if scopeCol != "" {
		q += ` WHERE "` + scopeCol + `" = ?`
		args = append(args, id)
	}
	rows, err := tx.Query(q, args...)
	if err != nil {
		return 0, err
	}
	var updates []update
	for rows.Next() {
		var rowid int64
		vals := make([]any, len(t.cols))
		ptrs := []any{&rowid}
		for i := range vals {
			ptrs = append(ptrs, &vals[i])
		}
		if err := rows.Scan(ptrs...); err != nil {
			rows.Close()
			return 0, err
		}
		for i, v := range vals {
			if nv, ok := rewriteValue(v, r); ok {
				updates = append(updates, update{rowid, t.cols[i], nv})
			} else if r.warn && mentionsValue(v, r) {
				log.Warn("namespace conversion: value left as it is", "table", t.name, "column", t.cols[i], "rowid", rowid)
			}
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	for _, u := range updates {
		if _, err := tx.Exec(fmt.Sprintf(`UPDATE "%s" SET "%s" = ? WHERE rowid = ?`, t.name, u.col), u.val, u.rowid); err != nil {
			return 0, err
		}
	}
	return len(updates), nil
}

func mentionsValue(v any, r renamer) bool {
	switch v := v.(type) {
	case string:
		return r.mentions([]byte(v))
	case []byte:
		return r.mentions(v)
	}
	return false
}

// rewriteValue returns v with the renamed strings replaced and whether it
// changed. JSON is walked (strings, keys, and base64 strings holding NBT),
// NBT is rewritten tag by tag, and other text is replaced as a whole.
func rewriteValue(v any, r renamer) (any, bool) {
	switch v := v.(type) {
	case string:
		nb, ok := rewriteBytes([]byte(v), r)
		return string(nb), ok
	case []byte:
		return rewriteBytes(v, r)
	}
	return v, false
}

func rewriteBytes(b []byte, r renamer) ([]byte, bool) {
	if t := bytes.TrimSpace(b); len(t) > 0 && (t[0] == '{' || t[0] == '[') && json.Valid(t) {
		nb, err := rewriteJSON(b, r)
		return nb, err == nil && !bytes.Equal(nb, b)
	}
	if !r.mentions(b) {
		return b, false
	}
	if b[0] == 10 {
		if nb, err := nbtrewrite.NBTKeys(b, r.rep, r.key); err == nil {
			return nb, !bytes.Equal(nb, b)
		}
	}
	if utf8.Valid(b) {
		nb := []byte(r.rep(string(b)))
		return nb, !bytes.Equal(nb, b)
	}
	return b, false
}

func rewriteJSON(b []byte, r renamer) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	changed := false
	v = walkJSON(v, r, &changed)
	if !changed {
		return b, nil
	}
	return json.Marshal(v)
}

func walkJSON(v any, r renamer, changed *bool) any {
	switch v := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, e := range v {
			nk := r.rep(k)
			if nk != k {
				*changed = true
			}
			out[nk] = walkJSON(e, r, changed)
		}
		return out
	case []any:
		for i, e := range v {
			v[i] = walkJSON(e, r, changed)
		}
		return v
	case string:
		if ns := r.rep(v); ns != v {
			*changed = true
			return ns
		}
		// []byte fields are base64 in JSON: lost items hold item NBT.
		e := r.embedded()
		if raw, err := base64.StdEncoding.DecodeString(v); err == nil && len(raw) > 0 && raw[0] == 10 && e.mentions(raw) {
			if nb, err := nbtrewrite.NBTKeys(raw, e.rep, e.key); err == nil {
				*changed = true
				return base64.StdEncoding.EncodeToString(nb)
			}
		}
	}
	return v
}
