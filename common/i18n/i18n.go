// Package i18n is the server's own message catalog, for the texts the game
// sends in a player's language (docs/tech/i18n.md). Texts are looked up by
// key in the catalog of the player's language, then in the default one
// (Korean), so a language can be added by a file and may be incomplete.
// Names of blocks and items are not here: the client translates those by
// itself from the resource pack's texts/*.lang and vanilla keys.
//
// This package holds the texts of the common packages (locales/); a
// server adds its own with Register.
package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"

	"github.com/df-mc/dragonfly/server/player"
	"golang.org/x/text/language"
)

// Default is the language everything is written in first; a key missing in
// another language shows the default one.
const Default = "ko"

//go:embed locales/*.json
var files embed.FS

// catalogs are the texts by language code ("ko", "en") and key.
var catalogs = map[string]map[string]string{}

func init() {
	sub, err := fs.Sub(files, "locales")
	if err != nil {
		panic(err)
	}
	Register(sub)
}

// Register adds the catalog files at the root of fsys ("ko.json",
// "en.json", ...) to the catalogs; a key already present is replaced.
// "ko.json" and "ko.vanilla.json" are both Korean: a language is the part of
// the name before the first dot, and its files are merged in name order (the
// generated vanilla names after the hand-written texts). Register panics on
// a bad file. Call it during package initialisation, before any lookup.
func Register(fsys fs.FS) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		panic(err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := fs.ReadFile(fsys, e.Name())
		if err != nil {
			panic(err)
		}
		lang, _, _ := strings.Cut(e.Name(), ".")
		c := map[string]string{}
		if err := json.Unmarshal(data, &c); err != nil {
			panic(fmt.Sprintf("i18n: %s: %v", e.Name(), err))
		}
		if catalogs[lang] == nil {
			catalogs[lang] = map[string]string{}
		}
		for k, v := range c {
			catalogs[lang][k] = v
		}
	}
}

// Languages are the codes that have a catalog.
func Languages() []string {
	var out []string
	for l := range catalogs {
		out = append(out, l)
	}
	return out
}

// Locale is anything that knows a player's client language.
type Locale interface{ Locale() language.Tag }

// Lang returns the catalog language of a client language: its base language
// when a catalog exists ("ko-KR" → "ko"), else the default.
func Lang(tag language.Tag) string {
	base, _ := tag.Base()
	if _, ok := catalogs[base.String()]; ok {
		return base.String()
	}
	return Default
}

// For returns the language of p; a nil p (a test, the console) is the default.
func For(p Locale) string {
	if p == nil {
		return Default
	}
	return Lang(p.Locale())
}

// Of is For for a *player.Player that may be nil.
func Of(p *player.Player) string {
	if p == nil {
		return Default
	}
	return For(p)
}

// Has reports whether a key exists in the default catalog.
func Has(key string) bool {
	_, ok := catalogs[Default][key]
	return ok
}

// T returns the text of key in lang, formatted with args (fmt verbs). An
// unknown key returns the key itself, so a missing text is easy to spot.
func T(lang, key string, args ...any) string {
	text, ok := catalogs[lang][key]
	if !ok {
		if text, ok = catalogs[Default][key]; !ok {
			return key
		}
	}
	if len(args) == 0 {
		return text
	}
	return fmt.Sprintf(text, args...)
}

// Lookup returns the text of key in lang's own catalog, without falling
// back to the default language.
func Lookup(lang, key string) (string, bool) {
	text, ok := catalogs[lang][key]
	return text, ok
}

// Keys are the keys of a language's catalog.
func Keys(lang string) []string {
	var out []string
	for k := range catalogs[lang] {
		out = append(out, k)
	}
	return out
}
