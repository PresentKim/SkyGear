package db

import "time"

// Invite is an invite to a save someone got: the save, who sent it and when.
type Invite struct {
	SaveID   int64
	SaveName string
	From     Account
	At       time.Time
}

// AddInvite invites to to the save saveID from from. An earlier invite to
// the same save is replaced.
func (d *DB) AddInvite(saveID int64, to, from string) error {
	_, err := d.sql.Exec(`INSERT INTO save_invites (save_id, to_id, from_id, created_at) VALUES (?, ?, ?, ?)
		ON CONFLICT(save_id, to_id) DO UPDATE SET from_id = excluded.from_id, created_at = excluded.created_at`, saveID, to, from, now())
	return err
}

// Invites returns the invites to got since since, newest first. Older ones
// are deleted.
func (d *DB) Invites(to string, since time.Time) ([]Invite, error) {
	if _, err := d.sql.Exec(`DELETE FROM save_invites WHERE to_id = ? AND created_at < ?`, to, since.UnixMilli()); err != nil {
		return nil, err
	}
	rows, err := d.sql.Query(`SELECT i.save_id, s.name, `+accountColumns+`, i.created_at
		FROM save_invites i JOIN saves s ON s.id = i.save_id JOIN accounts a ON a.id = i.from_id
		WHERE i.to_id = ? ORDER BY i.created_at DESC, i.rowid DESC`, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Invite{}
	for rows.Next() {
		var in Invite
		var created, seen, at int64
		if err := rows.Scan(&in.SaveID, &in.SaveName, &in.From.ID, &in.From.XUID, &in.From.Name, &in.From.GuideVersion, &in.From.Title, &in.From.NameColor, &created, &seen, &at); err != nil {
			return nil, err
		}
		in.From.CreatedAt, in.From.LastSeen, in.At = time.UnixMilli(created), time.UnixMilli(seen), time.UnixMilli(at)
		out = append(out, in)
	}
	return out, rows.Err()
}

// RemoveInvite deletes the invite to to the save saveID.
func (d *DB) RemoveInvite(saveID int64, to string) error {
	_, err := d.sql.Exec(`DELETE FROM save_invites WHERE save_id = ? AND to_id = ?`, saveID, to)
	return err
}

// HasInvite reports if to has an invite to the save saveID sent since since.
func (d *DB) HasInvite(saveID int64, to string, since time.Time) (bool, error) {
	return d.exists(`SELECT 1 FROM save_invites WHERE save_id = ? AND to_id = ? AND created_at >= ?`, saveID, to, since.UnixMilli())
}
