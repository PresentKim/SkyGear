package i18n_test

import (
	"testing"

	"github.com/presentkim/skygear/common/i18n/i18ntest"
)

func TestCatalogs(t *testing.T) { i18ntest.Catalogs(t) }

// The library's own packages: the server checks its packages itself.
func TestKeysInCode(t *testing.T) { i18ntest.KeysInCode(t, "..", 10, nil) }

// timed.go keeps the marker text stored in time-limited items.
func TestNoKoreanInCode(t *testing.T) {
	i18ntest.NoKoreanInCode(t, "..", nil, "timed/timed.go", "i18ntest/i18ntest.go")
}
