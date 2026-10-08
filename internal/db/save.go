package db

import (
	"errors"
	"time"
)

// Role is a member's permission level in a save.
type Role string

const (
	RoleHost   Role = "host"
	RoleCohost Role = "cohost"
	RoleGuest  Role = "guest"
)

// CanActivate reports if a member with the role may activate the instance.
// Co-hosts are planned (docs/design/01-platform.md §3) but not enabled yet.
func (r Role) CanActivate() bool { return r == RoleHost }

// Save is a save slot of an account.
type Save struct {
	ID           int64
	GameType     string
	Owner        string
	Slot         int
	Name         string
	InviteCode   string
	InstanceTime time.Duration
	Day          int
	DayTime      time.Duration
	CreatedAt    time.Time
	LastPlayed   time.Time
}

const saveColumns = `id, game_type, owner, slot, name, COALESCE(invite_code, ''), instance_time, day, day_time, created_at, last_played`

func scanSave(row interface{ Scan(...any) error }) (Save, error) {
	var s Save
	var it, dt, created, played int64
	err := row.Scan(&s.ID, &s.GameType, &s.Owner, &s.Slot, &s.Name, &s.InviteCode, &it, &s.Day, &dt, &created, &played)
	if err != nil {
		return Save{}, notFound(err)
	}
	s.InstanceTime, s.DayTime = time.Duration(it)*time.Millisecond, time.Duration(dt)*time.Millisecond
	s.CreatedAt, s.LastPlayed = time.UnixMilli(created), time.UnixMilli(played)
	return s, nil
}

// ErrSlotUsed is returned when creating a save in an occupied slot.
var ErrSlotUsed = errors.New("db: save slot already used")

// CreateSave creates a save in slot for owner and adds the owner as host.
func (d *DB) CreateSave(owner, gameType string, slot int, name string) (Save, error) {
	tx, err := d.sql.Begin()
	if err != nil {
		return Save{}, err
	}
	defer func() { _ = tx.Rollback() }()
	t := now()
	res, err := tx.Exec(`INSERT INTO saves (game_type, owner, slot, name, created_at, last_played) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(owner, game_type, slot) DO NOTHING`, gameType, owner, slot, name, t, t)
	if err != nil {
		return Save{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Save{}, ErrSlotUsed
	}
	id, _ := res.LastInsertId()
	if _, err := tx.Exec(`INSERT INTO save_members (save_id, account_id, role, joined_at) VALUES (?, ?, ?, ?)`, id, owner, RoleHost, t); err != nil {
		return Save{}, err
	}
	if err := tx.Commit(); err != nil {
		return Save{}, err
	}
	return d.Save(id)
}

// Save returns the save with id.
func (d *DB) Save(id int64) (Save, error) {
	return scanSave(d.sql.QueryRow(`SELECT `+saveColumns+` FROM saves WHERE id = ?`, id))
}

// SavesOf returns the saves owned by owner for a game type, ordered by slot.
func (d *DB) SavesOf(owner, gameType string) ([]Save, error) {
	rows, err := d.sql.Query(`SELECT `+saveColumns+` FROM saves WHERE owner = ? AND game_type = ? ORDER BY slot`, owner, gameType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Save
	for rows.Next() {
		s, err := scanSave(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// RenameSave changes the name of a save.
func (d *DB) RenameSave(id int64, name string) error {
	_, err := d.sql.Exec(`UPDATE saves SET name = ? WHERE id = ?`, name, id)
	return err
}

// DeleteSave deletes a save and all of its rows.
func (d *DB) DeleteSave(id int64) error {
	_, err := d.sql.Exec(`DELETE FROM saves WHERE id = ?`, id)
	return err
}

// UpdateSaveClock stores the instance clock of a save and marks it played.
func (d *DB) UpdateSaveClock(id int64, instanceTime time.Duration, day int, dayTime time.Duration) error {
	_, err := d.sql.Exec(`UPDATE saves SET instance_time = ?, day = ?, day_time = ?, last_played = ? WHERE id = ?`,
		instanceTime.Milliseconds(), day, dayTime.Milliseconds(), now(), id)
	return err
}

// Member is an account's membership of a save.
type Member struct {
	AccountID string
	Name      string
	Role      Role
	JoinedAt  time.Time
	Left      bool
}

// Members returns all members of a save, including ones that left.
func (d *DB) Members(saveID int64) ([]Member, error) {
	rows, err := d.sql.Query(`SELECT m.account_id, a.name, m.role, m.joined_at, m.left_at IS NOT NULL
		FROM save_members m JOIN accounts a ON a.id = m.account_id WHERE m.save_id = ? ORDER BY m.joined_at`, saveID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Member
	for rows.Next() {
		var m Member
		var joined int64
		if err := rows.Scan(&m.AccountID, &m.Name, &m.Role, &joined, &m.Left); err != nil {
			return nil, err
		}
		m.JoinedAt = time.UnixMilli(joined)
		out = append(out, m)
	}
	return out, rows.Err()
}

// MemberRole returns the role of an active (not left) member.
func (d *DB) MemberRole(saveID int64, accountID string) (Role, error) {
	var r Role
	err := d.sql.QueryRow(`SELECT role FROM save_members WHERE save_id = ? AND account_id = ? AND left_at IS NULL`, saveID, accountID).Scan(&r)
	return r, notFound(err)
}

// PlayerState returns the stored per-save state of a player. A player that
// never entered the save returns ErrNotFound.
func (d *DB) PlayerState(saveID int64, accountID string) (money int64, state []byte, err error) {
	err = d.sql.QueryRow(`SELECT money, state FROM save_players WHERE save_id = ? AND account_id = ?`, saveID, accountID).Scan(&money, &state)
	return money, state, notFound(err)
}

// SetPlayerState stores the per-save state blob of a player.
func (d *DB) SetPlayerState(saveID int64, accountID string, state []byte) error {
	_, err := d.sql.Exec(`INSERT INTO save_players (save_id, account_id, state) VALUES (?, ?, ?)
		ON CONFLICT(save_id, account_id) DO UPDATE SET state = excluded.state`, saveID, accountID, state)
	return err
}

// SaveState returns a game specific value of a save, or ErrNotFound.
func (d *DB) SaveState(saveID int64, key string) ([]byte, error) {
	var v []byte
	err := d.sql.QueryRow(`SELECT value FROM save_state WHERE save_id = ? AND key = ?`, saveID, key).Scan(&v)
	return v, notFound(err)
}

// SetSaveState stores a game specific value of a save.
func (d *DB) SetSaveState(saveID int64, key string, value []byte) error {
	_, err := d.sql.Exec(`INSERT INTO save_state (save_id, key, value) VALUES (?, ?, ?)
		ON CONFLICT(save_id, key) DO UPDATE SET value = excluded.value`, saveID, key, value)
	return err
}

// AddMember adds (or re-adds) an account to a save with role.
func (d *DB) AddMember(saveID int64, accountID string, role Role) error {
	_, err := d.sql.Exec(`INSERT INTO save_members (save_id, account_id, role, joined_at) VALUES (?, ?, ?, ?)
		ON CONFLICT(save_id, account_id) DO UPDATE SET role = excluded.role, left_at = NULL`, saveID, accountID, role, now())
	return err
}
