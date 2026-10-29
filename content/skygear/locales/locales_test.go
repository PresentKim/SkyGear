package locales_test

import (
	"testing"

	"github.com/presentkim/skygear/common/i18n/i18ntest"
	_ "github.com/presentkim/skygear/content/skygear/locales"
	_ "github.com/presentkim/skygear/platform/locales"
)

// TestCatalogs: the platform and the game catalogs together.
func TestCatalogs(t *testing.T) { i18ntest.Catalogs(t) }

// TestKeysInCode: the literal keys of the platform, the content and the
// command.
func TestKeysInCode(t *testing.T) {
	i18ntest.KeysInCode(t, "../../../platform", 50, nil)
	i18ntest.KeysInCode(t, "..", 100, nil)
	i18ntest.KeysInCode(t, "../../../cmd", 1, nil)
}

// TestNoKoreanInCode: what stays in code is command names (skygear
// menu.go).
func TestNoKoreanInCode(t *testing.T) {
	i18ntest.NoKoreanInCode(t, "..", nil, "../menu.go")
	i18ntest.NoKoreanInCode(t, "../../../cmd", nil)
}
