package db

import (
	"database/sql"
	"errors"
	"time"
)

// Account is a player account, shared by all saves.
type Account struct {
	ID           string // player UUID
	XUID         string
	Name         string
	GuideVersion int
	// Title is the equipped achievement title ID ("" for none); NameColor
	// the chosen name colour ID ("" for the default).
	Title, NameColor string
	CreatedAt        time.Time
	LastSeen         time.Time
}

// TouchAccount creates the account if needed and updates its name, XUID and
// last seen time. It returns the stored account.
func (d *DB) TouchAccount(id, xuid, name string) (Account, error) {
	t := now()
	_, err := d.sql.Exec(`INSERT INTO accounts (id, xuid, name, created_at, last_seen) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET xuid = excluded.xuid, name = excluded.name, last_seen = excluded.last_seen`,
		id, xuid, name, t, t)
	if err != nil {
		return Account{}, err
	}
	return d.Account(id)
}

const accountColumns = `a.id, a.xuid, a.name, a.guide_version, a.title, a.name_color, a.created_at, a.last_seen`

func scanAccount(row interface{ Scan(...any) error }) (Account, error) {
	var a Account
	var created, seen int64
	if err := row.Scan(&a.ID, &a.XUID, &a.Name, &a.GuideVersion, &a.Title, &a.NameColor, &created, &seen); err != nil {
		return Account{}, notFound(err)
	}
	a.CreatedAt, a.LastSeen = time.UnixMilli(created), time.UnixMilli(seen)
	return a, nil
}

// Account returns the account with id.
func (d *DB) Account(id string) (Account, error) {
	return scanAccount(d.sql.QueryRow(`SELECT `+accountColumns+` FROM accounts a WHERE a.id = ?`, id))
}

// SetGuideVersion records the server guide version the account has seen.
func (d *DB) SetGuideVersion(id string, v int) error {
	_, err := d.sql.Exec(`UPDATE accounts SET guide_version = ? WHERE id = ?`, v, id)
	return err
}

// SetTitle equips an achievement title ("" for none).
func (d *DB) SetTitle(id, title string) error {
	_, err := d.sql.Exec(`UPDATE accounts SET title = ? WHERE id = ?`, title, id)
	return err
}

// SetNameColor sets the name colour ID ("" for the default).
func (d *DB) SetNameColor(id, color string) error {
	_, err := d.sql.Exec(`UPDATE accounts SET name_color = ? WHERE id = ?`, color, id)
	return err
}

// Setting returns a setting of the account, "" when it was never set.
func (d *DB) Setting(id, key string) (string, error) {
	var v string
	err := d.sql.QueryRow(`SELECT value FROM account_settings WHERE account_id = ? AND key = ?`, id, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return v, err
}

// SetSetting stores a setting of the account.
func (d *DB) SetSetting(id, key, value string) error {
	_, err := d.sql.Exec(`INSERT INTO account_settings (account_id, key, value) VALUES (?, ?, ?)
		ON CONFLICT (account_id, key) DO UPDATE SET value = excluded.value`, id, key, value)
	return err
}

// GrantAchievement records an achievement and reports whether it is new.
func (d *DB) GrantAchievement(account, id string) (bool, error) {
	res, err := d.sql.Exec(`INSERT OR IGNORE INTO achievements (account_id, id, earned_at) VALUES (?, ?, ?)`, account, id, now())
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

// Achievements returns the IDs of an account's achievements, oldest first.
func (d *DB) Achievements(account string) ([]string, error) {
	rows, err := d.sql.Query(`SELECT id FROM achievements WHERE account_id = ? ORDER BY earned_at, id`, account)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// AddCounters adds to the counters of an account, creating them as needed.
func (d *DB) AddCounters(account string, delta map[string]int64) error {
	tx, err := d.sql.Begin()
	if err != nil {
		return err
	}
	for key, n := range delta {
		if _, err := tx.Exec(`INSERT INTO counters (account_id, key, value) SELECT id, ?, ? FROM accounts WHERE id = ?
			ON CONFLICT(account_id, key) DO UPDATE SET value = value + excluded.value`, key, n, account); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// Counters returns the counters of an account by key.
func (d *DB) Counters(account string) (map[string]int64, error) {
	rows, err := d.sql.Query(`SELECT key, value FROM counters WHERE account_id = ?`, account)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var key string
		var v int64
		if err := rows.Scan(&key, &v); err != nil {
			return nil, err
		}
		out[key] = v
	}
	return out, rows.Err()
}
