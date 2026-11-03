package db

// Settings returns every setting of the account by key in one query.
func (d *DB) Settings(accountID string) (map[string]string, error) {
	rows, err := d.sql.Query(`SELECT key, value FROM account_settings WHERE account_id = ?`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}
