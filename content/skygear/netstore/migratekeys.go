package netstore

import (
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/migrate"
)

// A save patch that renames an item has to rename it in the keys of the disks
// too. migrate rewrites the NBT behind a key; the key of the stack that
// results is made here, the way the running server makes it.
func init() { migrate.ItemKey = migratedKey }

// migratedKey is KeyOf for the little endian item NBT itemNBT.
func migratedKey(itemNBT []byte) (key string, ok bool) {
	defer func() {
		if recover() != nil {
			key, ok = "", false
		}
	}()
	st := core.DecodeItem(itemNBT)
	if st.Empty() {
		return "", false
	}
	return string(KeyOf(st)), true
}
