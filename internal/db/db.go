// Package db stores platform data (accounts, saves, members, per-save player
// state) in SQLite. World blocks live in per-save mcdb folders instead.
package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// ErrNotFound is returned when a requested row does not exist.
var ErrNotFound = errors.New("db: not found")

// DB is the platform database.
type DB struct {
	sql  *sql.DB
	path string
}

const memoryPath = ":memory:"

func fileDSN(path string) string {
	return "file:" + path + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
}

// Open opens (and migrates) the database at path. ":memory:" opens a
// private in-memory database, useful for tests.
func Open(path string) (*DB, error) {
	dsn := fileDSN(path)
	if path == memoryPath {
		dsn = "file::memory:?_pragma=foreign_keys(1)"
	}
	s, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	// SQLite allows one writer; a single connection also keeps :memory: shared.
	s.SetMaxOpenConns(1)
	d := &DB{sql: s, path: path}
	if err := d.migrate(); err != nil {
		_ = s.Close()
		return nil, err
	}
	return d, nil
}

// Close closes the database.
func (d *DB) Close() error { return d.sql.Close() }

// migrations are applied in order; the index+1 is stored in PRAGMA user_version.
// Never edit an applied migration, append a new one instead.
var migrations = []string{
	`CREATE TABLE accounts (
		id            TEXT PRIMARY KEY, -- player UUID
		xuid          TEXT NOT NULL DEFAULT '',
		name          TEXT NOT NULL,
		guide_version INTEGER NOT NULL DEFAULT 0,
		created_at    INTEGER NOT NULL,
		last_seen     INTEGER NOT NULL
	);
	CREATE TABLE saves (
		id            INTEGER PRIMARY KEY AUTOINCREMENT,
		game_type     TEXT NOT NULL,
		owner         TEXT NOT NULL REFERENCES accounts(id),
		slot          INTEGER NOT NULL,
		name          TEXT NOT NULL,
		invite_code   TEXT,
		instance_time INTEGER NOT NULL DEFAULT 0, -- total active milliseconds
		day           INTEGER NOT NULL DEFAULT 1,
		day_time      INTEGER NOT NULL DEFAULT 0, -- milliseconds into the current day
		created_at    INTEGER NOT NULL,
		last_played   INTEGER NOT NULL,
		UNIQUE (owner, game_type, slot)
	);
	CREATE UNIQUE INDEX saves_invite_code ON saves(invite_code) WHERE invite_code IS NOT NULL;
	CREATE TABLE save_members (
		save_id    INTEGER NOT NULL REFERENCES saves(id) ON DELETE CASCADE,
		account_id TEXT NOT NULL REFERENCES accounts(id),
		role       TEXT NOT NULL, -- host / cohost / guest
		joined_at  INTEGER NOT NULL,
		left_at    INTEGER,       -- set when kicked or left; the row is kept
		PRIMARY KEY (save_id, account_id)
	);
	CREATE TABLE save_players (
		save_id    INTEGER NOT NULL REFERENCES saves(id) ON DELETE CASCADE,
		account_id TEXT NOT NULL REFERENCES accounts(id),
		money      INTEGER NOT NULL DEFAULT 0,
		state      BLOB,
		PRIMARY KEY (save_id, account_id)
	);
	CREATE TABLE save_state (
		save_id INTEGER NOT NULL REFERENCES saves(id) ON DELETE CASCADE,
		key     TEXT NOT NULL,
		value   BLOB NOT NULL,
		PRIMARY KEY (save_id, key)
	);`,
	// 2: cheque ledger (docs/design/02-dragonfactory.md §7).
	`CREATE TABLE cheques (
		id          TEXT PRIMARY KEY,
		save_id     INTEGER NOT NULL REFERENCES saves(id) ON DELETE CASCADE,
		amount      INTEGER NOT NULL,
		issued_by   TEXT NOT NULL, -- account id, or '' for rewards
		issued_at   INTEGER NOT NULL,
		redeemed_by TEXT,
		redeemed_at INTEGER
	);`,
	// 3: friends (docs/design/01-platform.md §4). An accepted friendship is
	// stored as two rows, one per direction.
	`CREATE TABLE friend_requests (
		from_id    TEXT NOT NULL REFERENCES accounts(id),
		to_id      TEXT NOT NULL REFERENCES accounts(id),
		created_at INTEGER NOT NULL,
		PRIMARY KEY (from_id, to_id)
	);
	CREATE TABLE friends (
		account_id TEXT NOT NULL REFERENCES accounts(id),
		friend_id  TEXT NOT NULL REFERENCES accounts(id),
		since      INTEGER NOT NULL,
		PRIMARY KEY (account_id, friend_id)
	);
	CREATE TABLE blocks (
		account_id TEXT NOT NULL REFERENCES accounts(id),
		blocked_id TEXT NOT NULL REFERENCES accounts(id),
		created_at INTEGER NOT NULL,
		PRIMARY KEY (account_id, blocked_id)
	);
	CREATE INDEX accounts_name ON accounts(name COLLATE NOCASE);`,
	// 4: account achievements, the equipped title and the name colour
	// (docs/design/01-platform.md §5).
	`ALTER TABLE accounts ADD COLUMN title TEXT NOT NULL DEFAULT '';
	ALTER TABLE accounts ADD COLUMN name_color TEXT NOT NULL DEFAULT '';
	CREATE TABLE achievements (
		account_id TEXT NOT NULL REFERENCES accounts(id),
		id         TEXT NOT NULL,
		earned_at  INTEGER NOT NULL,
		PRIMARY KEY (account_id, id)
	);`,
	// 5: the counters of account achievements (docs/design/01-platform.md §5).
	`CREATE TABLE counters (
		account_id TEXT NOT NULL REFERENCES accounts(id),
		key        TEXT NOT NULL,
		value      INTEGER NOT NULL,
		PRIMARY KEY (account_id, key)
	);`,
	// 6: the settings a player chooses for itself, by key (the iron hammer's
	// diagonal vein mining, docs/design/02-dragonfactory.md §4).
	`CREATE TABLE account_settings (
		account_id TEXT NOT NULL REFERENCES accounts(id),
		key        TEXT NOT NULL,
		value      TEXT NOT NULL,
		PRIMARY KEY (account_id, key)
	);`,
}

func (d *DB) migrate() error {
	var v int
	if err := d.sql.QueryRow(`PRAGMA user_version`).Scan(&v); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}
	for i := v; i < len(migrations); i++ {
		tx, err := d.sql.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(migrations[i]); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migration %d: %w", i+1, err)
		}
		if _, err := tx.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, i+1)); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func now() int64 { return time.Now().UnixMilli() }

func notFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// Backup writes a consistent copy of the database to path (which must not
// exist yet) while the database stays in use. A file database is copied
// over its own connection, so other queries do not wait for the copy.
func (d *DB) Backup(path string) error {
	s := d.sql
	if d.path != memoryPath {
		var err error
		if s, err = sql.Open("sqlite", fileDSN(d.path)); err != nil {
			return fmt.Errorf("backup db: %w", err)
		}
		defer s.Close()
		s.SetMaxOpenConns(1)
	}
	if _, err := s.Exec(`VACUUM INTO ?`, path); err != nil {
		return fmt.Errorf("backup db: %w", err)
	}
	return nil
}
