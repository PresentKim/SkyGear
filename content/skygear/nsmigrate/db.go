package nsmigrate

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"unicode/utf8"

	_ "modernc.org/sqlite"
)

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

	tables, err := tableNames(tx)
	if err != nil {
		return 0, err
	}
	for _, t := range tables {
		n, err := convertTable(log, tx, t)
		if err != nil {
			return 0, fmt.Errorf("table %s: %w", t, err)
		}
		changed += n
	}
	return changed, tx.Commit()
}

func tableNames(tx *sql.Tx) ([]string, error) {
	rows, err := tx.Query(`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		names = append(names, n)
	}
	return names, rows.Err()
}

func convertTable(log *slog.Logger, tx *sql.Tx, table string) (int, error) {
	rows, err := tx.Query(fmt.Sprintf(`SELECT * FROM "%s" LIMIT 0`, table))
	if err != nil {
		return 0, err
	}
	cols, err := rows.Columns()
	rows.Close()
	if err != nil {
		return 0, err
	}
	type update struct {
		rowid int64
		col   string
		val   any
	}
	var updates []update
	sel := `SELECT rowid, "` + strings.Join(cols, `", "`) + `" FROM "` + table + `"`
	rows, err = tx.Query(sel)
	if err != nil {
		return 0, err
	}
	for rows.Next() {
		var rowid int64
		vals := make([]any, len(cols))
		ptrs := []any{&rowid}
		for i := range vals {
			ptrs = append(ptrs, &vals[i])
		}
		if err := rows.Scan(ptrs...); err != nil {
			rows.Close()
			return 0, err
		}
		for i, v := range vals {
			if nv, ok := convertValue(v); ok {
				updates = append(updates, update{rowid, cols[i], nv})
			} else if hasOldValue(v) {
				log.Warn("namespace conversion: value left as it is", "table", table, "column", cols[i], "rowid", rowid)
			}
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	for _, u := range updates {
		if _, err := tx.Exec(fmt.Sprintf(`UPDATE "%s" SET "%s" = ? WHERE rowid = ?`, table, u.col), u.val, u.rowid); err != nil {
			return 0, err
		}
	}
	return len(updates), nil
}

func hasOldValue(v any) bool {
	switch v := v.(type) {
	case string:
		return hasOld([]byte(v)) || hasDFKey([]byte(v))
	case []byte:
		return hasOld(v) || hasDFKey(v)
	}
	return false
}

// convertValue returns v with the old names replaced and whether it changed.
// JSON is walked (strings, and base64 strings holding NBT), NBT is rewritten
// tag by tag, and other text is replaced as it is.
func convertValue(v any) (any, bool) {
	switch v := v.(type) {
	case string:
		nb, ok := convertBytes([]byte(v))
		return string(nb), ok
	case []byte:
		return convertBytes(v)
	}
	return v, false
}

func convertBytes(b []byte) ([]byte, bool) {
	if t := bytes.TrimSpace(b); len(t) > 0 && (t[0] == '{' || t[0] == '[') && json.Valid(t) {
		nb, err := convertJSON(b)
		if err == nil && !bytes.Equal(nb, b) {
			return nb, true
		}
		if err == nil {
			return b, false
		}
	}
	if !hasOld(b) && !hasDFKey(b) {
		return b, false
	}
	if len(b) > 0 && b[0] == 10 {
		if nb, err := rewriteNBT(b, replaceKey); err == nil {
			return nb, !bytes.Equal(nb, b)
		}
	}
	if utf8.Valid(b) {
		nb := []byte(replaceKey(string(b)))
		return nb, !bytes.Equal(nb, b)
	}
	return b, false
}

func convertJSON(b []byte) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	changed := false
	v = walkJSON(v, &changed)
	if !changed {
		return b, nil
	}
	return json.Marshal(v)
}

func walkJSON(v any, changed *bool) any {
	switch v := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, e := range v {
			nk := replaceKey(k)
			if nk != k {
				*changed = true
			}
			out[nk] = walkJSON(e, changed)
		}
		return out
	case []any:
		for i, e := range v {
			v[i] = walkJSON(e, changed)
		}
		return v
	case string:
		if ns := replaceKey(v); ns != v {
			*changed = true
			return ns
		}
		// []byte fields are base64 in JSON: the lost items hold item NBT.
		if raw, err := base64.StdEncoding.DecodeString(v); err == nil && len(raw) > 0 && raw[0] == 10 && hasOld(raw) {
			if nb, err := rewriteNBT(raw, replaceOld); err == nil {
				*changed = true
				return base64.StdEncoding.EncodeToString(nb)
			}
		}
	}
	return v
}
