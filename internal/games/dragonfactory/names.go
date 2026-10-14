package dragonfactory

import (
	"github.com/presentkim/dragonfactory/internal/i18n"
	"github.com/presentkim/dragonfactory/internal/timed"
)

// Time-limited item messages use the Korean names.
func init() { timed.Name = itemName }

// koreanName returns the Korean name of a vanilla item the game shows in its
// own texts (missions, sidebar, forms), by item name without the "minecraft:"
// prefix, from the catalog ("vanilla.<name>"). Items without one show their
// identifier. The client translates names on its own where it can (WAILA).
func koreanName(name string) (string, bool) {
	key := "vanilla." + name
	if !i18n.Has(key) {
		return "", false
	}
	return i18n.T(i18n.Default, key), true
}
