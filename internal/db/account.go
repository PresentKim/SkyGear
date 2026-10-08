package db

import "time"

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

// Account returns the account with id.
func (d *DB) Account(id string) (Account, error) {
	var a Account
	var created, seen int64
	err := d.sql.QueryRow(`SELECT id, xuid, name, guide_version, title, name_color, created_at, last_seen FROM accounts WHERE id = ?`, id).
		Scan(&a.ID, &a.XUID, &a.Name, &a.GuideVersion, &a.Title, &a.NameColor, &created, &seen)
	if err != nil {
		return Account{}, notFound(err)
	}
	a.CreatedAt, a.LastSeen = time.UnixMilli(created), time.UnixMilli(seen)
	return a, nil
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
