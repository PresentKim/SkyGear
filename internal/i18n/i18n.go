// Package i18n is the server's own message catalog, for the texts the game
// sends in a player's language (docs/tech/i18n.md). Texts are looked up by
// key in the catalog of the player's language, then in the default one
// (Korean), so a language can be added by a file in locales/ and may be
// incomplete. Names of blocks and items are not here: the client translates
// those by itself from the resource pack's texts/*.lang and vanilla keys.
package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"

	"golang.org/x/text/language"
)

// Default is the language everything is written in first; a key missing in
// another language shows the default one.
const Default = "ko"

//go:embed locales/*.json
var files embed.FS

// catalogs are the texts by language code ("ko", "en") and key.
var catalogs = load()

func load() map[string]map[string]string {
	out := map[string]map[string]string{}
	entries, err := fs.ReadDir(files, "locales")
	if err != nil {
		panic(err)
	}
	for _, e := range entries {
		data, err := files.ReadFile("locales/" + e.Name())
		if err != nil {
			panic(err)
		}
		c := map[string]string{}
		if err := json.Unmarshal(data, &c); err != nil {
			panic(fmt.Sprintf("i18n: %s: %v", e.Name(), err))
		}
		out[strings.TrimSuffix(e.Name(), ".json")] = c
	}
	return out
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

// Keys are the keys of a language's catalog.
func Keys(lang string) []string {
	var out []string
	for k := range catalogs[lang] {
		out = append(out, k)
	}
	return out
}
