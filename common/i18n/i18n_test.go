package i18n

import (
	"testing"

	"golang.org/x/text/language"
)

func TestLookup(t *testing.T) {
	catalogs["xx"] = map[string]string{"only.xx": "x %d"}
	catalogs[Default]["only.default"] = "d"
	defer delete(catalogs, "xx")
	if catalogs["en"] == nil {
		catalogs["en"] = map[string]string{}
		defer delete(catalogs, "en")
	}
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
