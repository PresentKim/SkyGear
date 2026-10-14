package i18n

import (
	"regexp"
	"testing"

	"golang.org/x/text/language"
)

var verbs = regexp.MustCompile(`%[-+# 0-9.]*[a-zA-Z%]`)

// TestCatalogs: another language only has keys the default has, with the same
// format verbs.
func TestCatalogs(t *testing.T) {
	for _, l := range Languages() {
		for key, text := range catalogs[l] {
			def, ok := catalogs[Default][key]
			if !ok {
				t.Errorf("%s: key %s is not in the default catalog", l, key)
				continue
			}
			if len(verbs.FindAllString(text, -1)) != len(verbs.FindAllString(def, -1)) {
				t.Errorf("%s: %s has other format verbs than the default", l, key)
			}
		}
	}
}

func TestLookup(t *testing.T) {
	catalogs["xx"] = map[string]string{"only.xx": "x %d"}
	catalogs[Default]["only.default"] = "d"
	defer delete(catalogs, "xx")
	defer delete(catalogs[Default], "only.default")
	if got := T("xx", "only.default"); got != "d" {
		t.Errorf("a missing key must fall back to the default: %q", got)
	}
	if got := T("xx", "only.xx", 3); got != "x 3" {
		t.Errorf("format: %q", got)
	}
	if got := T("xx", "nope"); got != "nope" {
		t.Errorf("unknown key: %q", got)
	}
	if Lang(language.Korean) != "ko" || Lang(language.French) != Default || Lang(language.AmericanEnglish) != "en" {
		t.Errorf("languages: %s %s", Lang(language.Korean), Lang(language.French))
	}
}
