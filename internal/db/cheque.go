package db

import "errors"

// ErrChequeUsed is returned when redeeming a cheque that was already used.
var ErrChequeUsed = errors.New("db: cheque already redeemed")

// IssueCheque records a new cheque of a save.
func (d *DB) IssueCheque(id string, saveID int64, amount int64, issuedBy string) error {
	_, err := d.sql.Exec(`INSERT INTO cheques (id, save_id, amount, issued_by, issued_at) VALUES (?, ?, ?, ?, ?)`,
		id, saveID, amount, issuedBy, now())
	return err
}

// RedeemCheque marks a cheque of the save as used by an account and returns
// its amount. Unknown cheques return ErrNotFound, used ones ErrChequeUsed.
func (d *DB) RedeemCheque(id string, saveID int64, by string) (int64, error) {
	tx, err := d.sql.Begin()
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	var amount int64
	var used bool
	err = tx.QueryRow(`SELECT amount, redeemed_at IS NOT NULL FROM cheques WHERE id = ? AND save_id = ?`, id, saveID).Scan(&amount, &used)
	if err != nil {
		return 0, notFound(err)
	}
	if used {
		return 0, ErrChequeUsed
	}
	if _, err := tx.Exec(`UPDATE cheques SET redeemed_by = ?, redeemed_at = ? WHERE id = ?`, by, now(), id); err != nil {
		return 0, err
	}
	return amount, tx.Commit()
}
