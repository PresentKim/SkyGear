// Package naming identifies and names items and blocks: item keys, client
// translation keys and names in a player's language (catalog, pack texts).
package naming

import (
	"io/fs"
	"path"
	"strings"

	_ "github.com/presentkim/skygear/content/skygear/locales" // this server's texts

	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/slotdeco"
)

// VanillaName returns the name of a vanilla item or block in lang, by name
// without the "minecraft:" prefix, from the catalog ("vanilla.<name>").
func VanillaName(lang, name string) (string, bool) {
	key := "vanilla." + name
	if !i18n.Has(key) {
		return "", false
	}
	return i18n.T(lang, key), true
}

// PackNames are the names in the pack's texts/*.lang by catalog language
// ("ko" for ko_KR.lang) and translation key (LoadNames).
var PackNames = map[string]map[string]string{}

// LoadNames reads the names of the pack's items and blocks from its
// texts/*.lang, so the game's own texts name them in the player's language
// as the client does. Without it they are named by Name() (Korean).
func LoadNames(pack fs.FS) error {
	files, err := fs.Glob(pack, "texts/*.lang")
	if err != nil {
		return err
	}
	for _, f := range files {
		b, err := fs.ReadFile(pack, f)
		if err != nil {
			return err
		}
		lang, _, _ := strings.Cut(strings.ToLower(strings.TrimSuffix(path.Base(f), ".lang")), "_")
		names := map[string]string{}
		for _, line := range strings.Split(string(b), "\n") {
			if k, v, ok := strings.Cut(strings.TrimRight(line, "\r"), "="); ok {
				names[k] = v
			}
		}
		PackNames[lang] = names
	}
	return nil
}

// PackName is the name of a translation key of the pack in lang (the
// default language's when lang has none), without the slot marks.
func PackName(lang, key string) (string, bool) {
	n, ok := PackNames[lang][key]
	if !ok {
		n, ok = PackNames[i18n.Default][key]
	}
	return slotdeco.Strip(n), ok
}

// CustomItemName is the name of one of our items in lang.
func CustomItemName(lang string, c world.CustomItem) string {
	id, _ := c.EncodeItem()
	if n, ok := PackName(lang, "item."+id+".name"); ok {
		return n
	}
	return c.Name()
}
