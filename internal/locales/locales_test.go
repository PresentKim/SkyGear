package locales_test

import (
	"testing"

	"github.com/presentkim/dragonfactory/common/i18n/i18ntest"
	_ "github.com/presentkim/dragonfactory/internal/locales"
)

func TestCatalogs(t *testing.T) { i18ntest.Catalogs(t) }

// TestKeysInCode: the literal keys of internal/ and main.go.
func TestKeysInCode(t *testing.T) {
	i18ntest.KeysInCode(t, "..", 100, nil)
	i18ntest.KeysInCode(t, "../../main.go", 1, nil)
}

// TestNoKoreanInCode: what stays in code is command names (dragonfactory
// menu.go).
func TestNoKoreanInCode(t *testing.T) {
	i18ntest.NoKoreanInCode(t, "..", nil, "dragonfactory/menu.go")
	i18ntest.NoKoreanInCode(t, "../../main.go", nil)
}
