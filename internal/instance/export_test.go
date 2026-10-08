package instance

import "github.com/presentkim/dragonfactory/internal/db"

// addGuest inserts a guest member directly; invites arrive in M3.
func addGuest(d *db.DB, saveID int64, acc string) error {
	return d.AddMember(saveID, acc, db.RoleGuest)
}
