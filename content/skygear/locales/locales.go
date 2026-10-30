// Package locales registers Sky Gear's text catalog (<lang>.json and the
// other <lang>.*.json files, with the generated vanilla names in
// <lang>.vanilla.json) with common/i18n on import. The platform's texts are
// in platform/locales.
package locales

import (
	"embed"

	"github.com/presentkim/skygear/common/i18n"
)

//go:embed *.json
var files embed.FS

func init() { i18n.Register(files) }
