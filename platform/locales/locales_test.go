package locales_test

import (
	"testing"

	"github.com/presentkim/skygear/common/i18n/i18ntest"
	_ "github.com/presentkim/skygear/platform/locales"
)

func TestCatalogs(t *testing.T) { i18ntest.Catalogs(t) }

// TestNoKoreanInCode: the platform's texts are all in the catalog.
func TestNoKoreanInCode(t *testing.T) { i18ntest.NoKoreanInCode(t, "..", nil) }
