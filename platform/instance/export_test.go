package instance

import "github.com/presentkim/skygear/platform/db"

// addGuest inserts a guest member directly, without an invite.
func addGuest(d *db.DB, saveID int64, acc string) error {
	return d.AddMember(saveID, acc, db.RoleGuest)
}
