package db

import (
	"database/sql"
	"errors"
	"time"
)

var (
	// ErrSelf is returned for friend actions on oneself.
	ErrSelf = errors.New("db: cannot target yourself")
	// ErrAlreadyFriends is returned when the accounts are already friends.
	ErrAlreadyFriends = errors.New("db: already friends")
	// ErrBlocked is returned when either account blocked the other.
	ErrBlocked = errors.New("db: blocked")
)

// AccountByName finds an account by player name, ignoring case.
func (d *DB) AccountByName(name string) (Account, error) {
	return scanAccount(d.sql.QueryRow(`SELECT `+accountColumns+` FROM accounts a WHERE a.name = ? COLLATE NOCASE ORDER BY a.last_seen DESC LIMIT 1`, name))
}

func (d *DB) exists(q string, args ...any) (bool, error) {
	var one int
	err := d.sql.QueryRow(q, args...).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// IsFriend reports if a and b are friends.
func (d *DB) IsFriend(a, b string) (bool, error) {
	return d.exists(`SELECT 1 FROM friends WHERE account_id = ? AND friend_id = ?`, a, b)
}

// Blocked reports if a blocked b.
func (d *DB) Blocked(a, b string) (bool, error) {
	return d.exists(`SELECT 1 FROM blocks WHERE account_id = ? AND blocked_id = ?`, a, b)
}

// SendFriendRequest asks to to become from's friend. If to already asked
// from, they become friends right away and accepted is true.
func (d *DB) SendFriendRequest(from, to string) (accepted bool, err error) {
	if from == to {
		return false, ErrSelf
	}
	if ok, err := d.IsFriend(from, to); err != nil || ok {
		return false, errors.Join(err, map[bool]error{true: ErrAlreadyFriends}[ok])
	}
	for _, pair := range [][2]string{{from, to}, {to, from}} {
		if ok, err := d.Blocked(pair[0], pair[1]); err != nil || ok {
			return false, errors.Join(err, map[bool]error{true: ErrBlocked}[ok])
		}
	}
	if ok, err := d.exists(`SELECT 1 FROM friend_requests WHERE from_id = ? AND to_id = ?`, to, from); err != nil {
		return false, err
	} else if ok {
		return true, d.AcceptFriend(from, to)
	}
	_, err = d.sql.Exec(`INSERT INTO friend_requests (from_id, to_id, created_at) VALUES (?, ?, ?) ON CONFLICT DO NOTHING`, from, to, now())
	return false, err
}

// FriendRequests returns the accounts that sent `to` a friend request.
func (d *DB) FriendRequests(to string) ([]Account, error) {
	return d.accounts(`SELECT `+accountColumns+` FROM friend_requests r JOIN accounts a ON a.id = r.from_id WHERE r.to_id = ? ORDER BY r.created_at`, to)
}

// FriendRequest is a friend request someone sent: who and when.
type FriendRequest struct {
	From Account
	At   time.Time
}

// FriendRequestsTo returns the friend requests sent to `to`, oldest first.
func (d *DB) FriendRequestsTo(to string) ([]FriendRequest, error) {
	rows, err := d.sql.Query(`SELECT `+accountColumns+`, r.created_at FROM friend_requests r JOIN accounts a ON a.id = r.from_id WHERE r.to_id = ? ORDER BY r.created_at`, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []FriendRequest{}
	for rows.Next() {
		var r FriendRequest
		var created, seen, at int64
		if err := rows.Scan(&r.From.ID, &r.From.XUID, &r.From.Name, &r.From.GuideVersion, &r.From.Title, &r.From.NameColor, &created, &seen, &at); err != nil {
			return nil, err
		}
		r.From.CreatedAt, r.From.LastSeen, r.At = time.UnixMilli(created), time.UnixMilli(seen), time.UnixMilli(at)
		out = append(out, r)
	}
	return out, rows.Err()
}

// AcceptFriend accepts from's request to `to`.
func (d *DB) AcceptFriend(to, from string) error {
	tx, err := d.sql.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	t := now()
	if _, err := tx.Exec(`DELETE FROM friend_requests WHERE (from_id = ? AND to_id = ?) OR (from_id = ? AND to_id = ?)`, from, to, to, from); err != nil {
		return err
	}
	for _, pair := range [][2]string{{to, from}, {from, to}} {
		if _, err := tx.Exec(`INSERT INTO friends (account_id, friend_id, since) VALUES (?, ?, ?) ON CONFLICT DO NOTHING`, pair[0], pair[1], t); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// DeclineFriend removes from's request to `to`.
func (d *DB) DeclineFriend(to, from string) error {
	_, err := d.sql.Exec(`DELETE FROM friend_requests WHERE from_id = ? AND to_id = ?`, from, to)
	return err
}

// Friends returns a's friends ordered by name.
func (d *DB) Friends(a string) ([]Account, error) {
	return d.accounts(`SELECT `+accountColumns+` FROM friends f JOIN accounts a ON a.id = f.friend_id WHERE f.account_id = ? ORDER BY a.name COLLATE NOCASE`, a)
}

// RemoveFriend ends the friendship of a and b.
func (d *DB) RemoveFriend(a, b string) error {
	_, err := d.sql.Exec(`DELETE FROM friends WHERE (account_id = ? AND friend_id = ?) OR (account_id = ? AND friend_id = ?)`, a, b, b, a)
	return err
}

// Block makes a block b: their friendship and requests are removed.
func (d *DB) Block(a, b string) error {
	if a == b {
		return ErrSelf
	}
	if err := d.RemoveFriend(a, b); err != nil {
		return err
	}
	if _, err := d.sql.Exec(`DELETE FROM friend_requests WHERE (from_id = ? AND to_id = ?) OR (from_id = ? AND to_id = ?)`, a, b, b, a); err != nil {
		return err
	}
	_, err := d.sql.Exec(`INSERT INTO blocks (account_id, blocked_id, created_at) VALUES (?, ?, ?) ON CONFLICT DO NOTHING`, a, b, now())
	return err
}

// Unblock removes a's block of b.
func (d *DB) Unblock(a, b string) error {
	_, err := d.sql.Exec(`DELETE FROM blocks WHERE account_id = ? AND blocked_id = ?`, a, b)
	return err
}

// BlockedAccounts returns the accounts a blocked.
func (d *DB) BlockedAccounts(a string) ([]Account, error) {
	return d.accounts(`SELECT `+accountColumns+` FROM blocks b JOIN accounts a ON a.id = b.blocked_id WHERE b.account_id = ? ORDER BY a.name COLLATE NOCASE`, a)
}

func (d *DB) accounts(q string, args ...any) ([]Account, error) {
	rows, err := d.sql.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Account{}
	for rows.Next() {
		a, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
