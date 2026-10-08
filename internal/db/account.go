package db

import "time"

// Account is a player account, shared by all saves.
type Account struct {
	ID           string // player UUID
	XUID         string
	Name         string
	GuideVersion int
	CreatedAt    time.Time
	LastSeen     time.Time
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
	err := d.sql.QueryRow(`SELECT id, xuid, name, guide_version, created_at, last_seen FROM accounts WHERE id = ?`, id).
		Scan(&a.ID, &a.XUID, &a.Name, &a.GuideVersion, &created, &seen)
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
